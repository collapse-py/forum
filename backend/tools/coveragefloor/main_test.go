/*
coveragefloor 的測試（main_test.go）。

這支工具的全部價值在於「數字算錯時會安靜地通過」——它自己在 CI 的閘門上，
錯了沒有第二道防線。因此這裡釘三件事：

 1. readCoverage 的算術：count 與 set 兩種 mode 都必須以「count != 0」判定
    覆蓋，且 statements 是各區塊 numStmt 的總和而不是行數。
 2. readCoverage 的錯誤分類：檔案不存在、不是 cover profile、欄位壞掉、
    行過長，四者都必須回錯誤而不是被當成「0 個敘述」——後者會讓閘門以
    一個指向錯誤原因的訊息 exit 2。
 3. run 的四條離開路徑（0 / 1 / 2 / 2）與「哪個 writer 收到什麼」：
    stdout 只放量測結果、stderr 放錯誤。CI 日誌把兩者分開顯示，混在一起
    會讓失敗原因出現在一大段數字之後。
*/
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeProfile 把內容寫進測試專用的暫時檔案並回傳路徑。
//
// 一律走檔案而不是注入 io.Reader：readCoverage 的介面就是路徑，而
// 「檔案不存在」正是它要分類的錯誤之一，改用 Reader 會讓那條路徑永遠
// 測不到。
func writeProfile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("寫入暫時覆蓋率檔失敗: %v", err)
	}
	return path
}

/* ==========================================================================
   readCoverage
   ========================================================================== */

// TestReadCoverageCountMode 是基本算術：重複執行的區塊（count > 1）只能
// 算一次，count == 0 的區塊則完全不計入 covered。
func TestReadCoverageCountMode(t *testing.T) {
	path := writeProfile(t, "mode: count\n"+
		"forum/a.go:1.1,2.2 3 1\n"+
		"forum/a.go:3.1,4.2 2 0\n"+
		"forum/b.go:5.1,6.2 5 2\n")

	statements, covered, err := readCoverage(path)
	if err != nil {
		t.Fatalf("readCoverage = %v, want nil", err)
	}
	if statements != 10 {
		t.Errorf("statements = %d, want 10（3+2+5，不是行數 3）", statements)
	}
	if covered != 8 {
		t.Errorf("covered = %d, want 8（3+5；count=2 只算一次）", covered)
	}
}

// TestReadCoverageSetMode 確認 -covermode=set 的檔案也能讀：本地有人用 set
// 產生檔案時不該讓閘門失敗，而 set 的 1 與 count 的任意正數判斷方式相同。
func TestReadCoverageSetMode(t *testing.T) {
	path := writeProfile(t, "mode: set\n"+
		"forum/a.go:1.1,2.2 2 1\n"+
		"forum/a.go:3.1,4.2 4 0\n")

	statements, covered, err := readCoverage(path)
	if err != nil {
		t.Fatalf("readCoverage = %v, want nil", err)
	}
	if statements != 6 || covered != 2 {
		t.Errorf("statements, covered = %d, %d, want 6, 2", statements, covered)
	}
}

// TestReadCoverageToleratesWhitespace 確認空行與行首尾空白不會被當成壞行：
// 合併多份 profile 的工具常在區塊之間留下空行。
func TestReadCoverageToleratesWhitespace(t *testing.T) {
	path := writeProfile(t, "mode: set\n"+
		"\n"+
		"   \n"+
		"  forum/a.go:1.1,2.2 1 1  \n"+
		"\tforum/a.go:3.1,4.2 2 0\t\n")

	statements, covered, err := readCoverage(path)
	if err != nil {
		t.Fatalf("readCoverage = %v, want nil", err)
	}
	if statements != 3 || covered != 1 {
		t.Errorf("statements, covered = %d, %d, want 3, 1", statements, covered)
	}
}

// TestReadCoverageRejectsFileWithoutMode 確認「隨便一個文字檔」會回錯誤。
// 放行它的話，那個檔案會被當成「0 個敘述」，而錯誤訊息會指向「測試沒跑」
// 這個完全無關的原因。
func TestReadCoverageRejectsFileWithoutMode(t *testing.T) {
	path := writeProfile(t, "forum/a.go:1.1,2.2 3 1\n")

	_, _, err := readCoverage(path)
	if err == nil {
		t.Fatal("沒有 mode: 開頭的檔案應回錯誤")
	}
	if !strings.Contains(err.Error(), "mode:") {
		t.Errorf("錯誤訊息應指出缺少 mode:，得到 %v", err)
	}
}

// TestReadCoverageRejectsEmptyFile 確認空檔案（最常見的「測試根本沒跑」）
// 也走同一條錯誤路徑，而不是回 (0, 0, nil) 讓 run 誤判。
func TestReadCoverageRejectsEmptyFile(t *testing.T) {
	path := writeProfile(t, "")

	statements, covered, err := readCoverage(path)
	if err == nil {
		t.Fatal("空檔案應回錯誤，不能回 0 個敘述讓呼叫端自行解讀")
	}
	if statements != 0 || covered != 0 {
		t.Errorf("錯誤路徑應回 0, 0，得到 %d, %d", statements, covered)
	}
}

// TestReadCoverageReportsMissingFile 確認「路徑錯」與「格式錯」是兩種不同的
// 錯誤：前者是 os.Open 的錯誤，訊息會指向路徑。
func TestReadCoverageReportsMissingFile(t *testing.T) {
	_, _, err := readCoverage(filepath.Join(t.TempDir(), "no-such.out"))
	if err == nil {
		t.Fatal("檔案不存在時應回錯誤")
	}
	if !os.IsNotExist(err) {
		t.Errorf("錯誤應是檔案不存在（os.IsNotExist），得到 %v", err)
	}
}

// TestReadCoverageRejectsMalformedBlocks 逐一把 cover profile 的三個欄位
// 弄壞：欄位數不對、numStmt 不是整數、count 不是整數。三者都必須是錯誤，
// 而不是被跳過 —— 跳過會讓覆蓋率被低估，而低估的方向是「閘門誤擋」，
// 維運會以為是自己的測試退步了。
func TestReadCoverageRejectsMalformedBlocks(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{"欄位太少", "forum/a.go:1.1,2.2 3", "三個欄位"},
		{"欄位太多", "forum/a.go:1.1,2.2 3 1 extra", "三個欄位"},
		{"numStmt 不是整數", "forum/a.go:1.1,2.2 three 1", "不是整數"},
		{"count 不是整數", "forum/a.go:1.1,2.2 3 one", "不是整數"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeProfile(t, "mode: count\n"+c.line+"\n")
			_, _, err := readCoverage(path)
			if err == nil {
				t.Fatalf("壞行 %q 應回錯誤", c.line)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("錯誤訊息 = %v, want 包含 %q", err, c.want)
			}
		})
	}
}

// TestReadCoverageRejectsOverlongLine 確認掃描器的 1 MB 上限真的生效。
//
// 這條路徑的意義不在「大檔案」，而在「檔案被截斷」：一行超過上限時
// Scanner 會把錯誤延後到 Err()，若沒有檢查 Err 就會得到一個看似正常的
// 部分合計，而那個數字會讓閘門以一個難以理解的方式通過或失敗。
func TestReadCoverageRejectsOverlongLine(t *testing.T) {
	path := writeProfile(t, "mode: count\n"+strings.Repeat("a", 2*1024*1024)+"\n")

	_, _, err := readCoverage(path)
	if err == nil {
		t.Fatal("超過 1 MB 的行應讓 Scanner 回錯誤")
	}
	if !strings.Contains(err.Error(), "token too long") {
		t.Errorf("錯誤應是 token too long，得到 %v", err)
	}
}

/* ==========================================================================
   逐套件檢查
   ========================================================================== */

// TestPackageOf 確認套件分組的規則：目錄就是套件，根目錄底下的檔案歸在 "."。
//
// 這條規則是逐套件檢查的基礎：分錯組的話，「某個套件被掏空」就會被算成
// 別的套件的一部分，然後被總數蓋掉。
func TestPackageOf(t *testing.T) {
	cases := []struct {
		block string
		want  string
	}{
		{"main.go:12.3,45.6", "."},
		{"forum/forum/httpapi/server.go:1.1,2.2", "forum/forum/httpapi"},
		{"forum/forum/httpapi/forum_handlers.go:1.1,2.2", "forum/forum/httpapi"},
		// 範圍部分沒有冒號，因此取最後一個冒號才不會把路徑切錯。
		{"forum/a.go:12.3,45.6", "forum"},
	}
	for _, c := range cases {
		if got := packageOf(c.block); got != c.want {
			t.Errorf("packageOf(%q) = %q, want %q", c.block, got, c.want)
		}
	}
}

// TestRunPerPackagePasses 確認每個套件都達到地板時 exit 0，且 stdout 有
// 逐套件的清單。
func TestRunPerPackagePasses(t *testing.T) {
	// 套件 A：2 / 4 = 50%；套件 B：4 / 4 = 100%
	path := writeProfile(t, "mode: count\n"+
		"forum/a/one.go:1.1,2.2 2 1\n"+
		"forum/a/two.go:1.1,2.2 2 0\n"+
		"forum/b/one.go:1.1,2.2 4 1\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"coveragefloor", path, "50", "--per-package"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run = %d, want 0（stdout=%q stderr=%q）", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "逐套件覆蓋率") {
		t.Errorf("stdout 應有逐套件的標題，得到 %q", out)
	}
	if !strings.Contains(out, "forum/a") || !strings.Contains(out, "forum/b") {
		t.Errorf("stdout 應列出兩個套件，得到 %q", out)
	}
	if strings.Contains(out, "!") {
		t.Errorf("全部達標時不該有 ! 標記，得到 %q", out)
	}
	if stderr.Len() != 0 {
		t.Errorf("通過時不應寫 stderr，得到 %q", stderr.String())
	}
}

// TestRunPerPackageFailsWhenOnePackageIsLow 確認「總數過關但某個套件被掏空」
// 會被擋下 —— 這正是逐套件檢查存在的唯一理由。
func TestRunPerPackageFailsWhenOnePackageIsLow(t *testing.T) {
	// 套件 A：2 / 4 = 50%；套件 B：4 / 4 = 100%；總數 6 / 8 = 75%
	path := writeProfile(t, "mode: count\n"+
		"forum/a/one.go:1.1,2.2 2 1\n"+
		"forum/a/two.go:1.1,2.2 2 0\n"+
		"forum/b/one.go:1.1,2.2 4 1\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"coveragefloor", path, "60", "--per-package"}, &stdout, &stderr); code != 1 {
		t.Fatalf("run = %d, want 1（總數 75%% 過關，但套件 A 只有 50%%）", code)
	}
	if !strings.Contains(stdout.String(), "!") {
		t.Errorf("低於地板的套件應在 stdout 被標示，得到 %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "低於地板") {
		t.Errorf("stderr 應指出有套件低於地板，得到 %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "forum/a") {
		t.Errorf("stderr 應指出是哪個套件，得到 %q", stderr.String())
	}
}

// TestRunRejectsUnknownFlag 確認未知的第三個參數是用法錯誤（exit 2），不是
// 被當成「逐套件檢查」而靜默忽略。
func TestRunRejectsUnknownFlag(t *testing.T) {
	path := writeProfile(t, "mode: count\nforum/a.go:1.1,2.2 1 1\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"coveragefloor", path, "10", "--everything"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "用法") {
		t.Errorf("stderr = %q, want 包含用法", stderr.String())
	}
}

/* ==========================================================================
   run
   ========================================================================== */

// TestRunUsageOnWrongArgumentCount 確認參數數量不對時 exit 2，且用法只寫
// 到 stderr —— stdout 留給量測結果，混進去會讓 CI 的摘要夾著一行用法。
func TestRunUsageOnWrongArgumentCount(t *testing.T) {
	cases := [][]string{
		{"coveragefloor"},
		{"coveragefloor", "coverage.out"},
		{"coveragefloor", "coverage.out", "10", "extra"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 2 {
			t.Errorf("run(%v) = %d, want 2", args, code)
		}
		if !strings.Contains(stderr.String(), "用法") {
			t.Errorf("run(%v) 的 stderr = %q, want 包含用法", args, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%v) 不應寫 stdout，得到 %q", args, stdout.String())
		}
	}
}

// TestRunRejectsNonNumericFloor 確認地板值打錯字是「用法錯誤」（exit 2），
// 不是「覆蓋率不足」（exit 1）：兩者在 CI 上要看的處置完全不同。
func TestRunRejectsNonNumericFloor(t *testing.T) {
	path := writeProfile(t, "mode: set\nforum/a.go:1.1,2.2 1 1\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"coveragefloor", path, "abc"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "不是數字") {
		t.Errorf("stderr = %q, want 指出地板值不是數字", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("用法錯誤不應寫 stdout，得到 %q", stdout.String())
	}
}

// TestRunReportsUnreadableProfile 確認讀檔失敗也是 exit 2，且訊息帶著路徑
// （路徑錯是這條分支最常見的原因）。
func TestRunReportsUnreadableProfile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such.out")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"coveragefloor", missing, "10"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Errorf("stderr = %q, want 含路徑 %q", stderr.String(), missing)
	}
}

// TestRunRejectsProfileWithoutStatements 確認「零敘述」是錯誤而不是通過。
// 這是整支工具最重要的一條：否則刪光測試就是讓閘門變綠的最短路徑。
func TestRunRejectsProfileWithoutStatements(t *testing.T) {
	path := writeProfile(t, "mode: count\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"coveragefloor", path, "0"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run = %d, want 2（零敘述不可視為通過）", code)
	}
	if !strings.Contains(stderr.String(), "沒有任何敘述") {
		t.Errorf("stderr = %q, want 指出檔案裡沒有敘述", stderr.String())
	}
}

// TestRunFailsBelowFloor 確認低於地板 exit 1，且 stdout 有完整數字
// （含差值）而 stderr 有處置建議。
func TestRunFailsBelowFloor(t *testing.T) {
	// 1 / 4 = 25.0%
	path := writeProfile(t, "mode: count\n"+
		"forum/a.go:1.1,2.2 1 1\n"+
		"forum/a.go:3.1,4.2 3 0\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"coveragefloor", path, "50"}, &stdout, &stderr); code != 1 {
		t.Fatalf("run = %d, want 1", code)
	}

	wantOut := "總覆蓋率 25.0%（地板 50.0%）：失敗，低於地板 -25.0 個百分點\n" +
		"已覆蓋 1 / 4 個敘述（" + path + "）\n"
	if stdout.String() != wantOut {
		t.Errorf("stdout = %q, want %q", stdout.String(), wantOut)
	}
	if !strings.Contains(stderr.String(), "覆蓋率低於地板") {
		t.Errorf("stderr 應說明低於地板，得到 %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "補測試") {
		t.Errorf("stderr 應提供處置建議，得到 %q", stderr.String())
	}
}

// TestRunPassesAtOrAboveFloor 確認 >= 地板就通過，並且「剛好等於地板」也
// 算通過：地板是「不得低於」，不是「必須高於」。
func TestRunPassesAtOrAboveFloor(t *testing.T) {
	path := writeProfile(t, "mode: count\n"+
		"forum/a.go:1.1,2.2 2 1\n"+
		"forum/a.go:3.1,4.2 2 0\n")

	cases := []struct {
		name  string
		floor string
		delta string
	}{
		{"剛好等於地板", "50", "0.0"},
		{"高於地板", "40", "10.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run([]string{"coveragefloor", path, c.floor}, &stdout, &stderr); code != 0 {
				t.Fatalf("run = %d, want 0", code)
			}
			if !strings.Contains(stdout.String(), "通過") {
				t.Errorf("stdout = %q, want 包含通過", stdout.String())
			}
			if !strings.Contains(stdout.String(), "高於地板 "+c.delta) {
				t.Errorf("stdout = %q, want 差值 %s", stdout.String(), c.delta)
			}
			if stderr.Len() != 0 {
				t.Errorf("通過時不應寫 stderr，得到 %q", stderr.String())
			}
		})
	}
}
