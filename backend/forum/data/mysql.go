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
	forum_post_comments          文章留言：id / post_id / author_email / content / created_at
	forum_reports                檢舉工單（可指向文章或留言），含處理狀態與覆核者
	forum_profiles               使用者公開資料：author_email 為主鍵、public_key 為 email 的 SHA-256、avatar_url 只存檔名
	forum_users                  使用者帳號主檔與停權狀態（status = ACTIVE / SUSPENDED）
	forum_user_tags              標籤字典
	forum_user_tag_assignments   使用者與標籤的多對多關聯
	forum_follows                追蹤關聯，以 (follower_email, target_email) 複合主鍵表示「一人追一人一次」
	forum_request_metrics        分鐘級請求統計（監控頁的持久化來源；純衍生資料，可整表刪除）
	forum_admin_actions           管理員操作稽核紀錄（欄位級 diff；只能由時間清理，不可透過 API 刪除）
	forum_announcements           站內公告（同時只有一列 active；expires_at 可空＝永不自動過期）

關鍵設計決策

 1. 遷移刻意「線性執行」而非包在交易（Transaction）內。MySQL 的 DDL 本身具備隱含提交
    （implicit commit）語意，CREATE TABLE / ALTER TABLE 無法被 ROLLBACK，一旦包進交易只會
    製造「以為有原子性、實際沒有」的假象。加上 DDL 在 MySQL 中會取得隱式鎖，線性執行反而能
    避免兩個實例同時啟動時互相鎖住 metadata 而死鎖。代價是遷移中途失敗會留下半套 schema，
    但因為每一步都是冪等的，下一次啟動會自動補齊剩餘步驟。
 2. 所有 DDL 儘量使用 IF NOT EXISTS 語法，讓 MigrateMySQL 可以安全地重複執行（啟動時必定會
    執行一次）。仍舊需要 information_schema 探測的步驟，是因為舊版 MySQL 沒有
    DROP COLUMN IF EXISTS / 唯一索引「不存在才建立」這類語法。
 3. 全部資料表統一 ENGINE=InnoDB（交易與外鍵語意）與 DEFAULT CHARSET=utf8mb4。utf8mb4 是
    完整 Unicode（含 emoji 與 4-byte supplementary 平面），論壇內文屬使用者輸入，用舊的 utf8
    （MySQL 的 3-byte utf8mb3）會在寫入時直接報錯。
 4. 已知風險：schema 版本漂移。本檔案沒有版本號或 golang-migrate 這類版本表，因此無法偵測
    「某個欄位被人工改過」或「欄位型別與預期不符」；CREATE TABLE IF NOT EXISTS 只會在表
    不存在時建立，已存在的表其欄位差異必須靠後面逐一補的 ALTER 與人工介入處理。
*/
package data

import (
	"database/sql"
	"fmt"
	"time"

	// 以底線匯入（blank import）只為了註冊 "mysql" 這個 database/sql 驅動名稱；
	// 實際連線行為完全由下面傳入的 DSN 字串決定。
	_ "github.com/go-sql-driver/mysql"
)

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
// 冪等性來源：CREATE TABLE IF NOT EXISTS、ALTER TABLE ... ADD COLUMN IF NOT EXISTS、
// INSERT IGNORE，以及針對「舊版欄位是否存在」與「索引是否存在」的 information_schema 探測。
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
	//      id            – AUTO_INCREMENT 主鍵，前端使用的文章識別碼
	//      author_email  – 發文者 email，作為與 forum_users / forum_profiles 的關聯鍵；
	//                      長度 255 是為了容納較長的 email 字串
	//      content       – 純文字內容，TEXT 而非 VARCHAR，因內文無固定長度上限
	//      created_at    – 由應用層寫入 time.Now()（非資料庫 NOW()），時區為程式主機的本地時區
	//    索引理由：idx_forum_posts_created_at 服務於「依 created_at DESC 排序 + LIMIT/OFFSET
	//    分頁」這條主查詢（listForumPosts / 管理員文章列表），避免全表排序後再切頁。
	//    注意排序實際為 created_at DESC, id DESC，InnoDB 隱含的主鍵順序可協助 id 的穩定排序。
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_posts (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			author_email VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			INDEX idx_forum_posts_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		return err
	}

	// 2) image_url 為後來新增的需求（先發文、圖片由獨立的檔案伺服器保管），
	//    因此以 ADD COLUMN IF NOT EXISTS 追加，而非改建表。
	//    選 TEXT 而非 VARCHAR：沿用當初「存完整 URL、長度不可預期」的型別，
	//    即使現在只存檔名也不改——改欄位型別是 DDL、失去回滾機會，
	//    對一個上限 128 字元的值並不划算。
	//    儲存契約：只放檔名（如 "9f8c1a.png"），完整網址由應用層以
	//    FILES_SERVER_PUBLIC_URL 組成；舊資料的轉換見第 18 步。
	//    宣告 NOT NULL 卻未給 DEFAULT，代表所有既有列在 MySQL 嚴格模式下會被填入空字串
	//    （DDL 會以列層級的隱式默认值補齊），因此 createForumPost 仍需顯式傳入值。
	if _, err := db.Exec(`
		ALTER TABLE forum_posts ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL;
	`); err != nil {
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
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forum_post_comments (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
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
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
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
	var authorEmailColumnCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_post_likes' AND COLUMN_NAME = 'author_email'
	`).Scan(&authorEmailColumnCount); err != nil {
		return err
	}
	if authorEmailColumnCount == 0 {
		// NOT NULL 欄位在已有資料的表上新增時必須提供 DEFAULT ''，否則 MySQL 會以
		// strict mode 直接讓整句 DDL 失敗（Error 1364）。
		if _, err := db.Exec(`ALTER TABLE forum_post_likes ADD COLUMN author_email VARCHAR(255) NOT NULL DEFAULT ''`); err != nil {
			return err
		}
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
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
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
	if _, err := db.Exec(`
		ALTER TABLE forum_profiles ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL;
	`); err != nil {
		return err
	}

	// 15) public_key 補欄位（新增需求：訪客以雜湊值查公開頁面）。
	if _, err := db.Exec(`
		ALTER TABLE forum_profiles ADD COLUMN IF NOT EXISTS public_key VARCHAR(64) NOT NULL DEFAULT '';
	`); err != nil {
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
	//     為什麼需要探測：MySQL 沒有「ADD UNIQUE INDEX IF NOT EXISTS」語法，重複執行會得到
	//     Error 1061（重複的 key 名稱）。information_schema.STATISTICS 是索引的中繼資料表，
	//     以 INDEX_NAME 比對存在與否即可安全地做到冪等。
	//     為什麼要這個約束：公開頁面以 nickname 顯示作者，重複暱稱會讓身分無法辨識；
	//     加上唯一索引後，競態下的重複寫入會由資料庫擋下（1062），handler 據此回 409。
	//     已知風險：若既有資料本身就有重複暱稱，這一步會讓整個遷移失敗並中止啟動，
	//     必須先人工清理重複值 —— 這是刻意選擇「寧可啟動失敗也不靜默產生重複暱稱」。
	var nicknameIndexCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_profiles' AND INDEX_NAME = 'uq_forum_profiles_nickname'
	`).Scan(&nicknameIndexCount); err != nil {
		return err
	}
	if nicknameIndexCount == 0 {
		// 注意：PRIMARY KEY 之外的 UNIQUE INDEX 在 MySQL 中無法用 IF NOT EXISTS 建立，
		// 因此上方的 information_schema 探測是不可省略的前置步驟。
		if _, err := db.Exec(`
			ALTER TABLE forum_profiles ADD UNIQUE INDEX uq_forum_profiles_nickname (nickname)
		`); err != nil {
			return err
		}
	}

	// 18) 資料遷移：forum_posts.image_url 由「完整網址」改為「純檔名」。
	//     為什麼需要：此欄過去存的是檔案伺服器回傳的完整網址，導致換對外網域
	//     必須 UPDATE 全表，而且內部位址（http://192.168.66.5:7070/…）會被寫進
	//     資料庫。新契約只存檔名，完整網址由應用層以 FILES_SERVER_PUBLIC_URL 組出。
	//     作法：先以 SUBSTRING_INDEX(…, '?', 1) 去掉 ?token=…，再取最後一段路徑。
	//     WHERE image_url LIKE '%/%' 是冪等條件：轉換後的值不含 "/"，因此
	//     第二次啟動不會再動到任何一列。含 "?token=" 但無路徑者不可能出現
	//     （token 一定接在 /files/<檔名> 之後），故不另設條件。
	//     已知限制：此敘述只做字串切割，不驗證主機。舊資料若存的是第三方
	//     圖片網址，切完會得到一個指向自己檔案伺服器、但實際不存在的檔名，
	//     讀取時回空字串（前端不顯示縮圖）。應用層的 forumImageFileName
	//     會容忍尚未遷移的舊寫法，因此兩者不必同步進行。
	if _, err := db.Exec(`
		UPDATE forum_posts
		SET image_url = SUBSTRING_INDEX(SUBSTRING_INDEX(image_url, '?', 1), '/', -1)
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
	//     探測寫法照第 17 步：MySQL 沒有 ADD INDEX IF NOT EXISTS，重複執行會得到
	//     Error 1061（重複的 key 名稱）。
	var publicKeyIndexCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_profiles' AND INDEX_NAME = 'idx_forum_profiles_public_key'
	`).Scan(&publicKeyIndexCount); err != nil {
		return err
	}
	if publicKeyIndexCount == 0 {
		if _, err := db.Exec(`
			ALTER TABLE forum_profiles ADD INDEX idx_forum_profiles_public_key (public_key)
		`); err != nil {
			return err
		}
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
	//     探測寫法與第 20 步相同（MySQL 沒有 ADD INDEX IF NOT EXISTS，重複執行
	//     會得到 1061）。非唯一索引：這個欄位本來就有重複值，而且我們要的只是
	//     「等值過濾 + 前綴去重」，唯一性對這兩件事都沒有幫助。
	var postsAuthorIndexCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_posts' AND INDEX_NAME = 'idx_forum_posts_author_email'
	`).Scan(&postsAuthorIndexCount); err != nil {
		return err
	}
	if postsAuthorIndexCount == 0 {
		if _, err := db.Exec(`
			ALTER TABLE forum_posts ADD INDEX idx_forum_posts_author_email (author_email)
		`); err != nil {
			return err
		}
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
			id BIGINT NOT NULL AUTO_INCREMENT,
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
	//     探測寫法照第 17、20、21 步：MySQL 沒有「ADD INDEX IF NOT EXISTS」，
	//     重複執行同一句會得到 Error 1061（重複的 key 名稱），而整個
	//     MigrateMySQL 的回傳錯誤會讓主流程 Fatal —— 症狀是「第一次啟動成功、
	//     第二次之後每次啟動都死」，且錯誤訊息只有一個索引名，極難聯想到是
	//     遷移本身不冪等。因此索引一律先以 information_schema 確認不存在才建立。
	var usersCreatedAtIndexCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_users' AND INDEX_NAME = 'idx_forum_users_created_at'
	`).Scan(&usersCreatedAtIndexCount); err != nil {
		return err
	}
	if usersCreatedAtIndexCount == 0 {
		if _, err := db.Exec(`
			ALTER TABLE forum_users ADD INDEX idx_forum_users_created_at (created_at)
		`); err != nil {
			return err
		}
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
			id BIGINT NOT NULL AUTO_INCREMENT,
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
	//     欄位與索引拆成兩句：欄位用 ADD COLUMN IF NOT EXISTS（同第 14、15 步），
	//     索引則先探測 information_schema —— MySQL 沒有 ADD INDEX IF NOT EXISTS，
	//     兩者混在一句 ALTER 裡的話，第二次啟動會因重複的 key 名稱（1061）
	//     讓整個遷移失敗，而主流程會因此 Fatal。
	if _, err := db.Exec(`
		ALTER TABLE forum_posts
			ADD COLUMN IF NOT EXISTS pinned TINYINT NOT NULL DEFAULT 0 AFTER image_url;
	`); err != nil {
		return err
	}
	var postsFeedIndexCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'forum_posts' AND INDEX_NAME = 'idx_forum_posts_feed'
	`).Scan(&postsFeedIndexCount); err != nil {
		return err
	}
	if postsFeedIndexCount == 0 {
		if _, err := db.Exec(`
			ALTER TABLE forum_posts ADD INDEX idx_forum_posts_feed (pinned DESC, created_at DESC, id DESC)
		`); err != nil {
			return err
		}
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
	//     探測寫法照第 17、20、21、24、26 步：MySQL 沒有 ADD INDEX IF NOT EXISTS，
	//     重複執行會得到 Error 1061 而讓整個 MigrateMySQL 失敗（主流程 Fatal）。
	//
	//     注意 b) 的索引成本：每新增一筆按讚多寫一個 B-tree 節點。按讚是
	//     這個站最熱的寫入，但它的寫入量遠小於 forum_request_metrics 那類
	//     分鐘彙總的寫入量，所以這個取捨仍然划算。
	//
	//     a) 的兩個索引成本同理但更低：forum_posts 的寫入量遠低於按讚，
	//     而 author_email 是單欄索引（B-tree 節點數 = 該欄相異值數 × 深度），
	//     不隨該作者的貼文數成長。
	addIndexIfMissing := func(table, index, columns string) error {
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
		_, err := db.Exec("ALTER TABLE " + table + " ADD INDEX " + index + " (" + columns + ")")
		return err
	}
	if err := addIndexIfMissing("forum_post_comments",
		"idx_forum_post_comments_author_email", "author_email"); err != nil {
		return err
	}
	// 保留既有的 idx_forum_posts_created_at（搜尋結果、profile 貼文列表與
	// 管理端的多處查詢只按 created_at 排序），這一個是為了 author_email
	// 等值查詢而存在 —— 兩者服務不同的存取樣式，不是同一件事的兩種寫法。
	if err := addIndexIfMissing("forum_posts",
		"idx_forum_posts_author_email", "author_email"); err != nil {
		return err
	}
	if err := addIndexIfMissing("forum_post_likes",
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
	if _, err := db.Exec(`
		ALTER TABLE forum_posts
			ADD COLUMN IF NOT EXISTS updated_at DATETIME NULL AFTER pinned;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		ALTER TABLE forum_post_comments
			ADD COLUMN IF NOT EXISTS updated_at DATETIME NULL AFTER created_at;
	`); err != nil {
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
	if _, err := db.Exec(`
		ALTER TABLE forum_profiles ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL;
	`); err != nil {
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
	//     不加索引：唯一會用到它的查詢都是「已經拿著 id 找到那一列」（編輯、
	//     按讚、留言的存在性檢查），走主鍵；而列表端的過濾是與排序欄位一起
	//     出現的附加條件，不是篩選的主體（理由同第 28 步不加索引的判斷）。
	//
	//     資料不會自動消失：軟刪把「從資料庫抹除」的責任從 handler 移了出去，
	//     因此需要一個清理機制才能真正回收空間。當前刻意沒有排程器 ——
	//     貼文量在這個站點的規模下，保留數百筆已刪貼文的成本低於一個會
	//     自動刪資料的排程所帶來的風險。需要時機成熟再加（見
	//     docs/KNOWN_ISSUES.md）。
	if _, err := db.Exec(`
		ALTER TABLE forum_posts
			ADD COLUMN IF NOT EXISTS deleted_at DATETIME NULL AFTER updated_at;
	`); err != nil {
		return err
	}

	return nil
}
