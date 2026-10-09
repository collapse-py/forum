/*
ratelimit.go：這個服務自己的每 IP 計價器。

【為什麼需要它】
files_server 在很長一段時間裡**完全沒有限流**，而它有三類成本差異極大的端點：

	/upload   一次 50 MiB 的 multipart 解析 + 寫入磁碟
	/delete   一次檔案系統移除
	/files/*  一次 Redis EXISTS（每個靜態檔案請求都會發生）

後端有自己的限流器（backend/forum/httpapi/ratelimit.go），但它保護不到這個程序。
而 /files/ 是最脆弱的一條：媒體 token 驗證的失敗路徑過去還會為每一次失敗寫一行
日誌，因此「用隨機 token 打 /files/」同時消耗 Redis、耗磁碟（日誌），而且量級
完全由攻擊者決定 —— 沒有任何上限。

【為什麼不用 golang.org/x/time/rate】
那會引入一個外部相依，而這個服務現在的相依只有 go-redis 與標準函式庫
（見 go.mod）。權杖桶本身約 40 行，且這裡需要的功能只有一個：`allow(key) bool`。

【為什麼是記憶體而不是 Redis】
Redis 在這個程序裡已經代表「媒體 token 的那個儲存」，而限流是「這個程序自己的
保護」：Redis 掛掉時限流不該一起失效（fail-open），也不該讓每一個請求多一次
網路往返。多實例部署時每個實例各限各的 —— 那讓總上限變成 N 倍，但它仍然是一個
**上限**，而這正是這段程式要達成的唯一目標。

【為什麼限制鍵是 IP】
這個服務是內部端點：真正的呼叫端是論壇後端與使用者的瀏覽器。RemoteAddr 不經過
代理剝殼，因此它比「X-Forwarded-For 第一段」可信（後者任何呼叫端都能自己填）。
不支援信任代理白名單是刻意的簡化：這個服務的規模不值得為它引入一份設定。

【失敗模式】
被限制時回 429 並附 Retry-After。刻意不用 403：那看起來像「你沒有權限」，
而這裡想說的是「你太快了，等一下」。
*/

package main

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimiter 是一個每鍵一個權杖桶的計價器。
//
// 時間基准刻意是「惰性補充」而不是背景 goroutine：這個程式的哲學是不在建構子裡
// 啟動任何背景工作（見 backend/forum/httpapi 同一段理由），而權杖桶只需要在
// 被問到的時候算一次就夠了。
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	// rate 是每秒補充的權杖數，burst 是桶的容量（也是「一次可以連發幾個」的上限）。
	rate  float64
	burst float64
	// maxBuckets 是記憶體的使用上界。到達時先做一次惰性清除，因此它同時也是
	// 「多少個不同 IP 之後開始回收」的觸發點。
	maxBuckets int
	// idleFor 是一個桶多久沒被碰到就視為可回收。刻意遠大於任何合理的重訪間隔。
	idleFor time.Duration
}

// bucket 是單一鍵的狀態。
type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// newRateLimiter 建立一個計價器。rate 為 0 或負值代表「不限制」。
func newRateLimiter(rate float64, burst int, maxBuckets int) *rateLimiter {
	if rate <= 0 {
		return nil
	}
	if burst < 1 {
		burst = 1
	}
	return &rateLimiter{
		buckets:    map[string]*bucket{},
		rate:       rate,
		burst:      float64(burst),
		maxBuckets: maxBuckets,
		idleFor:    10 * time.Minute,
	}
}

// allow 消耗一個權杖，回傳是否放行。
//
// nil 接收者代表「這個端點不限制」：呼叫端因此不必在每一處都判斷計價器是否存在
// （少一個會忘記判斷的地方，而漏掉判斷的症狀是整個保護靜默失效）。
func (l *rateLimiter) allow(key string) bool {
	if l == nil {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buckets) >= l.maxBuckets {
		l.sweepLocked(now)
	}

	// 惰性補充：把「距上次補充的時間」換算成權杖，上限是 burst。時間来源是
	// 請求本身的時間戳，因此不需要背景 goroutine 也不能被時鐘跳動放大
	// （now.Sub(lastSeen) 只有在 lastSeen 是過去時才為正）。
	b, ok := l.buckets[key]
	if !ok {
		// 第一個請求從滿桶開始並直接消耗一個：一個新來的 IP 不該先被懲罰一次，
		// 但也不該因此多拿到一個免費的請求。
		l.buckets[key] = &bucket{tokens: l.burst - 1, lastSeen: now}
		return true
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.lastSeen).Seconds()*l.rate)
	b.lastSeen = now
	return b.take()
}

// take 消耗一個權杖；不足時回 false。呼叫端必須持有鎖。
func (b *bucket) take() bool {
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweepLocked 回收閒置的桶。呼叫端必須持有鎖。
func (l *rateLimiter) sweepLocked(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > l.idleFor {
			delete(l.buckets, key)
		}
	}
}

// middleware 把計價器包成中介層。nil 計價器時原樣放行（見 allow 的說明）。
func (l *rateLimiter) middleware(next http.Handler) http.Handler {
	if l == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientKey(r)) {
			// 回「還要等幾秒」：rate 是每秒補充數，因此它同時就是等待一秒之後
			// 大約會有多少權杖可用。向上取整，因為 0 會讓呼叫端立刻重試。
			seconds := int(math.Ceil(l.rate))
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientKey 是計價的鍵。只用 RemoteAddr（含埠），理由見檔頭。
func clientKey(r *http.Request) string {
	return r.RemoteAddr
}
