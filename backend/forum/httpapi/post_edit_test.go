/*
貼文與留言的編輯／刪除、單篇讀取，以及永久連結頁的路由（post_edit_test.go）。

這一批端點有四個性質，而它們的共同點是**壞掉時不會出錯**：

  1. 授權邊界
     編輯與刪除的 WHERE 子句都帶著 author_email，因此「改到別人的文章」
     這個結果不可能出現 —— 但也正因為它不可能出現，**寫錯條件時症狀是
     「編輯按鈕按下去，畫面顯示成功，內容卻沒變」**，而不是任何錯誤。
     未登入必須 401（否則同一件事會變成「別人的貼文被無聲改掉」）。

  2. 方法分派
     /api/forum/posts/ 是一個子樹，GET/PUT/DELETE 三種方法分派在
     handleForumPostAction 裡。用錯方法的症狀是 404（看起來像「這篇
     不存在」）而不是 405，因此這裡逐條釘死。

  3. 路徑解析
     forumCommentIDFromPath 與 forumPostPageID 是純函式，它們決定
     「這個網址對應到哪一筆」—— 解析錯了就是操作到別的資料。
     表格測試涵蓋尾斜線、多餘段位、非數字與非正值。

  4. 公開讀取不需要登入
     GET /api/forum/posts/{id} 掛在 requireLoginForWrite 之後，而它對 GET
     放行；/forum/post/{id} 頁面則在 server.go 另外註冊成公開路由。
     這兩條都是「分享連結給別人」的前提，被改成需要登入不會讓任何測試
     紅燈，只會讓連結對匿名訪客失效。

刻意沒有測的：實際的 UPDATE 與 DELETE 語句（需要真的 MySQL；這個專案
沒有替身，見 docs/KNOWN_ISSUES.md 的已知問題）。授權條件本身在 (1) 以「未登入被擋」
與純函式的路徑解析兩側守住，而 SQL 那一行是單一陳述句、沒有分支。
*/
package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forum/forum/config"
	"forum/forum/session"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newPostEditTestServer 建立一個帶 session 的測試 Server（沒有 db）。
//
// 沒有 db 是一件刻意的限制：這個檔案測的每一條路徑都在碰到資料庫**之前**
// 就回應（401、403、400、405、404），因此它可以在沒有 MySQL 的環境裡跑。
// 真正會寫入的那條路徑由 (1) 的授權邊界與 MigrateMySQL 的遷移測試把關。
func newPostEditTestServer(t *testing.T, trustedOrigins ...string) *Server {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &Server{
		cfg:      config.Config{TrustedOrigins: trustedOrigins},
		sessions: session.NewManager("FORUM_test", time.Hour, false, client),
	}
}

// loggedInRequest 造一個帶著有效 session cookie 的請求。
//
// 走 session.Manager.Create 而不是直接寫 Redis：token 的產生格式是 manager
// 的責任（見 session.go 的檔頭說明），測試自己組一個「看起來像」的 token
// 會在 manager 改格式時靜默失效 —— 而症狀是「測試突然全部變成未登入」。
func loggedInRequest(t *testing.T, server *Server, method, path, body string) *http.Request {
	t.Helper()
	token, err := server.sessions.Create("owner@example.com", false)
	if err != nil {
		t.Fatalf("建立 session 失敗: %v", err)
	}
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	request.AddCookie(&http.Cookie{Name: "FORUM_test", Value: token})
	return request
}

/* ==========================================================================
   授權
   ========================================================================== */

/*
未登入的寫入必須 401。

這是這一整批端點最重要的一條：它們的 SQL 都把 author_email 放進 WHERE，
所以「沒有登入 = author_email 是空字串 = 剛好改不到任何東西」看似安全。
但安全不該靠「比對恰好不匹配」：一旦有人日後把 author_email 從 WHERE 拿掉
（例如為了支援管理員代改），空字串的作者就會讓這個端點變成「任何未登入者
都能改任何一篇文章」。401 是那道閘門。
*/
func TestPostAndCommentMutationsRequireLogin(t *testing.T) {
	server := newPostEditTestServer(t)
	cases := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"編輯貼文", http.MethodPut, "/api/forum/posts/1", server.handleForumPostAction},
		{"刪除貼文", http.MethodDelete, "/api/forum/posts/1", server.handleForumPostAction},
		{"編輯留言", http.MethodPut, "/api/forum/posts/1/comments/2", server.handleForumPostAction},
		{"刪除留言", http.MethodDelete, "/api/forum/posts/1/comments/2", server.handleForumPostAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"content":"x"}`))
			tc.handler(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("回應 = %d, want 401（未登入的寫入必須被擋）", recorder.Code)
			}
		})
	}
}

/*
已登入但來源不可信必須 403。

這一條與 csrf_guard_test.go 互補：那支測試斷言「每個會改資料的 handler
**有呼叫** isTrustedOrigin」，而這支斷言「呼叫之後的判斷在設定了白名單時
真的會擋人」。少了後者，一個被改成永遠回 true 的 isTrustedOrigin 仍然會
讓前者長期呈綠 —— 而症狀是 CSRF 防護失效卻沒有任何錯誤（見該檔檔頭）。
*/
func TestPostAndCommentMutationsRejectUntrustedOrigin(t *testing.T) {
	server := newPostEditTestServer(t, "http://localhost:8088")
	cases := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"編輯貼文", http.MethodPut, "/api/forum/posts/1", server.handleForumPostAction},
		{"編輯留言", http.MethodPut, "/api/forum/posts/1/comments/2", server.handleForumPostAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := loggedInRequest(t, server, tc.method, tc.path, `{"content":"x"}`)
			request.Header.Set("Origin", "https://evil.example")
			tc.handler(recorder, request)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("回應 = %d, want 403（來源不在 TRUSTED_ORIGINS）", recorder.Code)
			}
		})
	}
}

/* ==========================================================================
   內容驗證
   ========================================================================== */

/*
內文驗證發生在**碰資料庫之前**，因此這些案例不需要 MySQL。

這一組驗證的理由與公告測試相同，但後果更糟：空白內容若被接受，會產生一篇
「有邊框、裡面是空的」貼文並出現在所有人的動態裡，而空內容無法從使用者的
介面分辨（那看起來就像貼文載入失敗）。10000 字上限則與 createForumPost
一致 —— 兩條路徑的上限必須相同，否則使用者可以靠編輯塞進一筆建立時擋不下的
內容，而公開頁沒有任何地方會把它截短。
*/
func TestPostAndCommentEditValidateContent(t *testing.T) {
	overlongPost := strings.Repeat("中", 10001)
	overlongComment := strings.Repeat("中", 2001)

	cases := []struct {
		name    string
		method  string
		path    string
		body    string
		wantMsg string
	}{
		{"貼文空白內容", http.MethodPut, "/api/forum/posts/1", `{"content":"   \n\t "}`, "內容不可為空"},
		{"貼文空字串", http.MethodPut, "/api/forum/posts/1", `{"content":""}`, "內容不可為空"},
		{"貼文缺 content 欄位", http.MethodPut, "/api/forum/posts/1", `{}`, "內容不可為空"},
		{"貼文超過 10000 字", http.MethodPut, "/api/forum/posts/1",
			`{"content":"` + overlongPost + `"}`, "內容最多 10000 字"},
		{"貼文主體不是 JSON", http.MethodPut, "/api/forum/posts/1", `not json`, "invalid request"},
		{"留言空白內容", http.MethodPut, "/api/forum/posts/1/comments/2", `{"content":"  "}`, "留言內容不可為空"},
		{"留言超過 2000 字", http.MethodPut, "/api/forum/posts/1/comments/2",
			`{"content":"` + overlongComment + `"}`, "留言最多 2000 字"},
		{"留言主體不是 JSON", http.MethodPut, "/api/forum/posts/1/comments/2", `{{{`, "invalid request"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newPostEditTestServer(t, "http://localhost:8088")
			recorder := httptest.NewRecorder()
			request := loggedInRequest(t, server, tc.method, tc.path, tc.body)
			request.Header.Set("Origin", "http://localhost:8088")
			server.handleForumPostAction(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("回應 = %d, want 400", recorder.Code)
			}
			// 訊息會直接顯示給使用者（requestJSON 的 fallback 與後端原文的
			// 優先序見 core.ts），因此必須是人看得懂的，而不是 decoder 的原文。
			if !strings.Contains(recorder.Body.String(), tc.wantMsg) {
				t.Errorf("body = %q，應包含 %q", recorder.Body.String(), tc.wantMsg)
			}
		})
	}
}

/* ==========================================================================
   方法分派
   ========================================================================== */

/*
/api/forum/posts/ 子樹的方法分派。

這些案例刻意全部選「在碰到資料庫之前就能決定回應」的路徑，因此不需要 MySQL。
其中「不支援的方法 → 405」特別重要：若分派漏了一個方法而讓它落到 404，
症狀是使用者在自己的貼文上按編輯，畫面卻說「找不到這篇文章」—— 那是一個
使用者完全無法自行判斷的訊息。
*/
func TestForumPostSubtreeMethodDispatch(t *testing.T) {
	server := newPostEditTestServer(t)
	cases := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"貼文不支援 POST", http.MethodPost, "/api/forum/posts/1", http.StatusMethodNotAllowed},
		{"貼文不支援 PATCH", http.MethodPatch, "/api/forum/posts/1", http.StatusMethodNotAllowed},
		{"留言不支援 POST", http.MethodPost, "/api/forum/posts/1/comments/2", http.StatusMethodNotAllowed},
		{"留言不支援 GET", http.MethodGet, "/api/forum/posts/1/comments/2", http.StatusMethodNotAllowed},
		{"未定義的子資源", http.MethodGet, "/api/forum/posts/1/abc", http.StatusNotFound},
		{"留言 id 不是數字", http.MethodPut, "/api/forum/posts/1/comments/abc", http.StatusNotFound},
		{"留言下有多餘段位", http.MethodGet, "/api/forum/posts/1/comments/2/abc", http.StatusNotFound},
		{"非正數的貼文 id", http.MethodGet, "/api/forum/posts/0", http.StatusNotFound},
		{"非正數的留言 id", http.MethodDelete, "/api/forum/posts/1/comments/0", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := loggedInRequest(t, server, tc.method, tc.path, `{"content":"x"}`)
			request.Header.Set("Origin", "http://localhost:8088")
			server.handleForumPostAction(recorder, request)
			if recorder.Code != tc.want {
				t.Fatalf("回應 = %d, want %d", recorder.Code, tc.want)
			}
		})
	}
}

/* ==========================================================================
   純函式：路徑解析
   ========================================================================== */

func TestForumCommentIDFromPath(t *testing.T) {
	cases := []struct {
		path  string
		want  int64
		wantOK bool
	}{
		{"/api/forum/posts/7/comments/12", 12, true},
		{"/api/forum/posts/7/comments/12/", 12, true},
		{"/api/forum/posts/7/comments/12/report", 0, false},
		{"/api/forum/posts/7/comments", 0, false},
		{"/api/forum/posts/7/comments/abc", 0, false},
		{"/api/forum/posts/7/comments/0", 0, false},
		{"/api/forum/posts/7/comments/-3", 0, false},
		{"/api/forum/posts/7/comments/12/extra", 0, false},
		{"/api/forum/posts/7", 0, false},
	}
	for _, tc := range cases {
		got, ok := forumCommentIDFromPath(tc.path)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("forumCommentIDFromPath(%q) = (%d, %v), want (%d, %v)",
				tc.path, got, ok, tc.want, tc.wantOK)
		}
	}
}

/*
永久連結頁的路徑解析。

這一條決定「哪些 /forum/post/* 會拿到頁面、哪些是明確的 404」。放寬成
「任何 /forum/post/ 之後都吐頁面」的症狀是 /forum/post/abc 顯示一個空白頁
（前端讀不到 id，於是停在「連結不合法」），而 /forum/post/ 會拿到一個看起來
正常、但永遠是空白的頁面 —— 兩者都不會有任何錯誤訊息。
*/
func TestForumPostPageID(t *testing.T) {
	cases := []struct {
		path   string
		wantOK bool
	}{
		{"/forum/post/1", true},
		{"/forum/post/12345", true},
		{"/forum/post/1/", true},
		{"/forum/post/0", false},
		{"/forum/post/-1", false},
		{"/forum/post/abc", false},
		{"/forum/post/", false},
		{"/forum/post", false},
		{"/forum/post/1/comments", false},
		{"/forum/others-profile", false},
		{"/forum", false},
	}
	for _, tc := range cases {
		if _, ok := forumPostPageID(tc.path); ok != tc.wantOK {
			t.Errorf("forumPostPageID(%q) ok = %v, want %v", tc.path, ok, tc.wantOK)
		}
	}
}