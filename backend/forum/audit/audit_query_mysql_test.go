/*
稽核查詢的真實 MySQL 測試（backend/forum/audit/audit_query_mysql_test.go）。

List / DistinctActions / DistinctActors / Prune 的正確性是「SELECT 出來的列
到底對不對」——SQL 字串形狀的斷言回答不了這個問題。約定與 data/migration_test
相同：FORUM_TEST_MYSQL_DSN 指向伺服器，每個測試自建自丟拋棄式資料庫；
未設定時 skip。

驗的是四件事：

 1. 篩選與分頁真的作用：actor / action 多選 / target / 時間區間各自要能
    篩出正確的列，Total 是**分頁前**的總數，排序是 created_at DESC, id DESC
    （次要鍵缺失時同秒內的多筆會在翻頁間跳動）。
 2. 面板清單（DistinctActions / DistinctActors）依序回傳資料庫中實際出現過
    的值，且超過 maxFacets 時截斷並記一行警告 —— 沒有那行警告，截斷的症狀
    是「篩選器少了幾個選項」，看起來像那些操作從來沒發生過。
 3. Prune 只刪過期列，保留期內的一列都不動。
 4. Run 的 ticker 真的會定期清理，且 ctx 取消後停止。
*/

package audit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"forum/forum/idgen"

	_ "github.com/go-sql-driver/mysql"
)

var auditDBCounter atomic.Int64

// auditServerDSN 剝成「只連伺服器」的 DSN，未設定 FORUM_TEST_MYSQL_DSN 時 skip。
func auditServerDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("FORUM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FORUM_TEST_MYSQL_DSN 未設定：跳過需要真實 MySQL 的稽核查詢測試。" +
			"設定方式見 .github/workflows/ci.yml 的 migrations job")
	}
	cleaned := dsn
	if i := strings.LastIndex(cleaned, "/"); i >= 0 {
		if j := strings.Index(cleaned[i:], "?"); j >= 0 {
			cleaned = cleaned[:i] + "/" + cleaned[i+j:]
		} else {
			cleaned = cleaned[:i] + "/"
		}
	}
	return cleaned
}

// auditTestDB 建立含 forum_admin_actions 的拋棄式資料庫。
// schema 內聯對齊 MigrateMySQL 第 23 步：這組測試只需要那一張表。
func auditTestDB(t *testing.T) *sql.DB {
	t.Helper()
	serverLevel := auditServerDSN(t)

	server, err := sql.Open("mysql", serverLevel)
	if err != nil {
		t.Fatalf("開啟伺服器連線失敗: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if err := server.Ping(); err != nil {
		t.Fatalf("MySQL 連不上: %v", err)
	}

	name := fmt.Sprintf("forum_audit_test_%d_%d", time.Now().UnixNano(), auditDBCounter.Add(1))
	if _, err := server.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("建立測試資料庫失敗: %v", err)
	}
	t.Cleanup(func() {
		if _, err := server.Exec("DROP DATABASE IF EXISTS `" + name + "`"); err != nil {
			t.Logf("清理測試資料庫 %s 失敗（請手動清理）: %v", name, err)
		}
	})

	base, params := serverLevel, ""
	if q := strings.Index(serverLevel, "?"); q >= 0 {
		base, params = serverLevel[:q], serverLevel[q:]
	}
	db, err := sql.Open("mysql", strings.TrimSuffix(base, "/")+"/"+name+params)
	if err != nil {
		t.Fatalf("連到測試資料庫失敗: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE forum_admin_actions (
		id BIGINT NOT NULL,
		actor_email VARCHAR(320) NOT NULL,
		action VARCHAR(64) NOT NULL,
		target_type VARCHAR(32) NOT NULL,
		target_id VARCHAR(320) NOT NULL,
		target_label VARCHAR(320) NOT NULL DEFAULT '',
		changes TEXT NULL,
		ip VARCHAR(64) NOT NULL DEFAULT '',
		request_id VARCHAR(64) NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		PRIMARY KEY (id),
		KEY idx_forum_admin_actions_created (created_at),
		KEY idx_forum_admin_actions_actor (actor_email, created_at),
		KEY idx_forum_admin_actions_target (target_type, target_id),
		KEY idx_forum_admin_actions_action (action)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		t.Fatalf("建立 forum_admin_actions 失敗: %v", err)
	}
	return db
}

// insertAudit 寫一筆稽核紀錄（走 Record，順帶覆蓋寫入路徑）。
// created 為 UTC 的 DATETIME 字面值，與 Filter.From/To 的 UTC 轉換對齊。
func insertAudit(t *testing.T, db *sql.DB, actor, action, targetType, targetID, created string) {
	t.Helper()
	if err := Record(context.Background(), db, Entry{
		ID:         idgen.Next(),
		ActorEmail: actor, Action: action, TargetType: targetType, TargetID: targetID,
		ClientIP: "192.0.2.1", RequestID: "req-" + targetID, CreatedAt: created,
	}); err != nil {
		t.Fatalf("準備稽核列失敗（%s %s）: %v", action, targetID, err)
	}
}

/* ==========================================================================
   List：篩選、排序、分頁
   ========================================================================== */

func TestListFiltersSortsAndPaginatesMySQL(t *testing.T) {
	db := auditTestDB(t)
	ctx := context.Background()

	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	stamp := func(minute int) string {
		return base.Add(time.Duration(minute) * time.Minute).Format("2006-01-02 15:04:05")
	}

	insertAudit(t, db, "alice@example.com", "user.suspend", "user", "u1@example.com", stamp(0))
	insertAudit(t, db, "alice@example.com", "post.delete", "post", "41", stamp(1))
	insertAudit(t, db, "bob@example.com", "user.suspend", "user", "u2@example.com", stamp(2))
	insertAudit(t, db, "bob@example.com", "post.delete", "post", "42", stamp(2)) // 同一秒的第二筆
	insertAudit(t, db, "bob@example.com", "tag.assign", "user", "u3@example.com", stamp(5))

	t.Run("無篩選：全部、由新到舊、同秒以 id 倒序", func(t *testing.T) {
		result, err := List(ctx, db, Filter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if result.Total != 5 || len(result.Items) != 5 {
			t.Fatalf("Total=%d len=%d, want 5/5", result.Total, len(result.Items))
		}
		// stamp(2) 的兩筆（id 3、4）必須 id 大者在前。
		if result.Items[0].ID < result.Items[1].ID {
			t.Errorf("同秒內沒有以 id 倒序: id[0]=%d id[1]=%d", result.Items[0].ID, result.Items[1].ID)
		}
		if result.Items[0].CreatedAt < result.Items[1].CreatedAt {
			t.Errorf("排序不是 created_at DESC: %s, %s", result.Items[0].CreatedAt, result.Items[1].CreatedAt)
		}
	})

	t.Run("actor 篩選", func(t *testing.T) {
		result, err := List(ctx, db, Filter{ActorEmail: "alice@example.com"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("Total = %d, want 2", result.Total)
		}
	})

	t.Run("action 多選（逗號分隔）", func(t *testing.T) {
		result, err := List(ctx, db, Filter{Action: "user.suspend, tag.assign"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if result.Total != 3 {
			t.Errorf("Total = %d, want 3（2 筆 suspend + 1 筆 assign）", result.Total)
		}
	})

	t.Run("target 篩選", func(t *testing.T) {
		result, err := List(ctx, db, Filter{TargetType: "post", TargetID: "41"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if result.Total != 1 || result.Items[0].TargetID != "41" {
			t.Errorf("Total=%d items=%+v, want 1 筆 post/41", result.Total, result.Items)
		}
	})

	t.Run("時間區間含兩端", func(t *testing.T) {
		result, err := List(ctx, db, Filter{From: base, To: base.Add(2 * time.Minute)})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if result.Total != 4 {
			t.Errorf("Total = %d, want 4（stamp 0,1,2,2）", result.Total)
		}
	})

	t.Run("分頁不影響 Total，超界回空頁", func(t *testing.T) {
		result, err := List(ctx, db, Filter{Limit: 2, Offset: 1})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(result.Items) != 2 || result.Total != 5 {
			t.Errorf("len=%d Total=%d, want 2/5（Total 是分頁前的總數）", len(result.Items), result.Total)
		}
		empty, err := List(ctx, db, Filter{Limit: 2, Offset: 100})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(empty.Items) != 0 || empty.Total != 5 {
			t.Errorf("深分頁空頁 len=%d Total=%d, want 0/5", len(empty.Items), empty.Total)
		}
	})

	t.Run("天文數字參數被收斂而不是回錯誤", func(t *testing.T) {
		result, err := List(ctx, db, Filter{Limit: 10_000_000, Offset: -3})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(result.Items) != 5 {
			t.Errorf("收斂後應回所有 5 筆，得到 %d", len(result.Items))
		}
	})

	t.Run("changes 往返：寫進去的 diff 讀得回來", func(t *testing.T) {
		if err := Record(ctx, db, Entry{
			ID:         idgen.Next(),
			ActorEmail: "alice@example.com", Action: "user.tag", TargetType: "user", TargetID: "u9",
			Changes:   []Change{{Field: "tags", Before: "a", After: "b,c"}},
			CreatedAt: stamp(9),
		}); err != nil {
			t.Fatalf("Record with changes: %v", err)
		}
		result, err := List(ctx, db, Filter{Action: "user.tag"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(result.Items) != 1 || len(result.Items[0].Changes) != 1 {
			t.Fatalf("changes 沒有往返: %+v", result.Items)
		}
		if c := result.Items[0].Changes[0]; c.Field != "tags" || c.Before != "a" || c.After != "b,c" {
			t.Errorf("Changes[0] = %+v, want tags/a/b,c", c)
		}
	})
}

/* ==========================================================================
   面板清單
   ========================================================================== */

func TestDistinctFacetsMySQL(t *testing.T) {
	db := auditTestDB(t)
	ctx := context.Background()

	insertAudit(t, db, "alice@example.com", "user.suspend", "user", "1", "2026-10-01 12:00:00")
	insertAudit(t, db, "bob@example.com", "post.delete", "post", "2", "2026-10-01 12:00:01")
	insertAudit(t, db, "alice@example.com", "user.suspend", "user", "3", "2026-10-01 12:00:02") // 重複動作
	insertAudit(t, db, "carol@example.com", "tag.assign", "user", "4", "2026-10-01 12:00:03")

	actions, err := DistinctActions(ctx, db)
	if err != nil {
		t.Fatalf("DistinctActions: %v", err)
	}
	want := []string{"post.delete", "tag.assign", "user.suspend"} // 依名稱排序
	if len(actions) != len(want) {
		t.Fatalf("actions = %v, want %v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Errorf("actions[%d] = %q, want %q", i, actions[i], want[i])
		}
	}

	actors, err := DistinctActors(ctx, db)
	if err != nil {
		t.Fatalf("DistinctActors: %v", err)
	}
	wantActors := []string{"alice@example.com", "bob@example.com", "carol@example.com"}
	if len(actors) != len(wantActors) {
		t.Fatalf("actors = %v, want %v", actors, wantActors)
	}
	for i := range wantActors {
		if actors[i] != wantActors[i] {
			t.Errorf("actors[%d] = %q, want %q", i, actors[i], wantActors[i])
		}
	}
}

// 超過 maxFacets 時截斷到 maxFacets，且必須記一行警告 —— 沒有那行，截斷的
// 症狀是「篩選器少了幾個選項」，看起來像那些操作從來沒發生過。
func TestDistinctActionsTruncatesWithWarningMySQL(t *testing.T) {
	db := auditTestDB(t)
	ctx := context.Background()

	rows := make([]string, 0, maxFacets+5)
	values := make([]string, 0, maxFacets+5)
	for i := 0; i < maxFacets+5; i++ {
		// id 由應用層給：schema 與 MigrateMySQL 第 23 步一致，沒有 AUTO_INCREMENT。
		rows = append(rows, "(?, 'x@example.com', ?, 'user', '1', '', NULL, '', '', '2026-10-01 12:00:00')")
		values = append(values, fmt.Sprintf("%d", idgen.Next()), fmt.Sprintf("action.%04d", i))
	}
	query := "INSERT INTO forum_admin_actions (id, actor_email, action, target_type, target_id, target_label, changes, ip, request_id, created_at) VALUES " +
		strings.Join(rows, ",")
	if _, err := db.ExecContext(ctx, query, expand(values)...); err != nil {
		t.Fatalf("準備大量動作失敗: %v", err)
	}

	// 替換 logPrintf 以擷取警告（套件內唯一的輸出途徑是個變數）。
	var mu sync.Mutex
	var warnings []string
	orig := logPrintf
	logPrintf = func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	t.Cleanup(func() { logPrintf = orig })

	actions, err := DistinctActions(ctx, db)
	if err != nil {
		t.Fatalf("DistinctActions: %v", err)
	}
	if len(actions) != maxFacets {
		t.Errorf("截斷後長度 = %d, want %d", len(actions), maxFacets)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "超過上限") {
		t.Errorf("截斷沒有記警告: %v", warnings)
	}
}

// expand 把字串值清單攤平成參數清單（每個 ? 對應一個值）。
func expand(values []string) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
}

/* ==========================================================================
   Prune / Run
   ========================================================================== */

func TestPruneRemovesOnlyExpiredMySQL(t *testing.T) {
	db := auditTestDB(t)
	ctx := context.Background()

	old := time.Now().UTC().Add(-100 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	recent := time.Now().UTC().Add(-time.Hour).Format("2006-01-02 15:04:05")
	for i := 0; i < 3; i++ {
		insertAudit(t, db, "a@example.com", "post.delete", "post", fmt.Sprint(i), old)
		insertAudit(t, db, "a@example.com", "post.delete", "post", fmt.Sprint(100+i), recent)
	}

	p := NewPruner(db, 90*24*time.Hour, time.Hour)
	removed := p.Prune(ctx)
	if removed != 3 {
		t.Errorf("Prune 刪除 %d 筆, want 3（只刪早於保留期的）", removed)
	}

	var remaining int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM forum_admin_actions").Scan(&remaining); err != nil {
		t.Fatalf("COUNT 失敗: %v", err)
	}
	if remaining != 3 {
		t.Errorf("剩餘 %d 筆, want 3（保留期內的一列都不該被刪）", remaining)
	}

	// 冪等：再跑一次沒有東西可刪。
	if got := p.Prune(ctx); got != 0 {
		t.Errorf("第二次 Prune 刪了 %d 筆, want 0", got)
	}
}

// Run 的 ticker 真的會定期清理；ctx 取消後停止。
func TestRunPrunesOnTickerMySQL(t *testing.T) {
	db := auditTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	old := time.Now().UTC().Add(-200 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	insertAudit(t, db, "a@example.com", "post.delete", "post", "1", old)

	p := NewPruner(db, 90*24*time.Hour, 20*time.Millisecond)
	done := make(chan struct{})
	go func() { p.Run(ctx); close(done) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var remaining int
		if err := db.QueryRow("SELECT COUNT(*) FROM forum_admin_actions").Scan(&remaining); err != nil {
			t.Fatalf("COUNT 失敗: %v", err)
		}
		if remaining == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("等待 Run 的定期清理超時")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ctx 取消後 Run 沒有返回")
	}
}
