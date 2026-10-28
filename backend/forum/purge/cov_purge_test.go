/*
不需要真實 MySQL 的 purge 套件測試（backend/forum/purge/cov_purge_test.go）。

purge_test.go 的端到端測試需要 FORUM_TEST_MYSQL_DSN；這個檔案改用一個
database/sql/driver 替身，讓 Prune 的每一條分支（沒有過期貼文、取回失敗、
掃描失敗、交易失敗、附圖回收的各種結果）在一般 `go test` 下都能執行。

替身的規則與 httpapi/cov_harness_test.go 相同：Query 沒登記就回空結果集、
Exec 預設成功、交易全部成功。差別是這裡的 Query 規則可以攜帶一串「依呼叫
次序」的回應，因為 Prune 會對同一個 imageReferenceCount 查詢連續問多個檔名。
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
	// queue 模擬 forum_media_deletions：以插入順序保存待刪檔名。
	queue    []string
	queueSet map[string]bool
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

// indexOf 回傳第一個包含 substr 的查詢在 seen 裡的位置；找不到回 -1。
// 用來釘住交易內的執行順序（例如留言檢舉必須在刪留言之前）。
func (s *purgeCovSource) indexOf(substr string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, query := range s.seen {
		if strings.Contains(query, substr) {
			return i
		}
	}
	return -1
}

// queuedNames 回傳目前模擬佇列裡的檔名（副本）。
func (s *purgeCovSource) queuedNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.queue...)
}

// enqueueLocked 模擬 INSERT IGNORE：重複檔名只保留第一列。
func (s *purgeCovSource) enqueueLocked(name string) {
	if s.queueSet == nil {
		s.queueSet = make(map[string]bool)
	}
	if s.queueSet[name] {
		return
	}
	s.queueSet[name] = true
	s.queue = append(s.queue, name)
}

func (s *purgeCovSource) dequeueLocked(name string) {
	if !s.queueSet[name] {
		return
	}
	delete(s.queueSet, name)
	for i, queued := range s.queue {
		if queued == name {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			return
		}
	}
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

func (c *purgeCovConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)

	// 佇列查詢由替身自己維護，讓「刪檔失敗 → 下一 tick 重試」可以在一般
	// go test 下完整走一遍，而不是靠測試手動餵回應。
	if strings.Contains(query, "FROM forum_media_deletions") {
		limit := len(c.src.queue)
		if len(args) > 0 {
			if n, ok := args[len(args)-1].Value.(int64); ok && int(n) < limit {
				limit = int(n)
			}
		}
		rows := make([][]purgeCovValue, 0, limit)
		for _, name := range c.src.queue[:limit] {
			rows = append(rows, []purgeCovValue{name})
		}
		return &purgeCovRows{cols: []string{"filename"}, rows: rows}, nil
	}

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

	switch {
	case strings.Contains(query, "INSERT IGNORE INTO forum_media_deletions"):
		// VALUES (?, ?), (?, ?)…：偶數位置是檔名，奇數位置是 created_at。
		for i := 0; i+1 < len(values); i += 2 {
			if name, ok := values[i].(string); ok {
				c.src.enqueueLocked(name)
			}
		}
	case strings.Contains(query, "DELETE FROM forum_media_deletions"):
		if len(values) == 1 {
			if name, ok := values[0].(string); ok {
				c.src.dequeueLocked(name)
			}
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

// purgeCovLockSQL 只出現在 deletePosts 交易內的重新確認查詢。
const purgeCovLockSQL = "FOR UPDATE"

// purgeCovLockRule 模擬交易內「仍符合刪除條件」的 id 集合。
//
// 多數測試會把 fetchExpiredPosts 回傳的 id 原樣放回來；TOCTOU 測試則刻意
// 少放一個 id，代表那一列在取回之後被管理員還原了。
func purgeCovLockRule(ids ...int64) *purgeCovQueryRule {
	rows := make([][]purgeCovValue, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, []purgeCovValue{id})
	}
	return &purgeCovQueryRule{
		substr:    purgeCovLockSQL,
		responses: []purgeCovResponse{{cols: []string{"id"}, rows: rows}},
	}
}

func purgeCovPosts(rows ...[]purgeCovValue) purgeCovResponse {
	return purgeCovResponse{
		cols: []string{"id", "image_url"},
		rows: rows,
	}
}

// purgeCovRefExactRule 對應引用計數的「新契約」那一段（JSON 陣列的等值判斷）。
//
// 與舊寫法那一段分成兩條規則：引用計數的查詢被拆成新契約與舊寫法兩段
// （見 imageReferenceCount 的說明），而兩段的子字串不同，測試才能分開
// 給它們不同的回應。
func purgeCovRefExactRule(responses ...purgeCovResponse) *purgeCovQueryRule {
	return &purgeCovQueryRule{substr: "JSON_CONTAINS(image_url, JSON_QUOTE(?))", responses: responses}
}

// purgeCovRefLegacyRule 對應引用計數的「舊寫法退路」那一段（字串收斂後等值）。
func purgeCovRefLegacyRule(responses ...purgeCovResponse) *purgeCovQueryRule {
	return &purgeCovQueryRule{substr: "image_url IS NOT NULL", responses: responses}
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
	src.addQuery(purgeCovLockRule(7))
	src.addExec(purgeCovExecRule{substr: "DELETE FROM forum_post_comments", err: errors.New("fk locked")})

	if _, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background()); err == nil {
		t.Fatal("交易內 DELETE 失敗時應回傳錯誤")
	}
	// 先刪關聯再刪主列：失敗發生在留言時，貼文主列不應被刪。
	if src.saw("DELETE FROM forum_posts WHERE id IN") {
		t.Errorf("留言刪除失敗後不應繼續刪貼文主列")
	}
}

// TestCovPruneLoopsUntilBatchIsShort 確認「這一批沒滿 → 沒有更多可清」。
func TestCovPruneLoopsUntilBatchIsShort(t *testing.T) {
	db, src := purgeCovDB(t)

	first := make([][]purgeCovValue, 0, pruneBatchSize)
	ids := make([]int64, 0, pruneBatchSize)
	for i := 0; i < pruneBatchSize; i++ {
		first = append(first, []purgeCovValue{int64(i + 1), ""})
		ids = append(ids, int64(i+1))
	}
	src.addQuery(purgeCovFetchRule(
		purgeCovResponse{cols: []string{"id", "image_url"}, rows: first},
		purgeCovPosts(), // 第二批為空 → 結束迴圈
	))
	src.addQuery(purgeCovLockRule(ids...))

	removed, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != pruneBatchSize {
		t.Errorf("removed = %d, want %d（滿批後應再取一批）", removed, pruneBatchSize)
	}
}

// TestCovPruneStopsAtTickLimit 釘住「每個 tick 最多處理 pruneMaxPerTick 列」。
//
// 沒有這個上限的話，保留期第一次到期時累積的幾十萬篇會在一個 tick 裡全部
// 清完：連線池被佔住、幾十萬次圖片引用查詢、整段時間裡正常請求都在排隊。
// 交易層的批次上限管不了這件事 —— 每個交易都很短，但交易**數量**沒有上限。
func TestCovPruneStopsAtTickLimit(t *testing.T) {
	db, src := purgeCovDB(t)

	full := make([][]purgeCovValue, 0, pruneBatchSize)
	ids := make([]int64, 0, pruneBatchSize)
	for i := 0; i < pruneBatchSize; i++ {
		full = append(full, []purgeCovValue{int64(i + 1), ""})
		ids = append(ids, int64(i+1))
	}
	// 每一批都回滿 → 迴圈只會被 tick 上限擋住。
	src.addQuery(purgeCovFetchRule(purgeCovResponse{cols: []string{"id", "image_url"}, rows: full}))
	src.addQuery(purgeCovLockRule(ids...))

	removed, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != pruneMaxPerTick {
		t.Errorf("removed = %d, want %d（每個 tick 的上限）", removed, pruneMaxPerTick)
	}
	fetches := 0
	for _, query := range src.seen {
		if strings.Contains(query, purgeCovFetchSQL) {
			fetches++
		}
	}
	if want := pruneMaxPerTick / pruneBatchSize; fetches != want {
		t.Errorf("取回批次數 = %d, want %d（上限 / 每批大小）", fetches, want)
	}
}

// TestCovPruneDeletesCommentReports 釘住「指向留言的檢舉工單也要一起清掉」。
//
// forum_reports.target_id 是多型參照（沒有外鍵），target_type 區分「檢舉
// 貼文」與「檢舉留言」。只刪 target_type='post' 會讓留言的工單留下來、
// 指向已經被刪掉的留言 —— 那種工單既沒辦法裁決也沒辦法關掉。
func TestCovPruneDeletesCommentReports(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1), ""})))
	src.addQuery(purgeCovLockRule(1))

	if _, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background()); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// 用 JOIN 直接刪，參數是貼文 id，不是先取回的留言 id 清單。
	args := src.execArgs("DELETE fr FROM forum_reports fr")
	if len(args) != 1 {
		t.Fatalf("留言檢舉工單的刪除次數 = %d, want 1", len(args))
	}
	if len(args[0]) != 1 || args[0][0] != int64(1) {
		t.Errorf("留言檢舉工單的刪除參數 = %v, want [1]（貼文 id）", args[0])
	}
	if !src.saw("JOIN forum_post_comments c ON c.id = fr.target_id") {
		t.Errorf("留言檢舉工單必須用 JOIN forum_post_comments 比對 target_id")
	}
	if !src.saw("fr.target_type = 'comment'") {
		t.Errorf("留言檢舉工單的刪除必須限定 target_type = 'comment'")
	}
	// 順序：先刪工單，再刪留言。反過來就找不到 target_id 了。
	reportIndex := src.indexOf("DELETE fr FROM forum_reports fr")
	commentIndex := src.indexOf("DELETE FROM forum_post_comments")
	if reportIndex < 0 || commentIndex < 0 || reportIndex >= commentIndex {
		t.Errorf("留言檢舉工單必須在刪留言之前刪除（工單 index=%d、留言 index=%d）", reportIndex, commentIndex)
	}
}

// TestCovPruneSkipsPostsRestoredBeforeDelete 釘住「取回與刪除之間被還原的
// 貼文不能被硬刪」。
//
// fetchExpiredPosts 讀到的是一個時間點的快照；管理員的還原是一個獨立的
// UPDATE。若 deletePosts 直接信任快照裡的 id，還原可能在取回之後、DELETE
// 之前提交 —— 使用者看到「還原成功」的貼文在下一個瞬間被硬刪，而且沒有任何
// 痕跡。deletePosts 因此在交易內用 SELECT ... FOR UPDATE 重新確認，這裡
// 讓鎖定查詢只回傳其中一個 id，模擬另一篇剛好被還原。
func TestCovPruneSkipsPostsRestoredBeforeDelete(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts(
		[]purgeCovValue{int64(1), "restored.png"},
		[]purgeCovValue{int64(2), "deleted.png"},
	)))
	// id=1 在取回後被還原：交易內的重新確認只看得到 id=2。
	src.addQuery(purgeCovLockRule(2))
	src.addQuery(purgeCovRefExactRule(purgeCovRef(0)))
	src.addQuery(purgeCovRefLegacyRule(purgeCovRef(0)))

	deleter := &purgeCovDeleter{}
	removed, err := NewPruner(db, deleter, time.Hour, time.Hour).Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1（只有仍在 deleted_at 的貼文算數）", removed)
	}

	postDeletes := src.execArgs("DELETE FROM forum_posts WHERE id IN")
	if len(postDeletes) != 1 {
		t.Fatalf("貼文主列刪除次數 = %d, want 1", len(postDeletes))
	}
	if len(postDeletes[0]) != 1 || postDeletes[0][0] != int64(2) {
		t.Errorf("貼文主列刪除參數 = %v, want [2]", postDeletes[0])
	}

	// 附圖回收也只能處理真的被刪掉的那一篇。
	if got := deleter.deletedNames(); !sameStringSet(got, "deleted.png") {
		t.Errorf("回收的附圖 = %v, want [deleted.png]", got)
	}

	// 佇列也只能排入真的被刪掉的那一篇；成功刪檔後應該出列。
	inserts := src.execArgs("INSERT IGNORE INTO forum_media_deletions")
	if len(inserts) != 1 {
		t.Fatalf("佇列寫入次數 = %d, want 1", len(inserts))
	}
	if len(inserts[0]) != 2 || inserts[0][0] != "deleted.png" {
		t.Errorf("佇列寫入參數 = %v, want [deleted.png <time>]", inserts[0])
	}
	if got := src.queuedNames(); len(got) != 0 {
		t.Errorf("刪檔成功後佇列應清空，got %v", got)
	}
}

// TestCovPruneFullyRestoredBatchDoesNothing 確認整批都在交易內被還原時，
// 不會刪掉任何關聯列，也不會回收任何附圖。
func TestCovPruneFullyRestoredBatchDoesNothing(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts(
		[]purgeCovValue{int64(1), "a.png"},
		[]purgeCovValue{int64(2), "b.png"},
	)))
	src.addQuery(purgeCovLockRule()) // 兩篇都已被還原

	deleter := &purgeCovDeleter{}
	removed, err := NewPruner(db, deleter, time.Hour, time.Hour).Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
	if src.saw("DELETE FROM forum_post_comments") || src.saw("DELETE FROM forum_posts WHERE id IN") {
		t.Errorf("整批被還原時不該送出任何刪除")
	}
	if src.saw("INSERT IGNORE INTO forum_media_deletions") {
		t.Errorf("整批被還原時不該寫入圖片刪除佇列")
	}
	if got := deleter.deletedNames(); len(got) != 0 {
		t.Errorf("整批被還原時不該回收附圖，got %v", got)
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
	src.addQuery(purgeCovLockRule(1, 2, 3, 4, 5, 6, 7))
	/*
	 * 引用計數分成「新契約（JSON 陣列）」與「舊寫法退路」兩段查詢（見
	 * imageReferenceCount 的說明），因此兩段各有自己的回應序列：
	 *
	 *   新契約：a.png → 0、b.png → 2、c.png → 查詢失敗、d.png → 0、e.png → 0
	 *   舊寫法：a.png → 0、d.png → 0、e.png → 0
	 *
	 * b.png 的新契約就有兩個引用，因此不會再問舊寫法；c.png 的查詢失敗，
	 * 因此也不會問舊寫法。
	 */
	src.addQuery(purgeCovRefExactRule(
		purgeCovRef(0),
		purgeCovRef(2),
		purgeCovResponse{err: errors.New("count failed")},
		purgeCovRef(0),
		purgeCovRef(0),
	))
	src.addQuery(purgeCovRefLegacyRule(
		purgeCovRef(0),
		purgeCovRef(0),
		purgeCovRef(0),
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
	if got := deleter.deletedNames(); !sameStringSet(got, wantDeleted...) {
		t.Errorf("刪除的檔案 = %v, want %v", got, wantDeleted)
	}

	// 佇列排入 a/b/c/d/e 各一次；b 仍被引用、a/e 成功刪檔，都應出列；
	// c 的引用查詢失敗、d 的檔案刪除失敗，必須留在佇列等下一 tick。
	inserts := src.execArgs("INSERT IGNORE INTO forum_media_deletions")
	if len(inserts) != 1 {
		t.Fatalf("佇列寫入次數 = %d, want 1", len(inserts))
	}
	var queued []string
	for i := 0; i+1 < len(inserts[0]); i += 2 {
		if name, ok := inserts[0][i].(string); ok {
			queued = append(queued, name)
		}
	}
	if !sameStringSet(queued, "a.png", "b.png", "c.png", "d.png", "e.png") {
		t.Errorf("佇列寫入的檔名 = %v，want a/b/c/d/e 各一次", queued)
	}
	if got := src.queuedNames(); !sameStringSet(got, "c.png", "d.png") {
		t.Errorf("佇列剩下的檔名 = %v, want [c.png d.png]", got)
	}

	if got := len(src.execArgs("DELETE FROM forum_media WHERE filename")); got != 2 {
		t.Errorf("forum_media 刪除次數 = %d, want 2（只有 a.png 與 e.png）", got)
	}
	for _, args := range src.execArgs("DELETE FROM forum_media WHERE filename") {
		if len(args) != 1 || (args[0] != "a.png" && args[0] != "e.png") {
			t.Errorf("forum_media 刪除了不該刪的檔名: %v", args)
		}
	}
}

// TestCovPruneKeepsMediaRecordWithoutFileServer 釘住「沒有檔案伺服器時
// 保留 forum_media 紀錄」這條規則。
//
// 為什麼要保留：那一列是「誰上傳了這個檔案」的唯一紀錄。檔案刪不掉卻把
// 紀錄刪掉，會留下一個沒有任何程式認得的孤兒 —— 連「使用者刪除自己上傳的
// 圖片」那條路徑都會回答「不是上傳者」，從此沒有任何程式能認領它。
func TestCovPruneKeepsMediaRecordWithoutFileServer(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1), "nofiles.png"})))
	src.addQuery(purgeCovLockRule(1))

	removed, err := NewPruner(db, nil, time.Hour, time.Hour).Prune(context.Background())
	if err != nil || removed != 1 {
		t.Fatalf("Prune = (%d, %v), want (1, nil)", removed, err)
	}
	if got := len(src.execArgs("DELETE FROM forum_media WHERE filename")); got != 0 {
		t.Errorf("沒有檔案伺服器時不該刪除 forum_media 紀錄（檔案還在、所有權紀錄是唯一的線索），次數 = %d", got)
	}
	if src.saw("INSERT IGNORE INTO forum_media_deletions") {
		t.Errorf("沒有檔案伺服器時不該寫入刪除佇列（佇列唯一的消費者就是檔案刪除）")
	}
}

// TestCovPruneMediaDeleteFailureKeepsQueue 釘住「檔案已刪、但 forum_media
// 紀錄刪不掉」時，佇列列必須留著。否則下一 tick 不會再看到這個檔名，
// 所有權紀錄就永遠留在表裡。
func TestCovPruneMediaDeleteFailureKeepsQueue(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1), "media.png"})))
	src.addQuery(purgeCovLockRule(1))
	src.addQuery(purgeCovRefExactRule(purgeCovRef(0)))
	src.addQuery(purgeCovRefLegacyRule(purgeCovRef(0)))
	src.addExec(purgeCovExecRule{substr: "DELETE FROM forum_media WHERE filename", err: errors.New("media locked")})

	deleter := &purgeCovDeleter{}
	removed, err := NewPruner(db, deleter, time.Hour, time.Hour).Prune(context.Background())
	if err != nil || removed != 1 {
		t.Fatalf("媒體紀錄刪除失敗不應影響 Prune 的結果，得到 (%d, %v)", removed, err)
	}
	if got := deleter.deletedNames(); !sameStringSet(got, "media.png") {
		t.Errorf("檔案仍應被刪除，got %v", got)
	}
	if got := src.queuedNames(); !sameStringSet(got, "media.png") {
		t.Errorf("forum_media 紀錄刪除失敗時佇列列必須保留，got %v", got)
	}
}

// TestCovPruneRetriesQueuedImageDeletion 是 M3 的回歸測試：第一次刪檔失敗，
// 第二次 Prune（即使沒有過期貼文）必須從佇列重試並收斂。
func TestCovPruneRetriesQueuedImageDeletion(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(
		purgeCovPosts([]purgeCovValue{int64(1), "retry.png"}),
		purgeCovPosts(),
	))
	src.addQuery(purgeCovLockRule(1))
	src.addQuery(purgeCovRefExactRule(purgeCovRef(0), purgeCovRef(0)))
	src.addQuery(purgeCovRefLegacyRule(purgeCovRef(0), purgeCovRef(0)))

	deleter := &purgeCovDeleter{fail: map[string]bool{"retry.png": true}}
	p := NewPruner(db, deleter, time.Hour, time.Hour)

	if removed, err := p.Prune(context.Background()); err != nil || removed != 1 {
		t.Fatalf("第一次 Prune = (%d, %v), want (1, nil)", removed, err)
	}
	if got := src.queuedNames(); !sameStringSet(got, "retry.png") {
		t.Fatalf("刪檔失敗後佇列應保留 retry.png，got %v", got)
	}
	if got := src.execArgs("DELETE FROM forum_media WHERE filename"); len(got) != 0 {
		t.Fatalf("刪檔失敗時不該刪 forum_media 紀錄，got %v", got)
	}

	deleter.setFail("retry.png", false)
	if removed, err := p.Prune(context.Background()); err != nil || removed != 0 {
		t.Fatalf("第二次 Prune = (%d, %v), want (0, nil)", removed, err)
	}
	if got := src.queuedNames(); len(got) != 0 {
		t.Errorf("重試成功後佇列應清空，got %v", got)
	}
	if got := src.execArgs("DELETE FROM forum_media WHERE filename"); len(got) != 1 {
		t.Errorf("重試成功後應刪一次 forum_media 紀錄，got %d", len(got))
	}
	if got := deleter.deletedNames(); !sameStringSet(got, "retry.png") {
		t.Errorf("成功刪除的檔案 = %v, want [retry.png]", got)
	}
}

// TestCovPruneQueueInsertFailureRollsBack 釘住「佇列寫入是交易的一部分」：
// 寫不進去時整筆交易必須回滾，不能留下「貼文已刪、檔名沒被記錄」的狀態。
func TestCovPruneQueueInsertFailureRollsBack(t *testing.T) {
	db, src := purgeCovDB(t)
	src.addQuery(purgeCovFetchRule(purgeCovPosts([]purgeCovValue{int64(1), "queue.png"})))
	src.addQuery(purgeCovLockRule(1))
	src.addExec(purgeCovExecRule{substr: "INSERT IGNORE INTO forum_media_deletions", err: errors.New("queue down")})

	deleter := &purgeCovDeleter{}
	if _, err := NewPruner(db, deleter, time.Hour, time.Hour).Prune(context.Background()); err == nil {
		t.Fatal("佇列寫入失敗時 Prune 必須回傳錯誤（否則貼文會被刪、檔名卻沒被記錄）")
	}
	if src.saw("DELETE FROM forum_post_comments") || src.saw("DELETE FROM forum_posts WHERE id IN") {
		t.Errorf("佇列寫入失敗後不應繼續刪除貼文")
	}
	if got := deleter.deletedNames(); len(got) != 0 {
		t.Errorf("佇列寫入失敗後不應嘗試刪檔，got %v", got)
	}
}

// sameStringSet 比較兩個字串集合（忽略順序）。檔案刪除是並行的，完成的
// 先後不該成為測試的斷言對象。
func sameStringSet(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	count := make(map[string]int, len(got))
	for _, value := range got {
		count[value]++
	}
	for _, value := range want {
		count[value]--
		if count[value] < 0 {
			return false
		}
	}
	return true
}

// purgeCovDeleter 可依檔名模擬檔案伺服器故障。
//
// 刪檔在正式程式裡是並行的，因此 deleted 的 append 必須加鎖；測試端在
// Prune 返回後才讀取，透過 WaitGroup 的 join 取得 happens-before。
type purgeCovDeleter struct {
	mu      sync.Mutex
	deleted []string
	fail    map[string]bool
}

func (d *purgeCovDeleter) DeleteFile(_ context.Context, name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fail[name] {
		return errors.New("file server down")
	}
	d.deleted = append(d.deleted, name)
	return nil
}

func (d *purgeCovDeleter) setFail(name string, fail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fail == nil {
		d.fail = make(map[string]bool)
	}
	d.fail[name] = fail
}

func (d *purgeCovDeleter) deletedNames() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.deleted...)
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
