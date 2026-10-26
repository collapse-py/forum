/*
公開貼文端點的覆蓋測試（httpapi/cov_forum_posts_test.go）。

這一份用 cov_harness_test.go 的資料庫替身，把 forum_handlers.go 裡「貼文列表、
單篇、建立、編輯、刪除」的成功路徑與錯誤分支都走過一次。既有的
post_edit_test.go 只涵蓋「碰資料庫之前」就回應的那些分支（401／403／400／405），
真正會讀寫資料庫的路徑則需要 MySQL；這個檔案補上的正是後者。

幾個刻意的手法：

  - 媒體 token 失敗（502）：把 srv.mediaRedis 設成 nil。替身不模擬 Redis 故障，
    而 createMediaToken 對 nil 的處理正是「上游不可用」那條分支。
  - UPDATE 影響 0 列（404）：替身的 Exec 固定回 RowsAffected=1，因此這兩條
    分支改用 sqlmock。它不是資料庫，只是 driver 層的替身，符合「不用 MySQL」。

所有測試函式都以 TestCovForumPosts 開頭，helper 以 covPosts 開頭。
*/
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// covPostsMainColumns 是 forumPostProjection 的欄位順序，必須與 scanForumPostRow
// 的 Scan 目標一一對應；少一欄會讓整個列表回 500，錯位則會安靜地顯示錯數字。
var covPostsMainColumns = []string{
	"id", "author_email", "author_avatar", "content", "created_at", "image_url",
	"pinned", "updated_at", "like_count", "comment_count", "liked_by_me", "bookmarked_by_me",
}

// covPostsMainRow 依投影順序組一列。avatar 與 updated 可傳 nil，分別代表
// LEFT JOIN 沒有個人資料、以及從未編輯過（sql.NullTime 的 Invalid）。
func covPostsMainRow(id int64, author string, avatar driverValue, content, image string,
	pinned int64, updated driverValue, likeCount, commentCount, liked, bookmarked int64) []driverValue {
	return []driverValue{id, author, avatar, content, time.Now(), image,
		pinned, updated, likeCount, commentCount, liked, bookmarked}
}

// covPostsEvilOriginRequest 造一個已登入但 Origin 不在白名單的請求。
func covPostsEvilOriginRequest(h *covHarness, method, path, body, email string) *http.Request {
	request := h.request(method, path, body, email, false)
	request.Header.Set("Origin", "https://evil.example")
	return request
}

func TestCovForumPostsDispatch(t *testing.T) {
	t.Run("GET 走列表", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
	})

	t.Run("POST 走建立", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":"新貼文"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusCreated)
		if !h.sawQuery("INSERT INTO forum_posts") {
			t.Error("沒有執行建立貼文的 INSERT")
		}
	})

	t.Run("其他方法回 405", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodPut, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})
}

func TestCovForumPostsList(t *testing.T) {
	t.Run("空清單", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{`"items":[]`, `"hasMore":false`} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
		if !h.sawQuery("FROM forum_posts fp") {
			t.Error("沒有執行貼文列表查詢")
		}
	})

	t.Run("無效 offset", func(t *testing.T) {
		h := newCovHarness(t)
		for _, query := range []string{"?offset=abc", "?offset=-1"} {
			recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts"+query, "", "", false)
			covWantStatus(t, recorder, http.StatusBadRequest)
			if !strings.Contains(recorder.Body.String(), "invalid offset") {
				t.Errorf("body = %s", recorder.Body.String())
			}
		}
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_posts fp", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("標籤查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(1, "a@example.com", nil, "內容", "", 0, nil, 0, 0, 0, 0))
		// 標籤是逐列查詢；任一列失敗都讓整頁失敗，避免同一篇在不同請求
		// 出現不同欄位組合（見 loadForumPosts 的說明）。
		h.queryError("FROM forum_user_tag_assignments a", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		// 欄位數與 Scan 目標不符：Rows.Scan 會回錯，loadForumPosts 必須把它
		// 當成整頁失敗而不是回一份殘缺的清單。
		h.rows("FROM forum_posts fp", []string{"id"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("媒體 token 失敗回 502", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(1, "a@example.com", "avatar.png", "內容", "", 0, nil, 0, 0, 0, 0))
		// 有頭像的頁面需要簽發 token；nil 代表 Redis 不可用，屬上游故障。
		h.srv.mediaRedis = nil
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusBadGateway)
		if !strings.Contains(recorder.Body.String(), "unable to create media token") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("滿頁時 hasMore 為 true", func(t *testing.T) {
		h := newCovHarness(t)
		rows := make([][]driverValue, 0, 25)
		for i := 0; i < 25; i++ {
			rows = append(rows, covPostsMainRow(int64(i+1), "a@example.com", nil, "x", "", 0, nil, 0, 0, 0, 0))
		}
		h.rows("FROM forum_posts fp", covPostsMainColumns, rows...)
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"hasMore":true`) {
			t.Errorf("body = %s", body)
		}
		if got := strings.Count(body, `"content":"x"`); got != 25 {
			t.Errorf("items 筆數 = %d, want 25", got)
		}
	})

	t.Run("含頭像標籤與編輯狀態", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.cfg.FilesServerPublicURL = "https://files.test"
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(7, "a@example.com", "avatar.png", "內文", "/files/pic.png",
				1, time.Now(), 3, 2, 1, 1))
		h.rows("FROM forum_user_tag_assignments a", []string{"name"}, []driverValue{"VIP"})
		h.rows("SELECT nickname FROM forum_profiles WHERE author_email", []string{"nickname"}, []driverValue{"作者"})

		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{
			`"author":"作者"`,
			`"authorTags":["VIP"]`,
			`"pinned":true`,
			`"edited":true`,
			`"liked":true`,
			`"bookmarked":true`,
			`"likeCount":3`,
			`"commentCount":2`,
			`https://files.test/files/pic.png?token=`,
			`https://files.test/files/avatar.png?token=`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
	})
}

func TestCovForumPostsOffsetParam(t *testing.T) {
	cases := []struct {
		name     string
		target   string
		want     int
		wantOK   bool
		wantCode int
	}{
		{"未設定", "/api/forum/posts", 0, true, 0},
		{"合法值", "/api/forum/posts?offset=12", 12, true, 0},
		{"非數字", "/api/forum/posts?offset=abc", 0, false, http.StatusBadRequest},
		{"負數", "/api/forum/posts?offset=-3", 0, false, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tc.target, nil)
			got, ok := forumOffsetParam(recorder, request)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("forumOffsetParam = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.wantOK)
			}
			if tc.wantCode != 0 {
				covWantStatus(t, recorder, tc.wantCode)
			}
		})
	}
}

func TestCovForumPostsLoadErrorMapping(t *testing.T) {
	h := newCovHarness(t)

	t.Run("媒體 token 錯誤回 502", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		h.srv.writeForumPostLoadError(recorder, fmt.Errorf("%w: redis down", errMediaTokenUnavailable), "boom")
		covWantStatus(t, recorder, http.StatusBadGateway)
	})

	t.Run("其他錯誤回 500 並保留訊息", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		h.srv.writeForumPostLoadError(recorder, errors.New("db down"), "load failed")
		covWantStatus(t, recorder, http.StatusInternalServerError)
		if !strings.Contains(recorder.Body.String(), "load failed") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})
}

func TestCovForumPostsVisibleByID(t *testing.T) {
	const query = "SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL"

	t.Run("存在且未刪除", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"count"}, []driverValue{int64(1)})
		ok, err := h.srv.forumPostVisibleByID(context.Background(), 5)
		if err != nil || !ok {
			t.Fatalf("forumPostVisibleByID = (%v, %v), want (true, nil)", ok, err)
		}
	})

	t.Run("不存在或已軟刪除", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(query, []string{"count"}, []driverValue{int64(0)})
		ok, err := h.srv.forumPostVisibleByID(context.Background(), 5)
		if err != nil || ok {
			t.Fatalf("forumPostVisibleByID = (%v, %v), want (false, nil)", ok, err)
		}
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(query, errors.New("db down"))
		if _, err := h.srv.forumPostVisibleByID(context.Background(), 5); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestCovForumPostsDetail(t *testing.T) {
	t.Run("成功回單篇", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(7, "a@example.com", nil, "永久連結內容", "", 0, nil, 1, 2, 0, 0))
		recorder := h.do(h.srv.handleForumPostDetail, http.MethodGet, "/api/forum/posts/7", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{`"item"`, `"id":7`, `"content":"永久連結內容"`, `"authorKey":"` + publicForumKey("a@example.com") + `"`} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
		if !strings.Contains(recorder.Body.String(), `"likeCount":1`) {
			t.Errorf("body = %s", body)
		}
	})

	t.Run("查無資料回 404", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostDetail, http.MethodGet, "/api/forum/posts/7", "", "", false)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("id 不合法回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostDetail, http.MethodGet, "/api/forum/posts/abc", "", "", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("載入失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_posts fp", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPostDetail, http.MethodGet, "/api/forum/posts/7", "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("媒體 token 失敗回 502", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(7, "a@example.com", nil, "內容", "/files/pic.png", 0, nil, 0, 0, 0, 0))
		h.srv.mediaRedis = nil
		recorder := h.do(h.srv.handleForumPostDetail, http.MethodGet, "/api/forum/posts/7", "", "", false)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})
}

func TestCovForumPostsCreateValidation(t *testing.T) {
	t.Run("未登入回 401", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":"x"}`, "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("來源不可信回 403", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := httptest.NewRecorder()
		h.srv.handleForumPosts(recorder, covPostsEvilOriginRequest(h, http.MethodPost, "/api/forum/posts", `{"content":"x"}`, "me@example.com"))
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("JSON 格式錯誤回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("空白內容回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":"  \n\t "}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "內容不可為空") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("超過 10000 字回 400", func(t *testing.T) {
		h := newCovHarness(t)
		body := `{"content":"` + strings.Repeat("中", 10001) + `"}`
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", body, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "內容最多 10000 字") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("第三方圖片網址回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts",
			`{"content":"x","imageUrl":"https://evil.example/pic.png"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "附圖必須是上傳到本站的圖片") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})
}

func TestCovForumPostsCreateImage(t *testing.T) {
	validBody := `{"content":"有圖","imageUrl":"/files/pic.png"}`

	t.Run("所有權查詢失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		// 沒有上傳紀錄（forum_media 查無資料），退路查詢也失敗：attach 檢查
		// 回傳錯誤，發文必須停下而不是把無法確認的檔名寫進資料庫。
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", validBody, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("不是自己的圖回 400", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT owner_email FROM forum_media WHERE filename", []string{"owner_email"}, []driverValue{"other@example.com"})
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", validBody, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "附圖必須是自己上傳的圖片") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("新簽 token 失敗回 502", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT owner_email FROM forum_media WHERE filename", []string{"owner_email"}, []driverValue{"me@example.com"})
		h.srv.mediaRedis = nil
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", validBody, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadGateway)
		if !strings.Contains(recorder.Body.String(), "unable to create media token") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("舊制檔案以引用計數通過", func(t *testing.T) {
		h := newCovHarness(t)
		// forum_media 沒有這一列（舊制檔案），退路查詢回答「自己的內容引用過它」。
		h.rows("FROM forum_posts WHERE author_email = ? AND image_url", []string{"count"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", validBody, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusCreated)
	})
}

func TestCovForumPostsCreateSuccess(t *testing.T) {
	t.Run("純文字貼文", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts",
			`{"content":"  純文字  "}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusCreated)
		body := recorder.Body.String()
		for _, want := range []string{`"ok":true`, `"content":"純文字"`, `"authorKey":"` + publicForumKey("me@example.com") + `"`} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
		if strings.Contains(body, `"imageUrl"`) {
			t.Errorf("沒有附圖時不該出現 imageUrl：%s", body)
		}
	})

	t.Run("附圖與作者資料", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.cfg.FilesServerPublicURL = "https://files.test"
		h.rows("SELECT owner_email FROM forum_media WHERE filename", []string{"owner_email"}, []driverValue{"me@example.com"})
		h.rows("FROM forum_user_tag_assignments a", []string{"name"}, []driverValue{"管理員"})
		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email", []string{"avatar_url"}, []driverValue{"avatar.png"})
		h.rows("SELECT nickname FROM forum_profiles WHERE author_email", []string{"nickname"}, []driverValue{"小明"})

		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts",
			`{"content":"有圖貼文","imageUrl":"/files/pic.png"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusCreated)
		body := recorder.Body.String()
		for _, want := range []string{
			`"content":"有圖貼文"`,
			`"author":"小明"`,
			`"authorTags":["管理員"]`,
			`https://files.test/files/pic.png?token=`,
			`https://files.test/files/avatar.png?token=`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
		if !h.sawQuery("INSERT INTO forum_posts") {
			t.Error("沒有執行建立貼文的 INSERT")
		}
	})
}

func TestCovForumPostsCreateDBErrors(t *testing.T) {
	t.Run("INSERT 失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_posts", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":"x"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("標籤查詢失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_user_tag_assignments a", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":"x"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
		if !strings.Contains(recorder.Body.String(), "unable to read forum user tags") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("頭像查詢失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT avatar_url FROM forum_profiles WHERE author_email", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":"x"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
		if !strings.Contains(recorder.Body.String(), "unable to read forum profile") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})
}

// TestCovForumPostsCreateIgnoresLastInsertID 頂住「id 不再從 INSERT 回讀」這個
// 改變。舊版在 Exec 之後呼叫 LastInsertId，失敗就回 500；新版 id 在寫入前
// 就由應用層產生，因此同樣的失敗 Result 不該影響任何東西。
func TestCovForumPostsCreateIgnoresLastInsertID(t *testing.T) {
	h := newCovHarness(t)
	mock, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = mock.Close() })
	h.srv.db = mock

	sqlMock.ExpectExec("INSERT INTO forum_posts").
		WillReturnResult(sqlmock.NewErrorResult(errors.New("no last insert id")))
	sqlMock.ExpectQuery("FROM forum_user_tag_assignments").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	sqlMock.ExpectQuery("SELECT avatar_url FROM forum_profiles").
		WillReturnRows(sqlmock.NewRows([]string{"avatar_url"}))

	recorder := h.do(h.srv.handleForumPosts, http.MethodPost, "/api/forum/posts", `{"content":"x"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusCreated)
	covWantCreatedID(t, recorder.Body.String())
	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未滿足的 SQL 預期: %v", err)
	}
}

func TestCovForumPostsUpdateValidation(t *testing.T) {
	t.Run("未登入回 401", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":"x"}`, "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("來源不可信回 403", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := httptest.NewRecorder()
		h.srv.handleForumPostUpdate(recorder, covPostsEvilOriginRequest(h, http.MethodPut, "/api/forum/posts/7", `{"content":"x"}`, "me@example.com"))
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("路徑 id 不合法回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/abc", `{"content":"x"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("JSON 格式錯誤回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("空白內容回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":"   "}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("超過 10000 字回 400", func(t *testing.T) {
		h := newCovHarness(t)
		body := `{"content":"` + strings.Repeat("中", 10001) + `"}`
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", body, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})
}

func TestCovForumPostsUpdateImage(t *testing.T) {
	const currentImageQuery = "SELECT image_url FROM forum_posts WHERE id"

	t.Run("讀取目前附圖失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError(currentImageQuery, errors.New("db down"))
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":"x"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("查無貼文回 404", func(t *testing.T) {
		h := newCovHarness(t)
		// 沒有登記任何規則 → QueryRow 得到 sql.ErrNoRows（不存在／不是你的／已刪除）。
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":"x"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("附圖網址不合法回 400", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(currentImageQuery, []string{"image_url"}, []driverValue{""})
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7",
			`{"content":"x","imageUrl":"https://evil.example/pic.png"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("所有權查詢失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(currentImageQuery, []string{"image_url"}, []driverValue{"old.png"})
		// 新檔名與目前的不同，因此會走 attach 檢查；兩段查詢都沒有紀錄
		// → 第二段回 sql.ErrNoRows → forumFileAttachable 回傳錯誤。
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7",
			`{"content":"x","imageUrl":"/files/new.png"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("不是自己的圖回 400", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(currentImageQuery, []string{"image_url"}, []driverValue{"old.png"})
		h.rows("SELECT owner_email FROM forum_media WHERE filename", []string{"owner_email"}, []driverValue{"other@example.com"})
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7",
			`{"content":"x","imageUrl":"/files/new.png"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "附圖必須是自己上傳的圖片") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})
}

func TestCovForumPostsUpdateSuccess(t *testing.T) {
	const currentImageQuery = "SELECT image_url FROM forum_posts WHERE id"
	const updateQuery = "UPDATE forum_posts SET content"

	t.Run("保留原圖", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(currentImageQuery, []string{"image_url"}, []driverValue{"old.png"})
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":"新內容"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"ok":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if !h.sawQuery(updateQuery) {
			t.Error("沒有執行更新貼文的 UPDATE")
		}
	})

	t.Run("同檔名不重查所有權", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(currentImageQuery, []string{"image_url"}, []driverValue{"/files/same.png"})
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7",
			`{"content":"新內容","imageUrl":"/files/same.png"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		if h.sawQuery("SELECT owner_email FROM forum_media") {
			t.Error("同一個檔名不該再查一次所有權")
		}
	})

	t.Run("換圖", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(currentImageQuery, []string{"image_url"}, []driverValue{"old.png"})
		h.rows("SELECT owner_email FROM forum_media WHERE filename", []string{"owner_email"}, []driverValue{"me@example.com"})
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7",
			`{"content":"新內容","imageUrl":"/files/new.png"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery("SELECT owner_email FROM forum_media") {
			t.Error("換圖前沒有檢查所有權")
		}
		if !h.sawQuery(updateQuery) {
			t.Error("沒有執行更新貼文的 UPDATE")
		}
	})

	t.Run("移除附圖", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows(currentImageQuery, []string{"image_url"}, []driverValue{"/files/old.png"})
		// 回收舊檔會先查引用計數；沒有登記 → 查詢失敗 → releaseForumImage
		// 直接放棄，不會呼叫檔案伺服器（測試因此不需要任何網路）。
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7",
			`{"content":"新內容","imageUrl":""}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery(updateQuery) {
			t.Error("沒有執行更新貼文的 UPDATE")
		}
	})
}

func TestCovForumPostsUpdateDBErrors(t *testing.T) {
	t.Run("UPDATE 失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT image_url FROM forum_posts WHERE id", []string{"image_url"}, []driverValue{""})
		h.execError("UPDATE forum_posts SET content", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":"x"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
		if !strings.Contains(recorder.Body.String(), "unable to update forum post") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})
}

func TestCovForumPostsUpdateZeroRowsNotFound(t *testing.T) {
	h := newCovHarness(t)
	mock, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = mock.Close() })
	h.srv.db = mock

	// 替身的 Exec 固定回 RowsAffected=1，而「不是自己的貼文」正是靠 0 列判定，
	// 因此這一條用 sqlmock 造出 0 列。
	sqlMock.ExpectQuery("SELECT image_url FROM forum_posts").
		WillReturnRows(sqlmock.NewRows([]string{"image_url"}).AddRow("old.png"))
	sqlMock.ExpectExec("UPDATE forum_posts SET content").
		WillReturnResult(sqlmock.NewResult(0, 0))

	recorder := h.do(h.srv.handleForumPostUpdate, http.MethodPut, "/api/forum/posts/7", `{"content":"x"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未滿足的 SQL 預期: %v", err)
	}
}

func TestCovForumPostsDelete(t *testing.T) {
	t.Run("方法錯誤回 405", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostDelete, http.MethodGet, "/api/forum/posts/7", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("未登入回 401", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostDelete, http.MethodDelete, "/api/forum/posts/7", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("來源不可信回 403", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := httptest.NewRecorder()
		h.srv.handleForumPostDelete(recorder, covPostsEvilOriginRequest(h, http.MethodDelete, "/api/forum/posts/7", "", "me@example.com"))
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("路徑 id 不合法回 400", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostDelete, http.MethodDelete, "/api/forum/posts/abc", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("UPDATE 失敗回 500", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("UPDATE forum_posts SET deleted_at", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPostDelete, http.MethodDelete, "/api/forum/posts/7", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
		if !strings.Contains(recorder.Body.String(), "unable to delete forum post") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("軟刪除成功", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPostDelete, http.MethodDelete, "/api/forum/posts/7", "", "me@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"ok":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if !h.sawQuery("UPDATE forum_posts SET deleted_at") {
			t.Error("沒有執行軟刪除的 UPDATE")
		}
	})
}

func TestCovForumPostsDeleteZeroRowsNotFound(t *testing.T) {
	h := newCovHarness(t)
	mock, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = mock.Close() })
	h.srv.db = mock

	// 0 列同時代表「不存在」「不是你的」「已刪除」，一律回 404 以免洩漏作者權限。
	sqlMock.ExpectExec("UPDATE forum_posts SET deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 0))

	recorder := h.do(h.srv.handleForumPostDelete, http.MethodDelete, "/api/forum/posts/7", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未滿足的 SQL 預期: %v", err)
	}
}

func TestCovForumPostsActionDispatch(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		path    string
		body    string
		want    int
		prepare func(h *covHarness)
	}{
		{"單篇 GET 查無資料", http.MethodGet, "/api/forum/posts/5", "", http.StatusNotFound, nil},
		{"單篇 PUT 空內容", http.MethodPut, "/api/forum/posts/5", `{}`, http.StatusBadRequest, nil},
		{"單篇 DELETE 成功", http.MethodDelete, "/api/forum/posts/5", "", http.StatusOK, nil},
		{"單篇 PATCH 不支援", http.MethodPatch, "/api/forum/posts/5", "", http.StatusMethodNotAllowed, nil},
		{"貼文 id 非數字", http.MethodGet, "/api/forum/posts/abc", "", http.StatusNotFound, nil},
		{"未定義子資源", http.MethodGet, "/api/forum/posts/5/abc", "", http.StatusNotFound, nil},
		{"貼文已刪除的留言列表", http.MethodGet, "/api/forum/posts/5/comments", "", http.StatusNotFound,
			func(h *covHarness) {
				// COUNT(*) 一定有一列；替身的空結果集會讓 QueryRow 回 sql.ErrNoRows
				// （那是「查詢失敗」），因此要明確登記 0。
				h.rows("SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL",
					[]string{"count"}, []driverValue{int64(0)})
			}},
		{"按讚方法錯誤", http.MethodGet, "/api/forum/posts/5/like", "", http.StatusMethodNotAllowed, nil},
		{"收藏方法錯誤", http.MethodGet, "/api/forum/posts/5/bookmark", "", http.StatusMethodNotAllowed, nil},
		{"檢舉空白原因", http.MethodPost, "/api/forum/posts/5/report", `{"reason":"  "}`, http.StatusBadRequest, nil},
		{"留言檢舉空白原因", http.MethodPost, "/api/forum/posts/5/comments/3/report", `{"reason":""}`, http.StatusBadRequest, nil},
		{"留言編輯空內容", http.MethodPut, "/api/forum/posts/5/comments/3", `{"content":""}`, http.StatusBadRequest, nil},
		{"留言刪除成功", http.MethodDelete, "/api/forum/posts/5/comments/3", "", http.StatusOK, nil},
		{"留言本體方法錯誤", http.MethodGet, "/api/forum/posts/5/comments/3", "", http.StatusMethodNotAllowed, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCovHarness(t)
			if tc.prepare != nil {
				tc.prepare(h)
			}
			recorder := h.do(h.srv.handleForumPostAction, tc.method, tc.path, tc.body, "me@example.com", false)
			covWantStatus(t, recorder, tc.want)
		})
	}
}

func TestCovForumPostsPostIDFromPath(t *testing.T) {
	cases := []struct {
		path    string
		want    int64
		wantErr bool
	}{
		{"/api/forum/posts/7", 7, false},
		{"/api/forum/posts/7/", 7, false},
		{"/api/forum/posts/7/like", 7, false},
		{"/api/forum/posts/7/comments/3/report", 7, false},
		{"/api/forum/posts/", 0, true},
		{"/api/forum/posts", 0, true},
		{"/api/forum/posts/abc", 0, true},
		{"/api/forum/posts/0", 0, true},
		{"/api/forum/posts/-2", 0, true},
		{"/forum/post/7", 0, true},
	}
	for _, tc := range cases {
		got, err := forumPostIDFromPath(tc.path)
		if (err != nil) != tc.wantErr {
			t.Errorf("forumPostIDFromPath(%q) err = %v, wantErr = %v", tc.path, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("forumPostIDFromPath(%q) = %d, want %d", tc.path, got, tc.want)
		}
	}
}

func TestCovForumPostsPageID(t *testing.T) {
	cases := []struct {
		path   string
		want   int64
		wantOK bool
	}{
		{"/forum/post/1", 1, true},
		{"/forum/post/12345", 12345, true},
		{"/forum/post/1/", 1, true},
		{"/forum/post/0", 0, false},
		{"/forum/post/-1", 0, false},
		{"/forum/post/abc", 0, false},
		{"/forum/post/", 0, false},
		{"/forum/post", 0, false},
		{"/forum/post/1/comments", 0, false},
		{"/forum/others-profile", 0, false},
	}
	for _, tc := range cases {
		got, ok := forumPostPageID(tc.path)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("forumPostPageID(%q) = (%d, %v), want (%d, %v)", tc.path, got, ok, tc.want, tc.wantOK)
		}
	}
}
