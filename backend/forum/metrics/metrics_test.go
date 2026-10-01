package metrics

import (
	"testing"
	"time"
)

/*
測試策略

這個套件沒有 I/O 之外的依賴，也沒有需要整合測試的對外互動，因此全部測試都
用「注入假時鐘 + 直接斷言快照」的方式寫。刻意不測的兩件事：

  - 資料庫落地（FlushPending / LoadHistory）：那需要真的 MySQL，而這個專案
    目前的測試策略是 httpapi 與 es 都以 httptest 假伺服器取代真實依賴。
    這裡若引入 miniredis 式的 MySQL 替身（sqlmock）會多一個不相稱的相依。
    取而代之的是把寫入語意（累加、不重複寫、只寫已結束的分鐘）寫在函式的
    註解裡，並讓 FlushPending 對 nil 資料庫安全返回，讓呼叫端不會 panic。
  - metricsMiddleware 的併發正確性：那屬於 httpapi 套件，而這個套件的函式
    本身全部在 mu 之下，-race 會直接驗證。
*/

// fixedClock 是一個可手動推進的時鐘。metrics.Options.Now 存在的唯一目的
// 就是讓測試能確定性地控制「目前是第幾分鐘」—— 時間軸的正確性完全取決於
// 分鐘邊界，用真實時鐘就只能靠 sleep 去等。
type fixedClock struct{ current time.Time }

func (c *fixedClock) now() time.Time { return c.current }

func (c *fixedClock) advance(d time.Duration) { c.current = c.current.Add(d) }

func newTestRegistry(t *testing.T, windowMinutes int) (*Registry, *fixedClock) {
	t.Helper()
	clock := &fixedClock{current: time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)}
	registry := New(Options{WindowMinutes: windowMinutes, MaxRoutes: 8, Now: clock.now})
	return registry, clock
}

/* ==========================================================================
   NormalizeRoute
   ========================================================================== */

func TestNormalizeRoute(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{"根路徑", "/", "/"},
		{"空字串視為根路徑", "", "/"},
		{"純數字片段換成 id", "/api/forum/posts/17/comments", "/api/forum/posts/:id/comments"},
		{"多個 id 片段", "/api/forum/posts/17/comments/9182/report", "/api/forum/posts/:id/comments/:id/report"},
		{"固定樣式原樣保留", "/api/forum/posts/17/like", "/api/forum/posts/:id/like"},
		{"沒有 id 的路由", "/api/forum/following/posts", "/api/forum/following/posts"},
		{"email 參數視為 id", "/api/admin/users/foo@bar.example/tags", "/api/admin/users/:id/tags"},
		{"建置資產收成一段", "/assets/index-a1b2c3d4.js", "/assets/*"},
		{"PWA 圖示收成一段", "/asset/logo.png", "/asset/*"},
		{"catch-all 靜態檔", "/robots.txt", "/*"},
		{"查詢字串不影響路由", "/api/forum/search?q=hello%20world", "/api/forum/search"},
		{"尾端斜線不產生空片段", "/api/forum/posts/", "/api/forum/posts"},
		{"非數字片段不當成 id", "/api/forum/posts/abc", "/api/forum/posts/abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeRoute(tc.path); got != tc.want {
				t.Fatalf("NormalizeRoute(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

// NormalizeRoute 存在的唯一理由是讓 map 的 key 數量維持在低基數。這支測試把
// 「一千個不同 id 只產生一條路由」這個性質寫成斷言，因為它一旦失效，
// 症狀是記憶體慢慢長大而不是任何明確的錯誤。
func TestNormalizeRouteKeepsCardinalityLow(t *testing.T) {
	distinct := make(map[string]struct{})
	for id := 1; id <= 1000; id++ {
		distinct[NormalizeRoute("/api/forum/posts/"+itoa(id)+"/comments")] = struct{}{}
	}
	if len(distinct) != 1 {
		t.Fatalf("1000 個不同 id 產生 %d 條路由, want 1", len(distinct))
	}
}

/* ==========================================================================
   Observe 與快照
   ========================================================================== */

func TestObserveClassifiesStatusCodes(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	registry.Observe("GET", "/api/forum/posts", 200, 5*time.Millisecond)
	registry.Observe("GET", "/api/forum/posts", 404, 3*time.Millisecond)
	registry.Observe("POST", "/api/forum/posts", 429, 1*time.Millisecond)
	registry.Observe("GET", "/api/forum/posts", 500, 20*time.Millisecond)

	snapshot := registry.Snapshot(24)
	if snapshot.Requests.Total != 4 {
		t.Fatalf("total = %d, want 4", snapshot.Requests.Total)
	}
	if snapshot.Requests.ClientErrors != 2 {
		t.Errorf("clientErrors = %d, want 2（404 與 429 都算 4xx）", snapshot.Requests.ClientErrors)
	}
	if snapshot.Requests.ServerErrors != 1 {
		t.Errorf("serverErrors = %d, want 1", snapshot.Requests.ServerErrors)
	}
}

func TestObserveAccumulatesPerRoute(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	for range 3 {
		registry.Observe("GET", "/api/forum/posts/1", 200, 2*time.Millisecond)
	}
	registry.Observe("POST", "/api/forum/posts", 201, 8*time.Millisecond)

	snapshot := registry.Snapshot(24)
	// 同一條路由的不同 id 必須合併成一列 —— 這是 NormalizeRoute 的第一個目的。
	if len(snapshot.Requests.Routes) != 2 {
		t.Fatalf("routes = %d, want 2", len(snapshot.Requests.Routes))
	}
	top := snapshot.Requests.Routes[0]
	if top.Route != "/api/forum/posts/:id" || top.Total != 3 {
		t.Fatalf("top route = %+v, want /api/forum/posts/:id with 3 requests", top)
	}
	if top.AvgMS != 2 {
		t.Errorf("avg = %v ms, want 2", top.AvgMS)
	}
}

func TestSnapshotSortsRoutesByCount(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	registry.Observe("GET", "/a", 200, time.Millisecond)
	for range 5 {
		registry.Observe("GET", "/b", 200, time.Millisecond)
	}
	registry.Observe("GET", "/c", 200, time.Millisecond)

	routes := registry.Snapshot(24).Requests.Routes
	if len(routes) != 3 || routes[0].Route != "/b" {
		t.Fatalf("routes = %+v, want /b first", routes)
	}
	// 數量相同時以路由字串排序，讓自動刷新不會讓表格整列跳動。
	if routes[0].Route != "/a" && routes[1].Route != "/a" {
		t.Errorf("相同數量時未依路由字串排序: %+v", routes)
	}
}

// maxRoutes 生效之後，新路由必須被併進 __other__ 而不是被丟掉。
// 丟掉會讓總數對不上（儀表板上最嚴重的錯誤類型：數字看起來合理但偏小）。
func TestObserveCapsRouteCardinality(t *testing.T) {
	registry, _ := newTestRegistry(t, 30) // MaxRoutes 為 8
	// 前八條路由各一筆，第九條起應該被併進 other。
	paths := []string{"/a", "/b", "/c", "/d", "/e", "/f", "/g", "/h", "/i", "/j"}
	for _, path := range paths {
		registry.Observe("GET", path, 200, time.Millisecond)
	}

	snapshot := registry.Snapshot(24)
	if snapshot.Requests.Total != int64(len(paths)) {
		t.Fatalf("total = %d, want %d（被併掉的路由仍要計數）", snapshot.Requests.Total, len(paths))
	}

	var other int64
	distinct := 0
	for _, route := range snapshot.Requests.Routes {
		if route.Route == otherRoute {
			other = route.Total
		}
		distinct++
	}
	if distinct != 9 {
		t.Errorf("distinct routes = %d, want 9（8 條加上 __other__）", distinct)
	}
	if other != 2 {
		t.Errorf("__other__ total = %d, want 2", other)
	}
}

/* ==========================================================================
   延遲直方圖
   ========================================================================== */

func TestQuantilesAreBucketUpperBounds(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	// 九次 1ms、一次 800ms。p50 應落在 1ms 那一格、p95 應落在 500ms 那一格
	// （累積到 500ms 的有九次，累積到 1s 的有十次；target = ceil(10*0.95) = 10，
	// 因此 p95 落在最後一格 = 1s）。
	for range 9 {
		registry.Observe("GET", "/slow", 200, time.Millisecond)
	}
	registry.Observe("GET", "/slow", 200, 800*time.Millisecond)

	snapshot := registry.Snapshot(24)
	route := snapshot.Requests.Routes[0]
	if route.P50MS != 1 {
		t.Errorf("p50 = %v ms, want 1", route.P50MS)
	}
	if route.P95MS != 1000 {
		t.Errorf("p95 = %v ms, want 1000（最後一格的上界）", route.P95MS)
	}
	if route.MaxMS != 800 {
		t.Errorf("max = %v ms, want 800（極值是精確值，不分桶）", route.MaxMS)
	}
	if route.AvgMS != 80.9 {
		t.Errorf("avg = %v ms, want 80.9", route.AvgMS)
	}
}

// 沒有任何請求時所有統計都必須是 0，而不是 NaN —— NaN 會讓 JSON 序列化直接
// 失敗（encoding/json 不接受 NaN），整個監控端點就會回 500。
func TestSnapshotOnEmptyRegistry(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	snapshot := registry.Snapshot(24)

	if snapshot.Requests.Total != 0 {
		t.Errorf("total = %d, want 0", snapshot.Requests.Total)
	}
	if snapshot.Requests.AvgDurationMS != 0 || snapshot.Requests.P95MS != 0 {
		t.Errorf("空資料時統計應為 0, got avg=%v p95=%v", snapshot.Requests.AvgDurationMS, snapshot.Requests.P95MS)
	}
	if snapshot.Runtime.LastPauseMS != 0 {
		t.Errorf("尚未發生 GC 時 lastPause 應為 0, got %v", snapshot.Runtime.LastPauseMS)
	}
}

/* ==========================================================================
   併發計數
   ========================================================================== */

func TestBeginEndTracksInFlight(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	registry.Begin()
	registry.Begin()
	registry.Begin()
	if got := registry.Snapshot(24).Requests.InFlight; got != 3 {
		t.Fatalf("inFlight = %d, want 3", got)
	}
	registry.End()
	snapshot := registry.Snapshot(24)
	if snapshot.Requests.InFlight != 2 {
		t.Errorf("inFlight = %d, want 2", snapshot.Requests.InFlight)
	}
	if snapshot.Requests.MaxInFlight != 3 {
		t.Errorf("maxInFlight = %d, want 3（峰值不會因為 End 而下降）", snapshot.Requests.MaxInFlight)
	}
}

// 併發觀測：這支測試的主要價值在於讓 `go test -race` 有東西可查。
// 若 Observe 與 Snapshot 之間少了 mu，`-race` 會在這裡報出資料競爭。
func TestConcurrentObserveAndSnapshot(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	const goroutines = 8
	const perGoroutine = 200

	done := make(chan struct{})
	for g := range goroutines {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			for i := range perGoroutine {
				registry.Observe("GET", "/api/forum/posts/"+itoa(id*perGoroutine+i), 200, time.Millisecond)
			}
		}(g)
	}
	for range goroutines {
		<-done
	}

	snapshot := registry.Snapshot(24)
	if want := int64(goroutines * perGoroutine); snapshot.Requests.Total != want {
		t.Fatalf("total = %d, want %d", snapshot.Requests.Total, want)
	}
	if snapshot.Requests.MaxInFlight != 0 {
		t.Errorf("maxInFlight = %d, want 0（這支測試沒有呼叫 Begin）", snapshot.Requests.MaxInFlight)
	}
}

/* ==========================================================================
   時間軸
   ========================================================================== */

func TestTimelineIsContinuous(t *testing.T) {
	registry, clock := newTestRegistry(t, 5)
	clock.advance(90 * time.Second)                      // 15:01:30
	registry.Observe("GET", "/a", 200, time.Millisecond) // 15:01:30
	registry.Observe("GET", "/a", 200, time.Millisecond) // 15:01:30
	clock.advance(2 * time.Minute)                       // 15:03:30
	registry.Observe("GET", "/a", 200, time.Millisecond) // 15:03:30

	points := registry.Snapshot(24).Timeline
	if len(points) != 5 {
		t.Fatalf("timeline length = %d, want 5", len(points))
	}
	// 連續性：相鄰兩點剛好差一分鐘，且最後一點是當前分鐘。
	for i := 1; i < len(points); i++ {
		prev, err := time.Parse(time.RFC3339, points[i-1].Minute)
		if err != nil {
			t.Fatalf("parse minute: %v", err)
		}
		cur, err := time.Parse(time.RFC3339, points[i].Minute)
		if err != nil {
			t.Fatalf("parse minute: %v", err)
		}
		if cur.Sub(prev) != time.Minute {
			t.Fatalf("點 %d 與 %d 之間隔了 %v, want 1m", i-1, i, cur.Sub(prev))
		}
	}
	// 有資料的分鐘總量正確，中間沒有資料的分鐘是 0 且 source 為空。
	// 視窗 5 格涵蓋 14:59 到 15:03，因此 15:01 是第三格（索引 2）。
	if points[2].Total != 2 || points[2].Source != "live" {
		t.Errorf("15:01 那格 = %+v, want 2 requests / live", points[2])
	}
	if points[3].Total != 0 || points[3].Source != "" {
		t.Errorf("沒有資料的分鐘應為 0 且無來源, got %+v", points[3])
	}
}

func TestTimelineReportsHistorySource(t *testing.T) {
	registry, clock := newTestRegistry(t, 5)
	clock.advance(10 * time.Minute) // 15:10:00
	// 直接注入一個歷史桶模擬「重啟前從資料庫讀回來的資料」。
	historyMinute := minuteIndex(clock.current.Add(-3 * time.Minute))
	registry.history[historyMinute] = &minuteBucket{minute: historyMinute, total: 42, clientErrors: 2}

	points := registry.Snapshot(24).Timeline
	found := false
	for _, point := range points {
		if point.Minute != minuteTime(historyMinute).Format(time.RFC3339) {
			continue
		}
		found = true
		if point.Source != "history" || point.Total != 42 {
			t.Errorf("歷史分鐘 = %+v, want source=history total=42", point)
		}
	}
	if !found {
		t.Fatal("時間軸上找不到注入的歷史分鐘")
	}
	if !registry.Snapshot(24).HistoryLoaded {
		t.Error("HistoryLoaded 應為 true")
	}
}

// 時間軸長度受 maxTimelinePoints 收斂：回傳比畫面能呈現更多的點只會讓回應變大。
func TestTimelineLengthIsCapped(t *testing.T) {
	registry, _ := newTestRegistry(t, defaultWindowMinutes) // 120 > 144? 否，120 < 144
	if got := len(registry.Snapshot(24).Timeline); got != defaultWindowMinutes {
		t.Fatalf("timeline length = %d, want %d", got, defaultWindowMinutes)
	}

	wide, _ := newTestRegistry(t, 1000) // > maxTimelinePoints
	if got := len(wide.Snapshot(24).Timeline); got != maxTimelinePoints {
		t.Fatalf("timeline length = %d, want %d（應被 maxTimelinePoints 收斂）", got, maxTimelinePoints)
	}
}

/* ==========================================================================
   持久化的守恆條件
   ========================================================================== */

func TestFlushPendingRequiresDatabase(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	if got := registry.FlushPending(t.Context(), nil); got != 0 {
		t.Errorf("nil 資料庫時 FlushPending 應回 0, got %d", got)
	}
	if err := registry.StartFlusher(t.Context(), FlusherOptions{}); err == nil {
		t.Error("沒有 *sql.DB 時 StartFlusher 應回傳錯誤")
	}
}

// 已結束的分鐘在寫出之後就不可變，因此把它從記憶體視窗外剪掉不會讓時間軸
// 少掉任何一格（那些分鐘在時間軸範圍內，而剪輯條件是「早於視窗」）。
// 這支測試守住的是「剪輯不會誤傷視窗內的桶」。
func TestPruneKeepsBucketsInsideWindow(t *testing.T) {
	registry, clock := newTestRegistry(t, 10)
	registry.Observe("GET", "/a", 200, time.Millisecond)
	clock.advance(5 * time.Minute)
	registry.Observe("GET", "/a", 200, time.Millisecond)
	clock.advance(20 * time.Minute) // 遠遠超出 10 分鐘視窗
	registry.Observe("GET", "/a", 200, time.Millisecond)
	registry.pruneLocked(clock.now())

	if len(registry.buckets) != 1 {
		t.Fatalf("buckets = %d, want 1（只有視窗內那一個）", len(registry.buckets))
	}
	latest := minuteIndex(clock.now())
	if _, ok := registry.buckets[latest]; !ok {
		t.Error("當前分鐘的桶不該被剪掉")
	}
}

// itoa 是 strconv.Itoa 的本地別名，測試裡只需要非負整數。
// 刻意不直接呼叫 strconv：這支檔案沒有任何生產程式碼依賴 strconv 的行為，
// 少一個 import 就少一個可以誤用的符號。
func itoa(n int) string {
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
