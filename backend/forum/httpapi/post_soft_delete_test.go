/*
貼文的軟刪除不變條件（post_soft_delete_test.go）。

刪除改成軟刪除（MigrateMySQL 第 30 步）之後，程式裡多了一條**壞掉時不會出錯**
的規則：每一條讀取 forum_posts 的 SQL 都必須帶 deleted_at IS NULL。漏掉的症狀
不是錯誤，而是某一個頁面繼續顯示一篇作者已經按了刪除的貼文 —— 那種事沒有人
會回報，因為它看起來就像「哦，那篇還在」。

因此這支測試把規則釘死。判定的對象是**每一個資料庫呼叫的 SQL 參數**，而取得
SQL 文本的方式是常量折疊：

	  每個資料庫呼叫的 SQL 參數，必須滿足四件事之一

		1. 帶 deleted_at —— 正常情況。
		2. 是 INSERT INTO forum_posts —— 新建的貼文不可能是刪除狀態。
		3. 是只有 FROM 子句的片段（例如 forumPostFrom）—— WHERE 由呼叫端組裝，
		   而那個呼叫端在本測試的判定裡本身就要帶條件。
		4. 在下方白名單裡，且每一項都寫下「為什麼可以不過濾」。

【為什麼是常量折疊而不是配對反引號】

這個規則的第一版用「抽出所有反引號字串」實作，而它有一個安靜的洞：靠字串拼接
組出的 SQL 它一條也看不到。洞不是理論上的 —— handleForumReport 的存在性檢查
長這樣：

	table := "forum_posts"
	if targetType == "comment" {
		table = "forum_post_comments"
	}
	s.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM "+table+" WHERE id = ?", targetID)

`"forum_posts"` 是一個解譯字串、而且不在 SQL 字面值裡，掃反引號的版本於是整段
略過，規則對它等同不存在。改成 go/ast 之後，這種寫法會被折疊成它可能的樣子，
漏過濾因此會被抓到（本檔的 TestCheckFileFlagsUnfilteredQuery 就是釘住這件事的
負向測試）。

【為什麼折疊出的是片段集合，而不是整句】

`"SELECT ... FROM " + table + " WHERE " + condition` 裡的 table 與 condition 各有
兩種可能值，但它們的組合只有兩種是真的（同一個 if 分支裡一起改）。做笛卡爾積會
憑空造出 "FROM forum_posts WHERE id = ?" 這種從來不存在的句子，然後把它報成違規
—— 一個會誤報的規則會被關掉，而那時真正的漏過濾也沒人管了。因此這裡只做聯集：
只要有任何一個片段提到 forum_posts，就要求同一組片段裡看得到 deleted_at。

【這個判定的極限，以及為什麼仍然值得】

  - 只看得到常數字串。SQL 由 fmt.Sprintf 或不可推的變數組出時，折疊結果是空的，
    那筆呼叫不會被檢查。本套件目前沒有這種形狀（唯一的動態表名就是上面那個
    if/else），而新增的查詢幾乎都是字面值或簡單拼接。
  - 「有 deleted_at」與「過濾的是 forum_posts」之間沒有做連繫。一張表有
    deleted_at、另一張沒有的查詢會通過。論壇只有 forum_posts 有這個欄位。
  - 白名單是子字串比對，因此「順帶」匹配到的查詢也會被豁免。為此每一條例外都
    必須在本次執行中真的匹配到至少一條 SQL（見 TestForumPostQueriesFilterSoftDeleted
    結尾的檢查）—— 過期的例外會被報出來，而不是默默留在清單裡當免死金牌。

【為什麼掃原始碼而不是跑 SQL】

這個專案的 handler 測試沒有 MySQL（見 docs/KNOWN_ISSUES.md 的已知問題），而這條
規則的價值恰恰在「新增一條查詢時」—— 那時候沒有人會為了它起一個資料庫。CI 裡
每次 go test 都會跑到這一段。

刻意不做自動斷言「白名單只能縮不能長」：那份清單的成長應該是一次**review 時
看得見的決定**（diff 裡多一行與一個理由），而不是被一個測試擋住。這裡的作法
是讓每一個例外都必須附上理由字串，而理由字串本身就是給下一個人看的文件。
*/
package httpapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

/* ==========================================================================
   折疊
   ========================================================================== */

// dbMethods 是 database/sql 一族「參數裡帶 SQL」的方法。
//
// 以方法名比對而不是以接收者型別比對：這個套件的呼叫長得五花八門（s.db、
// tx、以及未來任何借來的 *sql.DB），而型別資訊在這個測試裡拿不到。名稱只有六
// 個，且全部來自 database/sql 的公開 API，撞名的風險由「折疊結果必須含有 SQL
// 關鍵字才會被檢查」這道閘門抵掉。
var dbMethods = map[string]bool{
	"Query": true, "QueryContext": true,
	"QueryRow": true, "QueryRowContext": true,
	"Exec": true, "ExecContext": true,
}

// dbCallSQLArg 回傳這個 call 的 SQL 參數運算式；不是資料庫呼叫則回 nil。
//
// ctx 先行是本套件的慣例（QueryContext(ctx, sql, args...)），因此 SQL 是第二個
// 參數；沒有 ctx 的舊式呼叫（Query(sql, args...)）用第一個。取第二個而非第一個
// 是為了讓「忘記傳 ctx」這種寫法不會把 ctx 折疊成 SQL —— 它折疊不出關鍵字，
// 於是那筆呼叫只會被跳過，不會被誤判。
func dbCallSQLArg(call *ast.CallExpr) ast.Expr {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !dbMethods[selector.Sel.Name] {
		return nil
	}
	if len(call.Args) >= 2 {
		return call.Args[1]
	}
	if len(call.Args) == 1 {
		return call.Args[0]
	}
	return nil
}

// maxFoldedFragments 是折疊結果的上限。
//
// 折疊的是聯集而不是笛卡爾積，正常情況下一個變數只有兩三個可能值；這個上限只是
// 為了讓「某個常量引用鏈意外形成大量組合」時不會把測試變成記憶體炸彈。到達上限
// 時直接截斷：漏掉的可能性只會讓規則更嚴格（看得到的片段變少），不會讓它放水。
const maxFoldedFragments = 32

// foldStringExpr 收集一個運算式可能貢獻的**字面值片段**。
//
// 認得四種形狀：字面值、已記錄的識別元（區域變數或套件常量）、括號、以及字串
// 相加。其餘一律回 nil（未知）—— 包括 fmt.Sprintf 這類呼叫，它們的回傳值無法
// 在沒有執行期的情況下知道，而硬猜一個值比承認不知道更危險。
func foldStringExpr(expr ast.Expr, consts, locals map[string][]string) []string {
	switch node := expr.(type) {
	case *ast.BasicLit:
		if node.Kind != token.STRING {
			return nil
		}
		value, err := strconv.Unquote(node.Value)
		if err != nil {
			// 解不開的字面值（理論上不存在：Go 原始碼一定可解析）視為未知。
			return nil
		}
		return []string{value}
	case *ast.Ident:
		// 區域變數優先於常量：同名的話，函式內的那個才是 SQL 真正看到的。
		if values, ok := locals[node.Name]; ok {
			return values
		}
		return consts[node.Name]
	case *ast.ParenExpr:
		return foldStringExpr(node.X, consts, locals)
	case *ast.BinaryExpr:
		if node.Op != token.ADD {
			return nil
		}
		return unionFragments(foldStringExpr(node.X, consts, locals), foldStringExpr(node.Y, consts, locals))
	}
	return nil
}

// unionFragments 聯集兩個片段集合（去重、保序、有上限）。
func unionFragments(left, right []string) []string {
	merged := make([]string, 0, len(left)+len(right))
	seen := make(map[string]bool, len(left)+len(right))
	for _, value := range append(append([]string{}, left...), right...) {
		if seen[value] || len(merged) >= maxFoldedFragments {
			continue
		}
		seen[value] = true
		merged = append(merged, value)
	}
	return merged
}

// crossFragments 做笛卡爾積串接，同樣有上限。
//
// 只有 `+=` 用得到它：累加的最終值是「舊值 × 新值」的每一種組合，只記新值會讓
// 折疊結果看不到舊值裡的條件（例如 filters += " AND " + condition 之後，
// forumPostVisible 就從折疊結果裡消失了）。
func crossFragments(left, right []string) []string {
	merged := make([]string, 0, len(left)*len(right))
	for _, a := range left {
		for _, b := range right {
			if len(merged) >= maxFoldedFragments {
				return merged
			}
			merged = append(merged, a+b)
		}
	}
	return merged
}

// packageStringConsts 收集套件級 string 常量的值。
//
// 依宣告順序處理，因此常量引用先宣告的常量時也折疊得出來（forumPostFrom 引用
// forumPostProjection 這種形狀）。非字串常量（iota 計數、數字）不會有條目，
// 引用它們的識別元因此是「未知」而不是「空字串」。
func packageStringConsts(file *ast.File) map[string][]string {
	values := map[string][]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, value := range valueSpec.Values {
				fragments := foldStringExpr(value, values, nil)
				for _, name := range valueSpec.Names {
					values[name.Name] = unionFragments(values[name.Name], fragments)
				}
			}
		}
	}
	return values
}

// localStringValues 收集函式內區域字串變數的所有可能值。
//
// consts 必須一起傳進來：`filters := forumPostVisible` 這種「區域變數的值來自
// 套件常量」是這個套件組 WHERE 的標準寫法，看不到常量的話折疊結果會是空的，
// 於是整句查詢在判定時失去 deleted_at 那一邊 —— 那不是漏過濾，是折疊器自己瞎了。
//
// 刻意對 `=` 與 `:=` 都用**累加**而不是覆蓋：這個變數在 if 分支裡被改成別的表名
// 時，「它原來也是 forum_posts」必須留在集合裡，否則呼叫端折疊出來的 SQL 會
// 少掉一種可能 —— 而那正是 handleForumReport 的洞。代價是「先給 A 再改成 B」的
// 寫法會被記住兩個值，而這只會讓規則更嚴格（更多片段會被檢查），不會放水。
//
// 只認「單一識別元 = 單一運算式」：多重指定與解構在 SQL 組裝裡不存在，認了只會
// 增加出錯的面。
func localStringValues(body *ast.BlockStmt, consts map[string][]string) map[string][]string {
	values := map[string][]string{}
	ast.Inspect(body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		name, ok := assign.Lhs[0].(*ast.Ident)
		if !ok || name.Name == "_" {
			return true
		}
		rhs := foldStringExpr(assign.Rhs[0], consts, values)
		if assign.Tok == token.ADD_ASSIGN {
			values[name.Name] = unionFragments(values[name.Name], crossFragments(values[name.Name], rhs))
			return true
		}
		values[name.Name] = unionFragments(values[name.Name], rhs)
		return true
	})
	return values
}

/* ==========================================================================
   白名單：每一項都要有理由
   ========================================================================== */

// softDeleteExceptions 是「讀 forum_posts 卻可以不帶 deleted_at」的清單。
//
// 每一項由兩個部分組成：一個用來辨識這條 SQL 的子字串，以及一句**給下一個人
// 看**的理由。新增例外時請連同理由一起寫 —— 沒有理由的例外就是下一則 bug。
//
// 每一條都必須在執行中真的匹配到至少一條 SQL：匹配不到的例外代表它已經過期
// （對應的查詢被改掉了），測試會直接失敗（見 TestForumPostQueriesFilterSoftDeleted
// 的尾段）。少了這道檢查，白名單會慢慢長成一份沒人記得為什麼的免死名單。
var softDeleteExceptions = []struct {
	marker string
	reason string
}{
	{
		// 檢舉佇列（列表與單筆各一條）。被檢舉的貼文也許已經被刪，而覆核者
		// 要看的正是「當時被檢舉的是什麼」。目標真的不存在時，COALESCE 會
		// 給出空字串，前端據此顯示「內容已不存在」。
		marker: "FROM forum_reports",
		reason: "檢舉工單要保留被檢舉內容的原文給覆核者看，即使它已經被刪除",
	},
	{
		// forumEmailByPublicKey 的退路掃描。它回答「這個 public key 是誰」，
		// 而不是「他有哪些貼文可見」；把刪光貼文的使用者排除會讓他的個人頁
		// 連結與既有追蹤關係一起失效。
		marker: "SELECT DISTINCT author_email FROM forum_posts",
		reason: "public key → email 的退路解析與貼文可見性無關",
	},
}

/* ==========================================================================
   不變條件
   ========================================================================== */

// sqlKeywords 是「這段字串看起來像 SQL」的最低門檻。
//
// 為什麼需要它：裸表名（`table := "forum_posts"`）也是提到 forum_posts 的字串，
// 但它不是一條查詢，要求它帶 deleted_at 是荒謬的。反過來說，任何真的被組進查詢
// 的片段都至少含有一個子句關鍵字，因此這道閘門不會漏掉真正的 SQL —— 組裡有任一
// 片段像 SQL，整組就被當成 SQL 判定。
var sqlKeywords = []string{"SELECT", "INSERT", "UPDATE", "DELETE", "FROM ", "WHERE "}

func looksLikeSQL(fragment string) bool {
	for _, keyword := range sqlKeywords {
		if strings.Contains(fragment, keyword) {
			return true
		}
	}
	return false
}

// checkSQLFragments 對一組片段套用規則，回傳「檢查了幾筆」、違規敘述、以及命中的
// 例外理由。
//
// 判定的單位是**整組**而不是單一片段：`"SELECT ... FROM " + table + " WHERE " + cond`
// 折疊出來的是幾個片段，而「這條查詢有沒有過濾」是整條的事 —— 只要組裡任何一個
// 片段看得到 deleted_at，這條查詢就過濾了。反過來說，表名可能單獨住在一個片段裡
// （handleForumReport 的 table 變數），若要求「提到 forum_posts 的那個片段自己
// 帶條件」，那種寫法永遠抓不到。
//
// allowFromFragment 為 true 時，「只有 FROM 子句的片段」合法（WHERE 由呼叫端組裝）。
// 只有第二輪（不在資料庫呼叫參數裡的裸字面值，例如 forumPostFrom 常量）該傳 true：
// 第一輪看得到整句，因此一個沒有配上 WHERE 的 FROM 片段就是漏過濾，不是例外。
func checkSQLFragments(fragments []string, allowFromFragment bool) (checked int, problems []string, matched []string) {
	var sawSQL bool
	for _, fragment := range fragments {
		if looksLikeSQL(fragment) {
			sawSQL = true
			break
		}
	}
	if !sawSQL {
		// 沒有任何一個片段像 SQL：這是裸表名、別名條件之類的碎片，不是一條查詢。
		return 0, nil, nil
	}
	hasDeletedAt := false
	for _, fragment := range fragments {
		if strings.Contains(fragment, "deleted_at") {
			hasDeletedAt = true
			break
		}
	}
	for _, fragment := range fragments {
		if !strings.Contains(fragment, "forum_posts") {
			continue
		}
		exempt := ""
		for _, exception := range softDeleteExceptions {
			if strings.Contains(fragment, exception.marker) {
				exempt = exception.reason
				break
			}
		}
		if exempt != "" {
			matched = append(matched, exempt)
			continue
		}
		if strings.Contains(fragment, "INSERT INTO forum_posts") {
			// 新建的貼文 deleted_at 是 NULL，沒有東西需要過濾。
			continue
		}
		if allowFromFragment && !strings.Contains(fragment, "SELECT") &&
			strings.Contains(fragment, "FROM forum_posts") {
			// FROM 子句片段（forumPostFrom）：WHERE 由呼叫端組裝。
			continue
		}
		checked++
		if !hasDeletedAt {
			problems = append(problems, "    "+singleLineSQL(fragment))
		}
	}
	return checked, problems, matched
}

// checkFile 對單一檔案套用規則，回傳「檢查了幾筆」與違規清單。
//
// usedExceptions 由呼叫端持有並跨檔案累計，用來找出再也匹配不到任何 SQL 的死例外。
func checkFile(fset *token.FileSet, name string, file *ast.File, usedExceptions map[string]bool) (checked int, problems []string) {
	consts := packageStringConsts(file)

	// 第一輪：每一個資料庫呼叫的 SQL 參數，整體折疊後判定。
	// 這是主要的一輪 —— 拼接、常量、區域變數都在這裡被還原。
	var sqlArgRanges [][2]token.Pos
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		locals := localStringValues(function.Body, consts)
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sqlArg := dbCallSQLArg(call)
			if sqlArg == nil {
				return true
			}
			sqlArgRanges = append(sqlArgRanges, [2]token.Pos{sqlArg.Pos(), sqlArg.End()})
			fragments := foldStringExpr(sqlArg, consts, locals)
			if len(fragments) == 0 {
				return true
			}
			fragmentChecked, fragmentProblems, matched := checkSQLFragments(fragments, false)
			for _, reason := range matched {
				usedExceptions[reason] = true
			}
			checked += fragmentChecked
			for _, problem := range fragmentProblems {
				problems = append(problems, name+": 有一條讀取 forum_posts 的 SQL 沒有過濾軟刪除:\n"+
					problem+"\n\n"+
					"    軟刪除的貼文在任何讀取端都必須不存在（MigrateMySQL 第 30 步）。"+
					"加上 deleted_at IS NULL，或把它加進 softDeleteExceptions 並寫下理由。")
			}
			return true
		})
	}

	// 第二輪：沒有直接進資料庫呼叫的 SQL 字面值。
	//
	// 這一輪是為「SQL 住在常量或 helper 函式裡」的形狀留的：那種字面值不在任何
	// 呼叫參數中，第一輪看不到它。第一輪已經整體判定過的區間刻意跳過 —— 同一個
	// 片段被看兩次沒有好處，而「半句在上一個字面值、條件在下一個」的拼法會被
	// 這一輪誤判（那正是第一輪存在的理由）。
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		for _, span := range sqlArgRanges {
			if literal.Pos() >= span[0] && literal.End() <= span[1] {
				return true
			}
		}
		fragments := foldStringExpr(literal, consts, nil)
		if len(fragments) == 0 {
			return true
		}
		literalChecked, literalProblems, matched := checkSQLFragments(fragments, true)
		for _, reason := range matched {
			usedExceptions[reason] = true
		}
		checked += literalChecked
		for _, problem := range literalProblems {
			problems = append(problems, name+": 有一段提到 forum_posts 的 SQL 沒有過濾軟刪除:\n"+
				problem+"\n\n"+
				"軟刪除的貼文在任何讀取端都必須不存在（MigrateMySQL 第 30 步）。"+
				"加上 deleted_at IS NULL，或把它加進 softDeleteExceptions 並寫下理由。")
		}
		return true
	})

	return checked, problems
}

// singleLineSQL 把多行 SQL 壓成一行，讓錯誤訊息可以在終端機裡讀完。
func singleLineSQL(sql string) string {
	fields := strings.Fields(sql)
	return strings.Join(fields, " ")
}

// TestForumPostQueriesFilterSoftDeleted 釘死上面那條規則。
func TestForumPostQueriesFilterSoftDeleted(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("列不出套件檔案: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("一個 .go 都找不到 —— 這個測試的工作目錄不對，它什麼都沒驗")
	}

	usedExceptions := map[string]bool{}
	checked := 0
	for _, name := range files {
		// 測試檔不掃：它們沒有 SQL，而掃了只會讓規則的適用範圍變得含糊。
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		// SkipObjectResolution：這個測試只讀語法樹，不需要型別資訊，省掉解析
		// 另一半的成本。parse 失敗本身就要報 —— 那代表原始碼不是合法 Go，
		// 而不是「這條規則對它不適用」。
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Errorf("解析 %s 失敗（規則因此完全沒有檢查它）: %v", name, err)
			continue
		}
		fileChecked, problems := checkFile(fset, name, file, usedExceptions)
		checked += fileChecked
		for _, problem := range problems {
			t.Error(problem)
		}
	}
	if checked == 0 {
		t.Error("一條該檢查的 SQL 都沒有 —— 比對條件失效了（檔名規則改了？）")
	}

	// 白名單不允許有死條目：一條匹配不到任何 SQL 的例外，要嘛是它守住的查詢被
	// 改成了別種形狀（那條查詢現在過不過濾必須重新確認），要嘛是它從一開始就
	// 寫錯 marker。兩種情況都該在 diff 裡被看見，而不是留在清單裡。
	for _, exception := range softDeleteExceptions {
		if !usedExceptions[exception.reason] {
			t.Errorf("softDeleteExceptions 的 %q 沒有匹配到任何 SQL —— 這條例外已經過期，"+
				"請刪除它，或修正 marker", exception.marker)
		}
	}
}

/*
TestCheckFileFlagsUnfilteredQuery 是這條規則的負向測試。

它存在的理由很直接：一條「全都通過」的不變條件，與一條「什麼都沒檢查」的
不變條件，在輸出上看起來一模一樣。因此在真實碼庫全綠之外，必須有一個案例
證明規則真的會抓人 —— 而抓的正是它第一版漏掉的那種形狀（表名放在變數裡、
SQL 靠拼接組出）。
*/
func TestCheckFileFlagsUnfilteredQuery(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		wantFlag bool
	}{
		{
			name: "裸字面值漏過濾",
			src: `package p
			func q(db *sql.DB, id int64) {
				db.QueryRowContext(ctx, "SELECT COUNT(*) FROM forum_posts WHERE id = ?", id)
			}`,
			wantFlag: true,
		},
		{
			name: "表名放在變數裡、SQL 靠拼接（第一版的洞）",
			src: `package p
			func q(db *sql.DB, id int64, isComment bool) {
				table := "forum_posts"
				if isComment {
					table = "forum_post_comments"
				}
				db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE id = ?", id)
			}`,
			wantFlag: true,
		},
		{
			name: "條件放在另一個變數裡",
			src: `package p
			func q(db *sql.DB, id int64) {
				where := "id = ?"
				db.QueryContext(ctx, "SELECT id FROM forum_posts WHERE "+where, id)
			}`,
			wantFlag: true,
		},
		{
			name: "同一句裡有 deleted_at 就放行",
			src: `package p
			func q(db *sql.DB, id int64) {
				db.QueryContext(ctx, "SELECT id FROM forum_posts WHERE deleted_at IS NULL AND id = ?", id)
			}`,
			wantFlag: false,
		},
		{
			name: "表名是變數但條件帶著過濾",
			src: `package p
			func q(db *sql.DB, id int64, isComment bool) {
				table := "forum_posts"
				where := "id = ? AND deleted_at IS NULL"
				if isComment {
					table = "forum_post_comments"
					where = "id = ?"
				}
				db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE "+where, id)
			}`,
			wantFlag: false,
		},
		{
			name: "INSERT 不可能是刪除狀態",
			src: `package p
			func q(db *sql.DB) {
				db.ExecContext(ctx, "INSERT INTO forum_posts (author_email, content) VALUES (?, ?)", a, c)
			}`,
			wantFlag: false,
		},
		{
			name: "裸表名不是 SQL",
			src: `package p
			func q(db *sql.DB, isComment bool) {
				table := "forum_posts"
				if isComment {
					table = "forum_post_comments"
				}
				recordAudit(table)
			}`,
			wantFlag: false,
		},
		{
			name: "表名接上 SELECT 卻沒有 WHERE",
			src: `package p
			func q(db *sql.DB, isComment bool) {
				table := "forum_posts"
				if isComment {
					table = "forum_post_comments"
				}
				db.QueryContext(ctx, "SELECT COUNT(*) FROM "+table)
			}`,
			wantFlag: true,
		},
		{
			name: "FROM 片段由呼叫端組裝 WHERE",
			src: `package p
			const fromClause = "FROM forum_posts fp LEFT JOIN forum_profiles pr ON pr.author_email = fp.author_email"
			func q(db *sql.DB) {
				db.QueryContext(ctx, "SELECT fp.id, fp.content "+fromClause+" WHERE fp.deleted_at IS NULL")
			}`,
			wantFlag: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "sample.go", tc.src, parser.SkipObjectResolution)
			if err != nil {
				// 上面的樣例刻意不 import database/sql 也不宣告 ctx：parse 只管
				// 語法，不管名字有沒有定義，因此這裡失敗代表樣例寫錯了。
				t.Fatalf("解析樣例失敗: %v", err)
			}
			_, problems := checkFile(fset, "sample.go", file, map[string]bool{})
			if gotFlag := len(problems) > 0; gotFlag != tc.wantFlag {
				t.Errorf("檢查結果 = %v（%d 筆問題），want 違規 = %v\n%v",
					gotFlag, len(problems), tc.wantFlag, strings.Join(problems, "\n"))
			}
		})
	}
}

/*
TestFoldStringExprResolvesLocalsAndConsts 釘住折疊器本身。

規則的可信度建立在「折疊器認得出 SQL 長什麼樣」上，而它一旦退化（例如把區域
變數當成未知、於是任何拼接都放行），上面的不變條件會**靜靜地變成永遠綠** ——
那比沒有它更糟。每一個案例都對應一種會讓折疊失敗的寫法。
*/
func TestFoldStringExprResolvesLocalsAndConsts(t *testing.T) {
	src := `package p

const base = "FROM forum_posts fp"

func build(cond string, extra string) string {
	table := "forum_posts"
	if cond != "" {
		table = "forum_post_comments"
	}
	filters := "fp.deleted_at IS NULL"
	filters += " AND " + extra
	return "SELECT fp.id " + base + " WHERE " + filters + " LIMIT ?"
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("解析樣例失敗: %v", err)
	}
	consts := packageStringConsts(file)
	if got := consts["base"]; len(got) != 1 || got[0] != "FROM forum_posts fp" {
		t.Fatalf("常量 base 折疊成 %q，want [\"FROM forum_posts fp\"]", got)
	}

	var body *ast.BlockStmt
	for _, decl := range file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok && function.Name.Name == "build" {
			body = function.Body
		}
	}
	if body == nil {
		t.Fatal("找不到 build 函式")
	}
	locals := localStringValues(body, consts)

	for name, want := range map[string][]string{
		"table": {"forum_posts", "forum_post_comments"},
		// += 必須保留舊值：只記新值的話折疊結果會看不到 deleted_at，
		// 而那正好是 loadForumPosts 的寫法。
		"filters": {"fp.deleted_at IS NULL", "fp.deleted_at IS NULL AND "},
	} {
		got := locals[name]
		if len(got) != len(want) {
			t.Fatalf("區域變數 %s 折疊成 %q，want %q", name, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("區域變數 %s 的第 %d 個值 = %q，want %q", name, i, got[i], want[i])
			}
		}
	}
}

/*
TestLooksLikeSQLSeparatesTableNamesFromQueries 釘住「看起來像 SQL」那道閘門。

少了它，`table := "forum_posts"` 這種裸表名會被當成漏過濾的查詢 —— 規則會在第
一天就被人關掉。因此閘門本身也要有測試，而且兩個方向都要測。
*/
func TestLooksLikeSQLSeparatesTableNamesFromQueries(t *testing.T) {
	notSQL := []string{
		"forum_posts",
		"forum_post_comments",
		"id = ? AND deleted_at IS NULL",
		"fp.deleted_at IS NULL",
	}
	for _, fragment := range notSQL {
		if looksLikeSQL(fragment) {
			t.Errorf("%q 不該被當成 SQL", fragment)
		}
	}
	isSQL := []string{
		"SELECT COUNT(*) FROM forum_posts WHERE id = ?",
		"UPDATE forum_posts SET deleted_at = ?",
		"FROM forum_posts fp",
		"WHERE fp.deleted_at IS NULL",
	}
	for _, fragment := range isSQL {
		if !looksLikeSQL(fragment) {
			t.Errorf("%q 應該被當成 SQL", fragment)
		}
	}
}
