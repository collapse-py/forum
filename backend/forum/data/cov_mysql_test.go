/*
不需要真實 MySQL 的 data 套件測試（backend/forum/data/cov_mysql_test.go）。

這裡註冊一個極小的 database/sql/driver 替身，讓 MigrateMySQL 的完整線性流程
（35 步、數十句 DDL）在沒有 MySQL 的環境裡也能被執行：

  - QueryRow：依「SQL 片段」回傳登記的結果集；沒登記就回空結果集
    （對 Scan 而言是 sql.ErrNoRows）。
  - Exec：預設成功（LastInsertId=1、RowsAffected=1）；登記的片段可回錯誤。
  - BeginTx / Commit / Rollback：全部成功。

它與 sqlmock 的差異是不要求「每一筆查詢都被預期」、也不檢查順序；因此
MigrateMySQL 走到哪一步都不會因為少登記一筆查詢而失敗。真正的 MySQL 行為
（DDL 語法、冪等性）仍由 migration_test.go 在 FORUM_TEST_MYSQL_DSN 下驗證。
*/

package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

/* ==========================================================================
   驅動替身
   ========================================================================== */

type covDataValue = driver.Value

type covDataQueryRule struct {
	substr string
	cols   []string
	rows   [][]covDataValue
	err    error
}

type covDataExecRule struct {
	substr string
	err    error
}

type covDataSource struct {
	mu      sync.Mutex
	queries []covDataQueryRule
	execs   []covDataExecRule
	seen    []string
}

func (s *covDataSource) addQuery(rule covDataQueryRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, rule)
}

func (s *covDataSource) addExec(rule covDataExecRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.execs = append(s.execs, rule)
}

func (s *covDataSource) queryCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queries)
}

func (s *covDataSource) saw(substr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, query := range s.seen {
		if strings.Contains(query, substr) {
			return true
		}
	}
	return false
}

func (s *covDataSource) countSeen(substr string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, query := range s.seen {
		if strings.Contains(query, substr) {
			n++
		}
	}
	return n
}

var (
	covDataSeq      uint64
	covDataRegistry sync.Map
	covDataOnce     sync.Once
)

type covDataDriver struct{}

func (covDataDriver) Open(name string) (driver.Conn, error) {
	value, ok := covDataRegistry.Load(name)
	if !ok {
		return nil, fmt.Errorf("cov data driver: unknown dsn %q", name)
	}
	return &covDataConn{src: value.(*covDataSource)}, nil
}

type covDataConn struct{ src *covDataSource }

func (c *covDataConn) Prepare(string) (driver.Stmt, error) { return covDataStmt{}, nil }
func (c *covDataConn) Close() error                        { return nil }
func (c *covDataConn) Begin() (driver.Tx, error)           { return covDataTx{}, nil }
func (c *covDataConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return covDataTx{}, nil
}
func (c *covDataConn) Ping(context.Context) error { return nil }

func (c *covDataConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)
	for _, rule := range c.src.queries {
		if strings.Contains(query, rule.substr) {
			if rule.err != nil {
				return nil, rule.err
			}
			return &covDataRows{cols: rule.cols, rows: rule.rows}, nil
		}
	}
	return &covDataRows{}, nil
}

func (c *covDataConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)
	for _, rule := range c.src.execs {
		if strings.Contains(query, rule.substr) {
			if rule.err != nil {
				return nil, rule.err
			}
			break
		}
	}
	return covDataResult{}, nil
}

type covDataTx struct{}

func (covDataTx) Commit() error   { return nil }
func (covDataTx) Rollback() error { return nil }

type covDataStmt struct{}

func (covDataStmt) Close() error                               { return nil }
func (covDataStmt) NumInput() int                              { return -1 }
func (covDataStmt) Exec([]covDataValue) (driver.Result, error) { return covDataResult{}, nil }
func (covDataStmt) Query([]covDataValue) (driver.Rows, error)  { return &covDataRows{}, nil }

type covDataResult struct{}

func (covDataResult) LastInsertId() (int64, error) { return 1, nil }
func (covDataResult) RowsAffected() (int64, error) { return 1, nil }

type covDataRows struct {
	cols []string
	rows [][]covDataValue
	next int
}

func (r *covDataRows) Columns() []string { return r.cols }
func (r *covDataRows) Close() error      { return nil }
func (r *covDataRows) Next(dest []covDataValue) error {
	if r.next >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.next])
	r.next++
	return nil
}

// covDataDB 建立一個連到替身的 *sql.DB；queries/execs 的登記順序就是比對順序。
func covDataDB(t *testing.T) (*sql.DB, *covDataSource) {
	t.Helper()

	name := fmt.Sprintf("data-cov-%d", atomic.AddUint64(&covDataSeq, 1))
	src := &covDataSource{}
	covDataRegistry.Store(name, src)
	covDataOnce.Do(func() { sql.Register("forum-data-cov-stub", covDataDriver{}) })

	db, err := sql.Open("forum-data-cov-stub", name)
	if err != nil {
		t.Fatalf("sql.Open(cov data): %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		covDataRegistry.Delete(name)
	})
	return db, src
}

// covDataCountZero 登記「所有 information_schema 探測都回 0（不存在）」。
// information_schema 的探測都是一欄 COUNT(*)，因此一個規則可以同時服務
// COLUMNS 與 STATISTICS 兩種查詢。
func covDataCountZero(src *covDataSource) {
	src.addQuery(covDataQueryRule{
		substr: "information_schema",
		cols:   []string{"COUNT(*)"},
		rows:   [][]covDataValue{{int64(0)}},
	})
}

/* ==========================================================================
   indexAlreadyExists / addIndexIfNotExists / addUniqueIndexIfNotExists
   ========================================================================== */

func TestCovIndexAlreadyExists(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"一般錯誤", errors.New("boom"), false},
		{"1061 已存在", &mysql.MySQLError{Number: 1061}, true},
		{"包裝過的 1061", fmt.Errorf("alter failed: %w", &mysql.MySQLError{Number: 1061}), true},
		{"1062 資料重複", &mysql.MySQLError{Number: 1062}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := indexAlreadyExists(c.err); got != c.want {
				t.Fatalf("indexAlreadyExists(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestCovAddIndexIfNotExistsBranches(t *testing.T) {
	t.Run("索引已存在：不執行 ALTER", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{substr: "information_schema", cols: []string{"n"}, rows: [][]covDataValue{{int64(1)}}})
		if err := addIndexIfNotExists(db, "forum_posts", "idx_x", "created_at"); err != nil {
			t.Fatalf("addIndexIfNotExists: %v", err)
		}
		if src.saw("ADD INDEX idx_x") {
			t.Errorf("索引已存在時不應再送出 ALTER")
		}
	})

	t.Run("不存在：執行 ALTER", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		if err := addIndexIfNotExists(db, "forum_posts", "idx_x", "created_at"); err != nil {
			t.Fatalf("addIndexIfNotExists: %v", err)
		}
		if !src.saw("ADD INDEX idx_x (created_at)") {
			t.Errorf("不存在時應送出 ALTER")
		}
	})

	t.Run("探測失敗：回傳錯誤", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{substr: "information_schema", err: errors.New("metadata locked")})
		if err := addIndexIfNotExists(db, "forum_posts", "idx_x", "created_at"); err == nil {
			t.Fatal("探測失敗時應回傳錯誤")
		}
	})

	t.Run("ALTER 失敗（非 1061）：回傳錯誤", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		src.addExec(covDataExecRule{substr: "ADD INDEX idx_x", err: errors.New("syntax error")})
		if err := addIndexIfNotExists(db, "forum_posts", "idx_x", "created_at"); err == nil {
			t.Fatal("ALTER 失敗時應回傳錯誤")
		}
	})

	t.Run("ALTER 撞到併發 1061：視為成功", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		src.addExec(covDataExecRule{substr: "ADD INDEX idx_x", err: &mysql.MySQLError{Number: 1061}})
		if err := addIndexIfNotExists(db, "forum_posts", "idx_x", "created_at"); err != nil {
			t.Fatalf("1061 應被容忍（別人先建好了）, got %v", err)
		}
	})
}

func TestCovAddUniqueIndexIfNotExistsBranches(t *testing.T) {
	t.Run("索引已存在", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{substr: "information_schema", cols: []string{"n"}, rows: [][]covDataValue{{int64(1)}}})
		if err := addUniqueIndexIfNotExists(db, "forum_profiles", "uq_x", "nickname"); err != nil {
			t.Fatalf("addUniqueIndexIfNotExists: %v", err)
		}
		if src.saw("ADD UNIQUE INDEX uq_x") {
			t.Errorf("索引已存在時不應再送出 ALTER")
		}
	})

	t.Run("不存在：執行 ALTER", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		if err := addUniqueIndexIfNotExists(db, "forum_profiles", "uq_x", "nickname"); err != nil {
			t.Fatalf("addUniqueIndexIfNotExists: %v", err)
		}
		if !src.saw("ADD UNIQUE INDEX uq_x (nickname)") {
			t.Errorf("不存在時應送出 ADD UNIQUE INDEX")
		}
	})

	t.Run("探測失敗", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{substr: "information_schema", err: errors.New("boom")})
		if err := addUniqueIndexIfNotExists(db, "forum_profiles", "uq_x", "nickname"); err == nil {
			t.Fatal("探測失敗時應回傳錯誤")
		}
	})

	t.Run("ALTER 失敗（1062 重複資料）", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		src.addExec(covDataExecRule{substr: "ADD UNIQUE INDEX uq_x", err: &mysql.MySQLError{Number: 1062}})
		if err := addUniqueIndexIfNotExists(db, "forum_profiles", "uq_x", "nickname"); err == nil {
			t.Fatal("1062 必須讓遷移失敗（不能靜默產生重複暱稱）")
		}
	})

	t.Run("ALTER 撞到併發 1061", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		src.addExec(covDataExecRule{substr: "ADD UNIQUE INDEX uq_x", err: &mysql.MySQLError{Number: 1061}})
		if err := addUniqueIndexIfNotExists(db, "forum_profiles", "uq_x", "nickname"); err != nil {
			t.Fatalf("1061 應被容忍, got %v", err)
		}
	})
}

/* ==========================================================================
   MigrateMySQL
   ========================================================================== */

func TestCovMigrateMySQLNilDatabase(t *testing.T) {
	if err := MigrateMySQL(nil); err == nil {
		t.Fatal("nil 資料庫應回傳錯誤而不是 panic")
	}
}

func TestCovMigrateMySQLRunsAllSteps(t *testing.T) {
	db, src := covDataDB(t)
	covDataCountZero(src)

	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("MigrateMySQL: %v", err)
	}

	// 抽查路線的頭、中、尾：全部成功代表線性流程沒有在中途返回。
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS forum_posts",
		"CREATE TABLE IF NOT EXISTS forum_notifications",
		"CREATE TABLE IF NOT EXISTS forum_daily_stats",
		"ALTER TABLE forum_profiles ADD COLUMN IF NOT EXISTS avatar_url",
		"ADD INDEX idx_forum_posts_feed",
		"ADD UNIQUE INDEX uq_forum_profiles_nickname",
		"ADD INDEX idx_forum_posts_deleted_at_id",
		"ADD INDEX idx_forum_reports_target",
		"CREATE TABLE IF NOT EXISTS forum_media_deletions",
	} {
		if !src.saw(want) {
			t.Errorf("遷移路線沒有執行到 %q", want)
		}
	}

	if got := src.countSeen("ADD INDEX idx_forum_profiles_public_key"); got != 1 {
		t.Errorf("addIndexIfNotExists 的 ALTER 次數 = %d, want 1", got)
	}
}

func TestCovMigrateMySQLLegacyBranches(t *testing.T) {
	db, src := covDataDB(t)
	// 三種舊 schema 探測都回「存在」，讓 DROP／CHANGE 的分支被執行。
	src.addQuery(covDataQueryRule{
		substr: "COLUMN_NAME = 'author_email'",
		cols:   []string{"n"}, rows: [][]covDataValue{{int64(1)}},
	})
	src.addQuery(covDataQueryRule{
		substr: "COLUMN_NAME = 'user_email'",
		cols:   []string{"n"}, rows: [][]covDataValue{{int64(1)}},
	})
	src.addQuery(covDataQueryRule{
		substr: "COLUMN_NAME = 'title'",
		cols:   []string{"n"}, rows: [][]covDataValue{{int64(1)}},
	})
	covDataCountZero(src)

	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("MigrateMySQL（舊 schema）: %v", err)
	}
	for _, want := range []string{
		"DROP COLUMN author_email",
		"CHANGE COLUMN user_email author_email",
		"DROP COLUMN title",
	} {
		if !src.saw(want) {
			t.Errorf("舊 schema 遷移沒有執行 %q", want)
		}
	}
}

func TestCovMigrateMySQLProbeErrors(t *testing.T) {
	for _, probe := range []string{
		"COLUMN_NAME = 'author_email'",
		"COLUMN_NAME = 'user_email'",
		"COLUMN_NAME = 'title'",
	} {
		t.Run(probe, func(t *testing.T) {
			db, src := covDataDB(t)
			src.addQuery(covDataQueryRule{substr: probe, err: errors.New("probe failed")})
			covDataCountZero(src)
			if err := MigrateMySQL(db); err == nil {
				t.Fatalf("探測 %q 失敗時 MigrateMySQL 必須回傳錯誤", probe)
			}
		})
	}
}

// TestCovMigrateMySQLExecErrors 讓每一步 Exec 各失敗一次，覆蓋「中途失敗即
// 立刻回傳」的每一條 return 分支。每個子測試都有自己的替身，互不影響。
func TestCovMigrateMySQLExecErrors(t *testing.T) {
	cases := []struct {
		name  string
		exec  string
		probe string // 非空時讓該舊欄位探測回報「存在」
	}{
		{"建立貼文表", "CREATE TABLE IF NOT EXISTS forum_posts", ""},
		{"補 image_url", "ADD COLUMN IF NOT EXISTS image_url", ""},
		{"建立按讚表", "CREATE TABLE IF NOT EXISTS forum_post_likes", ""},
		{"建立留言表", "CREATE TABLE IF NOT EXISTS forum_post_comments", ""},
		{"建立檢舉表", "CREATE TABLE IF NOT EXISTS forum_reports", ""},
		{"補 author_email（舊表）", "ADD COLUMN author_email VARCHAR(255) NOT NULL DEFAULT ''", ""},
		{"丟 author_email（舊欄位遷移）", "DROP COLUMN author_email", "COLUMN_NAME = 'user_email'"},
		{"改名 user_email", "CHANGE COLUMN user_email author_email", "COLUMN_NAME = 'user_email'"},
		{"丟 title（舊欄位遷移）", "DROP COLUMN title", "COLUMN_NAME = 'title'"},
		{"建立 profiles", "CREATE TABLE IF NOT EXISTS forum_profiles", ""},
		{"建立 users", "CREATE TABLE IF NOT EXISTS forum_users", ""},
		{"建立標籤字典", "CREATE TABLE IF NOT EXISTS forum_user_tags", ""},
		{"建立標籤關聯", "CREATE TABLE IF NOT EXISTS forum_user_tag_assignments", ""},
		{"回填使用者", "INSERT IGNORE INTO forum_users", ""},
		{"補 bio", "ADD COLUMN IF NOT EXISTS bio", ""},
		{"補 public_key", "ADD COLUMN IF NOT EXISTS public_key", ""},
		{"回填 public_key", "UPDATE forum_profiles SET public_key", ""},
		{"暱稱唯一索引", "ADD UNIQUE INDEX uq_forum_profiles_nickname", ""},
		{"搬移 image_url", "UPDATE forum_posts", ""},
		{"建立 follows", "CREATE TABLE IF NOT EXISTS forum_follows", ""},
		{"public_key 索引", "ADD INDEX idx_forum_profiles_public_key", ""},
		{"posts.author_email 索引", "ADD INDEX idx_forum_posts_author_email", ""},
		{"建立 metrics", "CREATE TABLE IF NOT EXISTS forum_request_metrics", ""},
		{"建立稽核表", "CREATE TABLE IF NOT EXISTS forum_admin_actions", ""},
		{"users.created_at 索引", "ADD INDEX idx_forum_users_created_at", ""},
		{"建立公告表", "CREATE TABLE IF NOT EXISTS forum_announcements", ""},
		{"補 pinned", "ADD COLUMN IF NOT EXISTS pinned", ""},
		{"feed 索引", "ADD INDEX idx_forum_posts_feed", ""},
		{"comments.author_email 索引", "ADD INDEX idx_forum_post_comments_author_email", ""},
		{"likes.created_at 索引", "ADD INDEX idx_forum_post_likes_created_at", ""},
		{"補貼文 updated_at", "ADD COLUMN IF NOT EXISTS updated_at DATETIME NULL AFTER pinned", ""},
		{"補留言 updated_at", "ADD COLUMN IF NOT EXISTS updated_at DATETIME NULL AFTER created_at", ""},
		{"補 avatar_url", "ADD COLUMN IF NOT EXISTS avatar_url", ""},
		{"補 deleted_at", "ADD COLUMN IF NOT EXISTS deleted_at", ""},
		{"補 parent_id", "ADD COLUMN IF NOT EXISTS parent_id", ""},
		{"補 depth", "ADD COLUMN IF NOT EXISTS depth", ""},
		{"建立通知表", "CREATE TABLE IF NOT EXISTS forum_notifications", ""},
		{"建立媒體表", "CREATE TABLE IF NOT EXISTS forum_media", ""},
		{"建立收藏表", "CREATE TABLE IF NOT EXISTS forum_post_bookmarks", ""},
		{"建立日彙總表", "CREATE TABLE IF NOT EXISTS forum_daily_stats", ""},
		{"deleted_at_id 索引", "ADD INDEX idx_forum_posts_deleted_at_id", ""},
		{"reports.target 索引", "ADD INDEX idx_forum_reports_target", ""},
		{"posts.image_url 索引", "ADD INDEX idx_forum_posts_image_url", ""},
		{"profiles.avatar_url 索引", "ADD INDEX idx_forum_profiles_avatar_url", ""},
		{"建立圖片刪除佇列", "CREATE TABLE IF NOT EXISTS forum_media_deletions", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, src := covDataDB(t)
			if c.probe != "" {
				src.addQuery(covDataQueryRule{substr: c.probe, cols: []string{"n"}, rows: [][]covDataValue{{int64(1)}}})
			}
			covDataCountZero(src)
			src.addExec(covDataExecRule{substr: c.exec, err: errors.New("ddl failed")})
			if err := MigrateMySQL(db); err == nil {
				t.Fatalf("Exec %q 失敗時 MigrateMySQL 必須回傳錯誤", c.exec)
			}
		})
	}
}

/* ==========================================================================
   OpenMySQL
   ========================================================================== */

func TestCovOpenMySQLRejectsInvalidDSN(t *testing.T) {
	db, err := OpenMySQL("not-a-valid-dsn", 5, 2, time.Minute)
	if err == nil || db != nil {
		t.Fatalf("無效 DSN 應回 (nil, error)，得到 (%v, %v)", db, err)
	}
}

func TestCovOpenMySQLReturnsPingError(t *testing.T) {
	// 不存在的 unix socket：Ping 立刻失敗，不會碰到任何外部主機。
	// timeout 只是保險，避免平台上的 Dial 卡住。
	db, err := OpenMySQL("root@unix(/nonexistent-forum-cov.sock)/forum?timeout=50ms", 5, 2, time.Minute)
	if err == nil || db != nil {
		t.Fatalf("Ping 失敗時應回 (nil, error) 且關閉連線池，得到 (%v, %v)", db, err)
	}
}

/* ==========================================================================
   rollup
   ========================================================================== */

func TestCovRollupBackfillAndRefreshRunSQL(t *testing.T) {
	db, src := covDataDB(t)

	if err := BackfillDailyStats(context.Background(), db); err != nil {
		t.Fatalf("BackfillDailyStats: %v", err)
	}
	if err := RefreshDailyStats(context.Background(), db); err != nil {
		t.Fatalf("RefreshDailyStats: %v", err)
	}

	if got := src.countSeen("INSERT IGNORE INTO forum_daily_stats"); got != 1 {
		t.Errorf("回填次數 = %d, want 1", got)
	}
	// 刷新窗口是三天：每天一句 INSERT … ON DUPLICATE KEY UPDATE。
	if got := src.countSeen("ON DUPLICATE KEY UPDATE"); got != refreshWindowDays {
		t.Errorf("刷新次數 = %d, want %d", got, refreshWindowDays)
	}
}

func TestCovRollupErrors(t *testing.T) {
	t.Run("回填失敗", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addExec(covDataExecRule{substr: "INSERT IGNORE INTO forum_daily_stats", err: errors.New("boom")})
		if err := BackfillDailyStats(context.Background(), db); err == nil {
			t.Fatal("回填 Exec 失敗時應回傳錯誤")
		}
	})

	t.Run("刷新失敗", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addExec(covDataExecRule{substr: "INSERT INTO forum_daily_stats", err: errors.New("boom")})
		if err := RefreshDailyStats(context.Background(), db); err == nil {
			t.Fatal("刷新 Exec 失敗時應回傳錯誤")
		}
	})
}
