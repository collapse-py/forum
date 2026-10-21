/*
session 生命週期的測試（backend/forum/session/session_lifecycle_test.go）。

session_list_test.go 負責後臺的「列舉與批次撤銷」，這個檔負責**單一使用者**的
那條路徑：登入時 cookie 怎麼寫、每個請求的續期怎麼推、身分怎麼問、登出怎麼
真正生效。這些性質的共同點是「壞掉時不會報錯」——cookie 少一個屬性、續期少
推一次 TTL、IsAdmin 少比對一個欄位，全部都只是「某些情況下使用者莫名被登出
或莫名還有權限」，沒有任何一行錯誤訊息。

測試用 miniredis（專案既有的 Redis 替身；MySQL 沒有替身，但 Redis 有）。

刻意沒有測的：
  - Create 的 Expire 失敗分支（HSet 成功、Expire 失敗時要刪 key 的 fail-closed）。
    它需要「Redis 在兩道指令之間壞掉」，miniredis 沒有中途注入故障的鉤子；
    那段邏輯由程式碼結構保證（defer 刪除路徑就在旁邊）。
  - 多個 Manager 併發使用同一個 Redis。go-redis client 本身併發安全，
    而 Manager 建構後唯讀，這件事由型別保證而不是行為需要驗證。
*/
package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// requestWithToken 造一個帶 session cookie 的請求。
func requestWithToken(cookieName, token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/check", nil)
	if token != "" {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	}
	return r
}

// responseCookies 從 ResponseRecorder 解析出 Set-Cookie 的第一筆；沒有任何
// Set-Cookie 表頭時回傳 nil。
func responseCookies(t *testing.T, w *httptest.ResponseRecorder) []*http.Cookie {
	t.Helper()
	result := w.Result()
	defer result.Body.Close()
	return result.Cookies()
}

/* ==========================================================================
   cookie 簽發與清除
   ========================================================================== */

// Set-Cookie 的每個屬性都各自擋住一條真實的攻擊或失效路徑：漏 HttpOnly 等於
// 開放 XSS 竊取 session，漏 SameSite 等於跨站請求自帶登入態（CSRF 第一道
// 防線消失），Path 不是 "/" 則讓 /admin 或 /api 讀不到登入態。
func TestSetCookieWritesEveryAttribute(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	m := NewManager("FORUM_attr", 2*time.Hour, true, client)
	w := httptest.NewRecorder()
	m.SetCookie(w, "tok-abc")

	cookies := responseCookies(t, w)
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie 數量 = %d, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != "FORUM_attr" || c.Value != "tok-abc" {
		t.Errorf("cookie = %s=%q, want FORUM_attr=tok-abc", c.Name, c.Value)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /（前端、/forum、/admin、/api 共用同一張 cookie）", c.Path)
	}
	if !c.HttpOnly {
		t.Error("HttpOnly = false：JavaScript 可讀取，XSS 竊取 session 的路徑被打開")
	}
	if !c.Secure {
		t.Error("Secure = false，但 manager 以 cookieSecure=true 建構")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax（跨站 POST 不附帶 cookie 是 CSRF 第一道防線）", c.SameSite)
	}
	// Expires 必須由 expire 推導（≈ now+2h），與 Redis TTL 同源。
	if c.Expires.Before(time.Now().Add(90*time.Minute)) || c.Expires.After(time.Now().Add(151*time.Minute)) {
		t.Errorf("Expires = %v, want ≈ now+2h", c.Expires)
	}
}

// cookieSecure=false 時 Secure 必須真的不寫：本機 HTTP 開發時瀏覽器會丟棄
// 帶 Secure 的 cookie，而那個失效模式是「本機完全無法登入」。
func TestSetCookieOmitsSecureWhenDisabled(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	m := NewManager("FORUM_insecure", time.Hour, false, client)
	w := httptest.NewRecorder()
	m.SetCookie(w, "tok")

	c := responseCookies(t, w)[0]
	if c.Secure {
		t.Error("Secure = true，但 manager 以 cookieSecure=false 建構（HTTP 本地開發會無法登入）")
	}
}

// ClearCookie 的 MaxAge=-1 是 net/http 表達「立即刪除」的唯一寫法，而 Path 必須
// 與 SetCookie 完全一致 —— 瀏覽器比對的是 name+domain+path，Path 不同就刪不掉
// 舊 cookie，使用者看起來「已登出但仍是登入狀態」。
func TestClearCookieDeletesImmediatelyOnSamePath(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	m := NewManager("FORUM_clear", time.Hour, true, client)
	w := httptest.NewRecorder()
	m.ClearCookie(w)

	c := responseCookies(t, w)[0]
	if c.Name != "FORUM_clear" {
		t.Errorf("Name = %q, want FORUM_clear", c.Name)
	}
	if c.Value != "" {
		t.Errorf("Value = %q, want 空", c.Value)
	}
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want -1（唯一表達立即刪除的方式）", c.MaxAge)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /（必須與 SetCookie 相同，否則刪不掉舊 cookie）", c.Path)
	}
	if !c.HttpOnly || !c.Secure {
		t.Error("ClearCookie 的 HttpOnly / Secure 應與 SetCookie 一致")
	}
}

/* ==========================================================================
   登出與撤銷
   ========================================================================== */

// 登出的「伺服器端生效」靠 RemoveByRequest 刪 Redis key：只刪瀏覽器 cookie 的
// 話，舊 token 重送就仍然是登入狀態。
func TestRemoveByRequestDeletesRedisKey(t *testing.T) {
	m, _ := newTestManager(t, 0, time.Hour)
	token, err := m.Create("logout@example.com", false)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !m.Alive(token) {
		t.Fatal("前置條件失敗：session 未建立")
	}

	m.RemoveByRequest(requestWithToken("FORUM_test", token))

	if m.Alive(token) {
		t.Error("RemoveByRequest 之後 session 仍存活：登出沒有真正撤銷")
	}
}

// 沒有 cookie 或帶著不存在的 token：撤銷是冪等的，不得 panic、不得回錯誤。
func TestRemoveByRequestIsIdempotent(t *testing.T) {
	m, _ := newTestManager(t, 0, time.Hour)
	m.RemoveByRequest(requestWithToken("FORUM_test", ""))          // 無 cookie
	m.RemoveByRequest(requestWithToken("FORUM_test", "not-a-real")) // 不存在的 token
}

/* ==========================================================================
   滑動續期
   ========================================================================== */

// 續期是「每個請求都做」的工作：它必須 (1) 回 true、(2) 把 Redis TTL 推回完整
// expire、(3) 重發 cookie 讓瀏覽器端同步。漏掉 (2) 的症狀是「操作中被登出」。
func TestRefreshExtendsTTLAndReissuesCookie(t *testing.T) {
	m, mini := newTestManager(t, 0, time.Hour)
	token, err := m.Create("slide@example.com", false)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// 推進 50 分鐘：距過期只剩 10 分鐘。
	mini.FastForward(50 * time.Minute)

	w := httptest.NewRecorder()
	if !m.Refresh(requestWithToken("FORUM_test", token), w) {
		t.Fatal("Refresh 對有效 session 回 false")
	}

	cookies := responseCookies(t, w)
	if len(cookies) != 1 || cookies[0].Value != token {
		t.Fatalf("Refresh 應重發同值 cookie，得到 %+v", cookies)
	}

	// TTL 必須被推回接近完整 expire：再推 55 分鐘（累計 105 分鐘 > 60 分鐘的
	// expire）後 session 仍活著，代表 TTL 確實被重設過。
	mini.FastForward(55 * time.Minute)
	if !m.Alive(token) {
	t.Error("續期後再推 55 分鐘即過期：TTL 沒有被推回完整 expire")
	}
}

// cookie 缺失或 token 不存在時回 false，且**不得**重發 cookie —— 後者會讓
// 已過期的 session 在瀏覽器端看起來「永遠有效」。
func TestRefreshRejectsMissingAndUnknownSession(t *testing.T) {
	m, _ := newTestManager(t, 0, time.Hour)

	w := httptest.NewRecorder()
	if m.Refresh(requestWithToken("FORUM_test", ""), w) {
		t.Error("無 cookie 的 Refresh 應回 false")
	}
	if len(responseCookies(t, w)) != 0 {
		t.Error("無 cookie 的 Refresh 不應重發 cookie")
	}

	w = httptest.NewRecorder()
	if m.Refresh(requestWithToken("FORUM_test", "vanished"), w) {
		t.Error("不存在 token 的 Refresh 應回 false")
	}
	if len(responseCookies(t, w)) != 0 {
		t.Error("不存在 token 的 Refresh 不應重發 cookie")
	}
}

// 已過期（TTL 耗盡）的 session：Refresh 必須回 false，而不是對不存在的 key
// 設 Expire 後誤以為續期成功。
func TestRefreshRejectsExpiredSession(t *testing.T) {
	m, mini := newTestManager(t, 0, time.Hour)
	token, err := m.Create("expired@example.com", false)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	mini.FastForward(2 * time.Hour) // 超過 expire=1h

	w := httptest.NewRecorder()
	if m.Refresh(requestWithToken("FORUM_test", token), w) {
		t.Error("已過期 session 的 Refresh 應回 false")
	}
	if m.ResolveUser(requestWithToken("FORUM_test", token)) != "" {
		t.Error("已過期 session 不應解析出使用者")
	}
}

/* ==========================================================================
   身分判斷
   ========================================================================== */

func TestResolveUserReadsEmailFromRedis(t *testing.T) {
	m, _ := newTestManager(t, 0, time.Hour)
	token, err := m.Create("who@example.com", true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got := m.ResolveUser(requestWithToken("FORUM_test", token)); got != "who@example.com" {
		t.Errorf("ResolveUser = %q, want who@example.com", got)
	}
	if got := m.ResolveUser(requestWithToken("FORUM_test", "")); got != "" {
		t.Errorf("無 cookie 的 ResolveUser = %q, want 空", got)
	}
	// 不存在的 token：Redis 回錯誤，必須收斂成「未登入」而不是把錯誤往上丟。
	if got := m.ResolveUser(requestWithToken("FORUM_test", "ghost")); got != "" {
		t.Errorf("不存在 token 的 ResolveUser = %q, want 空", got)
	}
}

// is_admin 是登入當下寫死在 hash 的欄位：管理員與否必須由它決定，而不是由
// cookie 內容或 email 推算。這裡逐一釘住 true / false / 欄位缺失三態。
func TestIsAdminReadsHashFieldNotEmail(t *testing.T) {
	m, _ := newTestManager(t, 0, time.Hour)
	adminToken, err := m.Create("admin@example.com", true)
	if err != nil {
		t.Fatalf("Create admin: %v", err)
	}
	normalToken, err := m.Create("normal@example.com", false)
	if err != nil {
		t.Fatalf("Create normal: %v", err)
	}

	if !m.IsAdmin(requestWithToken("FORUM_test", adminToken)) {
		t.Error("is_admin=true 的 session，IsAdmin 應回 true")
	}
	if m.IsAdmin(requestWithToken("FORUM_test", normalToken)) {
		t.Error("is_admin=false 的 session，IsAdmin 應回 false")
	}
	if m.IsAdmin(requestWithToken("FORUM_test", "ghost")) {
		t.Error("不存在的 token，IsAdmin 應回 false（fail closed）")
	}
	if m.IsAdmin(requestWithToken("FORUM_test", "")) {
		t.Error("無 cookie，IsAdmin 應回 false")
	}
}

func TestIsLoggedInTokenAndAlive(t *testing.T) {
	m, _ := newTestManager(t, 0, time.Hour)
	token, err := m.Create("logged@example.com", false)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	r := requestWithToken("FORUM_test", token)
	if got := m.Token(r); got != token {
		t.Errorf("Token = %q, want %q", got, token)
	}
	if !m.IsLoggedIn(r) {
		t.Error("有效 session 的 IsLoggedIn 應回 true")
	}
	if got := m.Token(httptest.NewRequest(http.MethodGet, "/", nil)); got != "" {
		t.Errorf("無 cookie 的 Token = %q, want 空", got)
	}
	if m.IsLoggedIn(httptest.NewRequest(http.MethodGet, "/", nil)) {
		t.Error("無 cookie 的 IsLoggedIn 應回 false")
	}
	if m.Alive("") {
		t.Error("空 token 的 Alive 應回 false")
	}
	if !m.Alive(token) {
		t.Error("有效 token 的 Alive 應回 true")
	}
}

/* ==========================================================================
   其餘介面
   ========================================================================== */

// StartCleanup 是保留相容性的空實作：呼叫它必須無事發生（不 panic、不啟動
// 會干擾測試的背景工作）。呼叫本身即是覆蓋這個空函式的唯一方式。
func TestStartCleanupIsNoop(t *testing.T) {
	m, mini := newTestManager(t, 0, time.Hour)
	m.StartCleanup(time.Millisecond)
	if keys := mini.Keys(); len(keys) != 0 {
		t.Errorf("StartCleanup 不應對 Redis 寫入任何東西，卻留下 keys: %v", keys)
	}
}

// Create 的錯誤路徑：Redis 不可用時必須回錯誤，而不是回傳一個查不到的 token
// 讓呼叫端誤以為登入成功。
func TestCreateFailsWhenRedisUnavailable(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	m := NewManager("FORUM_test", time.Hour, false, client)

	mini.Close() // 模擬 Redis 掛掉
	if _, err := m.Create("down@example.com", false); err == nil {
		t.Fatal("Redis 不可用時 Create 應回錯誤（呼叫端應視為登入失敗）")
	}
	_ = client.Close()
}

// token 必須是 48 字元的 hex（24 位元組 crypto/rand）：這是「不可猜測」這個
// 安全性質的唯一載體，縮短或改用弱亂數都會讓 session 可被暴力枚舉。
func TestCreateIssuesLongRandomHexToken(t *testing.T) {
	m, _ := newTestManager(t, 0, time.Hour)

	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		token, err := m.Create(testEmail(i), false)
		if err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
		if len(token) != 48 {
			t.Fatalf("token 長度 = %d, want 48（24 位元組的 hex）", len(token))
		}
		for _, r := range token {
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
				t.Fatalf("token 含非 hex 字元 %q", r)
			}
		}
		if seen[token] {
			t.Fatalf("token 重複：%q", token)
		}
		seen[token] = true
	}
}
