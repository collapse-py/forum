package httpapi

/*
管理員操作的稽核接線（httpapi/audit_log.go）。

本檔案是 audit 套件與 HTTP handler 之間唯一的接線點，內容只有三樣東西：

  1. 動作名稱的有限集合（adminAction* 常數）
  2. recordAdminAction —— 從 *http.Request 取出操作者、IP、request ID，組出
     audit.Entry 交給 audit.Record
  3. handleAdminLog —— /api/admin/log 的查詢端點

為什麼動作名稱集中在這裡而不散落在各 handler

	名稱字串是稽核紀錄的對外契約：/admin/log 的篩選選項、前端的顏色判斷、
	以及三個月後回頭查資料時要輸入的字串，全都依賴它們穩定。若某個 handler
	自己拼一個 "user.suspended"（而另一處寫 "user.suspend"），沒有任何
	編譯期錯誤，只有「某一類操作查不到」的靜默失敗。因此名稱集中宣告成
	常數，且是「資源.動作」的形式 —— 資源那一段與 audit.TargetType 一致，
	所以從一筆紀錄就能看出它對應哪一類資源。

與其他 admin handler 的差異：這裡沒有寫入動作

	/api/admin/log 只有 GET。寫入稽核紀錄的是其他 handler，而那些寫入
	都必須在**自己的交易內**完成（見 audit.Record 的說明）—— 也就是說
	「稽核寫入失敗」會讓「操作」一起回滾。這是本設計最重要的性質，而它
	只有在使用 *sql.Tx 時才成立。
*/

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/audit"
	"forum/forum/logger"
)

/*
動作名稱。格式為 "<target_type>.<動作>"，前綴必須與 audit 的 TargetType
常數一致。唯一的例外是 adminActionSessionRevoke，理由寫在該常數的說明裡
（它是這個清單裡唯一一個 target 是「人」而動作作用在他所有 session 上的操作，
改成 user.sessions.revoke 會讓名稱長度超過一讀就懂的範圍，而前綴不一致的
成本只有「依 target_type 前綴分組」這個尚未存在的功能）。

命名規則：
  - 用動詞過去式（delete / update / resolve），不用命令式（deletePost）：
    紀錄是在描述「已發生的事」，讀起來是過去式才對。
  - 停權與恢復分成 suspend / reinstate 兩支，而不是一支帶方向的
    "user.status"：兩者的嚴重程度不同（停權會擋掉對方的所有寫入），
    而稽核紀錄的第一個用途就是快速分辨這類嚴重差異。
  - 代發文記在目標使用者名下（user.posts.create / user.comments.create）
    而不是在 post.create / comment.create：以管理員自己身分發的那條路徑才
    記 post.create。差別正是「稽核紀錄上的作者是不是操作者本人」，而這是
    代發文唯一需要被稽核的理由。
*/
const (
	adminActionUserSuspend   = "user.suspend"
	adminActionUserReinstate = "user.reinstate"
	adminActionUserTags      = "user.tags.update"
	adminActionUserPost      = "user.posts.create"
	adminActionUserComment   = "user.comments.create"
	adminActionUserContent   = "user.content.delete"

	adminActionPostCreate  = "post.create"
	adminActionPostUpdate  = "post.update"
	adminActionPostDelete  = "post.delete"
	adminActionCommentPost = "comment.create"
	adminActionCommentPut  = "comment.update"
	adminActionCommentDel  = "comment.delete"

	// adminActionPostIndexFailed 是「後臺的寫入成功了，但搜尋索引沒有同步」。
	//
	// 它與 session.revoke / ip.block 是同一類：動作發生在 MySQL 與 Elasticsearch
	// 兩個系統而紀錄寫在 MySQL，因此沒有「與操作同生共死」的保證 —— 索引更新
	// 刻意放在 Commit 之後（否則索引會指向被回滾的資料），而那意味著它失敗時
	// 操作已經無法回滾。
	//
	// 為什麼值得單獨記一筆：稽核紀錄是這個專案用來回答「使用者看到的是什麼」
	// 的依據。沒有這一筆時，刪文的稽核紀錄會完整地說「已刪除」，而搜尋結果裡
	// 還留著一個點進去是 404 的幽靈貼文 —— 兩邊對不起來，而且沒有任何地方
	// 留下差異。觸發手段不需要任何花招：讓 ES 暫時不可用即可。
	adminActionPostIndexFailed = "post.index_failed"

	adminActionReportResolve = "report.resolve"
	adminActionReportReject  = "report.reject"
	adminActionReportUpdate  = "report.update"
	adminActionReportDelete  = "report.delete"
	adminActionReportCreate  = "report.create"

	adminActionTagCreate = "tag.create"
	adminActionTagUpdate = "tag.update"
	adminActionTagDelete = "tag.delete"

	// adminActionSessionRevoke 是「強制登出某個帳號的所有 session」。
	//
	// 它是這個清單裡唯一一個**動作發生在 Redis 而記錄寫在 MySQL** 的操作，
	// 因此沒有「與操作同生共死」的保證。方向的取捨寫在
	// httpapi/session_admin_handlers.go 的檔頭；簡單說是「寧可少一筆紀錄，
	// 也不要一筆不實的紀錄」。
	//
	// target 用 user（被登出的人）而不是 session —— 稽核紀錄的問題是
	// 「誰被怎樣對待了」，而「這個人的 session 被全部撤銷」正是一個使用者
	// 會追問的問題。用 session 的話，那一筆紀錄就沒有可辨識的對象。
	//
	// 名稱因此是這個清單裡唯一一個前綴不等於 TargetType 的（檔頭的約定有
	// 記錄這個例外）。改成 user.sessions.revoke 會讓名稱多一個區段卻不增加
	// 任何資訊：稽核紀錄是按整串比對的（idx_forum_admin_actions_action 與
	// DISTINCT 篩選器都吃完整字串），前綴只在「依 target_type 分組查詢」
	// 那個尚未存在的功能上才有意義。
	adminActionSessionRevoke = "session.revoke"

	// IP 封鎖與解封。target 是 audit.TargetIP（那不是一個資料表的資源，而是
	// Redis 裡封鎖名單的識別值）。與 session.revoke 一樣，這兩個動作發生在
	// Redis 而紀錄寫在 MySQL，因此沒有「與操作同生共死」的保證，方向的取捨
	// 寫在 httpapi/blocklist.go 的 blockAddress。
	auditActionBlockAdd    = "ip.block"
	auditActionBlockRemove = "ip.unblock"
)

// recordAdminAction 記錄一次管理員操作，並把寫入結果原樣回傳。
//
// tx 必須是該操作自己的交易（*sql.Tx），不是 *sql.DB。這是「操作不可能
// 沒有稽核紀錄」的保證來源，而它有個容易被順手破壞的前提：Go 的
// database/sql 不會因為交易內某個語句失敗就自動中止交易。因此**呼叫端
// 必須在 Commit 之前檢查這裡回傳的錯誤**，失敗時讓交易回滾。只呼叫、
// 不看回傳值，等於把稽核降級成「盡力記錄」，那正是稽核日誌不可信的原因。
//
// tx 為 nil 時直接回 nil 而不寫入：寧可這次操作沒有紀錄，也不要寫一筆
// 與實際資料不一致的紀錄（一致性比完整性重要）。正式路徑上 tx 不會是 nil。
func (s *Server) recordAdminAction(r *http.Request, tx *sql.Tx, action, targetType, targetID, targetLabel string, changes ...audit.Change) error {
	if tx == nil {
		logger.WarnfContext(r.Context(), "[AUDIT] 略過紀錄（沒有交易可用）action=%s target=%s/%s", action, targetType, targetID)
		return nil
	}
	meta := logger.MetadataFromContext(r.Context())
	err := audit.Record(r.Context(), tx, audit.Entry{
		ActorEmail:  meta.UserEmail,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		TargetLabel: targetLabel,
		Changes:     changes,
		ClientIP:    s.clientIP(r),
		RequestID:   meta.RequestID,
		// 以字串而非 time.Time 寫入：DATETIME 欄位不帶時區，掃描回來
		// 也沒有時區。UTC 在寫入端就固定下來，顯示時再由前端轉換 ——
		// 若寫入端用本地時間，跨主機搬動資料庫之後所有紀錄都會錯 8 小時。
		CreatedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		logger.ErrorfContext(r.Context(), "[AUDIT] 寫入失敗（操作將被回滾）action=%s target=%s/%s: %v",
			action, targetType, targetID, err)
	}
	return err
}

// adminStatusChange 是「狀態欄位改變」的 Change。抽出來是因為停權與恢復
// 會在兩個分支裡用到同一個描述。
func adminStatusChange(before, after string) audit.Change {
	return audit.Change{Field: "status", Before: before, After: after}
}

// recordPostIndexFailure 在「後臺寫入成功但搜尋索引沒同步」時補記一筆稽核。
//
// 為什麼需要一個專用的記錄點：這時操作早已 Commit，回應也已即將送出，因此
// recordAdminAction 的「稽核與操作同生共死」前提在這裡**不成立**。而沒有紀錄
// 的後果不是「少一筆」而是「稽核紀錄說謊」—— 它會宣稱一次完整的刪文，而搜尋
// 結果裡留著幽靈貼文。
//
// 刻意寫成「開一個只為了寫這一行的交易」（理由與 session 強制登出、
// recordBlockAction 完全相同）：傳 nil 會讓 recordAdminAction 整筆跳過，那個
// nil 容忍是為了讓「稽核表還沒建好」時後臺仍可操作。
//
// beginAdminTx 拿不到 nil（它在 s.db 為 nil 時回錯誤），所以 recordAdminAction
// 的 nil 分支在這條路徑上走不到 —— 那不是缺陷，而是 beginAdminTx 已把「沒有
// 資料庫」變成一個明確的錯誤。這裡因此會記下「無法為索引失敗補記稽核」而
// 不是靜默略過：索引失敗已經是資料不一致，再加一層看不見的稽核缺失會讓
// 診斷難度加倍。
//
// 這個函式**只記錄、不改變回應**：索引失敗不該讓管理員看到「刪文失敗」而實際
// 上文章確實被刪掉了（那才是真正的資料不一致）。它回傳 error 讓呼叫端決定
// 記不記日誌，而這裡自己已經記了。
func (s *Server) recordPostIndexFailure(r *http.Request, postID int64, cause error) {
	logger.WarnfContext(r.Context(), "[SEARCH] 索引未同步，寫入稽核紀錄 post_id=%d: %v", postID, cause)

	tx, err := s.beginAdminTx(r)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[SEARCH] 無法為索引失敗補記稽核 post_id=%d: %v", postID, err)
		return
	}
	defer tx.Rollback()
	if err := s.recordAdminAction(r, tx, adminActionPostIndexFailed, audit.TargetPost,
		strconv.FormatInt(postID, 10), "",
		audit.Change{Field: "searchIndex", Before: "已同步", After: "同步失敗：" + cause.Error()},
	); err != nil {
		logger.ErrorfContext(r.Context(), "[SEARCH] 索引失敗的稽核寫入失敗 post_id=%d: %v", postID, err)
		return
	}
	if err := tx.Commit(); err != nil {
		logger.ErrorfContext(r.Context(), "[SEARCH] 索引失敗的稽核提交失敗 post_id=%d: %v", postID, err)
	}
}

// beginAdminTx 開啟一個供「操作 + 稽核」共用的交易。
//
// 所有會改變資料的後臺 handler 都必須經過這個函式取得 *sql.Tx，而不能直接
// 呼叫 s.db.ExecContext。理由是稽核必須與操作同生共死（見 audit 套件檔頭），
// 而「同生共死」只有在兩者使用同一個交易時才有可能。
//
// 這個函式存在的另一個理由是讓「哪些 handler 該用 tx」變成可搜尋的：搜尋
// beginAdminTx 就會列出全部需要稽核的寫入點，而搜尋 s.db.ExecContext 會
// 混進二十幾個純讀取與尚未稽核的寫入。
//
// s.db 為 nil 時回錯誤而不是 panic，理由與 probeDatabase 的 nil 防護相同：
// beginAdminTx 可能被背景 goroutine 呼叫（recordPostIndexFailure），而
// goroutine 裡的 panic **不會**被 net/http 的 per-connection recover 接住。
// 以目前的 main.go 不可達（OpenMySQL 失敗會直接 Fatal），但那正是未來
// 「稽核補記要能在資料庫沒接上時仍然不拖垮行程」會踩到的第一顆地雷。
//
// 回傳 error 而不是 (nil, nil)：呼叫端一律以 err != nil 判定失敗，因此
// 假成功會讓它拿著 nil 的 *sql.Tx 繼續走下去。
func (s *Server) beginAdminTx(r *http.Request) (*sql.Tx, error) {
	if s.db == nil {
		return nil, errNoDatabase
	}
	return s.db.BeginTx(r.Context(), nil)
}

// errNoDatabase 是 beginAdminTx 在沒有資料庫連線時的錯誤。
//
// 宣告成共用變數（而不是每次 new）是為了讓呼叫端可以用 == 比較，且錯誤訊息
// 只有一份 —— 這條路徑的失敗訊息會進日誌，重複的字串串接起來會讓 log 難以 grep。
var errNoDatabase = errors.New("httpapi: 需要資料庫連線才能開啟交易")

// onlyChanged 濾掉新舊值相同的欄位變更。
//
// 為什麼需要：多數 PUT 是「全欄位取代」，所以即使只有一個欄位真的改變了，
// 程式碼仍會把五個欄位都放進 diff。若不濾掉，稽核紀錄上會出現
// "status: PENDING → RESOLVED" 旁邊跟著四行 "reporter_email: a@b → a@b"，
// 那讓「有什麼改變」這一題需要逐行比對才能回答 —— 而這正是稽核紀錄
// 最重要的用途。
//
// 刻意不做「相同則省略欄位名」以外的處理：刪除與新增本來就沒有 before，
// 由呼叫端用空字串表示（audit.Change 的說明指出空字串與 nil 的差別）。
func onlyChanged(changes ...audit.Change) []audit.Change {
	out := make([]audit.Change, 0, len(changes))
	for _, change := range changes {
		if change.Before == change.After {
			continue
		}
		out = append(out, change)
	}
	return out
}

/* ==========================================================================
   GET /api/admin/log
   ========================================================================== */

// adminLogPageSize 是分頁預設筆數，與 audit.DefaultLimit 一致。
// 這裡寫死是為了讓「上一頁／下一頁」的按鈕行為不依賴後端的預設值 —
// 兩邊各有一份預設的話，改了其中一邊就會讓頁數計算錯位。
const adminLogPageSize = 50

// handleAdminLog 提供稽核紀錄的查詢。
//
//	GET /api/admin/log?actor=&action=&targetType=&targetId=&from=&to=&offset=
//
// 只讀取，所以不需要來源檢查（與其他 admin GET 相同）。分頁以 offset 為
// 單位而非 cursor：稽核紀錄的分頁是給管理員「從最新往回翻」的，offset
// 在這個方向上完全夠用，而 cursor 會讓網址列多出一個看不懂的參數。
func (s *Server) handleAdminLog(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	query := r.URL.Query()
	filter := audit.Filter{
		ActorEmail: query.Get("actor"),
		Action:     query.Get("action"),
		TargetType: query.Get("targetType"),
		TargetID:   query.Get("targetId"),
		Offset:     atoiDefault(query.Get("offset"), 0),
	}
	// from / to 接受兩種格式：date（YYYY-MM-DD，涵蓋整天）與
	// datetime-local（YYYY-MM-DDTHH:mm）。後者是 datetime-local 輸入框
	// 送出的原生格式，因此不能要求前端再轉一次。
	if raw := query.Get("from"); raw != "" {
		if parsed, ok := parseAdminLogTime(raw, false); ok {
			filter.From = parsed
		}
	}
	if raw := query.Get("to"); raw != "" {
		if parsed, ok := parseAdminLogTime(raw, true); ok {
			filter.To = parsed
		}
	}

	result, err := audit.List(r.Context(), s.db, filter)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[AUDIT] 查詢失敗: %v", err)
		internalError(w, "unable to list admin actions")
		return
	}
	actions, err := audit.DistinctActions(r.Context(), s.db)
	if err != nil {
		logger.WarnfContext(r.Context(), "[AUDIT] 無法列出動作清單（篩選器會只有已輸入的值）: %v", err)
		actions = []string{}
	}
	actors, err := audit.DistinctActors(r.Context(), s.db)
	if err != nil {
		logger.WarnfContext(r.Context(), "[AUDIT] 無法列出操作者清單: %v", err)
		actors = []string{}
	}

	writeOK(w, map[string]any{
		"ok":       true,
		"items":    result.Items,
		"total":    result.Total,
		"pageSize": adminLogPageSize,
		"actions":  actions,
		"actors":   actors,
	})
}

// parseAdminLogTime 解析 /api/admin/log 的 from 與 to。
//
// endOfDay 為 true 時，傳入 "2026-09-30" 會被解讀成當天 23:59:59，而不是
// 00:00:00 —— 否則「選 9/30 到 9/30」會得到零筆，而使用者看到的是
// 「今天沒有紀錄」這個錯誤結論。時間的部分一律以 UTC 儲存（與
// forum_admin_actions.created_at 一致），因此顯示時的時區轉換由前端負責。
func parseAdminLogTime(raw string, endOfDay bool) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	layouts := []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"}
	// datetime-local 的值不含時區，必須明講它是 UTC，不能用 time.Parse
	// 預設的本地時區 —— 那會讓篩選結果依管理員所在的機器而不同。
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			if endOfDay && layout == "2006-01-02" {
				return parsed.Add(24*time.Hour - time.Second), true
			}
			return parsed, true
		}
	}
	return time.Time{}, false
}

// atoiDefault 解析查詢參數的整數，失敗或缺漏時回 fallback。
//
// strconv.Atoi 會回錯而不是 0，因此不能直接用它 —— 「offset=abc」會讓
// 整頁查詢失敗，而那只是使用者的手滑。
func atoiDefault(raw string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return parsed
}
