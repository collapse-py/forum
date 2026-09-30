/*
config 負責把後端的設定檔（純文字 key=value）解析成型別化的 Config 結構。

【對外介面】
Config 型別：設定值容器，以「值」傳遞，欄位全部公開且沒有 getter。
Load(path)：讀檔、逐行解析、套用預設值後回傳 Config。
(*Config).IsAdminEmail：管理員白名單比對，供 OAuth 回呼判定是否為管理員。
本套件沒有其他匯出符號，也不註冊任何 HTTP 路由。

【主要依賴】
僅使用標準函式庫（os、bufio、strconv、strings、time），不依賴任何第三方套件，
因此可以在 main 的最早期、在 logger 尚未就緒之前安全呼叫。

【設定檔格式】
逐行 key=value，以第一個 "=" 切分，所以值本身可以含 "="（例如 MySQL DSN 的參數串）。
"#" 與 ";" 開頭為註解，空行忽略，兩側空白一律 TrimSpace。
清單型欄位以半形逗號分隔。未知的 key 被靜默忽略，方便各環境共用同一份範本檔。

【關鍵設計決策與限制】
解析風格刻意寬鬆：格式錯誤的數值會丟掉 strconv.Atoi 的 error 並沿用原值（多為 0），
再由 applyDefaults 補上合理預設。代價是「打錯字」與「沒設定」無法區分，例如
SESSION_EXPIRE_HOURS 誤寫成 abc 時會安靜地變成 72 小時而不是報錯。

設定是一次性載入的快照，沒有熱更新機制；修改檔案必須重啟行程才會生效。

憑證（GOOGLE_CLIENT_SECRET、FILES_SERVER_TOKEN、REDIS_PASSWORD、DB_DSN 中的帳密）
以純文字存放在設定檔，程式端不做任何加密或解密。部署時必須以檔案權限保護，
若要走環境變數注入，目前只有 FILES_SERVER_TOKEN 一個欄位支援。

applyDefaults 內的預設值以「本機開發能跑起來」為目標，不代表正式環境的安全建議值。

平台相依性：本套件完全不碰檔案路徑的組法，path 由呼叫端（main）以 filepath.Join
組成後原樣轉交，因此本檔在 Windows 與 POSIX 上行為一致。
*/
package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 是後端所有可調設定的單一容器，以「值」傳遞給各個套件。
//
// 設計取向：欄位全部公開，沒有 getter 與驗證方法，解析與兜底集中在 Load 與
// applyDefaults。優點是新增設定只要加一個欄位與一個 switch case；缺點是任何
// 拿到 Config 的套件都能改寫它，且從型別上無法區分「已套用預設值」與「零值」。
//
// 每個欄位後方以行內註解標示對應的設定檔鍵名；各欄位的語意、單位與使用位置
// 逐條列在下方。
//
// 【欄位語意】
// CookieName 是 session cookie 的名稱，session.Manager 讀寫 cookie 時使用。瀏覽器
// 的 cookie 依網域而非路徑作用，因此帶站台前綴（預設 "forum_forum"）以免與同網域的
// 其他應用互相覆寫使用者的登入狀態。
// SessionExpire 是 session 的存留時間，同時作為 Redis key 的 TTL 與 cookie 的
// Expires，兩者都由 session.Manager 套用，所以滑動續期會一併延長。設定檔單位為
// 小時，載入後轉為 time.Duration。
// CookieSecure 決定 session cookie 是否加 Secure 旗標。對外以 HTTPS 服務時應為
// true；以 http 供本機測試時必須為 false，否則瀏覽器不會回傳該 cookie，症狀是
// 「明明登入了卻一直顯示未登入」而且沒有明確錯誤訊息。
// ServerPort 是 net/http 的監聽位址，形如 "host:port"，開頭的「:」代表綁定所有
// 網路介面卡，實際對外服務通常再由反向代理做 TLS 終結。解析時不做格式驗證，
// 寫錯要到 ListenAndServe 才會失敗。
// PublicBaseURL 是本站對外的根網址，載入時去掉尾端「/」以便之後直接串接路徑。
// 程式內目前只有 applyDefaults 會讀它，用途是作為 TrustedOrigins 的預設來源。
// TrustedOrigins 是允許的來源白名單，由 httpapi.isTrustedOrigin 以 strings.EqualFold
// 比對來源的 Origin 標頭，缺少 Origin 時改比對 Referer。它是防止跨站請求偽造
// （CSRF）的主要防線，因此只應填真正會發出變更請求的來源。
// RateLimitRequests 是單一用戶端 IP 在 RateLimitWindow 內，被允許的「內容寫入」
// 請求次數（發文、留言、按讚、檢舉）。讀取端點刻意不受限流：貼文列表與留言
// 是匿名訪客也要能讀的公開內容，若一併限流，使用者按「載入更多」就會被擋。
// 套用位置見 httpapi.Server 的 writeRateLimiter。
// RateLimitWindow 是 RateLimitRequests 的滑動視窗長度。實作是記憶體內的時間戳
// 切片，不落地也不跨執行緒共享，因此重啟後計數會歸零。
//
// 為什麼限流分成三組而不是共用一個額度：不同端點的單次成本差異極大。
// 圖片上傳會呼叫外部檔案伺服器並在 Redis 建立存取 token，是最貴的操作；
// OAuth 跳轉會產生站外導向，次之；而留言只是寫一行 MySQL。把三者放在同一個
// 額度裡，結果必然是「要嘛放行上傳轟炸、要嘛把留言一起擋掉」。分組之後，
// 每組的額度可以各自貼近該端點的真實成本。
//
// 已知限制：以 IP 為 key 代表同一個 NAT 或辦公室出口後的所有使用者共享一份
// 額度。對「需登入才能發文」的論壇而言影響有限（正常使用不會在分鐘內連續
// 寫入達到預設值）；若要改以登入身分為 key，見 ratelimit.go 的 Middleware 說明。
// RateLimitUploadRequests 是圖片上傳端點（POST /api/forum/images）的額度。
// 刻意比寫入額度更緊：每次上傳都會讓外部服務建立一筆真實的儲存與 Redis token。
// RateLimitUploadWindow 是其滑動視窗長度。
// RateLimitAuthRequests 是 OAuth 端點（/auth/google、/auth/callback）的額度。
// 作用是擋掉對授權流程的轟炸與重導向洗版，不是防爆破密碼（本站沒有密碼）。
// RateLimitAuthWindow 是其滑動視窗長度。
// DbDSN 是 MySQL 連線字串，常見形式為
// user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local。
// parseTime=True 是必要的：程式以 time.Time 掃描 DATETIME 欄位，缺少這個參數時
// 驅動會回傳 []byte 而導致掃描失敗。此字串內含帳密，屬敏感資訊。
// DbMaxOpenConns 是連線池的最大開啟連線數。需與 MySQL 的 max_connections 一起評估，
// 否則池會一次吃滿資料庫的連線額度；多執行緒部署時應以「實例數 × 此值」估算。
// DbMaxIdleConns 是連線池保留的閒置連線數上限，範例設定為 MaxOpenConns 的四分之一
// （20 / 5）。設為 0 等同關閉 keep-alive，每個請求都要重新握手，高流量下延遲會
// 顯著上升。
// DbConnMaxLifetimeMinutes 是單一連線的最大存活時間。作用是讓重連分散在時間軸上，
// 而不是讓所有連線在資料庫端或負載平衡器的逾時時刻同時被淘汰再全部重連。
// FilesServerURL 是後端在後台要連線的檔案伺服器位址（上傳貼文附圖時呼叫），
// 通常是內網位址，瀏覽器不會也不應直接連到它。
// FilesServerPublicURL 是同一台檔案伺服器對外服務的位址。程式會把檔案伺服器回傳的
// 內網 URL 改寫成這個網域後才寫進資料庫，換部署環境時既有貼文的圖片才不會全部失效。
// FilesServerToken 是上傳圖片時放在 X-Upload-Token 標頭的共用密鑰。它是唯一支援
// 環境變數後備的欄位，因為它是唯一應該在部署時注入、而不是寫進版控檔案的設定。
// 留空時上傳請求不會帶驗證標頭。
// MediaTokenKeyPrefix 是論壇圖片存取 token 在 Redis 中的 key 前綴，必須與 session
// 使用的 "forum:session:" 區隔，否則兩種資料會互相覆蓋。
// MediaTokenTTLSecs 是圖片存取 token 的存活秒數。在這段時間內持有 token 的人都能
// 讀取該圖片，所以值應盡量貼近實際瀏覽行為。程式內兜底預設 2592000（30 天）只適合
// 開發環境；範例設定檔用 60 秒，並在使用者離開頁面時呼叫釋用 API 提前刪除。
// RedisAddr 是 Redis 位址（host:port），Redis 與 MySQL 假設與本服務同側部署。
// RedisPassword 是 Redis 的 AUTH 密碼，留空代表不驗證。同一個 Redis 也存放 session
// token，未啟用密碼等同把登入權杖暴露給能連到該主機的任何程序。
// RedisDB 是 Redis 的邏輯資料庫編號（0 到 15）。與其他應用共用 Redis 實例時，用不同
// 的 DB 編號比共用編號再靠 key 前綴隔離更不容易互相干擾，例如誤觸 FLUSHDB。
// GoogleClientID 是 Google Cloud Console 核發的 OAuth 2.0 Client ID。技術上屬公開
// 資訊（會出現在授權網址中），但仍不應寫死在版控檔案裡。
// GoogleClientSecret 是對應的 Client Secret，只在伺服器端交換 token 時使用，屬高敏感
// 憑證。程式端以明文存放，不做加密。
// GoogleRedirectURL 是 OAuth 授權回呼網址，必須與 Google Console 登記的授權網址
// 逐字相符（含結尾斜線與 http/https），不符時 token 交換會直接以 redirect_uri_mismatch
// 失敗，因此解析時刻意不去除尾端斜線。
// AllowedAdminEmail 是管理員白名單。登入時以此比對 Google 回傳的 email，命中者直接
// 在 session 內標記為管理員，不需資料庫中的角色資料；因此新增一項等同授予後臺權限，
// 屬高風險操作。逗號兩側可以有空白，解析時會去除。
// LogLevel 是日誌門檻，門檻低於此值的訊息完全不輸出；無法辨識的值由 logger 退回
// INFO。範例設定檔的正式環境使用 WARN，可減少磁碟寫入但也會失去錯誤前的診斷線索。
// LogFile 是日誌輸出檔案路徑，以附加（O_APPEND）模式開啟，檔案不存在時會被建立，
// 因此父目錄必須已存在。留空時只輸出到 stdout。
// LogFormat 是日誌格式。值為 "json" 時輸出結構化 JSON，便於日誌聚合系統按欄位查詢；
// 其他任何值（含預設 "text"）都輸出純文字行。
// ForumName 是站台對外顯示的名稱（例如 "forum 論壇"）。它是部署者的事實而不是
// 程式內的常數：頁面標題、導覽列標誌、後臺麵包屑、PWA manifest 與各頁 HTML 的
// <title> 全部由它取代（見 httpapi/site.go 的樣板機制）。留空時取預設值，因此
// 一份沒有這三行的舊設定檔仍然會得到可用的站名。
// ForumShortName 是標誌用的短名（例如 "forum"），用在空間有限的導覽列與後臺 rail
// 標記。它與 ForumName 分開是因為兩者長度限制不同：短名要塞進一顆膠囊按鈕大小的
// 方塊，長度必須可預期。留空時自動沿用 ForumName，此時站名直接出現在標誌位。
// ForumDescription 是 PWA manifest 的 description 欄位，也是瀏覽器「安裝」
// 對話框與桌面捷徑 tooltip 會用到的文字。留空時沿用 ForumName。
// ESURL 是 Elasticsearch 的位址（例如 http://192.168.66.5:9200）。留空代表
// 不啟用全文搜尋：搜尋會退回 MySQL 的 LIKE 比對（見 httpapi.search.go），
// 而 ES 的寫入、刪除與 /healthz 探測完全不發生。刻意不提供預設值 —— 內網位址
// 換環境就會失效，錯設定一個預設值只會讓人以為有搜尋功能其實打到別台機器。
// ESIndex 是貼文索引名稱。沿用舊版主站 config.conf 的 ES_INDEX 一詞，預設
// forum_posts；它必須與 ES 上既有的其他索引（例如舊功能的 history_posts）
// 不同名，否則兩種資料會混在同一个索引裡互相覆蓋。
type Config struct {
	CookieName               string        // COOKIE_NAME
	SessionExpire            time.Duration // SESSION_EXPIRE_HOURS（設定檔單位：小時）
	CookieSecure             bool          // COOKIE_SECURE
	ServerPort               string        // SERVER_PORT（net/http 的 "host:port" 形式）
	PublicBaseURL            string        // PUBLIC_BASE_URL
	TrustedOrigins           []string      // TRUSTED_ORIGINS（逗號分隔）
	RateLimitRequests        int           // RATE_LIMIT_REQUESTS
	RateLimitWindow          time.Duration // RATE_LIMIT_WINDOW_SECONDS（設定檔單位：秒）
	RateLimitUploadRequests  int           // RATE_LIMIT_UPLOAD_REQUESTS
	RateLimitUploadWindow    time.Duration // RATE_LIMIT_UPLOAD_WINDOW_SECONDS（設定檔單位：秒）
	RateLimitAuthRequests    int           // RATE_LIMIT_AUTH_REQUESTS
	RateLimitAuthWindow      time.Duration // RATE_LIMIT_AUTH_WINDOW_SECONDS（設定檔單位：秒）
	DbDSN                    string        // DB_DSN
	DbMaxOpenConns           int           // DB_MAX_OPEN_CONNS
	DbMaxIdleConns           int           // DB_MAX_IDLE_CONNS
	DbConnMaxLifetimeMinutes time.Duration // DB_CONN_MAX_LIFETIME_MINUTES（設定檔單位：分鐘）
	FilesServerURL           string        // FILES_SERVER_URL（後端後台連線用，通常是內網）
	FilesServerPublicURL     string        // FILES_SERVER_PUBLIC_URL（寫進貼文給瀏覽器用）
	FilesServerToken         string        // FILES_SERVER_TOKEN（後備值來自同名環境變數）
	MediaTokenKeyPrefix      string        // MEDIA_TOKEN_KEY_PREFIX
	MediaTokenTTLSecs        int           // MEDIA_TOKEN_TTL_SECONDS（設定檔單位：秒）
	RedisAddr                string        // REDIS_ADDR
	RedisPassword            string        // REDIS_PASSWORD
	RedisDB                  int           // REDIS_DB
	GoogleClientID           string        // GOOGLE_CLIENT_ID
	GoogleClientSecret       string        // GOOGLE_CLIENT_SECRET
	GoogleRedirectURL        string        // GOOGLE_REDIRECT_URL（須與 Google Console 登記值逐字相符）
	AllowedAdminEmail        []string      // ALLOWED_ADMIN_EMAIL（逗號分隔）
	ESURL                    string        // ES_URL（留空＝不啟用 Elasticsearch，搜尋退回 MySQL LIKE）
	ESIndex                  string        // ES_INDEX（預設 forum_posts）
	LogLevel                 string        // LOG_LEVEL
	LogFile                  string        // LOG_FILE
	LogFormat                string        // LOG_FORMAT（"json" 或 "text"）
	ForumName                string        // FORUM_NAME（站台顯示名稱，預設 "forum 論壇"）
	ForumShortName           string        // FORUM_SHORT_NAME（標誌短名，留空＝沿用 ForumName）
	ForumDescription         string        // FORUM_DESCRIPTION（manifest 說明，留空＝沿用 ForumName）
}

// Load 讀取並解析 path 指向的設定檔，回傳套用預設值後的 Config。
//
// path 必須是檔案系統上的完整路徑，由呼叫端組裝（main 使用相對於工作目錄的
// "config/config.conf"）。平台相依的路徑分隔符組法不屬於本套件的責任。
//
// 錯誤條件只有兩種：os.Open 失敗（檔案不存在或權限不足），以及掃描過程中
// scanner.Err() 非 nil（讀取途中發生 I/O 錯誤，或單行超過 Scanner 預設的 64KB 上限）。
// 個別設定項目的格式錯誤不會產生錯誤，而是被忽略並沿用預設值。
//
// 回傳值：即使在錯誤路徑上，cfg 也是可讀的結構，但其欄位只帶有 Load 開頭預先
// 寫入的三組限流設定，其餘皆為零值，applyDefaults 並未執行，因此呼叫端必須把
// err != nil 視為致命錯誤。
//
// 副作用：讀取檔案並在返回時關閉其句柄。不寫檔案、不發 HTTP 請求、不碰資料庫。
// 非併行安全：設計上只應在啟動階段呼叫一次，回傳的 Config 之後視為唯讀。
func Load(path string) (Config, error) {
	// 預先寫入即使設定檔完全不存在也能運作的預設值。三組限流設定在
	// applyDefaults 中都沒有對應的兜底，所以特別在這裡就先給值；
	// 即使設定檔把它們寫成 0，httpapi.NewRateLimiter 仍會再兜一次底。
	//
	// 三組的預設值刻意不同，反映各端點的真實成本：上傳最貴因此額度最緊，
	// 登入居中，內容寫入最便宜因此給得較寬。
	var cfg Config
	cfg.RateLimitRequests = 10
	cfg.RateLimitWindow = time.Minute
	cfg.RateLimitUploadRequests = 5
	cfg.RateLimitUploadWindow = time.Minute
	cfg.RateLimitAuthRequests = 10
	cfg.RateLimitAuthWindow = time.Minute

	file, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	// 必須確保句柄釋放。defer 在函式返回時執行，包含掃描迴圈中途 return 的情況；
	// 這裡是單純的讀取檔案且沒有其他資源競爭，因此 defer 是正確且安全的選擇。
	defer file.Close()

	// 用 bufio.Scanner 而非 io.ReadAll：設定檔是「一行一組設定」的形式，逐行處理
	// 即可，不需要把整份檔案留在記憶體中。
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 支援兩種註解符號：# 常見於 shell 風格的設定檔，; 常見於 ini 風格。
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// 只切第一個 "="。值裡經常含有 "="，例如 MySQL DSN 的
		// "?charset=utf8mb4&parseTime=True"，用 strings.Split 會把值截斷成後半段。
		// 沒有 "=" 的行視為雜訊（常是忘記刪除的說明文字）直接略過。
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// key 比對大小寫敏感且必須完全一致。沒有 default 分支是刻意的：
		// 新增或移除設定項時，未知的 key 會被靜默忽略，讓不同環境能共用同一份範本檔。
		switch key {
		case "COOKIE_NAME":
			cfg.CookieName = val
		case "SESSION_EXPIRE_HOURS":
			// Atoi 的錯誤刻意忽略：非數字會得到 0，之後由 applyDefaults 補成 72 小時。
			hours, _ := strconv.Atoi(val)
			cfg.SessionExpire = time.Duration(hours) * time.Hour
		case "COOKIE_SECURE":
			// 走 parseBool 而非 strconv.ParseBool，才能接受 true/1/yes/on 這些寫法。
			cfg.CookieSecure = parseBool(val)
		case "SERVER_PORT":
			// 原樣轉交，不做任何檢查：net/http 要求 "host:port" 形式，
			// 寫錯時會在 ListenAndServe 才以錯誤回應。
			cfg.ServerPort = val
		case "PUBLIC_BASE_URL":
			// 去掉尾端 "/"：後續可能直接用字串相加組出完整路徑，
			// 留著尾斜線會產生 "//" 這種語意不對但仍可請求的網址。
			cfg.PublicBaseURL = strings.TrimRight(val, "/")
		case "TRUSTED_ORIGINS":
			cfg.TrustedOrigins = parseList(val)
		case "RATE_LIMIT_REQUESTS":
			// 只接受正數；其他一律保留開頭預先寫入的預設值。
			if limit, ok := parsePositiveInt(val); ok {
				cfg.RateLimitRequests = limit
			}
		case "RATE_LIMIT_WINDOW_SECONDS":
			// 解析失敗或非正值時保留開頭預先寫入的一分鐘（見 parsePositiveSeconds）。
			if window, ok := parsePositiveSeconds(val); ok {
				cfg.RateLimitWindow = window
			}
		case "RATE_LIMIT_UPLOAD_REQUESTS":
			if limit, ok := parsePositiveInt(val); ok {
				cfg.RateLimitUploadRequests = limit
			}
		case "RATE_LIMIT_UPLOAD_WINDOW_SECONDS":
			// 注意不可寫成 `cfg.X, _ = parsePositiveSeconds(val)`：那會在
			// 解析失敗時把欄位覆寫成 0，等於親手把「視窗為 0」這個會讓限流
			// 失效的值寫進設定。必須先看 ok 再決定要不要賦值。
			if window, ok := parsePositiveSeconds(val); ok {
				cfg.RateLimitUploadWindow = window
			}
		case "RATE_LIMIT_AUTH_REQUESTS":
			if limit, ok := parsePositiveInt(val); ok {
				cfg.RateLimitAuthRequests = limit
			}
		case "RATE_LIMIT_AUTH_WINDOW_SECONDS":
			if window, ok := parsePositiveSeconds(val); ok {
				cfg.RateLimitAuthWindow = window
			}
		case "DB_DSN":
			// 內含帳密；由 SplitN 保證後面的 "=" 與參數串都完整保留下來。
			cfg.DbDSN = val
		case "DB_MAX_OPEN_CONNS":
			cfg.DbMaxOpenConns, _ = strconv.Atoi(val)
		case "DB_MAX_IDLE_CONNS":
			cfg.DbMaxIdleConns, _ = strconv.Atoi(val)
		case "DB_CONN_MAX_LIFETIME_MINUTES":
			mins, _ := strconv.Atoi(val)
			cfg.DbConnMaxLifetimeMinutes = time.Duration(mins) * time.Minute
		case "FILES_SERVER_URL":
			// 同樣去掉尾斜線，上傳時程式端會再 TrimRight 一次並接上 "/upload"。
			cfg.FilesServerURL = strings.TrimRight(val, "/")
		case "FILES_SERVER_PUBLIC_URL":
			cfg.FilesServerPublicURL = strings.TrimRight(val, "/")
		case "FILES_SERVER_TOKEN":
			// 此處只讀設定檔；環境變數的後備值在 applyDefaults 處理，
			// 讓「設定檔的值優先、環境變數次之」的優先序只寫在一個地方。
			cfg.FilesServerToken = val
		case "MEDIA_TOKEN_KEY_PREFIX":
			cfg.MediaTokenKeyPrefix = val
		case "MEDIA_TOKEN_TTL_SECONDS":
			cfg.MediaTokenTTLSecs, _ = strconv.Atoi(val)
		case "REDIS_ADDR":
			cfg.RedisAddr = val
		case "REDIS_PASSWORD":
			cfg.RedisPassword = val
		case "REDIS_DB":
			cfg.RedisDB, _ = strconv.Atoi(val)
		case "GOOGLE_CLIENT_ID":
			cfg.GoogleClientID = val
		case "GOOGLE_CLIENT_SECRET":
			// 敏感值：以明文載入、不做加密，也不會被寫進任何日誌。
			cfg.GoogleClientSecret = val
		case "GOOGLE_REDIRECT_URL":
			// 不做 TrimRight：Google 對 redirect_uri 的比對是逐字的，
			// 這裡若擅自去掉結尾斜線，反而會造成 redirect_uri_mismatch。
			cfg.GoogleRedirectURL = val
		case "ALLOWED_ADMIN_EMAIL":
			// 逗號分隔的白名單；解析時去除每項前後空白。
			cfg.AllowedAdminEmail = parseList(val)
		case "ES_URL":
			// 去掉尾斜線：es 套件會以 baseURL + "/{index}" 組出路徑，
			// 留著尾斜線會變成 "//forum_posts"（ES 容忍但 log 很難看）。
			cfg.ESURL = strings.TrimRight(val, "/")
		case "ES_INDEX":
			cfg.ESIndex = val
		case "LOG_LEVEL":
			// 不在此處驗證：logger.parseLevel 對無法辨識的值會退回 INFO，
			// 由該處統一處理所有合法與不合法情形。
			cfg.LogLevel = val
		case "LOG_FILE":
			cfg.LogFile = val
		case "LOG_FORMAT":
			cfg.LogFormat = val
		case "FORUM_NAME":
			// 原樣保留：站名可能含中文與空白，解析端沒有任何格式要求。
			cfg.ForumName = val
		case "FORUM_SHORT_NAME":
			cfg.ForumShortName = val
		case "FORUM_DESCRIPTION":
			cfg.ForumDescription = val
		}
	}

	// 迴圈正常結束不代表讀檔成功：Scanner 會把行過長、讀取錯誤等問題延後到
	// Err() 才回報，因此這裡一定要檢查。
	if err := scanner.Err(); err != nil {
		return cfg, err
	}

	// 預設值必須在整份檔案都解析完之後才套用，才能讓「檔案裡最後一次出現的值」
	// 贏過預設值。副作用是使用者無法用空字串停用某個功能：
	// 空字串與未設定在這個格式裡是無法區分的。
	cfg.applyDefaults()
	return cfg, nil
}

// applyDefaults 為「值為空或非正數」的欄位補上預設值。
// 只在 Load 成功解析完整份檔案之後呼叫一次，且必須以指標接收者，
// 才能把補上的值寫回呼叫端持有的 Config。
//
// 判斷條件刻意採用零值（""、nil、<= 0）而不是「這個欄位有沒有出現過」：
// 設定檔格式簡單，不值得為此維護一份「已設定欄位集合」。代價是無法設定
// 零值語意，例如把視窗設為 0 秒或把 cookie 名稱設為空字串都不可能。
func (c *Config) applyDefaults() {
	if c.CookieName == "" {
		// 預設名稱含站台前綴，避免與同網域其他應用共用同一個 cookie 名稱。
		c.CookieName = "FORUM_forum"
	}
	if c.SessionExpire <= 0 {
		// 與範例設定檔的 SESSION_EXPIRE_HOURS=72 一致。session 會在每次請求時
		// 滑動續期，所以這個值是「閒置多久後登入失效」而非登入後的絕對上限。
		c.SessionExpire = 72 * time.Hour
	}
	if c.ServerPort == "" {
		// net/http 需要 "host:port"；以 ":" 開頭代表綁定所有介面。
		c.ServerPort = ":8088"
	}
	if c.PublicBaseURL == "" {
		// 隱含的欄位順序相依：這裡直接使用 c.ServerPort，因此上面的補值必須
		// 先執行，否則會得到 "http://localhost" 這種沒有埠的網址。
		c.PublicBaseURL = "http://localhost" + c.ServerPort
	}
	if len(c.TrustedOrigins) == 0 {
		// 沒有白名單等於把所有來源視為可信，等同關閉 CSRF 防護，
		// 因此至少要給 PublicBaseURL 這一個值。
		c.TrustedOrigins = []string{c.PublicBaseURL}
	}
	if c.FilesServerURL == "" {
		c.FilesServerURL = "http://localhost:7070"
	}
	if c.FilesServerPublicURL == "" {
		// 與內部位址相同代表檔案伺服器直接對外服務，沒有內外網分離。
		c.FilesServerPublicURL = c.FilesServerURL
	}
	if c.FilesServerToken == "" {
		// 唯一從環境變數取值的位置：讓 CI/CD 或容器可以注入密鑰而不必改寫檔案。
		c.FilesServerToken = os.Getenv("FILES_SERVER_TOKEN")
	}
	if c.MediaTokenKeyPrefix == "" {
		// 與 session 的 "forum:session:" 前綴區隔，兩者共用同一個 Redis。
		c.MediaTokenKeyPrefix = "forum:token:"
	}
	if c.MediaTokenTTLSecs <= 0 {
		// 2592000 秒 = 30 天。這個兜底值對正式環境明顯過長，僅為讓開發環境
		// 忘記設定時圖片仍能顯示；正式環境應在設定檔明確設定較短的 TTL。
		c.MediaTokenTTLSecs = 2592000
	}
	if c.RedisAddr == "" {
		// 只連本機 loopback：Redis 與 MySQL 假設與本服務同側部署。
		c.RedisAddr = "127.0.0.1:6379"
	}
	// ESIndex 有預設值但 ESURL 沒有：索引名是本專案的命名慣例（可推導），
	// 位址則是環境相依的事實，只能由部署者提供。兩者一組有、一組無，
	// 正好對應「設定了位址就代表要啟用搜尋」這個判斷。
	if c.ESIndex == "" {
		c.ESIndex = "forum_posts"
	}
	if c.LogLevel == "" {
		// 與 logger 套件的 parseLevel 預設一致。
		c.LogLevel = "INFO"
	}
	if c.LogFormat == "" {
		// 只有 "json" 與 "text" 兩種輸出；其他值都會落到 text 分支。
		c.LogFormat = "text"
	}
	// 站名的三個欄位有相依的兜底順序，因此集中在最後：短名與說明預設都取
	// 「完整站名」，而完整站名必須先有值。順序若調換，短名會變成空字串 ——
	// 那會讓前端標誌整顆消失，而設定檔裡明明有站名。
	if c.ForumName == "" {
		c.ForumName = "FORUM 論壇"
	}
	if c.ForumShortName == "" {
		c.ForumShortName = c.ForumName
	}
	if c.ForumDescription == "" {
		c.ForumDescription = c.ForumName
	}
}

// IsAdminEmail 判斷 email 是否列在 ALLOWED_ADMIN_EMAIL 白名單中。
//
// 比對是大小寫敏感且不做任何正規化（不轉小寫、不去空白）。這是安全的，因為
// Google 回傳的 email 已是穩定的小寫形式，而白名單也應以相同字面書寫；反過來說，
// 白名單若寫成帶空白或大寫的樣子會靜默失效。
//
// 副作用：無，只讀取欄位。回傳值型別是值接收者，代表每次呼叫會複製整個 Config
// （含 slice 的標頭），但因為只讀且結構不大，這點複製成本可以忽略。
func (c Config) IsAdminEmail(email string) bool {
	for _, allowed := range c.AllowedAdminEmail {
		if email == allowed {
			return true
		}
	}
	// 迴圈走完代表白名單沒有命中。
	return false
}

// parseList 解析以半形逗號分隔的清單，回傳去除空白與空項目後的切片。
// 輸入為空字串時回傳 nil（而非長度為 0 的切片），讓呼叫端可以直接用
// len() == 0 判斷「未設定」，也讓 JSON 序列化時輸出 null 而非 []。
func parseList(val string) []string {
	if val == "" {
		return nil
	}
	// 一次切完再逐項清理。使用者常在逗號後加空白，例如
	// "a@example.com, b@example.com"，若不清理會導致比對永遠不成立。
	parts := strings.Split(val, ",")
	// 刻意不預先配置容量：清單通常很短，且預配置會預留 nils 讓程式碼多一步
	// 清理。改用 append 也讓結果的 nil 與空值行為保持一致。
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// 跳過空項目：例如結尾多打一個逗號會產生 ""，若保留下來，
		// 空字串就有可能與某個空的 Origin 比對成功。
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseBool 解析布林設定，容忍常見寫法：1、true、yes、on（不分大小寫、忽略空白）。
//
// 其他任何字串都視為 false，採取「明確為真才成立」的原則。這個方向是刻意的
// 偏嚴選擇：像 COOKIE_SECURE 這種安全相關旗標，打錯字（例如 flase）時寧可讓它
// 失效，也不要因為某種模糊比對而被意外開啟。
func parseBool(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// parsePositiveInt 解析「次數」設定字串，只接受正整數。
//
// 語意與 parsePositiveSeconds 相同：無法使用時回傳 (0, false)，呼叫端必須
// 保留預設值。特別地，0 與負數都不接受 —— 次數為 0 會讓每個請求的計數都
// >= limit，等於整站被封鎖；負數則語意不明。
//
// 兩者分開而不是讓 parsePositiveSeconds 回傳一個泛型數值，是因為單位不同
// （秒 vs 次），混用會讓呼叫端忘記乘上 time.Second。
func parsePositiveInt(val string) (int, bool) {
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// parsePositiveSeconds 把「秒數」設定字串轉成 time.Duration，只接受正整數。
//
// 回傳 (0, false) 表示設定值無法使用（非數字、或 <= 0）。呼叫端必須把這個
// false 當成「保留預設值」，不可把 0 當成合法視窗 —— 視窗為 0 會讓 cutoff
// 等於 now，所有時間戳都被視為過期，限流形同不存在；負值則會讓 cutoff 落在
// 未來，等於「所有請求都在視窗內」，整站會被立刻封鎖。兩個方向都是靜默的
// 嚴重錯誤，因此這個函式寧可拒絕也不猜。
//
// 抽出來是因為 RATE_LIMIT_WINDOW_SECONDS 與兩組新設定的判斷邏輯完全相同，
// 複製三份只會讓未來修正其中一份時漏掉另外兩份。
func parsePositiveSeconds(val string) (time.Duration, bool) {
	seconds, err := strconv.Atoi(val)
	if err != nil || seconds <= 0 {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}
