/*
CSP 樣板授權的測試（httpapi/securityheaders_test.go）。

這一組測的是一個「壞掉時不會有錯誤訊息」的機制：frontendShellFiles 少列一個
檔名，被漏掉的那一頁的 <style> 就沒有 SHA-256 授權，症狀是整頁沒有版面
（style.css 仍會作用，於是畫面看起來「有樣式但怪怪的」），主控台只有一行
Refused to apply inline style。因此這裡測的不是雜湊演算法本身，而是「清單與
實際送出的頁面殼是否一致」。

為了不依賴真的 dist 目錄，測試自己造一個暫存目錄並在每個殼裡放一段獨特的
樣式，再斷言每段樣式的雜湊都出現在結果裡。少列一個檔名就會少一個雜湊，這個
斷言會立刻失敗，而且不需要知道 SHA-256 的值是什麼。
*/

package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// styleShellHTML 產生一個帶指定 <style> 內容的 HTML 殼。
//
// 換行的位置刻意與真實的樣板一致（<style> 後立刻換行、</style> 前帶縮排），
// 因為 styleBlockContents 取的是標籤之間的原始位元組，含空白時機不同就會算出
// 不同的雜湊 —— 那正是 CSP 雜湊的運作方式。
func styleShellHTML(css string) string {
	return "<!doctype html><html><head><style>\n" + css + "\n    </style></head><body></body></html>\n"
}

// writeShell 在 dir 底下寫出一個 HTML 殼。
func writeShell(t *testing.T, dir, name, css string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(styleShellHTML(css)), 0o644); err != nil {
		t.Fatalf("寫入 %s: %v", name, err)
	}
}

// TestInlineStyleHashesCoverEveryShell 守住 frontendShellFiles 與實際檔案的
// 一對一關係：清單裡每個檔名都要被讀到，產生自己那段樣式的雜湊。
func TestInlineStyleHashesCoverEveryShell(t *testing.T) {
	dir := t.TempDir()
	for i, name := range frontendShellFiles {
		// 每段的內容都不同，因此雜湊必然不同；只要少讀一個檔案，數量就對不上。
		writeShell(t, dir, name, "/* probe "+strings.Repeat("x", i+1)+" */ .a{color:red}")
	}

	hashes := inlineStyleHashes(dir)
	if len(hashes) != len(frontendShellFiles) {
		t.Fatalf("得到 %d 個雜湊, want %d（frontendShellFiles 有 %d 個檔名）",
			len(hashes), len(frontendShellFiles), len(frontendShellFiles))
	}
}

// TestInlineStyleHashesSkipMissingFile 讀不到檔案時只少一個雜湊，不中止。
//
// 症狀是「那一頁沒有版面」，不是服務開不起來 —— 這正是 inlineStyleHashes
// 只 logger.Warnf 的理由（見它的註解）。因此這支測試固定這個行為：其餘檔案
// 的樣式仍然被授權。
func TestInlineStyleHashesSkipMissingFile(t *testing.T) {
	dir := t.TempDir()
	// 寫入清單裡第二個檔案，刻意不寫第一個。
	writeShell(t, dir, frontendShellFiles[1], "/* present */ .a{color:red}")

	hashes := inlineStyleHashes(dir)
	if len(hashes) != 1 {
		t.Fatalf("得到 %d 個雜湊, want 1（缺檔的那頁跳過，其餘仍授權）", len(hashes))
	}
}

// TestInlineStyleHashesAreDeterministic 相同輸入必須產生相同順序的雜湊。
//
// map 的迭代順序是隨機的，而這個函式的輸出會直接進到 CSP 字串裡。若順序不
// 固定，啟動日誌與測試每次看到的不一樣，會讓「雜湊有沒有變」這個問題變得
// 無法回答。
func TestInlineStyleHashesAreDeterministic(t *testing.T) {
	dir := t.TempDir()
	for i, name := range frontendShellFiles {
		writeShell(t, dir, name, "/* probe "+strings.Repeat("y", i+1)+" */ .b{color:blue}")
	}

	first := inlineStyleHashes(dir)
	for range 5 {
		next := inlineStyleHashes(dir)
		if len(next) != len(first) {
			t.Fatalf("雜湊數量不穩定：%d vs %d", len(first), len(next))
		}
		for i := range first {
			if first[i] != next[i] {
				t.Fatalf("第 %d 個雜湊不穩定：%s vs %s", i, first[i], next[i])
			}
		}
	}
}

// TestFrontendShellFilesMatchServerRoutes 守住 frontendShellFiles 與
// server.go 會送出的頁面是同一組。
//
// 這一組是刻意的白名單而不是正則：/admin/monitor/ 這種沒被列出的子路徑必須
// 404。若這份清單與 server.go 的頁面路由不同步，症狀是「某一頁有殼層但沒有
// CSP 授權」或反之 —— 兩者都是整頁沒有版面，而且很難回溯是哪一頁。
//
// 這個測試寫死兩份清單，因此新增頁面時必須同時更新兩處。
func TestFrontendShellFilesMatchServerRoutes(t *testing.T) {
	want := []string{
		"forum.html",
		"forum-login.html",
		"forum-new.html",
		"forum-profile.html",
		"forum-others-profile.html",
		"forum-following.html",
		"admin.html",
		"forum-admin.html",
		"forum-report.html",
		"forum-monitor.html",
		"audit-log.html",
		"forum-stats.html",
		"export.html",
		"sessions.html",
		"blocks.html",
		"announcements.html",
	}

	got := make(map[string]int, len(frontendShellFiles))
	for _, name := range frontendShellFiles {
		got[name]++
	}

	for _, name := range want {
		if got[name] == 0 {
			t.Errorf("frontendShellFiles 少了 %s（那一頁的 <style> 會被 CSP 擋下）", name)
		}
	}
	for _, name := range frontendShellFiles {
		if got[name] > 1 {
			t.Errorf("frontendShellFiles 有重複項目：%s", name)
		}
		delete(got, name)
	}
	for name := range got {
		t.Errorf("frontendShellFiles 多了 %s（那個檔案不會送出，雜湊是無效的放行）", name)
	}
}
