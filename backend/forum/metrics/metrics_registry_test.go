/*
metrics 註冊表其餘性質的測試（backend/forum/metrics/metrics_registry_test.go）。

metrics_test.go 覆蓋觀察與快照的主幹，這個檔補三塊「壞掉時監控頁會給出錯誤
數字、甚至直接崩潰」的部分：

 1. 分位數的溢出格。直方圖最右邊一格（≥10s）的索引是 overflowIndex，而
    latencyBounds 的長度**恰好**也是 overflowIndex —— 取上界時若直接以
    overflowIndex 當索引就會越界 panic。這不是假想：一次 ≥10s 的慢請求
    （例如檔案服務暫時無回應）就足以讓 p50/p95 落進溢出格，監控頁整頁崩掉。
 2. 限流器統計的中轉（SetLimitStats）：快照必須反映呼叫端灌入的值，且
    以 nil 重設時要真的清空 —— 累積舊值會讓停用的限流器永遠顯示舊計數。
 3. 起始時間與 TimelineMinutes 的三個分支（nil 註冊表、預設、封頂）。

持久化（FlushPending / LoadHistory / StartFlusher）的測試在
metrics_flush_mysql_test.go，那一塊需要真實 MySQL。
*/

package metrics

import (
	"testing"
	"time"
)

/* ==========================================================================
   分位數：溢出格
   ========================================================================== */

// 一次落在溢出格（≥10s）的觀測，配上任何分位數都必須回「最後一個邊界」
// （10s），而不是索引越界 panic。
func TestQuantileInOverflowBucketReturnsLastBound(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)

	// 15 秒 > 10s 的最後邊界：bucketIndex 回 overflowIndex。
	registry.Observe("POST", "/forum/posts", 200, 15*time.Second)

	wantUpperBoundMS := durationMS(latencyBounds[overflowIndex-1]) // 10000ms
	snap := registry.Snapshot(1)
	for _, got := range []struct {
		name string
		ms   float64
	}{
		{"P50MS", snap.Requests.P50MS},
		{"P95MS", snap.Requests.P95MS},
		{"P99MS", snap.Requests.P99MS},
	} {
		if got.ms != wantUpperBoundMS {
			t.Errorf("%s = %v, want %v（溢出格回最後一個邊界，不是 panic 也不是 0）",
				got.name, got.ms, wantUpperBoundMS)
		}
	}
	if snap.Requests.MaxMS != 15000 {
		t.Errorf("MaxMS = %v, want 15000（最大值取實際觀測，不受格上界影響）", snap.Requests.MaxMS)
	}
}

// 混合分佈：只有 p99 落進溢出格，p50 應仍停在較小的格上界。
func TestQuantileMixedDistributionKeepsLowerBoundForP50(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)

	// 98 個 5ms + 2 個溢出格：p99 的目標是第 99 個樣本（ceil(100×0.99)），
	// 它必須落在溢出格 —— 只放一個慢樣本的話第 99 個仍是 5ms（那是對的），
	// 但那驗不到溢出格的取上界。
	for i := 0; i < 98; i++ {
		registry.Observe("GET", "/forum", 200, 5*time.Millisecond)
	}
	registry.Observe("GET", "/forum", 200, 30*time.Second)
	registry.Observe("GET", "/forum", 200, 30*time.Second)

	snap := registry.Snapshot(1)
	if snap.Requests.P50MS != durationMS(5*time.Millisecond) {
		t.Errorf("P50MS = %v, want 5（98 個 5ms 樣本下 p50 不該被慢請求拉進溢出格）", snap.Requests.P50MS)
	}
	if snap.Requests.P99MS != durationMS(latencyBounds[overflowIndex-1]) {
		t.Errorf("P99MS = %v, want %v（第 100 個樣本在溢出格）",
			snap.Requests.P99MS, durationMS(latencyBounds[overflowIndex-1]))
	}
}

/* ==========================================================================
   限流器統計中轉
   ========================================================================== */

// SetLimitStats 是限流器（httpapi）與監控頁之間唯一的橋樑：灌進去什麼，
// 快照就要原樣帶出來；以 nil 重設後必須真的清空，而不是沿用舊值。
func TestSetLimitStatsFeedsAndClearsSnapshot(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)

	stats := []LimitStat{
		{Name: "write", Limit: 10, Window: time.Minute, Allowed: 120, Blocked: 3, Tracked: 7},
		{Name: "upload", Limit: 5, Window: time.Minute, Allowed: 40, Blocked: 1, Tracked: 2},
	}
	registry.SetLimitStats(stats)

	snap := registry.Snapshot(1)
	if len(snap.Limits) != 2 {
		t.Fatalf("Limits 長度 = %d, want 2", len(snap.Limits))
	}
	if snap.Limits[0].Name != "write" || snap.Limits[0].Blocked != 3 || snap.Limits[0].TrackedKeys != 7 {
		t.Errorf("Limits[0] = %+v, want write/blocked=3/trackedKeys=7", snap.Limits[0])
	}
	if snap.Limits[1].Name != "upload" || snap.Limits[1].WindowSeconds != 60 {
		t.Errorf("Limits[1] = %+v, want upload/windowSeconds=60", snap.Limits[1])
	}

	// nil 重設：停用的限流器不得繼續顯示舊計數。
	registry.SetLimitStats(nil)
	if got := registry.Snapshot(1).Limits; len(got) != 0 {
		t.Errorf("SetLimitStats(nil) 之後 Limits = %+v, want 空", got)
	}
}

/* ==========================================================================
   起始時間與時間軸長度
   ========================================================================== */

func TestStartedAtMatchesInjectedClock(t *testing.T) {
	registry, clock := newTestRegistry(t, 30)

	if got := registry.StartedAt(); !got.Equal(clock.current) {
		t.Errorf("StartedAt = %v, want %v（uptime 的基準必須與注入時鐘一致）", got, clock.current)
	}

	// 時鐘前進不影響 StartedAt：它是「啟動那一刻」，不是「現在」。
	clock.advance(90 * time.Minute)
	if got := registry.StartedAt(); !got.Equal(clock.current.Add(-90*time.Minute)) {
		t.Errorf("StartedAt 隨時鐘前進而改變 = %v", got)
	}
	if snap := registry.Snapshot(1); snap.UptimeSeconds != 5400 {
		t.Errorf("UptimeSeconds = %v, want 5400", snap.UptimeSeconds)
	}
}

func TestTimelineMinutesBranches(t *testing.T) {
	var nilRegistry *Registry
	if got := nilRegistry.TimelineMinutes(); got != 0 {
		t.Errorf("nil 註冊表的 TimelineMinutes = %d, want 0", got)
	}

	// 未設定視窗時回預設值。
	withDefaults := New(Options{})
	if got := withDefaults.TimelineMinutes(); got != defaultWindowMinutes {
		t.Errorf("預設 TimelineMinutes = %d, want %d", got, defaultWindowMinutes)
	}

	// 視窗超過呈現上限時以 maxTimelinePoints 收斂（回應體積與前端繪圖能力）。
	long := New(Options{WindowMinutes: maxTimelinePoints + 500})
	if got := long.TimelineMinutes(); got != maxTimelinePoints {
		t.Errorf("超長視窗的 TimelineMinutes = %d, want %d（呈現上限）", got, maxTimelinePoints)
	}

	// 視窗在上限之內時以視窗為準。
	short := New(Options{WindowMinutes: 45})
	if got := short.TimelineMinutes(); got != 45 {
		t.Errorf("TimelineMinutes = %d, want 45", got)
	}
}

/* ==========================================================================
   剪枝：歷史桶與視窗外的當前桶
   ========================================================================== */

// pruneLocked 必須同時清理 history 與 buckets：只清其一的症狀是「重啟後載入
// 的歷史讓記憶體無限增長」——它不在任何圖上，卻一直佔著。
func TestPruneDropsStaleHistoryAndBuckets(t *testing.T) {
	registry, clock := newTestRegistry(t, 30)

	staleMinute := minuteIndex(clock.current)
	clock.advance(60 * time.Minute) // 視窗（30 分）之外
	freshMinute := minuteIndex(clock.current)

	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.history[-1000] = &minuteBucket{minute: -1000}
	registry.history[staleMinute] = &minuteBucket{minute: staleMinute}
	registry.history[freshMinute] = &minuteBucket{minute: freshMinute}
	registry.buckets[-1000] = &minuteBucket{minute: -1000}
	// pruneLocked 是 FlushPending 在**有 db 時**才走的內部剪枝（db 為 nil 的
	// 提早返回不會碰到它）；套件內測試持有 mu 直接呼叫，釘住同一份實作。
	registry.pruneLocked(clock.current)

	if _, ok := registry.history[-1000]; ok {
		t.Error("history 的過期桶沒有被清掉（記憶體會隨重啟次數累積）")
	}
	if _, ok := registry.history[staleMinute]; ok {
		t.Error("剛離開視窗的 history 桶沒有被清掉")
	}
	if _, ok := registry.buckets[-1000]; ok {
		t.Error("buckets 的過期桶沒有被清掉")
	}
	if _, ok := registry.history[freshMinute]; !ok {
		t.Error("仍在視窗內的 history 桶被誤刪")
	}
}
