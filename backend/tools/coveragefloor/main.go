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
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "用法: coveragefloor <coverage.out> <地板百分比>")
		os.Exit(2)
	}
	path := os.Args[1]
	floorPercent, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "地板值 %q 不是數字: %v\n", os.Args[2], err)
		os.Exit(2)
	}

	statements, covered, err := readCoverage(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "讀取 %s 失敗: %v\n", path, err)
		os.Exit(2)
	}
	if statements == 0 {
		// 這是唯一一個必須當成錯誤的情況：一份空的覆蓋率檔代表「測試根本沒跑」
		// 或「專案已經沒有任何敘述」，兩者都不該被當成通過 —— 那等於給了一條
		// 「把測試全刪掉就沒事」的路徑。
		fmt.Fprintf(os.Stderr, "%s 裡沒有任何敘述：測試可能根本沒有執行\n", path)
		os.Exit(2)
	}

	actual := float64(covered) / float64(statements) * 100
	delta := actual - floorPercent
	verdict := "失敗"
	direction := "低於地板"
	if actual >= floorPercent {
		verdict = "通過"
		direction = "高於地板"
	}

	fmt.Printf("總覆蓋率 %.1f%%（地板 %.1f%%）：%s，%s %.1f 個百分點\n",
		actual, floorPercent, verdict, direction, delta)
	fmt.Printf("已覆蓋 %d / %d 個敘述（%s）\n", covered, statements, path)

	if actual < floorPercent {
		// 刻意建議「把地板一起調低」而不是「補測試」：地板代表的是「這個專案
		// 曾經測到哪裡」，把它調低是一個需要說明理由的決定，而補測試是預設
		// 動作。兩者都被說出來，是為了讓做決定的人看到「改地板」是一個選項，
		// 而不是偷偷放寬的捷徑。
		fmt.Fprintf(os.Stderr,
			"\n覆蓋率低於地板。若這是刻意的（例如剛剛刪除了已停用的程式碼），\n"+
				"請更新 .github/workflows/ci.yml 與 ROADMAP.md 的地板值並說明理由；\n"+
				"否則請補測試 —— 這個專案的測試優先序依「壞掉時的爆炸半徑」排序，\n"+
				"不是依哪個檔案最容易測。\n")
		os.Exit(1)
	}
}

// readCoverage 解析 Go 的 cover profile，回傳「總敘述數」與「被執行過的敘述數」。
//
// 格式是每行一筆：<file>:<startLine>.<startCol>,<endLine>.<endCol> <numStmt> <count>，
// 而 count 大於 0 代表那一段被執行過。
//
// 刻意同時支援 count 與 set 兩種 mode：兩者的判斷方式相同（count != 0），
// 而 CI 的 backend job 刻意用 -covermode=count（它的數字才是量測基準），
// 但本地有人用 -covercovermode=set 產生的檔案跑這個工具時不該失敗。
//
// 「檔案不存在」與「格式壞掉」分成兩種錯誤回報：前者通常是路徑錯，後者是
// 版本不相容，而兩者的處置完全不同。
func readCoverage(path string) (statements, covered int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 單行上限調高：cover profile 的一行是一段範圍加兩個數字，長度取決於
	// 檔案路徑，在這個專案的巢狀目錄結構下偶爾會超過 bufio 的 64KB 預設值
	// 之外的場景（例如測試同時涵蓋多個模組時合併輸出）。給 1 MB 是為了讓
	// 「檔案被截斷」這個錯誤不會被誤報成「格式錯誤」。
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	sawMode := false
	for scanner.Scan() {
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
			return 0, 0, fmt.Errorf("第 %d 行不是三個欄位: %q", statements, line)
		}
		numStmt, convErr := strconv.Atoi(fields[1])
		if convErr != nil {
			return 0, 0, fmt.Errorf("欄位 %q 不是整數: %w", fields[1], convErr)
		}
		count, convErr := strconv.Atoi(fields[2])
		if convErr != nil {
			return 0, 0, fmt.Errorf("欄位 %q 不是整數: %w", fields[2], convErr)
		}
		statements += numStmt
		if count > 0 {
			covered += numStmt
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return 0, 0, scanErr
	}
	if !sawMode {
		// 沒有 mode: 開頭就不是 cover profile。放行會讓一個隨便傳進來的檔案
		// 被當成「0 個敘述」，而那條路徑會 exit 2 —— 但訊息會指向錯誤的原因。
		return 0, 0, fmt.Errorf("%s 沒有 mode: 開頭，看起來不是 cover profile", path)
	}
	return statements, covered, nil
}
