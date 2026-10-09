/*
Package ipban 管理「被封鎖的來源位址」名單。

為什麼需要它

	既有的限流（ratelimit.go）是**行程內**的滑動視窗，而它是純記憶體的，原因
	寫在 ratelimit.go 檔頭：把計數器放進 Redis 會讓每個請求多一次網路往返。

	但純記憶體有兩個限制，而它們都不是「可以接受的取捨」而是「擋不住攻擊」：

	  1. **重啟即失效。** 攻擊者只要等一次部署（或一次崩潰）就重新拿到滿額度。
	     對付持續性的自動化攻擊，重啟是最省事的解鎖方式。
	  2. **不跨行程。** 負載平衡器後面有 N 個執行個體時，實際額度是 limit × N，
	     而輪到哪一台是使用者控制不了的。

	封鎖名單補上的正是這兩點：它存進 Redis，因此跨行程、跨重啟都存活。

與限流的分工

	兩者是**並存**而不是替代：限流處理「正常流量的瞬間尖峰」，封鎖處理
	「已經確認是濫用者」。一個被封鎖的 IP 不會走「計次、然後 429」那條路 ——
	它直接被擋，不必先讓它用掉十次額度。

	反過來，封鎖名單不處理正常流量：一個只有 11 次/分鐘的 IP 永遠不會被封，
	因為它沒有被列入名單。把它做成「自動封鎖」會誤傷 NAT 後面共用出口的
	所有人（見 ratelimit.go 檔頭的 per-client 說明），而誤封一個正常使用者
	比多讓一個腳本多打幾次嚴重得多。**因此封鎖永遠是管理員的決定。**

熱路徑的成本（這是本功能最需要被評估的一件事）

	IsBanned 是一次 ZSCORE。加上它的時機是每個「非 GET 且被限流器攔到」的
	請求，也就是：建立貼文、加留言、按讚、檢舉、上傳圖片、OAuth 登入跳轉。

	成本的實際大小：

	  - 指令本身：ZSCORE 是 O(log N)，N 是封鎖筆數（這個站上通常是 0 到數十）。
	    單一 key、單一 member，回應是 8 bytes 左右的整數。
	  - 往返：與本站已有的 Redis 呼叫同一條連線。session 套件在**每個**請求
	    上都已經做了一次 HGET（ResolveUser），而它只為了「知道有沒有登入」。
	    IsBanned 多出來的那一次與那一次性質完全相同。因此在本專案的架構下，
	    這個成本是「又多一次已經在做的事」，而不是「引進了一種新的延遲來源」。
	  - 與該請求原本的工作量相比：建立一則貼文要做一次 INSERT、可能一次 ES
	    索引（一次 HTTP 往返），加上使用者的配額檢查。ZSCORE 相對之下是雜音。
	  - **讀取端點完全不受影響。** 限流器只掛在寫入型路由上（見 ratelimit.go
	    檔頭的掛載位置說明），因此匿名訪客的瀏覽完全不會多付這一次。

	結論：這個成本可以接受，而它買到的東西是「封鎖在部署之後仍然有效」。

失敗時的行為：fail open

	Redis 故障時 IsBanned 會回錯誤，而呼叫端**放行**。理由：

	  - 封鎖檢查失敗就擋掉所有人，會把一次 Redis 抖動變成「整站不能發文」。
	    那是一個比「Redis 掛掉期間封鎖失效」嚴重得多的事故 —— 因為 Redis 掛掉
	    時本站的登入本來就已經受影響（session 查不到 = 未登入），再讓連登入都
	    做不到就是雪上加霜。
	  - 「Redis 掛掉時封鎖失效」是一個可接受且**可觀察**的狀態：監控頁的
	    Redis 探測會變紅，而 log 會有警告。

	同一個決定也意味著：**攻擊者只要製造 Redis 壓力，就能暫時解除對自己的
	封鎖**。這個取捨是明確做出的，不是疏漏。
*/

package ipban

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// banSetKey 是封鎖名單所在的 Redis key。
//
// 用 sorted set（ZSET）而不是 SET + 各自的 TTL，理由有三個：
//  1. 一個指令就能查詢（ZSCORE）、列出（ZRANGE 帶 score）與清理
//     （ZREMRANGEBYSCORE），不需要為每個 IP 各自管理一把 key。
//  2. 清理是「一次刪掉所有過期項」，不會像多把 SET 一樣留下孤兒 key。
//  3. score 存到期秒數，因此「剩多久到期」是查詢結果的一部分，不需要另外存。
//
// 前綴與既有的 FORUM:session:、FORUM:token: 一致，讓同一個 Redis DB 裡的
// 各功能仍然互不覆蓋。
const banSetKey = "FORUM:ipban"

// maxBanDuration 是單次封鎖的長度上限（預設 365 天）。
//
// 為什麼要有上限：score 是到期秒數，因此「永久封鎖」只能寫成一個極大的數字。
// 那種封鎖沒有辦法靠時間自動解除 —— 它會一直查到有人來解除為止，而管理員在
// 幾個月後已經不記得自己封過誰。與其給一個「永久」選項然後讓它變成無主的
// 封鎖，不如給一個很長但明確的長度：想要長期封鎖就封一年，明年再決定。
const maxBanDuration = 365 * 24 * time.Hour

// MaxBanDuration 是 maxBanDuration 的匯出版本，供後台在送出封鎖前先收斂
// （這樣管理員會拿到「已被收斂到一年」的結果而不是一個錯誤）。
//
// 匯出而不是讓後端重寫一遍數字：兩份數字一旦不同步，管理員輸入「一年」而
// 後端拒絕（因為它以為上限是兩年）會是一個沒有上下文的錯誤。
const MaxBanDuration = maxBanDuration

// minBanDuration 是「看起來不像手滑」的最小剩餘封鎖時間。
//
// 存在的理由是一個實務上的失誤模式：管理員輸入「1」以為是「1 天」而拿到
// 一分鐘的封鎖，攻擊者一分鐘後就能回來，而管理員會以為自己已經處理了。
//
// 檢查時刻意留了模糊地帶（下限的一半）：呼叫端計算 until 與這個函式比較之間，
// 時鐘已經往前走了。因此「剛好一分鐘」在實際執行時會變成 59.x 秒，而以
// minBanDuration 當門檻會讓那個**合法**的值是否被拒取決於程式跑到多快 ——
// 一個會隨執行速度改變結果的驗證等於沒有驗證。因此門檻是 minBanDuration/2：
// 一分鐘的封鎖永遠通過，低於 30 秒的才被拒。
const minBanDuration = time.Minute

// ErrNoStore 表示 Store 沒有可用的 Redis 連線。
//
// 它存在的理由與 audit.ErrNoDatabase 相同：讓「忘了注入」是一個明確的錯誤，
// 而不是讓「封鎖功能靜靜地什麼都不做」成為可能。後者會讓管理員以為自己封了某
// 個 IP，而它其實沒有生效 —— 而那種誤解會持續到有人受害。
var ErrNoStore = errors.New("ipban: 需要 Redis 連線才能管理封鎖名單")

// ErrInvalidIP 是給呼叫端顯示用的固定錯誤。
//
// 存在是為了讓「IP 格式不正確」有一致的訊息，而不是讓 net 套件的錯誤字串
// （"invalid IP address: 1.2.3"）漏到使用者面前 —— 那個字串是給程式看的。
var ErrInvalidIP = errors.New("ipban: IP 格式不正確")

// Entry 是一筆封鎖。
//
// Expires 是絕對時間（UTC）而不是剩餘秒數：剩下的時間每過一秒就少一，而
// 絕對時間不會 —— 用它來算剩餘時間才不會在重新整理之間跳動。
type Entry struct {
	IP      string    `json:"ip"`
	Expires time.Time `json:"expires"`
	// By 是把它列入名單的管理員 email；Reason 是當時填的原因。
	// 兩者只存在於記憶體與稽核紀錄裡，**不存進 Redis** ——
	// 理由見 Store 的「刻意不存」說明。
	By     string `json:"by"`
	Reason string `json:"reason"`
}

// RemainingSeconds 是距離到期的秒數，已過期者回 0。
func (e Entry) RemainingSeconds() int64 {
	remaining := time.Until(e.Expires).Seconds()
	if remaining < 0 {
		return 0
	}
	return int64(remaining)
}

// Store 是封鎖名單的存取層。
//
// 建構後除了 now 之外皆不變更，因此可以安全地被多個 goroutine 併發使用。
// 沒有 mutex：Redis 指令本身是原子的，而這個型別不持有任何行程內的狀態
// （連「上次錯誤是什麼」都不記 —— 那是呼叫端 logLoudly 的職責）。
type Store struct {
	rdb *redis.Client
	now func() time.Time
}

// New 建立一個 Store。rdb 為 nil 時所有操作都回 ErrNoStore（呼叫端 fail open）。
func New(rdb *redis.Client) *Store {
	return &Store{rdb: rdb, now: time.Now}
}

// SetClock 替換取時間的函式，只為讓測試能確定性地推進到期邊界。
func (s *Store) SetClock(now func() time.Time) { s.now = now }

// IsBanned 查詢某個 IP 目前是否被封鎖。
//
// 為什麼用 ZSCORE 而不是 ZISMEMBER：ZISMEMBER 只回答「在不在名單裡」，而名單裡
// 會有已過期的項目（清理是週期性的，不是每次查詢都做）。ZSCORE 順便給出到期
// 秒數，於是一次往返同時得到「是否封鎖」與「到時候」—— 而後者是回應
// Retry-After 表頭所需要的。
//
// 回傳的 until 在未封鎖時是零值。err 非 nil 時 banned 一定是 false：呼叫端
// 應該 fail open（見檔頭的失敗行為說明）。
func (s *Store) IsBanned(ctx context.Context, ip string) (bool, time.Time, error) {
	if s == nil || s.rdb == nil {
		return false, time.Time{}, ErrNoStore
	}
	score, err := s.rdb.ZScore(ctx, banSetKey, ip).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 名單裡沒有這個 IP —— 這不是錯誤，是絕大多數請求的結果。
			return false, time.Time{}, nil
		}
		return false, time.Time{}, fmt.Errorf("check ban: %w", err)
	}
	// score <= now 代表已過期。回 false 而不是 true：名單裡的項目在清理之前
	// 會一直存在，因此「查到有」不等於「仍然有效」。
	if int64(score) <= s.now().Unix() {
		return false, time.Time{}, nil
	}
	return true, time.Unix(int64(score), 0).UTC(), nil
}

// Ban 封鎖某個 IP 直到 until。
//
// 這個函式**不**接受「誰封的」與「原因」：那兩個值只進稽核紀錄，理由見下面
// 的「刻意不存」。它們由呼叫端（handler）在同一個請求裡傳給稽核，因此「封鎖」
// 與「記錄」是同一個動作的兩面，不會出現其中一面被忘記寫的情況。
//
// 重複封鎖是**更新到期時間**而不是報錯：管理員對同一個 IP 再次封鎖，合理的
// 直覺是「延長」而不是「失敗」。
func (s *Store) Ban(ctx context.Context, ip string, until time.Time) error {
	if s == nil || s.rdb == nil {
		return ErrNoStore
	}
	ip = NormalizeIP(ip)
	if ip == "" {
		return ErrInvalidIP
	}
	if until.After(s.now().Add(maxBanDuration)) {
		return fmt.Errorf("ipban: 封鎖不可超過 %v", maxBanDuration)
	}
	if until.Before(s.now().Add(minBanDuration / 2)) {
		return fmt.Errorf("ipban: 封鎖至少需 %v", minBanDuration)
	}
	// 順手清掉已過期的項目。這讓清理不必只依賴背景 goroutine：即使那個
	// goroutine 還沒跑過第一次，名單也不會在累積過期項。
	if err := s.rdb.ZRemRangeByScore(ctx, banSetKey, "-inf", strconv.FormatInt(s.now().Unix(), 10)).Err(); err != nil {
		return fmt.Errorf("prune bans: %w", err)
	}
	if err := s.rdb.ZAdd(ctx, banSetKey, redis.Z{Score: float64(until.Unix()), Member: ip}).Err(); err != nil {
		return fmt.Errorf("ban ip: %w", err)
	}
	return nil
}

// Unban 解除某個 IP 的封鎖，回傳它原本是否真的被封著。
//
// 回傳 existed 是為了讓後端能寫出準確的稽核紀錄：「解除了一個不存在的封鎖」
// 與「解除了一個存在的封鎖」是兩件事，而後者才需要管理員知道自己剛才的
// 動作有效果。
func (s *Store) Unban(ctx context.Context, ip string) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, ErrNoStore
	}
	ip = NormalizeIP(ip)
	if ip == "" {
		return false, ErrInvalidIP
	}
	removed, err := s.rdb.ZRem(ctx, banSetKey, ip).Result()
	if err != nil {
		return false, fmt.Errorf("unban ip: %w", err)
	}
	return removed > 0, nil
}

// List 列出目前有效的封鎖，依到期時間由近到遠。
//
// 「有效」是查詢時過濾的，而不是假設名單裡都是有效的：清理是週期性的
// （見 Prune），因此在一個清理間隔之內，名單裡會有已過期的項目。
//
// limit <= 0 時列出**前 200 筆**而不是全部。這個上限在這裡而不是呼叫端：
// 「呼叫端決定」的版本依賴每一個呼叫端都記得傳一個明智的值，而漏傳時的症狀是
// 後臺端點把整份封鎖名單拉回來 —— 那不會出錯，只會讓管理頁變慢。200 遠大於
// 任何人為維護的封鎖名單，而真的更多的時候畫面也已經看不完了。
func (s *Store) List(ctx context.Context, limit int) ([]Entry, error) {
	if s == nil || s.rdb == nil {
		return []Entry{}, ErrNoStore
	}
	if limit <= 0 {
		limit = 200
	}
	nowUnix := s.now().Unix()
	// ZRANGEBYSCORE 只取「還沒到期」那段（min = now+1），因此過濾在 Redis
	// 端就完成了；分頁限制同樣由 Redis 套用，不會把整個名單拉回來再切。
	// WITHSCORES 讓到期秒數與 member 一起回來。
	items, err := s.rdb.ZRangeByScoreWithScores(ctx, banSetKey, &redis.ZRangeBy{
		Min:    strconv.FormatInt(nowUnix+1, 10),
		Max:    "+inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("list bans: %w", err)
	}
	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		member, ok := item.Member.(string)
		if !ok {
			// 不該發生（member 一律以字串加入），但型別斷言失敗時寧可
			// 跳過一列也不要讓整個列表 500。
			continue
		}
		entries = append(entries, Entry{IP: member, Expires: time.Unix(int64(item.Score), 0).UTC()})
	}
	return entries, nil
}

// Prune 移除已過期的封鎖，回傳刪除的筆數。
//
// 呼叫者是背景 goroutine（見 Pruner）。Ban 也會順手清一次，因此這個
// goroutine 存在的原因是處理「沒有人再封任何東西」的情況 —— 一個只會在
// Ban 裡清理的設計，會讓最後一批封鎖永遠留在名單裡。
func (s *Store) Prune(ctx context.Context) (int, error) {
	if s == nil || s.rdb == nil {
		return 0, ErrNoStore
	}
	removed, err := s.rdb.ZRemRangeByScore(ctx, banSetKey, "-inf", strconv.FormatInt(s.now().Unix(), 10)).Result()
	if err != nil {
		return 0, fmt.Errorf("prune bans: %w", err)
	}
	return int(removed), nil
}

// Count 回傳名單中「目前仍有效」的筆數，包含已過期但尚未清理的項目。
//
// 這是給監控頁用的，而它的語意要說清楚：它是「名單大小」而不是「被封的人數」。
// 兩者的差別在於前者可能因為清理間隔而偏高，而後者要用 List 數。
func (s *Store) Count(ctx context.Context) (int64, error) {
	if s == nil || s.rdb == nil {
		return 0, ErrNoStore
	}
	return s.rdb.ZCard(ctx, banSetKey).Result()
}

/*
兩個刻意不做的功能

	1. 永久封鎖。score 是到期秒數，「永久」只能寫成一個極大的數字，而那種封鎖
	   沒有辦法靠時間自動解除（見 maxBanDuration 的說明）。

	2. 自動封鎖（超過 N 次就封）。理由寫在檔頭的「與限流的分工」：自動封鎖會
	   誤傷 NAT 後面共用出口的正常使用者，而誤封正常使用者的後果比多讓一個腳本
	   多打幾次嚴重。封鎖必須是管理員的決定。
*/

// Pruner 定期清理過期封鎖的背景 worker。
type Pruner struct {
	store    *Store
	interval time.Duration
}

// NewPruner 建立清理器。interval <= 0 時用一小時。
//
// 一小時的取捨：過期項目在清理之前會留在名單裡，而 IsBanned 會正確地把它們
// 視為「已過期」（見它的說明），因此清理間隔只影響「ZRANGE 的結果裡有多少
// 雜訊」與記憶體，不影響正確性。一小時讓磁碟與網路上的寫入量可以忽略。
func NewPruner(store *Store, interval time.Duration) *Pruner {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Pruner{store: store, interval: interval}
}

// Run 定期清理直到 ctx 被取消。
//
// 與 metrics 的 flusher 一樣用 context 而非 Stop()：main 的優雅停止流程
// （shutdown.go）只會取消一個 context，讓所有背景工作都靠它結束。
// ctx 取消時直接返回、不做最後一次清理：過期項目在查詢時本來就被視為未封鎖，
// 因此漏掉最後一次清理不會造成封鎖失效（見 NewPruner 的說明）。
func (p *Pruner) Run(ctx context.Context, removed func(int, error)) {
	if p == nil || p.store == nil {
		return
	}
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := p.store.Prune(ctx)
			if removed != nil {
				removed(count, err)
			}
		}
	}
}

// NormalizeIP 修剪並驗證一個 IP 字串。
//
// 驗證只用 net.ParseIP（不支援 CIDR 與範圍）—— 封鎖名單的 key 是單一 IP，
// 而接受「一段範圍」會讓「這個 IP 被封了嗎」變成一個需要逐一比對的問題。
// CIDR 封鎖是另一個功能，而它的實作方式（把範圍展開或改用另一套資料結構）
// 不該由這個函式順手決定。
//
// 回傳修剪後的字串；不合法時回空字串。
func NormalizeIP(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.ContainsAny(trimmed, " \t\r\n/") {
		// 含有空白或斜線 → 不是單一 IP（常見的是誤貼了 CIDR 或整段 XFF）。
		return ""
	}
	if !isValidIP(trimmed) {
		return ""
	}
	return trimmed
}

// isValidIP 判斷字串是否為合法的 IPv4 或 IPv6 位址。
//
// 獨立成函式而不是讓呼叫端直接用 net.ParseIP：這個套件的呼叫端是 HTTP handler，
// 它們需要的是「合法 / 不合法」而不是「解析結果與錯誤」。讓 net 套件的錯誤字串
// （"invalid IP address: 1.2.3"）漏到使用者面前會是英文的內部細節。
func isValidIP(value string) bool {
	return net.ParseIP(value) != nil
}
