/*
管理後台（使用者、標籤、session、封鎖名單）與維運端點的覆蓋測試
（httpapi/cov_admin_user_test.go）。

這一份用 cov_harness_test.go 的資料庫替身，把 handler 的成功路徑與每一條
錯誤分支各走一次：既有的 MySQL 測試需要真實資料庫，這裡則不需要。檔名與
所有測試、helper 都帶 `covAdminUser` 前綴，避免與其他測試檔的同名符號衝突。

涵蓋範圍（皆為本檔自行建立、不修改其他檔案）：
  - user_admin_handlers.go：標籤總表／單一標籤、使用者列表、狀態切換、
    使用者內容、標籤綁定、代發文與代留言。
  - session_admin_handlers.go：session 列表與強制登出。
  - blocklist.go：封鎖名單 API 與中介層。
  - auth_handlers.go：/api/check、/api/logout 與 OAuth 回呼的錯誤路徑。
  - health.go：/healthz 的就緒判定。
  - static.go：靜態路徑黑名單、快取分級與 /forum/* 頁面殼。
*/

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"forum/forum/ipban"
	"forum/forum/session"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

/* ==========================================================================
   測試輔助
   ========================================================================== */

// covAdminUserDoPath 與 h.do 相同，但在建立請求後改寫 URL.Path。
// 有些錯誤分支（例如 url.PathUnescape 失敗的 %zz）無法透過
// httptest.NewRequest 的 target 表達，因為它會先拒絕不合法的 URL。
func covAdminUserDoPath(h *covHarness, handler http.HandlerFunc, method, rawPath, body, email string, admin bool) *httptest.ResponseRecorder {
	h.t.Helper()
	request := h.request(method, "/cov-placeholder", body, email, admin)
	request.URL.Path = rawPath
	request.URL.RawPath = ""
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

// covAdminUserDoUntrusted 送出一個 Origin 不在白名單內的請求。
func covAdminUserDoUntrusted(h *covHarness, handler http.HandlerFunc, method, path, body, email string, admin bool) *httptest.ResponseRecorder {
	h.t.Helper()
	request := h.request(method, path, body, email, admin)
	request.Header.Set("Origin", "https://evil.test")
	request.Header.Del("Referer")
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

// covAdminUserDoCookie 直接以指定的 cookie 呼叫 handler，不另外建立 session。
// session 相關測試需要精確的 session 數量，若每次都用 h.do（它會為請求建立
// 一支新的 session）就無法斷言總數。
func covAdminUserDoCookie(h *covHarness, handler http.HandlerFunc, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	h.t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Origin", covOrigin)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

// covAdminUserClosedRedis 回傳一個已關閉的 Redis client。用它組成的 ipban.Store
// 每次操作都會立刻回錯誤，且不會建立任何連線。
func covAdminUserClosedRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	if err := client.Close(); err != nil {
		t.Fatalf("關閉 Redis client 失敗: %v", err)
	}
	return client
}

// covAdminUserCloseDB 關閉 harness 的資料庫連線，讓後續查詢與交易失敗。
func covAdminUserCloseDB(t *testing.T, h *covHarness) {
	t.Helper()
	if err := h.sqlDB.Close(); err != nil {
		t.Fatalf("關閉資料庫失敗: %v", err)
	}
}

// covAdminUserTagIDs 組出 {"tagIds":[...]} 的請求主體。
func covAdminUserTagIDs(count int) string {
	parts := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		parts = append(parts, strconv.Itoa(i))
	}
	return `{"tagIds":[` + strings.Join(parts, ",") + `]}`
}

// covAdminUserBan 直接把某個 IP 放進封鎖名單，繞過 handler 以便單獨測試
// 「已經被封鎖」的查詢路徑。
func covAdminUserBan(t *testing.T, srv *Server, ip string) time.Time {
	t.Helper()
	until := time.Now().Add(time.Hour).UTC()
	if err := srv.blocks.Ban(context.Background(), ip, until); err != nil {
		t.Fatalf("建立封鎖失敗: %v", err)
	}
	return until
}

/* ==========================================================================
   user_admin_handlers.go：標籤總表
   ========================================================================== */

func TestCovAdminUserTagsCollection(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTags, http.MethodGet, "/api/admin/tags", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("lists rows", func(t *testing.T) {
		h := newCovHarness(t)
		now := time.Now()
		h.rows("FROM forum_user_tags ORDER BY name", []string{"id", "name", "created_at", "updated_at"},
			[]driverValue{int64(1), "alpha", now, now},
			[]driverValue{int64(2), "beta", now, now},
		)
		recorder := h.do(h.srv.handleAdminTags, http.MethodGet, "/api/admin/tags", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"name":"alpha"`) || !strings.Contains(body, `"name":"beta"`) {
			t.Errorf("body = %s", body)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTags, http.MethodGet, "/api/admin/tags", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"items":[]`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("list query error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_user_tags ORDER BY name", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminTags, http.MethodGet, "/api/admin/tags", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("list scan error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_user_tags ORDER BY name", []string{"id"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleAdminTags, http.MethodGet, "/api/admin/tags", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("create", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTags, http.MethodPost, "/api/admin/tags", `{"name":"  vip  "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusCreated)
		body := recorder.Body.String()
		covWantCreatedID(t, body)
		if !strings.Contains(body, `"name":"vip"`) {
			t.Errorf("body = %s", body)
		}
		if !h.sawQuery("INSERT INTO forum_user_tags") {
			t.Error("沒有寫入標籤")
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_user_tags", &mysql.MySQLError{Number: 1062, Message: "duplicate"})
		recorder := h.do(h.srv.handleAdminTags, http.MethodPost, "/api/admin/tags", `{"name":"vip"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusConflict)
	})

	t.Run("create invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTags, http.MethodPost, "/api/admin/tags", "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("create invalid name", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTags, http.MethodPost, "/api/admin/tags", `{"name":"   "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)

		tooLong := `{"name":"` + strings.Repeat("標", 51) + `"}`
		recorder = h.do(h.srv.handleAdminTags, http.MethodPost, "/api/admin/tags", tooLong, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("create untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminTags, http.MethodPost, "/api/admin/tags", `{"name":"vip"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("create transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminTags, http.MethodPost, "/api/admin/tags", `{"name":"vip"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTags, http.MethodPut, "/api/admin/tags", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})
}

/* ==========================================================================
   user_admin_handlers.go：單一標籤
   ========================================================================== */

func TestCovAdminUserTagItem(t *testing.T) {
	validTagRules := func(h *covHarness) {
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"old"})
		h.rows("SELECT COUNT(*) FROM forum_user_tag_assignments WHERE tag_id", []string{"count"}, []driverValue{int64(3)})
	}

	t.Run("invalid id", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/abc", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)

		recorder = h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/0", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/1", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("delete success", func(t *testing.T) {
		h := newCovHarness(t)
		validTagRules(h)
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"ok":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if !h.sawQuery("DELETE FROM forum_user_tags WHERE id") {
			t.Error("沒有刪除標籤")
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("delete lookup error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT name FROM forum_user_tags WHERE id", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("delete count error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"old"})
		h.queryError("SELECT COUNT(*) FROM forum_user_tag_assignments WHERE tag_id", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("delete exec error", func(t *testing.T) {
		h := newCovHarness(t)
		validTagRules(h)
		h.execError("DELETE FROM forum_user_tags WHERE id", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("delete assignments error", func(t *testing.T) {
		h := newCovHarness(t)
		validTagRules(h)
		h.execError("DELETE FROM forum_user_tag_assignments WHERE tag_id", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("delete audit error", func(t *testing.T) {
		h := newCovHarness(t)
		validTagRules(h)
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("delete untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("delete transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminTag, http.MethodDelete, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("patch success", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"old"})
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":" new "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery("UPDATE forum_user_tags SET name") {
			t.Error("沒有更新標籤")
		}
	})

	t.Run("patch not found", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"new"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("patch conflict", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"old"})
		h.execError("UPDATE forum_user_tags SET name", &mysql.MySQLError{Number: 1062, Message: "duplicate"})
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"new"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusConflict)
	})

	t.Run("patch update error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"old"})
		h.execError("UPDATE forum_user_tags SET name", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"new"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("patch audit error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"old"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"new"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("patch lookup error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT name FROM forum_user_tags WHERE id", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"new"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("patch invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("patch empty name", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"  "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("patch untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"new"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("patch transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminTag, http.MethodPatch, "/api/admin/tags/7", `{"name":"new"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("unsupported method", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminTag, http.MethodPut, "/api/admin/tags/7", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})
}

/* ==========================================================================
   user_admin_handlers.go：使用者列表
   ========================================================================== */

func TestCovAdminUserList(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUsers, http.MethodGet, "/api/admin/users", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUsers, http.MethodPost, "/api/admin/users", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("empty", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUsers, http.MethodGet, "/api/admin/users", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"items":[]`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("rows with tags", func(t *testing.T) {
		h := newCovHarness(t)
		now := time.Now()
		h.rows("FROM forum_users u",
			[]string{"email", "status", "nickname", "created_at", "updated_at", "post_count", "comment_count", "like_count"},
			[]driverValue{"a@example.com", "ACTIVE", "A", now, now, int64(1), int64(2), int64(3)},
			[]driverValue{"b@example.com", "SUSPENDED", "B", now, now, int64(4), int64(5), int64(6)},
		)
		h.rows("SELECT t.id, t.name", []string{"id", "name", "created_at", "updated_at"},
			[]driverValue{int64(7), "vip", now, now},
		)
		recorder := h.do(h.srv.handleAdminUsers, http.MethodGet, "/api/admin/users", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{"a@example.com", "b@example.com", `"nickname":"A"`, `"name":"vip"`, `"postCount":1`} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
	})

	t.Run("query error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_users u", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUsers, http.MethodGet, "/api/admin/users", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("scan error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_users u", []string{"email"}, []driverValue{"a@example.com"})
		recorder := h.do(h.srv.handleAdminUsers, http.MethodGet, "/api/admin/users", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("tags query error", func(t *testing.T) {
		h := newCovHarness(t)
		now := time.Now()
		h.rows("FROM forum_users u",
			[]string{"email", "status", "nickname", "created_at", "updated_at", "post_count", "comment_count", "like_count"},
			[]driverValue{"a@example.com", "ACTIVE", "A", now, now, int64(0), int64(0), int64(0)},
		)
		h.queryError("SELECT t.id, t.name", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUsers, http.MethodGet, "/api/admin/users", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

/* ==========================================================================
   user_admin_handlers.go：帳號狀態與子資源分派
   ========================================================================== */

func TestCovAdminUserStatusUpdate(t *testing.T) {
	const userPath = "/api/admin/users/user@example.com"

	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"SUSPENDED"}`, "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("get method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, userPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("delete method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodDelete, userPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("post method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, userPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("missing email", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, "/api/admin/users/", `{"status":"ACTIVE"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid escape", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoPath(h, h.srv.handleAdminUser, http.MethodPatch, "/api/admin/users/%zz", `{"status":"ACTIVE"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid status", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"banana"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("not found", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"SUSPENDED"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("suspend", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email", []string{"status"}, []driverValue{"ACTIVE"})
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":" suspended "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"ok":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if !h.sawQuery("UPDATE forum_users SET status") {
			t.Error("沒有更新狀態")
		}
	})

	t.Run("reinstate", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email", []string{"status"}, []driverValue{"SUSPENDED"})
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"active"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
	})

	t.Run("lookup error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT status FROM forum_users WHERE email", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"ACTIVE"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("update error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email", []string{"status"}, []driverValue{"ACTIVE"})
		h.execError("UPDATE forum_users SET status", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"ACTIVE"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("audit error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT status FROM forum_users WHERE email", []string{"status"}, []driverValue{"ACTIVE"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"ACTIVE"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"ACTIVE"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminUser, http.MethodPatch, userPath, `{"status":"ACTIVE"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})
}

/* ==========================================================================
   user_admin_handlers.go：使用者內容（文章與留言）
   ========================================================================== */

func TestCovAdminUserContent(t *testing.T) {
	const contentPath = "/api/admin/users/user@example.com/content"

	postCols := []string{"id", "author_email", "content", "created_at", "image_url", "like_count", "comment_count"}
	commentCols := []string{"id", "post_id", "parent_id", "depth", "author_email", "content", "created_at"}

	t.Run("empty", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"posts":[]`) || !strings.Contains(body, `"comments":[]`) {
			t.Errorf("body = %s", body)
		}
	})

	t.Run("posts and comments with image", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.cfg.FilesServerPublicURL = "https://files.test"
		now := time.Now()
		h.rows("FROM forum_posts WHERE author_email", postCols,
			[]driverValue{int64(1), "user@example.com", "hello", now, "pic.png", int64(2), int64(3)},
		)
		h.rows("FROM forum_post_comments WHERE author_email", commentCols,
			[]driverValue{int64(5), int64(1), int64(0), 0, "user@example.com", "nice", now},
		)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{`"content":"hello"`, `"imageUrl":"https://files.test/files/pic.png?token=`, `"postId":1`, `"content":"nice"`} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
	})

	t.Run("post without usable image", func(t *testing.T) {
		h := newCovHarness(t)
		now := time.Now()
		h.rows("FROM forum_posts WHERE author_email", postCols,
			[]driverValue{int64(1), "user@example.com", "hello", now, "not-an-image.txt", int64(0), int64(0)},
		)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if strings.Contains(recorder.Body.String(), `"imageUrl"`) {
			t.Errorf("不可用的圖片不該出現在回應：%s", recorder.Body.String())
		}
	})

	t.Run("media token error", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.mediaRedis = nil
		now := time.Now()
		h.rows("FROM forum_posts WHERE author_email", postCols,
			[]driverValue{int64(1), "user@example.com", "hello", now, "pic.png", int64(0), int64(0)},
		)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})

	t.Run("post query error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_posts WHERE author_email", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("post scan error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts WHERE author_email", []string{"id"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("comment query error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_post_comments WHERE author_email", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("comment scan error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_post_comments WHERE author_email", []string{"id"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, contentPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("invalid email", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoPath(h, h.srv.handleAdminUser, http.MethodGet, "/api/admin/users/%zz/content", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})
}

/* ==========================================================================
   user_admin_handlers.go：使用者標籤綁定
   ========================================================================== */

func TestCovAdminUserTagBindings(t *testing.T) {
	const tagsPath = "/api/admin/users/user@example.com/tags"

	baseRules := func(h *covHarness) {
		h.rows("SELECT COUNT(*) FROM forum_users WHERE email", []string{"count"}, []driverValue{int64(1)})
	}

	putRules := func(h *covHarness) {
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(2)})
		h.rows("SELECT t.name", []string{"name"}, []driverValue{"alpha"}, []driverValue{"beta"})
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"alpha"})
		h.rows("SELECT t.id, t.name", []string{"id", "name", "created_at", "updated_at"},
			[]driverValue{int64(1), "alpha", time.Now(), time.Now()},
			[]driverValue{int64(2), "beta", time.Now(), time.Now()},
		)
	}

	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, tagsPath, "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("invalid email", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoPath(h, h.srv.handleAdminUser, http.MethodGet, "/api/admin/users/%zz/tags", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("user lookup error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT COUNT(*) FROM forum_users WHERE email", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, tagsPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("user not found", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_users WHERE email", []string{"count"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, tagsPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("get success", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT t.id, t.name", []string{"id", "name", "created_at", "updated_at"},
			[]driverValue{int64(3), "vip", time.Now(), time.Now()},
		)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, tagsPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"name":"vip"`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("get tags error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.queryError("SELECT t.id, t.name", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, tagsPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, tagsPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("too many tags", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(batchMaxTags+1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid tag ids", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, `{"tagIds":[0]}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)

		recorder = h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, `{"tagIds":[1,1]}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("unknown tag", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(2), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("tag validation error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.queryError("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("tag names load error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(1)})
		h.queryError("SELECT t.name", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("tag name lookup error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(1)})
		h.rows("SELECT t.name", []string{"name"}, []driverValue{"alpha"})
		h.queryError("SELECT name FROM forum_user_tags WHERE id", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("delete assignments error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(1)})
		h.rows("SELECT t.name", []string{"name"}, []driverValue{"alpha"})
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"alpha"})
		h.execError("DELETE FROM forum_user_tag_assignments WHERE user_email", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("insert assignment error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(1)})
		h.rows("SELECT t.name", []string{"name"}, []driverValue{"alpha"})
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"alpha"})
		h.execError("INSERT INTO forum_user_tag_assignments", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("audit error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(1)})
		h.rows("SELECT t.name", []string{"name"}, []driverValue{"alpha"})
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"alpha"})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("reload error", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT COUNT(*) FROM forum_user_tags WHERE id IN", []string{"count"}, []driverValue{int64(1)})
		h.rows("SELECT t.name", []string{"name"}, []driverValue{"alpha"})
		h.rows("SELECT name FROM forum_user_tags WHERE id = ?", []string{"name"}, []driverValue{"alpha"})
		h.queryError("SELECT t.id, t.name", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(1), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("update success", func(t *testing.T) {
		h := newCovHarness(t)
		putRules(h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, covAdminUserTagIDs(2), "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"items":[`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if !h.sawQuery("DELETE FROM forum_user_tag_assignments WHERE user_email") {
			t.Error("沒有清除既有綁定")
		}
		if !h.sawQuery("INSERT INTO forum_user_tag_assignments") {
			t.Error("沒有寫入新綁定")
		}
	})

	t.Run("clear success", func(t *testing.T) {
		h := newCovHarness(t)
		baseRules(h)
		h.rows("SELECT t.id, t.name", []string{"id", "name", "created_at", "updated_at"})
		recorder := h.do(h.srv.handleAdminUser, http.MethodPut, tagsPath, `{"tagIds":[]}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"items":[]`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})
}

/* ==========================================================================
   user_admin_handlers.go：代發文與代留言
   ========================================================================== */

func TestCovAdminUserCreatePost(t *testing.T) {
	const postsPath = "/api/admin/users/user@example.com/posts"

	t.Run("success", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, postsPath, `{"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusCreated)
		covWantCreatedID(t, recorder.Body.String())
		if !h.sawQuery("INSERT INTO forum_posts") {
			t.Error("沒有寫入文章")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, postsPath, `{"content":"hello"}`, "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminUser, http.MethodPost, postsPath, `{"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("invalid email", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoPath(h, h.srv.handleAdminUser, http.MethodPost, "/api/admin/users/%zz/posts", `{"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, postsPath, "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("empty content", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, postsPath, `{"content":"   "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("insert error", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_posts", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, postsPath, `{"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("audit error", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, postsPath, `{"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, postsPath, `{"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("get method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, postsPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})
}

func TestCovAdminUserCreateComment(t *testing.T) {
	const commentsPath = "/api/admin/users/user@example.com/comments"

	t.Run("success", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_posts WHERE id", []string{"count"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusCreated)
		covWantCreatedID(t, recorder.Body.String())
		if !h.sawQuery("INSERT INTO forum_post_comments") {
			t.Error("沒有寫入留言")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("invalid email", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoPath(h, h.srv.handleAdminUser, http.MethodPost, "/api/admin/users/%zz/comments", `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("empty content", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"  "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("post lookup error", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT COUNT(*) FROM forum_posts WHERE id", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("post not found", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_posts WHERE id", []string{"count"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("insert error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_posts WHERE id", []string{"count"}, []driverValue{int64(1)})
		h.execError("INSERT INTO forum_post_comments", errors.New("db down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("audit error", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT COUNT(*) FROM forum_posts WHERE id", []string{"count"}, []driverValue{int64(1)})
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminUser, http.MethodPost, commentsPath, `{"postId":5,"content":"hello"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("get method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminUser, http.MethodGet, commentsPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})
}

/* ==========================================================================
   user_admin_handlers.go：純函式
   ========================================================================== */

func TestCovAdminUserHelpers(t *testing.T) {
	h := newCovHarness(t)
	srv := h.srv

	if email, err := srv.decodeAdminUserEmail("user%40example.com"); err != nil || email != "user@example.com" {
		t.Fatalf("decodeAdminUserEmail = %q, %v", email, err)
	}
	if _, err := srv.decodeAdminUserEmail("%zz"); err == nil {
		t.Error("不合法的百分號編碼應該回錯誤")
	}
	if _, err := srv.decodeAdminUserEmail(""); err == nil {
		t.Error("空 email 應該回錯誤")
	}

	if got := joinTagNames([]string{"beta", "alpha"}); got != "alpha, beta" {
		t.Errorf("joinTagNames = %q", got)
	}
	if !sameTagSet([]string{"a", "b"}, []string{"b", "a"}) {
		t.Error("同一組標籤應視為相同")
	}
	if sameTagSet([]string{"a"}, []string{"b"}) {
		t.Error("不同標籤不應視為相同")
	}
	if sameTagSet([]string{"a"}, nil) {
		t.Error("長度不同不應視為相同")
	}
	if got := sortedTagNames([]string{"b", "a"}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("sortedTagNames = %v", got)
	}
}

/* ==========================================================================
   session_admin_handlers.go
   ========================================================================== */

func TestCovAdminUserSessionsList(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminSessions, http.MethodGet, "/api/admin/sessions", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminSessions, http.MethodPost, "/api/admin/sessions", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("empty", func(t *testing.T) {
		h := newCovHarness(t)
		cookie := h.login("admin@example.com", true)
		recorder := covAdminUserDoCookie(h, h.srv.handleAdminSessions, http.MethodGet, "/api/admin/sessions", "", cookie)
		covWantStatus(t, recorder, http.StatusOK)
		var payload adminSessionsPayload
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("解析回應失敗: %v；body = %s", err, recorder.Body.String())
		}
		if !payload.OK || len(payload.Items) != 1 || payload.TotalActive != 1 {
			t.Errorf("payload = %+v", payload)
		}
		if payload.Items[0].Email != "admin@example.com" || !payload.Items[0].IsAdmin {
			t.Errorf("payload = %+v", payload)
		}
		if payload.ExpireInHours != 1 {
			t.Errorf("ExpireInHours = %d, want 1", payload.ExpireInHours)
		}
	})

	t.Run("with sessions and filter", func(t *testing.T) {
		h := newCovHarness(t)
		cookie := h.login("admin@example.com", true)
		if _, err := h.srv.sessions.Create("a@example.com", false); err != nil {
			t.Fatalf("建立 session 失敗: %v", err)
		}
		if _, err := h.srv.sessions.Create("b@example.com", true); err != nil {
			t.Fatalf("建立 session 失敗: %v", err)
		}

		recorder := covAdminUserDoCookie(h, h.srv.handleAdminSessions, http.MethodGet, "/api/admin/sessions", "", cookie)
		covWantStatus(t, recorder, http.StatusOK)
		var payload adminSessionsPayload
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("解析回應失敗: %v", err)
		}
		if len(payload.Items) != 3 || payload.TotalActive != 3 {
			t.Errorf("payload = %+v", payload)
		}
		for _, item := range payload.Items {
			if item.TokenPrefix == "" {
				t.Error("tokenPrefix 不應為空")
			}
		}

		recorder = covAdminUserDoCookie(h, h.srv.handleAdminSessions, http.MethodGet, "/api/admin/sessions?email=a@example.com", "", cookie)
		covWantStatus(t, recorder, http.StatusOK)
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("解析回應失敗: %v", err)
		}
		if len(payload.Items) != 1 || payload.Items[0].Email != "a@example.com" || payload.TotalActive != 1 {
			t.Errorf("payload = %+v", payload)
		}
	})

	t.Run("limit and invalid limit", func(t *testing.T) {
		h := newCovHarness(t)
		cookie := h.login("admin@example.com", true)
		for _, email := range []string{"a@example.com", "b@example.com"} {
			if _, err := h.srv.sessions.Create(email, false); err != nil {
				t.Fatalf("建立 session 失敗: %v", err)
			}
		}

		recorder := covAdminUserDoCookie(h, h.srv.handleAdminSessions, http.MethodGet, "/api/admin/sessions?limit=1", "", cookie)
		covWantStatus(t, recorder, http.StatusOK)
		var payload adminSessionsPayload
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("解析回應失敗: %v", err)
		}
		if len(payload.Items) != 1 || payload.TotalActive != 3 {
			t.Errorf("payload = %+v", payload)
		}

		recorder = covAdminUserDoCookie(h, h.srv.handleAdminSessions, http.MethodGet, "/api/admin/sessions?limit=abc", "", cookie)
		covWantStatus(t, recorder, http.StatusOK)
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("解析回應失敗: %v", err)
		}
		if len(payload.Items) != 3 {
			t.Errorf("不合法的 limit 應退回預設值：payload = %+v", payload)
		}
	})

	t.Run("non admin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminSessions, http.MethodGet, "/api/admin/sessions", "", "user@example.com", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})
}

func TestCovAdminUserSessionsRevoke(t *testing.T) {
	const revokePath = "/api/admin/sessions/revoke"

	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"a@example.com"}`, "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminRevokeSessions, http.MethodGet, revokePath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"a@example.com"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid email", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"nope"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)

		recorder = h.do(h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"   "}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		h := newCovHarness(t)
		cookie := h.login("admin@example.com", true)
		for i := 0; i < 2; i++ {
			if _, err := h.srv.sessions.Create("victim@example.com", false); err != nil {
				t.Fatalf("建立 session 失敗: %v", err)
			}
		}
		recorder := covAdminUserDoCookie(h, h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"victim@example.com"}`, cookie)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"revoked":2`) || !strings.Contains(body, `"email":"victim@example.com"`) {
			t.Errorf("body = %s", body)
		}
		result, err := h.srv.sessions.List(context.Background(), session.ListOptions{Email: "victim@example.com"})
		if err != nil {
			t.Fatalf("列出 session 失敗: %v", err)
		}
		if result.TotalActive != 0 {
			t.Errorf("撤銷後仍有 %d 筆 session", result.TotalActive)
		}
	})

	t.Run("no sessions", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"nobody@example.com"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"revoked":0`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("audit error", func(t *testing.T) {
		h := newCovHarness(t)
		if _, err := h.srv.sessions.Create("victim@example.com", false); err != nil {
			t.Fatalf("建立 session 失敗: %v", err)
		}
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"victim@example.com"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
		if !strings.Contains(recorder.Body.String(), "稽核紀錄寫入失敗") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		if _, err := h.srv.sessions.Create("victim@example.com", false); err != nil {
			t.Fatalf("建立 session 失敗: %v", err)
		}
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminRevokeSessions, http.MethodPost, revokePath, `{"email":"victim@example.com"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovAdminUserSessionHelpers(t *testing.T) {
	if got := formatSessionTime(time.Time{}); got != "" {
		t.Errorf("零值時間應回空字串，got %q", got)
	}
	moment := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := formatSessionTime(moment); got != "2024-01-02T03:04:05Z" {
		t.Errorf("formatSessionTime = %q", got)
	}

	if sessionExpiresBefore(adminSessionView{}, adminSessionView{ExpiresAt: "2030-01-01T00:00:00Z"}) {
		t.Error("未知到期時間不該被視為較早到期")
	}
	if !sessionExpiresBefore(adminSessionView{ExpiresAt: "2020-01-01T00:00:00Z"}, adminSessionView{}) {
		t.Error("已知到期時間應排在未知之前")
	}
	if !sessionExpiresBefore(adminSessionView{ExpiresAt: "2020-01-01T00:00:00Z"}, adminSessionView{ExpiresAt: "2030-01-01T00:00:00Z"}) {
		t.Error("較早的時間應被視為較早到期")
	}
	if sessionExpiresBefore(adminSessionView{ExpiresAt: "2030-01-01T00:00:00Z"}, adminSessionView{ExpiresAt: "2020-01-01T00:00:00Z"}) {
		t.Error("較晚的時間不該被視為較早到期")
	}

	items := []adminSessionView{
		{Email: "unknown"},
		{Email: "late", ExpiresAt: "2030-01-01T00:00:00Z"},
		{Email: "early", ExpiresAt: "2020-01-01T00:00:00Z"},
	}
	sortSessionsByExpiry(items)
	if len(items) != 3 {
		t.Fatalf("排序後長度改變：%v", items)
	}

	// 兩筆依到期時間遞增排列的輸入，插入排序必須把較晚到期者移到前面，
	// 才會走到迴圈內搬移元素的分支。
	descending := []adminSessionView{
		{Email: "early", ExpiresAt: "2020-01-01T00:00:00Z"},
		{Email: "late", ExpiresAt: "2030-01-01T00:00:00Z"},
	}
	sortSessionsByExpiry(descending)
	if descending[0].Email != "late" || descending[1].Email != "early" {
		t.Errorf("排序結果 = %v，want [late early]", descending)
	}
}

/* ==========================================================================
   blocklist.go：後臺 API
   ========================================================================== */

func TestCovAdminUserBlocks(t *testing.T) {
	const blocksPath = "/api/admin/blocks"

	t.Run("unauthenticated", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodGet, blocksPath, "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPut, blocksPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("unavailable without store", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = nil
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodGet, blocksPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"available":false`) || !strings.Contains(body, `"items":[]`) {
			t.Errorf("body = %s", body)
		}
	})

	t.Run("list empty", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodGet, blocksPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"available":true`) || !strings.Contains(body, `"items":[]`) {
			t.Errorf("body = %s", body)
		}
	})

	t.Run("block list and unblock", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.9","minutes":1440,"reason":"spam"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"expiresAt"`) {
			t.Errorf("body = %s", recorder.Body.String())
		}

		recorder = h.do(h.srv.handleAdminBlocks, http.MethodGet, blocksPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), "203.0.113.9") {
			t.Errorf("清單缺少被封鎖的 IP：%s", recorder.Body.String())
		}

		recorder = h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.9","minutes":0}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"existed":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}

		recorder = h.do(h.srv.handleAdminBlocks, http.MethodGet, blocksPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if strings.Contains(recorder.Body.String(), "203.0.113.9") {
			t.Errorf("解封後仍在清單：%s", recorder.Body.String())
		}

		recorder = h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.9","minutes":0}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"existed":false`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("clamp long ban", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.10","minutes":1000000}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
	})

	t.Run("invalid ip", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"not-an-ip","minutes":10}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("reason too long", func(t *testing.T) {
		h := newCovHarness(t)
		body := `{"ip":"203.0.113.11","minutes":10,"reason":"` + strings.Repeat("字", 201) + `"}`
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath, body, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("invalid body", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath, "{", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("untrusted origin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covAdminUserDoUntrusted(h, h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.12","minutes":10}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("block without store", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = nil
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.13","minutes":10}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusServiceUnavailable)
	})

	t.Run("unblock without store", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = nil
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.13","minutes":0}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusServiceUnavailable)
	})

	t.Run("list error", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = ipban.New(covAdminUserClosedRedis(t))
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodGet, blocksPath, "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("block error", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = ipban.New(covAdminUserClosedRedis(t))
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.14","minutes":10}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("unblock error", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = ipban.New(covAdminUserClosedRedis(t))
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.14","minutes":0}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("block audit error", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.15","minutes":10,"reason":"spam"}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("unblock audit error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserBan(t, h.srv, "203.0.113.16")
		h.execError("INSERT INTO forum_admin_actions", errors.New("audit down"))
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.16","minutes":0}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("block transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.17","minutes":10}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("unblock transaction error", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserBan(t, h.srv, "203.0.113.18")
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleAdminBlocks, http.MethodPost, blocksPath,
			`{"ip":"203.0.113.18","minutes":0}`, "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

/* ==========================================================================
   blocklist.go：中介層與標記
   ========================================================================== */

func TestCovAdminUserBlocklistMiddleware(t *testing.T) {
	t.Run("nil store fails open", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = nil
		called := false
		next := func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}
		// 走 applyRateLimit（封鎖中介層唯一的掛載點）而不是直接呼叫
		// withBlocklistHandler：直接呼叫會違反 blocklistmount 不變條件，
		// 而從掛載點走過才是真正被服務的組裝方式。
		handler := h.srv.applyRateLimit(h.srv.writeRateLimiter, next, false)
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if !called || recorder.Code != http.StatusNoContent {
			t.Fatalf("called=%v code=%d", called, recorder.Code)
		}
	})

	t.Run("unbanned continues", func(t *testing.T) {
		h := newCovHarness(t)
		called := false
		next := func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}
		handler := h.srv.applyRateLimit(h.srv.writeRateLimiter, next, false)
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if !called || recorder.Code != http.StatusNoContent {
			t.Fatalf("called=%v code=%d", called, recorder.Code)
		}
	})

	t.Run("banned is rejected and marked", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserBan(t, h.srv, "192.0.2.1")
		called := false
		next := func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}
		handler := h.srv.applyRateLimit(h.srv.writeRateLimiter, next, false)
		request := withBannedMarker(httptest.NewRequest(http.MethodGet, "/", nil))
		marker, ok := request.Context().Value(bannedRequestKey{}).(*bannedMarker)
		if !ok {
			t.Fatal("context 裡沒有 bannedMarker")
		}
		recorder := httptest.NewRecorder()
		handler(recorder, request)
		if called {
			t.Error("被封鎖的請求不該繼續往下走")
		}
		covWantStatus(t, recorder, http.StatusForbidden)
		if recorder.Header().Get("Retry-After") == "" {
			t.Error("缺少 Retry-After 表頭")
		}
		if !marker.blocked {
			t.Error("請求沒有被標記為封鎖")
		}
	})

	t.Run("redis failure fails open", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.blocks = ipban.New(covAdminUserClosedRedis(t))
		called := false
		next := func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}
		handler := h.srv.applyRateLimit(h.srv.writeRateLimiter, next, false)
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if !called || recorder.Code != http.StatusNoContent {
			t.Fatalf("called=%v code=%d", called, recorder.Code)
		}
	})

	t.Run("candidates include forwarded address", func(t *testing.T) {
		h := newCovHarness(t)
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.RemoteAddr = "127.0.0.1:1234"
		request.Header.Set("X-Forwarded-For", "203.0.113.7")

		banned, until, err := h.srv.isAnyCandidateBanned(request, h.srv.blocks, h.srv.trustedProxies)
		if err != nil || banned || !until.IsZero() {
			t.Fatalf("banned=%v until=%v err=%v", banned, until, err)
		}

		covAdminUserBan(t, h.srv, "127.0.0.1")
		banned, _, err = h.srv.isAnyCandidateBanned(request, h.srv.blocks, h.srv.trustedProxies)
		if err != nil || !banned {
			t.Fatalf("第二個候選位址被封鎖時應回 banned=true：banned=%v err=%v", banned, err)
		}
	})

	t.Run("banned response without marker or future", func(t *testing.T) {
		// 沒有 marker 的請求：markBannedRequest 靜靜不做事，不該 panic。
		recorder := httptest.NewRecorder()
		writeBannedResponse(recorder, httptest.NewRequest(http.MethodGet, "/", nil), time.Now().Add(-time.Minute))
		covWantStatus(t, recorder, http.StatusForbidden)
		if recorder.Header().Get("Retry-After") != "" {
			t.Error("已到期的封鎖不該送出 Retry-After")
		}
	})

	t.Run("log throttle", func(t *testing.T) {
		logBlocklistFailure(errors.New("first"))
		logBlocklistFailure(errors.New("second"))
		logBlocklistRecovered()
		logBlocklistRecovered()
	})
}

/* ==========================================================================
   auth_handlers.go
   ========================================================================== */

func TestCovAdminUserAuthHandlers(t *testing.T) {
	t.Run("check anonymous", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleCheck, http.MethodGet, "/api/check", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"ok":false`) || !strings.Contains(body, `"isAdmin":false`) {
			t.Errorf("body = %s", body)
		}
	})

	t.Run("check user", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleCheck, http.MethodGet, "/api/check", "", "user@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"isAdmin":false`) {
			t.Errorf("body = %s", body)
		}
	})

	t.Run("check admin", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleCheck, http.MethodGet, "/api/check", "", "admin@example.com", true)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"isAdmin":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("check method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleCheck, http.MethodPost, "/api/check", "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("logout method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleLogout, http.MethodGet, "/api/logout", "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("logout removes session", func(t *testing.T) {
		h := newCovHarness(t)
		request := h.request(http.MethodPost, "/api/logout", "", "user@example.com", false)
		recorder := httptest.NewRecorder()
		h.srv.handleLogout(recorder, request)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"ok":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if h.srv.sessions.IsLoggedIn(request) {
			t.Error("登出後 session 仍然有效")
		}
	})

	t.Run("logout without session", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleLogout, http.MethodPost, "/api/logout", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
	})

	t.Run("callback without code", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleGoogleCallback, http.MethodGet, "/auth/callback", "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

/* ==========================================================================
   health.go
   ========================================================================== */

func TestCovAdminUserHealth(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleHealth, http.MethodGet, "/healthz", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{`"ok":true`, `"mysql":"ok"`, `"redis":"ok"`, `"es":"disabled"`} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleHealth, http.MethodPost, "/healthz", "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("mysql down", func(t *testing.T) {
		h := newCovHarness(t)
		covAdminUserCloseDB(t, h)
		recorder := h.do(h.srv.handleHealth, http.MethodGet, "/healthz", "", "", false)
		covWantStatus(t, recorder, http.StatusServiceUnavailable)
		if !strings.Contains(recorder.Body.String(), `"ok":false`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("redis down", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.mediaRedis = covAdminUserClosedRedis(t)
		recorder := h.do(h.srv.handleHealth, http.MethodGet, "/healthz", "", "", false)
		covWantStatus(t, recorder, http.StatusServiceUnavailable)
	})
}

/* ==========================================================================
   static.go
   ========================================================================== */

func TestCovAdminUserStaticPaths(t *testing.T) {
	blocked := map[string]bool{
		"":                     false,
		"/":                    false,
		"/config.conf":         true,
		"/go.mod":              true,
		"/service-worker.js":   false,
		"/forum-manifest.json": false,
		"/forum/":              false, // path.Clean 會去掉尾斜線，"/forum" 本身不在封鎖清單內。
		"/forum/post/1":        true,
		"/src/App.tsx":         true,
		"/範例/logo.png":         true,
		"/app.js":              false,
		"/package.json":        true,
		"/config.GO":           true,
		"/x.tsx":               true,
		"/x.jsx":               true,
		"/x.sql":               true,
		"/x.sum":               true,
		"/x.mod":               true,
	}
	for path, want := range blocked {
		if got := isBlockedStaticPath(path); got != want {
			t.Errorf("isBlockedStaticPath(%q) = %v, want %v", path, got, want)
		}
	}

	cacheable := map[string]bool{
		"/app.css":    true,
		"/app.JS":     true,
		"/logo.png":   true,
		"/font.woff":  true,
		"/font.woff2": true,
		"/pic.jpeg":   true,
		"/pic.webp":   true,
		"/pic.gif":    true,
		"/pic.svg":    true,
		"/pic.jpg":    true,
		"/font.ttf":   true,
		"/notes.txt":  false,
		"/":           false,
	}
	for path, want := range cacheable {
		if got := isCacheableStaticPath(path); got != want {
			t.Errorf("isCacheableStaticPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestCovAdminUserStaticServer(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"index.html":    "<html>index</html>",
		"app.css":       "body{}",
		"assets/app.js": "console.log(1)",
		"notes.txt":     "hello",
		"config.conf":   "secret",
		"src/app.js":    "console.log(2)",
	}
	for name, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("建立目錄失敗: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("寫入檔案失敗: %v", err)
		}
	}

	handler := safeStaticFileServer(dir)
	cases := []struct {
		method      string
		path        string
		wantStatus  int
		wantCache   string
		wantNoCache bool
	}{
		{http.MethodGet, "/", http.StatusOK, "no-cache, must-revalidate", false},
		{http.MethodGet, "/app.css", http.StatusOK, "public, max-age=86400", false},
		{http.MethodGet, "/assets/app.js", http.StatusOK, "public, max-age=31536000, immutable", false},
		{http.MethodGet, "/app.css?v=1", http.StatusOK, "public, max-age=31536000, immutable", false},
		{http.MethodGet, "/notes.txt", http.StatusOK, "", true},
		{http.MethodGet, "/config.conf", http.StatusNotFound, "", true},
		{http.MethodGet, "/src/app.js", http.StatusNotFound, "", true},
		{http.MethodPost, "/", http.StatusOK, "", true},
	}
	for _, testCase := range cases {
		recorder := httptest.NewRecorder()
		// 模擬 session 續期中介層在 handler 之前寫入的 Set-Cookie。
		recorder.Header().Set("Set-Cookie", "forum_session=secret; Path=/; HttpOnly")
		handler.ServeHTTP(recorder, httptest.NewRequest(testCase.method, testCase.path, nil))
		if recorder.Code != testCase.wantStatus {
			t.Errorf("%s %s = %d, want %d", testCase.method, testCase.path, recorder.Code, testCase.wantStatus)
			continue
		}
		got := recorder.Header().Get("Cache-Control")
		if testCase.wantNoCache {
			if got != "" {
				t.Errorf("%s %s 不該設定 Cache-Control，got %q", testCase.method, testCase.path, got)
			}
			continue
		}
		if got != testCase.wantCache {
			t.Errorf("%s %s Cache-Control = %q, want %q", testCase.method, testCase.path, got, testCase.wantCache)
		}
		// public 回應會被共享快取保存，不能連 session token 一起存。
		if strings.HasPrefix(got, "public") {
			if cookie := recorder.Header().Get("Set-Cookie"); cookie != "" {
				t.Errorf("%s %s 是 public 回應，不該保留 Set-Cookie，got %q", testCase.method, testCase.path, cookie)
			}
		}
	}
}

// TestCovAdminUserAssetHandlersDropSessionCookie 釘住 /assets/ 與 /asset/ 的
// 靜態資產回應不帶 Set-Cookie。
//
// 這兩條路由不經過 safeStaticFileServer（見 server.go 的路由表），因此它們
// 的 cookie 清除是各自 handler 的責任；只測 safeStaticFileServer 會漏掉
// 真正承載建置資產的那一條。
func TestCovAdminUserAssetHandlersDropSessionCookie(t *testing.T) {
	t.Run("/assets/（帶 hash、immutable）", func(t *testing.T) {
		dir := t.TempDir()
		assets := filepath.Join(dir, "assets")
		if err := os.MkdirAll(assets, 0o755); err != nil {
			t.Fatalf("建立 assets 目錄失敗: %v", err)
		}
		if err := os.WriteFile(filepath.Join(assets, "app.js"), []byte("console.log(1)"), 0o644); err != nil {
			t.Fatalf("寫入 app.js 失敗: %v", err)
		}

		recorder := httptest.NewRecorder()
		recorder.Header().Set("Set-Cookie", "forum_session=secret; Path=/; HttpOnly")
		hashedAssetHandler(dir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET /assets/app.js = %d, want %d", recorder.Code, http.StatusOK)
		}
		if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
			t.Errorf("Cache-Control = %q, want public, max-age=31536000, immutable", got)
		}
		if cookie := recorder.Header().Get("Set-Cookie"); cookie != "" {
			t.Errorf("public 資產不該保留 Set-Cookie，got %q", cookie)
		}
	})

	t.Run("/asset/（一手素材）", func(t *testing.T) {
		parent := t.TempDir()
		frontendDir := filepath.Join(parent, "dist")
		assetDir := filepath.Join(parent, "asset")
		if err := os.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("建立 frontend 目錄失敗: %v", err)
		}
		if err := os.MkdirAll(assetDir, 0o755); err != nil {
			t.Fatalf("建立 asset 目錄失敗: %v", err)
		}
		if err := os.WriteFile(filepath.Join(assetDir, "logo.png"), []byte("png"), 0o644); err != nil {
			t.Fatalf("寫入 logo.png 失敗: %v", err)
		}

		recorder := httptest.NewRecorder()
		recorder.Header().Set("Set-Cookie", "forum_session=secret; Path=/; HttpOnly")
		sourceAssetHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/asset/logo.png", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET /asset/logo.png = %d, want %d", recorder.Code, http.StatusOK)
		}
		if cookie := recorder.Header().Get("Set-Cookie"); cookie != "" {
			t.Errorf("/asset/ 不該保留 Set-Cookie，got %q", cookie)
		}
	})
}

func TestCovAdminUserForumPages(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"forum.html", "forum-new.html", "forum-profile.html", "forum-others-profile.html",
		"forum-following.html", "forum-notifications.html", "forum-bookmarks.html", "forum-post.html",
	} {
		content := "<html><title>{{FORUM_NAME}}</title></html>"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("寫入 %s 失敗: %v", name, err)
		}
	}

	h := newCovHarness(t)
	handler := h.srv.handleForumPage(dir)

	cases := []struct {
		path       string
		wantStatus int
	}{
		{"/forum", http.StatusOK},
		{"/forum/", http.StatusOK},
		{"/forum/new", http.StatusOK},
		{"/forum/profile", http.StatusOK},
		{"/forum/others-profile", http.StatusOK},
		{"/forum/following", http.StatusOK},
		{"/forum/notifications", http.StatusOK},
		{"/forum/bookmarks", http.StatusOK},
		{"/forum/post/42", http.StatusOK},
		{"/forum/post/0", http.StatusNotFound},
		{"/forum/post/abc", http.StatusNotFound},
		{"/forum/unknown", http.StatusNotFound},
	}
	for _, testCase := range cases {
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, testCase.path, nil))
		if recorder.Code != testCase.wantStatus {
			t.Errorf("GET %s = %d, want %d", testCase.path, recorder.Code, testCase.wantStatus)
		}
	}

	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, "/forum", nil))
	covWantStatus(t, recorder, http.StatusOK)
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if !strings.Contains(recorder.Body.String(), "測試論壇") {
		t.Errorf("站名佔位符沒有被取代：%s", recorder.Body.String())
	}

	headRecorder := httptest.NewRecorder()
	handler(headRecorder, httptest.NewRequest(http.MethodHead, "/forum", nil))
	covWantStatus(t, headRecorder, http.StatusOK)
	if headRecorder.Body.Len() != 0 {
		t.Errorf("HEAD 回應不該有 body：%s", headRecorder.Body.String())
	}

	missingDir := t.TempDir()
	missingRecorder := httptest.NewRecorder()
	h.srv.handleForumPage(missingDir)(missingRecorder, httptest.NewRequest(http.MethodGet, "/forum", nil))
	covWantStatus(t, missingRecorder, http.StatusNotFound)
}
