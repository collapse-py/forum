/*
Package purge 定期清除「軟刪除且已超過保留期」的貼文（backend/forum/purge/purge.go）。

【為什麼需要這個套件】

貼文的刪除是軟刪除（forum_posts.deleted_at，見 data/mysql.go 第 30 步）：
那一列永遠留在表裡，只是任何公開的讀取端都看不到它。docs/KNOWN_ISSUES.md
記錄的缺口正是這裡：「列數不會下降」——沒有 pruner、沒有保留期限、沒有
批次清除，而貼文的附圖因為仍被那篇已刪除的貼文引用，也不會被回收。

這個套件補的就是那個清理排程：

 1. 軟刪除超過 POST_RETENTION_DAYS 天的貼文，連同它的留言、按讚、收藏
    與指向它的檢舉工單一起硬刪。
 2. 該貼文的附圖在「不再被任何貼文或頭像引用」時，透過檔案伺服器刪除
    檔案，並移除 forum_media 的所有權紀錄。

【保留期限的語意】

保留期是「誤刪能救」與「資料不會無限累積」之間的刻度。90 天與稽核紀錄
（AUDIT_RETENTION_DAYS）相同：管理員用稽核與檢舉對帳時，兩邊需要同一尺度。
超過保留期的貼文不再可還原——後台的「已刪除清單」只列仍在保留期內的，
而還原端点对已硬刪的列回 404，這是清理排程存在的必然代價。

【為什麼清理在一個交易裡】

貼文列與它的留言／按讚／收藏／檢舉必须一起消失：只刪貼文而留下指向
已不存在 post_id 的留言，會讓「父文章存在嗎」的應用層檢查（見
KNOWN_ISSUES 的表格）從「還活著嗎」變成「根本不存在」，而那些檢查的
404 語意是刻意設計的一致性。交易讓「刪一半」不可能被观察到。

附圖的回收刻意放在 Commit **之後**：檔案伺服器是外部系統，它的故障不能
讓貼文清理回滾（那等于「清理失敗但資料还在」的另一种形式），而檔案刪除
失敗只留下一筆無引用的 forum_media 列——下一次清理或使用者自己的刪除
端點都能收斂（deleteForumFile 把 404 視為成功，重試安全）。

【批次與頻率】

每 tick 每表最多處理 batchSize 列，與 audit Pruner 同一個理由：一次刪掉
幾十萬列會讓交易與 undo log 膨脹，而這是可以慢慢做的清理工作。

【FileDeleter 為什麼是介面】

檔案的實際刪除发生在 httpapi.Server（它持有 FilesServerURL 與 token）。
purge 套件不該依賴 httpapi，因此 main 把 Server 的 DeleteForumFile 傳入。
nil 代表「沒有檔案伺服器」：列照刪，檔案留著——孤兒檔案由部署端的檔案
清理收斂，比讓清理排程因缺少外部服務而完全不跑好。
*/
package purge

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"forum/forum/logger"
)

// FileDeleter 是「刪除檔案伺服器上的一個檔案」的最小契約。
//
// 回傳錯誤代表「這次刪不掉」；呼叫端（Pruner）只記日誌，不讓外部系統的
// 故障影響貼文列的清理——兩者的失敗語意不同，綁在一起會讓「檔案伺服器
// 抖動」變成「清理排程停擺」。
type FileDeleter interface {
	DeleteFile(ctx context.Context, filename string) error
}

// Pruner 定期硬刪超過保留期的軟刪除貼文。
type Pruner struct {
	db        *sql.DB
	files     FileDeleter
	retention time.Duration
	interval  time.Duration
}

// NewPruner 建立清理器。retention <= 0 會被呼叫端（config）擋掉，這裡仍
// 保留防禦性回退，因為忘記設定不該讓清理變成「刪掉全部軟刪貼文」。
func NewPruner(db *sql.DB, files FileDeleter, retention, interval time.Duration) *Pruner {
	if retention <= 0 {
		retention = 90 * 24 * time.Hour
	}
	if interval <= 0 {
		interval = time.Hour
	}
	return &Pruner{db: db, files: files, retention: retention, interval: interval}
}

// Run 定期執行清理，直到 ctx 被取消。與 audit Pruner 同一個控制點：
// main 的優雅停止流程只取消一個 context。
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
			removed, err := p.Prune(ctx)
			if err != nil {
				logger.Warnf("[PURGE] 清理過期軟刪除貼文失敗: %v", err)
				continue
			}
			if removed > 0 {
				logger.Infof("[PURGE] 已清理 %d 篇超過保留期的軟刪除貼文", removed)
			}
		}
	}
}

// pruneBatchSize 是每 tick 處理的貼文上限。刻意不是「全部」：一次交易
// 刪幾十萬列會讓 undo log 膨脹，而清理可以慢慢做（見檔頭「批次與頻率」）。
const pruneBatchSize = 500

// Prune 硬刪 deleted_at 早於 cutoff 的貼文及其關聯，回傳清除的貼文筆數。
//
// nil 資料庫直接回 (0, nil)：與 audit Pruner 同一個理由——*sql.DB 的 nil
// 接收者呼叫方法時不會回傳錯誤，而是 panic，保護不能只放在 Run。
func (p *Pruner) Prune(ctx context.Context) (int, error) {
	if p == nil || p.db == nil {
		return 0, nil
	}

	cutoff := time.Now().Add(-p.retention)
	total := 0

	for {
		ids, images, err := p.fetchExpiredPosts(ctx, cutoff)
		if err != nil {
			return total, err
		}
		if len(ids) == 0 {
			return total, nil
		}

		if err := p.deletePosts(ctx, ids); err != nil {
			return total, err
		}
		p.releasePostImages(ctx, images)

		total += len(ids)
		if len(ids) < pruneBatchSize {
			return total, nil
		}
	}
}

// fetchExpiredPosts 取回一批超過保留期的軟刪除貼文（id 與附圖檔名）。
func (p *Pruner) fetchExpiredPosts(ctx context.Context, cutoff time.Time) ([]int64, []string, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, COALESCE(image_url, '') FROM forum_posts
		 WHERE deleted_at IS NOT NULL AND deleted_at < ? ORDER BY id LIMIT ?`,
		cutoff, pruneBatchSize)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0, pruneBatchSize)
	images := make([]string, 0, pruneBatchSize)
	for rows.Next() {
		var id int64
		var image string
		if err := rows.Scan(&id, &image); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		if image != "" {
			images = append(images, image)
		}
	}
	return ids, images, rows.Err()
}

// deletePosts 在一個交易裡硬刪貼文與它的留言／按讚／收藏／檢舉。
//
// 順序（先刪關聯再刪主列）不是偶然：關聯表沒有外鍵（見第 30 步的說明），
// 「父文章存在嗎」全靠應用層；交易內先清關聯，任何中途失敗都會回滾，
// 所以不存在「留言还在、貼文已不存在」的可见中間狀態。
func (p *Pruner) deletePosts(ctx context.Context, ids []int64) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	for _, query := range []string{
		`DELETE FROM forum_post_comments WHERE post_id IN (` + placeholders + `)`,
		`DELETE FROM forum_post_likes WHERE post_id IN (` + placeholders + `)`,
		`DELETE FROM forum_post_bookmarks WHERE post_id IN (` + placeholders + `)`,
		`DELETE FROM forum_reports WHERE target_type = 'post' AND target_id IN (` + placeholders + `)`,
		`DELETE FROM forum_posts WHERE id IN (` + placeholders + `)`,
	} {
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// releasePostImages 回收「只被已清理的貼文引用」的附圖。
//
// 引用計數在貼文列已刪除**之後**才查，所以被清理的貼文自然不計入——
// 這正是「不再被任何地方引用」的定義。查計數失敗時保守地跳過（把
// 「查不到」当成「沒人用」是最糟糕的誤判方向，見 httpapi 的
// releaseForumImage 同一條規則）。
func (p *Pruner) releasePostImages(ctx context.Context, images []string) {
	seen := make(map[string]bool, len(images))
	for _, raw := range images {
		name := imageFileName(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		references, err := p.imageReferenceCount(ctx, name)
		if err != nil {
			logger.WarnfContext(ctx, "[PURGE] 檢查圖片引用失敗，略過回收 filename=%s: %v", name, err)
			continue
		}
		if references > 0 {
			continue
		}
		if p.files != nil {
			if err := p.files.DeleteFile(ctx, name); err != nil {
				logger.WarnfContext(ctx, "[PURGE] 回收貼文附圖失敗 filename=%s: %v", name, err)
				continue
			}
		}
		if _, err := p.db.ExecContext(ctx, `DELETE FROM forum_media WHERE filename = ?`, name); err != nil {
			logger.WarnfContext(ctx, "[PURGE] 刪除圖片紀錄失敗 filename=%s: %v", name, err)
		}
	}
}

// imageReferenceCount 回傳有幾個地方在引用這個檔名（貼文附圖與頭像合計）。
//
// 與 httpapi.forumImageReferenceCount 相同的形状：含 LIKE 退路（舊資料可能
// 存著 /files/x.png 或完整網址），且檔名先跳脫 LIKE 的萬用字元——檔名白名單
// 裡有「_」，不跳脫會讓 %/a_b.png 也匹配 /aXb.png，多算一個引用。
func (p *Pruner) imageReferenceCount(ctx context.Context, name string) (int, error) {
	pattern := "%/" + escapeLike(name)
	var count int
	err := p.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM forum_posts WHERE image_url = ? OR image_url LIKE ?) +
			(SELECT COUNT(*) FROM forum_profiles WHERE avatar_url = ? OR avatar_url LIKE ?)`,
		name, pattern, name, pattern).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// imageFileName 把「任何一種本站檔案的寫法」收斂成純檔名；收斂不出來回空
// 字串。契約與 httpapi.forumImageFileName 相同：資料庫存純檔名，但舊資料
// 可能存 /files/x.png 或完整網址，取最後一段即可。
func imageFileName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		raw = raw[idx+1:]
	}
	return raw
}

// escapeLike 跳脫 MySQL LIKE 樣式的萬用字元與跳脫字元本身。
//
// 與 httpapi.escapeLikePattern 逐字相同。刻意不跨套件共用：httpapi 是
// handler 層，purge 是背景排程，讓背景排程依賴 handler 套件的工具函式
// 會讓「誰依賴誰」的方向反轉（purge 只需要三行的 Replacer）。
func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}
