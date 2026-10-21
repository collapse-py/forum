/*
metrics 持久化的真實 MySQL 測試（backend/forum/metrics/metrics_flush_mysql_test.go）。

FlushPending / LoadHistory / StartFlusher 的正確性是「寫進 forum_request_metrics
的列到底對不對」—— sqlmock 式替身只能回傳預先寫好的答案，回答不了
「INSERT 之後 SELECT 得到什麼」。因此這裡與 data/migration_test.go、
purge/purge_test.go 同一個約定：FORUM_TEST_MYSQL_DSN 指向伺服器，
每個測試自己建立與丟棄拋棄式資料庫；未設定時 skip。

驗的是四件事：

 1. 只寫「已結束的分鐘」：當前分鐘的桶絕不寫出（它的值還沒定型），而寫過的
    桶被標記 flushed、第二次呼叫不再重寫 —— 重寫會讓累加語意把同一分鐘算成
    兩倍。
 2. 寫入失敗時還原 flushed：下一次呼叫必須重試同一分鐘。少了這段，資料庫的
    短暫故障會讓監控圖永久缺一段。
 3. LoadHistory 讀回的桶永遠不被 FlushPending 寫回：它們已經在資料庫裡，
    再寫一次就是重複計算。
 4. StartFlusher 的三個性質：定期寫出、過期列被 pruneMetrics 刪除、ctx 取消時
    離開前再寫一次（main 的優雅停止依賴這條路徑）。
*/

package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var metricsDBCounter atomic.Int64

// safeClock 是可併發推進的時鐘。fixedClock（metrics_test.go）的 current 是
// 無鎖欄位，而 StartFlusher 的背景 goroutine 會同時讀它 —— 在 -race 下
// 會被抓（CI 的 backend job 開 -race），因此 flusher 測試用這個。
type safeClock struct {
	mu  sync.Mutex
	cur time.Time
}

func (c *safeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cur
}

func (c *safeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cur = c.cur.Add(d)
}

// metricsServerDSN 把 FORUM_TEST_MYSQL_DSN 剝成「只連伺服器」的形式，未設定時 skip。
// 剝法與 data/migration_test.go 的 serverDSN 相同（庫名清空、參數保留）。
func metricsServerDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("FORUM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FORUM_TEST_MYSQL_DSN 未設定：跳過需要真實 MySQL 的 metrics 持久化測試。" +
			"設定方式見 .github/workflows/ci.yml 的 migrations job")
	}
	cleaned := dsn
	if i := strings.LastIndex(cleaned, "/"); i >= 0 {
		if j := strings.Index(cleaned[i:], "?"); j >= 0 {
			cleaned = cleaned[:i] + "/" + cleaned[i+j:]
		} else {
			cleaned = cleaned[:i] + "/"
		}
	}
	return cleaned
}

// metricsTestDB 建立含 forum_request_metrics 的拋棄式資料庫。
// schema 直接內聯對齊 MigrateMySQL 第 22 步的 DDL：這組測試只需要那一張表，
// 跑完整遷移會讓每次測試多等半分鐘，而這不是遷移測試（那是 data 套件的職責）。
func metricsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	serverLevel := metricsServerDSN(t)

	server, err := sql.Open("mysql", serverLevel)
	if err != nil {
		t.Fatalf("開啟伺服器連線失敗: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if err := server.Ping(); err != nil {
		t.Fatalf("MySQL 連不上: %v", err)
	}

	name := fmt.Sprintf("forum_metrics_test_%d_%d", time.Now().UnixNano(), metricsDBCounter.Add(1))
	if _, err := server.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("建立測試資料庫失敗: %v", err)
	}
	t.Cleanup(func() {
		if _, err := server.Exec("DROP DATABASE IF EXISTS `" + name + "`"); err != nil {
			t.Logf("清理測試資料庫 %s 失敗（請手動清理）: %v", name, err)
		}
	})

	base, params := serverLevel, ""
	if q := strings.Index(serverLevel, "?"); q >= 0 {
		base, params = serverLevel[:q], serverLevel[q:]
	}
	db, err := sql.Open("mysql", strings.TrimSuffix(base, "/")+"/"+name+params)
	if err != nil {
		t.Fatalf("連到測試資料庫失敗: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	metricsCreateTable(t, db)
	return db
}

// metricsCreateTable 建立 forum_request_metrics（對齊 MigrateMySQL 第 22 步）。
func metricsCreateTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS forum_request_metrics (
		bucket_minute DATETIME NOT NULL,
		total BIGINT NOT NULL DEFAULT 0,
		client_errors BIGINT NOT NULL DEFAULT 0,
		server_errors BIGINT NOT NULL DEFAULT 0,
		duration_sum_ms BIGINT NOT NULL DEFAULT 0,
		PRIMARY KEY (bucket_minute)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		t.Fatalf("建立 forum_request_metrics 失敗: %v", err)
	}
}

// bucketRow 讀回單一分鐘的彙總列。
func bucketRow(t *testing.T, db *sql.DB, minute time.Time) (total, clientErrors, serverErrors, durationSum int64, ok bool) {
	t.Helper()
	err := db.QueryRow(`SELECT total, client_errors, server_errors, duration_sum_ms
		FROM forum_request_metrics WHERE bucket_minute = ?`, minute.UTC()).
		Scan(&total, &clientErrors, &serverErrors, &durationSum)
	if err == sql.ErrNoRows {
		return 0, 0, 0, 0, false
	}
	if err != nil {
		t.Fatalf("查詢彙總列失敗: %v", err)
	}
	return total, clientErrors, serverErrors, durationSum, true
}

/* ==========================================================================
   FlushPending
   ========================================================================== */

// 只寫已結束的分鐘，且每個桶只寫一次。
func TestFlushPendingWritesOnlyClosedMinutesOnceMySQL(t *testing.T) {
	db := metricsTestDB(t)
	registry, clock := newTestRegistry(t, 30)

	// 當前分鐘：三筆（2xx/4xx/5xx 各一），但分鐘還沒結束 —— 不得寫出。
	registry.Observe("GET", "/forum", 200, 10*time.Millisecond)
	registry.Observe("GET", "/forum", 404, 40*time.Millisecond)
	registry.Observe("GET", "/forum", 500, 5*time.Millisecond)

	if got := registry.FlushPending(context.Background(), db); got != 0 {
		t.Fatalf("當前分鐘的 FlushPending 寫出了 %d 個桶, want 0", got)
	}

	// 推進一分鐘：上一分鐘結束，可以寫出。
	clock.advance(time.Minute)
	closed := minuteTime(minuteIndex(clock.current.Add(-time.Minute)))

	if got := registry.FlushPending(context.Background(), db); got != 1 {
		t.Fatalf("寫出的桶數 = %d, want 1", got)
	}
	total, clientErrs, serverErrs, durSum, ok := bucketRow(t, db, closed)
	if !ok {
		t.Fatal("已結束的分鐘沒有寫進 forum_request_metrics")
	}
	if total != 3 || clientErrs != 1 || serverErrs != 1 {
		t.Errorf("計數 = (total=%d, client=%d, server=%d), want (3,1,1)", total, clientErrs, serverErrs)
	}
	if durSum != 55 { // 10ms + 40ms + 5ms
		t.Errorf("duration_sum_ms = %d, want 55", durSum)
	}

	// 第二次呼叫：同一分鐘不得重寫（累加語意會讓它變成 6）。
	if got := registry.FlushPending(context.Background(), db); got != 0 {
		t.Fatalf("重複 FlushPending 又寫出 %d 個桶, want 0（同一分鐘只寫一次）", got)
	}
	if total, _, _, _, _ := bucketRow(t, db, closed); total != 3 {
		t.Errorf("重複呼叫後 total = %d, want 3（重寫會讓累加語意翻倍）", total)
	}
}

// 寫入失敗（表不存在）時不得標記 flushed：下一次呼叫必須重試同一分鐘。
func TestFlushPendingRetriesAfterWriteFailureMySQL(t *testing.T) {
	db := metricsTestDB(t)
	registry, clock := newTestRegistry(t, 30)

	registry.Observe("GET", "/forum", 200, 5*time.Millisecond)
	clock.advance(time.Minute)
	closed := minuteTime(minuteIndex(clock.current.Add(-time.Minute)))

	// 模擬資料庫故障：把表刪掉，INSERT 一定失敗。
	if _, err := db.Exec(`DROP TABLE forum_request_metrics`); err != nil {
		t.Fatalf("刪表失敗: %v", err)
	}
	if got := registry.FlushPending(context.Background(), db); got != 0 {
		t.Fatalf("表不存在時寫出了 %d 個桶, want 0", got)
	}

	// 故障修復（表回來了）：同一分鐘必須被重試寫出，而不是被永久跳過。
	metricsCreateTable(t, db)
	if got := registry.FlushPending(context.Background(), db); got != 1 {
		t.Fatalf("故障修復後寫出 %d 個桶, want 1（失敗的分鐘必須重試）", got)
	}
	if total, _, _, _, ok := bucketRow(t, db, closed); !ok || total != 1 {
		t.Errorf("重試後的列 = (ok=%v, total=%d), want (true, 1)", ok, total)
	}
}

/* ==========================================================================
   LoadHistory
   ========================================================================== */

// 讀回的歷史桶進 history、出現在快照的 HistoryLoaded，而且**永遠不被寫回**。
func TestLoadHistoryIsNeverRewrittenMySQL(t *testing.T) {
	db := metricsTestDB(t)
	registry, _ := newTestRegistry(t, 120)

	past := minuteTime(minuteIndex(time.Now().UTC().Add(-2 * time.Hour)))
	for i := 0; i < 3; i++ {
		if _, err := db.Exec(`INSERT INTO forum_request_metrics
			(bucket_minute, total, client_errors, server_errors, duration_sum_ms)
			VALUES (?, ?, 0, 0, ?)`,
			past.Add(time.Duration(i)*time.Minute), int64(10+i), int64(100+i)); err != nil {
			t.Fatalf("準備歷史列失敗: %v", err)
		}
	}

	loaded, err := registry.LoadHistory(context.Background(), db, time.Now().UTC().Add(-3*time.Hour))
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if loaded != 3 {
		t.Fatalf("loaded = %d, want 3", loaded)
	}
	if !registry.Snapshot(1).HistoryLoaded {
		t.Error("HistoryLoaded = false：時間軸會把重啟前的歷史顯示成空")
	}

	// FlushPending 不得把讀回的桶寫回資料庫（它們已經在裡面）。
	if got := registry.FlushPending(context.Background(), db); got != 0 {
		t.Errorf("FlushPending 寫出了 %d 個歷史桶, want 0（重寫等於重複計算）", got)
	}

	// 值本身也要對：LoadHistory 掃的是同一張表的同一列。
	total, _, _, durSum, ok := bucketRow(t, db, past)
	if !ok || total != 10 || durSum != 100 {
		t.Errorf("歷史列被改變了: (ok=%v, total=%d, durSum=%d), want (true,10,100)", ok, total, durSum)
	}
}

func TestLoadHistoryNilDatabase(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	if _, err := registry.LoadHistory(context.Background(), nil, time.Now()); err != ErrNoDatabase {
		t.Errorf("nil db 的 LoadHistory = %v, want ErrNoDatabase", err)
	}
}

/* ==========================================================================
   StartFlusher
   ========================================================================== */

// StartFlusher 的完整生命週期：定期寫出、過期列被清除、ctx 取消時離開前再寫一次。
func TestStartFlusherFlushesPrunesAndStopsMySQL(t *testing.T) {
	db := metricsTestDB(t)
	clock := &safeClock{cur: time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)}
	registry := New(Options{WindowMinutes: 30, MaxRoutes: 8, Now: clock.Now})

	// 過期列：早於保留期（24h）的 cutoff，第一個 tick 就該被 pruneMetrics 刪掉。
	staleMinute := minuteTime(minuteIndex(clock.Now().Add(-48 * time.Hour)))
	if _, err := db.Exec(`INSERT INTO forum_request_metrics
		(bucket_minute, total, client_errors, server_errors, duration_sum_ms)
		VALUES (?, 1, 0, 0, 0)`, staleMinute); err != nil {
		t.Fatalf("準備過期列失敗: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := registry.StartFlusher(ctx, FlusherOptions{DB: db, Interval: 20 * time.Millisecond, RetentionHours: 24}); err != nil {
		t.Fatalf("StartFlusher: %v", err)
	}

	// 第一次定期寫入：觀察一個過去分鐘的請求，等 tick 把它寫出去。
	registry.Observe("GET", "/forum", 200, 15*time.Millisecond)
	clock.Advance(2 * time.Minute)
	firstMinute := minuteTime(minuteIndex(clock.Now().Add(-2 * time.Minute)))

	waitFor := func(desc string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if cond() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("等待 %s 超時", desc)
	}

	waitFor("定期寫入", func() bool {
		_, _, _, _, ok := bucketRow(t, db, firstMinute)
		return ok
	})
	waitFor("過期列被清除", func() bool {
		_, _, _, _, ok := bucketRow(t, db, staleMinute)
		return !ok
	})

	// 取消 ctx：goroutine 離開前要把最後一個已結束的分鐘寫出去
	//（main 的優雅停止依賴這條路徑）。
	registry.Observe("GET", "/forum", 200, 15*time.Millisecond)
	clock.Advance(2 * time.Minute)
	lastMinute := minuteTime(minuteIndex(clock.Now().Add(-2 * time.Minute)))
	cancel()

	waitFor("取消時的最後一次寫入", func() bool {
		_, _, _, _, ok := bucketRow(t, db, lastMinute)
		return ok
	})
}

// DB 為 nil 時 StartFlusher 必須拒絕啟動（ErrNoDatabase），而不是啟動一個
// 每個 tick 都失敗的 goroutine。
func TestStartFlusherRequiresDatabase(t *testing.T) {
	registry, _ := newTestRegistry(t, 30)
	if err := registry.StartFlusher(context.Background(), FlusherOptions{}); err != ErrNoDatabase {
		t.Errorf("nil db 的 StartFlusher = %v, want ErrNoDatabase", err)
	}
}
