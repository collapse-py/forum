package httpapi

/*
內容趨勢統計（httpapi/stats_handlers.go）。

這一組查詢回答的是「這站正在長成什麼樣子」，而既有的三個後臺頁都答不出來：
用戶管理只有一個時點的總數與停權計數，論壇文章只有依時間排序的清單，檢舉管理
只有待裁決的佇列。它們都是「現在是什麼」，沒有「變成這樣多久了」。

查詢的三個設計決定

	1. 全部是唯讀，因此不接稽核
		稽核記的是「誰改了資料」。這裡沒有任何寫入，也沒有任何個人可識別的
		輸出（熱門文章只帶前 80 字，與公開頁看到的一樣），所以不值得記。
		反過來說，若日後這個頁面加了「清除統計」之類的按鈕，就必須接上。

	2. 日別序列讀彙總表，不每次即時計算
		日別新增量讀 forum_daily_stats（第 35 步的彙總表），由 data 套件的
		BackfillDailyStats / RefreshDailyStats 定期重算（見 data/rollup.go）。
		這正是 KNOWN_ISSUES 與舊版檔頭寫明的「該改的時候」：即時計算的掃描
		量被視窗限制住，而視窗（90 天）本身有上限——內容量成長到視窗不再有
		界時，正確做法是加彙總表而不是縮視窗（縮視窗會讓圖上看不出季節性的
		起伏，而那正是這一頁存在的理由）。
		語意上的差別要誠實寫出來：彙總表記的是「那一天產生了多少」，含後來
		被軟刪除的貼文；舊的即時查詢記的是「現在還看得到多少」。前者才是
		「趨勢」該用的口径——一篇被刪的文章在它被寫的那一天確實發生。
		熱門文章／活躍作者仍走即時查询：它们需要内容与互动计数，无法由日別
		計數彙總出来，而它们的扫描同样被視窗限制住。

	3. 分日以「伺服器本地時區」為準，且由 Go 補齊缺漏的日期
		created_at 由應用層寫入 time.Now()（不是資料庫的 NOW()），所以它
		帶的是主機的本地時區（見 MigrateMySQL 第 1 步的說明）。因此
		DATE(created_at) 這個分組正好等於「主機上的自然日」，而
		NOW() 與 DATE() 在同一個連線裡，兩者一致。

		缺漏的日期由 Go 補成 0，而不是讓資料庫回傳稀疏的結果：那樣前端會
		把線段的空白誤解成「沒有資料」而不是「那天真的是零」。這與
		monitor 頁的時間軸是同一個原則。

	一個刻意不做的優化
		「熱門文章」用兩個相關子查詢（留言數、按讚數）而不是 JOIN + GROUP BY。
		相關子查詢每個都是一次主鍵／索引查找，而 JOIN 會讓留言表與按讚表
		各被掃一次再聚合。差別在文章數小的時候不明顯，但相關子查詢的行為
		不隨資料量變糟，而 JOIN 會。
*/

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/logger"
)

// 統計視窗的預設與上下限。
//
// 上限 90 天是因為圖的橫軸寬度有限，而 90 天已經超過一個學期 ——
// 再長的視窗需要的是另一種呈現（按月彙總），而不是把日別的柱子拉得更密。
const (
	defaultStatsDays = 30
	maxStatsDays     = 90
)

// 排行項目的顯示筆數。三個排行榜共用同一個常數，讓它們在視覺上等高 ——
// 讀者比較的是「誰在上面」，不是「這個榜多長」。
const statsTopLimit = 10

// 熱門文章摘要的字數。刻意與公開頁的貼文卡片預覽同量級：這裡的用途是讓
// 管理員認出「是哪一篇」，而不是在後臺重新閱讀全文。
const statsExcerptRunes = 80

// statsPayload 是 GET /api/admin/stats 的回應。
type statsPayload struct {
	OK     bool        `json:"ok"`
	Now    string      `json:"now"`
	Forum  string      `json:"forum"`
	Days   int         `json:"days"`
	Series statsSeries `json:"series"`

	// Totals 是整個視窗的合計，不是「目前的總數」。兩者不同：目前的總數在
	// 用戶管理頁看得到，而這裡要回答的是「這 30 天產生了多少」。
	Totals statsTotals `json:"totals"`

	TopPosts   []statsPost   `json:"topPosts"`
	TopTags    []statsTag    `json:"topTags"`
	TopAuthors []statsAuthor `json:"topAuthors"`
}

// statsSeries 是日別新增量。三個陣列與 Dates 一一對應、等長，且長度固定是
// Days —— 連沒有資料的日子都在陣列裡（值為 0），見檔頭第三點。
type statsSeries struct {
	Dates    []string `json:"dates"`
	Users    []int    `json:"users"`
	Posts    []int    `json:"posts"`
	Comments []int    `json:"comments"`
}

// statsTotals 是視窗內的合計。
type statsTotals struct {
	Users    int `json:"users"`
	Posts    int `json:"posts"`
	Comments int `json:"comments"`
	// Likes 是「視窗內新按的讚」，與熱門文章的「累計讚數」不同義。分開命名
	// 是為了讓「這 30 天有多少互動」與「哪篇文章最熱門」不會被混為一談。
	Likes int `json:"likes"`
}

type statsPost struct {
	ID          int64  `json:"id"`
	AuthorEmail string `json:"authorEmail"`
	Excerpt     string `json:"excerpt"`
	CreatedAt   string `json:"createdAt"`
	Comments    int    `json:"comments"`
	Likes       int    `json:"likes"`
}

type statsTag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Users int    `json:"users"`
}

type statsAuthor struct {
	Email    string `json:"email"`
	Posts    int    `json:"posts"`
	Comments int    `json:"comments"`
}

// handleAdminStats 回傳內容趨勢統計。
//
// GET only，且刻意不限流：它是管理員點擊才發出的低頻請求，與監控頁的十秒
// 輪詢不同。限流它的唯一效果是「後臺打不開統計」。
func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	days := statsDays(r.URL.Query().Get("days"))
	// 視窗的起點是「今天 00:00:00 往前數 days-1 天」而不是「現在往前 days×24
	// 小時」：後者的起點會落在今天的某個時點，導致「今天」這個桶只有半天的
	// 資料，看起來像流量突然掉了。這個偏差在分鐘級的圖上不顯眼，在日別的
	// 圖上會是一個每天都存在的假凹陷。
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -(days - 1))

	payload := statsPayload{
		OK:    true,
		Now:   now.UTC().Format(time.RFC3339),
		Forum: s.cfg.ForumName,
		Days:  days,
	}

	var err error
	if payload.Series, err = s.loadStatsSeries(r.Context(), start, days); err != nil {
		logger.ErrorfContext(r.Context(), "[STATS] 無法載入日別序列: %v", err)
		internalError(w, "unable to load content trends")
		return
	}
	if payload.Totals, err = s.loadStatsTotals(r.Context(), start); err != nil {
		logger.ErrorfContext(r.Context(), "[STATS] 無法載入合計: %v", err)
		internalError(w, "unable to load content trends")
		return
	}
	if payload.TopPosts, err = s.loadStatsTopPosts(r.Context(), start); err != nil {
		logger.ErrorfContext(r.Context(), "[STATS] 無法載入熱門文章: %v", err)
		internalError(w, "unable to load content trends")
		return
	}
	if payload.TopTags, err = s.loadStatsTopTags(r.Context()); err != nil {
		logger.ErrorfContext(r.Context(), "[STATS] 無法載入熱門標籤: %v", err)
		internalError(w, "unable to load content trends")
		return
	}
	if payload.TopAuthors, err = s.loadStatsTopAuthors(r.Context(), start); err != nil {
		logger.ErrorfContext(r.Context(), "[STATS] 無法載入熱門作者: %v", err)
		internalError(w, "unable to load content trends")
		return
	}

	writeOK(w, payload)
}

// statsDays 解析 days 查詢參數。
//
// 缺漏、不是數字、負數都用預設值；超過上限則收斂到上限 —— 而不是回 400。
// 理由是這是「把圖拉寬一點」的操作，回 400 會讓使用者在探索時撞到牆；
// 靜默收斂的後果僅僅是「你看不到比 90 天更長的區間」，而介面上寫得出來。
func statsDays(raw string) int {
	days, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || days <= 0 {
		return defaultStatsDays
	}
	if days > maxStatsDays {
		return maxStatsDays
	}
	return days
}

// loadStatsSeries 讀回彙總表的日別新增量，並補齊沒有資料的日子。
//
// 讀彙總表而不是即時 GROUP BY：理由見檔頭第二點。缺資料的日子（彙總表
// 沒有那列）照舊補 0——「那天真的是零」與「沒有資料」對前端是同一件事，
// 而對讀者必須是一條完整的線。
func (s *Server) loadStatsSeries(ctx context.Context, start time.Time, days int) (statsSeries, error) {
	type dayCounts struct {
		users    int
		posts    int
		comments int
		likes    int
	}
	counts := make(map[string]dayCounts, days)

	rows, err := s.db.QueryContext(ctx, `
		SELECT day, new_users, new_posts, new_comments, new_likes
		FROM forum_daily_stats WHERE day >= ? ORDER BY day`, start)
	if err != nil {
		return statsSeries{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var day time.Time
		var c dayCounts
		if err := rows.Scan(&day, &c.users, &c.posts, &c.comments, &c.likes); err != nil {
			return statsSeries{}, err
		}
		counts[day.Format("2006-01-02")] = c
	}
	if err := rows.Err(); err != nil {
		return statsSeries{}, err
	}

	series := statsSeries{
		Dates:    make([]string, 0, days),
		Users:    make([]int, 0, days),
		Posts:    make([]int, 0, days),
		Comments: make([]int, 0, days),
	}
	for i := range days {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		series.Dates = append(series.Dates, day)
		// 缺資料的日子給 0。map 查不到就是沒有那一列，因此 nil 切片與
		// 稀疏結果都會走到這個分支 —— 這正是我們要呈現的。
		c := counts[day]
		series.Users = append(series.Users, c.users)
		series.Posts = append(series.Posts, c.posts)
		series.Comments = append(series.Comments, c.comments)
		// likes 刻意不進日別序列：前端只渲染三条线（使用者／貼文／留言），
		// 按讚的視窗合計在 Totals 裡。多送一条前端不会用的序列只会让
		// 「payload 里有但图上没有」变成新的困惑来源。
		_ = c.likes
	}
	return series, nil
}

// loadStatsTotals 取回視窗內的合計。
//
// 讀彙總表的 SUM，而不是四張來源表各一次 COUNT：日別序列已經讀同一張表，
// 合計就是同一份資料的加總——兩處用不同來源会让「圖加起來不等于總數」
// 成為可能。語意與日別序列一致：記的是「視窗內產生了多少」，含後來被
// 軟刪除的貼文（見檔頭第二點）。
func (s *Server) loadStatsTotals(ctx context.Context, start time.Time) (statsTotals, error) {
	var totals statsTotals
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(new_users), 0), COALESCE(SUM(new_posts), 0),
			COALESCE(SUM(new_comments), 0), COALESCE(SUM(new_likes), 0)
		FROM forum_daily_stats WHERE day >= ?`, start).
		Scan(&totals.Users, &totals.Posts, &totals.Comments, &totals.Likes)
	return totals, err
}

// loadStatsTopPosts 取回視窗內互動最多的文章。
//
// 排序是 (留言 + 按讚) 而不是其中之一：對一個文字論壇而言，「有多少人願意
// 回應」比「有多少人按了讚」更能代表文章的份量，而兩者的相關性不夠高到
// 可以只挑一個當指標。相同分數時以 id 降冪（而非留言或讚數降冪）作為
// 穩定的次要排序 —— 否則同分項目在兩次請求之間會跳動。
//
// deleted_at IS NULL：統計衡量的是「這個站現在看得見的內容」。把已刪除的
// 貼文算進互動榜，會讓管理員去看一篇任何人都打不開的文章。
func (s *Server) loadStatsTopPosts(ctx context.Context, start time.Time) ([]statsPost, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.author_email, p.created_at, LEFT(p.content, `+strconv.Itoa(statsExcerptRunes*4)+`),
		       (SELECT COUNT(*) FROM forum_post_comments c WHERE c.post_id = p.id),
		       (SELECT COUNT(*) FROM forum_post_likes l WHERE l.post_id = p.id)
		FROM forum_posts p
		WHERE p.created_at >= ? AND p.deleted_at IS NULL
		ORDER BY 5 + 6 DESC, p.id DESC
		LIMIT `+strconv.Itoa(statsTopLimit), start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]statsPost, 0, statsTopLimit)
	for rows.Next() {
		var item statsPost
		if err := rows.Scan(&item.ID, &item.AuthorEmail, &item.CreatedAt, &item.Excerpt, &item.Comments, &item.Likes); err != nil {
			return nil, err
		}
		item.Excerpt = excerptRunes(item.Excerpt, statsExcerptRunes)
		items = append(items, item)
	}
	return items, rows.Err()
}

// loadStatsTopTags 取回綁定人數最多的標籤。
//
// 刻意不限制時間視窗：標籤是「身分分類」而不是「事件」，一個人三個月前被
// 標成「高雄」今天仍然是高雄。把它放進 30 天的視窗會讓「熱門標籤」變成
// 「最近剛好被套上的標籤」，而那是完全不同的一個問題。
//
// JOIN 而非 LEFT JOIN：只列出至少綁定一人的標籤。沒有人用的標籤是字典裡
// 的死字，列出來只會佔掉排行榜的名額。
func (s *Server) loadStatsTopTags(ctx context.Context) ([]statsTag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, COUNT(a.user_email)
		FROM forum_user_tags t
		JOIN forum_user_tag_assignments a ON a.tag_id = t.id
		GROUP BY t.id, t.name
		ORDER BY 3 DESC, t.name ASC
		LIMIT `+strconv.Itoa(statsTopLimit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]statsTag, 0, statsTopLimit)
	for rows.Next() {
		var item statsTag
		if err := rows.Scan(&item.ID, &item.Name, &item.Users); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// loadStatsTopAuthors 取回視窗內發文與留言最多的人。
//
// 刻意「不看暱稱、只看 email」：這個排行榜的用途是找出「該去看的帳號」
// （例如需要進一步處理的高活躍帳號），而 email 正是後臺其他頁面用來指稱
// 一個人的鍵。若改成顯示暱稱，同一個人在不同頁面就會有兩種指稱方式。
func (s *Server) loadStatsTopAuthors(ctx context.Context, start time.Time) ([]statsAuthor, error) {
	// 為什麼不合成一個 UNION 或 JOIN：這兩個計數的主鍵不同（文章是
	// author_email × 一列，留言也是 author_email × 一列，但數量級差一個
	// 數量級），合成會讓「沒有發過文但留了很多言」的人完全不出現在榜上 ——
	// 而那正是這個排行榜應該讓人看到的帳號。
	//
	// 但「先取文章前 10 名，再對這 10 個 email 各查一次留言數」的 N+1 也不對：
	// 那 10 次往返每次都是「掃視窗區間再逐列過濾 email」，同一段區間被重掃
	// 10 次，且成本隨排行長度線性成長。改成「文章前 10 名作為衍生表 +
	// 留言數衍生表一次聚合後 left join」，兩個子表各掃一次（各走
	// author_email 索引，遷移第 27 步補的），整條查詢只有一趟往返。
	//
	// 取捨要寫清楚（這是取捨而不是缺陷）：改動前是 10 次「單一作者的等值
	// 查詢」，每次只碰該作者的留言；現在是「掃過視窗內所有留言後依作者
	// 分組」各一次。往返數從 11 降到 1，但讀到的列數從「前 10 名的留言量」
	// 變成「視窗內的留言總量」。當留言總量遠大於前 10 名的留言量時，
	// 新寫法讀的列更多。兩者都走索引、都不退化到全表掃描，因此判斷標準
	// 是「排行榜的 10 個 id 對上全視窗聚合」這個數量級通常小得多。
	//
	// deleted_at IS NULL 與 loadStatsTopPosts 同一個理由：排行榜要能照著名單
	// 去後臺找到那篇文章，而軟刪除的貼文在任何列表都已經不存在。
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.author_email, p.posts, COALESCE(c.comments, 0)
		FROM (
			SELECT author_email, COUNT(*) AS posts
			FROM forum_posts
			WHERE created_at >= ? AND deleted_at IS NULL
			GROUP BY author_email
			ORDER BY posts DESC, author_email ASC
			LIMIT `+strconv.Itoa(statsTopLimit)+`
		) p
		LEFT JOIN (
			SELECT author_email, COUNT(*) AS comments
			FROM forum_post_comments
			WHERE created_at >= ?
			GROUP BY author_email
		) c ON c.author_email = p.author_email
		ORDER BY p.posts DESC, p.author_email ASC`, start, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]statsAuthor, 0, statsTopLimit)
	for rows.Next() {
		var item statsAuthor
		if err := rows.Scan(&item.Email, &item.Posts, &item.Comments); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// excerptRunes 把字串截到 n 個字元。
//
// 以 []rune 計算而非 len(string)：貼文是中文內容，len 會算成三倍而讓
// 「80 字」變成只有 27 個字被顯示。
func excerptRunes(value string, n int) string {
	runes := []rune(value)
	if len(runes) <= n {
		return value
	}
	return string(runes[:n]) + "…"
}
