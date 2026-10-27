/*
auth 封裝以 Google 為唯一識別證供應者的 OAuth2 授權碼流程（Authorization Code Flow）。

【對外介面】
Init 建立全域 oauth2.Config，必須在服務開始處理請求前呼叫一次。
AuthorizeURL 產生要導向 Google 的授權網址（帶 state 與 PKCE 的 code_challenge）。
IsSafeReturnPath 判斷一個「登入後要回到哪一頁」的值是否為站內相對路徑。
GetUserEmail 用回呼的 code（與当初的 code_verifier）換取 token，再向 Google 查詢 email。
GoogleUser 與 OauthConfig 是公開符號，供測試或其他套件引用。
路由與流程的編排都在 httpapi（見 httpapi/oauth_login.go 與 httpapi/auth_handlers.go）：
GET /auth/google 產生 nonce 與 verifier、把它們存進 Redis 並寫下一份綁定 cookie，
再以 state=nonce 導向 Google；GET /auth/callback 比對 state 與那份 cookie、從 Redis
取回（並刪除）nonce，然後呼叫 GetUserEmail。

【主要依賴】
golang.org/x/oauth2 及其 oauth2/google 套件，google.Endpoint 內含 Google 的
授權端點與 token 端點，因此本檔不需要自行組任何 Google URL。
golang.org/x/oauth2 內建的 context 感知 HTTP client。
forum/forum/logger，支援帶入 request context，日誌可附帶 request_id、ip、user_email。

【關鍵設計決策與限制】
state 是一次隨機 nonce，不是「回跳路徑」。它綁定發起登入的瀏覽器（同一個 nonce 存在
Redis 裡，並寫進一頁只給 /auth 用的 HttpOnly cookie），回呼時兩者必須相符且 nonce
尚未被用過 —— 這是 login CSRF 的防線：攻擊者無法「用自己的帳號完成登入、把登入態寫進
受害者的瀏覽器」，因為他拿不到受害者那份 cookie 對應的 nonce。回跳路徑因此改存在 Redis
裡（寫入前已通過 IsSafeReturnPath），不再由 state 攜帶。
配套的是 PKCE（code_verifier）：授權網址帶 S256 challenge，換 token 時帶 verifier，
讓被竊取的授權碼無法在另一處被換成 token。
本套件不碰 Redis 也不碰 cookie —— 那些是 httpapi 的工作，本檔只提供「產生網址」與
「用 code 換 email」。

只申請 userinfo.email 這一個 scope，不申請 openid 與 profile。代價是拿不到使用者
名稱與頭像，論壇以 email 作為唯一識別；好處是不需要通過 Google 對敏感或受限
scope 的審查。

Init 寫入的 OauthConfig 是無保護的全域變數，設計上假設只在啟動時寫入一次、
之後唯讀。若執行期間仍可能寫入，會與併發讀取產生資料競爭，本套件未提供鎖定。

GetUserEmail 假設 Init 已經呼叫：OauthConfig 為 nil 時，OauthConfig.Exchange 會在
讀取 c.RedirectURL 時發生 nil 解參考而 panic。呼叫端（httpapi 的登入與回呼兩端）都先
檢查 nil，因此不會走到那一步。

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

	// UserInfoURL 是向 Google 查詢 email 的端點。
	//
	// 為什麼是變數而不是直接寫在 GetUserEmail 裡：GetUserEmail 是本套件唯一
	// 有分支邏輯（狀態碼檢查、JSON 解析、錯誤分類）的函式，而它的兩次對外
	// 請求都寫死主機名時，整支函式就無法測試 —— 測試只能真的去呼叫 Google。
	// oauth2 的 Endpoint（TokenURL / AuthURL）本來就是透過 OauthConfig 注入的，
	// 這裡讓 userinfo 端點有同樣的注入點，兩者就一致了。
	//
	// 它與 OauthConfig 同樣是公開符號：理由不是「外部需要改它」，而是「跨套件的
	// 測試要能注入假端點」（httpapi 的回呼測試就是這個使用者）。
	// 限制與 OauthConfig 相同：只在啟動時或測試中寫入，之後唯讀。
	UserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
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

// AuthorizeURL 產生要把瀏覽器導向 Google 的授權網址。
//
// state 是一次性 nonce（由呼叫端產生、存進 Redis，並綁定在那個瀏覽器的一份 cookie
// 上），不是回跳路徑 —— 見檔頭對 login CSRF 的說明。codeVerifier 非空時附上海報
// （code_challenge 與 method=S256），換 token 時必須帶回原本的 verifier，因此被竊取
// 的授權碼在別處換不出 token。
//
// AuthCodeURL 會對每個參數做 URL 編碼，nonce 與 challenge 因此不需要自行處理。
// 純函式：不寫回應、不寫資料庫、不發請求。
// OauthConfig 為 nil 時回空字串 —— 呼叫端必須把它當成錯誤處理，不要拿它去導向。
func AuthorizeURL(state, codeVerifier string) string {
	if OauthConfig == nil {
		return ""
	}
	if codeVerifier == "" {
		return OauthConfig.AuthCodeURL(state)
	}
	// S256ChallengeOption 一次寫出 code_challenge 與 code_challenge_method=S256，
	// 因此不必自行組那兩個參數。verifier 本身絕不進授權網址 —— 那正是它存在的理由。
	return OauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(codeVerifier))
}

// IsSafeReturnPath 判斷一個「登入後要回到哪一頁」的值是否為安全的站內相對路徑。
//
// 拒絕的條件與依據：
// QueryUnescape 失敗代表含有不合法的百分號編碼，無法確定其意圖，直接視為不安全。
// 不以 "/" 開頭，可能被用來跳到 "https://evil.example" 這類絕對網址（開放轉向）。
// 以 "//" 開頭，是 protocol-relative URL，瀏覽器會當成 "https://evil.example"。
// 含反斜線，因為部分瀏覽器與舊版 URL 解析器會把 "/\evil.example" 正規化成
// "//evil.example"，等於繞過上一條檢查。
//
// 它是公開符號：寫入端與讀取端（都在 httpapi）都要過它。回跳路徑存在 Redis 裡，
// 但那筆資料的產生與消費是兩個不同的請求，兩邊各自檢查才不用假設「寫進去的那次
// 一定檢查過」。
// 副作用：無，是純函式，可安全併行呼叫。
func IsSafeReturnPath(returnPath string) bool {
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

// GetUserEmail 處理 OAuth2 回呼的核心步驟：用 code（與當初授權網址的 verifier）換取
// token，再以 token 向 Google 查詢使用者 email。
//
// 參數：r 必須是 /auth/callback 的請求，且帶有 Google 回傳的 "code" 查詢參數。
// 換 token 的請求掛在 r.Context() 上，使用者中途關閉分頁會直接取消該在途請求。
// codeVerifier 是發起登入時產生、存在 Redis 裡的那個 PKCE verifier（呼叫端從 state
// nonce 查出來）。傳空字串表示這次沒有 PKCE —— 那會退回不帶 verifier 的舊行為，
// 只應出現在測試裡；線上流程必須帶它。
//
// 回傳值：Google 帳號的 email；失敗時回傳空字串與非 nil 的 error。
//
// 錯誤條件：查詢字串沒有 code、token 交換失敗、userinfo 請求失敗、回應狀態碼
// 為 4xx/5xx、或回應內容無法解析成 JSON。錯誤訊息內含上游細節（token 交換的
// 失敗甚至帶著 Google 的回應本文），因此**只適合寫進日誌**：呼叫端若要回給
// 使用者，請改成固定字串。httpapi 的 handleGoogleCallback 現在正是這麼做。
//
// 副作用：對 oauth2.googleapis.com 與 googleapis.com 各發出一次對外 HTTPS 請求。
// 不寫資料庫、不設定 cookie。
func GetUserEmail(r *http.Request, codeVerifier string) (string, error) {
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
	// PKCE：把當初的 verifier 帶回給 token 端點。授權碼即使被竊取（瀏覽器延伸、
	// Referer、代理日誌），沒有 verifier 就換不出 token —— 這是 state nonce 之外
	// 的另一半，兩件事各自擋掉不同的竊取路徑。
	// 空字串時不帶任何選項（測試與非 PKCE 的呼叫），Google 不會因此失敗。
	exchangeOptions := []oauth2.AuthCodeOption{}
	if codeVerifier != "" {
		exchangeOptions = append(exchangeOptions, oauth2.VerifierOption(codeVerifier))
	}
	token, err := OauthConfig.Exchange(r.Context(), code, exchangeOptions...)
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
	resp, err := client.Get(UserInfoURL)
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
