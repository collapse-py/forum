/*
中介層與限流器的覆蓋測試（httpapi/cov_server_ratelimit_test.go）。

既有的測試多半直接呼叫 handler，因此 server.go 的中介層（requireLogin、
requireLoginForWrite、applyRateLimit）與 ratelimit.go 的到期清理、429 回應
一直沒有被執行到。這一檔補的正是那一層。
*/

package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCovServerRequireLoginRedirectsAnonymous(t *testing.T) {
	h := newCovHarness(t)
	called := false
	handler := h.srv.requireLogin(func(http.ResponseWriter, *http.Request) { called = true })

	recorder := h.do(handler, http.MethodGet, "/forum/profile?tab=posts", "", "", false)
	covWantStatus(t, recorder, http.StatusSeeOther)
	if called {
		t.Fatal("未登入時不該呼叫 next")
	}
	location := recorder.Header().Get("Location")
	if location == "" || location[:len("/forum/login?return=")] != "/forum/login?return=" {
		t.Fatalf("Location = %q", location)
	}
}

func TestCovServerRequireLoginAllowsNormalUser(t *testing.T) {
	h := newCovHarness(t)
	called := false
	handler := h.srv.requireLogin(func(http.ResponseWriter, *http.Request) { called = true })

	recorder := h.do(handler, http.MethodGet, "/forum/profile", "", "user@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	if !called {
		t.Fatal("正常使用者應通過")
	}
	// 沒有 forum_users 資料列 = 從未停權，不該因此被擋。
	if !h.sawQuery("SELECT status FROM forum_users") {
		t.Error("沒有查詢停權狀態")
	}
}

func TestCovServerRequireLoginRejectsSuspended(t *testing.T) {
	h := newCovHarness(t)
	h.rows("SELECT status FROM forum_users", []string{"status"}, []driverValue{"SUSPENDED"})
	handler := h.srv.requireLogin(func(http.ResponseWriter, *http.Request) {
		t.Fatal("停權帳號不該通過")
	})

	recorder := h.do(handler, http.MethodGet, "/forum/profile", "", "user@example.com", false)
	covWantStatus(t, recorder, http.StatusUnauthorized)
}

func TestCovServerRequireLoginDBError(t *testing.T) {
	h := newCovHarness(t)
	h.queryError("SELECT status FROM forum_users", errors.New("db down"))
	handler := h.srv.requireLogin(func(http.ResponseWriter, *http.Request) {
		t.Fatal("查詢失敗不該放行")
	})

	recorder := h.do(handler, http.MethodGet, "/forum/profile", "", "user@example.com", false)
	covWantStatus(t, recorder, http.StatusInternalServerError)
}

func TestCovServerRequireLoginAdminSkipsStatusLookup(t *testing.T) {
	h := newCovHarness(t)
	called := false
	handler := h.srv.requireLogin(func(http.ResponseWriter, *http.Request) { called = true })

	recorder := h.do(handler, http.MethodGet, "/forum/profile", "", "admin@example.com", true)
	covWantStatus(t, recorder, http.StatusOK)
	if !called {
		t.Fatal("管理員應通過")
	}
	if h.sawQuery("SELECT status FROM forum_users") {
		t.Error("管理員不該查停權狀態")
	}
}

func TestCovServerRequireLoginForWriteBypassesGet(t *testing.T) {
	h := newCovHarness(t)
	called := false
	handler := h.srv.requireLoginForWrite(func(http.ResponseWriter, *http.Request) { called = true })

	recorder := h.do(handler, http.MethodGet, "/api/forum/posts", "", "", false)
	covWantStatus(t, recorder, http.StatusOK)
	if !called {
		t.Fatal("GET 應直接放行")
	}
}

func TestCovServerRequireLoginForWriteChecksPost(t *testing.T) {
	h := newCovHarness(t)
	handler := h.srv.requireLoginForWrite(func(http.ResponseWriter, *http.Request) {
		t.Fatal("未登入的 POST 不該通過")
	})

	recorder := h.do(handler, http.MethodPost, "/api/forum/posts", `{}`, "", false)
	covWantStatus(t, recorder, http.StatusSeeOther)
}

func TestCovServerApplyRateLimitNilLimiter(t *testing.T) {
	h := newCovHarness(t)
	called := false
	handler := h.srv.applyRateLimit(nil, func(http.ResponseWriter, *http.Request) { called = true }, true)
	recorder := h.do(handler, http.MethodPost, "/api/forum/posts", `{}`, "user@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)
	if !called {
		t.Fatal("nil limiter 應直接放行")
	}
}

func TestCovServerApplyRateLimitSkipsGet(t *testing.T) {
	h := newCovHarness(t)
	limiter := NewRateLimiter(1, time.Minute)
	called := 0
	handler := h.srv.applyRateLimit(limiter, func(http.ResponseWriter, *http.Request) { called++ }, true)

	for i := 0; i < 3; i++ {
		recorder := h.do(handler, http.MethodGet, "/api/forum/posts", "", "user@example.com", false)
		covWantStatus(t, recorder, http.StatusOK)
	}
	if called != 3 {
		t.Fatalf("called = %d, want 3（GET 不受限流）", called)
	}
	if limiter.Stats().Allowed != 0 {
		t.Fatalf("GET 不該消耗額度：%+v", limiter.Stats())
	}
}

func TestCovServerApplyRateLimitBlocksPost(t *testing.T) {
	h := newCovHarness(t)
	limiter := NewRateLimiter(1, 2*time.Second)
	handler := h.srv.applyRateLimit(limiter, func(http.ResponseWriter, *http.Request) {}, true)

	first := h.do(handler, http.MethodPost, "/api/forum/posts", `{}`, "user@example.com", false)
	covWantStatus(t, first, http.StatusOK)

	second := h.do(handler, http.MethodPost, "/api/forum/posts", `{}`, "user@example.com", false)
	covWantStatus(t, second, http.StatusTooManyRequests)
	if second.Header().Get("Retry-After") == "" {
		t.Error("429 應帶 Retry-After")
	}
}

func TestCovServerRateLimitAllMethodsLimitsGet(t *testing.T) {
	h := newCovHarness(t)
	limiter := NewRateLimiter(1, time.Minute)
	handler := h.srv.rateLimitAllMethods(limiter, func(http.ResponseWriter, *http.Request) {})

	covWantStatus(t, h.do(handler, http.MethodGet, "/auth/google", "", "", false), http.StatusOK)
	covWantStatus(t, h.do(handler, http.MethodGet, "/auth/google", "", "", false), http.StatusTooManyRequests)
}

func TestCovServerRateLimiterSlidingWindow(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	if rl.Limit() != 2 || rl.Window() != time.Minute {
		t.Fatalf("Limit/Window = %d/%v", rl.Limit(), rl.Window())
	}
	now := time.Unix(1000, 0)
	rl.setClock(func() time.Time { return now })

	for i := 0; i < 2; i++ {
		allowed, retry := rl.Allow("ip")
		if !allowed || retry != 0 {
			t.Fatalf("第 %d 次：allowed=%v retry=%v", i+1, allowed, retry)
		}
	}
	allowed, retry := rl.Allow("ip")
	if allowed {
		t.Fatal("第三應被拒絕")
	}
	if retry != time.Minute {
		t.Fatalf("retry = %v, want 1m", retry)
	}
	if rl.TrackedKeys() != 1 {
		t.Fatalf("TrackedKeys = %d", rl.TrackedKeys())
	}
	stats := rl.Stats()
	if stats.Allowed != 2 || stats.Blocked != 1 {
		t.Fatalf("stats = %+v", stats)
	}

	// 視窗滑過後舊紀錄被剔除，重新可以通過。
	now = now.Add(time.Minute + time.Second)
	if allowed, _ := rl.Allow("ip"); !allowed {
		t.Fatal("視窗過後應放行")
	}
}

func TestCovServerRateLimiterCleanupRemovesIdleKeys(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	now := time.Unix(2000, 0)
	rl.setClock(func() time.Time { return now })
	rl.Allow("a")
	rl.Allow("b")
	if rl.TrackedKeys() != 2 {
		t.Fatalf("TrackedKeys = %d", rl.TrackedKeys())
	}

	now = now.Add(2 * time.Minute)
	if removed := rl.Cleanup(); removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if rl.TrackedKeys() != 0 {
		t.Fatalf("TrackedKeys = %d, want 0", rl.TrackedKeys())
	}
}

func TestCovServerRateLimiterDefaultsAndResolver(t *testing.T) {
	rl := NewRateLimiter(0, 0)
	if rl.Limit() != 10 || rl.Window() != time.Minute {
		t.Fatalf("預設值 = %d/%v", rl.Limit(), rl.Window())
	}
	rl.SetIPResolver(nil) // nil 應被忽略而不是 panic
	if rl.resolveIP == nil {
		t.Fatal("resolveIP 不該被清成 nil")
	}
	rl.SetIPResolver(func(*http.Request) string { return "custom" })
	if got := rl.resolveIP(httptest.NewRequest(http.MethodGet, "/", nil)); got != "custom" {
		t.Fatalf("resolveIP = %q", got)
	}
}

func TestCovServerRateLimiterMiddleware429(t *testing.T) {
	rl := NewRateLimiter(1, 1500*time.Millisecond)
	handler := rl.Middleware(func(http.ResponseWriter, *http.Request) {})
	request := httptest.NewRequest(http.MethodPost, "/api/forum/posts", nil)
	request.RemoteAddr = "203.0.113.9:12345"

	first := httptest.NewRecorder()
	handler(first, request)
	covWantStatus(t, first, http.StatusOK)

	second := httptest.NewRecorder()
	handler(second, request)
	covWantStatus(t, second, http.StatusTooManyRequests)
	if second.Header().Get("Retry-After") != "2" {
		t.Fatalf("Retry-After = %q, want 2（向上取整）", second.Header().Get("Retry-After"))
	}
}

func TestCovServerStartCleanupCancelledContext(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rl.StartCleanup(ctx, 0) // 已取消：直接返回，不啟動 goroutine

	// 未取消但立刻取消：涵蓋 ticker 路徑的建立與結束。
	ctx, cancel = context.WithCancel(context.Background())
	rl.StartCleanup(ctx, time.Millisecond)
	cancel()
}

func TestCovServerStartRateLimitCleanupNilSafe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	empty := &Server{}
	empty.StartRateLimitCleanup(ctx, time.Minute) // 三個 limiter 都是 nil

	h := newCovHarness(t)
	h.srv.StartRateLimitCleanup(ctx, time.Minute)
	if h.srv.Metrics() == nil || h.srv.Blocks() == nil {
		t.Fatal("NewServer 應建立 metrics 與 blocks")
	}
}

func TestCovServerSplitComma(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"a", 1},
		{"a,b", 2},
		{" a , , b ", 2},
		{"a,", 1},
		{",a", 1},
	}
	for _, tc := range cases {
		if got := splitComma(tc.in); len(got) != tc.want {
			t.Errorf("splitComma(%q) = %v, want %d 項", tc.in, got, tc.want)
		}
	}
}

func TestCovServerIsForumUserSuspended(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   bool
	}{
		{"停權", "SUSPENDED", true},
		{"正常", "ACTIVE", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newCovHarness(t)
			h.rows("SELECT status FROM forum_users", []string{"status"}, []driverValue{tc.status})
			got, err := h.srv.isForumUserSuspended(context.Background(), "user@example.com")
			if err != nil || got != tc.want {
				t.Fatalf("got=%v err=%v, want %v", got, err, tc.want)
			}
		})
	}

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT status FROM forum_users", errors.New("db down"))
		if _, err := h.srv.isForumUserSuspended(context.Background(), "user@example.com"); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestCovServerIsForumUserSuspendedNoRows(t *testing.T) {
	h := newCovHarness(t)
	got, err := h.srv.isForumUserSuspended(context.Background(), "nobody@example.com")
	if err != nil || got {
		t.Fatalf("got=%v err=%v（查無此人應視為未停權）", got, err)
	}
}
