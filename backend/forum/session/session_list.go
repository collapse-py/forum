package session

/*
Session 列舉與批次撤銷（session/session_list.go）。

這個檔案補上 Manager 原本刻意留下的那個擴充點：檔頭的 Info 型別說「預留日後
『列出某使用者所有 session』的擴充點」，這就是那個擴充點的實作。

為什麼需要
	一個後臺在「這個帳號是不是被盜用了」「把某個人的所有裝置都登出」這兩個
	問題上是瞎的。存取日誌看得到某個 IP 在做什麼，看不到「現在有幾支 session
	還活著、那些 session 是誰的」。而在 cookie 即憑證的設計下（見 session.go
	檔頭第一點），token 不會輪替也不綁 IP 或 UA —— 遭竊的 cookie 在過期前可被
	完整重用，因此「立刻撤銷所有 session」是唯一能提前止血的手段。

三個必須做對的地方

	1. 掃描一定要用 SCAN，絕不能用 KEYS
		KEYS 會讓 Redis 在掃完全部 keyspace 之前**阻塞**整個實例。這個 session
		庫與媒體 token 共用同一個 Redis，因此一個後臺頁面的 KEYS 會讓整站同時
		無法登入、無法讀圖 —— 包括「按一下頁面來解除封鎖」這件事本身。

		SCAN 的正確用法是帶著游標反覆呼叫，直到游標回 0。這個檔案的三個函式
		共用同一個迴圈形狀（scanLoop），因為把游標寫錯一次（最常見的錯誤是
		每次都傳 0，那會永遠只掃第一頁）就會讓整個功能靜默地只看到一部分
		session —— 而「少看到」看起來和「只有這幾支」一模一樣。

	2. 每批 key 的欄位與 TTL 用 pipeline 一次撈回
		掃描之後若對每個 key 各打一次 HGETALL 再各打一次 TTL，那是一支 session
		兩次往返。列出 200 支 session 就是 400 次往返 —— 在同機器的 Redis 上
		還好，跨網路就是好幾秒的空白頁。Pipeline 把每一批壓成一次往返。

	3. **不回傳完整 token**
		token 就是憑證本身（見 session.go 檔頭）。把它顯示在後臺畫面上，會讓
		「管理員截圖分享畫面」或「有人在旁邊看螢幕」變成一次完整的手法移交。
		因此 Record.Token 只留在記憶體裡供刪除使用，序列化時只給前 8 個字元
		（token 是 48 個 hex 字元，前 8 個足以讓人認出「是不是同一支」，
		不足以還原）。

	批次撤銷不依賴 token：它接收的是 email，掃描後比對 hash 裡的 email。
	這是刻意的 —— 若介面要傳 token 才能撤銷，那 token 就已經離開伺服器了。
*/

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

/*
掃描的兩個上限

	scanBatchSize  每次 Scan 呼叫要取多少個 key。

	              這是「一次向 Redis 要多少工作」的量，不是「掃描多深」——
	              SCAN 的深度由游標決定，而單次成本是 O(count) 而非
	              O(keyspace)。因此它可以設得比 SCAN 的預設（10）大很多：
	              批次大表示往返少，而 256 仍然是毫秒級就回來。

	maxScannedKeys  單次請求最多檢查多少個 key 就放棄。

	              為什麼需要放棄：這個掃描的成本是 O(session 總數)。正常情況下
	              站上同時登入的人數是幾十到幾百，掃描幾百個 key 是幾毫秒。但
	              若站上同時有十萬支 session（爬蟲大量登入、測試環境），這個
	              請求就會變成一個跑好幾秒的 SCAN —— 而它是管理員按一下就發出的
	              請求，那個等待時間會讓頁面看起來像壞掉。

	              因此達到上限就**回傳已掃到的部分並告知已截斷**，而不是掃到底。
	              截斷的結果對「找出那個帳號的 session」仍然有用（使用者的
	              session 通常在掃描的前段），而「沒掃到」這件事本身會被明確
	              告知，不會被偽裝成「這個帳號只有這些 session」。
*/
const (
	scanBatchSize   = 256
	maxScannedKeys  = 20000
	maxListSessions = 500
)

// tokenPrefixLength 是回傳給呼叫端的 token 前綴長度。
//
// 8 個 hex 字元 = 32 bits。作為「識別用途」足夠（十幾支 session 裡要撞到
// 同一個前綴的機率很低），作為「還原用途」完全不足（需要 192 bits）。
const tokenPrefixLength = 8

// createdAtField 是 session hash 裡記錄建立時間的欄位名。
//
// 這個欄位是後來才加的（見 Create 的說明），因此既有 session 沒有它。
const createdAtField = "created_at"

// Record 是一支 session 的列舉結果。
//
// CreatedAt 可能是零值：既有 session 是在 created_at 欄位存在之前建立的。
// 呼叫端必須把零值呈現為「未知」，而不是當成 1970 年 —— 那會讓管理員以為
// 這支 session 有五十年歷史。
type Record struct {
	// Token 是完整的憑證，**只供本套件內部刪除使用**，序列化時必須換成
	// TokenPrefix。
	Token string
	// Email 是這支 session 的身分。
	Email string
	// IsAdmin 是登入當下寫入的結果，不是即時比對。
	IsAdmin bool
	// CreatedAt 是建立時間；零值代表「既有 session，沒有這個欄位」。
	CreatedAt time.Time
	// TTL 是 Redis 回報的剩餘存活時間。負值代表 key 沒有 TTL（理論上不會發生，
	// 因為 Create 會在 Expire 失敗時刪 key），此時呼叫端應視為未知。
	TTL time.Duration
	// ExpiresAt 是「現在 + TTL」，也就是這支 session 真正會失效的時間點。
	// 刻意不用「CreatedAt + expire」：滑動續期會讓那個值與實際 TTL 脫節
	// （續期後 TTL 被重設，但 CreatedAt 沒變）。管理員在這裡要看到的是
	// 「還有多久會被登出」。
	ExpiresAt time.Time
	// TokenPrefix 是給呼叫端顯示用的截斷值。
	TokenPrefix string
}

// ListOptions 是 List 的查詢條件。
type ListOptions struct {
	// Email 非空時只回傳該 email 的 session。空字串代表全部。
	Email string
	// Limit 是回傳筆數上限，小於等於 0 時用 maxListSessions。
	Limit int
	// MaxScanned 是掃描 key 的上限，小於或超過 maxScannedKeys 時用後者。
	MaxScanned int
}

// ListResult 是一次列舉的結果與它的限制。
type ListResult struct {
	Sessions []Record
	// Scanned 是實際檢查過的 key 數（不只回傳的 session 數）。
	Scanned int
	// Truncated 為 true 代表掃描達到上限而提前放棄 —— 結果**不完整**。
	// 呼叫端必須把這件事告訴使用者，否則「沒列出來」會被讀成「不存在」。
	Truncated bool
	// TotalActive 是掃描期間實際數到的 session 總數（包含被 Limit 截掉的）。
	TotalActive int
	// ExpireIn 是本 Manager 的 session 存續時間，供介面顯示「每 X 小時續期」。
	ExpireIn time.Duration
}

// List 列出目前存在於 Redis 的 session。
//
// 掃描的是整個 FORUM:session: 命名空間（key 是隨機 token，無法由 email 反推
// token，因此沒有「只掃某個 email 的 key」這種可能）。這個限制是「token 即
// 憑證」設計的直接後果。
func (m *Manager) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	result := ListResult{Sessions: []Record{}, ExpireIn: m.expire}
	if m.rdb == nil {
		return result, nil
	}
	if opts.Limit <= 0 || opts.Limit > maxListSessions {
		opts.Limit = maxListSessions
	}
	if opts.MaxScanned <= 0 || opts.MaxScanned > maxScannedKeys {
		opts.MaxScanned = maxScannedKeys
	}

	err := m.scanSessions(ctx, opts.MaxScanned, seenKeys(func(record Record) error {
		if opts.Email != "" && record.Email != opts.Email {
			return nil
		}
		result.TotalActive++
		if len(result.Sessions) < opts.Limit {
			result.Sessions = append(result.Sessions, record)
		}
		return nil
	}), &result.Scanned, &result.Truncated)
	if err != nil {
		return result, err
	}
	return result, nil
}

// RevokeResult 是一次批次撤銷的結果。
type RevokeResult struct {
	// Revoked 是實際刪除的 session 數。
	Revoked int
	// Scanned 是掃描過程中檢查過的 key 數。
	Scanned int
}

// RevokeByEmail 刪除某個 email 的**所有** session。
//
// 這個函式是「強制登出」的全部實作。它掃出所有 key、挑出 hash 裡 email 相符
// 的，再刪除 —— 因為 key 是隨機 token，沒有辦法由 email 直接算出要刪哪幾把
// key（見檔頭第一點的說明）。
//
// 分批刪除（每次 500 把）而不是一次 DEL 全部：一個帳號可能有上百支 session
// （多裝置、反覆登入），而單次 DEL 太多 key 會讓 Redis 在該指令期間無法
// 處理其他連線。
//
// **掃描被截斷時回傳錯誤**，而不是回傳「已刪除 0 筆」。理由是語意：這個函式
// 的契約是「刪掉所有」，而在沒掃完的情況下它做不到 —— 若靜默回傳 0 筆，
// 後臺就會對使用者宣稱「已全部登出」而其實還有 session 活著。回錯則讓操作
// 失敗是誠實的（使用者會重試，而重試時可能就掃得完）。
func (m *Manager) RevokeByEmail(ctx context.Context, email string) (RevokeResult, error) {
	result := RevokeResult{}
	if m.rdb == nil {
		return result, nil
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return result, fmt.Errorf("email is required")
	}

	// pending 是「已確認屬於此 email、等待刪除」的 key 集合。累積到門檻才一次
	// DEL，而不是每找到一把就刪一次。
	const revokeBatch = 500
	pending := make([]string, 0, revokeBatch)

	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		deleted, err := m.rdb.Del(ctx, pending...).Result()
		if err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		// 用 Redis 回報的數字而不是 pending 的長度：掃描與刪除之間可能有
		// session 自然過期，那幾把刪不到（DEL 對不存在的 key 回 0）。用
		// pending 的長度會把「刪了 3 把」說成「刪了 4 把」。
		result.Revoked += int(deleted)
		pending = pending[:0]
		return nil
	}

	var truncated bool
	err := m.scanSessions(ctx, maxScannedKeys, seenKeys(func(record Record) error {
		if record.Email != email {
			return nil
		}
		pending = append(pending, sessionKey(record.Token))
		if len(pending) >= revokeBatch {
			return flush()
		}
		return nil
	}), &result.Scanned, &truncated)
	if err != nil {
		return result, err
	}
	if err := flush(); err != nil {
		return result, err
	}
	if truncated {
		return result, fmt.Errorf("revoke sessions: 掃描達到 %d 個 key 上限，未能確認是否已全部撤銷", maxScannedKeys)
	}
	return result, nil
}

// seenKeys 產生一個「對 SCAN 結果去重」的行數函式。
//
// 抽出來是因為 List 與 RevokeByEmail 需要完全相同的那一段程式碼，而兩邊各寫
// 一次就會有兩次寫錯的機會 —— 而寫錯的症狀是「同一支 session 在結果裡出現
// 兩次」（Revoke 較幸運，第二次 DEL 會回 0，不會重複計數）。
//
// 存在的必要性：Redis 的 SCAN **不保證**一次遍歷內不重複回傳同一個 key
// （它只保證「至少回傳一次」）。因此沒有這一層去重，列表裡就會出現重複列。
func seenKeys(visit func(Record) error) func(Record) error {
	seen := make(map[string]struct{}, 64)
	return func(record Record) error {
		if _, ok := seen[record.Token]; ok {
			return nil
		}
		seen[record.Token] = struct{}{}
		return visit(record)
	}
}

// scanSessions 是這個檔案的掃描迴圈。回調每收到一支 session 就被呼叫一次。
//
// 三個參數用指標是因為 scanned 與 truncated 要在迴圈中被更新，而 go 沒有
// 多重回傳值。用一個 struct 指標也可以，但多一個型別只為了兩個欄位不划算。
//
// visit 回傳錯誤會立刻中止掃描（並把錯誤往上報）。RevokeByEmail 藉由它來
// 「在累積到批次門檻時順便刪除」，因此「回傳錯誤」不只是「有問題」，也是
// 它用來表達「這一批處理完了，去刪」的手段。
//
// 上限的施加位置是這個函式裡最容易寫錯的地方，因此值得說明：SCAN 一次可能
// 回傳**整批** key（最多 COUNT 個），與游標深度無關。因此「在迴圈開頭檢查
// scanned >= maxKeys」是**不夠的** —— 第一次 Scan 就可能一次拿回 300 把 key，
// 而 maxKeys 是 5，那一批已經超出預算三倍了。
//
// 正確的做法是把超出預算的那一部分**切掉**，並因為「已經知道 keyspace 沒走完」
// 而立刻標記 truncated 並跳出。切掉而不是「照樣處理完再說」是因為那個上限的
// 整個用意就是「不要在這個請求裡做太多事」。
func (m *Manager) scanSessions(ctx context.Context, maxKeys int, visit func(Record) error, scanned *int, truncated *bool) error {
	// SCAN 的游標必須從 0 開始，然後**帶著回傳的游標**繼續呼叫。
	// 每次都傳 0 會讓迴圈永遠只掃第一頁 —— 而症狀是「看得到少數幾支
	// session，看起來一切正常」。
	var cursor uint64
	for {
		if *scanned >= maxKeys {
			*truncated = true
			return nil
		}
		keys, next, err := m.rdb.Scan(ctx, cursor, sessionKeyPrefix+"*", int64(scanBatchSize)).Result()
		if err != nil {
			return fmt.Errorf("scan sessions: %w", err)
		}
		cursor = next

		// 這一批可能整批超出剩餘預算。切掉多出來的，並因為已經知道還有 key
		// 沒被掃到而直接標記截斷。
		if remaining := maxKeys - *scanned; len(keys) > remaining {
			keys = keys[:remaining]
			*truncated = true
		}
		*scanned += len(keys)

		records, err := m.fetchRecords(ctx, keys)
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := visit(record); err != nil {
				return err
			}
		}
		if *truncated || cursor == 0 {
			return nil
		}
	}
}

// fetchRecords 以 pipeline 一次取回一批 key 的欄位與 TTL。
//
// 去重由呼叫端的 seenKeys 處理，因此這個函式不需要知道看過什麼。
func (m *Manager) fetchRecords(ctx context.Context, keys []string) ([]Record, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	// 一次往返撈回整批：N 個 HGETALL + N 個 TTL 壓成一次 Exec。
	// 逐個呼叫的話這裡會是 2N 次往返（見檔頭第二點的說明）。
	pipe := m.rdb.Pipeline()
	fields := make([]*redis.MapStringStringCmd, len(keys))
	ttls := make([]*redis.DurationCmd, len(keys))
	for i, key := range keys {
		fields[i] = pipe.HGetAll(ctx, key)
		ttls[i] = pipe.TTL(ctx, key)
	}
	// Exec 會回「至少一個指令失敗」的錯誤。對我們來說那不一定是問題：
	// HGETALL 對不存在的 key 回空 map 而不報錯，TTL 回 -2 也不報錯。真正
	// 該中止的是連線層級的錯誤 —— 那一種下一批也會失敗，繼續只會把同一個
	// 錯誤重複 N 次。因此只有在錯誤不是「讀不到值」時才中止。
	if _, err := pipe.Exec(ctx); err != nil && !isMissingValueError(err) {
		return nil, fmt.Errorf("read sessions: %w", err)
	}

	now := time.Now()
	records := make([]Record, 0, len(keys))
	for i, key := range keys {
		values, err := fields[i].Result()
		if err != nil || len(values) == 0 {
			// 空 hash 代表 key 在掃描與讀取之間過期了。跳過即可 —— 那正是
			// 「這個 session 剛好到期」的自然結果。
			continue
		}
		record := Record{
			Token:       strings.TrimPrefix(key, sessionKeyPrefix),
			Email:       values["email"],
			IsAdmin:     values["is_admin"] == "true",
			TokenPrefix: "",
		}
		// TTL 單獨處理，不直接用 ttls[i].Val()。
		//
		// 為什麼：pipe.Exec 在**指令層**失敗時（例如該指令的型別錯誤）仍會
		// 回 err，那時 !isMissingValueError 擋不住、程式會繼續往下走，而
		// 失敗的那一個 ttls[i] 會回傳零值 DurationCmd —— Val() 因此得到 0，
		// 於是 ExpiresAt = now，介面上顯示成「馬上要到期」。
		//
		// 顯示一個**錯誤的**到期時間比顯示「不知道」糟：前者會讓管理員以為
		// 這是異常狀態而去處理它。TTL = -1 是這個型別裡「不知道」的既有
		// 表示（ExpiresAt 保持零值，見 Record 的說明），因此沿用它而不是
		// 新造一個哨兵值。
		if err := ttls[i].Err(); err != nil {
			record.TTL = -1
		} else {
			record.TTL = ttls[i].Val()
		}
		record.TokenPrefix = tokenPrefix(record.Token)
		// created_at 是後來才加入的欄位，既有 session 沒有它。缺欄位時保持零值，
		// 由呼叫端呈現為「未知」（見 Record.CreatedAt 的說明）。
		if raw := values[createdAtField]; raw != "" {
			if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
				record.CreatedAt = time.Unix(seconds, 0).UTC()
			}
		}
		if record.TTL >= 0 {
			record.ExpiresAt = now.Add(record.TTL).UTC()
		}
		records = append(records, record)
	}
	return records, nil
}

// isMissingValueError 判斷錯誤是否只是「讀不到那個值」。
//
// 只認 redis.Nil：那是 HGETALL 對已消失的 key 所回傳的錯誤（回空 map），
// 以及 TTL 對不存在的 key 所回傳的 -2 之外的讀取失敗。它屬於「這支 session
// 剛好過期」的自然結果，應該跳過而不是中止整個掃描。
//
// 用 errors.Is 而不是 == ：同檔案其他地方的慣例是 errors.Is（見 ipban.go），
// 而兩者的差別只在錯誤「被包裝」時才顯現。目前 go-redis 直接回傳 redis.Nil
// 哨兵值所以不會出錯，但一旦上游改為包裝回傳，pipe.Exec 的處理就會從
// 「略過過期 session」變成「中止整批掃描」—— 症狀是 session 列表整頁空白，
// 而錯誤訊息裡看不出真正的原因。
//
// 其餘（例如連線錯誤）回 false —— 那是真的做不下去，繼續只會重複同樣的失敗。
func isMissingValueError(err error) bool {
	return errors.Is(err, redis.Nil)
}

// tokenPrefix 截出可安全顯示的 token 前綴。
func tokenPrefix(token string) string {
	if len(token) <= tokenPrefixLength {
		return token
	}
	return token[:tokenPrefixLength] + "…"
}
