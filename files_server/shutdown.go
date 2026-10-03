/*
shutdown.go 是 files_server 的「監聽與停止」部分：把設定組成一個帶逾時的
http.Server、等待停止訊號、排空在途請求。

【為什麼需要這個檔】
原本的最後一行是 log.Fatal(http.ListenAndServe(...))。那在收到 SIGTERM 時直接
死 —— 沒有逾時、沒有排空、defer 不執行。症狀不是抽象的「不乾淨」：一次 50 MB 的
圖片上傳被砍斷時，後端已經把貼文寫進 MySQL 了，而檔案沒有落地。於是資料庫裡有
一篇文章引用一張不存在的圖片，而沒有任何日誌、沒有任何錯誤碼。

【與 backend 的對應關係】
這個檔刻意複製 backend/shutdown.go 的結構與命名（newHTTPServer / notifyOnSignal
/ serveUntilSignal），包括「監聽錯誤是 exit 1、正常停止是 exit 0」的區分。
理由不是偷懶：兩個服務在同一个部署裡由同一個 stop 指令驅動，它們對「怎麼算
正常結束」的判斷必須一致 —— 一個回 0 另一個回 1 會讓部署端以為有東西壞了。

【主要依賴】
僅標準函式庫。刻意不引入 graceful-shutdown 套件（理由同 backend）。

【關鍵設計決策】
 1. 只設 ReadHeaderTimeout 與 IdleTimeout，不設 ReadTimeout / WriteTimeout。
    與後端同一個理由且更重要：上傳本文可達 50 MiB，且 ReadTimeout 會連
    「本文收完」一起計時，WriteTimeout 會在寫回應時就砍斷 50 MB 的下載。
 2. ReadHeaderTimeout 給 30 秒，比後端的 10 秒寬鬆，理由是這個服務的請求
    路徑很少（只有上傳、刪除、讀檔三條），而讀檔那一條會被瀏覽器反覆發起
    —— 標頭太小不構成 Slowloris 的條件，但太緊的標頭期限會誤傷行動網路上的
    慢速連線。刻意與後端不同值：兩個服務面對的用戶與連線型態不同，綁成同一個
    常數只是省下幾行字，卻讓「誰該調哪一個」變得不清楚。
 3. Shutdown 逾時之後必須補一次 Close：逾時只會讓 Shutdown 回 error，連線仍然
    開著，不補的話在途請求會一直吊住行程到部署端自己動手殺掉。
*/
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// idleTimeout 是 keep-alive 連線的閒置上限。
//
// 理由與後端相同：沒有 IdleTimeout 時，一個開著 keep-alive 的連線會一直佔著
// 一個 goroutine 直到客戶端自己關掉。這裡特別值得設，因為讀檔的連線是使用者
// 停留最久的（他們會反覆抓同一張圖片）。
const idleTimeout = 60 * time.Second

// readHeaderTimeout 是「從接到連線到讀完整份請求標頭」的期限。
//
// 只卡標頭、不卡本文，理由見檔頭第 1 點。上傳的本文可達 50 MiB，ReadTimeout
// 會連它一起計時而在慢速連線上把正常上傳砍斷。
const readHeaderTimeout = 30 * time.Second

// newHTTPServer 以設定組出帶逾時的 *http.Server（尚未開始監聽）。
//
// ReadTimeout 與 WriteTimeout 刻意留白（零值＝不限制），理由見檔頭。
func newHTTPServer(cfg *Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              listenAddr(cfg),
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// listenAddr 組出 "host:port" 形式的監聽位址。
//
// 抽出來是因為它在 newHTTPServer 與 main 的啟動日誌裡各用一次，而兩處必須
// 是同一個字串：日誌印出來的位址若與實際綁定的不同，會讓「連不上」這類問題
// 從查設定檔變成查日誌。
func listenAddr(cfg *Config) string {
	return cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
}

// shutdownTimeout 回傳停止時等待在途請求的期限。
//
// 掛在設定的 server.shutdown_timeout_seconds 上，與後端的
// SHUTDOWN_TIMEOUT_SECONDS 預設相同（15 秒）。兩者必須是同一個量級，原因見
// config.go 對該欄位的說明。
func shutdownTimeout(cfg *Config) time.Duration {
	return time.Duration(cfg.Server.ShutdownTimeoutSec) * time.Second
}

// notifyOnSignal 註冊要處理的停止訊號到 ch。
//
// 同時收 os.Interrupt 與 SIGTERM：前者是 Ctrl-C 與 docker stop，後者是
// systemd 與 k8s。兩個都不收的後果是部署端只能靠強制殺行程。
//
// 容量 1 由呼叫端決定：停止是單一事件，第二次訊號對這個設計沒有新意義。
func notifyOnSignal(ch chan<- os.Signal) {
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
}

// serveUntilSignal 在背景啟動監聽，並等待「收到停止訊號」或「監聽發生錯誤」。
//
// 回傳值語意（兩者不可混為一談）：
//   - nil：收到停止訊號，呼叫端接著走 http.Server.Shutdown 排空在途請求。
//   - 非 nil：ListenAndServe 回傳錯誤（最常見是綁定埠失敗），呼叫端視為致命。
//
// 把 http.ErrServerClosed 當成「不是錯誤」的理由：它只會在有人呼叫過 Shutdown
// 或 Close 之後出現，而這個套件裡唯一會那樣做的就是 main 的停止流程。讓它
// 冒泡出去會讓每次正常部署都印一行錯誤並以 1 結束。
func serveUntilSignal(httpSrv *http.Server, sigCh <-chan os.Signal) error {
	serveErr := make(chan error, 1)
	go func() {
		err := httpSrv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			// 呼叫端已要求停止：不是錯誤。
			err = nil
		}
		// 緩衝容量 1，因此這個送出永不阻塞，即使呼叫端已經因為其他錯誤而返回。
		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		// 監聽期間就出錯（埠被占用、權限不足…），不等停止訊號。
		return err
	case sig := <-sigCh:
		log.Printf("收到 %v，停止接受新連線", sig)
		return nil
	}
}

// drainUntilTimeout 排空在途請求，逾時則強制關閉連線。
//
// 逾時之後的 Close 是必要的：Shutdown 一旦逾時就會立即回 error，而此時連線
// 仍開著，不補這一步的話在途請求會一直吊住行程到部署端自己動手殺掉。
//
// 回傳值表示是否在期限內完成（供呼叫端決定日誌等級，也讓測試能直接斷言
// 「排空成功」而不用去解析日誌）。
func drainUntilTimeout(httpSrv *http.Server, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("優雅停止未能在 %s 內完成（%v），仍有在途請求將被中斷", timeout, err)
		if closeErr := httpSrv.Close(); closeErr != nil {
			log.Printf("強制關閉連線時發生錯誤: %v", closeErr)
		}
		return false
	}
	return true
}
