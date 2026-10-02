/*
shutdown.go 是 package main 的「監聽與停止」部分：把設定組成一個帶逾時的
http.Server、等待停止訊號、或監聽期間的致命錯誤。

【為什麼獨立成檔】
main.go 已經是「依固定順序組裝所有相依元件」的地方，逾時取值與停止流程是另一
個主題（服務怎麼結束，而不是服務怎麼啟動）。放在一起會讓 main 的閱讀順序被
「為什麼只設這兩個逾時」這類與組裝無關的理由打斷，而這些理由正是最不該被
省略的部分。

【對外介面（本檔全部匯出符號）】
newHTTPServer：以設定組出帶逾時的 *http.Server。
serveUntilSignal：啟動監聽並等到停止訊號或致命錯誤。
notifyOnSignal：註冊要處理的訊號。
本套件不註冊任何 HTTP 路由，也不匯出給其他套件使用。

【主要依賴】
僅使用標準函式庫（errors、net/http、os、os/signal、syscall、time）與
config、logger 兩個本專案套件（分別提供逾時欄位與停止訊息的日誌）。

【關鍵設計決策】
 1. 只設 ReadHeaderTimeout 與 IdleTimeout，不設 ReadTimeout / WriteTimeout。
    理由逐條寫在各自常數與函式的註解裡 —— 這是本檔最容易被「順手補上」而弄壞
    網站的設定，而補上的人不會重讀整個設計理由。
 2. 停止訊號與監聽錯誤必須分開回報。監聽期間出錯（埠被占用）不可降級，
    那是 exit 1；收到訊號則要排空在途請求，那是 exit 0。混為一談會讓部署端
    無法分辨「服務正常停止」與「服務壞掉」。
 3. 訊號處理在開始監聽之前就緒。容器環境的停止訊號可能在啟動瞬間送達，
    先監聽再註冊會讓那段空窗期內的訊號依預設行為直接終止行程。
 4. 不引入第三方 graceful shutdown 套件。標準函式庫的 Shutdown 已經涵蓋這個
    專案需要的一切（停止接受新連線 + 等在途請求），多一個相依只會增加
    「這個站能不能用預設值跑」這個問題的答案空間。
*/
package main

import (
	"errors"
	"forum/forum/config"
	"forum/forum/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// idleTimeout 是 keep-alive 連線的閒置上限：連線在這段時間沒有收到下一個請求
// 就被關閉。
//
// 刻意設定它的理由是「零值等於沒有限制」：沒有 IdleTimeout 時，一個開著
// keep-alive 的連線會一直佔著一個 goroutine 直到客戶端自己關掉，而對方什麼都
// 不必做就達到效果 —— 一個開著 keep-alive 的瀏覽器分頁就能讓這些 goroutine
// 累積到與流量無關的數量。
//
// 用 60 秒是因為它涵蓋瀏覽器在兩次請求之間的思考時間（讀一篇貼文、填一篇
// 回覆往往就落在這個量級），因此正常使用者不會感覺到差異，而「送完請求就
// 消失」的連線會在一分鐘內被回收。
//
// 刻意不設成常數設定檔的欄位：它沒有「設錯就出事」的方向 —— 太大只是多佔
// 一些資源，太小只會多幾次 TCP 交握，而前者的上限本來就被「反連線數」綁住。
const idleTimeout = 60 * time.Second

// flushTimeout 是停止流程最後那次分鐘彙總寫入的期限。
//
// 獨立於 SHUTDOWN_TIMEOUT_SECONDS：排空等待的是「使用者的請求」，這裡等的是
// 「自己寫資料庫」，兩者不該共用同一個預算。給 5 秒是因為這是幾十列的
// INSERT，寫不出去只會是資料庫本身有問題，而不該讓它拖住整個停止流程。
const flushTimeout = 5 * time.Second

// newHTTPServer 以設定檔組出帶逾時的 *http.Server（尚未開始監聽）。
//
// 逾時只設兩個，兩個都不設的話 net/http 的零值是「沒有限制」：
//
//   - ReadHeaderTimeout（來自 READ_HEADER_TIMEOUT_SECONDS）只卡「讀完請求
//     標頭」的時間，這是 Slowloris 的解藥：攻擊者送完一點點標頭後停住不再
//     送本文，每條連線就這樣佔著一個 goroutine 直到資源耗盡。
//   - IdleTimeout（idleTimeout）收掉閒置的 keep-alive 連線，理由見該常數。
//
// ReadTimeout 與 WriteTimeout **刻意留白**（零值＝不限制）：
//   - ReadTimeout 涵蓋整個請求本文，而貼文附圖的本文可達 50 MB、且後端還要
//     轉送給檔案伺服器（httpapi 的 ParseMultipartForm 與 60 秒轉送逾時）。
//     設一個「對上傳夠用」的數字，等於讓上傳在慢速連線上間歇性失敗 —— 那是
//     最難重現、也最難診斷的一類問題。
//   - WriteTimeout 從「開始寫回應」計時到「回應寫完」，而 CSV 匯出與依賴探測
//     的耗時會直接落在這段裡。
//
// 換言之，這兩個逾時若未來要加，必須同時評估上傳路徑與匯出路徑，而不是照
// 照常見範例填 30 秒。
func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.ServerPort,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// notifyOnSignal 註冊要處理的停止訊號到 ch。
//
// 同時收 os.Interrupt 與 SIGTERM 是一個刻意的選擇：
//   - os.Interrupt 是 Ctrl-C 與 docker stop 送來的訊號，本機開發最常見。
//   - SIGTERM 是 systemd、k8s、與大多數部署環境的標準停止訊號；它沒有對應的
//     常數，必須寫成 syscall.SIGTERM。
//
// 兩個都不收的後果不是「少一個快捷鍵」而是「部署端只能靠強制殺行程」：而
// 強制殺行程不會執行 defer、不會寫出最後一次監控彙總、也不會等在途請求。
//
// 容量 1 的緩衝由呼叫端決定：signal 這個套件在滿載時會丟棄後續訊號，而停止
// 本來就是單一事件，第二次 SIGTERM 對這個設計沒有新意義（真的需要強制結束
// 時再按一次即可，Go 的預設行為會讓第二次訊號直接終止行程）。
func notifyOnSignal(ch chan<- os.Signal) {
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
}

// serveUntilSignal 在背景啟動監聽，並等待「收到停止訊號」或「監聽發生錯誤」。
//
// 回傳值語意（兩者不可混為一談，理由見檔頭第 2 點）：
//   - nil：收到停止訊號。呼叫端接著走 http.Server.Shutdown 排空在途請求。
//   - 非 nil：ListenAndServe 回傳錯誤（最常見是綁定埠失敗）。呼叫端應視為
//     不可降級的致命錯誤。此時不呼叫 Shutdown —— 連線本來就沒有在服務。
//
// 為什麼把 ErrServerClosed 當成「不是錯誤」：它只會在有人呼叫過
// http.Server 的 Shutdown 或 Close 之後出現，而這個套件裡唯一會那樣做的
// 就是 main 的停止流程。讓它冒泡出去會讓 main 誤以為服務壞掉而走 Fatalf
// —— 症狀是「每次正常部署都印一行 Server stopped 的錯誤訊息並以 1 結束」。
//
// 呼叫端在收到 nil 後照樣可以再呼叫一次 Shutdown：那是冪等的，且回 nil。
func serveUntilSignal(httpSrv *http.Server, sigCh <-chan os.Signal) error {
	serveErr := make(chan error, 1)
	go func() {
		err := httpSrv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			// 呼叫端已要求停止：不是錯誤。
			err = nil
		}
		// 緩衝容量 1，因此這個送出永不阻塞，即使呼叫端已經因為其他錯誤
		// 而返回（例如 signal 先到、監聽錯誤後到）。
		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		// 監聽期間就出錯（埠被占用、權限不足…），不等停止訊號。
		return err
	case sig := <-sigCh:
		logger.Infof("[SERVER] 收到 %v，停止接受新連線", sig)
		return nil
	}
}
