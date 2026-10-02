/*
每 IP 統計（metrics/clients.go）。

這份資料回答的問題是「誰在打這台站」
    既有統計的維度是「哪條路由、幾次、多慢、錯幾次」，把來源混在一起。攻擊者
    從一個位址打進來時，儀表板上看到的是「POST /api/forum/posts 上升」——那是
    症狀，不是原因；管理員仍然不知道該封哪一個位址。

    限流器追蹤了用戶端數量（TrackedKeys），但那是給「記憶體有沒有長大」看的，
    不是給人看的名單：它只有數量，沒有誰、做了幾次、錯了幾次。

為什麼放在 metrics 而不是 ipban
    ipban 管的是「已確認濫用者的處置」，而它的名單只存 IP 與到期秒數（見
    ipban.Ban 的「刻意不存」說明）。這裡要做的是相反的事：**觀察**所有來源，
    包括完全正常的那些。把觀察寫進 ipban 會讓「沒被封的人」也進入一個以
    封鎖為前提的資料結構，而兩者的生命週期也不同（名單有到期，統計沒有）。

刻意不做的事

    1. **不寫進資料庫。** 分鐘彙總會落盤（StartFlusher），每 IP 統計不會。
       理由是資料性質不同：請求量與延遲是站的營運指標，而「某個位址在什麼時候
       打了本站」是一筆個人資料。落盤等於把它變成一份永久的訪客紀錄，而這個
       頁面唯一需要的是「現在正在怎樣」。重啟後歸零是刻意的：那一瞬間的價值
       遠低於「不留下長期個人資料」的價值。
    2. **不記錄 email、User-Agent、路徑參數。** 每 IP 統計的用途是找出異常
       來源，而做到那件事只需要「位址 + 計數」。多存一個維度就多一個可以被
       用來還原個人行為的欄位。
    3. **不做自動封鎖。** 理由寫在 ipban 檔頭：NAT 後面共用出口的正常使用者
       會被誤傷，而誤封正常使用者的後果比多讓一個腳本多打幾次嚴重。這裡
       提供的是「看到誰，然後由管理員決定」。

基數（cardinality）控制：這是這份資料唯一真正的風險
    IP 是攻擊者可控的維度。clientIP 優先採信 X-Forwarded-For（見 ratelimit.go
    的說明），因此一個腳本只要每次請求換一個假位址，就能讓 map 無上限成長 ——
    與 ratelimit.hits 同一類問題，只是這裡沒有背景清理器去補。

    兩道限制：

      1. **maxClients 上限（預設 200）**，超出時不是丟棄新來源，而是**驅逐
         最久沒出現的那一個**。丟棄是最容易寫的版本，但它的失敗模式很糟：一個
         輪換位址的攻擊者會讓所有真實使用者都擠在上限之外，而被保留下來的
         剛好全是攻擊者的假位址。驅逐最舊的等於讓「最近仍在活動的來源」一定
         有地方可去 —— 而那正是監控頁要回答的問題。驅逐的成本是一次 O(n) 掃描，
         只在新來源、且已在上限時發生。
      2. **閒置剪枝**：超過 clientIdleMinutes（預設等於記憶體視窗，120 分鐘）
         沒有再出現的來源會被移除，讓「昨天來過一次」的訪客不會長期佔位。

    驅逐掉的那個來源，它的計數會**整筆消失**，而不是併進一個彙總槽。這與路由
    的 `__other__` 不同，理由是 IP 欄位在這個頁面上是可操作的（管理員要能點
    「封鎖」），一個彙總槽裡的假位址既不能操作、也沒有意義。取而代之的是
    clientsDropped 計數器：達到上限時它會增加，讓「這一頁可能漏了東西」是可
    觀察的事實，而不是一個看不見的截斷。

併發
    與 Registry 的其他欄位共用同一把 mu。請求路徑上這是第二次加鎖（第一次是
    Observe），臨界區是幾個整數加法與一次 map 查找，因此不值得為了並行度把它
    拆成獨立結構 —— 那會讓「一次請求的路由統計與來源統計是否一致」變成一件
    需要推理的事。
*/

package metrics

import (
	"sort"
	"time"
)

// defaultMaxClients 是最多追蹤幾個不同來源位址的上限。
//
// 200 的數量級來自「這個規模的論壇在兩小時內會出現多少個真實來源」：幾百個
// 活躍讀者加上偶爾路過的爬蟲就差不多用掉了，剩下的額度留給攻擊者 —— 也就是
// 說，超過 200 之後這個頁面主要是在回答「有沒有人在輪換位址」，而那正是它
// 該回答的問題。
const defaultMaxClients = 200

// 來源的可信度標記。
//
// 存在的理由（也是 H4 那個審查意見的直接回應）：clientIP 優先採信
// X-Forwarded-For，而那是使用者可以自己設定的標頭。在沒有設定可信代理白名單
// 的部署下，這三種來源的**可信度完全不同**，但回應裡原本只給一個字串 —— 管理
// 員看到一個位址時無從判斷它是「連線對端」還是「自己剛才送的那個標頭」。
// 把它標出來之後，「這個數字可以拿來封人嗎」變成一個看得出來的事實。
//
// 注意這**不是**在修 H4：那個問題的正確修法是依 TRUSTED_PROXY_CIDRS 決定要不要
// 採信 XFF，而那需要先確認生產環境的代理拓撲。這個標記的誠實說法是「你知道
// 這個值是怎麼來的」，不是「這個值可信」。
const (
	// ClientSourceXFF 代表位址取自 X-Forwarded-For 最左一項。使用者可控。
	ClientSourceXFF = "xff"
	// ClientSourceRealIP 代表位址取自 X-Real-IP。同樣使用者可控。
	ClientSourceRealIP = "real-ip"
	// ClientSourcePeer 代表位址取自 RemoteAddr（去掉埠號）。這是 TCP 連線
	// 的真實對端，只有在服務可被直連時才等於真人；在反向代理後面它是代理。
	ClientSourcePeer = "peer"
)

// clientStat 是單一來源位址的統計。全部欄位由 Registry.mu 保護。
type clientStat struct {
	source string
	// total 是這個位址的總請求數，包含被限流與被封鎖擋下的那些：被擋下的
	// 請求同樣是「有人在打這個位址」的證據，把它們排除會讓最該被看到的
	// 位址看起來最冷清。
	total int64
	// clientErrors / serverErrors 與 RequestsSnapshot 的分類一致（4xx／5xx），
	// 因此可以並排比較「整站」與「這個位址」的錯誤率。
	clientErrors int64
	serverErrors int64
	// rateLimited 是回 429 的次數。它是「這個位址已經超出額度」的直接證據，
	// 而 4xx 裡還有 401/403/404，把它混在一起會讓「被限流」這個可行動的
	// 訊號被稀釋。
	rateLimited int64
	// banned 是「因在封鎖名單上而被擋下」的次數。它由 withBlocklistHandler
	// 透過 request context 標記（見 httpapi/blocklist.go），不是從狀態碼
	// 反推的：403 同時來自 requireAdminForum 與 CSRF 檢查，而把那些算成
	// 「被封鎖」會讓管理員以為有人在硬闖一個他根本沒封過的位址。
	banned     int64
	firstSeen  time.Time
	lastSeen   time.Time
	lastRoute  string
}

// ClientSnapshot 是單一來源位址的對外統計。
type ClientSnapshot struct {
	IP            string `json:"ip"`
	Source        string `json:"source"`
	Total         int64  `json:"total"`
	ClientErrors  int64  `json:"clientErrors"`
	ServerErrors  int64  `json:"serverErrors"`
	RateLimited   int64  `json:"rateLimited"`
	Banned        int64  `json:"banned"`
	FirstSeen     string `json:"firstSeen"`
	LastSeen      string `json:"lastSeen"`
	LastRoute     string `json:"lastRoute"`
}

// clientsLocked 把來源統計轉成排序後的對外清單。必須持有 mu。
//
// 依請求數排序，讓「打最多」的來源永遠在表格最上方。相同時依最後活動時間、
// 再依位址字串排序，讓每一輪自動刷新都得到相同的順序 —— 否則表格會在每次刷新
// 時整列跳動，那是自動刷新最惱人的副作用。
//
// 不設回傳筆數上限：來源數量本身已被 maxClients 綁住（見 defaultMaxClients）。
func (r *Registry) clientsLocked() []ClientSnapshot {
	// 中間型別帶著未格式化的 time.Time：排序要比的是時間點本身，而
	// ClientSnapshot 的 LastSeen 已經是 RFC3339 字串。拿字串比大小在這個
	// 專案會「碰巧正確」（RFC3339 是固定寬度且字典序即時序），但那是個會在
	// 有人改成別的時間格式時靜默壞掉的巧合 —— 排序鍵與顯示值不該是同一個值。
	type sortable struct {
		view     ClientSnapshot
		lastSeen time.Time
	}
	rows := make([]sortable, 0, len(r.clients))
	for ip, stat := range r.clients {
		rows = append(rows, sortable{
			lastSeen: stat.lastSeen,
			view: ClientSnapshot{
				IP:           ip,
				Source:       stat.source,
				Total:        stat.total,
				ClientErrors: stat.clientErrors,
				ServerErrors: stat.serverErrors,
				RateLimited:  stat.rateLimited,
				Banned:       stat.banned,
				FirstSeen:    stat.firstSeen.UTC().Format(time.RFC3339),
				LastSeen:     stat.lastSeen.UTC().Format(time.RFC3339),
				LastRoute:    stat.lastRoute,
			},
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].view.Total != rows[j].view.Total {
			return rows[i].view.Total > rows[j].view.Total
		}
		if !rows[i].lastSeen.Equal(rows[j].lastSeen) {
			return rows[i].lastSeen.After(rows[j].lastSeen)
		}
		return rows[i].view.IP < rows[j].view.IP
	})
	clients := make([]ClientSnapshot, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, row.view)
	}
	return clients
}

// ObserveClient 記錄一次請求的來源位址。
//
// 刻意與 Observe 分開而不是合併成一個簽章：Observe 的參數是 method / path /
// status / duration，這一組對「來源統計」全部多餘（duration 不進這張表、method
// 不進這張表）。把它們塞進一個結構會讓兩種截然不同的呼叫都變成要填四個用不到的
// 欄位，而那正是呼叫端會開始亂填的起點。route 在這裡仍然需要（作為「最近打到
// 哪一條路由」），因此由本函式自行正規化 —— 與 Observe 各算一次，兩次的成本
// 都只是幾個字串操作，換來的是呼叫端不會有機會傳進未正規化的路徑。
//
// banned 必須由呼叫端明確告知（見 blocklist.go 的說明），不能從 status 推導。
//
// ip 為空時整個呼叫被忽略：那是「拿不到來源」的情況，而把空字串當成一個 key
// 會讓所有這類請求共用同一份統計 —— 一個看起來像「某個神秘來源」的統計槽。
func (r *Registry) ObserveClient(ip, source, path string, status int, banned bool) {
	if ip == "" {
		return
	}
	now := r.now()
	route := NormalizeRoute(path)

	r.mu.Lock()
	defer r.mu.Unlock()

	stat, ok := r.clients[ip]
	if !ok {
		if len(r.clients) >= r.maxClients {
			r.evictOldestClientLocked()
			r.clientsDropped++
		}
		stat = &clientStat{source: source, firstSeen: now}
		r.clients[ip] = stat
	}
	stat.total++
	stat.lastSeen = now
	stat.lastRoute = route
	if stat.source == "" {
		// 第一次建立時 source 應該已經是某個常數；這裡仍然補一個預設值，
		// 因為「空字串」在 JSON 裡會讓前端的標記判斷落到未知分支。
		stat.source = ClientSourcePeer
	}
	switch {
	case status >= 500:
		stat.serverErrors++
	case status >= 400:
		stat.clientErrors++
	}
	if status == 429 {
		stat.rateLimited++
	}
	if banned {
		stat.banned++
	}
}

// evictOldestClientLocked 移除最久沒出現的來源。必須持有 mu。
//
// 掃描整個 map 取 lastSeen 最小者。O(n) 在這裡可以接受：它只在「新來源且已經
// 達到上限」時發生，而 n 被 maxClients 綁在 200 以內 —— 200 次比較遠小於這個
// 請求本來就要付的 MySQL 或 Redis 成本。
//
// 保守起見仍然檢查 len > 0：呼叫端雖已確認在上限時才會呼叫這裡，但一個會
// 對空 map 執行 delete 的函式不該把安全性建立在呼叫端的前提上。
func (r *Registry) evictOldestClientLocked() {
	if len(r.clients) == 0 {
		return
	}
	oldestIP := ""
	var oldest time.Time
	for ip, stat := range r.clients {
		if oldestIP == "" || stat.lastSeen.Before(oldest) {
			oldestIP, oldest = ip, stat.lastSeen
		}
	}
	if oldestIP != "" {
		delete(r.clients, oldestIP)
	}
}

// pruneClientsLocked 移除閒置過久的來源。必須持有 mu。
//
// 剪枝條件用「最後一次出現」而不是「建立時間」：一個三小時前來過、剛剛又來
// 一次的來源仍然值得顯示，而它的建立時間在三小時前。
//
// 呼叫者是 pruneLocked 與 Snapshot。前者由 flusher 每 20 秒觸發一次；後者讓
// 「沒有資料庫可寫」的部署（StartFlusher 回 ErrNoDatabase）仍然會在有人看監控
// 頁時清理，而不是讓閒置來源一路留到行程結束。
func (r *Registry) pruneClientsLocked(now time.Time) {
	cutoff := now.Add(-time.Duration(r.clientIdleMinutes) * time.Minute)
	for ip, stat := range r.clients {
		if stat.lastSeen.Before(cutoff) {
			delete(r.clients, ip)
		}
	}
}