package httpapi

/*
監控中介層與監控 API（httpapi/monitoring.go）。

本檔案是 metrics 套件與 httpapi 之間的接線點：把「一個 HTTP 請求」轉成
metrics.Observe 的一組參數，除此之外不碰統計的形狀。正規化規則（哪些路徑
片段算 id）、分桶與持久化策略都在 metrics 套件裡，而它不需要知道 HTTP 的
存在（見 metrics 套件檔頭的「對外介面」）。

掛載位置：LoggingMiddleware 的內側、mux 的外側

  SecurityHeaders → Refresh → LoggingMiddleware → metricsMiddleware → mux

為什麼在 LoggingMiddleware 的內側：

  1. LoggingMiddleware 已經把 http.ResponseWriter 包成會記錄狀態碼的
     StatusWriter（logger 套件），因此 metricsMiddleware 沿用同一個實作，
     不必再寫一份只為了取得狀態碼的包裝。兩層包裝是刻意的：外層記存取
     日誌、內層記統計，兩者讀的是同一份狀態碼，不會有分歧。
  2. duration 因此涵蓋的是「路由比對 + handler + 回應寫出」，不包含 session
     續期（Refresh 在更外層）與日誌格式化。監控頁顯示的耗時應該是使用者
     實際等到的時間，而不是把 Redis 往返算進去的數字。

為什麼不把統計直接塞進 LoggingMiddleware：

  那會讓 logger（輸出診斷訊息的套件）開始持有「要量哪些維度」的政策。正規化
  規則與持久化策略都不是日誌的職責，混在一起之後要換掉其中一邊就得拆開整個
  logger。

本檔案也提供 handleAdminMonitor。那支端點的依賴健康狀態刻意不放進
metrics.Snapshot：那些需要真的送出 ping（屬於 I/O），而 metrics.Snapshot 維持
純記憶體、可測試。分工的界線是「誰需要 I/O」。
*/

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"forum/forum/logger"
	"forum/forum/metrics"

	"github.com/redis/go-redis/v9"
)

// metricsMiddleware 產生記錄請求統計的中介層。
//
// nil-safe：s.metrics 為 nil（測試以 struct literal 構造 Server 而不填這個
// 欄位）時直接回傳 next，不做任何包裝 —— 否則那些測試會在建立路由時就 panic
// （與 rateLimit 的 nil-safe 是同一個理由）。
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	if s.metrics == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// marker 在 context 裡被 blocklist 中介層填入（見 markBannedRequest），
		// 因此 r 要先換成帶 marker 的那一份再往下傳。
		r = withBannedMarker(r)
		// Begin 必須在 handler 之前呼叫：End 會把 inFlight 減一，而這兩個
		// 數字要涵蓋 handler 正在執行的期間。defer 保證 panic 被 net/http
		// 轉成 500 時，計數仍然收得回來。
		s.metrics.Begin()
		rw := logger.NewStatusWriter(w)
		defer func() {
			s.metrics.End()
			d := time.Since(start)
			status := rw.StatusCode()
			s.metrics.Observe(r.Method, r.URL.Path, status, d)
			ip, source := s.clientIPDetail(r)
			marker, _ := r.Context().Value(bannedRequestKey{}).(*bannedMarker)
			s.metrics.ObserveClient(ip, source, r.URL.Path, status, marker != nil && marker.blocked)
		}()
		next.ServeHTTP(rw, r)
	})
}

/* ==========================================================================
   限流器計數
   ========================================================================== */

// limitStats 收集三個限流器目前的計數，餵給 metrics。
//
// 沒有被建立的限流器（測試以 struct literal 構造 Server）會被跳過而不是回
// 傳零值 —— 儀表板上出現「額度 0」的列會被誤讀成「整站被封鎖」。
func (s *Server) limitStats() []metrics.LimitStat {
	limiters := []struct {
		name    string
		limiter *RateLimiter
	}{
		{"content", s.writeRateLimiter},
		{"upload", s.uploadRateLimiter},
		{"auth", s.authRateLimiter},
	}

	stats := make([]metrics.LimitStat, 0, len(limiters))
	for _, entry := range limiters {
		if entry.limiter == nil {
			continue
		}
		snapshot := entry.limiter.Stats()
		stats = append(stats, metrics.LimitStat{
			Name:    entry.name,
			Limit:   entry.limiter.Limit(),
			Window:  entry.limiter.Window(),
			Allowed: snapshot.Allowed,
			Blocked: snapshot.Blocked,
			Tracked: entry.limiter.TrackedKeys(),
		})
	}
	return stats
}

/* ==========================================================================
   依賴健康狀態
   ========================================================================== */

// dependencyProbeTimeout 是監控端點探測單一依賴的逾時。
//
// 刻意比 /healthz 的 3 秒短：這個端點還要回傳記憶體統計、連線池水位與時間
// 軸，瀏覽器的自動刷新不可能等太久。三個探測並行執行，因此端到端的上限是
// 這個值而不是它的三倍。
const dependencyProbeTimeout = 2 * time.Second

// dependencyStatus 是單一依賴的探測結果。
//
// 刻意與 /healthz 的 map[string]string 分開：健康檢查只要一句話（「掛了」或
// 「正常」），監控頁要顯示的則是延遲數字、連線池水位與錯誤訊息。合成同一個
// 型別會讓兩邊都帶著對方不需要的欄位。
type dependencyStatus struct {
	// State 取 "ok" / "down" / "disabled"。disabled 不是健康的其中一種，而是
	// 根本沒啟用 —— 維運上要能區分「查得到但其實是降級模式」與「壓根沒設定」。
	State string `json:"state"`
	// LatencyMS 是探測本身的耗時。未實際送出探測時為 0。
	LatencyMS float64 `json:"latencyMs"`
	// Error 是失敗原因。不含連線字串等憑證，與 /healthz 的取捨相同。
	Error string `json:"error,omitempty"`
	// Detail 放該依賴特有的補充數值（連線池水位、Redis 記憶體用量…）。
	Detail map[string]any `json:"detail,omitempty"`
}

// monitorPayload 是 GET /api/admin/monitor 的完整回應。
//
// OK 欄位沿用全站慣例（response.go），讓前端只需要學會一種失敗形狀。
// 依賴探測失敗不會讓整個請求回 5xx：管理員正是最需要看到「哪一個掛了」的
// 時候，因此依賴狀態放在 body 裡，狀態碼維持 200。
type monitorPayload struct {
	OK    bool                        `json:"ok"`
	Now   string                      `json:"now"`
	Forum string                      `json:"forum"`
	Stats metrics.Snapshot            `json:"stats"`
	Deps  map[string]dependencyStatus `json:"dependencies"`
	// ProbesMS 是三個依賴探測的端到端耗時。它存在的理由是讓「儀表板轉圈」
	// 有辦法被解釋：管理員看到數字不更新時，可以先看這個值判斷是依賴卡住
	// 還是頁面本身有問題。
	ProbesMS float64 `json:"probesMs"`
	// ClientIPTrust 說明「這次請求的來源 IP 是怎麼被判定出來的」。
	//
	// 它存在的理由不是好看：限流與 IP 封鎖都建立在那個判定上，而在未設定
	// TRUSTED_PROXY_CIDRS 的部署裡，那個判定是「使用者自己送的標頭優先」。
	// 把它放在儀表板上，管理員才看得出自己處在哪一種信任模型裡 —— 否則
	// 「我的 IP 位置在監控頁上怎麼變來變去」與「我的封鎖怎麼失效了」是兩個
	// 沒有任何提示的症狀。
	ClientIPTrust clientIPTrust `json:"clientIpTrust"`
}

// clientIPTrust 是目前生效的可信任代理設定。
type clientIPTrust struct {
	// Mode 是 "trusted-proxies" 或 "legacy-headers"，對應介面上的兩種狀態。
	Mode string `json:"mode"`
	// Trusted 列出實際生效的位址段。宣告了但一筆都解析不出來時它會是空陣列，
	// 而 Mode 會退回 legacy-headers —— 那個組合本身就是「設定寫錯了」的訊號。
	Trusted []string `json:"trusted"`
	// Declared 是設定檔原文。讓管理員看得到「我寫了什麼」與「程式認得什麼」
	// 的落差，而不只是在結果上乾瞪眼。
	Declared string `json:"declared"`
	// Invalid 列出宣告了卻無法解析成位址段的項目。
	//
	// 它補的是「部分寫錯」這個盲點：只回 Mode 與 Trusted 時，一條打錯的
	// 位址段（例如 172.17.0.0/16 少打一個字）會靜默地永遠不生效，而畫面
	// 顯示的是一份看起來正常的白名單。症狀不是報錯，而是「那台代理的
	// 轉送標頭從未被採信」——限流與封鎖的分桶鍵因此變成代理的位址。
	// 宣告了但**全部**寫錯時它會列出全部項目，與 Declared 一起構成
	// 「設定完全無效」這個狀態。
	Invalid []string `json:"invalid"`
}

// handleAdminMonitor 回傳整份監控資料。
//
// GET only：這支端點沒有任何「動作」，允許其他方法只會讓它變成一個可以被
// 意外觸發的探測點。
func (s *Server) handleAdminMonitor(w http.ResponseWriter, r *http.Request) {
	// 身分檢查放在 method 分派之前，理由與其他後台端點相同：未登入者不該用
	// 「不支援的方法」反覆探測後台路由是否存在。
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	// metrics 為 nil 只可能發生在沒有注入它的測試；正式服務一定有。回一個
	// 明確的失敗而不是空資料，避免頁面顯示「一切正常」而其實沒有在量。
	if s.metrics == nil {
		internalError(w, "metrics registry is not available")
		return
	}

	// 限流器計數必須在取快照之前灌入，否則快照裡的 rateLimits 會是上一次
	// 刷新時的數字。
	s.metrics.SetLimitStats(s.limitStats())

	probesStart := time.Now()
	deps := s.probeDependencies(r.Context())
	payload := monitorPayload{
		OK:       true,
		Now:      time.Now().UTC().Format(time.RFC3339),
		Forum:    s.cfg.ForumName,
		Deps:     deps,
		ProbesMS: elapsedMS(probesStart),
	}
	// 快照在探測之後才取：ReadMemStats 會 stop-the-world 數十微秒到數毫秒，
	// 把它排在最後可以讓它量到的是「含依賴探測」的較真實狀態，而不是量到
	// 探測進行到一半的時間點。
	payload.Stats = s.metrics.Snapshot(s.cfg.MonitorRetentionHours)
	payload.ClientIPTrust = s.trustReport()

	writeOK(w, payload)
}

// trustReport 把目前的信任模型整理成給監控頁顯示的樣子。
//
// 兩種「設定寫錯」要分開回報，因為它們的症狀不同：
//
//   - 宣告了但一筆都解析不出來：Mode 是 legacy-headers 而 Declared 非空。
//     這個組合在舊的行為下完全不可見（症狀是限流與封鎖可被偽造標頭繞過，
//     卻沒有任何一行 log）。
//   - 部分解析失敗：Mode 是 trusted-proxies，而 Invalid 非空。若只回
//     Mode 與 Trusted，那條寫錯的位址段會靜默地永遠不生效，而畫面看起來
//     完全正常 —— 這是這一欄存在的理由（見 clientIPTrust.Invalid）。
func (s *Server) trustReport() clientIPTrust {
	report := clientIPTrust{Mode: "legacy-headers", Trusted: []string{}, Invalid: []string{}}
	if s.trustedProxies == nil {
		return report
	}
	report.Declared = s.trustedProxies.declared
	report.Invalid = append(report.Invalid, s.trustedProxies.invalid...)
	if !s.trustedProxies.configured {
		return report
	}
	report.Mode = "trusted-proxies"
	for _, network := range s.trustedProxies.nets {
		report.Trusted = append(report.Trusted, network.String())
	}
	return report
}

// probeDependencies 並行探測三個外部依賴。
//
// 為什麼並行：三個探測各自有兩秒逾時，串起來最壞是六秒 —— 而那正是瀏覽器
// 自動刷新最不能容忍的情況（依賴故障時，頁面會連「Redis 掛了但限流器還在
// 擋人」這種最該看到的資訊都來不及顯示）。
//
// 三者的 client 都設計為可併行呼叫：database/sql 的連線池、go-redis 的
// 連線池與 es.Client（內部是 http.Client）都沒有跨呼叫的序列化需求，因此
// 這裡不需要額外的鎖。WaitGroup 只用於等待，不承載資料；結果寫進 map 的
// 動作各自發生在不同的 key 上，因此也不競爭（map 本身只被主 goroutine 讀）。
func (s *Server) probeDependencies(parent context.Context) map[string]dependencyStatus {
	deps := make(map[string]dependencyStatus, 3)
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		collect = func(name string, status dependencyStatus) {
			mu.Lock()
			deps[name] = status
			mu.Unlock()
		}
	)

	run := func(name string, probe func(context.Context) dependencyStatus) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collect(name, probe(parent))
		}()
	}

	run("mysql", s.probeDatabase)
	run("redis", s.probeRedis)
	run("search", s.probeSearch)
	wg.Wait()

	return deps
}

// probeDatabase 回報 MySQL 連線池的水位與探測延遲。
//
// 連線池的四個數字（open / inUse / idle / waitCount）比「ping 有沒有成功」
// 更有診斷價值：池滿時應用層不會看到錯誤，只會看到延遲上升，而 waitCount
// 與 waitDuration 正是那個延遲的來源。
func (s *Server) probeDatabase(parent context.Context) dependencyStatus {
	// nil 防護與這個 handler 裡其他三個依賴一致（s.metrics、s.mediaRedis、
	// s.es）。差異在於崩潰的後果：probeDependencies 是用 go func() 啟動這三個
	// 探測，而 goroutine 裡的 panic **不會**被 net/http 的 per-connection
	// recover 接住（那個只涵蓋 handler 本身的 goroutine）。因此少了這一段，
	// 「資料庫沒接上」會讓整個行程崩潰，而不是回一個 dependencyStatus。
	// 以目前的 main.go 不可達（OpenMySQL 失敗會直接 Fatal），但那正是未來
	// 「監控要能在資料庫沒接上時仍然開得起來」會踩到的第一顆地雷。
	if s.db == nil {
		return dependencyStatus{State: "disabled", Detail: map[string]any{"engine": "none"}}
	}

	ctx, cancel := context.WithTimeout(parent, dependencyProbeTimeout)
	defer cancel()

	start := time.Now()
	status := dependencyStatus{State: "ok"}
	if err := s.db.PingContext(ctx); err != nil {
		status.State = "down"
		status.Error = err.Error()
	}

	stats := s.db.Stats()
	status.LatencyMS = elapsedMS(start)
	status.Detail = map[string]any{
		"maxOpenConnections": stats.MaxOpenConnections,
		"openConnections":    stats.OpenConnections,
		"inUse":              stats.InUse,
		"idle":               stats.Idle,
		"waitCount":          stats.WaitCount,
		"waitDurationMs":     float64(stats.WaitDuration) / float64(time.Millisecond),
		"maxIdleClosed":      stats.MaxIdleClosed,
		"maxLifetimeClosed":  stats.MaxLifetimeClosed,
	}
	return status
}

// probeRedis 回報 Redis 的探測延遲、連線池水位、鍵數與記憶體用量。
//
// INFO 與 DBSIZE 都是 best-effort：Redis 可能被設成不允許這些指令的權限
// （例如只給 GET/SET 的 ACL），那時只少了 Detail 裡的幾個數字，探測本身
// 仍然成功 —— 把它們升級成「Redis 掛了」是誤導。
func (s *Server) probeRedis(parent context.Context) dependencyStatus {
	if s.mediaRedis == nil {
		// nil 代表建構時沒有注入（媒體功能未啟用），與「連不上」不同：前者
		// 不該讓整站被判為不就緒（與 /healthz 的取捨相同）。
		return dependencyStatus{State: "disabled"}
	}

	ctx, cancel := context.WithTimeout(parent, dependencyProbeTimeout)
	defer cancel()

	start := time.Now()
	if err := s.mediaRedis.Ping(ctx).Err(); err != nil {
		return dependencyStatus{State: "down", Error: err.Error(), LatencyMS: elapsedMS(start)}
	}
	status := dependencyStatus{State: "ok", LatencyMS: elapsedMS(start)}

	pool := s.mediaRedis.PoolStats()
	status.Detail = map[string]any{
		"poolHits":       pool.Hits,
		"poolMisses":     pool.Misses,
		"poolTimeouts":   pool.Timeouts,
		"poolTotalConns": pool.TotalConns,
		"poolIdleConns":  pool.IdleConns,
		"poolStaleConns": pool.StaleConns,
	}

	if size, err := s.mediaRedis.DBSize(ctx).Result(); err == nil {
		status.Detail["keys"] = size
	}
	if used, ok := redisMemoryBytes(ctx, s.mediaRedis); ok {
		status.Detail["usedMemoryBytes"] = used
	}
	return status
}

// probeSearch 回報搜尋引擎的狀態。
//
// 三種狀態刻意分開：ES 沒設定（disabled）、設了但連不上（down）、正常（ok）。
// 依 /healthz 的說明，ES 掛掉不等於整站不就緒 —— 搜尋會退回 MySQL LIKE，
// 因此這個狀態只影響搜尋品質，而儀表板正是用來在品質變差時發現這件事的。
func (s *Server) probeSearch(parent context.Context) dependencyStatus {
	if !s.esEnabled() {
		return dependencyStatus{State: "disabled", Detail: map[string]any{"engine": "mysql"}}
	}

	ctx, cancel := context.WithTimeout(parent, dependencyProbeTimeout)
	defer cancel()

	start := time.Now()
	status := dependencyStatus{State: "ok", Detail: map[string]any{"engine": "elasticsearch"}}
	if err := s.es.Ping(ctx); err != nil {
		status.State = "down"
		status.Error = err.Error()
	}
	status.LatencyMS = elapsedMS(start)
	return status
}

// elapsedMS 計算自 start 以來經過的毫秒數。
func elapsedMS(start time.Time) float64 {
	return float64(time.Since(start)) / float64(time.Millisecond)
}

/* ==========================================================================
   Redis INFO 解析
   ========================================================================== */

// redisCommander 是 redisMemoryBytes 實際用到的 Redis 介面。
//
// 宣告成介面（而不是直接吃 *redis.Client）是為了讓 used_memory 的解析可以
// 在測試中以假實作驅動，不必真的連一台 Redis。介面只宣告需要的那一個方法，
// 因為介面一旦多宣告一個方法，*redis.Client 就得多滿足一個與本檔無關的
// 承諾。
type redisCommander interface {
	Info(ctx context.Context, section ...string) *redis.StringCmd
}

// redisMemoryBytes 從 INFO memory 取出 used_memory（bytes）。
//
// 自己解析字串而不依賴任何結構化解析：Client.Info 回傳的是原始文字
// （依 Redis 版本可能帶 # 註解行與 \r\n），而我們只要一個數字。找不到就回
// false —— 那是「這個 Redis 沒給我們 memory 資訊」，不是錯誤。
func redisMemoryBytes(ctx context.Context, client redisCommander) (int64, bool) {
	raw, err := client.Info(ctx, "memory").Result()
	if err != nil {
		return 0, false
	}
	for line := range strings.Lines(raw) {
		name, value, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(name) != "used_memory" {
			continue
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}
