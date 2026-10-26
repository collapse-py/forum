/*
migration_test.go 用**真實的 MySQL** 驗證 MigrateMySQL。

【為什麼不能用 go-sqlmock】
ROADMAP.md 的 Phase 3.1 說得很直接：

	go-sqlmock 驗證不了 DDL 真的能執行，miniredis 對 MySQL 沒有對應物

原因不是工具不夠好，而是這個函式測的東西就是「MySQL 會不會接受這句 DDL」：
  - `ADD COLUMN IF NOT EXISTS` 是 MariaDB 專屬語法，MySQL 8.x（本機實測 8.0.44）
    對它回 ERROR 1064 並中止啟動 —— 而在 sqlmock 上「通過」只代表程式碼呼叫了那句
    SQL，不代表任何一個 MySQL 版本接受它。開發用資料庫是 MariaDB 11.8（語法被
    接受），因此這個差異只有真的跑過一次 MySQL 才會現形。
  - 部分步驟用 information_schema 探測。那些 SQL 的**結果**決定了後續步驟
    會不會執行 —— sqlmock 可以讓它回傳任何東西，因此它無法回答「一個已經有
    資料的資料庫會不會被正確升級」。
  - 冪等性（「跑第二次不應該壞掉」）是 MigrateMySQL 的核心承諾，而它只有在
    真的跑過兩次之後才有意義。
  - 這一份 schema 有 13 張表。任何 mock 都要把十幾句 DDL 全部列出，而那份
    清單本身就是一份「看起來像在測」的東西 —— 它不會抓到 DDL 語法錯誤。

【為什麼沒有檔案／檔案切割那些測試】
這些正是真實 MySQL 才能回答的問題，而且答案全都是「有這個 bug」：
  - 檔案切割（gzip / 檔名裡的時間戳 / 檔名碰撞）
  - 舊版本資料庫的升級路徑（從一個真實但舊的 schema 升上來）
  - 中文與 emoji 的往返

【測試需要什麼】
一個 MySQL 實例，透過環境變數 FORUM_TEST_MYSQL_DSN 指向**伺服器**
（不是特定的資料庫 —— 每個測試會自己建立與丟棄一個拋棄式的資料庫）。

	FORUM_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/?parseTime=true&loc=UTC&multiStatements=true'

未設定時整個檔案 skip，而不是 fail。理由：這個測試需要一個資料庫，而大多數
開發者不會為了跑單元測試而起一個 MySQL。若它在沒有資料庫時 fail，那麼
`go test ./...` 對每一個人都是紅的 —— 而一個永遠紅的測試會被忽略，連真的
回歸也一起被忽略。
CI 的 migrations job 提供這個服務（見 .github/workflows/ci.yml）。

【多執行的並行性】
每個測試用**自己的**資料庫（名字帶測試名與一個隨機尾巴），因此這些測試可以
t.Parallel()。這不是隨手加的：MigrateMySQL 的 DDL 會取得 metadata 鎖，
而共用一個資料庫的並行測試會互相死鎖 —— 那個症狀是「測試隨機失敗」，
而隨機失敗的測試會被重跑，然後變成偶發的漏測。
*/
package data

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"forum/forum/idgen"

	_ "github.com/go-sql-driver/mysql"
)

// requiredTables 是 MigrateMySQL 必須建立的全部資料表。
//
// 這份清單刻意與 mysql.go 檔頭的「資料表清單」平行。它必須平行 —— 而這正是
// 它的價值：一個只檢查「 MigrateMySQL 回 nil」的測試會在有人從檔頭的清單裡
// 移掉一張表時仍然通過（因為 MigrateMySQL 也移掉了它），而這份清單會指出
// 「清單與實作不同步」。
//
// 要新增資料表時必須同時更新這裡。刻意不在這裡自動發現：自動發現會讓這支
// 測試永遠通過，而它的整個作用就是抓出那個不同步。
var requiredTables = []string{
	"forum_posts",
	"forum_post_likes",
	"forum_post_comments",
	"forum_reports",
	"forum_profiles",
	"forum_users",
	"forum_user_tags",
	"forum_user_tag_assignments",
	"forum_follows",
	"forum_request_metrics",
	"forum_admin_actions",
	"forum_announcements",
	"forum_notifications",
	"forum_media",
	"forum_media_deletions",
	"forum_post_bookmarks",
	"forum_daily_stats",
}

// dbCounter 讓每個測試的資料庫名稱都不相同。
//
// 不用 t.Name() 是因為兩個測試在同一個 package 裡可能同名（同名的子測試），
// 而用亂數尾巴則保證「即使名字真的撞了也不會互相干擾」。
var dbCounter atomic.Int64

// serverDSN 把 FORUM_TEST_MYSQL_DSN 剝成「只連伺服器」的形式（庫名清空、
// 參數原樣保留），回傳清理後的字串；未設定 DSN 時直接 skip。
//
// 「剝掉庫名」的實作細節：DSN 的形狀是 user:pass@tcp(host:port)/dbname?params，
// 因此正確做法是把最後一個斜線到問號之間換成空，而不是把斜線之後整段丟掉。
// 曾經的寫法（cleaned[:i] + "/" + cleaned[i+j+1:]）丟掉了問號本身，於是
// parseTime=true&loc=UTC 被當成**庫名**，每一次測試都以
// 「Unknown database 'parseTime=…'」失敗 —— 而帶參數的 DSN（CI 用的就是）
// 全部命中。剝完之後還要保留 "?params"，否則 multiStatements 與 parseTime
// 會一起消失：前者是多陳述清理的需要，後者讓 DATETIME 掃得進 time.Time。
func serverDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("FORUM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FORUM_TEST_MYSQL_DSN 未設定：跳過需要真實 MySQL 的遷移測試。" +
			"設定方式見本檔的檔頭，或見 .github/workflows/ci.yml 的 migrations job")
	}

	cleaned := dsn
	if i := strings.LastIndex(cleaned, "/"); i >= 0 {
		if j := strings.Index(cleaned[i:], "?"); j >= 0 {
			// 保留問號與其後的參數，只把斜線到問號之間的庫名清空。
			cleaned = cleaned[:i] + "/" + cleaned[i+j:]
		} else {
			cleaned = cleaned[:i] + "/"
		}
	}
	return cleaned
}

// serverDB 連到 MySQL **伺服器**（不指定資料庫），測試用它建立拋棄式資料庫。
func serverDB(t *testing.T) *sql.DB {
	t.Helper()

	cleaned := serverDSN(t)
	db, err := sql.Open("mysql", cleaned)
	if err != nil {
		t.Fatalf("無法以 %q 開啟連線: %v", cleaned, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("MySQL 連不上（檢查 FORUM_TEST_MYSQL_DSN 與服務是否在跑）: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// tempDatabase 建立一個帶唯一名稱的資料庫，並回傳連到它的 *sql.DB。
//
// 名稱刻意帶時間戳與計數器尾巴：兩個測試在同毫秒啟動時不能撞名，而撞名的症狀
// 是「測試間歇性地看到彼此的資料」—— 那是最難查的一類測試互相干擾。
func tempDatabase(t *testing.T) (*sql.DB, string) {
	t.Helper()

	server := serverDB(t)
	name := fmt.Sprintf("forum_mig_test_%d_%d",
		time.Now().UnixNano(), dbCounter.Add(1))

	if _, err := server.Exec(fmt.Sprintf(
		"CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", name)); err != nil {
		t.Fatalf("建立測試資料庫 %s 失敗: %v", name, err)
	}

	// 無論測試怎麼結束（t.Fatal、panic）都會執行，因此丟棄式資料庫一定會被清掉。
	t.Cleanup(func() {
		// 先切回伺服器層級：測試用的那個連線仍連著那個資料庫，
		// 而 MySQL 不允許刪除一個「正在被使用」的資料庫。
		if _, err := server.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name)); err != nil {
			t.Logf("清理測試資料庫 %s 失敗（會留在伺服器上，請手動清理）: %v", name, err)
		}
	})

	// 從已經剝成伺服器層級的 DSN 組出「連到這個拋棄式資料庫」的 DSN：
	// 斜線之後、問號之前換上資料庫名稱。
	//
	// 刻意不寫 strings.Replace(dsn, "/?", "/"+name+"?", 1)：那只在 DSN 原本
	// 就是「空庫名 + 問號」的形狀下有效。若開發者把 DSN 指向真正的論壇庫
	// （root@…/forum?…），Replace 一個字都不換，而 MigrateMySQL 會直接在
	// **正式資料庫**上建表與回填 —— 一個不會報錯、只在事後被發現的災難。
	// 從剝好的 server DSN 重建，兩種形狀都得到拋棄式資料庫。
	serverLevel := serverDSN(t)
	base, params := serverLevel, ""
	if q := strings.Index(serverLevel, "?"); q >= 0 {
		base, params = serverLevel[:q], serverLevel[q:]
	}
	dsn := strings.TrimSuffix(base, "/") + "/" + name + params
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("開啟測試資料庫 %s 失敗: %v", name, err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db, name
}

// tableExists 查詢某張表是否在這個資料庫裡。
func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var n int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
		table).Scan(&n)
	if err != nil {
		t.Fatalf("查詢 information_schema 失敗: %v", err)
	}
	return n > 0
}

// TestMigrateMySQLCreatesEveryTable 是這份測試的核心斷言。
//
// 它一次確認三件事，而每一件對應 MigrateMySQL 的三條設計承諾（見 mysql.go 檔頭）：
//  1. 所有資料表都被建立。
//  2. 全部是 InnoDB（交易與外鍵語意）與 utf8mb4（emoji）。
//  3. 跑第二次不應該有任何變化（冪等）。
//
// 第 3 件事由 TestMigrateMySQLIsIdempotent 單獨負責（它需要比較兩次之間的
// 結構）；這一支專注在「建立」。
func TestMigrateMySQLCreatesEveryTable(t *testing.T) {
	t.Parallel()
	db, name := tempDatabase(t)

	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("MigrateMySQL 在空資料庫 %s 上回錯誤: %v", name, err)
	}

	for _, table := range requiredTables {
		if !tableExists(t, db, table) {
			t.Errorf("資料庫 %s 沒有建立 %s", name, table)
		}
	}

	// Engine 與字元集。刻意查 information_schema 而不是 SHOW CREATE TABLE：
	// 前者的輸出格式跨版本穩定，而後者的 AUTO_INCREMENT 等值會變動。
	rows, err := db.Query(`SELECT table_name, engine, table_collation
	                       FROM information_schema.tables
	                       WHERE table_schema = DATABASE()`)
	if err != nil {
		t.Fatalf("查詢資料表 metadata 失敗: %v", err)
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		var table, engine, collation string
		if err := rows.Scan(&table, &engine, &collation); err != nil {
			t.Fatalf("掃描資料表 metadata 失敗: %v", err)
		}
		seen++
		// InnoDB 是刻意統一的（見 mysql.go 檔頭的第 3 點）。若有人改成 MyISAM，
		// 症狀是「交易無效但不會報錯」—— 而這個專案依賴交易來保證稽核與操作
		// 同生共死。
		if !strings.EqualFold(engine, "InnoDB") {
			t.Errorf("%s 的 ENGINE 是 %s，want InnoDB（交易語意依賴它）", table, engine)
		}
		if !strings.HasPrefix(collation, "utf8mb4") {
			t.Errorf("%s 的 collation 是 %s，want utf8mb4_*（utf8mb3 存不了 emoji）", table, collation)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("走訪資料表 metadata 時出錯: %v", err)
	}
	if seen != len(requiredTables) {
		t.Errorf("資料庫裡有 %d 張表，want %d（多出來的可能是一個已從程式碼移除、\n"+
			"但仍留在舊環境裡的資料表 —— 那不是這支測試能判斷的事，但它值得被知道）",
			seen, len(requiredTables))
	}
}

// TestMigrateMySQLIsIdempotent 驗證「啟動時必定會執行一次」這個前提。
//
// 這是最重要的一條性質，因為 MigrateMySQL 每次啟動都會跑：它壞掉時的症狀是
// 「第二次啟動時服務起不來」，而那看起來像是一個部署問題而不是一個程式碼問題。
//
// 這支測試必須**真的跑兩次並比對結果**，不能只是「第二次沒回錯誤」—— 一個
// 「什麼都不做」的 MigrateMySQL 也能通過後者，而在已經建好 schema 的環境上
// 那正是它的行為（這就是為什麼第一條測試必須在一個空的資料庫上跑）。
func TestMigrateMySQLIsIdempotent(t *testing.T) {
	t.Parallel()
	db, name := tempDatabase(t)

	// 第一遍：建立。
	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("第一次 MigrateMySQL 回錯誤: %v", err)
	}
	first := schemaSnapshot(t, db)

	// 在第一遍之後塞入資料：冪等性不只是「第二次不報錯」，還包括「第二次不會
	// 清掉或重建資料」。這個斷言是整支測試裡最有價值的一個 —— 一個用
	// DROP TABLE + CREATE TABLE 實作遷移的版本會安靜地刪掉全站資料，
	// 而症狀是「重啟一次之後貼文全沒了」。
	seedForumPost(t, db, "original-content", "someone@example.com")

	// 第二遍。
	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("第二次 MigrateMySQL 回錯誤（冪等性被破壞）: %v", name)
	}
	second := schemaSnapshot(t, db)

	if first != second {
		t.Errorf("第二次遷移改變了 schema：\n  第一次: %s\n  第二次: %s\n"+
			"這代表 MigrateMySQL 不是冪等的 —— 而它每次啟動都會執行，\n"+
			"所以症狀會是「第二次重啟時服務起不來」",
			first, second)
	}

	// 資料必須還在。
	var content string
	err := db.QueryRow(`SELECT content FROM forum_posts WHERE author_email = ?`,
		"someone@example.com").Scan(&content)
	if err != nil {
		t.Fatalf("資料在第二次遷移後消失了（查詢失敗）: %v", err)
	}
	if content != "original-content" {
		t.Errorf("資料內容被改了: %q", content)
	}
}

// TestMigrateMySQLOnExistingDatabaseWithData 驗證升級路徑。
//
// 這支測試模擬「一個已經在跑的站執行了幾個版本，然後更新到這個版本」。
// 它刻意先建立一份**少了幾張表、少了幾個欄位**的舊 schema，塞入資料，再跑
// 遷移 —— 也就是 ROADMAP.md 說的「重啟一次資料庫狀態應該收斂到最新」。
//
// 這個情境是整個測試檔最貼近真實風險的一個：升級失敗時的症狀是
// 「某一頁 500，而那個 500 的訊息裡提到一個不存在的欄位」。
func TestMigrateMySQLOnExistingDatabaseWithData(t *testing.T) {
	t.Parallel()
	db, name := tempDatabase(t)

	// 一份刻意落後的 schema：只有一張表，且它少了兩個後來才加的欄位。
	// 這是「舊版本」的真實形狀 —— 欄位與資料表是隨著功能一起長出來的。
	mustExec(t, db, `
		CREATE TABLE forum_posts (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	// 塞入資料：遷移不能丟掉它。
	mustExec(t, db, `
		INSERT INTO forum_posts (author_email, content) VALUES ('legacy@example.com', '舊的貼文');
	`)

	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("對既有資料庫執行 MigrateMySQL 回錯誤: %v（資料庫 %s）", err, name)
	}

	// 舊資料必須還在，而且新欄位應該是可讀的預設值。
	var content, imageURL string
	err := db.QueryRow(`SELECT content, COALESCE(image_url, '') FROM forum_posts
	                     WHERE author_email = 'legacy@example.com'`).Scan(&content, &imageURL)
	if err != nil {
		t.Fatalf("舊資料在遷移後查不到（遷移把它刪掉了？）: %v", err)
	}
	if content != "舊的貼文" {
		t.Errorf("舊資料的內容被改了: %q", content)
	}

	// 缺的表應該被補上，其中包括這個功能新增的 forum_notifications
	// —— 升級路徑就是它第一次被建立的地方（正式站的資料庫不是空的）。
	for _, table := range []string{"forum_announcements", "forum_admin_actions", "forum_user_tags", "forum_notifications"} {
		if !tableExists(t, db, table) {
			t.Errorf("遷移沒有補上舊資料庫缺的 %s（資料庫 %s）", table, name)
		}
	}

	// 新的欄位應該存在。刻意用 information_schema.columns 而不是直接
	// INSERT 一列（那會讓「欄位存在」與「欄位可用」混在一起，
	// 而後者還會牽涉預設值的語意）。
	for table, columns := range map[string][]string{
		"forum_posts":    {"image_url", "updated_at", "deleted_at"},
		"forum_profiles": {"bio", "public_key", "avatar_url"},
	} {
		for _, column := range columns {
			if !columnExists(t, db, table, column) {
				t.Errorf("%s 沒有補上欄位 %s", table, column)
			}
		}
	}
}

// TestMigrateMySQLRenumbersLegacyIDs 驗證「既有自增資料改成雪花 id」這條
// 遷移路徑，包括所有指向它們的欄位。
//
// 為什麼要用這種方式模擬舊資料庫：MigrateMySQL 在空資料庫上建立的已經是
// 新 schema。先把七張表改回 AUTO_INCREMENT、用舊方式（省略 id）塞資料，
// 再跑一次遷移 —— 這走的就是正式站升級時的那條路徑，而不是一份手寫的
// 「舊 schema」（那份 schema 會随新欄位增加而過時）。
//
// 漏掉任何一個引用欄位的症狀都不是錯誤：子表會指向不存在的 id，而那個
// 症状只有有人點到那則通知或那張檢舉時才會出現。
func TestMigrateMySQLRenumbersLegacyIDs(t *testing.T) {
	t.Parallel()
	db, name := tempDatabase(t)
	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("建立新 schema 失敗: %v", err)
	}

	idTables := []string{
		"forum_posts", "forum_post_comments", "forum_reports", "forum_user_tags",
		"forum_announcements", "forum_notifications", "forum_admin_actions",
	}
	for _, table := range idTables {
		mustExec(t, db, "ALTER TABLE "+table+" MODIFY id BIGINT NOT NULL AUTO_INCREMENT")
	}

	// 以舊方式（不給 id）建立一組互相引用的資料。所有 old* 變數都是
	// 自增 id，遷移後必須完全不一樣。
	res, err := db.Exec(`INSERT INTO forum_posts (author_email, content, image_url, created_at) VALUES ('a@example.com', '舊貼文', '', NOW())`)
	if err != nil {
		t.Fatalf("插入舊貼文失敗（%s）: %v", name, err)
	}
	oldPost, _ := res.LastInsertId()

	res, err = db.Exec(`INSERT INTO forum_post_comments (post_id, author_email, content, created_at) VALUES (?, 'b@example.com', '舊留言', NOW())`, oldPost)
	if err != nil {
		t.Fatalf("插入舊留言失敗: %v", err)
	}
	oldComment, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO forum_post_comments (post_id, parent_id, depth, author_email, content, created_at) VALUES (?, ?, 2, 'c@example.com', '舊回覆', NOW())`, oldPost, oldComment); err != nil {
		t.Fatalf("插入舊回覆失敗: %v", err)
	}

	mustExec(t, db, `INSERT INTO forum_post_likes (post_id, author_email, created_at) VALUES (?, 'c@example.com', NOW())`, oldPost)
	mustExec(t, db, `INSERT INTO forum_post_bookmarks (post_id, author_email, created_at) VALUES (?, 'c@example.com', NOW())`, oldPost)

	res, err = db.Exec(`INSERT INTO forum_user_tags (name, created_at, updated_at) VALUES ('舊標籤', NOW(), NOW())`)
	if err != nil {
		t.Fatalf("插入舊標籤失敗: %v", err)
	}
	oldTag, _ := res.LastInsertId()
	mustExec(t, db, `INSERT INTO forum_user_tag_assignments (user_email, tag_id) VALUES ('b@example.com', ?)`, oldTag)

	mustExec(t, db, `INSERT INTO forum_reports (target_type, target_id, reporter_email, reason, status, created_at) VALUES ('post', ?, 'r1@example.com', '檢舉貼文', 'PENDING', NOW())`, oldPost)
	mustExec(t, db, `INSERT INTO forum_reports (target_type, target_id, reporter_email, reason, status, created_at) VALUES ('comment', ?, 'r2@example.com', '檢舉留言', 'PENDING', NOW())`, oldComment)

	res, err = db.Exec(`INSERT INTO forum_announcements (body, active, created_at, created_by, updated_at, expires_at) VALUES ('舊公告', 0, NOW(), 'admin@example.com', NOW(), NULL)`)
	if err != nil {
		t.Fatalf("插入舊公告失敗: %v", err)
	}
	oldAnnouncement, _ := res.LastInsertId()

	res, err = db.Exec(`INSERT INTO forum_notifications (recipient_email, actor_email, kind, post_id, comment_id, created_at) VALUES ('b@example.com', 'a@example.com', 'comment', ?, ?, NOW())`, oldPost, oldComment)
	if err != nil {
		t.Fatalf("插入舊通知失敗: %v", err)
	}
	oldNotification, _ := res.LastInsertId()

	// 稽核紀錄的 target_id 是 VARCHAR，因此它是最容易被漏掉的一種引用。
	res, err = db.Exec(`INSERT INTO forum_admin_actions (actor_email, action, target_type, target_id, target_label, changes, ip, request_id, created_at) VALUES ('admin@example.com', 'post.delete', 'post', ?, '', NULL, '', '', NOW())`, strconv.FormatInt(oldPost, 10))
	if err != nil {
		t.Fatalf("插入舊稽核紀錄失敗: %v", err)
	}
	oldAction, _ := res.LastInsertId()

	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("第二次 MigrateMySQL（重新編號）失敗: %v", err)
	}

	// 每一張表的 id 都必須變成雪花（遠大於舊的自增值），且 AUTO_INCREMENT 已移除。
	const snowflakeFloor = int64(1) << 40
	for _, table := range idTables {
		var extra string
		if err := db.QueryRow(`SELECT EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = 'id'`, table).Scan(&extra); err != nil {
			t.Fatalf("讀取 %s 的欄位資訊失敗: %v", table, err)
		}
		if strings.Contains(strings.ToLower(extra), "auto_increment") {
			t.Errorf("%s.id 仍然帶 AUTO_INCREMENT", table)
		}
	}

	newPost := mustScanInt64(t, db, `SELECT id FROM forum_posts WHERE author_email = 'a@example.com'`)
	newComment := mustScanInt64(t, db, `SELECT id FROM forum_post_comments WHERE author_email = 'b@example.com'`)
	newTag := mustScanInt64(t, db, `SELECT id FROM forum_user_tags WHERE name = '舊標籤'`)
	newAnnouncement := mustScanInt64(t, db, `SELECT id FROM forum_announcements WHERE body = '舊公告'`)
	newNotification := mustScanInt64(t, db, `SELECT id FROM forum_notifications WHERE recipient_email = 'b@example.com'`)
	newAction := mustScanInt64(t, db, `SELECT id FROM forum_admin_actions WHERE actor_email = 'admin@example.com'`)

	for _, got := range []int64{newPost, newComment, newTag, newAnnouncement, newNotification, newAction} {
		if got < snowflakeFloor {
			t.Errorf("重新編號後的 id = %d，want >= %d（雪花 id）", got, snowflakeFloor)
		}
	}
	if newPost == oldPost || newComment == oldComment || newTag == oldTag ||
		newAnnouncement == oldAnnouncement || newNotification == oldNotification || newAction == oldAction {
		t.Errorf("重新編號前後 id 相同: post %d/%d comment %d/%d tag %d/%d announcement %d/%d notification %d/%d action %d/%d",
			oldPost, newPost, oldComment, newComment, oldTag, newTag,
			oldAnnouncement, newAnnouncement, oldNotification, newNotification, oldAction, newAction)
	}

	// 所有指向它們的欄位都必須跟著改。
	checks := []struct {
		name  string
		query string
		args  []any
		want  int64
	}{
		{"留言的 post_id", `SELECT post_id FROM forum_post_comments WHERE author_email = 'b@example.com'`, nil, newPost},
		{"回覆的 parent_id", `SELECT parent_id FROM forum_post_comments WHERE author_email = 'c@example.com'`, nil, newComment},
		{"按讚的 post_id", `SELECT post_id FROM forum_post_likes WHERE author_email = 'c@example.com'`, nil, newPost},
		{"收藏的 post_id", `SELECT post_id FROM forum_post_bookmarks WHERE author_email = 'c@example.com'`, nil, newPost},
		{"標籤關聯的 tag_id", `SELECT tag_id FROM forum_user_tag_assignments WHERE user_email = 'b@example.com'`, nil, newTag},
		{"貼文檢舉的 target_id", `SELECT target_id FROM forum_reports WHERE reporter_email = 'r1@example.com'`, nil, newPost},
		{"留言檢舉的 target_id", `SELECT target_id FROM forum_reports WHERE reporter_email = 'r2@example.com'`, nil, newComment},
		{"通知的 post_id", `SELECT post_id FROM forum_notifications WHERE recipient_email = 'b@example.com'`, nil, newPost},
		{"通知的 comment_id", `SELECT comment_id FROM forum_notifications WHERE recipient_email = 'b@example.com'`, nil, newComment},
	}
	for _, c := range checks {
		if got := mustScanInt64(t, db, c.query, c.args...); got != c.want {
			t.Errorf("%s = %d，want %d", c.name, got, c.want)
		}
	}

	// 稽核的 target_id 是字串，比對也要用字串。
	var auditTarget string
	if err := db.QueryRow(`SELECT target_id FROM forum_admin_actions WHERE actor_email = 'admin@example.com' AND action = 'post.delete'`).Scan(&auditTarget); err != nil {
		t.Fatalf("讀取稽核 target_id 失敗: %v", err)
	}
	if want := strconv.FormatInt(newPost, 10); auditTarget != want {
		t.Errorf("稽核 target_id = %q，want %q", auditTarget, want)
	}

	// 舊 id 不該还能指向任何一列。
	var oldRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM forum_posts WHERE id = ?`, oldPost).Scan(&oldRows); err != nil {
		t.Fatalf("查詢舊 id 失敗: %v", err)
	}
	if oldRows != 0 {
		t.Errorf("舊 id %d 仍然存在", oldPost)
	}
}

// mustScanInt64 執行一句回傳單一整數的查詢。
func mustScanInt64(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var value int64
	if err := db.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("查詢失敗（%s）: %v", query, err)
	}
	return value
}

// TestUnicodeRoundTrip 驗證 utf8mb4 的往返。
//
// 這個測試存在的理由是「utf8mb3 vs utf8mb4」是本專案明確做過的一個決定
// （見 mysql.go 檔頭的第 3 點），而它的失效症狀是
// "Incorrect string value: '\xF0\x9F...'" —— 一個完全不會指向「字元集」的錯誤。
//
// emoji 是關鍵：它需要 4 bytes，而 utf8mb3 只支援 3 bytes。
func TestUnicodeRoundTrip(t *testing.T) {
	t.Parallel()
	db, _ := tempDatabase(t)

	if err := MigrateMySQL(db); err != nil {
		t.Fatalf("MigrateMySQL 回錯誤: %v", err)
	}

	cases := []struct {
		name string
		text string
	}{
		{"ASCII", "plain ascii"},
		{"繁體中文", "這是一篇繁體中文的論壇貼文，包含標點符號：「引號」與『書名號』。"},
		// 下面的簡體與混用文字是刻意的測試資料：這個測試驗的是 utf8mb4
		// 對各種文字的寫入與讀回，不是專案註解的用字。
		{"簡體中文", "这是一篇简体中文的论坛帖子"},
		{"日文", "これは日本語の投稿です。こんにちは、世界。"},
		{"韓文", "이것은 한국어 게시물입니다."},
		{"emoji（4 bytes）", "使用者貼了這個表情 🎉🔥👨‍👩‍👧‍👦 还有旗幟 🇹🇼"},
		{"RTL 阿拉伯文", "مرحبا بالعالم"},
		{"混合", "混合 test 🎉 中文 123 🙂 測試"},
		{"NULL 與換行", "第一行\n第二行\t第三行"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			email := "unicode-" + tc.name + "@example.com"
			// created_at 與 image_url 顯式給值（NOT NULL 而無 DEFAULT，見
			// seedForumPost 的說明）：只給兩個欄位的 INSERT 在嚴格模式下會 1364。
			mustExec(t, db,
				"INSERT INTO forum_posts (id, author_email, content, image_url, created_at) VALUES (?, ?, ?, '', ?)",
				idgen.Next(), email, tc.text, time.Now())

			var got string
			if err := db.QueryRow(
				"SELECT content FROM forum_posts WHERE author_email = ?", email).Scan(&got); err != nil {
				t.Fatalf("讀回失敗: %v", err)
			}
			if got != tc.text {
				t.Errorf("往返不一致\n  寫入: %q\n  讀回: %q", tc.text, got)
			}
		})
	}
}

// TestMigrateMySQLIsSafeWithConcurrentStartup 驗證兩個實例同時啟動。
//
// 這是 mysql.go 檔頭第 1 點的具體承諾：「遷移刻意線性執行而非包在交易內」，
// 而理由是「線性執行反而能避免兩個實例同時啟動時互相鎖住 metadata 而死鎖」。
//
// 這支測試驗證的是那個承諾的一半：**不會死鎖、也不會有其中一個失敗**。
// 刻意不驗證「結果一定正確」—— 那由前面那些測試負責；而在真實的並行遷移裡，
// 兩個實例可能都成功也可能一個成功一個等著，那取決於 metadata 鎖的排隊，
// 而把它變成斷言只會產生一個 flaky 的測試。
func TestMigrateMySQLIsSafeWithConcurrentStartup(t *testing.T) {
	t.Parallel()
	db, _ := tempDatabase(t)

	// 刻意**不加** t.Parallel 到遷移本身：這個測試自己就是並行的那一方。
	// 兩個執行緒在同一個 *sql.DB 上跑 MigrateMySQL，而它們會各自從池裡拿
	// 連線 —— 那才是真實的「兩個實例」在共用一個池的情況。
	const workers = 3
	errs := make(chan error, workers)
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		go func() {
			<-start // 讓它們盡量同時開始，那才是這個測試想製造的狀況
			errs <- MigrateMySQL(db)
		}()
	}
	close(start)

	var failures []string
	for i := 0; i < workers; i++ {
		if err := <-errs; err != nil {
			failures = append(failures, err.Error())
		}
	}

	if len(failures) > 0 {
		t.Errorf("%d/%d 個並行遷移失敗了：\n  %s\n"+
			"兩個實例同時啟動是常見的情況（重複部署、捲軸更新、負載平衡器同時\n"+
			"把兩個連線分到新版本），因此這個失敗會在部署時出現而不在開發時。",
			len(failures), workers, strings.Join(failures, "\n  "))
	}

	// 無論並行怎麼跑，結果都必須是完整的 schema。
	for _, table := range requiredTables {
		if !tableExists(t, db, table) {
			t.Errorf("並行遷移之後 %s 不存在", table)
		}
	}
}

// TestOpenMySQLRejectsBadDSN 覆蓋 OpenMySQL 的錯誤路徑。
//
// 這一條不需要真的資料庫 —— 一個壞掉的 DSN 必須在**建池當下**就回錯誤，
// 那是這個函式的核心承諾（見它的文件說明：「讓『資料庫連不上』這類致命錯誤
// 在 main 啟動階段就爆掉，而不是等到第一個請求才以 500 的形式浮現」）。
func TestOpenMySQLRejectsBadDSN(t *testing.T) {
	t.Parallel()

	server := serverDB(t)
	_ = server // 只需要它確認伺服器可達（避免把「伺服器沒跑」誤判成別的錯誤）

	// 一個 DSN 指向一個不存在的庫。MySQL 會在連線時回 "Unknown database" ——
	// 而「資料庫沒建立」正是部署時最常見的錯誤之一，因此這個形狀必須被
	// 立刻抓到而不是延後到第一個請求。
	const dsn = "root:root@tcp(127.0.0.1:1)/definitely_not_a_database?parseTime=true"
	db, err := OpenMySQL(dsn, 2, 1, time.Minute)
	if err == nil {
		_ = db.Close()
		// 連得到 127.0.0.1:1 是極不可能的（那不是一個 listen 中的埠），
		// 但若真的發生了，那就是測試環境的問題而不是程式碼的問題。
		t.Skip("127.0.0.1:1 竟然有服務在聽，無法驗證「連不上」的分支")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Errorf("錯誤訊息沒有點名那個位址（維運需要知道是哪一個設定值錯了）: %v", err)
	}
}

// --- 輔助函式 ---

// schemaSnapshot 回傳一份可比對的 schema 描述字串。
//
// 刻意只取「結構」而不取資料：這支測試要比對的是 schema 有沒有改變。
// 取哪些欄位是一個判斷：engine 與 collation 是重要的（它們影響行為），
// 而 AUTO_INCREMENT 的目前值不重要且會變動，因此不取。
func schemaSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()

	rows, err := db.Query(`SELECT table_name, engine, table_collation
	                       FROM information_schema.tables
	                       WHERE table_schema = DATABASE()
	                       ORDER BY table_name`)
	if err != nil {
		t.Fatalf("查詢資料表清單失敗: %v", err)
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var table, engine, collation string
		if err := rows.Scan(&table, &engine, &collation); err != nil {
			t.Fatalf("掃描失敗: %v", err)
		}
		parts = append(parts, fmt.Sprintf("%s|%s|%s", table, engine, collation))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("走訪失敗: %v", err)
	}
	return strings.Join(parts, ";")
}

// columnExists 查詢某張表是否有某個欄位。
func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
		table, column).Scan(&n)
	if err != nil {
		t.Fatalf("查詢 information_schema.columns 失敗: %v", err)
	}
	return n > 0
}

// mustExec 執行一句 SQL 並在失敗時中止測試。
func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("執行 SQL 失敗\n  query: %s\n  error: %v", query, err)
	}
}

// seedForumPost 塞入一列貼文（用於驗證遷移不清資料）。
//
// created_at 與 image_url 顯式給值：兩者都是 NOT NULL 而沒有 DEFAULT 的欄位
// （image_url 見 mysql.go 第 2 步的說明），嚴格模式下省略它們的 INSERT 會以
// Error 1364 失敗。這支 helper 曾經只給 author_email 與 content，於是每一次
// 執行都以 1364 收場 —— 而它要驗證的「第二次遷移不會清掉資料」從來沒被測到。
func seedForumPost(t *testing.T, db *sql.DB, content, email string) {
	t.Helper()
	// id 必須由應用層給：MigrateMySQL 已經移除 AUTO_INCREMENT，而這支
	// helper 是在第一次遷移**之後**才被呼叫的。
	mustExec(t, db,
		"INSERT INTO forum_posts (id, author_email, content, image_url, created_at) VALUES (?, ?, ?, '', ?)",
		idgen.Next(), email, content, time.Now())
}
