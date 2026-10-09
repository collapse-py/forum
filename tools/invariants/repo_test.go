/*
repo_test.go：對真實後端原始碼跑的三條斷言。

【為什麼需要它】
三條不變條件分析器都刻意只認「它看得懂的形状」：adminauth 要求註冊的接收者叫
mux、handler 是直接的方法值；auditcheck 以**方法名**比對。這些寬鬆是必要的
（否則規則會誤報，而誤報會讓整條規則被 //nolint 關掉），但它們的共同代價是：
真實碼一旦改用另一種形狀，規則會**安靜地不再檢查任何東西**，而 CI 全綠。

adminauth.go 的檔頭聲稱這個缺口由 analyzer_test.go 的一支測試釘住，
auditcheck.go 的常數區也聲稱 beginAdminTx 只有一個定義 —— 兩者在這個檔被
加上來之前都只是註解裡的願望。這個檔把它們變成事實。

【為什麼不直接對 backend 跑分析器】
那需要把 backend 放進本模組的套件路徑（它是獨立的 Go 模組，相依也不同）。
用 go/parser 讀原始碼只依賴 AST 形狀，而這正是規則本身的判斷依據 ——
同一件事的兩種做法，不需要第二套 build tag。
*/

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// backendHTTPAPIDir 是 analyzer 守護的那個套件的目錄（相對於本測試的工作目錄：
// `go test` 在 tools/invariants/ 底下執行）。
const backendHTTPAPIDir = "../../backend/forum/httpapi"

// parseBackendFiles 解析 backend/forum/httpapi 底下的所有 .go 檔。
//
// 找不到目錄時回傳 nil：這個模組可以被單獨 clone 出來跑，而那時沒有真實後端
// 可讀 —— 回傳 nil 讓呼叫端可以 t.Skip，而不是把「沒有 backend」誤報成
// 「backend 不合規則」。
func parseBackendFiles(t *testing.T) (*token.FileSet, []*ast.File) {
	t.Helper()
	entries, err := os.ReadDir(backendHTTPAPIDir)
	if err != nil {
		return nil, nil
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(backendHTTPAPIDir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("解析 %s 失敗: %v", entry.Name(), err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		return nil, nil
	}
	return fset, files
}

// TestAdminRoutesAreDirectMethodValues 釘住 adminauth 的漏報缺口。
//
// 規則只認兩種形狀：接收者叫 mux、handler 是 s.handleXxx 這種方法值。真實
// server.go 只要有一條 /api/admin/ 路由改成閉包或中介層（mux.Handle(path,
// requireAdmin(handleXxx))），那條路由就從此不被檢查 —— 而它仍然是對外開放的
// 一個後台端點。這個測試逐條斷言「每一條 /api/admin/ 註冊都能被
// adminRouteRegistration 認出來」，因此那種改裝一定會紅燈。
func TestAdminRoutesAreDirectMethodValues(t *testing.T) {
	fset, files := parseBackendFiles(t)
	if files == nil {
		t.Skip("找不到 backend/forum/httpapi：這個模組被單獨使用，略過對真實碼的斷言")
	}

	registered := 0
	recognized := 0
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			route, handler, ok := adminRouteRegistration(call)
			if !ok {
				return true
			}
			registered++
			recognized++
			if handler == "" {
				t.Errorf("%s: 路由 %s 取出空的 handler 名稱", fset.Position(call.Pos()), route)
			}
			return true
		})
	}

	if registered == 0 {
		t.Fatal("在 backend/forum/httpapi 裡找不到任何 /api/admin/ 註冊：" +
			"要嘛是路徑搬走了，要嘛是規則認不出任何一條路由 —— 兩者都讓 adminauth 形同虛設")
	}
	// 兩者相等代表「每條路由都被認出來」：adminRouteRegistration 對看不懂的形狀
	// 回 ok=false，因此 registered 與 recognized 的落差就是漏報的條數。
	if registered != recognized {
		t.Errorf("/api/admin/ 註冊 %d 條，其中只有 %d 條被規則認出來（%d 條被安靜跳過）",
			registered, recognized, registered-recognized)
	}
	t.Logf("backend/forum/httpapi 有 %d 條 /api/admin/ 註冊，全部都是規則認得出的形狀", registered)
}

// TestBeginAdminTxHasOneDefinition 釘住 auditcheck 以名稱比對的前提。
//
// 規則用方法名 beginAdminTx / recordAdminAction / Commit 比對，因為它要能對著
// testdata 的假套件跑。代價是：如果真實碼裡出現了第二個同名方法（另一個型別
// 上剛好也叫 beginAdminTx），規則仍會把它的交易也算進來 —— 而那個交易可能有
// 完全不同的語意。這個測試讓「只有一個定義」成為事實，而不是註解裡的一句話。
func TestBeginAdminTxHasOneDefinition(t *testing.T) {
	fset, files := parseBackendFiles(t)
	if files == nil {
		t.Skip("找不到 backend/forum/httpapi：這個模組被單獨使用，略過對真實碼的斷言")
	}
	_ = fset // 位置資訊在這支測試裡不重要：斷言的是「有幾個定義」而不是「在哪一行」

	definitions := 0
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == beginAdminTxName {
				definitions++
			}
		}
	}
	if definitions != 1 {
		t.Errorf("%s 在 backend/forum/httpapi 裡有 %d 個定義，want 1。"+
			"auditcheck 以方法名比對（見 auditcheck.go 的常數區），第二個同名方法會讓"+
			"規則把它的交易也當成管理員交易 —— 要嘛改名，要嘛把規則改成型別比對",
			beginAdminTxName, definitions)
	}
}
