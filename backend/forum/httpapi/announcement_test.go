/*
站內公告與置頂的測試（httpapi/announcement_test.go）。

這個功能的每一個性質都是「壞掉時不會報錯」的，因此測試集中在四個地方：

  1. 內文驗證的邊界
     空白內容會產生一條「有邊框、裡面是空的」橫幅 —— 那比沒有公告更糟，
     因為它佔了版面卻沒有資訊。而超過 300 字在渲染時被截斷，會讓管理員
     以為整則都顯示出來了。

  2. 到期時間的三態
     expires 為 NULL（永不過期）、有值但未到、有值但已過。這三種在 SQL 的
     三值邏輯下行為不同，而錯寫成 `expires_at > NOW()` 會讓「永不過期」的
     公告**永遠查不到**（NULL > NOW() 的結果是 NULL，不是 true）。

  3. 授權
     公開端點不授權（它本來就是公開的），後臺端點必須擋住未登入與非管理員 ——
     它能改變全站每個訪客看到的內容。

  4. 內容驗證的錯誤訊息
     它們會被直接顯示給管理員，因此必須是人看得懂的，而不是 net/SQL 套件的
     原始字串。

刻意沒有測的：交易內「先關舊再開新」的順序。這個專案沒有 MySQL 替身
（見 docs/KNOWN_ISSUES.md 的已知問題），而那需要真的資料庫才能驗證；它由程式碼結構
保證 —— 兩道 UPDATE 與 INSERT 在同一個 tx，且 defer tx.Rollback()。
*/

package httpapi

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"forum/forum/session"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

/* ==========================================================================
   內容驗證
   ========================================================================== */

func TestNormalizeAnnouncement(t *testing.T) {
	t.Run("修剪空白後接受", func(t *testing.T) {
		body, expires, err := normalizeAnnouncement("  系統將於週四維護  ", 0)
		if err != nil {
			t.Fatalf("normalizeAnnouncement: %v", err)
		}
		if body != "系統將於週四維護" {
			t.Errorf("body = %q, want 修剪前後的內容", body)
		}
		if expires.Valid {
			t.Error("hours=0 應回傳「永不過期」（Valid 為 false）")
		}
	})

	t.Run("空內容是錯誤", func(t *testing.T) {
		// 這一條是「最容易被漏掉」的一條：空白內容不會讓任何 SQL 出錯，
		// 症狀是使用者看到一個空的橫幅。
		for _, input := range []string{"", "   ", "\n\t "} {
			if _, _, err := normalizeAnnouncement(input, 0); err == nil {
				t.Errorf("normalizeAnnouncement(%q) 應回傳錯誤", input)
			}
		}
	})

	t.Run("超過上限是錯誤", func(t *testing.T) {
		overlong := ""
		for range maxAnnouncementLength + 1 {
			overlong += "中"
		}
		_, _, err := normalizeAnnouncement(overlong, 0)
		if err == nil {
			t.Fatal("超過上限應回傳錯誤")
		}
		// 錯誤訊息必須指出實際長度：只說「太長」時管理員會反覆刪字試。
		if !contains(err.Error(), "301") {
			t.Errorf("錯誤訊息 = %q，應包含實際字數", err.Error())
		}
	})

	t.Run("剛好等於上限是允許的", func(t *testing.T) {
		exact := ""
		for range maxAnnouncementLength {
			exact += "文"
		}
		if _, _, err := normalizeAnnouncement(exact, 0); err != nil {
			t.Errorf("剛好 %d 字應通過: %v", maxAnnouncementLength, err)
		}
	})

	t.Run("負數小時是錯誤", func(t *testing.T) {
		if _, _, err := normalizeAnnouncement("內容", -1); err == nil {
			t.Error("負數有效時間應回傳錯誤")
		}
	})

	t.Run("有效時間上限是 365 天", func(t *testing.T) {
		if _, _, err := normalizeAnnouncement("內容", 365*24); err != nil {
			t.Errorf("剛好 365 天應通過: %v", err)
		}
		if _, _, err := normalizeAnnouncement("內容", 365*24+1); err == nil {
			t.Error("超過 365 天應回傳錯誤")
		}
	})
}

// contains 是 strings.Contains 的本地別名。這個檔案只需要它一次，而匯入
// strings 只為了一個呼叫點不划算 —— 與專案其他測試檔的 itoa 同樣的理由。
func contains(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

/* ==========================================================================
   授權
   ========================================================================== */

func newAnnouncementTestServer(t *testing.T) *Server {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &Server{sessions: session.NewManager("FORUM_test", time.Hour, false, client)}
}

// 後臺的公告端點能改變**每個訪客**看到的內容，因此未登入必須 401。
func TestAdminAnnouncementsRequireAdmin(t *testing.T) {
	server := newAnnouncementTestServer(t)
	cases := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
	}{
		{"列出", server.handleAdminAnnouncements, http.MethodGet, "/api/admin/announcements"},
		{"發佈", server.handleAdminAnnouncements, http.MethodPost, "/api/admin/announcements"},
		{"修改", server.handleAdminAnnouncement, http.MethodPatch, "/api/admin/announcements/1"},
		{"置頂", server.handleAdminPostPin, http.MethodPost, "/api/admin/forum/posts/1/pin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.path, nil)
			tc.handler(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("回應 = %d, want 401", recorder.Code)
			}
		})
	}
}

// 公開端點不該要求任何身分：它就是給匿名訪客看的。
//
// 這一條同時覆蓋了「沒有公告」這個壓倒性的常見情況：回應是 200 且
// announcement 為 null，而不是 404 —— 前端每個頁面載入都會問一次，若「沒有
// 公告」是 404，前端就得區分「正常的沒有」與「端點壞了」兩種 404。
//
// 這支測試需要一個真的 *sql.DB，而這個專案沒有 MySQL 替身（見 docs/KNOWN_ISSUES.md 的已知
// 問題），因此用 database/sql/driver 註冊一個最小驅動程式。它只用標準函式庫
// —— 匯入 database/sql/driver 不是「引入新的相依」。
var announcementDriverOnce sync.Once

// announcementRows 是一列都沒有的結果集，用來表示「查不到」。
type announcementRows struct{}

func (announcementRows) Columns() []string { return []string{"body", "created_at", "expires"} }
func (announcementRows) Close() error      { return nil }
func (announcementRows) Next(dest []driver.Value) error {
	// io.EOF 代表結果集已耗盡，也就是「沒有任何符合條件的公告」。
	return io.EOF
}

// announcementDriver 是一個永遠回空結果集的驅動程式。
type announcementDriver struct{}

func (announcementDriver) Open(string) (driver.Conn, error) { return announcementConn{}, nil }

type announcementConn struct{}

func (announcementConn) Prepare(string) (driver.Stmt, error) { return announcementStmt{}, nil }
func (announcementConn) Close() error                        { return nil }
func (announcementConn) Begin() (driver.Tx, error)           { return nil, io.EOF }

type announcementStmt struct{}

func (announcementStmt) Close() error  { return nil }
func (announcementStmt) NumInput() int { return -1 }
func (announcementStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, io.EOF
}
func (announcementStmt) Query([]driver.Value) (driver.Rows, error) { return announcementRows{}, nil }

func TestForumAnnouncementIsPublicAndNullWhenAbsent(t *testing.T) {
	announcementDriverOnce.Do(func() {
		sql.Register("forum-announcement-stub", announcementDriver{})
	})
	db, err := sql.Open("forum-announcement-stub", "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	server := &Server{db: db}

	recorder := httptest.NewRecorder()
	server.handleForumAnnouncement(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/announcement", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("回應 = %d, want 200（公開端點不該要求身分）", recorder.Code)
	}
	// 沒有公告時回 200 + announcement: null。
	if !strings.Contains(recorder.Body.String(), `"announcement":null`) {
		t.Errorf("body = %q，應為 announcement:null", recorder.Body.String())
	}
}

// 公開端點只接受 GET。加了 POST 就會讓「?…」之類的誤請求被當成寫入。
func TestForumAnnouncementRejectsNonGet(t *testing.T) {
	server := newAnnouncementTestServer(t)
	recorder := httptest.NewRecorder()
	server.handleForumAnnouncement(recorder, httptest.NewRequest(http.MethodPost, "/api/forum/announcement", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("回應 = %d, want 405", recorder.Code)
	}
}

/*
置頂路由的 id 解析。

這個路由的形狀是 /api/admin/forum/posts/{id}/pin，而 id 是用字串修剪從路徑上
取下來的。修剪函式的第二個參數是**字元集合**而不是尾綴，因此
"/api/admin/forum/posts/pin1/pin" 會被吃掉前導的 "pin" 而解析成 id 1 ——
網址指的貼文與實際被置頂的貼文不是同一篇，而稽核紀錄記的是後者。

症狀完全安靜：一個管理員照著某個畸形 URL 操作，紀錄裡出現一篇他沒有選的貼文。
因此這一節逐條釘住「什麼路徑解析成什麼 id」。

斷言直接打在純函式上而不是打 HTTP 回應：這條路由的前兩道關卡（管理員、來源）
都還沒碰到 id 解析，從 HTTP 端進來只會看到 401／403，測不到重點。
*/
func TestAdminPostPinIDFromPath(t *testing.T) {
	cases := []struct {
		path   string
		want   int64
		wantOK bool
	}{
		{"/api/admin/forum/posts/1/pin", 1, true},
		{"/api/admin/forum/posts/12/pin", 12, true},
		// 尾斜線不是這個路由的形狀（分流靠 HasSuffix "/pin"），因此它必須失敗。
		{"/api/admin/forum/posts/1/pin/", 0, false},
		// 這三筆是修掉 Trim 之後才失敗的：舊實作會把它們解析成 1、12、與 ""（→ 錯誤）。
		{"/api/admin/forum/posts/pin1/pin", 0, false},
		{"/api/admin/forum/posts/12pin/pin", 0, false},
		{"/api/admin/forum/posts/abc/pin", 0, false},
		{"/api/admin/forum/posts//pin", 0, false},
		// 非正值能解析出來，由 handler 的 id < 1 擋掉（兩道檢查缺一不可）。
		{"/api/admin/forum/posts/-1/pin", -1, true},
		{"/api/admin/forum/posts/0/pin", 0, true},
	}
	for _, tc := range cases {
		got, err := adminPostPinIDFromPath(tc.path)
		if tc.wantOK != (err == nil) || (err == nil && got != tc.want) {
			t.Errorf("adminPostPinIDFromPath(%q) = (%d, %v), want (%d, ok=%v)",
				tc.path, got, err, tc.want, tc.wantOK)
		}
	}
}
