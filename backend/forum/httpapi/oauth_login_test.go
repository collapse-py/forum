package httpapi

/*
oauth_login_test.go 釘住 OAuth 登入流程「對外看到什麼」的四件事：

 1. 授權網址帶的是隨機 nonce 與 PKCE challenge，回跳路徑與 verifier 都不在上面。
    這兩條若漏掉，症狀分別是 login CSRF（攻擊者能決定回跳目標、並把自己的登入態
    寫進受害者瀏覽器）與「被竊取的授權碼可在別處換出 token」，而日誌裡兩件事都不會
    顯得異常 —— 從伺服器的角度流程完全正常。
 2. 綁定 cookie 的形狀（HttpOnly、Path=/auth、SameSite=Lax）與它的值（就是那個 nonce）。
    它壞掉時的症状是「回呼永遠驗證失敗」，也就是所有人都登不進來。
 3. 回呼在 state 不可信時**不去換 token**。順序是這一層的重點：換 token 會送
    client_secret 出去，而且是唯一會花在 Google 上的請求。
 4. Redis 不可用時 fail closed（500），而不是放行一個沒有驗證的流程。

測試用假端點（httptest）取代 Google：OauthConfig 與 UserInfoURL 都是 auth 套件刻意
留的注入點。auth.OauthConfig 是全域變數，因此每支測試都要自己負責還原 —— 否則一個
測試的假端點會洩漏到下一個，而症状是「某個無關的測試突然連到 httptest 伺服器」。
*/

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"forum/forum/auth"

	"golang.org/x/oauth2"
)

// withOAuthEndpoints 把 auth 的全域設定指向兩支假端點（token 交換與 userinfo），
// 並在測試結束後還原。auth.OauthConfig 沒有鎖，因此這支檔案的測試不平行執行。
func withOAuthEndpoints(t *testing.T, tokenURL, infoURL string) {
	t.Helper()

	prevConfig := auth.OauthConfig
	prevInfo := auth.UserInfoURL
	auth.OauthConfig = &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "https://forum.example.com/auth/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.example.com/o/oauth2/auth",
			TokenURL: tokenURL,
		},
	}
	auth.UserInfoURL = infoURL
	t.Cleanup(func() {
		auth.OauthConfig = prevConfig
		auth.UserInfoURL = prevInfo
	})
}

// fakeGoogle 是一個「會記錄被呼叫幾次」的假 Google。token 端點的呼叫次數是第三條
// 測試的核心斷言（state 不可信時不該碰到它），因此計數必須可靠。
type fakeGoogle struct {
	mu        sync.Mutex
	tokenHits int
	userHits  int
	verifier  string
}

func (g *fakeGoogle) hits() (int, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.tokenHits, g.userHits
}

func (g *fakeGoogle) codeVerifier() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.verifier
}

func (g *fakeGoogle) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		g.mu.Lock()
		g.tokenHits++
		g.verifier = r.Form.Get("code_verifier")
		g.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
	}
}

func (g *fakeGoogle) userInfo(email string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		g.userHits++
		g.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","email":"` + email + `","verified_email":true}`))
	}
}

// startLogin 跑一次「發起登入」，回傳（nonce, verifier, 綁定 cookie, 授權網址）。
// 它是多個測試共用的第一段，因此寫成 helper 而不是在每支測試裡重寫一遍。
func startLogin(t *testing.T, srv *Server, returnParam string) (string, string, *http.Cookie, string) {
	t.Helper()

	path := "/auth/google"
	if returnParam != "" {
		path += "?return=" + url.QueryEscape(returnParam)
	}
	recorder := httptest.NewRecorder()
	srv.handleGoogleLogin(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("發起登入的回應 = %d, want 307；body=%s", recorder.Code, recorder.Body.String())
	}

	location := recorder.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("授權網址不是合法 URL: %v (%q)", err, location)
	}
	nonce := parsed.Query().Get("state")

	var bound *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == oauthStateCookieName {
			bound = cookie
		}
	}
	if bound == nil {
		t.Fatalf("回應沒有帶綁定 cookie，Location=%s", location)
	}
	stored, err := srv.mediaRedis.Get(context.Background(), oauthStateKey(nonce)).Result()
	if err != nil {
		t.Fatalf("Redis 裡找不到 state（nonce=%s）: %v", nonce, err)
	}
	var state oauthState
	if err := json.Unmarshal([]byte(stored), &state); err != nil {
		t.Fatalf("state 內容無法解析: %v", err)
	}
	return nonce, state.CodeVerifier, bound, location
}

// readStoredState 讀出 Redis 裡那一筆（不刪除）。測試用它檢查「寫進去的是什麼」。
func readStoredState(t *testing.T, srv *Server, nonce string) oauthState {
	t.Helper()
	value, err := srv.mediaRedis.Get(context.Background(), oauthStateKey(nonce)).Result()
	if err != nil {
		t.Fatalf("讀 state 失敗: %v", err)
	}
	var state oauthState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		t.Fatalf("state 內容無法解析: %v", err)
	}
	return state
}

// --- 發起端：授權網址與綁定 cookie 的形狀 ---

func TestOAuthLoginStoresNonceAndBindsBrowser(t *testing.T) {
	h := newCovHarness(t)
	withOAuthEndpoints(t, "http://127.0.0.1:1/token", "http://127.0.0.1:1/userinfo")

	nonce, verifier, bound, location := startLogin(t, h.srv, "/forum?tab=2")

	// nonce 是隨機值：48 個 hex 字元（24 位元組）。它不攜帶語意，因此值本身不該
	// 是任何使用者可控的東西。
	if len(nonce) != 48 || strings.ContainsAny(nonce, "/?") {
		t.Errorf("state = %q，想是一個 48 字元的隨機 hex", nonce)
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("授權網址非法: %v", err)
	}
	if got := parsed.Query().Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q, want S256（少了它 Google 會當成 plain）", got)
	}
	if got := parsed.Query().Get("code_challenge"); got == "" {
		t.Error("授權網址沒有 code_challenge —— PKCE 整個消失而流程看起來正常")
	}
	if got := oauth2.S256ChallengeFromVerifier(verifier); got != parsed.Query().Get("code_challenge") {
		t.Errorf("code_challenge 與 verifier 不相符（challenge = %q）", parsed.Query().Get("code_challenge"))
	}
	// 這兩樣東西絕不出現在第三方看得到的網址上。
	if strings.Contains(location, verifier) {
		t.Errorf("verifier 出現在授權網址裡: %s", location)
	}
	if strings.Contains(location, "test-secret") {
		t.Errorf("client_secret 出現在授權網址裡: %s", location)
	}
	if strings.Contains(location, "/forum") {
		t.Errorf("回跳路徑出現在授權網址裡: %s —— 那正是攻擊者可以控制的東西", location)
	}

	// 綁定 cookie：它就是把 nonce 綁在這個瀏覽器上。少了 HttpOnly 時 XSS 能拿走它；
	// Path 變大時它會在整站上來回送（多一個可被觀察的憑證）。
	if !bound.HttpOnly {
		t.Error("綁定 cookie 不是 HttpOnly")
	}
	if bound.Path != "/auth" {
		t.Errorf("綁定 cookie 的 Path = %q, want /auth（只有 /auth/* 需要它）", bound.Path)
	}
	if bound.SameSite != http.SameSiteLaxMode {
		t.Errorf("綁定 cookie 的 SameSite = %v, want Lax（回呼是頂層 GET 導覽）", bound.SameSite)
	}
	if bound.Value != nonce {
		t.Errorf("綁定 cookie 的值 = %q, want 與 state 相同 (%q)", bound.Value, nonce)
	}

	stored := readStoredState(t, h.srv, nonce)
	if stored.ReturnPath != "/forum?tab=2" {
		t.Errorf("存進 Redis 的回跳路徑 = %q, want /forum?tab=2", stored.ReturnPath)
	}
	if stored.CodeVerifier == "" {
		t.Error("Redis 裡沒有 verifier —— 回呼換不出 token")
	}
	if ttl := h.srv.mediaRedis.TTL(context.Background(), oauthStateKey(nonce)).Val(); ttl <= 0 || ttl > oauthStateTTL {
		t.Errorf("state 的 TTL = %v，必須在 0~%v 之間（沒有 TTL 時它會變成常駐的憑證）", ttl, oauthStateTTL)
	}
}

// 攻擊者控制的 return 必須被退回首頁 —— 這是「開放轉向」那條防線，與 nonce 無關。
func TestOAuthLoginStoresRootForUnsafeReturnPath(t *testing.T) {
	h := newCovHarness(t)
	withOAuthEndpoints(t, "http://127.0.0.1:1/token", "http://127.0.0.1:1/userinfo")

	for _, attack := range []string{
		"https://evil.example",
		"//evil.example",
		"/\\evil.example",
		"%2F%2Fevil.example",
		"/%2Fevil.example",
		"%5C%5Cevil.example",
		"",
	} {
		t.Run(attack, func(t *testing.T) {
			nonce, _, _, location := startLogin(t, h.srv, attack)
			if got := readStoredState(t, h.srv, nonce).ReturnPath; got != "/" {
				t.Errorf("攻擊者的 return %q 被存下了（Redis 裡 = %q，want \"/\"）", attack, got)
			}
			if strings.Contains(location, "evil.example") {
				t.Errorf("授權網址帶著攻擊者的目標: %s", location)
			}
		})
	}
}

// Redis 或 OAuth 設定不可用時必須 fail closed。導向 Google 再讓回呼失敗是最壞的
// 結果：使用者走完授權流程、被送回來、然後收到一個「流程已失效」，而伺服器端這一步
// 本來就還沒發生。
func TestOAuthLoginFailsClosed(t *testing.T) {
	h := newCovHarness(t)

	t.Run("沒有 Redis", func(t *testing.T) {
		withOAuthEndpoints(t, "http://127.0.0.1:1/token", "http://127.0.0.1:1/userinfo")
		h.srv.mediaRedis = nil
		recorder := httptest.NewRecorder()
		h.srv.handleGoogleLogin(recorder, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("回應 = %d, want 500（不能放過一個沒有記錄的流程）", recorder.Code)
		}
		if loc := recorder.Header().Get("Location"); loc != "" {
			t.Errorf("仍然導向 %q（使用者會卡在 Google 的授權頁上）", loc)
		}
	})

	t.Run("沒有 OAuth 設定", func(t *testing.T) {
		prev := auth.OauthConfig
		auth.OauthConfig = nil
		t.Cleanup(func() { auth.OauthConfig = prev })
		recorder := httptest.NewRecorder()
		h.srv.handleGoogleLogin(recorder, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("回應 = %d, want 500（忘記呼叫 auth.Init）", recorder.Code)
		}
		if loc := recorder.Header().Get("Location"); loc != "" {
			t.Errorf("仍然導向 %q", loc)
		}
	})
}

// --- 回呼端：state 的可信性 ---

// 攻擊者用自己的帳號走完登入，然後把那個流程的 state 送到受害者那里（沒有綁定
// cookie）。這一條釘住的是「站方不會因為收到一個 code 就簽發 session」。
func TestOAuthCallbackRejectsStateWithoutBindingCookie(t *testing.T) {
	h := newCovHarness(t)
	google := &fakeGoogle{}
	tokenSrv := httptest.NewServer(google.handler())
	defer tokenSrv.Close()
	infoSrv := httptest.NewServer(google.userInfo("attacker@example.com"))
	defer infoSrv.Close()
	withOAuthEndpoints(t, tokenSrv.URL, infoSrv.URL)

	nonce, _, _, _ := startLogin(t, h.srv, "/forum")

	recorder := httptest.NewRecorder()
	h.srv.handleGoogleCallback(recorder, httptest.NewRequest(http.MethodGet,
		"/auth/callback?code=attacker-code&state="+url.QueryEscape(nonce), nil))

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("沒有綁定 cookie 的回呼 = %d, want 400", recorder.Code)
	}
	if tokenHits, _ := google.hits(); tokenHits != 0 {
		t.Errorf("state 不可信時仍然呼叫了 token 端點 %d 次 —— 那是 login CSRF 的路徑", tokenHits)
	}
	if loc := recorder.Header().Get("Location"); loc != "" {
		t.Errorf("失敗的回呼仍然導向 %q", loc)
	}
	if strings.Contains(recorder.Body.String(), "attacker@example.com") {
		t.Errorf("回應洩漏了身分: %s", recorder.Body.String())
	}
}

func TestOAuthCallbackRejectsMismatchedBindingCookie(t *testing.T) {
	h := newCovHarness(t)
	google := &fakeGoogle{}
	tokenSrv := httptest.NewServer(google.handler())
	defer tokenSrv.Close()
	infoSrv := httptest.NewServer(google.userInfo("someone@example.com"))
	defer infoSrv.Close()
	withOAuthEndpoints(t, tokenSrv.URL, infoSrv.URL)

	nonce, _, _, _ := startLogin(t, h.srv, "/forum")

	request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state="+url.QueryEscape(nonce), nil)
	// 另一個流程的 nonce：兩邊都是真實存在的值，但不相符。
	request.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: "ffffffffffffffffffffffffffffffffffffffffffff"})
	recorder := httptest.NewRecorder()
	h.srv.handleGoogleCallback(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("state 與 cookie 不符的回呼 = %d, want 400", recorder.Code)
	}
	if tokenHits, _ := google.hits(); tokenHits != 0 {
		t.Errorf("不符時仍然換了 token（%d 次）", tokenHits)
	}
}

// nonce 必須是一次性的：被用過之後不能再被接受第二次。
func TestOAuthCallbackRejectsReplayedNonce(t *testing.T) {
	h := newCovHarness(t)
	google := &fakeGoogle{}
	tokenSrv := httptest.NewServer(google.handler())
	defer tokenSrv.Close()
	infoSrv := httptest.NewServer(google.userInfo("someone@example.com"))
	defer infoSrv.Close()
	withOAuthEndpoints(t, tokenSrv.URL, infoSrv.URL)

	nonce, _, bound, _ := startLogin(t, h.srv, "/forum")

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state="+url.QueryEscape(nonce), nil)
	firstReq.AddCookie(bound)
	h.srv.handleGoogleCallback(first, firstReq)
	if first.Code != http.StatusSeeOther {
		t.Fatalf("第一次回呼 = %d, want 303; body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc2&state="+url.QueryEscape(nonce), nil)
	secondReq.AddCookie(bound)
	h.srv.handleGoogleCallback(second, secondReq)
	if second.Code != http.StatusBadRequest {
		t.Errorf("重播的 nonce 被接受 = %d, want 400", second.Code)
	}
	if tokenHits, _ := google.hits(); tokenHits != 1 {
		t.Errorf("token 端點被呼叫 %d 次，want 1（第二次不該再換 token）", tokenHits)
	}
}

// 過期的 nonce（TTL 到了、或被別處刪掉）不能成為一條「沒有記錄但看起来合法」的登入。
func TestOAuthCallbackRejectsExpiredNonce(t *testing.T) {
	h := newCovHarness(t)
	google := &fakeGoogle{}
	tokenSrv := httptest.NewServer(google.handler())
	defer tokenSrv.Close()
	infoSrv := httptest.NewServer(google.userInfo("someone@example.com"))
	defer infoSrv.Close()
	withOAuthEndpoints(t, tokenSrv.URL, infoSrv.URL)

	nonce, _, bound, _ := startLogin(t, h.srv, "/forum")
	h.srv.mediaRedis.Del(context.Background(), oauthStateKey(nonce))

	request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state="+url.QueryEscape(nonce), nil)
	request.AddCookie(bound)
	recorder := httptest.NewRecorder()
	h.srv.handleGoogleCallback(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("沒有記錄的回呼 = %d, want 400", recorder.Code)
	}
	if tokenHits, _ := google.hits(); tokenHits != 0 {
		t.Errorf("沒有記錄時仍然換了 token（%d 次）", tokenHits)
	}
}

// Redis 裡那一筆被改動時（其他寫入者、或有人直接動 Redis），回呼仍然不能跳到站外。
// 這一條假設的是「寫進去時檢查過」不再成立的那一種情況。
func TestOAuthCallbackRevalidatesStoredReturnPath(t *testing.T) {
	h := newCovHarness(t)
	google := &fakeGoogle{}
	tokenSrv := httptest.NewServer(google.handler())
	defer tokenSrv.Close()
	infoSrv := httptest.NewServer(google.userInfo("someone@example.com"))
	defer infoSrv.Close()
	withOAuthEndpoints(t, tokenSrv.URL, infoSrv.URL)

	nonce, verifier, bound, _ := startLogin(t, h.srv, "/forum")
	payload, err := json.Marshal(oauthState{ReturnPath: "//evil.example", CodeVerifier: verifier})
	if err != nil {
		t.Fatalf("序列化失敗: %v", err)
	}
	h.srv.mediaRedis.Set(context.Background(), oauthStateKey(nonce), payload, oauthStateTTL)

	request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state="+url.QueryEscape(nonce), nil)
	request.AddCookie(bound)
	recorder := httptest.NewRecorder()
	h.srv.handleGoogleCallback(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("回呼 = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != "/" {
		t.Errorf("導向目標 = %q，被改動的站外目標必須退回首頁", loc)
	}
}

// 完整的成功路徑：授權碼帶 verifier 換 token、簽發 session、導回原本那一頁，
// 並把綁定 cookie 清掉。
func TestOAuthCallbackCompletesFlowAndClearsBindingCookie(t *testing.T) {
	h := newCovHarness(t)
	google := &fakeGoogle{}
	tokenSrv := httptest.NewServer(google.handler())
	defer tokenSrv.Close()
	infoSrv := httptest.NewServer(google.userInfo("someone@example.com"))
	defer infoSrv.Close()
	withOAuthEndpoints(t, tokenSrv.URL, infoSrv.URL)

	nonce, verifier, bound, _ := startLogin(t, h.srv, "/forum?tab=2&sort=new")

	request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state="+url.QueryEscape(nonce), nil)
	request.AddCookie(bound)
	recorder := httptest.NewRecorder()
	h.srv.handleGoogleCallback(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("回呼 = %d, want 303; body=%s", recorder.Code, recorder.Body.String())
	}
	if loc := recorder.Header().Get("Location"); loc != "/forum?tab=2&sort=new" {
		t.Errorf("導向目標 = %q（回跳路徑必須完整往返）", loc)
	}
	if got := google.codeVerifier(); got != verifier {
		t.Errorf("token 端點收到的 code_verifier = %q, want 原本那個 verifier", got)
	}

	// session cookie 必須被簽發（這才是「登入成功」的定義）。
	var sessionSet, bindingCleared bool
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == h.srv.sessionCookieName() && cookie.Value != "" {
			sessionSet = true
		}
		if cookie.Name == oauthStateCookieName && cookie.MaxAge < 0 {
			bindingCleared = true
		}
	}
	if !sessionSet {
		t.Error("回呼沒有簽發 session cookie")
	}
	if !bindingCleared {
		t.Error("回呼沒有清除綁定 cookie（它會在瀏覽器裡活到 TTL 結束）")
	}
	if _, err := h.srv.mediaRedis.Get(context.Background(), oauthStateKey(nonce)).Result(); err == nil {
		t.Error("用過的 nonce 還在 Redis 裡 —— 它必須是一次性的")
	}

	// 簽發的 session 是普通使用者（不是管理員），且前端查詢得到的是登入態。
	check := httptest.NewRecorder()
	checkReq := httptest.NewRequest(http.MethodGet, "/api/check", nil)
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == h.srv.sessionCookieName() {
			checkReq.AddCookie(cookie)
		}
	}
	h.srv.handleCheck(check, checkReq)
	if !strings.Contains(check.Body.String(), `"ok":true`) {
		t.Errorf("/api/check 沒有回報登入態: %s", check.Body.String())
	}
	if strings.Contains(check.Body.String(), `"isAdmin":true`) {
		t.Errorf("/api/check 回報了管理員: %s", check.Body.String())
	}
}

// 停權的使用者仍然不能透過 OAuth 登入（這一條與 state 無關，但走的是同一支回呼，
// 而它必須在「state 已驗證」之後才發生 —— 否則任何人都能用一個假 state 探測停權名單）。
func TestOAuthCallbackStillChecksSuspension(t *testing.T) {
	h := newCovHarness(t)
	google := &fakeGoogle{}
	tokenSrv := httptest.NewServer(google.handler())
	defer tokenSrv.Close()
	infoSrv := httptest.NewServer(google.userInfo("blocked@example.com"))
	defer infoSrv.Close()
	withOAuthEndpoints(t, tokenSrv.URL, infoSrv.URL)
	h.rows("SELECT status FROM forum_users", []string{"status"}, []driver.Value{"SUSPENDED"})

	nonce, _, bound, _ := startLogin(t, h.srv, "/forum")
	request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state="+url.QueryEscape(nonce), nil)
	request.AddCookie(bound)
	recorder := httptest.NewRecorder()
	h.srv.handleGoogleCallback(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("被停權的使用者登入 = %d, want 401", recorder.Code)
	}
}

// 綁定 cookie 的名字與 Path 必須與寫入時完全一致，否則瀏覽器不會刪掉它。
func TestClearOAuthStateCookieMatchesOriginalAttributes(t *testing.T) {
	h := newCovHarness(t)
	withOAuthEndpoints(t, "http://127.0.0.1:1/token", "http://127.0.0.1:1/userinfo")

	writeRec := httptest.NewRecorder()
	h.srv.handleGoogleLogin(writeRec, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
	clearRec := httptest.NewRecorder()
	h.srv.clearOAuthStateCookie(clearRec)

	var written, cleared *http.Cookie
	for _, cookie := range writeRec.Result().Cookies() {
		if cookie.Name == oauthStateCookieName {
			written = cookie
		}
	}
	for _, cookie := range clearRec.Result().Cookies() {
		if cookie.Name == oauthStateCookieName {
			cleared = cookie
		}
	}
	if written == nil || cleared == nil {
		t.Fatal("寫入或清除沒有產生 cookie")
	}
	if written.Path != cleared.Path || written.HttpOnly != cleared.HttpOnly || written.Secure != cleared.Secure {
		t.Errorf("清除的 cookie 屬性與寫入不同：path=%q/%q httpOnly=%v/%v secure=%v/%v",
			written.Path, cleared.Path, written.HttpOnly, cleared.HttpOnly, written.Secure, cleared.Secure)
	}
	if cleared.MaxAge >= 0 {
		t.Errorf("清除用的 MaxAge = %d，必須是負數", cleared.MaxAge)
	}
}

// matchesOAuthState 的邊界：兩邊都空時不相符（舊版流程與「有人直接貼回呼網址」都
// 落在這裡），而「一邊是另一邊的前綴」也不相符。
func TestMatchesOAuthState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		state  string
		cookie string
		want   bool
	}{
		{"相同", "nonce-value", "nonce-value", true},
		{"不同", "nonce-a", "nonce-b", false},
		{"前綴", "nonce", "nonce-extra", false},
		{"兩邊都空", "", "", false},
		{"state 空", "", "nonce-value", false},
		{"cookie 空", "nonce-value", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesOAuthState(tc.state, tc.cookie); got != tc.want {
				t.Errorf("matchesOAuthState(%q, %q) = %v, want %v", tc.state, tc.cookie, got, tc.want)
			}
		})
	}
}

// 路由層：/auth/google 走的是這個 handler（不是直接呼叫 auth 的那一支）。
// 這一條釘住的是「限流與 state 寫入在同一條路徑上」—— 若有人把路由改回
// auth.HandleLogin，nonce 就不會被寫下，所有人都登不進來。
func TestAuthGoogleRouteUsesStateAwareLogin(t *testing.T) {
	h := newCovHarness(t)
	withOAuthEndpoints(t, "http://127.0.0.1:1/token", "http://127.0.0.1:1/userinfo")

	recorder := h.do(h.srv.Handler().ServeHTTP, http.MethodGet, "/auth/google?return=/forum", "", "", false)
	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("透過路由的發起登入 = %d, want 307", recorder.Code)
	}
	parsed, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("授權網址非法: %v", err)
	}
	nonce := parsed.Query().Get("state")
	if _, err := h.srv.mediaRedis.Get(context.Background(), oauthStateKey(nonce)).Result(); err != nil {
		t.Errorf("路由那一條路徑沒有寫下 state（nonce=%s）", nonce)
	}
	if ttl := h.srv.mediaRedis.TTL(context.Background(), oauthStateKey(nonce)).Val(); ttl > oauthStateTTL+time.Second {
		t.Errorf("state 的 TTL = %v，超出設定的 %v", ttl, oauthStateTTL)
	}
}
