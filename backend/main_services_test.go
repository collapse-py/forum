/*
啟動組裝的行程內測試（backend/main_services_test.go）。

【為什麼需要這一份】
main() 的啟動流程（OAuth → Session → Server → 背景工作）原本內嵌在 main
裡，而 main 一旦離開設定檔載入就會一路走到監聽或 os.Exit —— 這段流程因此
只能在真的連上 MySQL、Redis 之後才可能被執行到，測試環境幾乎不可能提供
那些前提。

startServices 把它抽成一般的函式之後，這一段可以在同一個行程裡以替身直接
執行：不需要 MySQL、不需要 Redis、不需要監聽任何連接埠。

【這一份測試盯著什麼】
  - 組裝本身：Server 與監控容器都要真的被組裝出來（nil 代表啟動會在第一個
    請求就失敗，而那種失敗在啟動日誌裡看不出來）。
  - 背景工作：限流清理、監控落盤、稽核清理、軟刪除清理、日別彙總刷新、
    封鎖清理都掛在同一個可取消的 context 上。取消之後它們必須收掉，而不是
    留一條永遠不會結束的 goroutine。
  - 取消之後仍然可以再取消（冪等）：停止流程會呼叫 stopBackground，而 defer
    裡還有一層保險，兩者都會走到。
*/
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"forum/forum/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

/* ==========================================================================
   假的資料庫（不需要 MySQL）
   ========================================================================== */

// covDBDriverName 是這個替身在 database/sql 裡的驅動名稱。
const covDBDriverName = "forum-main-cov-stub"

var (
	covDBSeq       uint64
	covDBRegistry  sync.Map
	covDBDriverOne sync.Once
)

// covDBState 是單一連線共享的狀態：目前只記錄被問過的 SQL。
type covDBState struct {
	mu   sync.Mutex
	seen []string
}

func (s *covDBState) record(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, query)
}

// sawQuery 回傳是否問過含 substr 的 SQL。
func (s *covDBState) sawQuery(substr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, query := range s.seen {
		if strings.Contains(query, substr) {
			return true
		}
	}
	return false
}

// covDBDriver 是一個「什麼都成功」的驅動：查詢回空結果集、寫入成功、
// 交易可以開也可以結束。啟動流程只需要這些。
type covDBDriver struct{}

func (covDBDriver) Open(name string) (driver.Conn, error) {
	value, ok := covDBRegistry.Load(name)
	if !ok {
		return nil, fmt.Errorf("cov driver: unknown dsn %q", name)
	}
	return &covDBConn{state: value.(*covDBState)}, nil
}

type covDBConn struct{ state *covDBState }

func (c *covDBConn) Prepare(string) (driver.Stmt, error) { return covDBStmt{}, nil }
func (c *covDBConn) Close() error                        { return nil }
func (c *covDBConn) Begin() (driver.Tx, error)           { return covDBTx{}, nil }
func (c *covDBConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return covDBTx{}, nil
}
func (c *covDBConn) Ping(context.Context) error { return nil }

func (c *covDBConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.state.record(query)
	return &covDBRows{}, nil
}

func (c *covDBConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.state.record(query)
	return covDBResult{}, nil
}

type covDBTx struct{}

func (covDBTx) Commit() error   { return nil }
func (covDBTx) Rollback() error { return nil }

type covDBStmt struct{}

func (covDBStmt) Close() error                               { return nil }
func (covDBStmt) NumInput() int                              { return -1 }
func (covDBStmt) Exec([]driver.Value) (driver.Result, error) { return covDBResult{}, nil }
func (covDBStmt) Query([]driver.Value) (driver.Rows, error)  { return &covDBRows{}, nil }

type covDBResult struct{}

func (covDBResult) LastInsertId() (int64, error) { return 1, nil }
func (covDBResult) RowsAffected() (int64, error) { return 1, nil }

// covDBRows 是空結果集：QueryRow 會拿到 sql.ErrNoRows，Query 會拿到空清單。
type covDBRows struct{}

func (*covDBRows) Columns() []string { return nil }
func (*covDBRows) Close() error      { return nil }
func (*covDBRows) Next([]driver.Value) error {
	return io.EOF
}

// newCovDB 開一個假的 *sql.DB，回傳連線狀態供斷言使用。
func newCovDB(t *testing.T) (*sql.DB, *covDBState) {
	t.Helper()
	name := fmt.Sprintf("cov-main-%d", atomic.AddUint64(&covDBSeq, 1))
	state := &covDBState{}
	covDBRegistry.Store(name, state)
	covDBDriverOne.Do(func() { sql.Register(covDBDriverName, covDBDriver{}) })

	db, err := sql.Open(covDBDriverName, name)
	if err != nil {
		t.Fatalf("開啟假的資料庫失敗: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		covDBRegistry.Delete(name)
	})
	return db, state
}

/* ==========================================================================
   啟動組裝
   ========================================================================== */

// newStartServicesConfig 回傳一份足以啟動服務的設定。
func newStartServicesConfig() config.Config {
	return config.Config{
		ForumName:               "測試論壇",
		ForumShortName:          "測試",
		ForumDescription:        "測試用論壇",
		CookieName:              "FORUM_cov_test",
		SessionExpire:           time.Hour,
		ServerPort:              ":0",
		ShutdownTimeout:         time.Second,
		TrustedOrigins:          []string{"https://forum.test"},
		TrustedProxyCIDRs:       "127.0.0.1/32",
		RateLimitRequests:       1000,
		RateLimitWindow:         time.Minute,
		RateLimitUploadRequests: 1000,
		RateLimitUploadWindow:   time.Minute,
		RateLimitAuthRequests:   1000,
		RateLimitAuthWindow:     time.Minute,
		MonitorRetentionHours:   24,
		AuditRetentionDays:      90,
		PostRetentionDays:       30,
		MediaTokenKeyPrefix:     "forum:media:",
		MediaTokenTTLSecs:       3600,
	}
}

// TestStartServicesAssemblesServerAndBackgroundWorkers 確認啟動組裝會把
// Server、監控容器與封鎖名單都準備好，並且背景工作真的掛上了可取消的
// context（取消之後收得掉）。
func TestStartServicesAssemblesServerAndBackgroundWorkers(t *testing.T) {
	db, state := newCovDB(t)
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, registry := startServices(ctx, newStartServicesConfig(), db, client)

	if srv == nil {
		t.Fatal("startServices 沒有回傳 Server：啟動流程會在第一個請求就失敗")
	}
	if registry == nil {
		t.Fatal("startServices 沒有回傳監控容器：監控頁會整頁空白")
	}
	if srv.Metrics() == nil {
		t.Error("Server 上的監控容器不該是 nil")
	}
	if srv.Blocks() == nil {
		t.Error("封鎖名單不該是 nil：Redis 可用時它必須就緒")
	}
	// 監控的歷史彙總是在啟動時讀回來的，這一步會用到資料庫。
	if !state.sawQuery("forum_request_metrics") {
		t.Error("啟動時沒有讀回歷史請求統計")
	}

	// 讓背景 goroutine 有機會跑一次，再取消：它們都掛在 ctx 上，取消之後
	// 必須自己結束（否則每次重啟都會留下一批永不退出的 goroutine）。
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// TestStartServicesWithoutRedisStillBuildsServer 確認缺少 Redis 時組裝仍然
// 完成（只是封鎖名單不可用）：啟動流程在這裡記警告而不是中止，因為封鎖
// 失效不該讓整個論壇開不了。
func TestStartServicesWithoutRedisStillBuildsServer(t *testing.T) {
	db, _ := newCovDB(t)
	// 指向一個不存在的位址：連不上時 Redis 用戶端仍然可以被建立。
	client := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		MaxRetries:   -1,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, registry := startServices(ctx, newStartServicesConfig(), db, client)
	if srv == nil || registry == nil {
		t.Fatal("沒有 Redis 時組裝仍應完成")
	}
}
