package httpapi

/*
Session 管理（httpapi/session_admin_handlers.go）。

兩支端點：列出目前活躍的 session、強制登出。它們回答的是後臺原本答不出來的兩個
問題：「現在有誰在線上」「把某個人的所有裝置都登出來」。

三個與其他後臺端點不同的地方

	1. **強制登出的稽核寫在 MySQL，而動作在 Redis**
	   稽核紀錄在 MySQL、session 在 Redis，而兩個資料庫之間沒有交易可以橫跨
	   它們。因此這個操作**不可能**像其他後臺寫入那樣「操作與稽核同生共死」。

	   這裡選擇的方向是：先刪 Redis（動作），再寫稽核（紀錄）。理由是
	   「動手之後忘了記錄」比「記錄了但還沒動手」安全 —— 後者會讓稽核紀錄
	   宣稱「已撤銷」而 session 還活著，那是一個**不實的紀錄**，而稽核紀錄的
	   全部價值就在於它是真實的。

	   稽核寫入失敗時回 500 並在訊息裡說明「session 已撤銷但紀錄失敗」。
	   這會讓管理員困惑（他會想重試，而重試會顯示「已撤銷 0 筆」），但那個
	   困惑好過一個不實的稽核紀錄。已在程式碼註解中寫明。

	2. **不回傳完整 token**
	   token 就是憑證（見 session/session_list.go 檔頭）。回應裡只有前 8 個
	   字元，因此「管理員截圖分享畫面」不會變成一次完整的手法移交。

	3. **掃描可能被截斷，而截斷必須被告知**
	   掃描成本是 O(session 總數)，因此後端有一個 key 數上限（見 session_list.go
	   的說明）。達到上限時回應帶 truncated: true，介面必須把它顯示出來 ——
	   「沒列出來」若沒有被標示成「沒掃完」，看起來就會像「不存在」。

刻意不做的事

	- 不提供「登出所有 session」（包含管理員自己的）。那是一個非常容易誤按的
	  按鈕，而且按下之後管理員自己也被踢出，症狀是「我按了登出全部，結果
	  我也登出了，而且沒有辦法再進來登出所有人」。單一使用者的強制登出已經
	  覆蓋了實際的資安需求（處理帳號被盜用）。
	- 不顯示 IP 與 User-Agent。session hash 裡沒有存這兩項，而為了顯示它們
	  就得新增欄位 —— 那是為了診斷而擴大憑證的儲存面。token 不綁 IP/UA 這件事
	  本身就是 session.go 檔頭列出的已知限制，補上這兩項不會讓它消失。
*/

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/audit"
	"forum/forum/logger"
	"forum/forum/session"
)

// adminSessionView 是一支 session 對外的樣子。
//
// 刻意不直接回傳 session.Record：那個型別帶 Token（完整憑證），而這個 view
// 只有 TokenPrefix。用兩層型別而不是「序列化時記得換掉」是為了讓漏掉這件事
// 變成編譯期錯誤 —— 少一個欄位不會報錯，但少換一個 token 會造成一次憑證外洩。
type adminSessionView struct {
	Email   string `json:"email"`
	IsAdmin bool   `json:"isAdmin"`
	// TokenPrefix 是 token 的前 8 個字元（後接省略號）。用來讓管理員分辨
	// 「這是不是同一支 session」，不能拿來還原。
	TokenPrefix string `json:"tokenPrefix"`
	// CreatedAt 是建立時間；空字串代表「既有 session，沒有記錄」。
	CreatedAt string `json:"createdAt"`
	// ExpiresAt 是「現在 + TTL」，也就是真正會失效的時間點。
	ExpiresAt string `json:"expiresAt"`
	// TTLSeconds 是剩餘秒數；負值代表 Redis 沒有回報 TTL（理論上不會發生）。
	TTLSeconds int64 `json:"ttlSeconds"`
}

// adminSessionsPayload 是 GET /api/admin/sessions 的回應。
type adminSessionsPayload struct {
	OK bool `json:"ok"`
	// Items 是符合條件的 session（已套用 limit）。
	Items []adminSessionView `json:"items"`
	// TotalActive 是掃描期間數到的 session 總數。Items 與它不同是因為
	// 查詢條件與 limit —— 介面用它顯示「符合 3 筆，全站共 12 筆」。
	TotalActive int `json:"totalActive"`
	// Scanned 是檢查過的 key 數。顯示它讓「掃描很慢」變成一個可以被解釋的
	// 數字，而不是一個沒有上下文的等待。
	Scanned int `json:"scanned"`
	// Truncated 為 true 代表掃描達到上限而提前放棄，Items **不完整**。
	Truncated bool `json:"truncated"`
	// ExpireInHours 是 session 的存續時間（小時）。配合滑動續期即表示
	// 「連續 N 小時沒有活動才會被登出」，因此欄位旁必須有那句解釋 ——
	// 使用者看到「剩 3 小時」時會以為 3 小時後被登出，而實際上只要有活動
	// 就會一直延續。
	ExpireInHours int64 `json:"expireInHours"`
}

// handleAdminSessions 列出目前活躍的 session。
//
//	GET /api/admin/sessions?email=&limit=
//
// email 是精確比對（不是關鍵字搜尋）：session 裡沒有任何可模糊比對的欄位，
// 而「包含比對」在 email 上會產生一個很難解釋的結果（比到誰算誰）。
func (s *Server) handleAdminSessions(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	query := r.URL.Query()
	result, err := s.sessions.List(r.Context(), session.ListOptions{
		Email: strings.TrimSpace(query.Get("email")),
		Limit: atoiDefault(query.Get("limit"), 0),
	})
	if err != nil {
		logger.ErrorfContext(r.Context(), "[SESSION] 列出失敗: %v", err)
		internalError(w, "unable to list sessions")
		return
	}

	// 最新的排在最前面。掃描回來的順序是 Redis 內部的 key 順序，與時間無關
	// —— 而管理員找的是「剛剛登入的那一支」。
	items := make([]adminSessionView, 0, len(result.Sessions))
	for _, record := range result.Sessions {
		items = append(items, adminSessionView{
			Email:       record.Email,
			IsAdmin:     record.IsAdmin,
			TokenPrefix: record.TokenPrefix,
			CreatedAt:   formatSessionTime(record.CreatedAt),
			ExpiresAt:   formatSessionTime(record.ExpiresAt),
			TTLSeconds:  int64(record.TTL.Seconds()),
		})
	}
	sortSessionsByExpiry(items)

	writeOK(w, adminSessionsPayload{
		OK:            true,
		Items:         items,
		TotalActive:   result.TotalActive,
		Scanned:       result.Scanned,
		Truncated:     result.Truncated,
		ExpireInHours: int64(result.ExpireIn.Hours()),
	})
}

// handleAdminRevokeSessions 強制登出：刪除某個 email 的所有 session。
//
//	POST /api/admin/sessions/revoke  {"email": "user@example.com"}
//
// 沒有「確認」參數：這個操作不可逆（使用者必須重新走 Google 登入），而確認
// 流程屬於介面。需要的話介面可以用 <dialog> 先問一次 —— 實際上 UsersPage 就是
// 那樣做的。
func (s *Server) handleAdminRevokeSessions(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	// 來源檢查：這是狀態改變，瀏覽器會自動附上 session cookie，只有來源檢查
	// 能確認這個 POST 真的來自後臺頁面（理由同其他後臺寫入端點）。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := decodeLimitedJSON(w, r, &req, 4<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}
	email := strings.TrimSpace(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		badRequest(w, "invalid user email")
		return
	}

	// 1) 先做動作。Redis 與 MySQL 沒有共同交易（見檔頭第一點），因此這個順序
	//    是刻意的：寧可「動手之後忘了記錄」，也不要「記錄了但還沒動手」。
	revoked, err := s.sessions.RevokeByEmail(r.Context(), email)
	if err != nil {
		// 掃描被截斷時 RevokeByEmail 會回錯（它不能確認是否已全部撤銷）。
		// 這裡回 409 而不是 500：使用者重試很可能就掃得完（截斷是暫時性的
		// session 數量問題），而 409 的語意「目前無法完成，請再試」比 500
		// 的「伺服器壞了」準確。
		logger.ErrorfContext(r.Context(), "[SESSION] 撤銷失敗 email=%s: %v", email, err)
		writeError(w, http.StatusConflict, "無法確認是否已撤銷全部 session，請稍後再試")
		return
	}

	// 2) 再寫稽核。
	//
	//    這裡**必須**開一個交易，即使沒有任何 MySQL 資料要改。理由是
	//    recordAdminAction 在 tx 為 nil 時會回 nil 並且**完全不寫任何東西** ——
	//    那個 nil 容忍是為了讓「稽核表還沒建好」時後臺仍可操作，但用在這裡
	//    就會讓強制登out這個動作永遠沒有稽核紀錄。因此寧可多開一個只為了
	//    寫一行的交易。
	//
	//    revoked 是實際刪掉的數量（不是掃描時看到的數量）—— 兩者不同是因為
	//    有 session 在掃描與刪除之間自然過期，而稽核紀錄必須記實際結果。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[SESSION] 已撤銷 %d 筆 session，但無法開始稽核交易 email=%s: %v",
			revoked.Revoked, email, err)
		writeError(w, http.StatusInternalServerError,
			"已撤銷 "+strconv.Itoa(revoked.Revoked)+" 筆 session，但稽核紀錄寫入失敗")
		return
	}
	defer tx.Rollback()

	if err := s.recordAdminAction(r, tx, adminActionSessionRevoke, audit.TargetUser, email, email,
		audit.Change{Field: "revokedSessions", Before: "", After: strconv.Itoa(revoked.Revoked)},
	); err != nil {
		// 見檔頭第一點：動作已經生效，因此這裡必須明確說明「已撤銷但沒紀錄」，
		// 而不能假裝整個操作失敗（那會讓管理員以為 session 還活著）。
		logger.ErrorfContext(r.Context(), "[SESSION] 已撤銷 %d 筆 session，但稽核寫入失敗 email=%s: %v",
			revoked.Revoked, email, err)
		writeError(w, http.StatusInternalServerError,
			"已撤銷 "+strconv.Itoa(revoked.Revoked)+" 筆 session，但稽核紀錄寫入失敗")
		return
	}
	if err := tx.Commit(); err != nil {
		logger.ErrorfContext(r.Context(), "[SESSION] 已撤銷 %d 筆 session，但稽核交易提交失敗 email=%s: %v",
			revoked.Revoked, email, err)
		writeError(w, http.StatusInternalServerError,
			"已撤銷 "+strconv.Itoa(revoked.Revoked)+" 筆 session，但稽核紀錄寫入失敗")
		return
	}

	writeOK(w, map[string]any{
		"ok":      true,
		"email":   email,
		"revoked": revoked.Revoked,
		"scanned": revoked.Scanned,
	})
}

// formatSessionTime 把時間格式化成 RFC3339；零值回空字串。
//
// 空字串而不是 "0001-01-01T00:00:00Z"：後者是一個看起來像真實時間的東西，
// 而介面會把它顯示成「西元 1 年」。空字串讓前端可以用「未知」來呈現 —
// 這是既有 session 沒有 created_at 欄位時唯一誠實的顯示方式。
func formatSessionTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

// sortSessionsByExpiry 把即將到期的排最前面。
//
// 用插入排序而不是 sort.Slice：Items 已經是管理員看得到的順序（依建立時間
// 遞增），而這個排序是「穩定地把即將到期者往前移」—— 同到期時間者維持
// 原順序，因此管理員重整前後看到的相對順序不會跳動。n ≤ 500，插入排序的
// O(n²) 在這個規模下是 25 萬次比較，遠小於掃描本身的成本。
func sortSessionsByExpiry(items []adminSessionView) {
	for i := 1; i < len(items); i++ {
		current := items[i]
		j := i - 1
		for j >= 0 && sessionExpiresBefore(items[j], current) {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = current
	}
}

// sessionExpiresBefore 判斷 a 是否比 b 更早到期。
//
// 空字串（未知到期時間）排在最後：它代表「這支 session 的剩餘時間查不到」，
// 把它排在最前面會讓使用者以為有一支馬上要失效的 session，而那個資訊是
// 不存在的。
func sessionExpiresBefore(a, b adminSessionView) bool {
	if a.ExpiresAt == "" {
		return false
	}
	if b.ExpiresAt == "" {
		return true
	}
	return a.ExpiresAt < b.ExpiresAt
}
