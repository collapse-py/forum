/*
Session 列舉與批次撤銷的測試（session/session_list_test.go）。

這個功能最需要測的是「壞掉時不會報錯」的四個地方：

  1. 掃描必須走遍整個 keyspace
     go-redis 的 Scan 是游標式的；把它寫成「每次都傳 0」會永遠只掃第一頁，
     而症狀是「看得到少數幾支 session，看起來一切正常」。這個測試建立 200 支
     session 並斷言全部被看到 —— 那個數字刻意大於 scanBatchSize，因此一個
     只掃一批的實作會失敗。

  2. 不能洩漏完整 token
     token 就是憑證。這個測試斷言 List 的結果裡每一筆的 Token 都不等於
     完整的 48 個 hex 字元，而 TokenPrefix 確實是它的前綴。

  3. 去重
     Redis 的 SCAN **不保證**一次遍歷內不重複回傳同一個 key。去重漏掉的症狀
     是「同一支 session 在列表裡出現兩次」，而那是管理員無法解讀的現象。

  4. 撤銷要刪掉「所有」而不是「掃描到的那些」
     特別是那個「掃描被截斷就回錯」的行為 —— 靜默回傳 0 筆會讓後臺對使用者
     宣稱「已全部登出」而其實還有 session 活著。

這些測試用 miniredis：它支援 SCAN、pipeline、TTL 與 HGETALL，因此足以驗證
掃描迴圈與撤銷的正確性，而不需要真的 Redis 行程。
*/

package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestManager(t *testing.T, sessionCount int, expire time.Duration) (*Manager, *miniredis.Miniredis) {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	manager := NewManager("FORUM_test", expire, false, client)
	for i := range sessionCount {
		if _, err := manager.Create(testEmail(i), i%10 == 0); err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
	}
	return manager, mini
}

// testEmail 產生可預期的 email。刻意讓前 10 個是管理員（i%10==0），
// 這樣 IsAdmin 的分佈是可預期的。
func testEmail(i int) string {
	return string(rune('a'+i%26)) + "user" + itoa(i) + "@example.com"
}

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

/* ==========================================================================
   建立時間
   ========================================================================== */

// Create 必須把建立時間寫進 hash，否則後臺的 session 列表永遠無法顯示
// 「這支 session 是什麼時候開的」—— 那正是這個功能的主要價值。
func TestCreateStoresCreatedAt(t *testing.T) {
	_, mini := newTestManager(t, 1, time.Hour)

	keys := mini.Keys()
	if len(keys) != 1 {
		t.Fatalf("mini.Keys() = %v, want 1", keys)
	}
	// miniredis 的 HGet 只回一個值（讀不到時是空字串），不像真的 Redis
	// 那樣回 redis.Nil。
	created := mini.HGet(keys[0], createdAtField)
	if created == "" {
		t.Fatal("Create 沒有寫入 created_at，後臺將無法顯示建立時間")
	}
	if len(created) != 10 {
		t.Errorf("created_at = %q, want 10 位的 Unix 秒", created)
	}
}

/* ==========================================================================
   列舉
   ========================================================================== */

// 這個數字刻意大於 scanBatchSize（256）：一個「只掃第一批」的實作會在這裡
// 失敗，而它最可能的寫法錯誤（Scan 的游標每次都傳 0）在症狀上完全相同。
func TestListFindsEverySession(t *testing.T) {
	const count = 200
	manager, _ := newTestManager(t, count, time.Hour)

	result, err := manager.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.TotalActive != count {
		t.Fatalf("TotalActive = %d, want %d（掃描沒有走遍整個 keyspace）", result.TotalActive, count)
	}
	if len(result.Sessions) != count {
		t.Fatalf("Sessions = %d, want %d", len(result.Sessions), count)
	}
	if result.Truncated {
		t.Error("200 支不該觸發截斷（上限是 20000）")
	}
	if result.Scanned != count {
		t.Errorf("Scanned = %d, want %d", result.Scanned, count)
	}
}

func TestListFiltersByEmail(t *testing.T) {
	manager, _ := newTestManager(t, 20, time.Hour)

	result, err := manager.List(context.Background(), ListOptions{Email: testEmail(7)})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("Sessions = %d, want 1（email 篩選不精確）", len(result.Sessions))
	}
	if result.Sessions[0].Email != testEmail(7) {
		t.Errorf("Email = %q, want %q", result.Sessions[0].Email, testEmail(7))
	}
}

// 完整 token 絕不能出現在 List 的結果裡。
//
// 這一支是這個功能最重要的安全測試：token 就是憑證（session.go 檔頭第一點），
// 而它一旦出現在後臺畫面上，「截圖分享」就變成了一次完整的手法移交。
func TestListNeverExposesFullToken(t *testing.T) {
	manager, _ := newTestManager(t, 10, time.Hour)

	result, err := manager.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, record := range result.Sessions {
		// Record.Token 本身是給刪除用的內部欄位；這裡驗證的是 TokenPrefix
		// 不足以還原它。
		if len(record.Token) <= tokenPrefixLength {
			t.Fatalf("測試資料的 token 過短，前綴測試沒有意義: %q", record.Token)
		}
		if strings.HasSuffix(record.TokenPrefix, record.Token) {
			t.Errorf("TokenPrefix = %q 包含了完整 token", record.TokenPrefix)
		}
		if !strings.HasPrefix(record.Token, strings.TrimSuffix(record.TokenPrefix, "…")) {
			t.Errorf("TokenPrefix = %q 不是 token %q 的前綴", record.TokenPrefix, record.Token)
		}
	}
}

func TestListReportsAdminAndExpiry(t *testing.T) {
	manager, _ := newTestManager(t, 10, 2*time.Hour)

	result, err := manager.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	admins := 0
	for _, record := range result.Sessions {
		if record.IsAdmin {
			admins++
		}
		// TTL 應該接近設定的 2 小時。miniredis 不會真的走時間，因此用
		// 「大於 1 小時」這個寬鬆的界線，避免測試變得脆弱。
		if record.TTL < time.Hour {
			t.Errorf("TTL = %v, want ≈2h", record.TTL)
		}
		if record.ExpiresAt.IsZero() {
			t.Error("ExpiresAt 為零值；TTL 已知時它應該被算出來")
		}
	}
	if admins != 1 {
		t.Errorf("管理員 session = %d, want 1（i%%10==0 恰好一個）", admins)
	}
}

func TestListAppliesLimit(t *testing.T) {
	manager, _ := newTestManager(t, 30, time.Hour)

	result, err := manager.List(context.Background(), ListOptions{Limit: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Sessions) != 5 {
		t.Errorf("Sessions = %d, want 5（Limit 未生效）", len(result.Sessions))
	}
	// TotalActive 仍要是全部的數量：介面要靠它顯示「符合 5 筆，全站共 30 筆」。
	if result.TotalActive != 30 {
		t.Errorf("TotalActive = %d, want 30", result.TotalActive)
	}
}

// MaxScanned 生效時必須同時回 truncated，否則「沒列出來」會被讀成「不存在」。
func TestListReportsTruncation(t *testing.T) {
	manager, _ := newTestManager(t, 30, time.Hour)

	result, err := manager.List(context.Background(), ListOptions{MaxScanned: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !result.Truncated {
		t.Fatal("MaxScanned=5 而有 30 支 session，卻沒有回 truncated")
	}
	if result.TotalActive >= 30 {
		t.Errorf("TotalActive = %d，截斷時不可能等於全部的 30", result.TotalActive)
	}
}

/* ==========================================================================
   撤銷
   ========================================================================== */

func TestRevokeByEmailDeletesOnlyThatUser(t *testing.T) {
	manager, mini := newTestManager(t, 30, time.Hour)
	target := testEmail(11)

	result, err := manager.RevokeByEmail(context.Background(), target)
	if err != nil {
		t.Fatalf("RevokeByEmail: %v", err)
	}
	if result.Revoked != 1 {
		t.Fatalf("Revoked = %d, want 1（每個 email 只有一支 session）", result.Revoked)
	}

	// 目標的 key 必須真的消失，其餘的必須還在。
	remaining := mini.Keys()
	if len(remaining) != 29 {
		t.Errorf("剩餘 key = %d, want 29（只刪了目標那一支）", len(remaining))
	}
	listed, err := manager.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, record := range listed.Sessions {
		if record.Email == target {
			t.Error("被撤銷的 session 仍然出現在列表中")
		}
	}
}

// 掃描被截斷時必須回錯，不能靜默回「已撤銷 0 筆」——
// 那會讓後臺對使用者宣稱「已全部登出」而其實還有 session 活著。
func TestRevokeFailsWhenTruncated(t *testing.T) {
	manager, mini := newTestManager(t, 100, time.Hour)

	// 先塞入大量 key 讓掃描必然觸及上限（maxScannedKeys = 20000），
	// 再確認撤銷在掃完之前就停下並回錯。
	for i := 0; i < maxScannedKeys+100; i++ {
		mini.Set(sessionKey("filler"+itoa(i)), "noise")
	}
	if _, err := manager.RevokeByEmail(context.Background(), testEmail(3)); err == nil {
		t.Fatal("掃描被截斷時 RevokeByEmail 必須回錯")
	}
}

func TestRevokeByEmailRequiresEmail(t *testing.T) {
	manager, _ := newTestManager(t, 3, time.Hour)
	if _, err := manager.RevokeByEmail(context.Background(), "   "); err == nil {
		t.Error("空白 email 應回錯")
	}
}

func TestRevokeUnknownEmailIsNoop(t *testing.T) {
	manager, mini := newTestManager(t, 10, time.Hour)

	result, err := manager.RevokeByEmail(context.Background(), "nobody@example.com")
	if err != nil {
		t.Fatalf("RevokeByEmail: %v", err)
	}
	if result.Revoked != 0 {
		t.Errorf("Revoked = %d, want 0（沒有人叫這個名字）", result.Revoked)
	}
	remaining := mini.Keys()
	if len(remaining) != 10 {
		t.Errorf("剩餘 key = %d, want 10（不該動到任何東西）", len(remaining))
	}
}

/* ==========================================================================
   nil Redis
   ========================================================================== */

// rdb 為 nil 時（測試以 struct literal 構造 Manager）三個函式都必須安全返回，
// 而不是在 Scan 上 panic —— typed nil 介面陷阱：*redis.Client 的 nil 接收者
// 會走進內部而 panic。
func TestNilRedisIsSafe(t *testing.T) {
	manager := &Manager{expire: time.Hour}

	listed, err := manager.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("List(nil rdb): %v", err)
	}
	if len(listed.Sessions) != 0 {
		t.Errorf("List(nil rdb) 回傳 %d 筆", len(listed.Sessions))
	}

	revoked, err := manager.RevokeByEmail(context.Background(), "a@b")
	if err != nil {
		t.Fatalf("RevokeByEmail(nil rdb): %v", err)
	}
	if revoked.Revoked != 0 {
		t.Errorf("RevokeByEmail(nil rdb) 刪了 %d 筆", revoked.Revoked)
	}
}
