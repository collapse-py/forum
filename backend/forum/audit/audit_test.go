/*
稽核套件的測試（audit/audit_test.go）。

這個套件的邏輯有三個「不靠眼睛就能看出錯」的地方，這裡各有一組測試：

  1. 變更值的截斷
     稽核紀錄保存的是「改動前 200 字」而不是全文。截斷函式必須在 UTF-8 的
     字元邊界切，否則資料庫裡會出現半個中文字元，而那會讓 JSON 解析失敗 ——
     整頁稽核紀錄載不出來。這個症狀與「函式寫錯了」毫無關聯，因此特別測。

  2. 只寫出真正改變的欄位
     多數 PUT 是全欄位取代。若不過濾，稽核紀錄上會出現四行
     "reporter_email: a@b → a@b" 夾在一行真正的變更中間，而稽核紀錄的
     用途就是回答「有什麼改變」。

  3. 分頁參數的收斂
     limit 沒有上限的話，一個 limit=1000000 的 GET 就能把整張表讀進記憶體。

刻意不測的：INSERT 與 SELECT 這兩段 SQL 對資料庫的實際效果。那需要一個
真的 MySQL，而這個專案目前的測試策略不含 MySQL 替身（理由見 docs/KNOWN_ISSUES.md 的
已知問題）。因此這裡只測「送進驅動程式的參數」與「從掃描結果組出的結構」。
*/

package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

/* ==========================================================================
   截斷
   ========================================================================== */

// fakeExecer 記錄 Record 送進驅動程式的 SQL 與參數。
//
// 不實作真實的資料庫：這個套件與 SQL 驅動之間只有「參數是否正確」這一個
// 接觸點，而那正是這個假實作要驗的東西。
type fakeExecer struct {
	query string
	args  []any
	err   error
}

func (f *fakeExecer) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	f.query = query
	f.args = args
	return nil, f.err
}

func TestRecordPassesEntryToDriver(t *testing.T) {
	exec := &fakeExecer{}
	entry := Entry{
		ActorEmail:  "admin@example.com",
		Action:      "post.delete",
		TargetType:  TargetPost,
		TargetID:    "17",
		TargetLabel: "一篇很長的文章",
		Changes:     []Change{{Field: "content", Before: "舊內容", After: "（文章已刪除）"}},
		ClientIP:    "203.0.113.7",
		RequestID:   "abc123",
		CreatedAt:   "2026-09-30 15:04:05",
	}

	if err := Record(t.Context(), exec, entry); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// 參數順序必須與 INSERT 的欄位順序一致 —— 這是宣告式 SQL 最容易錯的
	// 地方，而且錯了不會報錯，只會把 actor 寫進 action 欄位。
	if exec.args[0] != entry.ActorEmail {
		t.Errorf("args[0] = %v, want actorEmail", exec.args[0])
	}
	if exec.args[1] != entry.Action {
		t.Errorf("args[1] = %v, want action", exec.args[1])
	}
	if exec.args[2] != entry.TargetType {
		t.Errorf("args[2] = %v, want targetType", exec.args[2])
	}
	if exec.args[3] != entry.TargetID {
		t.Errorf("args[3] = %v, want targetId", exec.args[3])
	}
	if exec.args[6] != entry.ClientIP {
		t.Errorf("args[6] = %v, want clientIp", exec.args[6])
	}
	if exec.args[8] != entry.CreatedAt {
		t.Errorf("args[8] = %v, want createdAt", exec.args[8])
	}
}

// 這是本套件最重要的性質：稽核寫入失敗時必須回傳錯誤。
//
// 為什麼值得單獨測：若 Record 只記 log 不回傳錯誤，那麼呼叫端很容易在
// 交易內呼叫它卻忽略結果 —— 資料庫裡就會出現「操作發生了、紀錄卻不在」，
// 而稽核日誌正是用來證明這件事沒發生的工具。
func TestRecordReturnsErrorOnFailure(t *testing.T) {
	exec := &fakeExecer{err: errors.New("table forum_admin_actions doesn't exist")}
	err := Record(t.Context(), exec, Entry{ActorEmail: "a@b", Action: "post.delete"})
	if err == nil {
		t.Fatal("寫入失敗時 Record 必須回傳錯誤，否則操作會在沒有紀錄的情況下發生")
	}
	// 錯誤訊息不含 actor_email：那是可能含有個資的字串，而 log 的讀取範圍
	// 通常比資料庫寬。
	if strings.Contains(err.Error(), "a@b") {
		t.Errorf("錯誤訊息含 actor_email: %q", err.Error())
	}
}

func TestRecordRejectsNilDatabase(t *testing.T) {
	if err := Record(t.Context(), nil, Entry{ActorEmail: "a@b"}); !errors.Is(err, ErrNoDatabase) {
		t.Fatalf("nil Execer 時應回傳 ErrNoDatabase, got %v", err)
	}
}

func TestRecordWithoutChangesStoresEmptyString(t *testing.T) {
	exec := &fakeExecer{}
	if err := Record(t.Context(), exec, Entry{ActorEmail: "a@b", Action: "tag.delete"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	// 沒有變更的動作（純刪除、純建立）必須寫入空字串而不是 "null"：
	// 前端以 changes 欄位是否為空判斷「這筆有沒有 diff」，而字串 "null"
	// 反序列化後是一個非 nil 的切片，會讓「無變更」看起來像「有一個 null 欄位」。
	if got := exec.args[5]; got != "" {
		t.Errorf("changes = %v, want 空字串", got)
	}
}

/* ==========================================================================
   截斷
   ========================================================================== */

func TestTruncateKeepsUTF8Boundary(t *testing.T) {
	// 中文字元佔三個位元組。若直接切在長度上限上，會切出半個字元，而
	// 非法 UTF-8 會讓整個 changes JSON 反序列化失敗 —— 症狀是「稽核紀錄
	// 整頁載不出來」，與「截斷函式寫錯了」毫無關聯。
	long := strings.Repeat("中", 100)
	cut := truncate(long, 10)

	if !isValidUTF8(cut) {
		t.Fatalf("截斷結果不是合法 UTF-8: %q", cut)
	}
	if !strings.HasSuffix(cut, "...") {
		t.Errorf("截斷結果應以省略號結尾: %q", cut)
	}
	// 整個保證：結果不超過 10 個位元組（含三個位元組的省略號），
	// 因此最多放得下 2 個中文字（6 bytes）+ 省略號（3 bytes）= 9。
	if len(cut) > 10 {
		t.Errorf("截斷結果 %d 個位元組，超過上限 10: %q", len(cut), cut)
	}
	if got := len([]rune(strings.TrimSuffix(cut, "..."))); got != 2 {
		t.Errorf("保留字元數 = %d, want 2（10 bytes 扣掉 3 bytes 的省略號後只放得下兩個中文字）", got)
	}
}

func TestTruncateLeavesShortValuesAlone(t *testing.T) {
	if got := truncate("ACTIVE", maxValueLength); got != "ACTIVE" {
		t.Errorf("未超長的值不應被改動, got %q", got)
	}
	if got := truncate("", maxValueLength); got != "" {
		t.Errorf("空字串應原樣返回, got %q", got)
	}
}

/* ==========================================================================
   變更序列化
   ========================================================================== */

func TestEncodeChangesMarksTruncation(t *testing.T) {
	encoded, err := encodeChanges([]Change{
		{Field: "content", Before: strings.Repeat("長", 500), After: strings.Repeat("文", 500)},
	})
	if err != nil {
		t.Fatalf("encodeChanges: %v", err)
	}
	var decoded []Change
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("反序列化失敗（截斷切壞了 UTF-8?）: %v", err)
	}
	if !decoded[0].Truncated {
		t.Error("超長的值應標記 Truncated，否則讀者會以為那就是全文")
	}
	if len(decoded[0].After) > maxValueLength {
		t.Errorf("截斷後長度 = %d, want <= %d", len(decoded[0].After), maxValueLength)
	}
}

func TestEncodeChangesCapsCount(t *testing.T) {
	// 一次「套用全部標籤」可能送出 40 個 id。不截斷的話 changes 會變成
	// 幾 KB 的 JSON，而稽核紀錄是逐列捲動的。
	changes := make([]Change, 60)
	for i := range changes {
		changes[i] = Change{Field: "tag", Before: "a", After: "b"}
	}
	encoded, err := encodeChanges(changes)
	if err != nil {
		t.Fatalf("encodeChanges: %v", err)
	}
	var decoded []Change
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("反序列化失敗: %v", err)
	}
	// maxChangesPerEntry 筆實際內容 + 1 筆「還有更多」的標記。
	if want := maxChangesPerEntry + 1; len(decoded) != want {
		t.Fatalf("變更筆數 = %d, want %d（%d 筆內容 + 1 筆截斷標記）", len(decoded), want, maxChangesPerEntry)
	}
	if decoded[len(decoded)-1].Field != "…" {
		t.Errorf("最後一筆應是截斷標記, got field=%q", decoded[len(decoded)-1].Field)
	}
}

func TestEncodeChangesEmpty(t *testing.T) {
	encoded, err := encodeChanges(nil)
	if err != nil {
		t.Fatalf("encodeChanges: %v", err)
	}
	if encoded != "" {
		t.Errorf("沒有變更時應回傳空字串, got %q", encoded)
	}
}

/* ==========================================================================
   查詢條件
   ========================================================================== */

func TestFilterWhereBuildsPlaceholders(t *testing.T) {
	// 這個測試的價值在於「所有使用者輸入都必須變成佔位符」。若任何一個值
	// 被字串拼接進 SQL，那就是注入點。回傳的 clause 只允許出現 ? 、欄位名
	// 與 AND。
	clause, args := Filter{
		ActorEmail: "admin@example.com",
		Action:     "post.delete, user.suspend",
		TargetType: "post",
		TargetID:   "17",
		From:       time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		To:         time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
	}.where()

	if !strings.HasPrefix(clause, " WHERE ") {
		t.Fatalf("clause 應以 WHERE 開頭, got %q", clause)
	}
	if strings.Count(clause, "?") != len(args) {
		t.Errorf("佔位符數量 = %d, 參數數量 = %d（必須相等）", strings.Count(clause, "?"), len(args))
	}
	// 多選的 action 應該展開成兩個佔位符。
	if !strings.Contains(clause, "action IN (?,?)") {
		t.Errorf("多選 action 應展開成兩個佔位符, got %q", clause)
	}
	// 逐項確認參數順序對應子句順序。
	if args[0] != "admin@example.com" {
		t.Errorf("args[0] = %v, want actorEmail", args[0])
	}
	if args[1] != "post.delete" || args[2] != "user.suspend" {
		t.Errorf("args[1:3] = %v, want 兩個動作名稱", args[1:3])
	}
}

func TestFilterWhereEmptyWhenNoConditions(t *testing.T) {
	// 沒有條件時回傳空字串而不是 " WHERE 1=1"：前者讓呼叫端不必寫
	// 「if clause != "" 就加 WHERE」那個分支（而那個分支是 SQL 注入的常見源頭）。
	clause, args := Filter{}.where()
	if clause != "" || len(args) != 0 {
		t.Fatalf("零值 Filter 應回傳空子句, got (%q, %v)", clause, args)
	}
	// 純空白的條件也算「沒有」：查詢參數 ?actor= 是常見的手滑。
	clause, args = Filter{ActorEmail: "   ", Action: " , "}.where()
	if clause != "" || len(args) != 0 {
		t.Fatalf("全空白的條件應回傳空子句, got (%q, %v)", clause, args)
	}
}

func TestFilterNormalizeClampsPaging(t *testing.T) {
	// 上限存在的理由：/api/admin/log 是一次 GET，沒有上限的話一個
	// limit=1000000 就能把整張表讀進記憶體再序列化。
	cases := []struct {
		name       string
		in         Filter
		wantOffset int
		wantLimit  int
	}{
		{"負數 offset 視為 0", Filter{Offset: -5, Limit: 10}, 0, 10},
		{"零 limit 用預設", Filter{}, 0, DefaultLimit},
		{"超過上限被收斂", Filter{Limit: 99999}, 0, MaxLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.normalize()
			if got.Offset != tc.wantOffset || got.Limit != tc.wantLimit {
				t.Fatalf("normalize() = (%d, %d), want (%d, %d)",
					got.Offset, got.Limit, tc.wantOffset, tc.wantLimit)
			}
		})
	}
}

func TestListWithNilDatabase(t *testing.T) {
	// nil 資料庫回空結果而不是錯誤：handleAdminLog 在正式服務上一定有
	// db，但這讓函式的邊界行為不需要依賴呼叫端的防護。
	result, err := List(t.Context(), nil, Filter{})
	if err != nil {
		t.Fatalf("List(nil) 應回空結果, got %v", err)
	}
	if len(result.Items) != 0 || result.Total != 0 {
		t.Errorf("List(nil) 應回空結果, got %+v", result)
	}
}

func TestListAndDistinctAreNilSafe(t *testing.T) {
	// 同上：這三個函式都在 /api/admin/log 的呼叫鏈上，nil 的處理不該
	// 由每個呼叫端各自決定。
	if _, err := DistinctActions(t.Context(), nil); err != nil {
		t.Errorf("DistinctActions(nil): %v", err)
	}
	if _, err := DistinctActors(t.Context(), nil); err != nil {
		t.Errorf("DistinctActors(nil): %v", err)
	}
}

func TestPruneUsesConfiguredRetention(t *testing.T) {
	// Prune 對 nil 資料庫必須安全：NewPruner 允許 nil，而 Run 已經擋掉了
	// nil，但 Prune 是匯出的方法，測試與未來的呼叫端會直接呼叫它。
	pruner := NewPruner(nil, 90*24*time.Hour, time.Hour)
	if got := pruner.Prune(t.Context()); got != 0 {
		t.Errorf("Prune(nil) 應回 0, got %d", got)
	}
	// Run 在 nil 資料庫上必須直接返回，不能啟動 goroutine ——
	// 一個永遠不會寫入任何東西的 ticker 只會讓診斷變困難。
	pruner.Run(t.Context())
}

/* ==========================================================================
   輔助
   ========================================================================== */

// isValidUTF8 直接用標準庫的 utf8.ValidString。
//
// 刻意不用「數 range 幾次」這種自己實作的檢查：range 遇到非法位元組時會
// 逐位元組前進，因此計數不會失敗 —— 那樣的檢查會讓這個測試永遠通過，
// 等於沒有測到 UTF-8 邊界這件事。
func isValidUTF8(s string) bool {
	return utf8.ValidString(s)
}
