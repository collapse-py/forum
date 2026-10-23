/*
coveragefloor 是一個極小的 CI 閘門：讀入 go test -coverprofile 產生的覆蓋率
檔案，若總覆蓋率低於地板值就以非零狀態結束。

【為什麼需要它】
「不讓覆蓋率掉下來」和「把覆蓋率拉高」是兩件事，而只有前者是這個專案現在
需要的。ROADMAP.md 的 Phase 1.3 說得很直接：地板的作用是防止退化，不是製造
動機；而刻意不設一個虛高的目標（例如 70%），因為那會誘發無行為斷言的測試來
拉數字 —— 那種測試是負債，它會在重構時壞掉，卻沒有指出任何東西壞了。

【為什麼自己寫而不是用現成的工具】
  - 現成工具（gocover-cobertura 之類）需要一個 action、一個 token、一份線上
    設定。這個專案連 CI 都是這一階段才新建的，而這個檢查的全部邏輯是 30 行。
  - 規則本身要能被人讀懂並在本地跑：維運在告警時應該能直接回答「這道閘門
    到底在量什麼」，而不必先去查一個第三方專案的文件。

【用法】

	go run ./tools/coveragefloor <coverage.out> <地板百分比，例如 18.7>

	go run ./tools/coveragefloor <coverage.out> <地板百分比> --per-package

	go run ./tools/coveragefloor <coverage.out> <地板百分比> --expect-packages <逗號分隔的套件清單>

第二個用法多一道「逐套件」檢查：除了總覆蓋率，每一個套件也要各自達到地板。
兩者的差別是「總數過關但某個套件被掏空」——那種退化在總數上看不出來，卻正是
真正會出事的地方（被掏空的那個套件正是壞掉時爆炸半徑最大的那個）。

第三個用法補上逐套件檢查**量不到**的那一種退化：一個套件的測試被整包刪掉時，
它不會以 0% 出現在 profile 裡，而是從 profile 裡消失 —— 因此逐套件檢查量不到
它，而總覆蓋率甚至會因為分母變小而上升。呼叫端交出「應該存在的套件清單」
（CI 裡是 go list 的輸出），少了哪一個就 fail。

【輸出】

	總覆蓋率 23.4%（地板 18.7%）：通過，高於地板 4.7 個百分點

或

	總覆蓋率 15.1%（地板 18.7%）：失敗，低於地板 3.6 個百分點

刻意同時印出是高於或低於、以及差幾個百分點：一個只說「失敗」的訊息會讓人
花時間去猜是哪一邊退化了，而這個數字唯一的用途就是被比較。
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// main 只負責把行程的進入點接到 run：真正的判斷與輸出都在 run 裡，
// 因為 run 不碰 os.Exit，測試才能斷言三種離開碼（0 / 1 / 2）。
func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

// run 是 main 的決策本體：回傳值就是行程的 exit code。
//
// 回傳值語意（與原本直接呼叫 os.Exit 的版本完全相同）：
//
//	0	覆蓋率 >= 地板
//	1	覆蓋率 < 地板（CI 要擋下來的情況）
//	2	用法錯誤、地板值不是數字、覆蓋率檔讀不到、或檔案裡沒有任何敘述
//
// stdout 只放「量測結果」，stderr 放錯誤與失敗時的處置建議：CI 日誌通常把
// 兩者分開顯示，而「失敗」與「為什麼失敗」擺在一起最容易被讀懂。
//
// args 的形狀與 os.Args 相同（args[0] 是程式名）；把整個 os.Args 傳進來而
// 不是只傳參數，是為了讓測試用的呼叫與真實呼叫長得一模一樣。
//
// 第三個參數（選用）是 --per-package：除了總覆蓋率，每個套件也要各自達到
// 地板。總數與逐套件是兩種不同的退化，前者看得見、後者看不見，因此兩道
// 閘門一起用。
func run(args []string, stdout, stderr io.Writer) int {
	perPackage := false
	var expectPackages []string

	// 位置參數只有兩個（profile 與地板值），旗標可以出現在任何位置。
	//
	// 刻意不用「len(args) == 3 或 4」那種寫法：它讓第三個旗標沒有地方放，
	// 而每加一個旗標就要改一次那個條件 —— 那是旗標數量與解析邏輯綁在一起的
	// 典型形狀。
	positional := make([]string, 0, 2)
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		switch {
		case arg == perPackageFlag:
			perPackage = true
		case arg == expectPackagesFlag:
			// 下一個 token 是另一個旗標時，把它當成套件清單會產生一個
			// 看起來像套件名稱、實際上永遠不會存在的值（例如
			// "--per-package"），錯誤訊息因此指向錯誤的地方。
			if i+1 >= len(rest) || strings.HasPrefix(rest[i+1], "-") {
				fmt.Fprintf(stderr, "%s 需要一個值（逗號分隔的套件清單）\n", expectPackagesFlag)
				return 2
			}
			i++
			packages := splitPackageList(rest[i])
			if len(packages) == 0 {
				// 明確傳入空值（--expect-packages "" 或 " , "）若被當成
				// 「沒有要求」，這道閘門就會靜默停用 —— 那正是它要防的
				// 退化。空值只能是用法錯誤。
				fmt.Fprintf(stderr, "%s 需要至少一個套件名稱（逗號分隔）\n", expectPackagesFlag)
				return 2
			}
			expectPackages = packages
		case strings.HasPrefix(arg, expectPackagesFlag+"="):
			packages := splitPackageList(strings.TrimPrefix(arg, expectPackagesFlag+"="))
			if len(packages) == 0 {
				fmt.Fprintf(stderr, "%s 需要至少一個套件名稱（逗號分隔）\n", expectPackagesFlag)
				return 2
			}
			expectPackages = packages
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 2 {
		fmt.Fprintf(stderr,
			"用法: coveragefloor <coverage.out> <地板百分比> [%s] [%s <套件清單>]\n",
			perPackageFlag, expectPackagesFlag)
		return 2
	}
	path := positional[0]
	floorPercent, err := strconv.ParseFloat(positional[1], 64)
	if err != nil {
		fmt.Fprintf(stderr, "地板值 %q 不是數字: %v\n", positional[1], err)
		return 2
	}

	profile, err := readProfile(path)
	if err != nil {
		fmt.Fprintf(stderr, "讀取 %s 失敗: %v\n", path, err)
		return 2
	}
	statements, covered := profile.statements, profile.covered
	if statements == 0 {
		// 這是唯一一個必須當成錯誤的情況：一份空的覆蓋率檔代表「測試根本沒跑」
		// 或「專案已經沒有任何敘述」，兩者都不該被當成通過 —— 那等於給了一條
		// 「把測試全刪掉就沒事」的路徑。
		fmt.Fprintf(stderr, "%s 裡沒有任何敘述：測試可能根本沒有執行\n", path)
		return 2
	}

	actual := float64(covered) / float64(statements) * 100
	delta := actual - floorPercent
	verdict := "失敗"
	direction := "低於地板"
	if actual >= floorPercent {
		verdict = "通過"
		direction = "高於地板"
	}

	fmt.Fprintf(stdout, "總覆蓋率 %.1f%%（地板 %.1f%%）：%s，%s %.1f 個百分點\n",
		actual, floorPercent, verdict, direction, delta)
	fmt.Fprintf(stdout, "已覆蓋 %d / %d 個敘述（%s）\n", covered, statements, path)

	// 逐套件檢查在總數之後：總數已經低於地板時就已經是失敗，逐套件的報告
	// 只會讓人先看到一個「整體不行」的結論再看到細節；反過來（總數過關但
	// 某個套件被掏空）才是逐套件檢查唯一能抓到的情況。
	if perPackage {
		if failed := reportPackages(profile.packages, floorPercent, stdout, stderr); failed {
			return 1
		}
	}

	// 「應該存在的套件」檢查在最後：它是一道獨立於覆蓋率數字的閘門，而它的
	// 失敗原因（某個套件的測試被整包刪掉）必須在逐套件報告之後才看，否則
	// 讀者會以為是某個套件的百分比太低。
	if len(expectPackages) > 0 {
		if failed := reportMissingPackages(profile.packages, expectPackages, stderr); failed {
			return 1
		}
	}

	if actual < floorPercent {
		// 刻意建議「把地板一起調低」而不是「補測試」：地板代表的是「這個專案
		// 曾經測到哪裡」，把它調低是一個需要說明理由的決定，而補測試是預設
		// 動作。兩者都被說出來，是為了讓做決定的人看到「改地板」是一個選項，
		// 而不是偷偷放寬的捷徑。
		fmt.Fprintf(stderr,
			"\n覆蓋率低於地板。若這是刻意的（例如剛剛刪除了已停用的程式碼），\n"+
				"請更新 .github/workflows/ci.yml 與 ROADMAP.md 的地板值並說明理由；\n"+
				"否則請補測試 —— 這個專案的測試優先序依「壞掉時的爆炸半徑」排序，\n"+
				"不是依哪個檔案最容易測。\n")
		return 1
	}
	return 0
}

// perPackageFlag 是開啟「逐套件檢查」的旗標。
const perPackageFlag = "--per-package"

// expectPackagesFlag 是交給工具「應該存在的套件清單」的旗標。
//
// 它補的是逐套件地板補不上的那一格。
const expectPackagesFlag = "--expect-packages"

// splitPackageList 把逗號分隔的套件清單切成一片。
//
// 空字串回傳 nil（而不是空切片）。呼叫端把「明確傳入但切完是空」視為
// 用法錯誤（見 run 的 --expect-packages 分支）：否則 `--expect-packages ""`
// 會靜默關掉「應該存在的套件」這道閘門。
func splitPackageList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// reportMissingPackages 確認預期清單裡的套件都出現在覆蓋率檔中。
//
// 回傳值的語意與 reportPackages 一致：**true 表示有問題**（有套件不見了）。
//
// 為什麼需要這道閘門：一個套件的測試被**整包刪掉**時，它不會以「0%」出現
// 在 profile 裡，而是**從 profile 裡消失** —— 因此逐套件地板量不到它（它
// 已經不在被量的集合裡），而總覆蓋率甚至會因為分母變小而**上升**。兩道
// 地板都看不見這種退化。
//
// 因此呼叫端要把「應該存在的套件清單」交給這個工具（CI 裡是 go list 的
// 輸出）：少了哪一個就 fail，並且明確說出「測試被整包刪掉」這個原因。
func reportMissingPackages(packages map[string]packageCoverage, expected []string, stderr io.Writer) bool {
	missing := make([]string, 0, len(expected))
	for _, name := range expected {
		if _, ok := packages[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return false
	}
	sort.Strings(missing)
	fmt.Fprintf(stderr,
		"\n下列套件沒有出現在覆蓋率檔裡：\n  %s\n"+
			"它們的測試可能整包被刪掉了 —— 一個套件從 profile 裡消失不會讓\n"+
			"逐套件地板變紅（因為它已經不在被量的集合裡），而總覆蓋率甚至會\n"+
			"因為分母變小而上升。兩道地板都看不見這種退化。\n",
		strings.Join(missing, "\n  "))
	return true
}

// packageCoverage 是一個套件的覆蓋率統計。
type packageCoverage struct {
	statements int
	covered    int
}

// reportPackages 印出逐套件的覆蓋率，並回傳是否有套件低於地板。
//
// 排序用套件名稱（不是百分比）：CI 日誌裡被比較的是「哪個套件退化了」，
// 而套件名稱是穩定的識別碼；用百分比排序會讓兩次執行的差異難以對照。
func reportPackages(packages map[string]packageCoverage, floorPercent float64, stdout, stderr io.Writer) bool {
	names := make([]string, 0, len(packages))
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)

	failed := false
	var failing []string
	fmt.Fprintf(stdout, "\n逐套件覆蓋率（地板 %.1f%%）：\n", floorPercent)
	for _, name := range names {
		stat := packages[name]
		if stat.statements == 0 {
			// 沒有敘述的套件（例如只有型別宣告）沒有東西可以退化，跳過。
			continue
		}
		percent := float64(stat.covered) / float64(stat.statements) * 100
		mark := " "
		if percent < floorPercent {
			mark = "!"
			failed = true
			failing = append(failing, fmt.Sprintf("%s（%.1f%%，%d / %d 個敘述）", name, percent, stat.covered, stat.statements))
		}
		fmt.Fprintf(stdout, "%s %-40s %6.1f%%（%d / %d 個敘述）\n",
			mark, name, percent, stat.covered, stat.statements)
	}
	if failed {
		fmt.Fprintf(stderr,
			"\n有套件低於地板 %.1f%%：\n  %s\n"+
				"逐套件的門檻存在的理由是：總數過關但某個套件被掏空，\n"+
				"那種退化在總覆蓋率上看不出來，卻正是真正會出事的地方。\n",
			floorPercent, strings.Join(failing, "\n  "))
	}
	return failed
}

// readCoverage 解析 Go 的 cover profile，回傳「總敘述數」與「被執行過的敘述數」。
//
// 這是 readProfile 的薄包裝：保留這個簽章是為了讓「總數」這個概念有個
// 獨立的、可直接測試的入口，而逐套件的統計（readProfile 才做）不需要
// 呼叫端各自重算。
func readCoverage(path string) (statements, covered int, err error) {
	profile, err := readProfile(path)
	if err != nil {
		return 0, 0, err
	}
	return profile.statements, profile.covered, nil
}

// coverageProfile 是一份 cover profile 的統計結果。
type coverageProfile struct {
	statements int
	covered    int
	packages   map[string]packageCoverage
}

// readProfile 解析 Go 的 cover profile，回傳總數與逐套件的統計。
//
// 格式是每行一筆：<file>:<startLine>.<startCol>,<endLine>.<endCol> <numStmt> <count>，
// 而 count 大於 0 代表那一段被執行過。
//
// 刻意同時支援 count、set 與 atomic 三種 mode：三者的判斷方式相同
// （count != 0 即為「有被執行過」），而覆蓋率百分比與計數器是不是 atomic
// 無關（2026-10-22 在本地以 count 與 atomic 各跑一次全模組，總數與逐套件
// 數字逐一相同）。CI 的 backend job 在 -race 之下只能使用 atomic
// （Go 1.26 起 -race 只接受 atomic），而本地有人用別的 mode 產生的檔案
// 跑這個工具時不該失敗。
//
// 「檔案不存在」與「格式壞掉」分成兩種錯誤回報：前者通常是路徑錯，後者是
// 版本不相容，而兩者的處置完全不同。
//
// 套件的分組依「檔案路徑的目錄部分」：cover profile 裡的路徑是相對於模組
// 根目錄的（forum/forum/httpapi/server.go），而套件就是那個目錄。根目錄
// 底下的檔案（main.go）落在「forum」這個模組前綴底下，而沒有前綴的
// （files_server 模組的 server.go）才會是 "." 這一組。
func readProfile(path string) (coverageProfile, error) {
	profile := coverageProfile{packages: map[string]packageCoverage{}}
	file, err := os.Open(path)
	if err != nil {
		return coverageProfile{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 單行上限調高：cover profile 的一行是一段範圍加兩個數字，長度取決於
	// 檔案路徑，在這個專案的巢狀目錄結構下偶爾會超過 bufio 的 64KB 預設值
	// 之外的場景（例如測試同時涵蓋多個模組時合併輸出）。給 1 MB 是為了讓
	// 「檔案被截斷」這個錯誤不會被誤報成「格式錯誤」。
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	sawMode := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "mode:") {
			sawMode = true
			continue
		}
		// 檔名裡的空白已經被 Scanner 處理掉；剩下的三段用 Fields 切開即可，
		// 不需要處理引號（cover profile 不會產生含空白的檔名）。
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return coverageProfile{}, fmt.Errorf("第 %d 行不是三個欄位: %q", lineNo, line)
		}
		numStmt, convErr := strconv.Atoi(fields[1])
		if convErr != nil {
			return coverageProfile{}, fmt.Errorf("第 %d 行的欄位 %q 不是整數: %w", lineNo, fields[1], convErr)
		}
		count, convErr := strconv.Atoi(fields[2])
		if convErr != nil {
			return coverageProfile{}, fmt.Errorf("第 %d 行的欄位 %q 不是整數: %w", lineNo, fields[2], convErr)
		}
		profile.statements += numStmt
		covered := 0
		if count > 0 {
			covered = numStmt
		}
		profile.covered += covered

		pkg := packageOf(fields[0])
		stat := profile.packages[pkg]
		stat.statements += numStmt
		stat.covered += covered
		profile.packages[pkg] = stat
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return coverageProfile{}, scanErr
	}
	if !sawMode {
		// 沒有 mode: 開頭就不是 cover profile。放行會讓一個隨便傳進來的檔案
		// 被當成「0 個敘述」，而那條路徑會 exit 2 —— 但訊息會指向錯誤的原因。
		return coverageProfile{}, fmt.Errorf("%s 沒有 mode: 開頭，看起來不是 cover profile", path)
	}
	return profile, nil
}

// packageOf 從 cover profile 的一行算出它屬於哪個套件。
//
// 輸入是 "<file>:<range>"；取**最後一個**冒號當分隔符，因為路徑本身理論上
// 可能含冒號，而範圍部分（12.3,45.6）一定沒有。
func packageOf(block string) string {
	file := block
	if idx := strings.LastIndex(block, ":"); idx >= 0 {
		file = block[:idx]
	}
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		return file[:idx]
	}
	return "."
}
