/*
不需要真實 MySQL 的 purge 套件測試（backend/forum/purge/cov_purge_test.go）。

purge_test.go 的端到端測試需要 FORUM_TEST_MYSQL_DSN；這個檔案改用一個
database/sql/driver 替身，讓 Prune 的每一條分支（沒有過期貼文、取回失敗、
掃描失敗、交易失敗、附圖回收的各種結果）在一般 `go test` 下都能執行。

替身的規則與 httpapi/cov_harness_test.go 相同：Query 沒登記就回空結果集、
Exec 預設成功、交易全部成功。差別是這裡的 Query 規則可以攜帶一串「依呼叫
次序」的回應，因為 Prune 會對同一個 imageReferenceCount 查詢連續問多个檔名。
*/

package purge

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/* ==========================================================================
   驅動替身
   ========================================================================== */

type purgeCovValue = driver.Value

type purgeCovResponse struct {
	cols    []string
	rows    [][]purgeCovValue
	err     error // Query 本身失敗
	rowsErr error // 資料列讀到一半／讀完後失敗（rows.Err）
}

type purgeCovQueryRule struct {
	substr    string
	responses []purgeCovResponse
	call      int
}

type purgeCovExecRule struct {
	substr string
	err    error
}

type purgeCovExecCall struct {
	query string
	args  []any
}

type purgeCovSource struct {
	mu       sync.Mutex
	queries  []*purgeCovQueryRule
	execs    []purgeCovExecRule
	seen     []string
	executed []purgeCovExecCall
	beginErr error
}

func (s *purgeCovSource) addQuery(rule *purgeCovQueryRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, rule)
}

func (s *purgeCovSource) addExec(rule purgeCovExecRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.execs = append(s.execs, rule)
}

func (s *purgeCovSource) saw(substr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, query := range s.seen {
		if strings.Contains(query, substr) {
			return true
		}
	}
	return false
}

func (s *purgeCovSource) execArgs(substr string) [][]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out [][]any
	for _, call := range s.executed {
		if strings.Contains(call.query, substr) {
			out = append(out, call.args)
		}
	}
	return out
}

var (
	purgeCovSeq      uint64
	purgeCovRegistry sync.Map
	purgeCovOnce     sync.Once
)

type purgeCovDriver struct{}

func (purgeCovDriver) Open(name string) (driver.Conn, error) {
	value, ok := purgeCovRegistry.Load(name)
	if !ok {
		return nil, fmt.Errorf("purge cov driver: unknown dsn %q", name)
	}
	return &purgeCovConn{src: value.(*purgeCovSource)}, nil
}

type purgeCovConn struct{ src *purgeCovSource }

func (c *purgeCovConn) Prepare(string) (driver.Stmt, error) { return purgeCovStmt{}, nil }
func (c *purgeCovConn) Close() error                        { return nil }
func (c *purgeCovConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *purgeCovConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	if c.src.beginErr != nil {
		return nil, c.src.beginErr
	}
	return purgeCovTx{}, nil
}
func (c *purgeCovConn) Ping(context.Context) error { return nil }

func (c *purgeCovConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)
	for _, rule := range c.src.queries {
		if !strings.Contains(query, rule.substr) {
			continue
		}
		response := rule.responses[len(rule.responses)-1]
		if rule.call < len(rule.responses) {
			response = rule.responses[rule.call]
		}
		rule.call++
		if response.err != nil {
			return nil, response.err
		}
		return &purgeCovRows{cols: response.cols, rows: response.rows, errAfter: response.rowsErr}, nil
	}
	return &purgeCovRows{}, nil
}

func (c *purgeCovConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)
	values := make([]any, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	c.src.executed = append(c.src.executed, purgeCovExecCall{query: query, args: values})
	for _, rule := range c.src.execs {
		if strings.Contains(query, rule.substr) && rule.err != nil {
			return nil, rule.err
		}
	}
	return purgeCovResult{}, nil
}

type purgeCovTx struct{}

func (purgeCovTx) Commit() error   { return nil }
func (purgeCovTx) Rollback() error { return nil }

type purgeCovStmt struct{}

func (purgeCovStmt) Close() error                                { return nil }
func (purgeCovStmt) NumInput() int                               { return -1 }
func (purgeCovStmt) Exec([]purgeCovValue) (driver.Result, error) { return purgeCovResult{}, nil }
func (purgeCovStmt) Query([]purgeCovValue) (driver.Rows, error)  { return &purgeCovRows{}, nil }

type purgeCovResult struct{}

func (purgeCovResult) LastInsertId() (int64, error) { return 1, nil }
func (purgeCovResult) RowsAffected() (int64, error) { return 1, nil }

type purgeCovRows struct {
	cols     []string
	rows     [][]purgeCovValue
	errAfter error
	next     int
}

func (r *purgeCovRows) Columns() []string { return r.cols }
func (r *purgeCovRows) Close() error      { return nil }
func (r *purgeCovRows) Next(dest []purgeCovValue) error {
	if r.next >= len(r.rows) {
		if r.errAfter != nil {
			return r.errAfter
		}
		return io.EOF
	}
	copy(dest, r.rows[r.next])
	r.next++
	return nil
}

// purgeCovDB 建立一個連到替身的 *sql.DB。
func purgeCovDB(t *testing.T) (*sql.DB, *purgeCovSource) {
	t.Helper()

	name := fmt.Sprintf("purge-cov-%d", atomic.AddUint64(&purgeCovSeq, 1))
	src := &purgeCovSource{}
	purgeCovRegistry.Store(name, src)
	purgeCovOnce.Do(func() { sql.Register("forum-purge-cov-stub", purgeCovDriver{}) })

	db, err := sql.Open("forum-purge-cov-stub", name)
	if err != nil {
		t.Fatalf("sql.Open(cov purge): %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		purgeCovRegistry.Delete(name)
	})
	return db, src
}

const purgeCovFetchSQL = "SELECT id, COALESCE(image_url"

func purgeCovFetchRule(responses ...purgeCovResponse) *purgeCovQueryRule {
	return &purgeCovQueryRule{substr: purgeCovFetchSQL, responses: responses}
}

func purgeCovPosts(rows ...[]purgeCovValue) purgeCovResponse {
	return purgeCovResponse{
		cols: []string{"id", "image_url"},
		rows: rows,
	}
}

func purgeCovRefRule(responses ...purgeCovResponse) *purgeCovQueryRule {
	return &purgeCovQueryRule{substr: "COUNT(*) FROM forum_posts WHERE image_url", responses: responses}
}

func purgeCovRef(count int) purgeCovResponse {
	return purgeCovResponse{cols: []string{"n"}, rows: [][]purgeCovValue{{int64(count)}}}
}

/* ==========================================================================
   Prune：沒有東西可清、取回失敗、掃描失敗
   ========================================================================== */

func TestCovPruneNoExpiredPosts(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts()))

	removed, err := NewPruner(db, nil, 90*24*time.Hour, time.Hour).Prune(context.Background())
	if err != nil || removed != 0 {
		t.Fatalf("沒有過期貼文時 Prune = (%d, %v), want (0, nil)", removed, err)
	}
	if !src.saw("deleted_at IS NOT NULL") {
		t.Errorf("Prune 沒有送出取回過期貼文的查詢")
	}
}

func TestCovPruneFetchErrors(t *testing.T) {
	t.Run("查詢失敗", func(t *testing.T) {
		db, src := purgeCovDB(t)
		src.addQuery(purgeCovFetchRule(purgeCovResponse{err: errors.New("query failed")}))
		if _, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background()); err == nil {
			t.Fatal("取回查詢失敗時應回傳錯誤")
		}
	})

	t.Run("掃描失敗", func(t *testing.T) {
		db, src := purgeCovDB(t)
		// 只給一個值給兩個欄位：image_url 會是 NULL，掃進 string 失敗。
		src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1)})))
		if _, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background()); err == nil {
			t.Fatal("資料列掃描失敗時應回傳錯誤")
		}
	})

	t.Run("rows.Err", func(t *testing.T) {
		db, src := purgeCovDB(t)
		src.addQuery(purgeCovFetchRule(purgeCovResponse{
			cols:    []string{"id", "image_url"},
			rowsErr: errors.New("stream broken"),
		}))
		if _, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background()); err == nil {
			t.Fatal("rows.Err 非 nil 時應回傳錯誤")
		}
	})
}

/* ==========================================================================
   Prune：交易與批次
   ========================================================================== */

func TestCovPruneBeginsTransaction(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1), ""})))
	src.beginErr = errors.New("cannot begin")

	removed, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background())
	if err == nil {
		t.Fatal("BeginTx 失敗時應回傳錯誤")
	}
	if removed != 0 {
		t.Errorf("交易失敗前不應計入任何貼文，removed=%d", removed)
	}
}

func TestCovPruneDeleteError(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(7), ""})))
	src.addExec(purgeCovExecRule{substr: "DELETE FROM forum_post_comments", err: errors.New("fk locked")})

	if _, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background()); err == nil {
		t.Fatal("交易內 DELETE 失敗時應回傳錯誤")
	}
	// 先刪關聯再刪主列：失敗發生在留言時，貼文主列不應被刪。
	if src.saw("DELETE FROM forum_posts WHERE id IN") {
		t.Errorf("留言刪除失敗後不應繼續刪貼文主列")
	}
}

func TestCovPruneLoopsUntilBatchIsShort(t *testing.T) {
	db, src := purgeCovDB(t)

	first := make([][]purgeCovValue, 0, pruneBatchSize)
	for i := 0; i < pruneBatchSize; i++ {
		first = append(first, []purgeCovValue{int64(i + 1), ""})
	}
	src.addQuery(purgeCovFetchRule(
		purgeCovResponse{cols: []string{"id", "image_url"}, rows: first},
		purgeCovPosts(), // 第二批為空 → 結束迴圈
	))

	removed, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != pruneBatchSize {
		t.Errorf("removed = %d, want %d（滿批後應再取一批）", removed, pruneBatchSize)
	}
}

/* ==========================================================================
   Prune：附圖回收
   ========================================================================== */

func TestCovPruneReleasesOnlyUnreferencedImages(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts(
		[]purgeCovValue{int64(1), "/files/"}, // 收斂後為空檔名：跳過
		[]purgeCovValue{int64(2), "a.png"},   // 引用 0、刪檔成功、刪紀錄
		[]purgeCovValue{int64(3), "a.png"},   // 重複檔名：同一批只處理一次
		[]purgeCovValue{int64(4), "b.png"},   // 仍被引用：保留
		[]purgeCovValue{int64(5), "c.png"},   // 引用查詢失敗：保守跳過
		[]purgeCovValue{int64(6), "d.png"},   // 檔案伺服器失敗：保留紀錄
		[]purgeCovValue{int64(7), "e.png"},   // 引用 0、刪檔成功
	)))
	src.addQuery(purgeCovRefRule(
		purgeCovRef(0), purgeCovRef(2),
		purgeCovResponse{err: errors.New("count failed")},
		purgeCovRef(0), purgeCovRef(0),
	))

	deleter := &purgeCovDeleter{fail: map[string]bool{"d.png": true}}
	removed, err := NewPruner(db, deleter, time.Hour, time.Hour).Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != 7 {
		t.Errorf("removed = %d, want 7（附圖回收失敗不影響貼文列的清理）", removed)
	}

	wantDeleted := []string{"a.png", "e.png"}
	if len(deleter.deleted) != len(wantDeleted) {
		t.Fatalf("刪除的檔案 = %v, want %v", deleter.deleted, wantDeleted)
	}
	for i := range wantDeleted {
		if deleter.deleted[i] != wantDeleted[i] {
			t.Errorf("deleted[%d] = %q, want %q", i, deleter.deleted[i], wantDeleted[i])
		}
	}

	if got := len(src.execArgs("DELETE FROM forum_media")); got != 2 {
		t.Errorf("forum_media 刪除次數 = %d, want 2（只有 a.png 與 e.png）", got)
	}
	for _, args := range src.execArgs("DELETE FROM forum_media") {
		if len(args) != 1 || (args[0] != "a.png" && args[0] != "e.png") {
			t.Errorf("forum_media 刪除了不該刪的檔名: %v", args)
		}
	}
}

func TestCovPruneReleasesImagesWithoutFileServer(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1), "nofiles.png"})))
	src.addQuery(purgeCovRefRule(purgeCovRef(0)))

	removed, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background())
	if err != nil || removed != 1 {
		t.Fatalf("Prune = (%d, %v), want (1, nil)", removed, err)
	}
	if got := len(src.execArgs("DELETE FROM forum_media")); got != 1 {
		t.Errorf("沒有檔案伺服器時仍應刪除 forum_media 紀錄，次數 = %d", got)
	}
}

func TestCovPruneMediaDeleteFailureIsLoggedOnly(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1), "media.png"})))
	src.addQuery(purgeCovRefRule(purgeCovRef(0)))
	src.addExec(purgeCovExecRule{substr: "DELETE FROM forum_media", err: errors.New("media locked")})

	removed, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background())
	if err != nil || removed != 1 {
		t.Fatalf("媒體紀錄刪除失敗不應影響 Prune 的結果，得到 (%d, %v)", removed, err)
	}
}

// purgeCovDeleter 可依檔名模擬檔案伺服器故障。
type purgeCovDeleter struct {
	deleted []string
	fail    map[string]bool
}

func (d *purgeCovDeleter) DeleteFile(_ context.Context, name string) error {
	if d.fail[name] {
		return errors.New("file server down")
	}
	d.deleted = append(d.deleted, name)
	return nil
}

/* ==========================================================================
   Run
   ========================================================================== */

func TestCovRunStopsOnCanceledContext(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消：Run 必須建立 ticker 後立刻從 ctx.Done 返回
	NewPruner(db, nil, time.Hour, time.Hour).Run(ctx)
}
