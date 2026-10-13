/*
Package logcaller 只有一支測試，且它位於 logger 套件的「外側」是刻意的。

理由：logger 的 caller 抑制規則（loggerPackageDir）會省略「呼叫點就在 logger
套件內」的堆疊帧，因此「業務碼的 caller 會被記錄」這個性質無法在 logger 套件
自己的測試裡驗證 —— 那個測試的檔案就在 logger 目錄下，正好會被規則命中。
把測試放到另一個目錄，偽造的「外部呼叫端」才騙不過抑制規則。

它同時也是 Init → 寫檔 → Close 這條真實路徑的端到端檢查：logger_test.go 裡
的測試都是替換 std.logger 成記憶體 writer；這裡走的是真正的檔案句柄。
*/
package logcaller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forum/forum/logger"
)

// TestBusinessCallerIsRecorded 守住「業務碼的 caller 會出現在日誌裡」。
//
// 抑制規則（呼叫點在 logger 套件內就省略 caller）是為了讓存取記錄不出現固定的
// middleware.go 座標；但同一個規則如果寫壞成「一律省略」，症狀是所有業務日誌
// 失去 caller —— 而那正是排查「這行是誰打的」唯一的線索，且沒有任何錯誤訊息。
// 這支測試就是那個失效的守門員。
func TestBusinessCallerIsRecorded(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "caller.log")
	t.Cleanup(func() { logger.Close() })

	if err := logger.Init("INFO", logFile, "text"); err != nil {
		t.Fatalf("Init 回錯誤: %v", err)
	}

	logger.Infof("來自 logger 套件外側的訊息")
	logger.Close()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("讀取日誌檔失敗: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, `caller="caller_test.go:`) {
		t.Errorf("業務碼的 caller 沒有被記錄（抑制規則可能把 caller 一律省略了）:\n%s", out)
	}
	if strings.Contains(out, "logger.go:") || strings.Contains(out, "middleware.go:") {
		t.Errorf("caller 指向 logger 套件自己的檔案了:\n%s", out)
	}
	if !strings.Contains(out, `msg="來自 logger 套件外側的訊息"`) {
		t.Errorf("訊息沒有以引號化的 msg 欄位結尾:\n%s", out)
	}
	// 行結構：time → level → caller → msg，沒有任何中繼欄位（背景工作）。
	line := strings.TrimSpace(out)
	for _, key := range []string{"request_id=", "ip=", "user_email="} {
		if strings.Contains(line, key) {
			t.Errorf("背景訊息不該有中繼欄位 %s:\n%s", key, line)
		}
	}
	if !strings.HasPrefix(line, "time=") {
		t.Errorf("行首不是 time=:\n%s", line)
	}
}
