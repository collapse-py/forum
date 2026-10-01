/*
Package audit 記錄並查詢「管理員在後臺做了什麼」。

為什麼需要這個套件

	目前這個專案的後臺有停權、刪文、刪留言、裁定檢舉、改標籤、代發文六類會
	改變資料的操作，而它們**完全沒有留下紀錄**。存取日誌（logger 的
	LoggingMiddleware）只記到「某個 IP 用某個 session 在 23:14 對
	/api/admin/forum/reports/9 送出 DELETE」這一層，它不包含決定性的資訊：
	那筆檢舉被裁定為成立還是不成立、刪掉的是哪一篇文、停權前後的狀態是什麼。
	於是「有人檢舉我的文、兩小時後文不見了」這種問題既答不出來，也無法
	向使用者交代。

	這個套件補上的正是那一層：一次「誰、什麼時候、對哪個對象、做了什麼、
	哪些欄位從什麼變成什麼」。

三個關鍵設計

 1. 寫入與操作必須在同一個交易裡，且寫入失敗要讓操作失敗
    Record 接受一個 Execer（*sql.Tx 與 *sql.DB 都滿足）並**回傳錯誤**。
    所有會改變資料的 handler 都以 tx 呼叫它，並且在 Commit 之前檢查這個
    錯誤；因此「操作成功但沒有稽核紀錄」在資料庫層就不可能發生。

    這一點需要一個明確的前提才能成立，而它很容易被順手破壞：Go 的
    database/sql **不會**因為交易內某一個語句出錯就自動中止交易。也就是
    說，若呼叫端在稽核寫入失敗之後仍然 Commit，那次操作就會真的發生而
    稽核表裡沒有它 —— 而稽核日誌最不可信的使用方式就是「用它來證明某
    件事沒發生過」。

    因此每個呼叫端都必須寫成：

    if err := s.recordAdminAction(...); err != nil {
    return  // 延遲的 tx.Rollback() 會撤掉這次操作
    }
    ... 其他寫入 ...
    tx.Commit()

    傳入 *sql.DB 而非 *sql.Tx 也能運作，但會失去那個保證（稽核寫入失敗
    時操作已經完成，且沒有東西會讓它回滾）。這個取捨由呼叫端在每個
    handler 的註解裡重申一次。

 2. 欄位級 diff 以 JSON 存在 changes 欄位，而不是每個欄位開一欄
    操作涵蓋六種資源（user / post / comment / report / tag），每種要
    記的欄位都不同。開一欄就要開六組，回傳時也還得在前端依類型分流。
    存 JSON 讓「新增欄位」不需要 schema 變更。

    取捨是變成 TEXT 欄位、無法用 SQL 對變更內容做條件篩選。對這個規模
    的論壇來說可以接受：要查「誰動過某個欄位」時，先用 actor/action/
    target 篩出少量列，再讀 JSON 即可。

 3. 變更值截斷，不存原文
    貼文與留言的內容可能長達 2000 字元。稽核紀錄要回答的是「這段文字被
    改成了什麼」，但不是為了備份全文 —— 因此每個值截到 200 字元並標記
    截斷。完整內容仍在 forum_posts / forum_comments，稽核表不重複一份
    可能含有個資的長文字。

不做的東西

  - 不稽核讀取操作。「管理員看了什麼」對這個論壇沒有稽核價值，而它的量級
    遠大於寫入（監控頁每十秒就讀一次 /api/admin/monitor）。
  - 不提供刪除稽核紀錄的 API。要移除紀錄得直接動資料庫，這是刻意的：
    一個能刪除自己紀錄的稽核日誌等於沒有稽核日誌。
  - 不驗證 actor_email 是否真的是管理員。那是呼叫端 requireAdminForum 的
    職責；這個套件信任傳入的值，因為它的唯一寫入途徑都在 admin handler 內。
*/
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNoDatabase 表示稽核寫入沒有可用的資料庫連線。
//
// 它讓「忘了注入」成為一個明確的錯誤（於是操作回滾），而不是寫入
// 一筆缺欄位的紀錄或直接被忽略 —— 兩者都會讓稽核紀錄變成不可信。
var ErrNoDatabase = errors.New("audit: 需要資料庫連線才能記錄操作")

// TargetType 是被操作的資源類別。宣告為字串型別的聯集而非 enum：新增一種
// 資源只要加一個常數，不需要 schema 變更，而這個表是純附加的。
const (
	TargetUser    = "user"
	TargetPost    = "post"
	TargetComment = "comment"
	TargetReport  = "report"
	TargetTag     = "tag"
	// TargetSystem 給不屬於任何單一資源的操作（例如批次停權多個使用者時，
	// 每個使用者各自記一筆 user.suspend，因此這個值目前沒有使用；保留它是
	// 為了讓「不指涉具體資源」的操作有一個明確的 target，而不是拿空字串。
	TargetSystem = "system"
	// TargetIP 給「以 IP 為對象」的操作（封鎖、解封）。它不是一個資料表的
	// 資源，而是限流與封鎖名單的識別值 —— 而那些狀態住在 Redis 而非 MySQL。
	TargetIP = "ip"
	// TargetAnnouncement 給站內公告。與 TargetIP 同理：它的狀態在 MySQL，但
	// 稽核紀錄需要一個能一眼看出「這筆是在動公告」的類別名。
	TargetAnnouncement = "announcement"
)

// maxValueLength 是單一變更值的最大保留長度。
//
// 200 字元足以回答「狀態從 ACTIVE 變成 SUSPENDED」「標題從 A 變成 B」這類
// 問題，同時讓一篇文章的完整內文不會在稽核表裡出現第二份 —— 那既浪費空間，
// 也讓這張表變成另一份需要一起刪除的個人資料副本。
const maxValueLength = 200

// maxChangesPerEntry 是單筆紀錄最多記幾個欄位變更。
//
// 沒有上限的話，一個「套用全部標籤」的操作（使用者有 40 個標籤）會讓
// changes 變成幾 KB 的 JSON，捲動稽核紀錄時每一列都很笨重。超過上限時
// 截斷並在 Truncated 標記 —— 標籤這種「整組替換」的操作，重點本來就是
// 「整組被換掉了」，不是逐項比對。
const maxChangesPerEntry = 20

// Change 是一個欄位的新舊值。
//
// 一律是字串而非 any：來源欄位有 int、*string、string 等多種型別，而稽核
// 紀錄只需要能被人讀出來。nil（欄位原本不存在）與空字串（欄位存在但為空）
// 刻意不同 —— 前端因此能把「原本沒有 nick_name」與「nick_name 被清空」
// 呈現成兩件不同的事。
type Change struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
	// Truncated 為 true 代表 Before 或 After 已被截到 maxValueLength。
	Truncated bool `json:"truncated,omitempty"`
}

// Entry 是一筆稽核紀錄。
//
// 寫入時以 JSON 序列化 changes；讀出時反序列化回 []Change。欄位一律用
// string（時間例外）以對應 MySQL 的 DATETIME 掃描。
type Entry struct {
	ID          int64    `json:"id"`
	ActorEmail  string   `json:"actorEmail"`
	Action      string   `json:"action"`
	TargetType  string   `json:"targetType"`
	TargetID    string   `json:"targetId"`
	TargetLabel string   `json:"targetLabel,omitempty"`
	Changes     []Change `json:"changes,omitempty"`
	ClientIP    string   `json:"clientIp,omitempty"`
	RequestID   string   `json:"requestId,omitempty"`
	CreatedAt   string   `json:"createdAt"`
}

// 資料表欄位名稱集中在這裡，避免「INSERT 與 SELECT 的欄位順序」這種
// 只有在欄位改名時才會浮現的錯誤散落在多處。
const (
	columnID          = "id"
	columnActorEmail  = "actor_email"
	columnAction      = "action"
	columnTargetType  = "target_type"
	columnTargetID    = "target_id"
	columnTargetLabel = "target_label"
	columnChanges     = "changes"
	columnClientIP    = "ip"
	columnRequestID   = "request_id"
	columnCreatedAt   = "created_at"
)

// Execer 是 Record 需要的介面，*sql.DB 與 *sql.Tx 都滿足它。
//
// 用介面而不是直接吃 *sql.Tx：*sql.DB 滿足它讓「沒有交易可用」的呼叫端
// 仍能記錄（只失去「寫入與操作同生共死」的保證），而測試可以用一個假的
// Execer 驗證 SQL 與序列化，不必準備資料庫。
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Record 寫入一筆稽核紀錄，並回傳寫入是否成功。
//
// 回傳錯誤（而不是只記 log）是刻意的，理由見檔頭第一點：這是唯一能讓
// 「操作與稽核同生共死」成立的機制。呼叫端必須在 Commit 之前檢查它，
// 失敗時讓交易回滾 —— 那樣這次操作就等於沒有發生過，而不是「發生了但
// 沒被記下來」。
//
// 錯誤訊息刻意不含 actor_email：它可能是含有個資的字串，而 log 的讀取範圍
// 通常比資料庫寬。target_type / target_id 沒有這個問題，保留下來是為了讓
// 故障時還看得出來是哪一類操作。
func Record(ctx context.Context, db Execer, entry Entry) error {
	if db == nil {
		return ErrNoDatabase
	}
	changes, err := encodeChanges(entry.Changes)
	if err != nil {
		// 序列化失敗只可能是 programmer error（Change 欄位不是字串）。
		// 這種情況下寧可讓整筆紀錄失敗（於是操作也回滾），也不要寫一筆
		// 沒有 diff 的紀錄 —— 一份宣稱完整卻少了欄位變更的稽核紀錄，比
		// 沒有紀錄更糟，因為它會讓人以為已經查過了。
		return fmt.Errorf("encode audit changes: %w", err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO forum_admin_actions
			(actor_email, action, target_type, target_id, target_label, changes, ip, request_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.ActorEmail,
		entry.Action,
		entry.TargetType,
		entry.TargetID,
		truncate(entry.TargetLabel, maxValueLength),
		changes,
		entry.ClientIP,
		entry.RequestID,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert admin action %s %s/%s: %w", entry.Action, entry.TargetType, entry.TargetID, err)
	}
	return nil
}

// encodeChanges 把欄位變更序列化為 JSON，並在寫入前做兩件事：截斷值、
// 限制筆數。
//
// 兩者都在「寫入前」而不是讀出時做：這張表是唯一會長期保存這些值的地方
// （access log 會輪替，而稽核紀錄不會），因此讓資料庫裡永遠只有已截斷的
// 版本，比在每個顯示處都要記得處理長度安全得多。
func encodeChanges(changes []Change) (string, error) {
	if len(changes) == 0 {
		return "", nil
	}
	if len(changes) > maxChangesPerEntry {
		changes = changes[:maxChangesPerEntry]
		// 附加一筆「還有 N 個」讓讀者知道被截斷，而不是誤以為只有這些。
		changes = append(changes, Change{Field: "…", Truncated: true})
	}
	for i := range changes {
		before, beforeCut := truncateValue(changes[i].Before, maxValueLength)
		after, afterCut := truncateValue(changes[i].After, maxValueLength)
		changes[i].Before = before
		changes[i].After = after
		if beforeCut || afterCut {
			changes[i].Truncated = true
		}
	}
	encoded, err := json.Marshal(changes)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// ellipsis 是截斷後補上的記號。用 ASCII 的三個點而不是「…」：後者佔三個
// 位元組，會讓「結果不超過 limit 個位元組」這個保證變成 limit+2。
//
// 顯示上少一點美觀，但換來的是一個可以被斷言的長度上界 —— 而長度上界正是
// 這個函式存在的理由（資料庫欄位是 VARCHAR(320)，若哪天 maxValueLength
// 被調到等於欄位寬度，多出來的位元組就會讓寫入失敗）。
const ellipsis = "..."

// truncateValue 把字串截到 limit 個位元組之內（含結尾的省略號），並回傳
// 是否真的截斷了。
//
// 為什麼以「位元組」而不是「字元」：Go 的字串是位元組，而 truncate 的
// 保證必須是可斷言的（「結果不超過 N 個位元組」）。中文因此實際保留下
// 少於 N/3 個字 —— 對這個用途（辨識一段文字）足夠，而讓長度保證變得
// 模糊才會是真正的問題。
//
// 切到合法 UTF-8 邊界是硬需求：切出半個字元會讓整個 changes JSON 反序列化
// 失敗，症狀是「稽核紀錄整頁載不出來」—— 與「截斷函式寫錯了」毫無關聯。
func truncateValue(value string, limit int) (string, bool) {
	if len(value) <= limit {
		return value, false
	}
	// 為省略號預留三個位元組，讓「不超過 limit」這個保證成立。
	cut := limit - len(ellipsis)
	if cut < 0 {
		// limit 小於省略號長度時只能直接切；這只在 maxValueLength 被調成
		// 極小值時才會發生，而那會讓紀錄沒有可讀性 —— 寧可切，也不要
		// 因為一個保證而回傳超長的字串。
		cut = 0
	}
	// 逐位元組回退到合法的字元開頭。
	for cut > 0 && !isUTF8Boundary(value, cut) {
		cut--
	}
	return value[:cut] + ellipsis, true
}

// truncate 把字串截到上限並補上省略號，不回傳是否截斷。
//
// 給「不需要知道是否截斷」的呼叫端用（目前只有 target_label）；會關心
// 截斷與否的地方請用 truncateValue，因為那個旗標是 Change.Truncated 的
// 來源，而它是讀者判斷「這是不是全文」的唯一依據。
func truncate(value string, limit int) string {
	cut, _ := truncateValue(value, limit)
	return cut
}

// isUTF8Boundary 判斷第 index 個位元組是否為一個字元的開頭。
// 續位元組的型樣是 10xxxxxx，因此開頭的位元組若不是 0b10xxxxxx 即為邊界。
func isUTF8Boundary(s string, index int) bool {
	if index <= 0 || index >= len(s) {
		return true
	}
	return s[index]&0xC0 != 0x80
}
