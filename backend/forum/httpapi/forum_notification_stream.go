/*
即時通知推播（backend/forum/httpapi/forum_notification_stream.go）

一、路由

	GET /api/forum/notifications/ws -> handleForumNotificationStream（需登入）

	升級成 WebSocket 之後那條連線只做一件事：把「剛剛建立的通知」推給收件者
	目前開著的頁面。前端據此即時更新導覽列的紅點，並在通知頁把新的一則插到
	最前面，不必重新整理。通知本身的寫入仍在 forum_notification_handlers.go
	的 createForumNotification，這裡只負責「已經寫進資料庫的那一則怎麼送到
	他眼前」。

二、為什麼是 WebSocket，而不是輪詢或 SSE

	輪詢（每 N 秒問一次 unread-count）在這一站不可行：每一次讀取都要付一次
	索引查詢的成本，而輪詢讓它變成人數 × 分頁數 × 頻率的固定負載 ——
	即使完全沒有通知也照付。

	SSE 只需要 http.Flusher（logger 的 StatusWriter 已經轉發它），對這個
	「伺服器推、瀏覽器收」的需求其實夠用，而且不必新增相依。選 WebSocket
	的理由只有一個但很實際：這條連線是雙向的，未來要在同一條連線上做
	「已讀回報」「正在輸入」這類上行訊息時不必再開第二條；代價（一個相依、
	升級後寫入必須序列化、心跳與逾時）在這個規模是可控的。反過來說，若確定
	永遠只有單向下行，SSE 會是更小的選擇 —— 記在這裡，免得未來有人以為
	WebSocket 是唯一做法。

三、這個 handler「升級完就回傳」

	升級成功後連線由下面的三個 goroutine 接手，handler 立刻回傳。理由不是
	效能而是帳目正確：LoggingMiddleware 與 metricsMiddleware 都是在 handler
	回傳之後才記帳，若 handler 留在連線裡等到對方斷線，那一筆存取記錄就會變成
	「status=200、duration=連線存續時間」，而監控頁的 inFlight 會把每一條開著
	的連線一直算成「進行中的請求」。

	因此升級之後所有需要 DB/Redis 的工作都在 goroutine 裡做，而且**不使用
	r.Context()** —— handler 一回傳，net/http 就會取消那個 context。握手當下
	還來得及抄下來的東西（session token、email、是否管理員）就在那裡抄下來。

四、訊息（伺服器 -> 瀏覽器，兩種，都是 JSON）

	{"type":"ready","unreadCount":3}
	    連線建立時送一次。它是前端未讀數的權威校正點：重新連線時補上斷線
	    期間累積的差額，因此前端不必為此再打一次 GET。
	{"type":"notification","unreadCount":4,"item":{...}}
	    item 的欄位與 GET /api/forum/notifications 的清單項目完全相同
	    （同一份組裝程式碼 forumNotificationItemFromRow），前端因此可以直接
	    把它插進列表。

	瀏覽器 -> 伺服器：沒有。上行訊息一律忽略（只讀不解析），但**仍然要持續
	讀取**：pong 是在 ReadMessage 裡處理的，而 pong 是「對方還活著」的唯一
	證據（瀏覽器不能從 JS 發 ping）。

五、三個上限（沒有上限的長連線等於一個免費的 DoS 介面）

	同一使用者同時 8 條：超過時關掉**最舊**的一條，而不是拒絕新的。拒絕新的
	會讓「剛開的那個分頁」永遠沒有即時更新，偏偏它才是使用者正在看的那一個；
	最舊的分頁最可能已經被擱置。被關掉的那一條只是不再有即時推播，重新整理
	時未讀數照樣正確（那是 GET 的責任，不是這條連線的）。
	全部同時 1000 條：滿了就拒絕新連線（4001 + 原因），不淘汰任何人 —— 全域
	額度若也用淘汰制，一個人開分頁就能把別人的連線踢掉。
	還有一個隱含的上限：慢速消費者。傳送佇列滿了（對方讀得比我們寫得慢）就是
	關掉那條連線，而不是阻塞 —— 阻塞會讓一個卡住的瀏覽器把整條推播路徑拖下水。

	四種關閉原因各有自己的應用層 close code（4000 系列，見下方常數宣告），而
	前端靠它決定「要不要重連」：被較新的連線取代時**不重連** —— 額度是每個
	使用者、跨裝置共用的，重連只會把對方踢掉、然後換自己被踢，兩條連線從此
	互相汰換。其餘三種都照退避重連（額度會釋放、session 可能重新登入）。

六、每分鐘重新確認一次身分

	session 可撤銷是這個專案選用 Redis 的主要理由（見 docs/ARCHITECTURE.md），
	因此一條開著的連線也必須跟著失效，否則「管理員強制登出某支被盜用的
	session」之後，那條連線仍然持續收到私人的通知。每分鐘用握手時抄下來的
	token 確認 session 還在，並重查一次停權狀態（管理員跳過，與 requireLogin
	同一條規則）。任何一步失敗 —— 包含 Redis 或 MySQL 故障 —— 一律視為失效
	並關閉連線：驗證失敗時放行不是這個專案的做法。

七、多實例（水平擴充）的已知限制

	這個 hub 在行程內。同一個使用者的兩個分頁連到不同實例時，只有「產生通知
	的那個實例」上的連線收得到推播，另一個實例要等下一次頁面載入才看到。
	要修就是把 publish 改成 Redis pub/sub 再各自轉發，這裡刻意不做：本專案的
	部署是單一行程，而多一層廣播等於多一組「實例重啟／訊息重播」的失敗模式。
	見 docs/KNOWN_ISSUES.md 的對應條目。
*/
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"forum/forum/logger"

	"github.com/gorilla/websocket"
)

const (
	// notificationStreamMaxPerUser 與 notificationStreamMaxTotal 是連線數的
	// 上限，理由見檔頭第五點。兩者都是「這個規模的論壇」的常數，不是每個
	// 部署點該各自決定的事，因此不進設定檔。
	notificationStreamMaxPerUser = 8
	notificationStreamMaxTotal   = 1000

	// notificationStreamSendBuffer 是每條連線的傳送佇列長度。推播是「有就送」
	// 的即時訊息，堆積沒有意義：滿了就代表對方讀得比我們寫得慢。
	notificationStreamSendBuffer = 16

	// 心跳與逾時。PongWait 必須明顯大於 PingPeriod（這裡取兩倍以上），否則
	// 正常的往返延遲就會被誤判成斷線。WriteWait 是一次寫入（含 ping 與
	// close frame）的上限，超過就當這條連線壞了。
	notificationStreamPingPeriod = 30 * time.Second
	notificationStreamPongWait   = 70 * time.Second
	notificationStreamWriteWait  = 10 * time.Second

	// 重新確認身分的間隔，見檔頭第六點。一次確認是一次 Redis HGET
	// （session 還在不在）加一次 MySQL SELECT（停權狀態），不需要更頻繁。
	notificationStreamSessionCheckInterval = 60 * time.Second

	// 背景查詢（未讀數、通知內容）的逾時。它們跑在沒有請求逾時的 goroutine
	// 裡，必須自己設上限，否則 DB 卡住時那些 goroutine 會一直掛著。
	notificationStreamQueryTimeout = 5 * time.Second

	// 上行訊息的讀取上限，壓到最小。瀏覽器端不會送任何東西，會送出東西的
	// 不是我們的程式。
	notificationStreamReadLimit = 512

	// 推播訊息的 type 值。前端以它分派，因此與 kind 一樣是契約的一部分。
	notificationStreamMessageReady  = "ready"
	notificationStreamMessagePushed = "notification"

	// close frame 的應用層 close code（4000–4999 是應用程式自訂的區間，因此
	// 瀏覽器不會把它當成協議錯誤）。前端以它決定「要不要重連」，見檔頭第五點。
	//
	// 全部用 1008（policy violation）也可以，但那樣前端只能比對原因字串 ——
	// 原因字串是寫給人看的，不該同時當成程式判斷的依據。
	notificationStreamCloseReplaced     = 4000 // 被同一使用者的較新連線取代
	notificationStreamCloseAtCapacity   = 4001 // 全站連線數已達上限
	notificationStreamCloseSlowConsumer = 4002 // 傳送佇列滿了（對方讀得太慢）
	notificationStreamCloseSessionEnded = 4003 // session 失效或帳號被停權
)

// notificationStreamUpgrader 是升級器。
//
// CheckOrigin 回 true 不是「不檢查」：來源檢查已經在 handler 內以
// isTrustedOrigin 明確做過一次（那是全站同一套規則，也會讀 TRUSTED_ORIGINS
// 設定）。交給 gorilla 的預設檢查反而會多出第二份不同定義的規則，兩者分歧時
// 要看哪一份才準就變成運氣問題。
//
// 讀寫緩衝刻意開小：這一條連線只送幾百 bytes 的 JSON。緩衝是在升級當下配置的，
// 而上限是 1000 條連線 —— 每個 KB 都要乘以那個數字。
var notificationStreamUpgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 1024,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// forumNotificationReady 是連線建立時送出的第一則訊息，見檔頭第四點。
type forumNotificationReady struct {
	Type        string `json:"type"`
	UnreadCount int    `json:"unreadCount"`
}

// forumNotificationPush 是一則即時通知，見檔頭第四點。
type forumNotificationPush struct {
	Type        string                `json:"type"`
	UnreadCount int                   `json:"unreadCount"`
	Item        forumNotificationItem `json:"item"`
}

/* ==========================================================================
   連線的集合（hub）
   ========================================================================== */

// notificationStreamSink 是 hub 對「一條連線」的全部需求。
//
// 用介面而不是直接持有 *notificationStreamConn：上限、淘汰與慢速消費者的規則
// 是純粹的資料結構行為，測試不該為此開一條真的 TCP／WebSocket 連線。
type notificationStreamSink interface {
	// deliver 非阻塞地把一段訊息放進該連線的傳送佇列，佇列滿了回 false。
	deliver(payload []byte) bool
	// closeWithReason 關閉連線，並盡力告訴對方「為什麼」與對應的 close code。
	// 必須可以重複呼叫（第一次有效）。
	closeWithReason(code int, reason string)
}

// notificationStreamHub 是所有開著的推播連線，依 email 分組。
//
// 鎖只保護這張表本身：deliver 與 closeWithReason 都在鎖外呼叫。它們會碰
// socket 與其他 goroutine，在鎖內做會讓「對方斷線 -> detach」排在全站推播
// 的後面，等於把一條 socket 的抖動放大成全站延遲。
type notificationStreamHub struct {
	mu sync.Mutex
	// byUser 是 email -> 連線，依加入順序排列（[0] 最舊，超過上限時淘汰它）。
	byUser map[string][]notificationStreamSink
	total  int
}

func newNotificationStreamHub() *notificationStreamHub {
	return &notificationStreamHub{byUser: make(map[string][]notificationStreamSink)}
}

// attach 登記一條連線。
//
// 回傳 replaced 是「因為超過同一使用者的上限而被擠掉」的連線（由呼叫端關閉），
// ok=false 代表全域額度已滿、這條新連線必須被拒絕。
func (h *notificationStreamHub) attach(email string, sink notificationStreamSink) (replaced []notificationStreamSink, ok bool) {
	if h == nil {
		// nil hub 代表這個 Server 沒有推播能力（測試以 struct literal 建構）。
		// 仍然回 ok：連線本身是有效的，只是永遠不會有推播 —— 推播是附加效果，
		// 不該讓「升級成功」取決於它。
		return nil, true
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.total >= notificationStreamMaxTotal {
		return nil, false
	}
	list := append(h.byUser[email], sink)
	h.total++
	for len(list) > notificationStreamMaxPerUser {
		replaced = append(replaced, list[0])
		list = list[1:]
		h.total--
	}
	h.byUser[email] = list
	return replaced, true
}

// detach 移除一條連線。找不到時什麼都不做 —— 被淘汰的連線也會走到這裡，
// 而它已經不在表上了。
func (h *notificationStreamHub) detach(email string, sink notificationStreamSink) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	list := h.byUser[email]
	for i, candidate := range list {
		if candidate == sink {
			list = append(list[:i], list[i+1:]...)
			h.total--
			break
		}
	}
	if len(list) == 0 {
		delete(h.byUser, email)
		return
	}
	h.byUser[email] = list
}

// connected 回報這個人現在有沒有開著的頁面。
//
// 用途是在做任何查詢之前先剪掉「對方根本不在線上」這個大宗情況：
// 建立通知與推播都是 best-effort，沒有人在聽的時候不需要為了組出推播內容
// 多查兩次資料庫。
func (h *notificationStreamHub) connected(email string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byUser[email]) > 0
}

// publish 把一段訊息送給這個人所有開著的頁面，回傳成功與被丟棄的連線數。
//
// 「成功」是「放進佇列」，不是「對方收到了」—— WebSocket 沒有應用層的送達
// 確認，而這裡不需要它：漏掉一則推播的後果只是那個分頁晚一點（重新載入時）
// 才知道，而不是資料不一致。
func (h *notificationStreamHub) publish(email string, payload []byte) (delivered, dropped int) {
	if h == nil {
		return 0, 0
	}
	h.mu.Lock()
	targets := append([]notificationStreamSink(nil), h.byUser[email]...)
	h.mu.Unlock()

	for _, sink := range targets {
		if sink.deliver(payload) {
			delivered++
			continue
		}
		dropped++
		sink.closeWithReason(notificationStreamCloseSlowConsumer, "client is too slow")
	}
	return delivered, dropped
}

/* ==========================================================================
   一條連線
   ========================================================================== */

// notificationStreamConn 是一條已升級的連線。
//
// 寫入（訊息、ping、close frame）只由 writeLoop 這一個 goroutine 執行：
// gorilla/websocket 不允許並行寫入，不是「最好不要」。其他 goroutine 只能
// 透過 send 佇列間接寫，或呼叫 closeWithReason 收線。
type notificationStreamConn struct {
	ws   *websocket.Conn
	send chan []byte

	// done 關閉代表「這條連線要收了」；closeCode 與 closeReason 必須在關閉之前
	// 寫入（channel 的關閉建立 happens-before，writeLoop 讀得到那兩行）。
	done        chan struct{}
	closeCode   int
	closeReason string
	closeOnce   sync.Once
}

func newNotificationStreamConn(ws *websocket.Conn) *notificationStreamConn {
	return &notificationStreamConn{
		ws:   ws,
		send: make(chan []byte, notificationStreamSendBuffer),
		done: make(chan struct{}),
	}
}

// deliver 實作 notificationStreamSink。已經在收線的連線回 true（訊息對它
// 而言只是晚了一步，不是「這個消費者太慢」）。
func (c *notificationStreamConn) deliver(payload []byte) bool {
	select {
	case <-c.done:
		return true
	default:
	}
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

// closeWithReason 實作 notificationStreamSink。重複呼叫只有第一次有效。
func (c *notificationStreamConn) closeWithReason(code int, reason string) {
	c.closeOnce.Do(func() {
		c.closeCode = code
		c.closeReason = reason
		close(c.done)
	})
}

// writeLoop 是這條連線唯一的寫入者：送出佇列裡的訊息、定期 ping，收到 done
// 時送 close frame 並關掉 socket。
func (c *notificationStreamConn) writeLoop() {
	ticker := time.NewTicker(notificationStreamPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			// 收線：把 code 與原因交給對方（best-effort，失敗不影響關閉）。
			// 4000 系列是應用程式自訂的區間，前端靠它決定要不要重連
			// （見常數宣告）—— 因此關閉的原因不只是給人看的診斷資訊。
			_ = c.ws.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(c.closeCode, c.closeReason),
				time.Now().Add(notificationStreamWriteWait))
			_ = c.ws.Close()
			return
		case payload := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(notificationStreamWriteWait))
			if err := c.ws.WriteMessage(websocket.TextMessage, payload); err != nil {
				// 寫不出去＝這條連線壞了。關掉 socket 讓讀取迴圈也結束
				// （它才是負責把連線從 hub 移除的人）。
				_ = c.ws.Close()
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(notificationStreamWriteWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				_ = c.ws.Close()
				return
			}
		}
	}
}

// readLoop 一直讀到對方離開或讀取逾時。回傳代表這條連線結束了。
//
// 讀到的內容一律丟棄（見檔頭第四點），但讀取本身不能省：pong 是由
// ReadMessage 處理的，而 SetPongHandler 更新讀取期限就是心跳的另一半。
func (c *notificationStreamConn) readLoop() {
	c.ws.SetReadLimit(notificationStreamReadLimit)
	_ = c.ws.SetReadDeadline(time.Now().Add(notificationStreamPongWait))
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(notificationStreamPongWait))
	})

	for {
		if _, _, err := c.ws.ReadMessage(); err != nil {
			return
		}
	}
}

/* ==========================================================================
   HTTP handler
   ========================================================================== */

// handleForumNotificationStream 是 GET /api/forum/notifications/ws 的進入點。
//
// 檢查順序與其他通知端點一致：先身分、再方法、最後來源。
func (s *Server) handleForumNotificationStream(w http.ResponseWriter, r *http.Request) {
	// 路由已掛 requireLogin，這裡再確認一次是縱深防禦（與其他通知端點一致）。
	email := s.sessions.ResolveUser(r)
	if email == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	// 跨站 WebSocket 的縱深防禦。SameSite=Lax 已經讓跨站的握手帶不到 cookie，
	// 但「不帶 cookie 的例外」不值得用一條能持續讀取私人通知的長連線去賭；
	// 規則與其他端點相同（TrustedOrigins）。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	// 握手當下還讀得到請求時先抄下來（見檔頭第三點）。ResolveUser 與 IsAdmin
	// 已經各查過一次 Redis，這裡只是多讀兩次同一份標頭。
	// isAdmin 併用 adminPrivilegeCurrent 回查白名單：這條連線會把後台的活動通知推給它，
	// 讓一支過期的 admin session 繼續收到管理員通知，等於把權限收回延遲到連線結束。
	// 刻意不在這裡撤銷 session —— 撤銷會讓下面每輪的 Alive 檢查失敗並關掉連線，而這條
	// 連線是唯讀的；「收回權限」這件事由寫入端點（requireAdminForum）做。
	token := s.sessions.Token(r)
	isAdmin := s.sessions.IsAdmin(r) && s.adminPrivilegeCurrent(r)

	ws, err := notificationStreamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade 自己已經把 HTTP 回應寫出去了（非 101 時是錯誤碼），
		// 這裡只留一筆日誌，別再寫任何東西到同一個 ResponseWriter。
		logger.WarnfContext(r.Context(), "[FORUM] 通知串流升級失敗: %v", err)
		return
	}
	s.serveForumNotificationStream(ws, token, email, isAdmin)
}

// serveForumNotificationStream 登記連線並把後續工作交給 goroutine，自己立刻
// 回傳（見檔頭第三點）。
func (s *Server) serveForumNotificationStream(ws *websocket.Conn, token, email string, isAdmin bool) {
	conn := newNotificationStreamConn(ws)
	replaced, ok := s.notifyStream.attach(email, conn)
	for _, victim := range replaced {
		victim.closeWithReason(notificationStreamCloseReplaced, "replaced by a newer connection")
	}
	if !ok {
		conn.closeWithReason(notificationStreamCloseAtCapacity, "server is at capacity")
		return
	}
	go conn.writeLoop()
	go s.watchForumNotificationSession(conn, token, email, isAdmin)
	go func() {
		// 讀取迴圈是這條連線的「擁有者」：它結束代表對方走了或連線壞了，
		// 因此由它負責把連線從 hub 移除並收掉寫入端。
		conn.readLoop()
		s.notifyStream.detach(email, conn)
		// 對方已經走了，沒有人收得到這個 close frame（1000 只是語意上誠實）。
		conn.closeWithReason(websocket.CloseNormalClosure, "connection closed")
	}()
}

// watchForumNotificationSession 先送 ready，之後每分鐘確認一次這條連線還算
// 不算登入狀態（見檔頭第六點）。
//
// ready 與巡檢共用同一個 goroutine：初始訊息只需要一次查詢，不必為它多養一個
// goroutine 與一條生命週期。
func (s *Server) watchForumNotificationSession(conn *notificationStreamConn, token, email string, isAdmin bool) {
	s.sendForumNotificationReady(conn, email)

	ticker := time.NewTicker(notificationStreamSessionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-conn.done:
			return
		case <-ticker.C:
			if !s.notificationStreamSessionValid(token, email, isAdmin) {
				conn.closeWithReason(notificationStreamCloseSessionEnded, "session ended")
				return
			}
		}
	}
}

// notificationStreamSessionValid 確認「這條連線在握手時用的那個身分」現在還
// 算不算數。任何一步不確定都回 false（fail closed）。
func (s *Server) notificationStreamSessionValid(token, email string, isAdmin bool) bool {
	if s.sessions == nil || !s.sessions.Alive(token) {
		return false
	}
	// 停權檢查只擋非管理員，與 requireLogin 同一條規則（管理員必須進得來
	// 才能處理被停權使用者的檢舉與解封）。
	if isAdmin {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), notificationStreamQueryTimeout)
	defer cancel()
	suspended, err := s.isForumUserSuspended(ctx, email)
	return err == nil && !suspended
}

// sendForumNotificationReady 送出連線建立時的未讀數。
//
// 失敗（DB 故障）只記日誌：前端本來就會自己問一次 /unread-count，少了這一則
// 只代表這次連線沒有校正點，下一個推播或下一次頁面載入都會補上。
func (s *Server) sendForumNotificationReady(conn *notificationStreamConn, email string) {
	ctx, cancel := context.WithTimeout(context.Background(), notificationStreamQueryTimeout)
	defer cancel()

	unread, err := s.forumUnreadNotificationCount(ctx, email)
	if err != nil {
		logger.WarnfContext(ctx, "[FORUM] 讀取未讀通知數失敗（略過串流的初始訊息）: %v", err)
		return
	}
	payload, err := json.Marshal(forumNotificationReady{Type: notificationStreamMessageReady, UnreadCount: unread})
	if err != nil {
		// 三個欄位的結構不可能序列化失敗，留一筆日誌是為了不在這裡靜默吞掉。
		logger.WarnfContext(ctx, "[FORUM] 序列化未讀通知數失敗: %v", err)
		return
	}
	conn.deliver(payload)
}

/* ==========================================================================
   通知 -> 推播
   ========================================================================== */

// notifyForumNotificationCreated 把剛寫入的通知推給收件者開著的頁面。
//
// 刻意非同步（自己開 goroutine）：推播是為了讓**別人**的畫面立刻更新，不該讓
// 留言者／追蹤者的回應多等 2~3 次往返（一次 SELECT、一次 COUNT，還可能一次
// media token）。pushForumNotification 內部用 context.Background + 逾時，
// 不沿用請求的 context —— handler 一回應，r.Context() 就會被取消。
//
// id <= 0 代表 createForumNotification 跳過了（自我通知），沒有東西可推。
func (s *Server) notifyForumNotificationCreated(recipient string, id int64) {
	if id <= 0 || s.notifyStream == nil {
		return
	}
	// 沒有連線就不必組內容：絕大多數的通知都是「對方不在線上」。
	if !s.notifyStream.connected(recipient) {
		return
	}
	go s.pushForumNotification(recipient, id)
}

// pushForumNotification 讀出剛建立的那一則通知、算出最新的未讀數，推給這個人
// 所有開著的頁面。
//
// 每一步失敗都只是「這一則沒有即時送達」（只記日誌）：它的持久化版本已經在
// 資料庫裡，下一次頁面載入或標記已讀都會看到正確的狀態。
func (s *Server) pushForumNotification(recipient string, id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), notificationStreamQueryTimeout)
	defer cancel()

	row, ok := s.loadForumNotificationRow(ctx, recipient, id)
	if !ok {
		return
	}
	// 頭像 token：與清單端點同一條政策（簽不出來就不顯示頭像，而不是不推播）。
	mediaToken := ""
	if row.Avatar != "" {
		token, err := s.createMediaToken(ctx)
		if err != nil {
			logger.WarnfContext(ctx, "[FORUM] 建立即時通知頭像 token 失敗 notification_id=%d: %v", id, err)
		} else {
			mediaToken = token
		}
	}
	// 未讀數與推播一起送：前端不必為了一個紅點再打一次 unread-count。
	unread, err := s.forumUnreadNotificationCount(ctx, recipient)
	if err != nil {
		logger.WarnfContext(ctx, "[FORUM] 讀取未讀通知數失敗（略過推播）notification_id=%d: %v", id, err)
		return
	}
	payload, err := json.Marshal(forumNotificationPush{
		Type:        notificationStreamMessagePushed,
		UnreadCount: unread,
		Item:        s.forumNotificationItemFromRow(row, mediaToken),
	})
	if err != nil {
		logger.WarnfContext(ctx, "[FORUM] 序列化即時通知失敗 notification_id=%d: %v", id, err)
		return
	}

	delivered, dropped := s.notifyStream.publish(recipient, payload)
	if dropped > 0 {
		logger.WarnfContext(ctx, "[FORUM] 即時通知丟棄了過慢的連線 notification_id=%d received=%d dropped=%d",
			id, delivered, dropped)
	}
}

// loadForumNotificationRow 讀出單一則通知（含觸發者的暱稱與頭像）。
//
// WHERE 同時帶 id 與 recipient_email：少了後者，任何一個 bug 都能把別人的通知
// 推給這個人 —— 而推播是一條持續送出私人資料的長連線，這一關不能只靠「id 是
// 自己剛剛寫入的」這個假設。
//
// 查不到（ErrNoRows）回 ok=false 而不是錯誤：通知在寫入與推播之間被刪掉、
// 或收件者根本不對，都不是需要告警的異常。
func (s *Server) loadForumNotificationRow(ctx context.Context, recipient string, id int64) (forumNotificationRow, bool) {
	var (
		row    forumNotificationRow
		readAt sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT n.kind, n.actor_email, COALESCE(p.nickname, ''), COALESCE(p.avatar_url, ''),
		       n.post_id, n.comment_id, n.read_at, n.created_at
		FROM forum_notifications n
		LEFT JOIN forum_profiles p ON p.author_email = n.actor_email
		WHERE n.id = ? AND n.recipient_email = ?`, id, recipient).
		Scan(&row.Kind, &row.ActorMail, &row.Nickname, &row.Avatar, &row.PostID, &row.CommentID, &readAt, &row.CreatedAt)
	if err == sql.ErrNoRows {
		return forumNotificationRow{}, false
	}
	if err != nil {
		logger.WarnfContext(ctx, "[FORUM] 讀取即時通知內容失敗 notification_id=%d: %v", id, err)
		return forumNotificationRow{}, false
	}
	row.ID = id
	row.Read = readAt.Valid
	return row, true
}
