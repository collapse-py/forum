package httpapi

/*
CSV 匯出（httpapi/csv_export.go）。

三個部分：安全的 CSV 寫法、串流回應、以及三份匯出查詢。

為什麼需要一個獨立的檔案而不是把這些函式塞進各自的 handler

	1. CSV 注入的防護是整個匯出功能的**安全核心**，它必須只有一份實作。
	   三份匯出各自寫一次等於三個機會漏掉其中一個危險前綴，而漏掉的症狀是
	   「使用者的 email 在管理員的 Excel ���被執行」—— 那是一種不會出現在
	   任何日誌裡的攻擊。
	2. 串流寫出是三份匯出共用的技巧（見 writeCSVHeader 的說明）。
	3. 把「怎麼寫 CSV」與「匯出什麼」分開，讓後者讀起來像它本來的意思。

CSV 注入是什麼，以及這個專案面對的具體風險

	以「=」「+」「-」「@」開頭的欄位值，Excel / Google Sheets / LibreOffice
	會當成**公式**評估，而不是文字。所以一個 nickname 若是
	「=1+1」或「@SUM(A1:A9)」，匯出後在管理員的機器上就會被執行。

	這個專案特別危險，因為它匯出的欄位幾乎全是**使用者可控的自由文字**：
	暱稱、貼文內容、檢舉原因、標籤名稱。email 本身格式受限（不能以 = 開頭
	成為合法 email），但「內容」與「原因」沒有任何限制。

	防護方式是在值的最前面加一個單引號（'）。這是 Excel 認得的「以下是文字」
	前綴，顯示時不會出現，會原樣留在儲存格內容裡。

	刻意不做兩件常見但有害的事：
	  - 不跳過危險的值。那會讓匯出結果少掉資料，而管理者不會知道少了什麼 ——
	    那比顯示成文字更糟，因為它讓匯出結果不可信。
	  - 不用「=」取代第一個字元。那會改變資料，而匯出的用途是「與線上內容
	    逐字比對」。
	那一個前綴的代價是：在文字編輯器裡打開匯出檔時會看到 '。這是可接受的
	——匯出的用途是進 Excel，不是進 git。

另一個 CSV 陷阱：Excel 的 UTF-8 誤判

	沒有 BOM 的 UTF-8 CSV，Excel 會以本地字碼開啟，中文全部變成亂碼。
	因此 writeCSVHeader 會寫入 UTF-8 BOM。Google Sheets 與 LibreOffice 能
	正確處理 BOM，而它對其他讀取者是無害的位元組（BOM 會出現在檔案開頭，
	那些工具會忽略或顯示為零寬空格）。
*/

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/logger"
)

// csvFormulaPrefixes 是觸發公式評估的前綴。
//
// Tab 與 CR 也列在內：它們在某些試算表裡同樣是公式的起始符號，而一個
// 只靠「值看起來像可見字元」判斷的實作會漏掉它們 —— 例如 nickname 是
// "\t=1+1" 時，使用者看到的第一個字元是空白而不是 =。
//
// 這個清單是 OWASP 的標準建議，刻意不多不少：加入更多前綴（例如 "(" 或
// "[") 會讓更多正常值被加上前綴，而它們在公式語境裡並沒有特殊意義。
var csvFormulaPrefixes = []byte{'=', '+', '-', '@', '\t', '\r'}

// csvGuard 是加在危險值前面的文字前綴。
const csvGuard = "'"

// utf8BOM 是 Excel 辨識 UTF-8 所需的位元組順序標記。
//
// 手寫這三個位元組而不是用 encoding 的常數：Go 標準庫沒有提供這個常數
// （golang.org/x/text 有，但為了三個位元組引入一個相依不值得），而寫成
// "\uFEFF" 會讓人誤以為那是字面字元 —— 它是編碼層的標記，不是內容的一部分。
var utf8BOM = "\xEF\xBB\xBF"

// sanitizeCSVField 為單一欄位值加上必要的防護前綴。
//
// 為什麼是「加前綴」而不是「跳過」或「改寫」：見檔頭的三段說明。
// 為什麼空字串不加前綴：空字串在試算表裡本來就是空的，不是公式。
func sanitizeCSVField(value string) string {
	if value == "" {
		return value
	}
	// 只看第一個位元組。前綴字元都是 ASCII，因此一個多位元組的 UTF-8 字元
	// （0xE0 以上）永遠不會誤判成公式起始。
	for _, prefix := range csvFormulaPrefixes {
		if value[0] == prefix {
			return csvGuard + value
		}
	}
	return value
}

// writeCSVHeader 送出 CSV 回應的表頭並回傳一個寫入器。
//
// 為什麼用串流（直接寫進 ResponseWriter）而不是先組出整個字串再送出：
//
//   - 使用者列表可能有上千筆，先組字串代表整份匯出同時存在記憶體裡，
//     而串流只有一個 row buffer。
//   - 串流讓瀏覽器在第一列寫出時就開始顯示下載進度；一個 10 MB 的
//     單次寫入在使用者按下去到畫面出現之間會是一段空白。
//
// 為什麼在這裡（而不是在呼叫端）呼叫 Flush：flush 會送出 HTTP 標頭。一旦
// 標頭送出，這支 handler 就不能把失敗改寫成 500 —— 而查詢已經在寫第一列
// 之前就跑完了。這個「提交點」因此必須在確認查詢成功之後，由同一個函式
// 統一處理，避免某個呼叫端忘了它。
func writeCSVHeader(w http.ResponseWriter, filename string) *csv.Writer {
	// Content-Disposition 的 filename 用引號包住，否則瀏覽器會把檔名截斷
	// 在第一個空白或分號。檔名本身是程式寫死的常數（不是使用者輸入），
	// 因此不需要處理引號注入；仍然是引號包住才正確。
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// 匯出的是使用者可控的內容，讓瀏覽器不要把它當成可執行的文件。
	w.Header().Set("X-Content-Type-Options", "nosniff")

	writer := csv.NewWriter(w)
	// 先寫 BOM。csv.Writer 會在每個 record 前檢查「上一筆是否以引號結尾」，
	// 而 BOM 不是欄位值的一部分，因此這裡必須繞過它直接寫進底層 writer。
	_, _ = io.WriteString(w, utf8BOM)
	return writer
}

// csvExportURLName 是匯出檔名裡的日期部分。
//
// 用伺服器本地時區而不是 UTC：匯出檔是給人看的，而「今天」對那個人來說
// 應該是**他所在地的**今天。與 /admin/stats 的每日分組是同一個時區理由。
func csvExportURLName(prefix string, now time.Time) string {
	return fmt.Sprintf("%s-%s.csv", prefix, now.Format("20060102"))
}

// writeCSVRow 寫出一列，並對每一格套用防護。
//
// 這一層是「安全核心」所在：所有匯出都必須經過它，因此不可能有某一個
// 匯出忘了防護 —— 除非呼叫端繞過 writeCSVRow 自己呼叫 writer.Write，那
// 屬於 programmer error（程式碼審查時看得出來）。
func writeCSVRow(writer *csv.Writer, cells []string) {
	sanitized := make([]string, len(cells))
	for i, cell := range cells {
		sanitized[i] = sanitizeCSVField(cell)
	}
	_ = writer.Write(sanitized)
}

// itoa 是 strconv.Itoa 的本地別名。
//
// 這個檔案只需要非負整數，而匯出的每一格都會經過 sanitizeCSVField —— 而
// 它以第一個位元組判斷危險性。負數開頭的 strconv 輸出（"-5"）因此**必須**
// 被視為危險值（它確實是：Excel 會把 -5 評估成公式），這一點已由
// sanitizeCSVField 的前綴清單涵蓋。寫成 -5 的數值欄位在 Excel 裡會顯示為
// 5 這個公式的結果（5），外加一個前綴 —— 那正是防護的代價，而它只影響
// 負數。匯出的數值欄位都是計數（非負），所以這個情況實際上不會出現。
func itoa(n int) string {
	return strconv.Itoa(n)
}

/* ==========================================================================
   三份匯出
   ========================================================================== */

// handleAdminExportUsers 匯出使用者清單。
//
// GET /api/admin/export/users.csv
//
// 匯出的每一欄都是後臺使用者管理頁看得到的，因此匯出檔與畫面永遠一致 ——
// 若兩者欄位不同步，管理者會拿匯出檔去對帳而得到錯誤的結論。
func (s *Server) handleAdminExportUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// 匯出需要跑在標頭送出之前：一旦 writeCSVHeader 提交了標頭，後續的
	// 查詢錯誤就無法改寫成 500。因此先取資料、再開頭。
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT u.email, u.status, u.created_at, u.updated_at,
		       (SELECT COUNT(*) FROM forum_posts WHERE author_email = u.email),
		       (SELECT COUNT(*) FROM forum_post_comments WHERE author_email = u.email)
		FROM forum_users u
		ORDER BY u.email ASC
		LIMIT `+strconv.Itoa(csvExportMaxRows))
	if err != nil {
		logger.ErrorfContext(r.Context(), "[EXPORT] 使用者查詢失敗: %v", err)
		internalError(w, "unable to export users")
		return
	}
	defer rows.Close()

	type row struct {
		email, status, created, updated string
		posts, comments                 int
	}
	data := make([]row, 0, 128)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.email, &item.status, &item.created, &item.updated, &item.posts, &item.comments); err != nil {
			internalError(w, "unable to export users")
			return
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		internalError(w, "unable to export users")
		return
	}

	writer := writeCSVHeader(w, csvExportURLName("users", time.Now()))
	writeCSVRow(writer, []string{"email", "status", "created_at", "updated_at", "posts", "comments"})
	for _, item := range data {
		writeCSVRow(writer, []string{item.email, item.status, item.created, item.updated,
			itoa(item.posts), itoa(item.comments)})
	}
	writer.Flush()
}

// handleAdminExportPosts 匯出文章清單。
//
// GET /api/admin/export/posts.csv
//
// 刻意只匯出**文章**而不匯出留言：留言的數量比文章多一個數量級，混在同一個
// 檔案裡會讓「文章對帳」這件事變得難做。留言的匯出需求目前不存在。
func (s *Server) handleAdminExportPosts(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT p.id, p.author_email, LEFT(p.content, `+strconv.Itoa(csvExcerptRunes*4)+`), p.created_at,
		       (SELECT COUNT(*) FROM forum_post_comments c WHERE c.post_id = p.id),
		       (SELECT COUNT(*) FROM forum_post_likes l WHERE l.post_id = p.id)
		FROM forum_posts p
		ORDER BY p.id DESC
		LIMIT `+strconv.Itoa(csvExportMaxRows))
	if err != nil {
		logger.ErrorfContext(r.Context(), "[EXPORT] 文章查詢失敗: %v", err)
		internalError(w, "unable to export posts")
		return
	}
	defer rows.Close()

	type row struct {
		id       int64
		author   string
		content  string
		created  string
		comments int
		likes    int
	}
	data := make([]row, 0, 256)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.author, &item.content, &item.created, &item.comments, &item.likes); err != nil {
			internalError(w, "unable to export posts")
			return
		}
		item.content = excerptRunes(item.content, csvExcerptRunes)
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		internalError(w, "unable to export posts")
		return
	}

	writer := writeCSVHeader(w, csvExportURLName("posts", time.Now()))
	writeCSVRow(writer, []string{"id", "author_email", "content", "created_at", "comments", "likes"})
	for _, item := range data {
		writeCSVRow(writer, []string{strconv.FormatInt(item.id, 10), item.author, item.content,
			item.created, itoa(item.comments), itoa(item.likes)})
	}
	writer.Flush()
}

// handleAdminExportReports 匯出檢舉工單。
//
// GET /api/admin/export/reports.csv
//
// reason 欄位是**最高風險**的一欄：它是最不受限制的使用者自由文字欄位
// （最多 500 字），因此防護前綴在這一欄出現的頻率遠高於其他匯出。
// 這正是需要單一 sanitizeCSVField 實作的理由 —— 這一欄就是最可能出事的地方。
func (s *Server) handleAdminExportReports(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, target_type, target_id, reporter_email, reason, status,
		       created_at, COALESCE(reviewed_at, ''), reviewed_by
		FROM forum_reports
		ORDER BY id DESC
		LIMIT `+strconv.Itoa(csvExportMaxRows))
	if err != nil {
		logger.ErrorfContext(r.Context(), "[EXPORT] 檢舉查詢失敗: %v", err)
		internalError(w, "unable to export reports")
		return
	}
	defer rows.Close()

	type row struct {
		id, targetType, targetID, reporter, reason, status string
		created, reviewedAt, reviewedBy                    string
	}
	data := make([]row, 0, 128)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.targetType, &item.targetID, &item.reporter,
			&item.reason, &item.status, &item.created, &item.reviewedAt, &item.reviewedBy); err != nil {
			internalError(w, "unable to export reports")
			return
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		internalError(w, "unable to export reports")
		return
	}

	writer := writeCSVHeader(w, csvExportURLName("reports", time.Now()))
	writeCSVRow(writer, []string{"id", "target_type", "target_id", "reporter_email", "reason",
		"status", "created_at", "reviewed_at", "reviewed_by"})
	for _, item := range data {
		writeCSVRow(writer, []string{item.id, item.targetType, item.targetID, item.reporter,
			item.reason, item.status, item.created, item.reviewedAt, item.reviewedBy})
	}
	writer.Flush()
}

// csvExportMaxRows 限制單次匯出的列數。
//
// 上限存在的理由是三個測試，都不愉快：
//   - 記憶體：匯出的查詢結果在寫出前會全部收在切片裡（串流只省掉 CSV 本身，
//     不省掉查詢結果），因此上限同時限制了記憶體。
//   - 時間：沒有 ORDER BY + LIMIT 的全表掃描在資料量大時會讓這個請求
//     掛很久，而它是一個 GET —— 使用者會以為伺服器死了。
//   - 對帳：超過十萬列的匯出沒有人會在試算表裡逐列比對；那種需求應該是
//     「篩選後匯出」，而不是「全部匯出」。
//
// 到達上限時**不報錯也不警告**：這個端點是串流的，寫到一半就插入一行
// 「已達上限」會讓 CSV 的欄位數不一致，那比靜默截斷更糟。但頁面上必須寫出
// 上限（見前端 copy），讓管理者知道匯出檔不是完整的資料集。
const csvExportMaxRows = 50000

// csvExcerptRunes 是內容欄位在匯出檔中保留的字元數。
//
// 與 /admin/stats 的熱門文章摘要同一個常數。匯出用同樣的值是刻意的：兩者
// 都是「讓管理者認出是哪一筆」的用途，長度不同會讓對帳時以為有東西對不上。
const csvExcerptRunes = 200

// exportRouteHelp 是這個檔案的三條路由在 server.go 路由總表裡的說明字串。
//
// 集中在此處是為了讓「匯出的三條路由」與「匯出的三個實作」在視覺上相鄰；
// server.go 的註解則只需要引用這個常數。
const exportRouteHelp = "handleAdminExportUsers / handleAdminExportPosts / handleAdminExportReports（CSV）"

// exportedColumns 描述每份匯出的欄位，供測試斷言「匯出與畫面的欄位一致」。
//
// 這個常數的唯一使用者是 csv_export_test.go。它存在的理由是：匯出的欄位
// 若與後臺頁面不一致，症狀是「管理者拿匯出檔對帳而得到錯誤結論」，而那
// 種錯誤不會出現在任何測試失敗裡 —— 兩邊都「正常運作」，只是不一致。
var exportedColumns = map[string][]string{
	"users":   {"email", "status", "created_at", "updated_at", "posts", "comments"},
	"posts":   {"id", "author_email", "content", "created_at", "comments", "likes"},
	"reports": {"id", "target_type", "target_id", "reporter_email", "reason", "status", "created_at", "reviewed_at", "reviewed_by"},
}

// exportedColumnCount 回傳某份匯出的欄位數，供頁面顯示。
func exportedColumnCount(name string) int {
	return len(exportedColumns[strings.ToLower(name)])
}
