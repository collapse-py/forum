/*
purge.Run 的真實 MySQL 測試（backend/forum/purge/purge_run_mysql_test.go）。

TestPruneMySQL 驗「清理一次」的正確性；這個檔驗「排程真的會定期清理」——
兩者的失效模式不同：Run 的 ticker 若沒被啟動（例如生命週期寫錯），單次
Prune 的測試永遠全綠，而過期貼文會一直留在資料庫裡。

約定同 purge_test.go：FORUM_TEST_MYSQL_DSN 未設定時 skip。
*/

package purge

import (
	"context"
	"testing"
	"time"
)

func TestRunPrunesOnTickerMySQL(t *testing.T) {
	db := purgeTestDB(t)

	old := time.Now().Add(-100 * 24 * time.Hour)
	id := insertPost(db, "ticker@example.com", "過期的貼文", "", &old)
	if id == 0 {
		t.Fatal("無法插入過期貼文")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := NewPruner(db, &recordingDeleter{}, 90*24*time.Hour, 20*time.Millisecond)
	done := make(chan struct{})
	go func() { p.Run(ctx); close(done) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var remaining int
		if err := db.QueryRow(`SELECT COUNT(*) FROM forum_posts WHERE id = ?`, id).Scan(&remaining); err != nil {
			t.Fatalf("查詢貼文失敗: %v", err)
		}
		if remaining == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("等待 Run 的定期清理超時（ticker 沒有運作？）")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ctx 取消後 Run 沒有返回")
	}
}
