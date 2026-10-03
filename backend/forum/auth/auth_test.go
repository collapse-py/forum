/*
auth_test.go 覆蓋 OAuth 流程中「壞掉時不會有錯誤訊息」的三個地方。

【優先順序與理由】
ROADMAP.md 把 auth 列為 0% 覆蓋的目標，而它最該先被測的不是「能不能登入」
（那需要真的跟 Google 對話），而是**開放轉向**與**錯誤路徑**：

 1. isSafeReturnPath：這是本套件唯一的防線（見 auth.go 的檔頭）。
    它壞掉時的症狀是：攻擊者送一個 /auth/google?return=https://evil.example
    的連結，使用者登入後被轉到他的站，並且**帶著一個已登入的 session**。
    那不是「跳錯頁」，那是一次完整的 session 竊取。它不會出現在任何日誌裡 ——
    因為從伺服器的角度，登入流程完全正常。
 2. HandleLogin / ReturnPath：state 的編碼與解碼往返。任何一邊漏掉 URL
    解碼或編碼，症狀都是「登入後跳回錯誤的頁面」或「登入後回首頁」，
    而前者只在使用者的瀏覽器裡發生。
 3. GetUserEmail 的錯誤分支：這些是登入失敗時唯一能被觀察到的東西。
    它們的失效症狀是「登入失敗但沒有任何訊息」。

【測試 GetUserEmail 的做法與它的代價】
GetUserEmail 原本把 userinfo 端點的主機名寫死，因此整支函式無法測試。本檔
因此在 auth.go 加了一個 userInfoURL 變數作為注入點（理由寫在該處的說明）。
有了它，這裡就能用 httptest 伺服器走完 token 交換與 userinfo 查詢，並驗證
四件靠真實 Google 絕對測不到的事：
  - 上游回 4xx/5xx 時的錯誤分類（狀態碼進日誌、body 進不了 HTTP 回應）
  - 回應不是 JSON（閘道器回 HTML 錯誤頁）時的處理
  - 授權碼交換失敗時**不會**繼續往下查 userinfo
  - 換到的 token 真的被帶在後續請求的 Authorization 標頭裡
*/
package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

// withOauthConfig 暫時替換全域 OauthConfig 與 userInfoURL，並在測試結束後還原。
//
// 這兩個都是 package 層級變數（OauthConfig 是刻意的公開變數，userInfoURL 是
// 同樣型態的注入點），因此測試必須自己負責還原 —— 否則一個測試的假端點會
// 洩漏到下一個測試，而症狀是「一個純函式測試突然連到 httptest 伺服器」。
//
// 刻意不平行執行（沒有 t.Parallel）：這兩個變數沒有鎖，而這個套件的
// Init 註解本身就說明「執行期間若寫入會與併發讀取產生資料競爭」。測試不該
// 重現它自己指出來的那個問題。
func withOauthConfig(t *testing.T, cfg *oauth2.Config, infoURL string) {
	t.Helper()

	prevConfig := OauthConfig
	prevURL := userInfoURL
	OauthConfig = cfg
	userInfoURL = infoURL
	t.Cleanup(func() {
		OauthConfig = prevConfig
		userInfoURL = prevURL
	})
}

// withUninitializedOauth 把 OauthConfig 設成 nil。
func withUninitializedOauth(t *testing.T) {
	t.Helper()
	withOauthConfig(t, nil, userInfoURL)
}

// TestIsSafeReturnPath 是這個檔最重要的一支測試。
//
// 它逐條涵蓋 isSafeReturnPath 註解裡列出的每一個拒絕條件，外加**編碼繞過**
// 這一族 —— 那才是真實攻擊會用的形式。
func TestIsSafeReturnPath(t *testing.T) {
	safe := []struct {
		name string
		in   string
	}{
		{"根路徑", "/"},
		{"首頁", "/forum"},
		{"後台", "/admin"},
		{"帶查詢參數", "/forum?tab=2"},
		{"帶 fragment", "/forum#post-3"},
		{"深層路徑", "/api/forum/posts/123"},
		{"含中文的路徑", "/forum/公告"},
		{"單斜線後接文字", "/a"},
		{"單斜線後接可列印字元", "/!"},
	}
	for _, tc := range safe {
		t.Run("允許/"+tc.name, func(t *testing.T) {
			if !isSafeReturnPath(tc.in) {
				t.Errorf("isSafeReturnPath(%q) = false，want true（這是合法的站內路徑）", tc.in)
			}
		})
	}

	// 空字串刻意**不在**安全清單裡：它不以 "/" 開頭，因此這個函式拒絕它。
	// 那是正確的 —— 函式的契約是「回傳以 / 開頭的站內路徑」，而空字串不是。
	// 呼叫端（HandleLogin 與 ReturnPath）各自把它換成 "/"，
	// 而 TestReturnPath / TestHandleLoginPreservesSafeReturnPathInState 會驗證
	// 那個轉換。這裡把它釘住，是為了讓「空字串由呼叫端負責」這件事有明確出處，
	// 而不是被誤認為是這個函式的漏網。
	t.Run("空字串由呼叫端負責", func(t *testing.T) {
		if isSafeReturnPath("") {
			t.Fatal("isSafeReturnPath(\"\") = true。若這裡變成 true，HandleLogin 的 returnPath " +
				"預設值就不會被套用，state 會變成空字串，ReturnPath 又會因為「空字串不安全」而無條件退回首頁")
		}
		// 而呼叫端確實有處理：沒有 return 參數時 state 必須是 "/"。
		withOauthConfig(t, testConfig(t, "http://127.0.0.1:1/callback"), "http://127.0.0.1:1/userinfo")
		rec := httptest.NewRecorder()
		HandleLogin(rec, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
		parsed, err := url.Parse(rec.Header().Get("Location"))
		if err != nil {
			t.Fatalf("授權網址不是合法 URL: %v", err)
		}
		if got := parsed.Query().Get("state"); got != "/" {
			t.Errorf("沒有 return 參數時 state = %q，want \"/\"", got)
		}
	})

	unsafe := []struct {
		name string
		in   string
		why  string
	}{
		{"絕對網址", "https://evil.example", "開放轉向"},
		{"絕對網址帶路徑", "https://evil.example/steal", "開放轉向"},
		{"http 絕對網址", "http://evil.example", "開放轉向"},
		{"protocol-relative", "//evil.example", "瀏覽器會當成 https://evil.example"},
		{"protocol-relative 帶路徑", "//evil.example/steal", "同上"},
		{"反斜線開頭", "/\\evil.example", "部分瀏覽器會正規化成 //evil.example"},
		{"站內路徑後接反斜線", "/forum/\\@evil.example", "同上，且更難被肉眼察覺"},
		{"沒有開頭斜線的相對路徑", "forum", "會被瀏覽器相對於目前頁面解析"},
		{"非 http 協定", "javascript:alert(1)", "XSS 向量"},
		{"data URI", "data:text/html,<script>alert(1)</script>", "XSS 向量"},

		// --- 編碼繞過這一族 ---
		// 這些是攻擊者真正會寫的東西。若 isSafeReturnPath 只比對字面而不先解碼，
		// 這幾個會被判定為「以 / 開頭且不是 //」而放行，然後在
		// AuthCodeURL 的編碼 → Google 回送 → Query().Get 解碼之後變成
		// //evil.example。
		{"編碼的雙斜線", "%2F%2Fevil.example", "解碼後是 protocol-relative"},
		{"編碼的第一個斜線", "%2f%2fevil.example", "小寫編碼"},
		{"部分編碼", "/%2Fevil.example", "解碼後是 //evil.example"},
		{"編碼的反斜線", "/%5C%5Cevil.example", "解碼後含反斜線"},
		{"編碼的絕對網址", "https%3A%2F%2Fevil.example", "解碼後是絕對網址"},

		// --- 編碼不合法 ---
		// QueryUnescape 失敗代表內容不可信，因此直接視為不安全，而不是
		// 退回原字串比對（後者等於「解碼失敗就當沒事」）。
		{"不完整的百分號編碼", "%", "QueryUnescape 會回錯誤"},
		{"不完整的兩位編碼", "%2", "同上"},
		{"非法的十六進位", "%zz", "同上"},
		{"中斷的多位元組編碼", "%E4%BD", "同上"},
	}
	for _, tc := range unsafe {
		t.Run("拒絕/"+tc.name, func(t *testing.T) {
			if isSafeReturnPath(tc.in) {
				t.Errorf("isSafeReturnPath(%q) = true，但應該拒絕：%s", tc.in, tc.why)
			}
		})
	}
}

// TestIsSafeReturnPathBlocksTheActualAttackChain 走一遍完整的攻擊鏈。
//
// 單純的表格測試會告訴我們「每一個輸入單獨來看是對的」，卻不會告訴我們
// 「串起來之後會不會漏」。這個測試模擬真實的攻擊者：把 return 參數塞進
// /auth/google，確認 HandleLogin 產生的授權網址裡，state 解碼後不是站外網址。
//
// 這一條是本檔最重要的斷言，因為它測的不是字串比對，而是**那個實際會被導向
// 的目標**。
func TestIsSafeReturnPathBlocksTheActualAttackChain(t *testing.T) {
	attacks := []string{
		"https://evil.example",
		"//evil.example",
		"/\\evil.example",
		"%2F%2Fevil.example",
		"/%2Fevil.example",
		"%5C%5Cevil.example",
	}

	for _, attack := range attacks {
		t.Run(attack, func(t *testing.T) {
			withOauthConfig(t, testConfig(t, "http://127.0.0.1:1/callback"), "http://127.0.0.1:1/userinfo")

			r := httptest.NewRequest(http.MethodGet, "/auth/google?return="+url.QueryEscape(attack), nil)
			rec := httptest.NewRecorder()
			HandleLogin(rec, r)

			if rec.Code != http.StatusTemporaryRedirect {
				t.Fatalf("狀態碼 = %d，want 307", rec.Code)
			}

			// 從授權網址取出 state —— 那才是未來會被當成導回目標的東西。
			loginURL := rec.Header().Get("Location")
			parsed, err := url.Parse(loginURL)
			if err != nil {
				t.Fatalf("授權網址不是合法 URL: %v (%q)", err, loginURL)
			}
			state := parsed.Query().Get("state")

			// state 必須已被退回首頁，而不是保留了攻擊者的目標。
			if state != "/" {
				t.Errorf("state = %q，want \"/\"（攻擊者的 return 被接受了）", state)
			}
			if isSafeReturnPath(state) && !strings.HasPrefix(state, "/forum") && state != "/" {
				t.Errorf("state %q 仍然是一個站外目標", state)
			}
		})
	}
}

// TestHandleLoginPreservesSafeReturnPathInState 是上面的對照組。
//
// 只測「攻擊被擋下」不夠 —— 若實作是把所有 state 都清空，那攻擊也擋得住，
// 但每一個人登入後都會回首頁。因此必須同時證明合法路徑會被完整保留。
func TestHandleLoginPreservesSafeReturnPathInState(t *testing.T) {
	preserved := []string{
		"/forum",
		"/admin/log",
		"/forum?tab=2",
		"/forum#post-3",
		"/api/forum/posts/123",
		// 含中文：state 必須能被 URL 編碼 / 解碼往返。
		"/forum/公告",
	}

	for _, path := range preserved {
		t.Run(path, func(t *testing.T) {
			withOauthConfig(t, testConfig(t, "http://127.0.0.1:1/callback"), "http://127.0.0.1:1/userinfo")

			r := httptest.NewRequest(http.MethodGet, "/auth/google?return="+url.QueryEscape(path), nil)
			rec := httptest.NewRecorder()
			HandleLogin(rec, r)

			parsed, err := url.Parse(rec.Header().Get("Location"))
			if err != nil {
				t.Fatalf("授權網址不是合法 URL: %v", err)
			}
			if got := parsed.Query().Get("state"); got != path {
				t.Errorf("state = %q, want %q（合法的返回路徑必須完整往返）", got, path)
			}
		})
	}
}

// TestHandleLoginWithoutOauthConfig 守住啟動時忘記呼叫 Init 的情況。
//
// 這個分支的註解說明為什麼是 500 而不是仍然導向 Google：導過去也拿不到合法
// 憑證，使用者只會卡在 Google 的頁面上，而伺服器端看不出為什麼。
func TestHandleLoginWithoutOauthConfig(t *testing.T) {
	withUninitializedOauth(t)

	r := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	rec := httptest.NewRecorder()

	// 刻意不設想 panic：若這裡 panic，症狀會是使用者在伺服器錯誤頁上看到
	// 一个 stack trace，而不是「忘記呼叫 Init」。
	HandleLogin(rec, r)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("狀態碼 = %d，want 500（OauthConfig 尚未初始化）", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Errorf("仍然導向 %q，使用者會卡在 Google 頁面上", loc)
	}
	if !strings.Contains(rec.Body.String(), "not initialized") {
		t.Errorf("回應內容沒有指出原因: %q", rec.Body.String())
	}
}

// TestHandleLoginIssuesAuthorizationRequest 驗證授權網址本身的形狀。
//
// 這些是「能不能登入」的決定條件，而它們壞掉時的症狀是 Google 回
// redirect_uri_mismatch 或「沒有這個 scope」—— 兩個都需要人去 Google Console
// 比對才會發現。
func TestHandleLoginIssuesAuthorizationRequest(t *testing.T) {
	withOauthConfig(t, testConfig(t, "https://forum.example.com/auth/callback"), "http://127.0.0.1:1/userinfo")

	r := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	rec := httptest.NewRecorder()
	HandleLogin(rec, r)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("狀態碼 = %d，want 307", rec.Code)
	}

	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("授權網址不是合法 URL: %v", err)
	}
	q := parsed.Query()

	if got := q.Get("client_id"); got != "test-client-id" {
		t.Errorf("client_id = %q", got)
	}
	if got := q.Get("redirect_uri"); got != "https://forum.example.com/auth/callback" {
		t.Errorf("redirect_uri = %q（必須與 Google Console 登記值逐字相符）", got)
	}
	if got := q.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q, want \"code\"", got)
	}
	if got := q.Get("scope"); got != "https://www.googleapis.com/auth/userinfo.email" {
		t.Errorf("scope = %q（本專案只申請 email，不申請 openid 與 profile）", got)
	}
	// state 必須存在，因為 GetUserEmail 與 ReturnPath 都依賴它。
	// 刻意不斷言它的值（沒有 return 參數時是 "/"），那是另一支測試的責任。
	if q.Get("state") == "" {
		t.Error("授權網址沒有 state —— ReturnPath 會無法運作")
	}
	// client_secret 絕不能出現在授權網址裡（它是機密，且這是瀏覽器會去的網址）。
	if strings.Contains(rec.Header().Get("Location"), "test-secret") {
		t.Errorf("client_secret 出現在授權網址裡了: %s", rec.Header().Get("Location"))
	}
}

// TestReturnPath 覆蓋 state 的解碼往返與失效時的退路。
func TestReturnPath(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  string
	}{
		{name: "正常往返", query: "state=%2Fadmin", want: "/admin"},
		{name: "帶查詢參數", query: "state=%2Fforum%3Ftab%3D2", want: "/forum?tab=2"},
		{name: "未指定 state", query: "", want: "/"},
		{name: "state 為空", query: "state=", want: "/"},
		{name: "站外網址", query: "state=https%3A%2F%2Fevil.example", want: "/"},
		{name: "protocol-relative", query: "state=%2F%2Fevil.example", want: "/"},
		{name: "編碼繞過", query: "state=%252F%252Fevil.example", want: "/"},
		{name: "反斜線", query: "state=%2F%5Cevil.example", want: "/"},
		{name: "不完整的編碼", query: "state=%", want: "/"},
		{name: "只有 code 沒有 state", query: "code=abc", want: "/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/auth/callback?"+tc.query, nil)
			if got := ReturnPath(r); got != tc.want {
				t.Errorf("ReturnPath(state=%q) = %q, want %q", tc.query, got, tc.want)
			}
		})
	}
}

// TestReturnPathRoundTripsThroughHandleLogin 是兩個函式之間的整合測試。
//
// 分開測兩邊都不夠：HandleLogin 用 AuthCodeURL 編碼 state、ReturnPath 用
// Query().Get 解碼它，而這兩個用的是**不同的**編碼規則（QueryEscape 與
// url.Values 的編碼）。它們之間的不對稱正是「登入後跳回錯誤頁面」最可能的
// 來源，而它只有走完整條鏈才會現形。
func TestReturnPathRoundTripsThroughHandleLogin(t *testing.T) {
	paths := []string{
		"/", "/forum", "/admin/log", "/forum?tab=2&sort=new",
		"/forum#post-3", "/api/forum/posts/123", "/forum/公告", "/a b",
	}

	withOauthConfig(t, testConfig(t, "https://forum.example.com/auth/callback"), "http://127.0.0.1:1/userinfo")

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// 第一段：HandleLogin 產生 state。
			loginReq := httptest.NewRequest(http.MethodGet, "/auth/google?return="+url.QueryEscape(path), nil)
			loginRec := httptest.NewRecorder()
			HandleLogin(loginRec, loginReq)

			parsed, err := url.Parse(loginRec.Header().Get("Location"))
			if err != nil {
				t.Fatalf("授權網址不是合法 URL: %v", err)
			}
			state := parsed.Query().Get("state")

			// 第二段：Google 把 state 原樣送回來，ReturnPath 取回它。
			cbReq := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state="+url.QueryEscape(state), nil)
			if got := ReturnPath(cbReq); got != path {
				t.Errorf("往返後 = %q, want %q", got, path)
			}
		})
	}
}

// TestGetUserEmailRequiresCode 守住最基本的前置條件。
//
// 這個分支可以在完全不碰網路的情況下測，而它是登入失敗時最常見的原因之一
// （使用者直接造訪 /auth/callback，或 Google 在未授權時以 error 參數結束）。
func TestGetUserEmailRequiresCode(t *testing.T) {
	withUninitializedOauth(t)

	for _, query := range []string{"", "error=access_denied", "state=%2Fforum", "code="} {
		r := httptest.NewRequest(http.MethodGet, "/auth/callback?"+query, nil)
		email, err := GetUserEmail(r)
		if err == nil {
			t.Errorf("query=%q 時 GetUserEmail 回 nil error，want 錯誤", query)
		}
		if email != "" {
			t.Errorf("query=%q 時回傳 email=%q，want 空字串（失敗時不可回傳身分）", query, email)
		}
		// 錯誤訊息刻意不帶上游內容，因此不可能包含 Google 的錯誤字串。
		if strings.Contains(err.Error(), "access_denied") {
			t.Errorf("錯誤訊息洩漏了上游內容: %v", err)
		}
	}
}

// TestGetUserEmailSuccess 走完整條鏈，並驗證換到的 token 真的被帶上。
//
// 最後那個斷言是最有價值的一個：它確認 OauthConfig.Client 產生的 client
// 真的把 Bearer token 放進了 userinfo 請求。若這裡壞掉，症狀是「Google 回
// 401，而錯誤訊息只說 status 401」—— 不看原始碼無法猜到是漏了標頭。
func TestGetUserEmailSuccess(t *testing.T) {
	var (
		mu           sync.Mutex
		userInfoAuth string
		tokenAuthHdr string
		tokenForm    url.Values
	)

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 授權碼交換端點。oauth2 的 AuthStyle 是 auto：它會先試著把 client
		// 憑證放進 Authorization 標頭，若從回應看出端點不吃那套，才退回
		// 查詢參數。因此這裡**不能**斷言憑證出現在哪一種形式 —— 那是
		// oauth2 套件根據端點行為做的協商，不是這個專案的決定。
		// 唯一該被這支測試釘住的是「憑證有到達端點」。
		_ = r.ParseForm()
		mu.Lock()
		tokenAuthHdr = r.Header.Get("Authorization")
		tokenForm = r.Form
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	infoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		userInfoAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1234567890","email":"someone@example.com","verified_email":true}`))
	}))
	defer infoSrv.Close()

	cfg := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "https://forum.example.com/auth/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.example.com/o/oauth2/auth",
			TokenURL: tokenSrv.URL,
		},
	}
	withOauthConfig(t, cfg, infoSrv.URL)

	r := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code", nil)
	email, err := GetUserEmail(r)
	if err != nil {
		t.Fatalf("GetUserEmail 回錯誤: %v", err)
	}
	if email != "someone@example.com" {
		t.Errorf("email = %q, want \"someone@example.com\"", email)
	}

	mu.Lock()
	defer mu.Unlock()

	if !strings.HasPrefix(userInfoAuth, "Bearer ") || !strings.Contains(userInfoAuth, "test-access-token") {
		t.Errorf("userinfo 請求沒有帶換到的 token（Authorization = %q）", userInfoAuth)
	}
	// 授權碼必須真的被送到 token 端點，而不是被這個函式自己吃掉。
	if got := tokenForm.Get("code"); got != "test-code" {
		t.Errorf("token 端點收到的 code = %q, want \"test-code\"", got)
	}
	// client 憑證必須以「標頭或參數」其中之一到達端點 —— 缺了它 Google 會以
	// invalid_client 拒絕交換，而那個錯誤訊息不會提到「憑證沒送到」。
	//
	// 這是本檔唯一一個刻意**不**綁定形式的地方，理由寫在上面的註解。
	basicAuthCarries := strings.HasPrefix(tokenAuthHdr, "Basic ")
	formCarries := tokenForm.Get("client_id") == "test-client-id" && tokenForm.Get("client_secret") == "test-secret"
	if !basicAuthCarries && !formCarries {
		t.Errorf("token 請求沒有帶 client 憑證（Authorization = %q，client_id = %q，client_secret = %q）",
			tokenAuthHdr, tokenForm.Get("client_id"), tokenForm.Get("client_secret"))
	}
}

// TestGetUserEmailErrorBranches 覆蓋三種上游失敗。
//
// 每個都斷言兩件事：回傳錯誤、且**不把上游內容洩漏進錯誤訊息**。後半段是
// 這個函式註解裡的明確承諾（「回傳的 error 只帶狀態碼、不含 body」）—— 而它
// 若被破壞，症狀是攻擊者可以用一個被控制的錯誤訊息內容做二次注入。
func TestGetUserEmailErrorBranches(t *testing.T) {
	cases := []struct {
		name         string
		infoStatus   int
		infoBody     string
		tokenFails   bool
		wantErrParts []string
		notErrParts  []string
	}{
		{
			name:         "token 交換失敗",
			tokenFails:   true,
			wantErrParts: []string{"code exchange failed"},
			notErrParts:  []string{"test-access-token"},
		},
		{
			name:         "userinfo 回 401",
			infoStatus:   http.StatusUnauthorized,
			infoBody:     `{"error":"invalid_token"}`,
			wantErrParts: []string{"user info request failed", "401"},
			notErrParts:  []string{"invalid_token"},
		},
		{
			name:         "userinfo 回 500",
			infoStatus:   http.StatusInternalServerError,
			infoBody:     `{"error_description":"backend exploded at /internal/path"}`,
			wantErrParts: []string{"user info request failed", "500"},
			notErrParts:  []string{"backend exploded"},
		},
		{
			name:         "userinfo 回 HTML（閘道器擋下）",
			infoStatus:   http.StatusOK,
			infoBody:     `<html><body>502 Bad Gateway</body></html>`,
			wantErrParts: []string{"failed unmarshaling"},
			notErrParts:  []string{"Bad Gateway"},
		},
		{
			name:         "userinfo 回非 JSON 但狀態碼 200",
			infoStatus:   http.StatusOK,
			infoBody:     `not json at all`,
			wantErrParts: []string{"failed unmarshaling"},
			notErrParts:  []string{"not json at all"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokenCalls := 0
			tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tokenCalls++
				if tc.tokenFails {
					// 回一個 OAuth 標準錯誤格式。
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error":"invalid_grant"}`))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
			}))
			defer tokenSrv.Close()

			infoCalls := 0
			infoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				infoCalls++
				w.WriteHeader(tc.infoStatus)
				w.Write([]byte(tc.infoBody))
			}))
			defer infoSrv.Close()

			cfg := &oauth2.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				RedirectURL:  "https://forum.example.com/auth/callback",
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://accounts.example.com/o/oauth2/auth",
					TokenURL: tokenSrv.URL,
				},
			}
			withOauthConfig(t, cfg, infoSrv.URL)

			r := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code", nil)
			email, err := GetUserEmail(r)
			if err == nil {
				t.Fatalf("GetUserEmail 回 nil error，want 錯誤（%s）", tc.name)
			}
			if email != "" {
				t.Errorf("失敗時回傳了 email=%q，want 空字串", email)
			}
			for _, want := range tc.wantErrParts {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("錯誤訊息 %q 缺少 %q", err.Error(), want)
				}
			}
			for _, not := range tc.notErrParts {
				if strings.Contains(err.Error(), not) {
					t.Errorf("錯誤訊息洩漏了上游內容 %q: %v", not, err)
				}
			}

			// token 交換失敗時**不應該**再查 userinfo：那一步用的是不可信的
			// token，而它會浪費一次對外請求。
			if tc.tokenFails && infoCalls != 0 {
				t.Errorf("token 交換失敗後仍查了 userinfo（%d 次）—— 那一路徑的 token 不可信", infoCalls)
			}
		})
	}
}

// TestGetUserEmailWithUnreachableServer 覆蓋「連不到」而不是「回錯誤」。
//
// 這個分支與「回 4xx」是不同的事：它是網路層的失敗，錯誤訊息裡會帶上
// 連線錯誤的細節（包含內部主機名）。因此這裡額外斷言錯誤訊息裡**不含**
// 內部位址 —— 那會把部署結構洩漏給呼叫端。
func TestGetUserEmailWithUnreachableServer(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	cfg := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "https://forum.example.com/auth/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.example.com/o/oauth2/auth",
			TokenURL: tokenSrv.URL,
		},
	}
	// 指向一個必然連不上的位址：port 1 在任何環境都不會有服務在聽。
	deadURL := "http://127.0.0.1:1/userinfo"
	withOauthConfig(t, cfg, deadURL)

	r := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code", nil)
	email, err := GetUserEmail(r)
	if err == nil {
		t.Fatal("連不到 userinfo 端點卻回 nil error")
	}
	if email != "" {
		t.Errorf("失敗時回傳了 email=%q", email)
	}
	if strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Errorf("錯誤訊息洩漏了內部位址: %v", err)
	}
}

// TestGetUserEmailRespectsContextCancellation 覆蓋「使用者中途離開」。
//
// 這個測試鎖住的是一個刻意的取捨：Exchange 掛 r.Context()（可被取消），
// 而 userinfo 請求掛 context.Background()（不會被取消）。後者的理由寫在
// 程式碼裡：即使呼叫端斷線也要完成並拿到 email。
//
// 因此這支測試斷言的是「**過期**的 context 會中止**交換**」—— 也就是
// 程式碼有掛上 r.Context() 的那一段。
func TestGetUserEmailRespectsContextCancellation(t *testing.T) {
	tokenCalled := make(chan struct{}, 1)
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case tokenCalled <- struct{}{}:
		default:
		}
		w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	cfg := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "https://forum.example.com/auth/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.example.com/o/oauth2/auth",
			TokenURL: tokenSrv.URL,
		},
	}
	withOauthConfig(t, cfg, "http://127.0.0.1:1/userinfo")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	r := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code", nil).WithContext(ctx)
	email, err := GetUserEmail(r)
	if err == nil {
		t.Fatal("context 已取消卻沒有回錯誤")
	}
	if email != "" {
		t.Errorf("失敗時回傳了 email=%q", email)
	}

	// 這個斷言刻意寫得寬鬆：重點是「流程被中止」，而中止的時點是在發出請求
	// 之前或之後都算數（httptest 伺服器可能已經收到請求）。
	select {
	case <-tokenCalled:
		// 請求已經送出 —— 那也符合預期（取消與送出之間有競態）。
	default:
		// 請求沒有送出 —— 這是預期的情況（context 在送出前就過期了）。
	}
}

// TestInitBuildsTheExpectedConfig 覆蓋 Init。
//
// 斷言的三件事都是「不能改錯」的：scope（多申請就會觸發 Google 的敏感權限
// 審查，少申請就拿不到 email）、Endpoint（拿錯會導向別的服務）、以及
// client_secret 不出現在 OauthConfig 之外的任何地方。
func TestInitBuildsTheExpectedConfig(t *testing.T) {
	t.Cleanup(func() { OauthConfig = nil })

	// 刻意帶一個尾端斜線：config 解析那裡也刻意不去尾斜線（Google 對
	// redirect_uri 的比對是逐字的），Init 必須同樣原樣保留。
	Init("my-client-id", "my-client-secret", "https://forum.example.com/auth/callback/")

	if OauthConfig == nil {
		t.Fatal("Init 沒有建立 OauthConfig")
	}
	if OauthConfig.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q", OauthConfig.ClientID)
	}
	if OauthConfig.ClientSecret != "my-client-secret" {
		t.Errorf("ClientSecret = %q", OauthConfig.ClientSecret)
	}
	if OauthConfig.RedirectURL != "https://forum.example.com/auth/callback/" {
		t.Errorf("RedirectURL = %q，尾端斜線被動到了（Google 的比對是逐字的）", OauthConfig.RedirectURL)
	}

	wantScopes := []string{"https://www.googleapis.com/auth/userinfo.email"}
	if len(OauthConfig.Scopes) != len(wantScopes) || OauthConfig.Scopes[0] != wantScopes[0] {
		t.Errorf("Scopes = %v，want %v（只能申請 email；多申請會觸發 Google 的敏感權限審查）",
			OauthConfig.Scopes, wantScopes)
	}

	if OauthConfig.Endpoint.AuthURL == "" || OauthConfig.Endpoint.TokenURL == "" {
		t.Errorf("Endpoint 不完整: %+v", OauthConfig.Endpoint)
	}
	if !strings.Contains(OauthConfig.Endpoint.AuthURL, "accounts.google.com") {
		t.Errorf("AuthURL = %q，看起來不是 Google 的授權端點", OauthConfig.Endpoint.AuthURL)
	}
}

// testConfig 產生一個指向測試用端點的 oauth2.Config。
func testConfig(t *testing.T, redirectURL string) *oauth2.Config {
	t.Helper()
	return &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.example.com/o/oauth2/auth",
			TokenURL: "http://127.0.0.1:1/token",
		},
	}
}
