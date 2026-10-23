/*
日別彙總的回填與刷新（backend/forum/data/rollup.go）。

【為什麼需要這個檔案】

/admin/stats 的日別序列（forum_daily_stats，MigrateMySQL 第 35 步）是純衍生
資料：它的正確性來自「定期重算」，而不是「每次寫入順帶更新」。寫入路徑上
順帶更新（在貼文 INSERT 的同一交易裡 increment 一列）會讓八個 handler 都要
記得做這件事——漏一處就是「圖上少一天」，而那種漏只能靠人工比對發現。
定期重算把「記得」這件事集中到一個地方：一次全表主鍵掃描換回所有計數的
一致性。

【兩條路徑的分工】

	BackfillDailyStats  把「彙總表裡還沒有的日期」從來源表算出來。冪等：
	已存在的日期刻意不覆寫——理由見下。

	RefreshDailyStats   重算最近 refreshWindowDays 天。只算「今天」不夠：
	行程在一天中途重啟時，今天這列可能只寫到一半；跨日界的重啟同理。三天
	的窗口讓「重啟後補回」不需要任何額外狀態——下一次 tick 自然把前幾天
	重算成完整值。

【為什麼已存在的日期不覆寫】

	這張表記的是**歷史量**：「那天產生了多少」，而不是「今天還看得到多少」。
	軟刪除的貼文照算（見 schema 的說明），因此管理員刪掉一篇文章之後，那一
	天的數字不會變 —— 那是「那天真的產生了那麼多篇」。

	清理排程（forum/purge）在保留期後硬刪貼文，照這個定義也不該改變那些
	數字：文章確實在那天被產生了，只是後來被回收了。因此 purge 不會刪
	forum_daily_stats 的任何一列，而 BackfillDailyStats 的 INSERT IGNORE
	也不會把它們改小。

	（早期版本把 INSERT IGNORE 的理由寫成「purge 可能已硬刪那些日期」——
	那是不成立的：purge 從來不動這張表。真正成立的理由是上面這個：這是
	歷史量，重算會讓數字隨著回收而變小，等於改寫歷史。）

【為什麼重算而不是累加】

累加（ON DUPLICATE KEY UPDATE new_posts = new_posts + 1）需要每次寫入都
觸碰這張表，而刪除（軟刪清理、留言刪除）沒有對應的減操作——累加表會
只增不減，日別序列與「那天真的發生了什麼」逐漸偏離。重算沒有這個問題：
每一列的值永遠是「來源表在那一天的真實計數」，代價只是每十分鐘一次有界
的 COUNT 掃描（窗口只有三天，走 created_at 索引）。

【時區】

created_at 由應用層以 time.Now()（主機本地時區）寫入，因此 DATE(created_at)
的分組等於「主機上的自然日」，而 RefreshDailyStats 的日界也用本地時區——
兩邊必須一致，否則「今天」的列會在兩個時區之間錯位。這與 stats_handlers.go
的說明是同一條規則。
*/
package data

import (
	"context"
	"database/sql"
	"time"
)

// refreshWindowDays 是刷新排程重算的天數。三天不是精確值，是「行程重啟
// 最可能跨的日界數」的上界：任何一天中途重啟，下一次 tick 都能把那幾天
// 重算成完整值。
const refreshWindowDays = 3

// BackfillDailyStats 把彙總表中缺失的日期從來源表回填。
//
// 冪等（INSERT IGNORE）：已存在的日期不覆寫，理由見檔頭。一次跑完所有
// 日期而不是分批：回填只在「表裡沒有那列」時寫入，而日期列的數量等於
// 站台的「有資料的天數」——對任何論壇規模都是有界的。
func BackfillDailyStats(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	_, err := db.ExecContext(ctx, `
		INSERT IGNORE INTO forum_daily_stats (day, new_users, new_posts, new_comments, new_likes)
		SELECT d.d,
			COALESCE(u.c, 0), COALESCE(p.c, 0), COALESCE(c.c, 0), COALESCE(l.c, 0)
		FROM (
			SELECT DATE(created_at) AS d FROM forum_users
			UNION SELECT DATE(created_at) FROM forum_posts
			UNION SELECT DATE(created_at) FROM forum_post_comments
			UNION SELECT DATE(created_at) FROM forum_post_likes
		) d
		LEFT JOIN (SELECT DATE(created_at) AS d, COUNT(*) AS c FROM forum_users GROUP BY DATE(created_at)) u ON u.d = d.d
		LEFT JOIN (SELECT DATE(created_at) AS d, COUNT(*) AS c FROM forum_posts GROUP BY DATE(created_at)) p ON p.d = d.d
		LEFT JOIN (SELECT DATE(created_at) AS d, COUNT(*) AS c FROM forum_post_comments GROUP BY DATE(created_at)) c ON c.d = d.d
		LEFT JOIN (SELECT DATE(created_at) AS d, COUNT(*) AS c FROM forum_post_likes GROUP BY DATE(created_at)) l ON l.d = d.d`)
	return err
}

// RefreshDailyStats 重算最近 refreshWindowDays 天的彙總列。
//
// 每次都是完整重算（ON DUPLICATE KEY UPDATE 覆寫），不是累加：見檔頭
// 「為什麼重算而不是累加」。四個 COUNT 都落在單日範圍的 created_at 索引
// 掃描上，量級是「那一天新增的筆數」。
func RefreshDailyStats(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for i := 0; i < refreshWindowDays; i++ {
		dayStart := today.AddDate(0, 0, -i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		_, err := db.ExecContext(ctx, `
			INSERT INTO forum_daily_stats (day, new_users, new_posts, new_comments, new_likes)
			VALUES (?,
				(SELECT COUNT(*) FROM forum_users WHERE created_at >= ? AND created_at < ?),
				(SELECT COUNT(*) FROM forum_posts WHERE created_at >= ? AND created_at < ?),
				(SELECT COUNT(*) FROM forum_post_comments WHERE created_at >= ? AND created_at < ?),
				(SELECT COUNT(*) FROM forum_post_likes WHERE created_at >= ? AND created_at < ?))
			ON DUPLICATE KEY UPDATE
				new_users = VALUES(new_users),
				new_posts = VALUES(new_posts),
				new_comments = VALUES(new_comments),
				new_likes = VALUES(new_likes)`,
			dayStart.Format("2006-01-02"),
			dayStart, dayEnd,
			dayStart, dayEnd,
			dayStart, dayEnd,
			dayStart, dayEnd)
		if err != nil {
			return err
		}
	}
	return nil
}
