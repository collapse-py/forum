/*
論壇核心流程的真實 MySQL 測試（backend/forum/httpapi/forum_core_mysql_test.go）。

既有測試的分工：post_edit_test.go / post_soft_delete_test.go 等把「碰到資料庫
之前」的授權與驗證邊界釘在沒有 DB 的環境裡，而形狀測試（comment_reply_test）
釘 SQL 字串。這個檔補中間那一段：**整條路徑真的對 MySQL 跑**時，寫入的列、
回讀的 JSON、軟刪除之後的可見性是否正確。

約定同 comment_reply_mysql_test.go：FORUM_TEST_MYSQL_DSN 未設定時 skip
（子樹 helper 直接沿用那個檔案的 subtreeTempDatabase）。

覆蓋的生命週期：
  1. 發文 → 列表（含計數零值與作者匿名代號）→ 編輯（updated_at 旗標）→ 軟刪除
     （列表與詳情都消失，但列仍在、deleted_at 已設 —— 後者是「還原」功能的
     前提，只有斷言 HTTP 層看不見是不夠的）。
  2. 驗證邊界：空白內容、超過 10000 rune、第三方圖片網址。
  3. 按讚往返：like / unlike 之後 forum_post_likes 的列數與計數。
  4. 留言往返：建立 → comment_count → 編輯 → 刪除。
  5. 檢舉寫入 forum_reports。
  6. 我的貼文只回自己的。
*/

package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// newCoreForumServer 組一個「貼文 CRUD 需要的最小依賴」的 Server：
// 真 MySQL（schema 為完整 MigrateMySQL）、miniredis（session 與媒體 token）。
func newCoreForumServer(t *testing.T, db *sql.DB) *Server {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &Server{
		// 2026-10-27 起，帶著 session cookie 的寫入請求必須帶來源訊號（見
		// csrf.go 的分岔）。這一支 Server 是手建工況，所以白名單要手動給一個值：
		// 留空代表「沒有任何可信來源」，那些帶 cookie 的請求會全部得到 403，
		// 而這一整批測試驗的是 CRUD 本身。
		cfg:        config.Config{TrustedOrigins: []string{covOrigin}},
		db:         db,
		sessions:   session.NewManager("FORUM_test", time.Hour, false, client),
		mediaRedis: client,
	}
}

// decodeJSONBody 解回 map，讓斷言可以自由挑欄位而不必為每個端點定義結構。
func decodeJSONBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("回應不是 JSON 物件: %v\nbody=%s", err, w.Body.String())
	}
	return payload
}

// postIDFromBody 取出回應裡的貼文 id。create 回 201 + {ok:true, item:{…}}，
// 貼文物件在 item 底下。
func postIDFromBody(t *testing.T, payload map[string]any) int64 {
	t.Helper()
	item, ok := payload["item"].(map[string]any)
	if !ok {
		t.Fatalf("回應缺少 item 物件: %v", payload)
	}
	id, ok := item["id"].(float64)
	if !ok {
		t.Fatalf("item 缺少數值欄位 id: %v", item)
	}
	return int64(id)
}

// dbPostRow 直接從資料庫讀回貼文列（作者、內容、deleted_at）。
func dbPostRow(t *testing.T, db *sql.DB, id int64) (author, content string, deletedAt sql.NullTime, updatedAt sql.NullTime) {
	t.Helper()
	err := db.QueryRow(`SELECT author_email, content, deleted_at, updated_at FROM forum_posts WHERE id = ?`, id).
		Scan(&author, &content, &deletedAt, &updatedAt)
	if err != nil {
		t.Fatalf("讀取貼文列 %d 失敗: %v", id, err)
	}
	return
}

/* ==========================================================================
   生命週期
   ========================================================================== */

func TestPostLifecycleCreateListEditSoftDeleteMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	// 1. 發文（201 + 貼文物件）。
	recorder := httptest.NewRecorder()
	server.handleForumPosts(recorder, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"第一篇"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("發文回應 = %d, want 201; body=%s", recorder.Code, recorder.Body.String())
	}
	id := postIDFromBody(t, decodeJSONBody(t, recorder))

	// 列存在、作者是登入者、尚未刪除。
	author, content, deletedAt, _ := dbPostRow(t, db, id)
	if author != "owner@example.com" || content != "第一篇" || deletedAt.Valid {
		t.Errorf("貼文列 = (%q, %q, deleted=%v), want (owner@example.com, 第一篇, false)", author, content, deletedAt.Valid)
	}

	// 2. 列表回得到它，且計數欄位是零值、作者是匿名代號（不是 email）。
	recorder = httptest.NewRecorder()
	server.handleForumPosts(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/posts", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("列表回應 = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	list := decodeJSONBody(t, recorder)
	posts, ok := list["items"].([]any)
	if !ok || len(posts) != 1 {
		t.Fatalf("列表 items = %v, want 1 篇", list["items"])
	}
	first := posts[0].(map[string]any)
	if first["content"] != "第一篇" {
		t.Errorf("列表內容 = %v", first["content"])
	}
	if fmt.Sprint(first["author"]) == "owner@example.com" {
		t.Error("列表把 email 當成作者名稱（必須是匿名代號或暱稱）")
	}
	if first["likeCount"] != float64(0) || first["commentCount"] != float64(0) {
		t.Errorf("新貼文計數 = like %v / comment %v, want 0/0", first["likeCount"], first["commentCount"])
	}

	// 3. 編輯：200、內容更新、updated_at 被寫入（「已編輯」旗標的來源）。
	recorder = httptest.NewRecorder()
	server.handleForumPostAction(recorder,
		loggedInRequest(t, server, http.MethodPut, fmt.Sprintf("/api/forum/posts/%d", id), `{"content":"第一篇（已修改）"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("編輯回應 = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	_, content, _, updatedAt := dbPostRow(t, db, id)
	if content != "第一篇（已修改）" {
		t.Errorf("編輯後內容 = %q", content)
	}
	if !updatedAt.Valid {
		t.Error("編輯沒有寫入 updated_at：前端的「已編輯」旗標永遠不會出現")
	}

	// 列表反映編輯後的內容（編輯與索引/列表同步）。
	recorder = httptest.NewRecorder()
	server.handleForumPosts(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/posts", nil))
	if first := decodeJSONBody(t, recorder)["items"].([]any)[0].(map[string]any); first["content"] != "第一篇（已修改）" {
		t.Errorf("編輯後列表內容 = %v", first["content"])
	}

	// 4. 軟刪除：列表與詳情都看不見，但列還在（deleted_at 已設）。
	recorder = httptest.NewRecorder()
	server.handleForumPostAction(recorder,
		loggedInRequest(t, server, http.MethodDelete, fmt.Sprintf("/api/forum/posts/%d", id), ""))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("刪除回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}

	_, _, deletedAt, _ = dbPostRow(t, db, id)
	if !deletedAt.Valid {
		t.Error("刪除沒有軟刪（deleted_at 為 NULL）：還原功能的前提不存在")
	}

	recorder = httptest.NewRecorder()
	server.handleForumPosts(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/posts", nil))
	if posts := decodeJSONBody(t, recorder)["items"].([]any); len(posts) != 0 {
		t.Errorf("軟刪除後列表仍有 %d 篇", len(posts))
	}

	recorder = httptest.NewRecorder()
	server.handleForumPostDetail(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/forum/posts/%d", id), nil))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("軟刪除後詳情回應 = %d, want 404", recorder.Code)
	}
}

/* ==========================================================================
   驗證邊界
   ========================================================================== */

func TestCreatePostValidationMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	cases := []struct {
		name string
		body string
	}{
		{"空白內容", `{"content":"   "}`},
		{"超過 10000 rune", fmt.Sprintf(`{"content":%q}`, strings.Repeat("字", 10001))},
		{"第三方圖片網址", `{"content":"x","imageUrl":"https://evil.example/x.png"}`},
		{"不是 JSON", `not json`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.handleForumPosts(recorder, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", c.body))
			if recorder.Code != http.StatusBadRequest {
				t.Errorf("回應 = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	// 剛好 10000 rune 是允許的（上限本身不是錯誤）。
	okRecorder := httptest.NewRecorder()
	server.handleForumPosts(okRecorder, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts",
		fmt.Sprintf(`{"content":%q}`, strings.Repeat("字", 10000))))
	if okRecorder.Code != http.StatusCreated {
		t.Errorf("10000 rune 的貼文回應 = %d, want 201; body=%s", okRecorder.Code, okRecorder.Body.String())
	}
}

/* ==========================================================================
   按讚往返
   ========================================================================== */

func TestLikeToggleRoundTripMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	create := httptest.NewRecorder()
	server.handleForumPosts(create, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"按讚目標"}`))
	id := postIDFromBody(t, decodeJSONBody(t, create))

	countLikes := func() int64 {
		t.Helper()
		var n int64
		if err := db.QueryRow(`SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ?`, id).Scan(&n); err != nil {
			t.Fatalf("計算讚數失敗: %v", err)
		}
		return n
	}

	// 讚一次：列存在、列表計數為 1、liked_by_me 為 1。
	recorder := httptest.NewRecorder()
	server.handleForumPostLike(recorder, loggedInRequest(t, server, http.MethodPost, fmt.Sprintf("/api/forum/posts/%d/like", id), ""))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("按讚回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}
	if got := countLikes(); got != 1 {
		t.Fatalf("按讚後讚數 = %d, want 1", got)
	}

	// 列表必須帶著同一個登入者讀取：liked 是「這個登入者按過沒」，匿名讀取
	// 得到 false 是正確行為，那驗不到按讚狀態的落庫。
	listing := httptest.NewRecorder()
	server.handleForumPosts(listing, loggedInRequest(t, server, http.MethodGet, "/api/forum/posts", ""))
	first := decodeJSONBody(t, listing)["items"].([]any)[0].(map[string]any)
	if first["likeCount"] != float64(1) || first["liked"] != true {
		t.Errorf("列表計數 = like %v / liked %v, want 1/true", first["likeCount"], first["liked"])
	}

	// 再按一次（取消讚）：列消失、計數回 0。
	recorder = httptest.NewRecorder()
	server.handleForumPostLike(recorder, loggedInRequest(t, server, http.MethodPost, fmt.Sprintf("/api/forum/posts/%d/like", id), ""))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("取消讚回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}
	if got := countLikes(); got != 0 {
		t.Errorf("取消讚後讚數 = %d, want 0", got)
	}
}

/* ==========================================================================
   留言往返
   ========================================================================== */

func TestCommentLifecycleMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	create := httptest.NewRecorder()
	server.handleForumPosts(create, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"留言宿主"}`))
	id := postIDFromBody(t, decodeJSONBody(t, create))

	// 建立留言：201、post 的 comment_count 變 1。
	recorder := httptest.NewRecorder()
	server.handleForumComments(recorder, loggedInRequest(t, server, http.MethodPost,
		fmt.Sprintf("/api/forum/posts/%d/comments", id), `{"content":"第一則留言"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("留言回應 = %d, want 201; body=%s", recorder.Code, recorder.Body.String())
	}
	comment := decodeJSONBody(t, recorder)
	commentItem, ok := comment["item"].(map[string]any)
	if !ok {
		t.Fatalf("留言回應缺少 item: %v", comment)
	}
	commentID, ok := commentItem["id"].(float64)
	if !ok {
		t.Fatalf("留言 item 缺少 id: %v", commentItem)
	}

	listing := httptest.NewRecorder()
	server.handleForumPosts(listing, httptest.NewRequest(http.MethodGet, "/api/forum/posts", nil))
	first := decodeJSONBody(t, listing)["items"].([]any)[0].(map[string]any)
	if first["commentCount"] != float64(1) {
		t.Errorf("commentCount = %v, want 1", first["commentCount"])
	}

	// 編輯留言：資料庫內容更新。
	recorder = httptest.NewRecorder()
	server.handleForumPostAction(recorder, loggedInRequest(t, server, http.MethodPut,
		fmt.Sprintf("/api/forum/posts/%d/comments/%d", id, int64(commentID)), `{"content":"改過的留言"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("編輯留言回應 = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var commentContent string
	if err := db.QueryRow(`SELECT content FROM forum_post_comments WHERE id = ?`, int64(commentID)).Scan(&commentContent); err != nil {
		t.Fatalf("讀取留言失敗: %v", err)
	}
	if commentContent != "改過的留言" {
		t.Errorf("編輯後留言 = %q", commentContent)
	}

	// 刪除留言：列消失、計數回 0。
	recorder = httptest.NewRecorder()
	server.handleForumPostAction(recorder, loggedInRequest(t, server, http.MethodDelete,
		fmt.Sprintf("/api/forum/posts/%d/comments/%d", id, int64(commentID)), ""))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("刪除留言回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}
	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM forum_post_comments WHERE id = ?`, int64(commentID)).Scan(&remaining); err != nil {
		t.Fatalf("計算留言失敗: %v", err)
	}
	if remaining != 0 {
		t.Errorf("刪除後留言列數 = %d, want 0", remaining)
	}
}

/* ==========================================================================
   檢舉與我的貼文
   ========================================================================== */

func TestReportPostWritesRowMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	create := httptest.NewRecorder()
	server.handleForumPosts(create, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"被檢舉的貼文"}`))
	id := postIDFromBody(t, decodeJSONBody(t, create))

	recorder := httptest.NewRecorder()
	server.handleForumPostAction(recorder, loggedInRequest(t, server, http.MethodPost,
		fmt.Sprintf("/api/forum/posts/%d/report", id), `{"reason":"垃圾內容"}`))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("檢舉回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}

	var targetType, targetID, reason string
	if err := db.QueryRow(`SELECT target_type, target_id, reason FROM forum_reports ORDER BY id DESC LIMIT 1`).
		Scan(&targetType, &targetID, &reason); err != nil {
		t.Fatalf("讀檢舉列失敗: %v", err)
	}
	if targetType != "post" || targetID != fmt.Sprint(id) || reason != "垃圾內容" {
		t.Errorf("檢舉列 = (%q, %q, %q), want (post, %d, 垃圾內容)", targetType, targetID, reason, id)
	}
}

func TestMyPostsReturnsOnlyOwnPostsMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	// owner 發一篇。
	create := httptest.NewRecorder()
	server.handleForumPosts(create, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"我的"}`))
	if create.Code != http.StatusCreated {
		t.Fatalf("發文回應 = %d; body=%s", create.Code, create.Body.String())
	}

	// 別人發一篇（另一個 session）。
	otherToken, err := server.sessions.Create("other@example.com", false)
	if err != nil {
		t.Fatalf("建立第二個 session 失敗: %v", err)
	}
	otherReq := httptest.NewRequest(http.MethodPost, "/api/forum/posts", strings.NewReader(`{"content":"別人的"}`))
	otherReq.AddCookie(&http.Cookie{Name: "FORUM_test", Value: otherToken})
	otherReq.Header.Set("Origin", covOrigin)
	otherRecorder := httptest.NewRecorder()
	server.handleForumPosts(otherRecorder, otherReq)
	if otherRecorder.Code != http.StatusCreated {
		t.Fatalf("第二篇發文回應 = %d; body=%s", otherRecorder.Code, otherRecorder.Body.String())
	}

	// owner 的「我的貼文」只回一篇。
	recorder := httptest.NewRecorder()
	server.handleForumMyPosts(recorder, loggedInRequest(t, server, http.MethodGet, "/api/forum/my-posts", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("我的貼文回應 = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	posts := decodeJSONBody(t, recorder)["items"].([]any)
	if len(posts) != 1 {
		t.Fatalf("我的貼文 = %d 篇, want 1", len(posts))
	}
	if posts[0].(map[string]any)["content"] != "我的" {
		t.Errorf("我的貼文內容 = %v", posts[0].(map[string]any)["content"])
	}

	// 未登入的「我的貼文」：401（它比公開列表多一道登入要求）。
	anon := httptest.NewRecorder()
	server.handleForumMyPosts(anon, httptest.NewRequest(http.MethodGet, "/api/forum/my-posts", nil))
	if anon.Code != http.StatusUnauthorized {
		t.Errorf("未登入的我的貼文 = %d, want 401", anon.Code)
	}
}

/* ==========================================================================
   詳情的 404
   ========================================================================== */

func TestPostDetailMissingAndDeletedAre404MySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	recorder := httptest.NewRecorder()
	server.handleForumPostDetail(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/posts/99999", nil))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("不存在的貼文詳情 = %d, want 404", recorder.Code)
	}

	// 未登入的詳情人可讀（永久連結刻意不需登入）。
	create := httptest.NewRecorder()
	server.handleForumPosts(create, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"公開可讀"}`))
	id := postIDFromBody(t, decodeJSONBody(t, create))
	public := httptest.NewRecorder()
	server.handleForumPostDetail(public, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/forum/posts/%d", id), nil))
	if public.Code != http.StatusOK {
		t.Errorf("未登入的詳情 = %d, want 200（永久連結不需登入）", public.Code)
	}
}

// keep context import used (subtree helpers需要 ctx 的變數在其他檔案).
var _ = context.Background
