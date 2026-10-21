/*
沒有資料庫時的行為（metrics/cov_metrics_test.go）。

這三條路徑是「監控是診斷工具，不該讓服務起不來」的實際落點：nil 資料庫
必須得到明確錯誤或 0，而不是 panic。既有的 metrics_flush_mysql_test.go 需要
真實 MySQL，因此這三個分支在一般 `go test ./...` 裡一直是 0%。
*/

package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCovMetricsNilDatabase(t *testing.T) {
	registry := New(Options{WindowMinutes: 10, MaxRoutes: 10})
	if registry == nil {
		t.Fatal("New 回傳 nil")
	}

	// FlushPending 沒有資料庫時回 0，呼叫端不必區分「沒有待寫入」與「沒有資料庫」。
	if written := registry.FlushPending(context.Background(), nil); written != 0 {
		t.Fatalf("FlushPending(nil) = %d, want 0", written)
	}

	// StartFlusher 必須在啟動 goroutine 前就失敗。
	if err := registry.StartFlusher(context.Background(), FlusherOptions{}); !errors.Is(err, ErrNoDatabase) {
		t.Fatalf("StartFlusher(no db) = %v, want ErrNoDatabase", err)
	}
	// 間隔與保留期為 0 時使用預設值；這條路徑仍然因為沒有 DB 而立即返回。
	if err := registry.StartFlusher(context.Background(), FlusherOptions{Interval: -1, RetentionHours: -1}); !errors.Is(err, ErrNoDatabase) {
		t.Fatalf("StartFlusher(no db, defaults) = %v, want ErrNoDatabase", err)
	}

	// LoadHistory 同樣回明確錯誤，而不是對 nil 呼叫 QueryContext。
	if loaded, err := registry.LoadHistory(context.Background(), nil, time.Now()); loaded != 0 || !errors.Is(err, ErrNoDatabase) {
		t.Fatalf("LoadHistory(nil) = %d, %v", loaded, err)
	}
}
