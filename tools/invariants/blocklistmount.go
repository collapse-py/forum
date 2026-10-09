/*
blocklistmount.go 是第三條不變條件的檢查：封鎖中介層只有一個掛載點。

【這條規則保護什麼】
blocklist.go 的檔頭把理由寫得很清楚：

	存在的理由是**掛載位置的正確性**：限流器掛在哪幾條路由上由 server.go 決定，
	而封鎖必須涵蓋同一組路由才不會出現「被封鎖的人仍可從某條沒掛封鎖的路由
	寫入」。

兩者的順序同樣不可交換：先查封鎖再查限流。反過來的話，被封鎖的 IP 會先累積
限流計數，而那個計數在解除封鎖之後仍然生效 —— 一個沒有管理員動作卻持續存在的
隱藏狀態，而且沒有任何地方會顯示它。

這兩件事都屬於「壞掉時不會有錯誤訊息」：限流照常運作、封鎖名單照常運作、
稽核照常寫入。唯一能發現的人是「被封鎖的人發現自己解封後仍然被限流」。

【規則的精確敘述】
所有 withBlocklistHandler 的呼叫必須發生在 applyRateLimit 函式本體之內。
applyRateLimit 是「先查封鎖、再查限流」唯一的組裝點。

【為什麼順序的斷言只到「函式」】
順序由 applyRateLimit 的函式本體決定（它先呼叫 withBlocklistHandler，
withBlocklistHandler 內部再呼叫 limiter.Middleware）。要驗證那個順序就得分析
兩種實作之間的控制流，成本遠高於收益，而且改錯時症狀是「限流計數在解封後
仍生效」，不是任何編譯期可察覺的東西。這個順序改由 blocklist.go 的註解與
server.go 的 applyRateLimit 註解負責 —— 那是它唯一可能被改動的地方，而那裡
本來就有把順序理由寫下來的註解。

【認得的呼叫形狀】
方法呼叫（s.withBlocklistHandler）與自由函式呼叫（withBlocklistHandler）兩者
都認，見 isNamedCall：只認方法呼叫的話，把這個函式重構成自由函式就能讓整條
規則安靜失效。

【為什麼是「只能在 applyRateLimit 裡」而不是「呼叫點只能有一個」】
實作上有兩個呼叫點，兩者都在 applyRateLimit 內（rateLimit 與
rateLimitAllMethods 兩條路徑共用它）。要求「只有一個呼叫點」會迫使其中一條路徑
改成複製組合邏輯 —— 那正是這條規則要防的事。把規則寫成「必須在
applyRateLimit 內」才能同時做到：不多一份組合邏輯、也不容許第二個掛載點。

【為什麼不檢查「封鎖在限流之前」的順序】
順序由 applyRateLimit 的函式本體決定（它先呼叫 withBlocklistHandler，
withBlocklistHandler 內部再呼叫 limiter.Middleware）。要驗證那個順序就得分析
兩種實作之間的控制流，成本遠高於收益，而且改錯時症狀是「限流計數在解封後
仍生效」，不是任何編譯期可察覺的東西。這個順序改由 blocklist.go 的註解與
server.go 的 applyRateLimit 註解負責 —— 那是它唯一可能被改動的地方，而那裡
本來就有把順序理由寫下來的註解。
*/
package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

const (
	withBlocklistHandlerName = "withBlocklistHandler"
	// soleMountPoint 是「先查封鎖、再查限流」唯一允許的組裝函式。
	soleMountPoint = "applyRateLimit"
)

// BlocklistMount 報告在 applyRateLimit 之外組裝封鎖中介層的行為。
var BlocklistMount = &analysis.Analyzer{
	Name:     "blocklistmount",
	Doc:      "withBlocklistHandler 只能在 applyRateLimit 內呼叫（單一掛載點）",
	Run:      runBlocklistMount,
	Requires: []*analysis.Analyzer{},
}

func runBlocklistMount(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// 宣告本身不是掛載點。
			if fn.Name.Name == withBlocklistHandlerName {
				continue
			}
			if fn.Name.Name == soleMountPoint {
				continue
			}
			walkBody(fn.Body, func(_ ast.Node, n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || !isNamedCall(call, withBlocklistHandlerName) {
					return true
				}
				pass.Reportf(call.Pos(),
					"%s 在 %s 之外呼叫 %s。"+
						"這個中介層的順序（先查封鎖、再查限流）只有一個組裝點是正確的；"+
						"多一個掛載點就會出現「被封鎖的人仍可從沒掛封鎖的路由寫入」，"+
						"或「解封後限流計數仍然生效」—— 兩者都不會產生任何錯誤訊息。"+
						"請把這段組合改為在 %s 內呼叫 %s。",
					fn.Name.Name, soleMountPoint, withBlocklistHandlerName,
					soleMountPoint, withBlocklistHandlerName)
				return true
			})
		}
	}
	return nil, nil
}
