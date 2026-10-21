/*
不需要真實 MySQL 的稽核查詢測試（backend/forum/audit/cov_query_test.go）。

audit_query_mysql_test.go 驗的是 SQL 對真實 MySQL 的效果；這個檔案改以一個
database/sql/driver 替身覆蓋「查詢成功、查詢失敗、掃描失敗、空結果、清理
批次」等分支，讓一般 `go test`（沒有 FORUM_TEST_MYSQL_DSN）也能走到
List / DistinctActions / DistinctActors / Pruner 的完整程式路徑。

替身與 httpapi/cov_harness_test.go 同一個形状：Query 沒登記就回空結果集、
Exec 預設成功。額外支援「依呼叫次序」的回應與可設定的 RowsAffected，因為
Prune 會用同一個 DELETE 連續批刪直到不足一批。
*/

package audit

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

type auditCovValue = driver.Value

type auditCovResponse struct {
	cols    []string
	rows    [][]auditCovValue
	err     error
	rowsErr error
}

type auditCovQueryRule struct {
	substr    string
	responses []auditCovResponse
	call      int
}

type auditCovExecRule struct {
	substr      string
	err         error
	affected    []int64
	affectedErr error
	call        int
}

type auditCovQueryCall struct {
	query string
	args  []any
}

type auditCovSource struct {
	mu      sync.Mutex
	queries []*auditCovQueryRule
	execs   []*auditCovExecRule
	seen    []string
	calls   []auditCovQueryCall
	execsN  int
}

func (s *auditCovSource) addQuery(rule *auditCovQueryRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, rule)
}

func (s *auditCovSource) addExec(rule *auditCovExecRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.execs = append(s.execs, rule)
}

func (s *auditCovSource) saw(substr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, query := range s.seen {
		if strings.Contains(query, substr) {
			return true
		}
	}
	return false
}

func (s *auditCovSource) callsFor(substr string) []auditCovQueryCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []auditCovQueryCall
	for _, call := range s.calls {
		if strings.Contains(call.query, substr) {
			out = append(out, call)
		}
	}
	return out
}

var (
	auditCovSeq      uint64
	auditCovRegistry sync.Map
	auditCovOnce     sync.Once
)

type auditCovDriver struct{}

func (auditCovDriver) Open(name string) (driver.Conn, error) {
	value, ok := auditCovRegistry.Load(name)
	if !ok {
		return nil, fmt.Errorf("audit cov driver: unknown dsn %q", name)
	}
	return &auditCovConn{src: value.(*auditCovSource)}, nil
}

type auditCovConn struct{ src *auditCovSource }

func (c *auditCovConn) Prepare(string) (driver.Stmt, error) { return auditCovStmt{}, nil }
func (c *auditCovConn) Close() error                        { return nil }
func (c *auditCovConn) Begin() (driver.Tx, error)           { return auditCovTx{}, nil }
func (c *auditCovConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return auditCovTx{}, nil
}
func (c *auditCovConn) Ping(context.Context) error { return nil }

func (c *auditCovConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)
	values := make([]any, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	c.src.calls = append(c.src.calls, auditCovQueryCall{query: query, args: values})
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
		return &auditCovRows{cols: response.cols, rows: response.rows, errAfter: response.rowsErr}, nil
	}
	return &auditCovRows{}, nil
}

func (c *auditCovConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.src.mu.Lock()
	defer c.src.mu.Unlock()
	c.src.seen = append(c.src.seen, query)
	c.src.execsN++
	for _, rule := range c.src.execs {
		if !strings.Contains(query, rule.substr) {
			continue
		}
		if rule.err != nil {
			return nil, rule.err
		}
		var affected int64 = 1
		if rule.call < len(rule.affected) {
			affected = rule.affected[rule.call]
		}
		rule.call++
		return auditCovResult{affected: affected, err: rule.affectedErr}, nil
	}
	return auditCovResult{affected: 1}, nil
}

type auditCovTx struct{}

func (auditCovTx) Commit() error   { return nil }
func (auditCovTx) Rollback() error { return nil }

type auditCovStmt struct{}

func (auditCovStmt) Close() error  { return nil }
func (auditCovStmt) NumInput() int { return -1 }
func (auditCovStmt) Exec([]auditCovValue) (driver.Result, error) {
	return auditCovResult{affected: 1}, nil
}
func (auditCovStmt) Query([]auditCovValue) (driver.Rows, error) { return &auditCovRows{}, nil }

type auditCovResult struct {
	affected int64
	err      error
}

func (r auditCovResult) LastInsertId() (int64, error) { return 1, nil }
func (r auditCovResult) RowsAffected() (int64, error) { return r.affected, r.err }

type auditCovRows struct {
	cols     []string
	rows     [][]auditCovValue
	errAfter error
	next     int
}

func (r *auditCovRows) Columns() []string { return r.cols }
func (r *auditCovRows) Close() error      { return nil }
func (r *auditCovRows) Next(dest []auditCovValue) error {
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

func auditCovDB(t *testing.T) (*sql.DB, *auditCovSource) {
	t.Helper()

	name := fmt.Sprintf("audit-cov-%d", atomic.AddUint64(&auditCovSeq, 1))
	src := &auditCovSource{}
	auditCovRegistry.Store(name, src)
	auditCovOnce.Do(func() { sql.Register("forum-audit-cov-stub", auditCovDriver{}) })

	db, err := sql.Open("forum-audit-cov-stub", name)
	if err != nil {
		t.Fatalf("sql.Open(cov audit): %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		auditCovRegistry.Delete(name)
	})
	return db, src
}

// auditCovEntryRow 依 List 的 SELECT 欄位順序組一列。
func auditCovEntryRow(id int64, actor, action, targetType, targetID, label, changes, ip, requestID, created string) []auditCovValue {
	return []auditCovValue{id, actor, action, targetType, targetID, label, changes, ip, requestID, created}
}

func auditCovEntryResponse(rows ...[]auditCovValue) auditCovResponse {
	return auditCovResponse{
		cols: []string{
			columnID, columnActorEmail, columnAction, columnTargetType, columnTargetID,
			columnTargetLabel, columnChanges, columnClientIP, columnRequestID, columnCreatedAt,
		},
		rows: rows,
	}
}

/* ==========================================================================
   List
   ========================================================================== */

func TestCovListSuccess(t *testing.T) {
	db, src := auditCovDB(t)
	src.addQuery(&auditCovQueryRule{
		substr:    "SELECT COUNT(*) FROM forum_admin_actions",
		responses: []auditCovResponse{{cols: []string{"n"}, rows: [][]auditCovValue{{int64(12)}}}},
	})
	src.addQuery(&auditCovQueryRule{
		substr: "ORDER BY created_at DESC",
		responses: []auditCovResponse{auditCovEntryResponse(
			auditCovEntryRow(2, "admin@example.com", "post.delete", "post", "9", "標題",
				`[{"field":"status","before":"A","after":"B"}]`, "203.0.113.9", "req-2", "2026-10-02 12:00:00"),
			auditCovEntryRow(1, "admin@example.com", "user.suspend", "user", "u@example.com", "",
				"", "", "", "2026-10-01 12:00:00"),
		)},
	})

	result, err := List(context.Background(), db, Filter{
		ActorEmail: " admin@example.com ",
		Action:     "post.delete, user.suspend",
		Limit:      25,
		Offset:     10,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 12 {
		t.Errorf("Total = %d, want 12", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("Items = %d, want 2", len(result.Items))
	}
	if result.Items[0].ID != 2 || result.Items[0].Changes[0].Field != "status" {
		t.Errorf("Items[0] = %+v, want id=2 且 changes 已反序列化", result.Items[0])
	}
	if result.Items[1].Changes != nil {
		t.Errorf("空 changes 應保持 nil, got %+v", result.Items[1].Changes)
	}

	// 分頁參數必須真的送進驅動（不是只存在 Filter 裡）。
	listCalls := src.callsFor("ORDER BY created_at DESC")
	if len(listCalls) != 1 {
		t.Fatalf("List 查詢次數 = %d, want 1", len(listCalls))
	}
	args := listCalls[0].args
	if len(args) != 5 {
		t.Fatalf("List 參數數量 = %d, want 5（2 個 action + actor + limit + offset）", len(args))
	}
	if args[0] != "admin@example.com" || args[1] != "post.delete" || args[2] != "user.suspend" ||
		args[3] != int64(25) || args[4] != int64(10) {
		t.Errorf("List 參數 = %v, want actor、兩個 action、limit=25、offset=10", args)
	}
	// 條件必須以佔位符送出，而不是字串拼接。
	if !strings.Contains(listCalls[0].query, "actor_email = ?") {
		t.Errorf("List 查詢沒有使用佔位符: %s", listCalls[0].query)
	}
}

func TestCovListEmpty(t *testing.T) {
	db, src := auditCovDB(t)
	src.addQuery(&auditCovQueryRule{
		substr:    "SELECT COUNT(*) FROM forum_admin_actions",
		responses: []auditCovResponse{{cols: []string{"n"}, rows: [][]auditCovValue{{int64(0)}}}},
	})
	src.addQuery(&auditCovQueryRule{
		substr:    "ORDER BY created_at DESC",
		responses: []auditCovResponse{auditCovEntryResponse()},
	})

	result, err := List(context.Background(), db, Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("空結果 = %+v, want Total=0 Items=[]", result)
	}
	if result.Items == nil {
		t.Error("Items 應為空切片而不是 nil（JSON 要序列化成 []）")
	}
}

func TestCovListErrors(t *testing.T) {
	t.Run("COUNT 失敗", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT COUNT(*) FROM forum_admin_actions",
			responses: []auditCovResponse{{err: errors.New("count failed")}},
		})
		if _, err := List(context.Background(), db, Filter{}); err == nil {
			t.Fatal("COUNT 失敗時應回傳錯誤")
		}
	})

	t.Run("列表查詢失敗", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT COUNT(*) FROM forum_admin_actions",
			responses: []auditCovResponse{{cols: []string{"n"}, rows: [][]auditCovValue{{int64(1)}}}},
		})
		src.addQuery(&auditCovQueryRule{
			substr:    "ORDER BY created_at DESC",
			responses: []auditCovResponse{{err: errors.New("list failed")}},
		})
		if _, err := List(context.Background(), db, Filter{}); err == nil {
			t.Fatal("列表查詢失敗時應回傳錯誤")
		}
	})

	t.Run("掃描失敗", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT COUNT(*) FROM forum_admin_actions",
			responses: []auditCovResponse{{cols: []string{"n"}, rows: [][]auditCovValue{{int64(1)}}}},
		})
		// id 欄位給字串：掃進 *int64 會失敗。
		src.addQuery(&auditCovQueryRule{
			substr:    "ORDER BY created_at DESC",
			responses: []auditCovResponse{auditCovEntryResponse([]auditCovValue{"not-a-number"})},
		})
		if _, err := List(context.Background(), db, Filter{}); err == nil {
			t.Fatal("掃描失敗時應回傳錯誤")
		}
	})

	t.Run("rows.Err", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT COUNT(*) FROM forum_admin_actions",
			responses: []auditCovResponse{{cols: []string{"n"}, rows: [][]auditCovValue{{int64(1)}}}},
		})
		src.addQuery(&auditCovQueryRule{
			substr: "ORDER BY created_at DESC",
			responses: []auditCovResponse{{
				cols:    []string{columnID},
				rowsErr: errors.New("stream broken"),
			}},
		})
		if _, err := List(context.Background(), db, Filter{}); err == nil {
			t.Fatal("rows.Err 非 nil 時應回傳錯誤")
		}
	})
}

/* ==========================================================================
   DistinctActions / DistinctActors
   ========================================================================== */

func TestCovDistinctSuccess(t *testing.T) {
	db, src := auditCovDB(t)
	src.addQuery(&auditCovQueryRule{
		substr: "SELECT DISTINCT action",
		responses: []auditCovResponse{{
			cols: []string{columnAction},
			rows: [][]auditCovValue{{"post.delete"}, {"user.suspend"}},
		}},
	})
	src.addQuery(&auditCovQueryRule{
		substr: "SELECT DISTINCT actor_email",
		responses: []auditCovResponse{{
			cols: []string{columnActorEmail},
			rows: [][]auditCovValue{{"alice@example.com"}, {"bob@example.com"}},
		}},
	})

	actions, err := DistinctActions(context.Background(), db)
	if err != nil {
		t.Fatalf("DistinctActions: %v", err)
	}
	if len(actions) != 2 || actions[0] != "post.delete" || actions[1] != "user.suspend" {
		t.Errorf("actions = %v", actions)
	}
	actors, err := DistinctActors(context.Background(), db)
	if err != nil {
		t.Fatalf("DistinctActors: %v", err)
	}
	if len(actors) != 2 || actors[0] != "alice@example.com" {
		t.Errorf("actors = %v", actors)
	}

	// 多取一筆作為截斷判斷：LIMIT 必須是 maxFacets+1。
	if calls := src.callsFor("SELECT DISTINCT action"); len(calls) != 1 || len(calls[0].args) != 1 || calls[0].args[0] != int64(maxFacets+1) {
		t.Errorf("DistinctActions 的 LIMIT 參數不對: %+v", calls)
	}
}

func TestCovDistinctTruncatesAndWarns(t *testing.T) {
	db, src := auditCovDB(t)

	makeRows := func(value func(i int) string) [][]auditCovValue {
		rows := make([][]auditCovValue, 0, maxFacets+1)
		for i := 0; i < maxFacets+1; i++ {
			rows = append(rows, []auditCovValue{value(i)})
		}
		return rows
	}
	src.addQuery(&auditCovQueryRule{
		substr:    "SELECT DISTINCT action",
		responses: []auditCovResponse{{cols: []string{columnAction}, rows: makeRows(func(i int) string { return fmt.Sprintf("action.%04d", i) })}},
	})
	src.addQuery(&auditCovQueryRule{
		substr:    "SELECT DISTINCT actor_email",
		responses: []auditCovResponse{{cols: []string{columnActorEmail}, rows: makeRows(func(i int) string { return fmt.Sprintf("user%04d@example.com", i) })}},
	})

	var mu sync.Mutex
	var warnings []string
	orig := logPrintf
	logPrintf = func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	t.Cleanup(func() { logPrintf = orig })

	actions, err := DistinctActions(context.Background(), db)
	if err != nil {
		t.Fatalf("DistinctActions: %v", err)
	}
	actors, err := DistinctActors(context.Background(), db)
	if err != nil {
		t.Fatalf("DistinctActors: %v", err)
	}
	if len(actions) != maxFacets || len(actors) != maxFacets {
		t.Fatalf("截斷後長度 = (%d, %d), want (%d, %d)", len(actions), len(actors), maxFacets, maxFacets)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(warnings) != 2 {
		t.Errorf("截斷警告 = %v, want 2 行（動作與操作者各一）", warnings)
	}
}

func TestCovDistinctErrors(t *testing.T) {
	t.Run("查詢失敗", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addQuery(&auditCovQueryRule{substr: "SELECT DISTINCT action", responses: []auditCovResponse{{err: errors.New("boom")}}})
		src.addQuery(&auditCovQueryRule{substr: "SELECT DISTINCT actor_email", responses: []auditCovResponse{{err: errors.New("boom")}}})
		if _, err := DistinctActions(context.Background(), db); err == nil {
			t.Error("DistinctActions 查詢失敗時應回傳錯誤")
		}
		if _, err := DistinctActors(context.Background(), db); err == nil {
			t.Error("DistinctActors 查詢失敗時應回傳錯誤")
		}
	})

	t.Run("掃描失敗", func(t *testing.T) {
		db, src := auditCovDB(t)
		// 空資料列：dest 保持 nil，掃進 *string 會失敗。
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT DISTINCT action",
			responses: []auditCovResponse{{cols: []string{columnAction}, rows: [][]auditCovValue{{}}}},
		})
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT DISTINCT actor_email",
			responses: []auditCovResponse{{cols: []string{columnActorEmail}, rows: [][]auditCovValue{{}}}},
		})
		if _, err := DistinctActions(context.Background(), db); err == nil {
			t.Error("DistinctActions 掃描失敗時應回傳錯誤")
		}
		if _, err := DistinctActors(context.Background(), db); err == nil {
			t.Error("DistinctActors 掃描失敗時應回傳錯誤")
		}
	})

	t.Run("rows.Err", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT DISTINCT action",
			responses: []auditCovResponse{{cols: []string{columnAction}, rowsErr: errors.New("broken")}},
		})
		src.addQuery(&auditCovQueryRule{
			substr:    "SELECT DISTINCT actor_email",
			responses: []auditCovResponse{{cols: []string{columnActorEmail}, rowsErr: errors.New("broken")}},
		})
		if _, err := DistinctActions(context.Background(), db); err == nil {
			t.Error("DistinctActions rows.Err 時應回傳錯誤")
		}
		if _, err := DistinctActors(context.Background(), db); err == nil {
			t.Error("DistinctActors rows.Err 時應回傳錯誤")
		}
	})
}

/* ==========================================================================
   Pruner / Run
   ========================================================================== */

func TestCovPruneBatchesUntilShort(t *testing.T) {
	db, src := auditCovDB(t)
	src.addExec(&auditCovExecRule{
		substr:   "DELETE FROM forum_admin_actions",
		affected: []int64{5000, 7},
	})

	removed := NewPruner(db, 90*24*time.Hour, time.Hour).Prune(context.Background())
	if removed != 5007 {
		t.Errorf("removed = %d, want 5007（滿批 5000 後再刪 7 筆）", removed)
	}
	if !src.saw("LIMIT ?") {
		t.Errorf("Prune 應使用 DELETE … LIMIT ?")
	}
}

func TestCovPruneErrorBranches(t *testing.T) {
	t.Run("DELETE 失敗", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addExec(&auditCovExecRule{substr: "DELETE FROM forum_admin_actions", err: errors.New("locked")})
		if got := NewPruner(db, time.Hour, time.Hour).Prune(context.Background()); got != 0 {
			t.Errorf("DELETE 失敗時應回 0, got %d", got)
		}
	})

	t.Run("RowsAffected 失敗", func(t *testing.T) {
		db, src := auditCovDB(t)
		src.addExec(&auditCovExecRule{
			substr:      "DELETE FROM forum_admin_actions",
			affectedErr: errors.New("driver lost count"),
		})
		if got := NewPruner(db, time.Hour, time.Hour).Prune(context.Background()); got != 0 {
			t.Errorf("RowsAffected 失敗時應回 0, got %d", got)
		}
	})
}

func TestCovRunStopsOnCanceledContext(t *testing.T) {
	db, _ := auditCovDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消：Run 建立 ticker 後應立刻從 ctx.Done 返回
	NewPruner(db, time.Hour, time.Hour).Run(ctx)
}
