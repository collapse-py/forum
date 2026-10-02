package httpapi

/*
封鎖名單與限流的串接（httpapi/blocklist.go）。

這個檔案是「Redis 裡的封鎖名單」（ipban 套件）與「行程內滑動視窗」
（ratelimit.go）之間的接線點，包含三個部分：

  - withBlocklistHandler  把「先查封鎖、再查限流」組成唯一一個中介層
  - handleAdminBlocks     後臺的封鎖／解封與清單
  - ipbanBroken + 記錄節流  讓「Redis 故障 → fail open」這件事可觀察但不灌爆 log

為什麼封鎖與限流是兩個獨立的機制（而不是把封鎖做成「額度設成 0」）

	「把某個 IP 的額度設成 0」聽起來很省事，但它是錯的：限流的記憶體狀態在
	行程重啟後消失、而且不跨行程。攻擊者只要等一次部署就重新拿到滿額度 ——
	也就是說，用限流實作的「封鎖」在一次維護窗口之後就自動解除，而且沒有
	任何人收到通知。封鎖必須存在於行程之外。

失敗時 fail open，以及為什麼

	Redis 故障時封鎖檢查會失敗，而這裡**放行**。理由寫在 ipban 套件檔頭：
	讓封鎖檢查失敗就擋掉所有人，會把一次 Redis 抖動變成「整站不能發文」，
	而那比「Redis 掛掉期間封鎖失效」嚴重得多（Redis 掛掉時本站的登入本來
	就已經受影響 —— session 查不到等於未登入）。

	ipbanLogThrottle 是這個決定必要的配套：fail open 時每個請求都會失敗，若
	每次都寫一行警告，一個 Redis 故障可以在幾秒內灌滿整個 log。因此記錄
	只在「狀態改變」時寫一次。

	這個決定的可觀察代價：攻擊者只要製造 Redis 壓力，就能暫時解除對自己的
	封鎖。監控頁的 Redis 探測會變紅，那是這個狀態唯一的提示。
*/

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"forum/forum/audit"
	"forum/forum/ipban"
	"forum/forum/logger"
)

// ipbanLogThrottle 是「封鎖檢查失敗」警告的最小間隔。
//
// 90 秒的取捨：夠長到 Redis 故障不會灌爆 log，夠短到管理員在處理故障的同時
// 還能看到「封鎖功能現在是失效的」。每 90 秒一行是一個可以接受的量。
const ipbanLogThrottle = 90 * time.Second

// ipbanBroken 記錄「封鎖檢查最近一次是否失敗」，用於節流記錄。
//
// 用 atomic 而不是 mutex：它每個請求都會讀一次（一個 bool 的 atomic load 是
// 奈秒級且無競爭），而 mutex 會讓這條熱路徑多一次鎖的爭用。
var ipbanBroken atomic.Bool

// logBlocklistFailure 在封鎖檢查失敗時記錄警告，且每 ipbanLogThrottle 只記一次。
//
// 成功時（logBlocklistRecovered）會把狀態翻回來，讓故障結束後又能記下一次
// —— 否則一次故障會讓之後所有的故障都靜默，而那是比 log 稍吵嚴重得多的問題。
func logBlocklistFailure(err error) {
	if ipbanBroken.Swap(true) {
		return
	}
	// 刻意不含 IP：它來自使用者可控的表頭（見 clientIP 的說明），而 log 的
	// 讀取範圍通常比資料庫寬。錯誤本身也不含它。
	logger.Warnf("[BLOCKLIST] 封鎖檢查失敗，本次請求一律放行（fail open）: %v", err)
}

func logBlocklistRecovered() {
	if ipbanBroken.Swap(false) {
		logger.Infof("[BLOCKLIST] 封鎖檢查恢復正常")
	}
}

// withBlocklistHandler 組裝「先查封鎖、再查限流」的中介層。
//
// 存在的理由是**掛載位置的正確性**：限流器掛在哪幾條路由上由 server.go 決定，
// 而封鎖必須涵蓋同一組路由才不會出現「被封鎖的人仍可從某條沒掛封鎖的路由
// 寫入」。讓這個組合只有一個進入點，就不會有兩處掛載因而不同步。
//
// 兩者的順序不可交換：先查封鎖（一次 Redis 往返，且被封鎖時直接回 403）再查
// 限流（記憶體）。反過來的話，被封鎖的 IP 會先累積限流計數，而那個計數會
// 在解除封鎖之後仍然生效 —— 一個沒有管理員動作卻持續存在的隱藏狀態。
func withBlocklistHandler(rl *RateLimiter, store *ipban.Store, next http.HandlerFunc) http.HandlerFunc {
	limited := rl.Middleware(next)
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			// 沒有封鎖名單（測試以 struct literal 構造 Server，或 Redis 未設定）。
			// 直接走限流 —— 見檔頭的 fail open。
			limited(w, r)
			return
		}
		ip := clientIP(r)
		banned, until, err := store.IsBanned(r.Context(), ip)
		if err != nil {
			logBlocklistFailure(err)
			limited(w, r)
			return
		}
		logBlocklistRecovered()
		if banned {
			writeBannedResponse(w, r, until)
			return
		}
		limited(w, r)
	}
}

// bannedRequestKey 是 metricsMiddleware 放進 context 的「本次請求擋下狀態」容器。
//
// 為什麼要透過 context 傳遞，而不是讓 metricsMiddleware 從狀態碼反推：403 在這
// 個專案裡同時來自封鎖名單、requireAdminForum（不是管理員）與 isTrustedOrigin
// （CSRF）。從狀態碼反推會讓「這個位址被硬闖了 N 次」這個數字混入完全不相干的
// 403，而管理員看到「被封鎖的位址正在嘗試寫入」時會據此判斷封鎖有效 —— 用錯的
// 數字會讓他得出相反的結論。
//
// 傳的是**指標**而不是 bool：blocklist 中介層在 metricsMiddleware 的內側，它
// 拿到的 r 是 metricsMiddleware 傳下去的那一份副本，因此在內層呼叫
// r.WithContext 造出來的新 request 不會回到外層。指標讓內外兩層指向同一個
// 可變狀態，這是 context 唯一被用來攜帶「跨中介層的單一事實」的地方（其他跨層
// 狀態走的是包裝 ResponseWriter 或明確的函式參數）。
//
// 替代方案是讓 metricsMiddleware 掛在 blocklist 的內側，但那會改變 stats 涵蓋
// 的請求集合 —— 它現在刻意涵蓋整棵 mux，包括被擋下的請求，而那正是「有人在打
// 這個位址」的證據。
type bannedRequestKey struct{}

// bannedMarker 是本次請求的擋下狀態。
//
// 單一 bool 欄位而非一個 map：目前只有封鎖名單一種來源需要標記，而「未來會有
// 第二種」不是現在就該為它設計的理由。真的加了第二種時，這裡換成計數器。
type bannedMarker struct {
	blocked bool
}

// withBannedMarker 在 context 裡放一個本次請求專用的 marker，回傳帶它的 request。
//
// 只由 metricsMiddleware 呼叫（見它的說明）。base 刻意用 r.Context() 而不是
// context.Background()：沿用原來的 context 才不會讓下游 handler 看不到用戶端
// 取消與逾時，而 marker 的生命週期只到本次回應結束，不會外洩。
func withBannedMarker(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), bannedRequestKey{}, &bannedMarker{}))
}

// markBannedRequest 標記這個請求是因封鎖名單而被擋下的。
//
// context 裡沒有 marker 時（例如這支中介層被單獨測試而沒有包在
// metricsMiddleware 內）靜靜不做事：那個情境沒有任何人在讀這個旗標，而為了
// 一個沒有人在讀的狀態去改變行為會讓單元測試變複雜。
func markBannedRequest(r *http.Request) {
	if marker, ok := r.Context().Value(bannedRequestKey{}).(*bannedMarker); ok {
		marker.blocked = true
	}
}

// writeBannedResponse 送出被封鎖的回應。
//
// 回應刻意不含「你被封鎖到什麼時候」：那是一個足以被拿來枚舉封鎖名單的資訊
// （攻擊者可以輪流試不同 IP 並記下哪些被回 403）。Retry-After 照樣送出，因為
// 它是 RFC 9110 對「暫時性拒絕」的標準做法，而且能讓自動化的客戶端在到期後
// 自動恢復 —— 那正是「限時封鎖」想要的行為。
//
// r 只為了寫進 context 標記，讓每 IP 監控知道這次 403 是「封鎖」而不是
// 「不是管理員」（見 markBannedRequest）。刻意不把 IP 或到期時間放進去：那兩者
// 已經由 metricsMiddleware 自己從 request 取得，而這裡多放一個值就多一個
// 「兩邊算出不同結果」的機會。
func writeBannedResponse(w http.ResponseWriter, r *http.Request, until time.Time) {
	markBannedRequest(r)
	if wait := time.Until(until); wait > 0 {
		seconds := int((wait + time.Second - 1) / time.Second)
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
	}
	// 403 而非 429：語意是「不被允許」而不是「請慢一點」。被封鎖的人看到
	// 429 會以為只要等一下就好，於是接著不斷重試。
	writeError(w, http.StatusForbidden, "blocked")
}

/* ==========================================================================
   後臺 API
   ========================================================================== */

// adminBlockRequest 是封鎖與解封共用的請求主體。
type adminBlockRequest struct {
	IP string `json:"ip"`
	// Minutes 是封鎖長度（分鐘）。<= 0 代表解封（見 handleAdminBlocks 的說明）。
	Minutes int `json:"minutes"`
	// Reason 是給稽核紀錄的說明。刻意不存進 Redis（見 ipban.Ban 的說明），
	// 但**必須**從前端送來 —— 否則稽核紀錄裡只有「封了某個 IP」，而「為什麼」
	// 正是事後最需要知道的資訊。
	Reason string `json:"reason"`
}

// adminBlockView 是一筆封鎖對外的樣子。
type adminBlockView struct {
	IP        string `json:"ip"`
	ExpiresAt string `json:"expiresAt"`
	// RemainingSeconds 與 ExpiresAt 一起給：前端可以用它算「多久」而不用自己
	// 做時區換算（而那個換算在 UTC 與本地時間之間很容易差一個小時，卻看不出
	// 來是錯的）。
	RemainingSeconds int64 `json:"remainingSeconds"`
}

// handleAdminBlocks 列出封鎖名單，或封鎖／解封某個 IP。
//
//	GET  /api/admin/blocks  列出目前有效的封鎖
//	POST /api/admin/blocks  {"ip":"…","minutes":1440,"reason":"…"}
//	                       minutes <= 0 代表解封
//
// 單一路由處理兩種動作：它們是同一件事的兩面（名單的新增與移除），而分成
// 兩條路由會讓「前端忘了其中一條」的失敗模式變得可能 —— 例如只做了 POST 而
// 沒有 DELETE，那使用者就只能加不能減。
//
// 刻意沒有限流：它是管理員在處理一個正在發生的濫用時要用的端點，限流它的
// 唯一效果是「攻擊還在、但管理員封不了」。
func (s *Server) handleAdminBlocks(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.listAdminBlocks(w, r)
	case http.MethodPost:
		// 來源檢查寫在 method 分派之內：GET 是唯讀，不需要防護。
		if !s.isTrustedOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		s.mutateAdminBlock(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) listAdminBlocks(w http.ResponseWriter, r *http.Request) {
	if s.blocks == nil {
		// 沒有封鎖名單可用。回空清單加 available:false 而不是 500：沒有 Redis
		// 的部署本來就沒有封鎖功能，讓後臺頁面開得起來比讓它報錯好，而
		// available 讓前端能說明「這台站沒有封鎖功能」而不是假裝名單是空的。
		writeOK(w, map[string]any{"ok": true, "items": []adminBlockView{}, "available": false})
		return
	}
	entries, err := s.blocks.List(r.Context(), 200)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[BLOCKLIST] 列出失敗: %v", err)
		internalError(w, "unable to list blocked addresses")
		return
	}
	items := make([]adminBlockView, 0, len(entries))
	for _, entry := range entries {
		items = append(items, adminBlockView{
			IP:               entry.IP,
			ExpiresAt:        entry.Expires.UTC().Format(time.RFC3339),
			RemainingSeconds: entry.RemainingSeconds(),
		})
	}
	writeOK(w, map[string]any{"ok": true, "items": items, "available": true})
}

func (s *Server) mutateAdminBlock(w http.ResponseWriter, r *http.Request) {
	var req adminBlockRequest
	// 4 KiB 上限：這個 body 裡只有 IP、一個數字與一段理由。
	if err := decodeLimitedJSON(w, r, &req, 4<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}
	ip := ipban.NormalizeIP(req.IP)
	if ip == "" {
		// 回固定訊息而不是 net 套件的錯誤字串（見 ipban.ErrInvalidIP 的說明）。
		badRequest(w, "IP 格式不正確")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if len([]rune(reason)) > 200 {
		badRequest(w, "原因最多 200 字")
		return
	}

	// minutes <= 0 代表解封。刻意不要求一個明確的 action 欄位：簽入與登出
	// 是同一個資源的兩面，而「你要移除嗎」在介面上是一個動作按鈕，不需要
	// 一個字串欄位來表達。
	if req.Minutes <= 0 {
		s.unblockAddress(w, r, ip)
		return
	}
	s.blockAddress(w, r, ip, reason, req.Minutes)
}

// blockAddress 封鎖一個 IP 並寫稽核。
//
// 封鎖在 Redis、稽核在 MySQL，兩者沒有共同交易（理由與 session 的強制登出
// 完全相同，見 session_admin_handlers.go 檔頭）。方向是**先封再記**：寧可
// 「封了但沒紀錄」，也不要「紀錄說封了但其實沒封」—— 後者會讓管理員在查
// 稽核紀錄時得到一個不實的結論。
func (s *Server) blockAddress(w http.ResponseWriter, r *http.Request, ip, reason string, minutes int) {
	if s.blocks == nil {
		// 沒有 Redis 就沒有封鎖名單。回 503（Service Unavailable）而不是 500：
		// 這個功能的不可用是「外部依賴沒接上」，不是「伺服器壞了」，而 503
		// 讓使用者知道稍後可能會恢復。
		writeError(w, http.StatusServiceUnavailable, "封鎖名單不可用（缺少 Redis）")
		return
	}
	now := time.Now()
	until := now.Add(time.Duration(minutes) * time.Minute)
	// 超過上限時就地收斂到上限，而不是回錯 —— 見 ipban.MaxBanDuration 的說明：
	// 「永久」在這個資料結構裡沒有辦法安全地表達。介面上寫出了上限，讓管理員
	// 看得到自己被收斂了。
	if max := now.Add(ipban.MaxBanDuration); until.After(max) {
		until = max
	}

	if err := s.blocks.Ban(r.Context(), ip, until); err != nil {
		if errors.Is(err, ipban.ErrInvalidIP) {
			badRequest(w, "IP 格式不正確")
			return
		}
		logger.ErrorfContext(r.Context(), "[BLOCKLIST] 封鎖失敗 ip=%s: %v", ip, err)
		internalError(w, "unable to block address")
		return
	}

	if err := s.recordBlockAction(r, auditActionBlockAdd, ip, reason, until); err != nil {
		// 動作已生效，因此明說「封了但沒紀錄」。
		logger.ErrorfContext(r.Context(), "[BLOCKLIST] 已封鎖 %s，但稽核寫入失敗: %v", ip, err)
		writeError(w, http.StatusInternalServerError, "已封鎖，但稽核紀錄寫入失敗")
		return
	}
	writeOK(w, map[string]any{
		"ok":        true,
		"ip":        ip,
		"expiresAt": until.UTC().Format(time.RFC3339),
	})
}

func (s *Server) unblockAddress(w http.ResponseWriter, r *http.Request, ip string) {
	if s.blocks == nil {
		writeError(w, http.StatusServiceUnavailable, "封鎖名單不可用（缺少 Redis）")
		return
	}
	existed, err := s.blocks.Unban(r.Context(), ip)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[BLOCKLIST] 解封失敗 ip=%s: %v", ip, err)
		internalError(w, "unable to unblock address")
		return
	}
	if existed {
		if err := s.recordBlockAction(r, auditActionBlockRemove, ip, "", time.Time{}); err != nil {
			logger.ErrorfContext(r.Context(), "[BLOCKLIST] 已解封 %s，但稽核寫入失敗: %v", ip, err)
			writeError(w, http.StatusInternalServerError, "已解封，但稽核紀錄寫入失敗")
			return
		}
	}
	// 解除一個不存在的封鎖回 200 且 existed=false：那不是錯誤，管理員點了
	// 「解封」而它已經解封了，就是他想要的結果。
	writeOK(w, map[string]any{"ok": true, "ip": ip, "existed": existed})
}

// recordBlockAction 寫一筆封鎖／解封的稽核紀錄。
func (s *Server) recordBlockAction(r *http.Request, action, ip, reason string, until time.Time) error {
	changes := []audit.Change{{Field: "ip", Before: "", After: ip}}
	if reason != "" {
		changes = append(changes, audit.Change{Field: "reason", Before: "", After: reason})
	}
	if !until.IsZero() {
		changes = append(changes, audit.Change{Field: "expiresAt", Before: "", After: until.UTC().Format(time.RFC3339)})
	}
	// 刻意開一個只為了寫一行的交易（理由同 session 強制登出）。傳 nil 會讓
	// recordAdminAction 整筆跳過 —— 那個 nil 容忍是為了讓「稽核表還沒建好」
	// 時後臺仍可操作，用在這裡會讓封鎖永遠沒有稽核紀錄。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.recordAdminAction(r, tx, action, audit.TargetIP, ip, ip, changes...); err != nil {
		return err
	}
	return tx.Commit()
}
