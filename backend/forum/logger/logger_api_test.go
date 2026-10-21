/*
logger 公開 API 的測試（backend/forum/logger/logger_api_test.go）。

logger_test.go 覆蓋格式、層級、輪替與中介層；這個檔把**每一個公開進入點**
都至少走一次，並釘住兩件「壞掉時不會有其他測試發現」的事：

 1. 每個 *f / Sprint / *Context 版本真的寫出對應等級與內容。這些是十幾個
    幾乎相同的包裝函式 —— 漏掉其中一個（例如 Errorf 沒接到 ERROR）在
    訊息格式的測試裡完全看不出來，只有逐個呼叫才抓得到。
 2. Fatal / Fatalf 以狀態碼 1 結束行程並把訊息寫出去。這兩個無法在
    目前行程裡測（os.Exit 不會回來），因此用子行程 re-exec 自己。
*/

package logger

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

/* ==========================================================================
   格式化進入點
   ========================================================================== */

func TestPrintfStyleEntrypointsWriteExpectedLevel(t *testing.T) {
	buf := capture(t)

	Debugf("debug-%d", 1)
	Infof("info-%d", 2)
	Warnf("warn-%d", 3)
	Errorf("error-%d", 4)

	out := buf.String()
	for _, want := range []string{"level=DEBUG msg=\"debug-1\"", "level=INFO msg=\"info-2\"",
		"level=WARN msg=\"warn-3\"", "level=ERROR msg=\"error-4\""} {
		if !strings.Contains(out, want) {
			t.Errorf("輸出缺少 %q:\n%s", want, out)
		}
	}
}

/* ==========================================================================
   Sprint 風格進入點
   ========================================================================== */

func TestSprintStyleEntrypointsJoinWithSpaces(t *testing.T) {
	buf := capture(t)

	Debug("a", "b", 1)
	Info("i")
	Warn("w")
	Error("e")

	out := buf.String()
	// fmt.Sprint 的空格規則：只有**兩邊都不是字串**時才加空白 ——
	// 「a","b",1 的結果是 "ab1"，不是 "a b 1" 也不是 "ab 1"。
	// 這個釘住的是呼叫端會實際看到的字串（想自行加分隔請用 %s 格式化）。
	for _, want := range []string{"level=DEBUG msg=\"ab1\"", "level=INFO msg=\"i\"",
		"level=WARN msg=\"w\"", "level=ERROR msg=\"e\""} {
		if !strings.Contains(out, want) {
			t.Errorf("輸出缺少 %q:\n%s", want, out)
		}
	}
}

/* ==========================================================================
   帶請求中繼資料的進入點
   ========================================================================== */

// *Context 版本必須把 request_id / IP / user_email 帶進每一行 —— 它們存在的
// 唯一理由就是把散落的日誌串到同一個請求上；漏掉中繼資料等於退回「十行 log
// 對不出是同一個請求」的舊狀態。
func TestContextEntrypointsCarryRequestMetadata(t *testing.T) {
	buf := capture(t)

	req := httptest.NewRequest("GET", "/forum", nil)
	req = WithMetadata(req, "203.0.113.9", "user@example.com", "req-abc123")
	ctx := req.Context()

	DebugfContext(ctx, "df-%d", 1)
	InfofContext(ctx, "if-%d", 2)
	WarnfContext(ctx, "wf-%d", 3)
	ErrorfContext(ctx, "ef-%d", 4)
	DebugContext(ctx, "d", 1)
	InfoContext(ctx, "i", 2)
	WarnContext(ctx, "w", 3)
	ErrorContext(ctx, "e", 4)

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 8 {
		t.Fatalf("只寫出 %d 行, want 8:\n%s", len(lines), out)
	}
	for i, line := range lines {
		if i == len(lines)-1 {
			break // 最後一行可能是空行
		}
		for _, field := range []string{`request_id="req-abc123"`, `ip="203.0.113.9"`, `user_email="user@example.com"`} {
			if !strings.Contains(line, field) {
				t.Errorf("第 %d 行缺少 %s:\n%s", i+1, field, line)
			}
		}
	}
	// 帶中繼資料時欄位順序是 level … 中繼 … msg，因此 level 與 msg 不相鄰 ——
	// 分開斷言，而不是假設無中繼資料時的 "level=X msg=Y" 連續形狀。
	for _, want := range []string{"level=DEBUG", "level=ERROR", "msg=\"df-1\"", "msg=\"ef-4\"",
		"msg=\"d1\"", "msg=\"e4\""} {
		if !strings.Contains(out, want) {
			t.Errorf("輸出缺少 %q:\n%s", want, out)
		}
	}
}

// 沒有任何中繼資料的 ctx：*Context 版本仍要寫出訊息（中繼資料欄位缺席即可，
// 不該整行消失或 panic）。
func TestContextEntrypointsTolerateEmptyContext(t *testing.T) {
	buf := capture(t)

	InfofContext(context.Background(), "bare-%d", 1)
	ErrorContext(context.Background(), "bare-e")

	out := buf.String()
	if !strings.Contains(out, "msg=\"bare-1\"") || !strings.Contains(out, "msg=\"bare-e\"") {
		t.Errorf("空 ctx 的 Context 進入點沒有寫出訊息:\n%s", out)
	}
}

/* ==========================================================================
   Fatal / Fatalf：子行程
   ========================================================================== */

const fatalHelperEnv = "GO_WANT_FATAL_HELPER"

// fatalHelperEntry 是子行程 re-exec 時實際呼叫 Fatal 的位置。
func fatalHelperEntry() {
	if os.Getenv(fatalHelperEnv) == "printf" {
		Fatalf("fatal-f-%d", 7)
		return
	}
	if os.Getenv(fatalHelperEnv) == "print" {
		Fatal("fatal-plain")
		return
	}
}

func TestFatalExitsWithStatusOne(t *testing.T) {
	fatalHelperEntry() // 子行程模式下不會回到這裡（os.Exit）

	for _, mode := range []string{"print", "printf"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestFatalExitsWithStatusOne")
			cmd.Env = append(os.Environ(), fatalHelperEnv+"="+mode)
			out, err := cmd.CombinedOutput()

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("子行程退出 = %v, want 狀態碼 1（os.Exit(1)）", err)
			}
			text := string(out)
			if mode == "printf" && !strings.Contains(text, "fatal-f-7") {
				t.Errorf("Fatalf 的訊息沒有寫出: %s", text)
			}
			if mode == "print" && !strings.Contains(text, "fatal-plain") {
				t.Errorf("Fatal 的訊息沒有寫出: %s", text)
			}
			if !strings.Contains(text, "level=ERROR") {
				t.Errorf("Fatal 訊息應以 ERROR 等級寫出: %s", text)
			}
		})
	}
}
