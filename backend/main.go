/*
main 是 forum 論壇後端的執行進入點（package main，不可被其他套件 import）。

【職責】
本檔只做「組裝與啟動」，不承載任何業務邏輯。啟動順序固定為：
讀取設定 → 初始化日誌 → 連線 Redis → 連線 MySQL → schema 遷移
→ 初始化 Google OAuth2 → 建立 Session Manager → 掛載 HTTP Handler → 監聽，
收到停止訊號後依 shutdown.go 的流程排空在途請求再結束。
逾時取值、訊號處理與監聽等待位於同模組的 shutdown.go。

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
GET /api/admin/monitor 監控資料（Go 執行期、MySQL 連線池、Redis、搜尋引擎、
請求統計與限流器計數），供 /admin/monitor 頁顯示。
GET /api/admin/log 管理員操作稽核紀錄（含欄位級 diff），供 /admin/log 頁顯示。
GET /api/admin/stats 內容趨勢統計（日別新增量與三份排行），供 /admin/stats 頁顯示。
其餘路徑由 ServeMux 的 "/" catch-all 提供前端靜態檔案。
本套件沒有匯出符號。

【速率限制】
寫入型端點依成本分成三組獨立額度（內容寫入、圖片上傳、OAuth），全部以
用戶端 IP 為單位，只擋非 GET 請求 —— 讀取端點對匿名訪客開放，限流它們會
直接壞掉首頁。超額回 429 並附 Retry-After。三組限流器的背景清理 goroutine
由本檔在啟動時呼叫 httpapi.Server.StartRateLimitCleanup 啟動，並共用同一個
可取消的背景 context（見 main 的 stopBackground 說明）。

【監控與持久化】
每個請求都會被 metrics 套件計數（正規化後的路由、狀態碼分類、延遲直方圖、
分鐘桶）。記憶體保留最近 120 分鐘；已結束的分鐘由背景 goroutine 每 20 秒寫進
forum_request_metrics，保留 MONITOR_RETENTION_HOURS 小時（預設 24）後刪除。
啟動時先把保留期內的既有彙總讀回記憶體，因此監控頁的時間軸在重啟後不會變空白。

【停止流程】
背景 goroutine（限流清理、監控落盤、ES 索引重建、稽核清理、封鎖清理）共用一個
context.WithCancel 建立的 context。優雅停止的順序是「停止接受新請求 → 等在途
請求 → 取消背景 context → 同步寫出最後一次分鐘彙總」，四步的順序本身是這段程式
唯一需要注意的地方，理由見各步旁的註解。

【主要依賴】
forum/forum/config 設定檔解析。
forum/forum/logger 全域 logger，其 Fatalf 會記錄後直接 os.Exit(1)。
forum/forum/data MySQL 連線池與建表遷移。
forum/forum/session 以 Redis 儲存 session 的 cookie 管理器。
forum/forum/auth Google OAuth2 授權碼流程。
forum/forum/es Elasticsearch 傳輸層（貼文搜尋與索引重建）。
forum/forum/metrics 請求統計容器與分鐘彙總的持久化。
forum/forum/audit 管理員操作稽核的寫入、查詢與依時間清理。
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

優雅停止由 notifyOnSignal 與 http.Server.Shutdown 組成：收到 SIGINT／SIGTERM 後
先停止接受新連線，再等在途請求完成（上限 SHUTDOWN_TIMEOUT_SECONDS），最後才讓
背景 goroutine 停止並寫出最後一次監控彙總。沒有這個流程時，部署（重啟、容器
更新、systemd restart）會直接中斷進行中的請求 —— 症狀是使用者看到「貼文貼到
一半不見了」、圖片上傳失敗，而且沒有任何錯誤訊息。

http.Server 刻意只設定 ReadHeaderTimeout 與 IdleTimeout，不設定 ReadTimeout 與
WriteTimeout。前者是 Slowloris 的解藥；後者兩者會誤傷貼文附圖的上傳路徑
（本文可達 50 MB，且後端還要轉送給檔案伺服器），那是全站最長單一請求。
沒有逾時的代價由 IdleTimeout 收斂：keep-alive 連線不會無限期佔住 goroutine。

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
	"forum/forum/audit"
	"forum/forum/auth"
	"forum/forum/config"
	"forum/forum/data"
	"forum/forum/es"
	"forum/forum/httpapi"
	"forum/forum/ipban"
	"forum/forum/logger"
	"forum/forum/metrics"
	"forum/forum/session"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
)

// main 依固定順序建立所有相依元件，最後進入阻塞監聽直到收到停止訊號。
//
// 回傳值：無。兩條離開路徑的差別是本函式唯一需要留意的地方：
//   - 監聽期間發生錯誤（埠被占用等）→ logger.Fatalf，exit code 1。
//   - 收到 SIGINT／SIGTERM → 走完優雅停止流程後正常返回，exit code 0。
//     因此部署端看到 exit code 0 才知道「停止是走完流程的」，而不是被中途砍掉。
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

	// 背景工作的唯一取消管道。五個背景 goroutine（限流清理、監控落盤、
	// ES 索引重建、稽核清理、封鎖清理）共用它，因此「停止」是一個決定而不是
	// 五個。
	//
	// 為什麼不用 context.Background()：那會讓每個 goroutine 各自為政，而
	// context 正是它們唯一能響應停止的機制 —— 取消之後就會自己結束，不必等
	// 行程被 OS 收掉。
	//
	// 取消時機刻意放在 http.Server.Shutdown **之後**（見檔案層的停止流程）：
	// 在途請求還在服務中時，限流清理與監控落盤都還有用（尤其是落盤 ——
	// 排空期間結束的分鐘桶要有人寫出去），先取消只會讓監控少一段資料。
	//
	// defer 只是保險：正常路徑會在停止流程裡明確呼叫 stopBackground。
	backgroundCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	// 啟動限流器的背景清理。三個限流器各自在記憶體裡保存「最近視窗內各次
	// 請求的時間戳」，而 Allow 只會把某個 key 的 slice 縮短、不會刪掉 key，
	// 因此沒有這道定期掃描，map 會隨時間只增不減（見 ratelimit.go 檔頭）。
	//
	// 清理頻率刻意等於視窗長度：視窗過了，最老那筆紀錄就會自然過期，
	// 此刻掃一次就能把它連同 key 一起回收。再更頻繁只是徒增鎖競爭。
	//
	// 這裡傳入 backgroundCtx（而非 context.Background()）：優雅停止時它會被取消，
	// 三條清理 goroutine 因此自動停止，不必修改 httpapi 套件。
	//
	// 延遲啟動而非在 NewServer 內啟動：NewServer 刻意不開背景 goroutine，
	// 「何時開始有背景工作」應該是呼叫端明確的決定，而不是建構式的副作用。
	srv.StartRateLimitCleanup(backgroundCtx, cfg.RateLimitWindow)

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

	 * 用可取消的 backgroundCtx：優雅停止時索引重建會被中斷。這是刻意的取捨 ——
	 * 重建是 O(全部貼文) 的工作，讓它跑完可能還要數秒，而那一刻行程本來就要
	 * 結束；下次啟動時會再重建一次，因此中斷它不會留下永久性的落後狀態。
	 */
	go func() {
		indexed, err := srv.RebuildSearchIndex(backgroundCtx)
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

	/*
	 * 啟動監控統計的持久化。
	 *
	 * 兩件事，順序有意義：
	 *
	 *  1. 先讀回重啟前的分鐘彙總（LoadHistory）。它只是把資料載進記憶體，
	 *     失敗時只會讓監控頁少了重啟前的曲線 —— 因此錯誤只記警告，不影響
	 *     服務啟動。監控是診斷工具，它自己缺一塊資料不該讓論壇開不了。
	 *
	 *  2. 再啟動定期寫入（StartFlusher）。放在 Handler() 之後才啟動是刻意的：
	 *     從這一行開始，mux 才真的開始收請求；先啟動 flusher 只會讓第一個
	 *     分鐘的桶多一點點內容，沒有實質差別，但「服務開始服務的時刻」因此
	 *     仍然是一個明確的分界。
	 *
	 * ctx 傳 backgroundCtx：優雅停止時它會被取消，StartFlusher 收到
	 * ctx.Done() 會再做一次收尾寫入。停止流程仍會在 main 裡同步補一次
	 * FlushPending（理由見該處），兩者不重複計數的保證來自 flushed 旗標
	 * 是在鎖內先標記再送出。
	 *
	 * 寫入週期 20 秒刻意不等於一分鐘：它寫的是「已結束的分鐘」，所以週期
	 * 只影響「資料落盤的延遲上限」與「最壞情況下重啟會遺失多久的資料」。
	 * 20 秒讓最壞情況是一分鐘（當前分鐘）加上零到 20 秒的排隊延遲，而不是
	 * 一分鐘加上最多一整分鐘的 tick 延遲。
	 */
	registry := srv.Metrics()
	if registry != nil {
		// since 對齊「時間軸實際會顯示的區間」，而不是 MONITOR_RETENTION_HOURS。
		//
		// 這個差別不是小數點：時間軸只有 TimelineMinutes 格（預設 120 分鐘），
		// 因此讀回 24 小時的 1440 列會在 20 秒內被 pruneLocked 裁到 120 列 ——
		// 其餘 1320 列純屬白讀（一次 DB 往返）、白佔記憶體，且沒有任何作用。
		// 用保留期當 since 也會讓 log 裡「已讀回 N 分鐘」變成一個與畫面
		// 完全無關的數字。
		since := time.Now().Add(-time.Duration(registry.TimelineMinutes()) * time.Minute)
		if loaded, err := registry.LoadHistory(context.Background(), db, since); err != nil {
			// 表不存在（首次部署尚未跑遷移）或權限不足都會走到這裡。監控頁
			// 仍會顯示「本次啟動以來」的即時曲線，因此這是警告而不是錯誤。
			logger.Warnf("[MONITOR] 無法讀回歷史請求統計（監控頁將只顯示本次啟動的資料）: %v", err)
		} else if loaded > 0 {
			logger.Infof("[MONITOR] 已讀回 %d 分鐘的歷史請求統計", loaded)
		}
		if err := registry.StartFlusher(backgroundCtx, metrics.FlusherOptions{
			DB:             db,
			Interval:       20 * time.Second,
			RetentionHours: cfg.MonitorRetentionHours,
		}); err != nil {
			// 拿不到 *sql.DB 只會發生在 main 的組裝被改動時；此時監控頁
			// 仍可用（只是沒有重啟前的歷史），因此記警告即可。
			logger.Warnf("[MONITOR] 請求統計持久化未啟用: %v", err)
		}
	}

	/*
	 * 啟動稽核紀錄的定期清理。
	 *
	 * 這裡刻意不設任何中斷條件：稽核紀錄的寫入發生在每個後台操作裡（有交易
	 * 保護），清理卻與論壇功能無關 —— 它只是讓這張表不會無限長大。若清理
	 * 因為資料庫暫時不可用而停下來，後台操作本身完全不受影響，只是紀錄會
	 * 暫時變多。因此用獨立的 goroutine 而不併入 metrics 的 flusher：
	 * 兩者的失敗語意不同，綁在一起會讓其中一個的問題被誤判成另一個的。
	 *
	 * 清理週期一小時一次。稽核紀錄的用途是「有人來查」的時候還查得到，
	 * 而多存幾小時完全沒有差別，因此小時級的粒度對這個用途綽綽有餘。
	 *
	 * ctx 傳 backgroundCtx：停止時清理迴圈會立刻結束。這裡沒有像監控落盤那樣
	 * 另做收尾寫入 —— 稽核紀錄的寫入發生在請求裡（在途請求完成後資料就已在
	 * 資料庫），清理本身沒有任何待收的東西。
	 */
	go audit.NewPruner(db, time.Duration(cfg.AuditRetentionDays)*24*time.Hour, time.Hour).Run(backgroundCtx)

	/*
	 * 啟動 IP 封鎖名單的定期清理。
	 *
	 * 清理的必要性：封鎖名單是 Redis sorted set，member 是 IP、score 是到期
	 * 秒數。查詢時（ipban.IsBanned）會正確地把已過期的項目視為未封鎖，因此
	 * **不清理不會造成功能錯誤** —— 它只會讓 ZSET 慢慢長大，而那個增長是
	 * 單調的。一小時清理一次的寫入量可以忽略（見 ipban.NewPruner）。
	 *
	 * 與 session 的 StartCleanup 一樣：都交給呼叫端決定何時停止，這裡傳
	 * backgroundCtx。
	 *
	 * 這裡的 go 關鍵字不可省略：Pruner.Run 與 metrics flusher 一樣是「阻塞
	 * 在 select 上直到 ctx 被取消」的迴圈，忘了 go 會讓主流程停在這裡，
	 * 連 http.ListenAndServe 都還沒被呼叫 —— 症狀是行程活著、日誌印出
	 * 「Forum server started」（那行在更前面）、但埠上沒有任何監聽，
	 * 而且接下來的啟動日誌一條都不會再出現。與上面的稽核清理同一個寫法。
	 */
	if srv.Blocks() != nil {
		go ipban.NewPruner(srv.Blocks(), time.Hour).Run(backgroundCtx,
			func(removed int, err error) {
				if err != nil {
					logger.Warnf("[BLOCKLIST] 無法清理過期封鎖: %v", err)
					return
				}
				if removed > 0 {
					logger.Infof("[BLOCKLIST] 已清理 %d 筆過期封鎖", removed)
				}
			})
		logger.Infof("[BLOCKLIST] IP 封鎖名單已啟用")
	} else {
		logger.Warnf("[BLOCKLIST] 未啟用 IP 封鎖名單（缺少 Redis 連線）")
	}

	logger.Infof("[AUDIT] 稽核紀錄保留 %d 天，過期紀錄每小時清理一次", cfg.AuditRetentionDays)

	// 逾時設定集中在 newHTTPServer 裡，這裡只負責組裝與啟動。
	httpSrv := newHTTPServer(cfg, srv.Handler())
	// 訊號處理必須在開始監聽**之前**就緒：容器環境的停止訊號可能在啟動的
	// 瞬間就送達，若先 ListenAndServe 再註冊，那段空窗期內的訊號會依預設
	// 行為直接終止行程（等於回到沒有優雅停止的狀態）。
	sigCh := make(chan os.Signal, 1)
	notifyOnSignal(sigCh)
	defer signal.Stop(sigCh)

	logger.Infof("[SERVER] Forum server started at %s", cfg.ServerPort)

	/*
	 * 進入監聽，收到停止訊號才返回。
	 *
	 * 這裡的順序是整個停止流程的起點：先讓 ListenAndServe 跑在 goroutine，
	 * 再用 select 等「伺服器錯誤」或「停止訊號」兩者之一。
	 *
	 * 刻意讓 serveUntilSignal 自己開 goroutine（而不是在 main 裡 go 出去再
	 * 開 channel 回傳）：錯誤與訊號誰先到是不確定的，而這個函式必須把
	 * 「監聽期間就出錯」與「被要求停止」這兩件事分開回報 —— 前者是致命錯誤
	 * （exit 1），後者要走完排空流程（exit 0）。
	 */
	if err := serveUntilSignal(httpSrv, sigCh); err != nil {
		// 綁定埠失敗（例如埠已被占用）屬於不可降級的致命錯誤。
		logger.Fatalf("[SERVER] Server stopped: %v", err)
	}

	/*
	 * 優雅停止：先停止接受新連線，再等在途請求完成。
	 *
	 * 為什麼不能反過來（先取消背景 ctx 再 Shutdown）：在途請求還在服務中，
	 * 限流與監控落盤都還有作用 —— 尤其監控，排空期間結束的分鐘桶要有人寫
	 * 出去，先取消只會讓監控圖少一段。
	 *
	 * 逾時上限取自 SHUTDOWN_TIMEOUT_SECONDS：Shutdown 一旦逾時就會立即返回
	 * error，而此時連線仍開著，因此必須補一次 Close 強制中斷，否則在途
	 * 請求會一直吊住行程到部署端自己動手殺掉。
	 */
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Warnf("[SERVER] 優雅停止未能在 %s 內完成（%v），仍有在途請求將被中斷", cfg.ShutdownTimeout, err)
		if closeErr := httpSrv.Close(); closeErr != nil {
			logger.Warnf("[SERVER] 強制關閉連線時發生錯誤: %v", closeErr)
		}
	} else {
		logger.Infof("[SERVER] 在途請求已全部完成")
	}

	// 背景工作收尾。放在 Shutdown 之後：此時不會再有任何請求進來，清理
	// goroutine 停止才不會漏掉東西。
	stopBackground()

	/*
	 * 最後一次分鐘彙總寫入，由主流程**同步**執行。
	 *
	 * 為什麼不只靠 StartFlusher 收到 ctx.Done() 的那次寫入：那是背景
	 * goroutine，而接下來 main 就會 return、行程隨即結束 —— 那次寫入很可能
	 * 還沒跑完。同步寫入才能保證「停止前的最後一段流量」真的落到資料庫。
	 *
	 * 與背景那次寫入不會重複計數：FlushPending 在鎖內先把候選分鐘標記成
	 * flushed 再送出，因此兩邊同時呼叫時各取各的、互不重疊。
	 *
	 * 用全新的 context 而非已逾時的 shutdownCtx：排空逾時不代表資料庫不可
	 * 寫入，而這個 context 只服務一次寫入，不需要與請求共用期限。
	 */
	if registry != nil {
		flushCtx, cancelFlush := context.WithTimeout(context.Background(), flushTimeout)
		if written := registry.FlushPending(flushCtx, db); written > 0 {
			logger.Infof("[MONITOR] 停止前已寫出最後 %d 分鐘的請求統計", written)
		}
		cancelFlush()
	}

	logger.Infof("[SERVER] Forum server stopped")
}
