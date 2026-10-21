/*
main 測試共用的輔助函式（backend/main_testhelpers_test.go）。

這些輔助函式存在的理由只有一個：main.go 的對外行為大多直接寫 os.Stdout／
讀 os.Args／用 flag 全域，因此在同一個行程裡呼叫它之前，必須先把這三個
全域狀態換成測試可觀察、可還原的版本。集中在一個檔案裡是為了讓「哪些全域
狀態被動過」一眼可見 —— 漏還原會讓後面的測試看到上一個測試的旗標。
*/
package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// writeConfigFile 把設定檔內容寫進測試專用的暫時檔案並回傳路徑。
//
// 一律用真實檔案而不是注入讀取器：config.Load 與 main 的介面都是路徑，
// 而「檔案不存在」正是 checkConfig 與 reportHealth 都要分類的錯誤。
func writeConfigFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.conf")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("寫入暫時設定檔失敗: %v", err)
	}
	return path
}

// captureStdout 在執行 fn 期間把 os.Stdout 換成暫存檔案，回傳它收到的內容。
//
// -check 的報告刻意直接寫 os.Stdout（它是一個 CLI 模式，沒有注入 writer 的
// 理由），因此測試只能從行程的 stdout 觀察。用檔案而不是 os.Pipe：Pipe 需要
// 一條額外的 goroutine 持續讀取，否則寫滿緩衝區就會卡住，而這裡的輸出量
// 遠小於任何緩衝區，但仍不值得為此引入併發。
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	tmp, err := os.CreateTemp(t.TempDir(), "stdout-*.txt")
	if err != nil {
		t.Fatalf("建立暫存 stdout 失敗: %v", err)
	}
	old := os.Stdout
	os.Stdout = tmp
	defer func() { os.Stdout = old }()

	fn()

	os.Stdout = old
	if err := tmp.Close(); err != nil {
		t.Fatalf("關閉暫存 stdout 失敗: %v", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("讀取暫存 stdout 失敗: %v", err)
	}
	return string(data)
}

// callMain 以指定的命令列參數呼叫 main()，並在返回後還原全域狀態。
//
// 每次呼叫都換一個新的 flag.CommandLine：main 會在同一組 FlagSet 上重新
// 註冊 -check／-healthz／-config，重複註冊同名旗標會 panic，因此「同一個
// 行程呼叫 main 兩次」只有在換掉 FlagSet 之後才成立。
//
// 用 ContinueOnError 而不是預設的 ExitOnError：旗標寫錯時 main 會忽略
// FlagSet.Parse 的回傳值（套件層級的 flag.Parse 本來就沒有回傳值），
// 讓它繼續走預設值遠比讓測試行程直接 exit 2 好除錯。
func callMain(t *testing.T, args ...string) {
	t.Helper()
	oldArgs, oldCmd := os.Args, flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("forum-test", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = append([]string{"forum"}, args...)
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
	}()

	main()
}
