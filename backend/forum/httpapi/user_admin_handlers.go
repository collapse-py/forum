// 本檔案實作「管理後台 → 使用者管理」與「使用者標籤」兩組 RESTful API handler。
//
// 路由註冊於 server.go 的 Handler()：
//
//	GET    /api/admin/tags                  -> handleAdminTags       列出所有標籤
//	POST   /api/admin/tags                  -> handleAdminTags       新增標籤
//	PATCH  /api/admin/tags/{id}             -> handleAdminTag        重新命名標籤
//	DELETE /api/admin/tags/{id}             -> handleAdminTag        刪除標籤
//	GET    /api/admin/users                 -> handleAdminUsers      列出所有使用者（含統計與標籤）
//	PATCH  /api/admin/users/{email}         -> handleAdminUser       停權 / 恢復帳號
//	GET    /api/admin/users/{email}/content -> handleAdminUserContent  檢視該使用者的文章與留言
//	PUT    /api/admin/users/{email}/tags    -> handleAdminUserTags     覆寫該使用者的標籤集合
//	POST   /api/admin/users/{email}/posts   -> createAdminUserPost     以該使用者身分發文
//	POST   /api/admin/users/{email}/comments-> createAdminUserComment  以該使用者身分留言
//
// 兩項貫穿全檔的約定：
//  1. 權限：所有對外進入點都先呼叫 requireAdminForum()（定義於 forum_admin_handlers.go），
//     內部子處理器則信任呼叫端已完成檢查，不再重複驗證。
//  2. 防偽 Cross-Site Request Forgery：所有會寫入資料的分支都必須通過 isTrustedOrigin()
//     （csrf.go，比對 Origin / Referer 與 config 的 TrustedOrigins 清單），唯讀 GET 免檢查。
//
// 資料表對應：
//   - forum_users                帳號主檔（email 為主鍵，status 為 ACTIVE / SUSPENDED）
//   - forum_profiles             個人檔案（暱稱等，以 author_email 為主鍵）
//   - forum_user_tags            標籤總表（name 有 UNIQUE 唯一索引）
//   - forum_user_tag_assignments 標籤綁定表（複合主鍵 user_email + tag_id，未設外鍵）
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"forum/forum/audit"
	"forum/forum/logger"

	"github.com/go-sql-driver/mysql"
)

// adminUser 是 GET /api/admin/users 回傳的單一使用者列。
// 欄位來源分散在多張表：forum_users（帳號與狀態）、forum_profiles（暱稱），
// 以及三個關聯子查詢（發文數、留言數、按讚數）。
type adminUser struct {
	// Email 為 Google 帳號信箱，同時是 forum_users 的主鍵。
	Email string `json:"email"`
	// Status 為帳號狀態，僅有 "ACTIVE"（正常）與 "SUSPENDED"（停權）兩種值。
	Status string `json:"status"`
	// Nickname 取自 forum_profiles；使用者尚未建立個人檔案時為空字串。
	Nickname string `json:"nickname"`
	// Tags 為該使用者的標籤集合，由 loadAdminUserTags 逐一補上（依名稱排序）。
	Tags []adminUserTag `json:"tags"`
	// PostCount 為該使用者在 forum_posts 的文章總數。
	PostCount int `json:"postCount"`
	// CommentCount 為該使用者在 forum_post_comments 的留言總數。
	CommentCount int `json:"commentCount"`
	// LikeCount 為該使用者在 forum_post_likes 的按讚總數。
	LikeCount int `json:"likeCount"`
	// CreatedAt / UpdatedAt 為帳號建立與最後異動時間；
	// UpdatedAt 也會在停權、恢復、發文、留言時被更新，因此列表以此欄位排序。
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// adminUserContent 是 GET /api/admin/users/{email}/content 的回應主體，
// 一次帶回該使用者的所有文章與所有留言，供後台展開式編輯畫面直接渲染。
type adminUserContent struct {
	// Posts 為該使用者的文章，每筆含按讚數與留言數（由子查詢即時統計）。
	Posts []adminForumPost `json:"posts"`
	// Comments 為該使用者的留言，含所屬文章的 postId 以便就地編輯。
	Comments []adminForumComment `json:"comments"`
}

// adminUserTag 是一個使用者標籤，同時用於三種回應：
//   - 標籤總表（GET /api/admin/tags）
//   - 單一使用者已綁定的標籤（GET /api/admin/users/{email}/tags）
//   - PUT 標籤綁定成功後回傳的最新結果
type adminUserTag struct {
	// ID 為 forum_user_tags 的自增主鍵。
	ID int64 `json:"id"`
	// Name 為標籤名稱，最長 50 字（VARCHAR(50)），於資料庫有 UNIQUE 唯一索引。
	Name string `json:"name"`
	// CreatedAt / UpdatedAt 為標籤建立與最後重新命名時間。
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// handleAdminTags 處理「標籤總表」的集合操作。
//
//	GET  /api/admin/tags  -> 200 {"items": [adminUserTag, ...]}，依名稱升冪排序
//	POST /api/admin/tags  -> 201 adminUserTag（body: {"name": "..."}）
//
// 權限：管理員。POST 另需通過 isTrustedOrigin 的來源檢查。
func (s *Server) handleAdminTags(w http.ResponseWriter, r *http.Request) {
	// 未登入或非管理員時直接回 401 並終止，後續邏輯不執行。
	if !s.requireAdminForum(w, r) {
		return
	}

	// ---- GET：列出全部標籤 ----
	if r.Method == http.MethodGet {
		rows, err := s.db.QueryContext(r.Context(), `SELECT id, name, created_at, updated_at FROM forum_user_tags ORDER BY name ASC`)
		if err != nil {
			internalError(w, "unable to load user tags")
			return
		}
		// rows 必須在函式結束前關閉，否則連線不歸還連線池。
		defer rows.Close()

		// 用 make 建立空 slice 而非 var 宣告，讓沒有標籤時 JSON 序列化成 [] 而非 null。
		tags := make([]adminUserTag, 0)
		for rows.Next() {
			var tag adminUserTag
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
				internalError(w, "unable to read user tags")
				return
			}
			tags = append(tags, tag)
		}
		// rows.Next() 結束後仍須檢查 rows.Err()，迭代過程中的網路錯誤不會被 Next 回報。
		if err := rows.Err(); err != nil {
			internalError(w, "unable to read user tags")
			return
		}
		writeOK(w, map[string]interface{}{"items": tags})
		return
	}

	// ---- POST：新增標籤 ----
	// 同時檢查 method 與來源：method 正確但來源不符回 403，method 不符回 405。
	if r.Method != http.MethodPost || !s.isTrustedOrigin(r) {
		if r.Method == http.MethodPost {
			writeError(w, http.StatusForbidden, "invalid origin")
		} else {
			methodNotAllowed(w)
		}
		return
	}

	// 僅需要 name 欄位，故以匿名 struct 當作請求主體。
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// 先去除前後空白，讓「只有空白的名稱」被判定為無效。
	req.Name = strings.TrimSpace(req.Name)
	// 以 rune（非 byte）計算長度，中文標籤名才不會被算成 3 倍長度。
	if req.Name == "" || len([]rune(req.Name)) > 50 {
		badRequest(w, "標籤名稱需為 1 至 50 字")
		return
	}

	// 建立與更新時間同一瞬間，由應用層產生（此專案未使用 DB 端的時間函式）。
	now := time.Now()
	// 稽核：新增標籤是字典層級的操作（不影響任何使用者），但仍需記錄 ——
	// 「這個名稱是什麼時候、經誰的手建立起來的」決定了後續看到這個標籤時
	// 該不該信任它。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to create user tag")
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `INSERT INTO forum_user_tags (name, created_at, updated_at) VALUES (?, ?, ?)`, req.Name, now, now)
	if err != nil {
		// MySQL 錯誤碼 1062 = ER_DUP_ENTRY（唯一索引 uq_forum_user_tags_name 衝突）。
		// 這是使用者可預期的營運錯誤，改回 409 Conflict 而非 500。
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			writeError(w, http.StatusConflict, "標籤已存在")
			return
		}
		internalError(w, "unable to create user tag")
		return
	}
	// 取出剛才 INSERT 產生的自增主鍵，直接回給前端，省去一次查詢。
	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to read created user tag")
		return
	}
	if err := s.recordAdminAction(r, tx, adminActionTagCreate, audit.TargetTag, strconv.FormatInt(id, 10), req.Name,
		audit.Change{Field: "name", Before: "", After: req.Name}); err != nil {
		internalError(w, "unable to create user tag")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to read created user tag")
		return
	}
	writeJSON(w, http.StatusCreated, adminUserTag{ID: id, Name: req.Name, CreatedAt: now, UpdatedAt: now})
}

// handleAdminTag 處理「單一標籤」操作，標籤 ID 由路徑末段取出。
//
//	DELETE /api/admin/tags/{id} -> 200 {"ok": true}，id 不存在時 404
//	PATCH  /api/admin/tags/{id} -> 200 {"ok": true}，body: {"name": "..."}
//
// 權限：管理員。兩個 method 皆需通過 isTrustedOrigin 來源檢查。
func (s *Server) handleAdminTag(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}

	// 剝除 "/api/admin/tags/" 前綴與兩端斜線後，剩下的字串必須是合法的正整數。
	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/tags/"), "/"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(w, "invalid tag id")
		return
	}

	// ---- DELETE：刪除標籤 ----
	if r.Method == http.MethodDelete {
		if !s.isTrustedOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		// 稽核：刪除標籤會同時消掉它綁在多少人身上，因此那個數字本身就是
		// 這次操作的重要資訊（「我剛剛讓 30 個人身上的分類消失」）。這也是
		// 原本把「清除綁定」視為可接受失敗的寫法在有了稽核之後必須改掉的
		// 原因：那個錯誤原本會被完全忽略，而現在它是稽核內容的一部分。
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to delete user tag")
			return
		}
		defer tx.Rollback()

		var beforeName string
		if err := tx.QueryRowContext(r.Context(),
			`SELECT name FROM forum_user_tags WHERE id = ?`, id).Scan(&beforeName); err != nil {
			if err == sql.ErrNoRows {
				// 標籤不存在；DELETE 不會回 ErrNoRows，但先讀一次才能分辨
				// 「不存在」與「刪了但沒清綁定」這兩種 404。
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load user tag")
			return
		}
		var assignedCount int
		if err := tx.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM forum_user_tag_assignments WHERE tag_id = ?`, id).Scan(&assignedCount); err != nil {
			internalError(w, "unable to load user tag assignments")
			return
		}

		result, err := tx.ExecContext(r.Context(), `DELETE FROM forum_user_tags WHERE id = ?`, id)
		if err != nil {
			internalError(w, "unable to delete user tag")
			return
		}
		// 影響列數為 0 代表標籤不存在；DELETE 不會回傳 ErrNoRows，必須靠 RowsAffected 判斷。
		if affected, _ := result.RowsAffected(); affected == 0 {
			http.NotFound(w, r)
			return
		}
		// forum_user_tag_assignments 沒有設定 FOREIGN KEY ... ON DELETE CASCADE，
		// 因此需在此手動清除所有指向此標籤的綁定記錄，否則會留下孤兒資料。
		// 錯誤不再忽略：它與標籤的刪除在同一個交易裡，忽略等於留下孤兒綁定
		// 並讓稽核紀錄宣稱「清掉了 N 個」而實際上沒有。
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM forum_user_tag_assignments WHERE tag_id = ?`, id); err != nil {
			internalError(w, "unable to delete user tag")
			return
		}
		if err := s.recordAdminAction(r, tx, adminActionTagDelete, audit.TargetTag, strconv.FormatInt(id, 10), beforeName,
			audit.Change{Field: "name", Before: beforeName, After: "（標籤已刪除）"},
			audit.Change{Field: "assignmentsRemoved", Before: "", After: strconv.Itoa(assignedCount)}); err != nil {
			internalError(w, "unable to delete user tag")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to delete user tag")
			return
		}
		writeOK(w, map[string]bool{"ok": true})
		return
	}

	// ---- PATCH：重新命名標籤 ----
	if r.Method != http.MethodPatch || !s.isTrustedOrigin(r) {
		if r.Method == http.MethodPatch {
			writeError(w, http.StatusForbidden, "invalid origin")
		} else {
			methodNotAllowed(w)
		}
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	// 驗證規則與新增標籤一致：1 至 50 字。
	if req.Name == "" || len([]rune(req.Name)) > 50 {
		badRequest(w, "標籤名稱需為 1 至 50 字")
		return
	}
	// 稽核：重新命名標籤要記下舊名 —— 稽核紀錄的讀者看到一個標籤名時，
	// 常見的問題是「這個人三個月前被標成 A，A 又是什麼」；沒有舊名就答不出來。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to update user tag")
		return
	}
	defer tx.Rollback()

	var beforeName string
	if err := tx.QueryRowContext(r.Context(),
		`SELECT name FROM forum_user_tags WHERE id = ?`, id).Scan(&beforeName); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		internalError(w, "unable to load user tag")
		return
	}

	result, err := tx.ExecContext(r.Context(), `UPDATE forum_user_tags SET name = ?, updated_at = ? WHERE id = ?`, req.Name, time.Now(), id)
	if err != nil {
		// 同樣處理唯一索引衝突：改名撞到既有標籤名稱時回 409。
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			writeError(w, http.StatusConflict, "標籤已存在")
			return
		}
		internalError(w, "unable to update user tag")
		return
	}
	// 注意：MySQL 對「值完全相同」的 UPDATE 也會回報 0 影響列，
	// 因此用舊名稱覆寫自己會被判成 404；正常情況前端會自動帶入現有名稱，實際上是送出新名稱。
	if affected, _ := result.RowsAffected(); affected == 0 {
		http.NotFound(w, r)
		return
	}
	if err := s.recordAdminAction(r, tx, adminActionTagUpdate, audit.TargetTag, strconv.FormatInt(id, 10), beforeName,
		audit.Change{Field: "name", Before: beforeName, After: req.Name}); err != nil {
		internalError(w, "unable to update user tag")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to update user tag")
		return
	}
	writeOK(w, map[string]bool{"ok": true})
}

// handleAdminUsers 列出所有使用者（僅支援 GET），供後台使用者管理表格一次載入。
//
//	GET /api/admin/users -> 200 {"items": [adminUser, ...]}
//
// 排序：updated_at 降冪（最近有活動的在前），updated_at 相同時以 email 升冪確保順序穩定。
// 權限：管理員。
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// 以三個關聯子查詢一次算完每位使用者的發文/留言/按讚數，避免 N+1 查詢。
	// COALESCE 處理尚未建立個人檔案（forum_profiles 無對應列）的情況，避免 Scan 到 NULL 失敗。
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT u.email, u.status, COALESCE(fp.nickname, ''), u.created_at, u.updated_at,
		       (SELECT COUNT(*) FROM forum_posts WHERE author_email = u.email),
		       (SELECT COUNT(*) FROM forum_post_comments WHERE author_email = u.email),
		       (SELECT COUNT(*) FROM forum_post_likes WHERE author_email = u.email)
		FROM forum_users u
		LEFT JOIN forum_profiles fp ON fp.author_email = u.email
		ORDER BY u.updated_at DESC, u.email ASC`)
	if err != nil {
		internalError(w, "unable to load users")
		return
	}
	defer rows.Close()

	items := make([]adminUser, 0)
	for rows.Next() {
		var item adminUser
		// Scan 的目標順序必須與 SELECT 欄位順序完全一致。
		if err := rows.Scan(&item.Email, &item.Status, &item.Nickname, &item.CreatedAt, &item.UpdatedAt, &item.PostCount, &item.CommentCount, &item.LikeCount); err != nil {
			internalError(w, "unable to read users")
			return
		}
		// 標籤無法在上方單一查詢中以固定欄位數取出，改為每位使用者各查一次。
		item.Tags, err = s.loadAdminUserTags(r.Context(), item.Email)
		if err != nil {
			internalError(w, "unable to read user tags")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		internalError(w, "unable to read users")
		return
	}
	writeOK(w, map[string]interface{}{"items": items})
}

// handleAdminUser 是「單一使用者」的路由分派器（dispatcher）。
//
// 它依路徑後綴把請求轉給對應的子處理器；若都沒有匹配，剩下的就是
// 「更新帳號狀態」的 PATCH 分支：
//
//	PATCH /api/admin/users/{email} -> 200 {"ok": true}，body: {"status": "ACTIVE"|"SUSPENDED"}
//
// 權限：管理員（已於本函式開頭統一檢查，子處理器信任此結果）。
func (s *Server) handleAdminUser(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}

	// 取出 "/api/admin/users/" 之後的完整尾段（含子資源路徑），後續以 HasSuffix 判斷。
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/users/"), "/")

	// GET /api/admin/users/{email}/content -> 檢視文章與留言
	if strings.HasSuffix(path, "/content") {
		s.handleAdminUserContent(w, r, strings.TrimSuffix(path, "/content"))
		return
	}
	// GET / PUT /api/admin/users/{email}/tags -> 讀取 / 覆寫標籤綁定
	if strings.HasSuffix(path, "/tags") {
		s.handleAdminUserTags(w, r, strings.TrimSuffix(path, "/tags"))
		return
	}
	// POST /api/admin/users/{email}/posts -> 以該使用者身分發文
	if strings.HasSuffix(path, "/posts") && r.Method == http.MethodPost {
		s.createAdminUserPost(w, r, strings.TrimSuffix(path, "/posts"))
		return
	}
	// POST /api/admin/users/{email}/comments -> 以該使用者身分留言
	if strings.HasSuffix(path, "/comments") && r.Method == http.MethodPost {
		s.createAdminUserComment(w, r, strings.TrimSuffix(path, "/comments"))
		return
	}

	// ---- 預設分支：更新帳號狀態（停權 / 恢復） ----
	if r.Method != http.MethodPatch {
		methodNotAllowed(w)
		return
	}
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	// email 可能是未跳碼的原文（後台模板直接插入），也可能是被 encodeURIComponent 編碼過
	// 的字串（%40 代表 @），故必須 PathUnescape 後才能拿去比對資料庫。
	email, err := url.PathUnescape(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/users/"), "/"))
	if err != nil || email == "" {
		badRequest(w, "invalid user email")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// 狀態一律轉為大寫後比對，讓前端傳 "active" 或 " active " 也能通過。
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	// 白名單驗證：只允許切換這兩個狀態，避免任意字串寫入 status 欄位。
	if req.Status != "ACTIVE" && req.Status != "SUSPENDED" {
		badRequest(w, "invalid user status")
		return
	}
	// 一併更新 updated_at，讓此使用者在使用者列表的排序往前移動。
	//
	// 稽核：停權與恢復分成兩個動作名稱（見 audit_log.go 的常數說明），且必須
	// 記下改動前的狀態 —— 資料庫裡只留得到「現在是 SUSPENDED」，答不出
	// 「原本是 ACTIVE 還是從未被停過」。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to update user")
		return
	}
	defer tx.Rollback()

	// 讀取改動前的狀態。影響列數為 0 與「狀態沒變」是兩件事：前者是 404，
	// 後者 MySQL 回報 0 變更列（DSN 沒加 clientFoundRows），因此必須先讀
	// 才能把兩者分開。
	var beforeStatus string
	if err := tx.QueryRowContext(r.Context(),
		`SELECT status FROM forum_users WHERE email = ?`, email).Scan(&beforeStatus); err != nil {
		if err == sql.ErrNoRows {
			// 此 email 尚未登入過（forum_users 無此列），回 404。
			http.NotFound(w, r)
			return
		}
		internalError(w, "unable to load user")
		return
	}

	result, err := tx.ExecContext(r.Context(), `UPDATE forum_users SET status = ?, updated_at = ? WHERE email = ?`, req.Status, time.Now(), email)
	if err != nil {
		internalError(w, "unable to update user")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		http.NotFound(w, r)
		return
	}
	action := adminActionUserReinstate
	if req.Status == "SUSPENDED" {
		action = adminActionUserSuspend
	}
	if err := s.recordAdminAction(r, tx, action, audit.TargetUser, email, email,
		adminStatusChange(beforeStatus, req.Status)); err != nil {
		internalError(w, "unable to update user")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to update user")
		return
	}
	// 停權效果由 server.go 的 requireLogin 讀取 status 判定，並擋下該使用者的寫入請求。
	writeOK(w, map[string]bool{"ok": true})
}

// handleAdminUserTags 讀取或覆寫單一使用者的標籤綁定。
//
//	GET /api/admin/users/{email}/tags -> 200 {"items": [adminUserTag, ...]}
//	PUT /api/admin/users/{email}/tags -> 200 {"items": [adminUserTag, ...]}（回傳寫入後的最新綁定）
//	                                    body: {"tagIds": [1, 3, 7]}
//
// PUT 採「全量覆寫」語意：先刪除該使用者所有既有綁定，再依請求內容重複插入，
// 因此傳入空陣列即代表清除此使用者的所有標籤。
// 本函式的請求追蹤日誌是全專案最密集的（decode 前後 email、tag_ids 比對），
// 用於排查「標籤寫入結果與預期不符」的問題。
//
// 權限：由呼叫端 handleAdminUser 完成管理員檢查。
func (s *Server) handleAdminUserTags(w http.ResponseWriter, r *http.Request, rawEmail string) {
	logger.InfofContext(r.Context(), "[ADMIN-TAGS] request method=%s path=%s raw_email=%s", r.Method, r.URL.Path, rawEmail)

	// 先把路徑中的 email 還原（%XX 解碼），解碼失敗即為非法請求。
	email, err := s.decodeAdminUserEmail(rawEmail)
	if err != nil {
		logger.WarnfContext(r.Context(), "[ADMIN-TAGS] invalid user email: %v", err)
		badRequest(w, err.Error())
		return
	}
	logger.InfofContext(r.Context(), "[ADMIN-TAGS] decoded email=%s", email)

	// 以「查得到使用者」作為 404 判斷，並同時擋住對不存在帳號建立標籤綁定的情況。
	var userCount int
	if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_users WHERE email = ?`, email).Scan(&userCount); err != nil {
		logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] user lookup failed email=%s: %v", email, err)
		internalError(w, "unable to load user")
		return
	}
	if userCount == 0 {
		logger.WarnfContext(r.Context(), "[ADMIN-TAGS] user not found email=%s", email)
		http.NotFound(w, r)
		return
	}

	// ---- GET：讀取目前綁定的標籤 ----
	if r.Method == http.MethodGet {
		tags, err := s.loadAdminUserTags(r.Context(), email)
		if err != nil {
			internalError(w, "unable to load user tags")
			return
		}
		writeOK(w, map[string]interface{}{"items": tags})
		return
	}

	// 其餘僅支援 PUT（以全量覆寫的方式更新標籤）。
	if r.Method != http.MethodPut {
		logger.WarnfContext(r.Context(), "[ADMIN-TAGS] unsupported method=%s", r.Method)
		methodNotAllowed(w)
		return
	}
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	// ---- PUT：解析請求主體 ----
	var req struct {
		TagIDs []int64 `json:"tagIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnfContext(r.Context(), "[ADMIN-TAGS] invalid JSON email=%s: %v", email, err)
		badRequest(w, "invalid request")
		return
	}
	logger.InfofContext(r.Context(), "[ADMIN-TAGS] parsed tag_ids=%v email=%s", req.TagIDs, email)

	// 格式驗證：每個 id 必須為正數，且不可重複。
	// 重複會導致後續 INSERT 觸發 (user_email, tag_id) 複合主鍵衝突，故在寫入前就擋掉。
	seen := make(map[int64]bool, len(req.TagIDs))
	for _, tagID := range req.TagIDs {
		if tagID <= 0 || seen[tagID] {
			badRequest(w, "invalid tag ids")
			return
		}
		seen[tagID] = true
	}

	// 存在性驗證：一次撈出「實際存在且符合請求 id」的標籤數量。
	// forum_user_tag_assignments 未設外鍵，若不先驗證就會寫入指向不存在標籤的孤兒綁定。
	var existingTagCount int
	if len(req.TagIDs) > 0 {
		// 依參數個數動態產生 "?,?,?" 佔位符；id 一律以佔位符傳值，絕不字串拼接進 SQL。
		placeholders := strings.TrimRight(strings.Repeat("?,", len(req.TagIDs)), ",")
		args := make([]interface{}, len(req.TagIDs))
		for index, tagID := range req.TagIDs {
			args[index] = tagID
		}
		if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_user_tags WHERE id IN (`+placeholders+`)`, args...).Scan(&existingTagCount); err != nil {
			logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] tag validation failed email=%s tag_ids=%v: %v", email, req.TagIDs, err)
			internalError(w, "unable to validate user tags")
			return
		}
	}
	// 實際筆數與請求筆數不符，代表有 id 在 forum_user_tags 中不存在。
	if existingTagCount != len(req.TagIDs) {
		logger.WarnfContext(r.Context(), "[ADMIN-TAGS] tag validation mismatch email=%s requested=%d existing=%d", email, len(req.TagIDs), existingTagCount)
		badRequest(w, "invalid tag ids")
		return
	}

	// ---- 以交易執行「先刪後插」，確保不會出現只有部分綁定成功的狀態 ----
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] begin transaction failed email=%s: %v", email, err)
		internalError(w, "unable to update user tags")
		return
	}
	// defer Rollback 是安全網：若下方任何一步失敗或提前 return，交易會自動回滾；
	// 若已成功 Commit，Rollback 會回傳 ErrTxDone（無副作用），可安全忽略。
	defer tx.Rollback()

	// 稽核的「改動前」必須在 DELETE 之前讀取：這兩步都在同一個交易裡，
	// 因此讀到的值就是操作前的狀態，不受本交易自身的寫入影響。
	//
	// diff 記的是「原本的標籤名清單」與「送來的標籤名清單」兩行，而不是
	// 逐項比對。理由是這個操作的本質是「整組被換掉」：逐項 diff 會產生
	// 2N 列重複內容（舊的 N 個 + 新的 N 個），而稽核要回答的問題是
	// 「這個人現在被標成什麼、原本被標成什麼」—— 那正是兩個整組清單。
	beforeTagNames, err := loadAdminUserTagNames(r.Context(), tx, email)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[AUDIT] 讀取改動前標籤失敗 email=%s: %v", email, err)
		internalError(w, "unable to update user tags")
		return
	}
	afterTagNames := make([]string, 0, len(req.TagIDs))
	for _, tagID := range req.TagIDs {
		var name string
		if err := tx.QueryRowContext(r.Context(), `SELECT name FROM forum_user_tags WHERE id = ?`, tagID).Scan(&name); err != nil {
			internalError(w, "unable to update user tags")
			return
		}
		afterTagNames = append(afterTagNames, name)
	}

	// 全量覆寫的第一步：清空此使用者的所有既有綁定（也涵蓋「取消全部標籤」的情境）。
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM forum_user_tag_assignments WHERE user_email = ?`, email); err != nil {
		logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] delete assignments failed email=%s: %v", email, err)
		internalError(w, "unable to update user tags")
		return
	}

	// 第二步：逐一插入請求中的每個標籤綁定。
	for _, tagID := range req.TagIDs {
		result, err := tx.ExecContext(r.Context(), `INSERT INTO forum_user_tag_assignments (user_email, tag_id) VALUES (?, ?)`, email, tagID)
		if err != nil {
			logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] insert assignment failed email=%s tag_id=%d: %v", email, tagID, err)
			internalError(w, "unable to update user tags")
			return
		}
		// 明確要求每筆 INSERT 影響 1 列；0 列代表綁定未寫入（與預期不符），不可靜默忽略。
		if affected, err := result.RowsAffected(); err != nil || affected != 1 {
			logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] unexpected insert result email=%s tag_id=%d affected=%d err=%v", email, tagID, affected, err)
			internalError(w, "unable to update user tags")
			return
		}
	}
	// 提交後重新讀取一次，讓回應內容即為資料庫的權威狀態（依名稱排序）。
	if err := s.recordAdminAction(r, tx, adminActionUserTags, audit.TargetUser, email, email,
		audit.Change{Field: "tags", Before: strings.Join(beforeTagNames, ", "), After: strings.Join(afterTagNames, ", ")}); err != nil {
		logger.ErrorfContext(r.Context(), "[AUDIT] 寫入標籤變更紀錄失敗 email=%s: %v", email, err)
		internalError(w, "unable to update user tags")
		return
	}
	if err := tx.Commit(); err != nil {
		logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] commit failed email=%s: %v", email, err)
		internalError(w, "unable to update user tags")
		return
	}

	// 提交後重新讀取一次，讓回應內容即為資料庫的權威狀態（依名稱排序）。
	tags, err := s.loadAdminUserTags(r.Context(), email)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[ADMIN-TAGS] reload assignments failed email=%s: %v", email, err)
		internalError(w, "unable to load user tags")
		return
	}
	writeOK(w, map[string]interface{}{"items": tags})
	logger.InfofContext(r.Context(), "[ADMIN-TAGS] updated email=%s tag_count=%d", email, len(tags))
}

// loadAdminUserTags 讀取指定使用者已綁定的所有標籤，依標籤名稱升冪排序。
// 找不到綁定時回傳空 slice（序列化為 []）而非 nil，前端可直接迭代。
// 被 handleAdminUsers（逐位使用者補標籤）與標籤讀寫流程共用；此函式不寫 HTTP 回應，
// 錯誤一律回傳給呼叫端統一處理。
func (s *Server) loadAdminUserTags(ctx context.Context, email string) ([]adminUserTag, error) {
	// 由綁定表 join 回標籤總表，才拿得到顯示用的名稱與時間戳。
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.created_at, t.updated_at
		FROM forum_user_tag_assignments a JOIN forum_user_tags t ON t.id = a.tag_id
		WHERE a.user_email = ? ORDER BY t.name ASC`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]adminUserTag, 0)
	for rows.Next() {
		var tag adminUserTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	// 迭代層級的錯誤於此回傳，統一以 rows.Err() 收斂。
	return tags, rows.Err()
}

// loadAdminUserTagNames 讀出某位使用者已綁定的標籤「名稱」清單（依名稱排序）。
//
// 與 loadAdminUserTags 的差別：後者回傳完整的 adminUserTag（含 id 與時間戳），
// 供前端渲染；這個只給稽核用，且刻意放在刪除既有綗定**之前**呼叫。
// 稽核紀錄要保存的是「這個人被標成什麼」，而標籤名稱比 id 有意義得多 ——
// 稽核紀錄的讀者是人，而人看到 id=7 只會去猜那是什麼。
//
// q 參數刻意接受查詢介面而非固定用 s.db：稽核的「改動前」讀取必須在
// 同一個交易裡，否則在並發的標籤覆寫下會讀到別人的結果。
func loadAdminUserTagNames(ctx context.Context, q queryer, email string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT t.name
		FROM forum_user_tag_assignments a JOIN forum_user_tags t ON t.id = a.tag_id
		WHERE a.user_email = ? ORDER BY t.name ASC`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := make([]string, 0, 4)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// queryer 是 *sql.DB 與 *sql.Tx 都滿足的查詢介面。
//
// 存在的理由與 audit.Execer 相同：讓「要嘛在交易內讀、要嘛在交易外讀」的
// 函式不必為兩種接收者各寫一份。
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// decodeAdminUserEmail 將路徑中的 email 片段還原為原始信箱字串。
// 之所以需要，是因為 email 含有「@」與「.」，前端可能以 encodeURIComponent 編碼後
// 放進 URL（例如 %40 代表 @）；同時容忍多餘的尾端斜線。
// 回傳 *requestError（定義於 forum_admin_handlers.go）而非通用 error，
// 讓呼叫端能把訊息原樣回給前端。
func (s *Server) decodeAdminUserEmail(raw string) (string, error) {
	email, err := url.PathUnescape(strings.Trim(raw, "/"))
	if err != nil || email == "" {
		return "", &requestError{message: "invalid user email"}
	}
	return email, nil
}

// handleAdminUserContent 取得單一使用者的全部文章與全部留言，供後台就地編輯。
//
//	GET /api/admin/users/{email}/content -> 200 adminUserContent {"posts": [...], "comments": [...]}
//
// 兩個查詢彼此獨立，且各自依 created_at DESC, id DESC 排序（id 作為同秒資料的穩定排序鍵）。
// 附圖與文章列表共用同一套組網址規則（forumImageURL），因此此處不會把庫中的
// 檔名或內部位址直接送給瀏覽器。
// 權限：由呼叫端 handleAdminUser 完成管理員檢查。
func (s *Server) handleAdminUserContent(w http.ResponseWriter, r *http.Request, rawEmail string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	email, err := s.decodeAdminUserEmail(rawEmail)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	// 預先建立空 slice，確保沒有文章/留言時 JSON 是 [] 而不是 null。
	result := adminUserContent{Posts: make([]adminForumPost, 0), Comments: make([]adminForumComment, 0)}

	// ---- 查一：此使用者的文章 ----
	// 按讚數與留言數以子查詢即時統計，因為後台也能編輯留言、更新後需立刻看到正確數字。
	postRows, err := s.db.QueryContext(r.Context(), `
		SELECT id, author_email, content, created_at, image_url,
		       (SELECT COUNT(*) FROM forum_post_likes WHERE post_id = forum_posts.id),
		       (SELECT COUNT(*) FROM forum_post_comments WHERE post_id = forum_posts.id)
		FROM forum_posts WHERE author_email = ? ORDER BY created_at DESC, id DESC`, email)
	if err != nil {
		internalError(w, "unable to load user posts")
		return
	}
	// 這裡刻意不用 defer Close：第一組結果必須在開啟第二組之前關閉，
	// 以免同時持有兩組 rows 而佔滿連線池，因此於每個結束分支手動 Close。
	for postRows.Next() {
		var post adminForumPost
		if err := postRows.Scan(&post.ID, &post.AuthorEmail, &post.Content, &post.CreatedAt, &post.ImageURL, &post.LikeCount, &post.CommentCount); err != nil {
			postRows.Close()
			internalError(w, "unable to read user posts")
			return
		}
		result.Posts = append(result.Posts, post)
	}
	if err := postRows.Err(); err != nil {
		postRows.Close()
		internalError(w, "unable to load user posts")
		return
	}
	postRows.Close()

	// ---- 附圖網址 ----
	// 庫中只存檔名，後台要顯示縮圖就必須換成公開網址並附上短期 token。
	// 整份清單共用一個 token（理由同 handleAdminForumPosts：每張圖各簽一支
	// 會讓 Redis key 數量等同文章數），且整份都沒有圖時完全不碰 Redis。
	// 逐筆組網址放在迴圈外統一處理：token 要等掃描結束、Rows 關閉後才能取得。
	if s.firstForumImageName(result.Posts) != "" {
		mediaToken, err := s.createMediaToken(r.Context())
		if err != nil {
			// 與列表端一致：以 502 表達「上游（Redis）故障」，而非文章讀取失敗。
			logger.ErrorfContext(r.Context(), "[FORUM] 建立圖片 token 失敗: %v", err)
			writeError(w, http.StatusBadGateway, "unable to create media token")
			return
		}
		for i := range result.Posts {
			result.Posts[i].ImageURL = s.forumImageURL(result.Posts[i].ImageURL, mediaToken)
		}
	} else {
		// 沒有任何一張可用的圖：把庫值清空，避免把檔名或內部位址直接送給瀏覽器。
		for i := range result.Posts {
			result.Posts[i].ImageURL = ""
		}
	}

	// ---- 查二：此使用者的留言 ----
	commentRows, err := s.db.QueryContext(r.Context(), `
		SELECT id, post_id, author_email, content, created_at
		FROM forum_post_comments WHERE author_email = ?
		ORDER BY created_at DESC, id DESC`, email)
	if err != nil {
		internalError(w, "unable to load user comments")
		return
	}
	// 第二組結果會一直用到函式結尾，直接 defer 即可。
	defer commentRows.Close()
	for commentRows.Next() {
		var comment adminForumComment
		if err := commentRows.Scan(&comment.ID, &comment.PostID, &comment.AuthorEmail, &comment.Content, &comment.CreatedAt); err != nil {
			internalError(w, "unable to read user comments")
			return
		}
		result.Comments = append(result.Comments, comment)
	}
	if err := commentRows.Err(); err != nil {
		internalError(w, "unable to read user comments")
		return
	}
	writeOK(w, result)
}

// createAdminUserPost 以「指定使用者」的身分代為發文（後台代操作功能）。
//
//	POST /api/admin/users/{email}/posts -> 201 {"ok": true, "id": 123}，body: {"content": "..."}
//
// 內容驗證沿用一般發文的規則：去除前後空白、非空、最多 10000 字。
// image_url 固定寫入空字串，後台目前不支援代為上傳圖片。
// 權限：管理員（由 handleAdminUser 檢查）+ 必須通過 isTrustedOrigin 來源檢查。
func (s *Server) createAdminUserPost(w http.ResponseWriter, r *http.Request, rawEmail string) {
	// 寫入端點一律先擋掉跨來源請求。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	email, err := s.decodeAdminUserEmail(rawEmail)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	var req adminForumPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// validateAdminForumPost 會先就地 TrimSpace 再檢查長度，故錯誤訊息可直接回傳前端。
	if err := validateAdminForumPost(&req); err != nil {
		badRequest(w, err.Error())
		return
	}
	// author_email 使用路徑中的目標 email，而非目前的管理員身分 — 這正是此功能的目的。
	createdAt := time.Now()
	// 稽核：記 user.posts.create，target 放在「被代發的使用者」而不是文章 ——
	// 見 audit_log.go 的常數說明。targetLabel 帶上文章編號與內容前綴，讓稽核
	// 紀錄不必另外查一次 forum_posts 才知道這是什麼。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to create user post")
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO forum_posts (author_email, content, image_url, created_at) VALUES (?, ?, '', ?)`,
		email, req.Content, createdAt)
	if err != nil {
		internalError(w, "unable to create user post")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to read created user post")
		return
	}
	if err := s.recordAdminAction(r, tx, adminActionUserPost, audit.TargetUser, email, email,
		audit.Change{Field: "postId", Before: "", After: strconv.FormatInt(id, 10)},
		audit.Change{Field: "authorEmail", Before: "", After: email},
		audit.Change{Field: "content", Before: "", After: req.Content}); err != nil {
		internalError(w, "unable to create user post")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to read created user post")
		return
	}
	// 搜尋索引（best-effort，失敗只記日誌）。這條路徑寫入的作者是目標使用者本人，
	// 因此索引裡的 authorEmail 必須跟著是對方 —— 後臺的「以信箱找出所有貼文」
	// 才不會漏掉代發的那些。刻意放在 Commit 之後（索引是外部系統）。
	s.indexForumPost(r.Context(), id, req.Content, email, createdAt)
	// 回傳新文章 id，讓前端可立即在畫面中插入該筆資料。
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
}

// createAdminUserComment 以「指定使用者」的身分代為留言（後台代操作功能）。
//
//	POST /api/admin/users/{email}/comments -> 201 {"ok": true, "id": 456}
//	                                        body: {"postId": 12, "content": "..."}
//
// 留言內容沿用一般留言規則（非空、最多 2000 字）；寫入前會確認目標文章仍存在，
// 避免透過已刪除的文章 id 產生孤兒留言。
// 權限：管理員（由 handleAdminUser 檢查）+ 必須通過 isTrustedOrigin 來源檢查。
func (s *Server) createAdminUserComment(w http.ResponseWriter, r *http.Request, rawEmail string) {
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	email, err := s.decodeAdminUserEmail(rawEmail)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	var req adminForumCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	if err := validateAdminForumComment(&req); err != nil {
		badRequest(w, err.Error())
		return
	}
	// 檢查目標文章存在性：COUNT 為 0 代表文章已被刪除，回 404 而非讓 INSERT 留下孤兒資料。
	var postCount int
	if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_posts WHERE id = ?`, req.PostID).Scan(&postCount); err != nil {
		internalError(w, "unable to load forum post")
		return
	}
	if postCount == 0 {
		http.NotFound(w, r)
		return
	}
	// 稽核：記 user.comments.create（理由同 createAdminUserPost）。留言與文章
	// 不同，這裡額外記下 postId —— 同一個管理員可以在不同文章下代留言，
	// 少了它就答不出「這句話是回應哪一篇」。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to create user comment")
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO forum_post_comments (post_id, author_email, content, created_at) VALUES (?, ?, ?, ?)`,
		req.PostID, email, req.Content, time.Now())
	if err != nil {
		internalError(w, "unable to create user comment")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to read created user comment")
		return
	}
	if err := s.recordAdminAction(r, tx, adminActionUserComment, audit.TargetUser, email, email,
		audit.Change{Field: "commentId", Before: "", After: strconv.FormatInt(id, 10)},
		audit.Change{Field: "postId", Before: "", After: strconv.FormatInt(req.PostID, 10)},
		audit.Change{Field: "authorEmail", Before: "", After: email},
		audit.Change{Field: "content", Before: "", After: req.Content}); err != nil {
		internalError(w, "unable to create user comment")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to read created user comment")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
}
