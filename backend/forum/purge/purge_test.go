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

與 data/migration_test.go 的理由相同：sqlmock 只能回傳「預先寫好的答案」，
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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"forum/forum/idgen"

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
		// 舊資料的完整網址帶 token（見 data.MigrateMySQL 第 18 步的說明）。
		{"http://localhost:7070/files/a_b.png?token=xyz", "a_b.png"},
		{"/files/a_b.png?token=xyz", "a_b.png"},
		// 舊版遷移的損壞值：先包成 JSON 再切路徑會把 `"]` 黏在檔名上。
		{`a_b.png"]`, "a_b.png"},
		{`["http://localhost:7070/files/a_b.png?token=xyz"]`, "a_b.png"},
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

// recordingDeleter 記錄被「刪除」的檔名，並可模擬檔案伺服器的故障。
//
// 刪檔在正式程式裡是並行的，因此 deleted 的 append 必須加鎖；測試端在
// Prune 返回後才讀取，透過 WaitGroup 的 join 取得 happens-before。
type recordingDeleter struct {
	mu      sync.Mutex
	deleted []string
	fail    bool
}

func (d *recordingDeleter) DeleteFile(_ context.Context, name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fail {
		return errors.New("file server down")
	}
	d.deleted = append(d.deleted, name)
	return nil
}

func (d *recordingDeleter) setFail(fail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fail = fail
}

func (d *recordingDeleter) deletedNames() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.deleted...)
}

var purgeDBCounter atomic.Int64

// purgeTestDB 建立一個跑清理排程專用 schema 的拋棄式資料庫；未設定
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
	// 刻意用 t.Cleanup 而不是 defer：Cleanup 以 LIFO 執行，而下面的「DROP
	// 拋棄式資料庫」也註冊在 Cleanup 裡。若這裡用 defer，server 會在函式返回時
	// **先於** Cleanup 被關閉，DROP 於是在已關閉的連線池上執行、再被
	// `_, _ =` 靜默吞掉 —— 症狀是每跑一次測試就在伺服器上留一個
	// forum_purge_test_* 資料庫（2026-10-23 在本機 MySQL 實測留下 6 個，
	// CI 的 runner 是一次性的所以沒有人看到）。註冊順序決定關閉順序：
	// 這裡先註冊 ⇒ 最後才執行，DROP 因此在連線池還開著的時候發生。
	t.Cleanup(func() { _ = server.Close() })
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

// migrateForPurge 只建立清理排程會觸碰的表，避免完整 MigrateMySQL 的
// 37 步在每次測試都跑一遍。列的形狀與 data/mysql.go 逐字相同（測試驗的是
// 「清理排程的 SQL 對真實 schema 成立」，用同一份 DDL 才有意義）。
func migrateForPurge(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS forum_posts (
			id BIGINT NOT NULL,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			image_url VARCHAR(255) NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NULL,
			pinned TINYINT NOT NULL DEFAULT 0,
			deleted_at DATETIME NULL,
			PRIMARY KEY (id),
			KEY idx_forum_posts_deleted_at_id (deleted_at, id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS forum_post_comments (
			id BIGINT NOT NULL,
			post_id BIGINT NOT NULL,
			parent_id BIGINT NULL,
			depth TINYINT NOT NULL DEFAULT 1,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_forum_post_comments_post_id (post_id)
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
			id BIGINT NOT NULL PRIMARY KEY,
			target_type VARCHAR(20) NOT NULL,
			target_id BIGINT NOT NULL,
			reporter_email VARCHAR(255) NOT NULL,
			reason VARCHAR(500) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			created_at DATETIME NOT NULL,
			reviewed_at DATETIME NULL,
			reviewed_by VARCHAR(255) NOT NULL DEFAULT '',
			KEY idx_forum_reports_target (target_type, target_id)
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
		`CREATE TABLE IF NOT EXISTS forum_media_deletions (
			filename VARCHAR(128) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (filename),
			KEY idx_forum_media_deletions_created_at (created_at)
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
	id := idgen.Next()
	if _, err := db.Exec(
		`INSERT INTO forum_posts (id, author_email, content, image_url, created_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, author, content, image, time.Now(), deletedAt); err != nil {
		return 0
	}
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
	commentID := idgen.Next()
	if _, err := db.Exec(`INSERT INTO forum_post_comments (id, post_id, author_email, content, created_at) VALUES (?, ?, 'b@example.com', '留言', ?)`, commentID, expired, old); err != nil {
		t.Fatalf("插入留言失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_post_likes (post_id, author_email, created_at) VALUES (?, 'c@example.com', ?)`, expired, old); err != nil {
		t.Fatalf("插入按讚失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_post_bookmarks (post_id, author_email, created_at) VALUES (?, 'd@example.com', ?)`, expired, old); err != nil {
		t.Fatalf("插入收藏失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_reports (id, target_type, target_id, reporter_email, reason, created_at) VALUES (?, 'post', ?, 'e@example.com', '檢舉', ?)`, idgen.Next(), expired, old); err != nil {
		t.Fatalf("插入檢舉失敗: %v", err)
	}
	// 留言檢舉走的是 deletePosts 的 JOIN 路徑；沒有這筆的話，那條 SQL 只會
	// 在單元測試的替身上被驗到，真實 MySQL 的語意（別名、JOIN、target_type）
	// 不會被執行。
	if _, err := db.Exec(`INSERT INTO forum_reports (id, target_type, target_id, reporter_email, reason, created_at) VALUES (?, 'comment', ?, 'h@example.com', '檢舉留言', ?)`, idgen.Next(), commentID, old); err != nil {
		t.Fatalf("插入留言檢舉失敗: %v", err)
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

	deletedFiles := deleter.deletedNames()
	if len(deletedFiles) != 1 || deletedFiles[0] != "old_a.png" {
		t.Fatalf("只應回收不再被引用的附圖，實際 %v", deletedFiles)
	}
	if countRows(db, "forum_media") != 1 {
		var remaining string
		_ = db.QueryRow("SELECT filename FROM forum_media").Scan(&remaining)
		t.Fatalf("forum_media 應只剩 shared.png，實際 %q", remaining)
	}
	if countRows(db, "forum_media_deletions") != 0 {
		t.Fatalf("成功刪檔後 forum_media_deletions 應清空，實際 %d", countRows(db, "forum_media_deletions"))
	}

	// 第二次 Prune 應是冪等的：沒有東西可刪。
	again, err := p.Prune(context.Background())
	if err != nil || again != 0 {
		t.Fatalf("第二次 Prune 應回 (0, nil)，得到 (%d, %v)", again, err)
	}
}

// TestDeletePostsRechecksRestoredPostsMySQL 在真實 MySQL 上驗證 H1 的
// TOCTOU 防線：取回（fetchExpiredPosts）之後才被還原的貼文，deletePosts
// 不能在交易裡把它硬刪。
//
// 單元測試用 driver 替身模擬「鎖定查詢只回一個 id」；這一支驗的是同一條
// 路徑在真實 schema 上的 SQL 語意（FOR UPDATE + deleted_at 條件）。
func TestDeletePostsRechecksRestoredPostsMySQL(t *testing.T) {
	db := purgeTestDB(t)

	old := time.Now().Add(-100 * 24 * time.Hour)
	id := insertPost(db, "restore@example.com", "取回後被還原", "", &old)
	if id == 0 {
		t.Fatal("無法插入過期貼文")
	}

	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	p := NewPruner(db, nil, 90*24*time.Hour, time.Hour)
	posts, err := p.fetchExpiredPosts(context.Background(), cutoff, 10)
	if err != nil {
		t.Fatalf("fetchExpiredPosts: %v", err)
	}
	if len(posts) != 1 || posts[0].id != id {
		t.Fatalf("取回的過期貼文 = %+v, want id=%d", posts, id)
	}

	// 模擬管理員在取回與刪除之間按下還原。
	if _, err := db.Exec(`UPDATE forum_posts SET deleted_at = NULL WHERE id = ?`, id); err != nil {
		t.Fatalf("還原貼文失敗: %v", err)
	}

	deleted, err := p.deletePosts(context.Background(), posts, cutoff)
	if err != nil {
		t.Fatalf("deletePosts: %v", err)
	}
	if len(deleted) != 0 {
		t.Errorf("deletePosts 刪除了已被還原的貼文: %v", deleted)
	}

	var deletedAt sql.NullTime
	if err := db.QueryRow(`SELECT deleted_at FROM forum_posts WHERE id = ?`, id).Scan(&deletedAt); err != nil {
		t.Fatalf("讀回貼文失敗（可能已被硬刪）: %v", err)
	}
	if deletedAt.Valid {
		t.Errorf("deleted_at = %v, want NULL", deletedAt.Time)
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

	// 檔案伺服器故障：貼文列照刪，forum_media 列保留，檔名留在佇列。
	if countRows(db, "forum_posts") != 0 {
		t.Fatalf("貼文列應被清理")
	}
	if countRows(db, "forum_media") != 1 {
		t.Fatalf("檔案刪除失敗時 forum_media 列應保留，實際 %d", countRows(db, "forum_media"))
	}
	if countRows(db, "forum_media_deletions") != 1 {
		t.Fatalf("檔案刪除失敗時檔名應留在佇列，實際 %d", countRows(db, "forum_media_deletions"))
	}

	// 檔案伺服器恢復後，下一次 Prune（即使沒有過期貼文）必須從佇列重試。
	deleter.setFail(false)
	if _, err := p.Prune(context.Background()); err != nil {
		t.Fatalf("重試 Prune 失敗: %v", err)
	}
	if countRows(db, "forum_media") != 0 {
		t.Fatalf("重試成功後 forum_media 列應被移除，實際 %d", countRows(db, "forum_media"))
	}
	if countRows(db, "forum_media_deletions") != 0 {
		t.Fatalf("重試成功後佇列應清空，實際 %d", countRows(db, "forum_media_deletions"))
	}
	if got := deleter.deletedNames(); len(got) != 1 || got[0] != "old_a.png" {
		t.Fatalf("重試成功後應刪除 old_a.png，實際 %v", got)
	}
}
