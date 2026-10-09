/*
adminauth.go 是第二條不變條件的檢查：管理端點必須先授權、再分派 method。

【這條規則保護什麼】
每一條 /api/admin/* 路由都必須在 method 分派**之前**呼叫 requireAdminForum。
順序有意義 —— 反過來（先 switch r.Method）會讓未登入者用「方法不支援」（405）
反覆探測這些路由是否存在。對一個沒有價值的資訊（哪些後台路徑存在）來說，
那是一個免費的枚舉介面，而且完全不會留下稽核紀錄：requireAdminForum 未通過時
不會寫任何東西，因此「誰在探測後台」在稽核日誌裡是看不到的。

【規則的精確敘述】
對每個以字面路徑 "/api/admin/..." 註冊到 mux 的 handler：

	該 handler 的函式本體必須包含一次 requireAdminForum 呼叫，
	且該呼叫的位置必須在任何 `switch r.Method`（或 switch on r.Method）
	之前。沒有 method switch 的 handler 不受這條限制（它只有一種方法）。

【為什麼只認「字面路徑 + 直接的方法值」】
server.go 全部 24 條後台路由都是 mux.HandleFunc("/api/admin/…", s.handleXxx)
這個形狀。放寬到「變數組成的路徑」或「包了中介層的 handler」不會讓這條規則更
有用，只會讓它需要一個小型的求值器 —— 而那會引入新的漏報（求出錯的路徑就
不會報）。規則寧可在遇到看不懂的形狀時安靜，也不要假裝守住它。

【誤報怎麼處理】
若註冊的 handler 不是一個方法值（例如是個閉包或中介層呼叫），本檢查會跳過它。
這是刻意的漏報：報一個無法理解的註冊會讓人用 //nolint 把它關掉，那比不報更糟。
這個缺口由 repo_test.go 的 TestAdminRoutesAreDirectMethodValues 釘住 —— 它解析
真實的 backend/forum/httpapi/server.go，斷言每一條 /api/admin/ 註冊都是
「接收者叫 mux + 直接方法值」這個形狀。把任一條路由改成閉包的那一天，那支測試
會紅燈；沒有它的話，那條路由會安靜地不再被檢查。
*/
package main

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	// adminRoutePrefix 是後台 API 路由的前綴。
	//
	// 刻意用字面比對而不是「路徑裡有一個 admin 分段」：後台**頁面**（/admin、
	// /admin/monitor、/admin/log…）也在 /admin 底下，但它們是 HTML 殼而不是
	// API 端點，而且它們的 handler 是閉包、授權是由頁面本身承擔的。用分段比對
	// 會讓規則把它們一起拖進來，然後為了消除誤報而被放寬到沒有防護力。
	adminRoutePrefix = "/api/admin/"
	requireAdminName = "requireAdminForum"
	// handlerNamePrefix 是「這支函式是個請求處理器」這個專案的命名慣例。
	//
	// 它存在的唯一理由是讓委派鏈的追蹤**只**沿著真正的 handler 走。若不限制，
	// 一個分流器裡的 s.list(w, r) 也會被跟進去，而 list 沒有授權檢查 → 診斷會
	// 指到一個輔助函式上，那既無法照著修，也會讓人開始懷疑這條規則是否可靠。
	//
	// 這個慣例同時是 securityheaders_test.go 的 TestFrontendShellFilesMatchServerRoutes
	// 與 server.go 路由總表都在依賴的東西，因此它不是這條規則發明的規則。
	handlerNamePrefix = "handle"
)

// AdminAuth 報告未在 method 分派之前授權的管理端點。
var AdminAuth = &analysis.Analyzer{
	Name:     "adminauth",
	Doc:      "/api/admin/* 的 handler 必須在 method 分派之前呼叫 requireAdminForum",
	Run:      runAdminAuth,
	Requires: []*analysis.Analyzer{},
}

func runAdminAuth(pass *analysis.Pass) (any, error) {
	// 先把檔案裡所有的 handler 函式建成「名稱 → 宣告」的索引。
	//
	// 刻意**不**排除方法（fn.Recv == nil 的那個條件）：全部 24 條後台路由的
	// handler 都是 *Server 的方法，而只索引自由函式的話這張表會是空的，
	// 整條規則會靜默地什麼都不報 —— 那是本專案最不能接受的一種失效，
	// 因為 CI 會全綠而規則已經失效。
	handlers := map[string]*ast.FuncDecl{}
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				handlers[fn.Name.Name] = fn
			}
		}
	}

	// visited 是遞迴的終止條件：委派鏈可能形成環（a 呼叫 b、b 呼叫 a），
	// 而那種程式碼本來就已經無限遞迴；檢查器不該因此掛住。
	//
	// 刻意**每條路由一個新的** visited，而不是全套件共用：共用會讓「同一支
	// 函式被兩條路由共用」時第二次的檢查被跳過，診斷因而指到第一次報的那裡。
	// 那正是這個測試資料裡 handleAdminExport / handleAdminLog 共用 list 時
	// 發生的情況 —— 診斷必須各自對著自己的路由。

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			route, handler, ok := adminRouteRegistration(call)
			if !ok {
				return true
			}
			fn, ok := handlers[handler]
			if !ok {
				// 不是直接的方法值（閉包、中介層之類）。刻意跳過，理由見檔頭。
				return true
			}
			checkAdminHandler(pass, handlers, fn, route, map[*ast.FuncDecl]bool{})
			return true
		})
	}
	return nil, nil
}

// adminRouteRegistration 從一個 mux.Handle(...) 呼叫裡取出「路徑字面量」與
// 「handler 方法名」。
//
// 同時接受 Handle 與 HandleFunc：兩者的簽章在這裡是同一個形狀
// （Handle(pattern, handler) / HandleFunc(pattern, handler)），因此 handler 一律
// 是第 1 個之後的參數。
//
// 這裡曾經把 HandleFunc 的 handler 索引寫成 2，結果 len(Args) <= 2 讓每一條
// 路由都被靜默跳過 —— 規則看起來在跑，實際上什麼都沒檢查，而 CI 全綠。
// 負向測試（testdata/src/adminauth/bad）是唯一能抓到這類錯誤的東西，因此
// 「先寫測試再寫規則」在這個專案裡不是流程偏好而是必要條件。
func adminRouteRegistration(call *ast.CallExpr) (route, handler string, ok bool) {
	sel, isMethod := call.Fun.(*ast.SelectorExpr)
	if !isMethod {
		return "", "", false
	}
	recv, isIdent := sel.X.(*ast.Ident)
	if !isIdent || recv.Name != "mux" {
		return "", "", false
	}
	if sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle" {
		return "", "", false
	}

	const handlerIdx = 1
	if len(call.Args) <= handlerIdx {
		return "", "", false
	}

	literal, isLit := call.Args[0].(*ast.BasicLit)
	if !isLit || literal.Kind != token.STRING {
		return "", "", false
	}
	route, err := strconv.Unquote(literal.Value)
	if err != nil || !strings.HasPrefix(route, adminRoutePrefix) {
		return "", "", false
	}

	// handler 必須是 s.handleXxx 這種「接收者 + 方法值」的形式。
	selector, isSelector := call.Args[handlerIdx].(*ast.SelectorExpr)
	if !isSelector {
		return "", "", false
	}
	return route, selector.Sel.Name, true
}

// checkAdminHandler 確認單一 handler 的授權檢查位置，並在它是純分流器時
// 沿委派鏈往下列查。
//
// 什麼算「純分流器」：沒有授權檢查、**而且沒有 switch r.Method**。後者這個
// 條件是關鍵 —— 一個會 switch 方法的 handler 一定在親自處理請求（即使它也
// 呼叫別的 handler，例如 case 裡呼叫 s.list(w, r)），把它當分流器只會讓診斷
// 指到一個不相干的函式上。分流器分流的依據是路徑或查詢參數
// （strings.HasSuffix(r.URL.Path, "/pin")），不是 method。
//
// 為什麼要跟著委派鏈走：ServeMux 對「樣式前綴」與「字面路徑」的處理一樣，
// 所以 /api/admin/forum/posts/{id}/pin 沒辦法單獨註冊（那會吃掉整個
// /{id} 前綴）。這個專案的解法是一個分流器
// （handleAdminPostOrPin → handleAdminPostPin 或 handleAdminForumPost），
// 而授權檢查在被分流到的兩支裡。分流器自己沒有 guard —— 對它直接報錯會是
// 一個無法照著修的誤報，而「用 //nolint 關掉整條規則」的代價遠大於讓
// 規則看穿一層委派。
//
// 語意是「每一支都必须有」而非「任一有」：分流器的所有可能去向都必須授權，
// 否則存在一條未授權的路徑。這與實際風險一致 —— 漏掉的那一支才是問題所在。
func checkAdminHandler(pass *analysis.Pass, handlers map[string]*ast.FuncDecl, fn *ast.FuncDecl, route string, visited map[*ast.FuncDecl]bool) {
	if visited[fn] {
		return
	}
	visited[fn] = true

	guardPos, hasGuard := requireAdminGuardPos(fn)
	methodSwitch := methodSwitchPos(fn)

	if hasGuard && methodSwitch.IsValid() && guardPos > methodSwitch {
		pass.Reportf(guardPos,
			"%s 的 %s 呼叫在 method 分派之後。"+
				"順序有意義：先分派 method 會讓未登入者用 405 反覆探測這些路由是否存在。"+
				"請把 if !s.%s(w, r) { return } 移到 switch r.Method 之前。",
			fn.Name.Name, requireAdminName, requireAdminName)
		return
	}
	if hasGuard {
		return
	}

	// 沒有直接授權檢查。若它是一個純分流器且每一支都自己授權，就不算問題。
	if !methodSwitch.IsValid() {
		if delegated := delegatedHandlers(fn, handlers); len(delegated) > 0 {
			for _, callee := range delegated {
				checkAdminHandler(pass, handlers, callee, route, visited)
			}
			return
		}
	}

	pass.Reportf(fn.Pos(),
		"%s 註冊在 %s，但函式本體沒有呼叫 %s。"+
			"沒有這個檢查，任何人都能呼叫這個後台端點 —— 而且因為未通過時不會寫稽核紀錄，"+
			"探測行為在稽核日誌裡完全看不到。",
		fn.Name.Name, route, requireAdminName)
}

// requireAdminGuardPos 找出函式本體中第一次 requireAdminForum 呼叫的位置。
func requireAdminGuardPos(fn *ast.FuncDecl) (token.Pos, bool) {
	found := token.NoPos
	walkBody(fn.Body, func(_ ast.Node, n ast.Node) bool {
		if found.IsValid() {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok && isNamedCall(call, requireAdminName) {
			found = call.Pos()
			return false
		}
		return true
	})
	return found, found.IsValid()
}

// delegatedHandlers 找出函式內「以 (w, r) 呼叫的另一個 handler」。
//
// 只認**以 w 與 r 為引數**的呼叫：那讓這張清單恰好等於「這個函式可能把
// 請求交給誰」，而不會把 logXxx、internalError 這種輔助函式算進來。
//
// 同時只認名稱以 handle 開頭的目標（見 handlerNamePrefix）：輔助函式會以
// (w, r) 收到請求的機率很高，而跟進去只會讓診斷指到不相干的地方。
//
// 刻意不判斷接收者（是否為 s）：這個套件的 handler 全是 *Server 的方法，
// 而多加一層型別判斷只會讓清單變小、規則變弱。
func delegatedHandlers(fn *ast.FuncDecl, handlers map[string]*ast.FuncDecl) []*ast.FuncDecl {
	var out []*ast.FuncDecl
	seen := map[string]bool{}
	walkBody(fn.Body, func(_ ast.Node, n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !isIdentNamed(call.Args[0], "w") || !isIdentNamed(call.Args[1], "r") {
			return true
		}
		name := sel.Sel.Name
		if seen[name] || !strings.HasPrefix(name, handlerNamePrefix) {
			return true
		}
		callee, ok := handlers[name]
		if !ok || callee == fn {
			return true
		}
		seen[name] = true
		out = append(out, callee)
		return true
	})
	return out
}

// isIdentNamed 判斷運算式是否為指定名稱的變數。
func isIdentNamed(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// methodSwitchPos 找出「對 r.Method 做分派的第一個 switch」。
//
// 刻意只認 tag 是欄位選取 r.Method 的 switch：那是這個專案的分派慣例，而
// 認「任何 switch」會把與方法無關的 switch（比對路徑尾綴、比對 query 參數）
// 也算進來，產生無法照著修的誤報。
func methodSwitchPos(fn *ast.FuncDecl) token.Pos {
	var found token.Pos
	walkBody(fn.Body, func(_ ast.Node, n ast.Node) bool {
		if found.IsValid() {
			return false
		}
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		if sel, ok := sw.Tag.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "r" && sel.Sel.Name == "Method" {
				found = sw.Pos()
				return false
			}
		}
		return true
	})
	return found
}
