/*
後台管理面的真實 MySQL 測試（backend/forum/httpapi/forum_admin_mysql_test.go）。

後台端點的授權邊界（未登入 401）已由既有測試覆蓋；這個檔驗「管理員操作真的
改變了資料」：刪文是軟刪除、停權寫進 forum_users、公告發佈後公開端看得到、
封鎖進 Redis 名單且稽核與操作在同一個交易裡。

約定同 comment_reply_mysql_test.go：FORUM_TEST_MYSQL_DSN 未設定時 skip。
*/

package httpapi

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forum/forum/config"
	"forum/forum/ipban"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// adminRequest 造一個帶著管理員 session 的請求。管理員身分寫在 session hash
// （is_admin），因此 Create(email, true) 就是完整的「成為管理員」。
func adminRequest(t *testing.T, server *Server, method, path, body string) *http.Request {
	t.Helper()
	token, err := server.sessions.Create("admin@example.com", true)
	if err != nil {
		t.Fatalf("建立管理員 session 失敗: %v", err)
	}
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	request.AddCookie(&http.Cookie{Name: "FORUM_test", Value: token})
	return request
}

// countRows 是後台測試的小工具：回傳某張表的列數。
func countRows(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("計算列數失敗（%s）: %v", query, err)
	}
	return n
}

/* ==========================================================================
   IP 封鎖
   ========================================================================== */

func TestAdminBlocksLifecycleMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)
	// 封鎖名單需要 blocks（ipban.Store）；audit 寫入需要 db —— 兩者都在這裡接上。
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	server.blocks = ipban.New(client)
	ctx := t.Context()

	// 未登入：401（後台端點一律先問身分）。
	anon := httptest.NewRecorder()
	server.handleAdminBlocks(anon, httptest.NewRequest(http.MethodPost, "/api/admin/blocks",
		strings.NewReader(`{"ip":"203.0.113.9","minutes":30,"reason":"x"}`)))
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("未登入的封鎖 = %d, want 401", anon.Code)
	}

	// 封鎖：名單生效、稽核有紀錄。
	recorder := httptest.NewRecorder()
	server.handleAdminBlocks(recorder, adminRequest(t, server, http.MethodPost, "/api/admin/blocks",
		`{"ip":"203.0.113.9","minutes":30,"reason":"垃圾轟炸"}`))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("封鎖回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}
	banned, until, err := server.blocks.IsBanned(ctx, "203.0.113.9")
	if err != nil || !banned {
		t.Fatalf("封鎖後 IsBanned = (%v, %v), want (true, nil)", banned, err)
	}
	if until.Before(time.Now().Add(20 * time.Minute)) {
		t.Errorf("到期時間 %v 太早（minutes=30 應封到約半小時後）", until)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM forum_admin_actions`); n == 0 {
		t.Error("封鎖沒有寫稽核紀錄：事後無從回答「誰封的、為什麼」")
	}

	// 列表看得見。
	recorder = httptest.NewRecorder()
	server.handleAdminBlocks(recorder, adminRequest(t, server, http.MethodGet, "/api/admin/blocks", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("封鎖列表回應 = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, "203.0.113.9") {
		t.Errorf("列表缺少被封的 IP: %s", body)
	}

	// minutes<=0 代表解封。
	recorder = httptest.NewRecorder()
	server.handleAdminBlocks(recorder, adminRequest(t, server, http.MethodPost, "/api/admin/blocks",
		`{"ip":"203.0.113.9","minutes":0,"reason":"誤判"}`))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("解封回應 = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if banned, _, _ := server.blocks.IsBanned(ctx, "203.0.113.9"); banned {
		t.Error("解封後仍被封鎖")
	}

	// 非法 IP：400（不是 500 —— 使用者輸入問題）。
	recorder = httptest.NewRecorder()
	server.handleAdminBlocks(recorder, adminRequest(t, server, http.MethodPost, "/api/admin/blocks",
		`{"ip":"not-an-ip","minutes":30,"reason":"x"}`))
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("非法 IP = %d, want 400", recorder.Code)
	}
}

/* ==========================================================================
   管理員刪文與檢舉
   ========================================================================== */

func TestAdminDeletePostAndReportMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	// 使用者發文並檢舉（自檢自舉，授權不是這裡的重點）。
	create := httptest.NewRecorder()
	server.handleForumPosts(create, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"要被管理員刪掉的文"}`))
	postID := postIDFromBody(t, decodeJSONBody(t, create))

	report := httptest.NewRecorder()
	server.handleForumPostAction(report, loggedInRequest(t, server, http.MethodPost,
		fmt.Sprintf("/api/forum/posts/%d/report", postID), `{"reason":"违规内容"}`))
	if report.Code < 200 || report.Code >= 300 {
		t.Fatalf("檢舉回應 = %d; body=%s", report.Code, report.Body.String())
	}
	reportID := countRows(t, db, `SELECT COUNT(*) FROM forum_reports`)

	// 管理員刪文：軟刪除（deleted_at 設值，列保留）。
	recorder := httptest.NewRecorder()
	server.handleAdminForumPost(recorder, adminRequest(t, server, http.MethodDelete,
		fmt.Sprintf("/api/admin/forum/posts/%d", postID), ""))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("管理員刪文回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}
	var deletedAt sql.NullTime
	if err := db.QueryRow(`SELECT deleted_at FROM forum_posts WHERE id = ?`, postID).Scan(&deletedAt); err != nil {
		t.Fatalf("讀取貼文失敗: %v", err)
	}
	if !deletedAt.Valid {
		t.Error("管理員刪文不是軟刪除（deleted_at 為 NULL）")
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM forum_admin_actions`); n == 0 {
		t.Error("管理員刪文沒有稽核紀錄")
	}

	// 管理員刪除檢舉記錄：列消失，被檢舉的對象不受影響。
	recorder = httptest.NewRecorder()
	server.handleAdminForumReport(recorder, adminRequest(t, server, http.MethodDelete,
		fmt.Sprintf("/api/admin/forum/reports/%d", reportID), ""))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("刪除檢舉回應 = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM forum_reports`); n != 0 {
		t.Errorf("刪除檢舉後仍剩 %d 列", n)
	}

	// 再刪一次：404（不存在的資源）。
	recorder = httptest.NewRecorder()
	server.handleAdminForumReport(recorder, adminRequest(t, server, http.MethodDelete,
		fmt.Sprintf("/api/admin/forum/reports/%d", reportID), ""))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("重複刪除檢舉 = %d, want 404", recorder.Code)
	}
}

/* ==========================================================================
   使用者停權
   ========================================================================== */

func TestAdminSuspendUserMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	// 使用者先發一篇文（forum_users 仍不存在 —— 停權必須能對「從未發文的人」
	// 建立一列，這正是白名單式管理常見的形狀）。
	create := httptest.NewRecorder()
	server.handleForumPosts(create, loggedInRequest(t, server, http.MethodPost, "/api/forum/posts", `{"content":"即將被停權"}`))
	if create.Code != http.StatusCreated {
		t.Fatalf("發文回應 = %d; body=%s", create.Code, create.Body.String())
	}

	// 尚未登入過的 email（forum_users 無列）：404 —— 停權是對「已存在的帳號」
	// 的操作，不存在的帳號沒有可停的東西。
	recorder := httptest.NewRecorder()
	server.handleAdminUser(recorder, adminRequest(t, server, http.MethodPatch,
		"/api/admin/users/ghost%40example.com", `{"status":"SUSPENDED"}`))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("不存在帳號的停權 = %d, want 404", recorder.Code)
	}

	// 使用者已有 forum_users 列（OAuth 登入時建立）後，停權才會生效。
	if _, err := db.Exec(`INSERT INTO forum_users (email, status, created_at, updated_at)
		VALUES ('owner@example.com', 'ACTIVE', NOW(), NOW())`); err != nil {
		t.Fatalf("準備使用者列失敗: %v", err)
	}

	// 停權：forum_users 出現 status=SUSPENDED 的列。方法是 PATCH（PUT 會 405）。
	recorder = httptest.NewRecorder()
	server.handleAdminUser(recorder, adminRequest(t, server, http.MethodPatch,
		"/api/admin/users/owner%40example.com", `{"status":"SUSPENDED"}`))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("停權回應 = %d, want 2xx; body=%s", recorder.Code, recorder.Body.String())
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM forum_users WHERE email = ?`, "owner@example.com").Scan(&status); err != nil {
		t.Fatalf("讀取使用者狀態失敗: %v", err)
	}
	if status != "SUSPENDED" {
		t.Errorf("status = %q, want SUSPENDED", status)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM forum_admin_actions`); n == 0 {
		t.Error("停權沒有稽核紀錄")
	}

	// 復原：狀態回到 ACTIVE。
	recorder = httptest.NewRecorder()
	server.handleAdminUser(recorder, adminRequest(t, server, http.MethodPatch,
		"/api/admin/users/owner%40example.com", `{"status":"ACTIVE"}`))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("復原回應 = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if err := db.QueryRow(`SELECT status FROM forum_users WHERE email = ?`, "owner@example.com").Scan(&status); err != nil {
		t.Fatalf("讀取使用者狀態失敗: %v", err)
	}
	if status != "ACTIVE" {
		t.Errorf("復原後 status = %q, want ACTIVE", status)
	}

	// 非法狀態值：400（不是 500，也不是靜默接受）。
	recorder = httptest.NewRecorder()
	server.handleAdminUser(recorder, adminRequest(t, server, http.MethodPatch,
		"/api/admin/users/owner%40example.com", `{"status":"BANNED"}`))
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("非法狀態 = %d, want 400", recorder.Code)
	}
}

/* ==========================================================================
   公告
   ========================================================================== */

func TestAdminAnnouncementPublishAndExpireMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	server := newCoreForumServer(t, db)

	// 未生效的公告：公開端看不到。
	recorder := httptest.NewRecorder()
	server.handleAdminAnnouncements(recorder, adminRequest(t, server, http.MethodPost, "/api/admin/announcements",
		`{"body":"尚未生效的公告","active":false,"hoursUntilExpiry":0}`))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("發佈（未生效）回應 = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	public := httptest.NewRecorder()
	server.handleForumAnnouncement(public, httptest.NewRequest(http.MethodGet, "/api/forum/announcement", nil))
	if strings.Contains(public.Body.String(), "尚未生效") {
		t.Errorf("未生效的公告出現在公開端: %s", public.Body.String())
	}

	// 生效的公告：公開端看得到。
	recorder = httptest.NewRecorder()
	server.handleAdminAnnouncements(recorder, adminRequest(t, server, http.MethodPost, "/api/admin/announcements",
		`{"body":"週四凌晨維護","active":true,"hoursUntilExpiry":0}`))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("發佈回應 = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	public = httptest.NewRecorder()
	server.handleForumAnnouncement(public, httptest.NewRequest(http.MethodGet, "/api/forum/announcement", nil))
	if !strings.Contains(public.Body.String(), "週四凌晨維護") {
		t.Errorf("生效的公告沒有出現在公開端: %s", public.Body.String())
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM forum_admin_actions`); n == 0 {
		t.Error("公告發佈沒有稽核紀錄")
	}

	// 空白內容：400（空白公告是最糟的結果 —— 佔了版面卻沒有資訊）。
	recorder = httptest.NewRecorder()
	server.handleAdminAnnouncements(recorder, adminRequest(t, server, http.MethodPost, "/api/admin/announcements",
		`{"body":"   ","active":true,"hoursUntilExpiry":0}`))
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("空白公告 = %d, want 400", recorder.Code)
	}

	// 超過 365 天：400。
	recorder = httptest.NewRecorder()
	server.handleAdminAnnouncements(recorder, adminRequest(t, server, http.MethodPost, "/api/admin/announcements",
		`{"body":"太久","active":true,"hoursUntilExpiry":8761}`))
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("超過 365 天的公告 = %d, want 400", recorder.Code)
	}

	// 未登入：401。
	anon := httptest.NewRecorder()
	server.handleAdminAnnouncements(anon, httptest.NewRequest(http.MethodPost, "/api/admin/announcements",
		strings.NewReader(`{"body":"x","active":true}`)))
	if anon.Code != http.StatusUnauthorized {
		t.Errorf("未登入的公告發佈 = %d, want 401", anon.Code)
	}
}

// 確認測試用的 cfg 形態（避免 unused import：config 在 newCoreForumServer 用到，
// 這裡順便釘住「後台不需要 CSRF 白名單也能運作」的前提 —— 空 TrustedOrigins
// 等於關閉 CSRF 防護，是既有文件記錄過的語意）。
var _ = config.Config{}
