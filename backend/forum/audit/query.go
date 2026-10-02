/*
稽核紀錄的查詢、清理與小工具（audit/query.go）。

與 audit.go 的分工：那個檔案負責「怎麼寫」，這個檔案負責「怎麼讀、怎麼刪」。
兩者刻意分開，因為寫入路徑會出現在每一個 admin handler（熱路徑，且必須
短到不會拖慢操作），而查詢只出現在 /api/admin/log 一個地方（冷路徑，可以
放心地做多條件組合）。
*/

package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// logPrintf 是這個套件唯一的輸出途徑。
//
// 刻意不匯入 forum/logger：那會讓 audit → logger，而 logger 不依賴任何
// 專案內的套件，方向上是乾淨的；真正的問題是 logger.Init 必須在 main
// 最早呼叫，而這個套件也可能被獨立測試。log.Printf 加上明確的 "audit: "
// 前綴已經足夠辨識。
var logPrintf = log.Printf

// Filter 是 /api/admin/log 的查詢條件。零值表示「不篩」。
type Filter struct {
	// ActorEmail 精確比對操作者。刻意不支援「包含」：email 是完整的識別值，
	// 而稽核紀錄裡的 email 一定是完整值。
	ActorEmail string
	// Action 精確比對動作名稱（例如 "user.suspend"）。多選以逗號分隔，
	// 空白忽略。
	Action string
	// TargetType 精確比對資源類別。
	TargetType string
	// TargetID 精確比對對象識別值（email 或數字字串）。
	TargetID string
	// From 與 To 限定 created_at 的區間，含兩端。零值表示不設限。
	From time.Time
	To   time.Time
	// Offset 與 Limit 分頁。Limit <= 0 時套用 DefaultLimit。
	Offset int
	Limit  int
}

// DefaultLimit、MaxLimit 與 MaxOffset 是分頁的預設與上限。
//
// MaxLimit 存在的理由是「稽核表可能被撐到很大」（一筆標籤全組替換就是 21 列的
// changes JSON），而 /admin/log 是一次 GET：沒有上限的話，一個手滑的
// limit=1000000 就能把整張表讀進記憶體再序列化。
//
// MaxOffset 是給深分頁用的：OFFSET N 會讓 MySQL 丟棄前 N 列再回傳，因此
// 「翻到第 5,000 頁」的成本是 O(N) 而 O(limit)。稽核紀錄是「從最新往回翻」
// 的讀取方式，90 天 × 每天數百筆就會到幾萬列 —— 沒有上限時一個手滑的
// offset=999999 是一次可被重複發動的全表掃描。
//
// 夾到上限而不是回 400，理由與 limit 一致：超過上限的後果僅僅是「翻不到那麼
// 舊的頁」，而介面上寫得出來（回 400 會讓整個稽核頁載不出來，那個連帶損失
// 遠大於深分頁的效能問題）。
const (
	DefaultLimit = 50
	MaxLimit     = 200
	MaxOffset    = 10000
)

// where 把 Filter 轉成 SQL 的 WHERE 子句與參數清單。
//
// 回傳空字串代表沒有任何條件 —— 呼叫端因此不需要寫「if conditions != ""
// 就加上 WHERE」這種分支，而那種分支是 SQL 注入最常見的破口。
func (f Filter) where() (string, []any) {
	var conditions []string
	var args []any

	add := func(condition string, value any) {
		conditions = append(conditions, condition)
		args = append(args, value)
	}

	if email := strings.TrimSpace(f.ActorEmail); email != "" {
		add(columnActorEmail+" = ?", email)
	}
	// 動作支援逗號分隔的多選。前端把多個選取的 action 合併成一個參數，
	// 這樣不必處理動態數量的佔位符。
	if actions := splitList(f.Action); len(actions) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(actions)), ",")
		conditions = append(conditions, columnAction+" IN ("+placeholders+")")
		for _, action := range actions {
			args = append(args, action)
		}
	}
	if targetType := strings.TrimSpace(f.TargetType); targetType != "" {
		add(columnTargetType+" = ?", targetType)
	}
	if targetID := strings.TrimSpace(f.TargetID); targetID != "" {
		add(columnTargetID+" = ?", targetID)
	}
	if !f.From.IsZero() {
		add(columnCreatedAt+" >= ?", f.From.UTC())
	}
	if !f.To.IsZero() {
		add(columnCreatedAt+" <= ?", f.To.UTC())
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// splitList 以逗號切分並去掉空白與空片段。
func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// normalize 把分頁參數收斂到合法範圍。負數 offset 視為 0（沒有「從最後一筆
// 往回數」的需求，那個語意在網址列裡難以表達），超過 MaxOffset 則夾到上限。
func (f Filter) normalize() Filter {
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Offset > MaxOffset {
		f.Offset = MaxOffset
	}
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	return f
}

// ListResult 是一頁稽核紀錄與對應的總數。
type ListResult struct {
	Items []Entry `json:"items"`
	// Total 是符合條件的總筆數，不受分頁影響。前端用它顯示「第 x-y 筆，
	// 共 n 筆」並計算分頁按鈕 —— 沒有它就只能顯示「這一頁有幾筆」，
	// 那不足以回答「有沒有人動過我的文」。
	Total int `json:"total"`
}

// List 依條件查詢稽核紀錄。
//
// 排序固定為 created_at DESC, id DESC。id 作為次要鍵是必要的：同一秒內的
// 多筆記錄（一次批次停權 50 個使用者就是 50 筆）若只用時間排序，先後順序
// 會不穩定，翻頁時可能看到重複或遺漏的列。
func List(ctx context.Context, db *sql.DB, filter Filter) (ListResult, error) {
	if db == nil {
		return ListResult{Items: []Entry{}}, nil
	}
	filter = filter.normalize()
	clause, args := filter.where()

	var total int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM forum_admin_actions"+clause, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count admin actions: %w", err)
	}

	query := "SELECT " + strings.Join([]string{
		columnID, columnActorEmail, columnAction, columnTargetType, columnTargetID,
		columnTargetLabel, columnChanges, columnClientIP, columnRequestID, columnCreatedAt,
	}, ", ") + " FROM forum_admin_actions" + clause +
		" ORDER BY " + columnCreatedAt + " DESC, " + columnID + " DESC LIMIT ? OFFSET ?"

	rows, err := db.QueryContext(ctx, query, append(append([]any{}, args...), filter.Limit, filter.Offset)...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list admin actions: %w", err)
	}
	defer rows.Close()

	items := make([]Entry, 0, filter.Limit)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, entry)
	}
	return ListResult{Items: items, Total: total}, rows.Err()
}

// scanner 是 List 與 countActions 共用的最小介面，讓兩者能共用 scanEntry。
type scanner interface {
	Scan(dest ...any) error
}

// scanEntry 依 SELECT 的欄位順序讀出一筆紀錄。
//
// 欄位順序必須與 List 的 SELECT 完全一致 —— 這是宣告式程式碼常見的
// 「改了其中一邊」錯誤，因此兩處的欄位清單刻意用同一組字串常數。
func scanEntry(row scanner) (Entry, error) {
	var entry Entry
	var changes sql.NullString
	if err := row.Scan(
		&entry.ID,
		&entry.ActorEmail,
		&entry.Action,
		&entry.TargetType,
		&entry.TargetID,
		&entry.TargetLabel,
		&changes,
		&entry.ClientIP,
		&entry.RequestID,
		&entry.CreatedAt,
	); err != nil {
		return Entry{}, fmt.Errorf("scan admin action: %w", err)
	}
	// changes 允許為 NULL（沒有欄位變更的操作，例如刪除）以及空字串。
	if changes.Valid && changes.String != "" {
		// 反序列化失敗不回傳錯誤：那一筆的其他欄位仍然有用，而讓整頁
		// 因為一筆壞掉的 JSON 而載不出來更糟。改以不顯示 diff 呈現。
		_ = json.Unmarshal([]byte(changes.String), &entry.Changes)
	}
	return entry, nil
}

// maxFacets 限制篩選器選項清單的長度。
//
// 200 的依據是「這個站實際會出現多少個不同值」：動作名稱是程式裡宣告的有限
// 集合（httpapi/audit_log.go，約 25 個），操作者則是管理員帳號數。兩者都遠
// 遠低於 200，因此這個上限在實務上不會被碰到 —— 它存在是為了讓「表格長大之後
// 這兩個查詢的成本隨資料量線性上升」這件事有界限的保險，而不是承認它可以長。
//
// 之所以是上限而不是快取或對照表：稽核紀錄**不提供刪除 API**（一個能刪除
// 自己紀錄的稽核日誌等於沒有稽核日誌），所以清單只增不減、要失效時機也只有
// 寫入 —— 一個 5 分鐘 TTL 的記憶體快取能解決的問題有限，而對照表會讓
// 「稽核紀錄寫入失敗 → 操作回滾」這個不變條件多出一個必須一起回滾的寫入，
// 代價遠大於收益。
const maxFacets = 200

// DistinctActions 回傳資料庫中實際出現過的動作名稱，依名稱排序。
//
// 為什麼要從資料庫取而不是把名稱寫死在前端：寫死會讓新版本加入的動作在
// 這個頁面上無法被篩選，而「篩不到」看起來像「這件事沒發生過」—— 那正是
// 稽核紀錄最不能被誤解的地方。
//
// 這支查詢走 idx_forum_admin_actions_action，隨資料量成長仍是一個索引掃描；
// 它是每頁載入一次的成本，與頁面本身的查詢同級。
//
// 被 maxFacets 截斷時會記一行警告 —— 這是刻意的：沒有那行，截斷的症狀是
// 「篩選器突然少了幾個選項」，看起來像那些操作從來沒發生過。
func DistinctActions(ctx context.Context, db *sql.DB) ([]string, error) {
	if db == nil {
		return []string{}, nil
	}
	// 多取一個作為「是否被截斷」的判斷依據，而不必再跑一次 COUNT(*)。
	rows, err := db.QueryContext(ctx,
		"SELECT DISTINCT "+columnAction+" FROM forum_admin_actions ORDER BY "+columnAction+" LIMIT ?", maxFacets+1)
	if err != nil {
		return nil, fmt.Errorf("distinct admin actions: %w", err)
	}
	defer rows.Close()

	actions := make([]string, 0, 16)
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(actions) > maxFacets {
		actions = actions[:maxFacets]
		logPrintf("audit: 動作清單超過上限 %d，篩選器只列出前 %d 個", maxFacets, maxFacets)
	}
	return actions, nil
}

// DistinctActors 回傳資料庫中實際出現過的操作者 email，依字母排序。
//
// 同樣是為了讓「這條路徑上有誰動過手」能從篩選器直接點，而不是要求管理員
// 逐字輸入 email（打錯一個字元的結果是「查無紀錄」，那看起來像清白的證明）。
//
// 走 idx_forum_admin_actions_actor (actor_email, created_at)，DISTINCT 打在
// 索引最左前綴上，MySQL 可以用 loose index scan，因此這支查詢不會像全表掃描
// 那樣隨資料量線性惡化 —— 但結果**數量**仍然沒有上界，所以還是套上 maxFacets。
func DistinctActors(ctx context.Context, db *sql.DB) ([]string, error) {
	if db == nil {
		return []string{}, nil
	}
	rows, err := db.QueryContext(ctx,
		"SELECT DISTINCT "+columnActorEmail+" FROM forum_admin_actions ORDER BY "+columnActorEmail+" LIMIT ?", maxFacets+1)
	if err != nil {
		return nil, fmt.Errorf("distinct admin actors: %w", err)
	}
	defer rows.Close()

	actors := make([]string, 0, 16)
	for rows.Next() {
		var actor string
		if err := rows.Scan(&actor); err != nil {
			return nil, err
		}
		actors = append(actors, actor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(actors) > maxFacets {
		actors = actors[:maxFacets]
		logPrintf("audit: 操作者清單超過上限 %d，篩選器只列出前 %d 個", maxFacets, maxFacets)
	}
	return actors, nil
}

// Pruner 定期刪除過期的稽核紀錄。
type Pruner struct {
	db        *sql.DB
	retention time.Duration
	interval  time.Duration
}

// NewPruner 建立清理器。retention <= 0 會被呼叫端（config）擋掉，這裡仍
// 保留防禦性回退，因為忘記設定不該讓清理變成「刪掉全部」。
func NewPruner(db *sql.DB, retention, interval time.Duration) *Pruner {
	if retention <= 0 {
		retention = 90 * 24 * time.Hour
	}
	if interval <= 0 {
		interval = time.Hour
	}
	return &Pruner{db: db, retention: retention, interval: interval}
}

// Run 定期執行清理，直到 ctx 被取消。
//
// 為什麼是「按 ctx 取消」而不是一個 Stop() 方法：與 metrics 套件的 flusher
// 同一個理由 —— 未來接上 signal.Notify 時，context 是唯一已經接好的取消管道。
func (p *Pruner) Run(ctx context.Context) {
	if p == nil || p.db == nil {
		return
	}
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.Prune(ctx)
		}
	}
}

// Prune 刪除 created_at 早於 cutoff 的紀錄，回傳刪除筆數。
//
// nil 資料庫直接回 0。這是「typed nil」的典型陷阱：*sql.DB 的 nil 接收者
// 呼叫方法時不會回傳錯誤，而是走進 database/sql 內部對 connMu 解鎖而
// panic。Run 有擋，但 Prune 是匯出的方法（測試與未來的呼叫端會直接呼叫），
// 保護不能只放在一處。
//
// 錯誤只記不傳：清理失敗的後果是「表長大一點點」，而下一次 tick 會再試。
// 讓它沿著呼叫鏈往上報只會讓呼叫端多一段沒有處理選項的錯誤分支。
//
// 批次刪除（DELETE … LIMIT）在這裡刻意不用：一次刪掉幾十萬列會讓交易
// 與 undo log 膨脹，而這是個可以慢慢做的清理工作。分成每 tick 固定上限
// 的批次，讓它每次 tick 多花一點時間就換到不會有任何單一長時間的鎖。
func (p *Pruner) Prune(ctx context.Context) int {
	if p == nil || p.db == nil {
		return 0
	}

	const batchSize = 5000

	cutoff := time.Now().Add(-p.retention)
	removed := 0
	for {
		result, err := p.db.ExecContext(ctx,
			"DELETE FROM forum_admin_actions WHERE "+columnCreatedAt+" < ? LIMIT ?", cutoff.UTC(), batchSize)
		if err != nil {
			logPrintf("audit: 無法清理過期紀錄（保留 %v）: %v", p.retention, err)
			return removed
		}
		affected, err := result.RowsAffected()
		if err != nil {
			logPrintf("audit: 無法取得清理筆數: %v", err)
			return removed
		}
		removed += int(affected)
		if int(affected) < batchSize {
			return removed
		}
	}
}
