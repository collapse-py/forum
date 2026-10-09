/*
server.go 是 files_server 的 HTTP 層：路由、兩個端點的處理、CORS 與媒體 token
中介層。

【為什麼獨立成檔】
原本這些路由寫在 main.go 裡的匿名函式，而匿名函式無法被測試直接呼叫 —— 這是整個
模組 0% 覆蓋的直接原因（測試只能對著一個正在監聽的埠送真實請求，那需要先組出
存儲目錄與一個假 Redis）。把它們收進一個可注入的 Server 型別之後，
「token 驗證、上傳限制、副檔名白名單、刪除路徑驗證」四件事都能用 httptest
直接打，不需任何外部相依。

【對外介面（本檔全部匯出符號）】
Server：持有設定、儲存後端與 Redis 用戶端。
NewServer：以設定與後端組出 Server。
(*Server).Handler：組出完整的中介層鏈（尚不監聽）。
resolveDir / isAllowedExt / originAllowed：三個純函式的決策，供測試直接驗證。

（真實的匯出面已經是上面這五個 + main.go 的 loadStorageConfig／NewServer。
其餘「函式」都在 server_test.go 裡補上，因為它們的行為必須被釘住 ——
「exported 的只有這四個」的說法曾經是錯的，而它在測試補齊之前看起來完全合理。）

【路由】

	POST|PUT /upload              上傳檔案，回 {"url":"..."}
	DELETE|POST /delete           刪除檔案（url 查詢參數或表單欄位）
	GET|HEAD /files/*             本機模式的靜態檔案，經媒體 token 中介層過濾
	其餘路徑                      404（本機模式）

【主要依賴】
標準函式庫、github.com/redis/go-redis/v9（只為了 Exists 與 Ping）。

【關鍵設計決策】
 1. 上傳與刪除共用同一個 upload.token 驗證，實作只有一份（validUploadToken）。
    兩者共用是刻意的：token 的用途是「呼叫者是後端」，而能上傳的人自然也能刪，
    拆成兩個 token 只會多出「忘記其中一邊」的可能。
 2. 儲存路徑的驗證在 delete 端點裡（dir 必須是 "files"、檔名取 Base），
    而不放在 storageBackend 裡。理由是介面不該知道本機目錄的結構，而這個服務
    的「合法路徑」只有一種形狀，放進介面等於要求每個後端各自重寫同一段驗證。
 3. 媒體 token 只擋 GET/HEAD 且只擋 /files/ 前綴：上傳與刪除是後端對後端的
    呼叫，用的是另一個 token 通道。把它們一起擋掉會讓後端的上傳也 401。
*/
package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// mediaTokenTTL 是單次 Redis 探查的期限。
//
// 刻意給一個短值：這個中介層在**每個靜態檔案請求**上都要查一次 Redis，而靜態
// 檔案是使用者停留最久的資源。Redis 卡住時，較長的期限會讓整個媒體服務一起
// 停擺（每個請求各自等滿），較短的則讓它快速回 503 — 症狀相同但恢復得快得多。
const mediaTokenTTL = 2 * time.Second

// mediaTokenStore 是這個中介層唯一需要 Redis 的能力。
//
// 刻意宣告成介面（而不是直接用 *redis.Client）：這讓測試可以塞一個假的進來，
// 驗證「token 不存在時回 401」與「Redis 故障時回 503」兩條分支 —— 兩者在真實
// Redis 上要分別靠「不寫入 key」與「指一個壞掉的位址」才能達成，後者根本不可靠。
type mediaTokenStore interface {
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
}

// 媒體 token 驗證失敗的計數與「上次寫日誌的時間」。
//
// 用 atomic 而不是 mutex：這個中介層在每個靜態檔案請求上都會碰到，而 mutex 會讓
// 所有圖片請求序列化。兩個值是分開的兩個變數而非一個 struct，為的是讓「讀時間」
// 與「加計數」不會有半更新的狀態。
var (
	mediaTokenFailures    atomic.Int64
	lastMediaTokenFailure atomic.Value // time.Time
)

// initLastFailure 讓 lastMediaTokenFailure 一開始就持有正確的型別。
//
// atomic.Value.Store 要求型別一致，而零值的 atomic.Value 第一次 Store 會固定型別。
// 若把這個初始化刪掉，第一次 Store 時的 time.Time 就成為固定的型別，看似可行 ——
// 但任何「先 Load 再 Store」的寫法都會panic。讓它一開始就正確。
func init() {
	lastMediaTokenFailure.Store(time.Time{})
}

// Server 持有這個服務處理請求所需的全部相依。
//
// 刻意用具名欄位而非 package 層的變數：原本的 redisClient 是 package 級別的
// 全域變數，由 main 在啟動時賦值，而中介層在建立時讀它。那個組合有兩個問題 ——
// 「賦值」與「讀取」分屬兩個檔案，而測試完全無法控制它。
type Server struct {
	cfg   *Config
	store storageBackend
	// redis 為 nil 代表「不做媒體 token 驗證」。這與 cfg.Redis.PublicFiles 為
	// true 是同義的兩種寫法，但來源不同：前者是「連不上所以降級」，後者是
	// 「部署者刻意公開檔案」。刻意都導向同一條程式路徑，而不是分成兩個分支 ——
	// 分成兩條的風險是將來只改到其中一條。
	redis mediaTokenStore
}

// NewServer 以設定與儲存後端組出 Server。redis 可為 nil（見 Server.redis）。
func NewServer(cfg *Config, store storageBackend, redis mediaTokenStore) *Server {
	return &Server{cfg: cfg, store: store, redis: redis}
}

// Handler 組出完整的中介層鏈。執行順序由外而內是：
// CORS → 限流 → ServeMux 路由比對 → 路由上的中介層 → 處理函式。
//
// 限流刻意是每條路由自己掛的（而不是包在 mux 外面）：這樣各端點可以有不同的
// 上限 —— 媒體與寫入的成本差了一個數量級（見 ratelimit.go 的檔頭）。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 寫入端點（上傳／刪除）共用同一個嚴格的限制，因為它們的成本同一個數量級
	// （一次 50 MiB 的解析、一次檔案系統移除）。
	// /upload 與 /delete 在這裡各建一個計價器而不是共用一個：兩種濫用模式不同
	// （上傳吃滿 50 MB，刪除吃檔案系統），共用會讓其中一種的量掩蓋另一種。
	uploadLimiter := newRateLimiter(s.cfg.RateLimit.WriteRate, s.cfg.RateLimit.WriteBurst, maxRateBuckets)
	deleteLimiter := newRateLimiter(s.cfg.RateLimit.WriteRate, s.cfg.RateLimit.WriteBurst, maxRateBuckets)
	// handleUpload / handleDelete 是 http.HandlerFunc（不是 http.Handler），因此多一層
	// 轉換。這裡刻意不把它們改成 ServeHTTP 方法：那個形狀會讓 handler 測試必須
	// 透過介面呼叫，而這個檔的測試全是直接呼叫方法。
	mux.Handle("/upload", uploadLimiter.middleware(http.HandlerFunc(s.handleUpload)))
	mux.Handle("/delete", deleteLimiter.middleware(http.HandlerFunc(s.handleDelete)))

	// 靜態檔案只在本機模式掛載，而且「是否本機模式」由後端自己回答
	// （staticRoot），不是這裡比對設定檔的字串。S3 模式的檔案由前端直接連 S3
	// （getBaseURL 回的就是 S3 位址），因此這裡**不能**掛 —— 那會讓一個不存在
	// 於本機的目錄也變成一個可以列目錄的 HTTP 根目錄。
	//
	// 只掛 /files/ 而不是 "/"，是因為上傳產生的 URL 形狀是 /files/<uuid>.<ext>，
	// 而 http.FileServer 在收到目錄請求時會**回傳一份 HTML 目錄列表**。若把
	// 檔案伺服器掛在 "/"，那麼 GET / 會列出儲存根目錄下的所有項目 —— 而且因為
	// mediaTokenMiddleware 只擋 /files/ 前綴，那份列表是**不需要 token**的。
	// 那不讀得到檔案內容（內容仍然需要 token），但它洩漏了儲存結構與全部檔名。
	// 把路由收斂到 /files/ 之後，儲存根目錄根本沒有任何路由可以列出。
	if root, ok := s.store.staticRoot(); ok {
		static := http.StripPrefix("/", http.FileServer(http.Dir(root)))
		// 媒體的限制遠比寫入寬鬆：一篇貼文一張圖，一頁動態十來張，而每一次都
		// 是一次 Redis EXISTS。上限的用途是擋掉「用隨機 token 打 /files/」，
		// 不是擋掉真人載入頁面。
		mediaLimiter := newRateLimiter(s.cfg.RateLimit.MediaRate, s.cfg.RateLimit.MediaBurst, maxRateBuckets)
		mux.Handle("/files/", mediaLimiter.middleware(s.mediaTokenMiddleware(rejectDirectoryListing(static))))
	}

	return corsMiddleware(s.cfg)(mux)
}

// maxRateBuckets 是計價器的記憶體上界。
//
// 到達時做一次惰性清除（見 rateLimiter.sweepLocked）。刻意不用背景 goroutine
// 定期清理：這個程式不在建構子裡啟動任何背景工作，而「等下一次請求順便清」對
// 一個保護性元件已經足夠。
const maxRateBuckets = 65536

// rejectDirectoryListing 拒絕任何指向目錄的請求。
//
// http.FileServer 在路徑指向目錄、而該目錄沒有 index.html 時，會回傳一份
// HTML 目錄列表。對這個服務而言那永遠是錯的答案：媒體檔案的存取一律需要
// token（見 mediaTokenMiddleware），而「列出有哪些檔案」不需要 token 卻會
// 洩漏同一份資訊 —— 那等於讓驗證機制有一個不需要 token 的旁路。
//
// 判斷條件刻意就是「路徑以 / 結尾」，而不是「解析後的目錄是否存在」：
// 後者需要真的做一次檔案系統呼叫，而前者涵蓋了所有可能（FileServer 對
// 目錄請求的處理就是「以 / 結尾」這個形式）。
func rejectDirectoryListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			// 用 404 而非 403：403 會確認「這個目錄存在」，而那正是這個
			// 回應要避免透露的資訊。
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleUpload 接受 multipart 上傳並回傳檔案的對外網址。
//
// 拒絕的順序刻意是「方法 → token → 大小上限 → 表單解析 → 檔名白名單 → 存檔」：
// 前三項都不需要讀取本文，因此成本最低；而「先解析表單再驗 token」的話，
// 每一次未授權的請求都會先把整個本文收進記憶體，正是最不該讓它發生的順序。
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validUploadToken(s.cfg, r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// MaxBytesReader 與 ParseMultipartForm 必須都給上限：前者讓 net/http 在
	// 讀取時就中止（記憶體不會先被撐爆），後者決定超過時回 400 而不是把整個
	// 本文留在磁碟上的暫存檔。只有後者時，一個 10 GB 的上傳會先被完整寫進
	// 暫存目錄。
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.Upload.MaxSize)

	if err := r.ParseMultipartForm(s.cfg.Upload.MaxSize); err != nil {
		http.Error(w, "Unable to parse form or file exceeds size limit", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to get file from request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 副檔名決定目錄與 Content-Type，仍以「上傳者給的檔名」為準判斷；
	// 但實際寫進磁碟的檔名是 UUID（見 newFileName），兩者不可混為一談。
	original := filepath.Base(handler.Filename)
	dir := resolveDir(original, s.cfg.Upload.AllowedFiles)
	if dir == "" {
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return
	}

	filename, err := newFileName(original)
	if err != nil {
		log.Printf("產生檔名失敗: %v", err)
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	rel, err := s.store.saveFile(r.Context(), dir, filename, file)
	if err != nil {
		log.Printf("儲存檔案失敗: %v", err)
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"url": s.store.getBaseURL() + rel})
}

// handleDelete 刪除指定網址的檔案。
//
// 支援查詢參數與表單欄位兩種來源：後端既有呼叫走 GET 風格的查詢參數，而
// 瀏覽器端（含 form 提交）只能給表單 —— 兩者都要支援才不會逼呼叫端為了傳一個
// 值而改用 POST。
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validUploadToken(s.cfg, r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileURL := r.URL.Query().Get("url")
	if fileURL == "" && r.Method == http.MethodPost {
		_ = r.ParseForm()
		fileURL = r.Form.Get("url")
	}

	if fileURL == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(fileURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	dir := parts[len(parts)-2]
	// 取最後一段的 Base：呼叫端傳來的 url 是完全不受信任的輸入，
	// "/files/../../etc/passwd" 這種形狀必須在這裡就被拆掉，而不是留給
	// filepath.Join 去處理（它會照樣解析出 baseDir 之外的路徑）。
	filename := filepath.Base(parts[len(parts)-1])
	// "." 與 ".." 也是合法的 filepath.Base 結果（"/files/." 與 "/files/.."），
	// 而它們不是檔名：path.Base(".") == "."，送到 deleteFile 之後會被 Join 成
	// <base>/files/. 與 <base> —— 也就是「把 files 目錄刪掉」與「把儲存根目錄
	// 刪掉」。目錄為空時 os.Remove 會成功，而整個儲存就此失效（每個上傳開始回
	// 500），直到有人重新建立目錄。這不是理論問題：逐檔刪除是這個端點的日常
	// 用途，而把最後一個檔刪掉之後 files/ 就空了。
	//
	// 前導 "." 一併拒絕：這同時擋掉隱藏檔與以 "." 開頭的相對路徑片段，
	// 與 upload 端點的副檔名白名單同一個方向（見 validExt）。
	if filename == "." || filename == ".." || strings.HasPrefix(filename, ".") {
		http.Error(w, "Invalid file name in URL", http.StatusBadRequest)
		return
	}

	if dir != "files" {
		http.Error(w, "Invalid directory in URL", http.StatusBadRequest)
		return
	}

	rel := "/" + dir + "/" + filename
	if err := s.store.deleteFile(r.Context(), rel); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		log.Printf("刪除檔案失敗: %v", err)
		http.Error(w, "Unable to delete file", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "deleted"})
}

// mediaTokenMiddleware 保護 /files/ 的讀取：必須帶一個存在於 Redis 的媒體 token。
//
// 刻意不保護上傳與刪除：它們是後端對後端的呼叫，走 upload.token 通道
// （validUploadToken）。把兩者混在同一個中介層裡會讓靜態檔案的 token 檢查
// 擋住後端自己的上傳。
//
// 降級路徑有兩條，都刻意導向「不驗證」：PublicFiles 為 true（部署者刻意公開）
// 與 redis 為 nil（連不上）。後者的取捨寫在 main.go：這個服務少一層驗證仍能
// 上傳，而上傳是它存在的主要理由。
func (s *Server) mediaTokenMiddleware(next http.Handler) http.Handler {
	if s.cfg.Redis.PublicFiles || s.redis == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/files/") {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing media token")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), mediaTokenTTL)
		defer cancel()

		key := mediaTokenKey(s.cfg, token)
		exists, err := s.redis.Exists(ctx, key).Result()
		if err != nil {
			// 與「token 不存在」分開：503 讓呼叫端知道可以重試，401 會讓它
			// 重新登入 —— 而正確的處置是等 Redis 恢復。混為一談的症狀是
			// Redis 短暫故障讓所有使用者被踢出登入狀態。
			//
			// 日誌做取樣：過去每一次失敗都寫一行，因此 Redis 故障時的輸出量等於
			// 「請求數」—— 而那時候最不需要的就是把磁碟也一起填滿。第一筆一定
			// 記（否則故障的第一個訊號會消失），之後每分鐘最多一筆摘要。
			mediaTokenFailures.Add(1)
			last, _ := lastMediaTokenFailure.Load().(time.Time)
			if first := mediaTokenFailures.Load() == 1; first || time.Since(last) >= time.Minute {
				lastMediaTokenFailure.Store(time.Now())
				log.Printf("Redis token check failed（第 %d 次，此後每分鐘至多一筆）: %v",
					mediaTokenFailures.Load(), err)
			}
			writeJSONError(w, http.StatusServiceUnavailable, "media token service unavailable")
			return
		}

		if exists == 0 {
			writeJSONError(w, http.StatusUnauthorized, "invalid or expired media token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON 以 JSON 回應並附上正確的 Content-Type。
//
// 刻意不直接 fmt.Fprintf：後者不會設定 Content-Type，瀏覽器會猜，而猜錯的
// 結果是下載一個檔案而不是顯示它。
func writeJSON(w http.ResponseWriter, payload map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	// 刻意不設狀態碼（隱含 200）並容忍編碼失敗：這個 payload 是一個 map[string]string，
	// 不可能編碼失敗；真的發生時回一個空物件遠好於讓整個請求 panic。
	_ = json.NewEncoder(w).Encode(payload)
}

// writeJSONError 以 JSON 回應一個錯誤物件，並帶上指定的狀態碼。
//
// 刻意維持 {"error":"..."} 這個形狀：後端依賴它（見 backend/forum/httpapi 讀取
// 上傳失敗回應的程式），而 /upload 與 /delete 之外的端點一律用純文字的
// http.Error。兩種形狀並存是既有的不一致，統一它需要同時改後端，因此這裡
// 只把「哪一條路徑用哪一種」的界線維持在原來的位置並加註說明。
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// resolveDir 依副檔名白名單決定檔案要放進哪個邏輯目錄。
//
// 回傳空字串代表「不允許」。這個「目錄名就是白名單」的設計是有意的：目錄同時
// 出現在 /files/<dir>/<name> 這個 URL 形狀裡，因此允許的副檔名也等於允許的
// URL 前綴 —— 兩者不可能分開設定而不製造出「能存但讀不到」的組合。
func resolveDir(filename string, allowed []string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if isAllowedExt(ext, allowed) {
		return "files"
	}
	return ""
}

// isAllowedExt 判斷副檔名是否在白名單中，比對時兩邊都轉小寫。
//
// 白名單那一邊轉小寫的理由：設定檔裡寫 ".JPG" 與 ".jpg" 應該是同一件事，
// 讓部署者因為大小寫而得到 400 沒有任何好處。
func isAllowedExt(ext string, allowed []string) bool {
	ext = strings.ToLower(ext)
	for _, a := range allowed {
		if strings.ToLower(a) == ext {
			return true
		}
	}
	return false
}

// mediaTokenKey 組出媒體 token 在 Redis 的 key。
//
// 前綴必須與後端的 MEDIA_TOKEN_KEY_PREFIX 相同，且都要與 session 的
// "forum:session:" 區隔。不一致的症狀是「上傳成功、貼文也存得下，但圖片一律
// 401/403」—— 後端寫進 A 前綴，這裡查 B 前綴。
func mediaTokenKey(cfg *Config, token string) string {
	return cfg.Redis.TokenKeyPrefix + token
}

// validUploadToken 檢查上傳／刪除請求帶的 token。
//
// 接受三種形式：無 token（僅當設定檔沒設 token 時）、原值、以及 Bearer 前綴
// 形式。最後一種是為了讓呼叫端可以直接沿用它對外服務的 Authorization 標頭
// 慣例，而不必為這個內部端點特別組一個 X-Upload-Token。
//
// 用 subtle.ConstantTimeCompare 而不是 ==：== 會在第一個不同的位元組就返回，
// 因此回應時間洩漏「前綴對了幾個字元」。這個 token 是長隨機字串，實務上難以
// 逐位元組重建，但比對本身沒有理由洩漏任何資訊 —— 而函式庫已經提供免費的常數
// 時間比對。
//
// 設定檔沒設 token 時全部放行是既有的行為，保留它（本機測試用），但它是一個
// 靜默的無驗證狀態 —— 那正是下面這個註解要留給部署者的提醒：
// 留空等於任何人只要找得到這個服務就能上傳與刪除，而沒有任何錯誤訊息會告訴你。
func validUploadToken(cfg *Config, r *http.Request) bool {
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.Header.Get("X-Upload-Token")
	}
	want := []byte(cfg.Upload.Token)
	return cfg.Upload.Token == "" ||
		subtle.ConstantTimeCompare([]byte(token), want) == 1 ||
		subtle.ConstantTimeCompare([]byte(token), []byte("Bearer "+cfg.Upload.Token)) == 1
}

// originAllowed 判斷來源是否在允許清單內；清單含 "*" 時全部放行。
func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

// addCORSHeaders 依設定補上 CORS 標頭；OPTIONS 預檢在來源被允許時就地回 204。
//
// 刻意「不」在來源被拒絕時也回 204：那會讓瀏覽器認為預檢通過，然後在真正的
// 請求上失敗。讓它落到路由比對，得到的 405／404 才是能指認問題的訊息。
//
// Vary: Origin 是必要的：下面回應的是「呼叫端自己送來的那個 Origin」，而快取
// 不區分 Origin 時，前一個來源拿到的 ACAO 會被送給下一個來源 —— 那等於把
// 一個被允許的來源名單快取成全部放行。預設設定是 AllowedOrigins = ["*"]，
// 更需要這一行讓收緊設定之後的行為可預期。
func addCORSHeaders(w http.ResponseWriter, r *http.Request, cfg *Config) {
	origin := r.Header.Get("Origin")
	if originAllowed(origin, cfg.CORS.AllowedOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
		if len(cfg.CORS.AllowedMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.CORS.AllowedMethods, ", "))
		}
		if len(cfg.CORS.AllowedHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.CORS.AllowedHeaders, ", "))
		}
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Content-Length")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
}

// corsMiddleware 把 CORS 補標頭與預檢處理包成中介層。
func corsMiddleware(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			addCORSHeaders(w, r, cfg)
			if r.Method == http.MethodOptions {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
