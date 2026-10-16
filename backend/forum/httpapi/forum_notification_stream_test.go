/*
即時通知推播的測試（httpapi/forum_notification_stream_test.go）。

這個功能有三塊「壞掉時不會出錯」的行為，因此特別需要機械式的把關：

 1. 連線的集合（hub）
    上限、淘汰與慢速消費者都在記憶體裡運作，而它們的症狀是資源無聲地長大或
    推播無聲地少送。這裡用假的 sink 把規則釘死 —— 不需要真的開一條 TCP，
    更不需要真的 WebSocket；hub 對連線的全部需求就是那個介面。

 2. 升級前的檢查
    401／403／405 這三條路徑都必須在碰資料庫**之前**結束（測試用的 Server
    沒有 db，碰到就會 panic），因此它們同時是「授權邊界」與「不會白花一次
    升級」的證明。

 3. 真的推得出去
    這一塊需要真的 MySQL（未讀數與通知內容都是查詢，這個專案沒有 DB 替身），
    因此放在檔尾以 FORUM_TEST_MYSQL_DSN 為條件的測試裡（-run MySQL），
    由 CI 的 migrations job 提供服務。它驗的是整條路徑：升級、ready、
    建立通知、推播、標記已讀、重新連線的 ready —— 也就是「一個開著頁面的
    使用者真的會在幾毫秒內看到紅點」這件事本身。

刻意沒有測的：gorilla/websocket 的心跳與逾時（那是套件自己的行為，這裡只
設定參數），以及多實例之間的廣播（本專案不做，見 KNOWN_ISSUES.md）。
*/
package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	_ "github.com/go-sql-driver/mysql"

	"forum/forum/session"
)

/* ==========================================================================
   連線的集合（hub）
   ========================================================================== */

// fakeStreamSink 是測試用的連線替身：只記錄 hub 對它做了什麼。
//
// full 代表「這條連線的佇列永遠是滿的」，也就是慢速消費者。
// closeCall 是一次收線的 code 與原因。兩者都是契約的一部分：code 決定前端
// 要不要重連，原因則只有診斷價值。
type closeCall struct {
	code   int
	reason string
}

type fakeStreamSink struct {
	mu       sync.Mutex
	received [][]byte
	closed   []closeCall
	full     bool
}

func (f *fakeStreamSink) deliver(payload []byte) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.full {
		return false
	}
	f.received = append(f.received, payload)
	return true
}

func (f *fakeStreamSink) closeWithReason(code int, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, closeCall{code: code, reason: reason})
}

func (f *fakeStreamSink) delivered() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.received...)
}

func (f *fakeStreamSink) closeReasons() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	reasons := make([]string, 0, len(f.closed))
	for _, call := range f.closed {
		reasons = append(reasons, call.reason)
	}
	return reasons
}

// TestNotificationStreamConnQueueSemantics 釘死一條連線對「送」與「關」的定義。
//
// 這兩個行為是 hub 那三條規則的前提：佇列滿了才叫慢速消費者，而已經在收線的
// 連線不算慢速（它只是晚了一步）。用 nil 的 *websocket.Conn 建構是刻意的 ——
// deliver 與 closeWithReason 不碰 socket（只有 writeLoop/readLoop 會），
// 因此這兩個方法可以在沒有任何網路的情況下驗證。
func TestNotificationStreamConnQueueSemantics(t *testing.T) {
	conn := newNotificationStreamConn(nil)

	// 佇列容量見 notificationStreamSendBuffer；先放滿。
	for i := 0; i < notificationStreamSendBuffer; i++ {
		if !conn.deliver([]byte("payload")) {
			t.Fatalf("第 %d 則就放不進佇列", i+1)
		}
	}
	if conn.deliver([]byte("overflow")) {
		t.Fatal("佇列滿了必須回 false（那是慢速消費者的定義）")
	}
	// 對方讀走一則之後就又有空間。
	<-conn.send
	if !conn.deliver([]byte("payload")) {
		t.Error("佇列有空間時必須收得下")
	}

	conn.closeWithReason(notificationStreamCloseSlowConsumer, "bye")
	if !conn.deliver([]byte("payload")) {
		t.Error("已經在收線的連線不算慢速消費者（訊息只是沒有送達）")
	}
	conn.closeWithReason(websocket.CloseNormalClosure, "again")
	if conn.closeReason != "bye" || conn.closeCode != notificationStreamCloseSlowConsumer {
		t.Errorf("第二次收線改寫了第一次的 code/reason: (%d, %q), want (%d, %q)",
			conn.closeCode, conn.closeReason, notificationStreamCloseSlowConsumer, "bye")
	}
}

// TestNotificationStreamHubPublishesToEveryConnection 釘死「推給這個人的每一條
// 連線，而且只推給這個人的連線」。
func TestNotificationStreamHubPublishesToEveryConnection(t *testing.T) {
	hub := newNotificationStreamHub()
	first := &fakeStreamSink{}
	second := &fakeStreamSink{}
	other := &fakeStreamSink{}

	for _, sink := range []notificationStreamSink{first, second} {
		if replaced, ok := hub.attach("a@example.com", sink); !ok || len(replaced) != 0 {
			t.Fatalf("attach 回 replaced=%d ok=%v, want 0/true", len(replaced), ok)
		}
	}
	if _, ok := hub.attach("b@example.com", other); !ok {
		t.Fatal("attach 不該拒絕第二個人的連線")
	}

	delivered, dropped := hub.publish("a@example.com", []byte("payload"))
	if delivered != 2 || dropped != 0 {
		t.Fatalf("publish 回 delivered=%d dropped=%d, want 2/0", delivered, dropped)
	}
	if len(first.delivered()) != 1 || len(second.delivered()) != 1 {
		t.Errorf("同一個人的每一條連線都要收到：first=%d second=%d",
			len(first.delivered()), len(second.delivered()))
	}
	if len(other.delivered()) != 0 {
		t.Error("別人的連線不得收到這個人的通知")
	}

	// 沒有連線的人：不是錯誤，只是沒有人可以送。
	if delivered, dropped := hub.publish("nobody@example.com", []byte("payload")); delivered != 0 || dropped != 0 {
		t.Errorf("對沒有連線的人 publish 回 %d/%d, want 0/0", delivered, dropped)
	}
	if hub.connected("nobody@example.com") {
		t.Error("connected 對沒有連線的人必須回 false")
	}
	if !hub.connected("a@example.com") {
		t.Error("connected 對有連線的人必須回 true")
	}
}

// TestNotificationStreamHubEvictsOldestPerUser 釘死「同一使用者的上限用淘汰
// 最舊，而不是拒絕新的」。
func TestNotificationStreamHubEvictsOldestPerUser(t *testing.T) {
	hub := newNotificationStreamHub()
	const email = "a@example.com"

	sinks := make([]*fakeStreamSink, 0, notificationStreamMaxPerUser)
	for i := 0; i < notificationStreamMaxPerUser; i++ {
		sink := &fakeStreamSink{}
		if replaced, ok := hub.attach(email, sink); !ok || len(replaced) != 0 {
			t.Fatalf("第 %d 條連線就被淘汰或拒絕（replaced=%d ok=%v）", i+1, len(replaced), ok)
		}
		sinks = append(sinks, sink)
	}

	newest := &fakeStreamSink{}
	replaced, ok := hub.attach(email, newest)
	if !ok {
		t.Fatal("超過同一使用者的上限時必須淘汰最舊的，而不是拒絕新的")
	}
	if len(replaced) != 1 || notificationStreamSink(sinks[0]) != replaced[0] {
		t.Fatalf("被淘汰的不是最舊的那一條：replaced=%d", len(replaced))
	}
	// 淘汰是「關掉」，呼叫端負責執行；hub 只是把它交出來。
	if len(sinks[0].closeReasons()) != 0 {
		t.Error("hub 不該自己關閉連線（那是呼叫端的工作，避免在鎖內呼叫 socket）")
	}

	hub.publish(email, []byte("payload"))
	if len(sinks[0].delivered()) != 0 {
		t.Error("被淘汰的連線不該再收到推播")
	}
	if len(newest.delivered()) != 1 {
		t.Error("最新那一條連線必須收得到")
	}
	// 被淘汰的那一條離開時也會呼叫 detach，而它已經不在表上：那必須是 no-op，
	// 否則 total 會被扣兩次（總量的上限因此永久少一格）。
	hub.detach(email, sinks[0])
	if len(newest.delivered()) != 1 {
		t.Error("detach 不該影響其他連線")
	}
}

// TestNotificationStreamHubRejectsWhenGlobalLimitReached 釘死「全域上限滿了是
// 拒絕新連線，而不是淘汰別人」。
func TestNotificationStreamHubRejectsWhenGlobalLimitReached(t *testing.T) {
	hub := newNotificationStreamHub()
	for i := 0; i < notificationStreamMaxTotal; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		if _, ok := hub.attach(email, &fakeStreamSink{}); !ok {
			t.Fatalf("第 %d 條連線就被拒絕（全域上限是 %d）", i+1, notificationStreamMaxTotal)
		}
	}
	replaced, ok := hub.attach("late@example.com", &fakeStreamSink{})
	if ok {
		t.Fatal("全域額度已滿時必須拒絕新連線")
	}
	if len(replaced) != 0 {
		t.Fatal("全域額度已滿時不得淘汰別人的連線（一個人開分頁就能踢掉所有人）")
	}
}

// TestNotificationStreamHubDropsSlowConsumer 釘死慢速消費者的處置：關掉那條
// 連線，而不是阻塞或其他連線一起陪葬。
func TestNotificationStreamHubDropsSlowConsumer(t *testing.T) {
	hub := newNotificationStreamHub()
	const email = "a@example.com"
	slow := &fakeStreamSink{full: true}
	quick := &fakeStreamSink{}
	hub.attach(email, slow)
	hub.attach(email, quick)

	delivered, dropped := hub.publish(email, []byte("payload"))
	if delivered != 1 || dropped != 1 {
		t.Fatalf("publish 回 delivered=%d dropped=%d, want 1/1", delivered, dropped)
	}
	if reasons := slow.closeReasons(); len(reasons) != 1 || reasons[0] != "client is too slow" {
		t.Errorf("慢速連線的關閉原因 = %v, want [client is too slow]", reasons)
	}
	if len(quick.closeReasons()) != 0 {
		t.Error("正常連線不得被關掉")
	}
	if len(quick.delivered()) != 1 {
		t.Error("正常連線必須收到推播（慢速的那一條不該拖累它）")
	}
}

// TestNotificationStreamHubNilIsNoOp 釘死「沒有 hub 的 Server 仍然可用」。
//
// 測試以 struct literal 構造 Server 時不會有 notifyStream，而推播是附加效果：
// nil 必須是 no-op，不能是 panic，也不能讓升級失敗。
func TestNotificationStreamHubNilIsNoOp(t *testing.T) {
	var hub *notificationStreamHub
	replaced, ok := hub.attach("a@example.com", &fakeStreamSink{})
	if !ok || len(replaced) != 0 {
		t.Fatalf("nil hub 的 attach 回 replaced=%d ok=%v, want 0/true", len(replaced), ok)
	}
	if hub.connected("a@example.com") {
		t.Error("nil hub 沒有連線")
	}
	if delivered, dropped := hub.publish("a@example.com", []byte("payload")); delivered != 0 || dropped != 0 {
		t.Errorf("nil hub 的 publish 回 %d/%d, want 0/0", delivered, dropped)
	}
	hub.detach("a@example.com", &fakeStreamSink{}) // 不得 panic
}

/* ==========================================================================
   訊息形狀
   ========================================================================== */

// TestNotificationStreamMessageShape 釘死兩種訊息的 JSON 形狀。
//
// 前端用 type 分派、用 unreadCount 更新紅點、用 item 插入列表。少了
// unreadCount（例如給它 omitempty）的症狀是「有通知但紅點不會動」，
// 那是這裡最貴的一個 silent failure。
func TestNotificationStreamMessageShape(t *testing.T) {
	ready, err := json.Marshal(forumNotificationReady{Type: notificationStreamMessageReady, UnreadCount: 0})
	if err != nil {
		t.Fatalf("序列化 ready 失敗: %v", err)
	}
	fields := decodeJSONObject(t, ready)
	if got := string(fields["type"]); got != `"ready"` {
		t.Errorf("ready 的 type = %s, want \"ready\"", got)
	}
	if _, ok := fields["unreadCount"]; !ok {
		t.Error("ready 必須帶 unreadCount（0 也要出現，前端才分得出「沒有未讀」與「沒有這個欄位」）")
	}

	pushed, err := json.Marshal(forumNotificationPush{
		Type:        notificationStreamMessagePushed,
		UnreadCount: 2,
		Item:        forumNotificationItem{ID: 7, Kind: forumNotificationFollow, Actor: "小明", ActorKey: "abc", CreatedAt: time.Now()},
	})
	if err != nil {
		t.Fatalf("序列化推播失敗: %v", err)
	}
	fields = decodeJSONObject(t, pushed)
	if got := string(fields["type"]); got != `"notification"` {
		t.Errorf("推播的 type = %s, want \"notification\"", got)
	}
	if _, ok := fields["item"]; !ok {
		t.Error("推播必須帶 item（前端直接把它插進列表）")
	}
	// follow 通知不帶 postId/commentId，與清單端點同一份組裝程式碼。
	var itemFields map[string]json.RawMessage
	if err := json.Unmarshal(fields["item"], &itemFields); err != nil {
		t.Fatalf("解析 item 失敗: %v", err)
	}
	if _, ok := itemFields["postId"]; ok {
		t.Error("follow 推播不得帶 postId")
	}
}

/* ==========================================================================
   升級前的檢查
   ========================================================================== */

// TestForumNotificationStreamRejectsBadRequest 守住「不該升級的三種情況」。
//
// 三個子測試的 Server 都沒有 db：任何一條路徑若在回應之前碰到資料庫就會
// panic（見本檔檔頭）。最後一條同時證明「來源檢查通過之後就輪到升級」——
// httptest.ResponseRecorder 不支援 Hijack，因此 gorilla 會以 500 收場。
func TestForumNotificationStreamRejectsBadRequest(t *testing.T) {
	server := newPostEditTestServer(t, "https://forum.example")

	t.Run("方法不支援", func(t *testing.T) {
		request := loggedInRequest(t, server, http.MethodPost, "/api/forum/notifications/ws", "")
		recorder := httptest.NewRecorder()
		server.handleForumNotificationStream(recorder, request)

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST 回應 = %d, want 405", recorder.Code)
		}
	})

	t.Run("不可信來源", func(t *testing.T) {
		request := loggedInRequest(t, server, http.MethodGet, "/api/forum/notifications/ws", "")
		request.Header.Set("Origin", "https://evil.example")
		recorder := httptest.NewRecorder()
		server.handleForumNotificationStream(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Errorf("不可信來源回應 = %d, want 403（跨站不得開啟一條持續讀取私人通知的連線）", recorder.Code)
		}
	})

	t.Run("來源可信時才會嘗試升級", func(t *testing.T) {
		request := loggedInRequest(t, server, http.MethodGet, "/api/forum/notifications/ws", "")
		request.Header.Set("Origin", "https://forum.example")
		// 帶上真實的升級標頭：這樣才會走到「取得底層連線」那一步，而不是被
		// gorilla 的協定檢查（缺 Upgrade 標頭 → 400）提前擋下。
		request.Header.Set("Connection", "Upgrade")
		request.Header.Set("Upgrade", "websocket")
		request.Header.Set("Sec-WebSocket-Version", "13")
		request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

		recorder := httptest.NewRecorder()
		server.handleForumNotificationStream(recorder, request)

		// 500 來自 gorilla（ResponseRecorder 不能被 Hijack）。這裡在意的是
		// 「不是 401/403/405」——那代表請求已經走到升級，前面的每一關都放行了。
		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("回應 = %d, want 500（升級失敗由 gorilla 回報）", recorder.Code)
		}
	})
}

// TestNotifyForumNotificationCreatedWithoutStream 釘死「沒有推播中心時不做事」。
//
// 這一條同時是「struct literal 的 Server 仍然可以呼叫這個函式」的證明：
// 它在 notifyStream 為 nil、或收件者沒有連線時都必須直接回傳，不啟動任何
// goroutine，也不碰資料庫（server.db 為 nil）。
func TestNotifyForumNotificationCreatedWithoutStream(t *testing.T) {
	server := newPostEditTestServer(t)

	server.notifyForumNotificationCreated("a@example.com", 1)

	serverWithHub := newPostEditTestServer(t)
	serverWithHub.notifyStream = newNotificationStreamHub()
	// 沒有連線的人：連查詢都不該發生。
	serverWithHub.notifyForumNotificationCreated("a@example.com", 1)
	// 沒有 id（createForumNotification 跳過的自我通知）：沒有東西可推。
	serverWithHub.notifyForumNotificationCreated("a@example.com", 0)
}

/* ==========================================================================
   真的伺服器上的串流
   ========================================================================== */

// TestForumNotificationStreamAgainstRealServer 在一個真的 HTTP 伺服器上驗三件事：
// 升級穿得過整個中介層堆疊、推播走得到連線、超過上限時最舊的連線被關掉而且帶著原因。
//
// 【為什麼是這一組】升級是這個功能最不容易從單元測試看出來的一件事：
// SecurityHeaders、Refresh、LoggingMiddleware 與 metrics 依序包在 mux 外層，其中
// 任何一層若換成一個「不轉發 Hijack」的 ResponseWriter，WebSocket 就永遠升級不了 ——
// 而症狀只是「即時通知不會動」（前端會靜默退回每次載入問一次未讀數），不會有任何錯誤。
// 推播與 close frame 同理：hub 的單元測試只驗「有沒有交給連線」，交出去之後能不能
// 真的變成對方讀得到的位元組，只有一條真的連線能回答。
//
// 【資料庫連不上是刻意的】位址指向一個拒絕連線的埠，而不是 nil：
//   - requireLogin 對管理員跳過停權查詢，因此請求會走到升級；
//   - 升級之後的查詢（ready 的未讀數）一律失敗並被記錄，但連線仍然開著 ——
//     而「連線開著、路由登記完成」正是這三個子測試需要的狀態。
//
// 資料庫真的有資料時的行為由 TestForumNotificationStreamMySQL 驗。
func TestForumNotificationStreamAgainstRealServer(t *testing.T) {
	server := newPostEditTestServer(t)
	server.db = unreachableTestDB(t)
	server.notifyStream = newNotificationStreamHub()

	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	// 管理員身分讓 requireLogin 不必查資料庫（見該函式的說明）。
	const admin = "admin@example.com"
	token, err := server.sessions.Create(admin, true)
	if err != nil {
		t.Fatalf("建立 session 失敗: %v", err)
	}
	header := http.Header{}
	header.Set("Cookie", "FORUM_test="+token)
	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/forum/notifications/ws"

	dial := func(t *testing.T) *websocket.Conn {
		t.Helper()
		conn, response, err := websocket.DefaultDialer.Dial(url, header)
		if err != nil {
			t.Fatalf("升級失敗（某一層中介層是否吃掉了 Hijack？）: %v", err)
		}
		if response.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("升級狀態 = %d, want 101", response.StatusCode)
		}
		// 連線由子測試的 t 收拾：額度在子測試之間因此是乾淨的。
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}

	t.Run("升級並登記進 hub", func(t *testing.T) {
		dial(t)
		if !server.notifyStream.connected(admin) {
			t.Error("連線沒有被登記進 hub：handler 沒有走完升級後的登記（或登記在別的 key 上）")
		}
	})

	t.Run("推播走得到這條連線", func(t *testing.T) {
		conn := dial(t)

		payload := []byte(`{"type":"notification","unreadCount":1,"item":{"id":1,"kind":"follow"}}`)
		delivered, dropped := server.notifyStream.publish(admin, payload)
		if delivered != 1 || dropped != 0 {
			t.Fatalf("publish 回 delivered=%d dropped=%d, want 1/0", delivered, dropped)
		}

		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatalf("設定讀取逾時失敗: %v", err)
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("讀取推播失敗: %v", err)
		}
		if string(raw) != string(payload) {
			t.Errorf("推播內容 = %s, want %s（訊息應逐位元組轉送）", raw, payload)
		}
	})

	t.Run("超過上限時最舊的連線被關掉並帶著原因", func(t *testing.T) {
		connections := make([]*websocket.Conn, 0, notificationStreamMaxPerUser)
		for i := 0; i < notificationStreamMaxPerUser; i++ {
			connections = append(connections, dial(t))
		}
		oldest := connections[0]
		if err := oldest.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatalf("設定讀取逾時失敗: %v", err)
		}

		// 第 9 條：超過同一使用者的上限，伺服器淘汰最舊的那一條。
		dial(t)

		_, _, err := oldest.ReadMessage()
		if err == nil {
			t.Fatal("最舊的連線應該被關掉")
		}
		var closeErr *websocket.CloseError
		if !errors.As(err, &closeErr) {
			t.Fatalf("錯誤型別 = %T, want *websocket.CloseError（%v）", err, err)
		}
		// code 是契約的一部分：前端靠「4000（被取代）」決定不要重連，
		// 否則兩條連線會互相汰換對方（見 forum_notification_stream.go 的常數）。
		if closeErr.Code != notificationStreamCloseReplaced {
			t.Errorf("close code = %d, want %d", closeErr.Code, notificationStreamCloseReplaced)
		}
		// 原因則會走進瀏覽器的 close 事件（前端把它記在 console 裡）。
		if closeErr.Text != "replaced by a newer connection" {
			t.Errorf("close 原因 = %q, want %q", closeErr.Text, "replaced by a newer connection")
		}
	})
}

// unreachableTestDB 回一個「連得上但是連不到」的 *sql.DB：位址指向一個拒絕連線
// 的埠。用它而不是 nil：nil 會在查詢時 panic（整個測試行程直接死掉），而連不上
// 的資料庫只會讓查詢回錯誤 —— 那正是上面那個測試要的「升級成功、查詢失敗」。
func unreachableTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", "root:irrelevant@tcp(127.0.0.1:1)/?timeout=1s")
	if err != nil {
		t.Fatalf("開啟測試連線失敗: %v", err)
	}
	// 一次就好：連不上時不必讓每個查詢各開一條註定失敗的連線。
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

/* ==========================================================================
   真的推得出去（需要真的 MySQL）
   ========================================================================== */

// streamTestMessage 是測試端看到的推播訊息。兩種訊息共用一個寬鬆的結構：
// 解析 forumNotificationItem 也順便證明推播的 item 欄位與清單端點完全同名。
type streamTestMessage struct {
	Type        string                `json:"type"`
	UnreadCount int                   `json:"unreadCount"`
	Item        forumNotificationItem `json:"item"`
}

// TestForumNotificationStreamMySQL 驗整條路徑：升級、ready、推播、已讀之後的
// ready。需要 FORUM_TEST_MYSQL_DSN（見本檔檔頭），一般 go test ./... 會跳過。
func TestForumNotificationStreamMySQL(t *testing.T) {
	// subtreeTempDatabase 是 comment_reply_mysql_test.go 的輔助函式，但它做的事
	// 是通用的：建立拋棄式資料庫 + 跑完整 MigrateMySQL。名字裡的 subtree 來自
	// 第一個使用它的測試，這裡沿用而不是複製一份（複製的那一份會漂移）。
	db := subtreeTempDatabase(t)

	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	sessions := session.NewManager("FORUM_test", time.Hour, false, client)
	server := &Server{db: db, sessions: sessions, notifyStream: newNotificationStreamHub()}

	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	const recipient = "stream-recipient@example.com"
	const actor = "stream-actor@example.com"
	token, err := sessions.Create(recipient, false)
	if err != nil {
		t.Fatalf("建立 session 失敗: %v", err)
	}

	dial := func(t *testing.T) *websocket.Conn {
		t.Helper()
		url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/forum/notifications/ws"
		header := http.Header{}
		header.Set("Cookie", "FORUM_test="+token)
		conn, _, err := websocket.DefaultDialer.Dial(url, header)
		if err != nil {
			t.Fatalf("WebSocket 連線失敗: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}
	expectReady := func(t *testing.T, conn *websocket.Conn, want int) {
		t.Helper()
		msg := readStreamMessage(t, conn)
		if msg.Type != notificationStreamMessageReady {
			t.Fatalf("第一則訊息 type = %q, want %q", msg.Type, notificationStreamMessageReady)
		}
		if msg.UnreadCount != want {
			t.Fatalf("ready 的未讀數 = %d, want %d", msg.UnreadCount, want)
		}
	}

	t.Run("連線後先收到未讀數", func(t *testing.T) {
		conn := dial(t)
		expectReady(t, conn, 0)
	})

	t.Run("建立通知會即時推播", func(t *testing.T) {
		conn := dial(t)
		expectReady(t, conn, 0)

		id, err := server.createForumNotification(context.Background(), recipient, actor, forumNotificationFollow, 0, 0)
		if err != nil {
			t.Fatalf("createForumNotification 失敗: %v", err)
		}
		if id <= 0 {
			t.Fatalf("createForumNotification 回 id=%d, want > 0", id)
		}
		// 正式路徑是 notifyForumCommentCreated / 追蹤切換在寫入成功後呼叫它，
		// 而它自己開 goroutine —— 這裡直接呼叫同一個函式。
		server.notifyForumNotificationCreated(recipient, id)

		msg := readStreamMessage(t, conn)
		if msg.Type != notificationStreamMessagePushed {
			t.Fatalf("推播 type = %q, want %q", msg.Type, notificationStreamMessagePushed)
		}
		if msg.UnreadCount != 1 {
			t.Errorf("推播的未讀數 = %d, want 1", msg.UnreadCount)
		}
		if msg.Item.ID != id || msg.Item.Kind != forumNotificationFollow {
			t.Errorf("推播的 item = id:%d kind:%q, want id:%d kind:%q",
				msg.Item.ID, msg.Item.Kind, id, forumNotificationFollow)
		}
		if msg.Item.Actor == "" || msg.Item.ActorKey == "" {
			t.Errorf("推播的觸發者不完整：actor=%q actorKey=%q", msg.Item.Actor, msg.Item.ActorKey)
		}
		if msg.Item.PostID != 0 || msg.Item.CommentID != 0 {
			t.Errorf("follow 通知不該帶 postId/commentId（%d/%d）", msg.Item.PostID, msg.Item.CommentID)
		}
		if msg.Item.Read {
			t.Error("剛建立的通知不得是已讀")
		}
	})

	t.Run("已讀之後重新連線的未讀數歸零", func(t *testing.T) {
		markAllNotificationsRead(t, httpServer.URL, token)

		conn := dial(t)
		expectReady(t, conn, 0)
	})

	t.Run("帶目標的推播保留 postId/commentId", func(t *testing.T) {
		conn := dial(t)
		expectReady(t, conn, 0)

		id, err := server.createForumNotification(context.Background(), recipient, actor, forumNotificationReply, 7, 9)
		if err != nil {
			t.Fatalf("createForumNotification 失敗: %v", err)
		}
		server.notifyForumNotificationCreated(recipient, id)

		msg := readStreamMessage(t, conn)
		if msg.Item.PostID != 7 || msg.Item.CommentID != 9 {
			t.Errorf("推播的目標 = %d/%d, want 7/9", msg.Item.PostID, msg.Item.CommentID)
		}
		if msg.Item.Kind != forumNotificationReply {
			t.Errorf("推播 kind = %q, want %q", msg.Item.Kind, forumNotificationReply)
		}
	})

	t.Run("讀不到別人的通知", func(t *testing.T) {
		id, err := server.createForumNotification(context.Background(), recipient, actor, forumNotificationFollow, 0, 0)
		if err != nil {
			t.Fatalf("createForumNotification 失敗: %v", err)
		}
		// 推播路徑的 WHERE 同時帶 id 與 recipient_email：少了後者，任何一個
		// bug 都能把別人的通知推給這個人。
		if _, ok := server.loadForumNotificationRow(context.Background(), "someone-else@example.com", id); ok {
			t.Fatal("loadForumNotificationRow 不得讀到別人的通知")
		}
		if _, ok := server.loadForumNotificationRow(context.Background(), recipient, id); !ok {
			t.Fatal("loadForumNotificationRow 讀不到自己的通知")
		}
	})
}

// readStreamMessage 讀一則推播訊息，逾時即失敗（連線壞掉或根本沒有推播時，
// 測試會停在那裡而不是無聲地通過）。
func readStreamMessage(t *testing.T, conn *websocket.Conn) streamTestMessage {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("設定讀取逾時失敗: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("讀取推播訊息失敗（逾時或連線中斷）: %v", err)
	}
	var msg streamTestMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("解析推播訊息失敗 raw=%s: %v", raw, err)
	}
	return msg
}

// markAllNotificationsRead 走真的 HTTP 端點把全部通知標成已讀。
func markAllNotificationsRead(t *testing.T, baseURL, token string) {
	t.Helper()
	body, err := json.Marshal(forumNotificationMarkRequest{})
	if err != nil {
		t.Fatalf("序列化請求失敗: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/forum/notifications", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("建立請求失敗: %v", err)
	}
	request.AddCookie(&http.Cookie{Name: "FORUM_test", Value: token})

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("標記已讀失敗: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("標記已讀回應 = %d, want 200", response.StatusCode)
	}
}
