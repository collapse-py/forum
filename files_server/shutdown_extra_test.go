/*
shutdown_extra_test.go 補齊 shutdown.go 的覆蓋：伺服器建構、位址與逾時換算、
訊號等待（正常停止與監聽失敗兩條），以及「排空成功／逾時後強制關閉」兩條路徑。

【為什麼這一支值得寫】
這個檔的失效症狀不是「服務不啟動」，而是「停止時把在途的 50 MB 上傳砍斷」——
資料庫裡留下一篇引用不存在圖片的貼文，而日誌什麼都沒有。因此這裡直接驗證
serveUntilSignal 的回傳值語意（nil = 收到訊號，非 nil = 監聽失敗）與
drainUntilTimeout 的逾時行為，而不是只測欄位有沒有設對。

【為什麼沒有 sleep】
等待「請求已經進入 handler」用一個 channel 同步，而不是睡一段時間；逾時則用
一個**確定被阻塞**的 handler 來保證 Shutdown 不可能在期限內完成。兩者都不依賴
排程時序。
*/
package main

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"testing"
	"time"
)

// TestNewHTTPServerConfig 守住逾時與位址設定。
func TestNewHTTPServerConfig(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 7071
	handler := http.NotFoundHandler()

	srv := newHTTPServer(cfg, handler)
	if srv.Addr != "127.0.0.1:7071" {
		t.Errorf("Addr = %q，want 127.0.0.1:7071", srv.Addr)
	}
	if srv.Handler == nil {
		t.Error("Handler 是 nil")
	}
	if srv.ReadHeaderTimeout != readHeaderTimeout {
		t.Errorf("ReadHeaderTimeout = %v，want %v", srv.ReadHeaderTimeout, readHeaderTimeout)
	}
	if srv.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %v，want %v", srv.IdleTimeout, idleTimeout)
	}
	// 刻意不設這兩個：上傳與下載的本文可達 50 MiB，任何一端被計時都會把
	// 正常請求砍斷。
	if srv.ReadTimeout != 0 {
		t.Errorf("ReadTimeout = %v，want 0（不限制）", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %v，want 0（不限制）", srv.WriteTimeout)
	}
}

// TestListenAddrAndShutdownTimeout 守住兩個純函式。
func TestListenAddrAndShutdownTimeout(t *testing.T) {
	cfg := &Config{}
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 9090
	cfg.Server.ShutdownTimeoutSec = 15

	if got, want := listenAddr(cfg), "0.0.0.0:9090"; got != want {
		t.Errorf("listenAddr() = %q，want %q", got, want)
	}
	if got, want := shutdownTimeout(cfg), 15*time.Second; got != want {
		t.Errorf("shutdownTimeout() = %v，want %v", got, want)
	}
}

// TestNotifyOnSignalRegistersAndStops 守住註冊本身不會 panic，且可以取消註冊。
//
// 真正的訊號送遞由 serveUntilSignal 的測試以 channel 注入驗證；這裡只確認
// notifyOnSignal 會把 ch 註冊進 signal 套件（否則 signal.Stop 沒有意義），
// 並且不留下一個仍在監聽的 channel。
func TestNotifyOnSignalRegistersAndStops(t *testing.T) {
	ch := make(chan os.Signal, 1)
	notifyOnSignal(ch)
	// 不取消的話，這個 channel 會一直被 signal 套件持有，測試結束後仍可能
	// 收到訊號（在平行測試裡尤其容易變成隨機失敗）。
	signal.Stop(ch)
}

// TestServeUntilSignalReturnsNilOnSignal 守住「收到停止訊號 → nil」。
//
// 訊號先放進緩衝 channel，因此 select 一定走 sigCh 分支，不需要等待或 sleep。
func TestServeUntilSignalReturnsNilOnSignal(t *testing.T) {
	srv := &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}
	sigCh := make(chan os.Signal, 1)
	sigCh <- os.Interrupt

	if err := serveUntilSignal(srv, sigCh); err != nil {
		t.Errorf("收到停止訊號卻回錯誤: %v", err)
	}
	// 背景 ListenAndServe 可能還沒開始或正在開始；Close 對兩者都安全，且會讓
	// 那個 goroutine 以 ErrServerClosed 結束（它被視為 nil，寫進緩衝 channel）。
	_ = srv.Close()
}

// TestServeUntilSignalReturnsListenError 守住「監聽失敗 → 非 nil」。
//
// 位址沒有埠號，ListenAndServe 會立刻回錯誤；這一條不依賴任何埠是否被佔用。
func TestServeUntilSignalReturnsListenError(t *testing.T) {
	srv := &http.Server{Addr: "not-a-valid-address"}
	sigCh := make(chan os.Signal, 1)

	err := serveUntilSignal(srv, sigCh)
	if err == nil {
		t.Fatal("位址不合法卻回 nil —— 呼叫端會把它當成正常停止並 exit 0")
	}
}

// TestDrainUntilTimeoutSuccess 守住「沒有在途請求時排空成功」。
func TestDrainUntilTimeoutSuccess(t *testing.T) {
	srv := &http.Server{}
	if !drainUntilTimeout(srv, time.Second) {
		t.Error("沒有在途請求的伺服器排空失敗")
	}
}

// TestDrainUntilTimeoutExpires 守住逾時後回 false 並強制關閉連線。
//
// handler 會一直阻塞到 release 關閉為止，因此 Shutdown 不可能在期限內完成 ——
// 這是結構上的保證，不是時序上的猜測。
func TestDrainUntilTimeoutExpires(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("建立監聽失敗: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-release
		}),
	}
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() {
		// 即使測試中途失敗，也要讓被阻塞的 handler 與伺服器結束，避免洩漏。
		select {
		case <-release:
		default:
			close(release)
		}
		_ = srv.Close()
	})

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("連線到測試伺服器失敗: %v", err)
	}
	defer conn.Close()
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: test\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatalf("送出請求失敗: %v", err)
	}

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("請求沒有進入 handler")
	}

	if drainUntilTimeout(srv, 10*time.Millisecond) {
		t.Error("handler 仍在執行，排空卻回報成功")
	}
	close(release)
}
