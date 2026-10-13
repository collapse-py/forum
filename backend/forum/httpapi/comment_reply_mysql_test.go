/*
留言子樹刪除的**真實執行**測試（comment_reply_mysql_test.go）。

comment_reply_test.go 釘住 forumCommentSubtreeJoin / …Condition 的**形狀**，
而 shape 斷言天生有一個盲點：MySQL 收不收這句話、刪的到底是不是那幾列。那是
四張 forum_post_comments 別名的多表 DELETE（自我 JOIN），任何一處語法或語意
錯誤都會讓每一次留言刪除回 500 —— 而形狀測試對這件事完全無聲。這個檔因此
只回答一個問題：對真的 MySQL 跑，刪除的列數與名單正確嗎。

【為什麼沒有這個測試之前它也不會被發現】
使用者與後台的兩條刪除路徑都只被 unit test 用形狀覆蓋過；CI 的 migrations job
過去也只跑 ./forum/data/...。一句沒有任何地方執行過的 DELETE，與一段沒有執行
過的程式碼是同一類風險。

【測試需要什麼】
與 forum/data/migration_test.go 同一個約定：環境變數 FORUM_TEST_MYSQL_DSN
指向伺服器（不是特定資料庫），每個測試自己建立與丟棄拋棄式資料庫。未設定時
skip —— 讓沒有 MySQL 的開發者與一般的 backend job 保持全綠。

	FORUM_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/?parseTime=true&loc=UTC&multiStatements=true'

CI 的 migrations job 提供這個服務，並以 -run 'MySQL' 只跑這個檔的測試。
*/
package httpapi

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"forum/forum/data"
)

// subtreeTestCounter 讓每個測試的資料庫名稱都不相同（同 migration_test.go
// 的 dbCounter：同名資料庫的症狀是測試間歇看到彼此的資料）。
var subtreeTestCounter atomic.Int64

// subtreeServerDSN 把 FORUM_TEST_MYSQL_DSN 剝成「只連伺服器」的形式（庫名清空、
// 參數原樣保留），未設定 DSN 時直接 skip。
//
// 剝法與 forum/data/migration_test.go 的 serverDSN 相同，而相同是有原因的：
// 那個函式曾經把問號一起丟掉，於是 parseTime=true&loc=UTC 被當成庫名，每一次
// 測試都以「Unknown database」失敗 —— CI 的 DSN 正是帶參數的那一種。此處兩支
// 分開寫而不是互相 import：跨 package 共用測試輔助函式會把 data 的測試檔
// 變成 httpapi 的編譯依賴（那個檔只在有 MySQL 時才有意義）。
func subtreeServerDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("FORUM_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FORUM_TEST_MYSQL_DSN 未設定：跳過需要真實 MySQL 的子樹刪除測試。" +
			"設定方式見本檔檔頭，或見 .github/workflows/ci.yml 的 migrations job")
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

// subtreeServerDB 連到 MySQL 伺服器層級（不指定資料庫）。
func subtreeServerDB(t *testing.T) *sql.DB {
	t.Helper()

	cleaned := subtreeServerDSN(t)
	db, err := sql.Open("mysql", cleaned)
	if err != nil {
		t.Fatalf("無法以 %q 開啟連線: %v", cleaned, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("MySQL 連不上（檢查 FORUM_TEST_MYSQL_DSN）: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// subtreeTempDatabase 建立拋棄式資料庫、跑完整 MigrateMySQL，並回傳連到它的 db。
//
// 跑完整遷移而不是只建兩張表：MigrateMySQL 第 31 步的 parent_id / depth 兩個
// ALTER 正是這個功能的一部分，只建表會讓「兩欄沒補上」這種迴歸測不出來。
func subtreeTempDatabase(t *testing.T) *sql.DB {
	t.Helper()

	server := subtreeServerDB(t)
	name := fmt.Sprintf("forum_subtree_test_%d_%d", time.Now().UnixNano(), subtreeTestCounter.Add(1))
	if _, err := server.Exec(fmt.Sprintf(
		"CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", name)); err != nil {
		t.Fatalf("建立測試資料庫 %s 失敗: %v", name, err)
	}
	t.Cleanup(func() {
		if _, err := server.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name)); err != nil {
			t.Logf("清理測試資料庫 %s 失敗（請手動清理）: %v", name, err)
		}
	})

	// 從剝成伺服器層級的 DSN 組出「連到這個拋棄式資料庫」的 DSN，而不是
	// 對原始 DSN 做字串替換：開發者可能把 DSN 指向真正的論壇庫
	//（root@…/forum?…），那時替換不會發生，MigrateMySQL 會直接在正式資料庫
	// 上建表。從剝好的 server DSN 重建，兩種形狀都落在拋棄式資料庫上。
	serverLevel := subtreeServerDSN(t)
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

	if err := data.MigrateMySQL(db); err != nil {
		t.Fatalf("MigrateMySQL 在 %s 上失敗: %v", name, err)
	}
	return db
}

// subtreeComment 是測試輔助結構：一則留言在資料庫中的位置。
type subtreeComment struct {
	id       int64
	parentID int64
	depth    int
	author   string
}

// insertSubtreePost 建一篇貼文並回傳其 id。
//
// image_url 顯式給空字串：MigrateMySQL 第 2 步那個欄位是 TEXT NOT NULL 而**沒有**
// DEFAULT，嚴格模式下列層級的隱含預設值不會自動補上（見該步的說明），因此省略
// 它的 INSERT 會以 1364 失敗。
func insertSubtreePost(t *testing.T, db *sql.DB, author string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO forum_posts (author_email, content, image_url, created_at) VALUES (?, ?, '', ?)`,
		author, "subtree test post", time.Now())
	if err != nil {
		t.Fatalf("插入貼文失敗: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("讀取貼文 id 失敗: %v", err)
	}
	return id
}

// insertSubtreeComment 建一則留言（parentID 0 代表第一層）並回傳其 id。
func insertSubtreeComment(t *testing.T, db *sql.DB, postID, parentID int64, depth int, author string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO forum_post_comments
		(post_id, parent_id, depth, author_email, content, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		postID, parentID, depth, author, "comment", time.Now())
	if err != nil {
		t.Fatalf("插入留言失敗: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("讀取留言 id 失敗: %v", err)
	}
	return id
}

// remainingSubtreeComments 回傳「這篇貼文下還活著的留言」，依 id 排序。
func remainingSubtreeComments(t *testing.T, db *sql.DB, postID int64) []subtreeComment {
	t.Helper()
	rows, err := db.Query(`SELECT id, parent_id, depth, author_email FROM forum_post_comments
	                       WHERE post_id = ? ORDER BY id`, postID)
	if err != nil {
		t.Fatalf("查詢剩餘留言失敗: %v", err)
	}
	defer rows.Close()

	var out []subtreeComment
	for rows.Next() {
		var c subtreeComment
		if err := rows.Scan(&c.id, &c.parentID, &c.depth, &c.author); err != nil {
			t.Fatalf("讀取剩餘留言失敗: %v", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iteration 剩餘留言失敗: %v", err)
	}
	return out
}

// commentIDs 只把 id 拿出來，讓斷言訊息保持可讀。
func commentIDs(comments []subtreeComment) []int64 {
	ids := make([]int64, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.id)
	}
	return ids
}

func sameIDs(got, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// userScope 是 handleForumCommentDelete 實際使用的授權 JOIN（作者本人 + 母文章
// 還活著），逐字照抄 —— 測試要驗的正是「這兩個條件一起進去時，哪些列被刪掉」。
const userScope = `JOIN forum_post_comments root
		      ON root.id = ? AND root.post_id = ? AND root.author_email = ?
		 JOIN forum_posts p ON p.id = ? AND p.deleted_at IS NULL`

/*
TestForumCommentSubtreeDeleteMySQL 對真的 MySQL 執行子樹刪除，逐項確認：

  1. 三層子樹整棵消失（自己 + 回覆 + 回覆的回覆），而別棵樹不受影響。
  2. 孫留言也真的在路徑上：少一路徑的症狀是「第一層刪了、孫子留在樹外」。
  3. 權限：不是那一則的作者時 0 列、什麼都不刪（子孫即使是你寫的也不留）。
  4. 母文章軟刪除後 0 列（使用者的刪除路徑要求貼文還活著）。
  5. 跨篇掛錯：拿 A 篇的留言 id 配 B 篇的 postID，0 列且 A 篇的資料不動。
*/
func TestForumCommentSubtreeDeleteMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	ctx := context.Background()

	postID := insertSubtreePost(t, db, "alice@example.com")
	// 樹一：第一層 a1 → a2（第二層）→ a3（第三層，最深）。
	a1 := insertSubtreeComment(t, db, postID, 0, 1, "alice@example.com")
	a2 := insertSubtreeComment(t, db, postID, a1, 2, "bob@example.com")
	a3 := insertSubtreeComment(t, db, postID, a2, 3, "carol@example.com")
	// 樹二：另一組第一層 + 它的回覆（作者不同，用來隔離兩棵樹）。
	b1 := insertSubtreeComment(t, db, postID, 0, 1, "dave@example.com")
	b2 := insertSubtreeComment(t, db, postID, b1, 2, "alice@example.com")

	// 插進去的當下就該是這樣（順便證明 MigrateMySQL 第 31 步的 parent_id / depth
	// 真的存在且收得下handler寫入的形狀）。
	if got := commentIDs(remainingSubtreeComments(t, db, postID)); !sameIDs(got, []int64{a1, a2, a3, b1, b2}) {
		t.Fatalf("插入後留言 = %v, want %v", got, []int64{a1, a2, a3, b1, b2})
	}

	// 1) 作者本人刪第一層 → 三列一起消失，樹二不動。
	affected, err := forumCommentSubtreeDelete(ctx, db, postID, a1, userScope, a1, postID, "alice@example.com", postID)
	if err != nil {
		t.Fatalf("子樹刪除（三層）回傳錯誤: %v", err)
	}
	if affected != 3 {
		t.Errorf("刪除第一層留言 affected = %d, want 3（自己 + 回覆 + 回覆的回覆）", affected)
	}
	if got := commentIDs(remainingSubtreeComments(t, db, postID)); !sameIDs(got, []int64{b1, b2}) {
		t.Errorf("刪除後剩餘留言 = %v, want %v（只有另一棵樹）", got, []int64{b1, b2})
	}

	// 2) 權限：dave 的留言由 alice 刪 → 0 列，樹二完好。
	affected, err = forumCommentSubtreeDelete(ctx, db, postID, b1, userScope, b1, postID, "alice@example.com", postID)
	if err != nil {
		t.Fatalf("子樹刪除（非作者）回傳錯誤: %v", err)
	}
	if affected != 0 {
		t.Errorf("非作者刪除 affected = %d, want 0", affected)
	}
	if got := commentIDs(remainingSubtreeComments(t, db, postID)); !sameIDs(got, []int64{b1, b2}) {
		t.Errorf("非作者刪除後剩餘留言 = %v, want %v（一點都不該少）", got, []int64{b1, b2})
	}

	// 3) 作者本人刪第二層 → 它與它的回覆一起消失，第一層同樹的留著。
	//    （重新建一棵，因為 b1 是 dave 的。）
	c1 := insertSubtreeComment(t, db, postID, 0, 1, "alice@example.com")
	c2 := insertSubtreeComment(t, db, postID, c1, 2, "bob@example.com")
	c3 := insertSubtreeComment(t, db, postID, c2, 3, "carol@example.com")
	if got := commentIDs(remainingSubtreeComments(t, db, postID)); !sameIDs(got, []int64{b1, b2, c1, c2, c3}) {
		t.Fatalf("插入第二棵樹後留言 = %v, want %v", got, []int64{b1, b2, c1, c2, c3})
	}
	affected, err = forumCommentSubtreeDelete(ctx, db, postID, c2, userScope, c2, postID, "bob@example.com", postID)
	if err != nil {
		t.Fatalf("子樹刪除（第二層）回傳錯誤: %v", err)
	}
	if affected != 2 {
		t.Errorf("刪除第二層留言 affected = %d, want 2（自己 + 第三層回覆）", affected)
	}
	if got := commentIDs(remainingSubtreeComments(t, db, postID)); !sameIDs(got, []int64{b1, b2, c1}) {
		t.Errorf("刪除第二層後剩餘留言 = %v, want %v（第一層的父親留著）", got, []int64{b1, b2, c1})
	}

	// 4) 母文章被軟刪除 → 使用者的刪除路徑跟著消失（handler 因此回 404）。
	if _, err := db.Exec(`UPDATE forum_posts SET deleted_at = ? WHERE id = ?`, time.Now(), postID); err != nil {
		t.Fatalf("軟刪除貼文失敗: %v", err)
	}
	before := commentIDs(remainingSubtreeComments(t, db, postID))
	affected, err = forumCommentSubtreeDelete(ctx, db, postID, c1, userScope, c1, postID, "alice@example.com", postID)
	if err != nil {
		t.Fatalf("子樹刪除（貼文已刪除）回傳錯誤: %v", err)
	}
	if affected != 0 {
		t.Errorf("貼文已軟刪除時 affected = %d, want 0", affected)
	}
	if got := commentIDs(remainingSubtreeComments(t, db, postID)); !sameIDs(got, before) {
		t.Errorf("貼文已軟刪除時的刪除動到了資料：%v → %v", before, got)
	}
}

/*
TestForumCommentSubtreeDeleteCrossPostMySQL 釘住 parentId 的跨篇防護。

少了 root.post_id 這個條件時，任何人只要知道別篇的任何留言 id，就能把自己的
回覆掛到那則留言底下 —— 而刪除那一則時，被刪的會是自己這篇的整棵樹。症狀不是
錯誤，是「回覆了卻在自己的貼文頁面上看不到」。
*/
func TestForumCommentSubtreeDeleteCrossPostMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	ctx := context.Background()

	postA := insertSubtreePost(t, db, "alice@example.com")
	postB := insertSubtreePost(t, db, "bob@example.com")
	// A 篇的一棵完整樹，以及 B 篇的一則留言。
	a1 := insertSubtreeComment(t, db, postA, 0, 1, "alice@example.com")
	a2 := insertSubtreeComment(t, db, postA, a1, 2, "alice@example.com")
	b1 := insertSubtreeComment(t, db, postB, 0, 1, "bob@example.com")

	// 用 B 篇的 postID 搭配 A 篇的留言 id：root.post_id 不匹配，什麼都不該刪。
	affected, err := forumCommentSubtreeDelete(ctx, db, postA, a1, userScope, a1, postB, "alice@example.com", postB)
	if err != nil {
		t.Fatalf("跨篇子樹刪除回傳錯誤: %v", err)
	}
	if affected != 0 {
		t.Errorf("跨篇刪除 affected = %d, want 0", affected)
	}
	if got := commentIDs(remainingSubtreeComments(t, db, postA)); !sameIDs(got, []int64{a1, a2}) {
		t.Errorf("A 篇剩餘留言 = %v, want %v（跨篇嘗試不該動到任何一列）", got, []int64{a1, a2})
	}
	if got := commentIDs(remainingSubtreeComments(t, db, postB)); !sameIDs(got, []int64{b1}) {
		t.Errorf("B 篇剩餘留言 = %v, want %v", got, []int64{b1})
	}
}

/*
TestForumCommentSubtreeDeleteLeafMySQL 確認最深層（第三層）自己也能被刪。

第三層不能再被回覆，但它可以被作者收回，而它的子孫數是 0 —— affected 必須
剛好是 1。少了這一案，一條只回傳 1 的重構（退回單列刪除）不會被發現。
*/
func TestForumCommentSubtreeDeleteLeafMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	ctx := context.Background()

	postID := insertSubtreePost(t, db, "alice@example.com")
	a1 := insertSubtreeComment(t, db, postID, 0, 1, "alice@example.com")
	a2 := insertSubtreeComment(t, db, postID, a1, 2, "alice@example.com")
	a3 := insertSubtreeComment(t, db, postID, a2, 3, "alice@example.com")

	affected, err := forumCommentSubtreeDelete(ctx, db, postID, a3, userScope, a3, postID, "alice@example.com", postID)
	if err != nil {
		t.Fatalf("刪除第三層留言回傳錯誤: %v", err)
	}
	if affected != 1 {
		t.Errorf("刪除第三層留言 affected = %d, want 1", affected)
	}
	if got := commentIDs(remainingSubtreeComments(t, db, postID)); !sameIDs(got, []int64{a1, a2}) {
		t.Errorf("刪除第三層後剩餘留言 = %v, want %v", got, []int64{a1, a2})
	}
}
