/*
frontend_shells_test.go 守住第四條不變條件：十六個 HTML 殼檔名必須在**三處**
同步。

 1. frontend/src/entries/        每個殼檔對應一支入口模組
 2. frontend/vite.config.ts      rollupOptions.input（決定 build 出什麼）
 3. httpapi.frontendShellFiles   CSP 的 style-src 'sha256-…' 授權清單

少一處的症狀不是錯誤訊息，而是**整頁沒有版面**：style.css 仍會作用（它是被
<link> 載入的外部檔案，style-src 'self' 已經涵蓋），於是畫面看起來「有樣式但
怪怪的」，而瀏覽器主控台只有一行 Refused to apply inline style。

【為什麼這條是「測試」而不是 Phase 2 的 analyzer】
它跨越 Go / TypeScript / HTML 三種語言，go/analysis 表達不了（它只能看 Go 的
AST）。其他三條規則的失敗模式是編譯失敗，這一條是測試失敗 —— 兩者刻意不混在
同一個工具裡，避免搞混哪一條失效了。ROADMAP.md 的 Phase 2.4 把這件事列為
「沿用 securityheaders_test.go 已有的模式」，因此它是一個測試。

【為什麼是三處而不是兩處】
securityheaders_test.go 已經守住第 2 與第 3 處（後端清單 vs server.go 會送出
的頁面）與雜湊的完整性。它守不到的是「建置產物裡真的有那一頁」與「那一頁的
入口模組存在」—— 後者壞掉時症狀是頁面載入後完全沒有反應（React 從來沒掛載），
而前者壞掉時症狀是那一頁 404。兩者都不會出現在任何日誌或狀態碼裡。

【為什麼從 testdata 推導而不是比對名稱】
這份測試刻意**不**假設「forum.html ↔ forum.tsx」這種命名規則，而是解析每個
殼檔裡的 <script type="module" src="/src/entries/X.tsx">。那個 src 才是
Vite 實際會載入的東西，因此它就是「這一頁的入口是哪一支」的事實來源。
靠命名慣例比對的話，改名（forum.html → index.html）會在沒有任何問題的情況下
讓這個測試紅掉 —— 一個會誤報的規則沒有人會留著。
*/
package httpapi

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// entryScriptRe 取出殼檔裡的入口模組路徑。
//
// 只認 type="module"：CSP 的 script-src 只有 'self'（沒有 'unsafe-inline'），
// 因此掛載點必須由外部模組建立，而各頁的 <style> 區塊不會被這個比對誤認
// （它是 <style> 不是 <script>）。
var entryScriptRe = regexp.MustCompile(`<script[^>]*\ssrc="(/src/entries/[^"]+)"`)

// viteInputRe 取出 vite.config.ts 的 rollupOptions.input 值。
//
// 刻意用比對而不是解析 TypeScript：這個檔案是設定檔，而我們要問的只是
// 「它列了哪些 .html」。用 AST 解析會帶來一個 @typescript-eslint 形式的相依
// 或一個自製解析器 —— 兩者對「檢查 16 個檔名」這個需求都太重了。
var viteInputRe = regexp.MustCompile(`^\s*[A-Za-z0-9_]+\s*:\s*'([^']+\.html)'`)

// TestFrontendShellsAreSyncedInThreePlaces 是這條不變條件的主測試。
//
// 逐處指名缺什麼，而不是只說「不同步」：新增一個頁面的人會同時看到三個檔名
// 清單，訊息裡直接給他該改哪三處，比讓他自己找出清單快得多，也比一個
// 「測試失敗」有用得多。
func TestFrontendShellsAreSyncedInThreePlaces(t *testing.T) {
	repo := repoRoot(t)
	frontendDir := filepath.Join(repo, "frontend")

	// 第 1 處（權威來源）：殼檔自己宣告的入口模組。
	shells := shellsWithEntries(t, frontendDir)
	if len(shells) == 0 {
		t.Fatal("在 frontend/ 找不到任何帶入口模組的 HTML 殼：測試的前置條件已失效")
	}

	// 第 2 處：src/entries/ 底下必須正好有這些入口模組。
	//
	// 兩邊都保留 .tsx 副檔名（不去除）：殼檔裡的 src 寫的就是 "admin.tsx"，
	// 那是 Vite 實際要解析的路徑。去掉副檔名比對等於允許 "admin.jsx" 通過，
	// 而那個檔在 build 時不存在 —— 症狀是「頁面 200 但畫面空白」。
	entriesDir := filepath.Join(frontendDir, "src", "entries")
	entryFiles, err := listFiles(entriesDir, ".tsx")
	if err != nil {
		t.Fatalf("讀取 src/entries 失敗: %v", err)
	}
	entrySet := toSet(entryFiles, "")

	wantEntries := map[string]bool{}
	for _, entries := range shells {
		for _, entry := range entries {
			wantEntries[filepath.Base(entry)] = true
		}
	}
	missingEntries := difference(wantEntries, entrySet)
	extraEntries := difference(entrySet, wantEntries)
	for _, name := range sortedKeys(missingEntries) {
		t.Errorf("frontend/src/entries/%s 不存在，但有殼檔宣告它是入口模組", name)
	}
	for _, name := range sortedKeys(extraEntries) {
		t.Errorf("frontend/src/entries/%s 沒有任何殼檔使用它（多階段建置會把它當成 tree-shaken 掉）", name)
	}

	// 第 3 處：vite.config.ts 的 rollupOptions.input。
	viteInputs, err := viteInputFiles(t, filepath.Join(frontendDir, "vite.config.ts"))
	if err != nil {
		t.Fatalf("解析 vite.config.ts 失敗: %v", err)
	}
	viteSet := toSet(viteInputs, "")
	wantShells := map[string]bool{}
	for shell := range shells {
		wantShells[shell] = true
	}
	for _, name := range sortedKeys(difference(wantShells, viteSet)) {
		t.Errorf("vite.config.ts 的 rollupOptions.input 少了 %s（build 不會產生那一頁）", name)
	}
	for _, name := range sortedKeys(difference(viteSet, wantShells)) {
		t.Errorf("vite.config.ts 的 rollupOptions.input 有 %s，但 frontend/ 底下沒有這個殼檔", name)
	}

	// 第 4 處：後端的 CSP 授權清單。
	gotShells := toSet(frontendShellFiles, "")
	for _, name := range sortedKeys(difference(wantShells, gotShells)) {
		t.Errorf("httpapi.frontendShellFiles 少了 %s（那一頁的 <style> 沒有 SHA-256 授權，症狀是整頁沒有版面）", name)
	}
	for _, name := range sortedKeys(difference(gotShells, wantShells)) {
		t.Errorf("httpapi.frontendShellFiles 有 %s，但沒有這個殼檔（那是無效的 CSP 放行）", name)
	}
}

// TestFrontendShellsAreNotEmpty 擋掉「殼檔存在但沒有入口模組」這種退化。
//
// 這一組本來會被上面的比對擋下（src/entries 少一支），但那個訊息說的是
// 「entries 少了 X」，不會提到「X.tsx 對應的殼是空的」。而後者才是真正的原因 ——
// 空殼的症狀是「頁面 200 但畫面空白」，與「入口模組遺失」的症狀不同。
func TestFrontendShellsAreNotEmpty(t *testing.T) {
	frontendDir := filepath.Join(repoRoot(t), "frontend")
	shells := shellsWithEntries(t, frontendDir)

	for shell, entries := range shells {
		for _, entry := range entries {
			path := filepath.Join(frontendDir, filepath.FromSlash(strings.TrimPrefix(entry, "/")))
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("%s 宣告的入口模組 %s 讀不到: %v", shell, entry, err)
				continue
			}
			if !strings.Contains(string(raw), "createRoot") {
				t.Errorf("%s 讀不到 React 掛載（沒有 createRoot）：%s", shell, entry)
			}
		}
	}
}

// shellsWithEntries 掃描 frontend/ 底下的 HTML 殼，回傳「殼檔名 → 入口模組」。
//
// 只看頂層目錄（不遞迴）：frontend/dist/ 與 node_modules/ 底下也有 .html，
// 把它們算進來會讓這份清單隨著「有沒有跑過 build」而改變。
func shellsWithEntries(t *testing.T, frontendDir string) map[string][]string {
	t.Helper()
	files, err := listFiles(frontendDir, ".html")
	if err != nil {
		t.Fatalf("讀取 frontend/ 失敗: %v", err)
	}
	if len(files) == 0 {
		return nil
	}

	shells := map[string][]string{}
	for _, name := range files {
		raw, err := os.ReadFile(filepath.Join(frontendDir, name))
		if err != nil {
			t.Errorf("讀取 %s 失敗: %v", name, err)
			continue
		}
		matches := entryScriptRe.FindAllStringSubmatch(string(raw), -1)
		if len(matches) == 0 {
			t.Errorf("%s 沒有 <script type=\"module\" src=\"/src/entries/...\">："+
				"CSP 的 script-src 只有 'self'（沒有 'unsafe-inline'），因此 React 必須由外部模組掛載",
				name)
			continue
		}
		var entries []string
		for _, m := range matches {
			entries = append(entries, m[1])
		}
		shells[name] = entries
	}
	return shells
}

// viteInputFiles 從 vite.config.ts 取出 rollupOptions.input 列出的 .html 值。
func viteInputFiles(t *testing.T, path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		if m := viteInputRe.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	if len(out) == 0 {
		return nil, errNoViteInputs
	}
	return out, nil
}

// listFiles 列出 dir 底下（不遞迴）具有指定副檔名的檔名。
func listFiles(dir, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// toSet 把檔名清單轉成集合；ext 非空時一併去掉該副檔名。
func toSet(names []string, ext string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		if ext != "" {
			n = strings.TrimSuffix(n, ext)
		}
		out[n] = true
	}
	return out
}

// difference 回傳 a 有而 b 沒有的項目。
func difference(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		if !b[k] {
			out[k] = true
		}
	}
	return out
}

// sortedKeys 讓失敗訊息每次的順序都一樣。
//
// 刻意排序而不是直接 range map：Go 的 map 迭代順序是隨機的，而一個每次跑
// 錯誤訊息順序都不同的測試，會讓人誤以為「有時只錯一個」。
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// errNoViteInputs 是「vite.config.ts 解析不出任何 .html」的錯誤。
//
// 單獨宣告成變數而不是用 errors.New 內嵌：呼叫端需要能用 errors.Is 判斷它是
// 「格式變了」而不是「檔案讀不到」—— 前者的處置是更新這個測試的正則，後者
// 是檢查工作目錄。兩者混成同一個錯誤訊息時，人會先花時間找檔案。
var errNoViteInputs = errViteInputFormat{}

type errViteInputFormat struct{}

func (errViteInputFormat) Error() string {
	return "vite.config.ts 解析不到任何 .html（rollupOptions.input 的格式可能變了，請更新本檔的 viteInputRe）"
}

// repoRoot 由本測試檔的原始碼位置往上找出倉庫根。
//
// 刻意用 runtime.Caller 而不是相對於 os.Getwd()：go test 執行每個測試時的
// 工作目錄是**套件來源目錄**（這裡是 backend/forum/httpapi），而 frontend/ 在
// 倉庫根的三層之上。用 cwd 推導會得到一個不存在的路徑，而症狀是「測試找不到
// frontend/」—— 那看起來像前端沒建置，而不是路徑算錯。
//
// 已知限制：runtime.Caller 回傳的是**建置時**的路徑。CI 上
// actions/checkout 給的是絕對路徑，因此成立；本機在 symlinked 目錄下建置則
// 可能需要多一層解析。與其為此引入一個路徑解析函式，不如讓失敗訊息說清楚。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失敗：無法定位本測試檔")
	}
	// httpapi → forum → backend → 倉庫根
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "frontend", "vite.config.ts")); err != nil {
		t.Fatalf("由 %s 推導出的倉庫根 %s 看起來不對（找不到 frontend/vite.config.ts）: %v",
			thisFile, root, err)
	}
	return root
}
