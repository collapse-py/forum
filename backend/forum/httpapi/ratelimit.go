package httpapi

/*
本檔案實作以「每個用戶端 IP 為單位」的記憶體限流，供中介層擋掉過度頻繁的請求。

演算法選擇：滑動視窗日誌（sliding window log）
  - 固定視窗（fixed window）實作最簡單，但存在跨視窗邊界的 2 倍突發：
    視窗結尾打滿 limit、下一秒是新視窗又能再打滿，實際上短時間內可達兩倍。
  - 令牌桶（token bucket）能平滑輸出並允許一定程度的突發，但每個用戶端
    仍需保存浮點狀態，且難以給出「任意連續 window 內不超過 limit 次」這個
    容易向用戶說明、也容易稽核的保證。
  - 本檔案的做法是替每個用戶端保存「最近 window 內各次請求的時間戳」，
    每次請求先剔除 cutoff 之前的紀錄，再看剩餘筆數是否已達 limit。
    優點是判定精確（無邊界突發）、不需要浮點運算；代價是每個用戶端要保存
    最多 limit 筆時間戳，且每次判定是 O(limit)。
  - 預設 limit 只有 10，因此這個代價在實務上非常小。

資料結構與併發安全
  - hits 是 map[key][]time.Time，key 為用戶端識別值，slice 依時間由舊到新排列。
  - Allow 對同一個 key 做「剔除 → 計數 → 追加」三個步驟，這是一個不可分割的
    讀取‑修改‑寫入。若沒有鎖，兩個同時進來的請求可能都讀到尚未達上限的長度
    而同時被放行，限流就被繞過了。因此 mu 涵蓋整個判定區段。
  - 用 sync.Mutex 而非 RWMutex：每次 Allow 都必然寫入 map，讀鎖帶來不了任何
    好處，反而增加額外成本。
  - 鎖是行程層級的，不跨行程。負載平衡器後面若有 N 個執行個體（例如
    Kubernetes Pod），實際上限會是 limit × N。要做到全站一致就得把計數器
    放進 Redis，但那會讓每個請求多一次網路往返，與本檔案「純記憶體」的
    取捨相反。

記憶體無上限成長的風險與緩解
  - hits 本身若沒有淘汰機制，map 的項目永遠不會被刪除：某個 key 的時間戳會在
    window 過去後被剔除並縮回 0 筆，但「key → 空 slice」這個項目本身還在。
    長時間執行、且持續接觸大量不同來源位址的話（例如公開入口被爬蟲掃過），
    map 會持續成長。這對論壇這類公開入口尤其明顯，因為 clientIP 優先採信
    X-Forwarded-For —— 見 clientIP 內的說明。
  - 緩解方式是 Cleanup：定期掃描並刪除已無資料的項目。它同時把仍然有效的
    slice 就地縮短，讓 map 的值不會長期停在過期內容上。
  - 由 StartCleanup 以背景 goroutine 定期呼叫；interval 由呼叫端決定，
    通常設為與 window 同級的時間尺度即可（每分鐘掃一次對應一分鐘的視窗）。
  - 殘餘風險：攻擊者若能旋轉來源位址，仍可以在一次掃描間隔內塞入大量新 key。
    這是「掃描式清理」的固有極限，徹底解決需要改用有 TTL 的外部儲存（Redis）。

時鐘
  - Allow 與 Cleanup 都透過 rl.now() 取得當前時間，預設指向 time.Now。
  - 抽出這個欄位（而不是直接呼叫 time.Now）純粹為了測試：限流是「時間往前
    走就會放行」的邏輯，注入假時鐘才能把視窗邊界行為寫成確定性的單元測試，
    不必真的等待一分鐘。setClock 就是為此存在，正式程式碼不應呼叫它。
  - now 與 cutoff 在進鎖之前算好：window 通常很短，這點時間差可忽略，
    而把時間計算留在鎖外能縮短臨界區。

per-client 而不是全域
  - 全域計數器會讓單一來源打光所有人的額度（noisy neighbor），一個惡意或
    誤寫的腳本就能讓整站被擋。
  - 改為 per-client 後，單一用戶端只會影響自己。代價是 key 空間無上限
    （見上），且攻擊者只要輪換來源位址就能重置自己的額度。
  - 已知取捨：以 IP 為 key 代表同一個 NAT／辦公室出口後的所有使用者共享
    一份額度。對「註冊 Google 帳號才能發文」的論壇而言影響有限（正常使用
    不會在分鐘內連續發十篇），因此保留簡單的 IP 維度；真要改以登入身分為
    key，Middleware 需改為接受自訂 key 函式（見 Middleware 的說明）。

掛載位置
  - 限流只套在高成本端點：內容寫入（發文、留言、按讚、檢舉）、圖片上傳
    （會呼叫外部檔案伺服器並在 Redis 建立存取 token）、以及 OAuth 登入跳轉。
  - 讀取端點（GET /api/forum/posts、留言列表、靜態資產）刻意不掛：它們是
    匿名訪客也要能用的公開內容，若一併限流，使用者按「載入更多」就會被擋。
  - 實際掛在哪幾條路由上由 server.go 決定，限流器本身對路由一無所知。
*/

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter 是以滑動視窗日誌為基礎的行程內限流器，欄位語意如下：
//
//	mu     保護 hits 與 allowed/blocked 計數的鎖。鎖定範圍是 Allow 與 Cleanup
//	       中「剔除過期 → 計數 → 追加」的整段，不可只鎖其中一步，否則會出現
//	       同時放行超過上限的情形。
//	hits   用戶端 key 到「最近 window 內各次請求時間戳」的對應，slice 依
//	       時間由舊到新排列。剔除時複用同一個底層陣列（times[:0]），因此
//	       每次請求不會造成新的配置。
//	limit  單一視窗內允許的最大請求數。
//	window 滑動視窗的長度。
//	now    取當前時間的函式，建構後固定指向 time.Now，除非測試以 setClock 換掉。
//	allowed/blocked 允許與被拒絕的累計次數，供監控頁顯示。它們與 hits 共用
//	       同一把鎖而不是用 atomics：允許／阻擋的計數只在 Allow 內發生，而
//	       Allow 本來就必須持有 mu，因此多一對 atomics 只會讓「這兩個數字
//	       與 hits 是否一致」變成一個需要推理的問題。
//
// limit、window 與 now 三個欄位在建構後皆不再變更（setClock 僅限測試使用），
// 可安全地被多個 goroutine 併發讀取。hits、allowed/blocked 與 mu 則由所有
// 存取路徑的鎖保護。
type RateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	allowed uint64
	blocked uint64
	limit   int
	window  time.Duration
	now     func() time.Time
}

// LimiterStats 是限流器可供觀察的累計計數。
type LimiterStats struct {
	// Allowed 是本次啟動以來被放行的請求數。
	Allowed uint64
	// Blocked 是本次啟動以來因超出額度而被拒絕（回 429）的請求數。
	Blocked uint64
}

// Stats 回傳目前的累計計數，並以一份快照的形式一次取得兩個值。
//
// 兩個值刻意在同一把鎖內讀出：分開呼叫兩次會得到「可能對不上」的組合
// （允許數已更新、阻擋數還沒），而儀表板上並排顯示的兩個數字必須彼此一致。
func (rl *RateLimiter) Stats() LimiterStats {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return LimiterStats{Allowed: rl.allowed, Blocked: rl.blocked}
}

// Limit 回傳單一視窗內允許的最大請求數。建構後固定不變，可無鎖讀取。
func (rl *RateLimiter) Limit() int {
	return rl.limit
}

// Window 回傳滑動視窗長度。建構後固定不變，可無鎖讀取。
func (rl *RateLimiter) Window() time.Duration {
	return rl.window
}

// TrackedKeys 回傳目前 map 中的 key 數量，並在讀取期間持有 mu。
//
// 存在的理由：StartCleanup 會讓背景 goroutine 與呼叫端同時觸及 hits。
// 測試若直接讀 len(rl.hits) 就是未同步的存取，`go test -race` 會報出資料
// 競爭 —— 而且那不是誤報，是真的競爭。正式程式碼若要觀察限流狀態（例如
// 監控頁的 metrics）也應走這個方法，不要直接碰 hits。
//
// 這個數字是「記憶體裡正在被追蹤的用戶端數」，不是「曾經出現過的使用者數」：
// 已無視窗內紀錄的 key 會被 Cleanup 刪掉，因此它同時也是限流器記憶體佔用
// 的直接指標 —— 這個值一路上升就代表 Cleanup 沒跟上（見檔頭的成長說明）。
func (rl *RateLimiter) TrackedKeys() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.hits)
}

// NewRateLimiter 建立限流器。limit 與 window 若為非正值會被改用預設值
// （10 次 / 1 分鐘），避免設定缺漏或解析失敗時變成「全部放行」或「全部拒絕」
// —— 尤其 limit <= 0 會讓每個請求的計數都 >= limit，等於整站被封鎖。
// 這個函式不做任何 I/O，也不啟動背景 goroutine；定期清理要另外呼叫 StartCleanup。
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		hits:   make(map[string][]time.Time),
		limit:  limit,
		window: window,
		// 明確指定 time.Now 而非留 nil：Allow 與 Cleanup 會無條件呼叫 rl.now()，
		// 留 nil 會在第一個請求就 panic。
		now: time.Now,
	}
}

// setClock 替換取時間的函式，只為讓測試能確定性地推進視窗邊界。
// 非測試用途：正式程式碼不應呼叫它，換掉時鐘等於讓限流在真實流量下失效。
func (rl *RateLimiter) setClock(now func() time.Time) {
	rl.now = now
}

// Allow 記錄一次針對 key 的請求，回傳是否允許以及「若被拒絕，還要等多久可以再試」。
//
// 回傳值語意：
//   - allowed 為 true 時，retryAfter 固定為 0（此時呼叫端不應使用它）。
//   - allowed 為 false 時，retryAfter 是距離「視窗內最老一筆記錄滑出」還有的
//     時間，也就是呼叫端最快可以再次嘗試的時刻。它是精確值而非上限估計，
//     因為滑動視窗的判定完全由最老的那筆決定。
//
// 時間複雜度 O(該用戶端在視窗內的紀錄數)，上限為 limit。
// 副作用：會寫入 hits，必須在持有 mu 的情況下呼叫（此方法自行加鎖）。
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	now := rl.now()
	// cutoff 是「仍算在視窗內」的分界點：以 ts > cutoff 判斷，落在分界點
	// 上的紀錄視為已過期。
	cutoff := now.Add(-rl.window)

	// 整段判定都在鎖內，保證剔除、計數與追加對其他 goroutine 是原子的。
	// defer 而非手動 Unlock：即使日後在這段加入會 panic 的程式碼，鎖仍會釋放。
	rl.mu.Lock()
	defer rl.mu.Unlock()

	times := rl.hits[key]
	kept := pruneExpired(times, cutoff)
	// 達到上限就拒絕。仍須把剔除後的 slice 寫回：否則 map 裡會一直留著
	// 過期紀錄，既重複掃描又佔用記憶體。
	if len(kept) >= rl.limit {
		rl.hits[key] = kept
		// kept[0] 必然存在：len(kept) >= rl.limit 且 rl.limit 至少為 1
		//（NewRateLimiter 已兜底）。留一個防禦性分支只是避免日後有人放寬
		// limit 的下限時在這裡 panic。
		if len(kept) == 0 {
			rl.blocked++
			return false, rl.window
		}
		// 最早滑出視窗的時刻 = 最老一筆記錄 + window。減去 now 就是還要等多久。
		// 這個差值恆為正：kept 裡每一筆都滿足 ts.After(cutoff)，而 cutoff = now - window。
		rl.blocked++
		return false, kept[0].Add(rl.window).Sub(now)
	}
	// 通過時才追加本次時間戳。被拒絕的請求不記錄，因此不會延長封鎖時間 ——
	// 使用者最多等過最早的紀錄滑出視窗就可再次通過，不會因持續重試而被
	// 永久鎖住。
	rl.hits[key] = append(kept, now)
	rl.allowed++
	return true, 0
}

// Cleanup 移除所有已無視窗內紀錄的 key，並就地縮短仍有資料的 slice。
// 回傳被刪除的項目數，供呼叫端記錄日誌或讓測試斷言。
//
// 之所以需要它：Allow 只會把某個 key 的 slice 縮到空，卻不會把 key 從 map 刪掉。
// 對一個長期運作、面對不斷輪替來源位址的公開入口，這會讓 map 只增不減。
//
// 整段掃描都在鎖內，期間會阻擋所有 Allow。單一 key 的 slice 最多 limit 筆
// （預設 10），因此總成本是 O(len(hits) × limit)；interval 設成與 window
// 同級（一分鐘量級）時，這段停滯遠小於兩次請求之間的自然間隔。
func (rl *RateLimiter) Cleanup() int {
	now := rl.now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	removed := 0
	// 迭代中刪除 map 的項目是 Go 明文允許的：被刪掉的 key 不會再被後續迭代
	// 取出，因此不需要另外收集要刪的清單。
	for key, times := range rl.hits {
		kept := pruneExpired(times, cutoff)
		if len(kept) == 0 {
			delete(rl.hits, key)
			removed++
			continue
		}
		// 只有真的縮短過才寫回，否則每次掃描都會無謂地產生一次 map 賦值。
		// 判斷方式是比較長度：pruneExpired 只做就地搬移，長度不變代表
		// 沒有任何元素被剔除。
		if len(kept) != len(times) {
			rl.hits[key] = kept
		}
	}
	return removed
}

// StartCleanup 以背景 goroutine 定期呼叫 Cleanup，直到 ctx 被取消。
//
// 為什麼用 ctx 而非單純的 time.Ticker：main 目前沒有 graceful shutdown
// （見 main.go 的說明），但把取消條件綁在 context 上代表未來接上
// signal.Notify 時，清理 goroutine 能自動跟著停下，不必再改這裡。
//
// interval <= 0 時退回一分鐘。這個函式立刻返回，不會阻塞呼叫端；
// ctx 已取消時它直接返回而不啟動 goroutine，避免關閉流程中反而多一個
// 永遠不會退出的執行緒。
func (rl *RateLimiter) StartCleanup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	if ctx.Err() != nil {
		return
	}
	go func() {
		// defer ticker.Stop() 而非在 select 各分支個別呼叫：無論從哪一條路徑
		// 離開迴圈，ticker 都一定被釋放。
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rl.Cleanup()
			}
		}
	}()
}

// pruneExpired 就地剔除 times 中不晚於 cutoff 的紀錄，回傳保留下來的前綴切片。
//
// 「就地」是重點：kept 與 times 共用同一個底層陣列，只是把要留的元素搬到前面
// 並把長度縮短，因此不會配置新 slice。這是安全的，因為呼叫端（Allow、Cleanup）
// 都在持有 mu 的情況下呼叫它，且元素只往前搬、不會被讀到尚未寫好的位置。
//
// 之所以獨立成函式：Allow 與 Cleanup 都需要這段完全相同的邏輯，複製兩份
// 只會讓未來修正其中一份時漏掉另一份。
func pruneExpired(times []time.Time, cutoff time.Time) []time.Time {
	kept := times[:0]
	// 由舊到新掃描，剔除所有早於 cutoff 的紀錄。時間戳本來就是遞增的，
	// 理論上可在遇到第一筆過期時就 break，但此處保留完整掃描以確保健壯。
	for _, ts := range times {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	return kept
}

// Middleware 是限流用的中介層：來源超出額度時以 429 拒絕且不呼叫 next。
//
// key 目前固定採 clientIP(r)。若日後要改成以登入身分為 key（例如放寬 NAT
// 共用出口的額度），把這裡換成一個接受 *http.Request 的函式即可，Allow 本身
// 對 key 的語意完全不敏感。
func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 以用戶端 IP 作為額度的分攤單位，理由見檔頭的 per-client 說明。
		key := clientIP(r)
		allowed, retryAfter := rl.Allow(key)
		if !allowed {
			// Retry-After（RFC 9110 §10.2.3）告知用戶端多久後可以再試，
			// 讓遵循標準的爬蟲與瀏覽器擴充套件能自動退避，而不是反覆重試
			// 讓封鎖持續延長。
			// 向上取整到整秒：該表頭的 delta-seconds 形式只接受整數，直接
			// 取整會讓宣告的等待時間比實際短到不足一秒。前面的 +1s-1ns 與
			// 後面的 /1s 都是整數運算，不經浮點，因此不會有四捨五入誤差。
			seconds := int((retryAfter + time.Second - 1) / time.Second)
			// 下限保護：Allow 在理論上保證 retryAfter > 0，但若時鐘被設成
			// 單調遞減的自訂來源（測試常見），取整後仍可能得到 0。宣告
			// Retry-After: 0 等於叫客戶端立刻重試，會讓 429 變成忙迴圈。
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			// 429 沿用 writeError，讓前端拿到與其他錯誤一致的結構。
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next(w, r)
	}
}

// clientIP 決定限流計數用的用戶端識別值，信任順序為 X-Forwarded-For 最左一項
// → X-Real-IP → RemoteAddr（去掉埠號）。
// 這是純函式，不做 I/O。
//
// 安全假設與限制：X-Forwarded-For 與 X-Real-IP 都是可由用戶端設定的表頭，
// 只有在「 本站前面確實有一道會覆寫這些表頭的代理，且使用者無法繞過它直連」
// 時，裡面的值才可信。當服務可被直連、或代理採用附加而非覆寫的寫法時，
// 攻擊者可以偽造來源來取得新的額度（限流被繞過），甚至用隨機值把 hits 撐大
// （見檔頭的記憶體成長說明）。因此本檔案的安全性依賴部署方式的假設，而非
// 程式本身的保證。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取最左一項：XFF 是由左往右「代理逐層附加」，最左邊才是原始的發起者；
		// 越靠右越接近本站、可信度越高。
		parts := splitComma(xff)
		// 若 XFF 全是逗號或空白，splitComma 會回傳長度 0 的切片，這時
		// 不應回傳空字串當作 key，否則所有這類請求會共用同一個額度。
		if len(parts) > 0 {
			return parts[0]
		}
	}
	// 部分反向代理（Nginx 預設）會設定 X-Real-IP，作為第二順位來源。
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// 最後退回 RemoteAddr（net/http 保證是 "host:port"），去掉埠號只留主機。
	ip := r.RemoteAddr
	// 從「最後一個冒號」切斷，因此 [::1]:8080 這種含冒號的 IPv6 也能正確
	// 去掉埠號。限制：若 RemoteAddr 是不含埠號的裸 IPv6（例如 "::1"），
	// 會被截成 ":"。此值只作為 map 的鍵使用，只要一致即可，不影響正確性。
	if idx := lastIndexByte(ip, ':'); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// splitComma 以逗號切分表頭值，只保留非空片段。
// 自行實作而非使用 strings.Split：後者會配置包含所有片段（含空字串）的
// 切片，而這裡只用到第一項，逐一掃描即可，省下配置與垃圾。
func splitComma(s string) []string {
	// cap 2 是對「代理層數」的猜測；XFF 片段更多時切片會自動擴充。
	parts := make([]string, 0, 2)
	start := 0
	// 迴圈條件用 i <= len(s) 而非 <，讓 i == len(s) 的情況（字串結尾剛好是
	// 逗號，或沒有尾端逗號的最後一段）也能被處理到。
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			// 略過空片段，因此 "a,,b" 得到 2 項而不是 3 項。
			if part != "" {
				parts = append(parts, part)
			}
			// 跳過逗號本身，讓下一段從 i + 1 開始。
			start = i + 1
		}
	}
	return parts
}

// trimSpace 去除字串兩端與中間的空白（僅限半形空格與水平 tab）。
// 不用 strings.TrimSpace：後者依 Unicode 規範還會處理 \n、\r、\v、\f 等
// 空白字元並逐字元解碼，而表頭值在實務上只會有空格與 tab，逐位元組處理
// 既正確又省事。兩端各用一個迴圈，條件都先檢查長度以免空字串出錯。
func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// lastIndexByte 從字串結尾往前找出最後一個符合的位元組位置，找不到回傳 -1。
// 以位元組而非字元為單位，不需處理 UTF-8；呼叫端依 -1 這個慣例決定是否
// 保留原字串。
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
