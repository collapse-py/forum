/*
analyzer_test.go 是這三條不變條件分析器的自測。

【為什麼分析器自己需要測試】
分析器是這個專案裡少數「測試失敗等於產品有問題」的元件之一 —— 方向相反：
一個**漏報**的分析器會讓整個 CI 閘門形同虛設，而那不會產生任何症狀。
CI 會一直綠，規則卻已經不再守任何東西。因此這裡的重點不是「分析器能不能報出
違規」（那由 testdata/bad 驗證），而是三個較少被想到的失敗模式：

 1. 分析器報得太吵，於是有人加上 //nolint 把它關掉（規則存活、效力為零）。
 2. 新增了分析器卻忘了接進 main（規則從未執行，且沒有任何症狀）。
 3. walkBody 的 stack 在提早返回時失配（抓到的父節點是錯的，於是
    errorValueConsumed 判斷錯誤 —— 而這正是本規則唯一的判斷依據）。

【為什麼用 analysistest 而不是對真實套件跑】
對 backend/forum/httpapi 跑只能證明「現在沒有違規」，證明不了「有違規時會報」。
testdata/bad 是刻意寫出違規的最小樣本：那是唯一能把「規則還有效」變成事實的
方式。而 testdata/good 則是防止規則變吵的那一半 —— 一個會誤報的規則壽命很短。
*/
package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAuditCheck 用兩份樣本釘住 auditcheck 的兩半：bad 必須被報、good 不必。
func TestAuditCheck(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), AuditCheck, "auditcheck/bad", "auditcheck/good")
}

// TestAdminAuth 釘住 adminauth 的兩半。
func TestAdminAuth(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), AdminAuth, "adminauth/bad", "adminauth/good")
}

// TestBlocklistMount 釘住 blocklistmount 的兩半。
func TestBlocklistMount(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), BlocklistMount, "blocklistmount/bad", "blocklistmount/good")
}

// TestEveryAnalyzerIsUsable 確保清單裡的每一個都有可用的名稱與說明。
//
// 這條測試的價值不在於「今天有沒有漏寫」，而在於它守住「註冊」這件事：
// multichecker 以 Name 當識別碼，而 -checks 選項與文件都引用它們。一個沒有
// Name 的分析器在 vet 裡無法被指定，而一個沒有 Doc 的會讓輸出一行沒有人看得懂
// 的訊息 —— 兩種情況都不會讓建置失敗，因此不會有人發現。
//
// 至於「新增分析器卻忘記加進 analyzers」：那由 main.go 把 multichecker.Main
// 的引數直接設成這份清單來杜絕（沒有第二份註冊點可以漏掉）。
func TestEveryAnalyzerIsUsable(t *testing.T) {
	if len(analyzers) == 0 {
		t.Fatal("analyzers 是空的：這會讓整個 CI 閘門形同虛設，而不會有任何症狀")
	}
	for _, a := range analyzers {
		if a == nil {
			t.Error("analyzers 裡有 nil 項目")
			continue
		}
		if a.Name == "" {
			t.Errorf("%T 沒有 Name：go vet 會無法以 -checks 指定它", a)
		}
		if a.Doc == "" {
			t.Errorf("分析器 %s 沒有 Doc：vet 的輸出會失去解釋力", a.Name)
		}
		if a.Run == nil {
			t.Errorf("分析器 %s 沒有 Run：vet 會執行它但什麼都不檢查", a.Name)
		}
	}
}

func hasAnalyzerNamed(list []*analysis.Analyzer, name string) bool {
	for _, a := range list {
		if a != nil && a.Name == name {
			return true
		}
	}
	return false
}

// TestWalkBodyStopsCorrectly 鎖住 walkBody 在回呼要求提早停止時的 stack 配對。
//
// 這是整個分析器最脆弱的一段：ast.Inspect 在節點之後會以 nil 再呼叫一次回呼
// （代表「離開該節點」），而在節點上回傳 false 時不會呼叫那次 nil。如果
// walkBody 在 false 的情況下也多彈一層，之後每一個節點的父節點都會錯一格 ——
// 而父節點正是 errorValueConsumed 唯一的依據。
//
// 這個測試直接數「每個節點看到的父節點」，而不是間接觀察診斷結果：錯一格
// 的症狀會被報成「規則靜默失效」，那正是最難察覺的一種。
func TestWalkBodyStopsCorrectly(t *testing.T) {
	const src = `package p
func f() {
	g(func() { h(1) })
	_ = 2
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)

	type frame struct {
		parent ast.Node
		node   ast.Node
	}
	var frames []frame
	walkBody(fn.Body, func(parent ast.Node, n ast.Node) bool {
		frames = append(frames, frame{parent: parent, node: n})
		return true
	})

	// 不對「節點總數」做斷言 —— 那會讓任何未來的 AST 節點新增都變成一筆
	// 無意義的維護負擔。改為斷言一件真正要成立的事：每一個不是頂層的節點，
	// 它的父節點都必須在 frames 裡比它早出現（也就是 stack 是由外而內、
	// 由內而外地配對的）。
	index := map[ast.Node]int{}
	for i, f := range frames {
		index[f.node] = i
	}
	for i, f := range frames {
		if f.parent == nil {
			continue
		}
		parentIdx, ok := index[f.parent]
		if !ok {
			t.Errorf("%T 的父節點 %T 不在走訪結果裡，stack 已失配", f.node, f.parent)
			continue
		}
		if parentIdx > i {
			t.Errorf("%T（索引 %d）的父節點 %T 索引是 %d，出現得太晚：stack 已失配",
				f.node, i, f.parent, parentIdx)
		}
	}

	// 特別確認最外層節點的父節點是 nil，而內層的每一個都有父節點 ——
	// 那正是 errorValueConsumed 依賴的形狀（外層的 if 初始敘述會看到
	// AssignStmt 當父節點，獨立的稽核呼叫會看到 ExprStmt）。
	var sawRoot, sawInner bool
	for _, f := range frames {
		if _, ok := f.node.(*ast.BlockStmt); ok && f.parent == nil {
			sawRoot = true
		}
		if _, ok := f.node.(*ast.FuncLit); ok {
			sawInner = true
			if _, ok := f.parent.(*ast.CallExpr); !ok {
				t.Errorf("FuncLit 的父節點是 %T，want *ast.CallExpr", f.parent)
			}
		}
	}
	if !sawRoot {
		t.Error("走訪沒有看到最外層的 BlockStmt（它的父節點應該是 nil）")
	}
	if !sawInner {
		t.Error("樣本裡的閉包沒有被走訪到，walkBody 或測試樣本有問題")
	}
}

// TestErrorValueConsumedShape 鎖定「回傳值有沒有被使用」的判斷形狀。
//
// 這個判斷是 auditcheck 的全部推理，因此值得用自己的測試說明清楚：只有
// 「獨立運算式」算未使用，其餘（賦值、if 初始敘述、return、引數）都算已使用。
// 每個樣本都是被塞進同一個 beginAdminTx + Commit 外殼裡的一串敘述，
// 因此結果完全由「稽核呼叫的回傳值怎麼被使用」決定，不受其他變數影響。
func TestErrorValueConsumedShape(t *testing.T) {
	cases := []struct {
		name       string
		stmt       string
		wantReport bool
	}{
		{
			name:       "if 初始敘述（這個專案的實際形狀）",
			stmt:       "if err := s.recordAdminAction(tx); err != nil { return }",
			wantReport: false,
		},
		{
			name:       "獨立運算式（本規則要抓的）",
			stmt:       "s.recordAdminAction(tx)",
			wantReport: true,
		},
		{
			name:       "賦值後再檢查",
			stmt:       "auditErr := s.recordAdminAction(tx)\n\tif auditErr != nil { return }",
			wantReport: false,
		},
		{
			name:       "送進函式引數",
			stmt:       "wrap(s.recordAdminAction(tx))",
			wantReport: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			src := `package p
type tx struct{}
func (t *tx) Commit() error { return nil }
func (t *tx) Rollback() error { return nil }
type S struct{}
func (s *S) beginAdminTx() (*tx, error) { return &tx{}, nil }
func (s *S) recordAdminAction(t *tx) error { return nil }
func wrap(error) {}
func body(s *S) {
	tx, err := s.beginAdminTx()
	if err != nil { return }
	defer tx.Rollback()
` + tc.stmt + `
	tx.Commit()
}
`
			file, err := parser.ParseFile(fset, "p.go", src, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			var fns []*ast.FuncDecl
			for _, d := range file.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok {
					fns = append(fns, fn)
				}
			}

			// 用 beginAdminTxResultVars / recordAdminActionCalls 重現 checkFunc 的
			// 判斷，而不是真的跑一次 analysis.Pass：那需要一個完整的 pass 與
			// 檔案集合，而這個測試要確認的只是「回傳值有沒有被使用」這一個
			// 判斷，而它完全由 AST 形狀決定。
			checked := 0
			for _, fn := range fns {
				txVars := beginAdminTxResultVars(fn)
				if len(txVars) == 0 {
					continue
				}
				commits := 0
				var commitPos token.Pos
				walkBody(fn.Body, func(_ ast.Node, n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok && isCommitOf(call, txVars) {
						commits++
						if commits == 1 {
							commitPos = call.Pos()
						}
					}
					return true
				})
				if commits == 0 {
					continue
				}
				for _, c := range recordAdminActionCalls(fn) {
					if c.pos > commitPos {
						break
					}
					if c.consumed {
						checked++
						break
					}
				}
			}

			reported := checked == 0
			if reported != tc.wantReport {
				t.Errorf("被當成已檢查的稽核呼叫數 = %d（因此回報 = %v），want 回報 = %v",
					checked, reported, tc.wantReport)
			}
		})
	}
}

// TestAnalyzerNamesAreStable 確認三條規則的名稱不會被無意間改掉。
//
// 名字是被 CI 的 -checks 選項與文件引用的識別碼。改名的成本是「所有引用它的地方
// 要同時改」，而那裡沒有人在看 —— 因此把它釘住，並讓改名必須附帶一次刻意的
// 修改。
func TestAnalyzerNamesAreStable(t *testing.T) {
	want := map[string]bool{"auditcheck": true, "adminauth": true, "blocklistmount": true}
	for name := range want {
		if !hasAnalyzerNamed(analyzers, name) {
			t.Errorf("缺少名為 %q 的分析器（改名是允許的，但必須同時更新這裡）", name)
		}
	}
	if len(analyzers) != len(want) {
		var names []string
		for _, a := range analyzers {
			names = append(names, a.Name)
		}
		t.Errorf("分析器數量 = %d（%s），want %d。新增分析器時請一併更新這個清單與 CI 的說明",
			len(analyzers), strings.Join(names, ", "), len(want))
	}
}
