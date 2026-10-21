/*
追蹤端點的覆蓋測試（httpapi/cov_forum_follow_test.go）。

這一份刻意用 cov_harness_test.go 的資料庫替身，把「有資料」與「沒資料」兩條
路徑都走過一次：既有的 forum_follow 測試只涵蓋未登入與路徑解析，成功的查詢、
交易、通知都落在需要 MySQL 的那一組裡。
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
)

func TestCovForumFollowsRequiresLogin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodGet, "/api/forum/follows", "", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovForumFollowsListEmpty(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodGet, "/api/forum/follows", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	body := recorder.Body.String()
	for _, want := range []string{`"items":[]`, `"hasMore":false`, `"selfKey":"` + publicForumKey("me@example.com") + `"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body 缺少 %q：%s", want, body)
		}
	}
	if !h.sawQuery("FROM forum_follows f") {
		t.Error("沒有執行追蹤清單查詢")
	}
}

func TestCovForumFollowsListWithAvatar(t *testing.T) {
	h := newCovHarness(t)
	h.rows("FROM forum_follows f",
		[]string{"target_email", "nickname", "avatar_url", "created_at"},
		[]driverValue{"target@example.com", "目標", "avatar.png", time.Now()},
	)
	recorder := h.do(h.srv.handleForumFollows, http.MethodGet, "/api/forum/follows", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"nickname":"目標"`) {
		t.Errorf("body = %s", recorder.Body.String())
	}
}

func TestCovForumFollowsListRejectsBadOffset(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodGet, "/api/forum/follows?offset=-1", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovForumFollowsMethodNotAllowed(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodPut, "/api/forum/follows", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusMethodNotAllowed)
}

func TestCovForumFollowsQueryError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("FROM forum_follows f", errors.New("db down"))
	recorder := h.do(h.srv.handleForumFollows, http.MethodGet, "/api/forum/follows", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovToggleForumFollowAdds(t *testing.T) {
	h := newCovHarness(t)
	key := strings.Repeat("a", 64)
	h.rows("FROM forum_profiles WHERE public_key",
		[]string{"author_email"},
		[]driverValue{"target@example.com"},
	)
	h.rows("SELECT COUNT(*) FROM forum_follows",
		[]string{"count"},
		[]driverValue{int64(0)},
	)

	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":"`+key+`"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"following":true`) {
		t.Errorf("body = %s", recorder.Body.String())
	}
	if !h.sawQuery("INSERT INTO forum_follows") {
		t.Error("沒有執行新增追蹤")
	}
	if !h.sawQuery("INSERT INTO forum_notifications") {
		t.Error("新增追蹤後沒有建立通知")
	}
}

func TestCovToggleForumFollowRemoves(t *testing.T) {
	h := newCovHarness(t)
	key := strings.Repeat("b", 64)
	h.rows("FROM forum_profiles WHERE public_key",
		[]string{"author_email"},
		[]driverValue{"target@example.com"},
	)
	h.rows("SELECT COUNT(*) FROM forum_follows",
		[]string{"count"},
		[]driverValue{int64(1)},
	)

	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":"`+key+`"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"following":false`) {
		t.Errorf("body = %s", recorder.Body.String())
	}
	if !h.sawQuery("DELETE FROM forum_follows") {
		t.Error("沒有執行取消追蹤")
	}
}

func TestCovToggleForumFollowBadKey(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":"too-short"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	if !strings.Contains(recorder.Body.String(), "invalid profile key") {
		t.Errorf("body = %s", recorder.Body.String())
	}
}

func TestCovToggleForumFollowSelf(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":"`+publicForumKey("me@example.com")+`"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
	if !strings.Contains(recorder.Body.String(), "cannot follow yourself") {
		t.Errorf("body = %s", recorder.Body.String())
	}
}

func TestCovToggleForumFollowNotFound(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":"`+strings.Repeat("c", 64)+`"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusNotFound)
}

func TestCovToggleForumFollowLookupFailure(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("FROM forum_profiles WHERE public_key", errors.New("db down"))
	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":"`+strings.Repeat("d", 64)+`"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovToggleForumFollowRejectsInvalidJSON(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusBadRequest)
}

func TestCovToggleForumFollowRejectsUntrustedOrigin(t *testing.T) {
	h := newCovHarness(t)
	request := httptest.NewRequest(http.MethodPost, "/api/forum/follows", strings.NewReader(`{"key":"x"}`))
	request.Header.Set("Origin", "https://evil.example")
	request.AddCookie(h.login("me@example.com", false))
	recorder := httptest.NewRecorder()
	h.srv.handleForumFollows(recorder, request)
	covWantStatus(t, recorder, http.StatusForbidden)
}

func TestCovForumEmailByPublicKeyFallsBackToPosts(t *testing.T) {
	h := newCovHarness(t)
	key := publicForumKey("target@example.com")
	h.rows("SELECT DISTINCT author_email FROM forum_posts",
		[]string{"author_email"},
		[]driverValue{"someone@example.com"},
		[]driverValue{"target@example.com"},
	)
	email, err := h.srv.forumEmailByPublicKey(context.Background(), key)
	if err != nil {
		t.Fatalf("forumEmailByPublicKey: %v", err)
	}
	if email != "target@example.com" {
		t.Fatalf("email = %q", email)
	}
}

func TestCovForumEmailByPublicKeyFallbackErrors(t *testing.T) {
	t.Run("掃描失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT DISTINCT author_email FROM forum_posts", errors.New("db down"))
		if _, err := h.srv.forumEmailByPublicKey(context.Background(), strings.Repeat("e", 64)); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("查無此人", func(t *testing.T) {
		h := newCovHarness(t)
		if _, err := h.srv.forumEmailByPublicKey(context.Background(), strings.Repeat("f", 64)); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

func TestCovForumFollowingPostsRequiresLogin(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollowingPosts, http.MethodGet, "/api/forum/following/posts", "", "", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovForumFollowingPostsEmpty(t *testing.T) {
	h := newCovHarness(t)
	recorder := h.do(h.srv.handleForumFollowingPosts, http.MethodGet, "/api/forum/following/posts", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"items":[]`) {
		t.Errorf("body = %s", recorder.Body.String())
	}
}

func TestCovForumFollowingPostsLoadError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT", errors.New("db down"))
	recorder := h.do(h.srv.handleForumFollowingPosts, http.MethodGet, "/api/forum/following/posts", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}
