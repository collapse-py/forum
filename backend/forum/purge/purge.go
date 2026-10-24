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
超過保留期的貼文不再可還原——而還原端點對已硬刪的列回 404，這是清理排程
存在的必然代價。

（後台的「已刪除清單」只過濾 deleted_at IS NOT NULL，**沒有**保留期條件：
一篇文章過期到被清理排程處理之前，它仍然列得出來、仍然可以還原。清理排
程每小時跑一次，因此那個窗口是一個小時加上尚未排到的批次。）

【與日別彙總的關係】

forum_daily_stats 記的是「那天產生了多少」的**歷史量**（軟刪除的貼文照算，
見 data/rollup.go 的說明），而清理排程把貼文硬刪掉並不改變那些數字 ——
文章確實在那天被產生了，只是後來被回收。因此這個套件刻意不動
forum_daily_stats 的任何一列。

【為什麼清理在一個交易裡】

貼文列與它的留言／按讚／收藏／檢舉必須一起消失：只刪貼文而留下指向
已不存在 post_id 的留言，會讓「父文章存在嗎」的應用層檢查（見
KNOWN_ISSUES 的表格）從「還活著嗎」變成「根本不存在」，而那些檢查的
404 語意是刻意設計的一致性。交易讓「刪一半」不可能被觀察到。

附圖的回收刻意放在 Commit **之後**：檔案伺服器是外部系統，它的故障不能
讓貼文清理回滾（那等於「清理失敗但資料還在」的另一種形式）。因此交易內
只把「這次硬刪的附圖檔名」寫進 forum_media_deletions 佇列；Commit 後才
嘗試刪檔。刪檔失敗、程式崩潰或檔案伺服器暫時不可用時，佇列列還在，
下一個 tick 會重試——「下一次清理會收斂」因此是交易保證，而不是願望。

【批次與頻率】

每個交易最多處理 pruneBatchSize 列，而每個 tick 最多處理 pruneMaxPerTick 列。
兩層上限是不同的事情：交易上限讓「一次刪掉幾十萬列」的單一交易與 undo
log 膨脹不會發生，tick 上限讓「保留期第一次到期時累積了幾十萬篇」不會
在一個 tick 裡全部清完 —— 那一輪會佔住連線池、跑幾十萬次圖片引用查詢，
並對檔案伺服器發幾十萬次刪除要求，整段時間裡論壇的正常請求都在排隊。
累積量因此以「每小時 pruneMaxPerTick 列」的速度排掉，剩下的交給後續的
tick。

【FileDeleter 為什麼是介面】

檔案的實際刪除發生在 httpapi.Server（它持有 FilesServerURL 與 token）。
purge 套件不該依賴 httpapi，因此 main 把 Server 的 DeleteFile 傳入。
nil 代表「沒有檔案伺服器」：列照刪，檔案留著——而且 **forum_media 的
所有權紀錄也留著**：那一列是「誰上傳了這個檔案」的唯一紀錄，刪掉它
會讓檔案變成沒有任何程式認得的孤兒（連「使用者刪除自己上傳的圖片」
那條路徑都會回答「不是上傳者」）。孤兒檔案由部署端的檔案清理收斂，
比讓清理排程因缺少外部服務而完全不跑好；這種模式下也不會寫入刪除佇列，
因為佇列唯一的消費者就是檔案刪除。

【圖片刪除佇列】

forum_media_deletions 與貼文刪除在同一個交易裡寫入，因此「貼文已硬刪」
與「檔名已排進待刪佇列」不可能只發生一半。佇列以檔名為主鍵（同一張圖被
多篇貼文引用時只需刪一次），每個 tick 先重試舊的佇列列，再處理本 tick
新刪的貼文；檔名仍被其他貼文或頭像引用時就從佇列移除、保留 forum_media
列，等最後一個引用消失時由那個引用者的刪除流程重新排入。
*/
package purge

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"sync"
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

// pruneBatchSize 是**每個交易**處理的貼文上限。
//
// 刻意不是「全部」：一次交易刪幾十萬列會讓 undo log 膨脹，而清理可以慢慢
// 做（見檔頭「批次與頻率」）。
const pruneBatchSize = 500

// pruneMaxPerTick 是**每個 tick**（每小時一次）處理的貼文總上限。
//
// 與 pruneBatchSize 是不同的事情：交易上限管的是「單一交易不能太大」，
// tick 上限管的是「一個 tick 不能佔用資源太久」。沒有後者的話，保留期
// 第一次到期時累積的幾十萬篇會在一個 tick 裡全部清完 —— 連線池被佔住、
// 幾十萬次圖片引用查詢、幾十萬次檔案伺服器刪除要求，而論壇的正常請求
// 在那整段時間裡都在排隊。
//
// 2000 列／小時等於每天 48,000 列：幾十萬篇的累積量會在幾天內排完，
// 而每個 tick 的成本維持在有界的範圍裡。
const pruneMaxPerTick = 2000

// 圖片刪除的並行度、單檔期限與每個 tick 的總預算。
//
// 舊版循序刪檔，而每個 DeleteFile 的 HTTP 期限是 30 秒：2000 張的最壞
// 情況是 16.7 小時，prune 每小時只跑一次，等於永遠追不上。這裡把外部
// 呼叫並行化（8 條）、把單檔期限縮到 5 秒，並用 10 分鐘的 tick 預算
// 蓋住整個圖片階段——檔案伺服器整體卡住時不會拖垮排程或關機流程。
//
// 10 分鐘 × 8 條 / 5 秒 ≈ 960 次「慢速失敗」；健康的檔案伺服器是毫秒級，
// 同一段時間足以處理數千張。沒跑完的留在佇列，下一個 tick 繼續。
const (
	mediaDeleteConcurrency = 8
	mediaDeleteTimeout     = 5 * time.Second
	mediaDeleteBudget      = 10 * time.Minute
)

// Prune 硬刪 deleted_at 早於 cutoff 的貼文及其關聯，回傳清除的貼文筆數。
//
// nil 資料庫直接回 (0, nil)：與 audit Pruner 同一個理由——*sql.DB 的 nil
// 接收者呼叫方法時不會回傳錯誤，而是 panic，保護不能只放在 Run。
//
// 一個 tick 最多硬刪 pruneMaxPerTick 列（見檔頭「批次與頻率」）：剩下的
// 交給下一個 tick。若批次中有貼文在交易內被還原，實際刪除數會少於取回數，
// 迴圈會再取下一批補足；還原的列已不再符合條件，不會被重複取回。
func (p *Pruner) Prune(ctx context.Context) (int, error) {
	if p == nil || p.db == nil {
		return 0, nil
	}

	// 圖片工作（佇列重試 + 本 tick 的立即回收）共用一個預算。預算用完時
	// 只停止圖片工作，貼文列仍然照常清理；沒刪到的檔名留在佇列等下一 tick。
	mediaCtx, cancelMedia := context.WithTimeout(ctx, mediaDeleteBudget)
	defer cancelMedia()

	// 先重試先前失敗的圖片。即使這個 tick 沒有過期貼文，佇列也必須被處理；
	// 否則一次檔案伺服器故障就會讓那些檔名永遠留在表裡。
	p.releaseQueuedImages(mediaCtx, pruneMaxPerTick)

	cutoff := time.Now().Add(-p.retention)
	total := 0

	for total < pruneMaxPerTick {
		// 最後一批不超過 tick 上限的剩餘量。
		limit := pruneBatchSize
		if remaining := pruneMaxPerTick - total; remaining < limit {
			limit = remaining
		}

		posts, err := p.fetchExpiredPosts(ctx, cutoff, limit)
		if err != nil {
			return total, err
		}
		if len(posts) == 0 {
			break
		}

		deleted, err := p.deletePosts(ctx, posts, cutoff)
		if err != nil {
			return total, err
		}
		// 只回收真的被硬刪的貼文附圖：批次取回與交易開始之間被管理員還原的
		// 貼文仍然存在，它的附圖也仍然被引用，不能放進回收清單。
		if p.files != nil {
			p.releasePostImages(mediaCtx, deletedPostImages(posts, deleted))
		}

		total += len(deleted)
		if len(posts) < limit {
			break
		}
	}
	return total, nil
}

// expiredPost 是清理排程眼中的一則過期貼文：主鍵與可能為空的附圖。
//
// id 與 image 放在同一個結構裡，是為了讓「實際刪了哪些貼文」與「回收哪些
// 附圖」能夠對齊。若拆成兩個平行 slice，只要交易內跳過任何一列（例如管理員
// 剛好還原），索引就會錯位，回收端可能刪掉仍然被引用的檔案。
type expiredPost struct {
	id     int64
	images []string
}

// fetchExpiredPosts 取回一批超過保留期的軟刪除貼文（id 與附圖檔名）。
//
// ORDER BY 與索引 idx_forum_posts_deleted_at_id 對齊：只按 id 排序時，
// MySQL 必須把符合 deleted_at 範圍的列全部取出來再 filesort；改成
// (deleted_at, id) 之後，複合索引本身就有序，LIMIT 可以在索引上停下。
func (p *Pruner) fetchExpiredPosts(ctx context.Context, cutoff time.Time, limit int) ([]expiredPost, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, COALESCE(image_url, '') FROM forum_posts
		 WHERE deleted_at IS NOT NULL AND deleted_at < ? ORDER BY deleted_at, id LIMIT ?`,
		cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]expiredPost, 0, limit)
	for rows.Next() {
		var post expiredPost
		var raw string
		if err := rows.Scan(&post.id, &raw); err != nil {
			return nil, err
		}
		post.images = parsePostImages(raw)
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

// deletePosts 在一個交易裡硬刪貼文與它的留言／按讚／收藏／檢舉，回傳真正
// 被刪除的貼文 id。
//
// 順序（先刪關聯再刪主列）不是偶然：關聯表沒有外鍵（見第 30 步的說明），
// 「父文章存在嗎」全靠應用層；交易內先清關聯，任何中途失敗都會回滾，
// 所以不存在「留言還在、貼文已不存在」的可見中間狀態。
//
// 為什麼要重新確認 deleted_at：fetchExpiredPosts 與這裡的交易是兩個獨立的
// 時間點。管理員的「還原」只是一個 UPDATE deleted_at = NULL；若直接信任
// 先前讀到的 id，還原可能在取回之後、DELETE 之前提交，使用者看到「還原
// 成功」的貼文卻在下一個瞬間被硬刪，而且沒有留下任何可追查的痕跡。
//
// 鎖定查詢是 current read（FOR UPDATE），在 REPEATABLE READ 下仍會看到
// 最新已提交的 deleted_at；被還原的列不會出現在結果裡。從鎖定到 Commit
// 為止，還原的 UPDATE 會等待；等到的是一列已經不存在（受影響 0 列），
// 而不是「先還原、再被刪掉」。
func (p *Pruner) deletePosts(ctx context.Context, posts []expiredPost, cutoff time.Time) ([]int64, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ids := make([]int64, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.id)
	}

	lockedIDs, err := lockExpiredPosts(ctx, tx, ids, cutoff)
	if err != nil {
		return nil, err
	}
	if len(lockedIDs) == 0 {
		// 整批都在取回後被還原：沒有任何東西要刪。Rollback 由 defer 負責。
		return nil, nil
	}

	placeholders, args := idQueryArgs(lockedIDs)

	/*
	 * 指向留言的檢舉工單。
	 *
	 * forum_reports.target_id 是多型參照（沒有外鍵），target_type 區分
	 * 「檢舉貼文」與「檢舉留言」。只刪 target_type='post' 會讓留言的工單
	 * 留下來、指向已經被刪掉的留言 —— 後台工單清單解析出的作者與內容會
	 * 是空的，而裁決端點對它 404。那種工單既沒辦法處理也沒辦法關掉。
	 *
	 * 這裡用 JOIN 直接從「貼文 → 留言 → 工單」刪除，而不是先把留言 id
	 * 全部取回再組 IN (...)。後者在熱門貼文上會把幾萬個 id 載進記憶體，
	 * 而且 MySQL 的 placeholder 上限是 65,535 —— 超過就讓整個清理交易
	 * 永遠失敗，連貼文都刪不掉。JOIN 讓資料庫自己用
	 * idx_forum_post_comments_post_id 與 idx_forum_reports_target 完成比對，
	 * 記憶體用量與貼文熱門程度無關。必須在刪留言之前執行。
	 */
	if _, err := tx.ExecContext(ctx,
		`DELETE fr FROM forum_reports fr
		 JOIN forum_post_comments c ON c.id = fr.target_id
		 WHERE fr.target_type = 'comment' AND c.post_id IN (`+placeholders+`)`,
		args...); err != nil {
		return nil, err
	}

	/*
	 * 把這次硬刪貼文的附圖排進待刪佇列。
	 *
	 * 佇列寫入與貼文刪除在同一個交易：Commit 成功就保證「檔名已被記錄」，
	 * 之後的刪檔失敗（或程式在 Commit 與刪檔之間崩潰）可以在下一個 tick
	 * 重試。沒有檔案伺服器時不排隊 —— 佇列唯一的消費者就是檔案刪除，
	 * 排進去只會累積沒有人處理的列（見檔頭 FileDeleter 的說明）。
	 */
	if p.files != nil {
		if err := enqueueDeletedImages(ctx, tx, posts, lockedIDs); err != nil {
			return nil, err
		}
	}

	for _, query := range []string{
		`DELETE FROM forum_post_comments WHERE post_id IN (` + placeholders + `)`,
		`DELETE FROM forum_post_likes WHERE post_id IN (` + placeholders + `)`,
		`DELETE FROM forum_post_bookmarks WHERE post_id IN (` + placeholders + `)`,
		`DELETE FROM forum_reports WHERE target_type = 'post' AND target_id IN (` + placeholders + `)`,
		`DELETE FROM forum_posts WHERE id IN (` + placeholders + `)`,
	} {
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return lockedIDs, nil
}

// lockExpiredPosts 在交易裡重新確認這批 id 仍然是「已軟刪除且超過保留期」，
// 並以 FOR UPDATE 鎖住它們，回傳仍符合條件的 id。
//
// 這是 deletePosts 的 TOCTOU 防線：鎖定之後，管理員的還原必須等本交易結束，
// 因此不可能發生「還原提交成功、貼文卻被硬刪」的交錯。
func lockExpiredPosts(ctx context.Context, tx *sql.Tx, ids []int64, cutoff time.Time) ([]int64, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders, args := idQueryArgs(ids)
	args = append(args, cutoff)
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM forum_posts
		 WHERE id IN (`+placeholders+`) AND deleted_at IS NOT NULL AND deleted_at < ?
		 ORDER BY id
		 FOR UPDATE`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locked := make([]int64, 0, len(ids))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		locked = append(locked, id)
	}
	return locked, rows.Err()
}

// idQueryArgs 把 id 清單組成 IN (?, ?, …) 的佔位符與參數。
func idQueryArgs(ids []int64) (string, []any) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	return placeholders, args
}

// deletedPostImages 從取回批次中挑出真正被硬刪的貼文附圖。
//
// 空檔名與重複檔名由 normalizedImageNames 收斂。
func deletedPostImages(posts []expiredPost, deletedIDs []int64) []string {
	if len(deletedIDs) == 0 {
		return nil
	}

	deleted := make(map[int64]struct{}, len(deletedIDs))
	for _, id := range deletedIDs {
		deleted[id] = struct{}{}
	}

	images := make([]string, 0, len(deletedIDs))
	for _, post := range posts {
		if len(post.images) == 0 {
			continue
		}
		if _, ok := deleted[post.id]; ok {
			images = append(images, post.images...)
		}
	}
	return images
}

// enqueueDeletedImages 把這次硬刪貼文的附圖檔名寫進待刪佇列。
//
// INSERT IGNORE 讓同一個檔名被多篇貼文引用（或上一次刪檔失敗仍留在佇列）
// 時只保留一列；created_at 只負責 FIFO 排序。
func enqueueDeletedImages(ctx context.Context, tx *sql.Tx, posts []expiredPost, deletedIDs []int64) error {
	names := normalizedImageNames(deletedPostImages(posts, deletedIDs))
	if len(names) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(names))
	args := make([]any, 0, len(names)*2)
	now := time.Now()
	for _, name := range names {
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, name, now)
	}
	_, err := tx.ExecContext(ctx,
		`INSERT IGNORE INTO forum_media_deletions (filename, created_at) VALUES `+strings.Join(placeholders, ","),
		args...)
	return err
}

// normalizedImageNames 把貼文附圖收斂成不重複的純檔名。
// parsePostImages 把 image_url 欄位（新制 JSON 陣列或舊制單一值）解析成檔名切片。
func parsePostImages(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(value), &arr); err == nil {
			out := make([]string, 0, len(arr))
			for _, v := range arr {
				if name := imageFileName(v); name != "" {
					out = append(out, name)
				}
			}
			return out
		}
	}
	if name := imageFileName(value); name != "" {
		return []string{name}
	}
	return nil
}
func normalizedImageNames(images []string) []string {
	seen := make(map[string]struct{}, len(images))
	names := make([]string, 0, len(images))
	for _, raw := range images {
		name := imageFileName(raw)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

// releasePostImages 回收「只被已清理的貼文引用」的附圖。
//
// 引用計數在貼文列已刪除**之後**才查，所以被清理的貼文自然不計入——
// 這正是「不再被任何地方引用」的定義。查計數失敗時保守地跳過（把
// 「查不到」當成「沒人用」是最糟糕的誤判方向，見 httpapi 的
// releaseForumImage 同一條規則），佇列列留著下一 tick 再試。
func (p *Pruner) releasePostImages(ctx context.Context, images []string) {
	if p.files == nil {
		return
	}
	p.releaseImages(ctx, normalizedImageNames(images))
}

// releaseQueuedImages 重試佇列中的圖片刪除（先進先出，最多 limit 筆）。
func (p *Pruner) releaseQueuedImages(ctx context.Context, limit int) {
	if p.files == nil || limit <= 0 {
		return
	}
	names, err := p.fetchQueuedImages(ctx, limit)
	if err != nil {
		logger.WarnfContext(ctx, "[PURGE] 讀取圖片刪除佇列失敗: %v", err)
		return
	}
	p.releaseImages(ctx, names)
}

// fetchQueuedImages 取出一批待刪檔名。
func (p *Pruner) fetchQueuedImages(ctx context.Context, limit int) ([]string, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT filename FROM forum_media_deletions ORDER BY created_at, filename LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := make([]string, 0, limit)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// releaseImages 先逐一套用引用計數，再並行刪除確認無人引用的檔案。
//
// 引用計數刻意留在單一 goroutine：它是兩個索引查詢，不是慢速的外部呼叫；
// 循序做讓「查不到引用」與「查到引用」的決策有穩定順序，也讓測試不必
// 對資料庫查詢的競態做文章。真正慢的是檔案伺服器的 DeleteFile，那一段
// 才需要並行與較短的單檔期限。
func (p *Pruner) releaseImages(ctx context.Context, names []string) {
	pending := make([]string, 0, len(names))
	for _, name := range names {
		references, err := p.imageReferenceCount(ctx, name)
		if err != nil {
			logger.WarnfContext(ctx, "[PURGE] 檢查圖片引用失敗，保留佇列 filename=%s: %v", name, err)
			continue
		}
		if references > 0 {
			// 仍被其他貼文或頭像引用：這張圖不該刪，佇列列也不再需要。
			// 最後一個引用消失時，那個引用者的刪除流程會重新排入。
			p.dequeueImage(ctx, name)
			continue
		}
		pending = append(pending, name)
	}

	p.forEachImage(ctx, pending, p.deleteImageFile)
}

// deleteImageFile 刪除檔案、移除 forum_media 紀錄，最後移除佇列列。
//
// 三步的順序是刻意的：先刪檔（外部系統），成功後才刪所有權紀錄；
// 兩者都成功才移除佇列列。任何一步失敗都保留佇列列，下一個 tick 重試。
// 檔案已不存在時 DeleteFile 回 404 視為成功（見 httpapi.deleteForumFile），
// 所以重試是安全的。
func (p *Pruner) deleteImageFile(ctx context.Context, name string) {
	fileCtx, cancel := context.WithTimeout(ctx, mediaDeleteTimeout)
	err := p.files.DeleteFile(fileCtx, name)
	cancel()
	if err != nil {
		logger.WarnfContext(ctx, "[PURGE] 回收貼文附圖失敗，保留佇列 filename=%s: %v", name, err)
		return
	}
	if _, err := p.db.ExecContext(ctx, `DELETE FROM forum_media WHERE filename = ?`, name); err != nil {
		logger.WarnfContext(ctx, "[PURGE] 刪除圖片紀錄失敗，保留佇列 filename=%s: %v", name, err)
		return
	}
	p.dequeueImage(ctx, name)
}

// dequeueImage 移除已處理完的佇列列。
//
// 失敗只記日誌：檔案已刪或已被引用，下一 tick 重試時 DeleteFile 的 404
// 仍會被視為成功，最終會收斂。
func (p *Pruner) dequeueImage(ctx context.Context, name string) {
	if _, err := p.db.ExecContext(ctx, `DELETE FROM forum_media_deletions WHERE filename = ?`, name); err != nil {
		logger.WarnfContext(ctx, "[PURGE] 移除圖片刪除佇列失敗 filename=%s: %v", name, err)
	}
}

// forEachImage 以固定並行度執行 fn，並在 ctx 取消時停止派工。
func (p *Pruner) forEachImage(ctx context.Context, names []string, fn func(context.Context, string)) {
	sem := make(chan struct{}, mediaDeleteConcurrency)
	var wg sync.WaitGroup
	for _, name := range names {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(ctx, name)
		}(name)
	}
	wg.Wait()
}

// imageReferenceCount 回傳有幾個地方在引用這個檔名（貼文附圖與頭像合計）。
//
// 與 httpapi.forumImageReferenceCount 相同的形狀：含 LIKE 退路（舊資料可能
// 存著 /files/x.png 或完整網址），且檔名先跳脫 LIKE 的萬用字元——檔名白名單
// 裡有「_」，不跳脫會讓 %/a_b.png 也匹配 /aXb.png，多算一個引用。
//
// 查詢拆成兩段的理由與 httpapi 那支相同：`OR ... LIKE` 會讓 MySQL 放棄索引
// 而做全表掃描，而這條查詢在清理排程裡是每個不重複檔名跑一次。先問等值
// （可用 idx_forum_posts_image_url / idx_forum_profiles_avatar_url），只有
// 等值沒查到時才補問 LIKE。語意相同，常見情況是一個索引查找。
func (p *Pruner) imageReferenceCount(ctx context.Context, name string) (int, error) {
	var exact int
	err := p.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM forum_posts WHERE image_url = ?) +
			(SELECT COUNT(*) FROM forum_profiles WHERE avatar_url = ?)`,
		name, name).Scan(&exact)
	if err != nil {
		return 0, err
	}
	if exact > 0 {
		return exact, nil
	}

	pattern := "%/" + escapeLike(name)
	var count int
	err = p.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM forum_posts WHERE image_url LIKE ?) +
			(SELECT COUNT(*) FROM forum_profiles WHERE avatar_url LIKE ?)`,
		pattern, pattern).Scan(&count)
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
