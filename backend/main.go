/*
main 是 forum 論壇後端的執行進入點（package main，不可被其他套件 import）。

【職責】
本檔只做「組裝與啟動」，不承載任何業務邏輯。啟動順序固定為：
讀取設定 → 初始化日誌 → 連線 Redis → 連線 MySQL → schema 遷移
→ 初始化 Google OAuth2 → 建立 Session Manager → 掛載 HTTP Handler → 阻塞監聽。

【對外介面：HTTP 路由】
路由全部在 httpapi.Server.Handler() 註冊，main 本身不註冊任何路由。
GET /healthz 健康檢查，同時 ping MySQL 與 Redis。
GET /auth/google 導向 Google 授權頁。
GET /auth/callback OAuth2 回呼，換取 session cookie 後導回登入前頁面。
POST /api/logout 登出，需通過 TrustedOrigins 的來源檢查。
GET /api/check 查詢目前登入狀態與是否為管理員。
GET /api/forum/search 貼文全文搜尋（ES 不可用時降級為 MySQL 關鍵字比對）。
GET 與 POST /api/forum/posts[/{id}] 論壇貼文的讀取、建立、更新、刪除。
POST /api/forum/images 上傳貼文附圖，轉呼叫外部檔案伺服器。
POST /api/forum/image-tokens/release 釋放已不再使用的圖片存取 token。
GET /api/forum/profile 目前使用者個人資料。
GET /api/forum/public-profile 他人的公開資料。
/api/admin/forum/**、/api/admin/users/**、/api/admin/tags/** 後臺管理 API。
其餘路徑由 ServeMux 的 "/" catch-all 提供前端靜態檔案。
本套件沒有匯出符號。

【速率限制】
寫入型端點依成本分成三組獨立額度（內容寫入、圖片上傳、OAuth），全部以
用戶端 IP 為單位，只擋非 GET 請求 —— 讀取端點對匿名訪客開放，限流它們會
直接壞掉首頁。超額回 429 並附 Retry-After。三組限流器的背景清理 goroutine
由本檔在啟動時呼叫 httpapi.Server.StartRateLimitCleanup 啟動。

【主要依賴】
forum/forum/config 設定檔解析。
forum/forum/logger 全域 logger，其 Fatalf 會記錄後直接 os.Exit(1)。
forum/forum/data MySQL 連線池與建表遷移。
forum/forum/session 以 Redis 儲存 session 的 cookie 管理器。
forum/forum/auth Google OAuth2 授權碼流程。
forum/forum/es Elasticsearch 傳輸層（貼文搜尋與索引重建）。
forum/forum/httpapi 路由、中介層與請求/回應格式。
github.com/redis/go-redis/v9 Redis 用戶端。
golang.org/x/oauth2 為間接依賴，經由 forum/forum/auth 使用。

【關鍵設計決策與限制】
啟動順序是硬性相依，不可任意調換。logger 必須最先完成，否則後續任何
logger.Fatalf 都會寫進預設 stdout，設定的日誌檔拿不到真正的原因；MigrateMySQL
必須在開始服務之前完成，否則第一批請求可能撞到尚未建好的資料表。

每一個相依元件失敗都以 logger.Fatalf 快速失敗（exit code 1），刻意不做降級：
設定檔遺失、Redis 或 MySQL 連不上都屬於「不啟動比錯誤啟動好」的情況。因為
logger.Fatalf 走 os.Exit(1)，被呼叫點之後的 defer 都不會執行，下方的 defer
只在正常流程走到底時才有意義。

沒有 graceful shutdown：未註冊 signal.Notify，也沒有在收到 SIGTERM 時停止接受
新連線並等待在途請求完成。部署時的優雅停止需由外部系統（容器或反向代理）
負責，否則進行中的請求會被硬生生中斷。

http.ListenAndServe 使用的是零值 Server，沒有設定 ReadHeaderTimeout、
ReadTimeout、WriteTimeout，也沒有 IdleTimeout。這在文字型論壇的流量下可接受，
但同樣代表缺少 ReadHeaderTimeout 帶來的 Slowloris 曝露面。

設定檔路徑由 filepath.Join("config", "config.conf") 相對於「行程的工作目錄」
組成，而不是執行檔所在目錄。好處是切換環境只需換工作目錄，代價是必須從
backend 目錄啟動，否則會因找不到設定檔而 Fatal。使用 filepath.Join 也讓此路徑
在 Windows 與 POSIX 上都能正確組出分隔符。

依賴注入採「值 / 單一實例」風格：Config 以值傳遞（淺複製，slice 欄位仍共用
底層陣列），*sql.DB 與 *redis.Client 以指標共用。沒有介面抽象，因此無法注入
測試替身。
*/
package main

import (
	"context"
	"errors"
	"forum/forum/auth"
	"forum/forum/config"
	"forum/forum/data"
	"forum/forum/es"
	"forum/forum/httpapi"
	"forum/forum/logger"
	"forum/forum/session"
	"net/http"
	"path/filepath"

	"github.com/redis/go-redis/v9"
)

// main 依固定順序建立所有相依元件，最後進入阻塞監聽。
//
// 回傳值：無。ListenAndServe 在伺服器正常運行期間不會返回，只有綁定失敗或
// 執行期間發生錯誤才會返回，且該路徑一律以 logger.Fatalf 結束行程。
func main() {
	// 設定檔路徑相對於工作目錄（見檔案層說明），不是執行檔目錄。
	cfg, err := config.Load(filepath.Join("config", "config.conf"))
	if err != nil {
		// 設定檔讀不到就無法推導出任何組態，連「先跑起來再說」都不可行，直接終止。
		logger.Fatalf("[CONFIG] Failed to load config.conf: %v", err)
	}

	// 必須排在所有其他初始化之前：後續每一步的失敗都會走 logger.Fatalf，
	// 若 logger 還沒就緒，錯誤原因會被寫到預設 stdout 而不進日誌檔。
	if err := logger.Init(cfg.LogLevel, cfg.LogFile, cfg.LogFormat); err != nil {
		logger.Fatalf("[LOGGER] Failed to initialize logger: %v", err)
	}
	// 正常結束時關閉日誌檔句柄；Fatalf 走 os.Exit(1)，不會觸發這個 defer。
	defer logger.Close()

	// 同一個 Redis 實例被兩處共用，肩負兩種職責：
	//   1. session 儲存（交給 session.Manager，key 前綴 "forum:session:"）
	//   2. 論壇圖片存取 token（httpapi 內以 mediaRedis 持有同一個 Client）
	// 因此 Redis 不可用等同整站不可用，不存在「只掛掉其中一項」的中間狀態。
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	// 啟動時主動 Ping：go-redis 預設是 lazy dial，不探測的話第一個 HTTP 請求
	// 才會發現連不上，症狀會被誤判成「使用者未登入」而難以排查。
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatalf("[REDIS] Connection failed: %v", err)
	}
	defer redisClient.Close()

	// sql.DB 是連線池而非單一連線；OpenMySQL 內部設定池上限、連線最大存活時間並 Ping，
	// 失敗時回傳 nil，此處的 err 判斷因此是必要的。
	db, err := data.OpenMySQL(cfg.DbDSN, cfg.DbMaxOpenConns, cfg.DbMaxIdleConns, cfg.DbConnMaxLifetimeMinutes)
	if err != nil {
		logger.Fatalf("[DB] 資料庫連線開啟失敗: %v", err)
	}
	// 由 main 承擔關閉責任：*sql.DB 被所有 handler 共用，沒有單一持有者適合 defer 關閉，
	// 而 main 是唯一能確定「所有請求都結束」的時點。
	defer db.Close()

	// 啟動時執行 schema 遷移（CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS），
	// 讓部署不必另外跑 migration 步驟。刻意不引入版本號管理：代價是無法 downgrade，
	// 且遷移語句必須永遠保持冪等，才能在每次啟動時安全重跑。
	if err := data.MigrateMySQL(db); err != nil {
		logger.Fatalf("[DB] Create MySQL tables failed: %v", err)
	}

	// auth.Init 把 OAuth2 設定寫進 auth 套件的全域變數。
	// 不呼叫的話 /auth/google 會因為 OauthConfig 為 nil 而回 500，
	// 且 GetUserEmail 會在解參考時 panic，所以這一步不能省略也不能延後到請求期間。
	auth.Init(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	// Session 本體存在 Redis，cookie 只承載 24 bytes 隨機 token（hex 後 48 字元），
	// 因此伺服器端可以主動撤銷。cookieSecure 由設定檔的 COOKIE_SECURE 控制：
	// 本機以 http 測試時必須為 false，否則瀏覽器不會回傳帶 Secure 的 cookie。
	sessions := session.NewManager(cfg.CookieName, cfg.SessionExpire, cfg.CookieSecure, redisClient)

	// Handler() 內含 session 續期、安全標頭、來源檢查（CSRF）、請求日誌
	// 與靜態檔案掛載等中介層，因此建構 Server 後不需要再額外包一層 middleware。
	srv := httpapi.NewServer(cfg, db, sessions, redisClient)

	// 啟動限流器的背景清理。三個限流器各自在記憶體裡保存「最近視窗內各次
	// 請求的時間戳」，而 Allow 只會把某個 key 的 slice 縮短、不會刪掉 key，
	// 因此沒有這道定期掃描，map 會隨時間只增不減（見 ratelimit.go 檔頭）。
	//
	// 清理頻率刻意等於視窗長度：視窗過了，最老那筆紀錄就會自然過期，
	// 此刻掃一次就能把它連同 key 一起回收。再更頻繁只是徒增鎖競爭。
	//
	// 這裡用 context.Background 而非可取消的 context，是因為本程式目前
	// 沒有 graceful shutdown（見檔案層說明），沒有任何地方會取消它。
	// 未來接上 signal.Notify 時，只需把這個 ctx 換掉，限流清理就會
	// 自動跟著停止，不必修改 httpapi 套件。
	//
	// 延遲啟動而非在 NewServer 內啟動：NewServer 刻意不開背景 goroutine，
	// 「何時開始有背景工作」應該是呼叫端明確的決定，而不是建構式的副作用。
	srv.StartRateLimitCleanup(context.Background(), cfg.RateLimitWindow)

	/*
	 * 啟動後在背景重建 Elasticsearch 的貼文索引。

	 * 為什麼要重建：ES 是可拋棄的衍生資料（內容的真相在 MySQL），
	 * 但它不會自己知道「MySQL 多了哪些貼文」—— 服務停機期間發的文、
	 * 索引被誤刪、或第一次接上 ES 的部署，索引都會落後於資料庫。
	 * 每次啟動重建一次以消除這個落差，之後的增量維護由各寫入路徑即時處理
	 * （createForumPost / handleAdminForumPost 的 PUT 與 DELETE / createAdminUserPost）。

	 * 為什麼在 goroutine 裡而不是啟動流程中：重建是 O(全部貼文) 的操作，
	 * 就算只有幾百篇也不該讓服務「啟動中」長達數秒。更重要的是它是
	 * best-effort 的——ES 掛掉時這裡只會留下警告日誌，論壇照常服務，
	 * 搜尋則降級為 MySQL LIKE（見 search.go）。

	 * context.Background 而非可取消的 context：與上面的限流清理相同，
	 * 本程式沒有 graceful shutdown，沒有任何地方會取消它。
	 */
	go func() {
		indexed, err := srv.RebuildSearchIndex(context.Background())
		if err != nil {
			// errors.Is 判定 ErrDisabled：那是「設定檔沒填 ES_URL」的預期情況，
			// 用 Info 記錄即可（等同於這台站沒有啟用搜尋），不該報錯嚇到維運。
			if errors.Is(err, es.ErrDisabled) {
				logger.Info("[SEARCH] 未設定 ES_URL，搜尋將使用 MySQL 關鍵字比對")
				return
			}
			logger.Warnf("[SEARCH] 貼文索引重建失敗（搜尋將降級為 MySQL 關鍵字比對）: %v", err)
			return
		}
		logger.Infof("[SEARCH] 貼文索引重建完成，共 %d 篇", indexed)
	}()

	logger.Infof("[SERVER] Forum server started at %s", cfg.ServerPort)
	// 沒有 ReadHeaderTimeout 等逾時設定，也沒有 graceful shutdown（見檔案層說明）。
	if err := http.ListenAndServe(cfg.ServerPort, srv.Handler()); err != nil {
		// 綁定埠失敗（例如埠已被占用）同樣屬於不可降級的致命錯誤；
		// ErrServerClosed 在本程式不會出現，因為沒有任何地方呼叫 srv.Shutdown。
		logger.Fatalf("[SERVER] Server stopped: %v", err)
	}
}
