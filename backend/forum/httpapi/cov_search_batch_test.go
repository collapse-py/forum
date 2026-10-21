/*
搜尋、CSV 匯出、批次操作、公告與通知的覆蓋測試（httpapi/cov_search_batch_test.go）。

這個檔案只使用 cov_harness_test.go 的 in-memory SQL 替身，因此不需要 MySQL、
不需要網路，也不會開啟真正的 WebSocket。每一條規則都以「夠獨特的 SQL 片段」
登記，避免先登記的寬鬆規則吃掉後面的查詢。

涵蓋的檔案與重點：

  - search.go：公開／後臺搜尋（ES 未設定時走 MySQL LIKE）、參數驗證、
    依 ID 組裝貼文與後臺貼文、可見性校正、ES 未啟用時的維護函式。
  - csv_export.go：三份匯出的空結果／有結果／查詢失敗／掃描失敗、
    CSV 注入防護在真實 handler 上的輸出、匯出輔助函式。
  - batch_handlers.go：兩支批次端點的驗證、成功、跳過、交易內失敗。
  - announcement_handlers.go：公開公告、後臺列表／發佈／修改、置頂與分流。
  - forum_notification_handlers.go：清單、未讀數、標記已讀、通知寫入。
  - forum_notification_stream.go：只涵蓋升級前的驗證分支（不開 socket）。
*/

package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"forum/forum/es"
)

/* ==========================================================================
   輔助函式
   ========================================================================== */

// covSearchBatchDoUntrusted 造一個帶管理員 cookie 但來源不可信的請求。
// 直接用 h.request 再覆寫 Origin，避免另外複製一份 cookie 邏輯。
func covSearchBatchDoUntrusted(h *covHarness, handler http.HandlerFunc, method, path, body, email string, admin bool) *httptest.ResponseRecorder {
	h.t.Helper()
	request := h.request(method, path, body, email, admin)
	request.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

// covSearchBatchPostColumns 是 forumPostProjection 的欄位順序。
func covSearchBatchPostColumns() []string {
	return []string{
		"fp.id", "fp.author_email", "author_avatar", "fp.content", "fp.created_at",
		"fp.image_url", "fp.pinned", "fp.updated_at", "like_count", "comment_count",
		"liked_by_me", "bookmarked_by_me",
	}
}

// covSearchBatchPostRow 造一列可被 scanForumPostRow 讀取的貼文。
func covSearchBatchPostRow(id int64) []driverValue {
	return []driverValue{
		id, "author@example.com", nil, "貼文內容",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		"photo.png", int64(0), nil, int64(2), int64(3), int64(1), int64(0),
	}
}

// covSearchBatchWantContains 斷言 body 含有所有片段。
func covSearchBatchWantContains(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("回應 body 缺少 %q；body = %s", want, body)
		}
	}
}

/* ==========================================================================
   search.go：參數解析與 LIKE 跳脫
   ========================================================================== */

func TestCovSearchBatchParseSearchQuery(t *testing.T) {
	long := strings.Repeat("中", queryMaxRunes+1)
	cases := []struct {
		name   string
		target string
		want   searchQuery
		ok     bool
	}{
		{"沒有 q", "/api/forum/search", searchQuery{}, false},
		{"只有空白", "/api/forum/search?q=%20%20", searchQuery{}, false},
		{"超過長度上限", "/api/forum/search?q=" + url.QueryEscape(long), searchQuery{}, false},
		{"只有關鍵字", "/api/forum/search?q=hello", searchQuery{Query: "hello", Limit: pageSizeMax}, true},
		{"修剪前後空白", "/api/forum/search?q=%20hello%20", searchQuery{Query: "hello", Limit: pageSizeMax}, true},
		{"合法的 offset 與 limit", "/api/forum/search?q=hello&offset=10&limit=5", searchQuery{Query: "hello", Offset: 10, Limit: 5}, true},
		{"非數字 offset 與過大 limit 退回預設", "/api/forum/search?q=hello&offset=abc&limit=999", searchQuery{Query: "hello", Limit: pageSizeMax}, true},
		{"負數與零退回預設", "/api/forum/search?q=hello&offset=-3&limit=0", searchQuery{Query: "hello", Limit: pageSizeMax}, true},
		{"limit 剛好上限", "/api/forum/search?q=hello&limit=25", searchQuery{Query: "hello", Limit: pageSizeMax}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.target, nil)
			got, ok := parseSearchQuery(request)
			if ok != tc.ok {
				t.Fatalf("parseSearchQuery ok = %v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("parseSearchQuery = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestCovSearchBatchEscapeLikePattern(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"plain", "plain"},
		{"100%", `100\%`},
		{"a_b", `a\_b`},
		{`a\b`, `a\\b`},
		{`%_\`, `\%\_\\`},
	}
	for _, tc := range cases {
		if got := escapeLikePattern(tc.input); got != tc.want {
			t.Errorf("escapeLikePattern(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

/* ==========================================================================
   search.go：公開搜尋
   ========================================================================== */

func TestCovSearchBatchForumSearchMySQLFallback(t *testing.T) {
	t.Run("有結果時組出貼文", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"total"}, []driverValue{int64(1)})
		h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"id"}, []driverValue{int64(7)})
		h.rows("fp.id IN (", covSearchBatchPostColumns(), covSearchBatchPostRow(7))
		h.rows("SELECT nickname FROM forum_profiles WHERE author_email",
			[]string{"nickname"}, []driverValue{"Alice"})
		h.rows("FROM forum_user_tag_assignments a",
			[]string{"name"}, []driverValue{"vip"})

		recorder := h.do(h.srv.handleForumSearch, http.MethodGet,
			"/api/forum/search?q=hello&offset=2&limit=5", "", "viewer@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(),
			`"engine":"mysql"`, `"total":1`, `"query":"hello"`, `"author":"Alice"`, `"vip"`)
		if !h.sawQuery("fp.id IN (") {
			t.Error("有命中時應以 ID 集合載入貼文")
		}
	})

	t.Run("沒有命中時不載入貼文", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"total"}, []driverValue{int64(0)})

		recorder := h.do(h.srv.handleForumSearch, http.MethodGet,
			"/api/forum/search?q=100%25", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"items":[]`, `"total":0`, `"engine":"mysql"`)
		if h.sawQuery("fp.id IN (") {
			t.Error("零命中時不應查詢貼文內容")
		}
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumSearch, http.MethodPost,
			"/api/forum/search?q=hello", "", "", false), http.StatusMethodNotAllowed)
	})

	t.Run("缺少關鍵字", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumSearch, http.MethodGet,
			"/api/forum/search", "", "", false), http.StatusBadRequest)
	})

	t.Run("COUNT 查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			errors.New("count boom"))
		covWantStatus(t, h.do(h.srv.handleForumSearch, http.MethodGet,
			"/api/forum/search?q=hello", "", "", false), http.StatusInternalServerError)
	})

	t.Run("ID 查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"total"}, []driverValue{int64(0)})
		h.queryError("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			errors.New("ids boom"))
		covWantStatus(t, h.do(h.srv.handleForumSearch, http.MethodGet,
			"/api/forum/search?q=hello", "", "", false), http.StatusInternalServerError)
	})

	t.Run("載入貼文失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"total"}, []driverValue{int64(1)})
		h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"id"}, []driverValue{int64(7)})
		h.queryError("fp.id IN (", errors.New("load boom"))
		covWantStatus(t, h.do(h.srv.handleForumSearch, http.MethodGet,
			"/api/forum/search?q=hello", "", "", false), http.StatusInternalServerError)
	})

	t.Run("標籤查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"total"}, []driverValue{int64(1)})
		h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
			[]string{"id"}, []driverValue{int64(7)})
		h.rows("fp.id IN (", covSearchBatchPostColumns(), covSearchBatchPostRow(7))
		h.rows("SELECT nickname FROM forum_profiles WHERE author_email",
			[]string{"nickname"}, []driverValue{"Alice"})
		h.queryError("FROM forum_user_tag_assignments a", errors.New("tags boom"))
		covWantStatus(t, h.do(h.srv.handleForumSearch, http.MethodGet,
			"/api/forum/search?q=hello", "", "", false), http.StatusInternalServerError)
	})
}

/* ==========================================================================
   search.go：後臺搜尋
   ========================================================================== */

func TestCovSearchBatchAdminForumSearch(t *testing.T) {
	t.Run("有結果時含作者 email 與留言", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND (content LIKE",
			[]string{"total"}, []driverValue{int64(1)})
		h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND (content LIKE",
			[]string{"id"}, []driverValue{int64(11)})
		h.rows("SELECT id, author_email, content, created_at, image_url",
			[]string{"id", "author_email", "content", "created_at", "image_url",
				"like_count", "comment_count", "pinned"},
			[]driverValue{int64(11), "bob@example.com", "後臺內容",
				time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), "pic.png",
				int64(4), int64(2), int64(1)})
		h.rows("FROM forum_post_comments WHERE post_id = ?",
			[]string{"id", "post_id", "parent_id", "depth", "author_email", "content", "created_at"},
			[]driverValue{int64(3), int64(11), int64(0), int64(1),
				"carol@example.com", "留言", time.Date(2026, 1, 2, 3, 5, 5, 0, time.UTC)})

		recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet,
			"/api/admin/forum/search?q=bob", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(),
			`"engine":"mysql"`, `"authorEmail":"bob@example.com"`, `"pinned":true`, "留言")
	})

	t.Run("零命中", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND (content LIKE",
			[]string{"total"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleAdminForumSearch, http.MethodGet,
			"/api/admin/forum/search?q=bob", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"items":[]`, `"total":0`)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminForumSearch, http.MethodGet,
			"/api/admin/forum/search?q=bob", "", "", false), http.StatusUnauthorized)
	})

	t.Run("非管理員", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminForumSearch, http.MethodGet,
			"/api/admin/forum/search?q=bob", "", "user@example.com", false), http.StatusUnauthorized)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminForumSearch, http.MethodPost,
			"/api/admin/forum/search?q=bob", "", "admin@example.com", true), http.StatusMethodNotAllowed)
	})

	t.Run("缺少關鍵字", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminForumSearch, http.MethodGet,
			"/api/admin/forum/search", "", "admin@example.com", true), http.StatusBadRequest)
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND (content LIKE",
			errors.New("count boom"))
		covWantStatus(t, h.do(h.srv.handleAdminForumSearch, http.MethodGet,
			"/api/admin/forum/search?q=bob", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("載入後臺貼文失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND (content LIKE",
			[]string{"total"}, []driverValue{int64(1)})
		h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND (content LIKE",
			[]string{"id"}, []driverValue{int64(11)})
		h.queryError("SELECT id, author_email, content, created_at, image_url",
			errors.New("load boom"))
		covWantStatus(t, h.do(h.srv.handleAdminForumSearch, http.MethodGet,
			"/api/admin/forum/search?q=bob", "", "admin@example.com", true), http.StatusInternalServerError)
	})
}

/* ==========================================================================
   search.go：可見性校正與 ES 未啟用時的分支
   ========================================================================== */

func TestCovSearchBatchVisibleForumPostIDs(t *testing.T) {
	t.Run("空清單不查資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		visible, err := h.srv.visibleForumPostIDs(t.Context(), nil)
		if err != nil {
			t.Fatalf("visibleForumPostIDs: %v", err)
		}
		if len(visible) != 0 {
			t.Errorf("visible = %v, want 空集合", visible)
		}
		if h.sawQuery("deleted_at IS NULL AND id IN") {
			t.Error("空清單不應查詢資料庫")
		}
	})

	t.Run("回傳仍可見的 id", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("deleted_at IS NULL AND id IN",
			[]string{"id"}, []driverValue{int64(3)}, []driverValue{int64(5)})
		visible, err := h.srv.visibleForumPostIDs(t.Context(), []int64{3, 4, 5})
		if err != nil {
			t.Fatalf("visibleForumPostIDs: %v", err)
		}
		if !visible[3] || !visible[5] || visible[4] {
			t.Errorf("visible = %v, want 3 與 5 可見、4 不可見", visible)
		}
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("deleted_at IS NULL AND id IN", errors.New("visible boom"))
		if _, err := h.srv.visibleForumPostIDs(t.Context(), []int64{1}); err == nil {
			t.Error("查詢失敗應回傳錯誤")
		}
	})

	t.Run("掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("deleted_at IS NULL AND id IN",
			[]string{"id", "extra"}, []driverValue{int64(1), "x"})
		if _, err := h.srv.visibleForumPostIDs(t.Context(), []int64{1}); err == nil {
			t.Error("欄位數不符應回傳掃描錯誤")
		}
	})
}

func TestCovSearchBatchSearchDisabledHelpers(t *testing.T) {
	h := newCovHarness(t)
	if h.srv.esEnabled() {
		t.Fatal("測試 harness 不應啟用 Elasticsearch")
	}
	if h.srv.SearchEnabled() {
		t.Error("SearchEnabled 應與 esEnabled 一致")
	}
	if err := h.srv.indexForumPost(t.Context(), 1, "content", "a@b", time.Now()); err != nil {
		t.Errorf("ES 未啟用時 indexForumPost 應回 nil: %v", err)
	}
	if err := h.srv.reindexForumPostByID(t.Context(), 1); err != nil {
		t.Errorf("ES 未啟用時 reindexForumPostByID 應回 nil: %v", err)
	}
	if err := h.srv.unindexForumPost(t.Context(), 1); err != nil {
		t.Errorf("ES 未啟用時 unindexForumPost 應回 nil: %v", err)
	}
	if _, err := h.srv.RebuildSearchIndex(t.Context()); !errors.Is(err, es.ErrDisabled) {
		t.Errorf("RebuildSearchIndex err = %v, want ErrDisabled", err)
	}
	if _, err := h.srv.ReconcileSearchIndex(t.Context()); !errors.Is(err, es.ErrDisabled) {
		t.Errorf("ReconcileSearchIndex err = %v, want ErrDisabled", err)
	}
}

/* ==========================================================================
   csv_export.go：輔助函式
   ========================================================================== */

func TestCovSearchBatchCSVHelpers(t *testing.T) {
	if got := formatCSVTime(sql.NullTime{}); got != "" {
		t.Errorf("formatCSVTime(NULL) = %q, want 空字串", got)
	}
	valid := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	if got := formatCSVTime(sql.NullTime{Time: valid, Valid: true}); got != "2026-02-03T04:05:06Z" {
		t.Errorf("formatCSVTime(valid) = %q", got)
	}
	if got := exportedColumnCount("Users"); got != 6 {
		t.Errorf("exportedColumnCount(Users) = %d, want 6", got)
	}
	if got := exportedColumnCount("reports"); got != 9 {
		t.Errorf("exportedColumnCount(reports) = %d, want 9", got)
	}
	if got := exportedColumnCount("unknown"); got != 0 {
		t.Errorf("exportedColumnCount(unknown) = %d, want 0", got)
	}
	if got := itoa(42); got != "42" {
		t.Errorf("itoa(42) = %q", got)
	}
	if got := csvExportURLName("users", time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)); got != "users-20260930.csv" {
		t.Errorf("csvExportURLName = %q", got)
	}
	if got := sanitizeCSVField("=1+1"); got != "'=1+1" {
		t.Errorf("sanitizeCSVField 在 handler 路徑上失效: %q", got)
	}
}

/* ==========================================================================
   csv_export.go：三份匯出
   ========================================================================== */

func TestCovSearchBatchExportUsers(t *testing.T) {
	const query = "SELECT u.email, u.status"

	t.Run("有資料時輸出 BOM 與防護前綴", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"email", "status", "created_at", "updated_at", "posts", "comments"},
			[]driverValue{"=cmd@example.com", "ACTIVE", "2026-01-02T03:04:05Z",
				"2026-01-02T03:04:05Z", int64(3), int64(4)})
		recorder := h.do(h.srv.handleAdminExportUsers, http.MethodGet,
			"/api/admin/export/users.csv", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.HasPrefix(body, "\xEF\xBB\xBF") {
			t.Error("CSV 應以 UTF-8 BOM 開頭")
		}
		covSearchBatchWantContains(t, body, "email,status,created_at,updated_at,posts,comments",
			"'=cmd@example.com", ",3,4")
		if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/csv") {
			t.Errorf("Content-Type = %q, want text/csv", got)
		}
		if got := recorder.Header().Get("Content-Disposition"); !strings.Contains(got, "users-") {
			t.Errorf("Content-Disposition = %q, want users-*.csv", got)
		}
	})

	t.Run("空資料庫只輸出表頭", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminExportUsers, http.MethodGet,
			"/api/admin/export/users.csv", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		lines := strings.Split(strings.TrimSpace(recorder.Body.String()), "\n")
		if len(lines) != 1 {
			t.Errorf("空匯出應只有表頭一行, got %d 行: %q", len(lines), recorder.Body.String())
		}
	})

	t.Run("查詢失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(query, errors.New("users boom"))
		covWantStatus(t, h.do(h.srv.handleAdminExportUsers, http.MethodGet,
			"/api/admin/export/users.csv", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("掃描失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"email"}, []driverValue{"only-one-column"})
		covWantStatus(t, h.do(h.srv.handleAdminExportUsers, http.MethodGet,
			"/api/admin/export/users.csv", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminExportUsers, http.MethodPost,
			"/api/admin/export/users.csv", "", "admin@example.com", true), http.StatusMethodNotAllowed)
	})

	t.Run("非管理員", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminExportUsers, http.MethodGet,
			"/api/admin/export/users.csv", "", "user@example.com", false), http.StatusUnauthorized)
	})
}

func TestCovSearchBatchExportPosts(t *testing.T) {
	const query = "SELECT p.id, p.author_email"

	t.Run("有資料時輸出內容摘要", func(t *testing.T) {
		h := newCovHarness(t)
		long := strings.Repeat("中", csvExcerptRunes+10)
		h.rows(query, []string{"id", "author_email", "content", "created_at", "comments", "likes"},
			[]driverValue{int64(9), "=evil@example.com", long, "2026-01-02T03:04:05Z", int64(2), int64(5)})
		recorder := h.do(h.srv.handleAdminExportPosts, http.MethodGet,
			"/api/admin/export/posts.csv", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		covSearchBatchWantContains(t, body, "id,author_email,content,created_at,comments,likes",
			"'=evil@example.com", ",2,5")
		if strings.Contains(body, long) {
			t.Error("內容欄位應被截短成 csvExcerptRunes")
		}
	})

	t.Run("空資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminExportPosts, http.MethodGet,
			"/api/admin/export/posts.csv", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), "id,author_email,content")
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(query, errors.New("posts boom"))
		covWantStatus(t, h.do(h.srv.handleAdminExportPosts, http.MethodGet,
			"/api/admin/export/posts.csv", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"id"}, []driverValue{int64(1)})
		covWantStatus(t, h.do(h.srv.handleAdminExportPosts, http.MethodGet,
			"/api/admin/export/posts.csv", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminExportPosts, http.MethodPut,
			"/api/admin/export/posts.csv", "", "admin@example.com", true), http.StatusMethodNotAllowed)
	})
}

func TestCovSearchBatchExportReports(t *testing.T) {
	const query = "SELECT id, target_type, target_id, reporter_email, reason, status"

	t.Run("有資料時處理可空覆核時間", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query,
			[]string{"id", "target_type", "target_id", "reporter_email", "reason",
				"status", "created_at", "reviewed_at", "reviewed_by"},
			[]driverValue{"7", "post", "42", "=bad@example.com", "@SUM(A1:A9)",
				"PENDING", "2026-01-02T03:04:05Z", nil, ""},
			[]driverValue{"8", "comment", "43", "ok@example.com", "正常原因",
				"RESOLVED", "2026-01-03T03:04:05Z",
				time.Date(2026, 1, 4, 5, 6, 7, 0, time.UTC), "admin@example.com"})
		recorder := h.do(h.srv.handleAdminExportReports, http.MethodGet,
			"/api/admin/export/reports.csv", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(),
			"id,target_type,target_id,reporter_email,reason,status,created_at,reviewed_at,reviewed_by",
			"'=bad@example.com", "'@SUM(A1:A9)", "2026-01-04T05:06:07Z")
	})

	t.Run("空資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminExportReports, http.MethodGet,
			"/api/admin/export/reports.csv", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), "reporter_email")
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(query, errors.New("reports boom"))
		covWantStatus(t, h.do(h.srv.handleAdminExportReports, http.MethodGet,
			"/api/admin/export/reports.csv", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"id"}, []driverValue{"1"})
		covWantStatus(t, h.do(h.srv.handleAdminExportReports, http.MethodGet,
			"/api/admin/export/reports.csv", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminExportReports, http.MethodGet,
			"/api/admin/export/reports.csv", "", "", false), http.StatusUnauthorized)
	})
}

/* ==========================================================================
   batch_handlers.go：批次標籤
   ========================================================================== */

// covSearchBatchPrimeTagRules 登記批次標籤成功路徑所需的查詢。
func covSearchBatchPrimeTagRules(h *covHarness) {
	h.rows("SELECT 1 FROM forum_users WHERE email = ?",
		[]string{"1"}, []driverValue{int64(1)})
	h.rows("FROM forum_user_tag_assignments a JOIN forum_user_tags t",
		[]string{"name"}, []driverValue{"old"})
	h.rows("SELECT id, name FROM forum_user_tags WHERE id IN",
		[]string{"id", "name"},
		[]driverValue{int64(1), "alpha"}, []driverValue{int64(2), "beta"})
}

func TestCovSearchBatchBatchTagsValidation(t *testing.T) {
	post := func(h *covHarness, body string) *httptest.ResponseRecorder {
		return h.do(h.srv.handleAdminBatchTags, http.MethodPost, "/api/admin/batch/tags",
			body, "admin@example.com", true)
	}

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminBatchTags, http.MethodPost,
			"/api/admin/batch/tags", `{"emails":["a@b"]}`, "", false), http.StatusUnauthorized)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminBatchTags, http.MethodGet,
			"/api/admin/batch/tags", "", "admin@example.com", true), http.StatusMethodNotAllowed)
	})

	t.Run("來源不可信", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, covSearchBatchDoUntrusted(h, h.srv.handleAdminBatchTags, http.MethodPost,
			"/api/admin/batch/tags", `{"emails":["a@b"]}`, "admin@example.com", true), http.StatusForbidden)
	})

	t.Run("JSON 格式錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, post(h, "{"), http.StatusBadRequest)
	})

	t.Run("沒有選擇帳號", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, post(h, `{"emails":[],"tagIds":[]}`), http.StatusBadRequest)
	})

	t.Run("email 格式錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, post(h, `{"emails":["not-an-email"]}`), http.StatusBadRequest)
	})

	t.Run("email 過長", func(t *testing.T) {
		h := newCovHarness(t)
		long := strings.Repeat("a", 256) + "@example.com"
		covWantStatus(t, post(h, `{"emails":["`+long+`"]}`), http.StatusBadRequest)
	})

	t.Run("email 數量超過上限", func(t *testing.T) {
		h := newCovHarness(t)
		emails := make([]string, 0, batchMaxEmails+1)
		for i := 0; i <= batchMaxEmails; i++ {
			emails = append(emails, `"a@example.com"`)
		}
		body := `{"emails":[` + strings.Join(emails, ",") + `]}`
		covWantStatus(t, post(h, body), http.StatusBadRequest)
	})

	t.Run("標籤數量超過上限", func(t *testing.T) {
		h := newCovHarness(t)
		ids := make([]string, 0, batchMaxTags+1)
		for i := 0; i <= batchMaxTags; i++ {
			ids = append(ids, "1")
		}
		body := `{"emails":["a@example.com"],"tagIds":[` + strings.Join(ids, ",") + `]}`
		covWantStatus(t, post(h, body), http.StatusBadRequest)
	})

	t.Run("標籤 id 非正數", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[0]}`), http.StatusBadRequest)
	})

	t.Run("標籤 id 重複", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[1,1]}`), http.StatusBadRequest)
	})

	t.Run("標籤不存在", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT id, name FROM forum_user_tags WHERE id IN",
			[]string{"id", "name"}, []driverValue{int64(1), "alpha"})
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[1,2]}`), http.StatusBadRequest)
	})

	t.Run("標籤查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT id, name FROM forum_user_tags WHERE id IN", errors.New("tags boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[1]}`), http.StatusBadRequest)
	})

	t.Run("沒有資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.db = nil
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[]}`), http.StatusInternalServerError)
	})
}

func TestCovSearchBatchBatchTags(t *testing.T) {
	post := func(h *covHarness, body string) *httptest.ResponseRecorder {
		return h.do(h.srv.handleAdminBatchTags, http.MethodPost, "/api/admin/batch/tags",
			body, "admin@example.com", true)
	}

	t.Run("套用標籤並記稽核", func(t *testing.T) {
		h := newCovHarness(t)
		covSearchBatchPrimeTagRules(h)
		recorder := post(h, `{"emails":["a@example.com"],"tagIds":[1,2]}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(),
			`"ok":true`, `"requested":1`, `"updated":1`)
		for _, query := range []string{
			"DELETE FROM forum_user_tag_assignments",
			"INSERT INTO forum_user_tag_assignments",
			"INSERT INTO forum_admin_actions",
		} {
			if !h.sawQuery(query) {
				t.Errorf("成功路徑應執行 %q", query)
			}
		}
	})

	t.Run("標籤集相同時不寫入", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT 1 FROM forum_users WHERE email = ?",
			[]string{"1"}, []driverValue{int64(1)})
		h.rows("FROM forum_user_tag_assignments a JOIN forum_user_tags t",
			[]string{"name"}, []driverValue{"beta"}, []driverValue{"alpha"})
		h.rows("SELECT id, name FROM forum_user_tags WHERE id IN",
			[]string{"id", "name"},
			[]driverValue{int64(1), "alpha"}, []driverValue{int64(2), "beta"})
		recorder := post(h, `{"emails":["a@example.com"],"tagIds":[1,2]}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"unchanged":1`)
		if h.sawQuery("DELETE FROM forum_user_tag_assignments") {
			t.Error("標籤集未變時不應刪除既有綁定")
		}
	})

	t.Run("帳號不存在時列入 skipped", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := post(h, `{"emails":["ghost@example.com"],"tagIds":[]}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), "帳號不存在", `"skipped"`)
	})

	t.Run("讀取帳號失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT 1 FROM forum_users WHERE email = ?", errors.New("users boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[]}`), http.StatusInternalServerError)
	})

	t.Run("讀取既有標籤失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT 1 FROM forum_users WHERE email = ?",
			[]string{"1"}, []driverValue{int64(1)})
		h.queryError("FROM forum_user_tag_assignments a JOIN forum_user_tags t",
			errors.New("before boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[]}`), http.StatusInternalServerError)
	})

	t.Run("刪除舊綁定失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covSearchBatchPrimeTagRules(h)
		h.execError("DELETE FROM forum_user_tag_assignments", errors.New("delete boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[1,2]}`), http.StatusInternalServerError)
	})

	t.Run("寫入新綁定失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covSearchBatchPrimeTagRules(h)
		h.execError("INSERT INTO forum_user_tag_assignments", errors.New("insert boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[1,2]}`), http.StatusInternalServerError)
	})

	t.Run("稽核寫入失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covSearchBatchPrimeTagRules(h)
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"tagIds":[1,2]}`), http.StatusInternalServerError)
	})
}

/* ==========================================================================
   batch_handlers.go：批次狀態
   ========================================================================== */

func TestCovSearchBatchBatchStatus(t *testing.T) {
	post := func(h *covHarness, body string) *httptest.ResponseRecorder {
		return h.do(h.srv.handleAdminBatchStatus, http.MethodPost, "/api/admin/batch/status",
			body, "admin@example.com", true)
	}

	t.Run("成功停權", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email = ?",
			[]string{"status"}, []driverValue{"ACTIVE"})
		recorder := post(h, `{"emails":["a@example.com"],"status":"suspended"}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"updated":1`)
		if !h.sawQuery("UPDATE forum_users SET status") {
			t.Error("狀態改變時應更新 forum_users")
		}
		if !h.sawQuery("INSERT INTO forum_admin_actions") {
			t.Error("狀態改變時應寫入稽核")
		}
	})

	t.Run("狀態相同時不寫入", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email = ?",
			[]string{"status"}, []driverValue{"SUSPENDED"})
		recorder := post(h, `{"emails":["a@example.com"],"status":" SUSPENDED "}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"unchanged":1`)
		if h.sawQuery("UPDATE forum_users SET status") {
			t.Error("狀態未變時不應更新")
		}
	})

	t.Run("帳號不存在時列入 skipped", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := post(h, `{"emails":["ghost@example.com"],"status":"SUSPENDED"}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), "帳號不存在")
	})

	t.Run("無效狀態", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"status":"DELETED"}`), http.StatusBadRequest)
	})

	t.Run("JSON 格式錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, post(h, "["), http.StatusBadRequest)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminBatchStatus, http.MethodPost,
			"/api/admin/batch/status", `{"emails":["a@b"],"status":"ACTIVE"}`, "", false),
			http.StatusUnauthorized)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminBatchStatus, http.MethodDelete,
			"/api/admin/batch/status", "", "admin@example.com", true), http.StatusMethodNotAllowed)
	})

	t.Run("來源不可信", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, covSearchBatchDoUntrusted(h, h.srv.handleAdminBatchStatus, http.MethodPost,
			"/api/admin/batch/status", `{"emails":["a@b"],"status":"ACTIVE"}`,
			"admin@example.com", true), http.StatusForbidden)
	})

	t.Run("讀取狀態失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT status FROM forum_users WHERE email = ?", errors.New("status boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"status":"SUSPENDED"}`),
			http.StatusInternalServerError)
	})

	t.Run("更新失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email = ?",
			[]string{"status"}, []driverValue{"ACTIVE"})
		h.execError("UPDATE forum_users SET status", errors.New("update boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"status":"SUSPENDED"}`),
			http.StatusInternalServerError)
	})

	t.Run("稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email = ?",
			[]string{"status"}, []driverValue{"ACTIVE"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit boom"))
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"status":"SUSPENDED"}`),
			http.StatusInternalServerError)
	})

	t.Run("沒有資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.db = nil
		covWantStatus(t, post(h, `{"emails":["a@example.com"],"status":"SUSPENDED"}`),
			http.StatusInternalServerError)
	})
}

/* ==========================================================================
   announcement_handlers.go：公開公告
   ========================================================================== */

func TestCovSearchBatchAnnouncementPublic(t *testing.T) {
	const query = "expires_at IS NULL OR expires_at > ?"

	t.Run("沒有公告回 null", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumAnnouncement, http.MethodGet,
			"/api/forum/announcement", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"announcement":null`)
	})

	t.Run("永不過期的公告", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"body", "created_at", "expires_at"},
			[]driverValue{"維護公告", "2026-01-02T03:04:05Z", nil})
		recorder := h.do(h.srv.handleForumAnnouncement, http.MethodGet,
			"/api/forum/announcement", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(),
			`"body":"維護公告"`, `"expiresAt":""`)
	})

	t.Run("有到期時間的公告", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"body", "created_at", "expires_at"},
			[]driverValue{"活動", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
				time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)})
		recorder := h.do(h.srv.handleForumAnnouncement, http.MethodGet,
			"/api/forum/announcement", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"expiresAt":"2026-02-03T04:05:06Z"`)
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(query, errors.New("announcement boom"))
		covWantStatus(t, h.do(h.srv.handleForumAnnouncement, http.MethodGet,
			"/api/forum/announcement", "", "", false), http.StatusInternalServerError)
	})

	t.Run("掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"body"}, []driverValue{"只有一欄"})
		covWantStatus(t, h.do(h.srv.handleForumAnnouncement, http.MethodGet,
			"/api/forum/announcement", "", "", false), http.StatusInternalServerError)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumAnnouncement, http.MethodPost,
			"/api/forum/announcement", "", "", false), http.StatusMethodNotAllowed)
	})
}

/* ==========================================================================
   announcement_handlers.go：後臺公告
   ========================================================================== */

func TestCovSearchBatchAnnouncementsAdmin(t *testing.T) {
	t.Run("列出公告含 effective", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT id, body, active, created_by",
			[]string{"id", "body", "active", "created_by", "created_at",
				"updated_by", "updated_at", "expires_at"},
			[]driverValue{int64(1), "現行公告", int64(1), "admin@example.com",
				"2026-01-02T03:04:05Z", "", "", nil},
			[]driverValue{int64(2), "過期公告", int64(1), "admin@example.com",
				"2026-01-01T00:00:00Z", "admin@example.com", "2026-01-01T01:00:00Z",
				time.Now().Add(-time.Hour)})
		recorder := h.do(h.srv.handleAdminAnnouncements, http.MethodGet,
			"/api/admin/announcements", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(),
			`"items"`, `"effective":true`, `"effective":false`, "現行公告")
	})

	t.Run("列出失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT id, body, active, created_by", errors.New("list boom"))
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodGet,
			"/api/admin/announcements", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("列表掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT id, body, active, created_by", []string{"id"}, []driverValue{int64(1)})
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodGet,
			"/api/admin/announcements", "", "admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("發佈成功", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements",
			`{"body":"新公告","active":true,"hoursUntilExpiry":24}`,
			"admin@example.com", true)
		covWantStatus(t, recorder, http.StatusCreated)
		covSearchBatchWantContains(t, recorder.Body.String(), `"ok":true`, `"id":1`)
		if !h.sawQuery("INSERT INTO forum_announcements") {
			t.Error("發佈應寫入 forum_announcements")
		}
		if !h.sawQuery("UPDATE forum_announcements SET active = 0") {
			t.Error("發佈啟用中的公告應先停用舊的")
		}
	})

	t.Run("發佈但不啟用時不動舊公告", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", `{"body":"稍後生效","active":false}`,
			"admin@example.com", true)
		covWantStatus(t, recorder, http.StatusCreated)
		if h.sawQuery("UPDATE forum_announcements SET active = 0") {
			t.Error("active=false 時不應停用線上公告")
		}
	})

	t.Run("發佈內容空白", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", `{"body":"   "}`, "admin@example.com", true),
			http.StatusBadRequest)
	})

	t.Run("發佈 JSON 錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", "}", "admin@example.com", true), http.StatusBadRequest)
	})

	t.Run("發佈來源不可信", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, covSearchBatchDoUntrusted(h, h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", `{"body":"x"}`, "admin@example.com", true),
			http.StatusForbidden)
	})

	t.Run("發佈時停用舊公告失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("UPDATE forum_announcements SET active = 0", errors.New("deactivate boom"))
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", `{"body":"x","active":true}`,
			"admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("發佈寫入失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_announcements (body, active", errors.New("insert boom"))
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", `{"body":"x","active":false}`,
			"admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("發佈稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit boom"))
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", `{"body":"x","active":false}`,
			"admin@example.com", true), http.StatusInternalServerError)
	})

	t.Run("發佈沒有資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.db = nil
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodPost,
			"/api/admin/announcements", `{"body":"x"}`, "admin@example.com", true),
			http.StatusInternalServerError)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodDelete,
			"/api/admin/announcements", "", "admin@example.com", true), http.StatusMethodNotAllowed)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncements, http.MethodGet,
			"/api/admin/announcements", "", "", false), http.StatusUnauthorized)
	})
}

func TestCovSearchBatchAnnouncementPatch(t *testing.T) {
	const selectQuery = "SELECT body, active, expires_at FROM forum_announcements WHERE id = ?"

	patch := func(h *covHarness, path, body string) *httptest.ResponseRecorder {
		return h.do(h.srv.handleAdminAnnouncement, http.MethodPatch, path, body,
			"admin@example.com", true)
	}

	t.Run("修改成功", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"body", "active", "expires_at"},
			[]driverValue{"舊內容", int64(1), nil})
		recorder := patch(h, "/api/admin/announcements/7",
			`{"body":"新內容","active":false,"hoursUntilExpiry":0}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"ok":true`)
		if !h.sawQuery("UPDATE forum_announcements") {
			t.Error("修改應更新 forum_announcements")
		}
	})

	t.Run("修改為啟用時先關閉其他公告", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"body", "active", "expires_at"},
			[]driverValue{"舊內容", int64(0), time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)})
		recorder := patch(h, "/api/admin/announcements/7",
			`{"body":"新內容","active":true,"hoursUntilExpiry":48}`)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery("WHERE active = 1 AND id <> ?") {
			t.Error("改為啟用時應停用其他公告")
		}
	})

	t.Run("公告不存在", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, patch(h, "/api/admin/announcements/404",
			`{"body":"x","active":false}`), http.StatusNotFound)
	})

	t.Run("id 格式錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, patch(h, "/api/admin/announcements/abc",
			`{"body":"x"}`), http.StatusBadRequest)
	})

	t.Run("id 為 0", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, patch(h, "/api/admin/announcements/0", `{"body":"x"}`), http.StatusBadRequest)
	})

	t.Run("JSON 錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, patch(h, "/api/admin/announcements/7", "{"), http.StatusBadRequest)
	})

	t.Run("內容空白", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, patch(h, "/api/admin/announcements/7", `{"body":"  "}`), http.StatusBadRequest)
	})

	t.Run("讀取舊內容失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(selectQuery, errors.New("select boom"))
		covWantStatus(t, patch(h, "/api/admin/announcements/7",
			`{"body":"x","active":false}`), http.StatusInternalServerError)
	})

	t.Run("更新失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"body", "active", "expires_at"},
			[]driverValue{"舊內容", int64(1), nil})
		h.execError("UPDATE forum_announcements", errors.New("update boom"))
		covWantStatus(t, patch(h, "/api/admin/announcements/7",
			`{"body":"x","active":false}`), http.StatusInternalServerError)
	})

	t.Run("稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"body", "active", "expires_at"},
			[]driverValue{"舊內容", int64(1), nil})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit boom"))
		covWantStatus(t, patch(h, "/api/admin/announcements/7",
			`{"body":"x","active":false}`), http.StatusInternalServerError)
	})

	t.Run("沒有資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.db = nil
		covWantStatus(t, patch(h, "/api/admin/announcements/7",
			`{"body":"x","active":false}`), http.StatusInternalServerError)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncement, http.MethodGet,
			"/api/admin/announcements/7", "", "admin@example.com", true), http.StatusMethodNotAllowed)
	})

	t.Run("來源不可信", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, covSearchBatchDoUntrusted(h, h.srv.handleAdminAnnouncement, http.MethodPatch,
			"/api/admin/announcements/7", `{"body":"x"}`,
			"admin@example.com", true), http.StatusForbidden)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminAnnouncement, http.MethodPatch,
			"/api/admin/announcements/7", `{"body":"x"}`, "", false), http.StatusUnauthorized)
	})
}

/* ==========================================================================
   announcement_handlers.go：置頂與分流
   ========================================================================== */

func TestCovSearchBatchAdminPostPin(t *testing.T) {
	const selectQuery = "SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL"

	pin := func(h *covHarness, path, body string) *httptest.ResponseRecorder {
		return h.do(h.srv.handleAdminPostPin, http.MethodPost, path, body,
			"admin@example.com", true)
	}

	t.Run("置頂成功", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"pinned"}, []driverValue{int64(0)})
		recorder := pin(h, "/api/admin/forum/posts/3/pin", `{"pinned":true}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"pinned":true`)
		if !h.sawQuery("UPDATE forum_posts SET pinned") {
			t.Error("狀態改變時應更新 pinned")
		}
		if !h.sawQuery("INSERT INTO forum_admin_actions") {
			t.Error("置頂應寫入稽核")
		}
	})

	t.Run("狀態未變時直接成功", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"pinned"}, []driverValue{int64(1)})
		recorder := pin(h, "/api/admin/forum/posts/3/pin", `{"pinned":true}`)
		covWantStatus(t, recorder, http.StatusOK)
		if h.sawQuery("UPDATE forum_posts SET pinned") {
			t.Error("狀態未變時不應更新")
		}
	})

	t.Run("取消置頂", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"pinned"}, []driverValue{int64(1)})
		recorder := pin(h, "/api/admin/forum/posts/3/pin", `{"pinned":false}`)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"pinned":false`)
	})

	t.Run("貼文不存在", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, pin(h, "/api/admin/forum/posts/404/pin", `{"pinned":true}`),
			http.StatusNotFound)
	})

	t.Run("id 格式錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, pin(h, "/api/admin/forum/posts/abc/pin", `{"pinned":true}`),
			http.StatusBadRequest)
	})

	t.Run("JSON 錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, pin(h, "/api/admin/forum/posts/3/pin", "{"), http.StatusBadRequest)
	})

	t.Run("讀取狀態失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(selectQuery, errors.New("pin select boom"))
		covWantStatus(t, pin(h, "/api/admin/forum/posts/3/pin", `{"pinned":true}`),
			http.StatusInternalServerError)
	})

	t.Run("更新失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"pinned"}, []driverValue{int64(0)})
		h.execError("UPDATE forum_posts SET pinned", errors.New("pin update boom"))
		covWantStatus(t, pin(h, "/api/admin/forum/posts/3/pin", `{"pinned":true}`),
			http.StatusInternalServerError)
	})

	t.Run("稽核失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(selectQuery, []string{"pinned"}, []driverValue{int64(0)})
		h.execError("INSERT INTO forum_admin_actions", errors.New("pin audit boom"))
		covWantStatus(t, pin(h, "/api/admin/forum/posts/3/pin", `{"pinned":true}`),
			http.StatusInternalServerError)
	})

	t.Run("沒有資料庫", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.db = nil
		covWantStatus(t, pin(h, "/api/admin/forum/posts/3/pin", `{"pinned":true}`),
			http.StatusInternalServerError)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminPostPin, http.MethodGet,
			"/api/admin/forum/posts/3/pin", "", "admin@example.com", true),
			http.StatusMethodNotAllowed)
	})

	t.Run("來源不可信", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, covSearchBatchDoUntrusted(h, h.srv.handleAdminPostPin, http.MethodPost,
			"/api/admin/forum/posts/3/pin", `{"pinned":true}`,
			"admin@example.com", true), http.StatusForbidden)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleAdminPostPin, http.MethodPost,
			"/api/admin/forum/posts/3/pin", `{"pinned":true}`, "", false), http.StatusUnauthorized)
	})
}

func TestCovSearchBatchAdminPostDispatch(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"pinned"}, []driverValue{int64(0)})
	recorder := h.do(h.srv.handleAdminPostOrPin, http.MethodPost,
		"/api/admin/forum/posts/3/pin", `{"pinned":true}`, "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)

	h2 := newCovHarness(t)
	covWantStatus(t, h2.do(h2.srv.handleAdminPostOrPin, http.MethodGet,
		"/api/admin/forum/posts/3/restore", "", "admin@example.com", true),
		http.StatusMethodNotAllowed)

	h3 := newCovHarness(t)
	covWantStatus(t, h3.do(h3.srv.handleAdminPostOrPin, http.MethodGet,
		"/api/admin/forum/posts/3", "", "admin@example.com", true),
		http.StatusMethodNotAllowed)
}

/* ==========================================================================
   forum_notification_handlers.go：清單、未讀數、標記已讀
   ========================================================================== */

func TestCovSearchBatchNotifications(t *testing.T) {
	const listQuery = "LEFT JOIN forum_profiles p ON p.author_email = n.actor_email"
	const countQuery = "SELECT COUNT(*) FROM forum_notifications WHERE recipient_email = ? AND read_at IS NULL"

	t.Run("列出通知並附未讀數", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(listQuery,
			[]string{"n.id", "n.kind", "n.actor_email", "nickname", "avatar_url",
				"n.post_id", "n.comment_id", "n.read_at", "n.created_at"},
			[]driverValue{int64(2), "comment", "bob@example.com", "Bob", "",
				int64(10), int64(20), nil, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
			[]driverValue{int64(1), "follow", "carol@example.com", "", "avatar.png",
				int64(0), int64(0), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
		h.rows(countQuery, []string{"count"}, []driverValue{int64(3)})
		recorder := h.do(h.srv.handleForumNotifications, http.MethodGet,
			"/api/forum/notifications", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(),
			`"hasMore":false`, `"unreadCount":3`, `"actor":"Bob"`, `"kind":"follow"`)
	})

	t.Run("清單查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(listQuery, errors.New("list boom"))
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodGet,
			"/api/forum/notifications", "", "me@example.com", false), http.StatusInternalServerError)
	})

	t.Run("清單掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(listQuery, []string{"n.id"}, []driverValue{int64(1)})
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodGet,
			"/api/forum/notifications", "", "me@example.com", false), http.StatusInternalServerError)
	})

	t.Run("未讀數查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(countQuery, errors.New("count boom"))
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodGet,
			"/api/forum/notifications", "", "me@example.com", false), http.StatusInternalServerError)
	})

	t.Run("limit 非法", func(t *testing.T) {
		for _, target := range []string{
			"/api/forum/notifications?limit=abc",
			"/api/forum/notifications?limit=0",
			"/api/forum/notifications?limit=51",
		} {
			h := newCovHarness(t)
			covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodGet,
				target, "", "me@example.com", false), http.StatusBadRequest)
		}
	})

	t.Run("offset 非法", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodGet,
			"/api/forum/notifications?offset=-1", "", "me@example.com", false), http.StatusBadRequest)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodGet,
			"/api/forum/notifications", "", "", false), http.StatusUnauthorized)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodPut,
			"/api/forum/notifications", "", "me@example.com", false), http.StatusMethodNotAllowed)
	})

	t.Run("標記單則已讀", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(countQuery, []string{"count"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleForumNotifications, http.MethodPost,
			"/api/forum/notifications", `{"id":5}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"unreadCount":0`)
		if !h.sawQuery("WHERE id = ? AND recipient_email = ?") {
			t.Error("標記單則應把 id 與收件者放進 WHERE")
		}
	})

	t.Run("標記全部已讀", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(countQuery, []string{"count"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleForumNotifications, http.MethodPost,
			"/api/forum/notifications", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery("UPDATE forum_notifications SET read_at = ? WHERE recipient_email = ?") {
			t.Error("空 body 應標記全部未讀為已讀")
		}
	})

	t.Run("標記 JSON 錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodPost,
			"/api/forum/notifications", "{", "me@example.com", false), http.StatusBadRequest)
	})

	t.Run("標記來源不可信", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, covSearchBatchDoUntrusted(h, h.srv.handleForumNotifications, http.MethodPost,
			"/api/forum/notifications", `{"id":5}`, "me@example.com", false),
			http.StatusForbidden)
	})

	t.Run("標記更新失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("UPDATE forum_notifications SET read_at", errors.New("mark boom"))
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodPost,
			"/api/forum/notifications", `{"id":5}`, "me@example.com", false),
			http.StatusInternalServerError)
	})

	t.Run("標記後讀取未讀數失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(countQuery, errors.New("count boom"))
		covWantStatus(t, h.do(h.srv.handleForumNotifications, http.MethodPost,
			"/api/forum/notifications", `{"id":5}`, "me@example.com", false),
			http.StatusInternalServerError)
	})
}

func TestCovSearchBatchNotificationUnreadCount(t *testing.T) {
	const countQuery = "SELECT COUNT(*) FROM forum_notifications WHERE recipient_email = ? AND read_at IS NULL"

	t.Run("成功", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(countQuery, []string{"count"}, []driverValue{int64(4)})
		recorder := h.do(h.srv.handleForumNotificationUnreadCount, http.MethodGet,
			"/api/forum/notifications/unread-count", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		covSearchBatchWantContains(t, recorder.Body.String(), `"unreadCount":4`)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotificationUnreadCount, http.MethodGet,
			"/api/forum/notifications/unread-count", "", "", false), http.StatusUnauthorized)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotificationUnreadCount, http.MethodPost,
			"/api/forum/notifications/unread-count", "", "me@example.com", false),
			http.StatusMethodNotAllowed)
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(countQuery, errors.New("count boom"))
		covWantStatus(t, h.do(h.srv.handleForumNotificationUnreadCount, http.MethodGet,
			"/api/forum/notifications/unread-count", "", "me@example.com", false),
			http.StatusInternalServerError)
	})
}

/* ==========================================================================
   forum_notification_stream.go：升級前的驗證分支
   ========================================================================== */

func TestCovSearchBatchNotificationStreamValidation(t *testing.T) {
	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotificationStream, http.MethodGet,
			"/api/forum/notifications/ws", "", "", false), http.StatusUnauthorized)
	})

	t.Run("方法錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, h.do(h.srv.handleForumNotificationStream, http.MethodPost,
			"/api/forum/notifications/ws", "", "me@example.com", false),
			http.StatusMethodNotAllowed)
	})

	t.Run("來源不可信", func(t *testing.T) {
		h := newCovHarness(t)
		covWantStatus(t, covSearchBatchDoUntrusted(h, h.srv.handleForumNotificationStream,
			http.MethodGet, "/api/forum/notifications/ws", "", "me@example.com", false),
			http.StatusForbidden)
	})
}

/* ==========================================================================
   forum_notification_handlers.go：通知寫入與收件者決定
   ========================================================================== */

func TestCovSearchBatchNotificationCreate(t *testing.T) {
	t.Run("缺收件者或觸發者時跳過", func(t *testing.T) {
		h := newCovHarness(t)
		cases := [][2]string{
			{"", "actor@example.com"},
			{"recipient@example.com", ""},
			{"same@example.com", "same@example.com"},
			{"   ", "actor@example.com"},
		}
		for _, pair := range cases {
			id, err := h.srv.createForumNotification(t.Context(), pair[0], pair[1],
				forumNotificationComment, 1, 2)
			if err != nil || id != 0 {
				t.Errorf("createForumNotification(%q, %q) = (%d, %v), want (0, nil)",
					pair[0], pair[1], id, err)
			}
		}
		if h.sawQuery("INSERT INTO forum_notifications") {
			t.Error("跳過的情況不應寫入通知")
		}
	})

	t.Run("寫入成功", func(t *testing.T) {
		h := newCovHarness(t)
		id, err := h.srv.createForumNotification(t.Context(), "recipient@example.com",
			"actor@example.com", forumNotificationReply, 10, 20)
		if err != nil {
			t.Fatalf("createForumNotification: %v", err)
		}
		if id != 1 {
			t.Errorf("通知 id = %d, want 1（替身的 LastInsertId）", id)
		}
	})

	t.Run("寫入失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_notifications", errors.New("notify boom"))
		if _, err := h.srv.createForumNotification(t.Context(), "recipient@example.com",
			"actor@example.com", forumNotificationFollow, 0, 0); err == nil {
			t.Error("寫入失敗應回傳錯誤")
		}
	})

	t.Run("回覆通知父留言作者", func(t *testing.T) {
		h := newCovHarness(t)
		request := h.request(http.MethodPost, "/api/forum/posts/1/comments", "", "actor@example.com", false)
		h.srv.notifyForumCommentCreated(request, 1, 2, "actor@example.com", "parent@example.com")
		if !h.sawQuery("INSERT INTO forum_notifications") {
			t.Error("回覆應寫入 kind=reply 的通知")
		}
	})

	t.Run("頂層留言查不到貼文作者", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT author_email FROM forum_posts WHERE id = ?", errors.New("author boom"))
		request := h.request(http.MethodPost, "/api/forum/posts/1/comments", "", "actor@example.com", false)
		h.srv.notifyForumCommentCreated(request, 1, 2, "actor@example.com", "")
		if h.sawQuery("INSERT INTO forum_notifications") {
			t.Error("查不到收件者時不應寫入通知")
		}
	})

	t.Run("頂層留言通知貼文作者", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email FROM forum_posts WHERE id = ?",
			[]string{"author_email"}, []driverValue{"poster@example.com"})
		request := h.request(http.MethodPost, "/api/forum/posts/1/comments", "", "actor@example.com", false)
		h.srv.notifyForumCommentCreated(request, 1, 2, "actor@example.com", "")
		if !h.sawQuery("INSERT INTO forum_notifications") {
			t.Error("頂層留言應通知貼文作者")
		}
	})

	t.Run("自己回覆自己不寫通知", func(t *testing.T) {
		h := newCovHarness(t)
		request := h.request(http.MethodPost, "/api/forum/posts/1/comments", "", "same@example.com", false)
		h.srv.notifyForumCommentCreated(request, 1, 2, "same@example.com", "same@example.com")
		if h.sawQuery("INSERT INTO forum_notifications") {
			t.Error("收件者等於觸發者時不應寫入通知")
		}
	})
}
