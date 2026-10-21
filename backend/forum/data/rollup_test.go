/*
日別彙總的回填與刷新測試（backend/forum/data/rollup_test.go）。

需要真實 MySQL（FORUM_TEST_MYSQL_DSN，與 migration_test.go 同一個約定）。
驗的是三件事：

 1. 回填把「所有有資料的日期」算成正確的四個計數——包括跨日期、
    包括只有某一張表有資料的日期（UNION + LEFT JOIN 的形状正是为了
    这个：逐表 INSERT IGNORE 会让先插入的日期只剩一个計數）。
 2. 冪等：跑两次不会翻倍。
 3. 刷新重算最近三天：今天新增的按讚在下一次刷新后出现在今天的列，
    而更早的日期不被覆寫（那是回填的职责范围）。
*/
package data

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func rollupRow(db *sql.DB, day string) (users, posts, comments, likes int, exists bool) {
	err := db.QueryRow(`SELECT new_users, new_posts, new_comments, new_likes FROM forum_daily_stats WHERE day = ?`, day).
		Scan(&users, &posts, &comments, &likes)
	if err == sql.ErrNoRows {
		return 0, 0, 0, 0, false
	}
	return users, posts, comments, likes, err == nil
}

func TestRollupBackfillAndRefreshMySQL(t *testing.T) {
	db, _ := tempDatabase(t)
	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("MigrateMySQL 失敗: %v", err)
	}

	ctx := context.Background()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	oldDay := today.AddDate(0, 0, -5)

	// 舊日期：兩筆使用者、三筆貼文（其中一筆已軟刪——計數是「那天產生」）、
	// 兩筆留言、四筆按讚。
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(`INSERT INTO forum_users (email, status, created_at, updated_at) VALUES (?, 'ACTIVE', ?, ?)`,
			"u"+string(rune('a'+i))+"@example.com", oldDay.Add(time.Duration(i)*time.Hour), oldDay.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatalf("插入使用者失敗: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		deleted := sql.NullTime{}
		if i == 2 {
			deleted = sql.NullTime{Time: now, Valid: true}
		}
		if _, err := db.Exec(`INSERT INTO forum_posts (author_email, content, image_url, created_at, deleted_at) VALUES (?, ?, '', ?, ?)`,
			"a@example.com", "post "+string(rune('a'+i)), oldDay.Add(time.Duration(i)*time.Hour), deleted); err != nil {
			t.Fatalf("插入貼文失敗: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(`INSERT INTO forum_post_comments (post_id, author_email, content, created_at) VALUES (1, ?, ?, ?)`,
			"b"+string(rune('a'+i))+"@example.com", "comment", oldDay.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatalf("插入留言失敗: %v", err)
		}
	}
	for i := 0; i < 4; i++ {
		if _, err := db.Exec(`INSERT INTO forum_post_likes (post_id, author_email, created_at) VALUES (1, ?, ?)`,
			"c"+string(rune('a'+i))+"@example.com", oldDay.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatalf("插入按讚失敗: %v", err)
		}
	}

	// 今天：一筆貼文、一筆按讚。
	if _, err := db.Exec(`INSERT INTO forum_posts (author_email, content, image_url, created_at) VALUES (?, ?, '', ?)`,
		"d@example.com", "today post", now); err != nil {
		t.Fatalf("插入今日貼文失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_post_likes (post_id, author_email, created_at) VALUES (4, ?, ?)`,
		"e@example.com", now); err != nil {
		t.Fatalf("插入今日按讚失敗: %v", err)
	}

	if err := BackfillDailyStats(ctx, db); err != nil {
		t.Fatalf("回填失敗: %v", err)
	}

	// 兩個日期鍵的口徑不同，這是刻意的：
	//
	//   oldStr  —— BackfillDailyStats 以 DATE(created_at) 分組，而 time.Time
	//               寫入時以 DSN 的 loc（CI 與本測試的約定是 UTC）落欄，
	//               因此回填列的鍵是「該瞬間的 UTC 日期」。主機時區不是
	//               UTC 時本地日期會與它差一天（本地午夜 = 前一日 16:00Z），
	//               直接用本地日期字串查會得到「彙總列不存在」。
	//   todayStr —— RefreshDailyStats 以主機本地時區的日界寫入 day 鍵
	//               （見 rollup.go 檔頭「時區」一節），因此刷新列的鍵是
	//               本地日期。
	oldStr := oldDay.UTC().Format("2006-01-02")
	todayStr := today.Format("2006-01-02")

	u, p, c, l, ok := rollupRow(db, oldStr)
	if !ok {
		t.Fatalf("舊日期 %s 的彙總列不存在", oldStr)
	}
	if u != 2 || p != 3 || c != 2 || l != 4 {
		t.Fatalf("舊日期計數 = (%d,%d,%d,%d), want (2,3,2,4)", u, p, c, l)
	}

	// 冪等：第二次回填不应翻倍。
	if err := BackfillDailyStats(ctx, db); err != nil {
		t.Fatalf("第二次回填失敗: %v", err)
	}
	u, p, c, l, _ = rollupRow(db, oldStr)
	if u != 2 || p != 3 || c != 2 || l != 4 {
		t.Fatalf("第二次回填後計數 = (%d,%d,%d,%d), want (2,3,2,4)（冪等）", u, p, c, l)
	}

	// 刷新重算今天：新增的貼文與按讚必须出现在今天的列。
	if err := RefreshDailyStats(ctx, db); err != nil {
		t.Fatalf("刷新失敗: %v", err)
	}
	u, p, c, l, ok = rollupRow(db, todayStr)
	if !ok {
		t.Fatalf("今天的彙總列不存在")
	}
	if p != 1 || l != 1 {
		t.Fatalf("今天計數 = (posts=%d, likes=%d), want (1,1)", p, l)
	}

	// 刷新只覆寫最近三天：舊日期不被觸碰（它已被清理排程或歷史決定，
	// 重算反而可能少算——見 rollup.go 檔頭）。
	u, p, c, l, _ = rollupRow(db, oldStr)
	if u != 2 || p != 3 || c != 2 || l != 4 {
		t.Fatalf("刷新不应觸碰舊日期，實際 = (%d,%d,%d,%d)", u, p, c, l)
	}
}

func TestRollupNilSafety(t *testing.T) {
	if err := BackfillDailyStats(context.Background(), nil); err != nil {
		t.Fatalf("nil 資料庫的回填應回 nil，得到 %v", err)
	}
	if err := RefreshDailyStats(context.Background(), nil); err != nil {
		t.Fatalf("nil 資料庫的刷新應回 nil，得到 %v", err)
	}
}
