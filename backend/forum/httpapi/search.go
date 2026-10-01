/*
search.go：貼文搜尋（公開頁與後臺）與 Elasticsearch 索引的維護。

本檔是 ES 功能的唯一入口，共三段職責：

  一、搜尋端點
      GET /api/forum/search          公開頁搜尋（去識別化，回 forumPost）
      GET /api/admin/forum/search    後臺搜尋（回 adminForumPost，含作者 email）
      兩者共用同一組參數解析與同一條降級路徑，差別只在輸出形狀與權限檢查。

  二、搜尋的資料來源
      ES 只負責「找出符合條件的貼文 ID」，貼文本體一律回 MySQL 讀取
      （理由見 module/es 的檔頭說明）。因此搜尋結果與貼文列表是同一種結構，
      前端可以用同一張卡片元件渲染，計數、附圖 token 與去識別化規則也
      必然一致，不會出現「列表有作者名、搜尋結果沒有」這類分歧。

  三、索引的維護
      建立／刪除／修改貼文時同步更新 ES；啟動時可全量重建。
      所有寫入 ES 的呼叫都是 best-effort：失敗只寫日誌，絕不影響 MySQL
      已完成的寫入（理由見 indexForumPost）。

【降級：ES 壞掉不等於不能搜尋】

      ES 未設定（ES_URL 留空）→ 一開始就用 MySQL LIKE。
      ES 已設定但連不上／查詢失敗 → 記警告後當次改用 MySQL LIKE。

      回應中的 engine 欄位會告訴前端這次結果是誰給的，前端據此顯示
      「以資料庫關鍵字比對」之類的說明。使用者因此知道搜尋結果可能不如
      全文檢索完整，而不是誤以為「站上只有這幾篇」。

      降級只在「發生錯誤」時發生：ES 正常回傳 0 筆時不會退回 MySQL，
      否則 ES 與 MySQL 的比對規則不同（分詞 vs LIKE）會讓兩邊的結果
      混在同一次回應裡，前端無從判斷該怎麼解讀。
*/

package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/es"
	"forum/forum/logger"
)

// 搜尋的參數上限。
//
//	queryMaxRunes 對齊 MySQL 與 ES 兩側的實際需求：關鍵字是給人打的，
//	100 字已遠超一般使用者的輸入；把它限制住也順手擋掉「把整篇文章貼進
//	搜尋框」這種會讓 ES 分詞成本暴增的請求。
//	pageSizeMax 與貼文列表的 pageSize 相同，讓兩種瀏覽方式一頁的資訊量一致。
const (
	queryMaxRunes = 100
	pageSizeMax   = 25
)

// engine 欄位的兩個可能值。它們是 API 契約的一部分（前端會依此顯示說明），
// 因此以常數而非字面值散落在程式碼中。
const (
	engineElasticsearch = "elasticsearch"
	engineMySQL         = "mysql"
)

// rebuildBatchSize 是全量重建時每次從 MySQL 取多少篇貼文。
//
// 分頁讀取（而非一次把所有列讀進記憶體）有兩個理由：貼文表會成長，
// 一次全讀會讓記憶體用量與資料量成正比；而且每批讀完就關閉 rows
// 再送出 ES 請求，不會出現「持有 MySQL 連線時又對外發 HTTP」的狀況
// （連線池被單一 goroutine 佔滿時，其他請求會全部卡住）。
const rebuildBatchSize = 500

/* ==========================================================================
   搜尋端點
   ========================================================================== */

// searchQuery 是解析並驗證過的搜尋參數。
//
// Offset 已保證 >= 0，Limit 已保證落在 1..pageSizeMax 之間，Query 已去除
// 前后空白且長度合法。handler 因此不需要重複做防禦。
type searchQuery struct {
	Query  string
	Offset int
	Limit  int
}

// parseSearchQuery 解析 q / offset / limit 三個查詢參數。
//
// q 空白或超長一律回 400：空白代表這是「沒輸入就送出表單」的無意義請求，
// 與其回一份全部貼文（昂貴且毫無用處），不如明確拒絕。
// offset / limit 的非法值則夾到合法範圍，理由是它們屬於「翻頁」性質的參數，
// 打錯字時退回第一頁比整個請求失敗體驗好（與 listAdminForumPosts 的寬容一致）。
func parseSearchQuery(r *http.Request) (searchQuery, bool) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		return searchQuery{}, false
	}
	if len([]rune(query)) > queryMaxRunes {
		return searchQuery{}, false
	}
	parsed := searchQuery{Query: query, Limit: pageSizeMax}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			parsed.Offset = value
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 && value <= pageSizeMax {
			parsed.Limit = value
		}
	}
	return parsed, true
}

// handleForumSearch 是公開頁的搜尋端點（GET /api/forum/search）。
//
// 唯讀，且刻意不掛限流：與貼文列表同一個道理 —— 匿名訪客也要能搜尋，
// 限流它只會讓功能在正常流量下壞掉。真正需要防的是寫入端點。
//
// 回應：{"items": [forumPost], "total": N, "query": "...", "engine": "..."}
// 形狀刻意與 GET /api/forum/posts 相同（多兩個查詢相關欄位），
// 前端因此能用同一個 render 路徑處理搜尋結果。
func (s *Server) handleForumSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	query, ok := parseSearchQuery(r)
	if !ok {
		badRequest(w, "請輸入 1 至 100 字的搜尋關鍵字")
		return
	}
	// 目前瀏覽者只影響每篇貼文的 liked 旗標；未登入時為空字串，
	// SQL 裡的 EXISTS 子查詢自然恆為 false。
	ids, total, engine, err := s.searchForumPostIDs(r.Context(), query.Query, query.Offset, query.Limit, false)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[SEARCH] 搜尋貼文失敗: %v", err)
		internalError(w, "unable to search forum posts")
		return
	}
	// 命中 0 筆時不回 MySQL：這一趟連 per-row 查詢都不該發生。
	items := make([]forumPost, 0, len(ids))
	if len(ids) > 0 {
		items, err = s.loadForumPostsByIDs(r, ids)
		if err != nil {
			logger.ErrorfContext(r.Context(), "[SEARCH] 讀取搜尋結果失敗: %v", err)
			internalError(w, "unable to load forum posts")
			return
		}
	}
	writeOK(w, map[string]interface{}{
		"items":  items,
		"total":  total,
		"query":  query.Query,
		"engine": engine,
	})
}

// handleAdminForumSearch 是後臺的搜尋端點（GET /api/admin/forum/search）。
//
// 與公開版的差異只有兩處，刻意不抽象成共用的 handler：
//  1. 權限：走 requireAdminForum，結果含作者真實 email（供治理判斷）。
//  2. 查詢條件多一條 author_email 的完全比對，讓管理員可以直接以
//     「某個信箱」找出該人的所有貼文（ES 用 term、MySQL 用 =，兩側一致）。
//
// 後臺刻意不掛限流（理由見 server.go 的路由總表）。
func (s *Server) handleAdminForumSearch(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	query, ok := parseSearchQuery(r)
	if !ok {
		badRequest(w, "請輸入 1 至 100 字的搜尋關鍵字")
		return
	}
	ids, total, engine, err := s.searchForumPostIDs(r.Context(), query.Query, query.Offset, query.Limit, true)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[SEARCH] 後臺搜尋貼文失敗: %v", err)
		internalError(w, "unable to search forum posts")
		return
	}
	items := make([]adminForumPost, 0, len(ids))
	if len(ids) > 0 {
		items, err = s.loadAdminForumPostsByIDs(r.Context(), ids)
		if err != nil {
			logger.ErrorfContext(r.Context(), "[SEARCH] 讀取後臺搜尋結果失敗: %v", err)
			internalError(w, "unable to load forum posts")
			return
		}
	}
	writeOK(w, map[string]interface{}{
		"items":  items,
		"total":  total,
		"query":  query.Query,
		"engine": engine,
	})
}

/* ==========================================================================
   搜尋的資料來源
   ========================================================================== */

// searchForumPostIDs 取得符合條件的貼文 ID。
//
// 回傳的 engine 說明這次結果的來源，呼叫端必須把它放進回應，
// 否則前端無法區分「ES 真的只有這些」與「ES 壞掉所以湊合用 LIKE」。
//
// 降級只在錯誤時發生，理由見檔頭說明。
func (s *Server) searchForumPostIDs(ctx context.Context, query string, offset, limit int, includeAuthor bool) ([]int64, int, string, error) {
	if s.esEnabled() {
		result, err := s.es.Search(ctx, query, offset, limit, includeAuthor)
		if err == nil {
			ids := make([]int64, 0, len(result.Hits))
			for _, hit := range result.Hits {
				ids = append(ids, hit.ID)
			}
			return ids, result.Total, engineElasticsearch, nil
		}
		// ErrDisabled 只會在「設定了位址卻又變成未啟用」時出現，屬於程式錯誤；
		// 因此照樣記錄，不假裝它是預期情況。
		if errors.Is(err, es.ErrDisabled) {
			logger.ErrorfContext(ctx, "[SEARCH] ES 未啟用卻收到查詢: %v", err)
		} else {
			logger.WarnfContext(ctx, "[SEARCH] Elasticsearch 查詢失敗，改用 MySQL 比對: %v", err)
		}
	}
	ids, total, err := s.searchForumPostIDsMySQL(ctx, query, offset, limit, includeAuthor)
	if err != nil {
		return nil, 0, engineMySQL, err
	}
	return ids, total, engineMySQL, nil
}

// searchForumPostIDsMySQL 是 ES 不可用時的降級搜尋：以 content LIKE ? 找出
// 符合的貼文，依建立時間新到舊排序。
//
// 與 ES 的差異是必然的，介面上也刻意不假裝兩者等價：
//   - 沒有分詞。LIKE 是逐字比對，搜「檢舉」不會命中「言論檢舉活動」以外
//     任何拆開的結果，也不會有相關性排序（只依時間）。
//   - 命中關鍵字的位置不影響排序，因此結果的相關性明顯較差。
//
// LIKE 的樣式必須自行跳脫 % 與 _，否則使用者搜尋「100%」會變成
// 「開頭是 100 的任何字串」，而 _ 會單獨撈出大量無關結果。
func (s *Server) searchForumPostIDsMySQL(ctx context.Context, query string, offset, limit int, includeAuthor bool) ([]int64, int, error) {
	pattern := "%" + escapeLikePattern(query) + "%"
	// 條件字串以 if/else 白名單組出，值一律走佔位符（與 forum_admin_handlers.go
	// 的動態 SQL 準則相同：白名單決定結構、佔位符綁定資料）。
	condition := `content LIKE ?`
	args := []interface{}{pattern}
	if includeAuthor {
		// author_email 用完全比對而非 LIKE，與 ES 的 term 語意對齊：
		// 「找出某個信箱的所有貼文」是治理需求，「找出信箱裡含某段字的人」
		// 不是（那會讓任何人都能用片段信箱反推他人）。
		condition = `(content LIKE ? OR author_email = ?)`
		args = append(args, query)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM forum_posts WHERE `+condition, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM forum_posts WHERE `+condition+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`,
		append(append([]interface{}{}, args...), limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	return ids, total, rows.Err()
}

// escapeLikePattern 跳脫 MySQL LIKE 樣式中的兩個萬用字元與跳脫字元本身。
//
// 順序不能改：backslash 必須先處理，否則「\%」會在第二步被再跳脫一次，
// 變成「\\%」而使跳脫失效。strings.NewReplacer 是單趟掃描，
// 不會重複套用同一組規則，因此依宣告順序即可。
func escapeLikePattern(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

// loadForumPostsByIDs 依指定的 ID 順序組出公開頁的貼文物件。
//
// 排序以呼叫端給的 ids 為準：SQL 的 IN 查詢沒有順序保證，而搜尋結果的
// 順序來自 ES 的相關性分數，必須原樣保留。做法是查完後以 ids 建立索引表
// 再回填，沒有為此多打一次帶 CASE WHEN 的 SQL。
//
// 這個函式刻意獨立於 listForumPosts 的分頁流程：兩者的資料來源不同
// （一個是分頁掃描、一個是 ID 集合），強行共用只會讓主列表多一層間接。
// 但「回傳哪些欄位」不能各寫一份，因此 SELECT 的投影用的是 forumPostProjection
// 這個共用常數 —— 那才是「三種來源必須回同一種欄位」這條不變條件所在的位置。
// 真正需要共用的部分（去識別化、標籤、圖片 token）本來就已經是共用函式，
// 結果形狀因此必然一致。
func (s *Server) loadForumPostsByIDs(r *http.Request, ids []int64) ([]forumPost, error) {
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids)+1)
	// 使用者的 email 放在第一位，對應 SELECT 裡第一個問號（liked_by_me 的
	// EXISTS 子查詢）。未登入時是空字串，該子查詢恆為 false。
	args = append(args, s.sessions.ResolveUser(r))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT `+forumPostProjection+`
		FROM forum_posts fp
		WHERE fp.id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	/*
	 * 依 ES 給的順序組出結果，並以 index 記住每個 ID 的位置。
	 *
	 * 這裡刻意不在掃描時另外維一份 map 指標、在之後才回填作者資訊：掃描階段
	 * append 進切片的是「值的一份複製」，之後對切片元素的修改不會反映到當初
	 * 指向的變數，於是若最終結果是從舊的 map 取值組出，就會把未覆寫的
	 * author_email 原樣送到瀏覽器（公開 API 洩漏 email 的事故）。因此改成：
	 * 先照順序填好切片，再就地補作者，最後丟掉查不到的。
	 */
	ordered := make([]forumPost, 0, len(ids))
	indexOf := make(map[int64]int, len(ids))
	// 圖片 token 延遲建立、整頁共用一把，理由與 listForumPosts 相同。
	mediaToken := ""
	for rows.Next() {
		var post forumPost
		// pinned 與 liked 一样以 0/1 讀入（MySQL 沒有原生布林），理由見
		// forum_handlers.go 的 loadForumPosts。
		var liked, pinned int
		if err := rows.Scan(&post.ID, &post.Author, &post.Content, &post.CreatedAt, &post.ImageURL, &pinned, &post.LikeCount, &post.CommentCount, &liked); err != nil {
			return nil, err
		}
		post.Liked = liked == 1
		// 搜尋結果也帶 pinned：使用者從搜尋找到一篇置頂文章時，應看到與首頁
		// 相同的徽章。少了它，同一篇文章在兩個地方會長得不一樣。
		post.Pinned = pinned == 1
		imageName := s.forumImageFileName(post.ImageURL)
		if imageName != "" && mediaToken == "" {
			token, err := s.createMediaToken(r.Context())
			if err != nil {
				return nil, err
			}
			mediaToken = token
		}
		post.ImageURL = s.forumImageURL(imageName, mediaToken)
		indexOf[post.ID] = len(ordered)
		ordered = append(ordered, post)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 補作者顯示名、雜湊 key 與標籤。這三項都需要 per-row 查詢
	// （最壞情況 25 篇 = 75 次往返），與 listForumPosts 承擔的 N+1 相同；
	// 單頁筆數有 pageSizeMax 封頂，因此代價可控。
	for index := range ordered {
		authorEmail := ordered[index].Author
		ordered[index].Author = s.forumAuthor(r, authorEmail)
		ordered[index].AuthorKey = publicForumKey(authorEmail)
		tags, err := s.forumAuthorTags(r.Context(), authorEmail)
		if err != nil {
			return nil, err
		}
		ordered[index].AuthorTags = tags
	}
	// 依 ES 的相關性順序重排，並丟掉查不到的那幾筆（貼文在搜尋之後、組裝回應
	// 之前被刪除的情況，或索引裡有 MySQL 已不存在的殘留文件）。
	// 寧可少一筆也不要拿錯順序的內容。
	result := make([]forumPost, 0, len(ordered))
	for _, id := range ids {
		if index, ok := indexOf[id]; ok {
			result = append(result, ordered[index])
		}
	}
	return result, nil
}

// loadAdminForumPostsByIDs 依指定的 ID 順序組出後臺的貼文物件（含真實 email
// 與完整留言），語意與 loadForumPostsByIDs 對稱。
func (s *Server) loadAdminForumPostsByIDs(ctx context.Context, ids []int64) ([]adminForumPost, error) {
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, author_email, content, created_at, image_url,
		       (SELECT COUNT(*) FROM forum_post_likes WHERE post_id = forum_posts.id),
		       (SELECT COUNT(*) FROM forum_post_comments WHERE post_id = forum_posts.id)
		FROM forum_posts WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 與 loadForumPostsByIDs 同樣的理由：結果必須照呼叫端給的順序組出，而且
	// 補上圖片網址與留言之後要拿到的是「補完的那一份」。因此先照順序填切片、
	// 用 indexOf 記位置，再就地補資料，最後才重排。
	ordered := make([]adminForumPost, 0, len(ids))
	indexOf := make(map[int64]int, len(ids))
	for rows.Next() {
		var item adminForumPost
		if err := rows.Scan(&item.ID, &item.AuthorEmail, &item.Content, &item.CreatedAt, &item.ImageURL, &item.LikeCount, &item.CommentCount); err != nil {
			return nil, err
		}
		indexOf[item.ID] = len(ordered)
		ordered = append(ordered, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 附圖需要短期 token 才能讀取；整頁共用一把，理由見 listAdminForumPosts。
	mediaToken := ""
	for index := range ordered {
		imageName := s.forumImageFileName(ordered[index].ImageURL)
		if imageName != "" && mediaToken == "" {
			token, err := s.createMediaToken(ctx)
			if err != nil {
				return nil, err
			}
			mediaToken = token
		}
		ordered[index].ImageURL = s.forumImageURL(imageName, mediaToken)
		// 留言一併帶出，讓後臺在搜尋結果裡仍能就地編輯／刪除（與文章列表一致）。
		comments, err := s.loadAdminForumComments(ctx, ordered[index].ID, ordered[index].CommentCount)
		if err != nil {
			return nil, err
		}
		ordered[index].Comments = comments
	}
	result := make([]adminForumPost, 0, len(ordered))
	for _, id := range ids {
		if index, ok := indexOf[id]; ok {
			result = append(result, ordered[index])
		}
	}
	return result, nil
}

/* ==========================================================================
   索引維護
   ========================================================================== */

// esEnabled 報告搜尋是否走 Elasticsearch。
//
// 單一判斷點：所有 ES 呼叫都經過它，因此「ES 未設定」在整個程式裡只有一種
// 表現方式，測試也只要不填 Server.es 就能得到「純 MySQL」的行為。
func (s *Server) esEnabled() bool {
	return s.es != nil && s.es.Enabled()
}

// indexForumPost 在貼文建立或修改後更新 ES 索引。
//
// Best-effort 是刻意的：MySQL 已經寫好了，這裡失敗只代表「搜尋結果會晚一步
// 才有這篇」，把它升級成請求失敗只會讓一個輔助功能反過來擋住使用者發文。
// 補救手段是啟動時的全量重建（RebuildSearchIndex）。
//
// 呼叫端不需要處理錯誤，也不需要知道 ES 是否啟用。
func (s *Server) indexForumPost(ctx context.Context, id int64, content, authorEmail string, createdAt time.Time) {
	if !s.esEnabled() {
		return
	}
	doc := es.PostDocument{ID: id, Content: content, AuthorEmail: authorEmail, CreatedAt: createdAt}
	if err := s.es.IndexPost(ctx, doc); err != nil {
		logger.WarnfContext(ctx, "[SEARCH] 索引貼文失敗 post_id=%d: %v", id, err)
	}
}

// reindexForumPostByID 依 id 回讀貼文並重建其索引文件。
//
// 存在的原因是「更新」路徑不知道全部欄位：管理介面只能改 content，但索引
// 文件還有 authorEmail 與 createdAt（後臺以作者精確比對、同分時以時間排序）。
// 只把改動的欄位送進索引會把未帶到的欄位清成零值，因此一律回讀整列。
//
// 貼文已不存在（sql.ErrNoRows）或 ES 未啟用時靜默返回：刪除路徑已經會呼叫
// unindexForumPost，這裡再補一次刪除語意是合理的冪等行為。
func (s *Server) reindexForumPostByID(ctx context.Context, id int64) {
	if !s.esEnabled() {
		return
	}
	var doc es.PostDocument
	var authorEmail string
	// err 逐項區分是為了讓「查不到」與「查詢失敗」走不同處置：
	// 前者是正常的競態（貼文剛好被另一個請求刪掉），後者要留下日誌。
	err := s.db.QueryRowContext(ctx,
		`SELECT id, author_email, content, created_at FROM forum_posts WHERE id = ?`, id).
		Scan(&doc.ID, &authorEmail, &doc.Content, &doc.CreatedAt)
	if err == sql.ErrNoRows {
		s.unindexForumPost(ctx, id)
		return
	}
	if err != nil {
		logger.WarnfContext(ctx, "[SEARCH] 讀取貼文以更新索引失敗 post_id=%d: %v", id, err)
		return
	}
	doc.AuthorEmail = authorEmail
	if err := s.es.IndexPost(ctx, doc); err != nil {
		logger.WarnfContext(ctx, "[SEARCH] 更新貼文索引失敗 post_id=%d: %v", id, err)
	}
}

// unindexForumPost 在貼文被刪除後移除其 ES 文件。
//
// 這一點對「通過（刪文）」的檢舉流程特別重要：該功能會留下檢舉紀錄，
// 而刪文後如果索引沒清掉，搜尋結果就會出現一個點進去是 404 的幽靈貼文。
func (s *Server) unindexForumPost(ctx context.Context, id int64) {
	if !s.esEnabled() {
		return
	}
	if err := s.es.DeletePost(ctx, id); err != nil {
		logger.WarnfContext(ctx, "[SEARCH] 移除貼文索引失敗 post_id=%d: %v", id, err)
	}
}

// RebuildSearchIndex 從 MySQL 全量重建貼文索引，回傳送出的文件筆數。
//
// 呼叫時機：啟動後於背景 goroutine 中執行一次（見 main.go）。
// 為什麼不是每次請求都做：重建是 O(全部貼文) 的操作，與請求頻率無關。
// 為什麼不需要排程器：貼文數量在這個站點的規模下，重建一次的成本遠低於
// 讓索引長期與 MySQL 不同步的代價；真到了需要增量同步的量級，
// 這裡才值得換成「啟動時只補差異（依 id 與 updated_at 比對）」。
//
// 冪等：以 MySQL 主鍵當 ES 文件 ID，重複執行只是覆寫。
func (s *Server) RebuildSearchIndex(ctx context.Context) (int, error) {
	if !s.esEnabled() {
		return 0, es.ErrDisabled
	}
	if err := s.es.EnsureIndex(ctx); err != nil {
		return 0, err
	}
	// 以 id 為分頁依據（而不是 created_at）：id 是主鍵，單調遞增且唯一，
	// 分頁結果不會因為有同秒建立的貼文而漏筆或重複。
	sent := 0
	var lastID int64
	for {
		rows, err := s.db.QueryContext(ctx, `
			SELECT id, author_email, content, created_at
			FROM forum_posts WHERE id > ? ORDER BY id ASC LIMIT ?`, lastID, rebuildBatchSize)
		if err != nil {
			return sent, err
		}
		batch := make([]es.PostDocument, 0, rebuildBatchSize)
		for rows.Next() {
			var doc es.PostDocument
			if err := rows.Scan(&doc.ID, &doc.AuthorEmail, &doc.Content, &doc.CreatedAt); err != nil {
				rows.Close()
				return sent, err
			}
			batch = append(batch, doc)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return sent, err
		}
		// 必須在送出 ES 請求前關閉 rows：批次之間若有 HTTP 往返，
		// 持有連線等對外回應會佔住連線池（見 rebuildBatchSize 的說明）。
		rows.Close()
		if len(batch) == 0 {
			return sent, nil
		}
		if err := s.es.IndexPosts(ctx, batch); err != nil {
			return sent, err
		}
		sent += len(batch)
		lastID = batch[len(batch)-1].ID
		if len(batch) < rebuildBatchSize {
			return sent, nil
		}
	}
}
