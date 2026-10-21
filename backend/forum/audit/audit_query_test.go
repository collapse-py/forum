/*
稽核查詢層的純邏輯測試（backend/forum/audit/audit_query_test.go）。

query.go 的 MySQL 路徑由 audit_query_mysql_test.go 覆蓋；這個檔只回答
「不需要資料庫也必須成立」的性質：

 1. 分頁參數的收斂。負 offset 與天文數字 limit 是網址列直接可達的輸入，
    沒有收斂的後果是一次能把整張稽核表讀進記憶體（MaxLimit）或一次可被
    重複發動的全表掃描（MaxOffset）。
 2. WHERE 的組裝。空 Filter 必須回空字串 —— 呼叫端依賴這件事而不寫
    「if conditions != \"\" 就加 WHERE」，那個分支正是 SQL 注入最常見的破口。
 3. scanEntry 的三態：有 changes、NULL changes、壞掉的 JSON。第三態必須
    「不報錯但不顯示 diff」—— 讓整頁因為一筆壞 JSON 而載不出來更糟。
 4. Pruner 的 nil 安全。Prune 是匯出方法，*sql.DB 的 typed nil 接收者會
    走進 database/sql 內部 panic，保護不能只放在 Run 一處。
*/

package audit

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

/* ==========================================================================
   Filter：分頁收斂與 WHERE 組裝
   ========================================================================== */

func TestFilterNormalizeClampsPagination(t *testing.T) {
	cases := []struct {
		name           string
		in             Filter
		wantOffset     int
		wantLimit      int
	}{
		{"負 offset 視為 0", Filter{Offset: -5, Limit: 10}, 0, 10},
		{"offset 超過上限夾到 MaxOffset", Filter{Offset: MaxOffset + 1, Limit: 10}, MaxOffset, 10},
		{"limit 為 0 用預設", Filter{Offset: 3}, 3, DefaultLimit},
		{"負 limit 用預設", Filter{Limit: -1}, 0, DefaultLimit},
		{"limit 超過上限夾到 MaxLimit", Filter{Limit: 1_000_000}, 0, MaxLimit},
		{"合法值原樣保留", Filter{Offset: 40, Limit: 25}, 40, 25},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.in.normalize()
			if got.Offset != c.wantOffset || got.Limit != c.wantLimit {
				t.Errorf("normalize() = (offset=%d, limit=%d), want (%d, %d)",
					got.Offset, got.Limit, c.wantOffset, c.wantLimit)
			}
		})
	}
}

// 空 Filter 的 WHERE 必須是空字串（呼叫端靠它跳過 WHERE 關鍵字）；任何一個
// 條件設定後都要以 " WHERE " 開頭，且所有條件以 AND 串接。
func TestFilterWhereShape(t *testing.T) {
	if clause, args := (Filter{}).where(); clause != "" || args != nil {
		t.Errorf("空 Filter 的 where() = (%q, %v), want (\"\", nil)", clause, args)
	}

	clause, args := (Filter{
		ActorEmail:  " admin@example.com ",
		Action:      "user.suspend, post.delete",
		TargetType:  "user",
		TargetID:    "victim@example.com",
		From:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}).where()

	if !strings.HasPrefix(clause, " WHERE ") {
		t.Fatalf("條件非空時 clause 必須以 WHERE 開頭: %q", clause)
	}
	if strings.Count(clause, " AND ") != 5 {
		t.Errorf("六個條件應以五個 AND 串接: %q", clause)
	}
	// 動作多選：IN (?,?) 兩個佔位符，參數清單 2(動作) + 3(actor/targetType/targetID) + 2(時間) = 7。
	if !strings.Contains(clause, "action IN (?,?)") {
		t.Errorf("動作多選應展開成 IN 佔位符清單: %q", clause)
	}
	if len(args) != 7 {
		t.Errorf("參數數量 = %d, want 7", len(args))
	}
	if args[0] != "admin@example.com" {
		t.Errorf("ActorEmail 未修剪空白: %v", args[0])
	}
}

func TestSplitListDropsEmptyEntries(t *testing.T) {
	got := splitList(" a , ,b, , c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitList = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if got := splitList(""); len(got) != 0 {
		t.Errorf("空字串應回空清單，得到 %v", got)
	}
}

/* ==========================================================================
   scanEntry 的三態
   ========================================================================== */

// auditRow 是 scanner 的替身：把一列的值依 scanEntry 的欄位順序回出。
type auditRow struct {
	id      int64
	actor   string
	action  string
	ttype   string
	tid     string
	label   string
	changes sql.NullString
	ip      string
	reqid   string
	created string
	err     error
}

func (r auditRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 10 {
		return sql.ErrNoRows
	}
	*(dest[0].(*int64)) = r.id
	*(dest[1].(*string)) = r.actor
	*(dest[2].(*string)) = r.action
	*(dest[3].(*string)) = r.ttype
	*(dest[4].(*string)) = r.tid
	*(dest[5].(*string)) = r.label
	*(dest[6].(*sql.NullString)) = r.changes
	*(dest[7].(*string)) = r.ip
	*(dest[8].(*string)) = r.reqid
	*(dest[9].(*string)) = r.created
	return nil
}

func TestScanEntryDecodesChanges(t *testing.T) {
	entry, err := scanEntry(auditRow{
		id: 7, actor: "a@example.com", action: "post.delete", ttype: "post", tid: "12",
		label: "標題", changes: sql.NullString{String: `[{"field":"status","before":"A","after":"B"}]`, Valid: true},
		ip: "203.0.113.9", reqid: "req-1", created: "2026-10-01 12:00:00",
	})
	if err != nil {
		t.Fatalf("scanEntry: %v", err)
	}
	if entry.ID != 7 || entry.ActorEmail != "a@example.com" || len(entry.Changes) != 1 {
		t.Fatalf("scanEntry 結果 = %+v", entry)
	}
	if entry.Changes[0].Field != "status" || entry.Changes[0].Before != "A" || entry.Changes[0].After != "B" {
		t.Errorf("Changes[0] = %+v, want status/A/B", entry.Changes[0])
	}
}

// NULL 與空字串都代表「沒有 diff」：不得報錯，Changes 保持空。
func TestScanEntryToleratesMissingChanges(t *testing.T) {
	for name, changes := range map[string]sql.NullString{
		"NULL":        {Valid: false},
		"空字串":        {String: "", Valid: true},
		"壞掉的 JSON":   {String: "{not json", Valid: true},
		"不是陣列的 JSON": {String: `{"a":1}`, Valid: true},
	} {
		t.Run(name, func(t *testing.T) {
			entry, err := scanEntry(auditRow{id: 1, actor: "x", action: "y", ttype: "z", tid: "1", changes: changes})
			if err != nil {
				t.Fatalf("scanEntry(%s) 不應報錯: %v", name, err)
			}
			if len(entry.Changes) != 0 {
				t.Errorf("Changes = %+v, want 空（壞 JSON 以不顯示 diff 呈現，不該讓整頁失敗）", entry.Changes)
			}
		})
	}
}

/* ==========================================================================
   nil 安全
   ========================================================================== */

// Prune 是匯出方法：nil 接收者與 nil 資料庫都必須回 0，而不是走進
// database/sql 內部 panic（*sql.DB 的 typed nil 陷阱）。
func TestPruneNilSafety(t *testing.T) {
	var nilPruner *Pruner
	if got := nilPruner.Prune(context.Background()); got != 0 {
		t.Errorf("nil Pruner.Prune = %d, want 0", got)
	}
	p := NewPruner(nil, time.Hour, time.Hour)
	if got := p.Prune(context.Background()); got != 0 {
		t.Errorf("nil db.Prune = %d, want 0", got)
	}
}

func TestRunReturnsImmediatelyWithoutDatabase(t *testing.T) {
	done := make(chan struct{})
	var nilPruner *Pruner
	go func() { nilPruner.Run(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("nil Pruner.Run 沒有立刻返回")
	}

	p := NewPruner(nil, time.Hour, time.Hour)
	done = make(chan struct{})
	go func() { p.Run(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("nil db 的 Run 沒有立刻返回（會卡在 ticker 上）")
	}
}

// nil 資料庫的三個讀取端點都要回「空結果」而不是錯誤：監控與稽核頁在
// 未設定 DB 的開發環境必須能開（只是空的）。
func TestNilDatabaseEndpointsReturnEmpty(t *testing.T) {
	if items, err := List(context.Background(), nil, Filter{}); err != nil || items.Total != 0 || len(items.Items) != 0 {
		t.Errorf("nil db.List = (%+v, %v), want 空結果無錯誤", items, err)
	}
	if actions, err := DistinctActions(context.Background(), nil); err != nil || len(actions) != 0 {
		t.Errorf("nil db.DistinctActions = (%v, %v), want ([], nil)", actions, err)
	}
	if actors, err := DistinctActors(context.Background(), nil); err != nil || len(actors) != 0 {
		t.Errorf("nil db.DistinctActors = (%v, %v), want ([], nil)", actors, err)
	}
}

/* ==========================================================================
   NewPruner 的防禦性回退
   ========================================================================== */

// 忘記設定保留期不該讓清理變成「刪掉全部」：非正值一律回退成 90 天 / 1 小時。
func TestNewPrunerFallsBackOnNonPositiveValues(t *testing.T) {
	p := NewPruner(nil, 0, 0)
	if p.retention != 90*24*time.Hour {
		t.Errorf("retention = %v, want 90 天（非正值的防禦性回退）", p.retention)
	}
	if p.interval != time.Hour {
		t.Errorf("interval = %v, want 1h", p.interval)
	}
	p = NewPruner(nil, -time.Second, -time.Second)
	if p.retention != 90*24*time.Hour || p.interval != time.Hour {
		t.Errorf("負值未回退: (retention=%v, interval=%v)", p.retention, p.interval)
	}
}
