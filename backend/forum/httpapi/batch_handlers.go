package httpapi

/*
批次操作（httpapi/batch_handlers.go）。

兩支端點：批次套標籤、批次改帳號狀態。它們存在的理由是「一個一個點」的
成本 —— 清理一批垃圾帳號要送 N 次請求、管理員在瀏覽器裡 N 次按鈕與 N 次確認。

三個必須成立的性質

	1. 每個受影響的對象各自記一筆稽核
		「一次停權 50 個帳號」若只記一筆，稽核紀錄就答不出「這個 email 什麼
		時候被停權的」—— 而那正是稽核紀錄存在的理由。因此這個檔案裡的每個
		受影響者都呼叫一次 recordAdminAction。若使用者後來要回答「我為什麼被
		停權」，查到的是他自己那筆，而不是一筆含 50 個 email 的彙總。

	2. 全部成功或全部不動
		整批放在**一個**交易裡。批次停權若在第 30 個帳號失敗，那麼前 29 個已經
		停權、第 30 個之後沒動 —— 管理者看到的是一個部分套用的狀態，而且沒有
		任何地方記錄「哪 29 個成功、哪 1 個失敗」。用單一交易讓這種狀態不可能
		出現。

		代價是交易會比較大（200 個 UPDATE + 200 筆 INSERT），而 InnoDB 會把
		它持有的鎖保留到提交。這個規模遠低於值得擔心的門檻（200 列的 UPDATE
		大約是毫秒級），因此「原子性」的價值高於交易大小的成本。

	3. 已知的 email 與不存在的 email 必須能區分
		批次列表裡混入一個不存在的 email（匯出的舊名單、已被刪的帳號）是常見
		的。整批回 400 會讓使用者不知道是哪一個錯；整批忽略則會讓他以為都成功
		了。因此回應帶 skipped 清單，逐項列出並附原因。

	4. 兩支端點都要有防禦性上限，而且是**兩個**：emails 有 batchMaxEmails、
		tagIds 有 batchMaxTags。批次標籤的寫入迴圈是巢狀的
		（email × tag），只擋其中一維等於沒擋 —— 詳見兩個常數的說明。

刻意不做的事

	- 沒有一個「把這個標籤加給所有人」之類的語意。批次標籤採「覆寫」語意
	  （與單一使用者的 PUT 相同）：送來的清單就是結果。理由是「加標籤」與
	  「設定標籤」在後臺的其他操作裡已經有一致的對應（見 handleAdminUserTags），
	  批次不該是例外 —— 兩個語意相近但不同的操作並存，使用者一定會猜錯其中
	  一個，而猜錯的那個是「加」對「設」，後果是使用者無預期地失去標籤。
	- 不把 batchMaxEmails 當成「太大就靜默截斷」。超過時回 400 並說明上限：
	  一個只做了前 200 個的批次，看起來和「全部成功」一模一樣。
*/

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"forum/forum/audit"
	"forum/forum/logger"
)

// batchMaxEmails 是單次批次允許的 email 數。
//
// 後端把這當作**防禦性上限**（一個惡意請求可以送 10 萬個 email 讓交易撐爆
// 記憶體），前端把它當作「使用者的合理上限」並在超過時提示。兩者共用同一個
// 常數，是為了讓前後端對「太大」的判斷一致 —— 兩份數字各自設定的話，會出現
// 「前端以為送得完、後端回 400」而前端不知道該怎麼解釋。
const batchMaxEmails = 200

// batchMaxTags 是單次批次可套用的標籤數。
//
// 這個上限與 batchMaxEmails 同屬**防禦性上限**，而理由完全一樣：寫入迴圈是
// 巢狀的（每個 email × 每個 tag 一列 INSERT），所以只限制 emails 是不夠的 ——
// `[1,2,3,…]` 每個 id 兩個位元組，16 KiB 的 body 上限剛好塞得下約 8,000 個
// id，而那會讓單一交易產生 200 × 8000 = 160 萬次單列 INSERT，所有 row lock
// 持有到 Commit。這條路由又刻意不掛限流，因此那是一次可被重複發動的資源耗盡。
//
// 20 的取捨與 audit.maxChangesPerEntry 同量級：後臺的標籤字典是給人維護的
// 有限清單，「一次給 200 個人套 20 個標籤」已經涵蓋任何合理的清理作業。
// 超過時回 400 而不是靜默截斷 —— 理由與 batchMaxEmails 相同。
const batchMaxTags = 20

// batchSkipped 記錄一個被跳過的 email 與原因。
//
// 逐項帶原因（而不是一個總數）是這個結構存在的理由：管理員需要知道
// 「跳過的 4 個是因為帳號不存在，還是因為格式錯誤」—— 兩者的處理方式
// 完全不同（前者是資料過期，後者是前端送錯）。
type batchSkipped struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

// batchResult 是批次端點的回應。
//
// Counts 是實際被改動的數量，Skipped 是被跳過的清單。兩者相加不等於
// Requested 是正常的（因為重複的 email 會被去重），因此回應帶上 requested
// 讓管理者能對帳，而不是靠猜。
type batchResult struct {
	OK        bool           `json:"ok"`
	Requested int            `json:"requested"`
	Counts    map[string]int `json:"counts"`
	Skipped   []batchSkipped `json:"skipped"`
}

/* ==========================================================================
   email 清單的解析與驗證
   ========================================================================== */

// normalizeBatchEmails 去重、修剪空白並檢查格式，回傳乾淨的清單。
//
// 回傳錯誤訊息一律是可以直接顯示給管理員的繁體中文：這個端點的呼叫者是
// 後臺的「批次」按鈕，而「有 3 個 email 不合法」這種回饋必須指出是哪 3 個
// —— 只說 "invalid request" 會讓他一個一個試。
//
// 保留輸入順序，因此 skipped 清單的順序與管理員在畫面上的勾選順序一致，
// 讓他能把被跳過的那幾個對回畫面。
func normalizeBatchEmails(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, errBatchNoSelection
	}
	if len(raw) > batchMaxEmails {
		return nil, errBatchTooLarge
	}

	seen := make(map[string]bool, len(raw))
	emails := make([]string, 0, len(raw))
	for _, item := range raw {
		email := strings.TrimSpace(item)
		if email == "" {
			continue
		}
		// 只檢查「有 @ 且沒有空白」。更嚴格的 email 格式驗證（RFC 5322）
		// 在這個情境下沒有價值：輸入來自畫面上已存在的帳號清單，格式一定
		// 正確；而放寬到「看起來像 email」反而能接受到某些真實存在但格式
		// 非標準的帳號。
		if !strings.Contains(email, "@") || strings.ContainsAny(email, " \t\r\n") {
			return nil, &requestError{message: "email 格式不正確：" + email}
		}
		// 長度上限與欄位寬度對齊（forum_users.email 是 VARCHAR(255)）。
		//
		// 沒有這一段時，一個 400 字元的「email」會一路走到稽核寫入，而那裡
		// target_id 是 VARCHAR(320) NOT NULL —— MySQL 回 error 1406，整批
		// 200 人的交易回滾並回 500。正確的答案是 400（輸入不合法），不是 500。
		//
		// 這裡必須自行擋住而不能只依賴 audit.Record 截斷 TargetLabel：同一個
		// 值在一欄被截斷、在另一欄不截的不對稱，本身就是這個 bug 的成因。
		//
		// 以**字元**計數而非位元組：VARCHAR(255) 的上限是字元（MySQL 對
		// utf8mb4 以字元計），而 len() 算的是位元組 —— 用它會讓一個 200 字的
		// CJK 字串（600 bytes）在欄位裡明明放得下，卻被這裡擋下並回一則寫著
		//「字元」的訊息。擋得比較嚴謹不是問題，單位不符才會浪費診斷時間。
		if utf8.RuneCountInString(email) > 255 {
			return nil, &requestError{message: "email 長度超過 255 字元：" + email}
		}
		if seen[email] {
			continue
		}
		seen[email] = true
		emails = append(emails, email)
	}
	if len(emails) == 0 {
		return nil, errBatchNoSelection
	}
	return emails, nil
}

// 批次請求的兩種固定錯誤。用變數而非每次 new，是為了讓 errBatchNoSelection
// 這個值可以被 == 比較（而 *requestError 每次都是不同指標）。
var (
	errBatchNoSelection = &requestError{message: "請至少選擇一個帳號"}
	errBatchTooLarge    = &requestError{message: "一次最多處理 200 個帳號"}
)

/* ==========================================================================
   批次標籤
   ========================================================================== */

// handleAdminBatchTags 批次覆寫多位使用者的標籤。
//
//	POST /api/admin/batch/tags  {"emails": ["a@b", "c@d"], "tagIds": [1, 2]}
//
// 語意是**覆寫**：每位存在且標籤合法的使用者，其綁定被換成 body 裡的清單。
// 傳空陣列即清除所有標籤 —— 與單一使用者的 PUT 相同（見 handleAdminUserTags）。
func (s *Server) handleAdminBatchTags(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	// 來源檢查在 method 分派之後：非 POST 的請求在這個端點沒有語意，
	// 因此先回 405 比較誠實（不支援的方法是「路由存在但方法不對」，
	// 不可信來源的 POST 才需要 403）。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	// 16 KiB 上限：200 個 email 平均 50 bytes 就是 10 KB。這個端點沒有
	// 任何理由收更多資料，因此擋掉灌水的 payload。
	var req struct {
		Emails []string `json:"emails"`
		TagIDs []int64  `json:"tagIds"`
	}
	if err := decodeLimitedJSON(w, r, &req, 16<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}

	emails, err := normalizeBatchEmails(req.Emails)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to update user tags")
		return
	}
	defer tx.Rollback()

	// 標籤 id 的驗證**在交易內**做（傳 tx 而不是 s.db）。
	//
	// 在交易外驗證會留下 TOCTOU：驗證通過、交易開啟之間，handleAdminTag 的
	// DELETE 可以把該標籤連同綁定一起刪掉；而 forum_user_tag_assignments
	// 沒有外鍵，因此後續的 INSERT 仍會成功，留下指向不存在標籤的孤兒綁定 ——
	// 那正是這個驗證存在的理由（見 handleAdminUserTags 的說明）。
	// 驗證與寫入在同一個交易裡，「標籤存在」就是寫入當下的事實。
	//
	// 驗證同時取回名稱，因為稽核紀錄要存「這個人的標籤從什麼變成什麼」，
	// 而名稱比 id 有意義得多。
	tagIDs, tagNameByID, err := validateBatchTagIDs(r, tx, req.TagIDs)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	// 依請求的 id 順序取回名稱（而不是 map 的迭代順序，那隨機）。
	//
	// 這個切片是「請求送來的順序」，也就是管理員在後臺點核取方塊的順序
	// （frontend/src/admin/provider.tsx 的 onToggleTag 是 append）。它只作為
	// 集合比對的輸入，不直接拿來組成稽核字串 —— 理由見 sameTagSet。
	tagNames := make([]string, 0, len(tagIDs))
	for _, id := range tagIDs {
		tagNames = append(tagNames, tagNameByID[id])
	}
	// 稽核紀錄的 diff 必須是穩定的 —— 同一個操作在兩次稽核裡應該看起來
	// 一樣，否則「比對兩筆紀錄」這件事就沒有意義了。這裡刻意用
	// joinTagNames（名稱排序）而不是請求順序：只有兩邊排序一致，
	// 下方的 sameTagSet 守衛與 audit_log.go 的 onlyChanged 才會對
	// 「同一組標籤」給出一致的答案。
	after := joinTagNames(tagNames)

	result := batchResult{OK: true, Requested: len(emails),
		Counts: map[string]int{"updated": 0, "unchanged": 0}, Skipped: []batchSkipped{}}

	for _, email := range emails {
		// 只問「帳號存不存在」，不取 status。批次標籤不修改 status，
		// 取回來只會讓讀者誤以為它和稽核的 Before 有關。
		var exists int
		if err := tx.QueryRowContext(r.Context(), `SELECT 1 FROM forum_users WHERE email = ?`, email).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				result.Skipped = append(result.Skipped, batchSkipped{Email: email, Reason: "帳號不存在"})
				continue
			}
			logger.ErrorfContext(r.Context(), "[BATCH] 讀取帳號失敗 email=%s: %v", email, err)
			internalError(w, "unable to update user tags")
			return
		}
		beforeNames, err := loadAdminUserTagNames(r.Context(), tx, email)
		if err != nil {
			internalError(w, "unable to update user tags")
			return
		}

		// 標籤集沒有變化時**什麼都不做**，也不記稽核 —— 與 handleAdminBatchStatus
		// 的 before == status 守衛同一個理由：稽核紀錄記的是「發生了什麼改變」。
		//
		// 這不是邊緣情況。MySQL 預設的 utf8mb4 排序規則大小寫不敏感，所以
		// A@B.com 與 a@b.com 在資料庫裡是同一列，卻是去重後 emails 切片裡的
		// 兩筆 —— 因此「對同一批使用者重複套用相同標籤」不需要任何花招就會觸發
		// 大量 Before == After 的紀錄，而那正好污染稽核紀錄存在的理由
		// （「這個帳號的標籤被動過幾次」）。audit_log.go 的 onlyChanged 是為此
		// 而存在的，這裡的守衛是同一件事在寫入端的版本。
		//
		// 比的是**集合**而不是字串：beforeNames 是名稱序（loadAdminUserTagNames
		// 的 ORDER BY t.name），tagNames 是請求順序。字串比對在兩者不同源時
		// 幾乎永遠不成立，症狀不是報錯，而是每個使用者都被重寫一次並留下一筆
		// `alpha, beta → beta, alpha` 的假變更紀錄 —— 正是這段註解要消除的東西。
		if sameTagSet(beforeNames, tagNames) {
			result.Counts["unchanged"]++
			continue
		}

		if _, err := tx.ExecContext(r.Context(),
			`DELETE FROM forum_user_tag_assignments WHERE user_email = ?`, email); err != nil {
			internalError(w, "unable to update user tags")
			return
		}
		// 單一多列 INSERT，而不是每個 (email, tag) 一列。
		//
		// 往返數因此從 O(emails × tags) 降到 O(1)：原本最壞情況是 200 × 8000
		// = 160 萬次 Exec（在同一個交易裡，全部鎖持有到 Commit），現在是每個
		// 使用者 1 次。佔位符數量由 len() 決定，不是使用者可控的結構字串，
		// 因此沒有注入面。
		if err := insertBatchTagAssignments(r, tx, email, tagIDs); err != nil {
			internalError(w, "unable to update user tags")
			return
		}
		// 每一個受影響的使用者各自一筆稽核（見檔頭第一點）。兩邊的名字都走
		// joinTagNames，因此 Before 與 After 都是名稱序 —— 字串相同就真的代表
		// 標籤集沒變。
		if err := s.recordAdminAction(r, tx, adminActionUserTags, audit.TargetUser, email, email,
			audit.Change{Field: "tags", Before: joinTagNames(beforeNames), After: after}); err != nil {
			internalError(w, "unable to update user tags")
			return
		}
		result.Counts["updated"]++
	}

	if err := tx.Commit(); err != nil {
		internalError(w, "unable to update user tags")
		return
	}
	writeOK(w, result)
}

// insertBatchTagAssignments 用單一多列 INSERT 寫入一位使用者的全部標籤綁定。
//
// tagIDs 為空時不發任何陳述句：清空標籤是由呼叫端的前一個 DELETE 達成的，
// 而「INSERT ... VALUES ()」在 MySQL 上是語法錯誤 —— 因此這個空集合分支
// 不是最佳化，是必要條件。
func insertBatchTagAssignments(r *http.Request, tx *sql.Tx, email string, tagIDs []int64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	// 每列 2 個佔位符。數量由 len(tagIDs) 決定（上限 batchMaxTags = 20，
	// 也就是最多 40 個佔位符），因此不是使用者可控的結構字串，沒有注入面。
	var query strings.Builder
	query.WriteString(`INSERT INTO forum_user_tag_assignments (user_email, tag_id) VALUES `)
	args := make([]any, 0, len(tagIDs)*2)
	for i, id := range tagIDs {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString("(?,?)")
		args = append(args, email, id)
	}
	_, err := tx.ExecContext(r.Context(), query.String(), args...)
	return err
}

// validateBatchTagIDs 驗證一組標籤 id，回傳去重後的清單與 id→名稱對照。
//
// 存在的理由是把「驗證 + 取名」綁在一起：稽核紀錄需要名稱，而名稱只能在
// 驗證通過之後安全地取（未驗證的 id 可能不存在，Scan 會失敗）。把它們分成
// 兩支函式的話，其中一支會被迫容忍「查不到」的情形，而那正是我們要擋的。
//
// q 接受 queryer 而非固定 *sql.DB：呼叫端必須傳交易（見 handleAdminBatchTags
// 對 TOCTOU 的說明），而這個檔案另外兩處呼叫需要非交易版本 —— 用介面讓
// 「在交易內讀」成為型別上的事實，而不是靠呼叫端記得傳對。
func validateBatchTagIDs(r *http.Request, q queryer, raw []int64) ([]int64, map[int64]string, error) {
	// 長度檢查放在重複檢查之前：重複去重後可能遠小於原長度，而防禦性上限要
	// 對「請求送了多少東西」生效，不是對「去重後剩多少」。
	if len(raw) > batchMaxTags {
		return nil, nil, &requestError{message: "一次最多套用 " + strconv.Itoa(batchMaxTags) + " 個標籤"}
	}
	tagNameByID := make(map[int64]string, len(raw))
	tagIDs := make([]int64, 0, len(raw))
	for _, id := range raw {
		if id <= 0 {
			return nil, nil, &requestError{message: "標籤 id 必須是正數"}
		}
		if _, exists := tagNameByID[id]; exists {
			// 重複會讓後面的 INSERT 撞上 (user_email, tag_id) 複合主鍵，
			// 錯誤訊息會是看不出原因的唯一鍵衝突。因此在這裡擋掉。
			return nil, nil, &requestError{message: "標籤 id 重複"}
		}
		tagNameByID[id] = ""
		tagIDs = append(tagIDs, id)
	}
	if len(tagIDs) == 0 {
		// 空的 tagIds 是合法的（清除所有標籤），因此不報錯。
		return tagIDs, tagNameByID, nil
	}
	// 一次查出所有名稱。逐個查的話是 N 次往返，而 N ≤ batchMaxTags。
	placeholders := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
	args := make([]any, len(tagIDs))
	for i, id := range tagIDs {
		args[i] = id
	}
	rows, err := q.QueryContext(r.Context(),
		`SELECT id, name FROM forum_user_tags WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	found := 0
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		tagNameByID[id] = name
		found++
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if found != len(tagIDs) {
		// 有 id 不存在。指出是哪一個 —— 逐字比對是為了找出那個沒被查到的。
		for _, id := range tagIDs {
			if tagNameByID[id] == "" {
				return nil, nil, &requestError{message: "標籤 id 不存在：" + strconv.FormatInt(id, 10)}
			}
		}
		return nil, nil, &requestError{message: "標籤 id 不存在"}
	}
	return tagIDs, tagNameByID, nil
}

/* ==========================================================================
   批次帳號狀態
   ========================================================================== */

// handleAdminBatchStatus 批次改多位使用者的帳號狀態。
//
//	POST /api/admin/batch/status  {"emails": ["a@b"], "status": "SUSPENDED"}
//
// 停權與恢復走同一支端點（與單一使用者的 PATCH 相同）：動作是狀態的函數，
// 分成兩條路由只會讓稽核紀錄的動作名稱再多一種拼法。
func (s *Server) handleAdminBatchStatus(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Emails []string `json:"emails"`
		Status string   `json:"status"`
	}
	if err := decodeLimitedJSON(w, r, &req, 16<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}

	emails, err := normalizeBatchEmails(req.Emails)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	// 與單一使用者路徑相同的正規化與白名單（handleAdminUser 的 PATCH 分支）。
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status != "ACTIVE" && status != "SUSPENDED" {
		badRequest(w, "invalid user status")
		return
	}

	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to update user")
		return
	}
	defer tx.Rollback()

	result := batchResult{OK: true, Requested: len(emails), Counts: map[string]int{"updated": 0, "unchanged": 0}, Skipped: []batchSkipped{}}
	action := adminActionUserReinstate
	if status == "SUSPENDED" {
		action = adminActionUserSuspend
	}

	for _, email := range emails {
		var before string
		if err := tx.QueryRowContext(r.Context(), `SELECT status FROM forum_users WHERE email = ?`, email).Scan(&before); err != nil {
			if err == sql.ErrNoRows {
				result.Skipped = append(result.Skipped, batchSkipped{Email: email, Reason: "帳號不存在"})
				continue
			}
			logger.ErrorfContext(r.Context(), "[BATCH] 讀取帳號失敗 email=%s: %v", email, err)
			internalError(w, "unable to update user")
			return
		}
		if before == status {
			// 已經是目標狀態。不記稽核 —— 稽核紀錄記的是「發生了什麼改變」，
			// 而「把 SUSPENDED 設成 SUSPENDED」什麼都沒改。記下來只會讓
			// 「這個帳號被停權過幾次」這個問題的答案失真。
			result.Counts["unchanged"]++
			continue
		}
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE forum_users SET status = ?, updated_at = ? WHERE email = ?`,
			status, time.Now(), email); err != nil {
			internalError(w, "unable to update user")
			return
		}
		if err := s.recordAdminAction(r, tx, action, audit.TargetUser, email, email,
			adminStatusChange(before, status)); err != nil {
			internalError(w, "unable to update user")
			return
		}
		result.Counts["updated"]++
	}

	if err := tx.Commit(); err != nil {
		internalError(w, "unable to update user")
		return
	}
	writeOK(w, result)
}

// decodeLimitedJSON 以大小上限解碼 JSON body。
//
// 包 MaxBytesReader 是為了擋掉灌水：這兩個端點的 body 只有 email 與幾個
// 數字，16 KiB 遠超合理需求。錯誤一律回同一個訊息 —— 觸及上限的錯誤
// （"http: request body too large"）不該被回給使用者，因為那洩漏了實作細節，
// 而且對「你的請求太大」這個事實沒有任何幫助。
func decodeLimitedJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(dst)
}
