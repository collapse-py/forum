/*
Package data 負責 MySQL 連線的建立與 schema 遷移。

本檔案只提供兩個對外函式：

  - OpenMySQL：包裝 database/sql 的連線池設定，並在建池當下立即 Ping 一次，
    讓「資料庫連不上」這類致命錯誤在 main 啟動階段就爆掉，而不是等到第一個請求
    才以 500 的形式浮現。
  - MigrateMySQL：以「線性執行、冪等」的方式把 schema 補齊到程式碼預期的最新狀態。

資料表清單（schema 全貌）

	forum_posts                  討論區文章主檔：id / author_email / content / image_url / created_at / updated_at / pinned / deleted_at
	forum_post_likes             按讚表，以 (post_id, author_email) 複合主鍵表示「一人一文一讚」
	forum_post_comments          文章留言：id / post_id / parent_id / depth / author_email / content / created_at
	                           （parent_id 指回應的另一則留言，depth 是層級 1..3，見第 31 步）
	forum_reports                檢舉工單（可指向文章或留言），含處理狀態與覆核者
	forum_profiles               使用者公開資料：author_email 為主鍵、public_key 為 email 的 SHA-256（舊形態）、public_key_hmac 為目前派生的識別碼（HMAC，未設定秘密時為空）、avatar_url 只存檔名
	forum_users                  使用者帳號主檔與停權狀態（status = ACTIVE / SUSPENDED）
	forum_user_tags              標籤字典
	forum_user_tag_assignments   使用者與標籤的多對多關聯
	forum_follows                追蹤關聯，以 (follower_email, target_email) 複合主鍵表示「一人追一人一次」
	forum_post_bookmarks         收藏關聯，以 (post_id, author_email) 複合主鍵表示「一人一文一收藏」
	forum_media                  上傳檔案的所有權紀錄，以檔名為主鍵（檔名是檔案伺服器上的鍵）
	                             （第 33 步；「刪除自己的圖片」與「換圖」的授權依據）
	forum_media_deletions        待刪除的圖片檔名佇列（第 37 步；與貼文硬刪同一個交易寫入，失敗可重試）
	forum_request_metrics        分鐘級請求統計（監控頁的持久化來源；純衍生資料，可整表刪除）
	forum_daily_stats            日別新增量彙總（內容趨勢頁的讀取來源；純衍生資料，可整表重建）
	forum_admin_actions           管理員操作稽核紀錄（欄位級 diff；只能由時間清理，不可透過 API 刪除）
	forum_announcements           站內公告（同時只有一列 active；expires_at 可空＝永不自動過期）
	forum_notifications           站內通知（留言／回覆／追蹤；read_at 可空＝未讀）

關鍵設計決策

 1. 遷移刻意「線性執行」而非包在交易（Transaction）內。MySQL 的 DDL 本身具備隱含提交
    （implicit commit）語意，CREATE TABLE / ALTER TABLE 無法被 ROLLBACK，一旦包進交易只會
    製造「以為有原子性、實際沒有」的假象。加上 DDL 在 MySQL 中會取得隱式鎖，線性執行反而能
    避免兩個實例同時啟動時互相鎖住 metadata 而死鎖。代價是遷移中途失敗會留下半套 schema，
    但因為每一步都是冪等的，下一次啟動會自動補齊剩餘步驟。
 2. 冪等性有兩個來源，分工是「伺服器支援的用語法、不支援的用探測」：
    CREATE TABLE IF NOT EXISTS 與 INSERT IGNORE 直接用語法；**補欄位與補索引**一律走
    addColumnIfNotExists / addIndexIfNotExists 的「information_schema 探測 + 建立」，
    因為 MySQL 沒有 ADD COLUMN IF NOT EXISTS（那是 MariaDB 專屬語法，MySQL 8.x 對它
    回 ERROR 1064）、沒有 ADD INDEX IF NOT EXISTS，也沒有 DROP COLUMN IF EXISTS。
    探測與建立之間的窗口（兩個實例同時啟動時會被命中）由容忍 Error 1060（欄位）
    與 1061（索引）補上 —— 見 TestMigrateMySQLIsSafeWithConcurrentStartup。
 3. 全部資料表統一 ENGINE=InnoDB（交易與外鍵語意）與 DEFAULT CHARSET=utf8mb4。utf8mb4 是
    完整 Unicode（含 emoji 與 4-byte supplementary 平面），論壇內文屬使用者輸入，用舊的 utf8
    （MySQL 的 3-byte utf8mb3）會在寫入時直接報錯。
 4. 已知風險：schema 版本漂移。本檔案沒有版本號或 golang-migrate 這類版本表，因此無法偵測
    「某個欄位被人工改過」或「欄位型別與預期不符」；CREATE TABLE IF NOT EXISTS 只會在表
    不存在時建立，已存在的表其欄位差異必須靠後面逐一補的 ALTER 與人工介入處理。
*/
package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"forum/forum/idgen"

	// 這裡不再用底線匯入：除了註冊驅動，還要用 mysql.MySQLError 判斷
	// 「索引已存在」（Error 1061）—— 見 indexAlreadyExists。
	mysql "github.com/go-sql-driver/mysql"
)

// indexAlreadyExists 判斷一個 ALTER TABLE … ADD INDEX 的錯誤是不是
// 「索引已經存在」（MySQL Error 1061）。
//
// 為什麼需要它：MySQL 沒有 ADD INDEX IF NOT EXISTS（唯一索引也沒有），因此本檔
// 所有索引建立都是「information_schema 探測 + 建立」的兩步。兩步之間有窗口，
// 而兩個實例同時啟動（重複部署、滾動更新、負載平衡器把兩個連線分到新版本）時
// 那個窗口會被命中：兩邊都探到「不存在」、兩邊都建立、後者拿 1061 並讓整個遷移
// 失敗 —— 服務起不來，而這正是 mysql.go 檔頭第 1 點與
// TestMigrateMySQLIsSafeWithConcurrentStartup 釘住的承諾。
//
// 「已經存在」是探測想得到的答案，只是別人先建好了，因此它必須走成功而不是失敗。
//
// 反面：Error 1062（唯一索引撞到重複資料）仍然必須失敗並中止啟動。見第 17 步
// 的說明 —— 靜默留下重複暱稱比啟動失敗更糟，而那正是這個索引存在的理由。
func indexAlreadyExists(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1061
}

// addIndexIfNotExists 以「探測 + 建立 + 容忍併發 1061」補上一個索引。
//
// 取代各步驟手寫的 information_schema 探測：探測的兩個步驟在兩處以上之後就會
// 開始出現「其中一處忘了容忍 1061」這種只會在並發部署時出現的差異。冪等性
// 由這一個函式負責，呼叫端只描述「我要哪個索引」。
func addIndexIfNotExists(db *sql.DB, table, index, columns string) error {
	var count int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?
	`, table, index).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	// 1061 = 另一個實例在探測與建立之間先建好了，那正是我們要的結果。
	if _, err := db.Exec("ALTER TABLE " + table + " ADD INDEX " + index + " (" + columns + ")"); err != nil {
		if indexAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

// addUniqueIndexIfNotExists 是 addIndexIfNotExists 的唯一索引版。
//
// MySQL 的唯一索引同樣沒有 IF NOT EXISTS，而它與非唯一索引的差別只在失敗的
// 理由：1061（已存在）容忍，1062（既有資料重複）必須讓啟動失敗（第 17 步）。
func addUniqueIndexIfNotExists(db *sql.DB, table, index, columns string) error {
	var count int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?
	`, table, index).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if _, err := db.Exec("ALTER TABLE " + table + " ADD UNIQUE INDEX " + index + " (" + columns + ")"); err != nil {
		if indexAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

// columnAlreadyExists 判斷一個 ALTER TABLE … ADD COLUMN 的錯誤是不是
// 「欄位已經存在」的 Error 1060（MySQL 與 MariaDB 一樣）。
//
// 為什麼需要它：`ADD COLUMN IF NOT EXISTS` 是 **MariaDB 專屬**語法。MySQL 8.x 對它
// 一律回 ERROR 1064（本機實測 8.0.44、CI 的 8.0.29 皆然），而這個專案的目標是
// MySQL —— 2026-10-23 之前的遷移正是踩在這裡：開發用資料庫是 MariaDB 11.8（語法
// 被接受，因此本機開發從來沒事），CI 的 mysql:8.0.29 上整批遷移測試全紅。兩種
// 伺服器都能接受的寫法只有「探測 + 建立」。
//
// 1060 的語意與索引那邊的 1061 相同：探測與建立之間的窗口被另一個實例填滿，
// 「對方先補好了」正是我們要的結果，因此它必須走成功而不是失敗。
//
// 反面（刻意不處理）：欄位已存在但**型別**與預期不符時不會被發現。那是檔頭第 4 點
// 的 schema 漂移風險，不在此處的範圍 —— 在這裡改型別是 DDL、失去回滾機會，而且
// 沒有訊號能區分「人工改過」與「舊版本本來就長這樣」。
func columnAlreadyExists(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1060
}

// addColumnIfNotExists 以「探測 + 建立 + 容忍併發 1060」補上一個欄位。
//
// 是 addIndexIfNotExists 的欄位版，理由也相同：冪等性由這一個函式負責，呼叫端只
// 描述「我要哪個欄位、它長什麼樣」。手寫探測的代價不是行數，而是「其中一處忘了
// 容忍 1060」這種只會在兩個實例同時啟動時現身的差異。
//
// definition 是欄位定義本身（"TEXT NOT NULL"、"DATETIME NULL AFTER pinned"），
// 因此 AFTER 這類位置子句也由呼叫端表達。table / column 是本檔的字面值，與索引的
// DDL 一樣以字串組接（DDL 無法參數化）；探測查詢刻意也把它們寫成字面值而非
// 綁定參數 —— 這樣「哪個欄位的探測掛了」會直接出現在錯誤輸出與測試規則裡。
func addColumnIfNotExists(db *sql.DB, table, column, definition string) error {
	var count int
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s' AND COLUMN_NAME = '%s'
	`, table, column)).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if _, err := db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition); err != nil {
		if columnAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

// legacyForumImageName 把「任何一種本站檔案的寫法」收斂成純檔名：先丟掉 `?`
// 之後的查詢字串（舊的完整網址帶 `?token=…`），再取最後一段路徑
// （`/files/x.png` 或完整網址），最後丟掉舊版遷移黏在尾段的 `"]`（舊版遷移先包
// JSON 再切路徑留下的 `x.png"]` 損壞值；只丟 `]` 會剩下一個引號）。語意與
// httpapi.forumImageFileName 及 purge.imageFileName 相同；三份刻意各寫各的 ——
// data 是 schema 層，不該依賴 handler 層的工具。
//
// 它與 SQL 那邊的四步 SUBSTRING_INDEX（httpapi.forumImageLegacyPostsExpr）逐字
// 對應，因此「遷移後存的值」與「查詢時收斂出的值」是同一個規則，兩邊不會不一致。
// 合法檔名的白名單是 [A-Za-z0-9._-]，`"` 與 `]` 不在其中，因此看到第一個就切掉
// 不會誤傷任何合法檔名。
func legacyForumImageName(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if i := strings.Index(value, "?"); i >= 0 {
		value = value[:i]
	}
	if i := strings.LastIndex(value, "/"); i >= 0 {
		value = value[i+1:]
	}
	if i := strings.IndexAny(value, `"]`); i >= 0 {
		value = value[:i]
	}
	return value
}

// forumImageValueToCanonical 回傳新契約的寫法：JSON 陣列字串（空值為 `[]`）。
//
// 三種輸入都處理：
//   - 已是 JSON 陣列：逐元素收斂（舊版可能把完整網址包進陣列裡）。
//   - 單一值（純檔名、/files/x.png、完整網址、`x.png"]` 損壞值）：收斂成一個檔名，
//     包成單元素陣列。
//   - 收斂不出任何東西（空字串、空陣列、元素全收斂不出來）：`[]`。
//
// 已知限制與第 18 步相同：這個收斂只切字串、不驗證主機。第三方圖片網址會得到
// 一個指向自己檔案伺服器、但實際不存在的檔名 —— 讀取時回空字串（前端不顯示縮圖），
// 而 CSP 的 img-src 本來就只放行本站的檔案伺服器。
func forumImageValueToCanonical(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "[]"
	}
	if strings.HasPrefix(value, "[") {
		var items []string
		if err := json.Unmarshal([]byte(value), &items); err == nil {
			names := make([]string, 0, len(items))
			for _, item := range items {
				if name := legacyForumImageName(item); name != "" {
					names = append(names, name)
				}
			}
			if len(names) == 0 {
				return "[]"
			}
			// json.Marshal 對 []string 不會失敗；萬一失敗就保留原值，
			// 「不丟資料」比「變成空陣列」重要。
			encoded, err := json.Marshal(names)
			if err != nil {
				return value
			}
			return string(encoded)
		}
		// 不是合法 JSON（例如 `[broken`）：當單值處理，下面的包法會得到合法 JSON。
	}
	name := legacyForumImageName(value)
	if name == "" {
		return "[]"
	}
	encoded, err := json.Marshal([]string{name})
	if err != nil {
		return value
	}
	return string(encoded)
}

// normalizeForumImageValues 把 forum_posts.image_url 裡「非新契約」的值收斂成
// 新契約（見 forumImageValueToCanonical），並修掉舊版遷移順序留下的損壞值。
//
// 為什麼用 Go 而不是 SQL：舊值有四種形狀，而「JSON 陣列裡的元素也要收斂」在
// SQL 裡需要 JSON_TABLE —— 那對**不合法**的 JSON 會直接報錯（ER_INVALID_JSON）。
// MigrateMySQL 是同步跑的，一條報錯的 UPDATE 會讓伺服器啟動失敗，而這裡只是在
// 修一個資料格式的問題。Go 逐列收斂，非法 JSON 走單值那條路。
//
// 冪等：新契約的值（`["a.png"]`）不含 `/` 也不含 `?`，且以 `[` 開頭，因此下面的
// 探測條件不會命中它們，第二次啟動不動任何一列。
const forumImageLegacyRowsWhere = `image_url LIKE '%/%' OR image_url LIKE '%?%' OR (LEFT(TRIM(image_url), 1) <> '[' AND TRIM(image_url) <> '')`

func normalizeForumImageValues(db *sql.DB) error {
	var legacy int
	if err := db.QueryRow(`SELECT COUNT(*) FROM forum_posts WHERE ` + forumImageLegacyRowsWhere).Scan(&legacy); err != nil {
		return err
	}
	if legacy == 0 {
		return nil
	}

	rows, err := db.Query(`SELECT id, image_url FROM forum_posts WHERE ` + forumImageLegacyRowsWhere)
	if err != nil {
		return err
	}
	defer rows.Close()

	type canonicalUpdate struct {
		id       int64
		imageURL string
	}
	updates := make([]canonicalUpdate, 0, legacy)
	for rows.Next() {
		var id int64
		var raw string
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		if canonical := forumImageValueToCanonical(raw); canonical != raw {
			updates = append(updates, canonicalUpdate{id: id, imageURL: canonical})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, u := range updates {
		if _, err := db.Exec(`UPDATE forum_posts SET image_url = ? WHERE id = ?`, u.imageURL, u.id); err != nil {
			return err
		}
	}
	return nil
}

// OpenMySQL 依傳入的 DSN 建立 *sql.DB 連線池，並設定連線池的容量與存活期限。
//
// 參數：
//   - dsn：go-sql-driver/mysql 的 DSN 字串，須自行攜帶 parseTime=true，
//     否則 DATETIME 欄位會以 []byte 回傳，無法直接掃描進 time.Time。
//   - maxOpenConns：連線池同時可開啟的上限，超過時 database/sql 會阻塞等待，
//     這是防止 MySQL 的 max_connections 被單一服務打爆的主要防線。
//   - maxIdleConns：閒置連線上限；值太小會讓高併發時頻繁重建連線（每次都要重新握手），
//     通常設定為與 maxOpenConns 相當以利重用。
//   - connMaxLifetime：單一連線的最長存活時間。MySQL 預設 wait_timeout 會主動關閉舊連線，
//     若連線池不主動回收，會出現「第一次請求成功、之後固定報 broken pipe / EOF」的現象，
//     設定此值讓連線在失效前就被平滑替換。
//
// 回傳值：建立成功時回傳 *sql.DB；sql.Open 或 Ping 失敗時回傳 nil 與錯誤，
// 且已建立的連線會先被 Close，避免洩漏。
//
// 副作用：sql.Open 本身是延遲連線的（lazy），此處的 db.Ping() 才是第一個真正連線的動作。
func OpenMySQL(dsn string, maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) (*sql.DB, error) {
	// 此處不會真正連線，只註冊驅動並建立空連線池；驅動名稱 "mysql" 需與上面的底線匯入對應。
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// 主動 Ping 一次作為「啟動自檢」：把 DSN 寫錯、密碼不對、資料庫不存在等問題
	// 在 main 啟動時就以 Fatal 呈現，避免第一個使用者請求才收到內部錯誤。
	if err := db.Ping(); err != nil {
		// 連線池已建立但不可用，必須主動關閉，否則底層 socket 會被留到行程結束。
		db.Close()
		return nil, err
	}

	return db, nil
}

// MigrateMySQL 以冪等方式把資料表與欄位補齊到程式碼預期的 schema 版本。
//
// 呼叫時機：main 啟動流程中、開發 HTTP server 之前（main.go 的 logger.Fatalf([DB] ...) 分支）。
// 因為所有 handler 都假設表與欄位已存在，遷移必須先於服務對外提供請求。
//
// 執行方式：逐條陳述式線性執行，任一步失敗即立刻回傳錯誤且不再往下跑（fail fast）。
// 由於 DDL 在 MySQL 具隱含提交語意，無法回滾，因此不做交易包裝；呼叫端（main）收到
// 錯誤後直接 Fatal 結束行程，由下一次啟動重跑已冪等的遷移。
//
// 冪等性來源：CREATE TABLE IF NOT EXISTS、INSERT IGNORE，以及
// addColumnIfNotExists / addIndexIfNotExists 對「欄位、舊版欄位、索引是否存在」的
// information_schema 探測（MySQL 沒有 ADD COLUMN IF NOT EXISTS —— 那是 MariaDB 專屬）。
//
// 錯誤條件：db 為 nil、回傳錯誤的 SQL 語法錯誤、權限不足（缺少 ALTER 權限）、
// 或 DDL 對應的資料表已存在但結構衝突（例如既有索引名稱重複）。
//
// 副作用：會在目標資料庫建立資料表、欄位與索引，並於論壇資料中新增的舊資料回填列。
func MigrateMySQL(db *sql.DB) error {
	// 防呆：nil 會在後續 db.Exec 時 panic 成難以定位的 nil pointer dereference，
	// 在此提前轉換成可被呼叫端正常處理的 error。
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	// 1) 文章主檔。IF NOT EXISTS 讓全新環境與既有環境走同一條程式路徑。
	//    欄位語意：
	//      id            – 主鍵，由應用層的雪花演算法產生（見 forum/idgen）
	//      author_email  – 發文者 email，作為與 forum_users / forum_profiles 的關聯鍵；
	//                      長度 255 是為了容納較長的 email 字串
	//      content       – 純文字內容，TEXT 而非 VARCHAR，因內文無固定長度上限
	//      created_at    – 由應用層寫入 time.Now()（非資料庫 NOW()），時區為程式主機的本地時區
	//    索引理由：idx_forum_posts_created_at 服務於「依 created_at DESC 排序 + LIMIT/OFFSET
	//    分頁」這條主查詢（listForumPosts / 管理員文章列表），避免全表排序後再切頁。
	//    注意排序實際為 created_at DESC, id DESC，InnoDB 隱含的主鍵順序可協助 id 的穩定排序。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_posts (
			id BIGINT NOT NULL PRIMARY KEY,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			INDEX idx_forum_posts_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 2) image_url 為後來新增的需求（先發文、圖片由獨立的檔案伺服器保管），
	//    因此以 addColumnIfNotExists 追加，而非改建表。
	//    選 TEXT 而非 VARCHAR：沿用當初「存完整 URL、長度不可預期」的型別，
	//    即使現在只存檔名也不改——改欄位型別是 DDL、失去回滾機會，
	//    對一個上限 128 字元的值並不划算。
	//    儲存契約：只放檔名（如 "9f8c1a.png"），完整網址由應用層以
	//    FILES_SERVER_PUBLIC_URL 組成；舊資料的轉換見第 18 步。
	//    宣告 NOT NULL 卻未給 DEFAULT，代表所有既有列在 MySQL 嚴格模式下會被填入空字串
	//    （DDL 會以列層級的隱式預設值補齊），因此 createForumPost 仍需顯式傳入值。
	if err := addColumnIfNotExists(db, "forum_posts", "image_url", "TEXT NOT NULL"); err != nil {
		return err
	}
	// 2b) 多圖改版：把舊的單一檔名/網址轉成 JSON 陣列字串，讓 image_url 欄位
	//    統一以 JSON 陣列表示（空值為 []）。冪等：已為 JSON 陣列的列不會再被轉。
	//
	// 2b) 多圖改版：把舊的單一檔名/網址轉成 JSON 陣列字串，讓 image_url 欄位
	//    統一以 JSON 陣列表示（空值為 []）。冪等：已為 JSON 陣列的列不會再被轉。
	//
	//    順序是這條遷移的關鍵：**先收斂成純檔名，再包成 JSON**（見下方
	//    normalizeForumImageValues）。先包再切（第 18 步對整條字串做 SUBSTRING_INDEX）
	//    會把 `["http://…/files/x.png?token=t"]` 一路切成 `x.png"]`，而那個值不再含
	//    `/`，因此第 18 步的冪等條件永遠不會再動它。第 18 步現在會丟掉尾段的 `"]`
	//    （與這裡的 Go 收斂同一個規則），但**多圖**的舊值被切之後仍只剩最後一個
	//    檔名，前面的圖直接消失 —— 那才是 2b 必須先收斂的理由。
	if err := normalizeForumImageValues(db); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE forum_posts SET image_url = '[]' WHERE TRIM(image_url) = ''`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE forum_posts SET image_url = CONCAT('["', image_url, '"]')
		WHERE TRIM(image_url) <> '' AND LEFT(TRIM(image_url), 1) <> '['`); err != nil {
		return err
	}

	// 3) 按讚表。複合主鍵 (post_id, author_email) 同時扮演「一人對同一篇文章只能按一次讚」
	//    的業務規則與「查詢某人的按讚清單」所需的索引，因此不需要額外建立 (post_id, author_email) 索引。
	//    idx_forum_post_likes_post_id 是多餘但必要的：文章列表頁會對每篇文章做
	//    SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ? 的相關子查詢，
	//    該索引讓每次計數不必掃全表。
	//    注意：MySQL 會自動以複合主鍵的最左前綴 (post_id) 產生索引，故此 INDEX 嚴格說是
	//    重複宣告；此處保留是為了讓意圖明確並兼顧舊版資料表（其主鍵可能不同）。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_post_likes (
			post_id BIGINT NOT NULL,
			author_email VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (post_id, author_email),
			INDEX idx_forum_post_likes_post_id (post_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 4) 留言表。post_id 沒有宣告外鍵：留言與文章的關聯由應用層維護（handler 會先
	//    SELECT COUNT(*) 確認文章存在），且不設 ON DELETE CASCADE，刪文章時由 handler
	//    決定是否連帶清除，避免誤刪。
	//    idx_forum_post_comments_post_id 服務於「取某篇文章的留言」與留言數統計子查詢；
	//    idx_forum_post_comments_created_at 則服務於管理員以時間排序瀏覽留言。
	//    parent_id / depth（留言的層級）在第 31 步補上：彼時所有既有留言都是第一層，
	//    而那個預設值正是它們在沒有回覆功能之前的樣子。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_post_comments (
			id BIGINT NOT NULL PRIMARY KEY,
			post_id BIGINT NOT NULL,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			INDEX idx_forum_post_comments_post_id (post_id),
			INDEX idx_forum_post_comments_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 5) 檢舉工單。target_type + target_id 是「多型別參照」（polymorphic reference），
	//    MySQL 無法對多型別外鍵下 FK 約束，所以只能靠應用層驗證目標存在。
	//      status 允許值：PENDING / RESOLVED / REJECTED，預設 PENDING
	//      reviewed_at 允許 NULL（尚未覆核）；reviewed_by 預設空字串而非 NULL，
	//      讓「未覆核」有兩種可區分的表示，掃碼時較不易誤判。
	//    uq_forum_reports_reporter_target：同一檢舉者對同一目標只能檢舉一次，
	//    讓 handleCreateForumReport 可以把 MySQL error 1062 直接翻譯成
	//    「你已檢舉過此內容」的 409 回應，重複送出不會產生多筆工單。
	//    idx_forum_reports_status_created：(status, created_at) 的複合索引用於管理端
	//    「依狀態過濾並依 created_at DESC 排序」的檢舉列表查詢。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_reports (
			id BIGINT NOT NULL PRIMARY KEY,
			target_type VARCHAR(20) NOT NULL,
			target_id BIGINT NOT NULL,
			reporter_email VARCHAR(255) NOT NULL,
			reason VARCHAR(500) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			created_at DATETIME NOT NULL,
			reviewed_at DATETIME NULL,
			reviewed_by VARCHAR(255) NOT NULL DEFAULT '',
			UNIQUE KEY uq_forum_reports_reporter_target (reporter_email, target_type, target_id),
			INDEX idx_forum_reports_status_created (status, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 6) 舊版資料相容性修補（author_email 補欄位）。
	//    為何需要：forum_post_likes 在更早的版本可能是「一列一位使用者對一篇文章的讚」，
	//    欄位名稱為 user_email，且完全缺少 author_email。步驟 3 的 CREATE TABLE IF NOT EXISTS
	//    對「已存在」的表不會做任何事，所以舊表必須靠這裡補齊欄位才能讓新版程式正常運作。
	//    探測技巧：information_schema.COLUMNS 是 MySQL 的中繼資料表，DATABASE() 會展開成
	//    目前連線的 schema 名稱，用 TABLE_SCHEMA = DATABASE() 即可把查詢限制在目標資料庫
	//    （避免誤查到同名的其他資料庫），再用 TABLE_NAME 與 COLUMN_NAME 過濾。
	//    這是必要的，因為 MySQL 沒有「ADD COLUMN IF NOT EXISTS」可用於比對既有欄位型別的情境
	//    （此處的 author_email 為 NOT NULL，必須帶 DEFAULT '' 才能在有資料的表上成功）。
	// 探測與容忍 1060 由 addColumnIfNotExists 負責（它送出的 SQL 與過去手寫探測的
	// 版本逐字相同），這裡只需要描述欄位定義本身。
	if err := addColumnIfNotExists(db, "forum_post_likes", "author_email", "VARCHAR(255) NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	// 7) 舊欄位遷移：user_email → author_email。
	//    為何要改名：全站統一以 author_email 表示「內容作者」，改名的目的是讓欄位語意一致，
	//    避免 user_email（帳號主體）與 author_email（內容歸屬）在閱讀 SQL 時被誤認為同義。
	//    先 DROP 再 CHANGE 而非直接 CHANGE：CHANGE COLUMN 若新名稱已被占用會失敗，
	//    先刪掉上一步可能補上的 author_email 可保證這段遷移在任何起點下都只執行一次，
	//    屬於自我冪等（idempotent）的寫法。
	var legacyUserEmailColumnCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_post_likes' AND COLUMN_NAME = 'user_email'
	`).Scan(&legacyUserEmailColumnCount); err != nil {
		return err
	}
	if legacyUserEmailColumnCount > 0 {
		if _, err := db.Exec(`ALTER TABLE forum_post_likes DROP COLUMN author_email`); err != nil {
			return err
		}
		// CHANGE COLUMN 同時重新命名並套用型別約束；不寫 DEFAULT 是因為既有列的值
		// 會被保留，不需要也不應該改動資料內容。
		if _, err := db.Exec(`
			ALTER TABLE forum_post_likes
			CHANGE COLUMN user_email author_email VARCHAR(255) NOT NULL
		`); err != nil {
			return err
		}
	}

	// 8) 舊欄位遷移：移除 forum_posts.title。
	//    為何要 drop：產品決策改成「純文字貼文」，title 已無對應的輸入欄位與驗證邏輯，
	//    留在 schema 只會誤導後續開發者以為該欄位仍有效。
	//    為什麼需要探測：MySQL 沒有 DROP COLUMN IF EXISTS，若欄位已不存在會整句報錯
	//    而讓遷移中斷；先以 information_schema 確認存在才執行，確保重複啟動安全。
	//    注意：DROP COLUMN 會永久遺失資料，無法回滾（DDL 隱含提交）。
	var forumTitleColumnCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_posts' AND COLUMN_NAME = 'title'
	`).Scan(&forumTitleColumnCount); err != nil {
		return err
	}
	if forumTitleColumnCount > 0 {
		if _, err := db.Exec(`ALTER TABLE forum_posts DROP COLUMN title`); err != nil {
			return err
		}
	}

	// 9) 個人公開資料。author_email 直接當主鍵（而非另設 id + UNIQUE），
	//    因為公開資料與使用者是一對一，email 即可作為天然鍵，省去代理鍵與額外查找。
	//      public_key  – email 的 SHA-256（64 個十六進位字元）。前端以這個雜湊值互相連結，
	//                   讓訪客查詢他人公開頁面（handleForumPublicProfile）時不必在 URL
	//                   暴露原始 email；應用層的 publicForumKey 與此處的 SHA2 必須演算法一致。
	//      nickname    – 顯示名稱，欄位長度 30 與應用層的驗證一致；
	//                   UNIQUE 會讓重複暱稱在 INSERT 時觸發 error 1062，
	//                   handler 再翻譯成「暱稱已被使用」的 409。
	//      bio         – 自我介紹，TEXT（上限 500 字由應用層驗證）。
	//      updated_at  – 由應用層寫入；管理端使用者列表以 u.updated_at 排序。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_profiles (
			author_email VARCHAR(255) PRIMARY KEY,
			public_key VARCHAR(64) NOT NULL DEFAULT '',
			nickname VARCHAR(30) NOT NULL,
			bio TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 10) 帳號主檔。獨立的 forum_users 是為了讓「停權」這類帳號層級狀態能跨資料表統一生效：
	//     requireLogin 每次都會查 forum_users.status，被停權的使用者連讀取既有文章都會被擋下，
	//     不會因為放寬單一表而漏掉。status 目前僅使用 ACTIVE 與 SUSPENDED 兩種值。
	//     email 為主鍵，同樣採一對一對應的天然鍵設計。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_users (
			email VARCHAR(255) PRIMARY KEY,
			status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 11) 標籤字典表。UNIQUE(name) 讓標籤名稱天然去重，
	//     管理端新增標籤時重複名稱會直接得到 1062 而不必先做SELECT 檢查。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_user_tags (
			id BIGINT NOT NULL PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE KEY uq_forum_user_tags_name (name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 12) 使用者—標籤關聯表。複合主鍵 (user_email, tag_id) 同時保證「不重複掛同一標籤」
	//     並提供以 user_email 為條件的查詢索引（forumAuthorTags 走的就是這條）。
	//     idx_..._tag_id 則服務反向操作：刪除標籤時依 tag_id 批次清除關聯。
	//     刻意不設外鍵：刪標籤的關聯清除由應用層在交易中手動執行（見 handleAdminTag 的 DELETE），
	//     避免外鍵的鎖定順序與既有交易互相等待。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_user_tag_assignments (
			user_email VARCHAR(255) NOT NULL,
			tag_id BIGINT NOT NULL,
			PRIMARY KEY (user_email, tag_id),
			INDEX idx_forum_user_tag_assignments_tag_id (tag_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 13) 資料回填：把 forum_posts / forum_post_comments / forum_post_likes / forum_profiles
	//     中已存在卻從未寫入 forum_users 的 email 補建為 ACTIVE 帳號。
	//     為什麼需要：forum_users 是後來才引入的停權檢查表，在它出現之前累積的資料不會自動
	//     有對應帳號列，會導致 isForumUserSuspended 雖對 sql.ErrNoRows 視為未停權（可正常運作），
	//     但管理端使用者列表會看不到這些早期使用者。此段把清單補齊以維持資料一致性。
	//     為什麼用 INSERT IGNORE：email 是主鍵，重複資料必然撞主鍵；IGNORE 讓回填可重複執行
	//     且不因已存在而中斷，INSERT ... SELECT 在單一陳述式內完成也避免了大筆逐列寫入。
	//     UNION（非 UNION ALL）會去重，四張表同一個 email 只會插入一列。
	//     WHERE email <> '' 過濾掉舊資料中可能存在的空字串作者（NOT NULL 不代表值非空）。
	//     使用 NOW() 而非應用層時間：此處是一次性回填，統一由資料庫時區落戳即可。
	if _, err := db.Exec(`
		INSERT IGNORE INTO forum_users (email, status, created_at, updated_at)
		SELECT email, 'ACTIVE', NOW(), NOW() FROM (
			SELECT author_email AS email FROM forum_posts
			UNION SELECT author_email FROM forum_post_comments
			UNION SELECT author_email FROM forum_post_likes
			UNION SELECT author_email FROM forum_profiles
		) AS existing_users WHERE email <> '';
	`); err != nil {
		return err
	}

	// 14) forum_profiles 的欄位補齊，理由同第 6 步：CREATE TABLE IF NOT EXISTS 對既有表無效。
	//     bio 為 NOT NULL TEXT，新增加到已有資料的表時 MySQL 會填入空字串。
	if err := addColumnIfNotExists(db, "forum_profiles", "bio", "TEXT NOT NULL"); err != nil {
		return err
	}

	// 15) public_key 補欄位（新增需求：訪客以雜湊值查公開頁面）。
	if err := addColumnIfNotExists(db, "forum_profiles", "public_key", "VARCHAR(64) NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	// 16) public_key 回填。空字串代表「尚未計算」，以 SHA2(author_email, 256) 補上 64 字元雜湊。
	//     SHA2 為 MySQL 內建函式，不需應用層逐列往返，單一 UPDATE 即可完成。
	//     必須與應用層的 publicForumKey（Go 的 sha256.Sum256 + %x）保持位元級一致，
	//     否則既有使用者的公開頁面會查不到資料；SHA2 對 email 採 UTF-8 位元組計算，
	//     與 Go 對字串直接 Sum256 的結果相同。
	//     WHERE 條件限制在 public_key = ''，使此敘述可重複執行且不會覆寫既有值。
	if _, err := db.Exec(`
		UPDATE forum_profiles SET public_key = SHA2(author_email, 256) WHERE public_key = '';
	`); err != nil {
		return err
	}

	// 17) 暱稱唯一索引。
	//     為什麼要這個約束：公開頁面以 nickname 顯示作者，重複暱稱會讓身分無法辨識；
	//     加上唯一索引後，競態下的重複寫入會由資料庫擋下（1062），handler 據此回 409。
	//     已知風險：若既有資料本身就有重複暱稱，這一步會讓整個遷移失敗並中止啟動，
	//     必須先人工清理重複值 —— 這是刻意選擇「寧可啟動失敗也不靜默產生重複暱稱」。
	//
	//     建立走 addUniqueIndexIfNotExists：MySQL 沒有「ADD UNIQUE INDEX IF NOT
	//     EXISTS」，而它內部對 Error 1061（別的實例剛建好）與 1062（資料本身重複）
	//     的差別對待 —— 前者通過、後者中止 —— 見該函式與 indexAlreadyExists 的說明。
	if err := addUniqueIndexIfNotExists(db, "forum_profiles",
		"uq_forum_profiles_nickname", "nickname"); err != nil {
		return err
	}

	// 18) 資料遷移：forum_posts.image_url 由「完整網址」改為「純檔名」。
	//     為什麼需要：此欄過去存的是檔案伺服器回傳的完整網址，導致換對外網域
	//     必須 UPDATE 全表，而且內部位址（http://192.168.66.5:7070/…）會被寫進
	//     資料庫。新契約只存檔名，完整網址由應用層以 FILES_SERVER_PUBLIC_URL 組出。
	//     作法：先以 SUBSTRING_INDEX(…, '?', 1) 去掉 ?token=…，再取最後一段路徑，
	//     最後丟掉舊版包成 JSON 後留下的尾段 `"]`（與第 2b 步的 Go 收斂、
	//     httpapi/purge 的 SQL 表達式同一個規則）。合法檔名的白名單是
	//     [A-Za-z0-9._-]，`"` 與 `]` 不在其中，因此那一步不會誤傷任何合法檔名。
	//     WHERE image_url LIKE '%/%' 是冪等條件：轉換後的值不含 "/"，因此
	//     第二次啟動不會再動到任何一列。含 "?token=" 但無路徑者不可能出現
	//     （token 一定接在 /files/<檔名> 之後），故不另設條件。
	//
	//     這一步現在是**第二道**防線而不是唯一一道：第 2b 步的
	//     normalizeForumImageValues 先把值收斂成純檔名再包成 JSON 陣列，因此正常
	//     情況下這裡查不到任何列（條件命中不了新契約的值）。保留它是為了讓「只有
	//     SQL、沒有 Go 收斂」的舊版本仍能修正單一值。已知限制：對**多元素**的 JSON
	//     陣列做字串切割只能得到最後一段（前面的圖會消失）—— 那正是 2b 必須先收斂
	//     的理由（見 2b 的說明），這裡只當兜底。
	//     已知限制：此敘述只做字串切割，不驗證主機。舊資料若存的是第三方
	//     圖片網址，切完會得到一個指向自己檔案伺服器、但實際不存在的檔名，
	//     讀取時回空字串（前端不顯示縮圖）。應用層的 forumImageFileName
	//     會容忍尚未遷移的舊寫法，因此兩者不必同步進行。
	if _, err := db.Exec(`
		UPDATE forum_posts
		SET image_url = SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(image_url, '?', 1), '/', -1), ']', 1), '"', 1)
		WHERE image_url LIKE '%/%';
	`); err != nil {
		return err
	}

	// 19) 追蹤關聯表。複合主鍵 (follower_email, target_email) 同時扮演
	//     「一人對另一人只追蹤一次」的業務規則與兩種查詢所需的索引：
	//       - 查自己的追蹤清單：WHERE follower_email = ?  → 主鍵最左前綴
	//       - 追蹤動態：IN (SELECT target_email ... WHERE follower_email = ?)
	//         同樣走最左前綴，不需要額外索引
	//       - 取消追蹤的 DELETE：完整複合主鍵
	//     idx_forum_follows_target 服務反向操作（依被追蹤者查追蹤者），
	//     目前沒有端點使用它，保留是為了日後要清理或做統計時不必回頭改 schema。
	//
	//     為什麼 target 存 email 而不是 public_key：public_key 是 SHA-256(email)，
	//     若以它作為關聯鍵，追蹤動態的條件就得寫成
	//     SHA2(fp.author_email, 256) IN (...) —— MySQL 無法為運算式建索引，
	//     等於每次請求都掃全表。存 email 則 IN 子查詢走複合主鍵，與按讚表
	//     （forum_post_likes）同一種形狀，email 也永遠不會因此下放到公開 API。
	//
	//     刻意不設外鍵，理由同 forum_post_comments：關聯由應用層維護，避免外鍵的
	//     鎖定順序與既有交易互相等待。使用者帳號只有停權、沒有刪除路徑，因此也不需要
	//     ON DELETE CASCADE。
	//
	//     注意：複合主鍵無法阻止自我追蹤（('a','a') 是合法的主鍵值），
	//     那一條規則由 handler 檢查（見 forum_follow_handlers.go）。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_follows (
			follower_email VARCHAR(255) NOT NULL,
			target_email VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (follower_email, target_email),
			INDEX idx_forum_follows_target (target_email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 20) 為 forum_profiles.public_key 補索引。
	//
	//     為什麼需要：handleForumPublicProfile 的註解宣稱「public_key 有索引」，
	//     但實際上 forum_profiles 的主鍵是 author_email，public_key 上從來沒有
	//     索引，那句註解是不準的。追蹤功能的 key→email 解析（forumEmailByPublicKey）
	//     會拿同一個欄位當查詢條件，因此這裡把它補成索引。
	//
	//     為什麼是「非唯一」索引：UNIQUE 會讓「既有資料有重複值」直接讓整個遷移
	//     失敗、服務無法啟動。第 16 步已把空值回填為 SHA2(email, 256)，
	//     理論上不會重複，但把服務的可啟動性押在這個假設上不划算 ——
	//     索引的唯一性對這個查詢毫無幫助（等值比對本來就只會有一列）。
	//
	//     建立走 addIndexIfNotExists（探測 + 容忍併發的 Error 1061），理由見
	//     indexAlreadyExists：兩邊同時探到「不存在」時，後者拿 1061 不是失敗。
	if err := addIndexIfNotExists(db, "forum_profiles",
		"idx_forum_profiles_public_key", "public_key"); err != nil {
		return err
	}

	// 21) 為 forum_posts.author_email 補索引。
	//
	//     兩處用到它，缺了都是全表掃描：
	//
	//       1. 公開個人頁的貼文列表（handleForumPublicPosts）以 author_email 為
	//          唯一條件，那是一個不需登入的端點 —— 一個會掃全表又沒限流的端點
	//          是最容易被拿來打的一種。
	//       2. 金鑰 → email 的退路解析（forumEmailByPublicKey）要跑
	//          `SELECT DISTINCT author_email FROM forum_posts`。有了這個索引，
	//          那條查詢是「走索引、不碰資料列」的掃描，成本與貼文筆數脫鉤到
	//          只有索引項的讀取量 —— 對一位從未設定暱稱的早期使用者按追蹤鈕
	//          會走這條路徑，而它掛在內容寫入限流之下，本來就不該更貴。
	//
	//     建立走 addIndexIfNotExists（非唯一索引，理由同第 20 步）。
	if err := addIndexIfNotExists(db, "forum_posts",
		"idx_forum_posts_author_email", "author_email"); err != nil {
		return err
	}

	// 22) 分鐘級請求統計（監控頁 /admin/monitor 的持久化來源）。
	//
	//     一列代表一分鐘，因此主鍵就是 bucket_minute，沒有自增 id：這個表
	//     永遠是 upsert 與範圍掃描兩種存取，用代理鍵只會讓兩者都要多走一次
	//     二級索引。時間軸的查詢是 WHERE bucket_minute >= ? ORDER BY
	//     bucket_minute，剛好就是主鍵的前綴掃描。
	//
	//     欄位刻意「足夠畫圖就好」：只存總量、4xx、5xx 與耗時總和，沒有
	//     依路由分拆的明細表。原因是明細會隨路由數量線性成長，而維運真正
	//     會回頭查「三小時前那波流量來自哪條路由」的機率非常低 —— 記憶體裡
	//     的 120 分鐘即時資料已經涵蓋那個情境（見 metrics 套件的說明）。
	//     需要依路由的長期資料是 access log 的工作，不是這張表的。
	//
	//     duration_sum_ms 存的是總和而非平均：平均在累加多個執行個體的分鐘
	//     時無法正確合併（(10+20)/2 不等於 (30+40)/2），存總和則可以在查詢
	//     時用 SUM()/COUNT() 得到正確的加權平均。
	//
	//     這張表是純衍生資料：刪掉它只會讓監控頁少一段重啟前的歷史，論壇
	//     的任何功能都不依賴它（見 metrics 套件的「為什麼不做的東西」）。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_request_metrics (
			bucket_minute DATETIME NOT NULL,
			total BIGINT NOT NULL DEFAULT 0,
			client_errors BIGINT NOT NULL DEFAULT 0,
			server_errors BIGINT NOT NULL DEFAULT 0,
			duration_sum_ms BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket_minute)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 23) 管理員操作稽核紀錄。
	//
	//     這張表的唯一用途是回答「誰在什麼時候對哪個對象做了什麼、哪些欄位
	//     從什麼變成什麼」。它涵蓋後臺所有會改變資料的操作（停權、刪文、
	//     裁定檢舉、改標籤、代發文），而 access log 做不到這件事：log 只
	//     記到「某個 IP 對 /api/admin/forum/reports/9 送出 DELETE」。
	//
	//     索引的三個方向各對應一個實際會被問的問題：
	//       idx_created (created_at)          「最近發生了什麼」「清理過期」
	//       idx_actor   (actor_email, created_at) 「某個管理員做了什麼」
	//       idx_target  (target_type, target_id) 「某個對象被動過幾次」
	//     刻意不為 changes 開全文索引：欄位是 JSON，而「哪個欄位被改成某值」
	//     這種查詢在這個規模下先用 actor/action/target 篩出少量列再讀即可。
	//
	//     actor_email 沒有外鍵也沒有索引型別上的限制：它是文字而不是
	//     forum_users 的參照，因為管理員不一定要有 forum_users 的列
	//     （ALLOWED_ADMIN_EMAIL 白名單裡的人可能從未發文）。
	//
	//     這張表**沒有**提供任何刪除 API，只有依時間的清理
	//     （audit.Pruner，保留期由 AUDIT_RETENTION_DAYS 控制）。一個能刪
	//     除自己紀錄的稽核日誌等於沒有稽核日誌。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_admin_actions (
			id BIGINT NOT NULL,
			actor_email VARCHAR(320) NOT NULL,
			action VARCHAR(64) NOT NULL,
			target_type VARCHAR(32) NOT NULL,
			target_id VARCHAR(320) NOT NULL,
			target_label VARCHAR(320) NOT NULL DEFAULT '',
			changes TEXT NULL,
			ip VARCHAR(64) NOT NULL DEFAULT '',
			request_id VARCHAR(64) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_forum_admin_actions_created (created_at),
			KEY idx_forum_admin_actions_actor (actor_email, created_at),
			KEY idx_forum_admin_actions_target (target_type, target_id),
			KEY idx_forum_admin_actions_action (action)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 24) forum_users 的 created_at 索引。
	//
	//     為什麼在這裡才加：第 10 步建表時沒有這個索引，而它唯一的用途是
	//     內容趨勢統計（/admin/stats）的「每日新使用者」查詢
	//     （WHERE created_at >= ? GROUP BY DATE(created_at)）。在那之前沒有
	//     任何查詢會依 created_at 過濾 forum_users，所以這個索引是純粹的
	//     新需求帶來的。
	//
	//     沒有的話那一條查詢是全表掃描。論壇的使用者數不多（每個都要經過
	//     Google 登入才會建立），所以全表掃描在這個規模下其實不痛 —— 但
	//     它是整個 /admin/stats 裡唯一一個「沒有任何可用索引」的查詢，
	//     也就是唯一一個會隨使用者數成長而變慢的。與其讓它成為未來除錯時
	//     的疑點，不如現在補上；索引維護的成本是每次建立帳號多寫一筆
	//     B-tree，而建立帳號是後臺最少見的操作之一。
	//
	//     刻意**不**為 forum_posts.author_email 與 forum_posts 補複合索引：
	//     「熱門作者排行」那條查詢會 GROUP BY author_email 而沒有可用索引，
	//     但它被 WHERE created_at >= ? 限制在 30 天內，且論壇的文章總量本來
	//     就遠小於使用者數的成長速度。為了一條有界的查詢在「最熱的表」上
	//     加索引，是把成本放在每次發文而不是偶爾開一次後臺 —— 取捨不划算。
	//     同一個理由也讓我沒有為 forum_post_likes.created_at 加索引 —— 但這條
	//     結論後來被推翻了：/admin/stats 的「視窗內按讚合計」
	//     （SELECT COUNT(*) FROM forum_post_likes WHERE created_at >= ?）確實
	//     用到它，而它並不屬於「最近 30 天」的量，會隨**全站累積按讚數**成長。
	//     該索引已在第 27 步補上，這裡保留原始推理作為「前提被推翻」的紀錄。
	//
	//     建立走 addIndexIfNotExists（探測 + 容忍併發的 Error 1061），理由見
	//     indexAlreadyExists：兩邊同時探到「不存在」時，後者拿 1061 不是失敗。
	if err := addIndexIfNotExists(db, "forum_users",
		"idx_forum_users_created_at", "created_at"); err != nil {
		return err
	}

	// 25) 站內公告與文章置頂。
	//
	//     forum_announcements 一列是一則公告，而**同一時間只有一則是 active**。
	//     這個限制是刻意的：公告在公開頁上是一條橫幅，若同時有兩三條，使用者
	//     會看到一個被橫幅佔掉的上半頁，而其中兩條還可能是互相衝突的
	//     （一則說「活動改期」、一則說「活動照常」）。表格保留多列是為了
	//     留下「這則公告是什麼時候、經誰發布、之後被誰關掉」的歷史，而那正是
	//     稽核紀錄之外的另一半脈絡。
	//
	//     expires_at 允許 NULL（永不自動到期）。它存在的理由是「過期」與
	//     「手動停用」是兩件事：活動結束的公告會自然過期，而放錯一則需要
	//     立刻消失 —— 前者不該要求管理員記得回來關，後者不該等到過期時間。
	//
	//     沒有「排序」欄位：既然同時只有一則 active，排序就沒有作用對象。
	//     也沒有 FOREIGN KEY 指向 forum_users：公告可能由一個已刪除的
	//     管理員帳號發布，而那一列仍然要留下。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_announcements (
			id BIGINT NOT NULL,
			body VARCHAR(300) NOT NULL,
			active TINYINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			created_by VARCHAR(320) NOT NULL,
			updated_at DATETIME NOT NULL,
			updated_by VARCHAR(320) NOT NULL DEFAULT '',
			expires_at DATETIME NULL,
			PRIMARY KEY (id),
			KEY idx_forum_announcements_active_expires (active, expires_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 26) forum_posts.pinned 與對應的動態索引。
	//
	//     pinned 是 TINYINT 而非 BOOLEAN：MySQL 的 BOOLEAN 其實是 TINYINT(1)，
	//     宣告成 TINYINT 讓「只有 0/1 兩種值」這件事在型別上直接可見。
	//
	//     索引刻意是**遞減**的（MySQL 8.0+ 才支援，README 的環境需求是 MySQL
	//     8.x）：動態的排序是 pinned DESC, created_at DESC, id DESC，而
	//     MySQL 的「反向掃描索引」會把**所有**欄位一起反向，因此一個
	//     (pinned, created_at) 的遞增索引無法服務這個排序 —— 查詢最佳化器會
	//     判定它不能用而退回 filesort。
	//
	//     保留既有的 idx_forum_posts_created_at：搜尋結果、
	//     profile 的貼文列表與管理端的多處查詢都只按 created_at 排序，
	//     那些地方不該為了支援置頂而多付一個無法命中的索引。
	//
	//     欄位與索引拆成兩句：欄位走 addColumnIfNotExists（同第 14、15 步），
	//     索引則交給 addIndexIfNotExists —— 兩者混在一句 ALTER 裡的話，第二次
	//     啟動會因重複的 key 名稱（1061）讓整個遷移失敗，而主流程會因此 Fatal。
	if err := addColumnIfNotExists(db, "forum_posts", "pinned", "TINYINT NOT NULL DEFAULT 0 AFTER image_url"); err != nil {
		return err
	}
	//     索引同樣走 addIndexIfNotExists：探測與建立之間的窗口在兩個實例同時
	//     啟動時會被命中，而 Error 1061 在那個情境下是「別人先建好了」而不是
	//     失敗（見 indexAlreadyExists）—— 這正是本檔第 1 點宣稱的
	//     「兩個實例同時啟動不會互相卡死」的落點。
	if err := addIndexIfNotExists(db, "forum_posts",
		"idx_forum_posts_feed", "pinned DESC, created_at DESC, id DESC"); err != nil {
		return err
	}

	// 27) 補上被新查詢推翻的「刻意不加」判斷（修正第 24 步的結論）。
	//
	//     第 24 步當時的推理是對的，但前提被這批變更改變了 —— 索引的理由從
	//     「現在沒有查詢會用到它」變成「有查詢會用到它」，所以必須補：
	//
	//     a) forum_post_comments.author_email 與 **forum_posts.author_email**
	//        兩張表各有一條以 author_email 為條件的查詢退化成全表掃描：
	//        使用者 CSV 匯出（httpapi/csv_export.go 的預先聚合衍生表）
	//        與熱門作者排行（stats_handlers.go）都要對「每個作者算一次」，
	//        因此前者是「每一列輸出一掃」、後者是「整張留言表掃一次」。
	//        csvExportMaxRows 只限制回傳列數，**不限制衍生表讀取的列數**，
	//        因此症狀是「留言數萬後按一次匯出就轉圈到逾時」。
	//
	//        刻意把兩個 author_email 索引寫在一起：只補留言那一半會讓
	//        註解與呼叫端（csv_export.go、stats_handlers.go 的說明）
	//        宣稱「已修好」，而實際上文章表仍然每次掃全表 —— 註解與
	//        實作不符會讓後人不再往下追。
	//
	//     b) forum_post_likes.created_at
	//        第 24 步的註解說「那條查詢根本沒用到它」，但
	//        /admin/stats 的「視窗內按讚合計」（SELECT COUNT(*) ...
	//        WHERE created_at >= ?）確實用到 —— 它是那支端點裡唯一一個
	//        沒有可用索引的子查詢，成本隨**全站累積按讚數**（而非最近 N 天）
	//        成長。
	//
	//     建立走 addIndexIfNotExists：探測與建立之間的窗口在兩個實例同時啟動時
	//     會被命中，而 Error 1061 在那個情境下是「別人先建好了」而不是失敗
	//     （見 indexAlreadyExists）。
	//
	//     注意 b) 的索引成本：每新增一筆按讚多寫一個 B-tree 節點。按讚是
	//     這個站最熱的寫入，但它的寫入量遠小於 forum_request_metrics 那類
	//     分鐘彙總的寫入量，所以這個取捨仍然划算。
	//
	//     a) 的兩個索引成本同理但更低：forum_posts 的寫入量遠低於按讚，
	//     而 author_email 是單欄索引（B-tree 節點數 = 該欄相異值數 × 深度），
	//     不隨該作者的貼文數成長。
	if err := addIndexIfNotExists(db, "forum_post_comments",
		"idx_forum_post_comments_author_email", "author_email"); err != nil {
		return err
	}
	// 保留既有的 idx_forum_posts_created_at（搜尋結果、profile 貼文列表與
	// 管理端的多處查詢只按 created_at 排序），這一個是為了 author_email
	// 等值查詢而存在 —— 兩者服務不同的存取樣式，不是同一件事的兩種寫法。
	if err := addIndexIfNotExists(db, "forum_posts",
		"idx_forum_posts_author_email", "author_email"); err != nil {
		return err
	}
	if err := addIndexIfNotExists(db, "forum_post_likes",
		"idx_forum_post_likes_created_at", "created_at"); err != nil {
		return err
	}

	// 28) 貼文與留言的「最後修改時間」。
	//
	//     為什麼需要：公開端新增了「作者可以編輯自己的貼文／留言」之後，讀者
	//     會遇到一個新問題 —— 眼前這段話可能已經不是作者最初寫的。沒有這個
	//     欄位就只能在介面上假裝沒這件事，而假裝是論壇裡最不該做的一種掩蓋。
	//
	//     允許 NULL，且 NULL 的語意是「從未被編輯過」。這是本步驟唯一的關鍵
	//     決定：若宣告 NOT NULL DEFAULT CURRENT_TIMESTAMP 或用計數器欄位，
	//     每一筆既有資料都會看起來「剛剛被改過」，於是「已編輯」標記會出現在
	//     每一篇文章上 —— 那等於沒有標記，只是讓全站多一個雜訊。既有資料
	//     保持 NULL 正是它應得的答案：這些文章確實沒有被編輯過。
	//
	//     型別選 DATETIME 而非 TIMESTAMP：這個專案的 created_at 一律是應用層
	//     寫入的 time.Now()（主機本地時區，見第 1 步），TIMESTAMP 會做時區
	//     轉換而讓兩欄在同一列裡表示不同的時區，那是比「沒有時區資訊」更糟的
	//     歧義。
	//
	//     各自一句 ALTER 而非併成一句：兩張表互相獨立，合併後若其中一張表已經
	//     有該欄位，另一張是否被補上就不再是這段 SQL 能表達的事。
	//
	//     不加索引：唯一會用到它的地方是「取出一篇貼文看它有沒有被編輯過」，
	//     而那一定已經有 id 等值條件走主鍵。為此在「最熱的表」上加一個只服務
	//     單一列讀取的索引，與第 24 步不為熱門作者排行加 author_email 索引是
	//     同一個判斷。
	if err := addColumnIfNotExists(db, "forum_posts", "updated_at", "DATETIME NULL AFTER pinned"); err != nil {
		return err
	}
	if err := addColumnIfNotExists(db, "forum_post_comments", "updated_at", "DATETIME NULL AFTER created_at"); err != nil {
		return err
	}

	// 29) 使用者頭像。
	//
	//     為什麼需要：匿名論壇的對外身分只有 SHA256(email) 與暱稱，而暱稱是自由
	//     填寫的文字（還可能沒填）。少了頭像，「分辨這是誰」只能靠讀名字 ——
	//     一張自選圖片讓一個人在動態裡被認出的成本遠低於讀六個中文字。
	//
	//     儲存契約與 forum_posts.image_url 完全一致（見第 2 步）：只存**檔名**，
	//     完整網址由應用層以 FILES_SERVER_PUBLIC_URL 組成。這同時代表「換對外
	//     網域不必搬資料」，以及內部位址（http://192.168.66.5:7070）不會進資料庫。
	//     副檔名白名錄、路徑穿越與第三方主機的防線全部沿用 forumImageFileName
	//     （寫入時收斂、讀取時驗證），這裡不另做一套。
	//
	//     刻意**不**建索引：這個欄位的唯一讀法是「已經拿著 author_email 或
	//     public_key 找到那一列」，走的就是主鍵或 idx_forum_profiles_public_key，
	//     索引沒有作用對象。而在 forum_profiles 這種一列一人的表上加一個只服務
	//     單列讀取的索引，與第 24 步不為熱門作者排行加索引是同一個判斷。
	//
	//     刻意宣告 NOT NULL 而不給 DEFAULT：與第 2 步同一個理由 —— 嚴格模式下
	//     既有列會被補上空字串，而空字串在這裡的語意就是「沒有頭像」，應用層
	//     的 forumImageFileName 對它回傳空字串，前端因此不渲染 <img> 而退回
	//     暱稱首字。不需要「從未設定」與「已移除」兩種表示。
	//
	//     選 TEXT 而非 VARCHAR：沿用 image_url 的型別決定（長度不可預期的舊契約
	//     留下的形狀），對一個上限 128 字元的值不改欄位型別 —— 改型別是 DDL、
	//     失去回滾機會，並不划算。
	if err := addColumnIfNotExists(db, "forum_profiles", "avatar_url", "TEXT NOT NULL"); err != nil {
		return err
	}

	// 30) forum_posts.deleted_at：貼文刪除改成軟刪（soft delete）。
	//
	//     為什麼需要：刪除曾經是 DELETE FROM forum_posts —— 一刀下去，貼文、它
	//     被檢舉的脈絡、以及「這篇曾經存在」的事實同時消失。改為軟刪之後：
	//
	//       a) 誤刪可以救回來。使用者按刪除的當下沒有第二次確認以外的緩衝，
	//          而硬刪除讓那一下變成不可逆。
	//       b) 管理側仍握有脈絡。檢舉工單（forum_reports）指向的貼文被硬刪後，
	//          forum_admin_actions 的稽核紀錄雖然記了「刪文」，但內容只剩截斷
	//          後的前 200 字；軟刪讓原文仍在表裡。
	//       c) 外部的引用不會變成死線。留言、按讚、檢舉都還在，只是不再顯示。
	//
	//     NULL 的語意是「沒有被刪除」；有值則是刪除發生的時間。與 updated_at
	//     （第 28 步）同一個約定：NULL 代表「從未發生」，既有資料因此一律是
	//     「未刪除」，回填作業等於不存在。
	//
	//     刻意不用 TINYINT deleted 旗標：時間戳同時回答「有沒有被刪」與「什麼
	//     時候被刪」，而後者是清理工作（見下）唯一需要排序的依據。旗標要再補
	//     一個 deleted_at 才能做同一件事。
	//
	//     型別選 DATETIME 而非 TIMESTAMP：與第 28 步同一個理由 —— 本專案的
	//     時間欄位一律由應用層寫入 time.Now()（主機本地時區），TIMESTAMP 會
	//     做時區轉換，讓同一列裡的兩個欄位表示不同時區。
	//
	//     查詢側的索引在第 36 步補上（清理排程每小時掃過期貼文）：當時
	//     「唯一會用到它的查詢都是拿著 id 找到那一列」的判斷是對的，因為
	//     還沒有清理排程；排程上線後那條判斷就不成立了。
	//
	//     資料會自動消失：軟刪把「從資料庫抹除」的責任從 handler 移了出去，
	//     而清理排程（forum/purge）接管了它 —— 保留期由 POST_RETENTION_DAYS
	//     決定（預設 90 天），到期後連留言／按讚／收藏／檢舉與不再被引用的
	//     附圖一併清掉。
	if err := addColumnIfNotExists(db, "forum_posts", "deleted_at", "DATETIME NULL AFTER updated_at"); err != nil {
		return err
	}

	// 31) 留言的層級：forum_post_comments.parent_id 與 depth。
	//
	//     為什麼需要：留言原本只有「貼文 → 留言」一層，而討論串裡的回覆對象經常是
	//     另一則留言（「我同意三樓」「你第二點不對」）。少了 parent_id 就無法回答
	//     「這則在回覆誰」，也沒有辦法把同一篇的留言收斂成樹；介面因此只能把所有
	//     留言排成一條時間線，讀者要自己從內容推測誰在跟誰說話。
	//
	//     最多三層（forumCommentMaxDepth）是產品決定，而它在 schema 上的落點是
	//     depth：寫入時取「父留言的 depth + 1」，因此第四層在 handler 就被擋掉，
	//     不會進到資料庫。depth 是衍生值（理論上可以沿著 parent_id 鏈現算），但
	//     它不可能漂移 —— 沒有任何端點會改寫父留言（parent_id 寫入後即固定），
	//     而刪除留言是連同整個子樹一起刪（見 handleForumCommentDelete 的說明），
	//     所以「子留言比父留言深」這個不變式只有建立時會被檢查一次，之後不會被
	//     任何路徑破壞。
	//
	//     兩個表示的選擇：
	//       - parent_id 用 0 而不是 NULL 表示「沒有父留言」：掃描進 int64 即可，
	//         NULL 要動用 sql.NullInt64，而「第一層」是多數列。0 同時讓「是不是
	//         回覆」變成一個 > 0 的比較。
	//       - depth 用 TINYINT：上限 3，而它唯一的用途是「還能不能再往下一層」。
	//
	//     不加索引：parent_id 的唯一讀法是「拿著父留言的 id 找那一列」，走主鍵；
	//     列表端一次取出整篇貼文的留言（WHERE post_id = ?），層級由前端組樹，
	//     資料庫不需要知道誰的子孫是誰。為此在留言這種最熱的表上加索引，與第 28
	//     步不為 updated_at 加索引是同一個判斷。
	//
	//     NOT NULL + DEFAULT 讓既有資料自動成為「depth=1 的第一層留言」—— 那正是
	//     它們在這個功能之前的樣子，回填作業等於不存在。
	//
	//     各自一句 ALTER 而非併成一句：兩條都走 addColumnIfNotExists，分開寫
	//     讓「只補了其中一欄」的狀態可被敘述（下次啟動會補上另一欄），合併後
	//     其中一欄已存在時整句的行為需要另外查證。
	if err := addColumnIfNotExists(db, "forum_post_comments", "parent_id", "BIGINT NOT NULL DEFAULT 0 AFTER post_id"); err != nil {
		return err
	}
	if err := addColumnIfNotExists(db, "forum_post_comments", "depth", "TINYINT NOT NULL DEFAULT 1 AFTER parent_id"); err != nil {
		return err
	}

	// 32) 站內通知（forum_notifications）。
	//
	//     為什麼需要：留言、回覆與追蹤過去都是「單向」的 —— 有人回應了你的貼文、
	//     回覆了你的留言、或追蹤了你，除了自己回去翻動態之外沒有任何提示。通知把
	//     這三件事變成收件者看得到的事件。
	//
	//     一列是一則通知，形狀刻意「反正規化」：recipient_email 是收件者、
	//     actor_email 是觸發者，兩者都是 email（理由同第 19 步的 forum_follows：
	//     用 public_key 當關聯鍵會讓查詢變成 SHA2(...) 而無法走索引）。
	//     post_id 與 comment_id 一起放進同一張表，而不是「看 kind 決定去哪張表
	//     查目標」—— 那會讓列表端必須為每一列各自 JOIN，而這裡的三種 kind
	//     都只需要「點了要跳到哪裡」這一個數字，多一欄的成本遠低於一次 JOIN。
	//
	//     kind 目前有三種值：comment（有人在你貼文留言）、reply（有人回覆你的
	//     留言）、follow（有人追蹤你）。用 VARCHAR 而非 ENUM：新增一種事件時
	//     ENUM 要 ALTER TABLE 改型別，而這裡沒有任何 CHECK 依賴它。
	//
	//     read_at 允許 NULL，NULL 的語意是「未讀」。與 updated_at（第 28 步）、
	//     deleted_at（第 30 步）同一個約定：NULL 代表「這件事還沒發生」，
	//     因此既有列一律是未讀，回填作業等於不存在。刻意不用 TINYINT 旗標：
	//     時間戳同時回答「讀過沒」與「什麼時候讀的」。
	//
	//     索引兩個方向：
	//       idx_forum_notifications_recipient (recipient_email)
	//         「我的通知」列表（WHERE recipient_email = ? ORDER BY id DESC）。
	//         InnoDB 會把主鍵（id）接在二級索引之後，因此這個索引本身就能
	//         服務 id 的遞減排序，不需要額外的排序欄位。
	//       idx_forum_notifications_unread (recipient_email, read_at)
	//         未讀計數（WHERE recipient_email = ? AND read_at IS NULL）——
	//         導覽列的鈴鐺每個頁面載入都會問一次，它必須只碰索引不碰資料列。
	//
	//     沒有外鍵、沒有 ON DELETE CASCADE：關聯由應用層維護，理由同
	//     forum_follows 與 forum_post_comments。收件者帳號只有停權、沒有刪除路徑，
	//     而指向的貼文／留言被刪除時，通知仍然保留（它記錄的是「當時發生過」）。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_notifications (
			id BIGINT NOT NULL,
			recipient_email VARCHAR(255) NOT NULL,
			actor_email VARCHAR(255) NOT NULL,
			kind VARCHAR(16) NOT NULL,
			post_id BIGINT NOT NULL DEFAULT 0,
			comment_id BIGINT NOT NULL DEFAULT 0,
			read_at DATETIME NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_forum_notifications_recipient (recipient_email),
			KEY idx_forum_notifications_unread (recipient_email, read_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 33) 上傳檔案的所有權紀錄（forum_media）。
	//
	//     為什麼需要：圖片上傳到檔案伺服器之後，本站對「這個檔名是誰上傳的」
	//     沒有任何紀錄 —— forum_posts.image_url 與 forum_profiles.avatar_url
	//     只回答「誰現在用它」。少了所有權，「刪除自己的圖片」就無法判斷
	//     一顆檔名該不該被刪，而「換圖」時的自動回收也只能靠引用關係猜測。
	//
	//     一列代表「某個檔名由某個人上傳」。檔名是主鍵：檔案伺服器以此為鍵，
	//     而本站的圖片契約也是「資料庫只存檔名」（見第 18 步），因此這裡存
	//     同一把鍵是最自然的對應。
	//
	//     為什麼沒有外鍵、沒有 TTL 欄位：
	//       - 外鍵：這個專案的關聯一律由應用層維護（同 forum_follows、
	//         forum_notifications），而 forum_media 指向的是檔案伺服器上的
	//         實體，不是另一張表。
	//       - TTL：到期回收是「更舊的孤兒檔」那條線的事，而刪除端點只需要
	//         「誰上傳的」與「還有沒有人在用」兩個問題的答案。先不猜測那個
	//         機制會長什麼樣子。
	//
	//     沒有 owner_email 的索引：唯一的查詢是「by filename」（主鍵）與
	//     「這個檔名的擁有者是誰」（也是主鍵）。「某人上傳過什麼」目前沒有
	//     任何呼叫端，加索引只會讓每一筆寫入多一次維護。
	//
	//     舊資料（這個表存在之前上傳的檔案）沒有列，它們的所有權判定走
	//     「目前被誰引用」的退路（見 forumFileAttachable）。因此這個表是
	//     「新制」的紀錄，而不是歷史的回填 —— 回填做不到，因為過去的上傳
	//     事件沒有任何地方留過作者。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_media (
			filename VARCHAR(128) NOT NULL,
			owner_email VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (filename)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 34) 收藏（forum_post_bookmarks）。
	//
	//     為什麼需要：行動網路時代的閱讀常常是「先存著，之後再看」。既有的
	//     按讚是公開的（讚數會顯示），而收藏是私人的備忘 —— 兩者是不同的
	//     語意，因此是獨立的表而不是按讚的一個旗標。
	//
	//     形狀與 forum_post_likes 逐字相同：複合主鍵 (post_id, author_email)
	//     表示「一人一文一收藏」，取消收藏是 DELETE 而不是旗標（沒有歷史
	//     語意，理由同按讚與追蹤）。
	//
	//     索引的取捨也與按讚相同：複合主鍵的最左前綴支援「這篇有幾個人
	//     收藏」；反方向的「我收藏了哪些」需要例外一條以 author_email 開頭
	//     的索引（收藏頁的主查詢，見 handleForumBookmarks）。
	//
	//     created_at 是收藏的時間。目前**沒有**任何查詢用它排序：收藏頁刻意
	//     沿用全站動態的排序（見 handleForumBookmarks 的說明），而「這筆是
	//     什麼時候建立的」仍然值得留著 —— 取消再收藏會換到一個新的時間戳，
	//     那正是它該有的行為。它不參與任何唯一性判斷。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_post_bookmarks (
			post_id BIGINT NOT NULL,
			author_email VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (post_id, author_email),
			KEY idx_forum_post_bookmarks_author (author_email, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 35) 日別新增量彙總（forum_daily_stats）。
	//
	//     為什麼需要：/admin/stats 的日別序列過去每次讀取都即時計算
	//     （GROUP BY DATE(created_at)），掃描量被視窗限制住。當內容量
	//     成長到「最近 90 天」也不再有界時，這個查詢會開始變慢——
	//     docs/KNOWN_ISSUES.md 與 stats_handlers.go 檔頭都寫明了「那時該
	//     加一張每日彙總表」。這張表就是那個補上。
	//
	//     形狀：一列一天，四個計數是「那一天新增的」使用者／貼文／留言／
	//     按讚。貼文計數含後來被軟刪除的——它回答的是「那天產生了多少」，
	//     不是「今天還看得到多少」（後者在用戶管理與文章列表都有）。
	//
	//     它是純衍生資料，與 forum_request_metrics 同一個定位：整表刪除
	//     只會讓趨勢頁少一段歷史，論壇的任何功能都不依賴它。彙總由
	//     data.RefreshDailyStats 定期重算最近幾天，更早的日期由
	//     data.BackfillDailyStats 從來源表回填一次（INSERT IGNORE，已存在的
	//     日期不覆寫——那些日期可能已被清理排程硬刪，重算反而會少算）。
	//
	//     主鍵就是日期：一天一列是這張表的唯一契約，而「按日期範圍讀」
	//     正是唯一的查詢形狀，不需要其他索引。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_daily_stats (
			day DATE NOT NULL,
			new_users INT NOT NULL DEFAULT 0,
			new_posts INT NOT NULL DEFAULT 0,
			new_comments INT NOT NULL DEFAULT 0,
			new_likes INT NOT NULL DEFAULT 0,
			PRIMARY KEY (day)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 36) 查詢用索引：軟刪除清理排程、檢舉工單與圖片引用檢查。
	//
	//     這些查詢都是「結果集很小、但會經常跑」的型態。沒有索引時它們是
	//     全表掃描，而掃描的成本與**整張表**成正比 —— 表愈大，愈會在
	//     平常的請求路徑上拖慢整個站（清理排程與正常請求共用同一個連線池）。
	//
	//     a) forum_posts (deleted_at, id)：清理排程每小時取「已刪除且超過
	//        保留期」的下一批（WHERE deleted_at IS NOT NULL AND deleted_at < ?
	//        ORDER BY deleted_at, id LIMIT ?）。單欄 deleted_at 索引只能加速
	//        WHERE，排序仍要 filesort；複合索引讓範圍與排序都由同一棵 B-tree
	//        完成，LIMIT 可以在索引上停下。舊的單欄索引
	//        idx_forum_posts_deleted_at 在既有部署留著也無害（它是複合索引的
	//        前綴），新安裝不再建立。
	//
	//     b) forum_reports (target_type, target_id)：硬刪貼文時用 JOIN 刪除
	//        「指向該貼文留言」的檢舉工單（見 purge.deletePosts）。唯一鍵
	//        (reporter_email, target_type, target_id) 的前綴是檢舉者，查不到
	//        「某個目標被誰檢舉」，因此需要這條以目標開頭的索引。
	//
	//     c) forum_posts.image_url 與 forum_profiles.avatar_url：圖片引用檢查
	//        （「這個檔案還被誰引用？」）。等值比對可以用索引；前綴長度 64
	//        足以涵蓋檔名（UUID v4 + 副檔名，約 40 個字元），而 utf8mb4 下
	//        64 個字元是 256 bytes，遠低於 InnoDB 的 3072 bytes 上限。
	//
	//     刻意不加 forum_posts.content 的索引：全文檢索本來就是全表掃描，
	//     而且那條路徑（ES 或 MySQL LIKE）用不上 B-tree 索引。
	if err := addIndexIfNotExists(db, "forum_posts", "idx_forum_posts_deleted_at_id", "deleted_at, id"); err != nil {
		return err
	}
	if err := addIndexIfNotExists(db, "forum_reports", "idx_forum_reports_target", "target_type, target_id"); err != nil {
		return err
	}
	if err := addIndexIfNotExists(db, "forum_posts", "idx_forum_posts_image_url", "image_url(64)"); err != nil {
		return err
	}
	if err := addIndexIfNotExists(db, "forum_profiles", "idx_forum_profiles_avatar_url", "avatar_url(64)"); err != nil {
		return err
	}

	// 37) 圖片刪除佇列（forum_media_deletions）。
	//
	//     為什麼需要：清理排程硬刪貼文後，附圖的檔案刪除發生在交易之外。
	//     檔案伺服器可能失敗，程式也可能在 Commit 與刪檔之間崩潰；此時貼文
	//     已經不存在，下一次 Prune 不會再看到那些檔名。若只記一筆日誌，那個
	//     檔案就永遠不會被回收。這張表把「待刪除的檔名」與貼文交易寫在一起：
	//     交易成功 = 佇列一定有紀錄，重試因此是安全的。
	//
	//     filename 是主鍵：同一個檔名被多篇貼文引用時只需要刪一次。
	//     created_at 只用來排序（先進先出），並有索引支援 LIMIT 批次。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_media_deletions (
			filename VARCHAR(128) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (filename),
			KEY idx_forum_media_deletions_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 38) 主鍵改用雪花 ID。
	//
	//     為什麼要做成遷移而不是只改 CREATE TABLE：CREATE TABLE IF NOT EXISTS
	//     對已存在的表什麼都不做，而既有的表仍然帶著 AUTO_INCREMENT。若只改
	//     建表語句，新環境用雪花 id、舊環境用自增 id，兩種行為的差異會在下一次
	//     「新舊資料混在一起」時浮現，而不是在部署時。
	//
	//     這一步做兩件事，順序不能顛倒：
	//       a) 把既有資料重新編號成雪花 id，並同步所有指向它們的欄位。
	//       b) 移除 id 欄位的 AUTO_INCREMENT。
	//     先移除再改資料的話，應用層還沒改完的 INSERT（滾動部署的舊實例）
	//     會因為缺少 id 而失敗；先改資料則那段時間的寫入仍然可用。
	//
	//     既有資料的重新編號是使用者明確要求的：新舊 id 混存會讓「id 排序等於
	//     時間排序」在兩套 id 的邊界上失真（舊 id 很小、新 id 很大，順序雖然
	//     正確，但兩者的間隔沒有任何意義）。重新編號讓整張表回到同一個序列。
	//     代價是 id 會變，因此所有外部引用（ES 索引、書籤、分享連結）都必須
	//     重建或失效 —— ES 在啟動時會全量重建，舊文件由定期對帳清除；外部
	//     書籤則無法補救，這是這個選擇的已知成本。
	//
	//     冪等與並行：偵測「id 是否仍是 AUTO_INCREMENT」決定要不要做，並以
	//     MySQL 的 named lock（GET_LOCK）把整個步驟序列化。沒有那把鎖時，
	//     兩個實例同時啟動會各自讀到舊 id、各自產生新 id，後者覆蓋前者的
	//     對應關係，子表的引用就會指向不存在的 id。
	if err := migrateToSnowflakeIDs(db); err != nil {
		return fmt.Errorf("snowflake id migration: %w", err)
	}

	// 39) 公開識別碼的第二種形態：forum_profiles.public_key_hmac。
	//
	//     為什麼要另一個欄位而不是直接覆寫 public_key：public_key 裡存的是舊的
	//     SHA-256(email)，而舊連結（書籤、外部分享、舊版前端留下的網址）用的就是
	//     那個值。覆寫它等於讓所有舊連結一次性失效；兩欄位並存讓「目前的形態」
	//     與「舊連結還能解析」同時成立（查詢端同時比對兩者，見 handleForumPublicProfile）。
	//
	//     值由應用層派生（HMAC(secret, email)），不是 SQL 的 SHA2 —— HMAC 需要
	//     部署秘密，而秘密不該出現在 SQL 字串裡（它會進日誌與 general log）。
	//     因此這一步只加欄位與索引，回填由 data.BackfillProfileAuthorKeys 在啟動時做。
	//     未設定秘密時該欄位保持空字串：空字串永遠不會被 64 字元的 key 命中，
	//     所以「沒秘密」的部署行為完全等同第 16 步的舊形態。
	if err := addColumnIfNotExists(db, "forum_profiles", "public_key_hmac", "VARCHAR(64) NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	//     索引的理由與第 20 步相同：/api/forum/public-profile 與 /api/forum/public-posts
	//     是不限流的公開端點，以這欄位為條件的查詢必須走索引。非唯一（同第 20 步的
	//     取捨：既有資料若出現重複值，UNIQUE 會讓整個啟動失敗）。
	if err := addIndexIfNotExists(db, "forum_profiles",
		"idx_forum_profiles_public_key_hmac", "public_key_hmac"); err != nil {
		return err
	}

	return nil
}

// authorKeyBackfillBatch 是回填公開識別碼時每批處理的列數。
//
// 分批是因為 forum_profiles 可能有數萬列，而每一列都要在應用層算一次 HMAC：
// 一次載入全部 email 會讓記憶體用量與單次請求的體積都失去上界。
const authorKeyBackfillBatch = 500

// BackfillProfileAuthorKeys 把 forum_profiles.public_key_hmac 補成「目前的派生方式」
// 應有的值（第 39 步只加欄位，值不在 SQL 裡算）。
//
// keyFor 是唯一的派生來源，由呼叫端傳入（main.go 以 httpapi.DeriveAuthorKey 組出閉包）。
// 本套件刻意不知道秘密是什麼：秘密不進 SQL 字串（它會進 MySQL 的 general log 與
// 日誌），data 也不需要同步依賴 config 或 httpapi。
//
// keyFor == nil 代表「尚未設定秘密」，本函式不做任何事 —— 那種部署的公開識別碼
// 本來就是第 16 步的 SHA-256，欄位保持空字串是正確狀態。
//
// 冪等：只動 public_key_hmac = '' 的列。已回填的列不重算，所以換秘密不會被
// 這一步靜默覆蓋（換秘密時既有 key 會失效，那是部署者明示的代價，舊連結仍由
// public_key 欄位解析）。每批處理後該批的列不再符合條件，因此迴圈自然終止；
// 上限是「表大小 / 批次」次往返，只在首次啟動或手動清空後發生。
func BackfillProfileAuthorKeys(ctx context.Context, db *sql.DB, keyFor func(email string) string) error {
	if db == nil {
		return errors.New("nil database")
	}
	if keyFor == nil {
		return nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		rows, err := db.QueryContext(ctx,
			`SELECT author_email FROM forum_profiles WHERE public_key_hmac = '' LIMIT ?`, authorKeyBackfillBatch)
		if err != nil {
			return fmt.Errorf("read profiles without author key: %w", err)
		}
		emails := make([]string, 0, authorKeyBackfillBatch)
		for rows.Next() {
			var email string
			if err := rows.Scan(&email); err != nil {
				rows.Close()
				return fmt.Errorf("scan profile email: %w", err)
			}
			emails = append(emails, email)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("scan profile emails: %w", err)
		}
		rows.Close()

		if len(emails) == 0 {
			return nil
		}

		// 預言式：同一條 UPDATE 語句對每批的列重複使用，省掉每列一次 prepare。
		stmt, err := db.PrepareContext(ctx,
			`UPDATE forum_profiles SET public_key_hmac = ? WHERE author_email = ? AND public_key_hmac = ''`)
		if err != nil {
			return fmt.Errorf("prepare author key update: %w", err)
		}
		for _, email := range emails {
			key := keyFor(email)
			if key == "" {
				// 派生回空字串只可能是 keyFor 實作的 bug（HMAC 永遠 64 字元）；
				// 寧可停下來也不要寫進一個「查不到的空值」。錯誤訊息刻意不含
				// email：那會跟著日誌把個資寫出去（見 docs/SECURITY.md 的日誌界限）。
				stmt.Close()
				return errors.New("derive author key returned empty value")
			}
			if _, err := stmt.ExecContext(ctx, key, email); err != nil {
				stmt.Close()
				return fmt.Errorf("write author key: %w", err)
			}
		}
		stmt.Close()

		if len(emails) < authorKeyBackfillBatch {
			return nil
		}
	}
}

// snowflakeMigrationLock 是重新編號時持有的 MySQL named lock。
//
// GET_LOCK 的鎖是**全伺服器共用**的，因此同一台 MySQL 上若有第二個資料庫
// 也在跑這個專案，兩邊的啟動遷移會互相等待。這是刻意的取捨：等待的時間是
// 一次啟動遷移（通常幾百毫秒），而把資料庫名稱接進鎖名會讓名字有機會超過
// MySQL 的 64 字元上限（那會讓遷移直接失敗）。正確性優先於跨資料庫的平行。
const snowflakeMigrationLock = "forum:snowflake_id_migration"

// snowflakeRenameBatch 是重新編號時每批處理的列數。
//
// 不一次載入全部 id 是因為 forum_admin_actions 可能有數十萬列；分批讓記憶體
// 用量有界，而 keyset 分頁（WHERE id > ? ORDER BY id）不需要 OFFSET 的線性掃描。
const snowflakeRenameBatch = 1000

// snowflakeIDTable 描述「一張有 id 主鍵、要改成雪花 id 的表」以及所有指向它的欄位。
//
// references 的順序不重要，但完整性很重要：漏掉一個引用欄位時，遷移會成功，
// 而那個欄位會指向一個不存在的 id —— 症狀是「某則通知點進去 404」，只有在
// 有人實際點到時才會發現。
type snowflakeIDTable struct {
	name       string
	references []snowflakeIDReference
}

// snowflakeIDReference 是一個「存著某張表 id」的欄位。
//
// text 為 true 代表該欄位是 VARCHAR（forum_admin_actions.target_id 就是這種
// 多型別字串），比對與寫入都必須經過 CAST；其餘為 BIGINT。
type snowflakeIDReference struct {
	table  string
	column string
	where  string
	text   bool
}

// snowflakeIDTables 是這次遷移涵蓋的全部表。
//
// 順序有意義：被引用的表先編號。posts 先於 comments（comments.post_id），
// tags 先於 assignments（assignments.tag_id）。自身引用（comments.parent_id）
// 在同一個交易裡先改 id 再改 parent_id，因為 parent_id 仍然存著舊值。
var snowflakeIDTables = []snowflakeIDTable{
	{
		name: "forum_user_tags",
		references: []snowflakeIDReference{
			{table: "forum_user_tag_assignments", column: "tag_id"},
			{table: "forum_admin_actions", column: "target_id", where: "target_type = 'tag'", text: true},
		},
	},
	{
		name: "forum_posts",
		references: []snowflakeIDReference{
			{table: "forum_post_likes", column: "post_id"},
			{table: "forum_post_comments", column: "post_id"},
			{table: "forum_post_bookmarks", column: "post_id"},
			{table: "forum_notifications", column: "post_id"},
			{table: "forum_reports", column: "target_id", where: "target_type = 'post'"},
			{table: "forum_admin_actions", column: "target_id", where: "target_type = 'post'", text: true},
		},
	},
	{
		name: "forum_post_comments",
		references: []snowflakeIDReference{
			// 自身引用：先更新 id，再讓 parent_id 走同一份對應表。
			{table: "forum_post_comments", column: "parent_id"},
			{table: "forum_notifications", column: "comment_id"},
			{table: "forum_reports", column: "target_id", where: "target_type = 'comment'"},
			{table: "forum_admin_actions", column: "target_id", where: "target_type = 'comment'", text: true},
		},
	},
	{
		name: "forum_reports",
		references: []snowflakeIDReference{
			{table: "forum_admin_actions", column: "target_id", where: "target_type = 'report'", text: true},
		},
	},
	{
		name: "forum_announcements",
		references: []snowflakeIDReference{
			{table: "forum_admin_actions", column: "target_id", where: "target_type = 'announcement'", text: true},
		},
	},
	// 通知的 id 沒有任何欄位指向它，但稽核紀錄可能把通知當成 target
	// （目前沒有這種動作）。保留在清單裡是因為「這張表有 id」本身就是
	// 納入的理由；漏掉它會讓新舊 id 混存。
	{name: "forum_notifications"},
	// 稽核紀錄本身沒有任何欄位指向它（target_id 指的是被操作的對象，
	// 不是另一筆稽核紀錄）。
	{name: "forum_admin_actions"},
}

// migrateToSnowflakeIDs 把既有自增主鍵重新編號為雪花 id，並移除 AUTO_INCREMENT。
//
// 只在「id 欄位仍然是 AUTO_INCREMENT」時動作，因此：
//   - 全新資料庫（建表時就沒有 AUTO_INCREMENT）什麼都不做。
//   - 已經遷移過的資料庫什麼都不做。
//   - 中途失敗的資料庫會從還沒完成的表繼續（每張表各自一個交易）。
//
// 並行：整個步驟在同一個 MySQL named lock 下執行。兩個實例同時啟動時，
// 後者會等前者完成，然後看到「已經不是 AUTO_INCREMENT」而跳過。
func migrateToSnowflakeIDs(db *sql.DB) error {
	ctx := context.Background()

	// 一定要用單一 *sql.Conn：GET_LOCK 是連線層級的，若走 db.Exec，鎖會留在
	// 池裡的某條連線上，而後續的語句可能被分派到別條連線，等於沒有鎖。
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 等待上限刻意給得比一般啟動遷移長：大站（例如數十萬筆稽核紀錄）的
	// 重新編號可能超過一分鐘，而第二個實例若在 30 秒就放棄，症狀會是
	// 「滾動部署期間新實例一直重啟」，直到第一個實例做完為止。
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, 300)`, snowflakeMigrationLock).Scan(&acquired); err != nil {
		return fmt.Errorf("取得遷移鎖失敗: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return fmt.Errorf("等待遷移鎖逾時（另一個實例正在重新編號？）")
	}
	// RELEASE_LOCK 用 DO 而不是 SELECT：DO 不產生結果集，不需要 Scan。
	// 鎖也會在連線關閉時自動釋放，這裡只是提早交還。
	defer conn.ExecContext(ctx, `DO RELEASE_LOCK(?)`, snowflakeMigrationLock)

	for _, table := range snowflakeIDTables {
		auto, err := tableUsesAutoIncrement(ctx, conn, table.name)
		if err != nil {
			return fmt.Errorf("%s: 探測 AUTO_INCREMENT 失敗: %w", table.name, err)
		}
		if !auto {
			continue
		}
		if err := renumberTable(ctx, conn, table); err != nil {
			return fmt.Errorf("%s: 重新編號失敗: %w", table.name, err)
		}
		if err := dropAutoIncrement(ctx, conn, table.name); err != nil {
			return fmt.Errorf("%s: 移除 AUTO_INCREMENT 失敗: %w", table.name, err)
		}
	}
	return nil
}

// tableUsesAutoIncrement 回報 id 欄位是否仍帶 AUTO_INCREMENT。
func tableUsesAutoIncrement(ctx context.Context, conn *sql.Conn, table string) (bool, error) {
	var extra string
	err := conn.QueryRowContext(ctx, `
		SELECT EXTRA FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = 'id'
	`, table).Scan(&extra)
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(extra), "auto_increment"), nil
}

// rowQuerier 是 QueryRowContext 的最小介面。*sql.Conn 與 *sql.Tx 都實作它，
// 因此同一份探測可以在「那條連線」與「那條連線上的交易」裡使用 —— 重新編號
// 的暫存表是連線層級的，探測必須與它在同一條連線（與同一個交易）裡。
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// columnCollation 回報一個文字欄位的 collation 與它所屬的 charset。
// 該欄位沒有 collation（數值或 BLOB）時回兩個空字串。
//
// 為什麼重新編號需要它：MySQL 8 對「DEFAULT CHARSET=utf8mb4 但沒寫 COLLATE」的表
// 用的是 **charset 的預設 collation**（utf8mb4_0900_ai_ci），不是資料庫的 collation；
// 而 `CAST(m.old_id AS CHAR)` 帶著連線的 collation（utf8mb4_general_ci）。兩邊都是
// IMPLICIT，MySQL 找不到共同的 collation，因此回
// Error 1267 (HY000): Illegal mix of collations ... for operation '='。
// 這是第 38 步（雪花 id 重新編號）在 CI 的 mysql:8.0.29 上的失敗；本機的開發資料庫
// 是 MariaDB，那裡 utf8mb4 的預設 collation 正好就是 utf8mb4_general_ci，兩者相同，
// 因此這個錯誤在本機出不來（見 migration_test.go 的 collation 測試）。
//
// 比對被換成「欄位自己的 collation」，而不是隨意挑一個：
//   - 顯式的 COLLATE 壓過欄位的 IMPLICIT collation，兩側因此一致，1267 消失。
//   - 比對用的 collation 與索引的 collation 相同，因此
//     idx_forum_admin_actions_target (target_type, target_id) 仍然能被 JOIN 用到。
//     兩側都強制 utf8mb4_general_ci 也能消除 1267，但索引必然用不上，
//     一個有數十萬筆稽核紀錄的站會變成「遷移跑不完」。
//   - 欄位的 collation 不被改寫：那是 DDL，會改變該欄位的排序與大小寫語意，
//     而且這個專案沒有降級路徑。
//   - COLLATE 只能套在它所屬 charset 的運算式上，所以 CAST 的結果要先 CONVERT
//     到那個 charset。charset 由 collation 名稱的前綴推得，而不是探測另一個欄位：
//     MySQL 的 information_schema 叫它 CHARACTER_SET_NAME，MariaDB 叫它
//     CHARSET_NAME，同一句 SQL 無法同時相容兩個名字；而 collation 名稱在兩種
//     伺服器都叫 COLLATION_NAME。
func columnCollation(ctx context.Context, q rowQuerier, table, column string) (collation, charset string, err error) {
	var value sql.NullString
	// table / column 是本檔的字面值（與 DDL 一樣組接字串，探測的 DDL 無法參數化），
	// 因此「哪個欄位的探測掛了」會直接出現在錯誤輸出裡，理由同 addColumnIfNotExists。
	if err := q.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COLLATION_NAME FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s' AND COLUMN_NAME = '%s'
	`, table, column)).Scan(&value); err != nil {
		return "", "", fmt.Errorf("探測 %s.%s 的 collation 失敗: %w", table, column, err)
	}
	if !value.Valid || value.String == "" {
		// 沒有 collation：原本的比對就是對的，不必強制。
		return "", "", nil
	}
	charset = collationCharset(value.String)
	// 這兩個名稱會被組進 SQL。它們來自伺服器自己的 information_schema，不是使用者
	// 輸入，但「不尋常的名稱」在這裡值得直接拒絕：拼出一句語法錯誤的 UPDATE 比
	// 回一個指明是哪個 collation 的錯誤更難查。
	if !isSQLName(value.String) || !isSQLName(charset) {
		return "", "", fmt.Errorf("%s.%s 的 collation 名稱不尋常: %q", table, column, value.String)
	}
	return value.String, charset, nil
}

// collationCharset 推得一個 collation 所屬的 charset。MySQL 與 MariaDB 的 collation
// 名稱都是「<charset>_...」；唯一的例外是 binary（它沒有前綴）。
func collationCharset(collation string) string {
	if i := strings.IndexByte(collation, '_'); i > 0 {
		return collation[:i]
	}
	return collation
}

// isSQLName 判斷一個 charset / collation 名稱是否只由字母、數字與底線組成。
func isSQLName(name string) bool {
	for _, r := range name {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return name != ""
}

// renumberTable 在一個交易內把一張表的 id 換成新的雪花 id，並同步所有引用。
//
// 交易的範圍刻意是「一張表」而不是「全部表」：過程中會 UPDATE 使用者的資料，
// 一次交易鎖住七張表會讓停機時間與日誌大小成正比；一張表一個交易則讓失敗時
// 只留下「這張表已完成、下一張還沒」的狀態，而那樣的狀態是下一步可以接續的。
func renumberTable(ctx context.Context, conn *sql.Conn, table snowflakeIDTable) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 暫存表是連線層級的；用 IF EXISTS 清掉上一次中斷留下的同名表。
	// DROP TEMPORARY TABLE 與 CREATE TEMPORARY TABLE 都不會隱含提交，
	// 因此它們可以安全地待在這個交易裡。
	if _, err := tx.ExecContext(ctx, `DROP TEMPORARY TABLE IF EXISTS _snowflake_id_map`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE TEMPORARY TABLE _snowflake_id_map (
			old_id BIGINT NOT NULL PRIMARY KEY,
			new_id BIGINT NOT NULL
		) ENGINE=InnoDB
	`); err != nil {
		return err
	}

	// 依舊 id 遞增產生新 id。產生的順序就是新 id 的順序，因此原本的
	// 「先建立的文章 id 較小」在遷移後仍然成立。
	var lastID int64
	for {
		rows, err := tx.QueryContext(ctx,
			"SELECT id FROM "+table.name+" WHERE id > ? ORDER BY id LIMIT ?", lastID, snowflakeRenameBatch)
		if err != nil {
			return err
		}
		ids := make([]int64, 0, snowflakeRenameBatch)
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if len(ids) == 0 {
			break
		}

		placeholders := make([]string, 0, len(ids))
		args := make([]any, 0, len(ids)*2)
		for _, oldID := range ids {
			placeholders = append(placeholders, "(?, ?)")
			args = append(args, oldID, idgen.Next())
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO _snowflake_id_map (old_id, new_id) VALUES "+strings.Join(placeholders, ", "),
			args...); err != nil {
			return err
		}
		lastID = ids[len(ids)-1]
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE "+table.name+" t JOIN _snowflake_id_map m ON t.id = m.old_id SET t.id = m.new_id"); err != nil {
		return err
	}

	// 接著更新所有指向這張表的欄位。每一句都在同一個交易裡，因此
	// 「id 換了但子表還在指舊值」的時間窗口不存在。
	for _, ref := range table.references {
		// 自身引用（comments.parent_id）必須在 id 更新之後才做：此時
		// parent_id 還是舊值，而 id 已經是新的，兩者不會互相干擾。
		query := "UPDATE " + ref.table + " t JOIN _snowflake_id_map m ON "
		if ref.text {
			// 文字欄位的比對必須兩側同一 collation（見 columnCollation）：
			// CAST 帶著連線的 collation，欄位帶著自己的，MySQL 8 對這種組合回 1267。
			collation, charset, err := columnCollation(ctx, tx, ref.table, ref.column)
			if err != nil {
				return err
			}
			oldID := "CAST(m.old_id AS CHAR)"
			if collation != "" {
				oldID = fmt.Sprintf("CONVERT(%s USING %s) COLLATE %s", oldID, charset, collation)
			}
			// SET 側不強制 collation：寫入是轉換，不是比對，沒有 1267 的問題。
			query += "t." + ref.column + " = " + oldID + " SET t." + ref.column + " = CAST(m.new_id AS CHAR)"
		} else {
			query += "t." + ref.column + " = m.old_id SET t." + ref.column + " = m.new_id"
		}
		if ref.where != "" {
			query += " WHERE " + ref.where
		}
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("更新 %s.%s 失敗: %w", ref.table, ref.column, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	// 暫存表是連線層級的：不主動清掉，它會跟著連線回到池裡佔用記憶體，
	// 直到那條連線被關閉。DROP TEMPORARY TABLE 不會隱含提交，而這裡
	// 已經在交易之外。
	_, err = conn.ExecContext(ctx, `DROP TEMPORARY TABLE IF EXISTS _snowflake_id_map`)
	return err
}

// dropAutoIncrement 移除 id 欄位的 AUTO_INCREMENT，保留原本的型別（含 unsigned）。
//
// 不用寫死 BIGINT：這個專案的建表語句是 BIGINT，但舊環境可能是 BIGINT UNSIGNED
// （早期的測試 schema 就是）。MODIFY 時若把 unsigned 拿掉，MySQL 會重建整張表，
// 而那次重建是不必要的。
func dropAutoIncrement(ctx context.Context, conn *sql.Conn, table string) error {
	var columnType string
	if err := conn.QueryRowContext(ctx, `
		SELECT COLUMN_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = 'id'
	`, table).Scan(&columnType); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, "ALTER TABLE "+table+" MODIFY COLUMN id "+columnType+" NOT NULL")
	return err
}
