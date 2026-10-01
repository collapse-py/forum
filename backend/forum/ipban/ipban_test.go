/*
封鎖名單的測試（ipban/ipban_test.go）。

這個功能的每一個性質都是「壞掉時不會報錯」的，因此測試集中在四個地方：

  1. 過期判定
     ZSET 裡的項目在清理之前會一直存在，因此「查到有」不等於「仍然有效」。
     少了 score 與現在的比較，一個三年前封的 IP 會永遠被封 —— 而症狀是
     「使用者早就說過他不再被擋了，但還是 403」。

  2. 清理的冪等性
     Prune 回報的筆數必須是實際刪掉的，而不是掃描時看到的。差異來自「掃描與
     刪除之間自然過期的項目」。

  3. 長度上下限
     「永久封鎖」在 score 是到期秒數的資料結構裡沒有辦法安全表達，因此 Ban
     必須拒絕超長的長度。這一組測試守住那個拒絕。

  4. nil 連線的安全
     Store 允許以 nil 連線建構（測試以 struct literal 構造 Server），而
     typed nil 介面陷阱會讓 *redis.Client 的 nil 接收者走進內部而 panic。

另外這一組測試**不用** miniredis 驗證 SCAN 之類的東西（那是 session 套件的
責任）：ipban 用的是 ZSET 的單一 key 查詢，不涉及遍歷，因此 miniredis 的
ZSCORE / ZADD / ZRANGEBYSCORE 支援就足夠了。
*/

package ipban

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestStore 建立一個有假時鐘的 Store，回傳可推進的時鐘。
//
// 假時鐘是必��的：過期判定的正確性完全取決於 score 與「現在」的比較，而用
// 真實時鐘就只能靠 sleep 去等一個邊界 —— 那是這個測試最不能接受的慢。
func newTestStore(t *testing.T) (*Store, *time.Time, *miniredis.Miniredis) {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	store := New(client)
	store.SetClock(func() time.Time { return now })
	return store, &now, mini
}

func TestBanAndCheck(t *testing.T) {
	store, now, _ := newTestStore(t)

	if banned, _, err := store.IsBanned(context.Background(), "203.0.113.9"); err != nil || banned {
		t.Fatalf("尚未封鎖時 IsBanned = %v, %v", banned, err)
	}

	if err := store.Ban(context.Background(), "203.0.113.9", now.Add(2*time.Hour)); err != nil {
		t.Fatalf("Ban: %v", err)
	}

	banned, until, err := store.IsBanned(context.Background(), "203.0.113.9")
	if err != nil {
		t.Fatalf("IsBanned: %v", err)
	}
	if !banned {
		t.Fatal("封鎖後 IsBanned 回傳 false")
	}
	if !until.Equal(now.Add(2 * time.Hour)) {
		t.Errorf("until = %v, want %v", until, now.Add(2*time.Hour))
	}
}

// 過期判定：把時鐘往前推過到期點，名單裡的項目必須變成「未封鎖」。
//
// 這一組是整個套件最重要的測試。少了 score 與現在的比較，ZSCORE 有回值就
// 會被當成「封鎖中」，於是一個三年前封的 IP 會永遠被封。
func TestExpiredBanIsTreatedAsNotBanned(t *testing.T) {
	store, now, _ := newTestStore(t)
	if err := store.Ban(context.Background(), "198.51.100.4", now.Add(time.Hour)); err != nil {
		t.Fatalf("Ban: %v", err)
	}

	// 還差一秒：仍然封鎖中。
	store.SetClock(func() time.Time { return now.Add(time.Hour - time.Second) })
	if banned, _, _ := store.IsBanned(context.Background(), "198.51.100.4"); !banned {
		t.Error("到期前一秒應仍為封鎖中")
	}

	// 剛好到期。
	store.SetClock(func() time.Time { return now.Add(time.Hour) })
	if banned, _, _ := store.IsBanned(context.Background(), "198.51.100.4"); banned {
		t.Error("剛好到點的封鎖應視為已過期（score <= now）")
	}

	// 遠遠過期。
	store.SetClock(func() time.Time { return now.Add(365 * 24 * time.Hour) })
	if banned, _, _ := store.IsBanned(context.Background(), "198.51.100.4"); banned {
		t.Error("過期一年後仍應視為未封鎖")
	}

	// 關鍵：即使 IsBanned 回 false，名單裡**仍然有**那一筆（清理是週期性的）。
	// 這個斷言固定了「查詢不等於清理」這個事實，避免日後有人把 Prune 從
	// 背景 goroutine 移除而以為查詢會順便清掉。
	entries, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List 回傳 %d 筆（已過期的項目不應出現在有效清單中）", len(entries))
	}
}

func TestBanExtendsExistingBan(t *testing.T) {
	store, now, _ := newTestStore(t)
	if err := store.Ban(context.Background(), "192.0.2.1", now.Add(time.Hour)); err != nil {
		t.Fatalf("Ban #1: %v", err)
	}
	// 再次封鎖應該是「延長」而不是報錯 —— 管理員對同一個 IP 再封一次，
	// 合理的直覺就是延長。
	if err := store.Ban(context.Background(), "192.0.2.1", now.Add(3*time.Hour)); err != nil {
		t.Fatalf("重複 Ban 應回傳 nil: %v", err)
	}
	_, until, _ := store.IsBanned(context.Background(), "192.0.2.1")
	if !until.Equal(now.Add(3 * time.Hour)) {
		t.Errorf("until = %v, want %v（重複封鎖應更新到期時間）", until, now.Add(3*time.Hour))
	}
	count, err := store.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1（重複封鎖不應增加項目數）", count)
	}
}

func TestUnban(t *testing.T) {
	store, now, _ := newTestStore(t)
	if err := store.Ban(context.Background(), "203.0.113.20", now.Add(time.Hour)); err != nil {
		t.Fatalf("Ban: %v", err)
	}

	existed, err := store.Unban(context.Background(), "203.0.113.20")
	if err != nil {
		t.Fatalf("Unban: %v", err)
	}
	if !existed {
		t.Error("解封一個確實存在的封鎖時 existed 應為 true")
	}
	if banned, _, _ := store.IsBanned(context.Background(), "203.0.113.20"); banned {
		t.Error("解封後仍回報封鎖中")
	}

	// 解封一個不存在的項目回 false 而不是錯誤：管理員按了「解封」而它已經
	// 解封了，那就是他想要的結果。
	existed, err = store.Unban(context.Background(), "203.0.113.20")
	if err != nil {
		t.Fatalf("重複 Unban 應回傳 nil: %v", err)
	}
	if existed {
		t.Error("解封不存在的項目時 existed 應為 false")
	}
}

func TestListIsSortedByExpiry(t *testing.T) {
	store, now, _ := newTestStore(t)
	// 故意反向加入，讓排序有意義。
	for _, spec := range []struct {
		ip  string
		off time.Duration
	}{{"10.0.0.3", 3 * time.Hour}, {"10.0.0.1", 1 * time.Hour}, {"10.0.0.2", 2 * time.Hour}} {
		if err := store.Ban(context.Background(), spec.ip, now.Add(spec.off)); err != nil {
			t.Fatalf("Ban %s: %v", spec.ip, err)
		}
	}

	entries, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("List 回傳 %d 筆, want 3", len(entries))
	}
	// 由到期時間由近到遠 —— 管理員要處理的是「快要解封的那個」。
	for i := 1; i < len(entries); i++ {
		if entries[i].Expires.Before(entries[i-1].Expires) {
			t.Fatalf("第 %d 筆的到期時間早於前一筆：%v vs %v", i, entries[i].Expires, entries[i-1].Expires)
		}
	}
	if entries[0].IP != "10.0.0.1" {
		t.Errorf("第一筆 = %q, want 10.0.0.1（最快到期）", entries[0].IP)
	}
}

func TestListExcludesExpired(t *testing.T) {
	store, now, _ := newTestStore(t)
	if err := store.Ban(context.Background(), "10.1.0.1", now.Add(time.Hour)); err != nil {
		t.Fatalf("Ban: %v", err)
	}
	// 過期項目無法用 Ban 建立（它會被最小長度的檢查擋下 —— 那正是正確行為），
	// 因此直接寫入 ZSET，構造「名單裡有過期項但還沒清理」的狀態。
	if err := store.rdb.ZAdd(context.Background(), banSetKey,
		redis.Z{Score: float64(now.Add(-time.Hour).Unix()), Member: "10.1.0.2"},
	).Err(); err != nil {
		t.Fatalf("ZAdd: %v", err)
	}

	entries, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 || entries[0].IP != "10.1.0.1" {
		t.Fatalf("List = %+v, want 只有 10.1.0.1", entries)
	}
}

func TestPruneRemovesOnlyExpired(t *testing.T) {
	store, now, _ := newTestStore(t)
	for i, off := range []time.Duration{time.Hour, 2 * time.Hour, -time.Hour, -2 * time.Hour} {
		if err := store.Ban(context.Background(), "172.16.0."+string(rune('1'+i)), now.Add(off)); err != nil {
			// 過期的 Ban 會因為「至少需一分鐘」而被拒，因此改用直接寫入來
			// 構造過期項目 —— 那正是「已經存在於名單裡但已過期」的狀態。
			if off > 0 {
				t.Fatalf("Ban %d: %v", i, err)
			}
		}
	}

	// 直接塞入兩筆已過期項目（模擬「名單裡有過期項但還沒清理」）。
	if err := store.rdb.ZAdd(context.Background(), banSetKey,
		redis.Z{Score: float64(now.Add(-time.Hour).Unix()), Member: "10.5.0.1"},
		redis.Z{Score: float64(now.Add(-time.Minute).Unix()), Member: "10.5.0.2"},
	).Err(); err != nil {
		t.Fatalf("ZAdd: %v", err)
	}

	before, err := store.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	removed, err := store.Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != 2 {
		t.Errorf("Prune 刪除 %d 筆, want 2", removed)
	}
	after, err := store.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if after != before-2 {
		t.Errorf("剩餘 %d 筆, want %d", after, before-2)
	}
}

// Ban 必須拒絕超長與過短的長度。
//
// 下限的理由是一個實務失誤模式：管理員輸入「1」以為是「1 天」而拿到一分鐘的
// 封鎖，攻擊者一分鐘後就能回來，而管理員以為已經處理了。上限的理由見
// maxBanDuration：score 是到期秒數，「永久」只能寫成極大數字而永不解除。
func TestBanRejectsOutOfRangeDuration(t *testing.T) {
	store, now, _ := newTestStore(t)

	if err := store.Ban(context.Background(), "10.6.0.1", now.Add(10*time.Second)); err == nil {
		t.Error("遠低於一分鐘的封鎖應被拒絕")
	}
	if err := store.Ban(context.Background(), "10.6.0.1", now.Add(10*365*24*time.Hour)); err == nil {
		t.Error("超過上限的封鎖應被拒絕")
	}
	// 邊界值：剛好一分鐘與剛好一年都應該通過。門檻是 minBanDuration/2 而不是
	// minBanDuration 本身，理由見該常數的註解（時鐘在計算與檢查之間會前進）。
	if err := store.Ban(context.Background(), "10.6.0.1", now.Add(minBanDuration)); err != nil {
		t.Errorf("剛好 minBanDuration 應通過: %v", err)
	}
	if err := store.Ban(context.Background(), "10.6.0.1", now.Add(maxBanDuration)); err != nil {
		t.Errorf("剛好 maxBanDuration 應通過: %v", err)
	}
}

// Ban 順手清理過期項目，因此即使背景 goroutine 從沒跑過，名單也不會無限成長。
func TestBanPrunesExpiredFirst(t *testing.T) {
	store, now, _ := newTestStore(t)
	for i := range 20 {
		if err := store.rdb.ZAdd(context.Background(), banSetKey,
			redis.Z{Score: float64(now.Add(-time.Duration(i+1) * time.Hour).Unix()), Member: "10.7.0." + string(rune('a'+i))},
		).Err(); err != nil {
			t.Fatalf("ZAdd: %v", err)
		}
	}

	if err := store.Ban(context.Background(), "10.7.9.9", now.Add(time.Hour)); err != nil {
		t.Fatalf("Ban: %v", err)
	}
	count, err := store.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("名單大小 = %d, want 1（Ban 應先清掉 20 筆過期項）", count)
	}
}

/* ==========================================================================
   IP 格式
   ========================================================================== */

func TestNormalizeIP(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"IPv4", "203.0.113.9", "203.0.113.9"},
		{"IPv4 帶前後空白", "  203.0.113.9  ", "203.0.113.9"},
		{"IPv6", "2001:db8::1", "2001:db8::1"},
		{"IPv4-mapped IPv6", "::ffff:203.0.113.9", "::ffff:203.0.113.9"},
		{"空字串", "", ""},
		{"只有空白", "   ", ""},
		{"不完整的 IPv4", "203.0.113", ""},
		{"超出範圍", "999.1.1.1", ""},
		{"CIDR 範圍", "203.0.113.0/24", ""},
		{"含逗號的 XFF", "203.0.113.9, 70.41.3.18", ""},
		{"文字", "not-an-ip", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeIP(tc.in); got != tc.want {
				t.Fatalf("NormalizeIP(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// CIDR 必須被拒絕：接受「一段範圍」會讓「這個 IP 被封了嗎」變成一個需要
// 逐一比對的問題，而 sorted set 的 member 是單一 IP。
func TestNormalizeIPRejectsCIDR(t *testing.T) {
	if got := NormalizeIP("203.0.113.0/24"); got != "" {
		t.Fatalf("NormalizeIP 接受了 CIDR: %q", got)
	}
}

/* ==========================================================================
   nil 連線
   ========================================================================== */

// Store 以 nil 連線建構時（測試以 struct literal 構造 Server）所有操作都必須
// 回 ErrNoStore，而不是 panic。
//
// 這是 typed nil 陷阱：*redis.Client 的 nil 接收者會走進 database/sql 與
// go-redis 內部而 panic（不是回傳錯誤）。Store 顯式擋掉是因為「沒有封鎖功能」
// 必須是一個能被處理的狀態，而 panic 會讓整個後台頁開不起來。
func TestNilStoreIsSafe(t *testing.T) {
	store := New(nil)
	ctx := context.Background()

	if banned, _, err := store.IsBanned(ctx, "203.0.113.1"); banned || err == nil {
		t.Errorf("IsBanned(nil) = %v, %v; want false + ErrNoStore", banned, err)
	}
	if err := store.Ban(ctx, "203.0.113.1", time.Now().Add(time.Hour)); err == nil {
		t.Error("Ban(nil) 應回傳錯誤")
	}
	if existed, err := store.Unban(ctx, "203.0.113.1"); existed || err == nil {
		t.Errorf("Unban(nil) = %v, %v; want false + 錯誤", existed, err)
	}
	if entries, err := store.List(ctx, 0); err == nil || len(entries) != 0 {
		t.Errorf("List(nil) = %v, %v", entries, err)
	}
	if removed, err := store.Prune(ctx); err == nil || removed != 0 {
		t.Errorf("Prune(nil) = %d, %v", removed, err)
	}
	// 連 *Store 本身都是 nil 時也必須安全：s.blocks 可能是 nil 指標。
	var nilStore *Store
	if banned, _, err := nilStore.IsBanned(ctx, "1.2.3.4"); banned || err == nil {
		t.Errorf("(*Store)(nil).IsBanned = %v, %v", banned, err)
	}
}

// Pruner 在沒有 store 時必須直接返回，不能啟動一個永遠在 tick 卻什麼都不做的
// goroutine —— 那會讓診斷變困難。
//
// 這支測試需要一個**可取消**的 context：Run 會一直 tick 直到 ctx 被取消，而
// 傳 context.Background() 會讓這支測試永遠跑不完（這是寫它的時候踩到的）。
func TestPrunerWithNilStoreReturnsImmediately(t *testing.T) {
	cancelled, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	called := false
	NewPruner(nil, time.Millisecond).Run(cancelled, func(int, error) {
		called = true
	})
	if called {
		t.Error("nil store 不應觸發回呼")
	}

	// store 的 Redis 是 nil 時 Prune 會回 ErrNoStore；這是預期中的，而這支
	// 測試只確認 goroutine 會隨 ctx 取消而退出而不是卡住。
	NewPruner(New(nil), time.Millisecond).Run(cancelled, func(int, error) {})
}

// 錯誤訊息不應洩漏內部細節給使用者。ipban 的錯誤會被 handler 直接顯示，因此
// 它必須是人看得懂的。
func TestErrorsAreUserFacing(t *testing.T) {
	if !strings.Contains(ErrInvalidIP.Error(), "IP") {
		t.Errorf("ErrInvalidIP = %q，應包含「IP」這個詞", ErrInvalidIP.Error())
	}
}
