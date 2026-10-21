/*
留言、按讚與檢舉端點的覆蓋測試（httpapi/cov_forum_comments_test.go）。

這份檔案用 cov_harness_test.go 的資料庫替身，把 forum_handlers.go 裡「留言／
按讚／檢舉」這一組 handler 的成功與失敗路徑都走過一次。既有測試的分工是：
comment_reply_test.go 釘住純函式與 SQL 形狀、*_mysql_test.go 需要真實 MySQL
（未設定 DSN 時 skip），因此「handler 真的走到哪一條分支」在一般 CI 裡幾乎
沒有被執行。這裡補的就是那一段。

替身的限制與對應作法：
  - covResult 的 RowsAffected 恆為 1，因此「影響 0 列 → 404」的分支改用本檔
    自帶的 zero-driver（covCommentsZeroDB）替換 s.db；它只回 0 列，其餘形狀
    與 harness 相同。
  - 交易（按讚）的 Commit 不會失敗，因此那個分支仍無法覆蓋；BeginTx 失敗則
    以關閉 *sql.DB 的方式觸發（database/sql 會直接回 ErrDBClosed）。
  - 同理，留言 INSERT 的 LastInsertId 失敗與 rows.Err() 中斷也無法用這個替身
    觸發（兩者在真實驅動層才算得出來），因此 handleForumComments 不是 100%。
  - 留言列表的頭像 token 走真實的 miniredis；要讓簽發失敗只需關掉 client
    —— 那條路徑在正式環境是「Redis 掛掉，留言仍可讀、只是沒有頭像」。

所有測試名稱以 TestCovForumComments 開頭，helper 以 covComments 開頭。
*/

package httpapi

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

/* ==========================================================================
   測試替身與小工具
   ========================================================================== */

var covCommentsZeroOnce sync.Once

// covCommentsZeroDriver 讓 Exec 回報「影響 0 列」。harness 的 covResult 固定
// 回 1，因此 update/delete 的 404 分支只能靠這個替身走到。
type covCommentsZeroDriver struct{}

func (covCommentsZeroDriver) Open(string) (driver.Conn, error) { return covCommentsZeroConn{}, nil }

type covCommentsZeroConn struct{}

func (covCommentsZeroConn) Prepare(string) (driver.Stmt, error) { return covStmt{}, nil }
func (covCommentsZeroConn) Close() error                        { return nil }
func (covCommentsZeroConn) Begin() (driver.Tx, error)           { return covTx{}, nil }
func (covCommentsZeroConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return covTx{}, nil
}

func (covCommentsZeroConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return covCommentsZeroResult{}, nil
}

func (covCommentsZeroConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &covRows{}, nil
}

type covCommentsZeroResult struct{}

func (covCommentsZeroResult) LastInsertId() (int64, error) { return 0, nil }
func (covCommentsZeroResult) RowsAffected() (int64, error) { return 0, nil }

// covCommentsZeroDB 建立一個「所有 Exec 都回 0 列」的 *sql.DB，供 404 分支使用。
func covCommentsZeroDB(t *testing.T) *sql.DB {
	t.Helper()
	covCommentsZeroOnce.Do(func() {
		sql.Register("forum-cov-comments-zero", covCommentsZeroDriver{})
	})
	db, err := sql.Open("forum-cov-comments-zero", "cov-comments-zero")
	if err != nil {
		t.Fatalf("sql.Open(cov-comments-zero): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// covCommentsPostVisible 登記「這篇貼文存在且未刪除」。
func covCommentsPostVisible(h *covHarness) {
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(1)})
}

// covCommentsUntrusted 直接呼叫 handler，但帶一個不在白名單內的 Origin。
func covCommentsUntrusted(t *testing.T, h *covHarness, handler http.HandlerFunc,
	method, path, body, email string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Origin", "https://evil.example")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if email != "" {
		request.AddCookie(h.login(email, false))
	}
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

// covCommentsWantBody 斷言回應 body 含有所有片段。
func covCommentsWantBody(t *testing.T, recorder *httptest.ResponseRecorder, wants ...string) {
	t.Helper()

	body := recorder.Body.String()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("body 缺少 %q：%s", want, body)
		}
	}
}

// covCommentsWantNoBody 斷言回應 body 不含任何一個片段。
func covCommentsWantNoBody(t *testing.T, recorder *httptest.ResponseRecorder, unwanted ...string) {
	t.Helper()

	body := recorder.Body.String()
	for _, value := range unwanted {
		if strings.Contains(body, value) {
			t.Errorf("body 不該含有 %q：%s", value, body)
		}
	}
}

// covCommentsCommentColumns 是留言列表 SELECT 的掃描欄位順序，必須與
// handleForumComments 的 rows.Scan 完全一致。
var covCommentsCommentColumns = []string{
	"id", "author_email", "author_avatar", "content", "created_at", "updated_at", "parent_id", "depth",
}

/* ==========================================================================
   純函式與小型查詢 helper
   ========================================================================== */

func TestCovForumCommentsParentDepthAllowsReply(t *testing.T) {
	cases := []struct {
		depth int
		want  bool
	}{
		{0, true},
		{1, true},
		{2, true},
		{3, false},
		{4, false},
	}
	for _, tc := range cases {
		if got := forumParentDepthAllowsReply(tc.depth); got != tc.want {
			t.Errorf("forumParentDepthAllowsReply(%d) = %v, want %v", tc.depth, got, tc.want)
		}
	}
}

func TestCovForumCommentsParentCommentInfo(t *testing.T) {
	t.Run("找到父留言", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT depth, author_email FROM forum_post_comments WHERE id = ? AND post_id = ?",
			[]string{"depth", "author_email"},
			[]driverValue{int64(2), "parent@example.com"},
		)
		depth, author, err := h.srv.forumParentCommentInfo(context.Background(), 1, 9)
		if err != nil {
			t.Fatalf("forumParentCommentInfo: %v", err)
		}
		if depth != 2 || author != "parent@example.com" {
			t.Fatalf("depth=%d author=%q", depth, author)
		}
	})

	t.Run("父留言不存在", func(t *testing.T) {
		h := newCovHarness(t)
		if _, _, err := h.srv.forumParentCommentInfo(context.Background(), 1, 9); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})

	t.Run("資料庫錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT depth, author_email FROM forum_post_comments", errors.New("db down"))
		if _, _, err := h.srv.forumParentCommentInfo(context.Background(), 1, 9); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestCovForumCommentsPostAuthorEmail(t *testing.T) {
	t.Run("找到作者", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"author_email"}, []driverValue{"owner@example.com"})
		email, err := h.srv.forumPostAuthorEmail(context.Background(), 1)
		if err != nil {
			t.Fatalf("forumPostAuthorEmail: %v", err)
		}
		if email != "owner@example.com" {
			t.Fatalf("email = %q", email)
		}
	})

	t.Run("貼文不存在", func(t *testing.T) {
		h := newCovHarness(t)
		if _, err := h.srv.forumPostAuthorEmail(context.Background(), 1); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})

	t.Run("資料庫錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT author_email FROM forum_posts", errors.New("db down"))
		if _, err := h.srv.forumPostAuthorEmail(context.Background(), 1); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestCovForumCommentsCommentIDFromPath(t *testing.T) {
	cases := []struct {
		path string
		id   int64
		ok   bool
	}{
		{"/api/forum/posts/1/comments/2", 2, true},
		{"/api/forum/posts/1/comments/2/", 2, true},
		{"/api/forum/posts/1/comments//2", 2, true},
		{"/api/forum/posts/1/comments/0", 0, false},
		{"/api/forum/posts/1/comments/-3", 0, false},
		{"/api/forum/posts/1/comments/abc", 0, false},
		{"/api/forum/posts/1/comments/2/abc", 0, false},
		{"/api/forum/posts/1/comments", 0, false},
		{"/api/forum/posts/1/comments/", 0, false},
		{"/api/forum/posts/1/report", 0, false},
	}
	for _, tc := range cases {
		id, ok := forumCommentIDFromPath(tc.path)
		if id != tc.id || ok != tc.ok {
			t.Errorf("forumCommentIDFromPath(%q) = (%d, %v), want (%d, %v)",
				tc.path, id, ok, tc.id, tc.ok)
		}
	}
}

func TestCovForumCommentsReportTargetID(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		kind     string
		wantID   int64
		wantFail bool
	}{
		{"留言", "/api/forum/posts/1/comments/5/report", "comment", 5, false},
		{"留言缺 marker", "/api/forum/posts/1/report", "comment", 0, true},
		{"留言非數字", "/api/forum/posts/1/comments/abc/report", "comment", 0, true},
		{"留言為零", "/api/forum/posts/1/comments/0/report", "comment", 0, true},
		{"留言為負", "/api/forum/posts/1/comments/-1/report", "comment", 0, true},
		{"貼文", "/api/forum/posts/9/report", "post", 9, false},
		{"貼文路徑錯誤", "/api/forum/posts/abc/report", "post", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := forumReportTargetID(tc.path, tc.kind)
			if tc.wantFail {
				if err == nil {
					t.Fatalf("want error, got id=%d", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("forumReportTargetID: %v", err)
			}
			if id != tc.wantID {
				t.Fatalf("id = %d, want %d", id, tc.wantID)
			}
		})
	}
}

/* ==========================================================================
   GET /api/forum/posts/{id}/comments
   ========================================================================== */

func TestCovForumCommentsGetRejectsInvalidPostID(t *testing.T) {
	// 第二個路徑剝掉 /comments 之後只剩前綴，forumPostIDFromPath 的「空字串」
	// 分支因此被執行（那是 /api/forum/posts/ 打歪的形狀）。
	for _, path := range []string{
		"/api/forum/posts/abc/comments",
		"/api/forum/posts//comments",
	} {
		t.Run(path, func(t *testing.T) {
			h := newCovHarness(t)
			recorder := h.do(h.srv.handleForumComments, http.MethodGet, path, "", "", false)
			covWantStatus(t, recorder, http.StatusBadRequest)
			covCommentsWantBody(t, recorder, "invalid post id")
		})
	}
}

func TestCovForumCommentsGetPostCheckError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts", errors.New("db down"))
	recorder := h.do(h.srv.handleForumComments, http.MethodGet, "/api/forum/posts/7/comments", "", "", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to load forum post")
}

func TestCovForumCommentsGetPostNotFound(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(0)})
	recorder := h.do(h.srv.handleForumComments, http.MethodGet, "/api/forum/posts/7/comments", "", "", false)
	covWantStatus(t, recorder, http.StatusNotFound)
	covCommentsWantBody(t, recorder, "post not found")
}

func TestCovForumCommentsGetEmpty(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	// 合法的 limit / offset 值走一次：預設值以外的那兩段解析也需要被執行。
	recorder := h.do(h.srv.handleForumComments, http.MethodGet,
		"/api/forum/posts/7/comments?limit=12&offset=3", "", "", false)
	covWantStatus(t, recorder, http.StatusOK)
	covCommentsWantBody(t, recorder, `"items":[]`, `"hasMore":false`)
	if !h.sawQuery("FROM forum_post_comments fc") {
		t.Error("沒有執行留言列表查詢")
	}
}

func TestCovForumCommentsGetRejectsInvalidPaging(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  string
	}{
		{"limit 為零", "?limit=0", "invalid comment limit"},
		{"limit 過大", "?limit=13", "invalid comment limit"},
		{"limit 非數字", "?limit=abc", "invalid comment limit"},
		{"offset 為負", "?offset=-1", "invalid comment offset"},
		{"offset 非數字", "?offset=abc", "invalid comment offset"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCovHarness(t)
			covCommentsPostVisible(h)
			recorder := h.do(h.srv.handleForumComments, http.MethodGet,
				"/api/forum/posts/7/comments"+tc.query, "", "", false)
			covWantStatus(t, recorder, http.StatusBadRequest)
			covCommentsWantBody(t, recorder, tc.want)
		})
	}
}

func TestCovForumCommentsGetListQueryError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.queryError("FROM forum_post_comments fc", errors.New("db down"))
	recorder := h.do(h.srv.handleForumComments, http.MethodGet, "/api/forum/posts/7/comments", "", "", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to load comments")
}

func TestCovForumCommentsGetListsComments(t *testing.T) {
	h := newCovHarness(t)
	// 頭像網址由 FILES_SERVER_PUBLIC_URL 現組；測試設定檔沒有填，補一個才能
	// 斷言「庫存檔名 → 公開網址」這一段真的有發生。
	h.srv.cfg.FilesServerPublicURL = "https://files.test"
	covCommentsPostVisible(h)
	h.rows("SELECT nickname FROM forum_profiles WHERE author_email = ?",
		[]string{"nickname"}, []driverValue{"小明"})
	h.rows("FROM forum_post_comments fc", covCommentsCommentColumns,
		[]driverValue{int64(5), "a@example.com", "avatar.png", "第一則", time.Now(), nil, int64(0), int64(1)},
		[]driverValue{int64(6), "b@example.com", nil, "第二則", time.Now(), time.Now(), int64(5), int64(2)},
	)

	recorder := h.do(h.srv.handleForumComments, http.MethodGet, "/api/forum/posts/7/comments", "", "", false)
	covWantStatus(t, recorder, http.StatusOK)
	covCommentsWantBody(t, recorder,
		`"id":5`, `"author":"小明"`, `"content":"第一則"`, `"depth":1`,
		`"parentId":5`, `"depth":2`, `"edited":true`, `"hasMore":false`,
		`"authorAvatar":"https://files.test/files/avatar.png?token=`)
	// 第二則的 updated_at 非 NULL，代表編輯過；第一則是 NULL，不該出現 edited。
	if !strings.Contains(recorder.Body.String(), `"id":6`) {
		t.Errorf("body = %s", recorder.Body.String())
	}
}

func TestCovForumCommentsGetScanFailure(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	// 欄位數與 Scan 目的地不一致：替身仍會回一列，讓 handler 走到掃描錯誤。
	h.rows("FROM forum_post_comments fc", []string{"id"},
		[]driverValue{int64(5)})
	recorder := h.do(h.srv.handleForumComments, http.MethodGet, "/api/forum/posts/7/comments", "", "", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to read comments")
}

func TestCovForumCommentsGetAvatarTokenFailure(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("FROM forum_post_comments fc", covCommentsCommentColumns,
		[]driverValue{int64(5), "a@example.com", "avatar.png", "有頭像", time.Now(), nil, int64(0), int64(1)},
	)
	// 關掉 Redis：留言仍然讀得到，只是簽不出頭像 token，因此不該回 500，
	// 也不該送出一個沒有 token 的（必然破圖的）頭像網址。
	if err := h.client.Close(); err != nil {
		t.Fatalf("關閉 redis client: %v", err)
	}
	recorder := h.do(h.srv.handleForumComments, http.MethodGet, "/api/forum/posts/7/comments", "", "", false)
	covWantStatus(t, recorder, http.StatusOK)
	covCommentsWantBody(t, recorder, `"content":"有頭像"`)
	covCommentsWantNoBody(t, recorder, `"authorAvatar"`)
}

func TestCovForumCommentsGetMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumComments, http.MethodPatch, "/api/forum/posts/7/comments", "", "", false)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

/* ==========================================================================
   POST /api/forum/posts/{id}/comments
   ========================================================================== */

func TestCovForumCommentsPostRejectsUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covCommentsUntrusted(t, h, h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com")
	covWantStatus(t, recorder, http.StatusForbidden)
	covCommentsWantBody(t, recorder, "invalid origin")
}

func TestCovForumCommentsPostRejectsInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "invalid request")
}

func TestCovForumCommentsPostRejectsBadContent(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"空白", `{"content":"   "}`, "留言內容不可為空"},
		{"超長", `{"content":"` + strings.Repeat("x", 2001) + `"}`, "留言最多 2000 字"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCovHarness(t)
			recorder := h.do(h.srv.handleForumComments, http.MethodPost,
				"/api/forum/posts/7/comments", tc.body, "me@example.com", false)
			covWantStatus(t, recorder, http.StatusBadRequest)
			covCommentsWantBody(t, recorder, tc.want)
		})
	}
}

func TestCovForumCommentsPostRejectsNegativeParent(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi","parentId":-1}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "invalid parent comment")
	// 形狀檢查必須發生在任何資料庫查詢之前。
	if h.sawQuery("SELECT COUNT(*) FROM forum_posts") {
		t.Error("負數 parentId 不該先查文章存在性")
	}
}

func TestCovForumCommentsPostPostCheckError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts", errors.New("db down"))
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to load forum post")
}

func TestCovForumCommentsPostPostNotFound(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(0)})
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
	covCommentsWantBody(t, recorder, "post not found")
}

func TestCovForumCommentsPostParentNotFound(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	// 沒有登記父留言查詢 → 替身回空結果集 → sql.ErrNoRows。
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi","parentId":99}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
	covCommentsWantBody(t, recorder, "parent comment not found")
}

func TestCovForumCommentsPostParentLookupError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.queryError("SELECT depth, author_email FROM forum_post_comments", errors.New("db down"))
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi","parentId":99}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to load parent comment")
}

func TestCovForumCommentsPostParentTooDeep(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("SELECT depth, author_email FROM forum_post_comments WHERE id = ? AND post_id = ?",
		[]string{"depth", "author_email"},
		[]driverValue{int64(forumCommentMaxDepth), "parent@example.com"},
	)
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi","parentId":99}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "回覆已達最深層級")
	if h.sawQuery("INSERT INTO forum_post_comments") {
		t.Error("超過層級上限時不該寫入留言")
	}
}

func TestCovForumCommentsPostInsertError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.execError("INSERT INTO forum_post_comments", errors.New("db down"))
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to create comment")
}

func TestCovForumCommentsPostCreatesTopLevel(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("SELECT author_email FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"author_email"}, []driverValue{"owner@example.com"})
	h.rows("SELECT nickname FROM forum_profiles WHERE author_email = ?",
		[]string{"nickname"}, []driverValue{"小明"})

	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"  第一則留言  "}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusCreated)
	covCommentsWantBody(t, recorder,
		`"ok":true`, `"author":"小明"`, `"content":"第一則留言"`, `"depth":1`,
		`"authorKey":"`+publicForumKey("me@example.com")+`"`)
	// 第一層留言省略 parentId。
	covCommentsWantNoBody(t, recorder, `"parentId"`)
	if !h.sawQuery("INSERT INTO forum_post_comments") {
		t.Error("沒有寫入留言")
	}
	// 頂層留言要通知貼文作者；通知寫入是 best-effort，但這條路徑應該走到。
	if !h.sawQuery("INSERT INTO forum_notifications") {
		t.Error("頂層留言沒有建立通知")
	}
}

func TestCovForumCommentsPostCreatesReply(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("SELECT depth, author_email FROM forum_post_comments WHERE id = ? AND post_id = ?",
		[]string{"depth", "author_email"},
		[]driverValue{int64(2), "parent@example.com"},
	)

	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"回覆","parentId":99}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusCreated)
	covCommentsWantBody(t, recorder, `"depth":3`, `"parentId":99`)
	if !h.sawQuery("INSERT INTO forum_notifications") {
		t.Error("回覆沒有建立通知")
	}
	// 回覆的收件者是父留言作者，因此不需要再查貼文作者。
	if h.sawQuery("SELECT author_email FROM forum_posts") {
		t.Error("回覆不該查貼文作者")
	}
}

func TestCovForumCommentsPostAvatarLookupError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.queryError("SELECT avatar_url FROM forum_profiles", errors.New("db down"))
	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to read forum profile")
}

func TestCovForumCommentsPostNotificationFailureIsBestEffort(t *testing.T) {
	t.Run("查不到貼文作者", func(t *testing.T) {
		h := newCovHarness(t)
		covCommentsPostVisible(h)
		h.queryError("SELECT author_email FROM forum_posts", errors.New("db down"))
		recorder := h.do(h.srv.handleForumComments, http.MethodPost,
			"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
		// 通知收件者查不到不該讓已寫入的留言變成 500。
		covWantStatus(t, recorder, http.StatusCreated)
	})

	t.Run("通知寫入失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covCommentsPostVisible(h)
		h.rows("SELECT author_email FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
			[]string{"author_email"}, []driverValue{"owner@example.com"})
		h.execError("INSERT INTO forum_notifications", errors.New("db down"))
		recorder := h.do(h.srv.handleForumComments, http.MethodPost,
			"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusCreated)
	})
}

func TestCovForumCommentsPostIncludesAvatar(t *testing.T) {
	h := newCovHarness(t)
	h.srv.cfg.FilesServerPublicURL = "https://files.test"
	covCommentsPostVisible(h)
	h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
		[]string{"avatar_url"}, []driverValue{"photo.png"})

	recorder := h.do(h.srv.handleForumComments, http.MethodPost,
		"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusCreated)
	covCommentsWantBody(t, recorder, `"authorAvatar":"https://files.test/files/photo.png?token=`)
}

func TestCovForumCommentsPostMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumComments, http.MethodPatch,
		"/api/forum/posts/7/comments", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

/* ==========================================================================
   PUT /api/forum/posts/{id}/comments/{cid}
   ========================================================================== */

func TestCovForumCommentsUpdateRequiresLogin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsUpdateHandler(h), http.MethodPut,
		"/api/forum/posts/7/comments/9", `{"content":"hi"}`, "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovForumCommentsUpdateRejectsUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covCommentsUntrusted(t, h, covCommentsUpdateHandler(h), http.MethodPut,
		"/api/forum/posts/7/comments/9", `{"content":"hi"}`, "me@example.com")
	covWantStatus(t, recorder, http.StatusForbidden)
}

// covCommentsUpdateHandler 把帶 commentID 的 method 包成 http.HandlerFunc，
// 讓 untrusted-origin 的共用小工具也能套用。
func covCommentsUpdateHandler(h *covHarness) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.srv.handleForumCommentUpdate(w, r, 9)
	}
}

func TestCovForumCommentsUpdateRejectsBadPath(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsUpdateHandler(h), http.MethodPut,
		"/api/forum/posts/abc/comments/9", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "invalid post id")
}

func TestCovForumCommentsUpdateRejectsInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsUpdateHandler(h), http.MethodPut,
		"/api/forum/posts/7/comments/9", `{"content":`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "invalid request")
}

func TestCovForumCommentsUpdateRejectsBadContent(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"空白", `{"content":"  "}`, "留言內容不可為空"},
		{"超長", `{"content":"` + strings.Repeat("x", 2001) + `"}`, "留言最多 2000 字"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCovHarness(t)
			recorder := h.do(covCommentsUpdateHandler(h), http.MethodPut,
				"/api/forum/posts/7/comments/9", tc.body, "me@example.com", false)
			covWantStatus(t, recorder, http.StatusBadRequest)
			covCommentsWantBody(t, recorder, tc.want)
		})
	}
}

func TestCovForumCommentsUpdateDBError(t *testing.T) {
	h := newCovHarness(t)
	h.execError("UPDATE forum_post_comments SET content", errors.New("db down"))
	recorder := h.do(covCommentsUpdateHandler(h), http.MethodPut,
		"/api/forum/posts/7/comments/9", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to update comment")
}

func TestCovForumCommentsUpdateSuccess(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsUpdateHandler(h), http.MethodPut,
		"/api/forum/posts/7/comments/9", `{"content":"  改過的內容  "}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	covCommentsWantBody(t, recorder, `"ok":true`)
	if !h.sawQuery("UPDATE forum_post_comments SET content") {
		t.Error("沒有執行留言更新")
	}
}

func TestCovForumCommentsUpdateNotFound(t *testing.T) {
	h := newCovHarness(t)
	// 影響 0 列（不存在／不是你的／母文已刪）：回應必須是 404，且不洩漏是哪一種。
	h.srv.db = covCommentsZeroDB(t)
	recorder := h.do(covCommentsUpdateHandler(h), http.MethodPut,
		"/api/forum/posts/7/comments/9", `{"content":"hi"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
}

/* ==========================================================================
   DELETE /api/forum/posts/{id}/comments/{cid} 與子樹刪除 helper
   ========================================================================== */

func TestCovForumCommentsDeleteRequiresLogin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsDeleteHandler(h), http.MethodDelete,
		"/api/forum/posts/7/comments/9", "", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovForumCommentsDeleteRejectsUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covCommentsUntrusted(t, h, covCommentsDeleteHandler(h), http.MethodDelete,
		"/api/forum/posts/7/comments/9", "", "me@example.com")
	covWantStatus(t, recorder, http.StatusForbidden)
}

// covCommentsDeleteHandler 把帶 commentID 的 method 包成 http.HandlerFunc。
func covCommentsDeleteHandler(h *covHarness) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.srv.handleForumCommentDelete(w, r, 9)
	}
}

func TestCovForumCommentsDeleteRejectsBadPath(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsDeleteHandler(h), http.MethodDelete,
		"/api/forum/posts/abc/comments/9", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "invalid post id")
}

func TestCovForumCommentsDeleteDBError(t *testing.T) {
	h := newCovHarness(t)
	h.execError("DELETE c FROM forum_post_comments", errors.New("db down"))
	recorder := h.do(covCommentsDeleteHandler(h), http.MethodDelete,
		"/api/forum/posts/7/comments/9", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to delete forum comment")
}

func TestCovForumCommentsDeleteSuccess(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsDeleteHandler(h), http.MethodDelete,
		"/api/forum/posts/7/comments/9", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	covCommentsWantBody(t, recorder, `"ok":true`, `"removed":1`)
	if !h.sawQuery("DELETE c FROM forum_post_comments") {
		t.Error("沒有執行子樹刪除")
	}
}

func TestCovForumCommentsDeleteNotFound(t *testing.T) {
	h := newCovHarness(t)
	h.srv.db = covCommentsZeroDB(t)
	recorder := h.do(covCommentsDeleteHandler(h), http.MethodDelete,
		"/api/forum/posts/7/comments/9", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovForumCommentsSubtreeDeleteHelper(t *testing.T) {
	t.Run("成功", func(t *testing.T) {
		h := newCovHarness(t)
		affected, err := forumCommentSubtreeDelete(context.Background(), h.sqlDB, 7, 9,
			"JOIN forum_post_comments root ON root.id = ?", int64(9))
		if err != nil {
			t.Fatalf("forumCommentSubtreeDelete: %v", err)
		}
		if affected != 1 {
			t.Fatalf("affected = %d, want 1", affected)
		}
		if !h.sawQuery("DELETE c FROM forum_post_comments c") {
			t.Error("沒有執行子樹刪除 SQL")
		}
	})

	t.Run("資料庫錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("DELETE c FROM forum_post_comments", errors.New("db down"))
		if _, err := forumCommentSubtreeDelete(context.Background(), h.sqlDB, 7, 9,
			"JOIN forum_post_comments root ON root.id = ?", int64(9)); err == nil {
			t.Fatal("want error")
		}
	})
}

/* ==========================================================================
   POST /api/forum/posts/{id}/like
   ========================================================================== */

func TestCovForumCommentsLikeMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumPostLike, http.MethodGet, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovForumCommentsLikeRequiresLogin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovForumCommentsLikeRejectsUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covCommentsUntrusted(t, h, h.srv.handleForumPostLike, http.MethodPost,
		"/api/forum/posts/7/like", "", "me@example.com")
	covWantStatus(t, recorder, http.StatusForbidden)
}

func TestCovForumCommentsLikeRejectsBadPath(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/abc/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "invalid post id")
}

func TestCovForumCommentsLikeBeginTransactionError(t *testing.T) {
	h := newCovHarness(t)
	// 先建立請求（含 session cookie），再關閉連線池，讓 BeginTx 直接失敗。
	request := h.request(http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	if err := h.sqlDB.Close(); err != nil {
		t.Fatalf("關閉 db: %v", err)
	}
	recorder := httptest.NewRecorder()
	h.srv.handleForumPostLike(recorder, request)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to begin like transaction")
}

func TestCovForumCommentsLikePostLookupError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts", errors.New("db down"))
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to load forum post")
}

func TestCovForumCommentsLikePostNotFound(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(0)})
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
	covCommentsWantBody(t, recorder, "post not found")
}

func TestCovForumCommentsLikeStatusLookupError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.queryError("FROM forum_post_likes WHERE post_id = ? AND author_email = ?", errors.New("db down"))
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to check like status")
}

func TestCovForumCommentsLikeAdds(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	// 比對順序是登記順序：先登記帶 author_email 的「我是否按讚」，再登記
	// 只帶 post_id 的總數。總數查詢不含 author_email，因此不會誤中第一條。
	h.rows("FROM forum_post_likes WHERE post_id = ? AND author_email = ?",
		[]string{"count"}, []driverValue{int64(0)})
	h.rows("SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ?",
		[]string{"count"}, []driverValue{int64(3)})

	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	covCommentsWantBody(t, recorder, `"liked":true`, `"count":3`)
	if !h.sawQuery("INSERT INTO forum_post_likes") {
		t.Error("未按讚時應新增一列")
	}
}

func TestCovForumCommentsLikeAddError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("FROM forum_post_likes WHERE post_id = ? AND author_email = ?",
		[]string{"count"}, []driverValue{int64(0)})
	h.execError("INSERT INTO forum_post_likes", errors.New("db down"))
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to add like")
}

func TestCovForumCommentsLikeRemoves(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("FROM forum_post_likes WHERE post_id = ? AND author_email = ?",
		[]string{"count"}, []driverValue{int64(1)})
	h.rows("SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ?",
		[]string{"count"}, []driverValue{int64(0)})

	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	covCommentsWantBody(t, recorder, `"liked":false`, `"count":0`)
	if !h.sawQuery("DELETE FROM forum_post_likes") {
		t.Error("已按讚時應刪除那一列")
	}
}

func TestCovForumCommentsLikeRemoveError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("FROM forum_post_likes WHERE post_id = ? AND author_email = ?",
		[]string{"count"}, []driverValue{int64(1)})
	h.execError("DELETE FROM forum_post_likes", errors.New("db down"))
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to remove like")
}

func TestCovForumCommentsLikeCountError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.rows("FROM forum_post_likes WHERE post_id = ? AND author_email = ?",
		[]string{"count"}, []driverValue{int64(0)})
	// 總數查詢的錯誤規則必須排在「我是否按讚」之後，否則會先被它吃掉。
	h.queryError("SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ?", errors.New("db down"))
	recorder := h.do(h.srv.handleForumPostLike, http.MethodPost, "/api/forum/posts/7/like", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to count likes")
}

/* ==========================================================================
   POST /api/forum/posts/{id}/report 與 .../comments/{cid}/report
   ========================================================================== */

func TestCovForumCommentsReportMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodGet,
		"/api/forum/posts/7/report", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

// covCommentsReportHandler 把 targetType 綁進 handler，方便共用。
func covCommentsReportHandler(h *covHarness, targetType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.srv.handleForumReport(w, r, targetType)
	}
}

func TestCovForumCommentsReportRejectsUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	recorder := covCommentsUntrusted(t, h, covCommentsReportHandler(h, "post"), http.MethodPost,
		"/api/forum/posts/7/report", `{"reason":"spam"}`, "me@example.com")
	covWantStatus(t, recorder, http.StatusForbidden)
}

func TestCovForumCommentsReportRejectsInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodPost,
		"/api/forum/posts/7/report", `{"reason":`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	covCommentsWantBody(t, recorder, "invalid request")
}

func TestCovForumCommentsReportRejectsBadReason(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"空字串", `{"reason":""}`},
		{"純空白", `{"reason":"   "}`},
		{"超長", `{"reason":"` + strings.Repeat("x", 501) + `"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCovHarness(t)
			recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodPost,
				"/api/forum/posts/7/report", tc.body, "me@example.com", false)
			covWantStatus(t, recorder, http.StatusBadRequest)
			covCommentsWantBody(t, recorder, "檢舉原因需為 1 至 500 字")
		})
	}
}

func TestCovForumCommentsReportRejectsBadTarget(t *testing.T) {
	cases := []struct {
		name   string
		kind   string
		path   string
		reason string
	}{
		{"留言缺 marker", "comment", "/api/forum/posts/7/report", `{"reason":"spam"}`},
		{"留言非數字", "comment", "/api/forum/posts/7/comments/abc/report", `{"reason":"spam"}`},
		{"貼文非數字", "post", "/api/forum/posts/abc/report", `{"reason":"spam"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCovHarness(t)
			recorder := h.do(covCommentsReportHandler(h, tc.kind), http.MethodPost,
				tc.path, tc.reason, "me@example.com", false)
			covWantStatus(t, recorder, http.StatusBadRequest)
			covCommentsWantBody(t, recorder, "invalid report target")
		})
	}
}

func TestCovForumCommentsReportTargetLookupError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT COUNT(*) FROM forum_posts", errors.New("db down"))
	recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodPost,
		"/api/forum/posts/7/report", `{"reason":"spam"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to load report target")
}

func TestCovForumCommentsReportTargetNotFound(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
		[]string{"count"}, []driverValue{int64(0)})
	recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodPost,
		"/api/forum/posts/7/report", `{"reason":"spam"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
	covCommentsWantBody(t, recorder, "content not found")
}

func TestCovForumCommentsReportDuplicate(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	// 1062 = ER_DUP_ENTRY：同一個人重複檢舉同一內容，由唯一鍵擋下。
	h.execError("INSERT INTO forum_reports",
		&mysql.MySQLError{Number: 1062, Message: "Duplicate entry"})
	recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodPost,
		"/api/forum/posts/7/report", `{"reason":"spam"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusConflict)
	covCommentsWantBody(t, recorder, "你已檢舉過此內容")
}

func TestCovForumCommentsReportInsertError(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	h.execError("INSERT INTO forum_reports", errors.New("db down"))
	recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodPost,
		"/api/forum/posts/7/report", `{"reason":"spam"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
	covCommentsWantBody(t, recorder, "unable to create report")
}

func TestCovForumCommentsReportPostSuccess(t *testing.T) {
	h := newCovHarness(t)
	covCommentsPostVisible(h)
	recorder := h.do(covCommentsReportHandler(h, "post"), http.MethodPost,
		"/api/forum/posts/7/report", `{"reason":"  spam  "}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusCreated)
	covCommentsWantBody(t, recorder, `"ok":true`)
	if !h.sawQuery("INSERT INTO forum_reports") {
		t.Error("沒有寫入檢舉")
	}
}

func TestCovForumCommentsReportCommentSuccess(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT COUNT(*) FROM forum_post_comments WHERE id = ?",
		[]string{"count"}, []driverValue{int64(1)})
	recorder := h.do(covCommentsReportHandler(h, "comment"), http.MethodPost,
		"/api/forum/posts/7/comments/9/report", `{"reason":"spam"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusCreated)
	if !h.sawQuery("INSERT INTO forum_reports") {
		t.Error("沒有寫入留言檢舉")
	}
}

/* ==========================================================================
   handleForumPostAction 的分派（本組子資源的唯一入口）
   ========================================================================== */

func TestCovForumCommentsDispatchRoutes(t *testing.T) {
	t.Run("留言列表", func(t *testing.T) {
		h := newCovHarness(t)
		covCommentsPostVisible(h)
		recorder := h.do(h.srv.handleForumPostAction, http.MethodGet, "/api/forum/posts/7/comments", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
	})

	t.Run("留言檢舉", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_post_comments WHERE id = ?",
			[]string{"count"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleForumPostAction, http.MethodPost,
			"/api/forum/posts/7/comments/9/report", `{"reason":"spam"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusCreated)
	})

	t.Run("貼文檢舉", func(t *testing.T) {
		h := newCovHarness(t)
		covCommentsPostVisible(h)
		recorder := h.do(h.srv.handleForumPostAction, http.MethodPost,
			"/api/forum/posts/7/report", `{"reason":"spam"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusCreated)
	})

	t.Run("按讚", func(t *testing.T) {
		h := newCovHarness(t)
		covCommentsPostVisible(h)
		h.rows("FROM forum_post_likes WHERE post_id = ? AND author_email = ?",
			[]string{"count"}, []driverValue{int64(0)})
		h.rows("SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ?",
			[]string{"count"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleForumPostAction, http.MethodPost,
			"/api/forum/posts/7/like", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
	})

	t.Run("留言本體方法不支援", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostAction, http.MethodGet,
			"/api/forum/posts/7/comments/9", "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("未知子路徑", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostAction, http.MethodGet,
			"/api/forum/posts/7/nope", "", "", false)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("單篇方法不支援", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostAction, http.MethodPatch,
			"/api/forum/posts/7", "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})
}
