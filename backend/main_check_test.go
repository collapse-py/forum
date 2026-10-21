/*
-check 報告輔助函式的測試（backend/main_check_test.go）。

main.go 的 -check 模式在啟動前回答「這個行程會以什麼組態啟動」。它的輔助
函式各自守著一件「印錯了也不會有人發現」的事：

  - dbEndpoint：從 DSN 抽出主機位址而**不印帳密**。-check 的輸出會出現在
    CI 日誌與 issue 裡，把整個 DSN 印出去等於把憑證貼上公開區域。
  - humanizeSeconds：2592000（30 天）必須被標示為「開發用兜底值」——
    一個寫著 2592000 秒的 TTL 很容易被讀者當成刻意設定的數字而放過。
  - orNotSet：ES_URL 與 LOG_FILE 為空的後果完全不同（搜尋不啟用 vs 日誌
    只進 stdout），刻意不共用一個「（未設定）」。
  - checkConfig：設定檔讀不到時必須回錯誤（exit 1），合法檔案則完成報告。

刻意沒有測的：報告的完整文案。那是會隨維運需求調整的文字，釘住全文的測試
只會讓每次調整文案都變成三處修改；這裡釘的是「不印憑證」與分支語意。
*/

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forum/forum/config"
)

func TestDBEndpointParsesHostPortWithoutCredentials(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		want string
	}{
		{"完整 DSN", "forum:secret@tcp(192.168.1.10:3306)/forum?parseTime=true", "192.168.1.10:3306"},
		{"無帳密", "tcp(localhost:3306)/forum", "localhost:3306"},
		{"空 DSN", "", "（未設定）"},
		{"無法解析（沒有括號）", "forum:secret@dbhost", "（無法解析 DSN）"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dbEndpoint(c.dsn)
			if got != c.want {
				t.Errorf("dbEndpoint(%q) = %q, want %q", c.dsn, got, c.want)
			}
			// 任何情況下都不得印出帳密。
			if strings.Contains(got, "secret") || strings.Contains(got, "forum:") {
				t.Errorf("dbEndpoint 的輸出洩漏了憑證: %q", got)
			}
		})
	}
}

func TestPresentMarksSecretPresenceAndLengthOnly(t *testing.T) {
	if got := present(""); got != "未設定" {
		t.Errorf("present(\"\") = %q, want 未設定", got)
	}
	got := present("s3cr3t-value")
	if !strings.Contains(got, "12") {
		t.Errorf("present 應包含長度: %q", got)
	}
	if strings.Contains(got, "s3cr3t") {
		t.Errorf("present 洩漏了內容: %q", got)
	}
}

func TestOrNotSetKeepsReasonsDistinct(t *testing.T) {
	if got := orNotSet("", "搜尋不會啟用"); got != "（搜尋不會啟用）" {
		t.Errorf("空值 = %q, want 帶說明的括號文字", got)
	}
	if got := orNotSet("/var/log/forum.log", "日誌只進 stdout"); got != "/var/log/forum.log" {
		t.Errorf("非空值應原樣回傳, 得到 %q", got)
	}
	// 兩個空值的說明必須可区分：共用一個「（未設定）」會讓讀者以為嚴重度相同。
	es := orNotSet("", "搜尋不啟用")
	log := orNotSet("", "日誌只進 stdout")
	if es == log {
		t.Error("ES 與 LOG 的空值說明不可区分")
	}
}

func TestHumanizeSecondsHighlightsDevFallback(t *testing.T) {
	cases := []struct {
		seconds int
		want    string
	}{
		{0, "無限期"},
		{-5, "無限期"},
		{2592000, "30 天 —— 這是開發用兜底值，正式環境應設成秒數級"},
		{172800, "2 天"},
		{7200, "2 小時"},
		{300, "5 分鐘"},
		{45, "45 秒"},
	}
	for _, c := range cases {
		if got := humanizeSeconds(c.seconds); got != c.want {
			t.Errorf("humanizeSeconds(%d) = %q, want %q", c.seconds, got, c.want)
		}
	}
}

func TestEsAuthFromMapsFields(t *testing.T) {
	auth := esAuthFrom(config.Config{ESUsername: "u", ESPassword: "p", ESAPIKey: "k"})
	if auth.Username != "u" || auth.Password != "p" || auth.APIKey != "k" {
		t.Errorf("esAuthFrom 欄位搬運不完整: %+v", auth)
	}
}

/* ==========================================================================
   checkConfig 本體
   ========================================================================== */

func TestCheckConfigFailsOnMissingFile(t *testing.T) {
	err := checkConfig(filepath.Join(t.TempDir(), "no-such.conf"))
	if err == nil {
		t.Fatal("設定檔不存在時 checkConfig 應回錯誤（exit 1）")
	}
}

// 合法檔案（即使幾乎是空的）必須完成報告且不回錯誤：-check 的失敗模式是
// 「檔案層級問題」，設定值本身的問題只出現在報告文字裡（exit 仍為 0，
// 除非 Validate 找到必須修的項目）。
func TestCheckConfigCompletesForMinimalConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.conf")
	body := "SERVER_PORT=:8088\nCOOKIE_SECURE=false\nLOG_MAX_SIZE_MB=0\nLOG_MAX_FILES=5\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("寫入暫時設定檔失敗: %v", err)
	}
	if err := checkConfig(path); err != nil {
		t.Fatalf("最小合法設定檔的 checkConfig = %v, want nil", err)
	}
}
