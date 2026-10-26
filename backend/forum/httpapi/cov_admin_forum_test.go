/*
後臺論壇管理端點的覆蓋測試（httpapi/cov_admin_forum_test.go）。

這一份補的是 forum_admin_handlers.go、audit_log.go、monitoring.go 與
stats_handlers.go 裡「需要資料庫才會走到」的成功路徑。既有的
forum_admin_mysql_test.go 需要真的 MySQL，這裡改用 cov_harness_test.go 的
in-memory 驅動：查詢只要事先登記結果集，handler 就能一路走到寫入、稽核與
回應組裝。

測試原則
  - 每個失敗分支（401 / 400 / 403 / 404 / 405 / 500）都至少有一個案例，
    因為那些分支各自對應一個獨立的 return。
  - 寫入路徑一定同時斷言稽核 INSERT 有被執行：稽核與操作共用同一個交易，
    漏掉它等於讓「誰改了什麼」的保證失效。
  - 交易內的 SELECT 一律登記結果集，未登記的 QueryRow 會得到 sql.ErrNoRows，
    那是 404 分支而不是成功路徑。

命名一律以 TestCovAdminForum 開頭、helper 以 covAdminForum 開頭，避免與
其他並行撰寫的覆蓋測試檔衝突。
*/

package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forum/forum/audit"

	"github.com/redis/go-redis/v9"
)

// covAdminForumTime 是固定的時間戳，讓掃描 time.Time 的欄位不必用 time.Now()。
func covAdminForumTime() time.Time {
	return time.Date(2026, 10, 1, 8, 30, 0, 0, time.UTC)
}

// covAdminForumWantBody 檢查回應本文包含所有片段。
func covAdminForumWantBody(t *testing.T, recorder *httptest.ResponseRecorder, wants ...string) {
	t.Helper()
	body := recorder.Body.String()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("body 缺少 %q：%s", want, body)
		}
	}
}

// covAdminForumUntrusted 送出帶有不可信 Origin 的請求（h.do 永遠帶可信來源）。
func covAdminForumUntrusted(h *covHarness, handler http.HandlerFunc, method, path, body, email string, admin bool) *httptest.ResponseRecorder {
	h.t.Helper()
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Origin", "https://evil.example")
	if email != "" {
		request.AddCookie(h.login(email, admin))
	}
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

// covAdminForumReportCols 是 forum_reports 列表／單筆查詢的欄位順序。
func covAdminForumReportCols() []string {
	return []string{
		"id", "target_type", "target_id", "reporter_email", "reason", "status",
		"created_at", "reviewed_at", "reviewed_by", "target_author", "target_content",
	}
}

// covAdminForumPostCols 是後臺文章列表／搜尋的欄位順序。
func covAdminForumPostCols() []string {
	return []string{"id", "author_email", "content", "created_at", "image_url", "like_count", "comment_count", "pinned"}
}

// covAdminForumCommentCols 是後臺留言查詢的欄位順序。
func covAdminForumCommentCols() []string {
	return []string{"id", "post_id", "parent_id", "depth", "author_email", "content", "created_at"}
}

/* ==========================================================================
   requireAdminForum / handleAdminForumPosts
   ========================================================================== */

func TestCovAdminForumRequireAdminForum(t *testing.T) {
	h := newCovHarness(t)

	t.Run("未登入", func(t *testing.T) {
		recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})
	t.Run("非管理員", func(t *testing.T) {
		recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "user@example.com", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})
	t.Run("直接呼叫", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := h.request(http.MethodGet, "/api/admin/forum/posts", "", "", false)
		if h.srv.requireAdminForum(recorder, request) {
			t.Fatal("requireAdminForum 對未登入請求回 true")
		}
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})
}

func TestCovAdminForumPostsMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodPut, "/api/admin/forum/posts", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumPostsListEmpty(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL", []string{"count"}, []driverValue{int64(0)})

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"items":[]`, `"total":0`, `"page":1`, `"pages":0`)
	if !h.sawQuery("LIMIT ? OFFSET ?") {
		t.Error("沒有執行文章列表查詢")
	}
}

func TestCovAdminForumPostsListWithRows(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL", []string{"count"}, []driverValue{int64(26)})
	h.rows("SELECT id, author_email, content, created_at, image_url", covAdminForumPostCols(),
		[]driverValue{int64(1), "a@example.com", "第一篇", covAdminForumTime(), "photo.png", int64(3), int64(2), int64(1)},
		[]driverValue{int64(2), "b@example.com", "第二篇", covAdminForumTime(), "second.png", int64(0), int64(0), int64(0)},
	)
	h.rows("FROM forum_post_comments WHERE post_id = ?", covAdminForumCommentCols(),
		[]driverValue{int64(10), int64(1), int64(0), int64(1), "c@example.com", "第一則留言", covAdminForumTime()},
	)

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts?page=1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder,
		`"total":26`, `"pages":2`, `"authorEmail":"a@example.com"`,
		`"pinned":true`, `"commentCount":2`, `"content":"第一則留言"`,
	)
	if !h.sawQuery("FROM forum_post_comments WHERE post_id = ?") {
		t.Error("沒有載入文章的留言")
	}
}

func TestCovAdminForumPostsListDeletedFilter(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NOT NULL", []string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT id, author_email, content, created_at, image_url", covAdminForumPostCols(),
		[]driverValue{int64(9), "gone@example.com", "被刪除的", covAdminForumTime(), "", int64(0), int64(0), int64(0)},
	)

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts?deleted=1&page=abc", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"deleted":true`, `"total":1`, `"page":1`)
	if !h.sawQuery("ORDER BY deleted_at DESC, id DESC") {
		t.Error("已刪除清單沒有依刪除時間排序")
	}
}

func TestCovAdminForumPostsListCountError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL", errors.New("db down"))

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostsListRowsError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL", []string{"count"}, []driverValue{int64(1)})
	h.queryError("SELECT id, author_email, content, created_at, image_url", errors.New("db down"))

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostsListScanError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL", []string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT id, author_email, content, created_at, image_url", covAdminForumPostCols(),
		[]driverValue{"not-an-id", "a@example.com", "x", covAdminForumTime(), "", int64(0), int64(0), int64(0)},
	)

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostsCreateOK(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodPost, "/api/admin/forum/posts",
		`{"content":"  後臺公告  "}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusCreated)
	covAdminForumWantBody(t, recorder, `"ok":true`)
	covWantCreatedID(t, recorder.Body.String())
	if !h.sawQuery("INSERT INTO forum_posts") {
		t.Error("沒有執行新增文章")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("新增文章沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumPostsCreateInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodPost, "/api/admin/forum/posts",
		`{"content":`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumPostsCreateBlankContent(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodPost, "/api/admin/forum/posts",
		`{"content":"   "}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covAdminForumWantBody(t, recorder, "內容不可為空")
}

func TestCovAdminForumPostsCreateTooLong(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodPost, "/api/admin/forum/posts",
		`{"content":"`+strings.Repeat("字", 10001)+`"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covAdminForumWantBody(t, recorder, "內容最多 10000 字")
}

func TestCovAdminForumPostsCreateUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covAdminForumUntrusted(h, h.srv.handleAdminForumPosts, http.MethodPost,
		"/api/admin/forum/posts", `{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)
	if h.sawQuery("INSERT INTO forum_posts") {
		t.Error("不可信來源仍然執行了寫入")
	}
}

func TestCovAdminForumPostsCreateInsertError(t *testing.T) {
	h := newCovHarness(t)
	h.execError("INSERT INTO forum_posts", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodPost, "/api/admin/forum/posts",
		`{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostsCreateAuditError(t *testing.T) {
	h := newCovHarness(t)
	h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodPost, "/api/admin/forum/posts",
		`{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

/* ==========================================================================
   handleAdminForumPost（PUT / DELETE / 405）
   ========================================================================== */

func TestCovAdminForumPostUpdateOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT content FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"content"}, []driverValue{"舊內容"})

	recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, "/api/admin/forum/posts/5",
		`{"content":"新內容"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	if !h.sawQuery("UPDATE forum_posts SET content = ?") {
		t.Error("沒有執行文章內容更新")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("更新文章沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumPostUpdateInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, "/api/admin/forum/posts/5",
		`{"content":`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumPostUpdateBlankContent(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, "/api/admin/forum/posts/5",
		`{"content":" "}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumPostUpdateNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, "/api/admin/forum/posts/5",
		`{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumPostUpdateExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT content FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"content"}, []driverValue{"舊"})
	h.execError("UPDATE forum_posts SET content", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, "/api/admin/forum/posts/5",
		`{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostDeleteOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT author_email, content FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"author_email", "content"}, []driverValue{"a@example.com", "要刪的"})

	recorder := h.do(h.srv.handleAdminForumPost, http.MethodDelete, "/api/admin/forum/posts/5", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	if !h.sawQuery("UPDATE forum_posts SET deleted_at = ?") {
		t.Error("沒有執行軟刪除")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("刪除文章沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumPostDeleteNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPost, http.MethodDelete, "/api/admin/forum/posts/5", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumPostDeleteExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT author_email, content FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"author_email", "content"}, []driverValue{"a@example.com", "x"})
	h.execError("UPDATE forum_posts SET deleted_at = ?", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumPost, http.MethodDelete, "/api/admin/forum/posts/5", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostInvalidID(t *testing.T) {
	h := newCovHarness(t)
	for _, path := range []string{"/api/admin/forum/posts/abc", "/api/admin/forum/posts/0"} {
		recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, path, `{"content":"x"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	}
}

func TestCovAdminForumPostUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covAdminForumUntrusted(h, h.srv.handleAdminForumPost, http.MethodPut,
		"/api/admin/forum/posts/5", `{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)
}

func TestCovAdminForumPostMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumPost, http.MethodGet, "/api/admin/forum/posts/5", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

/* ==========================================================================
   handleAdminPostOrPin / handleAdminPostPin
   ========================================================================== */

func TestCovAdminForumPostOrPinDispatch(t *testing.T) {
	t.Run("pin", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"pinned"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleAdminPostOrPin, http.MethodPost, "/api/admin/forum/posts/7/pin",
			`{"pinned":true}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covAdminForumWantBody(t, recorder, `"pinned":true`)
	})
	t.Run("restore", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT deleted_at, content FROM forum_posts WHERE id = ?",
			[]string{"deleted_at", "content"}, []driverValue{nil, "內容"})
		recorder := h.do(h.srv.handleAdminPostOrPin, http.MethodPost, "/api/admin/forum/posts/7/restore",
			"", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covAdminForumWantBody(t, recorder, `"restored":false`)
	})
	t.Run("單篇文章", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT content FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"content"}, []driverValue{"舊"})
		recorder := h.do(h.srv.handleAdminPostOrPin, http.MethodPut, "/api/admin/forum/posts/7",
			`{"content":"新"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
	})
}

func TestCovAdminForumPostPinOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"pinned"}, []driverValue{int64(0)})

	recorder := h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/3/pin",
		`{"pinned":true}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"pinned":true`)
	if !h.sawQuery("UPDATE forum_posts SET pinned = ?") {
		t.Error("沒有執行置頂更新")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("置頂沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumPostUnpinOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"pinned"}, []driverValue{int64(1)})

	recorder := h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/3/pin",
		`{"pinned":false}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"pinned":false`)
}

func TestCovAdminForumPostPinNoChange(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"pinned"}, []driverValue{int64(1)})

	recorder := h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/3/pin",
		`{"pinned":true}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	if h.sawQuery("UPDATE forum_posts SET pinned = ?") {
		t.Error("狀態沒變卻執行了 UPDATE")
	}
}

func TestCovAdminForumPostPinNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/3/pin",
		`{"pinned":true}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumPostPinExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"pinned"}, []driverValue{int64(0)})
	h.execError("UPDATE forum_posts SET pinned", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/3/pin",
		`{"pinned":true}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostPinInvalidInput(t *testing.T) {
	h := newCovHarness(t)

	recorder := h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/abc/pin",
		`{"pinned":true}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)

	recorder = h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/3/pin",
		`{"pinned":`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumPostPinGuards(t *testing.T) {
	h := newCovHarness(t)

	recorder := h.do(h.srv.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/3/pin",
		`{"pinned":true}`, "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)

	recorder = h.do(h.srv.handleAdminPostPin, http.MethodGet, "/api/admin/forum/posts/3/pin",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)

	recorder = covAdminForumUntrusted(h, h.srv.handleAdminPostPin, http.MethodPost,
		"/api/admin/forum/posts/3/pin", `{"pinned":true}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)
}

/* ==========================================================================
   handleAdminPostRestore
   ========================================================================== */

func TestCovAdminForumPostRestoreOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT deleted_at, content FROM forum_posts WHERE id = ?",
		[]string{"deleted_at", "content"}, []driverValue{covAdminForumTime(), "被刪的內容"})

	recorder := h.do(h.srv.handleAdminPostRestore, http.MethodPost, "/api/admin/forum/posts/4/restore",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"restored":true`)
	if !h.sawQuery("UPDATE forum_posts SET deleted_at = NULL") {
		t.Error("沒有執行還原更新")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("還原沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumPostRestoreNotDeleted(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT deleted_at, content FROM forum_posts WHERE id = ?",
		[]string{"deleted_at", "content"}, []driverValue{nil, "還在的內容"})

	recorder := h.do(h.srv.handleAdminPostRestore, http.MethodPost, "/api/admin/forum/posts/4/restore",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"restored":false`)
	if h.sawQuery("UPDATE forum_posts SET deleted_at = NULL") {
		t.Error("未刪除的貼文仍執行了 UPDATE")
	}
}

func TestCovAdminForumPostRestoreNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminPostRestore, http.MethodPost, "/api/admin/forum/posts/4/restore",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumPostRestoreExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT deleted_at, content FROM forum_posts WHERE id = ?",
		[]string{"deleted_at", "content"}, []driverValue{covAdminForumTime(), "內容"})
	h.execError("UPDATE forum_posts SET deleted_at = NULL", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminPostRestore, http.MethodPost, "/api/admin/forum/posts/4/restore",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumPostRestoreGuards(t *testing.T) {
	h := newCovHarness(t)

	recorder := h.do(h.srv.handleAdminPostRestore, http.MethodPost, "/api/admin/forum/posts/4/restore",
		"", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)

	recorder = h.do(h.srv.handleAdminPostRestore, http.MethodGet, "/api/admin/forum/posts/4/restore",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)

	recorder = covAdminForumUntrusted(h, h.srv.handleAdminPostRestore, http.MethodPost,
		"/api/admin/forum/posts/4/restore", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)

	for _, path := range []string{"/api/admin/forum/posts/0/restore", "/api/admin/forum/posts/abc/restore"} {
		recorder = h.do(h.srv.handleAdminPostRestore, http.MethodPost, path, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	}
}

func TestCovAdminForumPostRestoreIDFromPath(t *testing.T) {
	cases := []struct {
		path string
		id   int64
		ok   bool
	}{
		{"/api/admin/forum/posts/7/restore", 7, true},
		{"/api/admin/forum/posts/0/restore", 0, true},
		{"/api/admin/forum/posts/abc/restore", 0, false},
		{"/api/admin/forum/posts/7/restore/", 0, false},
		{"/wrong/7/restore", 0, false},
	}
	for _, testCase := range cases {
		id, err := adminPostRestoreIDFromPath(testCase.path)
		if testCase.ok {
			if err != nil || id != testCase.id {
				t.Errorf("adminPostRestoreIDFromPath(%q) = (%d, %v), want (%d, nil)", testCase.path, id, err, testCase.id)
			}
			continue
		}
		if err == nil {
			t.Errorf("adminPostRestoreIDFromPath(%q) 應該失敗，得到 %d", testCase.path, id)
		}
	}
}

/* ==========================================================================
   handleAdminForumComments（集合）
   ========================================================================== */

func TestCovAdminForumCommentsRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodGet, "/api/admin/forum/comments?postId=1", "", "user@example.com", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumCommentsMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPut, "/api/admin/forum/comments?postId=1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumCommentsGetInvalidPostID(t *testing.T) {
	h := newCovHarness(t)
	for _, path := range []string{"/api/admin/forum/comments", "/api/admin/forum/comments?postId=0", "/api/admin/forum/comments?postId=abc"} {
		recorder := h.do(h.srv.handleAdminForumComments, http.MethodGet, path, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	}
}

func TestCovAdminForumCommentsGetEmpty(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodGet, "/api/admin/forum/comments?postId=1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"items":[]`)
	if !h.sawQuery("FROM forum_post_comments c") {
		t.Error("沒有執行留言列表查詢")
	}
}

func TestCovAdminForumCommentsGetRows(t *testing.T) {
	h := newCovHarness(t)
	h.rows("FROM forum_post_comments c", covAdminForumCommentCols(),
		[]driverValue{int64(11), int64(1), int64(0), int64(1), "c@example.com", "第一層", covAdminForumTime()},
		[]driverValue{int64(12), int64(1), int64(11), int64(2), "d@example.com", "第二層", covAdminForumTime()},
	)

	recorder := h.do(h.srv.handleAdminForumComments, http.MethodGet, "/api/admin/forum/comments?postId=1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"content":"第一層"`, `"depth":2`, `"parentId":11`)
}

func TestCovAdminForumCommentsGetQueryError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("FROM forum_post_comments c", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodGet, "/api/admin/forum/comments?postId=1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumCommentsGetScanError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("FROM forum_post_comments c", covAdminForumCommentCols(),
		[]driverValue{"not-an-id", int64(1), int64(0), int64(1), "c@example.com", "x", covAdminForumTime()},
	)
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodGet, "/api/admin/forum/comments?postId=1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumCommentsCreateOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})

	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":1,"content":"  管理員留言  "}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusCreated)
	covAdminForumWantBody(t, recorder, `"ok":true`)
	covWantCreatedID(t, recorder.Body.String())
	if !h.sawQuery("INSERT INTO forum_post_comments") {
		t.Error("沒有執行新增留言")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("新增留言沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumCommentsCreateUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covAdminForumUntrusted(h, h.srv.handleAdminForumComments, http.MethodPost,
		"/api/admin/forum/comments", `{"postId":1,"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)
}

func TestCovAdminForumCommentsCreateInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumCommentsCreateBlankContent(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":1,"content":"  "}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covAdminForumWantBody(t, recorder, "留言內容不可為空")
}

func TestCovAdminForumCommentsCreateTooLong(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":1,"content":"`+strings.Repeat("字", 2001)+`"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covAdminForumWantBody(t, recorder, "留言最多 2000 字")
}

func TestCovAdminForumCommentsCreatePostNotFound(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(0)})
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":1,"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumCommentsCreateCountError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":1,"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumCommentsCreateInsertError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
	h.execError("INSERT INTO forum_post_comments", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":1,"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

/* ==========================================================================
   handleAdminForumComment（單則）
   ========================================================================== */

func TestCovAdminForumCommentRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
		`{"content":"x"}`, "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumCommentInvalidID(t *testing.T) {
	h := newCovHarness(t)
	for _, path := range []string{"/api/admin/forum/comments/abc", "/api/admin/forum/comments/0"} {
		recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, path,
			`{"content":"x"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	}
}

func TestCovAdminForumCommentUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covAdminForumUntrusted(h, h.srv.handleAdminForumComment, http.MethodPut,
		"/api/admin/forum/comments/9", `{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)
}

func TestCovAdminForumCommentMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodGet, "/api/admin/forum/comments/9", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumCommentUpdateOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT content FROM forum_post_comments WHERE id = ?",
		[]string{"content"}, []driverValue{"舊留言"})

	recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
		`{"content":"新留言"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	if !h.sawQuery("UPDATE forum_post_comments SET content = ?") {
		t.Error("沒有執行留言更新")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("更新留言沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumCommentUpdateInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
		`{"content":`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumCommentUpdateBlankContent(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
		`{"content":" "}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumCommentUpdateNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
		`{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumCommentUpdateExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT content FROM forum_post_comments WHERE id = ?",
		[]string{"content"}, []driverValue{"舊"})
	h.execError("UPDATE forum_post_comments SET content", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
		`{"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumCommentDeleteOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT author_email, content, post_id FROM forum_post_comments WHERE id = ?",
		[]string{"author_email", "content", "post_id"},
		[]driverValue{"c@example.com", "要刪的留言", int64(3)})

	recorder := h.do(h.srv.handleAdminForumComment, http.MethodDelete, "/api/admin/forum/comments/9", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"ok":true`, `"removed":1`)
	if !h.sawQuery("DELETE c ") {
		t.Error("沒有執行留言子樹刪除")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("刪除留言沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumCommentDeleteNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodDelete, "/api/admin/forum/comments/9", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumCommentDeleteExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT author_email, content, post_id FROM forum_post_comments WHERE id = ?",
		[]string{"author_email", "content", "post_id"},
		[]driverValue{"c@example.com", "x", int64(3)})
	h.execError("DELETE c ", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumComment, http.MethodDelete, "/api/admin/forum/comments/9", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

/* ==========================================================================
   handleAdminForumReports（集合）
   ========================================================================== */

func TestCovAdminForumReportsRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodGet, "/api/admin/forum/reports", "", "user@example.com", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumReportsMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPut, "/api/admin/forum/reports", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumReportsInvalidStatus(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodGet, "/api/admin/forum/reports?status=closed", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covAdminForumWantBody(t, recorder, "invalid report status")
}

func TestCovAdminForumReportsEmpty(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodGet, "/api/admin/forum/reports", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"items":[]`, `"total":0`, `"status":"PENDING"`)
	if !h.sawQuery("WHERE status = ?") {
		t.Error("預設狀態沒有附加篩選條件")
	}
}

func TestCovAdminForumReportsAll(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodGet, "/api/admin/forum/reports?status=all", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"status":"ALL"`, `"items":[]`)
	if h.sawQuery("WHERE status = ?") {
		t.Error("status=ALL 仍然附加了篩選條件")
	}
}

func TestCovAdminForumReportsRows(t *testing.T) {
	h := newCovHarness(t)
	h.rows("FROM forum_reports", covAdminForumReportCols(),
		[]driverValue{int64(1), "post", int64(3), "r@example.com", "spam", "PENDING", covAdminForumTime(), nil, "", "a@example.com", "被檢舉的貼文"},
		[]driverValue{int64(2), "comment", int64(4), "r2@example.com", "rude", "RESOLVED", covAdminForumTime(), covAdminForumTime(), "admin@example.com", "c@example.com", "被檢舉的留言"},
	)

	recorder := h.do(h.srv.handleAdminForumReports, http.MethodGet, "/api/admin/forum/reports?status=resolved", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder,
		`"total":2`, `"status":"RESOLVED"`, `"targetAuthor":"a@example.com"`,
		`"targetContent":"被檢舉的留言"`, `"reviewedBy":"admin@example.com"`,
	)
}

func TestCovAdminForumReportsQueryError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("FROM forum_reports", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodGet, "/api/admin/forum/reports", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportsCreateOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})

	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
		`{"targetType":" POST ","targetId":3,"reporterEmail":" r@example.com ","reason":" 垃圾訊息 ","status":"pending"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusCreated)
	covAdminForumWantBody(t, recorder, `"ok":true`)
	covWantCreatedID(t, recorder.Body.String())
	if !h.sawQuery("INSERT INTO forum_reports") {
		t.Error("沒有執行新增檢舉")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("新增檢舉沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumReportsCreateCommentTarget(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_post_comments WHERE id = ?",
		[]string{"count"}, []driverValue{int64(1)})

	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
		`{"targetType":"comment","targetId":9,"reporterEmail":"r@example.com","reason":"不當留言","status":"RESOLVED"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusCreated)
	if !h.sawQuery("SELECT COUNT(*) FROM forum_post_comments WHERE id = ?") {
		t.Error("留言目標沒有查證存在性")
	}
}

func TestCovAdminForumReportsCreateInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
		`{"targetType":`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumReportsCreateInvalidFields(t *testing.T) {
	h := newCovHarness(t)
	cases := []string{
		`{"targetType":"video","targetId":1,"reporterEmail":"r@example.com","reason":"x"}`,
		`{"targetType":"post","targetId":0,"reporterEmail":"r@example.com","reason":"x"}`,
		`{"targetType":"post","targetId":1,"reporterEmail":"","reason":"x"}`,
		`{"targetType":"post","targetId":1,"reporterEmail":"r@example.com","reason":""}`,
	}
	for _, body := range cases {
		recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
			body, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	}
}

func TestCovAdminForumReportsCreateTargetMissing(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(0)})
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
		`{"targetType":"post","targetId":3,"reporterEmail":"r@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumReportsCreateTargetError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
		`{"targetType":"post","targetId":3,"reporterEmail":"r@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportsCreateInsertError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
	h.execError("INSERT INTO forum_reports", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
		`{"targetType":"post","targetId":3,"reporterEmail":"r@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportsCreateUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covAdminForumUntrusted(h, h.srv.handleAdminForumReports, http.MethodPost,
		"/api/admin/forum/reports", `{"targetType":"post","targetId":3,"reporterEmail":"r@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)
}

/* ==========================================================================
   handleAdminForumReport（單筆）
   ========================================================================== */

func TestCovAdminForumReportRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodGet, "/api/admin/forum/reports/1", "", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumReportInvalidID(t *testing.T) {
	h := newCovHarness(t)
	for _, path := range []string{"/api/admin/forum/reports/abc", "/api/admin/forum/reports/0"} {
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodGet, path, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	}
}

func TestCovAdminForumReportGetOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("FROM forum_reports WHERE id = ?", covAdminForumReportCols(),
		[]driverValue{int64(1), "post", int64(3), "r@example.com", "spam", "PENDING", covAdminForumTime(), nil, "", "a@example.com", "貼文"},
	)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodGet, "/api/admin/forum/reports/1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"item":`, `"targetType":"post"`, `"reason":"spam"`, `"targetAuthor":"a@example.com"`)
}

func TestCovAdminForumReportGetNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodGet, "/api/admin/forum/reports/1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumReportGetQueryError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("FROM forum_reports WHERE id = ?", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodGet, "/api/admin/forum/reports/1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportUntrustedWrite(t *testing.T) {
	h := newCovHarness(t)
	recorder := covAdminForumUntrusted(h, h.srv.handleAdminForumReport, http.MethodDelete,
		"/api/admin/forum/reports/1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusForbidden)
}

func TestCovAdminForumReportDeleteOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT status, reason FROM forum_reports WHERE id = ?",
		[]string{"status", "reason"}, []driverValue{"PENDING", "spam"})

	recorder := h.do(h.srv.handleAdminForumReport, http.MethodDelete, "/api/admin/forum/reports/1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"ok":true`)
	if !h.sawQuery("DELETE FROM forum_reports") {
		t.Error("沒有執行檢舉刪除")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("刪除檢舉沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumReportDeleteNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodDelete, "/api/admin/forum/reports/1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumReportDeleteExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT status, reason FROM forum_reports WHERE id = ?",
		[]string{"status", "reason"}, []driverValue{"PENDING", "spam"})
	h.execError("DELETE FROM forum_reports", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodDelete, "/api/admin/forum/reports/1", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportPutOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT target_type, target_id, reporter_email, reason, status",
		[]string{"target_type", "target_id", "reporter_email", "reason", "status"},
		[]driverValue{"post", "3", "old@example.com", "舊原因", "PENDING"})

	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
		`{"targetType":" POST ","targetId":7,"reporterEmail":" new@example.com ","reason":" 新原因 ","status":"resolved"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"ok":true`)
	if !h.sawQuery("UPDATE forum_reports") {
		t.Error("沒有執行檢舉全欄位更新")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("成立檢舉沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumReportPutInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
		`{"targetType":`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumReportPutInvalidFields(t *testing.T) {
	h := newCovHarness(t)
	cases := []string{
		`{"targetType":"user","targetId":1,"reporterEmail":"a@example.com","reason":"x"}`,
		`{"targetType":"post","targetId":0,"reporterEmail":"a@example.com","reason":"x"}`,
		`{"targetType":"post","targetId":1,"reporterEmail":"a@example.com","reason":""}`,
		`{"targetType":"post","targetId":1,"reporterEmail":"a@example.com","reason":"x","status":"CLOSED"}`,
	}
	for _, body := range cases {
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
			body, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	}
}

func TestCovAdminForumReportPutTargetMissing(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(0)})
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
		`{"targetType":"post","targetId":7,"reporterEmail":"a@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumReportPutTargetError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
		`{"targetType":"post","targetId":7,"reporterEmail":"a@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportPutNotFound(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
		`{"targetType":"post","targetId":7,"reporterEmail":"a@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumReportPutExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT target_type, target_id, reporter_email, reason, status",
		[]string{"target_type", "target_id", "reporter_email", "reason", "status"},
		[]driverValue{"post", "3", "old@example.com", "舊原因", "PENDING"})
	h.execError("UPDATE forum_reports", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
		`{"targetType":"post","targetId":7,"reporterEmail":"a@example.com","reason":"新原因"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportPatchOK(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT reason, status FROM forum_reports WHERE id = ?",
		[]string{"reason", "status"}, []driverValue{"舊原因", "PENDING"})

	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
		`{"status":"RESOLVED","reason":" 新原因 "}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	if !h.sawQuery(", reason = ?") {
		t.Error("PATCH 帶 reason 時沒有附加更新欄位")
	}
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("PATCH 成立檢舉沒有寫入稽核紀錄")
	}
}

func TestCovAdminForumReportPatchNoStatus(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT reason, status FROM forum_reports WHERE id = ?",
		[]string{"reason", "status"}, []driverValue{"原因", "RESOLVED"})

	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
		`{}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"ok":true`)
}

func TestCovAdminForumReportPatchInvalidStatus(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
		`{"status":"CLOSED"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covAdminForumWantBody(t, recorder, "invalid report status")
}

func TestCovAdminForumReportPatchReasonTooLong(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
		`{"status":"RESOLVED","reason":"`+strings.Repeat("字", 501)+`"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covAdminForumWantBody(t, recorder, "檢舉原因最多 500 字")
}

func TestCovAdminForumReportPatchNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
		`{"status":"REJECTED"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovAdminForumReportPatchExecError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT reason, status FROM forum_reports WHERE id = ?",
		[]string{"reason", "status"}, []driverValue{"原因", "PENDING"})
	h.execError("UPDATE forum_reports SET status", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
		`{"status":"REJECTED"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportPatchMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPost, "/api/admin/forum/reports/1",
		`{"status":"RESOLVED"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumReportBadBodyBeforeMethodCheck(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumReport, http.MethodPost, "/api/admin/forum/reports/1",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumAdminReportAction(t *testing.T) {
	cases := []struct{ after, want string }{
		{"RESOLVED", adminActionReportResolve},
		{"REJECTED", adminActionReportReject},
		{"PENDING", adminActionReportUpdate},
		{"", adminActionReportUpdate},
	}
	for _, testCase := range cases {
		if got := adminReportAction("PENDING", testCase.after); got != testCase.want {
			t.Errorf("adminReportAction(PENDING, %q) = %q, want %q", testCase.after, got, testCase.want)
		}
	}
}

/* ==========================================================================
   handleAdminForumSearch
   ========================================================================== */

func TestCovAdminForumSearchRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search?q=x", "", "user@example.com", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumSearchMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodPost, "/api/admin/forum/search?q=x", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumSearchMissingQuery(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumSearchQueryTooLong(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search?q="+strings.Repeat("a", 101), "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovAdminForumSearchEmpty(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND", []string{"count"}, []driverValue{int64(0)})

	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search?q=hello&offset=2&limit=10", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"items":[]`, `"total":0`, `"engine":"mysql"`, `"query":"hello"`)
}

func TestCovAdminForumSearchRows(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND", []string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND", []string{"id"}, []driverValue{int64(7)})
	h.rows("SELECT id, author_email, content, created_at, image_url", covAdminForumPostCols(),
		[]driverValue{int64(7), "a@example.com", "命中關鍵字", covAdminForumTime(), "", int64(1), int64(0), int64(0)},
	)

	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search?q=關鍵字", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"total":1`, `"content":"命中關鍵字"`, `"engine":"mysql"`)
	if !h.sawQuery("FROM forum_post_comments WHERE post_id = ?") {
		t.Error("搜尋結果沒有載入留言")
	}
}

func TestCovAdminForumSearchCountError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search?q=x", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumSearchIDsError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND", []string{"count"}, []driverValue{int64(0)})
	h.queryError("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search?q=x", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumSearchLoadError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND", []string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND", []string{"id"}, []driverValue{int64(7)})
	h.queryError("SELECT id, author_email, content, created_at, image_url", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet, "/api/admin/forum/search?q=x", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

/* ==========================================================================
   handleAdminLog
   ========================================================================== */

func TestCovAdminForumLogRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminLog, http.MethodGet, "/api/admin/log", "", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumLogMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminLog, http.MethodPost, "/api/admin/log", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumLogEmpty(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_admin_actions", []string{"count"}, []driverValue{int64(0)})
	h.rows("SELECT DISTINCT action FROM forum_admin_actions", []string{"action"},
		[]driverValue{"post.delete"}, []driverValue{"report.resolve"})
	h.rows("SELECT DISTINCT actor_email FROM forum_admin_actions", []string{"actor_email"},
		[]driverValue{"admin@example.com"})
	h.rows("FROM forum_admin_actions ORDER BY created_at", []string{
		"id", "actor_email", "action", "target_type", "target_id", "target_label",
		"changes", "ip", "request_id", "created_at",
	})

	recorder := h.do(h.srv.handleAdminLog, http.MethodGet, "/api/admin/log", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder,
		`"items":[]`, `"total":0`, `"pageSize":50`,
		`"actions":["post.delete","report.resolve"]`,
		`"actors":["admin@example.com"]`,
	)
}

func TestCovAdminForumLogWithFilters(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_admin_actions", []string{"count"}, []driverValue{int64(1)})
	h.rows("ORDER BY created_at DESC, id DESC", []string{
		"id", "actor_email", "action", "target_type", "target_id", "target_label",
		"changes", "ip", "request_id", "created_at",
	}, []driverValue{
		int64(1), "admin@example.com", "post.delete", "post", "7", "一篇貼文",
		`[{"field":"content","before":"a","after":"b"}]`, "1.2.3.4", "req-1", "2026-10-01 10:00:00",
	})

	recorder := h.do(h.srv.handleAdminLog, http.MethodGet,
		"/api/admin/log?actor=admin@example.com&action=post.delete,report.resolve&targetType=post&targetId=7"+
			"&from=2026-09-30&to=2026-10-01T12:00&offset=abc",
		"", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"total":1`, `"action":"post.delete"`, `"actorEmail":"admin@example.com"`, `"changes"`)
	for _, clause := range []string{"actor_email = ?", "action IN (?,?)", "target_type = ?", "target_id = ?", "created_at >= ?", "created_at <= ?"} {
		if !h.sawQuery(clause) {
			t.Errorf("查詢缺少篩選條件 %q", clause)
		}
	}
}

func TestCovAdminForumLogListError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_admin_actions", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminLog, http.MethodGet, "/api/admin/log", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumLogFacetErrors(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_admin_actions", []string{"count"}, []driverValue{int64(0)})
	h.queryError("SELECT DISTINCT action FROM forum_admin_actions", errors.New("db down"))
	h.queryError("SELECT DISTINCT actor_email FROM forum_admin_actions", errors.New("db down"))
	h.rows("FROM forum_admin_actions ORDER BY created_at", []string{
		"id", "actor_email", "action", "target_type", "target_id", "target_label",
		"changes", "ip", "request_id", "created_at",
	})

	recorder := h.do(h.srv.handleAdminLog, http.MethodGet, "/api/admin/log", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"actions":[]`, `"actors":[]`)
}

func TestCovAdminForumLogParseAdminLogTime(t *testing.T) {
	cases := []struct {
		raw      string
		endOfDay bool
		ok       bool
		want     string
	}{
		{"", false, false, ""},
		{"   ", true, false, ""},
		{"not-a-date", false, false, ""},
		{"2026-09-30", false, true, "2026-09-30T00:00:00Z"},
		{"2026-09-30", true, true, "2026-09-30T23:59:59Z"},
		{"2026-09-30T12:34", true, true, "2026-09-30T12:34:00Z"},
		{"2026-09-30T12:34:56", false, true, "2026-09-30T12:34:56Z"},
	}
	for _, testCase := range cases {
		parsed, ok := parseAdminLogTime(testCase.raw, testCase.endOfDay)
		if ok != testCase.ok {
			t.Errorf("parseAdminLogTime(%q, %v) ok = %v, want %v", testCase.raw, testCase.endOfDay, ok, testCase.ok)
			continue
		}
		if ok && parsed.UTC().Format(time.RFC3339) != testCase.want {
			t.Errorf("parseAdminLogTime(%q, %v) = %s, want %s", testCase.raw, testCase.endOfDay, parsed.UTC().Format(time.RFC3339), testCase.want)
		}
	}
}

func TestCovAdminForumLogAtoiDefault(t *testing.T) {
	cases := []struct {
		raw      string
		fallback int
		want     int
	}{
		{"", 0, 0},
		{"abc", 5, 5},
		{" 7 ", 0, 7},
		{"-3", 0, -3},
	}
	for _, testCase := range cases {
		if got := atoiDefault(testCase.raw, testCase.fallback); got != testCase.want {
			t.Errorf("atoiDefault(%q, %d) = %d, want %d", testCase.raw, testCase.fallback, got, testCase.want)
		}
	}
}

func TestCovAdminForumOnlyChanged(t *testing.T) {
	changes := onlyChanged(
		audit.Change{Field: "same", Before: "a", After: "a"},
		audit.Change{Field: "changed", Before: "", After: "b"},
	)
	if len(changes) != 1 || changes[0].Field != "changed" {
		t.Fatalf("onlyChanged = %#v, want 只有 changed", changes)
	}

	change := adminStatusChange("ACTIVE", "SUSPENDED")
	if change.Field != "status" || change.Before != "ACTIVE" || change.After != "SUSPENDED" {
		t.Fatalf("adminStatusChange = %#v", change)
	}
}

func TestCovAdminForumRecordPostIndexFailure(t *testing.T) {
	h := newCovHarness(t)
	request := h.request(http.MethodPost, "/api/admin/forum/posts/1/restore", "", "admin@example.com", true)
	h.srv.recordPostIndexFailure(request, 5, errors.New("es down"))
	if !h.sawQuery("INSERT INTO forum_admin_actions") {
		t.Error("索引失敗沒有補記稽核")
	}

	// 沒有資料庫時只記錄、不 panic（beginAdminTx 回 errNoDatabase）。
	h.srv.db = nil
	h.srv.recordPostIndexFailure(request, 5, errors.New("es down"))
}

/* ==========================================================================
   monitoring.go
   ========================================================================== */

func TestCovAdminForumMonitorRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminMonitor, http.MethodGet, "/api/admin/monitor", "", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumMonitorMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminMonitor, http.MethodPost, "/api/admin/monitor", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumMonitorOK(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminMonitor, http.MethodGet, "/api/admin/monitor", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder,
		`"ok":true`, `"forum":"測試論壇"`,
		`"mysql":{"state":"ok"`, `"redis":{"state":"ok"`, `"search":{"state":"disabled"`,
		`"mode":"trusted-proxies"`, `"declared":"127.0.0.1/32"`, `"probesMs":`,
	)
}

func TestCovAdminForumMonitorNilMetrics(t *testing.T) {
	h := newCovHarness(t)
	h.srv.metrics = nil
	recorder := h.do(h.srv.handleAdminMonitor, http.MethodGet, "/api/admin/monitor", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covAdminForumWantBody(t, recorder, "metrics registry is not available")
}

func TestCovAdminForumMonitorDisabledDependencies(t *testing.T) {
	h := newCovHarness(t)
	h.srv.db = nil
	h.srv.mediaRedis = nil
	recorder := h.do(h.srv.handleAdminMonitor, http.MethodGet, "/api/admin/monitor", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"engine":"none"`, `"redis":{"state":"disabled"`)
}

func TestCovAdminForumMonitorDatabaseDown(t *testing.T) {
	h := newCovHarness(t)
	brokenDB, err := sql.Open("forum-cov-stub", "missing-cov-dsn")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer brokenDB.Close()
	h.srv.db = brokenDB

	recorder := h.do(h.srv.handleAdminMonitor, http.MethodGet, "/api/admin/monitor", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder, `"mysql":{"state":"down"`)
}

func TestCovAdminForumMonitorTrustReport(t *testing.T) {
	h := newCovHarness(t)

	report := h.srv.trustReport()
	if report.Mode != "trusted-proxies" || len(report.Trusted) != 1 || report.Declared != "127.0.0.1/32" {
		t.Fatalf("trustReport = %#v", report)
	}

	h.srv.trustedProxies = nil
	report = h.srv.trustReport()
	if report.Mode != "legacy-headers" || len(report.Trusted) != 0 {
		t.Fatalf("nil trustReport = %#v", report)
	}

	h.srv.trustedProxies = parseTrustedProxyCIDRs("10.0.0.0/8,not-a-cidr")
	report = h.srv.trustReport()
	if report.Mode != "trusted-proxies" || len(report.Invalid) != 1 || report.Invalid[0] != "not-a-cidr" {
		t.Fatalf("部分無效 trustReport = %#v", report)
	}

	h.srv.trustedProxies = parseTrustedProxyCIDRs("only-broken")
	report = h.srv.trustReport()
	if report.Mode != "legacy-headers" || len(report.Invalid) != 1 {
		t.Fatalf("全部無效 trustReport = %#v", report)
	}
}

// covAdminForumRedisCommander 是 redisCommander 的測試替身。
type covAdminForumRedisCommander struct {
	raw string
	err error
}

func (f covAdminForumRedisCommander) Info(ctx context.Context, _ ...string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "info")
	if f.err != nil {
		cmd.SetErr(f.err)
		return cmd
	}
	cmd.SetVal(f.raw)
	return cmd
}

func TestCovAdminForumMonitorRedisMemoryBytes(t *testing.T) {
	cases := []struct {
		name   string
		client covAdminForumRedisCommander
		want   int64
		ok     bool
	}{
		{"正常", covAdminForumRedisCommander{raw: "# Memory\r\nused_memory:2048\r\nused_memory_human:2K\r\n"}, 2048, true},
		{"沒有 used_memory", covAdminForumRedisCommander{raw: "used_memory_human:2K\r\n"}, 0, false},
		{"非數字", covAdminForumRedisCommander{raw: "used_memory:not-a-number\r\n"}, 0, false},
		{"INFO 失敗", covAdminForumRedisCommander{err: errors.New("redis down")}, 0, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := redisMemoryBytes(context.Background(), testCase.client)
			if ok != testCase.ok || got != testCase.want {
				t.Fatalf("redisMemoryBytes = (%d, %v), want (%d, %v)", got, ok, testCase.want, testCase.ok)
			}
		})
	}
}

func TestCovAdminForumMonitorMetricsMiddleware(t *testing.T) {
	h := newCovHarness(t)
	called := false
	handler := h.srv.metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/posts", nil))
	if !called {
		t.Fatal("metricsMiddleware 沒有呼叫下一個 handler")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("狀態碼 = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	// metrics 為 nil 時直接回傳 next，不包裝。
	bare := &Server{}
	nextCalled := false
	wrapped := bare.metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))
	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/admin/monitor", nil))
	if !nextCalled {
		t.Fatal("nil metrics 時沒有直接呼叫 next")
	}
}

/* ==========================================================================
   stats_handlers.go
   ========================================================================== */

func TestCovAdminForumStatsRequiresAdmin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "user@example.com", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovAdminForumStatsMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleAdminStats, http.MethodPost, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovAdminForumStatsEmpty(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COALESCE(SUM(new_users)", []string{"users", "posts", "comments", "likes"},
		[]driverValue{int64(0), int64(0), int64(0), int64(0)})

	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder,
		`"ok":true`, `"days":30`, `"forum":"測試論壇"`,
		`"dates":[`, `"topPosts":[]`, `"topTags":[]`, `"topAuthors":[]`,
	)
}

func TestCovAdminForumStatsWithRows(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT day, new_users, new_posts, new_comments, new_likes",
		[]string{"day", "new_users", "new_posts", "new_comments", "new_likes"},
		[]driverValue{covAdminForumTime(), int64(1), int64(2), int64(3), int64(4)},
	)
	h.rows("SELECT COALESCE(SUM(new_users)", []string{"users", "posts", "comments", "likes"},
		[]driverValue{int64(10), int64(20), int64(30), int64(40)})
	h.rows("ORDER BY 5 + 6 DESC", []string{"id", "author_email", "created_at", "excerpt", "comments", "likes"},
		[]driverValue{int64(7), "a@example.com", "2026-10-01 08:30:00", strings.Repeat("字", 90), int64(5), int64(6)},
	)
	h.rows("FROM forum_user_tags t", []string{"id", "name", "users"},
		[]driverValue{int64(1), "go", int64(9)})
	h.rows("SELECT p.author_email, p.posts", []string{"author_email", "posts", "comments"},
		[]driverValue{"a@example.com", int64(4), int64(2)})

	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats?days=7", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	covAdminForumWantBody(t, recorder,
		`"days":7`, `"users":10`, `"posts":20`, `"comments":30`, `"likes":40`,
		`"name":"go"`, `"excerpt":"`+strings.Repeat("字", 80)+`…"`,
	)
}

func TestCovAdminForumStatsSeriesError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT day, new_users, new_posts", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumStatsTotalsError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("COALESCE(SUM(new_users)", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumStatsTopPostsError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COALESCE(SUM(new_users)", []string{"users", "posts", "comments", "likes"},
		[]driverValue{int64(0), int64(0), int64(0), int64(0)})
	h.queryError("ORDER BY 5 + 6 DESC", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumStatsTopTagsError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COALESCE(SUM(new_users)", []string{"users", "posts", "comments", "likes"},
		[]driverValue{int64(0), int64(0), int64(0), int64(0)})
	h.queryError("FROM forum_user_tags t", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumStatsTopAuthorsError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COALESCE(SUM(new_users)", []string{"users", "posts", "comments", "likes"},
		[]driverValue{int64(0), int64(0), int64(0), int64(0)})
	h.queryError("SELECT p.author_email, p.posts", errors.New("db down"))
	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumStatsDays(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 30},
		{"abc", 30},
		{"0", 30},
		{"-5", 30},
		{"1", 1},
		{"7", 7},
		{"90", 90},
		{"91", 90},
		{" 12 ", 12},
	}
	for _, testCase := range cases {
		if got := statsDays(testCase.raw); got != testCase.want {
			t.Errorf("statsDays(%q) = %d, want %d", testCase.raw, got, testCase.want)
		}
	}
}

func TestCovAdminForumStatsExcerptRunes(t *testing.T) {
	short := "短字串"
	if got := excerptRunes(short, 80); got != short {
		t.Errorf("excerptRunes(%q) = %q", short, got)
	}
	long := strings.Repeat("字", 85)
	if got := excerptRunes(long, 80); got != strings.Repeat("字", 80)+"…" {
		t.Errorf("excerptRunes 長字串結果不正確：%q", got)
	}
}

/* ==========================================================================
   其餘錯誤分支（載入失敗、稽核寫入失敗、掃描失敗）
   ========================================================================== */

func TestCovAdminForumRecordAdminActionNilTx(t *testing.T) {
	h := newCovHarness(t)
	request := h.request(http.MethodGet, "/api/admin/log", "", "admin@example.com", true)
	if err := h.srv.recordAdminAction(request, nil, "post.create", "post", "1", "label"); err != nil {
		t.Fatalf("nil tx 應該直接回 nil，得到 %v", err)
	}
}

func TestCovAdminForumRecordPostIndexFailureAuditError(t *testing.T) {
	h := newCovHarness(t)
	h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
	request := h.request(http.MethodPost, "/api/admin/forum/posts/1/restore", "", "admin@example.com", true)
	h.srv.recordPostIndexFailure(request, 5, errors.New("es down"))
}

func TestCovAdminForumPostsListMediaTokenError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL", []string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT id, author_email, content, created_at, image_url", covAdminForumPostCols(),
		[]driverValue{int64(1), "a@example.com", "有圖", covAdminForumTime(), "photo.png", int64(0), int64(0), int64(0)},
	)
	h.srv.mediaRedis = nil

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covAdminForumWantBody(t, recorder, "unable to create media token")
}

func TestCovAdminForumPostsListCommentsError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE deleted_at IS NULL", []string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT id, author_email, content, created_at, image_url", covAdminForumPostCols(),
		[]driverValue{int64(1), "a@example.com", "內容", covAdminForumTime(), "", int64(0), int64(1), int64(0)},
	)
	h.queryError("FROM forum_post_comments WHERE post_id = ?", errors.New("db down"))

	recorder := h.do(h.srv.handleAdminForumPosts, http.MethodGet, "/api/admin/forum/posts", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportsScanError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("FROM forum_reports", covAdminForumReportCols(),
		[]driverValue{"not-an-id", "post", int64(3), "r@example.com", "spam", "PENDING", covAdminForumTime(), nil, "", "", ""},
	)
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodGet, "/api/admin/forum/reports", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportsCreateAuditError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
	h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
	recorder := h.do(h.srv.handleAdminForumReports, http.MethodPost, "/api/admin/forum/reports",
		`{"targetType":"post","targetId":3,"reporterEmail":"r@example.com","reason":"x"}`,
		"admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumReportLoadErrors(t *testing.T) {
	t.Run("DELETE 讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT status, reason FROM forum_reports WHERE id = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodDelete, "/api/admin/forum/reports/1", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("PUT 讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"count"}, []driverValue{int64(1)})
		h.queryError("SELECT target_type, target_id, reporter_email, reason, status", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
			`{"targetType":"post","targetId":3,"reporterEmail":"r@example.com","reason":"x"}`,
			"admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("PATCH 讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT reason, status FROM forum_reports WHERE id = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
			`{"status":"RESOLVED"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovAdminForumReportAuditErrors(t *testing.T) {
	t.Run("DELETE 稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status, reason FROM forum_reports WHERE id = ?",
			[]string{"status", "reason"}, []driverValue{"PENDING", "spam"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodDelete, "/api/admin/forum/reports/1", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("PUT 稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"count"}, []driverValue{int64(1)})
		h.rows("SELECT target_type, target_id, reporter_email, reason, status",
			[]string{"target_type", "target_id", "reporter_email", "reason", "status"},
			[]driverValue{"post", "3", "r@example.com", "舊", "PENDING"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodPut, "/api/admin/forum/reports/1",
			`{"targetType":"post","targetId":3,"reporterEmail":"r@example.com","reason":"新"}`,
			"admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("PATCH 稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT reason, status FROM forum_reports WHERE id = ?",
			[]string{"reason", "status"}, []driverValue{"原因", "PENDING"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminForumReport, http.MethodPatch, "/api/admin/forum/reports/1",
			`{"status":"RESOLVED"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovAdminForumPostLoadAndAuditErrors(t *testing.T) {
	t.Run("PUT 讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT content FROM forum_posts WHERE id = ? AND deleted_at IS NULL", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, "/api/admin/forum/posts/5",
			`{"content":"x"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("PUT 稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT content FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"content"}, []driverValue{"舊"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminForumPost, http.MethodPut, "/api/admin/forum/posts/5",
			`{"content":"x"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("DELETE 讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT author_email, content FROM forum_posts WHERE id = ? AND deleted_at IS NULL", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminForumPost, http.MethodDelete, "/api/admin/forum/posts/5", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("DELETE 稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email, content FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"author_email", "content"}, []driverValue{"a@example.com", "x"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminForumPost, http.MethodDelete, "/api/admin/forum/posts/5", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovAdminForumPostRestoreLoadAndAuditErrors(t *testing.T) {
	t.Run("讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT deleted_at, content FROM forum_posts WHERE id = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminPostRestore, http.MethodPost, "/api/admin/forum/posts/4/restore", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT deleted_at, content FROM forum_posts WHERE id = ?",
			[]string{"deleted_at", "content"}, []driverValue{covAdminForumTime(), "內容"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminPostRestore, http.MethodPost, "/api/admin/forum/posts/4/restore", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovAdminForumCommentsCreateAuditError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
	h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
	recorder := h.do(h.srv.handleAdminForumComments, http.MethodPost, "/api/admin/forum/comments",
		`{"postId":1,"content":"x"}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovAdminForumCommentLoadAndAuditErrors(t *testing.T) {
	t.Run("PUT 讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT content FROM forum_post_comments WHERE id = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
			`{"content":"x"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("PUT 稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT content FROM forum_post_comments WHERE id = ?",
			[]string{"content"}, []driverValue{"舊"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminForumComment, http.MethodPut, "/api/admin/forum/comments/9",
			`{"content":"x"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("DELETE 讀取失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT author_email, content, post_id FROM forum_post_comments WHERE id = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminForumComment, http.MethodDelete, "/api/admin/forum/comments/9", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
	t.Run("DELETE 稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email, content, post_id FROM forum_post_comments WHERE id = ?",
			[]string{"author_email", "content", "post_id"},
			[]driverValue{"c@example.com", "x", int64(3)})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminForumComment, http.MethodDelete, "/api/admin/forum/comments/9", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovAdminForumMonitorLimitStatsNilLimiter(t *testing.T) {
	h := newCovHarness(t)
	h.srv.writeRateLimiter = nil
	stats := h.srv.limitStats()
	if len(stats) != 2 {
		t.Fatalf("limitStats 筆數 = %d, want 2", len(stats))
	}
}

func TestCovAdminForumMonitorRedisDown(t *testing.T) {
	h := newCovHarness(t)
	if err := h.client.Close(); err != nil {
		t.Fatalf("關閉 Redis client: %v", err)
	}
	status := h.srv.probeRedis(context.Background())
	if status.State != "down" || status.Error == "" {
		t.Fatalf("probeRedis = %#v, want down 且有錯誤訊息", status)
	}
}

func TestCovAdminForumStatsSeriesScanError(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT day, new_users, new_posts, new_comments, new_likes",
		[]string{"day", "new_users", "new_posts", "new_comments", "new_likes"},
		[]driverValue{"not-a-time", int64(1), int64(2), int64(3), int64(4)},
	)
	recorder := h.do(h.srv.handleAdminStats, http.MethodGet, "/api/admin/stats", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}
