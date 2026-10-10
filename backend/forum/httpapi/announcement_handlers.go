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

	  - 發佈**啟用中**的新公告時，在同一個交易裡把舊的設為不啟用
	    （publishAnnouncement）
	  - 改為啟用時同樣先把其他全部關掉（updateAnnouncement）

	兩處都以「這一則真的會顯示」為前提：發佈一則 active=false 的公告
	（先寫好、稍後才生效）不關任何東西 —— 關了的話，「先寫好」這個動作
	反而會把線上正在顯示的公告弄不見，而那顯然不是任何人預期的事。

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
	//
	// **判斷用的時鐘由 Go 端提供，而不是 NOW()**：expires_at 是 DATETIME，
	// 寫入端傳的是 time.Time 而 driver 依 DSN 的 loc 序列化成主機掛鐘字串；
	// 而 NOW() 是 MySQL **session** 時區，兩者只有在 session 時區剛好等於
	// 主機時區時才相同（站台 UTC+8 + 雲端 MySQL UTC 是常見組合）。後臺的
	// Effective 用 time.Now() 比同一個欄位，因此把 ? 帶進來讓兩邊共用同一個
	// clock，是唯一能保證「後臺說顯示中、公開頁就有橫幅」的做法。
	//
	// 放棄 SQL 端的常數比較不影響索引：條件是 (active, expires_at)，而
	// idx_forum_announcements_active_expires 的前導欄是 active 的等值條件，
	// 命中的本來就只有 active 的那一列（同一時間最多一則）。
	//
	// expires_at 同樣刻意**不**在 SQL 裡 DATE_FORMAT：DATETIME 沒有時區資訊，
	// DATE_FORMAT 用的是 MySQL session 時區，把結果硬寫上一個 "Z" 等於
	// 宣告它是 UTC，而實際上它是主機的掛鐘時間。掃成 time.Time 再自行 Format
	// 就沒有這個問題 —— DSN 帶 loc=Local，掃出來的值帶真實偏移，轉 UTC 後
	// 格式與 createdAt 完全一致。
	row := s.db.QueryRowContext(r.Context(), `
		SELECT body, created_at, expires_at
		FROM forum_announcements
		WHERE active = 1 AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY created_at DESC, id DESC
		LIMIT 1`, time.Now())

	var announcement forumAnnouncement
	var expires sql.NullTime
	if err := row.Scan(&announcement.Body, &announcement.PublishedAt, &expires); err != nil {
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
	announcement.ExpiresAt = formatAnnouncementExpiry(expires)
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
		SELECT id, body, active, created_by, created_at, updated_by, updated_at, expires_at
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
	var expires sql.NullTime
	if err := row.Scan(&item.ID, &item.Body, &active, &item.CreatedBy, &item.CreatedAt,
		&item.UpdatedBy, &item.UpdatedAt, &expires); err != nil {
		return adminAnnouncement{}, err
	}
	item.Active = active == 1
	item.ExpiresAt = formatAnnouncementExpiry(expires)
	// Effective 用**時間比較**而不是字串比較。字串比較在兩個時鐘不一致時
	// （例如 expires_at 是主機掛鐘時間、now 是 UTC）會安靜地給出錯誤答案，
	// 而這個欄位正是管理員用來判斷「橫幅到底還在不在」的唯一依據。
	//
	// 判斷條件與公開端點的 SQL（active = 1 AND (expires_at IS NULL OR
	// expires_at > ?)）**共用同一個 clock**：那個 ? 傳的就是 time.Now()，
	// 與這裡的比較同一個來源。兩邊因此不會出現「後臺說顯示中、公開頁沒有」。
	// 這裡也刻意用 time.Now()（本機時區）而不是 UTC：掃回來的 expires.Time
	// 帶 DSN 的 loc，與寫入端的序列化方式對稱，UTC 反而會差一個偏移。
	item.Effective = item.Active && (!expires.Valid || expires.Time.After(time.Now()))
	return item, nil
}

// formatAnnouncementExpiry 把可空的到期時間轉成對外的字串。
//
// 空字串代表「永不自動過期」（前端以 omitempty 省略這個欄位）。
//
// 一律轉成 UTC 的 RFC3339，因此格式與 createdAt / updatedAt 完全一致 ——
// 這正是前端能把它當成一般時間戳處理（顯示相對時間、做到期比較）的原因。
// 曾在 SQL 裡用 DATE_FORMAT 加上假的 "Z"，結果是站台不在 UTC 時這裡會與
// Effective 判定用上兩個不同的時鐘。
func formatAnnouncementExpiry(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}

// announcementRequest 是發佈與修改共用的請求主體。
type announcementRequest struct {
	Body string `json:"body"`
	// Active 為 false 代表「發佈但不顯示」。存在的理由是一則要寫好、稍後才
	// 生效的公告很常見，而為了這個用途去建一列再刪掉並沒有比較好。
	Active bool `json:"active"`
	// HoursUntilExpiry <= 0 代表永不自動過期。
	//
	// 這個欄位是「要設定的時長」而不是「剩餘時長」—— 後端收到什麼就覆蓋成什麼，
	// 因此**沒有**「保持原值」的語意。這代表任何只想切換 active 的呼叫端都必須
	// 自己帶上原本的到期語意：固定送 0 會把一則有期限的公告變成永久顯示，而畫面
	// 上看不出這個差別。前端因此在切換狀態前把絕對時間換算成剩餘小時
	// （見 AnnouncementsPage.tsx 的 setAnnouncementActive）。
	HoursUntilExpiry int `json:"hoursUntilExpiry"`
}

// normalizeAnnouncement 修剪並驗證一則公告，回傳整理後的值與到期時間。
//
// expires 用「幾小時後」而不是絕對時間戳：管理員在後臺填的是「這個公告
// 三天後失效」而不是精確到分鐘的時間，而絕對時間戳在跨時區的後臺與
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
	//
	// 關鍵是「只在真的要顯示時才關」：active=false 的語意是「發佈但不顯示」
	// （一則先寫好、稍後才生效的公告）。若無條件關掉舊的，勾掉「顯示」再
	// 發佈就會讓線上正在顯示的公告憑空消失 —— 那是欄位本身要支援的情境，
	// 結果變成「順手把別人的公告關掉」。沒有任何錯誤訊息，因為每一步都成功。
	if req.Active {
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE forum_announcements SET active = 0, updated_at = ?, updated_by = ? WHERE active = 1`,
			now, actor); err != nil {
			internalError(w, "unable to publish announcement")
			return
		}
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
	// expires_at 掃成可空時間再格式化（理由同 formatAnnouncementExpiry），
	// 否則 Before 會是主機掛鐘時間的字串而 After 是 UTC 的字串，格式不同
	// 會讓 onlyChanged 幾乎永遠判定 expiresAt「有變更」，
	// 於是稽核紀錄裡出現大量假變更。
	var beforeBody string
	var beforeExpiresRaw sql.NullTime
	var beforeActive int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT body, active, expires_at FROM forum_announcements WHERE id = ?`, id).
		Scan(&beforeBody, &beforeActive, &beforeExpiresRaw); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		internalError(w, "unable to update announcement")
		return
	}
	beforeExpires := formatAnnouncementExpiry(beforeExpiresRaw)

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

// adminPostPinIDFromPath 從 /api/admin/forum/posts/{id}/pin 取出貼文編號。
//
// 為什麼抽成純函式：它是這條路由唯一決定「操作哪一篇文章」的環節，而它的失敗
// 完全安靜 —— 解析出一個合法的 id 只會讓請求繼續往下，沒有任何訊息告訴管理員
// 「你操作的其實是另一篇」。抽出來才能被表格測試直接釘住（見 announcement_test.go
// 的 TestAdminPostPinIDFromPath）。
//
// TrimSuffix 而不是 Trim：後者的第二個參數是**字元集合**而不是尾綴，因此
// "/api/admin/forum/posts/pin1/pin" 會被吃掉前導的 "pin" 而解析成 id 1 ——
// 操作的貼文與網址指的不是同一篇，稽核紀錄也會記下一顆操作者沒選的貼文。
// 兩個 Trim 各自的失敗模式都必須保留：前綴對不上時整個字串原樣留下來，
// ParseInt 自然失敗；尾綴對不上時也一樣。
func adminPostPinIDFromPath(path string) (int64, error) {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(path, "/api/admin/forum/posts/"), "/pin")
	return strconv.ParseInt(trimmed, 10, 64)
}

// handleAdminPostPin 置頂或取消置頂一篇文章。
//
// POST（而非 PATCH）搭配 body 裡的 pinned 布林：它與「切換」語意一致 ——
// 前端送的是「我要它變成這個狀態」，而這是一個「翻轉」指令。翻轉指令在
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

	id, err := adminPostPinIDFromPath(r.URL.Path)
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

	// deleted_at IS NULL：置頂一篇已刪除的貼文沒有意義 —— 它不會出現在任何
	// 公開頁，而稽核會記下一筆「置頂了某篇貼文」卻看不到效果的操作。與
	// SELECT 保持同一個條件，兩邊才不會出現「讀得到、改不到」的落差。
	var before int
	if err := tx.QueryRowContext(r.Context(), `SELECT pinned FROM forum_posts WHERE id = ? AND deleted_at IS NULL`, id).Scan(&before); err != nil {
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

	result, err := tx.ExecContext(r.Context(), `UPDATE forum_posts SET pinned = ? WHERE id = ? AND deleted_at IS NULL`, boolToInt(req.Pinned), id)
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
