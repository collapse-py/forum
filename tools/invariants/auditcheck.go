/*
auditcheck.go 是第一條不變條件的檢查：稽核與操作必須同生共死。

【這條規則保護什麼】
docs/ARCHITECTURE.md 的「管理員操作稽核」一節說明它的脆弱程度：

	這件事很容易被無聲破壞：Go 的 database/sql 不會因為交易內某個語句失敗
	就中止交易，所以呼叫了 recordAdminAction 卻不看回傳值，等於把稽核降級
	成「盡力記錄」。

症狀是：不會出現在任何日誌、任何狀態碼、任何測試失敗裡 —— 使用者照樣得到
200，回應內容完全正常，而「誰刪了這篇文章」從此查不到。

【規則的精確敘述】
對每個「在同一函式內呼叫 beginAdminTx 並且對該交易呼叫 tx.Commit()」的函式：

	在 tx.Commit() 之前，必須至少有一次 recordAdminAction(...) 呼叫，
	而它的**回傳值必須被當成 error 使用**（賦值給一個變數、或被 return、
	或被丟進 if 的初始敘述），不能是一個被丟棄的獨立運算式。

【為什麼要追 tx 變數而不是只看「有沒有呼叫 Commit」】
因為這個套件裡另有兩處 tx.Commit（forum_follow_handlers.go 的追蹤切換、
forum_handlers.go 的按讚）走的是 s.db.Begin 而非 beginAdminTx —— 它們不是
管理員操作，本來就不需要稽核。把「有 Commit」當條件會讓這兩處被誤報。

【誤報率為什麼可控】
規則只看「同一函式內的相對位置」，不做跨函式的資料流分析。三種可能讓人誤判的
寫法都被刻意排除在規則之外：

  - 把稽核寫進一個 helper 函式，helper 自己開交易、自己稽核、自己提交
    （blocklist.go 的 recordBlockAction 就是這個形狀）—— beginAdminTx、
    Commit 與稽核都在同一個函式裡，規則判定正確。
  - 用 defer 提交 —— tx.Commit 不出現在函式本體，規則不會觸發。
  - 同一個函式裡開兩個交易 —— 規則以「最早的那個 Commit 之前有沒有稽核」
    判斷，因此只要第一個 Commit 之前有稽核就會通過。這是刻意的保守選擇：
    一個會誤報的規則會被關掉或加白名單，一個漏報的規則沒有人會發現。
*/
package main

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

const (
	// beginAdminTxName / recordAdminActionName / commitName 以方法名比對，
	// 而不是以「接收者是否為本專案的 *Server」比對。
	//
	// 為什麼不用型別資訊：這個分析器要能被 analysistest 對著 testdata 裡的
	// 假套件跑，而那些假套件沒有本專案的 Server 型別。以名稱比對讓測試可以用
	// 一個極小的樣本涵蓋規則；代價是「別的型別剛好也叫這個名字」，而這個
	// 儲存庫裡 beginAdminTx 只有一個定義（見 analyzer_test.go 的斷言）。
	beginAdminTxName      = "beginAdminTx"
	recordAdminActionName = "recordAdminAction"
	commitName            = "Commit"
)

// AuditCheck 報告 beginAdminTx → Commit 之間沒有檢查稽核錯誤的函式。
var AuditCheck = &analysis.Analyzer{
	Name:     "auditcheck",
	Doc:      "beginAdminTx 之後、tx.Commit() 之前必須檢查 recordAdminAction 的錯誤",
	Run:      runAuditCheck,
	Requires: []*analysis.Analyzer{},
}

// auditCall 是一次 recordAdminAction 呼叫，以及它的回傳值是否被使用。
type auditCall struct {
	pos token.Pos
	// consumed 為 false 代表它是個被丟棄的獨立運算式 —— 也就是本規則要抓的
	// 「稽核降級成盡力記錄」。
	consumed bool
}

// runAuditCheck 走訪檔案裡的每個函式並套用規則。
//
// 刻意不用 inspect.Analyzer：這個規則需要的是「某個節點的父節點是什麼」，
// 而取得父節點最可靠也最簡單的做法是寫一個帶 parent 的手動走訪（見 walkBody）。
// 為了取得父節點而先引入 inspector 再寫一層 stack 配對狀態機，只會讓這個
// 30 行的規則變成 120 行、且多出幾種會靜默抓錯節點的失敗模式。
func runAuditCheck(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			checkFunc(pass, fn)
			return true
		})
	}
	return nil, nil
}

// checkFunc 對單一函式套用規則。
func checkFunc(pass *analysis.Pass, fn *ast.FuncDecl) {
	txVars := beginAdminTxResultVars(fn)
	if len(txVars) == 0 {
		return
	}

	// 找出最早的那個 Commit。beginAdminTx 的宣告本身不會出現在這裡，
	// 因為它裡沒有 .Commit() 呼叫。
	var (
		commitPos token.Pos
		commits   int
	)
	walkBody(fn.Body, func(parent ast.Node, n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isCommitOf(call, txVars) {
			return true
		}
		commits++
		if commits == 1 {
			commitPos = call.Pos()
		}
		return true
	})

	if commits == 0 {
		// 沒有 Commit：這個交易會被回滾（例如開了交易後因為某個錯誤提早
		// return）。這條路徑上沒有任何東西被提交，因此不需要稽核。
		return
	}

	for _, call := range recordAdminActionCalls(fn) {
		if call.pos > commitPos {
			break
		}
		if call.consumed {
			return
		}
	}

	pass.Reportf(commitPos,
		"%s 以 beginAdminTx 開啟交易並提交，但 tx.Commit() 之前沒有檢查 recordAdminAction 的錯誤。"+
			"這會把稽核降級成「盡力記錄」，而症狀不會出現在任何日誌或任何狀態碼裡。"+
			"請把稽核寫成 if err := s.recordAdminAction(...); err != nil { ...; return } 放在 Commit 之前。",
		fn.Name.Name)
}

// walkBody 走訪 body 的每個節點，並把每個節點的**直接父節點**一起交給回呼。
//
// 這是整個分析器的基礎設施：判斷「回傳值有沒有被使用」需要知道呼叫的父節點
// 是什麼，而 ast.Inspect 刻意不提供這項資訊（它要的就是「不要讓你依賴父節點」）。
// 這裡自己維護一層 stack 是安全的，因為呼叫端都是同步的、不會提前返回。
func walkBody(body *ast.BlockStmt, fn func(parent ast.Node, n ast.Node) bool) {
	var stack []ast.Node
	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		var parent ast.Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		stack = append(stack, n)
		if !fn(parent, n) {
			// 回呼要求停止深入：必須把已推入的節點彈出，否則 stack 會失配。
			// 這一行是這個函式唯一需要小心的地方，因此下面用一個獨立的
			// 小測試鎖住它（見 TestWalkBodyStopsCorrectly）。
			stack = stack[:len(stack)-1]
			return false
		}
		return true
	})
}

// beginAdminTxResultVars 找出「由 beginAdminTx 呼叫取得的第一個結果」所綁定的變數名。
//
// 只接受 `tx, err := s.beginAdminTx(r)` 與 `var tx, _ = s.beginAdminTx(r)` 兩種形狀。
// 不接受 `tx = s.beginAdminTx(r)`（tx 在別處宣告）：那種情形下本分析器追不到
// 變數的生命週期，而漏報與誤報之間，這裡選擇漏報 —— 誤報會讓整個規則被關掉。
func beginAdminTxResultVars(fn *ast.FuncDecl) map[string]bool {
	vars := map[string]bool{}
	walkBody(fn.Body, func(_ ast.Node, n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			if stmt.Tok != token.DEFINE || len(stmt.Lhs) == 0 {
				return true
			}
			if !isNamedMethodCall(stmt.Rhs[0], beginAdminTxName) {
				return true
			}
			if id, ok := stmt.Lhs[0].(*ast.Ident); ok {
				vars[id.Name] = true
			}
		case *ast.ValueSpec:
			if len(stmt.Names) == 0 || len(stmt.Values) == 0 {
				return true
			}
			if !isNamedMethodCall(stmt.Values[0], beginAdminTxName) {
				return true
			}
			vars[stmt.Names[0].Name] = true
		}
		return true
	})
	return vars
}

// isCommitOf 判斷這是否為「對 txVars 內任一變數呼叫 Commit」。
func isCommitOf(call *ast.CallExpr, txVars map[string]bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != commitName {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && txVars[id.Name]
}

// recordAdminActionCalls 收集函式內所有 recordAdminAction 呼叫（依原始碼順序）。
func recordAdminActionCalls(fn *ast.FuncDecl) []auditCall {
	var calls []auditCall
	walkBody(fn.Body, func(parent ast.Node, n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isNamedMethodCall(call, recordAdminActionName) {
			return true
		}
		// 唯一的「沒有被使用」形狀是把它當成一個獨立運算式：
		//
		//	s.recordAdminAction(r, tx, ...)     ← 回傳值被丟棄
		//
		// 其餘形狀（賦值、if 初始敘述、return、當作引數）全都代表呼叫端
		// 知道這件事可能失敗，因此都算「有被使用」。
		_, discarded := parent.(*ast.ExprStmt)
		calls = append(calls, auditCall{pos: call.Pos(), consumed: !discarded})
		return true
	})
	return calls
}

// isNamedMethodCall 判斷 expr 是否為「某個接收者上的指定方法呼叫」。
func isNamedMethodCall(expr ast.Expr, method string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == method
}
