/*
認證流程的 HTTP 層：Google OAuth 回调、登出與身分查詢。

整體流程：
  1. GET /auth/google（auth.HandleLogin）把使用者導向 Google，並把原本想
     造訪的路徑塞進 OAuth 的 state 參數（已驗證為站內路徑）。
  2. GET /auth/callback（本檔 handleGoogleCallback）用 code 換 token、取出
     email，做停權檢查與 forum_users 落檔，簽發 session token 寫入 cookie，
     再依 state 把使用者送回原頁。
  3. GET /api/check 讓前端在載入時查詢目前身分；POST /api/logout 銷毀 session。

三個 handler 都遵循同一個錯誤路徑原則：對外只回固定字串（internalError
的訊息），把細節寫進日誌，避免把內部錯誤資訊洩漏給瀏覽器。
*/

package httpapi

import (
	"database/sql"
	"forum/forum/auth"
	"forum/forum/logger"
	"net/http"
	"time"
)

// handleGoogleCallback 是 OAuth 授權碼的回調端點，完成身分落地與 session 簽發。
//
// 成功路徑回 303 See Other 而非 302 Found：RFC 9110 對 302 只規定使用者代理
// 「得」把 POST 改成 GET，實作上各家不一；303 則明確要求下一個請求一定是
// GET。回調端點只需要重導向、不該重送任何 body，用 303 可省掉這個不確定性。
//
// 錯誤路徑一律以 JSON 錯誤回應（4xx/5xx）而非重導向：此時使用者手上沒有
// 有效的 session 狀態，導回登入頁只會造成重新觸發 OAuth 的無意義迴圈。
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// err 的文字確實會回傳給瀏覽器；可控的原因是 auth.GetUserEmail 回傳的是
	// 自己包裝的固定格式字串（如 "code exchange failed: ..."），而不是直接
	// 轉貼 Google 的原始回應內容。
	email, err := auth.GetUserEmail(r)
	if err != nil {
		internalError(w, "Google login failed: "+err.Error())
		return
	}

	// 管理員身分以設定檔（ALLOWED_ADMIN_EMAIL）為唯一真實來源，登入當下就
	// 寫進 session，後續請求不再回查設定，可避免每個請求都做一次檔案判斷。
	isAdmin := s.cfg.IsAdminEmail(email)
	// 這行日誌是「某個 email 拿到管理員權限」的唯一稽核紀錄，屬必要輸出。
	logger.InfofContext(r.Context(), "[AUTH] Google login callback: email=%s is_admin=%v", email, isAdmin)
	if !isAdmin {
		// 只有非管理員才檢查停權。管理員必須在停權機制之外，否則一旦誤被
		// 設為 SUSPENDED 就沒有人能把它改回來。
		var status string
		err := s.db.QueryRowContext(r.Context(), `SELECT status FROM forum_users WHERE email = ?`, email).Scan(&status)
		// 條件式（err == nil）判斷：查無此列時不視為錯誤，因為使用者可能
		// 是第一次登入，forum_users 尚未有他的資料，下面的 UPSERT 就會建立。
		if err == nil && status == "SUSPENDED" {
			unauthorized(w, "此帳號已被停權")
			return
		}
		// 真正的查詢錯誤必須擋下：無法確認狀態時若放行，等於讓停權機制
		// 因為資料庫故障而失效。
		if err != nil && err != sql.ErrNoRows {
			internalError(w, "unable to verify user status")
			return
		}
		now := time.Now()
		// UPSERT：首次登入建立紀錄、再次登入只更新 updated_at。
		// 刻意不更新 status —— 停權狀態不可被「登入」這個動作洗掉，
		// 否則使用者只要重新登入就能解除停權。
		if _, err := s.db.ExecContext(r.Context(), `
			INSERT INTO forum_users (email, status, created_at, updated_at) VALUES (?, 'ACTIVE', ?, ?)
			ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`, email, now, now); err != nil {
			internalError(w, "unable to record user")
			return
		}
	}

	// 簽發 Redis session token。失敗代表 Redis 不可用，此時寧可回 500 讓
	// 使用者重試，也不要發出一個無法驗證的空 session。
	token, err := s.sessions.Create(email, isAdmin)
	if err != nil {
		internalError(w, "unable to generate session")
		return
	}
	// cookie 為 HttpOnly + SameSite=Lax + Path=/，由 session.Manager 設定；
	// HttpOnly 讓 cookie 無法被 JavaScript 讀取，降低 XSS 盜用風險。
	s.sessions.SetCookie(w, token)
	// 目標路徑取自 OAuth 的 state（由 HandleLogin 寫入，並在 auth.ReturnPath
	// 再次以 isSafeReturnPath 驗證為站內路徑），因此不會是外部網址。
	// 驗證兩次是刻意的：state 由外部輸入，無法假設它一定安全。
	http.Redirect(w, r, auth.ReturnPath(r), http.StatusSeeOther)
}

// handleLogout 銷毀目前請求對應的 session 並清除瀏覽器 cookie。
//
// 掛在 requireTrustedOrigin 之下（見 server.go 的路由表）：單純清除 cookie
// 看似無害，但若允許跨站觸發，使用者在別的頁面放一個自動提交的表單就能被
// 強制登出。同一個中介層也被用於 handleForumReport，兩者共用同一份
// TrustedOrigins 白名單。
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// 只接受 POST：GET 形式的登出連結會被瀏覽器預先載入、跨站圖片或
	// 郵件中的連結無預警觸發，等同於把上述 CSRF 風險重新打開。
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	// 順序有意義：先刪 Redis 端的 session，再清 cookie。反過來的話，若
	// 清除 cookie 成功但刪除失敗，使用者會拿著仍有效的 token 卻以為已登出。
	s.sessions.RemoveByRequest(r)
	s.sessions.ClearCookie(w)
	// 回應固定 {"ok":true} 形狀，讓前端 fetch 能用同一套解析路徑處理。
	writeOK(w, map[string]bool{"ok": true})
}

// handleCheck 供前端在載入時查詢目前身分，不改變任何狀態，因此只接受 GET。
//
// 這支 API 刻意不做 requireLogin：它正是用來回答「我登入了嗎」的介面，
// 若需要先登入才能查詢就毫無意義。未登入時回 200 + ok:false，而不是 401，
// 因為「未登入」對這個端點而言是正常結果而非錯誤。
func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	// 分開呼叫 IsLoggedIn 與 IsAdmin：前者確認 session key 仍存在於 Redis，
	// 後者只讀 session 裡的欄位。已登入但非管理員是合法狀態，必須一併回報。
	loggedIn := s.sessions.IsLoggedIn(r)
	isAdmin := s.sessions.IsAdmin(r)
	writeOK(w, map[string]bool{
		"ok":      loggedIn,
		"isAdmin": isAdmin,
	})
}
