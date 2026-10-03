/*
csrf_guard_test.go 守住 csrf.go 定義的那條不變條件：

	**每一條會改變資料的路徑都必須呼叫 isTrustedOrigin。**

【為什麼這條值得一個測試，而不只是靠審查】

isTrustedOrigin 壞掉時的症狀是「CSRF 防護失效但沒有任何錯誤」：受害者會
自願發出那個請求（表單自動送出、連結被點、sendBeacon），從伺服器的角度來看
它與使用者自己的請求完全一樣 —— 沒有異常、沒有日誌。這是本專案少數「失效
完全靜默」的不變條件，因此它特別需要機械式的把關。

2026-10-03 的實測就抓到一個真實缺口：handleForumImageTokensRelease（釋放
圖片 token）只掛了 requireLogin 與限流，沒有來源檢查，而它會對 mediaRedis
執行 DEL。csrf.go 檔頭說這份清單「靠程式碼審查維持」，而審查漏掉了一格 ——
這支測試就是為了讓那種漏掉變成編譯後第一次執行就紅燈。

【這個測試怎麼判定一個 handler「有守住」】

一個寫入 handler 的來源檢查有兩種合法位置，兩種都算守住：

  1. 直接在 handler（或它呼叫的任一 helper）裡呼叫 s.isTrustedOrigin(r)。
     這是專案的主力寫法（30 處守衛中的 28 處）。
  2. 整條路由在註冊時被 s.requireTrustedOrigin 包住（/api/logout 就是這樣）。

第 1 種必須**沿著呼叫鏈遞迴**去找，因為主力寫法是把守衛放在被呼叫的 helper
裡：例如 handleForumPosts 本身沒有守衛，POST 交給 createForumPost，而守衛在
createForumPost（見 forum_handlers.go）。只看 handler 自己的函式本體會把這
種（正確的）寫法誤判成缺口 —— 本測試第一版就是這樣誤報了五個。

第 2 種要認得「以 method value 傳入」（s.requireTrustedOrigin(s.handleLogout)），
不只是「以呼叫結果傳入」。

【判定「這個 handler 會改變資料」】

只在 handler 把 r.Method 與 http.MethodPost/Put/Patch/Delete 拿來比較（== 或
!=）或放進 switch 的 case 時才算。這裡刻意不比「整個函式有沒有出現
MethodPost」：handleForumImageUpload 會用 http.MethodPost 組一個**發往外部
檔案伺服器**的請求，那不是本 handler 的 incoming 方法，用寬鬆的判法會把它
（以及未來任何碰巧用到常數的唯讀 handler）誤判成寫入端點。寬鬆的判法會產生
假陽性，而假陽性會讓人開始繞過這道閘門 —— 那比漏掉一個真的缺口更糟。

【為什麼放在測試而不是 tools/invariants 的分析器】

這條不變條件本身是純 Go 的，照專案在其他三條上的取捨應該做成分析器（失敗
模式是編譯失敗）。但它的判定需要「沿呼叫鏈遞迴 + 認得 method value 形式的
中介層包裝」，而分析器必須在**單一檔案**的層級做这件事才不會漏掉跨檔的
helper（handleForumPosts → createForumPost 就在同一檔，但同類情況可以跨檔）。
把它做成同套件內的測試、就地用 go/parser 讀整個套件來源，既拿到跨檔的能
力，又不需要新增建置步驟，且已經被既有的 go test ./... 這道閘門守住。取捨
寫在這裡，是為了讓後人知道這是刻意的選擇而不是「還沒做成分析器」。
*/
package httpapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// handlerFacts 是這個套件裡一支 handler（或 helper）與 CSRF 相關的三件事。
type handlerFacts struct {
	mutating    bool            // 會依 r.Method 進入寫入分支
	callsOrigin bool            // 函式本體（遞迴前）直接呼叫 isTrustedOrigin
	calls       map[string]bool // 呼叫到的套件內函式名（用來遞迴）
	pos         string
}

// parseHandlerFacts 讀整個 httpapi 套件（不含 _test.go），為每支函式建立
// handlerFacts，並找出以 requireTrustedOrigin 包裝的 handler 名稱。
//
// 回傳 (facts, wrapped)：wrapped 是那些在路由註冊處被 requireTrustedOrigin
// 包住的 handler 名稱集合。
func parseHandlerFacts(t *testing.T) (map[string]*handlerFacts, map[string]bool) {
	t.Helper()
	// go test 把工作目錄設成套件來源目錄，因此 "." 就是 httpapi 本身。
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("解析 httpapi 套件失敗：%v", err)
	}

	facts := map[string]*handlerFacts{}
	wrapped := map[string]bool{}

	isMutatingMethod := func(name string) bool {
		switch name {
		case "MethodPost", "MethodPut", "MethodPatch", "MethodDelete":
			return true
		}
		return false
	}
	isRequestMethod := func(e ast.Expr) bool {
		s, ok := e.(*ast.SelectorExpr)
		return ok && s.Sel.Name == "Method" // r.Method
	}

	for _, pkg := range pkgs {
		for _, af := range pkg.Files {
			for _, decl := range af.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				f := &handlerFacts{
					calls: map[string]bool{},
					pos:   fset.Position(fn.Pos()).String(),
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					switch x := n.(type) {
					case *ast.CallExpr:
						switch fun := x.Fun.(type) {
						case *ast.SelectorExpr:
							f.calls[fun.Sel.Name] = true
							if fun.Sel.Name == "isTrustedOrigin" {
								f.callsOrigin = true
							}
						case *ast.Ident:
							f.calls[fun.Name] = true
						}
					case *ast.BinaryExpr:
						// r.Method == http.MethodX / r.Method != http.MethodX
						if isRequestMethod(x.X) && isMutatingMethod(selName(x.Y)) {
							f.mutating = true
						}
						if isRequestMethod(x.Y) && isMutatingMethod(selName(x.X)) {
							f.mutating = true
						}
					case *ast.SwitchStmt:
						// switch r.Method { case http.MethodPost: ... }
						if x.Tag != nil && isRequestMethod(x.Tag) {
							ast.Inspect(x.Body, func(m ast.Node) bool {
								if s, ok := m.(*ast.SelectorExpr); ok && isMutatingMethod(s.Sel.Name) {
									f.mutating = true
								}
								return true
							})
						}
					}
					return true
				})
				facts[fn.Name.Name] = f
			}
			// 找出 s.requireTrustedOrigin(<handler>) 的包裝點。參數可能是
			// method value（s.handleLogout）或 inline call（s.handleX(...)）。
			ast.Inspect(af, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				se, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || se.Sel.Name != "requireTrustedOrigin" {
					return true
				}
				for _, a := range call.Args {
					switch h := a.(type) {
					case *ast.SelectorExpr:
						wrapped[h.Sel.Name] = true
					case *ast.CallExpr:
						if s2, ok := h.Fun.(*ast.SelectorExpr); ok {
							wrapped[s2.Sel.Name] = true
						}
					}
				}
				return true
			})
		}
	}
	return facts, wrapped
}

func selName(e ast.Expr) string {
	if s, ok := e.(*ast.SelectorExpr); ok {
		return s.Sel.Name
	}
	return ""
}

// reachesOrigin 沿著套件內的呼叫鏈遞迴，判斷 fn 最終有沒有呼叫到 isTrustedOrigin。
func reachesOrigin(facts map[string]*handlerFacts, fn string, seen map[string]bool) bool {
	if seen[fn] {
		return false
	}
	seen[fn] = true
	f := facts[fn]
	if f == nil {
		return false
	}
	if f.callsOrigin {
		return true
	}
	for c := range f.calls {
		if c == "isTrustedOrigin" {
			continue
		}
		if _, ok := facts[c]; ok && reachesOrigin(facts, c, seen) {
			return true
		}
	}
	return false
}

// TestEveryMutatingHandlerChecksOrigin 是這支測試的斷言。
//
// 它枚舉 httpapi 裡每一支「依 r.Method 進入寫入分支」的 handler，斷言每一支
// 都以其中一種方式守住來源檢查（見檔頭）。任何漏掉的守衛都會在這裡被列出，
// 失敗訊息會寫明症狀是「CSRF 防護失效但沒有任何錯誤」。
func TestEveryMutatingHandlerChecksOrigin(t *testing.T) {
	facts, wrapped := parseHandlerFacts(t)

	var gaps []string
	checked := 0
	for name, f := range facts {
		if !strings.HasPrefix(name, "handle") || !f.mutating {
			continue
		}
		checked++
		if wrapped[name] {
			continue // 整條路由被 requireTrustedOrigin 包住（/api/logout）
		}
		if reachesOrigin(facts, name, map[string]bool{}) {
			continue // 沿呼叫鏈可達 isTrustedOrigin
		}
		gaps = append(gaps, name+"  ("+f.pos+")")
	}

	if len(gaps) > 0 {
		sort.Strings(gaps)
		t.Errorf("下列會改變資料的 handler 沒有任何來源檢查（csrf.go 的不變條件）：\n  %s\n"+
			"症狀是「CSRF 防護失效但沒有任何錯誤」：受害者會自願發出跨站請求，"+
			"伺服器端看不到任何異常。修法是在 handler（或它呼叫的 helper）裡加 "+
			"if !s.isTrustedOrigin(r) { writeError(w, 403, ...); return }，"+
			"或在路由註冊處用 requireTrustedOrigin 包住。",
			strings.Join(gaps, "\n  "))
	}

	// 順便守住「這支測器本身有在做事」：一個把 mutating 判定寫壞到「偵測不到
	// 任何寫入端點」的版本，會讓上面的迴圈永遠空轉而測試長期呈綠。因此要求它
	// 至少找到一個已知的寫入端點。
	if checked == 0 {
		t.Fatal("沒有偵測到任何會改變資料的 handler —— mutating 的判定規則壞掉了，" +
			"這支測試等於沒有作用")
	}
}
