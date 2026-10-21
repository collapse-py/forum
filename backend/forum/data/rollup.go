/*
日別彙總的回填與刷新（backend/forum/data/rollup.go）。

【為什麼需要這個檔案】

/admin/stats 的日別序列（forum_daily_stats，MigrateMySQL 第 35 步）是純衍生
資料：它的正確性来自「定期重算」，而不是「每次寫入順帶更新」。寫入路徑上
順帶更新（在貼文 INSERT 的同一交易裡 increment 一列）会让八個 handler 都要
記得做这件事——漏一处就是「圖上少一天」，而那种漏只能靠人工比对发现。
定期重算把「记得」这件事集中到一个地方：一次全表主鍵扫描换回所有計數的
一致性。

【兩條路徑的分工】

	BackfillDailyStats  把「彙總表裡还没有的日期」從來源表算出来。冪等：
	已存在的日期刻意不覆寫——那些日期可能已被清理排程（purge）硬刪，重算
	反而少算；而「已存在的日期」正是刷新排程負責的範圍。

	RefreshDailyStats   重算最近 refreshWindowDays 天。只算「今天」不够：
	行程在一天中途重啟时，今天这列可能只寫到一半；跨日界的重啟同理。三天
	的窗口让「重启后补回」不需要任何额外状态——下一次 tick 自然把前几天
	重算成完整值。

【為什麼重算而不是累加】

累加（ON DUPLICATE KEY UPDATE new_posts = new_posts + 1）需要每次寫入都
觸碰这张表，而删除（軟刪清理、留言刪除）没有对应的减操作——累加表会
只增不减，日別序列与「那天真的发生了什么」逐渐偏离。重算没有这个问题：
每一列的值永远是「來源表在那一天的真实計數」，代价只是每十分钟一次有界
的 COUNT 扫描（窗口只有三天，走 created_at 索引）。

【時區】

created_at 由應用層以 time.Now()（主機本地時區）寫入，因此 DATE(created_at)
的分组等于「主機上的自然日」，而 RefreshDailyStats 的日界也用本地時區——
兩邊必须一致，否则「今天」的列会在两个时区之间错位。这与 stats_handlers.go
的说明是同一条规则。
*/
package data

import (
	"context"
	"database/sql"
	"time"
)

// refreshWindowDays 是刷新排程重算的天數。三天不是精确值，是「行程重啟
// 最可能跨的日界數」的上界：任何一天中途重啟，下一次 tick 都能把那几天
// 重算成完整值。
const refreshWindowDays = 3

// BackfillDailyStats 把彙總表中缺失的日期從來源表回填。
//
// 冪等（INSERT IGNORE）：已存在的日期不覆寫，理由见檔頭。一次跑完所有
// 日期而不是分批：回填只在「表裡没有那列」时寫入，而日期列的数量等于
// 站台的「有資料的天數」——对任何论坛规模都是有界的。
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
// 「為什麼重算而不是累加」。四个 COUNT 都落在單日範圍的 created_at 索引
// 扫描上，量級是「那一天新增的筆數」。
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
