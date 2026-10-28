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
	// consume 為 true 時命中一次後清空列：讓同一條 SELECT 在下一輪回空，
	// 用來測分批迴圈而不會變成無限迴圈。
	consume bool
	// iterErr 是列讀完後 Next 回的回覆錯誤（不是 io.EOF），讓 rows.Err() 有錯誤可觀察。
	iterErr error
}

type covDataExecRule struct {
	substr string
	err    error
}

type covDataSource struct {
	mu       sync.Mutex
	queries  []covDataQueryRule
	execs    []covDataExecRule
	prepares []covDataExecRule
	seen     []string
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

// addPrepare 讓「準備這條語句」本身失敗（例如欄位不存在）。只有走預備語句的
// 路徑（db.PrepareContext）會命中；conn 的 ExecContext 快路不受影響。
func (s *covDataSource) addPrepare(rule covDataExecRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prepares = append(s.prepares, rule)
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

func (c *covDataConn) Prepare(query string) (driver.Stmt, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	for _, rule := range c.src.prepares {
		if strings.Contains(query, rule.substr) {
			return nil, rule.err
		}
	}
	return &covDataStmt{src: c.src, query: query}, nil
}
func (c *covDataConn) Close() error              { return nil }
func (c *covDataConn) Begin() (driver.Tx, error) { return covDataTx{}, nil }
func (c *covDataConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return covDataTx{}, nil
}
func (c *covDataConn) Ping(context.Context) error { return nil }

// pickQueryLocked 依登記順序找出第一條命中的 query 規則；呼叫端必須已持鎖。
// consume 的規則在回列後清空自己的列，因此同一條 SQL 下一輪拿到空結果。
func (s *covDataSource) pickQueryLocked(query string) (driver.Rows, error) {
	for i := range s.queries {
		rule := &s.queries[i]
		if !strings.Contains(query, rule.substr) {
			continue
		}
		if rule.err != nil {
			return nil, rule.err
		}
		rows := &covDataRows{cols: rule.cols, rows: rule.rows, iterErr: rule.iterErr}
		if rule.consume {
			rule.rows = nil
		}
		return rows, nil
	}
	return &covDataRows{}, nil
}

func (c *covDataConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)
	return c.src.pickQueryLocked(query)
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

type covDataStmt struct {
	src   *covDataSource
	query string
}

func (s *covDataStmt) Close() error  { return nil }
func (s *covDataStmt) NumInput() int { return -1 }

// Exec 記錄預備語句的 SQL，並套用 exec 規則：與 conn 的 ExecContext 快路一致。
func (s *covDataStmt) Exec([]covDataValue) (driver.Result, error) {
	s.src.mu.Lock()
	defer s.src.mu.Unlock()
	s.src.seen = append(s.src.seen, s.query)
	for _, rule := range s.src.execs {
		if strings.Contains(s.query, rule.substr) {
			if rule.err != nil {
				return nil, rule.err
			}
			break
		}
	}
	return covDataResult{}, nil
}

func (s *covDataStmt) Query([]covDataValue) (driver.Rows, error) {
	s.src.mu.Lock()
	defer s.src.mu.Unlock()
	s.src.seen = append(s.src.seen, s.query)
	return s.src.pickQueryLocked(s.query)
}

type covDataResult struct{}

func (covDataResult) LastInsertId() (int64, error) { return 1, nil }
func (covDataResult) RowsAffected() (int64, error) { return 1, nil }

type covDataRows struct {
	cols    []string
	rows    [][]covDataValue
	next    int
	iterErr error
}

func (r *covDataRows) Columns() []string { return r.cols }
func (r *covDataRows) Close() error      { return nil }
func (r *covDataRows) Next(dest []covDataValue) error {
	if r.next >= len(r.rows) {
		if r.iterErr != nil {
			return r.iterErr
		}
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
	// 雪花 id 遷移的 named lock。替身沒有真正的鎖，回 1 代表「取得」；
	// 沒有這個規則時 QueryRow 會是 ErrNoRows，而遷移會把它當成失敗。
	src.addQuery(covDataQueryRule{
		substr: "GET_LOCK",
		cols:   []string{"GET_LOCK"},
		rows:   [][]covDataValue{{int64(1)}},
	})
	// 第 2b 步（normalizeForumImageValues）的「還有多少舊寫法的列」計數。回 0 代表
	// 「已經都是新契約」，那個分支會直接返回，不再逐列 SELECT、也不 UPDATE。
	// 沒有這條規則時，整條遷移路線會死在「no rows in result set」上。
	src.addQuery(covDataQueryRule{
		substr: "SELECT COUNT(*) FROM forum_posts WHERE image_url LIKE",
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
   columnAlreadyExists / addColumnIfNotExists
   ========================================================================== */

func TestCovColumnAlreadyExists(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"一般錯誤", errors.New("boom"), false},
		{"1060 欄位已存在", &mysql.MySQLError{Number: 1060}, true},
		{"包裝過的 1060", fmt.Errorf("alter failed: %w", &mysql.MySQLError{Number: 1060}), true},
		{"1061 是索引的錯不是欄位的", &mysql.MySQLError{Number: 1061}, false},
		{"1064 語法錯誤：不可被當成「已存在」吞掉", &mysql.MySQLError{Number: 1064}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := columnAlreadyExists(c.err); got != c.want {
				t.Fatalf("columnAlreadyExists(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

// addColumnIfNotExists 的分支測試與 addIndexIfNotExists 一一對應：兩者的冪等性
// 是同一套承諾（探測 + 建立 + 容忍另一個實例先完成），因此漏掉任何一邊的分支
// 都代表同一種防線在兩處有不同的覆蓋。
//
// 1064 那條分支是 2026-10-23 那次 CI 失敗的原始現場：遷移若寫回
// `ADD COLUMN IF NOT EXISTS`（MariaDB 專屬），MySQL 收到的就是 1064，
// 而它**必須**讓遷移失敗 —— 被吞掉的話，缺欄位會在第一個請求時才以 500 浮現。
func TestCovAddColumnIfNotExistsBranches(t *testing.T) {
	t.Run("欄位已存在：不執行 ALTER", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{substr: "information_schema", cols: []string{"n"}, rows: [][]covDataValue{{int64(1)}}})
		if err := addColumnIfNotExists(db, "forum_posts", "image_url", "TEXT NOT NULL"); err != nil {
			t.Fatalf("addColumnIfNotExists: %v", err)
		}
		if src.saw("ADD COLUMN image_url") {
			t.Errorf("欄位已存在時不應再送出 ALTER")
		}
	})

	t.Run("不存在：執行 ALTER", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		if err := addColumnIfNotExists(db, "forum_posts", "image_url", "TEXT NOT NULL"); err != nil {
			t.Fatalf("addColumnIfNotExists: %v", err)
		}
		if !src.saw("ALTER TABLE forum_posts ADD COLUMN image_url TEXT NOT NULL") {
			t.Errorf("不存在時應送出 ALTER")
		}
	})

	t.Run("探測失敗：回傳錯誤", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{substr: "information_schema", err: errors.New("metadata locked")})
		if err := addColumnIfNotExists(db, "forum_posts", "image_url", "TEXT NOT NULL"); err == nil {
			t.Fatal("探測失敗時應回傳錯誤")
		}
	})

	t.Run("ALTER 失敗（非 1060）：回傳錯誤", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		src.addExec(covDataExecRule{substr: "ADD COLUMN image_url", err: errors.New("syntax error")})
		if err := addColumnIfNotExists(db, "forum_posts", "image_url", "TEXT NOT NULL"); err == nil {
			t.Fatal("ALTER 失敗時應回傳錯誤")
		}
	})

	t.Run("ALTER 撞到併發 1060：視為成功", func(t *testing.T) {
		db, src := covDataDB(t)
		covDataCountZero(src)
		src.addExec(covDataExecRule{substr: "ADD COLUMN image_url", err: &mysql.MySQLError{Number: 1060}})
		if err := addColumnIfNotExists(db, "forum_posts", "image_url", "TEXT NOT NULL"); err != nil {
			t.Fatalf("1060 應被容忍（另一個實例先補好了）, got %v", err)
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
		"ALTER TABLE forum_profiles ADD COLUMN avatar_url",
		"ADD INDEX idx_forum_posts_feed",
		"ADD UNIQUE INDEX uq_forum_profiles_nickname",
		"ADD INDEX idx_forum_posts_deleted_at_id",
		"ADD INDEX idx_forum_reports_target",
		"CREATE TABLE IF NOT EXISTS forum_media_deletions",
		// 第 39 步：公開識別碼的 HMAC 欄位與它的索引（值由應用層回填，不在 SQL 裡算）。
		"ADD COLUMN public_key_hmac",
		"ADD INDEX idx_forum_profiles_public_key_hmac",
	} {
		if !src.saw(want) {
			t.Errorf("遷移路線沒有執行到 %q", want)
		}
	}

	// 兩把識別碼各有一條索引，且都只被建立一次（探測到「已存在」時不重複 ALTER）。
	// 計數用「含括號與欄名」的完整片段而不是只比索引名前綴：
	// idx_forum_profiles_public_key_hmac 包含 idx_forum_profiles_public_key，
	// 只比前綴會讓一條變成兩條。
	if got := src.countSeen("ADD INDEX idx_forum_profiles_public_key (public_key)"); got != 1 {
		t.Errorf("addIndexIfNotExists 的 ALTER 次數 = %d, want 1", got)
	}
	if got := src.countSeen("ADD INDEX idx_forum_profiles_public_key_hmac (public_key_hmac)"); got != 1 {
		t.Errorf("public_key_hmac 的 ALTER 次數 = %d, want 1", got)
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

// TestCovRenumberTextReferenceCollation 釘住「文字參考的比對長什麼樣」。
//
// 它釘的是對外看到的 SQL：比對那一側帶著被探測到的 collation，而寫入那一側不帶。
// 真正的 Error 1267 只有真實的 MySQL 8 會回（見 migration_test.go 的 collation 測試），
// 假驅動看不見那個錯誤，但它能證明兩件會靜默失效的事：
//   - 探測到的 collation 真的被用進比對（少了它，CI 的 MySQL 8 會 1267，而本機沒事）；
//   - collation 探測失敗時整個重新編號中止，而不是拼出一句「沒有 collation」的 UPDATE。
func TestCovRenumberTextReferenceCollation(t *testing.T) {
	// 探測的 SQL 片段：columnCollation 把表與欄位寫成字面值，因此片段本身就是規格。
	const probe = "TABLE_NAME = 'forum_admin_actions' AND COLUMN_NAME = 'target_id'"

	for _, tt := range []struct {
		name       string
		probe      covDataQueryRule
		contain    []string
		notContain []string
		wantErr    bool
	}{
		{
			name: "MySQL 8 的預設 collation：比對被換成欄位自己的",
			probe: covDataQueryRule{
				substr: probe, cols: []string{"COLLATION_NAME"},
				rows: [][]covDataValue{{"utf8mb4_0900_ai_ci"}},
			},
			contain: []string{
				"CONVERT(CAST(m.old_id AS CHAR) USING utf8mb4) COLLATE utf8mb4_0900_ai_ci",
				"SET t.target_id = CAST(m.new_id AS CHAR)",
				"WHERE target_type = 'post'",
			},
			// 寫入那一側不需要 CONVERT／COLLATE：寫入是轉換，不是比對。
			notContain: []string{"CONVERT(CAST(m.new_id"},
		},
		{
			name: "探測回 NULL（沒有 collation 的欄位）：不強制",
			probe: covDataQueryRule{
				substr: probe, cols: []string{"COLLATION_NAME"},
				rows: [][]covDataValue{{nil}},
			},
			contain:    []string{"t.target_id = CAST(m.old_id AS CHAR)"},
			notContain: []string{"COLLATE"},
		},
		{
			name:    "探測失敗：中止",
			probe:   covDataQueryRule{substr: probe, err: errors.New("information_schema down")},
			wantErr: true,
		},
		{
			name: "探測不到那一欄：中止",
			probe: covDataQueryRule{
				substr: probe, cols: []string{"COLLATION_NAME"},
			},
			wantErr: true,
		},
		{
			name: "不尋常的 collation 名稱：中止，而不是拼進 SQL",
			probe: covDataQueryRule{
				substr: probe, cols: []string{"COLLATION_NAME"},
				rows: [][]covDataValue{{"utf8mb4'; DROP TABLE forum_posts; --"}},
			},
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, src := covDataDB(t)
			// 一批一列、讀完就清空：分批迴圈因此會結束而不會變成無限迴圈。
			src.addQuery(covDataQueryRule{
				substr: "SELECT id FROM forum_posts WHERE id > ?",
				cols:   []string{"id"}, rows: [][]covDataValue{{int64(7)}}, consume: true,
			})
			src.addQuery(tt.probe)

			conn, err := db.Conn(context.Background())
			if err != nil {
				t.Fatalf("取得連線失敗: %v", err)
			}
			defer conn.Close()

			// snowflakeIDTables[1] 是 forum_posts：它引用 forum_admin_actions.target_id，
			// 而 CI 的失敗正是「forum_posts 重新編號時更新那一欄」那一句。
			err = renumberTable(context.Background(), conn, snowflakeIDTables[1])
			if tt.wantErr {
				if err == nil {
					t.Fatal("應該回錯誤，而不是把沒有 collation 的比對拼出去")
				}
				return
			}
			if err != nil {
				t.Fatalf("renumberTable: %v", err)
			}
			for _, want := range tt.contain {
				if !src.saw(want) {
					t.Errorf("重新編號的 UPDATE 沒有包含 %q", want)
				}
			}
			for _, absent := range tt.notContain {
				if src.saw(absent) {
					t.Errorf("重新編號的 UPDATE 不該包含 %q", absent)
				}
			}
		})
	}
}

func TestCovMigrateMySQLProbeErrors(t *testing.T) {
	// 'image_url' 是 addColumnIfNotExists 的探測（其餘三個是第 6/7/8 步的探測）：
	// 兩種探測都必須在失敗時中止遷移，否則「某個欄位補不補得上」會靜默取決於
	// information_schema 恰好回什麼。
	for _, probe := range []string{
		"COLUMN_NAME = 'author_email'",
		"COLUMN_NAME = 'user_email'",
		"COLUMN_NAME = 'title'",
		"COLUMN_NAME = 'image_url'",
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
		{"補 image_url", "ADD COLUMN image_url", ""},
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
		{"補 bio", "ADD COLUMN bio", ""},
		{"補 public_key", "ADD COLUMN public_key", ""},
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
		{"補 pinned", "ADD COLUMN pinned", ""},
		{"feed 索引", "ADD INDEX idx_forum_posts_feed", ""},
		{"comments.author_email 索引", "ADD INDEX idx_forum_post_comments_author_email", ""},
		{"likes.created_at 索引", "ADD INDEX idx_forum_post_likes_created_at", ""},
		{"補貼文 updated_at", "ADD COLUMN updated_at DATETIME NULL AFTER pinned", ""},
		{"補留言 updated_at", "ADD COLUMN updated_at DATETIME NULL AFTER created_at", ""},
		{"補 avatar_url", "ADD COLUMN avatar_url", ""},
		{"補 deleted_at", "ADD COLUMN deleted_at", ""},
		{"補 parent_id", "ADD COLUMN parent_id", ""},
		{"補 depth", "ADD COLUMN depth", ""},
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
