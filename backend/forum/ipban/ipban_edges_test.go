/*
ipban 的邊界與失敗分支測試（backend/forum/ipban/ipban_edges_test.go）。

ipban_test.go 覆蓋正常路徑（封鎖、過期、延長、解除、排序、清理），這個檔補
「壞掉時呼叫端拿到什麼」的那一半：

 1. RemainingSeconds：過期項目必須回 0 而不是負數 —— 它會被直接算進
    Retry-After 之類的顯示，負數秒數是前端沒準備過的形狀。
 2. nil store 的每一個匯出方法：rdb 為 nil 時全部回 ErrNoStore（呼叫端
    fail open），而不是 panic。
 3. 輸入邊界：不可解析的 IP、超過上限／低於下限的封鎖時長。
 4. Redis 故障（連線斷）：IsBanned 必須回錯誤而不是「未封鎖」——呼叫端
    依賴「err 非 nil ⇒ banned 必為 false」來決定 fail open 的記錄方式。
 5. Pruner 的 ticker 真的會定期呼叫清理，且 ctx 取消後停止。
*/

package ipban

import (
	"context"
	"errors"
	"testing"
	"time"
)

/* ==========================================================================
   RemainingSeconds
   ========================================================================== */

func TestRemainingSecondsFloorsAtZero(t *testing.T) {
	// Expires 刻意落在「90 整 + 0.5 秒」，而不是剛好 90 整。
	//
	// RemainingSeconds 是「呼叫當下」的 time.Until 再截斷（floor）。到期時間
	// 剛好是 90.000000 時，兩次 time.Now() 之間任何微小流逝都會把它推到
	// 89.999999 → 截成 89：Linux 上兩次 Now 必然不同（Windows 的時脈粒度讓
	// 差值量化成 0，所以本地會過、CI 必紅 —— 2026-10-23 的 backend job 就
	// 是死在 `RemainingSeconds = 89, want 90`）。
	//
	// 那不是實作的 bug，是斷言踩在截斷邊界上。0.5 秒緩衝把要測的事保留
	// （未來到期 → 回傳整秒的剩餘時間），同時讓「時間會流逝」不再決定成敗。
	future := Entry{Expires: time.Now().Add(90*time.Second + 500*time.Millisecond)}
	if got := future.RemainingSeconds(); got != 90 {
		t.Errorf("未來到期的 RemainingSeconds = %d, want 90", got)
	}

	past := Entry{Expires: time.Now().Add(-time.Hour)}
	if got := past.RemainingSeconds(); got != 0 {
		t.Errorf("已過期的 RemainingSeconds = %d, want 0（負數秒數不是前端準備過的形狀）", got)
	}
}

/* ==========================================================================
   nil store：每一個匯出方法
   ========================================================================== */

func TestNilStoreMethodsReturnErrNoStore(t *testing.T) {
	var nilStore *Store
	ctx := context.Background()

	if banned, until, err := nilStore.IsBanned(ctx, "203.0.113.9"); banned || !until.IsZero() || !errors.Is(err, ErrNoStore) {
		t.Errorf("IsBanned = (%v, %v, %v), want (false, 零值, ErrNoStore)", banned, until, err)
	}
	if err := nilStore.Ban(ctx, "203.0.113.9", time.Now().Add(time.Hour)); !errors.Is(err, ErrNoStore) {
		t.Errorf("Ban = %v, want ErrNoStore", err)
	}
	if existed, err := nilStore.Unban(ctx, "203.0.113.9"); existed || !errors.Is(err, ErrNoStore) {
		t.Errorf("Unban = (%v, %v), want (false, ErrNoStore)", existed, err)
	}
	if items, err := nilStore.List(ctx, 0); len(items) != 0 || !errors.Is(err, ErrNoStore) {
		t.Errorf("List = (%v, %v), want (空, ErrNoStore)", items, err)
	}
	if removed, err := nilStore.Prune(ctx); removed != 0 || !errors.Is(err, ErrNoStore) {
		t.Errorf("Prune = (%v, %v), want (0, ErrNoStore)", removed, err)
	}
	if count, err := nilStore.Count(ctx); count != 0 || !errors.Is(err, ErrNoStore) {
		t.Errorf("Count = (%v, %v), want (0, ErrNoStore)", count, err)
	}

	// New(nil) 建立的 Store 同樣要走 ErrNoStore 路徑，而不是解引用 nil rdb。
	constructed := New(nil)
	if _, _, err := constructed.IsBanned(ctx, "203.0.113.9"); !errors.Is(err, ErrNoStore) {
		t.Errorf("New(nil).IsBanned = %v, want ErrNoStore", err)
	}
}

/* ==========================================================================
   輸入邊界
   ========================================================================== */

func TestBanAndUnbanRejectInvalidIP(t *testing.T) {
	store, _, _ := newTestStore(t)
	until := time.Now().Add(time.Hour)

	for _, raw := range []string{"", "   ", "not-an-ip", "999.999.999.999", "2001:db8:::1"} {
		if err := store.Ban(context.Background(), raw, until); !errors.Is(err, ErrInvalidIP) {
			t.Errorf("Ban(%q) = %v, want ErrInvalidIP", raw, err)
		}
		if existed, err := store.Unban(context.Background(), raw); existed || !errors.Is(err, ErrInvalidIP) {
			t.Errorf("Unban(%q) = (%v, %v), want (false, ErrInvalidIP)", raw, existed, err)
		}
	}
}

func TestBanDurationBoundsUseStoreClock(t *testing.T) {
	store, now, _ := newTestStore(t)
	ctx := context.Background()

	// 超過上限（365 天）：永久封鎖是刻意不做的功能。
	if err := store.Ban(ctx, "203.0.113.1", now.Add(maxBanDuration+time.Hour)); err == nil {
		t.Error("超過 maxBanDuration 的封鎖應被拒絕（永久封鎖刻意不做）")
	}
	// 低於下限（1 分鐘的一半以下）：太短的封鎖讓管理員誤以為擋住了。
	if err := store.Ban(ctx, "203.0.113.2", now.Add(10*time.Second)); err == nil {
		t.Error("低於 minBanDuration 的封鎖應被拒絕")
	}
	// 剛好等於邊界：允許。
	if err := store.Ban(ctx, "203.0.113.3", now.Add(maxBanDuration)); err != nil {
		t.Errorf("剛好等於 maxBanDuration 應被接受: %v", err)
	}
	if err := store.Ban(ctx, "203.0.113.4", now.Add(minBanDuration/2)); err != nil {
		t.Errorf("剛好等於 minBanDuration/2 應被接受: %v", err)
	}
}

/* ==========================================================================
   Redis 故障
   ========================================================================== */

// IsBanned 在 Redis 斷線時必須回錯誤：呼叫端（withBlocklistHandler）依賴
//「err 非 nil ⇒ banned 必為 false」來 fail open 並記錄，把故障靜默成
//「未封鎖」會讓封鎖名單看起來運作正常。
func TestIsBannedReportsRedisFailure(t *testing.T) {
	store, _, mini := newTestStore(t)
	mini.Close()

	banned, until, err := store.IsBanned(context.Background(), "203.0.113.9")
	if err == nil {
		t.Fatal("Redis 斷線時 IsBanned 應回錯誤（呼叫端 fail open 並記錄）")
	}
	if banned || !until.IsZero() {
		t.Errorf("err 非 nil 時 banned/until 應為 false/零值, 得到 (%v, %v)", banned, until)
	}
}

func TestListAndCountOnEmptyStore(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()

	items, err := listOrDefault(t, store, ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("空名單的 List = %v, want 空", items)
	}
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("空名單的 Count = %d, want 0", count)
	}
}

// listOrDefault 呼叫 List 並在 err 為 nil 時回傳項目；這裡獨立包一層只是為了
// 讓 t.Helper 的行號指向呼叫端。
func listOrDefault(t *testing.T, store *Store, ctx context.Context) ([]Entry, error) {
	t.Helper()
	return store.List(ctx, 0)
}

/* ==========================================================================
   Pruner
   ========================================================================== */

func TestNewPrunerDefaultsInterval(t *testing.T) {
	if p := NewPruner(nil, 0); p.interval != time.Hour {
		t.Errorf("interval = %v, want 1h（非正值的防禦性回退）", p.interval)
	}
	if p := NewPruner(nil, -time.Minute); p.interval != time.Hour {
		t.Errorf("負 interval = %v, want 1h", p.interval)
	}
}

func TestPrunerRunTicksAndStopsOnCancel(t *testing.T) {
	store, now, _ := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 一個已過期的封鎖：第一個 tick 就該被清掉。
	if err := store.Ban(ctx, "203.0.113.7", now.Add(time.Minute)); err != nil {
		t.Fatalf("Ban: %v", err)
	}
	*now = now.Add(2 * time.Minute)

	pruned := make(chan int, 8)
	p := NewPruner(store, 20*time.Millisecond)
	done := make(chan struct{})
	go func() {
		p.Run(ctx, func(n int, _ error) { pruned <- n })
		close(done)
	}()

	select {
	case n := <-pruned:
		if n != 1 {
			t.Errorf("Pruner 清掉了 %d 筆, want 1", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Pruner 的 ticker 沒有定期清理")
	}

	// nil store 的 Run 必須立刻返回。
	nilDone := make(chan struct{})
	go func() { NewPruner(nil, time.Millisecond).Run(ctx, nil); close(nilDone) }()
	select {
	case <-nilDone:
	case <-time.After(2 * time.Second):
		t.Fatal("nil store 的 Run 沒有立刻返回")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ctx 取消後 Run 沒有返回")
	}
}
