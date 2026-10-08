/*
main 是檔案伺服器的執行進入點（package main，不可被其他套件 import）。

【職責】
本檔只做「組裝與啟動」，不承載業務邏輯。啟動順序固定為：
讀取設定 → 建立儲存後端 → 連線 Redis（失敗則降級）→ 組出 HTTP Handler
→ 監聽，收到停止訊號後依 shutdown.go 的流程排空在途請求再結束。
逾時取值、訊號處理與監聽等待位於同模組的 shutdown.go。

【檔案分工】
config.go    設定檔（TOML）解析與兜底值。
storage.go   儲存後端（本機磁碟 / S3）與檔名產生。
server.go    路由、上傳與刪除、媒體 token 中介層。
shutdown.go  監聽、停止訊號、優雅停止。
測試分佈與各檔相同：每個 _test.go 只驗證同名的非測試檔。

【為什麼不保留 package 層的 redisClient 全域變數】
原本 redisClient 是 package 級別的變數，由本檔賦值、server.go 的中介層讀取。
把它收進 Server 結構體之後，測試可以注入一個假的介面實作（見 server.go 的
mediaTokenStore），因此「token 不存在回 401」與「Redis 故障回 503」兩條分支
都能直接驗證 —— 前者是本服務的安全邊界，後者是它最容易被靜默降級掉的地方。

【主要依賴】
config.go / storage.go / server.go 的說明涵蓋各自的相依。

【關鍵設計決策與限制】
 1. Redis 連不上時**不**停止啟動，只記警告並關閉靜態檔案的 token 驗證。
    這是刻意的取捨：上傳與刪除不依賴 Redis，而「圖片要帶 token 才能讀」是
    這個服務最外層的防護。停用它的後果是已發行的媒體連結不再需要 token
    就能讀取，而繼續啟動（拒絕服務）會讓後端連上傳都做不了 —— 那個故障的
    半徑大得多。docs/DEPLOYMENT.md 因此要求正式環境必須讓 Redis 可用。
 2. 停止流程的逾時取自 server.shutdown_timeout_seconds，預設 15 秒，
    必須大於部署端的停止上限（compose 的 stop_grace_period、systemd 的
    TimeoutStopSec）。反過來的話，「貼文存得下但圖片上傳失敗」會在每次
    部署時發生 —— 那是排空被從中途砍掉的症狀。
 3. log.Fatal 走 os.Exit(1)，因此其後的 defer 都不會執行。停止流程用
    log.Printf 而非 Fatal，正是為了讓 defer（redisClient.Close）與
    os.Exit(0) 的路徑都能正常走完。
*/
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/redis/go-redis/v9"
)

// mediaPingTimeout 是啟動時探測 Redis 的期限。
//
// 刻意短（2 秒）：這是啟動路徑上的阻塞點，Redis 的位址寫錯時每多等一秒就多
// 慢一秒。超時後會降級（見檔頭第 1 點），因此不需要更寬的期限。
const mediaPingTimeout = 2 * time.Second

func main() {
	// 設定檔路徑相對於工作目錄（見 config.go 的說明），不是執行檔目錄。
	cfg, err := loadConfig("config.conf")
	if err != nil {
		log.Fatalf("載入配置失敗: %v", err)
	}

	backend, err := newStorageBackend(cfg)
	if err != nil {
		log.Fatalf("初始化存儲失敗: %v", err)
	}

	redisClient := connectRedis(cfg)

	srv := NewServer(cfg, backend, mediaTokenStoreOrNil(redisClient))
	// 逾時設定集中在 newHTTPServer 裡，這裡只負責組裝與啟動。
	httpSrv := newHTTPServer(cfg, srv.Handler())

	// 訊號處理必須在開始監聽**之前**就緒：容器環境的停止訊號可能在啟動的
	// 瞬間就送達，若先 ListenAndServe 再註冊，那段空窗期內的訊號會依預設
	// 行為直接終止行程（等於回到沒有優雅停止的狀態）。
	sigCh := make(chan os.Signal, 1)
	notifyOnSignal(sigCh)
	defer signal.Stop(sigCh)

	log.Printf("Starting files server on %s (storage=%s)", listenAddr(cfg), cfg.Storage.Type)
	if cfg.Upload.Token != "" {
		log.Println("Upload token authentication enabled")
	}
	if redisClient != nil {
		log.Println("Static file media token authentication via Redis enabled")
	}

	// 監聽期間發生錯誤（埠被占用等）不可降級：exit 1。收到停止訊號則排空在途
	// 請求後正常返回，exit 0。部署端靠這個區分分辨「正常停止」與「服務壞掉」。
	if err := serveUntilSignal(httpSrv, sigCh); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}

	if drainUntilTimeout(httpSrv, shutdownTimeout(cfg)) {
		log.Printf("在途請求已全部完成")
	}
	log.Println("Files server stopped")

	// 排在停止流程之後：Redis 用戶端仍可能被排空中完成的請求用到，而
	// os.Exit 會跳過 defer，所以這裡必須同步呼叫。
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Printf("關閉 Redis 連線時發生錯誤: %v", err)
		}
	}
}

// connectRedis 建立 Redis 用戶端並主動探測一次。
//
// 探測失敗時回 nil（而不是回一個連不上的用戶端）：後續對 nil 用戶端的呼叫會
// panic，而「不驗證靜態檔案 token」是一個明確的降級行為，記在日誌裡即可 ——
// 見檔頭第 1 點的理由。
func connectRedis(cfg *Config) *redis.Client {
	if cfg.Redis.Addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), mediaPingTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis 連線失敗，將跳過靜態檔案 token 驗證: %v", err)
		client.Close()
		return nil
	}
	log.Printf("Redis 連線成功 addr=%s db=%d", cfg.Redis.Addr, cfg.Redis.DB)
	return client
}

// mediaTokenStoreOrNil 把 *redis.Client 轉成中介層需要的介面，並在它為 nil 時
// 回 nil（而非「包住一個 nil 指標的介面」）。
//
// 這是 Go 裡最容易踩的陷阱之一：把 nil 指標指派給介面會得到一個非 nil 的介面
// 值，介面內的 == nil 判斷因此失效，症狀是 Stop 流程中呼叫 Close 時 panic ——
// 而那正是「正常停止」該走通的路徑。
func mediaTokenStoreOrNil(client *redis.Client) mediaTokenStore {
	if client == nil {
		return nil
	}
	return client
}
