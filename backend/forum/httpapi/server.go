/*
httpapi 套件的組裝層：把設定、MySQL 連線、Redis session 與前端靜態檔案組成
一個可直接交給 http.ListenAndServe 的 http.Handler。

對外介面只有 Server、NewServer 與 (*Server).Handler；requireLogin、
frontendRoot、safeStaticFileServer 等其餘符號都是套件內部實作細節，
不供其他套件依賴。

關鍵設計決策：

 1. 路由集中在 Handler() 一處註冊。Go 1.22+ 的 ServeMux 樣式分兩類：不含
    尾斜線者（如 "/forum"）為完全比對，含尾斜線者（如 "/forum/"）為子樹比對。
    子樹樣式會吃掉該前綴下所有未被更具體樣式攔截的路徑，所以部分 handler 必須
    在函式內部自行比對 r.URL.Path，未定義的子路徑才不會被誤當成有效請求。
 2. 驗證與授權分兩層：requireLogin 負責「是誰」，requireAdminForum（在
    forum_admin_handlers.go）負責「能不能」。因此 /api/admin/* 路由刻意不掛
    requireLogin，改由各 handler 自行呼叫 requireAdminForum，避免出現兩套
    重複的權限判斷。
 3. 中介層採「包裹 handler」而非 http.Server 全域包裹，唯一套在整棵 mux
    外層的只有 Refresh 與 LoggingMiddleware，理由見 Handler() 末尾。
 4. 前端檔案路徑不寫死：frontendRoot、frontendAssetPath、frontendAssetsRoot
    會依執行檔位置與工作目錄試多組候選路徑，兼顧 go run、編譯後 binary 與
    容器部署三種啟動方式。
*/

package httpapi

import (
	"context"
	"database/sql"
	"forum/forum/auth"
	"forum/forum/config"
	"forum/forum/es"
	"forum/forum/ipban"
	"forum/forum/logger"
	"forum/forum/metrics"
	"forum/forum/session"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Server 收攏所有 HTTP handler 需要的依賴。全部欄位由 NewServer 填入，
// 而 handler 一律以 method value（s.handleXxx）的形式傳給 ServeMux，
// 因此測試時可以只填少數欄位、其餘留 nil，不必建置真實的 DB 或 Redis。
type Server struct {
	// cfg 為啟動時載入的設定快照，以值型別持有，避免外部在執行期竄改。
	// 供 IsAdminEmail、TrustedOrigins、FilesServer 內外部 URL 轉換等使用。
	cfg config.Config
	// db 為 *sql.DB 連線池，供貼文、留言、檢舉與帳號狀態查詢共用。
	// 本檔不負責關閉它，連線生命週期由 main 控管。
	db *sql.DB
	// sessions 包裝 Redis session：token 產生、cookie 寫入、sliding expiration
	// 與身分解析都經由它，handler 不直接操作 session key。
	sessions *session.Manager
	// 限流器依「端點成本」分成三組，各自獨立的額度。分成三組的理由見
	// config.RateLimitRequests 的說明：圖片上傳會呼叫外部服務並在 Redis
	// 建立 token，OAuth 會產生站外導向，而留言只是寫一行 MySQL，混用同一份
	// 額度的結果必然是「要嘛放行上傳轟炸、要嘛把留言一起擋掉」。
	// 三者都是 nil-safe 的：未建立時對應的 Middleware 不會被掛上（見 Handler）。
	writeRateLimiter  *RateLimiter
	uploadRateLimiter *RateLimiter
	authRateLimiter   *RateLimiter
	// mediaRedis 與 sessions 共用同一個 Redis client，但用途不同：這裡只用於
	// 媒體存取 token 的簽發與釋放（handleForumImageTokensRelease），
	// 另外也供 /healthz 做存活探測，因此允許為 nil（探測時會略過 Redis）。
	mediaRedis *redis.Client
	// es 是 Elasticsearch 的傳輸層，供貼文搜尋與索引維護使用（search.go）。
	// 允許為 nil：未在設定檔填 ES_URL 時 NewServer 會讓它保持 nil，
	// 搜尋則整條走 MySQL LIKE（見 search.go 的降級說明），/healthz 也略過它。
	es *es.Client
	// metrics 收集請求統計，供後臺監控頁（/admin/monitor）顯示。
	// 由 NewServer 建構；持久化（把分鐘彙總寫進 MySQL）不由這裡啟動，
	// 理由與三個限流器的清理相同 —— 「何時開始有背景工作」是呼叫端的決定，
	// 由 main 以 Registry.StartFlusher 表達。監控端點是監控資料的主要消費者，
	// 但 metricsMiddleware 與 handleAdminMonitor 兩處都對 nil 安全。
	metrics *metrics.Registry
	// blocks 是 IP 封鎖名單（Redis sorted set）。它與三個限流器是並存的兩個
	// 機制，理由見 httpapi/blocklist.go 檔頭。
	// 允許為 nil（測試以 struct literal 構造 Server、或 Redis 未設定時）；
	// 兩種情況下 withBlocklistHandler 都會直接放行。
	blocks *ipban.Store
	// trustedProxies 是可信任反向代理的位址段（TRUSTED_PROXY_CIDRS），
	// 決定要不要採信 X-Forwarded-For / X-Real-IP。信任模型與取捨見
	// httpapi/trustedproxy.go 的檔頭。
	//
	// 允許為 nil：nil 與「未設定」語意相同（退回標頭優先的舊行為），因此
	// 測試以 struct literal 構造 Server 時不需要填它。
	trustedProxies *trustedProxySet
}

// NewServer 以依賴注入的方式組裝 Server。cfg、db、sessions、redisClient 都由
// main 在啟動時建立，redisClient 同時作為 session 儲存與媒體 token 儲存。
// 回傳值不會啟動任何背景 goroutine —— 限流的定期清理由呼叫端另外以
// StartRateLimitCleanup 啟動，刻意不藏在建構子裡，讓「何時開始有背景工作」
// 是一個明確的決定。Handler() 每次呼叫都會重新建立一份 ServeMux，因此可
// 安全地重複呼叫，但實務上只在 main 呼叫一次。
//
// 唯一的副作用是一次性日誌：信任模型的安全後果必須在**啟動時**就看得到，而不
// 是等管理員恰好打開監控頁（見 logTrustedProxyMode）。它沒有啟動 goroutine，
// 因此與上面的分工並不衝突。
//
// es.Client 刻意不當成參數：它只是 cfg.ESURL 與 cfg.ESIndex 兩個字串的
// 組裝結果，沒有連線要在這裡建立（es.Client 內部是 http.Client，沒有
// dial），因此在這裡就地建構能讓「設定檔有沒有填 ES_URL」成為唯一的事實來源。
// 填了就是啟用，沒填 s.es 保持 nil、搜尋退回 MySQL（見 search.go）。
//
// 監控統計容器 metrics.Registry 同樣就地建構：它沒有任何外部依賴（不連資料庫、
// 不連 Redis、不 dial），建構本身只是一個 map 與幾個計數器。把它的參數形狀
// （保留幾分鐘、追蹤幾條路由）留給呼叫端反而是錯的 —— 那兩個值是「這個規模的
// 論壇」的常數，不是每個部署點該各自決定的事。
func NewServer(cfg config.Config, db *sql.DB, sessions *session.Manager, redisClient *redis.Client) *Server {
	srv := &Server{
		cfg:               cfg,
		db:                db,
		sessions:          sessions,
		writeRateLimiter:  NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow),
		uploadRateLimiter: NewRateLimiter(cfg.RateLimitUploadRequests, cfg.RateLimitUploadWindow),
		authRateLimiter:   NewRateLimiter(cfg.RateLimitAuthRequests, cfg.RateLimitAuthWindow),
		mediaRedis:        redisClient,
		// 封鎖名單與 session 與媒體 token 共用同一個 Redis 實例（同一條連線
		// 池），因此這裡不另外建構連線。
		blocks: ipban.New(redisClient),
		// 信任模型在這裡一次解析完（之後不可變，因此讀取無需鎖）：所有需要
		// 「這次請求來自哪裡」的地方共用同一份判斷，讓限流、封鎖與稽核紀錄
		// 不可能對同一個請求得出不同的答案。
		trustedProxies: parseTrustedProxyCIDRs(cfg.TrustedProxyCIDRs),
		metrics: metrics.New(metrics.Options{
			// 記憶體視窗刻意比資料庫保留期長：頁面重整時看到的是「自上次
			// 重新整理以來」的完整曲線，而不是只有最後 24 分鐘。時間軸的實際
			// 長度另由 Snapshot 的 maxTimelinePoints 收斂。
			WindowMinutes: 120,
			MaxRoutes:     200,
		}),
	}
	// 限流器的分攤鍵來源接到 Server 的信任模型上。刻意在這裡接而不是讓
	// RateLimiter 自己去讀設定：限流器不該知道設定檔的存在。
	for _, limiter := range []*RateLimiter{srv.writeRateLimiter, srv.uploadRateLimiter, srv.authRateLimiter} {
		if limiter != nil {
			limiter.SetIPResolver(srv.clientIP)
		}
	}
	if cfg.ESURL != "" {
		srv.es = es.New(cfg.ESURL, cfg.ESIndex)
	}
	srv.logTrustedProxyMode()
	return srv
}

// logTrustedProxyMode 在啟動時把「來源 IP 的信任模型」寫進日誌。
//
// 為什麼不能只靠監控頁：未設定 TRUSTED_PROXY_CIDRS 時程式採信使用者可控的
// X-Forwarded-For，因此限流、IP 封鎖與稽核紀錄的來源位址三者同時可被單一
// 標頭繞過。這個狀態若只在管理員恰好打開監控頁時才看得見，就等於把它留在
// 「管理員剛好知道要去看」的位置 —— 而啟動日誌是每個部署都一定會看的東西。
//
// 兩種「設定寫錯」分開報，嚴重程度不同：
//
//   - 宣告了卻一筆都用不了 → Error。那是設定寫錯，不是部署選擇。
//   - 部分項目寫錯 → Warn，並逐項列出寫錯的值。「部分寫錯」比較容易被忽略：
//     畫面看起來完全正常，只有那幾條位址段永遠不會被採信。
//
// 完全沒宣告 → Warn。這是相容性模式的刻意選擇，但後果仍然必須讓維運知道。
//
// 訊息裡一律寫出**後果**而不只是 key 名：讀日誌的維運不該還要去查設定檔才
// 知道這個警告代表什麼。
//
// 這是啟動時的一次性診斷，因此刻意放在 NewServer 而不是 Handler()：後者每次
// 呼叫都會重新註冊路由，放在那裡會讓同一段警告寫出無數次。
func (s *Server) logTrustedProxyMode() {
	report := s.trustReport()
	if report.Mode == "trusted-proxies" {
		if len(report.Invalid) > 0 {
			logger.Warnf("[TRUSTED-PROXY] TRUSTED_PROXY_CIDRS 有 %d 項無法解析（%s）；這些位址段的轉送標頭永遠不會被採信，來自它們的請求會以連線對端分桶，也就是共用同一組限流額度與封鎖查詢。",
				len(report.Invalid), strings.Join(report.Invalid, "、"))
		}
		return
	}
	if report.Declared != "" {
		logger.Errorf("[TRUSTED-PROXY] 宣告了 TRUSTED_PROXY_CIDRS（%s）但沒有任何一項能解析成位址段，因此限流與 IP 封鎖仍可被使用者自送的 X-Forwarded-For 繞過，稽核紀錄的來源位址也不可當成證據。",
			report.Declared)
		return
	}
	logger.Warnf("[TRUSTED-PROXY] 未設定 TRUSTED_PROXY_CIDRS：程式採信 X-Forwarded-For 最左項。若本站前面沒有會覆寫該標頭且使用者無法繞過的代理，限流與 IP 封鎖都可被單一偽造標頭繞過，稽核紀錄的來源位址也不可當成證據。")
}

// Metrics 回傳監控統計容器，供 main 啟動持久化 goroutine 與讀回歷史。
//
// 刻意匯出：統計容器的生命週期由 NewServer 開始，但「把分鐘彙總寫進資料庫」
// 與「啟動時讀回歷史」都是需要資料庫與背景 goroutine 的工作，屬於 main 的職責
// （與 StartRateLimitCleanup 由 main 呼叫是同一個分工）。若把這兩件事藏進
// NewServer，Server 就會在建立時啟動 goroutine，而測試每建構一次 Server 就
// 多一條永遠不會退出的 goroutine。
//
// 回傳值可能為 nil（測試以 struct literal 構造 Server），呼叫端必須自行判斷。
func (s *Server) Metrics() *metrics.Registry {
	return s.metrics
}

// Blocks 回傳 IP 封鎖名單的存取層，供 main 啟動過期清理。
//
// 與 Metrics 同一個理由：清理是有背景工作的，而「何時開始有背景工作」屬於
// 呼叫端的決定，不該由 NewServer 偷偷啟動 goroutine（否則每建構一次 Server
// 就多一條永遠不會退出的 goroutine）。
//
// 回傳值可能為 nil（測試以 struct literal 構造 Server），呼叫端必須自行判斷。
func (s *Server) Blocks() *ipban.Store {
	return s.blocks
}

// StartRateLimitCleanup 為三個限流器啟動定期清理的背景 goroutine。
//
// 存在的理由：限流器的 hits map 只會在 Allow 時把某個 key 的 slice 縮短，
// 卻不會刪掉 key 本身。對一個長期運作、公開入口會被爬蟲掃過的論壇，
// 這個 map 會只增不減（見 ratelimit.go 檔頭的記憶體成長說明）。
//
// interval 傳 0 代表讓每個限流器採用自己的預設（見 StartCleanup）；
// 呼叫端若想讓清理頻率跟視窗長度一致，應傳入與 RateLimitWindow 同一個量級的
// 時間值 —— 太频繁只是徒增鎖競爭，太稀疏則舊 key 存活較久。
//
// 這個函式必須在 NewServer 之後、開始服務之前呼叫。ctx 取消時三個 goroutine
// 都會停止，因此未來接上 graceful shutdown 不需要再改動這裡。
//
// 這裡刻意只回傳、不等待：StartCleanup 本身會立刻返回。
func (s *Server) StartRateLimitCleanup(ctx context.Context, interval time.Duration) {
	limiters := []*RateLimiter{s.writeRateLimiter, s.uploadRateLimiter, s.authRateLimiter}
	for _, limiter := range limiters {
		if limiter != nil {
			limiter.StartCleanup(ctx, interval)
		}
	}
}

// rateLimit 把限流中介層掛到 handler 上，但放行 GET 請求。
//
// 為什麼要這個薄包裝而不是在 Handler 裡直接呼叫 limiter.Middleware：
//  1. nil-safe。測試常以 struct literal 構造 Server 而不填這三個欄位
//     （見 forum_handlers_test.go），若直接呼叫 nil 指標的 Middleware 會在
//     建構路由時就 panic，讓那些測試完全跑不起來。
//  2. 集中表達意圖：Handler 裡每一條掛限流的路由都寫成 s.rateLimit(
//     s.writeRateLimiter, s.handleXxx)，一眼看得出「這條受哪一組額度管」。
//
// 為什麼放行 GET：內容端點的 GET 是公開讀取（貼文列表、留言），匿名訪客與
// 「載入更多」都會打到。限流它們會直接壞掉首頁，判斷標準與
// requireLoginForWrite 相同：會改動資料的方法才需要保護。
//
// 不適用於本身即為 GET 的端點 —— 那種情況必須用 rateLimitAllMethods，
// 否則限流會變成完全無作用的裝飾品（見該函式的說明）。
func (s *Server) rateLimit(limiter *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return s.applyRateLimit(limiter, next, true)
}

// rateLimitAllMethods 限流所有 HTTP 方法，包含 GET。
//
// 存在的唯一理由：/auth/google 與 /auth/callback 本身「就是」GET 導向
// （OAuth 授權碼流程是整頁導向，不是表單 POST）。若對它們套用會放行 GET 的
// rateLimit，限流器永遠不會被觸發，設定檔裡的 RATE_LIMIT_AUTH_* 會變成
// 沒有任何作用的死設定 —— 這正是本專案先前 RATE_LIMIT_REQUESTS 的處境，
// 不可重蹈。
//
// 這裡用「所有方法都限流」而不是「只擋 GET」是安全的：OAuth 流程的使用者
// 只會導向一次，正常使用不會撞到預設的 10 次 / 分鐘。
func (s *Server) rateLimitAllMethods(limiter *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return s.applyRateLimit(limiter, next, false)
}

// applyRateLimit 是上述兩個函式的共同實作。skipGet 決定 GET 是否直接放行。
//
// 把判斷集中在這裡（而不是複製到兩個函式）是為了讓「哪些端點該用哪一個」
// 成為唯一需要決定的事：內容端點用 rateLimit，GET 端點用 rateLimitAllMethods。
//
// 封鎖檢查在這一層**之下**（withBlocklistHandler 先於 limiter.Middleware 執行，
// 理由見 blocklist.go）：它必須涵蓋與限流完全相同的那組路由，否則會出現
// 「被封鎖的人仍可從某條沒掛封鎖的路由寫入」。讓它們由同一個函式組裝就是
// 為了杜絕那種不同步。
func (s *Server) applyRateLimit(limiter *RateLimiter, next http.HandlerFunc, skipGet bool) http.HandlerFunc {
	if limiter == nil {
		return next
	}
	if !skipGet {
		return s.withBlocklistHandler(limiter, s.blocks, next)
	}
	// 走 limiter.Middleware 的完整流程（含 Retry-After 與 429），
	// 因此包一層只做方法判斷，而不是自己呼叫 Allow —— 那樣會漏掉
	// Retry-After 的計算與取整。
	limited := s.withBlocklistHandler(limiter, s.blocks, next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}
		limited(w, r)
	})
}

// requireLogin 驗證「請求者是否為有效會話」，不通過時導向登入頁並附上
// return 參數，讓登入完成後能回到原本想造訪的頁面。
//
// 回應型態刻意分成兩種：
//   - 完全沒有會話：瀏覽器直接造訪頁面時需要的是重新導向而不是 JSON 錯誤，
//     因此回 303 See Other，避免使用者看到一段原始 JSON。
//   - 已登入但被停權：回 401，呼叫端多半是 XHR，前端據此提示並跳出登入。
//
// 副作用：每個通過此關的請求都會多查一次 forum_users.status。
func (s *Server) requireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := s.sessions.ResolveUser(r)
		if email == "" {
			// 帶上 RequestURI（path + query）而非只有 path，使用者回到原頁時
			// 篩選條件等狀態才不會遺失；QueryEscape 避免目標網址本身的參數
			// 污染登入頁網址。return 值在 auth.HandleLogin 會再經
			// isSafeReturnPath 驗證，非站內路徑會被丟棄。
			location := "/forum/login?return=" + url.QueryEscape(r.RequestURI)
			w.Header().Set("Location", location)
			// 303 而非 302：確保瀏覽器後續一定用 GET 重新請求，
			// 就算原本這次是用 POST 觸發的。
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		if !s.sessions.IsAdmin(r) {
			// 停權檢查只擋非管理員：管理員必須仍能進入系統，才能處理被停權
			// 使用者的檢舉與解封。session 的 is_admin 在登入當下就由
			// cfg.AllowedAdminEmail 決定，不會因為使用者被設為 SUSPENDED 而改變。
			suspended, err := s.isForumUserSuspended(r, email)
			if err != nil {
				// 查不到狀態不代表可以放行。DB 故障是伺服器問題而非授權結果，
				// 此時回 500 讓呼叫端重試，絕不能因為「查不到」就當成已驗證。
				internalError(w, "unable to verify user status")
				return
			}
			if suspended {
				unauthorized(w, "此帳號已被停權")
				return
			}
		}
		next(w, r)
	}
}

// requireLoginForWrite 只在「會改動資料的請求方法」上要求登入，GET 直接放行。
//
// 為什麼 GET 可以免驗證：論壇的貼文列表、留言與公開個人頁本來就是公開內容，
// 匿名訪客也要讀得到。若一併擋下，未登入者連首頁都開不了，等於把整個論壇變成
// 必須登入才能瀏覽的系統；而且寫入才需要知道「作者是誰」，讀取不需要。
func (s *Server) requireLoginForWrite(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只豁免 GET。HEAD / OPTIONS 理論上同屬讀取，但前端沒有使用它們，
		// 與其為罕見情境放寬規則，不如維持「非 GET 一律需驗證」的一致性。
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}
		// 把呼叫端已傳入的 next 直接交給 requireLogin，而不是另存一份，
		// 避免多包一層閉包、也讓「先判斷方法、再判斷身分」的順序一目了然。
		s.requireLogin(next)(w, r)
	}
}

// isForumUserSuspended 查詢 forum_users.status，回傳該帳號是否為 SUSPENDED。
//
// 邊界情況：查無此列（sql.ErrNoRows）視為「非停權」而不是錯誤，因為管理員帳號
// 與尚未完成首次登入的使用者都可能還不存在於該表，requireLogin 不該因此失敗。
// 其他資料庫錯誤一律上浮，由呼叫端回 500。
func (s *Server) isForumUserSuspended(r *http.Request, email string) (bool, error) {
	var status string
	// 以 request context 執行，client 斷線時可一併取消查詢，避免佔用連線。
	err := s.db.QueryRowContext(r.Context(), `SELECT status FROM forum_users WHERE email = ?`, email).Scan(&status)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// 以字面量精確比對，未來若新增其他狀態值會預設視為可存取；
	// 要改成「預設封鎖」時必須同步調整這裡與前端顯示。
	return status == "SUSPENDED", nil
}

// frontendRoot 推測前端檔案根目錄。候選順序代表優先度：dist 優先於未建置的
// 原始碼目錄，讓正式部署吃到打包結果，而開發環境沒有 dist 時自動退回原始碼。
// 一個候選都找不到時回傳預設路徑而非 panic：前端缺檔不該讓整個 API 服務起不來，
// 錯誤會在使用者造訪頁面時才以 404 的形式浮現。
//
// 注意：退回原始碼目錄只保證 HTML 與 CSS 可用，頁面邏輯不會動 —— 前端已全面
// 改用 TypeScript，未建置時的 *.ts 對瀏覽器而言不是可執行的 JavaScript，
// 而且 static.go 的 blockedSuffixes 也擋掉了 .ts（不公開原始碼）。
// 換言之未建置就是不能跑，開發時請先 npm run build。候選順序保留原始碼目錄
// 只是為了讓「忘記建置」得到明確的 404，而不是整頁壞掉。
func frontendRoot() string {
	candidates := []string{
		filepath.Join("..", "frontend", "dist"),
		filepath.Join("frontend", "dist"),
		filepath.Join("..", "frontend"),
		filepath.Join("frontend"),
		"web",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join("..", "frontend")
}

// frontendAssetPath 解析單一檔案的實際位置：檔案存在就直接用它；否則若
// frontendDir 指向 dist，就改試 dist 的上一層。兩段式嘗試是為了同時支援
// 「已打包」（檔案在 dist 根目錄）與「未打包」（檔案在上一層）兩種形態。
// 兩處都不存在時原樣回傳，交由 http.ServeFile 回 404。
func frontendAssetPath(frontendDir, name string) string {
	filePath := filepath.Join(frontendDir, name)
	if _, err := os.Stat(filePath); err == nil {
		return filePath
	}
	if filepath.Base(frontendDir) == "dist" {
		return filepath.Join(filepath.Dir(frontendDir), name)
	}
	return filePath
}

// frontendAssetsRoot 定位帶內容 hash 的建置資產目錄。與 frontendRoot 不同，
// 這裡把「執行檔所在目錄」排最前面：正式部署多是 backend/server 對應
// frontend/web/dist/assets 的相對關係，若只靠目前工作目錄猜測，在 systemd、
// Docker 等以不同 cwd 啟動的環境就會失效。
func frontendAssetsRoot(frontendDir string) string {
	candidates := []string{
		filepath.Join(frontendDir, "assets"),
		filepath.Join(frontendDir, "dist", "assets"),
		filepath.Join("..", "frontend", "web", "dist", "assets"),
		filepath.Join("frontend", "web", "dist", "assets"),
	}
	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		candidates = append([]string{
			filepath.Join(executableDir, "..", "frontend", "web", "dist", "assets"),
		}, candidates...)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join(frontendDir, "assets")
}

// Handler 建立並回傳整個論壇後端的 http.Handler：先解析前端資產位置、組出
// 完整路由表，再依序套上 session 更新與存取記錄兩個中介層。
// 每次呼叫都會重新建立 ServeMux 與重新解析前端路徑，因此不應放在請求路徑上。
func (s *Server) Handler() http.Handler {
	frontendDir := frontendRoot()
	// 啟動時把實際採用的路徑寫進日誌：前端檔案 404 是本專案最常見的部署錯誤，
	// 沒有這行只能靠猜是哪一組候選路徑沒命中。站名一併記下來：改錯設定檔時
	// 「送出的站名是什麼」是第一個要確認的事實，而它此時只存在於設定值裡。
	logger.Infof("[HTTP] frontend root=%s assets=%s forum=%q", frontendDir, filepath.Join(frontendDir, "assets"), s.cfg.ForumName)
	mux := http.NewServeMux()

	/*
		路由總表。中介層以「包裹 handler」的方式套用，實際執行順序由外而內為
		SecurityHeaders → Refresh → LoggingMiddleware → metricsMiddleware →
		ServeMux 路由比對 → 路由上掛的中介層 → handler。

		基礎設施（不需認證）
		  /healthz                        handleHealth
		  /auth/google                    rateLimitAllMethods(auth) → auth.HandleLogin（307 轉 Google）
		  /auth/callback                  rateLimitAllMethods(auth) → handleGoogleCallback
		  /api/logout                     requireTrustedOrigin → handleLogout（僅接受 POST）
		  /api/check                      handleCheck

		論壇（讀取可匿名，寫入需登入並限流）
		  /api/forum/posts                requireLoginForWrite → rateLimit(write) → handleForumPosts
		  /api/forum/images               requireLogin → rateLimit(upload) → handleForumImageUpload
		  /api/forum/image-tokens/release requireLogin → rateLimit(upload) → handleForumImageTokensRelease
		  /api/forum/posts/{id}...        requireLoginForWrite → rateLimit(write) → handleForumPostAction
		                                 （尾綴分派：/comments、/like、/report）
		  /api/forum/profile              requireLogin → rateLimit(write) → handleForumProfile
		  /api/forum/follows              requireLogin → rateLimit(write) → handleForumFollows
		                                 （GET 回追蹤清單、POST 切換追蹤）
		  /api/forum/following/posts      requireLogin → handleForumFollowingPosts（私有唯讀）
		  /api/forum/public-profile       handleForumPublicProfile（刻意公開且不限流）
		  /api/forum/public-posts         handleForumPublicPosts（依金鑰讀取某人貼文，公開且不限流）
		  /api/forum/search               handleForumSearch（公開唯讀；ES 不可用時降級 MySQL LIKE）

		後台（路由層不掛 requireLogin 也不掛限流；權限由各 handler 內的
		requireAdminForum 把關，理由見該處註解）
		  /api/admin/forum/posts[/{id}]   handleAdminForumPosts / handleAdminForumPost
		  /api/admin/forum/comments[/{id}] handleAdminForumComments / handleAdminForumComment
		  /api/admin/forum/reports[/{id}] handleAdminForumReports / handleAdminForumReport
		  /api/admin/forum/search         handleAdminForumSearch（含作者 email 精確比對）
		  /api/admin/users[/{...}]        handleAdminUsers / handleAdminUser
		  /api/admin/tags[/{...}]         handleAdminTags / handleAdminTag
		  /api/admin/monitor              handleAdminMonitor（依賴狀態 + 請求統計）
		  /api/admin/log                  handleAdminLog（管理員操作稽核紀錄查詢）
		  /api/admin/stats                handleAdminStats（內容趨勢統計）
		  /api/admin/export/*.csv         handleAdminExport*（CSV 匯出，附 Content-Disposition）
		  /api/admin/batch/tags           handleAdminBatchTags（批次覆寫標籤，每人一筆稽核）
		  /api/admin/batch/status         handleAdminBatchStatus（批次停權／復原，每人一筆稽核）
		  /api/admin/sessions             handleAdminSessions（列出活躍 session，只給 token 前綴）
		  /api/admin/sessions/revoke      handleAdminRevokeSessions（強制登出）
		  /api/admin/blocks               handleAdminBlocks（IP 封鎖名單；GET 列出、POST 封鎖／解封）
		  /api/forum/announcement         handleForumAnnouncement（公開：目前生效的公告，無則 announcement:null）
		  /api/admin/announcements[/{id}] handleAdminAnnouncements / handleAdminAnnouncement（後臺公告）
		  /api/admin/forum/posts/{id}/pin handleAdminPostOrPin → handleAdminPostPin（置頂／取消置頂）

		靜態頁面與資產（見下方各路由的個別說明）
		  /forum-manifest.json、/service-worker.js、/admin*、/assets/、
		  /forum*、/asset/、其餘落入 safeStaticFileServer

		限流的四個要點：
		  1. 內容端點用 rateLimit：只擋非 GET，因為讀取端點對匿名訪客開放，
		     限流它們會直接壞掉首頁與「載入更多」。判斷標準與
		     requireLoginForWrite 相同。
		  2. OAuth 端點用 rateLimitAllMethods：它們本身即為 GET 導向，
		     若用會放行 GET 的版本，限流就是沒有作用的裝飾品。
		  3. 限流在 requireLogin 之「內」：先確認身分再吃額度，未登入的
		     垃圾流量不會消耗已登入使用者的份額。
		  4. 超額時回 429 並附 Retry-After，見 ratelimit.go 的 Middleware。
	*/

	mux.HandleFunc("/healthz", s.handleHealth)
	// OAuth 兩條路由共用 auth 額度。/auth/callback 也要限：它是授權碼換 token
	// 的端點，是轟炸 Google 端點與消耗自身 quota 的合理目標。
	// 刻意用 rateLimitAllMethods 而非 rateLimit：這兩條本身即為 GET 導向，
	// 用會放行 GET 的版本會讓限流形同不存在（見該函式的說明）。
	mux.HandleFunc("/auth/google", s.rateLimitAllMethods(s.authRateLimiter, auth.HandleLogin))
	mux.HandleFunc("/auth/callback", s.rateLimitAllMethods(s.authRateLimiter, s.handleGoogleCallback))
	// 登出是「改變伺服器端狀態」的動作，因此掛 requireTrustedOrigin 做
	// 跨站請求偽造防護：即使攻擊者能讓 victim's 瀏覽器送出 POST，
	// 來源網域不在 TrustedOrigins 就會被擋下。
	mux.HandleFunc("/api/logout", s.requireTrustedOrigin(s.handleLogout))
	mux.HandleFunc("/api/check", s.handleCheck)

	// 依金鑰讀取某人的公開貼文列表。刻意公開且不限流，理由與 public-profile 相同：
	// 貼文本來就是公開內容，而這支端點的條件「author_email 等值 + created_at 排序」
	// 走的是索引（見 MigrateMySQL 第 21 步），不像搜尋那樣要掃全文。
	mux.HandleFunc("/api/forum/public-posts", s.handleForumPublicPosts)
	// 貼文列表（GET）開放匿名，發文（POST）需登入才能決定作者。
	// 發文另外掛上 write 額度：它是論壇最核心的寫入端點，也是灌水的主要目標。
	mux.HandleFunc("/api/forum/posts", s.requireLoginForWrite(
		s.rateLimit(s.writeRateLimiter, s.handleForumPosts)))
	// 圖片上傳與 token 釋放一律需登入：兩者都會動到 mediaRedis 的資源配額。
	// 掛 upload 額度（最緊的一組）：每次上傳都會讓外部檔案伺服器建立真實的
	// 儲存與一筆 Redis token，是全站單次成本最高的操作。
	mux.HandleFunc("/api/forum/images", s.requireLogin(
		s.rateLimit(s.uploadRateLimiter, s.handleForumImageUpload)))
	mux.HandleFunc("/api/forum/image-tokens/release", s.requireLogin(
		s.rateLimit(s.uploadRateLimiter, s.handleForumImageTokensRelease)))
	// 留言、按讚、檢舉都掛在 /api/forum/posts/ 之下。GET 取留言為公開讀取，
	// 其餘寫入方法需要登入 —— 這正是 requireLoginForWrite 存在的理由。
	// 這一條同時承載四種尾綴（/comments、/like、/report），因此一組限流同時
	// 管到留言、按讚與檢舉。它們都是便宜的 MySQL 寫入，共用一份額度是合理的。
	mux.HandleFunc("/api/forum/posts/", s.requireLoginForWrite(
		s.rateLimit(s.writeRateLimiter, s.handleForumPostAction)))
	// 個人資料的 PUT 也是寫入，因此掛上 write 額度。
	// 它的呼叫頻率遠低於發文（正常使用只在調整簡介時動一次），但惡意或
	// 失控的腳本可以藉此反覆寫入 MySQL，不該沒有上限。
	mux.HandleFunc("/api/forum/profile", s.requireLogin(
		s.rateLimit(s.writeRateLimiter, s.handleForumProfile)))
	// 追蹤：GET 回自己的追蹤清單，POST 切換追蹤／取消追蹤。
	// 掛 write 額度的理由與按讚相同（它就是一次資料庫開關）—— 但 s.rateLimit
	// 會放行 GET，因此讀取清單不吃額度，惡意輪詢 GET 不會把使用者的發文配額吃掉。
	mux.HandleFunc("/api/forum/follows", s.requireLogin(
		s.rateLimit(s.writeRateLimiter, s.handleForumFollows)))
	// 追蹤者的貼文動態。純私有讀取（沒有登入就沒有追蹤清單），因此只掛
	// requireLogin，不掛限流 —— 限流 GET 的前提是匿名訪客也要能讀，而這裡本來就擋掉了匿名。
	mux.HandleFunc("/api/forum/following/posts", s.requireLogin(s.handleForumFollowingPosts))
	// 公開個人頁刻意不掛任何認證中介層，因為它就是設計給未登入訪客看的。
	mux.HandleFunc("/api/forum/public-profile", s.handleForumPublicProfile)
	// 貼文搜尋。唯讀且刻意不限流：與貼文列表同一個道理，匿名訪客也要能搜尋。
	// 沒有掛 requireLoginForWrite 是因為它只有 GET 一種方法需要；把唯讀端點
	// 綁到「依方法決定是否驗證」的中介層只會多一層間接。
	mux.HandleFunc("/api/forum/search", s.handleForumSearch)

	// 後台路由一律不掛 requireLogin：管理員與停權檢查的順序在不同 handler
	// 間並不一致，交由 requireAdminForum 單點把關較不易漏掉。
	//
	// 後台刻意「不」掛限流：這裡的操作全都要管理員身分（已由 requireAdminForum
	// 把關），而管理員有合理的批次操作需求（例如一次替多個使用者指派標籤）。
	// 把面向一般使用者的額度套在後台，只會在管理員做正事時擋下他，卻擋不住
	// 真正的攻擊者 —— 攻擊者拿到管理員 session 之前就已經能打普通端點了。
	// 這裡的風險控管手段是授權檢查，不是速率限制。
	mux.HandleFunc("/api/admin/forum/posts", s.handleAdminForumPosts)
	// 這條前綴同時服務 /api/admin/forum/posts/{id}（修改／刪除）與
	// /api/admin/forum/posts/{id}/pin（置頂），因此掛的是分流用的
	// handleAdminPostOrPin 而非 handleAdminForumPost —— 理由見該函式的說明。
	// 特別注意：同一個樣板**不能**註冊兩次，Go 1.22 起的 ServeMux 會在啟動時
	// 直接 panic（conflicts with pattern），而症狀是「行程一啟動就死、
	// 日誌停在最後一行」，不會有任何 HTTP 層的錯誤訊息。
	mux.HandleFunc("/api/admin/forum/posts/", s.handleAdminPostOrPin)
	mux.HandleFunc("/api/admin/forum/comments", s.handleAdminForumComments)
	mux.HandleFunc("/api/admin/forum/comments/", s.handleAdminForumComment)
	mux.HandleFunc("/api/admin/forum/reports", s.handleAdminForumReports)
	mux.HandleFunc("/api/admin/forum/reports/", s.handleAdminForumReport)
	// 貼文搜尋（後臺）。權限由 handleAdminForumSearch 內的 requireAdminForum 把關，
	// 與其他 /api/admin/* 路由一致 —— 路由層不掛認證中介層的理由見上方註解。
	mux.HandleFunc("/api/admin/forum/search", s.handleAdminForumSearch)
	mux.HandleFunc("/api/admin/users", s.handleAdminUsers)
	mux.HandleFunc("/api/admin/users/", s.handleAdminUser)
	mux.HandleFunc("/api/admin/tags", s.handleAdminTags)
	mux.HandleFunc("/api/admin/tags/", s.handleAdminTag)
	// 監控端點。與其他 /api/admin/* 相同：路由層不掛認證中介層，權限由
	// handler 內的 requireAdminForum 把關（理由見上方註解）。
	//
	// 刻意不掛 rateLimitAllMethods：它每十秒被監控頁輪詢一次，是這台機器上
	// 最規則的合法流量。限流它只會在真正出事時多一條混淆的訊息。
	mux.HandleFunc("/api/admin/monitor", s.handleAdminMonitor)
	// 管理員操作稽核紀錄（唯讀查詢）。與其他 /api/admin/* 相同：路由層不掛認證
	// 中介層，權限由 handler 內的 requireAdminForum 把關。
	//
	// 刻意沒有限流：它是管理員點擊才發出的低頻請求，與監控頁的十秒輪詢不同。
	// 限流它的唯一效果是「稽核紀錄打不開」—— 那個頁面正是事故時第一個該看的
	// 地方，在那個時刻擋下它是最壞的取捨。
	mux.HandleFunc("/api/admin/log", s.handleAdminLog)
	// 內容趨勢統計。唯讀、刻意不限流（理由同 /api/admin/log）。
	mux.HandleFunc("/api/admin/stats", s.handleAdminStats)
	// CSV 匯出。三條路由刻意各自獨立而不是用一個 ?kind= 參數：它們的
	// 欄位、查詢與上限都不同，而一個共用 handler 會讓「匯出哪一份」變成
	// 一個執行期才決定的分支，讀碼時看不出每份匯出到底送了什麼。
	// 檔名以 .csv 結尾是為了讓 ServeMux 的「完全比對」不會吃掉子路徑
	// （見下方關於比對樣式的說明）。
	mux.HandleFunc("/api/admin/export/users.csv", s.handleAdminExportUsers)
	mux.HandleFunc("/api/admin/export/posts.csv", s.handleAdminExportPosts)
	mux.HandleFunc("/api/admin/export/reports.csv", s.handleAdminExportReports)
	// 批次操作。POST 才有語意，路由層掛一個空 handler 是為了讓非 POST 得到
	// 405 而不是落到 catch-all 的 404（「這條路由存在但方法不對」比
	// 「沒有這條路由」誠實）。
	mux.HandleFunc("/api/admin/batch/tags", s.handleAdminBatchTags)
	mux.HandleFunc("/api/admin/batch/status", s.handleAdminBatchStatus)
	// Session 管理。刻意**不**掛在 rateLimitAllMethods 下，理由同其他
	// /api/admin/*：限流它的唯一效果是在「有人疑似被盜帳號、需要立刻
	// 強制登出」的那一刻把後臺打不開。
	mux.HandleFunc("/api/admin/sessions", s.handleAdminSessions)
	mux.HandleFunc("/api/admin/sessions/revoke", s.handleAdminRevokeSessions)
	// 站內公告。公開的那一支刻意放在 /api/forum/ 下（與其他公開讀取同一個
	// 命名空間），而且**不掛限流**：它是每個頁面載入都會打一次的低成本查詢，
	// 限流它的唯一效果是「公告機制壞掉時連診斷都做不了」。
	mux.HandleFunc("/api/forum/announcement", s.handleForumAnnouncement)
	mux.HandleFunc("/api/admin/announcements", s.handleAdminAnnouncements)
	mux.HandleFunc("/api/admin/announcements/", s.handleAdminAnnouncement)
	// IP 封鎖名單。刻意**不**掛在 rateLimitAllMethods 下：那會讓管理員在
	// 處理一個正在發生的濫用時被自己正在用的功能擋住。
	mux.HandleFunc("/api/admin/blocks", s.handleAdminBlocks)

	// 下列 /forum/* 靜態路由都同時註冊「不帶尾斜線」與「帶尾斜線」兩個樣式，
	// 並在 handler 內手動比對 r.URL.Path。這是 Go 1.22+ ServeMux 語意的必然結果：
	// 樣式不含尾斜線時為「完全比對」，所以 "/forum/login" 不會匹配 "/forum/login/"；
	// 而含尾斜線的樣式是「子樹比對」，"/forum/login/" 會匹配該前綴下的所有路徑。
	// 若不手動比對，註冊子樹樣式就等於把 /forum/login/anything 也導到同一個
	// HTML。實測 ServeMux 的匹配與註冊順序無關，只看哪個樣式更具體，因此
	// 把比對邏輯留在 handler 裡是唯一不依賴 mux 內部規則的寫法。
	mux.HandleFunc("/forum-manifest.json", s.handleForumManifest(frontendDir))
	mux.HandleFunc("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		// Service Worker 的作用域預設為「註冊它的路徑所在的目錄」。這支檔案
		// 掛在根目錄 /service-worker.js，預設作用域已經是全站 /，所以這行
		// 是明確宣告而非補救；保留它是為了讓意圖不必靠閱讀者自行推導。
		w.Header().Set("Service-Worker-Allowed", "/")
		http.ServeFile(w, r, filepath.Join(frontendDir, "service-worker.js"))
	})
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		// 這個比對在目前的註冊方式下其實不會命中："/admin" 是完全比對樣式，
		// "/admin/" 根本不會進來（實測會落到最後的 catch-all）。保留它是防禦性
		// 寫法 —— 若日後有人把樣式改成子樹比對，未定義的子路徑仍會被擋成 404。
		if r.URL.Path != "/admin" && r.URL.Path != "/admin/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "admin.html"))
	})
	mux.HandleFunc("/admin/forum", func(w http.ResponseWriter, r *http.Request) {
		// 論壇管理後台。同樣是防禦性比對；這三個 /admin 樣式彼此不前綴重疊
		// （/admin/forum 與 /admin/forum-report 都是完全比對），不會互相吃掉。
		if r.URL.Path != "/admin/forum" && r.URL.Path != "/admin/forum/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "forum-admin.html"))
	})
	mux.HandleFunc("/admin/forum-report", func(w http.ResponseWriter, r *http.Request) {
		// 檢舉管理頁。
		if r.URL.Path != "/admin/forum-report" && r.URL.Path != "/admin/forum-report/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "forum-report.html"))
	})
	mux.HandleFunc("/admin/monitor", func(w http.ResponseWriter, r *http.Request) {
		// 監控儀表板。防禦性比對與另外兩個 /admin 樣式相同：這些樣式是
		// 「完全比對」，/admin/monitor/ 不會進來（會落到 catch-all 的靜態
		// 檔處理而拿到 404），保留比對是為了日後有人改成子樹比對時，未定義
		// 的子路徑仍然被擋成 404。
		if r.URL.Path != "/admin/monitor" && r.URL.Path != "/admin/monitor/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "forum-monitor.html"))
	})
	mux.HandleFunc("/admin/log", func(w http.ResponseWriter, r *http.Request) {
		// 管理員操作稽核紀錄頁。防禦性比對同 /admin/monitor。
		if r.URL.Path != "/admin/log" && r.URL.Path != "/admin/log/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "audit-log.html"))
	})
	mux.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		// 內容趨勢統計頁。防禦性比對同 /admin/monitor。
		if r.URL.Path != "/admin/stats" && r.URL.Path != "/admin/stats/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "forum-stats.html"))
	})
	mux.HandleFunc("/admin/export", func(w http.ResponseWriter, r *http.Request) {
		// 匯出與批次操作頁。防禦性比對同 /admin/monitor。
		// 路由名刻意不含「csv」：/admin/export 是頁面，而 /api/admin/export/*.csv
		// 是下載。兩者若同名，ServeMux 的最長前綴比對會讓頁面路由吃掉下載
		// 路由（或反過來），症狀是下載得到一個 HTML 頁。
		if r.URL.Path != "/admin/export" && r.URL.Path != "/admin/export/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "export.html"))
	})
	mux.HandleFunc("/admin/sessions", func(w http.ResponseWriter, r *http.Request) {
		// 登入與 Session 管理頁。防禦性比對同 /admin/monitor。
		if r.URL.Path != "/admin/sessions" && r.URL.Path != "/admin/sessions/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "sessions.html"))
	})
	mux.HandleFunc("/admin/blocks", func(w http.ResponseWriter, r *http.Request) {
		// IP 封鎖名單頁。防禦性比對同 /admin/monitor。
		if r.URL.Path != "/admin/blocks" && r.URL.Path != "/admin/blocks/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "blocks.html"))
	})
	mux.HandleFunc("/admin/announcements", func(w http.ResponseWriter, r *http.Request) {
		// 站內公告管理頁。防禦性比對同 /admin/monitor。
		if r.URL.Path != "/admin/announcements" && r.URL.Path != "/admin/announcements/" {
			http.NotFound(w, r)
			return
		}
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "announcements.html"))
	})
	assetServer := http.FileServer(http.Dir(frontendAssetsRoot(frontendDir)))
	mux.Handle("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 僅 GET 掛 immutable：建置工具產出的資產檔名帶內容 hash，內容變了
		// 檔名就會變，所以可以安心地宣告「一年內不會變」。非 GET 請求不該
		// 帶著這份快取標頭 —— 萬一之後有其他方法走到這裡，標頭語意就不對了。
		if r.Method == http.MethodGet {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		// StripPrefix 讓底層 FileServer 以 assets/ 之後的路徑去對應磁碟目錄。
		http.StripPrefix("/assets/", assetServer).ServeHTTP(w, r)
	}))
	// 登入頁同時註冊帶尾斜線與不帶兩個樣式：前者的子樹比對讓 "/forum/login/"
	// 進得來，handleForumLoginPage 內的列舉再把未定義的子路徑擋成 404。
	mux.HandleFunc("/forum/login", s.handleForumLoginPage(frontendDir))
	mux.HandleFunc("/forum/login/", s.handleForumLoginPage(frontendDir))
	// 論壇首頁與他人個人頁同為公開頁面，不掛認證。
	mux.HandleFunc("/forum", s.handleForumPage(frontendDir))
	mux.HandleFunc("/forum/others-profile", s.handleForumPage(frontendDir))
	mux.HandleFunc("/forum/others-profile/", s.handleForumPage(frontendDir))
	// 其餘 /forum/* 頁面（新增文章、個人資料等）需要登入。此處掛最後只是閱讀
	// 順序：ServeMux 取最具體的樣式，所以 "/forum/" 子樹樣式不會蓋掉上面
	// 明確註冊的公開頁面，註冊先後並不影響結果。
	mux.HandleFunc("/forum/", s.requireLogin(s.handleForumPage(frontendDir)))
	// /asset/ 是給後台使用的原始素材（圖示等），路徑固定相對於 backend 的
	// 上層目錄，不隨前端建置輸出位置變動。
	mux.Handle("/asset/", http.StripPrefix("/asset/", http.FileServer(http.Dir(filepath.Join("..", "frontend", "asset")))))

	//  "/" 是 catch-all 路由。位置無關緊要（匹配只看具體程度），"/" 是最不
	// 具體的樣式，所以前面沒被認領的路徑自然會落到這裡。
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 根路徑與 /index.html 是入口，一律導到論壇首頁，讓未登入者從
		// requireLogin 的 303 之後有明確的落點。
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			// 其餘路徑交給 safeStaticFileServer：它會擋掉設定檔、副檔名白名單
			// 以外的檔案並設定快取標頭，比直接暴露目錄安全。
			safeStaticFileServer(frontendDir).ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, "/forum", http.StatusFound)
	})

	// 為什麼是 SecurityHeaders 在最外、Refresh 在其內、Logging 在最內：
	//   SecurityHeaders 必須包在 mux 外面：它設定的是「這份回應該帶哪些
	//   標頭」，必須在 Refresh 寫入 Set-Cookie 之前就決定好，否則標頭的
	//   設定時機會落在 cookie 之後、而快取標頭之內，反而讓它只在部分路徑
	//   生效。放在最外層也確保 JSON、靜態資產與 Service Worker 全部一致。
	//   Refresh 必須包在 mux 外面：sliding expiration 要在「任何」回應（包含
	//   304、302、500）被寫出之前就補上 Set-Cookie，若放進 mux 內部，它只能
	//   看到自己那幾條路由的回應，其餘路徑的 cookie 就永遠不會被更新。
	//   LoggingMiddleware 則必須包在 Refresh 外面，這樣它的 responseWriter 才
	//   包住整棵樹，能記到真正寫出的最終 status code，且 duration 涵蓋了
	//   session 存取的耗時。它同時需要 s.sessions.ResolveUser 辨識使用者，
	//   而該函式讀的是 cookie + Redis，與 Refresh 使用的是同一份 session 狀態。
	//
	// 兩個變數不可共用同一個名稱：Go 的閉包捕捉的是「變數」而不是「當下的值」，
	// 若把 Refresh 的閉包指派回 logged 本身，閉包內的 logged.ServeHTTP 就會
	// 指向自己，形成無限遞迴並在第一個請求就 stack overflow。
	// 這個錯誤不會被編譯器抓到，只有真的送出請求才會爆，因此以不同名字
	// 分開三層是刻意的防呆。
	//
	// metricsMiddleware 的位置：LoggingMiddleware 的內側、mux 的外側。
	//   - 在 LoggingMiddleware 內側 → 沿用它已經包好的 StatusWriter，不必
	//     再寫一份只為取得狀態碼的包裝（見 monitoring.go 的說明）。
	//   - 在 mux 外側 → 每個請求（含靜態資產與 404）都會被計入，且計到的
	//     狀態碼是「mux 最終寫出的那一個」，包含 catch-all 靜態檔的結果。
	observed := s.metricsMiddleware(mux)
	logged := logger.LoggingMiddleware(observed, s.sessions.ResolveUser)
	refreshed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 滑動式過期：每個請求都延長一次 session TTL，並重寫 cookie，
		// 使用者持續使用就不會被登出。
		s.sessions.Refresh(r, w)
		logged.ServeHTTP(w, r)
	})
	// 安全標頭套在最外層，因此 HTML、JSON、靜態資產與 Service Worker
	// 全部一致。只需要安全標頭的測試可直接呼叫 withSecurityHeaders，
	// 不必走完整條 Handler()（後者需要 session 與 DB）。
	//
	// 各頁 HTML 的 <style> 區塊雜湊在這裡算：frontendDir 上面才剛解析出來，
	// 而 style-src 沒有 'unsafe-inline'，沒有這組雜湊那些區塊會被整片擋下。
	return s.withSecurityHeaders(refreshed, inlineStyleHashes(frontendDir))
}
