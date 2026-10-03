/*
auth 封裝以 Google 為唯一識別證供應者的 OAuth2 授權碼流程（Authorization Code Flow）。

【對外介面】
Init 建立全域 oauth2.Config，必須在服務開始處理請求前呼叫一次。
HandleLogin 是 GET /auth/google 的 handler，產生授權網址並轉向 Google。
ReturnPath 從 OAuth 回呼的 state 參數取回登入前想前往的路徑。
GetUserEmail 用回呼的 code 換取 token，再向 Google 查詢使用者 email。
GoogleUser 與 OauthConfig 是公開符號，供測試或其他套件引用。
路由實際註冊於 httpapi.Server.Handler()：
GET /auth/google 由 auth.HandleLogin 處理；
GET /auth/callback 由 httpapi 自己的 handleGoogleCallback 處理，該函式內部呼叫
GetUserEmail 與 ReturnPath，並負責建立 session cookie 與導回。

【主要依賴】
golang.org/x/oauth2 及其 oauth2/google 套件，google.Endpoint 內含 Google 的
授權端點與 token 端點，因此本檔不需要自行組任何 Google URL。
golang.org/x/oauth2 內建的 context 感知 HTTP client。
forum/forum/logger，支援帶入 request context，日誌可附帶 request_id、ip、user_email。

【關鍵設計決策與限制】
state 參數被重複利用為「登入後要回到哪一頁」的載體，而不是標準建議的 CSRF 隨機
nonce。取捨是不必落地 cookie 或 session 就能完成導回，但 state 並未綁定發起登入
的瀏覽器，因此無法防禦 login CSRF（攻擊者可用自己的帳號完成登入並把狀態寫進
受害者的瀏覽器）。本套件唯一的防線是 isSafeReturnPath 這道開放轉向檢查。

只申請 userinfo.email 這一個 scope，不申請 openid 與 profile。代價是拿不到使用者
名稱與頭像，論壇以 email 作為唯一識別；好處是不需要通過 Google 對敏感或受限
scope 的審查。

本套件完全不碰 cookie：session 的建立、續期與清除都由呼叫端
（httpapi 與 forum/forum/session）負責，auth 只提供「這個 email 是誰」。

Init 寫入的 OauthConfig 是無保護的全域變數，設計上假設只在啟動時寫入一次、
之後唯讀。若執行期間仍可能寫入，會與併發讀取產生資料競爭，本套件未提供鎖定。

GetUserEmail 假設 Init 已經呼叫：OauthConfig 為 nil 時，OauthConfig.Exchange 會在
讀取 c.RedirectURL 時發生 nil 解參考而 panic。HandleLogin 有檢查 nil 並回應 500，
但 GetUserEmail 沒有，因此呼叫端必須自行確保初始化已完成。

Google 回傳的 email 對應到 forum_users 資料表的 email 欄位（VARCHAR(255) 主鍵
角色），但本套件本身不寫資料庫。實作也沒有檢查 Google 回應中的 verified_email
旗標，僅以「申請 email scope」這件事實作為信任前提。
*/
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"forum/forum/logger"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleUser 是向 Google userinfo 端點請求後，本套件唯一需要的一小塊資料。
// 刻意只保留 email：encoding/json 會忽略回應中的其他欄位（id、name、picture、
// verified_email 等），因此不需要為 Google 未來新增欄位維護對應欄位。
type GoogleUser struct {
	// Email 來自 https://www.googleapis.com/oauth2/v2/userinfo 回應的 "email" 欄位，
	// 對應到 forum_users.email。它是全站的唯一使用者識別鍵：session 的 email 欄位、
	// 貼文的 author_email、允許的管理員白名單比對都以此為準。
	// 未使用 pointer 型別，因此欄位不存在時會是空字串而非 nil。
	Email string `json:"email"`
}

var (
	// OauthConfig 是整個 auth 套件唯一的 OAuth2 設定來源，由 Init 寫入。
	// 之所以是公開變數而非用函式取得，是刻意的簡化：設定只在啟動時設定一次、
	// 之後唯讀，透過 getter 取得只會增加呼叫點的樣板碼。
	// 限制：Init 必須在服務開始處理請求之前完成；執行期間若仍寫入，
	// 會與併發讀取產生資料競爭，本套件沒有提供任何鎖定或 once 保護。
	OauthConfig *oauth2.Config

	// userInfoURL 是向 Google 查詢 email 的端點。
	//
	// 為什麼是變數而不是直接寫在 GetUserEmail 裡：GetUserEmail 是本套件唯一
	// 有分支邏輯（狀態碼檢查、JSON 解析、錯誤分類）的函式，而它的兩次對外
	// 請求都寫死主機名時，整支函式就無法測試 —— 測試只能真的去呼叫 Google。
	// oauth2 的 Endpoint（TokenURL / AuthURL）本來就是透過 OauthConfig 注入的，
	// 這裡讓 userinfo 端點有同樣的注入點，兩者就一致了。
	//
	// 限制與 OauthConfig 相同：只在啟動時或測試中寫入，之後唯讀。
	userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// Init 建立 Google OAuth2 用戶端設定並寫入全域變數 OauthConfig。
//
// 參數：
// clientID 是 Google Cloud Console 核發的 OAuth 2.0 Client ID。
// clientSecret 是對應的 Client Secret，只在伺服器端交換 token 時使用。
// redirectURL 是授權回呼網址，必須與 Google Console 登記的值逐字相符，
// 不符時後續的 token 交換會以 redirect_uri_mismatch 失敗。
//
// 錯誤條件：無，永不回傳錯誤。傳入空字串也不會在此報錯，會延後到實際使用
// OauthConfig 的端點才以 500 或 OAuth 錯誤反應表現。
//
// 副作用：寫入全域 OauthConfig，並以 INFO 記錄 client_id 與 redirect_url。
// 不寫資料庫、不設定 cookie、不發 HTTP 請求。
func Init(clientID, clientSecret, redirectURL string) {
	// 把 client_id 與回呼網址寫進日誌，是為了讓部署者能從日誌確認「這台機器
	// 實際跑的是哪一組憑證」，多環境並存時特別重要。client_secret 屬敏感值，
	// 絕不進入日誌。
	logger.Infof("[AUTH] Initializing Google OAuth: client_id=%s redirect_url=%s", clientID, redirectURL)
	OauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		// 只申請 email scope：足以識別使用者，且不觸發 Google 對敏感或受限
		// scope 的審查。放棄 openid 與 profile 的代價是拿不到暱稱與頭像。
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
		},
		// 授權頁與 token 端點由套件提供，因此不必自行組 URL。AuthStyle 留空
		// 預設為 auto：先嘗試以標頭帶憑證，失敗再退回查詢參數。
		Endpoint: google.Endpoint,
	}
}

// HandleLogin 是 GET /auth/google 的 handler：產生 Google 授權網址並把瀏覽器導過去。
//
// 副作用：寫出 307 轉向回應，Location 標頭指向 Google 授權頁。不寫資料庫、
// 不設定 cookie。
// 無錯誤回傳值：所有失敗情境都以 HTTP 回應表達。
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	// OauthConfig 為 nil 代表啟動時忘了呼叫 Init。此時回 500 而不是仍然導向
	// Google，因為導過去也拿不到合法憑證完成流程，只會讓使用者卡在 Google 頁面。
	if OauthConfig == nil {
		logger.ErrorContext(r.Context(), "[AUTH] OAuth config not initialized")
		// 用 500 而非 503：這是伺服器設定錯誤而非暫時性不可用，重試沒有意義。
		http.Error(w, "OAuth config not initialized", http.StatusInternalServerError)
		return
	}
	// 登入完成後要回到哪一頁，由呼叫端（/forum/login?return=...）提供。
	// 使用前必須通過 isSafeReturnPath，否則退回首頁，避免被當成開放轉向的跳板。
	returnPath := r.URL.Query().Get("return")
	if !isSafeReturnPath(returnPath) {
		returnPath = "/"
	}
	// 第二個參數會成為 OAuth 的 state 參數。AuthCodeURL 會對它做 URL 編碼，
	// 因此即使 returnPath 含有 / 與 ? 也不會破壞查詢字串結構。
	loginURL := OauthConfig.AuthCodeURL(returnPath)
	// 授權網址含有 client_id 與 state，記在 DEBUG 級別供排查
	// 「登入後跳錯頁」這類問題；常駐環境維持 INFO 以上時不會產生噪音。
	logger.DebugfContext(r.Context(), "[AUTH] Redirecting user to Google login: url=%s", loginURL)
	// 選 307 Temporary Redirect 而非 302：307 在 HTTP 語意上要求用戶端保留
	// 原始的方法與主體，比 302 更精確地表達「這是登入流程中的一段轉址，
	// 不是資源的永久位置」。此端點僅供瀏覽器以 GET 進入，兩者在此路徑上的
	// 實際行為差異不大，選 307 是為了語意正確。
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

// ReturnPath 從 OAuth 回呼的查詢參數 state 取回登入前指定的返回路徑。
//
// 不驗證 state 的 CSRF 性質（見檔案層說明），只做開放轉向防護。
// 回傳值保證是以 "/" 開頭的站內路徑，或是使用者未指定時的 "/"。
// 副作用：無，只讀取 r 的查詢字串。
func ReturnPath(r *http.Request) string {
	// Google 會原樣送回 state，而 Query().Get 會做一次 URL 解碼，正好還原
	// AuthCodeURL 當初編碼的內容。若 state 遺失或為空字串，這裡會得到 ""。
	returnPath := r.URL.Query().Get("state")
	// 使用者按取消、Google 拒絕授權、state 被竄改等情況都可能得不到安全值，
	// 此時一律回首頁，不嘗試猜測原本的目標。
	if !isSafeReturnPath(returnPath) {
		return "/"
	}
	return returnPath
}

// isSafeReturnPath 判斷 returnPath 是否為安全的「站內相對路徑」。
//
// 拒絕的條件與依據：
// QueryUnescape 失敗代表含有不合法的百分號編碼，無法確定其意圖，直接視為不安全。
// 不以 "/" 開頭，可能被用來跳到 "https://evil.example" 這類絕對網址（開放轉向）。
// 以 "//" 開頭，是 protocol-relative URL，瀏覽器會當成 "https://evil.example"。
// 含反斜線，因為部分瀏覽器與舊版 URL 解析器會把 "/\evil.example" 正規化成
// "//evil.example"，等於繞過上一條檢查。
//
// 副作用：無，是純函式，可安全併行呼叫。
func isSafeReturnPath(returnPath string) bool {
	// 先解碼再判斷：若直接比對字面，攻擊者可用 %2F%2Fevil.example 這類編碼形式
	// 繞過 "//" 檢查。解碼失敗不回退到原字串比對，因為失敗本身就代表內容不可信。
	decoded, err := url.QueryUnescape(returnPath)
	if err != nil {
		return false
	}
	// 允許 "/admin"、"/?tab=2" 這類站內路徑。Query() 與上面這次呼叫都已解碼，
	// 因此此處不會再看到 %XX 序列，判斷的是實際會被導向的目標。
	return strings.HasPrefix(decoded, "/") && !strings.HasPrefix(decoded, "//") && !strings.Contains(decoded, "\\")
}

// GetUserEmail 處理 OAuth2 回呼的核心步驟：用 code 換取 token，再以 token 向 Google
// 查詢使用者 email。
//
// 參數：r 必須是 /auth/callback 的請求，且帶有 Google 回傳的 "code" 查詢參數。
// 換 token 的請求掛在 r.Context() 上，使用者中途關閉分頁會直接取消該在途請求。
//
// 回傳值：Google 帳號的 email；失敗時回傳空字串與非 nil 的 error。
//
// 錯誤條件：查詢字串沒有 code、token 交換失敗、userinfo 請求失敗、回應狀態碼
// 為 4xx/5xx、或回應內容無法解析成 JSON。錯誤訊息內含上游細節，呼叫端若要
// 回給使用者應自行過濾（目前 httpapi 的 handleGoogleCallback 會原樣帶入回應）。
//
// 副作用：對 oauth2.googleapis.com 與 googleapis.com 各發出一次對外 HTTPS 請求。
// 不寫資料庫、不設定 cookie。
func GetUserEmail(r *http.Request) (string, error) {
	// 沒有 code 代表這不是合法的回呼：常見於使用者直接造訪 /auth/callback，
	// 或 Google 在使用者未授權時以 error 查詢參數結束流程。
	// 錯誤刻意不帶上游內容，避免把內部細節洩漏給呼叫端。
	code := r.URL.Query().Get("code")
	if code == "" {
		logger.ErrorContext(r.Context(), "[AUTH] No code found in callback URL")
		return "", fmt.Errorf("no code in query")
	}

	// 授權碼交換。這一步會把 client_secret 送到 Google，因此只能在伺服器端進行。
	// 掛上 r.Context() 讓請求可被取消，而不是在使用者已離開後仍繼續等待。
	// 注意：OauthConfig 為 nil 時此行會 panic，呼叫端必須先確認已呼叫 Init。
	logger.DebugContext(r.Context(), "[AUTH] Exchanging code for token")
	token, err := OauthConfig.Exchange(r.Context(), code)
	if err != nil {
		// 常見成因：redirect_uri 與 Google Console 登記值不一致、授權碼已被
		// 使用過（重播攻擊）、或使用者授權的是另一組 client。
		// 換取失敗時回傳的 Token 不可信，因此直接中止，不嘗試後續步驟。
		logger.ErrorfContext(r.Context(), "[AUTH] Code exchange failed: %v", err)
		return "", fmt.Errorf("code exchange failed: %s", err.Error())
	}

	logger.DebugContext(r.Context(), "[AUTH] Fetching user info from Google")
	// 這裡刻意使用 context.Background() 而非 r.Context()：本次下游請求因此不受
	// 用戶端取消影響，即使呼叫端已斷線也能完成並取得 email。代價是使用者離開
	// 後，這次對 Google 的請求仍會執行完畢。
	client := OauthConfig.Client(context.Background(), token)
	// 使用 oauth2 的 client 而非裸 http.Client：它會在 token 過期時自動以
	// refresh_token 換發新 token，並在需要時於請求前先做一次更新。
	resp, err := client.Get(userInfoURL)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[AUTH] Failed getting user info: %v", err)
		return "", fmt.Errorf("failed getting user info: %s", err.Error())
	}
	// 必須 Close：這是 http.ResponseWriter 慣例的反面案例，body 未關閉時底層
	// 連線不會被回收回 keep-alive 池，長時間執行會耗盡可用連線。
	// Close 的錯誤忽略是慣例做法，這裡沒有可採取的補救動作。
	defer resp.Body.Close()

	// 4xx/5xx 一律視為失敗。刻意先把 body 讀出來寫進日誌：Google 的錯誤細節
	// （error、error_description）只出現在 body，是排查授權問題唯一有效的線索。
	// io.ReadAll 的錯誤忽略是刻意的，此時 body 內容已無意義，重點在狀態碼。
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logger.ErrorfContext(r.Context(), "[AUTH] User info request failed: status=%d body=%s", resp.StatusCode, string(body))
		// 回傳的 error 只帶狀態碼、不含 body，避免上游細節一路傳到 HTTP 回應。
		return "", fmt.Errorf("user info request failed: status %d", resp.StatusCode)
	}

	// userinfo 回應只有數百 bytes，整份讀進記憶體再解析即可，不需串流處理。
	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		// 常見於連線在讀取途中被中斷（代理逾時、來源端提前關閉）。
		logger.ErrorfContext(r.Context(), "[AUTH] Failed reading user info response body: %v", err)
		return "", fmt.Errorf("failed reading response body: %s", err.Error())
	}

	// 只關心 email 欄位，回應中的其餘欄位由 encoding/json 忽略。
	// 使用 struct 而非 map 同樣是為了讓「哪些欄位被信任」明確寫在型別上。
	var user GoogleUser
	if err := json.Unmarshal(contents, &user); err != nil {
		// 回應不保證是 JSON：上游被閘道器擋下時可能回傳 HTML 錯誤頁，
		// 因此把無法解析的原始內容一併記入日誌才有診斷價值。
		logger.ErrorfContext(r.Context(), "[AUTH] Failed unmarshaling user info: error=%v body=%s", err, string(contents))
		return "", fmt.Errorf("failed unmarshaling user info: %s", err.Error())
	}

	// 記錄 email 是為了留下登入軌跡；配合 logger 的 user_email metadata，
	// 可以從日誌還原「哪個帳號在哪個時間完成登入」。
	// 這裡不額外檢查 Email 是否為空，也不檢查 Google 回應中的 verified_email：
	// 目前以「已成功申請並取得 email scope」作為信任前提，屬信任假設而非驗證。
	logger.InfofContext(r.Context(), "[AUTH] Successfully retrieved user email: email=%s", user.Email)
	return user.Email, nil
}
