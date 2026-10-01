/*
監控接線層的測試（httpapi/monitoring_test.go）。

這一組測的是「統計有沒有被正確接上」，不是統計本身（那是 metrics 套件的
工作）。真正會在部署後出事、而這裡擋得住的三件事：

  1. metricsMiddleware 有沒有真的記到路由與狀態碼，以及 inFlight 有沒有在
     handler 結束後收得回來。寫錯（例如忘了 End）不會讓任何請求失敗，症狀
     是儀表板上的「進行中請求」慢慢只增不減 —— 那種 bug 不會在整合測試裡
     浮現。
  2. 限流器的允許／阻擋計數是否與實際判定一致。計數寫在 Allow 的哪一個分支，
     決定了「被拒絕的請求」會不會被算成允許。
  3. /api/admin/monitor 未登入時必須 401，且授權判斷排在 method 分派之前。
     這個端點會回傳連線池水位、Redis 鍵數與執行期統計，任何人可讀等同把內部
     拓撲公開；順序反過來則可用一個不支援的 method 拿到 405，等於洩漏
     「這條路由存在且需要管理員身分」。

刻意沒有測的：handleAdminMonitor 回 200 的路徑。那需要一個真的 *sql.DB
（probeDatabase 會呼叫 PingContext 與 Stats），而這個專案目前的測試策略是
以假伺服器／假 Redis 取代真實依賴，不含 MySQL 替身。為了測一條 happy path
而引入 sqlmock 與它的相依，不划算 —— 那一段的價值在「欄位有沒有漏」，
而欄位清單由 metrics.Snapshot 的 JSON 標籤決定，同樣沒有測試手段能證明
它與前端契約一致。

狀態隔離：與同套件其他測試一致，每個測試自行建立 Server，不使用 package
層級共用變數。
*/

package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forum/forum/metrics"
	"forum/forum/session"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

/* ==========================================================================
   metricsMiddleware
   ========================================================================== */

func TestMetricsMiddlewareRecordsRouteAndStatus(t *testing.T) {
	server := &Server{metrics: metrics.New(metrics.Options{})}
	inner := server.metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	inner.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/forum/posts/42", nil))

	snapshot := server.metrics.Snapshot(0)
	if snapshot.Requests.Total != 1 {
		t.Fatalf("total = %d, want 1", snapshot.Requests.Total)
	}
	if len(snapshot.Requests.Routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(snapshot.Requests.Routes))
	}
	// 路徑必須被正規化：/42 與 /43 是同一條路由，否則攻擊者亂數測 id 就會把
	// map 撐大（metrics 套件檔頭的說明）。
	if route := snapshot.Requests.Routes[0].Route; route != "/api/forum/posts/:id" {
		t.Errorf("route = %q, want /api/forum/posts/:id", route)
	}
	// 418 是 4xx，因此必須被分類為用戶端錯誤。
	if snapshot.Requests.ClientErrors != 1 {
		t.Errorf("clientErrors = %d, want 1（418 屬於 4xx）", snapshot.Requests.ClientErrors)
	}
}

// handler 只寫 body 而不呼叫 WriteHeader 時，狀態碼必須被算成 200。
// 這個 case 會在「StatusWriter 忘了預填」時失敗，而症狀是每個正常請求都被
// 記成 4xx（未初始化的欄位是 0，不是任何合法狀態碼）。
func TestMetricsMiddlewareDefaultsToOK(t *testing.T) {
	server := &Server{metrics: metrics.New(metrics.Options{})}
	inner := server.metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	inner.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	snapshot := server.metrics.Snapshot(0)
	if snapshot.Requests.ClientErrors != 0 || snapshot.Requests.ServerErrors != 0 {
		t.Fatalf("只寫 body 時錯誤數應為 0, got client=%d server=%d",
			snapshot.Requests.ClientErrors, snapshot.Requests.ServerErrors)
	}
}

func TestMetricsMiddlewareReleasesInFlight(t *testing.T) {
	server := &Server{metrics: metrics.New(metrics.Options{})}
	var inFlightDuringRequest int64
	inner := server.metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlightDuringRequest = server.metrics.Snapshot(0).Requests.InFlight
	}))

	inner.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if inFlightDuringRequest != 1 {
		t.Errorf("handler 執行期間 inFlight = %d, want 1", inFlightDuringRequest)
	}
	if got := server.metrics.Snapshot(0).Requests.InFlight; got != 0 {
		t.Errorf("handler 結束後 inFlight = %d, want 0（End 沒被呼叫）", got)
	}
}

// metrics 為 nil 時中介層必須原樣放行，否則以 struct literal 構造 Server 的
// 測試（本站既有測試的寫法）會在建立路由時就 panic。
func TestMetricsMiddlewareIsNilSafe(t *testing.T) {
	server := &Server{}
	inner := server.metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	inner.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", recorder.Code)
	}
}

// 併發觀測：這支測試的主要價值在 `go test -race`（需要 CGO）。若 Observe 與
// Snapshot 之間少了鎖，-race 會在這裡報出資料競爭。
func TestMetricsMiddlewareHandlesConcurrentRequests(t *testing.T) {
	server := &Server{metrics: metrics.New(metrics.Options{})}
	inner := server.metricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	const goroutines, perGoroutine = 8, 50
	done := make(chan struct{})
	for g := range goroutines {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			for i := range perGoroutine {
				inner.ServeHTTP(httptest.NewRecorder(),
					httptest.NewRequest(http.MethodGet, "/api/forum/posts/"+strconvItoa(id*perGoroutine+i), nil))
			}
		}(g)
	}
	for range goroutines {
		<-done
	}

	snapshot := server.metrics.Snapshot(0)
	if want := int64(goroutines * perGoroutine); snapshot.Requests.Total != want {
		t.Fatalf("total = %d, want %d", snapshot.Requests.Total, want)
	}
	if snapshot.Requests.InFlight != 0 {
		t.Fatalf("inFlight = %d, want 0", snapshot.Requests.InFlight)
	}
}

/* ==========================================================================
   限流器計數
   ========================================================================== */

func TestRateLimiterCountsAllowedAndBlocked(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	// 前兩次在額度內，第三次被拒。
	for i, want := range []bool{true, true, false} {
		allowed, _ := limiter.Allow("1.2.3.4")
		if allowed != want {
			t.Fatalf("第 %d 次 Allow = %v, want %v", i+1, allowed, want)
		}
	}

	stats := limiter.Stats()
	if stats.Allowed != 2 {
		t.Errorf("allowed = %d, want 2", stats.Allowed)
	}
	if stats.Blocked != 1 {
		t.Errorf("blocked = %d, want 1", stats.Blocked)
	}
	if got := limiter.TrackedKeys(); got != 1 {
		t.Errorf("trackedKeys = %d, want 1", got)
	}
	if limiter.Limit() != 2 || limiter.Window() != time.Minute {
		t.Errorf("limit/window = %d/%v, want 2/1m", limiter.Limit(), limiter.Window())
	}
}

// 被拒絕的請求不得延長封鎖時間（Allow 原本就不把它記進 hits）。若計數寫在
// 「剔除之後、判定之前」的位置，這個行為會被順手破壞：把被拒的請求也記進
// hits 會讓它自己把自己鎖得更久。這個測試守住「加上計數沒有改變既有限流
// 語意」這件事。
func TestRateLimiterCountingDoesNotChangeBlocking(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	base := time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)
	current := base
	limiter.setClock(func() time.Time { return current })

	for range 2 {
		if allowed, _ := limiter.Allow("ip"); !allowed {
			t.Fatal("前兩次應放行")
		}
	}
	for range 3 {
		if allowed, _ := limiter.Allow("ip"); allowed {
			t.Fatal("超出額度後應持續拒絕")
		}
	}

	// 推進 61 秒：第一筆記錄滑出視窗，此時必須重新放行。
	current = base.Add(61 * time.Second)
	if allowed, _ := limiter.Allow("ip"); !allowed {
		t.Fatal("視窗滑過後應放行；被拒絕的請求若被記進 hits 就會失敗")
	}
	if stats := limiter.Stats(); stats.Blocked != 3 {
		t.Errorf("blocked = %d, want 3", stats.Blocked)
	}
}

func TestLimitStatsSkipsMissingLimiters(t *testing.T) {
	server := &Server{writeRateLimiter: NewRateLimiter(10, time.Minute)}

	stats := server.limitStats()
	if len(stats) != 1 {
		t.Fatalf("limitStats() = %d 筆, want 1（只有 content 被建立）", len(stats))
	}
	if stats[0].Name != "content" || stats[0].Limit != 10 {
		t.Errorf("stats[0] = %+v, want name=content limit=10", stats[0])
	}
}

/* ==========================================================================
   /api/admin/monitor 的授權
   ========================================================================== */

// newMonitorTestServer 以 miniredis 建立帶真實 session.Manager 的 Server。
// 之所以需要真的 session：IsAdmin 會讀 cookie 並查 Redis 的 hash，用 nil
// 取代只會得到一個 panic 而不是「未登入」。
func newMonitorTestServer(t *testing.T) *Server {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &Server{
		sessions: session.NewManager("FORUM_test", time.Hour, false, client),
		metrics:  metrics.New(metrics.Options{}),
	}
}

func TestAdminMonitorRequiresAdmin(t *testing.T) {
	server := newMonitorTestServer(t)

	// 兩種方法都必須 401。POST 特別重要：若 method 分派排在授權檢查之前，
	// 它會回 405，那等於告訴未登入者「這條路由存在、而且不接受 POST」。
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		server.handleAdminMonitor(recorder, httptest.NewRequest(method, "/api/admin/monitor", nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s 的回應 = %d, want 401", method, recorder.Code)
		}
	}
}

// 帶著「已登入但不是管理員」的 session 必須仍然是 401。這個 case 與上面的
// 「完全沒有 cookie」不同：前者要證明 is_admin 欄位真的被檢查，而不是只看
// 有沒有 session。
func TestAdminMonitorRejectsNonAdminSession(t *testing.T) {
	server := newMonitorTestServer(t)
	token, err := server.sessions.Create("user@example.com", false)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/admin/monitor", nil)
	request.AddCookie(&http.Cookie{Name: "FORUM_test", Value: token})

	recorder := httptest.NewRecorder()
	server.handleAdminMonitor(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("回應 = %d, want 401（非管理員即使已登入）", recorder.Code)
	}
}

/* ==========================================================================
   依賴探測
   ========================================================================== */

// ES 未設定時必須回 disabled 而不是 down —— 這兩者對維運的意義完全不同
// （見 health.go 的說明），而它不需要資料庫，因此可以在這裡直接測。
func TestProbeSearchReportsDisabled(t *testing.T) {
	server := &Server{}
	status := server.probeSearch(t.Context())
	if status.State != "disabled" {
		t.Fatalf("state = %q, want disabled", status.State)
	}
	if status.Detail["engine"] != "mysql" {
		t.Errorf("engine = %v, want mysql（未啟用 ES 時搜尋走 MySQL LIKE）", status.Detail["engine"])
	}
}

func TestProbeRedisReportsDisabledWithoutClient(t *testing.T) {
	server := &Server{}
	status := server.probeRedis(t.Context())
	if status.State != "disabled" {
		t.Fatalf("state = %q, want disabled（mediaRedis 為 nil 不等於故障）", status.State)
	}
}

/* ==========================================================================
   Redis INFO 解析
   ========================================================================== */

// fakeRedisCommander 以預先寫好的 INFO 文字驅動 used_memory 的解析。
// 宣告成介面（見 redisCommander）就是為了讓這件事不必真的連 Redis。
type fakeRedisCommander struct {
	raw string
	err error
}

func (f fakeRedisCommander) Info(_ context.Context, _ ...string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	if f.err != nil {
		cmd.SetErr(f.err)
		return cmd
	}
	cmd.SetVal(f.raw)
	return cmd
}

func TestRedisMemoryBytes(t *testing.T) {
	// 真實的 INFO memory 回應形狀：開頭有註解行，欄位以 \r\n 分隔，
	// used_memory 不是第一行也不是最後一行。
	realistic := "# Memory\r\nused_memory:1048576\r\nused_memory_human:1.00M\r\nmaxmemory:0\r\n"

	cases := []struct {
		name  string
		input fakeRedisCommander
		want  int64
		ok    bool
	}{
		{"正常回應", fakeRedisCommander{raw: realistic}, 1048576, true},
		{"只有註解行", fakeRedisCommander{raw: "# Memory\r\n"}, 0, false},
		{"空回應", fakeRedisCommander{raw: ""}, 0, false},
		{"欄位不存在", fakeRedisCommander{raw: "# Memory\r\nmaxmemory:0\r\n"}, 0, false},
		{"值不是數字", fakeRedisCommander{raw: "used_memory:not-a-number\r\n"}, 0, false},
		{"指令失敗（ACL 不允許 INFO）", fakeRedisCommander{err: errors.New("NOPERM")}, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := redisMemoryBytes(t.Context(), tc.input)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("redisMemoryBytes() = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

// strconvItoa 是 strconv.Itoa 的本地別名。這個檔案只需要非負整數，少一個
// import 就少一個與被測行為無關的符號。
func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
