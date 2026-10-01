package httpapi

/*
站內公告與文章置頂（httpapi/announcement_handlers.go）。

四支端點：

	GET   /api/forum/announcement              公開：取得目前生效的公告（沒有則回 null）
	GET   /api/admin/announcements             後臺：列出公告（含已停用與已過期）
	POST  /api/admin/announcements             後臺：發佈新公告（自動停用舊的）
	PATCH /api/admin/announcements/{id}        後臺：改內容／改到期／重新啟用或停用
	POST  /api/admin/forum/posts/{id}/pin      後臺：置頂或取消置頂

「同時只有一則公告」是這個檔案最重要的約束

	公告在公開頁上是一條橫幅，因此兩條同時生效會互相衝突 —— 使用者會看到
	一個被橫幅佔掉的上半頁，其中還可能是「活動改期」與「活動照常」這種互相
	打架的內容。表格保留多列是為了留下歷史（什麼時候、經誰發、之後被誰關），
	而「只有一列 active」則由兩件事保證：

	  - 發佈新公告時在同一個交易裡把舊的設為不啟用（publishAnnouncement）
	  - 改為啟用時同樣先把其他全部關掉（updateAnnouncement）

	這兩處刻意都放在**交易內**。若先插新的再關舊的，中間任何一瞬間讀到資料的
	請求都會看到兩則，而那個瞬間足以讓使用者截到一張有矛盾的畫面。

公開端點為什麼回 null 而不是 404

	前端每個頁面載入都會問一次「有沒有公告」。若沒有公告是「成功但沒有東西」，
	回 404 會讓前端必須區分「沒有公告」（正常）與「端點壞了」（異常）兩種
	404 —— 而前者是壓倒性的常見情況。因此回 200 + announcement: null。
*/

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/audit"
	"forum/forum/logger"
)

// maxAnnouncementLength 是公告內文的最大字元數。
//
// 300 的取捨：公告是一條橫幅，而這個專案沒有富文字（只有純文字與換行），
// 300 字在中文大約是兩到三行 —— 超過那個長度它就不再是「公告」而是一篇文章，
// 而文章應該是貼文。限制在建構時就擋（400）而不是在渲染時截斷，理由是
// 截斷會讓管理員以為整則都顯示出來了。
const maxAnnouncementLength = 300

// minAnnouncementLength 是公告內文的最小字元數（去除前後空白後）。
//
// 只有空白會產生一條「看得到邊框、裡面是空的」橫幅 —— 那比沒有公告更糟，
// 因為它佔了版面卻沒有資訊。
const minAnnouncementLength = 1

// forumAnnouncement 是一則公告對外的樣子。
type forumAnnouncement struct {
	Body string `json:"body"`
	// PublishedAt 是 created_at（首次發佈時間），前端用來決定是否顯示
	// 「發佈於」。updated_at 刻意不對外：使用者關心的是「這則公告是什麼時候
	// 開始的」，而「被誰改過幾次」是管理員的資訊。
	PublishedAt string `json:"publishedAt"`
	// ExpiresAt 為空字串代表永不自動過期。
	ExpiresAt string `json:"expiresAt"`
}

// adminAnnouncement 是後臺視圖，多了 id、active 與建立者。
type adminAnnouncement struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	Active    bool   `json:"active"`
	CreatedBy string `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
	UpdatedBy string `json:"updatedBy,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	// Effective 是「實際上會不會顯示」：active 為真且（未設到期或尚未到期）。
	// 刻意與 active 分開 —— 一則 active 但已過期的公告在列表裡看起來是
	// 開著的，而它實際上什麼都不顯示；不分開的話管理員會以為橫幅還在。
	Effective bool `json:"effective"`
}

// handleForumAnnouncement 是公開端點：回傳目前生效的公告。
//
// GET only，且**不掛限流**：它是每個頁面載入都會打一次的低成本唯讀查詢
// （一列的索引掃描），而限流它的唯一效果是讓「公告機制壞掉時連診斷都
// 做不了」。
func (s *Server) handleForumAnnouncement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// 只取「active 且（未設到期 或 到期時間還沒到）」。expires_at 為 NULL 的
	// 比較結果在 SQL 三值邏輯下是 NULL（不是 true），因此必須用
	// IS NULL 明確表達，否則永不过期的公告永遠查不到。
	row := s.db.QueryRowContext(r.Context(), `
		SELECT body, created_at, COALESCE(DATE_FORMAT(expires_at, '%Y-%m-%dT%H:%i:%sZ'), '')
		FROM forum_announcements
		WHERE active = 1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC, id DESC
		LIMIT 1`)

	var announcement forumAnnouncement
	if err := row.Scan(&announcement.Body, &announcement.PublishedAt, &announcement.ExpiresAt); err != nil {
		if err == sql.ErrNoRows {
			// 沒有公告是常見且正常的情況，因此回 200 + null 而不是 404。
			// 理由見檔頭。
			writeOK(w, map[string]any{"ok": true, "announcement": nil})
			return
		}
		logger.ErrorfContext(r.Context(), "[ANNOUNCE] 讀取失敗: %v", err)
		internalError(w, "unable to load announcement")
		return
	}
	writeOK(w, map[string]any{"ok": true, "announcement": announcement})
}

// handleAdminAnnouncements 列出全部公告（GET）或發佈新公告（POST）。
func (s *Server) handleAdminAnnouncements(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.listAdminAnnouncements(w, r)
	case http.MethodPost:
		if !s.isTrustedOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		s.publishAnnouncement(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) listAdminAnnouncements(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, body, active, created_by, created_at, updated_by, updated_at,
		       COALESCE(DATE_FORMAT(expires_at, '%Y-%m-%dT%H:%i:%sZ'), '')
		FROM forum_announcements
		ORDER BY created_at DESC, id DESC
		LIMIT 100`)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[ANNOUNCE] 列出失敗: %v", err)
		internalError(w, "unable to list announcements")
		return
	}
	defer rows.Close()

	items := make([]adminAnnouncement, 0, 8)
	for rows.Next() {
		item, err := scanAdminAnnouncement(rows)
		if err != nil {
			internalError(w, "unable to list announcements")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		internalError(w, "unable to list announcements")
		return
	}
	writeOK(w, map[string]any{"ok": true, "items": items})
}

// announcementRowScanner 是讀一列所需的最小介面，*sql.Rows 滿足它。
//
// 宣告成介面只是為了讓 scanAdminAnnouncement 不必依賴 *sql.Rows 的其餘
// 方法（那些在這裡一個都用不到），這與 audit 套件裡 scanner 的理由相同。
type announcementRowScanner interface {
	Scan(dest ...any) error
}

// scanAdminAnnouncement 依 SELECT 的欄位順序讀出一列。
func scanAdminAnnouncement(row announcementRowScanner) (adminAnnouncement, error) {
	var item adminAnnouncement
	var active int
	if err := row.Scan(&item.ID, &item.Body, &active, &item.CreatedBy, &item.CreatedAt,
		&item.UpdatedBy, &item.UpdatedAt, &item.ExpiresAt); err != nil {
		return adminAnnouncement{}, err
	}
	item.Active = active == 1
	item.Effective = item.Active && (item.ExpiresAt == "" || item.ExpiresAt > time.Now().UTC().Format(time.RFC3339))
	return item, nil
}

// announcementRequest 是發佈與修改共用的請求主體。
type announcementRequest struct {
	Body string `json:"body"`
	// Active 為 false 代表「發佈但不顯示」。存在的理由是一則要寫好、稍後才
	// 生效的公告很常見，而為了這個用途去建一列再刪掉並沒有比較好。
	Active bool `json:"active"`
	// HoursUntilExpiry <= 0 代表永不自動過期。
	HoursUntilExpiry int `json:"hoursUntilExpiry"`
}

// normalizeAnnouncement 修剪並驗證一則公告，回傳整理後的值與到期時間。
//
// expires 用「幾小時後」而不是絕對時間戳：管理員在後臺填的是「這個公告
// 三天後失效」而不是精確到分鐘的時���，而絕對時間戳在跨時區的後臺與
// 絕大多數瀏覽者之間是一個容易出錯的單位轉換。
func normalizeAnnouncement(body string, hours int) (string, sql.NullTime, error) {
	trimmed := strings.TrimSpace(body)
	length := len([]rune(trimmed))
	if length < minAnnouncementLength {
		return "", sql.NullTime{}, &requestError{message: "公告內容不可為空"}
	}
	if length > maxAnnouncementLength {
		return "", sql.NullTime{}, &requestError{
			message: "公告最多 " + strconv.Itoa(maxAnnouncementLength) + " 字（目前 " + strconv.Itoa(length) + " 字）",
		}
	}
	if hours < 0 {
		return "", sql.NullTime{}, &requestError{message: "有效時間不可為負數"}
	}
	if hours == 0 {
		return trimmed, sql.NullTime{}, nil
	}
	// 上限 365 天與 ipban 的封鎖上限同理由：到期時間是一個欄位，
	// 而「一年後」是使用者真的能預期的長度。
	if hours > 365*24 {
		return "", sql.NullTime{}, &requestError{message: "有效時間最多 365 天"}
	}
	expires := time.Now().Add(time.Duration(hours) * time.Hour)
	return trimmed, sql.NullTime{Time: expires, Valid: true}, nil
}

// publishAnnouncement 發佈一則新公告，並在同一個交易裡停用舊的。
func (s *Server) publishAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req announcementRequest
	if err := decodeLimitedJSON(w, r, &req, 4<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}
	body, expires, err := normalizeAnnouncement(req.Body, req.HoursUntilExpiry)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	actor := s.sessions.ResolveUser(r)
	now := time.Now()
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to publish announcement")
		return
	}
	defer tx.Rollback()

	// 先停用舊的再插新的。順序在這個交易裡其實無關紧要（外鍵與唯一鍵都沒有
	// 參與），但「先關再開」讓「如果這一步之後失敗」的狀態是「沒有公告」
	// 而不是「兩則都開著」—— 前者只是橫幅消失，後者會顯示互相衝突的內容。
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE forum_announcements SET active = 0, updated_at = ?, updated_by = ? WHERE active = 1`,
		now, actor); err != nil {
		internalError(w, "unable to publish announcement")
		return
	}

	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO forum_announcements (body, active, created_at, created_by, updated_at, updated_by, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		body, boolToInt(req.Active), now, actor, now, actor, expires)
	if err != nil {
		internalError(w, "unable to publish announcement")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to publish announcement")
		return
	}

	if err := s.recordAdminAction(r, tx, "announcement.publish", audit.TargetAnnouncement,
		strconv.FormatInt(id, 10), body,
		audit.Change{Field: "body", Before: "", After: body},
		audit.Change{Field: "active", Before: "", After: strconv.FormatBool(req.Active)},
		audit.Change{Field: "expiresAt", Before: "", After: formatExpiry(expires)},
	); err != nil {
		internalError(w, "unable to publish announcement")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to publish announcement")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
}

// handleAdminAnnouncement 更新一則既有的公告。
func (s *Server) handleAdminAnnouncement(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodPatch {
		methodNotAllowed(w)
		return
	}
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/announcements/"), "/"), 10, 64)
	if err != nil || id < 1 {
		badRequest(w, "invalid announcement id")
		return
	}
	var req announcementRequest
	if err := decodeLimitedJSON(w, r, &req, 4<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}
	body, expires, err := normalizeAnnouncement(req.Body, req.HoursUntilExpiry)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	actor := s.sessions.ResolveUser(r)
	now := time.Now()
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to update announcement")
		return
	}
	defer tx.Rollback()

	// 改為啟用時，先把其他全部關掉（理由見檔頭的「同時只有一則公告」）。
	if req.Active {
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE forum_announcements SET active = 0, updated_at = ?, updated_by = ? WHERE active = 1 AND id <> ?`,
			now, actor, id); err != nil {
			internalError(w, "unable to update announcement")
			return
		}
	}

	// 讀取改動前的內容與狀態，稽核紀錄需要它們。
	var beforeBody, beforeExpires string
	var beforeActive int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT body, active, COALESCE(DATE_FORMAT(expires_at, '%Y-%m-%dT%H:%i:%sZ'), '') FROM forum_announcements WHERE id = ?`, id).
		Scan(&beforeBody, &beforeActive, &beforeExpires); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		internalError(w, "unable to update announcement")
		return
	}

	result, err := tx.ExecContext(r.Context(), `
		UPDATE forum_announcements
		SET body = ?, active = ?, updated_at = ?, updated_by = ?, expires_at = ?
		WHERE id = ?`,
		body, boolToInt(req.Active), now, actor, expires, id)
	if err != nil {
		internalError(w, "unable to update announcement")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		// 與其他後臺寫入一致：0 影響列代表這則公告不存在。
		http.NotFound(w, r)
		return
	}

	if err := s.recordAdminAction(r, tx, "announcement.update", audit.TargetAnnouncement,
		strconv.FormatInt(id, 10), body,
		onlyChanged(
			audit.Change{Field: "body", Before: beforeBody, After: body},
			audit.Change{Field: "active", Before: strconv.FormatBool(beforeActive == 1), After: strconv.FormatBool(req.Active)},
			audit.Change{Field: "expiresAt", Before: beforeExpires, After: formatExpiry(expires)},
		)...); err != nil {
		internalError(w, "unable to update announcement")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to update announcement")
		return
	}
	writeOK(w, map[string]bool{"ok": true})
}

// handleAdminPostOrPin 分流 /api/admin/forum/posts/{id}… 這個前綴下的兩件事：
// 單篇文章的修改／刪除（既有），以及 /{id}/pin（置頂）。
//
// 為什麼要分流而不是把 /{id}/pin 註冊成另一條路由：ServeMux 對兩條樣式
// （"/api/admin/forum/posts/" 與 "/api/admin/forum/posts/"）的處理是一樣的
// —— 都是「前綴比對」，後註冊的會贏。因此若把 /{id}/pin 單獨註冊，它會
// **吃掉**整個 /{id} 前綴，而既有的修改與刪除就會全部 404。分流是唯一
// 不會讓兩者互相干擾的做法。
func (s *Server) handleAdminPostOrPin(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/pin") {
		s.handleAdminPostPin(w, r)
		return
	}
	s.handleAdminForumPost(w, r)
}

// handleAdminPostPin 置頂或取消置頂一篇文章。
//
// POST（而非 PATCH）搭配 body 裡的 pinned 布林：它與「切換」語意一致 ——
// 前端送的是「我要它變成這個狀態」，而���是一個「翻轉」指令。翻轉指令在
// 連點兩下時會得到相反的結果（而那正是使用者會做的事：再按一次取消）。
func (s *Server) handleAdminPostPin(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/forum/posts/"), "/pin"), 10, 64)
	if err != nil || id < 1 {
		badRequest(w, "invalid post id")
		return
	}
	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := decodeLimitedJSON(w, r, &req, 1<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}

	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to update post")
		return
	}
	defer tx.Rollback()

	var before int
	if err := tx.QueryRowContext(r.Context(), `SELECT pinned FROM forum_posts WHERE id = ?`, id).Scan(&before); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		internalError(w, "unable to load forum post")
		return
	}
	// 狀態沒變時直接回成功，不記稽核：稽核記的是「發生了什麼改變」，
	// 而「把未置頂設成未置頂」什麼都沒改。記下來只會讓「這篇被置頂過幾次」
	// 的答案失真。
	if (before == 1) == req.Pinned {
		writeOK(w, map[string]bool{"ok": true, "pinned": req.Pinned})
		return
	}

	result, err := tx.ExecContext(r.Context(), `UPDATE forum_posts SET pinned = ? WHERE id = ?`, boolToInt(req.Pinned), id)
	if err != nil {
		internalError(w, "unable to update post")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		http.NotFound(w, r)
		return
	}

	action := "post.unpin"
	if req.Pinned {
		action = "post.pin"
	}
	// diff 帶上前後狀態的布林，而不是「0 → 1」：稽核紀錄的讀者是人，
	// 而 pinned/unpinned 與「有沒有被置頂」是同一件事的兩種說法。
	if err := s.recordAdminAction(r, tx, action, audit.TargetPost, strconv.FormatInt(id, 10), "",
		audit.Change{Field: "pinned", Before: strconv.FormatBool(before == 1), After: strconv.FormatBool(req.Pinned)},
	); err != nil {
		internalError(w, "unable to update post")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to update post")
		return
	}
	// 置頂不重建搜尋索引：索引內容是文章文字，置頂不改變它，而重建會呼叫
	// Elasticsearch —— 在一個只是調整顯示順序的操作上做那是純粹的浪費。
	writeOK(w, map[string]bool{"ok": true, "pinned": req.Pinned})
}

// formatExpiry 把可空的到期時間轉成稽核紀錄裡的字串。
func formatExpiry(value sql.NullTime) string {
	if !value.Valid {
		return "（永不過期）"
	}
	return value.Time.UTC().Format(time.RFC3339)
}

// boolToInt 把布林轉成 MySQL TINYINT 的 0/1。
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
