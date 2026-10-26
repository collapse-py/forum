/*
idgen 的測試（idgen/snowflake_test.go）。

這裡測的是「不能靠眼睛看出來」的四件事：

 1. 位元配置。53 位元是這個專案與前端 JavaScript 的契約；多一個位元就代表
    某些 id 在瀏覽器裡會被四捨五入，而那只有部分資料會壞。
 2. 唯一與遞增。同一隻行程內重複發號會直接撞主鍵；遞增則是「id 排序等於
    時間排序」這個既有查詢假設的來源。
 3. 序號用盡。同一毫秒的第 65 個 id 必須換到下一個毫秒，而不是回繞成 0
    而與前面的 id 重複。
 4. 時鐘回退。NTP 校正很常見，而它不該讓新 id 比舊 id 小。

時鐘一律注入，測試不依賴真實時間的流動（除了序號用盡時的極短等待）。
*/
package idgen

import (
	"sync"
	"testing"
	"time"
)

// fakeClock 每次被讀取就前進 step，讓「同一毫秒內發多個 id」與
// 「毫秒前進」都能被精確控制。
type fakeClock struct {
	mu    sync.Mutex
	base  time.Time
	step  time.Duration
	calls int
}

func newFakeClock(start time.Time, step time.Duration) *fakeClock {
	return &fakeClock{base: start, step: step}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.base
	c.base = c.base.Add(c.step)
	c.calls++
	return now
}

func (c *fakeClock) set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.base = now
}

// testEpoch 是 fakeClock 的起點：epoch 之後一小段，避免剛好踩在邊界上。
var testEpoch = time.UnixMilli(EpochMillis).Add(time.Hour)

func TestBitLayout(t *testing.T) {
	// 這一行是「前端 precision」的守門員：53 位元是 JavaScript number 能
	// 精確表示的最大位元數。
	if totalBits != 53 {
		t.Fatalf("位元總數 = %d，want 53（超過就無法用 JSON number 送給前端）", totalBits)
	}
	if timestampShift != sequenceBits+nodeBits {
		t.Fatalf("timestampShift = %d，want %d", timestampShift, sequenceBits+nodeBits)
	}
}

func TestNewRejectsOutOfRangeNodeID(t *testing.T) {
	for _, nodeID := range []int64{-1, MaxNodeID + 1, 1000} {
		if _, err := New(nodeID); err == nil {
			t.Errorf("New(%d) 應回傳錯誤", nodeID)
		}
	}
	for _, nodeID := range []int64{0, 1, MaxNodeID} {
		if _, err := New(nodeID); err != nil {
			t.Errorf("New(%d) = %v，want nil", nodeID, err)
		}
	}
}

func TestNextIsUniqueMonotonicAndFitsIn53Bits(t *testing.T) {
	g, err := New(7)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// 每讀一次前進 1 毫秒：序號永遠是 0，測的是時間戳這一段。
	g.now = newFakeClock(testEpoch, time.Millisecond).Now

	prev := int64(0)
	for i := 0; i < 1000; i++ {
		id := g.Next()
		if id <= prev {
			t.Fatalf("第 %d 個 id = %d，沒有比前一個 %d 大", i, id, prev)
		}
		if id < 0 || id >= int64(1)<<totalBits {
			t.Fatalf("第 %d 個 id = %d 超出 %d 位元範圍", i, id, totalBits)
		}
		if got := (id >> nodeShift) & MaxNodeID; got != 7 {
			t.Fatalf("第 %d 個 id 的節點編號 = %d，want 7", i, got)
		}
		prev = id
	}
}

// burstClock 每 every 次讀取才前進 1 毫秒。every 大於 64 時，同一毫秒
// 一定會被迫發出第 65 個 id —— 那正是序號用盡的分支。
type burstClock struct {
	mu    sync.Mutex
	base  time.Time
	every int
	calls int
}

func (c *burstClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.calls%c.every == 0 {
		c.base = c.base.Add(time.Millisecond)
	}
	return c.base
}

// TestNextSurvivesSequenceExhaustion 是這個套件最重要的一條：
// 同一毫秒的第 65 個 id 若回繞，就會與第 1 個 id 完全相同。
func TestNextSurvivesSequenceExhaustion(t *testing.T) {
	g, err := New(3)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// 每 70 次讀取才前進 1 毫秒：序號一定有用盡的機會。
	g.now = (&burstClock{base: testEpoch, every: 70}).Now

	seen := make(map[int64]struct{})
	prev := int64(0)
	for i := 0; i < 500; i++ {
		id := g.Next()
		if _, ok := seen[id]; ok {
			t.Fatalf("第 %d 個 id = %d 重複了", i, id)
		}
		seen[id] = struct{}{}
		if id <= prev {
			t.Fatalf("第 %d 個 id = %d，沒有比前一個 %d 大", i, id, prev)
		}
		prev = id
	}
}

// TestNextKeepsMonotonicWhenClockGoesBack 模擬 NTP 把時鐘往回撥。
// 期待的結果不是「報錯」而是「繼續發比舊 id 大的值」—— 資料庫裡已經有
// 舊 id，重新發一個較小的值會讓「依 id 排序」的列表把新資料排到最前面。
func TestNextKeepsMonotonicWhenClockGoesBack(t *testing.T) {
	g, err := New(0)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	clock := newFakeClock(testEpoch, 0)
	g.now = clock.Now

	first := g.Next()
	clock.set(testEpoch.Add(-time.Second))
	for i := 0; i < 5; i++ {
		id := g.Next()
		if id <= first {
			t.Fatalf("時鐘回退後 id = %d，沒有比回退前的 %d 大", id, first)
		}
	}
}

func TestConfigureRejectsInvalidNodeID(t *testing.T) {
	if err := Configure(MaxNodeID + 1); err == nil {
		t.Fatal("Configure 應拒絕超出範圍的節點編號")
	}
	// 成功路徑：換成節點 5 之後，行程層級發出的 id 必須帶有那個節點編號。
	// 測試結束後換回 0，避免這個全域狀態影響同一 package 的其他測試。
	t.Cleanup(func() { _ = Configure(0) })
	if err := Configure(5); err != nil {
		t.Fatalf("Configure(5): %v", err)
	}
	id := Next()
	if got := (id >> nodeShift) & MaxNodeID; got != 5 {
		t.Fatalf("行程層級 id 的節點編號 = %d，want 5", got)
	}
}
