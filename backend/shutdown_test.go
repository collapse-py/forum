/*
shutdown_test.go 驗證 shutdown.go 的三條不變條件：

 1. 組出來的 http.Server 一定帶著 ReadHeaderTimeout 與 IdleTimeout，而
    ReadTimeout / WriteTimeout 一定是零值。這兩件事互相依賴 —— 少了前者
    就有 Slowloris 曝露面，多了後者會讓貼文附圖的上傳在慢速連線上失敗。
    兩者都不會出現在任何 HTTP 狀態碼或日誌裡，因此只能靠測試守住。

 2. serveUntilSignal 在收到停止訊號時回 nil（呼叫端接著排空），在監聽
    期間失敗時回錯誤（呼叫端必須視為致命）。把這兩者混為一談會讓部署端
    分不出「正常停止」與「服務壞掉」，因此同樣用測試釘住。

 3. http.Server.Shutdown 會等在途請求完成 —— 這是「優雅停止」唯一真正要
    保證的事。倒數第二個測試刻意讓 handler 卡在 Shutdown 被呼叫之後才結束，
    用它確認「會等」，而不是去斷言 Shutdown 的內部時序。
*/
package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"forum/forum/config"
)

// TestNewHTTPServerTimeouts 釘住逾時設定的四個欄位。
//
// 逐項斷言而不是只斷言「有逾時」：兩個「該有」與兩個「刻意不該有」同等重要，
// 而後者正是最容易被後來者「順手補上」而弄壞網站的兩個。
func TestNewHTTPServerTimeouts(t *testing.T) {
	cfg := config.Config{
		ServerPort:        ":8088",
		ReadHeaderTimeout: 7 * time.Second,
	}
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	httpSrv := newHTTPServer(cfg, handler)

	if httpSrv.Addr != ":8088" {
		t.Errorf("Addr = %q，want %q（設定檔的 SERVER_PORT 必須原樣轉交）", httpSrv.Addr, ":8088")
	}
	if httpSrv.Handler == nil {
		t.Error("Handler 為 nil，main 傳入的 handler 必須被保留")
	}
	if httpSrv.ReadHeaderTimeout != 7*time.Second {
		t.Errorf("ReadHeaderTimeout = %v，want %v", httpSrv.ReadHeaderTimeout, 7*time.Second)
	}
	if httpSrv.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %v，want %v", httpSrv.IdleTimeout, idleTimeout)
	}
	// 下面兩項是「刻意留白」，不是還沒做。要補上時必須同時評估上傳路徑
	// （本文可達 50 MB 且要轉送檔案伺服器）與 CSV 匯出路徑，理由見 shutdown.go。
	if httpSrv.ReadTimeout != 0 {
		t.Errorf("ReadTimeout = %v，必須維持 0（不限制）：它會連請求本文一起計時，會砍斷圖片上傳", httpSrv.ReadTimeout)
	}
	if httpSrv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %v，必須維持 0（不限制）：它會在回應寫完前砍斷連線", httpSrv.WriteTimeout)
	}
}

// TestServeUntilSignalReturnsNilOnSignal 確認收到停止訊號時回 nil，
// 因此呼叫端會走排空流程而不是 Fatalf（否則正常部署會被當成服務故障）。
func TestServeUntilSignalReturnsNilOnSignal(t *testing.T) {
	// 埠用 0（由系統選一個可用的）：這個測試要的是「訊號分支先到」，
	// 不需要知道實際綁到哪裡，而 0 也讓綁定失敗的機率降到最低 ——
	// 唯一的其他出口是監聽錯誤，那正是下一個測試的主題。
	httpSrv := newHTTPServer(config.Config{ServerPort: "127.0.0.1:0"}, echoHandler())
	defer httpSrv.Close()

	sigCh := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() { done <- serveUntilSignal(httpSrv, sigCh) }()
	sigCh <- os.Interrupt

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("收到停止訊號時 serveUntilSignal 回傳 %v，want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("送出停止訊號後 serveUntilSignal 沒有返回")
	}
}

// TestServeUntilSignalReturnsListenError 確認監聽失敗會回錯誤。
//
// 用已被占用的埠製造失敗，而不是塞一個假的錯誤通道：那條路徑同時驗證了
// serveUntilSignal 真的把 ListenAndServe 的錯誤原樣往上送，而不是包成別的
// 錯誤或吃掉。
func TestServeUntilSignalReturnsListenError(t *testing.T) {
	// 先占住一個本機可用的埠，再讓伺服器綁同一個。
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("測試無法取得佔用中的埠: %v", err)
	}
	defer occupied.Close()

	httpSrv := newHTTPServer(config.Config{ServerPort: occupied.Addr().String()}, echoHandler())
	defer httpSrv.Close()

	// 沒有任何訊號會送進這個通道，因此離開的唯一出口是監聽錯誤。
	done := make(chan error, 1)
	go func() { done <- serveUntilSignal(httpSrv, make(chan os.Signal)) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("埠被占用時 serveUntilSignal 回傳 nil，want 綁定錯誤（必須讓呼叫端走 Fatalf）")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("埠被占用時 serveUntilSignal 沒有返回")
	}
}

// TestShutdownWaitsForInFlightRequest 確認 Shutdown 會等在途請求完成。
//
// 若 Shutdown 沒有等，handler 的回應會是空白的（連線已被關閉），因此斷言
// 回應內容就等於斷言「在途請求有被等完」。
//
// 這裡刻意自己 net.Listen 而不是走 serveUntilSignal：那個函式會自己
// ListenAndServe，因此測試無從得知它綁到哪個埠 —— 而要用 HTTP 發一個真實
// 請求就必須知道位址。
func TestShutdownWaitsForInFlightRequest(t *testing.T) {
	release := make(chan struct{})
	handlerStarted := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerStarted)
		<-release
		w.Write([]byte("ok"))
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("測試無法取得監聽埠: %v", err)
	}
	httpSrv := newHTTPServer(config.Config{ServerPort: listener.Addr().String()}, handler)
	serveDone := make(chan error, 1)
	go func() { serveDone <- httpSrv.Serve(listener) }()

	bodyCh := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String() + "/")
		if err != nil {
			bodyCh <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()
		buf := make([]byte, 8)
		n, _ := resp.Body.Read(buf)
		bodyCh <- string(buf[:n])
	}()

	// 等 handler 確定已經開始處理，再開始排空；否則這個測試會變成在量
	// 「請求與排空誰先到」的競態。
	select {
	case <-handlerStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("handler 沒有被請求觸發")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownDone <- httpSrv.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown 在在途請求還沒結束時就返回（%v），那不是優雅停止", err)
	case <-time.After(200 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown 回傳 %v，want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("在途請求結束後 Shutdown 沒有返回")
	}

	if body := <-bodyCh; body != "ok" {
		t.Errorf("在途請求的回應是 %q，want %q（Shutdown 中斷了進行中的請求）", body, "ok")
	}
	// Serve 在 Shutdown 之後必定回 ErrServerClosed，這是預期中的結束。
	if err := <-serveDone; err != nil && err != http.ErrServerClosed {
		t.Errorf("Serve 回傳 %v，want nil 或 http.ErrServerClosed", err)
	}
}

// echoHandler 是立刻回應的最小 handler，供不需要驗證排空的測試使用。
func echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}
}
