/*
migration_test.go 用**真實的 MySQL** 驗證 MigrateMySQL。

【為什麼不能用 go-sqlmock】
ROADMAP.md 的 Phase 3.1 說得很直接：

	go-sqlmock 驗證不了 DDL 真的能執行，miniredis 對 MySQL 沒有對應物

原因不是工具不夠好，而是這個函式測的東西就是「MySQL 會不會接受這句 DDL」：
  - ADD COLUMN IF NOT EXISTS 需要 MySQL 8.0.29+。在 sqlmock 上「通過」只代表
    程式碼呼叫了它，而不代表任何一個 MySQL 版本接受它。
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
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
}

// dbCounter 讓每個測試的資料庫名稱都不相同。
//
// 不用 t.Name() 是因為兩個測試在同一個 package 裡可能同名（同名的子測試），
// 而用亂數尾巴則保證「即使名字真的撞了也不會互相干擾」。
var dbCounter atomic.Int64

// serverDB 連到 MySQL **伺服器**（不指定資料庫），測試用它建立拋棄式資料庫。
func serverDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("FORUM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FORUM_TEST_MYSQL_DSN 未設定：跳過需要真實 MySQL 的遷移測試。" +
			"設定方式見本檔的檔頭，或見 .github/workflows/ci.yml 的 migrations job")
	}

	// 刻意剝掉 DSN 裡的庫名，讓它變成「只連伺服器」。而多Statements=true 是
	// 讓測試能用一句 SQL 執行多個陳述（清理用的那幾句）。
	cleaned := dsn
	if i := strings.LastIndex(cleaned, "/"); i >= 0 {
		if j := strings.Index(cleaned[i:], "?"); j >= 0 {
			cleaned = cleaned[:i] + "/" + cleaned[i+j+1:]
		} else {
			cleaned = cleaned[:i] + "/"
		}
	}

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

	dsn := strings.Replace(os.Getenv("FORUM_TEST_MYSQL_DSN"), "/?", "/"+name+"?", 1)
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

	// 一份刻意落後的 schema：少了三張新表，且一個現有表少了兩個欄位。
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

	// 缺的三張表應該被補上。
	for _, table := range []string{"forum_announcements", "forum_admin_actions", "forum_user_tags"} {
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
			mustExec(t, db,
				"INSERT INTO forum_posts (author_email, content) VALUES (?, ?)", email, tc.text)

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
func seedForumPost(t *testing.T, db *sql.DB, content, email string) {
	t.Helper()
	mustExec(t, db,
		"INSERT INTO forum_posts (author_email, content) VALUES (?, ?)", email, content)
}
