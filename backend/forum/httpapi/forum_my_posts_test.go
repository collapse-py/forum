/*
自己的貼文清單（httpapi/forum_my_posts_test.go）。

handleForumMyPosts 與 handleForumPublicPosts 的差別只在「條件用哪一個 email」，
而那正是它值得一個測試的原因：這支端點掛在 requireLogin 之後，若身分檢查被
移除，症狀不會是任何錯誤 —— 沒有 session 的請求會拿空字串去查
`author_email = ''`，回 200 加一個空清單，個人頁顯示「你還沒有發表貼文」。
沒有任何地方看得出來這是未授權的回應。

因此這裡固定兩件事：
  1. 沒有 session 時回 401（且不得碰資料庫，因此不需要 MySQL 替身）。
  2. 只接受 GET —— 加了 POST 就會讓一支「列出自己貼文」的唯讀端點變成可被
     跨站表單打到的目標，而症狀同樣是無聲的。

刻意沒有測的：分頁與 hasMore 推測。它需要真的資料庫（這個專案沒有 MySQL 替身，
見 docs/KNOWN_ISSUES.md 的已知問題），而那部分由 loadForumPosts 與 listForumPosts 共用同一支
查詢，已經有後者的說明與 forumPostProjection 對著 idx_forum_posts_feed 的約束。
*/
package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forum/forum/session"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newMyPostsTestServer 以 miniredis 建立帶真實 session.Manager 的 Server。
//
// 需要真的 session 是因為這個測試要證明的正是「身分檢查有效」：用 nil 取代
// 只會得到一個 panic，而不是「未登入 → 401」。
func newMyPostsTestServer(t *testing.T) *Server {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	// db 刻意留 nil：通過身分檢查的請求會立刻在查詢上 panic，而這兩支測試
	// 都必須在碰到資料庫之前就結束（否則測的是 panic，不是狀態碼）。
	return &Server{sessions: session.NewManager("FORUM_test", time.Hour, false, client)}
}

func TestForumMyPostsRequiresLogin(t *testing.T) {
	server := newMyPostsTestServer(t)

	recorder := httptest.NewRecorder()
	server.handleForumMyPosts(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/my-posts", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("回應 = %d, want 401（沒有 session 時不得回應任何貼文）", recorder.Code)
	}
	// 401 的回應體不該帶 items：前端 requestJSON 讀的是回應碼，這裡只是把
	// 「不得洩漏任何貼文」固定成一個可斷言的形狀。
	if body := recorder.Body.String(); strings.Contains(body, `"items"`) {
		t.Errorf("body = %q，未登入的回應不該帶 items", body)
	}
}

func TestForumMyPostsRejectsNonGet(t *testing.T) {
	server := newMyPostsTestServer(t)

	recorder := httptest.NewRecorder()
	server.handleForumMyPosts(recorder, httptest.NewRequest(http.MethodPost, "/api/forum/my-posts", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("回應 = %d, want 405（唯讀端點不接受 POST）", recorder.Code)
	}
}