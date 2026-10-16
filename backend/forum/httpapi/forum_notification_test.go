/*
站內通知（httpapi/forum_notification_test.go）。

通知這個功能有一半的邏輯「壞掉時不會出錯」，因此特別需要機械式的把關：

 1. 對外的 JSON 形狀
    forumNotificationItem 的 PostID / CommentID / ActorAvatar 都掛了
    omitempty，而前端據此判斷「這一則有沒有可跳的目的地」。少了 omitempty
    的症狀不是錯誤，是每一則 follow 通知都帶著 "postId":0，前端於是把人
    導去 /forum/post/0。這裡把「哪些欄位在哪些 kind 下會出現」逐一釘死。

 2. best-effort 的跳過規則
    createForumNotification 在 recipient / actor 為空或相等時必須直接
    return nil。這條規則若被移除，症狀是 DB 端才炸開（空字串寫進 NOT NULL
    欄位）或使用者收到「自己追蹤自己」這種只可能來自 bug 的通知。測試用的
    Server 刻意沒有 db：跳過路徑若真的去碰資料庫，這裡會以 panic 收場，
    而不是靜默通過。

 3. 授權與來源檢查
    三個端點都掛 requireLogin，而標記已讀另外要過 isTrustedOrigin。這一類
    檢查在路由層與 handler 內各有一份，remove 掉 router 那一份不會有任何
    測試紅燈 —— 因此這裡直接呼叫 handler，釘死「沒有 session 一律 401」
    「來源不可信一律 403」。

刻意沒有測的：清單查詢與標記已讀的實際 SQL。它們需要真的 MySQL（這個專案
沒有替身，見 docs/KNOWN_ISSUES.md 的已知問題），而授權條件（recipient_email
放進 WHERE）在論述上由 listForumNotifications / markForumNotificationsRead
的註解守住。這裡測的是「不回資料庫」的那幾條路徑。
*/
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// decodeJSONObject 把一段 JSON 物件拆成「欄位名 -> 原始值」，讓測試可以分別
// 斷言「某欄位在不在」（omitempty）與「某欄位的值」。用 json.RawMessage 而
// 不是 interface{}：前者保留原始形態，數字不會被轉成 float64 而失真。
func decodeJSONObject(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("解析 JSON 失敗: %v（raw=%s）", err, raw)
	}
	return fields
}

// TestForumNotificationItemJSONShape 釘死對外通知的欄位增減規則。
//
// 兩種 kind 各測一次，因為它們的差別正好落在 omitempty 上：
// follow 沒有目標（postId / commentId 皆為 0）且沒有頭像，三個欄位都該消失；
// comment 有目標與頭像，三個欄位都該出現。read 兩種都要出現 —— 它是前端
// 決定那一列要不要變灰的唯一依據，不能因為 false 就被 omitempty 吃掉。
func TestForumNotificationItemJSONShape(t *testing.T) {
	createdAt := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	t.Run("follow 省略目標與頭像", func(t *testing.T) {
		raw, err := json.Marshal(forumNotificationItem{
			ID:        4,
			Kind:      forumNotificationFollow,
			Actor:     "小明",
			ActorKey:  "abc",
			Read:      true,
			CreatedAt: createdAt,
		})
		if err != nil {
			t.Fatalf("序列化失敗: %v", err)
		}
		fields := decodeJSONObject(t, raw)

		for _, key := range []string{"postId", "commentId", "actorAvatar"} {
			if _, ok := fields[key]; ok {
				t.Errorf("follow 通知不該帶 %q（值為零值時應被 omitempty 省略），raw=%s", key, raw)
			}
		}
		// read 沒有 omitempty：false 也必須出現，否則前端會把未讀誤判成已讀。
		readRaw, ok := fields["read"]
		if !ok {
			t.Fatalf("回應缺少 read 欄位，raw=%s", raw)
		}
		var read bool
		if err := json.Unmarshal(readRaw, &read); err != nil {
			t.Fatalf("解析 read 失敗: %v", err)
		}
		if !read {
			t.Errorf("read = false, want true（follow 通知序列化後仍須保留真正的值）")
		}
	})

	t.Run("comment 帶目標與頭像", func(t *testing.T) {
		raw, err := json.Marshal(forumNotificationItem{
			ID:          9,
			Kind:        forumNotificationComment,
			Actor:       "小華",
			ActorKey:    "def",
			ActorAvatar: "https://cdn.example/avatar.png",
			PostID:      12,
			CommentID:   34,
			// read 刻意留 false，證明它不會被 omitempty 吃掉。
			CreatedAt: createdAt,
		})
		if err != nil {
			t.Fatalf("序列化失敗: %v", err)
		}
		fields := decodeJSONObject(t, raw)

		for _, key := range []string{"postId", "commentId", "actorAvatar", "read"} {
			if _, ok := fields[key]; !ok {
				t.Errorf("comment 通知應帶 %q，raw=%s", key, raw)
			}
		}
		var read bool
		if err := json.Unmarshal(fields["read"], &read); err != nil {
			t.Fatalf("解析 read 失敗: %v", err)
		}
		if read {
			t.Errorf("read = true, want false（未讀的通知不得被序列化成已讀）")
		}
	})
}

// TestCreateForumNotificationSkipsInvalidRecipient 釘死 best-effort 的跳過規則。
//
// 這個測試的 Server 沒有 db：任何一個案例若沒有被擋在最前面，就會在
// ExecContext 上 panic —— 因此「回 nil 且不 panic」正是「沒有碰資料庫」的證明。
func TestCreateForumNotificationSkipsInvalidRecipient(t *testing.T) {
	server := newPostEditTestServer(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		recipient string
		actor     string
	}{
		{name: "沒有收件者", recipient: "", actor: "a@example.com"},
		{name: "收件者只有空白", recipient: "   ", actor: "a@example.com"},
		{name: "沒有觸發者", recipient: "a@example.com", actor: ""},
		{name: "觸發者只有空白", recipient: "a@example.com", actor: "  "},
		{name: "收件者等於觸發者", recipient: "a@example.com", actor: "a@example.com"},
		{name: "收件者等於觸發者（含空白）", recipient: " a@example.com ", actor: "a@example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := server.createForumNotification(ctx, tc.recipient, tc.actor, forumNotificationComment, 1, 2)
			if err != nil {
				t.Errorf("createForumNotification 回 %v, want nil（應在碰資料庫前跳過）", err)
			}
			if id != 0 {
				t.Errorf("createForumNotification 回 id=%d, want 0（跳過的路徑沒有通知可推）", id)
			}
		})
	}
}

// TestForumNotificationsRequiresLogin 守住「四個端點都不得對未登入者回應」。
//
// 這幾條路徑會在 ResolveUser 取得空字串後立刻回 401，因此不需要 MySQL；
// 若身分檢查被移除，請求會往下走到查詢並 panic（server.db 為 nil）。
func TestForumNotificationsRequiresLogin(t *testing.T) {
	server := newPostEditTestServer(t)

	cases := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
	}{
		{"列表", server.handleForumNotifications, http.MethodGet, "/api/forum/notifications"},
		{"標記已讀", server.handleForumNotifications, http.MethodPost, "/api/forum/notifications"},
		{"未讀數", server.handleForumNotificationUnreadCount, http.MethodGet, "/api/forum/notifications/unread-count"},
		{"即時推播", server.handleForumNotificationStream, http.MethodGet, "/api/forum/notifications/ws"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.handler(recorder, httptest.NewRequest(tc.method, tc.path, nil))

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("回應 = %d, want 401（沒有 session 時不得回應任何通知）", recorder.Code)
			}
		})
	}
}

// TestForumNotificationMarkReadChecksOrigin 守住標記已讀的來源檢查。
//
// 症狀是「CSRF 防護失效但沒有任何錯誤」：攻擊者頁面送出的表單會帶著合法
// cookie 把受害者的通知標成已讀。因此帶一個不在白名單的 Origin 時必須 403，
// 且不得碰資料庫（server.db 為 nil）。
func TestForumNotificationMarkReadChecksOrigin(t *testing.T) {
	server := newPostEditTestServer(t)

	request := loggedInRequest(t, server, http.MethodPost, "/api/forum/notifications", `{"id":1}`)
	request.Header.Set("Origin", "https://evil.example")

	recorder := httptest.NewRecorder()
	server.handleForumNotifications(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("回應 = %d, want 403（不可信來源不得標記已讀）", recorder.Code)
	}
}

// TestForumNotificationsRejectsBadInput 守住「在碰資料庫前就擋下」的輸入錯誤。
//
// 兩條路徑都必須在查詢之前結束：limit 不合法（400）與方法不支援（405）。
// 這裡刻意用已登入的請求，證明擋下它們的不是身分檢查而是輸入檢查本身。
func TestForumNotificationsRejectsBadInput(t *testing.T) {
	server := newPostEditTestServer(t)

	t.Run("limit 不合法", func(t *testing.T) {
		for _, limit := range []string{"0", "-1", "abc", "999"} {
			request := loggedInRequest(t, server, http.MethodGet, "/api/forum/notifications?limit="+limit, "")
			recorder := httptest.NewRecorder()
			server.handleForumNotifications(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("limit=%q 回應 = %d, want 400", limit, recorder.Code)
			}
		}
	})

	t.Run("方法不支援", func(t *testing.T) {
		request := loggedInRequest(t, server, http.MethodDelete, "/api/forum/notifications", "")
		recorder := httptest.NewRecorder()
		server.handleForumNotifications(recorder, request)

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("DELETE 回應 = %d, want 405", recorder.Code)
		}
	})
}
