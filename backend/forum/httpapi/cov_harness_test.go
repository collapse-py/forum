/*
測試用資料庫替身與 Server 建構器（httpapi/cov_harness_test.go）。

這個檔案只給測試使用，目的是讓 handler 的「成功路徑」在沒有 MySQL 的環境裡
也能被執行。做法是在 database/sql/driver 上註冊一個極小的 in-memory 驅動：

  - QueryContext：依「SQL 片段」回傳預先登記的結果集；沒有登記就回空結果集。
  - ExecContext：預設成功（LastInsertId = 1、RowsAffected = 1）。
  - BeginTx / Commit / Rollback：全部成功。

與 sqlmock 的差異：這個替身**不要求**每一筆查詢都被預期，也不檢查順序。
handler 只要走到某條路徑，就不會因為「少登記一筆查詢」而失敗。代價是它不適合
用來斷言 SQL 形狀 —— 需要的話請用 sqlmock 或既有的 `*_mysql_test.go`。

命名一律以 `cov` 開頭，避免與既有測試檔的 helper 衝突。
*/

package httpapi

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"forum/forum/config"
	"forum/forum/session"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const covCookieName = "FORUM_cov_test"

// driverValue 只是 driver.Value 的短別名，讓測試檔註冊結果集時不必每次寫
// []driver.Value{...} 而多匯入一個套件。
type driverValue = driver.Value

// covOrigin 是測試 Server 的 TRUSTED_ORIGINS 唯一允許的來源。
const covOrigin = "https://forum.test"

type covQueryRule struct {
	substr string
	cols   []string
	rows   [][]driver.Value
	err    error
}

type covExecRule struct {
	substr string
	err    error
}

// covDB 是一份測試連線共享的規則表。
type covDB struct {
	mu      sync.Mutex
	queries []covQueryRule
	execs   []covExecRule
	seen    []string
}

func (d *covDB) addQuery(rule covQueryRule) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queries = append(d.queries, rule)
}

func (d *covDB) addExec(rule covExecRule) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.execs = append(d.execs, rule)
}

func (d *covDB) matchQuery(query string) (covQueryRule, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen = append(d.seen, query)
	for _, rule := range d.queries {
		if strings.Contains(query, rule.substr) {
			return rule, true
		}
	}
	return covQueryRule{}, false
}

func (d *covDB) matchExec(query string) (covExecRule, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen = append(d.seen, query)
	for _, rule := range d.execs {
		if strings.Contains(query, rule.substr) {
			return rule, true
		}
	}
	return covExecRule{}, false
}

func (d *covDB) sawQuery(substr string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, query := range d.seen {
		if strings.Contains(query, substr) {
			return true
		}
	}
	return false
}

var (
	covDBSeq      uint64
	covDBRegistry sync.Map
	covDriverOnce sync.Once
)

type covDriver struct{}

func (covDriver) Open(name string) (driver.Conn, error) {
	value, ok := covDBRegistry.Load(name)
	if !ok {
		return nil, fmt.Errorf("cov driver: unknown dsn %q", name)
	}
	return &covConn{db: value.(*covDB)}, nil
}

type covConn struct{ db *covDB }

func (c *covConn) Prepare(string) (driver.Stmt, error) { return covStmt{}, nil }
func (c *covConn) Close() error                        { return nil }
func (c *covConn) Begin() (driver.Tx, error)           { return covTx{}, nil }
func (c *covConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return covTx{}, nil
}
func (c *covConn) Ping(context.Context) error { return nil }

func (c *covConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	rule, ok := c.db.matchQuery(query)
	if !ok {
		// 沒有登記 → 空結果集。對 QueryRow 而言就是 sql.ErrNoRows，對 Query
		// 而言就是「這個清單是空的」，兩者都是 handler 必須處理的正常情況。
		return &covRows{}, nil
	}
	if rule.err != nil {
		return nil, rule.err
	}
	return &covRows{cols: rule.cols, rows: rule.rows}, nil
}

func (c *covConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	rule, ok := c.db.matchExec(query)
	if ok && rule.err != nil {
		return nil, rule.err
	}
	return covResult{}, nil
}

type covTx struct{}

func (covTx) Commit() error   { return nil }
func (covTx) Rollback() error { return nil }

type covStmt struct{}

func (covStmt) Close() error                               { return nil }
func (covStmt) NumInput() int                              { return -1 }
func (covStmt) Exec([]driver.Value) (driver.Result, error) { return covResult{}, nil }
func (covStmt) Query([]driver.Value) (driver.Rows, error)  { return &covRows{}, nil }

type covResult struct{}

func (covResult) LastInsertId() (int64, error) { return 1, nil }
func (covResult) RowsAffected() (int64, error) { return 1, nil }

type covRows struct {
	cols []string
	rows [][]driver.Value
	next int
}

func (r *covRows) Columns() []string { return r.cols }
func (r *covRows) Close() error      { return nil }
func (r *covRows) Next(dest []driver.Value) error {
	if r.next >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.next])
	r.next++
	return nil
}

// covWantCreatedID 解析回應中的 id，要求它存在且為正數。
//
// 建立類端點不再回傳替身的 LastInsertId（那個值永遠是 1），而是應用層
// 產生的雪花 id，因此測試不能斷言固定值；能斷言的是「有回一個可用的 id」。
func covWantCreatedID(t *testing.T, body string) int64 {
	t.Helper()
	var payload struct {
		ID   int64 `json:"id"`
		Item struct {
			ID int64 `json:"id"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("解析回應失敗: %v（body = %s）", err, body)
	}
	// 公開頁的建立端點把物件包在 item 裡，後臺則直接回 id。
	id := payload.ID
	if id == 0 {
		id = payload.Item.ID
	}
	if id <= 0 {
		t.Fatalf("回應的 id = %d，want > 0（body = %s）", id, body)
	}
	return id
}

// covHarness 把 Server、資料庫替身與 Redis 綁在一起。
type covHarness struct {
	t      *testing.T
	srv    *Server
	db     *covDB
	sqlDB  *sql.DB
	client *redis.Client
}

func newCovHarness(t *testing.T) *covHarness {
	t.Helper()

	name := fmt.Sprintf("cov-%d", atomic.AddUint64(&covDBSeq, 1))
	state := &covDB{}
	covDBRegistry.Store(name, state)
	covDriverOnce.Do(func() { sql.Register("forum-cov-stub", covDriver{}) })

	db, err := sql.Open("forum-cov-stub", name)
	if err != nil {
		t.Fatalf("sql.Open(cov): %v", err)
	}

	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
		covDBRegistry.Delete(name)
	})

	cfg := config.Config{
		ForumName:        "測試論壇",
		ForumShortName:   "測試",
		ForumDescription: "測試用論壇",
		TrustedOrigins:   []string{covOrigin},
		// 明確設定一個可解析的代理網段：NewServer 在空值時會印出「目前不信任
		// X-Forwarded-For」的警告，幾百個測試各印一次會把真正的失敗淹掉。
		TrustedProxyCIDRs:       "127.0.0.1/32",
		RateLimitRequests:       1000,
		RateLimitWindow:         time.Minute,
		RateLimitUploadRequests: 1000,
		RateLimitUploadWindow:   time.Minute,
		RateLimitAuthRequests:   1000,
		RateLimitAuthWindow:     time.Minute,
	}
	srv := NewServer(cfg, db, session.NewManager(covCookieName, time.Hour, false, client), client)

	return &covHarness{t: t, srv: srv, db: state, sqlDB: db, client: client}
}

// rows 登記一筆「SQL 含 substr 時回傳這個結果集」的規則。後登記的規則優先
// 使用的前提是它先被加入；比對順序是加入順序。
func (h *covHarness) rows(substr string, cols []string, rows ...[]driver.Value) {
	h.db.addQuery(covQueryRule{substr: substr, cols: cols, rows: rows})
}

// queryError 讓含 substr 的查詢回傳 err。
func (h *covHarness) queryError(substr string, err error) {
	h.db.addQuery(covQueryRule{substr: substr, err: err})
}

// execError 讓含 substr 的寫入回傳 err。
func (h *covHarness) execError(substr string, err error) {
	h.db.addExec(covExecRule{substr: substr, err: err})
}

func (h *covHarness) sawQuery(substr string) bool { return h.db.sawQuery(substr) }

// login 建立一個真的 session（miniredis），回傳可直接加進請求的 cookie。
func (h *covHarness) login(email string, admin bool) *http.Cookie {
	h.t.Helper()
	token, err := h.srv.sessions.Create(email, admin)
	if err != nil {
		h.t.Fatalf("建立 session 失敗: %v", err)
	}
	return &http.Cookie{Name: covCookieName, Value: token}
}

// request 建立一個帶 Origin 的請求；email 非空時附上登入 cookie。
func (h *covHarness) request(method, path, body, email string, admin bool) *http.Request {
	h.t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Origin", covOrigin)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if email != "" {
		req.AddCookie(h.login(email, admin))
	}
	return req
}

// do 直接呼叫 handler，回傳 recorder。
func (h *covHarness) do(handler http.HandlerFunc, method, path, body, email string, admin bool) *httptest.ResponseRecorder {
	h.t.Helper()
	recorder := httptest.NewRecorder()
	handler(recorder, h.request(method, path, body, email, admin))
	return recorder
}

func covWantStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("回應 = %d, want %d；body = %s", recorder.Code, want, recorder.Body.String())
	}
}
