/*
RSS 摘要的測試（httpapi/forum_rss_test.go）。

這個端點有兩個「壞掉時不會有任何錯誤」的性質，測試因此集中在這兩處：

  1. 標題推導
     貼文沒有標題欄位，標題由內文折疊空白與截斷而來。折疊與截斷寫錯的症狀
     是「訂閱清單裡出現一整篇文」或「所有標題都是空白的」—— 兩種都不會讓
     任何請求失敗，只有人去看閱讀器時才會發現。

  2. 快取三態
     ETag 相同時必須回 304（否則閱讀器每次輪詢都重抓整份文件），ETag 不同
     時必須回 200（否則訂閱者永遠看不到新貼文 —— 這是比上一條更嚴重的錯，
     因為它是「安靜地停止更新」）。這一組用最小驅動程式餵入確定的資料列，
     因此可以在沒有 MySQL 的環境驗證整條組稿流程。

     還有一個隱含的必要條件：**文件裡不能有會自己變動的欄位**，否則前兩態
     在跨秒之後都會失效（雜湊每秒都不同）。它由 TestForumRSSBuildsDocument
     釘住 lastBuildDate＝最新貼文的時間，而不是由這一組自己驗 —— 一次
     time.Now() 就會讓 304 永遠不發生，而那是沒有錯誤訊息的失效。

刻意沒有測的：SQL 本身的內容（deleted_at IS NULL、ORDER BY、LIMIT）。這個專案
沒有 MySQL 替身，而那條查詢的軟刪除過濾由 post_soft_delete_test.go 的靜態檢查
守住。
*/

package httpapi

import (
	"database/sql"
	"database/sql/driver"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"forum/forum/config"
)

/* ==========================================================================
   標題推導
   ========================================================================== */

func TestForumRSSTitle(t *testing.T) {
	long := strings.Repeat("字", 61)
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"單行原樣", "今天天氣很好", "今天天氣很好"},
		{"折疊換行與多個空白", "第一行\n\n第二行   第三行", "第一行 第二行 第三行"},
		{"折疊製表與回車", "a\tb\rc", "a b c"},
		{"剛好 60 字不截斷", strings.Repeat("字", 60), strings.Repeat("字", 60)},
		{"61 字截斷並加省略號", long, strings.Repeat("字", 60) + "…"},
		{"純空白得到空字串", "   \n\t ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := forumRSSTitle(tc.content); got != tc.want {
				t.Errorf("forumRSSTitle(%q) = %q, want %q", tc.content, got, tc.want)
			}
		})
	}
}

/* ==========================================================================
   最小驅動程式：餵入確定的資料列
   ========================================================================== */

// rssTestDriverOnce 確保 sql.Register 只被呼叫一次（重複註冊會 panic）。
// 同一個驅動程式名稱在測試之間共用，資料列由 rssTestRows 這個套件變數決定；
// 這些測試刻意不平行（Go 測試預設循序執行）。
var (
	rssTestDriverOnce sync.Once
	rssTestMu       sync.Mutex
	rssTestRows     = [][]driver.Value{}
)

type rssTestRowsSource struct{}

func (rssTestRowsSource) Columns() []string { return []string{"id", "content", "created_at"} }
func (rssTestRowsSource) Close() error      { return nil }
func (rssTestRowsSource) Next(dest []driver.Value) error {
	rssTestMu.Lock()
	defer rssTestMu.Unlock()
	if len(rssTestRows) == 0 {
		return io.EOF
	}
	copy(dest, rssTestRows[0])
	rssTestRows = rssTestRows[1:]
	return nil
}

type rssTestDriver struct{}

func (rssTestDriver) Open(string) (driver.Conn, error) { return rssTestConn{}, nil }

type rssTestConn struct{}

func (rssTestConn) Prepare(string) (driver.Stmt, error) { return rssTestStmt{}, nil }
func (rssTestConn) Close() error                        { return nil }
func (rssTestConn) Begin() (driver.Tx, error)           { return nil, io.EOF }

type rssTestStmt struct{}

func (rssTestStmt) Close() error  { return nil }
func (rssTestStmt) NumInput() int { return -1 }
func (rssTestStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, io.EOF
}
func (rssTestStmt) Query([]driver.Value) (driver.Rows, error) { return rssTestRowsSource{}, nil }

// newRSSTestServer 建立一個帶 stub 資料庫的 Server。
//
// cfg 以值型別持有（Server 的欄位就是值型別），因此直接賦值即可；站名與對外
// 網址是必填，因為頻道的 title 與 item 的 link 都從它們組出來。
func newRSSTestServer(t *testing.T, rows [][]driver.Value) *Server {
	t.Helper()
	rssTestDriverOnce.Do(func() { sql.Register("forum-rss-stub", rssTestDriver{}) })
	rssTestMu.Lock()
	rssTestRows = rows
	rssTestMu.Unlock()

	db, err := sql.Open("forum-rss-stub", "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Server{
		db: db,
		cfg: config.Config{
			ForumName:        "測試論壇",
			ForumDescription: "測試用",
			PublicBaseURL:    "https://forum.example",
		},
	}
}

/* ==========================================================================
   整份文件
   ========================================================================== */

func TestForumRSSBuildsDocument(t *testing.T) {
	created := time.Date(2026, 10, 17, 8, 30, 0, 0, time.UTC)
	server := newRSSTestServer(t, [][]driver.Value{
		{int64(7), "含有 <標籤> 與 & 符號的內容\n第二行", created},
	})

	recorder := httptest.NewRecorder()
	server.handleForumRSS(recorder, httptest.NewRequest(http.MethodGet, "/feed.xml", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("回應 = %d, want 200（body=%s）", recorder.Code, recorder.Body.String())
	}
	if ct := recorder.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/rss+xml") {
		t.Errorf("Content-Type = %q, want application/rss+xml...", ct)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"<title>測試論壇</title>",
		"<link>https://forum.example/forum</link>",
		"<link>https://forum.example/forum/post/7</link>",
		"<pubDate>Sat, 17 Oct 2026 08:30:00 GMT</pubDate>",
		`<guid isPermaLink="true">https://forum.example/forum/post/7</guid>`,
		// lastBuildDate 必須等於最新一則貼文的時間，而不是「現在」。
		// 這一行是那個設計的迴歸測試：改回 time.Now() 之後，文件會通過
		// 底下所有斷言（它是合法的 XML、每個欄位都在），只有這裡會紅 ——
		// 而那個錯誤的症狀正是「ETag 每秒都不同，304 永遠不發生」，
		// 在真實環境裡沒有任何錯誤訊息。日期刻意與 fixture 相同而非用
		// 寬鬆的比對：內容沒變就不該有一個會自己變動的欄位。
		"<lastBuildDate>Sat, 17 Oct 2026 08:30:00 GMT</lastBuildDate>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("RSS 缺少 %q\n---\n%s", want, body)
		}
	}
	// encoding/xml 必須把內文的角括號與 & 轉義；不轉義的話閱讀器會直接
	// 把內文當成 XML 標籤解析 —— 那份 feed 在客戶端眼中是壞的。
	if strings.Contains(body, "含有 <標籤>") {
		t.Errorf("內文沒有被轉義：%s", body)
	}
	if !strings.Contains(body, "含有 &lt;標籤&gt; 與 &amp; 符號的內容") {
		t.Errorf("內文轉義結果不如預期\n---\n%s", body)
	}
	// 標題折疊空白（換行變成空格）但不截斷（這篇很短）。
	if !strings.Contains(body, "<title>含有 &lt;標籤&gt; 與 &amp; 符號的內容 第二行</title>") {
		t.Errorf("標題不如預期\n---\n%s", body)
	}
	// 整份文件必須是合法的 XML（最後一道保險：任何欄位漏轉義都會在這裡失敗）。
	var parsed struct {
		XMLName xml.Name `xml:"rss"`
		Version string   `xml:"version,attr"`
	}
	if err := xml.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("RSS 不是合法 XML: %v", err)
	}
	if parsed.Version != "2.0" {
		t.Errorf("rss version = %q, want 2.0", parsed.Version)
	}
}

// 沒有貼文時必須仍然是一份合法的 RSS（items 為空），而不是 404：
// 訂閱一個還沒有內容的站是合法狀態，而閱讀器對 404 的解讀是「feed 壞了」。
func TestForumRSSIsValidWhenEmpty(t *testing.T) {
	server := newRSSTestServer(t, nil)

	recorder := httptest.NewRecorder()
	server.handleForumRSS(recorder, httptest.NewRequest(http.MethodGet, "/feed.xml", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("回應 = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "<channel>") || strings.Contains(body, "<item>") {
		t.Errorf("空摘要的內容不如預期\n---\n%s", body)
	}
	// 沒有貼文時 lastBuildDate 整欄省略（omitempty）：它是唯一會與內容無關地
	// 變動的欄位，而空 feed 沒有任何「最新內容的時間」可以填。填 time.Now()
	// 的症狀是每一份空 feed 都有不同的 ETag —— 一個還沒有貼文的站，輪詢
	// 每一次都會重抓整份文件。
	if strings.Contains(body, "<lastBuildDate>") {
		t.Errorf("空摘要不該帶 lastBuildDate\n---\n%s", body)
	}
	if err := xml.Unmarshal([]byte(body), new(struct {
		XMLName xml.Name `xml:"rss"`
	})); err != nil {
		t.Fatalf("空摘要不是合法 XML: %v", err)
	}
}

/* ==========================================================================
   快取三態
   ========================================================================== */

// 這一組的三個狀態都建立在「文件本身只由內容決定」這個性質上：
//
//   - 內容相同 → 同一個雜湊 → 304。文件裡**不能有會自己變動的欄位**
//     （lastBuildDate 由最新貼文的時間推導，見 TestForumRSSBuildsDocument
//     的斷言），否則同一個內容會在下一秒得到不同的 ETag，304 永遠不會發生。
//   - 內容改變 → 新的雜湊 → 200。
//   - 304 不帶 body 與 Content-Type。
/* ==========================================================================
   快取與 session 的邊界
   ========================================================================== */

// TestForumRSSNeverCarriesSessionCookie 釘住一條安全不變條件：RSS 回應
// 不得帶著 session token 出去。
//
// 背景：每一個請求（含這一個）都會經過最外層的 session 續期 middleware，
// 續期會在回應上寫入 `Set-Cookie: <session token>`。回應內容是公開的，但
// 這個表頭不是。若回應同時標了 `Cache-Control: public`，共享快取
// （nginx / Caddy / Cloudflare）就會把它存下來，而 RFC 9111 §3.1 要求
// 快取連同收到的所有表頭一起存；回應沒有 Vary: Cookie，所以 cookie 不在
// cache key 裡 —— 任何訪客都會拿到登入者的 token（完整帳號接管）。
//
// 因此兩件事必須同時成立：
//  1. Cache-Control 是 private，不是 public。
//  2. 回應上沒有 Set-Cookie（handler 必須把續期寫進來的那個拿掉）。
//
// 這個測試用「先把 Set-Cookie 塞進 recorder 的表頭」模擬 middleware 的
// 行為：handler 必須自己清掉它，而不是假設「表頭本來就不會有」。
func TestForumRSSNeverCarriesSessionCookie(t *testing.T) {
	created := time.Date(2026, 10, 17, 8, 30, 0, 0, time.UTC)
	server := newRSSTestServer(t, [][]driver.Value{{int64(1), "內容", created}})

	recorder := httptest.NewRecorder()
	// 模擬 session 續期 middleware 先寫進來的表頭。
	recorder.Header().Set("Set-Cookie", "forum_session=s3cr3t-token; Path=/; HttpOnly")

	server.handleForumRSS(recorder, httptest.NewRequest(http.MethodGet, "/feed.xml", nil))

	if got := recorder.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("RSS 回應帶了 Set-Cookie %q —— 共享快取會把它一起存下來並發給所有訪客", got)
	}
	cache := recorder.Header().Get("Cache-Control")
	if strings.Contains(cache, "public") {
		t.Errorf("Cache-Control = %q，含 public —— 共享快取會存下這份回應", cache)
	}
	if !strings.Contains(cache, "private") {
		t.Errorf("Cache-Control = %q, want 含 private", cache)
	}
	if !strings.Contains(cache, "max-age") {
		t.Errorf("Cache-Control = %q, want 含 max-age（ETag 的 304 好處靠它）", cache)
	}
}

func TestForumRSSCaching(t *testing.T) {
	created := time.Date(2026, 10, 17, 8, 30, 0, 0, time.UTC)
	server := newRSSTestServer(t, [][]driver.Value{{int64(1), "內容", created}})

	// seed 重新放回同一組資料列。stub 驅動程式每被問一次就吃掉一行（它沒有
	// 游標狀態，見 rssTestRowsSource），因此每個請求都要先放回自己那一組 ——
	// 忘了放不會有任何錯誤，只會讓那一份文件變成「沒有貼文」的空 feed，而
	// 空 feed 的雜湊當然不同：症狀是一個看起來像快取壞掉的失敗。
	seed := func() {
		rssTestMu.Lock()
		rssTestRows = [][]driver.Value{{int64(1), "內容", created}}
		rssTestMu.Unlock()
	}

	first := httptest.NewRecorder()
	server.handleForumRSS(first, httptest.NewRequest(http.MethodGet, "/feed.xml", nil))
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("第一次回應沒有 ETag")
	}
	if cache := first.Header().Get("Cache-Control"); !strings.Contains(cache, "max-age") {
		t.Errorf("Cache-Control = %q, want 含 max-age", cache)
	}

	// 同一份內容再問一次：先確認「輪詢時算出來的雜湊會不會變」。內容沒變而
	// 雜湊變了（文件裡有一個會自己走動的欄位，例如 time.Now() 組出來的
	// lastBuildDate）就是底下 304 永遠不會發生的原因，因此在這裡先釘住它。
	seed()
	repeat := httptest.NewRecorder()
	server.handleForumRSS(repeat, httptest.NewRequest(http.MethodGet, "/feed.xml", nil))
	if repeatETag := repeat.Header().Get("ETag"); repeatETag != etag {
		t.Fatalf("同一份內容的 ETag 不穩定：%q → %q（文件裡有一個會自己變動的欄位）", etag, repeatETag)
	}

	// 同一份內容 + 同一個 ETag → 304 且沒有 body。
	seed()
	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/feed.xml", nil)
	request.Header.Set("If-None-Match", etag)
	server.handleForumRSS(second, request)
	if second.Code != http.StatusNotModified {
		t.Fatalf("帶相同 ETag 的回應 = %d, want 304（body=%s）", second.Code, second.Body.String())
	}
	if second.Body.Len() != 0 {
		t.Errorf("304 帶了 body: %q", second.Body.String())
	}
	if got := second.Header().Get("ETag"); got != etag {
		t.Errorf("304 的 ETag = %q, want %q", got, etag)
	}
	if ct := second.Header().Get("Content-Type"); ct != "" {
		t.Errorf("304 帶了 Content-Type %q（某些客戶端會據此等待 body）", ct)
	}

	// 內容變了 → 新的雜湊 → 200（而不是沿用舊的 304，那會讓訂閱者永遠看不到
	// 新貼文）。
	rssTestMu.Lock()
	rssTestRows = [][]driver.Value{
		{int64(2), "新貼文", created},
		{int64(1), "內容", created},
	}
	rssTestMu.Unlock()
	third := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/feed.xml", nil)
	request.Header.Set("If-None-Match", etag)
	server.handleForumRSS(third, request)
	if third.Code != http.StatusOK {
		t.Fatalf("內容改變後的回應 = %d, want 200", third.Code)
	}
	if newETag := third.Header().Get("ETag"); newETag == etag || newETag == "" {
		t.Errorf("內容改變後 ETag = %q，應與舊值 %q 不同", newETag, etag)
	}
}

/* ==========================================================================
   方法
   ========================================================================== */

// RSS 是唯讀的：POST 必須 405，而且要在碰資料庫之前就被擋下。
func TestForumRSSRejectsNonGet(t *testing.T) {
	server := newRSSTestServer(t, nil)
	// 清空資料列：這個測試不該碰任何查詢，若它意外打了資料庫也只會拿到空集合，
	// 斷言仍然只看狀態碼。
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		server.handleForumRSS(recorder, httptest.NewRequest(method, "/feed.xml", nil))
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s 的回應 = %d, want 405", method, recorder.Code)
		}
	}
}
