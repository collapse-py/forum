/*
seed 往既有資料庫填入一批可重複產生的測試資料。

【為什麼是這個工具】

專案刻意不把 schema 寫進 SQL 檔（見 deploy/mysql-init/01-init.sql 的說明），
因此「填測試資料」也必須走同一條路：讀現有 schema、尊重現有的約束
（雪花主鍵、暱稱唯一、複合主鍵），而不是塞一份與程式碼脫節的 CSV。

【它填什麼】

預設 10,000 筆核心資料：300 位使用者 + 6,000 篇貼文 + 3,700 則留言，
另外附帶按讚與收藏（可用 -likes / -bookmarks 調成 0）。數量全部是 flag，
所以要單獨補 10,000 篇貼文就是 -users 0 -posts 10000 -comments 0。

【主鍵怎麼來】

不用 idgen.Next()，而是以「目標建立時間」反推雪花 id
（(ms - epoch) << 12 | node << 6），因為測試資料的建立時間散布在過去 N 天：
以現在時間發號會得到「id 全部大於貼文時間順序」的資料，而動態排序是
(created_at DESC, id DESC)，兩者不一致會讓分頁在邊界上出現重複或漏列。
epoch 與位元配置直接取 idgen 的常數，避免這裡的公式日後與正式碼脫節。

【為什麼全部寫在同一個交易裡】

10,000 筆分批 INSERT 中途失敗時，若每批各自 commit，資料庫會留下「一半
使用者、四分之一貼文」的狀態，而那個狀態不會出現在任何錯誤訊息裡。
單一交易讓結果只有「全部成功」與「完全沒動」兩種，-dry-run 用的也是同一條
路徑（最後 rollback），因此 dry-run 看到的 SQL 與正式執行完全相同。

【重跑的行為】

刻意不加自動清理：刪除與插入寫在同一支工具裡，總有一天會有人在錯誤的
資料庫上按下它。重跑遇到同前綴的資料會直接失敗並提示改用 -prefix。

用法：

	DB_DSN='root:***@tcp(host:3306)/forum?charset=utf8mb4&parseTime=True&loc=Local' \
	  go run ./tools/seed -dry-run \
	  -es -es-url http://host:9200 -es-user elastic -es-pass ***
*/
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"forum/forum/es"
	"forum/forum/idgen"

	_ "github.com/go-sql-driver/mysql"
)

// ---------------------------------------------------------------------------
// 旗標
// ---------------------------------------------------------------------------

var (
	dsn = flag.String("dsn", envOr("DB_DSN", ""),
		"MySQL DSN（預設取環境變數 DB_DSN）")

	users    = flag.Int("users", 300, "要建立的使用者數（forum_users + forum_profiles）")
	posts    = flag.Int("posts", 6000, "要建立的貼文數（forum_posts）")
	comments = flag.Int("comments", 3700, "要建立的留言數（forum_post_comments）")
	likes    = flag.Int("likes", 2000, "要建立的按讚數（forum_post_likes）")
	bookmk   = flag.Int("bookmarks", 500, "要建立的收藏數（forum_post_bookmarks）")

	days   = flag.Int("days", 90, "貼文與留言的時間散布範圍（天，往前推）")
	prefix = flag.String("prefix", "seed", "測試資料的前綴（信箱與暱稱都以它開頭）")
	node   = flag.Int("node", 0, "雪花節點編號，需與 SNOWFLAKE_NODE_ID 一致")
	seed   = flag.Int64("rand-seed", 1, "隨機種子（相同種子產生相同內容）")

	dryRun = flag.Bool("dry-run", false, "走完整流程但最後 rollback，且不寫 ES")

	useES     = flag.Bool("es", false, "建立完成後把貼文寫進 Elasticsearch")
	esURL     = flag.String("es-url", envOr("ES_URL", ""), "ES 位址（預設取環境變數 ES_URL）")
	esIndex   = flag.String("es-index", envOr("ES_INDEX", "forum_posts"), "ES 索引名")
	esUser    = flag.String("es-user", envOr("ES_USERNAME", ""), "ES 使用者名")
	esPass    = flag.String("es-pass", envOr("ES_PASSWORD", ""), "ES 密碼")
	esAPIKey  = flag.String("es-api-key", envOr("ES_API_KEY", ""), "ES API key（若有，優先於帳號密碼）")
	batchSize = flag.Int("batch", 200, "每批 INSERT 的列數")
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// 資料模型
// ---------------------------------------------------------------------------

type userRow struct {
	email     string
	nickname  string
	bio       string
	publicKey string
	createdAt time.Time
}

type postRow struct {
	id        int64
	author    string
	content   string
	createdAt time.Time
}

type commentRow struct {
	id        int64
	postID    int64
	parentID  int64
	depth     int64
	author    string
	content   string
	createdAt time.Time
}

// ---------------------------------------------------------------------------
// 雪花 id 分配
// ---------------------------------------------------------------------------

// idAllocator 依「建立時間遞增」的順序發號，避開資料庫裡既有的 id。
//
// 為什麼要避開既有值：主鍵衝突會讓整批交易 rollback，而錯誤訊息只會說
// "Duplicate entry"，不會指出是哪一列。先把既有 id 讀進 map 是 1 次查詢，
// 比撞到之後再猜便宜得多。
type idAllocator struct {
	taken map[int64]bool
	last  int64
	node  int64
}

func newIDAllocator(db *sql.DB, table string, node int64) (*idAllocator, error) {
	a := &idAllocator{taken: make(map[int64]bool), node: node}
	rows, err := db.Query("SELECT id FROM " + table)
	if err != nil {
		return nil, fmt.Errorf("讀取 %s 既有 id: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		a.taken[id] = true
		if id > a.last {
			a.last = id
		}
	}
	return a, rows.Err()
}

// next 以 t 反推雪花 id，並保證嚴格遞增且不與既有值衝突。
func (a *idAllocator) next(t time.Time) int64 {
	elapsed := t.UnixMilli() - idgen.EpochMillis
	if elapsed < 0 {
		elapsed = 0
	}
	id := (elapsed << 12) | (a.node << 6)
	if id <= a.last {
		id = a.last + 1
	}
	for a.taken[id] {
		id++
	}
	a.taken[id] = true
	a.last = id
	return id
}

// ---------------------------------------------------------------------------
// 內容產生
// ---------------------------------------------------------------------------

var (
	topics = []string{
		"期末考", "選課", "社團活動", "宿舍生活", "校園午餐", "打工", "畢業專題",
		"圖書館", "天氣", "機車", "租房", "系上聚會", "運動", "實習",
		"成果展", "社團練習", "早八的課", "公車", "系館冷氣", "選修學分",
	}
	feelings = []string{
		"很煩", "很爽", "很累", "很期待", "很緊張", "很無奈", "很感動",
		"很困惑", "很滿意", "很傻眼", "很懷念",
	}
	openers = []string{
		"說真的", "老實講", "認真問", "想分享一下", "先講結論",
		"昨天遇到一件事", "趁還有印象趕快寫下來", "",
	}
	bodies = []string{
		"圖書館八點就滿位了，只好回宿舍念。",
		"系上統計的時間表到現在還沒公佈。",
		"隔壁寢室半夜還在打遊戲，戴耳塞也沒用。",
		"排了二十分鐘才買到，結果麵已經糊掉。",
		"騎到一半下雨，淋得全身都濕了。",
		"跟組員開了三次會，進度還是在原地。",
		"早上七點五十到，整間教室只剩下最後一排。",
		"跟老闆說好做三週，結果第一週就開始加班。",
		"行李收了三個小時，最後還是少帶充電線。",
		"教練說這週要把動作練熟，明天要驗收。",
		"老師說這門課不點名，結果第一次小考就當了兩成。",
		"社團辦的攤位比預期多了兩倍的人，準備的東西一下就沒了。",
		"搬宿舍那天電梯壞掉，五樓用扛的。",
		"系統維護到凌晨，等到睡著都沒搶到。",
		"跟隔壁系合辦，場地臨時換到小禮堂。",
	}
	tails = []string{
		"有人也這樣覺得嗎？", "想聽聽大家的看法。", "請問有人遇過嗎？",
		"有推薦的解法嗎？", "歡迎在下面聊聊。", "先謝謝回覆的人。",
		"純粹發洩，不用安慰也沒關係。", "",
	}
	commentLines = []string{
		"同感。", "我也是這樣覺得。", "推一個。", "這篇寫得很好。",
		"希望原po能撐住。", "我上週也遇到一樣的問題。", "樓上講到重點了。",
		"真的假的？", "感謝分享，收下了。", "我覺得要看科系。",
		"先留個言，晚點回來補充。", "這答案正是我要找的。",
		"校門口那家比較便宜，可以試試。", "建議直接問助教比較快。",
		"看到這篇想起我大一的時候。", "+1，我也是過來人。",
	}
	emoji = []string{"👍", "😂", "🙏", "😭", "✨", ""}
)

// publickeyFor 與 httpapi.publicForumKey 位元級一致（SHA-256(email) 的小寫十六進位），
// 這樣 seed 出來的使用者在公開頁上與真實使用者走同一條查詢路徑。
func publickeyFor(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	flag.Parse()
	if strings.TrimSpace(*dsn) == "" {
		fmt.Fprintln(os.Stderr, "缺少 DSN：請給 -dsn 或設定 DB_DSN")
		os.Exit(2)
	}

	ctx := context.Background()
	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		fail("開啟連線", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		fail("連線失敗", err)
	}

	// 前綴檢查：重跑時寧可失敗，也不要產生 seed0001 與 seed0002 這種
	// 兩批混在一起、之後分不清哪批可以刪的資料。
	var existing int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM forum_users WHERE email LIKE ?",
		*prefix+"%").Scan(&existing); err != nil {
		fail("檢查既有 seed 資料", err)
	}
	if existing > 0 {
		fmt.Fprintf(os.Stderr,
			"資料庫裡已經有 %d 筆前綴為 %q 的使用者。要再填一批請改用 -prefix，"+
				"要先清掉請自行刪除（本工具刻意不提供刪除）。\n", existing, *prefix)
		os.Exit(1)
	}

	rng := rand.New(rand.NewSource(*seed))
	now := time.Now()

	// ---- 產生資料 ---------------------------------------------------------
	var userRows []userRow
	for i := 1; i <= *users; i++ {
		email := fmt.Sprintf("%s%04d@example.test", *prefix, i)
		created := randTime(rng, now.AddDate(0, 0, -*days*2), now)
		userRows = append(userRows, userRow{
			email:     email,
			nickname:  fmt.Sprintf("%s-%04d", *prefix, i),
			bio:       pick(rng, bodies),
			publicKey: publickeyFor(email),
			createdAt: created,
		})
	}

	postsAlloc, err := newIDAllocator(db, "forum_posts", int64(*node))
	if err != nil {
		fail("讀取 forum_posts 既有 id", err)
	}
	var postRows []postRow
	for i := 0; i < *posts; i++ {
		created := randTime(rng, now.AddDate(0, 0, -*days), now)
		postRows = append(postRows, postRow{
			author:    pick(rng, userRows).email,
			content:   makePost(rng),
			createdAt: created,
		})
	}
	// 時間遞增後才發號，id 才會與 created_at 同向（見 idAllocator 的說明）。
	sort.Slice(postRows, func(i, j int) bool { return postRows[i].createdAt.Before(postRows[j].createdAt) })
	for i := range postRows {
		postRows[i].id = postsAlloc.next(postRows[i].createdAt)
	}

	commentsAlloc, err := newIDAllocator(db, "forum_post_comments", int64(*node))
	if err != nil {
		fail("讀取 forum_post_comments 既有 id", err)
	}
	// 留言先只決定「內容、作者、時間」，排序並發號之後才決定誰回覆誰：
	// parent_id 必須指向一個 id 已經確定、且時間較早的留言。
	type pendingComment struct {
		row       commentRow
		wantReply bool
	}
	var pending []pendingComment
	for i := 0; i < *comments; i++ {
		post := pick(rng, postRows)
		// 留言時間落在貼文之後、現在之前。
		upper := now
		if !upper.After(post.createdAt) {
			upper = post.createdAt.Add(time.Second)
		}
		pending = append(pending, pendingComment{
			row: commentRow{
				postID:    post.id,
				parentID:  0,
				depth:     1,
				author:    pick(rng, userRows).email,
				content:   makeComment(rng),
				createdAt: randTime(rng, post.createdAt, upper),
			},
			wantReply: rng.Intn(4) == 0,
		})
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].row.createdAt.Before(pending[j].row.createdAt)
	})

	commentRows := make([]commentRow, 0, len(pending))
	rootOf := map[int64]int64{}     // postID -> 該貼文第一則留言的 id
	rootAt := map[int64]time.Time{} // 與上面對應的建立時間，用來保證回覆一定比較晚
	for _, p := range pending {
		c := p.row
		c.id = commentsAlloc.next(c.createdAt)
		// 約四分之一是回覆：掛在該貼文的第一則留言下，depth=2。
		if root, ok := rootOf[c.postID]; ok && p.wantReply && rootAt[c.postID].Before(c.createdAt) {
			c.parentID, c.depth = root, 2
		} else if _, seen := rootOf[c.postID]; !seen {
			rootOf[c.postID], rootAt[c.postID] = c.id, c.createdAt
		}
		commentRows = append(commentRows, c)
	}

	likePairs, err := pickPairs(rng, *likes, postRows, userRows)
	if err != nil {
		fail("產生按讚", err)
	}
	bookmarkPairs, err := pickPairs(rng, *bookmk, postRows, userRows)
	if err != nil {
		fail("產生收藏", err)
	}

	// ---- 寫入 -------------------------------------------------------------
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fail("開始交易", err)
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	if err := insertRows(ctx, tx, "forum_users",
		[]string{"email", "status", "created_at", "updated_at"},
		len(userRows), *batchSize, func(i int) []any {
			u := userRows[i]
			return []any{u.email, "ACTIVE", u.createdAt, u.createdAt}
		}); err != nil {
		fail("寫入 forum_users", err)
	}

	if err := insertRows(ctx, tx, "forum_profiles",
		[]string{"author_email", "public_key", "nickname", "bio", "updated_at", "avatar_url"},
		len(userRows), *batchSize, func(i int) []any {
			u := userRows[i]
			return []any{u.email, u.publicKey, u.nickname, u.bio, u.createdAt, ""}
		}); err != nil {
		fail("寫入 forum_profiles", err)
	}

	if err := insertRows(ctx, tx, "forum_posts",
		[]string{"id", "author_email", "content", "created_at", "image_url", "pinned", "updated_at", "deleted_at"},
		len(postRows), *batchSize, func(i int) []any {
			p := postRows[i]
			return []any{p.id, p.author, p.content, p.createdAt, "", 0, nil, nil}
		}); err != nil {
		fail("寫入 forum_posts", err)
	}

	if err := insertRows(ctx, tx, "forum_post_comments",
		[]string{"id", "post_id", "parent_id", "depth", "author_email", "content", "created_at", "updated_at"},
		len(commentRows), *batchSize, func(i int) []any {
			c := commentRows[i]
			return []any{c.id, c.postID, c.parentID, c.depth, c.author, c.content, c.createdAt, nil}
		}); err != nil {
		fail("寫入 forum_post_comments", err)
	}

	if err := insertRows(ctx, tx, "forum_post_likes",
		[]string{"post_id", "author_email", "created_at"},
		len(likePairs), *batchSize, func(i int) []any {
			p := likePairs[i]
			return []any{p.post.id, p.email, p.at}
		}); err != nil {
		fail("寫入 forum_post_likes", err)
	}

	if err := insertRows(ctx, tx, "forum_post_bookmarks",
		[]string{"post_id", "author_email", "created_at"},
		len(bookmarkPairs), *batchSize, func(i int) []any {
			p := bookmarkPairs[i]
			return []any{p.post.id, p.email, p.at}
		}); err != nil {
		fail("寫入 forum_post_bookmarks", err)
	}

	summary := fmt.Sprintf(
		"使用者 %d、貼文 %d、留言 %d、按讚 %d、收藏 %d（合計 %d 筆）",
		len(userRows), len(postRows), len(commentRows), len(likePairs), len(bookmarkPairs),
		len(userRows)*2+len(postRows)+len(commentRows)+len(likePairs)+len(bookmarkPairs),
	)

	if *dryRun {
		if err := tx.Rollback(); err != nil {
			fail("rollback", err)
		}
		committed = true
		fmt.Printf("dry-run 完成（未寫入）：%s\n", summary)
		return
	}
	if err := tx.Commit(); err != nil {
		fail("提交交易", err)
	}
	committed = true
	fmt.Printf("MySQL 寫入完成：%s\n", summary)

	// ---- Elasticsearch ----------------------------------------------------
	if !*useES {
		fmt.Println("未指定 -es，跳過 Elasticsearch 索引。")
		return
	}
	if strings.TrimSpace(*esURL) == "" {
		fmt.Fprintln(os.Stderr, "-es 需要 -es-url（或 ES_URL）")
		os.Exit(1)
	}
	client := es.New(*esURL, *esIndex, es.Auth{
		Username: *esUser,
		Password: *esPass,
		APIKey:   *esAPIKey,
	})
	if err := client.EnsureIndex(ctx); err != nil {
		fail("建立/確認 ES 索引", err)
	}
	docs := make([]es.PostDocument, 0, len(postRows))
	for _, p := range postRows {
		docs = append(docs, es.PostDocument{
			ID:          p.id,
			Content:     p.content,
			AuthorEmail: p.author,
			CreatedAt:   p.createdAt,
		})
	}
	if err := client.IndexPosts(ctx, docs); err != nil {
		fail("寫入 ES 索引", err)
	}
	fmt.Printf("Elasticsearch 完成：%d 篇已寫入索引 %s\n", len(docs), client.IndexName())
}

// ---------------------------------------------------------------------------
// 輔助
// ---------------------------------------------------------------------------

type pair struct {
	post  postRow
	email string
	at    time.Time
}

// pickPairs 產生 count 個不重複的 (貼文, 使用者) 組合，且不含貼文作者本人
// —— 自己按讚自己的貼文在介面上會出現「讚數 +1 但作者是你」的奇怪狀態。
func pickPairs(rng *rand.Rand, count int, posts []postRow, users []userRow) ([]pair, error) {
	if len(posts) == 0 || len(users) == 0 {
		if count > 0 {
			return nil, fmt.Errorf("貼文或使用者為 0，無法建立關聯資料")
		}
		return nil, nil
	}
	seen := make(map[int64]map[string]bool, count)
	out := make([]pair, 0, count)
	for attempts := 0; len(out) < count && attempts < count*20; attempts++ {
		p := pick(rng, posts)
		u := pick(rng, users)
		if u.email == p.author {
			continue
		}
		if seen[p.id] == nil {
			seen[p.id] = make(map[string]bool)
		}
		if seen[p.id][u.email] {
			continue
		}
		seen[p.id][u.email] = true
		out = append(out, pair{post: p, email: u.email, at: later(rng, p.createdAt)})
	}
	if len(out) < count {
		return nil, fmt.Errorf("組合數不足：只要到 %d / %d（貼文與使用者的組合可能已被用完）", len(out), count)
	}
	return out, nil
}

func insertRows(ctx context.Context, tx *sql.Tx, table string, cols []string,
	count, batch int, row func(i int) []any) error {
	if count == 0 {
		return nil
	}
	if batch <= 0 {
		batch = 200
	}
	placeholders := "(" + strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",") + ")"
	for start := 0; start < count; start += batch {
		end := start + batch
		if end > count {
			end = count
		}
		var sb strings.Builder
		args := make([]any, 0, (end-start)*len(cols))
		for i := start; i < end; i++ {
			if i > start {
				sb.WriteByte(',')
			}
			sb.WriteString(placeholders)
			args = append(args, row(i)...)
		}
		query := "INSERT INTO " + table + " (" + strings.Join(cols, ",") + ") VALUES " + sb.String()
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("%s 批次 %d-%d: %w", table, start, end, err)
		}
	}
	return nil
}

func randTime(rng *rand.Rand, from, to time.Time) time.Time {
	if !to.After(from) {
		return from
	}
	return from.Add(time.Duration(rng.Int63n(int64(to.Sub(from)))))
}

// later 回傳一個不早於 t、且盡量在「現在」之前的時刻，用於按讚／收藏。
func later(rng *rand.Rand, t time.Time) time.Time {
	now := time.Now()
	if !now.After(t) {
		return now
	}
	return t.Add(time.Duration(rng.Int63n(int64(now.Sub(t)))))
}

func pick[T any](rng *rand.Rand, xs []T) T {
	return xs[rng.Intn(len(xs))]
}

func makePost(rng *rand.Rand) string {
	parts := []string{}
	if o := pick(rng, openers); o != "" {
		parts = append(parts, o+"：")
	}
	parts = append(parts,
		"最近的"+pick(rng, topics)+" "+pick(rng, feelings)+"。",
		pick(rng, bodies))
	if t := pick(rng, tails); t != "" {
		parts = append(parts, t)
	}
	return strings.Join(parts, "")
}

func makeComment(rng *rand.Rand) string {
	line := pick(rng, commentLines)
	if e := pick(rng, emoji); e != "" && rng.Intn(2) == 0 {
		line += " " + e
	}
	return line
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s 失敗：%v\n", what, err)
	os.Exit(1)
}
