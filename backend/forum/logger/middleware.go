/*
Package logger 的 HTTP 存取記錄中介層與請求中繼資料管理。

職責

  - 為每個進來的請求指派一個 request_id，並把該 ID 回寫到回應標頭 X-Request-ID，
    讓前端或上游代理在回報問題時能直接引用。
  - 解析出請求的使用者身分（email）與來源 IP，一併寫入 context。
    任何深度的 logger.xxxContext 呼叫都會自動附上這三個欄位（實作見 logger.go）。
  - 記錄每個請求的方法、路徑、協定、最終狀態碼與處理耗時，並依嚴重度自動分級。

對外介面

	MetadataFromContext  從 context 讀出中繼資料（供 logger 與 handler 使用）
	WithMetadata         把中繼資料寫入 request 的 context
	SetUserEmail/GetUserEmail  以 context 為媒介單獨傳遞使用者 email
	Metadata             中繼資料的值型別
	UserResolver         由呼叫端注入的「身分解析函式」型別
	LoggingMiddleware     產生存取記錄中介層的進入點
	UserEmailKey/MetadataKey/RequestIDKey  三個 context key

關鍵設計決策

  - 為什麼把 email 寫進 context：身分解析（讀 cookie + 查 Redis）成本高，若每層 handler
    都自行解析會重複付費。LoggingMiddleware 解析一次後放入 context，往後所有需要身分的
    程式碼都能共用同一個結果，日誌也能自動帶上 user_email 而不必在每個 log 呼叫點手動帶。
  - 為什麼用自訂的 contextKey 型別而非直接用 string：Go 的 context 會對 key 做
    介面相等比較，若用裸 string 就有與其他套件撞 key、誤取到別的值。使用未匯出的
    專屬型別可讓「誤用別套件的 key」在編譯期就不可能發生。
  - 為什麼包一層 StatusWriter：為了在 handler 回應之後仍能知道實際狀態碼。
    直接包 http.ResponseWriter 會讓實作失去 http.Hijacker / http.Flusher 等選用介面，
    因此這裡逐一轉發 Hijack 與 Flush，確保串流／長連線與即時回應仍可運作。
  - 中介層在 Handler() 中被放在 session Refresh 之外側、mux 之內側：
    因此存取記錄的 duration 不含 Redis 的 session 續期時間，反映的是實際處理耗時；
    反過來說，若把 LoggingMiddleware 放到最外層，續期延遲會被算進每筆 API 的耗時。

敏感資訊

  - 會記錄的敏感欄位：來源 IP 與使用者 email（皆為明文，無遮蔽）。
  - 不會記錄的：Cookie、session token、OAuth code/token。logger 本身不會主動讀取這些標頭。
  - requestID 由 crypto/rand 產生，僅用於關聯記錄，與任何身分憑證無關。

已知限制（刻意留白、未實作）

  - 檔案切割（rotation）不在本檔案範圍：log 檔案由 Init 直接開啟，無大小上限、無時間輪替，
    檔名完全取自 config 的 LOG_FILE，換檔需由外部（logrotate 或容器日誌收集）負責。
  - 這個檔案也不做日誌內容的遮蔽；sanitizeLogValue（位於 logger.go）只處理換行與長度。
*/
package logger

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// contextKey 是本檔案專用的 context key 型別，刻意不匯出且不與任何字串型別相容。
// 依 Go 官方建議，這樣可以避免與其他套件的 key 發生比對碰撞。
type contextKey string

// UserEmailKey 為單獨存放使用者 email 的 context key。
// 與 Metadata 內的 UserEmail 分開，是為了讓「只想傳 email、不需要整套中繼資料」
// 的呼叫端（SetUserEmail / GetUserEmail）不必構造 Metadata。
const UserEmailKey contextKey = "user_email"

// MetadataKey 為存放 Metadata 結構的 context key，供 LoggingMiddleware 寫入、
// 供 MetadataFromContext 與 logger 讀出。
const MetadataKey contextKey = "metadata"

// RequestIDKey 為單獨存放 request ID 的 context key。
// 目前 LoggingMiddleware 將 request ID 一併放進 Metadata，本 key 是預留给
// 只需要 request ID 而不想建立完整 Metadata 的呼叫端使用的。
const RequestIDKey contextKey = "request_id"

/*
Metadata 為單一請求的日誌中繼資料。欄位皆為字串，且一律以值型別（value type）存放：
寫入 context 後即使呼叫端再修改本地變數，context 內的內容也不受影響。

	IP         請求來源位址，取自 X-Forwarded-For / X-Real-IP / RemoteAddr。
	           屬於外部可控輸入，輸出前仍會再過一次 sanitizeLogValue。
	UserEmail  已登入使用者的 email；未登入時 LoggingMiddleware 會填 "anonymous"
	           而非空字串，使日誌中「有訪客但未登入」的情況不會與「完全沒有 metadata」混淆。
	           為明文，未做遮蔽，會同時出現在 log 檔案中。
	RequestID  本次請求的 16 個十六進位字元識別碼，同時回寫到 X-Request-ID 回應標頭。
*/
type Metadata struct {
	IP        string
	UserEmail string
	RequestID string
}

// MetadataFromContext 從 ctx 取出 Metadata，並以 "-" 填補缺失的欄位。
//
// 設計理由：logger 需要固定格式的欄位，若某欄為空字串，text 輸出會出現 request_id= 這類
// 語意不明的空缺。以 "-" 作為佔位符可以讓「欄位不存在」在日誌中明確可見。
// 取值失敗（ctx 為 nil 或 key 不存在）時以型別斷言的結果為準，直接回傳全 "- 的 Metadata，
// 呼叫端（logger.logWithContext）因此不需要額外處理 nil。
func MetadataFromContext(ctx context.Context) Metadata {
	// 斷言的第二個回傳值刻意捨棄：context 中若存了別的型別，統一視為「沒有 metadata」。
	meta, _ := ctx.Value(MetadataKey).(Metadata)
	// 逐欄補 "-" 而非在函式開頭一次判斷：LoggingMiddleware 一定會填滿三欄，
	// 只有背景工作（如 main 的啟動訊息）才會落到這些分支。
	if meta.IP == "" {
		meta.IP = "-"
	}
	if meta.UserEmail == "" {
		meta.UserEmail = "-"
	}
	if meta.RequestID == "" {
		meta.RequestID = "-"
	}
	return meta
}

// WithMetadata 回傳一個帶有中繼資料的新 *http.Request。
//
// 不直接修改傳入的 r 本身：*http.Request 是以值傳遞的結構，改欄位只影響本地複本，
// 必須呼叫 r.WithContext 產生新指標才能讓呼叫端（LoggingMiddleware）真正沿用新的 context。
// 呼叫端需記得接住回傳值，否則 metadata 不會生效。
func WithMetadata(r *http.Request, ip, userEmail, requestID string) *http.Request {
	ctx := context.WithValue(r.Context(), MetadataKey, Metadata{
		IP:        ip,
		UserEmail: userEmail,
		RequestID: requestID,
	})
	return r.WithContext(ctx)
}

// SetUserEmail 回傳一個帶有使用者 email 的新 *http.Request，供只想傳遞身分、
// 不需要完整中繼資料的程式碼使用（此為預留介面，目前處理路徑皆走 LoggingMiddleware）。
func SetUserEmail(r *http.Request, email string) *http.Request {
	ctx := context.WithValue(r.Context(), UserEmailKey, email)
	return r.WithContext(ctx)
}

// GetUserEmail 讀出 SetUserEmail 寫入的 email。
// 取不到時回傳空字串而非錯誤，因為「查無登入者」在論壇的讀取型 API 中屬正常情況，
// 由呼叫端自行判斷要導向登入頁或以匿名身分繼續。
func GetUserEmail(r *http.Request) string {
	if email, ok := r.Context().Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

/*
StatusWriter 包裝 http.ResponseWriter，唯一的目的是「記住實際回傳的狀態碼」。

為什麼需要：net/http 只在 handler 呼叫 WriteHeader 時才決定狀態碼，之後無法查詢。
而很多 handler 會直接呼叫 Write 或 writeJSON 而不呼叫 WriteHeader，此時狀態碼隱含為 200。
把狀態碼存下來，存取記錄才能在 handler 結束後仍取得正確的數值。
代價：包裝後本型別只轉發 Hijack 與 Flush，未實作 io.ReaderFrom 與已被棄用的
http.CloseNotifier，因此 http.ServeFile 會失去 io.Copy 的 sendfile 快速路徑，
大檔案的傳輸效能可能略降；換取的是可控的狀態碼記錄。

為什麼匯出：存取記錄（LoggingMiddleware）與監控統計（httpapi 的 metrics 中介層）
都需要「handler 結束後知道狀態碼」，兩者若各自實作一份包裝，就會有兩份
Hijack/Flush 轉發邏輯必須同步維護 —— 而其中一份漏轉發的症狀是 WebSocket 升級
在某些路徑上失效，非常難歸因。因此這裡匯出唯一一份實作，兩邊共用。
*/
type StatusWriter struct {
	http.ResponseWriter
	// statusCode 記錄已寫出的狀態碼。NewStatusWriter 建立時預填
	// http.StatusOK，對應「handler 只寫內容、未呼叫 WriteHeader」的情況。
	statusCode int
}

// NewStatusWriter 包裝 w 並預填狀態碼為 200。
//
// 預填 200 是必要的：statusCode 若是零值，0 不是任何合法的 HTTP 狀態碼，
// 而「只寫 body 不寫狀態碼」在 net/http 裡是完全正常（且常見）的寫法。
func NewStatusWriter(w http.ResponseWriter) *StatusWriter {
	return &StatusWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

// StatusCode 回傳目前已寫出的狀態碼。在 handler 回應完之後呼叫才有意义。
func (sw *StatusWriter) StatusCode() int {
	return sw.statusCode
}

// Hijack 轉發給底層的 http.Hijacker 讓 WebSocket 等協定升級仍可進行。
// 底層不支援時回傳明確錯誤，讓呼叫端知道無法升級，而不是得到 nil 造成後續 panic。
func (sw *StatusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := sw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

// Flush 轉發給底層的 http.Flusher，讓支援串流／SSE 的 handler 能逐段推送內容。
// 以 if ok 檢查而非回傳錯誤：Flush 是「盡力而為」的最佳努力操作，底層不支援時靜默略過即可，
// 這也讓測試用的 httptest.ResponseRecorder 等實作能安全通過。
func (sw *StatusWriter) Flush() {
	flusher, ok := sw.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

// WriteHeader 記錄狀態碼後轉發給底層。
// 刻意不防止重複呼叫：net/http 對第二次 WriteHeader 會發出「superfluous WriteHeader」警告
// 且忽略之，此處維持相同行為以免改變任何 handler 的既有語意。
func (sw *StatusWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

// UserResolver 為身分解析函式的型別，由呼叫端注入。
// 注入而非在套件內直接依賴 session 套件，是為了讓 logger 不與 session 形成循環相依，
// 也讓測試可以塞入假的解析邏輯。
type UserResolver func(r *http.Request) string

/*
isNoisyRequest 判斷該請求是否屬於「不值得寫存取記錄」的雜訊型別。

雜訊定義：健康檢查端點（/api/check、/healthz）、靜態資源路徑（/asset/、/static/）
以及帶有常見靜態副檔名（.js/.css/.png 等）的請求。這類請求量最大、診斷價值最低，
寫進日誌只會把重要的記錄淹沒。

使用副檔名判斷而非完整白名單：靜態檔的檔名由建置工具產生、數量不固定，
用「有這些副檔名就視為靜態」能在不維護清單的前提下涵蓋新增的檔案。
注意 .json 也在清單內，但這意味著以 .json 結尾的 API 端點也會被視為雜訊。

目前狀態：本專案尚未呼叫此函式（LoggingMiddleware 仍記錄所有請求）。
保留它的原因是雜訊判定邏輯已驗證過，待要啟用時（例如加上取樣或分檔輸出）可直接接上；
在正式接線前，請勿宣稱存取記錄已排除靜態資源。
*/
func isNoisyRequest(r *http.Request) bool {
	// 讀 r.URL.Path 而非 r.RequestURI：Path 已完成路徑清理且不含查詢字串，
	// 可避免把使用者帶入的查詢參數帶進判斷邏輯。
	path := strings.TrimSpace(r.URL.Path)
	// 空路徑視為根路徑，避免因為空字串而漏掉比對。
	if path == "" {
		path = "/"
	}

	switch path {
	case "/api/check", "/healthz":
		return true
	}

	// 兩種靜態掛載路徑：/asset/ 為後端掛載的舊資源目錄，/static/ 為常見的靜態資源前綴。
	if strings.HasPrefix(path, "/asset/") || strings.HasPrefix(path, "/static/") {
		return true
	}

	// 以最後一個 "." 之後的副檔名判斷。先確認確實含有 "."，避免 strings.LastIndex
	// 回傳 -1 時發生切片越界。
	if strings.Contains(path, ".") {
		// 轉小寫，讓 .PNG 與 .png 同樣被視為靜態資源。
		suffix := strings.ToLower(path[strings.LastIndex(path, "."):])
		switch suffix {
		case ".js", ".css", ".json", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp", ".map", ".wasm":
			return true
		}
	}

	return false
}

// LoggingMiddleware 產生 HTTP 存取記錄中介層。
//
// 參數：
//   - next：實際處理請求的 handler（實務上為 http.ServeMux）。
//   - resolveUser：身分解析函式，由呼叫端注入（本專案傳入 session.Manager.ResolveUser）。
//     允許為 nil，此時所有請求都以 "anonymous" 記錄，適合測試或不需身分的場景。
//
// 處理流程：
//  1. 記錄起始時間，解析來源 IP 與使用者身分。
//  2. 產生 request ID，寫入 context，並回寫 X-Request-ID 回應標頭。
//     標頭必須在呼叫 next 之前寫入，否則 handler 一旦輸出內容就無法再補標頭。
//  3. 以 StatusWriter 包住 w，呼叫 next。
//  4. 依狀態碼與耗時決定日誌等級：5xx → ERROR，4xx 或逾時 3 秒 → WARN，其餘 → INFO。
//
// 錯誤處理：resolveUser 或 newRequestID 失敗都不會讓請求失敗，改以替代值繼續
// （身分為 "anonymous"、request ID 為時間戳）。存取記錄是診斷工具，
// 不該因為無法記錄而阻斷使用者操作。
//
// 效能考量：resolveUser 在本專案中會讀 cookie 並對 Redis 做一次 HGET，
// 也就是每個請求（含靜態資源）都會多一次網路往返。這是為了讓每條記錄都能帶上
// user_email 而付出的成本；若流量成長造成壓力，應考慮改為只在非雜訊請求上解析身分。
//
// 敏感資訊：記錄中不含 cookie、token 或請求本文，只含 request ID、方法、路徑、協定、
// 狀態碼、耗時，以及由中繼欄位提供的 IP 與 email。
func LoggingMiddleware(next http.Handler, resolveUser UserResolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 在做任何額外工作前取時間基準，讓 duration 涵蓋整個處理過程。
		start := time.Now()

		ip := getClientIP(r)
		// 先給一個明確的非空預設值，讓 metadata 永遠帶得出欄位（見 MetadataFromContext 的說明）。
		user := "anonymous"
		if resolveUser != nil {
			if u := resolveUser(r); u != "" {
				user = u
			}
		}
		requestID := newRequestID()
		// 必須接住回傳的新 request：WithMetadata 是以 WithContext 產生複本，
		// 後續 next.ServeHTTP 與最後的日誌都要使用這個新指標才能讀到 metadata。
		r = WithMetadata(r, ip, user, requestID)
		// 在 next 之前寫入回應標頭。刻意不寫入 r.Header（請求標頭）：
		// 讓上游服務（如檔案伺服器）能看到此 ID 需要明確複製，且可能造成偽造風險。
		w.Header().Set("X-Request-ID", requestID)

		// 預填 200：handler 若只呼叫 Write 而未呼叫 WriteHeader，狀態碼應視為 200。
		rw := NewStatusWriter(w)
		next.ServeHTTP(rw, r)

		// handler 回傳後才計算耗時；此時尚未記錄，因此耗時只涵蓋處理時間。
		duration := time.Since(start)
		// 記錄 path 而非完整 RequestURI：查詢字串常帶 token 或使用者輸入，
		// 只記路徑可大幅降低敏感資訊外洩的機會。
		msg := fmt.Sprintf("[HTTP] request_id=%s %s %s %s status=%d duration=%s",
			requestID, r.Method, r.URL.Path, r.Proto, rw.statusCode, duration)

		// 分級規則：伺服器端錯誤最嚴重優先判定；4xx 與逾時並列為 WARN；
		// 逾時門檻定在 3 秒，與前端一般互動的等待容忍度對齊，避免正常但偏慢的
		// 資料庫或外部檔案伺服器呼叫被誤標成警告。兩者並列而非串接，是因為
		// 5xx 一定也要被視為錯誤，即使它同時也很慢。
		switch {
		case rw.statusCode >= 500:
			ErrorContext(r.Context(), msg)
		case rw.statusCode >= 400 || duration >= 3*time.Second:
			WarnContext(r.Context(), msg)
		default:
			InfoContext(r.Context(), msg)
		}
	})
}

// newRequestID 產生 16 個十六進位字元（64 bits）的請求識別碼。
//
// 使用 crypto/rand 而非數學亂數：request ID 會出現在回應標頭中，是外部可見的輸入，
// 必須避免可預測性被拿來猜測或碰撞他人請求的記錄。
// 失敗時（系統亂數來源不可用）退回 UnixNano 時間戳：寧可是一個較弱但仍可用的
// 唯一性來源，也不要讓整個請求失敗 —— 這個值只影響診斷，不影響功能。
func newRequestID() string {
	// 8 bytes = 64 bits，在單機的請求量下碰撞機率可忽略。
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// getClientIP 取得請求來源 IP，判斷順序為 X-Forwarded-For → X-Real-IP → RemoteAddr。
//
// 信任 X-Forwarded-For 的理由：本專案部署在反向代理之後，RemoteAddr 只會拿到代理的位址。
// 取 XFF 的第一段是「最靠近用戶端」的那個位址。
//
// 安全提醒（重要）：此實作「無條件信任」X-Forwarded-For。當服務可被直接連線
// （未經過可信代理、或代理未清洗此標頭）時，呼叫端可以任意偽造 IP，使日誌中的 IP
// 欄位不可作為稽核依據。理論上應只信任已知代理的 IP 段，但本專案未實作該白名單。
// 同一段理由也適用於 X-Real-IP。
//
// 邊界處理：RemoteAddr 的格式為 "IP:Port"，以「最後一個冒號」切分以同時支援
// IPv4 與 [IPv6]:Port 兩種形式（IPv6 的位址本身含冒號，故必須取最後一個）。
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF 可能是 "client, proxy1, proxy2" 的逗號分隔清單，取第一段即原始用戶端。
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip := r.RemoteAddr
	// 沒有代理可提供標頭時才退回 RemoteAddr。idx == -1 表示沒有冒號，
	// 此時保持原值不變（極端情況下 RemoteAddr 可能只有純位址）。
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
