package httpapi

/*
forum_admin_handlers.go：論壇內容管理的後端 HTTP 處理層（管理員專用）。

職責
	提供後台維護論壇文章、留言與檢舉所需的 API：列表、新增、編輯、刪除與檢舉審核。
	與面向一般使用者的 forum_handlers.go 分開實作，關鍵差異是資料的暴露程度：
	本檔案直接回傳 authorEmail（真實電子郵件）並允許修改、刪除任何人的內容，
	不做 forum_handlers.go 那種去識別化與暱稱化處理，因此權限檢查必須更嚴格。

HTTP 路由對照（註冊於 server.go 的 Handler()，皆未套用 requireLogin / requireTrustedOrigin 中介層）
	GET    /api/admin/forum/posts          → handleAdminForumPosts  → listAdminForumPosts（?page=N，每頁 25 筆）
	POST   /api/admin/forum/posts          → handleAdminForumPosts  → createAdminForumPost
	PUT    /api/admin/forum/posts/{id}     → handleAdminForumPost（只改 content）
	DELETE /api/admin/forum/posts/{id}     → handleAdminForumPost
	GET    /api/admin/forum/comments       → handleAdminForumComments（?postId=N 必填）
	POST   /api/admin/forum/comments       → handleAdminForumComments
	PUT    /api/admin/forum/comments/{id}  → handleAdminForumComment（只改 content）
	DELETE /api/admin/forum/comments/{id}  → handleAdminForumComment
	GET    /api/admin/forum/reports        → handleAdminForumReports（?status=PENDING|RESOLVED|REJECTED|ALL）
	POST   /api/admin/forum/reports        → handleAdminForumReports → createAdminForumReport
	GET    /api/admin/forum/reports/{id}   → handleAdminForumReport
	PUT    /api/admin/forum/reports/{id}   → handleAdminForumReport（全欄位取代）
	PATCH  /api/admin/forum/reports/{id}   → handleAdminForumReport（部分更新）
	DELETE /api/admin/forum/reports/{id}   → handleAdminForumReport
	使用這些 API 的頁面：/admin/forum（forum-admin.js）、/admin/forum-report（forum-report.js）、
	以及 /admin（admin.html 的「使用者內容」區塊會直接呼叫本檔案的
	/api/admin/forum/posts|comments/{id}，新增內容時則走 user_admin_handlers.go 的
	/api/admin/users/{email}/posts|comments）。

資料表依賴（schema 定義於 forum/data/mysql.go 的 MigrateMySQL）
	forum_posts		文章主表；本檔案會 UPDATE content 與 DELETE 整列
	forum_post_comments	留言子表，post_id 邏輯上指向 forum_posts.id，但沒有宣告外鍵
	forum_post_likes	按讚表（複合主鍵 post_id + author_email），本檔案只用於統計數量
	forum_reports		檢舉表，含 status 狀態機、reviewed_at / reviewed_by 審核軌跡，
				以及唯一鍵 uq_forum_reports_reporter_target（reporter_email, target_type, target_id）

關鍵設計決策
	身分與來源分離：requireAdminForum 判斷「是不是管理員」（session），
	csrf.go 的 isTrustedOrigin 判斷「請求來自哪個來源」（Origin / Referer）。兩者互相獨立，
	不能互相取代，因此每一條寫入路徑都必須同時通過這兩道檢查。
	檢查以函式呼叫而非中介層呈現，理由見 requireAdminForum 的說明。
	回應格式：成功一律走 response.go 的 writeOK / writeJSON，失敗走 writeError 系列，
	訊息只給可公開的短句，SQL 細節不外洩；本檔案刻意不寫 logger。
	不使用交易：本檔案每個 handler 最多兩句 SQL（檢查 + 寫入），彼此不需原子性；
	真的需要交易的地方（例如按讚的檢查與寫入）在 forum_handlers.go 內以 tx 處理，
	並搭配 defer tx.Rollback() 當安全網。
	回應格式的例外：404 一律使用 http.NotFound（text/plain）而非 JSON，
	前端只檢查 response.ok、不解析錯誤內容，因此兩種格式並存不影響使用。
*/

import (
	"context"
	"database/sql"
	"encoding/json"
	"forum/forum/audit"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/*
adminForumPost 是管理介面使用的文章視圖模型，JSON 欄位為 camelCase。
本型別跨檔案共用：user_admin_handlers.go 的 adminUserContent 也以它組出單一使用者的文章清單。

	ID		文章主鍵，對應 forum_posts.id
	AuthorEmail	作者信箱原文。管理介面刻意顯示真實信箱供後台稽核，
			與一般使用者面經去識別化處理的輸出不同
	Content		文章內文（TEXT）
	CreatedAt	建立時間。DB DSN 帶 parseTime=True，DATETIME 因此能直接掃描成 time.Time
	ImageURL		附圖網址；無圖時為空字串，omitempty 讓 JSON 省略此欄位。
			資料庫只存檔名，回應的則是 forumImageURL 以 FILES_SERVER_PUBLIC_URL
			組出並附上 ?token= 的短期存取網址（單一使用者頁亦同，見
			handleAdminUserContent）
	LikeCount	按讚數，由 forum_post_likes 的相關子查詢彙總而來（每次查詢即時計算）
	CommentCount	留言數，由 forum_post_comments 的相關子查詢彙總；
			同時被當成 Comments 的預估容量
	Comments	該文章的所有留言（含完整內容），讓後台能在列表頁就地展開編輯／刪除。
			為了這個需求不做留言分頁，代價是每頁每篇文章各多一次查詢
*/
type adminForumPost struct {
	ID           int64               `json:"id"`
	AuthorEmail  string              `json:"authorEmail"`
	Content      string              `json:"content"`
	CreatedAt    time.Time           `json:"createdAt"`
	ImageURL     string              `json:"imageUrl,omitempty"`
	LikeCount    int                 `json:"likeCount"`
	CommentCount int                 `json:"commentCount"`
	Comments     []adminForumComment `json:"comments"`
}

/*
adminForumComment 是留言的視窗模型，欄位直接對應 forum_post_comments：

	ID		留言主鍵
	PostID		所屬文章 ID（forum_post_comments.post_id）
	AuthorEmail	留言者信箱原文，同樣只在管理介面顯示
	Content		留言內文（TEXT）
	CreatedAt	建立時間

本型別跨檔案共用：user_admin_handlers.go 的 adminUserContent 組出使用者的留言清單時也用它。
*/
type adminForumComment struct {
	ID          int64     `json:"id"`
	PostID      int64     `json:"postId"`
	AuthorEmail string    `json:"authorEmail"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"createdAt"`
}

/*
adminForumPostRequest 是後台新增／修改文章的請求本文（JSON），對應 POST /api/admin/forum/posts
與 PUT /api/admin/forum/posts/{id}。

	Content 文章內文，由 validateAdminForumPost 修剪空白並檢查長度

刻意不提供 authorEmail 與 imageUrl：作者一律取自目前的管理員 session（留下真實操作者），
附圖則只能透過一般使用者的上傳流程寫入。少了這兩個欄位，後台就無法冒用他人身分發文，
也無法繞過檔案上傳的驗證。
*/
type adminForumPostRequest struct {
	Content string `json:"content"`
}

/*
adminForumCommentRequest 是後台留言的請求本文（JSON），對應 POST /api/admin/forum/comments
與 PUT /api/admin/forum/comments/{id}。

	PostID	 目標文章 ID，僅 POST（新增留言）會用到；
			PUT 只依路徑中的留言 ID 定位，因此 body 帶的 PostID 會被忽略
	Content  留言內文，由 validateAdminForumComment 修剪空白並檢查長度

PostID 不在 validateAdminForumComment 的檢查範圍內：它屬於「引用完整性」，
必須在建 INSERT 之前實際查證父文章是否存在（見 handleAdminForumComments 的 POST 分支），
而 PUT 情境根本不需要這個欄位。本型別也跨檔案供 user_admin_handlers.go 的
createAdminUserComment 使用。
*/
type adminForumCommentRequest struct {
	PostID  int64  `json:"postId"`
	Content string `json:"content"`
}

/*
adminForumReport 是檢舉的視圖模型：欄位對應 forum_reports 表，再加兩個即時衍生的欄位。

	ID		檢舉主鍵
	TargetType	目標種類，僅 'post' 或 'comment'（沒有外鍵，靠字串區分哪張表）
	TargetID	目標在該表的主鍵。只存數字而沒有外鍵，是為了讓「被檢舉的內容被刪除後，
			檢舉記錄仍能留下」；代價是必須另外查證目標是否存在（ensureForumReportTarget）
	ReporterEmail	檢舉人信箱。它與 TargetType、TargetID 組成唯一鍵
			uq_forum_reports_reporter_target，因此同一個人對同一目標只能檢舉一次
	Reason		檢舉原因，上限 500 字（VARCHAR(500)，以字元計）
	Status		狀態機欄位：PENDING（待處理）→ RESOLVED（成立／已處理）或 REJECTED（不成立）。
			本檔案的兩種更新路徑都會依 status 決定 reviewed_at 要寫時間還是清成 NULL，
			因此「退回待處理」會同時清掉審核時間
	CreatedAt	建立時間
	ReviewedAt	審核時間。型別是 *time.Time 而非 time.Time，因為資料庫欄位允許 NULL
			（reviewed_at DATETIME NULL）：尚未審核的檢舉必須掃描成 nil，
			若用 time.Time 會在掃描 NULL 時直接報錯
	ReviewedBy	審核者信箱，記錄當下操作的管理員；資料庫預設空字串，
			配合 omitempty，未審核時前端看不到此欄
	TargetAuthor	被檢舉內容的作者（由 SQL 的 CASE WHEN 子查詢即時查出，不是本表的欄位）。
			目標已被刪除時為空字串，見列表查詢裡 COALESCE 的說明
	TargetContent	被檢舉內容的原文（同上；目標已刪除時為空字串）

角色對照（容易混淆的三組）

	TargetAuthor	是被檢舉對象的作者，內容可能已被刪除，此時子查詢回傳 NULL
	ReporterEmail	是提出檢舉的人
	ReviewedBy	是處理檢舉的管理員
*/
type adminForumReport struct {
	ID            int64      `json:"id"`
	TargetType    string     `json:"targetType"`
	TargetID      int64      `json:"targetId"`
	ReporterEmail string     `json:"reporterEmail"`
	Reason        string     `json:"reason"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
	ReviewedAt    *time.Time `json:"reviewedAt,omitempty"`
	ReviewedBy    string     `json:"reviewedBy,omitempty"`
	TargetAuthor  string     `json:"targetAuthor"`
	TargetContent string     `json:"targetContent"`
}

/*
adminForumReportRequest 是後台的檢舉表單本文（JSON），同時用於
POST /api/admin/forum/reports（新增）、PUT /api/admin/forum/reports/{id}（全欄位取代）
與 PATCH /api/admin/forum/reports/{id}（部分更新）。

	TargetType	'post' 或 'comment'；寫入前會被轉為小寫
	TargetID	目標主鍵；必須 >= 1，且目標必須真的存在
	ReporterEmail	檢舉人信箱；會被 TrimSpace，PUT 與 POST 會寫入，PATCH 不會
	Reason		檢舉原因；PUT 與 POST 必填 1~500 字，PATCH 有帶才更新
	Status		狀態機值，會被轉為大寫；空字串在 POST／PUT 視為 PENDING，
			在 PATCH 也會被視為「退回 PENDING」

PUT 與 PATCH 對同一組欄位的處理刻意不同，差異見 handleAdminForumReport 的說明。
*/
type adminForumReportRequest struct {
	TargetType    string `json:"targetType"`
	TargetID      int64  `json:"targetId"`
	ReporterEmail string `json:"reporterEmail"`
	Reason        string `json:"reason"`
	Status        string `json:"status"`
}

/*
requireAdminForum 檢查目前請求是否帶有管理員 session，若不是就立即回應 401。
回傳 true 代表後續程式碼可以安全假設「呼叫端已通過管理員檢查」。

與 csrf.go 的 isTrustedOrigin 如何分工

	兩者回答的是不同問題，互補而非替代：

		requireAdminForum（身分）
			依據 session cookie 內的 is_admin 欄位（session.Manager.IsAdmin）判斷
			「呼叫者是誰、是不是管理員」，完全不看請求來自哪裡。

		isTrustedOrigin（來源，csrf.go）
			比對 Origin／Referer 與設定的 TrustedOrigins，判斷「這個請求從哪個網站發出」，
			完全不看登入狀態。兩者皆未提供時它會視為通過（同源請求的正常情況）。

	缺一不可的原因：只做身分檢查時，攻擊者可從任意外站對已登入管理員的瀏覽器發出
	帶 cookie 的請求（CSRF）；只做來源檢查時，任何人都能帶著合法 Origin 直接呼叫管理 API。

為什麼不是 requireAdmin(next) 形式的中介層

	身分與來源檢查的適用範圍不同：GET 唯讀、不需要來源檢查，寫入才需要。中介層只能整條路由
	包一層，套在集合路由上會連 GET 一起擋掉；若在中介層內部再判 method，等於把 method 分派
	邏輯分散到兩個檔案。

	各 handler 要能精確控制「先查哪一件事、對哪一種錯誤回哪一種狀態碼」。回傳 bool 讓呼叫端
	以 if !s.requireAdminForum(w, r) { return } 立即中止，401 的內容由本函式統一發出，
	不必在每個 handler 重複寫一次。

	server.go 以 mux.HandleFunc 直接註冊這些後台路由（未掛 requireLogin 等中介層），保留顯式
	呼叫可讓「這個 handler 有沒有做身分檢查」在程式碼上一目了然。
*/
func (s *Server) requireAdminForum(w http.ResponseWriter, r *http.Request) bool {
	// 身分不符回 401（而非 403）：前端依賴這個狀態碼。
	// forum-admin.js 的 loadPosts 與 forum-report.js 的 loadReports 都用
	// response.status === 401 導回 /admin 重新登入。
	if !s.sessions.IsAdmin(r) {
		unauthorized(w, "admin access required")
		return false
	}
	// 不做其他檢查：停權（forum_users.status = SUSPENDED）只在使用者面生效，
	// 管理員即使被停權仍可進入後台（server.go 的 requireLogin 也對管理員略過停權檢查）。
	return true
}

/*
handleAdminForumPosts 處理 /api/admin/forum/posts 的集合層級操作，
路由在 server.go 中以 mux.HandleFunc 直接註冊，沒有套用任何中介層。

	GET	listAdminForumPosts：分頁列出全部文章，每頁附帶各文章的完整留言
	POST	createAdminForumPost：以目前管理員的身分新增文章
	其他	methodNotAllowed 回 405

檢查順序的兩個考量

	身分檢查放在 method 分派之前：即使請求用了不支援的方法，也要先確定對方是管理員，
	否則未登入者可以用「不支援的方法」反覆探測這些後台路由是否存在。

	來源檢查（isTrustedOrigin）刻意不放進這個 switch，而是由各寫入分支自行呼叫
	（見 createAdminForumPost）。GET 是唯讀，不做來源檢查；把檢查留在寫入分支，
	才能讓同一個 handler 同時安全地服務讀取與寫入兩種請求。
*/
func (s *Server) handleAdminForumPosts(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.listAdminForumPosts(w, r)
	case http.MethodPost:
		s.createAdminForumPost(w, r)
	default:
		methodNotAllowed(w)
	}
}

/*
handleAdminForumReports 處理 /api/admin/forum/reports 的集合層級操作。

	GET	依 status 查詢檢舉清單，並附上被檢舉內容的作者與原文
	POST	createAdminForumReport：讓管理員手動建立一筆檢舉（補登或人工回報用）
	其他	methodNotAllowed 回 405

status 查詢參數的語意

	先 TrimSpace 再轉大寫，讓 ?status=resolved 與 ?status=RESOLVED 等價。
	未帶參數時預設 PENDING，因為後台的主要工作流是「只看待處理的檢舉」。

	"ALL" 不是資料庫裡的狀態值，而是查詢層的語意值：不加 WHERE 條件、列出全部。
	不屬於 PENDING / RESOLVED / REJECTED / ALL 的值一律回 400，而不是默默忽略，
	否則前端打錯字（例如 ?status=resolvedd）會拿到一份看似合理、實際過期的清單。

回應中的 total 是「本次篩選後的筆數」而非全部筆數，因此同時回傳 status 讓前端回填篩選器。
*/
func (s *Server) handleAdminForumReports(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		s.createAdminForumReport(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = "PENDING"
	}
	if status != "ALL" && status != "PENDING" && status != "RESOLVED" && status != "REJECTED" {
		badRequest(w, "invalid report status")
		return
	}
	// 查詢用 CASE WHEN 加相關子查詢把「被檢舉的內容」即時接進結果集：forum_reports 只存
	// target_type 與 target_id，沒有 author / content 欄位。若改成先查檢舉清單再逐筆撈內容，
	// 就會變成 1 + N 次查詢；反過來把內容複製進檢舉表又會有資料不一致與欄位重複的問題。
	// 兩個子查詢分別對 forum_posts.id 與 forum_post_comments.id 執行，兩者都是主鍵查找。
	//
	// 兩個衍生欄位都套 COALESCE(..., '')：被檢舉的內容被刪除後（例如這一頁的
	// 「通過（刪文）」剛刪掉目標），子查詢會回 NULL，而 database/sql 無法把 NULL
	// 掃描進 string 欄位，沒有 COALESCE 時整筆查詢會以 500 結束 —— 一筆過期的檢舉
	// 會讓管理介面看不到其餘所有檢舉。空字串代表「目標已不存在」，前端據此顯示提示。
	query := `
		SELECT id, target_type, target_id, reporter_email, reason, status, created_at, reviewed_at, reviewed_by,
		       COALESCE(CASE WHEN target_type = 'post' THEN (SELECT author_email FROM forum_posts WHERE id = target_id)
		                    ELSE (SELECT author_email FROM forum_post_comments WHERE id = target_id) END, ''),
		       COALESCE(CASE WHEN target_type = 'post' THEN (SELECT content FROM forum_posts WHERE id = target_id)
		                    ELSE (SELECT content FROM forum_post_comments WHERE id = target_id) END, '')
		FROM forum_reports`
	// 預留容量 1：只有需要 WHERE 時才會用到，避免每次請求都重新配置切片。
	args := make([]interface{}, 0, 1)
	// 只有非 ALL 才附加篩選條件。「ALL」必須在這裡被翻譯掉，否則會變成
	// WHERE status = 'ALL' 而永遠查不到任何資料。
	if status != "ALL" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	// 排序補上 id 作為第二順位：created_at 是 DATETIME（秒級精度），同一秒內建立多筆是
	// 常見情況，只用 created_at 排序會讓同一頁的內容在兩次請求之間跳動（分頁不穩定）。
	query += ` ORDER BY created_at DESC, id DESC`
	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		internalError(w, "unable to load forum reports")
		return
	}
	// defer 必須寫在錯誤檢查之後：QueryContext 失敗時 rows 為 nil，對 nil 呼叫 Close 會 panic。
	defer rows.Close()
	// 初始化為空切片而非 nil，讓沒有資料時 JSON 輸出 [] 而不是 null，
	// 前端（forum-admin.js / forum-report.js）就可以直接 items.map(...) 不必判空。
	items := make([]adminForumReport, 0)
	for rows.Next() {
		var item adminForumReport
		// Scan 的目標順序必須與上面 SELECT 的欄位順序完全一致。
		// 最後兩個欄位是被檢舉內容的作者與原文（衍生欄位，可能為 NULL）。
		// ReviewedAt 掃描進 *time.Time：未審核的檢舉在資料庫中就是 NULL，會得到 nil。
		if err := rows.Scan(&item.ID, &item.TargetType, &item.TargetID, &item.ReporterEmail, &item.Reason, &item.Status, &item.CreatedAt, &item.ReviewedAt, &item.ReviewedBy, &item.TargetAuthor, &item.TargetContent); err != nil {
			internalError(w, "unable to read forum reports")
			return
		}
		items = append(items, item)
	}
	// rows.Err() 必須在迴圈結束後另外檢查：迭代過程中發生的錯誤（連線中斷等）不會讓
	// rows.Next() 直接回傳 false，漏檢會把「只讀到一部分」誤當成「讀完了」而回 200。
	if err := rows.Err(); err != nil {
		internalError(w, "unable to read forum reports")
		return
	}
	// total 用 len(items) 即可：結果已全部載入記憶體，不必再打一次 COUNT(*)。
	writeOK(w, map[string]interface{}{"items": items, "total": len(items), "status": status})
}

/*
handleAdminForumReport 處理 /api/admin/forum/reports/{id} 的單筆檢舉操作。

	GET	getAdminForumReport：取回單筆檢舉（含被檢舉內容的作者與原文）
	DELETE	刪除這筆檢舉記錄，被檢舉的文章／留言本體不動
	PUT	全欄位取代：target_type、target_id、reporter_email、reason、status 全部依 body 覆寫
	PATCH	部分更新：只改 status，以及 body 有帶的 reason
	其他	methodNotAllowed 回 405

PUT 與 PATCH 的差異是本函式最需要注意的地方

	兩者都會更新狀態機與審核軌跡（reviewed_at、reviewed_by），但驗證與寫入範圍不同。
	PUT 走 validateAdminForumReport 的全量驗證（reason 必填 1~500 字、目標必須存在），
	並一次改寫所有業務欄位，語意等同「把這筆檢舉改成我送來的樣子」。
	PATCH 允許 body 只帶 status：frontend/web/forum-admin.js 的「已處理／不成立」按鈕
	送的就是 PATCH {"status": "RESOLVED"}。PATCH 不會寫入 target 與 reporter，
	維持「審核動作不應該改動檢舉內容本身」這個語意。

狀態與審核軌跡的連動規則（PUT 與 PATCH 一致）

	status 為 PENDING → reviewed_at 寫 SQL NULL（尚未審核）
	status 為 RESOLVED 或 REJECTED → reviewed_at 寫入當下時間，reviewed_by 寫入管理員信箱
	因此把已審核的檢舉改回 PENDING 會清掉審核時間，但 reviewed_by 仍會被覆寫成操作者，
	資料上可能出現「PENDING 卻有 reviewed_by」的組合；判斷是否審核過要以 reviewed_at 為準。
*/
func (s *Server) handleAdminForumReport(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	// 路徑參數解析：server.go 以 "/api/admin/forum/reports/"（結尾帶斜線）註冊這條路由，
	// ServeMux 只做前綴比對，尾端的 ID 必須自己切出來：
	// TrimPrefix 去掉路由前綴、Trim 去掉多餘的斜線（/api/admin/forum/reports/7/ 也能解析）、
	// ParseInt 轉為整數。
	// 解析失敗或 id < 1 一律回 400：id 是 AUTO_INCREMENT 主鍵，只可能 >= 1，因此 0 或負數
	// 必定是客戶端送錯，而不是「資源不存在」。用 400 表達請求不合法比回 404 誠實，
	// 也省下一次必然落空的資料庫查詢。
	// 附帶行為：ParseInt 接受前導「+」，所以 /api/admin/forum/reports/+7 會被當成 7。
	// 這是 ParseInt 的既定行為，在此無害（仍須通過管理員與來源檢查），故不額外收緊。
	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/forum/reports/"), "/"), 10, 64)
	if err != nil || id < 1 {
		badRequest(w, "invalid report id")
		return
	}
	// GET 是唯讀路徑，不做來源檢查：管理頁與自己同源載入，不需要 CSRF 防護；
	// 跨站請求即使通過也讀不到回應內容。把來源檢查放在 GET 之後才不會擋掉正常讀取。
	if r.Method == http.MethodGet {
		item, err := s.getAdminForumReport(r.Context(), id)
		if err == sql.ErrNoRows {
			// 直接用 == 比對 sentinel：QueryRowContext 會原樣回傳驅動程式的錯誤，
			// 這條呼叫鏈上沒有任何錯誤包裝。
			http.NotFound(w, r)
			return
		}
		if err != nil {
			internalError(w, "unable to load forum report")
			return
		}
		// 單筆查詢包成 {"item": ...}，與集合路由的 {"items": [...]} 區分開。
		writeOK(w, map[string]interface{}{"item": item})
		return
	}
	// 從這裡開始都是會改動資料的操作，一律要求可信來源。檢查刻意放在 method 分派之前：
	// 來源不可信時，任何非 GET 的請求都只會得到 403，不會洩漏「這個 ID 支援哪些方法」。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	if r.Method == http.MethodDelete {
		// 刪除的是檢舉記錄本身，被檢舉的文章／留言保持不動。
		// forum_reports 沒有任何外鍵（target_id 只是數字），所以不需要先刪子表，
		// 也不會因外鍵約束而失敗。
		//
		// 稽核：刪除檢舉是不可逆的（無論被檢舉的內容之後如何，檢舉本身都不在了），
		// 因此把刪除前的 status 與 reason 一起記進 diff —— 事後要回答
		// 「這筆被刪掉的是誰檢舉的、原因是什麼」時，資料必須在紀錄裡。
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to delete forum report")
			return
		}
		defer tx.Rollback()

		var beforeStatus, beforeReason string
		if err := tx.QueryRowContext(r.Context(),
			`SELECT status, reason FROM forum_reports WHERE id = ?`, id).Scan(&beforeStatus, &beforeReason); err != nil {
			if err == sql.ErrNoRows {
				// 資源不存在 → 404。放在交易內是安全的：這個分支會回滾，
				// 交易裡只有一次唯讀查詢，沒有任何寫入。
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load forum report")
			return
		}

		result, err := tx.ExecContext(r.Context(), `DELETE FROM forum_reports WHERE id = ?`, id)
		if err != nil {
			internalError(w, "unable to delete forum report")
			return
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			// 影響 0 筆代表 WHERE id = ? 沒有比對到任何資料，因此回 404。
			// 對同一筆檢舉重複送出 DELETE 時，第二次必然落到這個分支，此時資源確實已經
			// 不存在，語意仍然正確。
			http.NotFound(w, r)
			return
		}
		if err := s.recordAdminAction(r, tx, adminActionReportDelete, audit.TargetReport, strconv.FormatInt(id, 10), beforeStatus,
			audit.Change{Field: "reason", Before: beforeReason, After: "（檢舉記錄已刪除）"}); err != nil {
			internalError(w, "unable to delete forum report")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to delete forum report")
			return
		}
		writeOK(w, map[string]bool{"ok": true})
		return
	}
	// 這裡就解碼 body（而不是確定 method 之後才解），因此「不支援的方法」與「壞掉的 body」
	// 兩種情況會先撞到哪一個並沒有保證：body 為空時 Decode 回傳 io.EOF，
	// 於是空 body 的未知 method 拿到 400 而非 405。實務上前端只會送 PUT／PATCH／DELETE，
	// 這個行為差異可以接受。
	var req adminForumReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// 狀態一律正規化為大寫：資料庫的 status 欄位、列表查詢的 status 參數
	// 與狀態機的白名單比較全部使用大寫字面值，先在此收斂可讓三處的比較基準一致。
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if r.Method == http.MethodPut {
		// PUT 是全欄位取代，連 targetType 與 reporterEmail 都要一起正規化。
		// 這兩項在 PATCH 分支不做正規化，因為 PATCH 根本不會寫入它們。
		req.TargetType = strings.ToLower(strings.TrimSpace(req.TargetType))
		req.ReporterEmail = strings.TrimSpace(req.ReporterEmail)
		req.Reason = strings.TrimSpace(req.Reason)
		if req.Status == "" {
			// 沒指定狀態時回到 PENDING：PUT 的語意是「取代成我送的內容」，
			// 送空白等於要求這筆檢舉回到待處理。
			req.Status = "PENDING"
		}
		if err := validateAdminForumReport(&req); err != nil {
			// 驗證失敗的訊息來自 requestError，是可以直接顯示給使用者的繁體中文。
			badRequest(w, err.Error())
			return
		}
		// 目標存在性必須在寫入前確認：forum_reports 沒有外鍵，資料庫不會擋下
		// 指向已刪除文章／留言的檢舉，這道檢查是唯一的防線。
		if err := s.ensureForumReportTarget(r.Context(), req.TargetType, req.TargetID); err != nil {
			if err == sql.ErrNoRows {
				// 目標已不存在 → 404，而不是建立一筆指向虛無的檢舉。
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load report target")
			return
		}
		// reviewed_at 用 interface{} 承接：欄位可為 NULL 時，只有把 nil 介面傳給驅動程式
		// 才會寫入 SQL NULL；若傳入零值 time.Time，會存成 '0001-01-01 00:00:00'。
		var reviewedAt interface{}
		if req.Status != "PENDING" {
			// 只有離開 PENDING 才視為「已審核」，此時才記時間。
			reviewedAt = time.Now()
		}
		// reviewed_by 記錄的是當下操作的管理員（session 的 email 欄位），
		// 不是被檢舉內容的作者（TargetAuthor）、也不是檢舉人（ReporterEmail）。
		// requireAdminForum 已保證 is_admin 為 true，因此這裡不會寫入空字串。
		// 這個 UPDATE 的所有值都來自 body 或 session，沒有一個字元被拼進 SQL 文字。
		actor := s.sessions.ResolveUser(r)
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to update forum report")
			return
		}
		defer tx.Rollback()

		// PUT 是全欄位取代，因此 diff 要涵蓋它改動的每一欄：被檢舉對象、檢舉人、
		// 原因、狀態。只記狀態會讓「審核動作不該改動檢舉內容」這個規則被繞過時
		// 完全看不出來 —— 而那正是 PUT 最需要被稽核的情況。
		var beforeTargetType, beforeTargetID, beforeReporter, beforeReason, beforeStatus string
		if err := tx.QueryRowContext(r.Context(), `
			SELECT target_type, target_id, reporter_email, reason, status
			FROM forum_reports WHERE id = ?`, id).
			Scan(&beforeTargetType, &beforeTargetID, &beforeReporter, &beforeReason, &beforeStatus); err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load forum report")
			return
		}

		result, err := tx.ExecContext(r.Context(), `
			UPDATE forum_reports
			SET target_type = ?, target_id = ?, reporter_email = ?, reason = ?, status = ?, reviewed_at = ?, reviewed_by = ?
			WHERE id = ?`, req.TargetType, req.TargetID, req.ReporterEmail, req.Reason, req.Status, reviewedAt, actor, id)
		if err != nil {
			internalError(w, "unable to update forum report")
			return
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			// 同樣把「影響 0 筆」當成 404。注意 reviewed_at 每次都是新的 time.Now()，
			// 因此只要 status 有離開 PENDING，實際變更的列數就會是 1。
			http.NotFound(w, r)
			return
		}
		if err := s.recordAdminAction(r, tx, adminReportAction(beforeStatus, req.Status), audit.TargetReport, strconv.FormatInt(id, 10), beforeStatus,
			onlyChanged(
				audit.Change{Field: "targetType", Before: beforeTargetType, After: req.TargetType},
				audit.Change{Field: "targetId", Before: beforeTargetID, After: strconv.FormatInt(req.TargetID, 10)},
				audit.Change{Field: "reporterEmail", Before: beforeReporter, After: req.ReporterEmail},
				audit.Change{Field: "reason", Before: beforeReason, After: req.Reason},
				audit.Change{Field: "status", Before: beforeStatus, After: req.Status},
			)...); err != nil {
			internalError(w, "unable to update forum report")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to update forum report")
			return
		}
		writeOK(w, map[string]bool{"ok": true})
		return
	}
	// 走到這裡代表 method 既不是 GET、DELETE，也不是 PUT（PUT 分支的所有路徑都 return）。
	// 這道檢查刻意擋在 405 之前：body 有帶不合法狀態時先回 400，因為對方確實送出了
	// 一個看懂了的請求，只是內容不合法；只有連方法都不支援才回 405。
	if req.Status != "" && req.Status != "PENDING" && req.Status != "RESOLVED" && req.Status != "REJECTED" {
		badRequest(w, "invalid report status")
		return
	}
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		// 實際上等價於「不是 PATCH 就 405」：PUT 已在前面整段 return，永遠走不到這裡。
		// 保留 method != MethodPut 這個條件是為了讓意圖明確，也避免日後有人把 PUT 分支
		// 改成 fall-through 時誤判（fall-through 會讓 PUT 被當成 PATCH 再執行一次）。
		methodNotAllowed(w)
		return
	}
	if req.Status == "" {
		// PATCH 未帶 status 時被視為「退回待處理」，而且下方會把 reviewed_at 清成 NULL，
		// 等同 PATCH {"status": "PENDING"}。也就是說 PATCH {} 會重置整條審核軌跡；
		// 呼叫端必須確保 status 一定有帶，前端目前兩種用法（forum-admin.js 的 PATCH、
		// forum-report.js 的 PUT）都會帶。
		req.Status = "PENDING"
	}
	if req.Reason != "" {
		// reason 為空字串代表「不更新這個欄位」（部分更新語意），但只要有值就必須符合上限：
		// MySQL 的 VARCHAR(500) 超出時會回 1406 資料過長，在這裡先擋下來才能回 400 而不是 500。
		// 同時再 Trim 一次：validateAdminForumReport 在 PUT 路徑會修剪，這裡是 PATCH 專屬路徑。
		req.Reason = strings.TrimSpace(req.Reason)
		// 以 rune 計算長度而非位元組：VARCHAR(500) 的限制是「字元」，中文用 len(string)
		// 會算成 3 倍而誤判超長。
		if len([]rune(req.Reason)) > 500 {
			badRequest(w, "檢舉原因最多 500 字")
			return
		}
	}
	// reviewed_at 同樣以 interface{} 承接，nil 會被寫成 SQL NULL。
	var reviewedAt interface{}
	if req.Status == "PENDING" {
		// 退回 PENDING 時明確清空審核時間，語意是「尚未審核」。
		// 與 PUT 分支「保持 nil 介面」的寫法等價，只是這裡寫得較直白。
		reviewedAt = nil
	} else {
		reviewedAt = time.Now()
	}
	// 動態組裝 UPDATE：reason 有值才附加「, reason = ?」，沒有值就完全不碰該欄位。
	// 這裡可以安全地拼接「欄位名」，因為拼接的內容是本函式寫死的字面值，
	// 沒有任何使用者輸入參與；相對地，所有「值」都必須以佔位符 ? 交給驅動程式做轉義，
	// 因為值來自 HTTP body、完全可以被構造。若改用 fmt.Sprintf 把值塞進 SQL 字串，
	// 等於直接開了 SQL 注入。
	// 這是本檔案處理動態 SQL 的唯一准則：白名單決定「結構」，佔位符綁定「資料」。
	// 同一個原則也出現在 ensureForumReportTarget（表名來自 if/else 白名單，ID 走佔位符）。
	query := `UPDATE forum_reports SET status = ?, reviewed_at = ?, reviewed_by = ?`
	// reviewed_by 記錄的是當下操作的管理員（session 的 email 欄位），不是被檢舉對象、
	// 也不是檢舉人。requireAdminForum 已保證 is_admin 為 true。
	actor := s.sessions.ResolveUser(r)
	// args 的順序必須與 query 中佔位符出現的順序嚴格一致，否則值會被寫進錯誤的欄位。
	args := []interface{}{req.Status, reviewedAt, actor}
	if req.Reason != "" {
		query += `, reason = ?`
		args = append(args, req.Reason)
	}
	query += ` WHERE id = ?`
	args = append(args, id)

	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to update forum report")
		return
	}
	defer tx.Rollback()

	// 讀取改動前的 reason 與 status。PATCH 只會動 status 與（選擇性的）reason，
	// 因此 diff 只涵蓋這兩個欄位；target 與 reporter 不在此列，因為 PATCH 的
	// 語意就是不碰它們。
	var beforeReason, beforeStatus string
	if err := tx.QueryRowContext(r.Context(),
		`SELECT reason, status FROM forum_reports WHERE id = ?`, id).Scan(&beforeReason, &beforeStatus); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		internalError(w, "unable to load forum report")
		return
	}

	result, err := tx.ExecContext(r.Context(), query, args...)
	if err != nil {
		internalError(w, "unable to update forum report")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		// 影響 0 筆 → 沒有任何檢舉符合這個 id → 404。
		// 與 DELETE／PUT 分支的判斷方式一致：把「沒有列被改動」一律解讀為資源不存在。
		// 附帶限制：對同一筆檢舉重複送出相同的 status 時，因為 reviewed_at 每次都是新的
		// time.Now()，實際變更列數仍會是 1，因此不會誤判成 404。
		http.NotFound(w, r)
		return
	}
	changes := onlyChanged(audit.Change{Field: "status", Before: beforeStatus, After: req.Status})
	if req.Reason != "" {
		changes = onlyChanged(append(changes, audit.Change{Field: "reason", Before: beforeReason, After: req.Reason})...)
	}
	if err := s.recordAdminAction(r, tx, adminReportAction(beforeStatus, req.Status), audit.TargetReport, strconv.FormatInt(id, 10), beforeStatus, changes...); err != nil {
		internalError(w, "unable to update forum report")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to update forum report")
		return
	}
	writeOK(w, map[string]bool{"ok": true})
}

/*
adminReportAction 依「從什麼狀態變到什麼狀態」決定稽核的動作名稱。

	→ RESOLVED	report.resolve（成立）
	→ REJECTED	report.reject（不成立）
	其他（含 PENDING）	report.update

分成三支而不是一律記 "report.update" 的理由：這三個動作在後臺的語意與
後果都不同 —— resolve 通常伴隨刪除被檢舉的內容，reject 表示內容沒問題。
把它們混在 "update" 底下，稽核紀錄就答不出「這週有幾件檢舉被判定為成立」，
而那正是檢舉機制唯一需要被追蹤的數字。
*/
func adminReportAction(beforeStatus, afterStatus string) string {
	switch afterStatus {
	case "RESOLVED":
		return adminActionReportResolve
	case "REJECTED":
		return adminActionReportReject
	default:
		return adminActionReportUpdate
	}
}

/*
createAdminForumReport 對應 POST /api/admin/forum/reports，讓管理員手動建立一筆檢舉記錄
（例如使用者已在站外回報、由客服人工登錄，或補登歷史資料）。
呼叫前已由 handleAdminForumReports 完成 requireAdminForum 身分檢查。

回應

	201	{"ok": true, "id": 新建立的檢舉主鍵}
	400	body 不是合法 JSON、欄位驗證失敗
	403	來源不可信（isTrustedOrigin 不通過）
	404	target_type / target_id 指向的文章或留言不存在
	500	其餘資料庫錯誤

與一般使用者檢舉路徑（forum_handlers.go 的 handleForumReport）的差異：
那條路徑的 reporter_email 固定取自 session，管理員後台才允許在表單上指定檢舉人。
*/
func (s *Server) createAdminForumReport(w http.ResponseWriter, r *http.Request) {
	// 寫入前擋跨站請求：session cookie 由瀏覽器自動附上，只有來源檢查能確認這個 POST
	// 真的來自管理頁。沒有套用 csrf.go 的 requireTrustedOrigin 中介層，因為同一條路由
	// 還要服務 GET（唯讀，不需要來源檢查），中介層無法只包住寫入分支。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	var req adminForumReportRequest
	// 這個 handler 特別加了 MaxBytesReader：body 含自由文字（reason、reporterEmail），
	// 16 KiB 上限可擋掉灌水或刻意超大的 payload。解碼失敗（含觸及上限的
	// http: request body too large）一律回 400，不向外透露失敗的具體原因。
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// 四個欄位在此統一正規化：targetType 進 DB 用小寫、status 進 DB 用大寫。
	// 這不只是為了與既有資料一致——Go 端的 == 比較是大小寫敏感的，
	// ensureForumReportTarget 用 targetType == "comment" 決定查哪張表、
	// validateAdminForumReport 也用小寫比較，因此必須先收斂成單一 casing。
	req.TargetType = strings.ToLower(strings.TrimSpace(req.TargetType))
	req.ReporterEmail = strings.TrimSpace(req.ReporterEmail)
	req.Reason = strings.TrimSpace(req.Reason)
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.Status == "" {
		// 缺省為 PENDING：後台手動新增的檢舉一開始就是待處理狀態。
		req.Status = "PENDING"
	}
	if err := validateAdminForumReport(&req); err != nil {
		badRequest(w, err.Error())
		return
	}
	// 目標存在性檢查放在 INSERT 之前：forum_reports.target_id 沒有外鍵，
	// 資料庫不會擋下指向不存在文章／留言的檢舉，這道檢查是唯一的防線。
	// 檢查與寫入之間沒有交易，兩者之間若有其他人刪掉目標，會留下一筆孤兒檢舉；
	// 以管理後台的低併發與「孤兒檢舉仍可被檢視」的特性來看，這個縫隙可接受。
	if err := s.ensureForumReportTarget(r.Context(), req.TargetType, req.TargetID); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		internalError(w, "unable to load report target")
		return
	}
	// created_at 由應用層的 time.Now() 產生，與其他 handler 一致；DSN 的 parseTime=True
	// 讓寫入的時間能被還原成 time.Time，loc=Local 則保證時區與伺服器一致。
	// reviewed_at / reviewed_by 不在 INSERT 欄位中：新建立的檢舉一定是「尚未審核」，
	// 這兩個欄位交由資料庫預設（NULL 與空字串），也讓 INSERT 語句保持簡單。
	//
	// 稽核：管理員手動建立的檢舉要記 reason 與 reporterEmail。理由是這條路徑
	// 最容易產生「站上根本沒有人檢舉過」的紀錄（客服代登、或補登歷史資料），
	// 事後要能分辨它與使用者自己送的檢舉。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to create forum report")
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO forum_reports (target_type, target_id, reporter_email, reason, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, req.TargetType, req.TargetID, req.ReporterEmail, req.Reason, req.Status, time.Now())
	if err != nil {
		// 這裡沒有比對 MySQL 錯誤碼 1062（唯一鍵 uq_forum_reports_reporter_target 衝突）：
		// 使用者面建立檢舉的入口（forum_handlers.go 的 handleForumReport）會把 1062 轉成
		// 409「你已檢舉過此內容」，這個後台入口沒有對應的處理，重複建立會落到下方的 500。
		// 屬於已知的兩條路徑不一致；要補上時請一併確認前端是否需要 409。
		internalError(w, "unable to create forum report")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to read created report")
		return
	}
	if err := s.recordAdminAction(r, tx, adminActionReportCreate, audit.TargetReport, strconv.FormatInt(id, 10), req.Status,
		audit.Change{Field: "target", Before: "", After: req.TargetType + "/" + strconv.FormatInt(req.TargetID, 10)},
		audit.Change{Field: "reporterEmail", Before: "", After: req.ReporterEmail},
		audit.Change{Field: "reason", Before: "", After: req.Reason},
	); err != nil {
		internalError(w, "unable to create forum report")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to read created report")
		return
	}
	// 回 201 並附上新主鍵，前端據此提示「已新增」並重新載入清單。
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
}

/*
getAdminForumReport 依主鍵取回單筆檢舉，並附上被檢舉內容的作者（TargetAuthor）與原文（TargetContent）。

參數

	ctx	查詢用的 context，呼叫端傳入 r.Context()，讓客戶端斷線能取消查詢
	id	檢舉主鍵；由呼叫端負責先做格式與範圍驗證

錯誤

	sql.ErrNoRows	id 不存在；handleAdminForumReport 據此回 404
	其他錯誤	原樣回傳，由呼叫端轉成 500
	即使出錯也會一併回傳（此時為零值）item，呼叫端必須先檢查 err 再使用。

SQL 與 handleAdminForumReports 的列表查詢刻意保持一致，讓單筆與列表的欄位語意不會分歧。
*/
func (s *Server) getAdminForumReport(ctx context.Context, id int64) (adminForumReport, error) {
	var item adminForumReport
	// 兩個衍生欄位用 CASE WHEN 加相關子查詢在同一次查詢內解決，不必為了顯示作者與原文
	// 再多打兩次資料庫。target_type 不是 'post' 時一律視為 comment，
	// 與 ensureForumReportTarget 的預設行為相同：即使資料庫出現未知值也不會讓查詢失敗。
	//
	// COALESCE(..., '') 的理由與 handleAdminForumReports 的列表查詢相同：目標已被刪除時
	// 子查詢回 NULL，database/sql 掃描 NULL 進 string 會直接讓這次查詢回 500。改用空字串
	// 表示「目標已不存在」，單筆與列表因此對同一種資料給出相同的結果。
	err := s.db.QueryRowContext(ctx, `
		SELECT id, target_type, target_id, reporter_email, reason, status, created_at, reviewed_at, reviewed_by,
		       COALESCE(CASE WHEN target_type = 'post' THEN (SELECT author_email FROM forum_posts WHERE id = target_id)
		                    ELSE (SELECT author_email FROM forum_post_comments WHERE id = target_id) END, ''),
		       COALESCE(CASE WHEN target_type = 'post' THEN (SELECT content FROM forum_posts WHERE id = target_id)
		                    ELSE (SELECT content FROM forum_post_comments WHERE id = target_id) END, '')
		FROM forum_reports WHERE id = ?`, id).Scan(
		&item.ID, &item.TargetType, &item.TargetID, &item.ReporterEmail, &item.Reason, &item.Status,
		&item.CreatedAt, &item.ReviewedAt, &item.ReviewedBy, &item.TargetAuthor, &item.TargetContent)
	// 連同 err 回傳 item：呼叫端在 err == nil 時才會使用它，這樣寫可以省掉一個區域變數。
	return item, err
}

/*
ensureForumReportTarget 確認檢舉目標（文章或留言）真的存在。

參數

	ctx	查詢用的 context
	targetType	"post" 或 "comment"，決定查哪張表
	targetID	目標主鍵

回傳

	nil		目標存在
	sql.ErrNoRows	目標不存在；呼叫端（handleAdminForumReport、createAdminForumReport）
			以「目標不存在 → 404」的既有分支處理，不需另訂錯誤型別
	其他錯誤	資料庫錯誤，原樣回傳

為什麼需要這個檢查

	forum_reports 只存 target_type（'post' / 'comment'）與 target_id，沒有指向
	forum_posts / forum_post_comments 的外鍵。這是有意的設計：被檢舉的內容被刪除後，
	檢舉記錄仍應保留（管理員還需要知道曾經有人檢舉過它）。代價是資料庫完全不會阻止
	指向不存在目標的檢舉，所以每次寫入前都必須自行驗證。

安全性：表名以字串拼接是安全的，因為 table 只可能是本函式內的兩個字面值，
無論外部輸入為何都不會影響它；targetID 則一律以佔位符傳入，不進入 SQL 文字。
這個「白名單決定結構、佔位符綁定資料」的分工是本檔案處理動態 SQL 的唯一准則。
*/
func (s *Server) ensureForumReportTarget(ctx context.Context, targetType string, targetID int64) error {
	// 預設視為 post。呼叫端都會先跑過 validateAdminForumReport（擋掉 post/comment 以外的值），
	// 所以這裡不需要再回傳錯誤，也讓本函式可以獨立於驗證流程安全運作。
	table := "forum_posts"
	if targetType == "comment" {
		table = "forum_post_comments"
	}
	var count int
	// 用 COUNT(*) 而不是 SELECT ... ：兩張目標表的欄位並不完全一致（留言沒有 image_url），
	// COUNT(*) 可以讓兩種目標共用同一段 SQL；而 id 是主鍵、比對成本等同一次索引查找，
	// 因此多算一次 COUNT 的成本可以忽略。
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE id = ?", targetID).Scan(&count); err != nil {
		return err
	}
	// 手動產生 sql.ErrNoRows：COUNT 查詢在沒有資料列時會回傳 0 而不是 ErrNoRows，
	// 沿用同一個 sentinel 讓呼叫端只需一組錯誤處理。
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

/*
validateAdminForumReport 檢查後台檢舉表單的欄位合法性，供 POST（新增）與 PUT（全欄位取代）共用。

回傳

	nil		通過
	*requestError	其 Error() 就是要塞給前端的繁體中文訊息，呼叫端一律以 badRequest 回 400

前置條件與副作用

	本函式不修改傳入的結構，也不修剪字串；呼叫端負責先做 ToLower / ToUpper / TrimSpace。
	因此它假設 targetType 已是小寫、status 已是大寫。
	「為什麼不把正規化收進來」：PATCH 分支只更新 status 與 reason，不該連帶改寫
	targetType 的大小寫與其他欄位，把正規化留在呼叫端才能各自決定範圍。
*/
func validateAdminForumReport(req *adminForumReportRequest) error {
	// target_type 沒有 CHECK 約束，值域只能靠這裡擋。這兩個值同時決定
	// ensureForumReportTarget 查哪張表，以及 SQL 裡 CASE WHEN 走哪個分支。
	if req.TargetType != "post" && req.TargetType != "comment" {
		return &requestError{message: "檢舉目標必須是 post 或 comment"}
	}
	// id 與檢舉人都不可以是空值：id 對應主鍵（必須 >= 1）；ReporterEmail 是唯一鍵
	// uq_forum_reports_reporter_target 的一部分，寫入空字串會讓不同的檢舉人看起來是同一人
	// 而互相衝突。
	if req.TargetID < 1 || req.ReporterEmail == "" {
		return &requestError{message: "檢舉目標與檢舉人不可為空"}
	}
	// 以 rune 計算長度：VARCHAR(500) 的限制是「字元」，而 len(string) 算的是位元組，
	// 中文會被算成 3 倍，會誤擋合法且不長的中文理由。
	// 空白字元也要擋下：呼叫端已 TrimSpace，純空白等同沒填。
	if req.Reason == "" || len([]rune(req.Reason)) > 500 {
		return &requestError{message: "檢舉原因需為 1 至 500 字"}
	}
	// 狀態機白名單。與 handleAdminForumReports 的查詢參數驗證是同一組值，
	// 差別只在這裡擋寫入、那裡擋查詢；兩邊都寫死同樣的值以免查詢與寫入的合法集合不一致。
	if req.Status != "PENDING" && req.Status != "RESOLVED" && req.Status != "REJECTED" {
		return &requestError{message: "invalid report status"}
	}
	return nil
}

/*
listAdminForumPosts 對應 GET /api/admin/forum/posts：分頁列出全部文章，
並在每篇文章內嵌該文章的所有留言，讓後台列表頁能就地展開留言編輯與刪除。

回應

	200	{"items": [...], "total": 總文章數, "page": 目前頁碼, "pages": 總頁數}
	500	資料庫或 media token 服務異常
	（401 由 handleAdminForumPosts 處理；本函式不判斷 method）

分頁為什麼用 page / pages 而不是 offset / hasMore

	使用面（frontend/web/forum-admin.js 的 renderPagination）是照 pages 逐頁渲染頁碼按鈕，
	因此後端必須給出確切的總頁數；使用者面 listForumPosts 採「往下滑」的互動，
	只需要 hasMore 就能工作。兩種互動的資料需求不同，所以實作也不同。
*/
func (s *Server) listAdminForumPosts(w http.ResponseWriter, r *http.Request) {
	// 刻意忽略 Atoi 的錯誤：沒有 page 參數與 page=abc 都會得到 0，統一夾成第 1 頁。
	// 選擇寬容而非回 400，是因為 page 只影響顯示位置，錯誤參數退化成首頁不會造成資料錯誤；
	// 反之，回 400 會讓整個管理頁（連帶文章、留言、檢舉三區塊）都載入失敗。
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	// page <= 0 會讓 offset 變成負數，因此先夾在 1 再計算 offset。
	if page < 1 {
		page = 1
	}
	// 每頁固定 25 筆：同時限制單次回應的大小，以及「每篇文章再查一次留言」所產生的查詢次數。
	const pageSize = 25
	offset := (page - 1) * pageSize

	// 先算總數：pages 需要它。COUNT(*) 在 InnoDB 上需要掃描，但列表本身已是分頁的，
	// 用一次額外計數換前端能畫出完整頁碼是划算的。
	var total int
	if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_posts`).Scan(&total); err != nil {
		internalError(w, "unable to count forum posts")
		return
	}
	// 按讚數與留言數以相關子查詢在主查詢內一次算完，避免在 Go 迴圈裡為每篇文章各打兩次
	// COUNT（那樣會變成 3N 次往返）。兩張子表都有以 post_id 開頭的索引
	// （idx_forum_post_likes_post_id、idx_forum_post_comments_post_id），
	// 因此每個子查詢都是索引查找而非全表掃描。
	// LIMIT / OFFSET 也使用佔位符：MySQL 允許 prepared statement 這麼做，
	// 不必把數字字串拼接進 SQL（拼接數字雖無注入風險，仍會失去使用 prepared statement 的能力）。
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, author_email, content, created_at, image_url,
		       (SELECT COUNT(*) FROM forum_post_likes WHERE post_id = forum_posts.id),
		       (SELECT COUNT(*) FROM forum_post_comments WHERE post_id = forum_posts.id)
		FROM forum_posts ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		internalError(w, "unable to load forum posts")
		return
	}
	// defer 寫在錯誤檢查之後：QueryContext 失敗時 rows 為 nil。
	defer rows.Close()

	// items 預設為空切片，沒有資料時 JSON 是 [] 而不是 null，前端可直接 map。
	items := make([]adminForumPost, 0)
	// mediaToken 延遲建立、整頁共用一個：附圖放在 /files/ 代理後需要短期 token 才能讀取，
	// 而每張圖各自申請會在 25 筆時產生 25 個 token。只有真的遇到有附圖的文章才申請，
	// 整頁無圖時完全不打 Redis。
	mediaToken := ""
	for rows.Next() {
		var item adminForumPost
		// Scan 順序對應 SELECT 的六個欄位加上兩個子查詢結果。
		if err := rows.Scan(&item.ID, &item.AuthorEmail, &item.Content, &item.CreatedAt, &item.ImageURL, &item.LikeCount, &item.CommentCount); err != nil {
			internalError(w, "unable to read forum posts")
			return
		}
		// 先取出庫值（純檔名）：以它的正規化結果判斷要不要簽發 token，
		// 舊資料若存的是無法辨識的值，就不會為了它白白打一次 Redis。
		imageName := s.forumImageFileName(item.ImageURL)
		if imageName != "" && mediaToken == "" {
			// 第一次遇到附圖才申請，後續文章沿用同一個 token。
			mediaToken, err = s.createMediaToken(r.Context())
			if err != nil {
				internalError(w, "unable to create media token")
				return
			}
		}
		// 庫值只是檔名，給瀏覽器的網址由 forumImageURL 以 FILES_SERVER_PUBLIC_URL
		// 現組成「公開網域 + /files/ + 檔名 + token」；無法辨識的舊值會變成空字串。
		item.ImageURL = s.forumImageURL(imageName, mediaToken)
		// 這裡是刻意的 N+1：每篇文章額外查一次留言，換取後台列表頁能直接就地編輯／刪除留言，
		// 不必另外開一個「留言管理」頁。成本由 pageSize = 25 封頂（每次請求最多 25 次），
		// 且走 idx_forum_post_comments_post_id 索引。
		// 副作用：此時外層 rows 仍未關閉，內層查詢必須另外向連線池借一條連線
		// （config 設定 DB_MAX_OPEN_CONNS=20、DB_MAX_IDLE_CONNS=5）。
		// 若把連線池調成 1，內層查詢會一直等不到空閒連線（外層 rows 要等迴圈結束才釋放），
		// 因此這個寫法隱含「連線池必須 > 1」的前提。
		comments, err := s.loadAdminForumComments(r.Context(), item.ID, item.CommentCount)
		if err != nil {
			internalError(w, "unable to load forum comments")
			return
		}
		item.Comments = comments
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		internalError(w, "unable to read forum posts")
		return
	}
	// pages 用整數運算做向上取整：(total + pageSize - 1) / pageSize。
	// 直接整除會在最後一頁不滿時少算一頁（例如 26 筆、每頁 25 會算成 1 頁）。
	// total = 0 時結果為 0；前端 forum-admin.js 以 (data.pages || 1) 兜底，仍會顯示第 1 頁按鈕。
	writeOK(w, map[string]interface{}{"items": items, "total": total, "page": page, "pages": (total + pageSize - 1) / pageSize})
}

/*
loadAdminForumComments 取出某篇文章的全部留言，依建立時間遞增。

抽出來的原因不是為了精簡，而是有第二個呼叫端：search.go 的
loadAdminForumPostsByIDs 也需要「後臺形態的貼文（含留言）」，否則搜尋結果
在後臺會失去就地編輯／刪除留言的能力。兩個呼叫端的差異只有「留言要不要
一起帶」，把 SQL 留在同一處才不會讓兩邊的排序或欄位悄悄分歧。

capacity 傳入 CommentCount 當預設容量（主查詢剛算出來的數字），省掉
append 的重新配置；數量為 0 時 make(..., 0) 仍回傳非 nil 切片，因此 JSON
是 [] 而不是 null，前端不需要防禦 null。

rows 用 defer 關閉（而非原來在每個錯誤分支手動 Close）：這個函式是獨立
的，defer 在回傳時一定會執行，呼叫端因此不必擔心連線是否歸還。呼叫端
那邊仍然要記住「外層 rows 未關閉時，內層查詢需要另外借一條連線」這個前提。
*/
func (s *Server) loadAdminForumComments(ctx context.Context, postID int64, capacity int) ([]adminForumComment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, post_id, author_email, content, created_at
		FROM forum_post_comments WHERE post_id = ?
		ORDER BY created_at ASC, id ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := make([]adminForumComment, 0, capacity)
	for rows.Next() {
		var comment adminForumComment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.AuthorEmail, &comment.Content, &comment.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	// rows.Err() 必須檢查：迭代中途的錯誤不會讓 Next() 回 false，
	// 漏檢會把「只讀到一部分」當成完整資料而讓管理介面少顯示留言。
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

/*
createAdminForumPost 對應 POST /api/admin/forum/posts，讓管理員以自己的身分發文
（公告、活動通知、置頂補充）。呼叫前已由 handleAdminForumPosts 完成 requireAdminForum。

回應

	201	{"ok": true, "id": 新文章主鍵}
	400	body 不是合法 JSON，或內容驗證失敗（空白／超過 10000 字）
	403	來源不可信
	500	資料庫錯誤
*/
func (s *Server) createAdminForumPost(w http.ResponseWriter, r *http.Request) {
	// 寫入前擋跨站請求：session cookie 由瀏覽器自動附上，只有來源檢查能確認這個 POST
	// 來自管理頁本身。沒有套用 csrf.go 的 requireTrustedOrigin 中介層，因為同一條路由
	// 還要服務 GET（唯讀，不需要來源檢查），中介層無法只包住寫入分支。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	var req adminForumPostRequest
	// 沒有套用 MaxBytesReader：此路徑的 body 只含一個 content 欄位，
	// 且 validateAdminForumPost 會擋掉超過 10000 字的內容（以 rune 計；UTF-8 每字 1~4 bytes，
	// 最壞約 40 KB），足以限制實際寫入量，額外的 body 上限對這個單欄位表單屬於多餘的防線。
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	if err := validateAdminForumPost(&req); err != nil {
		badRequest(w, err.Error())
		return
	}
	// 作者取自 session（也就是發文的管理員本人），不接受 body 傳入 authorEmail：
	// 後台發文要留下真實操作者紀錄，否則無法追蹤誰以管理員身分發布。
	// requireAdminForum 已確認同一個 session 的 is_admin 為 true，因此這裡不會寫入空字串。
	author := s.sessions.ResolveUser(r)
	// image_url 固定寫空字串：這個後台表單只有文字欄位，後台無法上傳附圖
	// （圖片只能經由 /api/forum/images 由登入使用者上傳並取得媒體 token）。
	// 寫空字串而非 NULL，是因為該欄位為 NOT NULL，且此處兩種寫法的語意相同。
	createdAt := time.Now()
	// 稽核：記 post.create，因為這裡是以管理員「自己」的身分發文，作者欄位
	// 就是操作者本人。以他人身分代發的那條路徑（user_admin_handlers.go 的
	// createAdminUserPost）才記 user.posts.create —— 兩者的差別正是「稽核
	// 紀錄上作者是不是操作者」。
	tx, err := s.beginAdminTx(r)
	if err != nil {
		internalError(w, "unable to create forum post")
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO forum_posts (author_email, content, image_url, created_at) VALUES (?, ?, ?, ?)`,
		author, req.Content, "", createdAt)
	if err != nil {
		internalError(w, "unable to create forum post")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to read created post")
		return
	}
	if err := s.recordAdminAction(r, tx, adminActionPostCreate, audit.TargetPost, strconv.FormatInt(id, 10), req.Content,
		audit.Change{Field: "authorEmail", Before: "", After: author},
		audit.Change{Field: "content", Before: "", After: req.Content}); err != nil {
		internalError(w, "unable to create forum post")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, "unable to read created post")
		return
	}
	// 與一般使用者的發文路徑一樣，MySQL 寫入成功後才更新搜尋索引（best-effort）。
	// 刻意放在 Commit 之後：索引是外部系統，這個交易只涵蓋 MySQL。
	s.indexForumPost(r.Context(), id, req.Content, author, createdAt)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
}

/*
handleAdminForumPost 處理 /api/admin/forum/posts/{id} 的單篇文章操作。

	PUT	只更新 content；作者、建立時間、附圖都不動
	DELETE	刪除整篇文章
	其他	methodNotAllowed 回 405

刪除的順序與外鍵的關係

	forum_post_comments 與 forum_post_likes 在 MigrateMySQL 中只建立了 INDEX，
	並沒有宣告 FOREIGN KEY 指向 forum_posts。因此刪除文章不需要先刪子表，
	也不會因外鍵約束失敗——這裡的單句 DELETE 就是完整流程，不需要交易。

	代價是會留下孤兒留言與孤兒按讚。它們不會出現在任何查詢結果中
	（列表都以 forum_posts 為主表，留言以 post_id = 存在的主鍵為條件），
	等同軟刪除。若日後改成宣告 ON DELETE CASCADE，這裡「先刪父表」就會失敗，
	必須改成在同一個交易裡先刪留言與按讚再刪文章。
*/
func (s *Server) handleAdminForumPost(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	// 路徑參數解析：server.go 以 "/api/admin/forum/posts/" 註冊這條路由，ServeMux 只做前綴
	// 比對，尾端的 ID 要自己切（TrimPrefix 去前綴、Trim 去多餘斜線、ParseInt 轉數字）。
	// id < 1 一律回 400：AUTO_INCREMENT 主鍵不可能 <= 0，這類請求必定是路徑錯誤而非
	// 「找不到」，用 400 表達請求不合法，同時省下一次必然空結果的 SQL。
	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/forum/posts/"), "/"), 10, 64)
	if err != nil || id < 1 {
		badRequest(w, "invalid post id")
		return
	}
	// 來源檢查刻意放在 method 分派之前（順序與 handleAdminForumComment 相反）：
	// 來源不可信時，非 POST 的請求只會拿到 403，不會洩漏「這篇文章存在、但不支援某方法」。
	// 也因為下方只處理 PUT 與 DELETE，此處的檢查不會擋掉任何唯讀操作。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	switch r.Method {
	case http.MethodPut:
		// PUT 的語意是「整篇內容取代」，但範圍只限 content 欄位：作者與建立時間屬於
		// 內容的來源紀錄，管理介面刻意不提供修改入口。
		var req adminForumPostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w, "invalid request")
			return
		}
		if err := validateAdminForumPost(&req); err != nil {
			badRequest(w, err.Error())
			return
		}
		// 只更新 content：不用動態組裝 SQL，因為後台能修改的欄位就只有這一個。
		// 若日後開放修改附圖或作者，請比照 handleAdminForumReport 的動態 UPDATE 做法，
		// 欄位名用白名單字串、值一律走佔位符。
		//
		// 稽核：改文必須記下改動前後的內容。原因是這是「以管理員身分改寫使用者
		// 的文字」，而使用者看到的是被改過的版本 —— 沒有前後對照就無法向他們
		// 說明改了什麼。兩個值都會被 audit 截到 200 字元（長文的前 200 字
		// 足以辨識，完整內容仍在 forum_posts）。
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to update forum post")
			return
		}
		defer tx.Rollback()

		var beforeContent string
		if err := tx.QueryRowContext(r.Context(),
			`SELECT content FROM forum_posts WHERE id = ?`, id).Scan(&beforeContent); err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load forum post")
			return
		}

		result, err := tx.ExecContext(r.Context(), `UPDATE forum_posts SET content = ? WHERE id = ?`, req.Content, id)
		if err != nil {
			internalError(w, "unable to update forum post")
			return
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			// 影響 0 筆就回 404：對不存在的文章做 UPDATE 不會出錯，只會沒有列被改動。
			// 需注意本專案 DSN 沒有加 clientFoundRows=true，MySQL 回報的是「實際變更列數」，
			// 因此「文章存在但內容與送進來完全相同」也會落到這個分支（此時 404 語意略為不精確，
			// 但對管理介面而言影響有限：使用者看到的就是「沒改成」）。
			http.NotFound(w, r)
			return
		}
		if err := s.recordAdminAction(r, tx, adminActionPostUpdate, audit.TargetPost, strconv.FormatInt(id, 10), beforeContent,
			audit.Change{Field: "content", Before: beforeContent, After: req.Content}); err != nil {
			internalError(w, "unable to update forum post")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to update forum post")
			return
		}
		// 內容改了，搜尋索引必須跟著改，否則管理介面搜得到使用者已看不到的舊文字。
		// 用「依 id 回讀整列再重建索引」而不是「把新內容塞進索引」：索引還有作者與
		// 建立時間兩個欄位（後臺以作者精確比對、同分時以時間排序），只送 content
		// 會把這兩項清成空值。
		//
		// 刻意放在 Commit 之後：搜尋索引是外部系統，而這個交易只涵蓋 MySQL。
		// 若在 Commit 之前呼叫而它失敗，索引會指向一筆被回滾的文章；反過來
		// （先提交再索引）最壞是索引暫時過期，而 reindexForumPostByID 會在
		// 下次讀取時以 id 為準修正。
		s.reindexForumPostByID(r.Context(), id)
		writeOK(w, map[string]bool{"ok": true})
	case http.MethodDelete:
		// 單一 DELETE 即完成：留言與按讚沒有外鍵約束，不需要先刪子表（理由見上方說明）。
		//
		// 稽核：刪文是不可逆的，而它是這個後臺最常被使用的操作（檢舉的
		// 「通過（刪文）」就走這條路）。把刪除前的內容與作者記下來，事後才能
		// 回答「這篇文是誰寫的、寫了什麼、為什麼被刪」。
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to delete forum post")
			return
		}
		defer tx.Rollback()

		var beforeAuthor, beforeContent string
		if err := tx.QueryRowContext(r.Context(),
			`SELECT author_email, content FROM forum_posts WHERE id = ?`, id).Scan(&beforeAuthor, &beforeContent); err != nil {
			if err == sql.ErrNoRows {
				// 沒有比對到資料列即視為不存在 → 404。
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load forum post")
			return
		}

		result, err := tx.ExecContext(r.Context(), `DELETE FROM forum_posts WHERE id = ?`, id)
		if err != nil {
			internalError(w, "unable to delete forum post")
			return
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			// 沒有比對到資料列即視為不存在 → 404。
			http.NotFound(w, r)
			return
		}
		if err := s.recordAdminAction(r, tx, adminActionPostDelete, audit.TargetPost, strconv.FormatInt(id, 10), beforeContent,
			audit.Change{Field: "authorEmail", Before: beforeAuthor, After: "（文章已刪除）"},
			audit.Change{Field: "content", Before: beforeContent, After: "（文章已刪除）"}); err != nil {
			internalError(w, "unable to delete forum post")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to delete forum post")
			return
		}
		// 一定要同步移除索引文件。檢舉的「通過（刪文）」走的正是這條路徑，
		// 留下殘留文件的話，搜尋結果會出現點進去是 404 的幽靈貼文。
		s.unindexForumPost(r.Context(), id)
		writeOK(w, map[string]bool{"ok": true})
	default:
		// 不可信來源已在前面被 403 擋下；走到這裡代表對方確實是管理員，
		// 只是用了這條路由不接受的方法（GET、HEAD 等），因此回 405 表示「路由存在但方法不對」。
		methodNotAllowed(w)
	}
}

/*
handleAdminForumComments 處理 /api/admin/forum/comments 的集合層級操作。

	GET	?postId=N 取得單篇文章的留言，依建立時間由舊到新排序（無分頁）
	POST	在指定文章下新增留言
	其他	methodNotAllowed 回 405

為什麼 GET 用查詢參數而不是路徑參數

	後台文章列表要能「展開某篇文章的留言」。若每篇文章各開一條路由，就會需要
	/api/admin/forum/posts/{id}/comments 這種路徑；改用 postId 查詢參數之後，
	posts 與 comments 兩組路由就足以涵蓋列表、展開與新增三種需求。
*/
func (s *Server) handleAdminForumComments(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		// postId 必填且必須 >= 1：留言沒有自己的列表頁，缺少 postId 就無法定位查詢範圍。
		// 解析失敗或 < 1 一律回 400，理由與路徑 ID 的處理相同
		//（AUTO_INCREMENT 主鍵不可能 <= 0，必定是請求本身不合法）。
		postID, err := strconv.ParseInt(r.URL.Query().Get("postId"), 10, 64)
		if err != nil || postID < 1 {
			badRequest(w, "invalid post id")
			return
		}
		// 母文章不存在時回空清單而非 404：文章被刪除後後台仍可能請求它的留言，
		// 此時「沒有留言」與「文章不存在」對前端畫面而言結果相同，不必額外區分。
		rows, err := s.db.QueryContext(r.Context(), `
			SELECT id, post_id, author_email, content, created_at
			FROM forum_post_comments WHERE post_id = ?
			ORDER BY created_at ASC, id ASC`, postID)
		if err != nil {
			internalError(w, "unable to load forum comments")
			return
		}
		defer rows.Close()
		// 預設空切片：沒有留言時 JSON 輸出 [] 而不是 null，前端 map 不必判空。
		items := make([]adminForumComment, 0)
		for rows.Next() {
			var item adminForumComment
			if err := rows.Scan(&item.ID, &item.PostID, &item.AuthorEmail, &item.Content, &item.CreatedAt); err != nil {
				internalError(w, "unable to read forum comments")
				return
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			// 迭代中途的錯誤不會讓 Next() 回傳 false，漏檢會把局部資料當成完整清單。
			internalError(w, "unable to read forum comments")
			return
		}
		writeOK(w, map[string]interface{}{"items": items})
	case http.MethodPost:
		// 來源檢查寫在 POST 分支內而不是函式開頭：GET 是唯讀，不需要防護。
		// 把檢查留在會改動資料的分支，才能讓同一個 handler 同時安全地服務兩者。
		if !s.isTrustedOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		var req adminForumCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w, "invalid request")
			return
		}
		// 這裡只驗 content：PostID 的存在性交給下面的 COUNT 檢查，因為那屬於「引用完整性」
		// 而非「格式合法性」，兩者對應的錯誤回應不同（400 vs 404）。
		if err := validateAdminForumComment(&req); err != nil {
			badRequest(w, err.Error())
			return
		}
		// 目標文章必須存在：forum_post_comments.post_id 沒有外鍵，
		// 這道 COUNT 檢查是防止產生孤兒留言的唯一一道，也是不能直接 INSERT 的原因。
		// 檢查與寫入之間沒有交易：兩者之間若有人刪掉文章，會留下一筆孤兒留言；
		// 以管理後台的低併發來看，這個縫隙可接受（孤兒留言不會出現在任何查詢結果中）。
		var postCount int
		if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_posts WHERE id = ?`, req.PostID).Scan(&postCount); err != nil {
			internalError(w, "unable to load forum post")
			return
		}
		if postCount == 0 {
			// 這裡用 http.NotFound（text/plain）而非 writeError 的 JSON 錯誤，
			// 與本檔案其他 404 保持一致；前端只檢查 response.ok、不解析錯誤內容，格式差異不影響使用。
			http.NotFound(w, r)
			return
		}
		// 留言作者取自 session（也就是留言的管理員本人），不接受 body 傳入，
		// 與 createAdminForumPost 相同：留下真實操作者。
		// 另注意 PostID 來自 body 而非路徑，因此完全依賴上面的存在性檢查把關。
		//
		// 稽核：管理員新增的留言要以留言的身分記錄（user.comments.create），
		// 而不是 comment.create —— 這筆留言在公開頁上會被當成該管理員說的話，
		// 稽核要能從「這位管理員代發過什麼」查出來。
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to create forum comment")
			return
		}
		defer tx.Rollback()

		author := s.sessions.ResolveUser(r)
		result, err := tx.ExecContext(r.Context(), `
			INSERT INTO forum_post_comments (post_id, author_email, content, created_at) VALUES (?, ?, ?, ?)`,
			req.PostID, author, req.Content, time.Now())
		if err != nil {
			internalError(w, "unable to create forum comment")
			return
		}
		id, err := result.LastInsertId()
		if err != nil {
			internalError(w, "unable to read created comment")
			return
		}
		if err := s.recordAdminAction(r, tx, adminActionCommentPost, audit.TargetComment, strconv.FormatInt(id, 10), req.Content,
			audit.Change{Field: "postId", Before: "", After: strconv.FormatInt(req.PostID, 10)},
			audit.Change{Field: "authorEmail", Before: "", After: author},
			audit.Change{Field: "content", Before: "", After: req.Content}); err != nil {
			internalError(w, "unable to create forum comment")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to read created comment")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
	default:
		methodNotAllowed(w)
	}
}

/*
handleAdminForumComment 處理 /api/admin/forum/comments/{id} 的單則留言操作。

	PUT	更新留言內容
	DELETE	刪除留言
	其他	methodNotAllowed 回 405

與 handleAdminForumPost 的兩處差異（實作順序不同，行為差異僅限錯誤狀態碼）

	檢查順序：這裡先做來源檢查再解析路徑 ID，handleAdminForumPost 則先解析 ID。
	因此「合法方法 + 不支援的 ID 格式」時，本函式會先回 403、handleAdminForumPost 會先回 400。
	兩者對「不支援的方法 + 不可信來源」都回 403。前端只依 response.ok 判斷成敗，
	不依賴 400 與 403 的區別，因此這個差異不影響現有頁面。
*/
func (s *Server) handleAdminForumComment(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	// 這個 handler 只支援 PUT 與 DELETE，兩者都會改動資料，所以來源檢查放在最前面，
	// 後面的每個 method 都已確定屬於寫入路徑。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	// 路徑參數解析：TrimPrefix 去掉 "/api/admin/forum/comments/"、Trim 去掉多餘的斜線、
	// ParseInt 轉為主鍵。解析失敗或 id < 1 一律回 400：AUTO_INCREMENT 主鍵不可能 <= 0，
	// 這類請求必定是路徑寫錯而不是「找不到」。
	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/forum/comments/"), "/"), 10, 64)
	if err != nil || id < 1 {
		badRequest(w, "invalid comment id")
		return
	}
	switch r.Method {
	case http.MethodPut:
		// adminForumCommentRequest 同時帶 PostID，但這裡只使用 body 的 content：
		// 刻意不允許把留言搬到另一篇文章，避免出現 post_id 與路徑語意不一致的資料。
		var req adminForumCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w, "invalid request")
			return
		}
		if err := validateAdminForumComment(&req); err != nil {
			badRequest(w, err.Error())
			return
		}
		// 只更新 content：留言的作者、建立時間、所屬文章都是不可變的來源紀錄。
		//
		// 稽核：同 post.update —— 記下改動前後的內容，因為使用者看到的是被改過
		// 的版本，沒有前後對照就無法向他們說明。
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to update forum comment")
			return
		}
		defer tx.Rollback()

		var beforeContent string
		if err := tx.QueryRowContext(r.Context(),
			`SELECT content FROM forum_post_comments WHERE id = ?`, id).Scan(&beforeContent); err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load forum comment")
			return
		}

		result, err := tx.ExecContext(r.Context(), `UPDATE forum_post_comments SET content = ? WHERE id = ?`, req.Content, id)
		if err != nil {
			internalError(w, "unable to update forum comment")
			return
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			// 影響 0 筆 → 沒有留言符合這個 id → 404。與文章、檢舉的處理一致。
			// 附帶限制：DSN 沒加 clientFoundRows=true，MySQL 回報的是實際變更列數，
			// 因此「留言存在但內容與送進來完全相同」也會落到這個分支。
			http.NotFound(w, r)
			return
		}
		if err := s.recordAdminAction(r, tx, adminActionCommentPut, audit.TargetComment, strconv.FormatInt(id, 10), beforeContent,
			audit.Change{Field: "content", Before: beforeContent, After: req.Content}); err != nil {
			internalError(w, "unable to update forum comment")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to update forum comment")
			return
		}
		writeOK(w, map[string]bool{"ok": true})
	case http.MethodDelete:
		// 刪除留言不影響其父文章，也沒有任何表以留言為父（forum_reports 只存數字 target_id，
		// 沒有外鍵），因此單句 DELETE 即完成，不需要先刪子表。
		//
		// 稽核：同 post.delete —— 記下刪除前的作者與內容。
		tx, err := s.beginAdminTx(r)
		if err != nil {
			internalError(w, "unable to delete forum comment")
			return
		}
		defer tx.Rollback()

		var beforeAuthor, beforeContent string
		if err := tx.QueryRowContext(r.Context(),
			`SELECT author_email, content FROM forum_post_comments WHERE id = ?`, id).Scan(&beforeAuthor, &beforeContent); err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			internalError(w, "unable to load forum comment")
			return
		}

		result, err := tx.ExecContext(r.Context(), `DELETE FROM forum_post_comments WHERE id = ?`, id)
		if err != nil {
			internalError(w, "unable to delete forum comment")
			return
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			http.NotFound(w, r)
			return
		}
		if err := s.recordAdminAction(r, tx, adminActionCommentDel, audit.TargetComment, strconv.FormatInt(id, 10), beforeContent,
			audit.Change{Field: "authorEmail", Before: beforeAuthor, After: "（留言已刪除）"},
			audit.Change{Field: "content", Before: beforeContent, After: "（留言已刪除）"}); err != nil {
			internalError(w, "unable to delete forum comment")
			return
		}
		if err := tx.Commit(); err != nil {
			internalError(w, "unable to delete forum comment")
			return
		}
		writeOK(w, map[string]bool{"ok": true})
	default:
		// 不可信來源已在前面被 403 擋下；走到這裡代表對方確實是管理員，
		// 只是用了本路由不接受的方法。
		methodNotAllowed(w)
	}
}

/*
validateAdminForumComment 檢查後台留言的內容欄位，並在檢查前先就地 TrimSpace。
供 handleAdminForumComments 的 POST／PUT 與 handleAdminForumComment 的 PUT 使用，
也供 user_admin_handlers.go 的 createAdminUserComment 使用。

回傳

	nil		通過
	*requestError	其 Error() 就是要回給前端的繁體中文訊息，呼叫端一律以 badRequest 回 400

不檢查 PostID 的原因

	PostID 屬於「引用完整性」而非「格式合法性」：它必須在建 INSERT 之前實際查證父文章是否存在
	（見 handleAdminForumComments 的 POST 分支），錯誤回應也應該是 404 而不是 400。
	而在 PUT 情境中 PostID 根本不被使用，因此把它放進這個格式驗證函式只會造成誤解。

字串以 rune 計算長度而非位元組，理由見 validateAdminForumPost。
*/
func validateAdminForumComment(req *adminForumCommentRequest) error {
	// 就地修剪，讓「只貼空白」與「完全沒填」走同一個錯誤訊息。
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return &requestError{message: "留言內容不可為空"}
	}
	// 以 rune 計算：2000 是應用層自行設定的字元上限（與使用者面留言的 2000 字上限一致），
	// MySQL 的 TEXT 本身能存更多。用 len(string) 會把中文算成 3 倍位元組而誤判超長。
	if len([]rune(req.Content)) > 2000 {
		return &requestError{message: "留言最多 2000 字"}
	}
	return nil
}

/*
validateAdminForumPost 檢查後台文章的內容欄位，並在檢查前先就地 TrimSpace。
供 createAdminForumPost 與 handleAdminForumPost 的 PUT 使用，
也供 user_admin_handlers.go 的 createAdminUserPost 使用。

回傳

	nil		通過
	*requestError	其 Error() 就是要回給前端的繁體中文訊息，呼叫端一律以 badRequest 回 400

為什麼以 rune 而不是 len(string) 計算長度

	10000 是「字」數。len(string) 算的是位元組，UTF-8 中文每字 3 bytes，
	會讓合法內容在約 3333 字就被誤判超長；len([]rune(s)) 才與限制的語意一致。
	這也與使用者在貼上內容時看到的字數相同，回傳的錯誤訊息才不會名不副實。

只驗 content 的原因：這個請求型別刻意不提供 authorEmail 與 imageUrl，
後台無法改動文章作者或附圖（見 createAdminForumPost 的說明），沒有其他欄位需要驗證。
*/
func validateAdminForumPost(req *adminForumPostRequest) error {
	// 就地修剪，讓「只貼空白」與「完全沒填」走同一個錯誤訊息。
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return &requestError{message: "內容不可為空"}
	}
	if len([]rune(req.Content)) > 10000 {
		return &requestError{message: "內容最多 10000 字"}
	}
	return nil
}

/*
requestError 是「訊息可以直接顯示給使用者」的驗證錯誤型別。

存在的目的：讓驗證函式同時滿足兩個需求——回傳 error 讓呼叫端用單一 if 判斷失敗，
又能讓 badRequest(w, err.Error()) 輸出繁體中文訊息，而不是把內部錯誤細節丟給前端。
與一般 fmt.Errorf 的差別在於「這個字串就是要給人看的」這個語意。

使用範圍：本套件（httpapi）內。message 未導出，因此其他套件無法讀取；
user_admin_handlers.go 的 decodeAdminUserEmail 也回傳這個型別（定義於本檔案）。
目前使用它的函式：validateAdminForumReport、validateAdminForumComment、validateAdminForumPost。
*/
type requestError struct{ message string }

// Error 實作 error 介面，讓 *requestError 能直接回傳給呼叫端。
// 刻意只回傳 message、不包裝底層原因：這類錯誤都源自使用者輸入，沒有值得診斷的底層細節。
func (e *requestError) Error() string { return e.message }
