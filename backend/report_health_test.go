/*
reportHealth（-healthz）的測試（backend/report_health_test.go）。

reportHealth 是容器 healthcheck 的實作。它的每一個分支都會被部署端直接
解讀成「要不要重啟容器」，因此這裡逐條釘住回傳值的語意：

  - 設定檔讀不到 → 錯誤（不是「健康」）。
  - DB_DSN 留空 → 跳過 MySQL 探測，只印一行說明（不是錯誤）。
  - DSN 解析失敗 → 錯誤，且訊息說明是 DSN 本身不能用。
  - MySQL Ping 失敗 → 錯誤。
  - Redis Ping 失敗 → 錯誤。
  - 兩個探測都成功 → nil。

「跳過」與「失敗」刻意分開：把未設定的相依報成不健康會讓一個純快取式
部署永遠被重啟，而把失敗報成跳過則會讓壞掉的相依永遠看起來正常。
*/
package main

import (
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

// TestReportHealthFailsOnMissingConfig 確認設定檔層級的問題是錯誤：
// 沒有設定檔就無從得知要探測哪裡，不能回 nil。
func TestReportHealthFailsOnMissingConfig(t *testing.T) {
	err := reportHealth(t.TempDir() + "/no-such.conf")
	if err == nil {
		t.Fatal("設定檔不存在時 reportHealth 應回錯誤")
	}
	if !strings.Contains(err.Error(), "設定檔不可用") {
		t.Errorf("錯誤訊息 = %v, want 指出設定檔不可用", err)
	}
}

// TestReportHealthSkipsMySQLAndReportsRedisOK 確認兩件事：
//
//  1. DB_DSN 留空是「無法驗證」而不是「不健康」，因此整體仍回 nil。
//  2. Redis 可達時印出 OK 並回 nil。
//
// Redis 用 miniredis（行程內的測試替身）而不是真的伺服器：這個測試要驗的
// 是 reportHealth 對「Ping 成功」的反應，不是 Redis 本身。
func TestReportHealthSkipsMySQLAndReportsRedisOK(t *testing.T) {
	mini := miniredis.RunT(t)
	cfgPath := writeConfigFile(t, "DB_DSN=\nREDIS_ADDR="+mini.Addr()+"\n")

	var err error
	out := captureStdout(t, func() {
		err = reportHealth(cfgPath)
	})
	if err != nil {
		t.Fatalf("相依可用時 reportHealth = %v, want nil", err)
	}
	if !strings.Contains(out, "[HEALTH] MySQL: 未設定 DSN") {
		t.Errorf("沒有 MySQL 時應印出跳過探測，得到:\n%s", out)
	}
	if !strings.Contains(out, "[HEALTH] Redis: OK") {
		t.Errorf("Redis 可達時應印出 OK，得到:\n%s", out)
	}
}

// TestReportHealthRejectsUnparsableDSN 確認 DSN 語法錯誤會在連線之前就被
// 分辨出來，且訊息指向 DSN（而不是「MySQL 不可用」—— 那會讓人去查資料庫）。
func TestReportHealthRejectsUnparsableDSN(t *testing.T) {
	cfgPath := writeConfigFile(t, "DB_DSN=this-is-not-a-dsn\nREDIS_ADDR=127.0.0.1:1\n")

	err := reportHealth(cfgPath)
	if err == nil {
		t.Fatal("無法解析的 DSN 應回錯誤")
	}
	if !strings.Contains(err.Error(), "MySQL DSN 無法使用") {
		t.Errorf("錯誤訊息 = %v, want 指出 DSN 無法使用", err)
	}
}

// TestReportHealthReportsMySQLUnavailable 確認 Ping 失敗是錯誤。
//
// 用 127.0.0.1:1：loopback 上不會有 MySQL，連線立刻被拒絕，因此不需要
// 真的伺服器，也不會依賴任何外部網路。
func TestReportHealthReportsMySQLUnavailable(t *testing.T) {
	cfgPath := writeConfigFile(t, "DB_DSN=forum:secret@tcp(127.0.0.1:1)/forum\nREDIS_ADDR=127.0.0.1:1\n")

	err := reportHealth(cfgPath)
	if err == nil {
		t.Fatal("MySQL 連不上時 reportHealth 應回錯誤")
	}
	if !strings.Contains(err.Error(), "MySQL 不可用") {
		t.Errorf("錯誤訊息 = %v, want 指出 MySQL 不可用", err)
	}
}

// TestReportHealthReportsRedisUnavailable 確認 Redis Ping 失敗是錯誤。
//
// 位址刻意不帶埠號：net.Dial 會立刻回「missing port」，不經過 DNS 也不
// 發出任何封包，因此這條測試不需要網路，也不會有逾時等待。
func TestReportHealthReportsRedisUnavailable(t *testing.T) {
	cfgPath := writeConfigFile(t, "DB_DSN=\nREDIS_ADDR=redis-without-port\n")

	var err error
	captureStdout(t, func() {
		err = reportHealth(cfgPath)
	})
	if err == nil {
		t.Fatal("Redis 連不上時 reportHealth 應回錯誤")
	}
	if !strings.Contains(err.Error(), "Redis 不可用") {
		t.Errorf("錯誤訊息 = %v, want 指出 Redis 不可用", err)
	}
}
