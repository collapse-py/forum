/*
purge 套件的測試（backend/forum/purge/purge_test.go）。

【測試的兩層】

 1. 不需要資料庫的純邏輯：nil 安全、檔名收斂、LIKE 跳脫。這些是「清理排程
    的失敗模式」裡最容易寫錯的部分——imageFileName 收斂錯方向會讓檔案刪不掉
    或誤刪，escapeLike 漏一個萬用字元會讓引用計數多算。

 2. 需要真實 MySQL 的端到端清理（TestPruneMySQL）：與 data/migration_test.go
    同一個約定，FORUM_TEST_MYSQL_DSN 未設定時 skip。它驗的是整條路徑——
    過期貼文連同留言／按讚／收藏／檢舉一起消失、保留期內的貼文不動、
    仍被引用的附圖不回收、只被清理貼文引用的附圖被回收。

【為什麼不用 sqlmock】

與 data/migration_test.go 的理由相同：sqlmock 只能回傳「预先写好的答案」，
而清理排程的正確性恰恰是「DELETE 之後 SELECT 看不到」——那需要真的資料庫
真的刪。
*/
package purge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	// 空白匯入：TestPruneMySQL 以 sql.Open("mysql", …) 開連線，沒有這個
	// 匯入時 sql.Open 會在 FORUM_TEST_MYSQL_DSN 設定的環境下直接回
	//「unknown driver」——而未設定 DSN 的環境永遠不會碰到它。
	_ "github.com/go-sql-driver/mysql"
)

func TestPruneNilSafety(t *testing.T) {
	p := NewPruner(nil, nil, 90*24*time.Hour, time.Hour)
	if removed, err := p.Prune(context.Background()); removed != 0 || err != nil {
		t.Fatalf("nil 資料庫的 Prune 應回 (0, nil)，得到 (%d, %v)", removed, err)
	}
	// Run 對 nil 資料庫必須立刻返回，而不是卡在 ticker 上。
	done := make(chan struct{})
	go func() { p.Run(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run 對 nil 資料庫沒有返回")
	}
}

func TestNewPrunerDefaults(t *testing.T) {
	p := NewPruner(nil, nil, 0, 0)
	if p.retention != 90*24*time.Hour || p.interval != time.Hour {
		t.Fatalf("retention/interval 的兜底值不對: %v / %v", p.retention, p.interval)
	}
}

func TestImageFileName(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"a_b.png", "a_b.png"},
		{"/files/a_b.png", "a_b.png"},
		{"http://localhost:7070/files/a_b.png", "a_b.png"},
		{"", ""},
		{"   ", ""},
		{"/files/", ""},
	}
	for _, c := range cases {
		if got := imageFileName(c.raw); got != c.want {
			t.Errorf("imageFileName(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`a_b%c\d`); got != `a\_b\%c\\d` {
		t.Fatalf("escapeLike 的跳脫順序錯誤: %q", got)
	}
}

// recordingDeleter 記錄被「刪除」的檔名，並可模擬檔案伺服器的故障。
type recordingDeleter struct {
	deleted []string
	fail    bool
}

func (d *recordingDeleter) DeleteFile(_ context.Context, name string) error {
	if d.fail {
		return errors.New("file server down")
	}
	d.deleted = append(d.deleted, name)
	return nil
}

var purgeDBCounter atomic.Int64

// purgeTestDB 建立一個跑完整 MigrateMySQL 的拋棄式資料庫；未設定
// FORUM_TEST_MYSQL_DSN 時 skip（與 data/migration_test.go 同一個約定）。
func purgeTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("FORUM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FORUM_TEST_MYSQL_DSN 未設定：跳過需要真實 MySQL 的清理測試。")
	}
	cleaned := dsn
	if i := strings.LastIndex(cleaned, "/"); i >= 0 {
		if j := strings.Index(cleaned[i:], "?"); j >= 0 {
			cleaned = cleaned[:i] + "/" + cleaned[i+j:]
		} else {
			cleaned = cleaned[:i] + "/"
		}
	}
	server, err := sql.Open("mysql", cleaned)
	if err != nil {
		t.Fatalf("無法開啟連線: %v", err)
	}
	defer server.Close()
	if err := server.Ping(); err != nil {
		t.Fatalf("MySQL 連不上: %v", err)
	}

	name := fmt.Sprintf("forum_purge_test_%d_%d", time.Now().UnixNano(), purgeDBCounter.Add(1))
	if _, err := server.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("建立測試資料庫失敗: %v", err)
	}
	t.Cleanup(func() { _, _ = server.Exec("DROP DATABASE " + name) })

	// 從伺服器層級 DSN 重建「連到拋棄式資料庫」的 DSN，而不是直接字串串接：
	// cleaned 在 DSN 帶參數時以 "?params" 結尾（CI 的 DSN 就是），直接接
	// name 會把庫名塞進參數區（multiStatements=trueforum_purge_test_…），
	// 症狀是「invalid bool value」而不是一個指向 DSN 組裝的錯誤訊息。
	base, params := cleaned, ""
	if q := strings.Index(cleaned, "?"); q >= 0 {
		base, params = cleaned[:q], cleaned[q:]
	}
	db, err := sql.Open("mysql", strings.TrimSuffix(base, "/")+"/"+name+params)
	if err != nil {
		t.Fatalf("連到測試資料庫失敗: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("測試資料庫連不上: %v", err)
	}
	if err := migrateForPurge(db); err != nil {
		t.Fatalf("建立 schema 失敗: %v", err)
	}
	return db
}

// migrateForPurge 只建立清理排程會觸碰的表，避免整個 MigrateMySQL 的
// 34 步在每次測試都跑一遍。列的形状與 data/mysql.go 逐字相同（測試驗的是
// 「清理排程的 SQL 對真實 schema 成立」，用同一份 DDL 才有意義）。
func migrateForPurge(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS forum_posts (
			id BIGINT NOT NULL AUTO_INCREMENT,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			image_url VARCHAR(255) NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NULL,
			pinned TINYINT NOT NULL DEFAULT 0,
			deleted_at DATETIME NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS forum_post_comments (
			id BIGINT NOT NULL AUTO_INCREMENT,
			post_id BIGINT NOT NULL,
			parent_id BIGINT NULL,
			depth TINYINT NOT NULL DEFAULT 1,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS forum_post_likes (
			post_id BIGINT NOT NULL,
			author_email VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (post_id, author_email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS forum_post_bookmarks (
			post_id BIGINT NOT NULL,
			author_email VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (post_id, author_email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS forum_reports (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			target_type VARCHAR(20) NOT NULL,
			target_id BIGINT NOT NULL,
			reporter_email VARCHAR(255) NOT NULL,
			reason VARCHAR(500) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			created_at DATETIME NOT NULL,
			reviewed_at DATETIME NULL,
			reviewed_by VARCHAR(255) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS forum_profiles (
			author_email VARCHAR(255) NOT NULL,
			public_key VARCHAR(64) NOT NULL,
			nickname VARCHAR(100) NULL,
			avatar_url VARCHAR(255) NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (author_email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS forum_media (
			filename VARCHAR(128) NOT NULL,
			owner_email VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (filename)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func insertPost(db *sql.DB, author, content, image string, deletedAt *time.Time) int64 {
	res, err := db.Exec(
		`INSERT INTO forum_posts (author_email, content, image_url, created_at, deleted_at) VALUES (?, ?, ?, ?, ?)`,
		author, content, image, time.Now(), deletedAt)
	if err != nil {
		return 0
	}
	id, _ := res.LastInsertId()
	return id
}

func countRows(db *sql.DB, table string) int {
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
	return n
}

func TestPruneMySQL(t *testing.T) {
	db := purgeTestDB(t)

	old := time.Now().Add(-100 * 24 * time.Hour)
	recent := time.Now().Add(-10 * 24 * time.Hour)

	// 過期貼文：有附圖、留言、按讚、收藏、檢舉。
	expired := insertPost(db, "a@example.com", "過期的文章", "old_a.png", &old)
	if expired == 0 {
		t.Fatal("無法插入過期貼文")
	}
	if _, err := db.Exec(`INSERT INTO forum_post_comments (post_id, author_email, content, created_at) VALUES (?, 'b@example.com', '留言', ?)`, expired, old); err != nil {
		t.Fatalf("插入留言失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_post_likes (post_id, author_email, created_at) VALUES (?, 'c@example.com', ?)`, expired, old); err != nil {
		t.Fatalf("插入按讚失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_post_bookmarks (post_id, author_email, created_at) VALUES (?, 'd@example.com', ?)`, expired, old); err != nil {
		t.Fatalf("插入收藏失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_reports (target_type, target_id, reporter_email, reason, created_at) VALUES ('post', ?, 'e@example.com', '檢舉', ?)`, expired, old); err != nil {
		t.Fatalf("插入檢舉失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_media (filename, owner_email, created_at) VALUES ('old_a.png', 'a@example.com', ?)`, old); err != nil {
		t.Fatalf("插入媒體紀錄失敗: %v", err)
	}

	// 保留期內的軟刪除貼文：不應被清理。
	kept := insertPost(db, "f@example.com", "還在保留期內", "", &recent)
	if kept == 0 {
		t.Fatal("無法插入保留期貼文")
	}

	// 過期貼文 B：附圖仍被另一篇「活的」貼文引用，檔案不應回收。
	if insertPost(db, "g@example.com", "活的文章引用舊圖", "shared.png", nil) == 0 {
		t.Fatal("無法插入引用共享圖片的活貼文")
	}
	if _, err := db.Exec(`INSERT INTO forum_media (filename, owner_email, created_at) VALUES ('shared.png', 'g@example.com', ?)`, old); err != nil {
		t.Fatalf("插入媒體紀錄失敗: %v", err)
	}
	ghost := insertPost(db, "g@example.com", "已刪除但引用同一張圖", "shared.png", &old)
	if ghost == 0 {
		t.Fatal("無法插入引用共享圖片的過期貼文")
	}

	deleter := &recordingDeleter{}
	p := NewPruner(db, deleter, 90*24*time.Hour, time.Hour)

	removed, err := p.Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune 失敗: %v", err)
	}
	if removed != 2 {
		t.Fatalf("應清理 2 篇過期貼文，實際 %d", removed)
	}

	if countRows(db, "forum_posts") != 2 {
		t.Fatalf("forum_posts 應只剩 2 列（保留期貼文 + 活貼文），實際 %d", countRows(db, "forum_posts"))
	}
	if countRows(db, "forum_post_comments") != 0 {
		t.Fatalf("過期貼文的留言應一併清除，實際 %d", countRows(db, "forum_post_comments"))
	}
	if countRows(db, "forum_post_likes") != 0 {
		t.Fatalf("過期貼文的按讚應一併清除，實際 %d", countRows(db, "forum_post_likes"))
	}
	if countRows(db, "forum_post_bookmarks") != 0 {
		t.Fatalf("過期貼文的收藏應一併清除，實際 %d", countRows(db, "forum_post_bookmarks"))
	}
	if countRows(db, "forum_reports") != 0 {
		t.Fatalf("指向過期貼文的檢舉應一併清除，實際 %d", countRows(db, "forum_reports"))
	}

	if len(deleter.deleted) != 1 || deleter.deleted[0] != "old_a.png" {
		t.Fatalf("只應回收不再被引用的附圖，實際 %v", deleter.deleted)
	}
	if countRows(db, "forum_media") != 1 {
		var remaining string
		_ = db.QueryRow("SELECT filename FROM forum_media").Scan(&remaining)
		t.Fatalf("forum_media 應只剩 shared.png，實際 %q", remaining)
	}

	// 第二次 Prune 應是冪等的：沒有東西可刪。
	again, err := p.Prune(context.Background())
	if err != nil || again != 0 {
		t.Fatalf("第二次 Prune 應回 (0, nil)，得到 (%d, %v)", again, err)
	}
}

func TestPruneKeepsImageWhenFileServerFails(t *testing.T) {
	db := purgeTestDB(t)

	old := time.Now().Add(-100 * 24 * time.Hour)
	id := insertPost(db, "a@example.com", "過期的文章", "old_a.png", &old)
	if id == 0 {
		t.Fatal("無法插入過期貼文")
	}
	if _, err := db.Exec(`INSERT INTO forum_media (filename, owner_email, created_at) VALUES ('old_a.png', 'a@example.com', ?)`, old); err != nil {
		t.Fatalf("插入媒體紀錄失敗: %v", err)
	}

	deleter := &recordingDeleter{fail: true}
	p := NewPruner(db, deleter, 90*24*time.Hour, time.Hour)
	if _, err := p.Prune(context.Background()); err != nil {
		t.Fatalf("Prune 失敗: %v", err)
	}

	// 檔案伺服器故障：貼文列照刪，forum_media 列保留（見檔頭：外部系統的
	// 故障不影響列的清理，孤兒檔案由下一次清理或使用者端收斂）。
	if countRows(db, "forum_posts") != 0 {
		t.Fatalf("貼文列應被清理")
	}
	if countRows(db, "forum_media") != 1 {
		t.Fatalf("檔案刪除失敗時 forum_media 列應保留，實際 %d", countRows(db, "forum_media"))
	}
}
