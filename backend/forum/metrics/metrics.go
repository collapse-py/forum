/*
Package metrics 收集後端的請求統計，供後臺監控頁（/admin/monitor）顯示。

【對外介面】

	New / Registry             統計容器
	(*Registry).Observe        記錄一次已完成的請求
	(*Registry).Middleware     產生記錄用中介層
	(*Registry).SetLimitStats  回報限流器的允許／阻擋計數
	(*Registry).Snapshot       取一份可安全 JSON 序列化的快照
	(*Registry).StartFlusher   定期把分鐘桶寫進 MySQL
	(*Registry).LoadHistory    啟動時從 MySQL 讀回既有歷史
	NormalizeRoute             把具體路徑收斂成低基數的路由樣式
	LimitStat                  單一限流器要回報的計數

【為什麼路徑必須正規化】

	/api/forum/posts/17/comments 與 /api/forum/posts/9182/comments 是同一條路由。
	若直接以 r.URL.Path 當 key，攻擊者只要不斷亂數測試 id，map 就會無上限成長
	（與 ratelimit.go 的 hits map 同一類問題），而且「最忙的路由」永遠看不到
	—— 會被上千個只出現一次的 id 稀釋掉。因此每段純數字或含 @ 的片段都換成
	:id，靜態資產換成 *，並且對總數設一個上限，超過後的新 key 併入 __other__：
	寧可少一條統計，也不能讓監控自己變成記憶體洩漏。

【延遲為什麼是直方圖而不是平均值】

	平均值會被幾次慢查詢拉高，看不出「多數請求其實很快」；而把所有耗時都存起來
	（精確分位數）又會讓記憶體隨流量線性成長。本檔案採 Prometheus 同一套做法：
	固定邊界的累積直方圖，p50 / p95 / p99 是「該分位數落在哪一格」的上界值。
	因此顯示出來的是離散值（例如 p95 恆為 250ms 或 500ms），這是刻意的取捨，
	介面上必須照實標示為分位數上界，而不是假裝是精確值。

【分鐘桶與資料庫】

	記憶體內只保留最近 windowMinutes 個分鐘的桶（預設 120），寫進
	forum_request_metrics 的內容保留 retentionHours 小時（預設 24，於匯出時
	一併刪除過期列）。

	只寫「已結束的分鐘」（minute < 當前分鐘），而且同一個桶只寫一次。這兩條
	限制合起來讓寫入語意變得非常單純：

	  - 已結束的分鐘不再會有新請求進來，因此寫出去的數值是該分鐘的最終值，
	    不需要「上次寫到哪」的差值帳本，也就不會有累加兩次的風險。
	  - 寫入仍使用 INSERT ... ON DUPLICATE KEY UPDATE 的「累加」語意，這是
	    為了多執行個體：負載平衡器後面若有 N 個執行個體，各自會寫自己看到的
	    那一份，同一分鐘的 N 筆加總起來才是全站總量。若改成覆寫，N 個個體會
	    互相蓋掉彼此，數字看起來正常但其實只剩其中一台的量。

	代價是：行程非正常結束時，尚未寫出的當前分鐘（最多 60 秒）會消失，且
	尚未輪到的已結束分鐘要等下一次 tick。兩者都最多是一分鐘的資料，與「監控
	本來就是取樣」這個前提相符。

	歷史與即時是兩份資料：LoadHistory 讀回來的桶放進 history，永遠不會被再次
	寫回（否則會重複累加），Snapshot 才把兩者合併成時間軸。

【併發】

	單一 sync.Mutex 保護所有欄位。請求路徑上只有一次 map 查找與幾個整數加法，
	臨界區極短，因此不需要讀寫鎖，也不值得為了並行度把它拆細 —— 那會讓
	「一次請求的計數是否原子」變成一件需要推理的事。

	inFlight / maxInFlight 例外，用 atomics：它們在每個請求的入口與出口各被
	讀寫一次，是整份統計裡最常被即時讀到的兩個數字。

【不做的東西】

	不記錄請求本文、不記錄 cookie、不記錄 email。監控頁只需要「哪條路由、幾次、
	多慢、錯幾次」，任何個人可識別的資料放進來都會讓這個頁面變成一個新的
	隱私與資料外洩面。
*/
package metrics

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// latencyBounds 是延遲直方圖的累積邊界。
//
// 宣告為固定長度的陣列而不是 slice，只為了讓 len() 在編譯期就是常數
// （overflowIndex 由它推導）。兩者內容與行為完全相同。
//
// 刻意從 1ms 起跳而不是 0：本專案的端點成本差異很大（讀取只是一次索引查詢，
// 上傳要呼叫外部檔案服務），1ms 以下的分佈對維運沒有判讀價值，而把它們與
// 10ms 混在同一格會讓「讀取端點的 p95」看起來像是一個偏慢的數字。
var latencyBounds = [...]time.Duration{
	1 * time.Millisecond,
	2 * time.Millisecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	25 * time.Millisecond,
	50 * time.Millisecond,
	100 * time.Millisecond,
	250 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

// overflowIndex 是直方圖的溢出格位置。>= 最後一個邊界的耗時都落在這裡。
const overflowIndex = len(latencyBounds)

// otherRoute 是路由數量達到上限後，所有新路由被併入的統計槽名稱。
// 它是一個不可能與真實路徑撞名的字串（真實路徑不會以底線開頭）。
const otherRoute = "__other__"

// defaultMaxRoutes 是最多追蹤幾條不同路由的上限。
//
// 這個數字是「正常情況下路由總數」的數量級之上的餘裕：本專案的 ServeMux
// 註冊了約四十條樣式，正規化之後的 distinct key 應該在五十以內。給到 200
// 為了容忍日後新增路由而不必回來改常數。
const defaultMaxRoutes = 200

// defaultWindowMinutes 是記憶體內保留的分鐘桶數量。
//
// 120 分鐘（兩小時）涵蓋「維運在下班前埋一個問題、回來上班看圖」的典型間隔，
// 記憶體成本則是每桶約 56 bytes，可以忽略。
const defaultWindowMinutes = 120

// defaultRetentionHours 是資料庫內分鐘彙總的保留小時數。
//
// 它**只**決定資料庫保留，不決定時間軸長度 —— 後者由 windowMinutes 與
// maxTimelinePoints 決定（見 TimelineMinutes）。兩者刻意分開：保留期是
// 「寫多少進資料庫」的維運政策，時間軸長度是「畫幾格」的呈現選擇。
// 混為一談的後果是設定 retention 的人以為圖會變長，而管理員讀到「24 小時前
// 的資料在圖上」時會去一個永遠不存在的格子裡找。
const defaultRetentionHours = 24

// maxTimelinePoints 限制時間軸回傳的長度。
//
// 前端把它畫成固定寬度的長條圖，回傳比畫面能呈現更多只會讓回應變大。144
// 個點正好是 24 小時的分鐘級。
//
// 它是**呈現上限**，不是時間軸長度：實際長度是 min(windowMinutes,
// maxTimelinePoints)，見 TimelineMinutes。
const maxTimelinePoints = 144

// ErrNoDatabase 表示持久化功能沒有拿到可用的 *sql.DB，因此不會啟動。
//
// 監控統計是診斷工具，缺資料庫只會讓「時間軸沒有重啟前的歷史」，不該讓
// 論壇服務無法啟動 —— 這是它被當成錯誤回傳（而非 panic）的原因。
var ErrNoDatabase = errors.New("metrics: 需要 *sql.DB 才能啟用監控持久化")

// routeStat 是單一路由（method + 樣式化路徑）的累計統計。全部欄位由 mu 保護。
type routeStat struct {
	route        string
	method       string
	total        int64
	clientErrors int64
	serverErrors int64
	durationSum  time.Duration
	maxDuration  time.Duration
	// buckets 是累積直方圖：第 i 格代表耗時 <= latencyBounds[i] 的請求數。
	// 因為是累積的，一次 observe 只需要在第一個「邊界 >= 實測耗時」的位置 +1，
	// 不用回頭修正後面的格子。
	buckets []int64
}

func newRouteStat(method, route string) *routeStat {
	return &routeStat{
		route:   route,
		method:  method,
		buckets: make([]int64, overflowIndex+1),
	}
}

// observe 記錄一次耗時，更新總量、極值與直方圖。
func (s *routeStat) observe(d time.Duration) {
	s.total++
	s.durationSum += d
	if d > s.maxDuration {
		s.maxDuration = d
	}
	s.buckets[bucketIndex(d)]++
}

// bucketIndex 回傳耗時應該累加到哪一格。
//
// 邊界比對用第一個 >= 的位置（含等號），因此一個剛好等於邊界的耗時會落在
// 較小的那一格，這讓「<= 邊界」這個描述與實際累加方式一致。
func bucketIndex(d time.Duration) int {
	for i, bound := range latencyBounds {
		if d <= bound {
			return i
		}
	}
	return overflowIndex
}

// minuteBucket 是單一分鐘的彙總，時間軸與資料庫都以它為單位。
type minuteBucket struct {
	minute        int64 // Unix 分鐘，見 minuteIndex
	total         int64
	clientErrors  int64
	serverErrors  int64
	durationSumMS int64
	// flushed 標記是否已寫進資料庫。已結束的分鐘只寫一次，因此這個旗標
	// 只需要一位元組，但它讓「同一分鐘被寫兩次」變成不可能。
	flushed bool
}

// LimitStat 是單一限流器要回報給監控頁的計數。
//
// 刻意是純資料而非 *httpapi.RateLimiter：限流器在 httpapi 套件內、統計容器
// 在 metrics 套件內，後者若直接持有前者就形成反向相依。用一個純結構中轉，
// 兩邊都不必知道對方的存在。
type LimitStat struct {
	Name    string
	Limit   int
	Window  time.Duration
	Allowed uint64
	Blocked uint64
	// Tracked 為目前記憶體中被追蹤的用戶端數量（hits map 的 key 數）。
	Tracked int
}

// minuteIndex 把時刻換算成「Unix 分鐘」—— 也就是自 1970-01-01T00:00:00Z
// 起算的第幾分鐘。
//
// 之所以不用 time.Time.Unix()：那是「秒」，而這個套件所有以分鐘為 key 的
// 地方（記憶體桶、資料庫主鍵、時間軸）都需要分鐘。兩者差一個 60 倍，把它
// 搞混不會造成編譯錯誤，只會讓整條時間軸落在 5374 年 —— 這個錯誤正是本
// 檔案的測試第一次執行時抓到的。
//
// 對負數時間戳（1970 年之前）這個除法是朝零取整，與 Truncate 的行為不同。
// 本專案不會有那樣的時間戳，因此不為它多加處理。
func minuteIndex(t time.Time) int64 {
	return t.Unix() / 60
}

// minuteTime 把 Unix 分鐘換算回時刻（UTC）。minuteIndex 的逆運算。
func minuteTime(minute int64) time.Time {
	return time.Unix(minute*60, 0).UTC()
}

// Registry 是請求統計的容器。零值不可用，必須由 New 建立。
type Registry struct {
	mu        sync.Mutex
	startedAt time.Time
	now       func() time.Time

	// windowMinutes 是 buckets 保留的分鐘數，也是時間軸長度的來源。
	windowMinutes int
	maxRoutes     int
	// maxClients 與 clientIdleMinutes 守著「每 IP 統計」的記憶體上限，
	// 理由與風險見 clients.go 的檔頭。
	maxClients        int
	clientIdleMinutes int

	routes  map[string]*routeStat
	clients map[string]*clientStat
	// clientsDropped 是「因為達到 maxClients 而被驅逐掉的來源」的累計次數。
	// 它存在的唯一理由是可觀察性：驅逐會讓某個來源的計數整筆消失，而這個
	// 計數器讓「這一頁可能漏了東西」變成介面上看得見的事實。
	clientsDropped int64
	buckets        map[int64]*minuteBucket
	// history 是從資料庫讀回來、只用於顯示的歷史桶。與 buckets 分開是因為
	// 兩者的寫入語意相反：buckets 會被寫進資料庫，history 永遠不會。
	history map[int64]*minuteBucket
	limits  []LimitStat

	total        int64
	clientErrors int64
	serverErrors int64
	durationSum  time.Duration
	maxDuration  time.Duration
	bucketsAll   []int64

	// inFlight 與 maxInFlight 用 atomics 而非 mu：Begin/End 走在每個請求的
	// 入口與出口，而「現在有幾個請求在處理中」是監控頁最常被讀的數字。
	inFlight    int64
	maxInFlight int64
}

// Options 是 New 的可調參數。零值會套用檔頭說明的預設。
type Options struct {
	// WindowMinutes 為記憶體內保留的分鐘桶數量，小於等於 0 時用預設 120。
	WindowMinutes int
	// MaxRoutes 為最多追蹤幾條不同路由，小於等於 0 時用預設 200。
	MaxRoutes int
	// MaxClients 為最多追蹤幾個不同來源位址，小於等於 0 時用預設 200。
	MaxClients int
	// ClientIdleMinutes 為一個來源在多久沒有再出現後被視為閒置而移除，
	// 小於等於 0 時用 WindowMinutes（因此預設就是記憶體視窗的長度）。
	ClientIdleMinutes int
	// Now 為取時間的函式，nil 時用 time.Now。測試可注入假時鐘。
	Now func() time.Time
}

// New 建立一個空的 Registry，並把建立時刻記為 uptime 的起點。
//
// 刻意不在這裡讀任何依賴或啟動 goroutine：統計容器的建立不該有副作用，
// 「何時開始把資料寫進資料庫」由呼叫端以 StartFlusher 明確決定
// （與 httpapi.NewServer 不啟動限流清理 goroutine 是同一個理由）。
func New(opts Options) *Registry {
	windowMinutes := opts.WindowMinutes
	if windowMinutes <= 0 {
		windowMinutes = defaultWindowMinutes
	}
	maxRoutes := opts.MaxRoutes
	if maxRoutes <= 0 {
		maxRoutes = defaultMaxRoutes
	}
	maxClients := opts.MaxClients
	if maxClients <= 0 {
		maxClients = defaultMaxClients
	}
	// 預設讓來源閒置期等於記憶體視窗：兩個視窗回答的是同一個問題
	//（「最近這段時間發生了什麼」），長度不一致只會讓管理員困惑於「為什麼
	// 曲線上還看得到那個位址，但它的計數已經歸零」。
	clientIdleMinutes := opts.ClientIdleMinutes
	if clientIdleMinutes <= 0 {
		clientIdleMinutes = windowMinutes
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Registry{
		startedAt:         now(),
		now:               now,
		windowMinutes:     windowMinutes,
		maxRoutes:         maxRoutes,
		maxClients:        maxClients,
		clientIdleMinutes: clientIdleMinutes,
		routes:            make(map[string]*routeStat),
		clients:           make(map[string]*clientStat),
		buckets:           make(map[int64]*minuteBucket),
		history:           make(map[int64]*minuteBucket),
		bucketsAll:        make([]int64, overflowIndex+1),
	}
}

// StartedAt 回傳建立時刻，供 uptime 與「目前時間」一起回報。
func (r *Registry) StartedAt() time.Time {
	return r.startedAt
}

// Begin 標記一個請求開始處理。呼叫端必須在回應寫出後呼叫 End。
func (r *Registry) Begin() {
	current := atomic.AddInt64(&r.inFlight, 1)
	// 只在刷新最大值時寫入，因此用 CAS 而不是先讀再寫：後者會讓同時結束的
	// 兩個請求中較小的值覆蓋掉較大的峰值。
	for {
		peak := atomic.LoadInt64(&r.maxInFlight)
		if current <= peak || atomic.CompareAndSwapInt64(&r.maxInFlight, peak, current) {
			return
		}
	}
}

// End 標記一個請求結束。
func (r *Registry) End() {
	atomic.AddInt64(&r.inFlight, -1)
}

// Observe 記錄一次已完成的請求。
//
// status 只用來分類（4xx 歸 clientErrors、5xx 歸 serverErrors），不保存完整
// 的狀態碼分佈：對這個規模的論壇，「哪條路由在回 4xx」遠比「哪個狀態碼出現
// 幾次」有診斷價值，而後者會讓每個路由多帶一個 map。
//
// 呼叫端應該在回應「確定寫出」之後呼叫（Middleware 是唯一的使用者）。
func (r *Registry) Observe(method, path string, status int, d time.Duration) {
	now := r.now()
	minute := minuteIndex(now)
	route := NormalizeRoute(path)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.total++
	r.durationSum += d
	if d > r.maxDuration {
		r.maxDuration = d
	}
	switch {
	case status >= 500:
		r.serverErrors++
	case status >= 400:
		r.clientErrors++
	}
	r.bucketsAll[bucketIndex(d)]++

	r.routeLocked(method, route).observe(d)

	bucket := r.bucketLocked(minute)
	bucket.total++
	bucket.durationSumMS += int64(d / time.Millisecond)
	switch {
	case status >= 500:
		bucket.serverErrors++
	case status >= 400:
		bucket.clientErrors++
	}
}

// routeLocked 取得（或建立）一條路由的統計槽。必須在持有 mu 時呼叫。
//
// 超過 maxRoutes 後不再建立新的 key，改為併入 otherRoute —— 那個槽刻意不受
// 上限限制，因此上限生效之後新路由仍然有地方可去，計數不會整段遺失。
func (r *Registry) routeLocked(method, route string) *routeStat {
	key := method + " " + route
	if stat, ok := r.routes[key]; ok {
		return stat
	}
	if len(r.routes) >= r.maxRoutes {
		otherKey := method + " " + otherRoute
		if stat, ok := r.routes[otherKey]; ok {
			return stat
		}
		stat := newRouteStat(method, otherRoute)
		r.routes[otherKey] = stat
		return stat
	}
	stat := newRouteStat(method, route)
	r.routes[key] = stat
	return stat
}

// bucketLocked 取得（或建立）某分鐘的桶。必須在持有 mu 時呼叫。
func (r *Registry) bucketLocked(minute int64) *minuteBucket {
	if bucket, ok := r.buckets[minute]; ok {
		return bucket
	}
	bucket := &minuteBucket{minute: minute}
	r.buckets[minute] = bucket
	return bucket
}

// NormalizeRoute 把具體路徑收斂成低基數的路由樣式。
//
// 三條規則，順序有意義：
//  1. /assets/ 與 /asset/ 之後的整段換成 "*"。這兩個掛載點送出的是帶內容
//     hash 的建置檔名，每次建置都會換一組，保留原值等於每次部署都多出
//     數十條新路由。
//  2. 以 "/" 開頭的純數字或含 "@" 的片段換成 ":id"（/api/forum/posts/17/comments
//     → /api/forum/posts/:id/comments；管理端以 email 當路徑參數，那會是
//     255 種以上的高基數）。
//  3. 最後一段帶 "." 的路徑把該段換成 "*"（catch-all 靜態檔）。只檢查最後一段
//     而不是整條路徑很重要：/api/admin/users/foo@bar.example/tags 的中間段
//     含有句點，但那是 email 的一部分而不是檔名，用整條路徑判斷會把一個
//     高基數的管理端路由誤判成靜態檔，然後收斂成 /api/* —— 那會讓所有管理
//     端 API 的統計混在一起，正好毀掉這張表最想做的事。
func NormalizeRoute(path string) string {
	if path == "" {
		return "/"
	}
	// 查詢字串不屬於路由的一部分。r.URL.Path 已經不含查詢字串，但這個函式
	// 也會被呼叫端拿去做其他比對，因此自己再切一次。
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		path = path[:idx]
	}
	if path == "/" {
		return "/"
	}

	for _, prefix := range []string{"/assets/", "/asset/"} {
		if strings.HasPrefix(path, prefix) {
			return prefix + "*"
		}
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	last := len(segments) - 1

	// 規則 3 先於規則 2：靜態檔的檔名本來就會被規則 2 換成 :id，先判斷才能
	// 分辨「這一段是檔名」而不是「這一段是識別值」。
	if strings.LastIndexByte(segments[last], '.') > 0 {
		segments[last] = "*"
		return "/" + strings.Join(segments, "/")
	}

	for i, segment := range segments {
		if isIdentifier(segment) {
			segments[i] = ":id"
		}
	}
	return "/" + strings.Join(segments, "/")
}

// isIdentifier 判斷一段路徑是否為「識別值」而非固定字面。
//
// 純數字是 id；含 @ 的是 email（管理端以 email 當路徑參數）。其餘
// （comments、like、report、tags）都是程式裡註冊的固定樣式，必須原樣保留 ——
// 把它們也換成 :id 會讓 /posts/1/comments 與 /posts/1/like 變成同一條路由，
// 等於把留言與按讚的統計混在一起。
func isIdentifier(segment string) bool {
	if segment == "" {
		return false
	}
	if strings.IndexByte(segment, '@') >= 0 {
		return true
	}
	for i := 0; i < len(segment); i++ {
		if segment[i] < '0' || segment[i] > '9' {
			return false
		}
	}
	return true
}

/* ==========================================================================
   快照
   ========================================================================== */

// RouteSnapshot 是單一路由的對外統計。
type RouteSnapshot struct {
	Method       string  `json:"method"`
	Route        string  `json:"route"`
	Total        int64   `json:"total"`
	ClientErrors int64   `json:"clientErrors"`
	ServerErrors int64   `json:"serverErrors"`
	AvgMS        float64 `json:"avgMs"`
	P50MS        float64 `json:"p50Ms"`
	P95MS        float64 `json:"p95Ms"`
	MaxMS        float64 `json:"maxMs"`
}

// LimitSnapshot 是單一限流器的對外統計。
type LimitSnapshot struct {
	Name          string  `json:"name"`
	Limit         int     `json:"limit"`
	WindowSeconds float64 `json:"windowSeconds"`
	Allowed       uint64  `json:"allowed"`
	Blocked       uint64  `json:"blocked"`
	TrackedKeys   int     `json:"trackedKeys"`
}

// TimelinePoint 是時間軸上的一分鐘。
//
// Source 說明這一格來自哪裡：
//   - "live"    本次啟動以來記憶體中的桶
//   - "history" 從資料庫讀回、代表重啟之前的紀錄
//   - ""        該分鐘沒有任何資料（服務未啟動或該分鐘沒流量）
//
// 前端用它在圖上標出「這段是重啟前的紀錄」，因為那段的數字並不反映
// 目前這個行程的狀態。
type TimelinePoint struct {
	Minute        string  `json:"minute"`
	Total         int64   `json:"total"`
	ClientErrors  int64   `json:"clientErrors"`
	ServerErrors  int64   `json:"serverErrors"`
	AvgDurationMS float64 `json:"avgDurationMs"`
	Source        string  `json:"source"`
}

// RuntimeSnapshot 是 Go 執行期本身的使用量。
type RuntimeSnapshot struct {
	Version     string  `json:"version"`
	Goroutines  int     `json:"goroutines"`
	NumCPU      int     `json:"numCpu"`
	GOMAXPROCS  int     `json:"gomaxprocs"`
	GCCycles    uint32  `json:"gcCycles"`
	AllocBytes  uint64  `json:"allocBytes"`
	SysBytes    uint64  `json:"sysBytes"`
	HeapAlloc   uint64  `json:"heapAllocBytes"`
	HeapInUse   uint64  `json:"heapInUseBytes"`
	HeapObjects uint64  `json:"heapObjects"`
	StackInUse  uint64  `json:"stackInUseBytes"`
	LastPauseMS float64 `json:"lastGcPauseMs"`
}

// RequestsSnapshot 是請求統計的總覽。
type RequestsSnapshot struct {
	Total         int64           `json:"total"`
	ClientErrors  int64           `json:"clientErrors"`
	ServerErrors  int64           `json:"serverErrors"`
	InFlight      int64           `json:"inFlight"`
	MaxInFlight   int64           `json:"maxInFlight"`
	AvgDurationMS float64         `json:"avgDurationMs"`
	P50MS         float64         `json:"p50Ms"`
	P95MS         float64         `json:"p95Ms"`
	P99MS         float64         `json:"p99Ms"`
	MaxMS         float64         `json:"maxMs"`
	Routes        []RouteSnapshot `json:"routes"`
}

// Snapshot 是整份監控資料的容器。
//
// Runtime 由 runtime.ReadMemStats 現場取得（它會觸發 stop-the-world，因此
// 不在 Observe 的路徑上）；Requests 與 Timeline 來自記憶體；Limits 由
// httpapi 透過 SetLimitStats 灌入。UptimeSeconds 與 StartedAt 分開提供：
// 前者直接顯示，後者讓前端可以自己算「這份資料有多舊」。
//
// 依賴健康狀態（MySQL / Redis / Elasticsearch）刻意不在這個型別裡 ——
// 它們需要真的送出 ping，屬於 I/O。呼叫端在取完本快照之後另外補上，這樣
// Snapshot 本身維持純記憶體、可測試。
type Snapshot struct {
	StartedAt      string           `json:"startedAt"`
	Now            string           `json:"now"`
	UptimeSeconds  float64          `json:"uptimeSeconds"`
	WindowMinutes  int              `json:"windowMinutes"`
	RetentionHours int              `json:"retentionHours"`
	MaxRoutes      int              `json:"maxRoutes"`
	Runtime        RuntimeSnapshot  `json:"runtime"`
	Requests       RequestsSnapshot `json:"requests"`
	Clients        []ClientSnapshot `json:"clients"`
	// MaxClients 與 ClientsDropped 一起讓「來源清單被截斷了」這件事可見：
	// 沒有它們，一個輪換位址的攻擊者會讓畫面看起來只是「今天沒什麼人來」。
	MaxClients   int              `json:"maxClients"`
	ClientsDropped int64          `json:"clientsDropped"`
	Timeline     []TimelinePoint  `json:"timeline"`
	Limits       []LimitSnapshot  `json:"rateLimits"`
	HistoryLoaded bool            `json:"historyLoaded"`
}

// Snapshot 組出一份可 JSON 序列化的統計快照。
func (r *Registry) Snapshot(retentionHours int) Snapshot {
	if retentionHours <= 0 {
		retentionHours = defaultRetentionHours
	}

	// ReadMemStats 會停止其他 goroutine，因此刻意在鎖外呼叫；它不碰本型別
	// 的任何欄位，不需要與 mu 同步。
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	// PauseNs 是最近 256 次 GC 的暫停時間環形緩衝，索引由 NumGC 取模。
	// 索引 0 對應「次數為 0 時的暫停」，在第一次 GC 之前沒有實際意義，
	// 因此顯示 0 而不是把未初始化的槽位當成真實數據。
	var lastPause float64
	if mem.NumGC > 0 {
		lastPause = float64(mem.PauseNs[mem.NumGC%256]) / float64(time.Millisecond)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	// 剪枝放在這裡（持有 mu）而不是只留給 flusher：沒有資料庫可寫的部署不會
	// 啟動 StartFlusher，那時唯一的清理機會就是有人真的在看監控頁。閒置來源
	// 對記憶體的影響很小，但「讓 map 的內容與 now 之後仍然一致」這件事值得
	// 在唯一保證有人在意它的時刻做掉。
	r.pruneClientsLocked(now)
	return Snapshot{
		StartedAt:      r.startedAt.UTC().Format(time.RFC3339),
		Now:            now.UTC().Format(time.RFC3339),
		UptimeSeconds:  now.Sub(r.startedAt).Seconds(),
		WindowMinutes:  r.windowMinutes,
		RetentionHours: retentionHours,
		MaxRoutes:      r.maxRoutes,
		Runtime: RuntimeSnapshot{
			Version:     runtime.Version(),
			Goroutines:  runtime.NumGoroutine(),
			NumCPU:      runtime.NumCPU(),
			GOMAXPROCS:  runtime.GOMAXPROCS(0),
			GCCycles:    mem.NumGC,
			AllocBytes:  mem.Alloc,
			SysBytes:    mem.Sys,
			HeapAlloc:   mem.HeapAlloc,
			HeapInUse:   mem.HeapInuse,
			HeapObjects: mem.HeapObjects,
			StackInUse:  mem.StackInuse,
			LastPauseMS: lastPause,
		},
		Requests: RequestsSnapshot{
			Total:         r.total,
			ClientErrors:  r.clientErrors,
			ServerErrors:  r.serverErrors,
			InFlight:      atomic.LoadInt64(&r.inFlight),
			MaxInFlight:   atomic.LoadInt64(&r.maxInFlight),
			AvgDurationMS: averageMS(r.total, r.durationSum),
			P50MS:         quantileMS(r.bucketsAll, r.total, 0.50),
			P95MS:         quantileMS(r.bucketsAll, r.total, 0.95),
			P99MS:         quantileMS(r.bucketsAll, r.total, 0.99),
			MaxMS:         durationMS(r.maxDuration),
			Routes:        r.routesLocked(),
		},
		Clients:        r.clientsLocked(),
		MaxClients:     r.maxClients,
		ClientsDropped: r.clientsDropped,
		Timeline:       r.timelineLocked(now),
		Limits:         r.limitsLocked(),
		HistoryLoaded:  len(r.history) > 0,
	}
}

// routesLocked 把路由統計轉成排序後的對外清單。必須持有 mu。
//
// 依請求數排序，讓「最忙的路由」永遠在表格最上方。數量相同時以路由字串
// 再以 method 排序，讓同一份資料每次刷新都得到相同的順序 —— 否則表格會在
// 每次自動刷新時整列跳動，那是自動刷新最惱人的副作用。
func (r *Registry) routesLocked() []RouteSnapshot {
	routes := make([]RouteSnapshot, 0, len(r.routes))
	for _, stat := range r.routes {
		if stat.total == 0 {
			continue
		}
		routes = append(routes, RouteSnapshot{
			Method:       stat.method,
			Route:        stat.route,
			Total:        stat.total,
			ClientErrors: stat.clientErrors,
			ServerErrors: stat.serverErrors,
			AvgMS:        averageMS(stat.total, stat.durationSum),
			P50MS:        quantileMS(stat.buckets, stat.total, 0.50),
			P95MS:        quantileMS(stat.buckets, stat.total, 0.95),
			MaxMS:        durationMS(stat.maxDuration),
		})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Total != routes[j].Total {
			return routes[i].Total > routes[j].Total
		}
		if routes[i].Route != routes[j].Route {
			return routes[i].Route < routes[j].Route
		}
		return routes[i].Method < routes[j].Method
	})
	return routes
}

// limitsLocked 把限流器計數轉成對外清單。必須持有 mu。
func (r *Registry) limitsLocked() []LimitSnapshot {
	limits := make([]LimitSnapshot, 0, len(r.limits))
	for _, stat := range r.limits {
		limits = append(limits, LimitSnapshot{
			Name:          stat.Name,
			Limit:         stat.Limit,
			WindowSeconds: stat.Window.Seconds(),
			Allowed:       stat.Allowed,
			Blocked:       stat.Blocked,
			TrackedKeys:   stat.Tracked,
		})
	}
	return limits
}

// timelineLocked 合併歷史與即時桶，輸出最近 TimelineMinutes 個連續分鐘。必須持有 mu。
//
// 連續性刻意用「從當前分鐘往回走」來決定，而不是只列出有資料的分鐘：圖上
// 出現空格才是「那三分鐘真的沒有請求」。若省略空格，讀者會把線段的空白
// 誤解成「零」而不是「沒有資料」—— 這兩者在監控上是完全不同的意思。
func (r *Registry) timelineLocked(now time.Time) []TimelinePoint {
	currentMinute := minuteIndex(now)
	length := r.TimelineMinutes()

	points := make([]TimelinePoint, 0, length)
	for i := length - 1; i >= 0; i-- {
		minute := currentMinute - int64(i)
		point := TimelinePoint{
			Minute: minuteTime(minute).Format(time.RFC3339),
		}
		// 即時的優先於歷史的：同一個分鐘若兩者都有，代表行程在那一分鐘內
		// 重啟過，此時顯示本次啟動觀測到的那一份，並由 source 標示出來。
		if bucket, ok := r.buckets[minute]; ok {
			point.Source = "live"
			fillTimelinePoint(&point, bucket)
		} else if bucket, ok := r.history[minute]; ok {
			point.Source = "history"
			fillTimelinePoint(&point, bucket)
		}
		points = append(points, point)
	}
	return points
}

// fillTimelinePoint 把桶的數值填進時間軸的一格。
func fillTimelinePoint(point *TimelinePoint, bucket *minuteBucket) {
	point.Total = bucket.total
	point.ClientErrors = bucket.clientErrors
	point.ServerErrors = bucket.serverErrors
	point.AvgDurationMS = averageMS(bucket.total, time.Duration(bucket.durationSumMS)*time.Millisecond)
}

// averageMS 以毫秒為單位回傳平均耗時。count 為 0 時回 0 而非 NaN。
func averageMS(count int64, sum time.Duration) float64 {
	if count <= 0 {
		return 0
	}
	return float64(sum) / float64(count) / float64(time.Millisecond)
}

// durationMS 把耗時轉成毫秒。
func durationMS(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// quantileMS 由累積直方圖回傳分位數的「所在格上界」。
//
// 回傳上界而不是格內插值是有意的保守：插值會給出一個看起來精確、實際是從
// 匯總資料編造出來的數字。介面上的說明文字因此必須標示為分位數上界。
// 資料量為 0 時回 0；落在溢出格時同樣回最後一個邊界（維持上界語意）。
func quantileMS(buckets []int64, total int64, q float64) float64 {
	if total <= 0 {
		return 0
	}
	target := int64(math.Ceil(float64(total) * q))
	if target < 1 {
		target = 1
	}
	var cumulative int64
	for i, count := range buckets {
		cumulative += count
		if cumulative >= target {
			bound := overflowIndex
			if i < overflowIndex {
				bound = i
			}
			return durationMS(latencyBounds[bound])
		}
	}
	return durationMS(latencyBounds[overflowIndex-1])
}

/* ==========================================================================
   限流器統計
   ========================================================================== */

// SetLimitStats 灌入限流器的計數。呼叫端（httpapi）在每次取快照前呼叫。
//
// 這是「由呼叫端灌入」而不是讓 metrics 持有 *RateLimiter 的原因見 LimitStat
// 的說明。用一份純資料中轉，兩邊都不必知道對方的存在。
func (r *Registry) SetLimitStats(stats []LimitStat) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limits = append(make([]LimitStat, 0, len(stats)), stats...)
}

/* ==========================================================================
   持久化
   ========================================================================== */

// FlusherOptions 是 StartFlusher 的可調參數。
type FlusherOptions struct {
	// DB 為寫入目標。nil 時 StartFlusher 回傳 ErrNoDatabase 且不啟動
	// goroutine —— 測試可以只驗證記憶體統計而不必準備資料庫。
	DB *sql.DB
	// Interval 為寫入週期，小於等於 0 時用 20 秒。
	Interval time.Duration
	// RetentionHours 為資料庫內保留的小時數，小於等於 0 時用 24。
	RetentionHours int
}

// StartFlusher 以背景 goroutine 定期把已結束的分鐘桶寫進 MySQL，直到 ctx 被取消。
//
// 只寫「已結束的分鐘」（minute < 當前分鐘），且每個桶只寫一次；寫入使用
// 累加語意以支援多執行個體。完整理由見檔頭。
//
// 過期資料的清除與寫入同一個 tick 進行，用 >= now-retention 的條件，因此是
// 最多每 tick 一次的主鍵範圍掃描。這個成本相對於每 tick 數十列的寫入可以
// 忽略，但它確實會隨保留期長度線性增加 —— 保留期是設定值，不該無上限拉長。
func (r *Registry) StartFlusher(ctx context.Context, opts FlusherOptions) error {
	if opts.DB == nil {
		return ErrNoDatabase
	}
	if opts.Interval <= 0 {
		opts.Interval = 20 * time.Second
	}
	if opts.RetentionHours <= 0 {
		opts.RetentionHours = defaultRetentionHours
	}
	// cutoff 在啟動時算一次即可：保留期的基準是「服務啟動的時間」，每次 tick
	// 重算只會讓刪除條件每次 tick 都往前挪動一點點，對兩小時的保留期沒有
	// 任何意義，卻讓「這個時鐘到底是不是單調的」變成一個隱含前提。
	cutoff := r.now().Add(-time.Duration(opts.RetentionHours) * time.Hour)

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				// 離開前再寫一次，把最後一次 tick 之後結束的分鐘送出。
				// 本程式目前沒有 graceful shutdown（見 main.go 的說明），
				// 因此這條路徑實際上不會被觸發；但它讓未來接上 signal.Notify
				// 時不必回頭修改這個檔案。
				r.FlushPending(context.WithoutCancel(ctx), opts.DB)
				return
			case <-ticker.C:
				r.FlushPending(ctx, opts.DB)
				pruneMetrics(ctx, opts.DB, cutoff)
			}
		}
	}()
	return nil
}

// FlushPending 把所有「已結束且尚未寫出」的分鐘桶寫進資料庫。
//
// 回傳實際寫出的分鐘數，供測試斷言。寫入失敗時不標記 flushed，因此下一個
// tick 會重試 —— 資料庫短暫故障不該讓監控資料靜默地少掉一段。
//
// 呼叫端不應為了「補寫」而手動呼叫它：回傳的是本行程觀測到的分鐘，重複
// 呼叫會讓同一分鐘的數字被累加兩次（這是刻意保留的語意，見檔頭）。
func (r *Registry) FlushPending(ctx context.Context, db *sql.DB) int {
	if db == nil {
		return 0
	}
	// 當前分鐘仍然會有新請求，因此不能寫；它的最終值要等下一個分鐘。
	currentMinute := minuteIndex(r.now())

	r.mu.Lock()
	pending := make([]minuteBucket, 0, len(r.buckets))
	for minute, bucket := range r.buckets {
		if minute >= currentMinute || bucket.flushed {
			continue
		}
		candidate := *bucket
		candidate.flushed = true
		pending = append(pending, candidate)
	}
	// map 迭代順序隨機，因此依分鐘排序讓寫入順序固定，除錯時比對 log 會
	// predictable（同一組分鐘每次都是同樣的先後）。
	sort.Slice(pending, func(i, j int) bool { return pending[i].minute < pending[j].minute })
	// 寫入前先標記 flushed：已結束的分鐘不會再變動，因此這個「先佔用」的
	// 動作不需要任何重試邏輯。寫入失敗時把旗標還原（見下方），讓下一個 tick
	// 重試同一分鐘 —— 而不是重寫，那會讓累加語意把同一分鐘算成兩倍。
	for i := range pending {
		r.buckets[pending[i].minute].flushed = true
	}
	r.pruneLocked(r.now())
	r.mu.Unlock()

	written := 0
	for _, bucket := range pending {
		minute := minuteTime(bucket.minute)
		_, err := db.ExecContext(ctx, `
			INSERT INTO forum_request_metrics
				(bucket_minute, total, client_errors, server_errors, duration_sum_ms)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				total = total + VALUES(total),
				client_errors = client_errors + VALUES(client_errors),
				server_errors = server_errors + VALUES(server_errors),
				duration_sum_ms = duration_sum_ms + VALUES(duration_sum_ms)
		`, minute, bucket.total, bucket.clientErrors, bucket.serverErrors, bucket.durationSumMS)
		if err != nil {
			// 還原旗標讓下一個 tick 重試。中止整批的話，單一分鐘的暫時性
			// 失敗（例如剛好跨過分鐘邊界時的鎖等待）會擋住全部資料。
			r.mu.Lock()
			if current, ok := r.buckets[bucket.minute]; ok {
				current.flushed = false
			}
			r.mu.Unlock()
			continue
		}
		written++
	}
	return written
}

// pruneLocked 丟棄超出記憶體視窗的分鐘桶。必須持有 mu。
//
// 歷史桶也在清理範圍內：LoadHistory 讀的是「保留期內」而非「視窗內」，
// 一個寫滿 24 小時歷史的行程如果只看 buckets 不看 history，記憶體會是
// 24 × 60 個桶。兩者的清理條件相同，因為時間軸的長度本來就由視窗決定。
func (r *Registry) pruneLocked(now time.Time) {
	cutoff := minuteIndex(now.Add(-time.Duration(r.windowMinutes) * time.Minute))
	for minute := range r.buckets {
		if minute < cutoff {
			delete(r.buckets, minute)
		}
	}
	for minute := range r.history {
		if minute < cutoff {
			delete(r.history, minute)
		}
	}
	// 每 IP 統計的剪枝條件不同（用「最後活動」而不是分鐘索引），因此獨立
	// 呼叫而不是併進上面的迴圈。放在這裡是為了讓「不論有沒有資料庫，來源都會
	// 被清理」這個不變條件只有一個實作點。
	r.pruneClientsLocked(now)
}

// TimelineMinutes 回傳時間軸實際會畫幾格。
//
// 匯出的理由：時間軸長度不是呼叫端能自己算出來的（它同時取決於
// windowMinutes 與 maxTimelinePoints 兩個值），而 main 需要它來決定
// LoadHistory 要從資料庫讀回多久 —— 讀得比它多等於白讀、白佔記憶體：
// pruneLocked 會在 20 秒內把超出視窗的桶丟掉，那些列永遠不會出現在任何
// 一張圖上。
//
// 刻意**不**讓 MONITOR_RETENTION_HOURS 影響這個值：那個設定是資料庫保留政策，
// 與「畫幾格」是兩個問題（見 defaultRetentionHours 的說明）。讓它兩邊都管
// 會造成「設定 72 小時以為圖會變長」這種預期落空，而且回應會膨脹到前端
// 畫不下的程度。
func (r *Registry) TimelineMinutes() int {
	if r == nil {
		return 0
	}
	length := r.windowMinutes
	if length <= 0 {
		length = defaultWindowMinutes
	}
	if length > maxTimelinePoints {
		length = maxTimelinePoints
	}
	return length
}

// LoadHistory 從資料庫讀回 since 之後的分鐘彙總，讓時間軸在重啟後仍有東西可畫。
//
// 讀到的桶放進 history，永遠不會被 FlushPending 寫回 —— 它們已經在資料庫
// 裡，再寫一次會讓累加語意變成重複計算。
//
// 表不存在（遷移尚未執行，或權限不足）時回傳明確錯誤但不 panic：監控頁是
// 診斷工具，它自己壞掉不該影響論壇服務，因此呼叫端只把這個錯誤記進日誌。
func (r *Registry) LoadHistory(ctx context.Context, db *sql.DB, since time.Time) (int, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	rows, err := db.QueryContext(ctx, `
		SELECT bucket_minute, total, client_errors, server_errors, duration_sum_ms
		FROM forum_request_metrics
		WHERE bucket_minute >= ?
		ORDER BY bucket_minute ASC
	`, since.UTC())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	loaded := 0
	for rows.Next() {
		var minute time.Time
		var total, clientErrors, serverErrors, durationSum int64
		if err := rows.Scan(&minute, &total, &clientErrors, &serverErrors, &durationSum); err != nil {
			return loaded, err
		}
		key := minuteIndex(minute.UTC())
		r.mu.Lock()
		r.history[key] = &minuteBucket{
			minute:        key,
			total:         total,
			clientErrors:  clientErrors,
			serverErrors:  serverErrors,
			durationSumMS: durationSum,
		}
		r.mu.Unlock()
		loaded++
	}
	return loaded, rows.Err()
}

// pruneMetrics 刪除 cutoff 之前的分鐘彙總行。
//
// 沒有回傳錯誤：這是純粹的回收工作，失敗只會讓資料庫慢慢長大（下次 tick
// 還會再試），不值得為它讓整個 flusher 的錯誤處理變複雜。
func pruneMetrics(ctx context.Context, db *sql.DB, cutoff time.Time) {
	_, _ = db.ExecContext(ctx, `DELETE FROM forum_request_metrics WHERE bucket_minute < ?`, cutoff.UTC())
}
