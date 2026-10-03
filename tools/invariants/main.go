/*
main 把本專案的不變條件註冊成 go vet 可執行檔。

【為什麼需要這個模組】
這個專案的不變條件是**專屬的**：它們約束的不是「型別對不對」或「有沒有測試」，
而是「稽核有沒有在 Commit 之前被檢查」、「管理端點有沒有先授權再分派方法」、
「封鎖中介層是不是只有一個掛載點」。通用工具（go vet、覆蓋率、lint）抓不到
它們 —— 一個 handler 從 0% 變成 30% 覆蓋，與這三件事毫無關係。

在這個模組之前，它們靠紀律維持，而紀律的上限是「有人記得看」。這裡把它們變成
「靠工具」：失敗模式是**編譯失敗**，不是「有人沒注意到」。

【為什麼是獨立的 Go 模組，而不是 backend 的一個 package】
它需要 golang.org/x/tools，而 backend 的相依項刻意維持在四個（miniredis、
mysql、go-redis、oauth2）。把分析工具的相依塞進 runtime 相依裡，等於讓每個部署
環境都多下載一份只有 CI 需要的程式碼。

【使用方式】

	# 建置分析器
	cd tools/invariants && go build -o invariants .

	# 檢查 backend（backend 目錄下）
	go vet -vettool=<repo>/tools/invariants/invariants ./...

	# 分析器自測（含負向測試：刻意寫違規程式碼，確認會被報）
	go test ./...

【這裡守的三條不變條件】

	auditcheck     稽核與操作同生共死：beginAdminTx → 檢查 recordAdminAction → Commit
	adminauth      管理端點在 method 分派之前呼叫 requireAdminForum
	blocklistmount withBlocklistHandler 只在 applyRateLimit 內被呼叫

第四條「HTML 殼三處同步」跨越 Go / TS / HTML 三種語言，go/analysis 表達不了，
以 backend 的一個測試實作（見 backend/forum/httpapi/frontend_shells_test.go）。
把交叉語言的那條放在測試而非分析器，是刻意的：它的失敗模式是「測試失敗」，
與其他三條的「編譯失敗」不同，而把它放在一起只會讓人搞混哪一條失效了。
*/
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

// analyzers 是這個專案所有不變條件分析器的**唯一**清單。
//
// 刻意讓 main 直接呼叫 multichecker.Main(analyzers...)：那樣「新增分析器但忘記
// 接進 vet」這個失敗模式就從型別上消失了 —— 沒有第二份註冊清單可以漏掉。
// analyzer_test.go 只需要守住另一半（每個項目都要有可用的 Name 與 Doc）。
//
// 第四條不變條件（HTML 殼三處同步）刻意不在這份清單裡，理由見檔頭。
var analyzers = []*analysis.Analyzer{
	AuditCheck,
	AdminAuth,
	BlocklistMount,
}

func main() {
	multichecker.Main(analyzers...)
}
