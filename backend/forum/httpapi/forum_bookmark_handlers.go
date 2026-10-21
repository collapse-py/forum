/*
收藏（backend/forum/httpapi/forum_bookmark_handlers.go）。

一、HTTP 路由對照

	GET  /api/forum/bookmarks              -> handleForumBookmarks     我收藏的貼文（需登入）
	POST /api/forum/posts/{id}/bookmark    -> handleForumPostBookmark  切換某一篇的收藏

二、為什麼切換掛在 /posts/{id}/ 之下而不是 /bookmarks

	與按讚完全相同的形狀（/api/forum/posts/{id}/like）：收藏的對象是一篇
	貼文，切換的語意是「這一篇」。清單則是「我的收藏」這個集合本身，放在
	/api/forum/bookmarks。兩者分開之後，路由的字面意義與資源的層級一致。

三、為什麼沒有「收藏數」

	按讚的數字是公開的（forumPostProjection 的 like_count），收藏刻意不是：
	收藏是私人的備忘，公開數量會讓「誰在關注什麼」變成可推測的訊號。因此
	回應只有 { bookmarked }，而投影裡的 bookmarked_by_me 也只反映請求者
	自己的狀態（與 liked_by_me 同一個形狀）。

四、清單的排序

	刻意沿用全站動態的排序（forumPostFeedOrder：置頂優先、再依發表時間）。
	不是「依收藏時間」：收藏頁渲染的是同一支 PostCard、由同一支 useFeed
	分頁，四種列表（動態、追蹤動態、個人頁、收藏）用同一個排序讓「置頂的
	貼文在每個列表都看得到」這條規則沒有例外。代價是收藏頁不反映「我上次
	收藏的是哪一篇」，而那件事在二十筆以內的清單裡並不重要。
*/
package httpapi

import (
	"net/http"
	"time"

	"forum/forum/logger"
)

// handleForumBookmarks 回傳「我收藏的貼文」分頁。
//
// 唯讀且純私有：每一筆都是自己按過收藏的貼文，查詢以 author_email = ?
// 收斂在「自己」。路由掛 requireLogin（理由與 following/posts 相同：沒有
// 登入就沒有這份清單），因此不掛限流。
func (s *Server) handleForumBookmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	email := s.sessions.ResolveUser(r)
	// 路由已用 requireLogin 包住，此處重複確認是縱深防禦。
	if email == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}

	// 與全站動態、追蹤動態同一個單頁筆數（三者的單頁成本相同）。
	const pageSize = 25

	offset, ok := forumOffsetParam(w, r)
	if !ok {
		return
	}

	// 條件走 forum_post_bookmarks 的 idx_forum_post_bookmarks_author
	// （author_email 為最左前綴），因此這是索引查詢而不是全表掃描。
	// 軟刪除過濾不在這裡：loadForumPosts 會帶上 forumPostVisible，收藏頁
	// 因此與其他列表看到同一批「還活著」的貼文 —— 一篇被作者刪除的貼文
	// 不該繼續出現在任何公開列表中，即使有人收藏過它。
	posts, err := s.loadForumPosts(r,
		`fp.id IN (
		     SELECT post_id FROM forum_post_bookmarks WHERE author_email = ?
		 )`, []interface{}{email}, pageSize, offset)
	if err != nil {
		s.writeForumPostLoadError(w, err, "unable to load bookmarked posts")
		return
	}

	writeOK(w, map[string]interface{}{
		"items":   posts,
		// 與其他列表同一套推測：滿頁即猜還有下一頁。
		"hasMore": len(posts) == pageSize,
	})
}

// handleForumPostBookmark 切換收藏狀態：已收藏則取消，未收藏則新增。
//
// 檢查順序刻意是「身分 → Origin → 路徑 → 資料庫」，與 handleForumPostLike
// 相同：Origin 驗證成本最低且擋掉跨站偽造，放在碰資料庫之前可避免無謂的
// 交易開銷。
func (s *Server) handleForumPostBookmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	// 縱深防禦：中介層 requireLoginForWrite 對 POST 已做過登入與停權檢查。
	author := s.sessions.ResolveUser(r)
	if author == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	postID, err := forumPostIDFromPath(r.URL.Path)
	if err != nil {
		badRequest(w, "invalid post id")
		return
	}

	// 交易邊界涵蓋「讀取目前狀態 → 寫入」，理由與 handleForumPostLike 相同：
	// 沒有交易時兩個並發請求可能都讀到未收藏而都送出 INSERT。
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 開始收藏交易失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to begin bookmark transaction")
		return
	}
	// 安全網：本函式所有提早 return 的分支都會走到這裡。Commit 成功之後
	// 再次呼叫 Rollback 會回 sql.ErrTxDone，屬預期且被忽略。
	defer tx.Rollback()

	// 交易內確認文章存在：schema 沒有外鍵，否則會出現指向不存在文章的收藏。
	// deleted_at IS NULL 與按讚同一個理由：對一篇已經刪掉的貼文收藏，只會在
	// 表裡留下一筆沒有人能再看到的紀錄。
	var postCount int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM forum_posts WHERE id = ? AND deleted_at IS NULL`, postID).Scan(&postCount); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 檢查文章失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to load forum post")
		return
	}
	if postCount == 0 {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}

	// 主鍵是 (post_id, author_email)，因此這筆查詢是唯一的（最多一列）。
	var existing int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM forum_post_bookmarks WHERE post_id = ? AND author_email = ?`,
		postID, author).Scan(&existing); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 檢查收藏狀態失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to check bookmark status")
		return
	}

	bookmarked := existing > 0
	if bookmarked {
		// DELETE 而非 UPDATE 設成某個停用旗標：收藏沒有「歷史」語意，
		// 讓資料列完全消失可以讓唯一的索引同時是複合主鍵（與按讚、追蹤
		// 同樣的取捨）。
		if _, err := tx.ExecContext(r.Context(),
			`DELETE FROM forum_post_bookmarks WHERE post_id = ? AND author_email = ?`,
			postID, author); err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 取消收藏失敗 post_id=%d: %v", postID, err)
			internalError(w, "unable to remove bookmark")
			return
		}
	} else if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO forum_post_bookmarks (post_id, author_email, created_at) VALUES (?, ?, ?)`,
		postID, author, time.Now()); err != nil {
		// 並發下兩個請求都可能通過上面的檢查，複合主鍵會讓其中一個收到
		// 1062。刻意不為它特別分流（理由與按讚相同）：收藏只是一個開關，
		// 使用者再按一次即可。
		logger.ErrorfContext(r.Context(), "[FORUM] 新增收藏失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to add bookmark")
		return
	}

	if err := tx.Commit(); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 儲存收藏失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to save bookmark")
		return
	}

	// 回傳切換後的狀態：連點兩下的合併結果只有後端知道，前端不該自己猜
	// （與按讚、追蹤同一條契約）。
	writeOK(w, map[string]bool{"bookmarked": !bookmarked})
}
