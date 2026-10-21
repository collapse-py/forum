/*
論壇 RSS 摘要（backend/forum/httpapi/forum_rss.go）。

一、這是什麼

	GET /feed.xml —— 最新貼文的 RSS 2.0 摘要，公開、不需登入、不受限流。
	前端各頁的 <head> 有一條 rel="alternate" 的探索連結（見 frontend/*.html），
	使用者把這個網址貼進任何閱讀器就能訂閱這個站。

二、為什麼是 RSS 而不是 JSON feed

	這個專案的需求是「讓使用者用既有工具追蹤新內容」，而 RSS 是那個工具
	生態系唯一普遍支援的格式。它不需要本站做任何新東西（沒有推送、沒有
	註冊），因此是一個可以完全由既有的貼文列表衍生出來的端點。

三、內容取捨（每一條都是隱私或穩定性的決定）

  - 不含作者：RSS 2.0 的 <author> 規格要求 email 形式，而本站的底線是
    email 不離開後端；合理的替代（dc:creator 的暱稱）需要額外命名空間，
    而暱稱在「發完就忘」的匿名站上也不是穩定資訊。閱讀器要看向誰發的文
    可以點進貼文頁 —— 每一則 item 的 link 就是永久連結。
  - 不含圖片：描述裡的圖片需要 media token，而 token 的壽命是分鐘級 ——
    RSS 是「存起來慢慢讀」的格式，簽一個幾分鐘後失效的網址只會得到一堆
    破圖。貼文頁有圖，連結就在那裡。
  - 全站公開內容，含停權作者：與動態列表同一條規則（listForumPosts 不
    過濾停權）。RSS 過濾而列表不過濾會讓兩個清單對「現在有什麼」給出
    不同答案。
  - 時間排序，不套用置頂：RSS 的讀者預期「由新到舊」，一篇置頂半年的
    文章出現在每一份摘要頂端會讓閱讀器把它當成新內容。置頂是站內瀏覽
    的排版決定，不是內容的時序。
  - 50 則上限：RSS 是「最近發生什麼」而不是全文匯出。上限也讓每次請求的
    成本有界（與其他列表端點同一條理由）。

四、快取

	每一次請求都重新查詢與組稿（沒有快取層），但回應帶 ETag 與
	Cache-Control。ETag 是整份文件的 SHA-256：內容沒變就回 304，因此
	閱讀器的定期輪詢幾乎不傳輸內容；內容一變（新貼文、編輯、刪除、還原）
	雜湊就變，不會給出過期的 304。

	「內容沒變」的前提是文件裡沒有任何欄位會自己變動。lastBuildDate 因此
	由**最新一則貼文的時間**推導，而不是產生回應的時刻（見下方的組稿處）——
	後者會讓每一次請求的雜湊都不同，304 永遠不會發生，而症狀只是閱讀器
	每次輪詢都重抓整份文件：沒有任何錯誤、沒有任何日誌。

五、軟刪除

	只列 deleted_at IS NULL 的貼文（forumPostVisible 的同一條規則）：
	被作者或管理員刪除的貼文在任何公開端點都不存在，RSS 不該是例外。
	還原（後台的 /restore）之後它會自然回到摘要裡。
*/
package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/logger"
)

// forumRSSItemLimit 是摘要最多帶幾則貼文。
//
// 與其他列表端點的單頁筆數（25 / 50）同一個量級：RSS 是「最近發生什麼」
// 的摘要，不是全文匯出。閱讀器抓到的內容本來就會被自己的規則截短。
const forumRSSItemLimit = 50

// rssTimeLayout 是 RSS 的日期格式。
//
// RFC 822（RSS 2.0 的規格書引用它）要求時區寫成 GMT 而不是 +0000，
// 因此先把時間轉 UTC 再以固定字串 "GMT" 收尾，而不是用 Go 的
// time.RFC1123Z（它產出 +0000，多數閱讀器能吃，但沒必要冒這個險）。
const rssTimeLayout = "Mon, 02 Jan 2006 15:04:05 GMT"

// rssDocument 是整份摘要的根元素。欄位標籤直接對應 RSS 2.0 的元素名。
type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

// rssChannel 是頻道層級的中介資料。
//
// lastBuildDate 是 omitempty：沒有貼文時整欄省略。RSS 2.0 把它列為選填，
// 而「一份沒有任何內容的 feed」不該宣稱自己建於某個時刻 —— 省略比填一個
// 與內容無關的時間更誠實（那個時間會是唯一會自己變動的欄位，見 handleForumRSS）。
type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Items         []rssItem `xml:"item"`
}

// rssItem 是一則貼文。description 帶全文（貼文本身通常很短，摘要在這個
// 站上沒有比全文更省的理由；而需要摘要的閱讀器會自己截短）。
type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	PubDate     string  `xml:"pubDate"`
	GUID        rssGUID `xml:"guid"`
}

// rssGUID 是永久連結形式的識別碼。isPermaLink="true" 告訴閱讀器這個值
// 可以當網址開 —— 它確實就是貼文頁的網址。
type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// handleForumRSS 產生並送出 RSS 摘要。
//
// 公開端點：不驗 Origin（GET 沒有跨站寫入的問題）、不掛限流（與公開貼文
// 列表同一個理由：它是唯讀的公開內容，而限流匿名讀者是搜尋引擎與閱讀器
// 會踩到的路徑）。成本的上界由 LIMIT 與 ETag 共同守住。
func (s *Server) handleForumRSS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, content, created_at FROM forum_posts
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, forumRSSItemLimit)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 載入 RSS 貼文失敗: %v", err)
		internalError(w, "unable to load feed")
		return
	}
	defer rows.Close()

	// 空結果也要是一個合法的 RSS（items 為空），而不是 404：訂閱一個還沒有
	// 貼文的站是合法狀態，而閱讀器對 404 的處理是「這個 feed 壞了」。
	items := make([]rssItem, 0, forumRSSItemLimit)
	for rows.Next() {
		var id int64
		var content string
		var createdAt time.Time
		if err := rows.Scan(&id, &content, &createdAt); err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 讀取 RSS 貼文失敗: %v", err)
			internalError(w, "unable to read feed")
			return
		}
		link := strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/forum/post/" + strconv.FormatInt(id, 10)
		title := forumRSSTitle(content)
		if title == "" {
			// 內容不可能是空的（建立與編輯都擋掉空白），但標題折疊之後為空
			// 的理論情況仍要有可讀的替代值，而不是一個空白標題。
			title = "貼文 #" + strconv.FormatInt(id, 10)
		}
		items = append(items, rssItem{
			Title:       title,
			Link:        link,
			Description: content,
			PubDate:     createdAt.UTC().Format(rssTimeLayout),
			GUID:        rssGUID{IsPermaLink: "true", Value: link},
		})
	}
	if err := rows.Err(); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取 RSS 貼文串流失敗: %v", err)
		internalError(w, "unable to read feed")
		return
	}

	description := s.cfg.ForumDescription
	if description == "" {
		description = s.cfg.ForumName
	}
	// lastBuildDate 由最新一則貼文（items[0]；查詢是 created_at DESC）的時間
	// 推導，而不是 time.Now()。理由只有一個，但它是這一節的全部：這份文件
	// 帶 ETag（整份文件的 SHA-256），而 now() 每一秒都不同 —— 用它會讓
	// 「內容完全沒變」的兩次請求得到不同的雜湊，304 因此永遠不會發生，
	// 閱讀器每次輪詢都重抓整份文件。貼文的時間是內容的一部分：內容沒變，
	// 它就不變。編輯既有貼文不會改變這個值，而那份文件的雜湊仍然會變
	//（內容本身在雜湊裡），因此這不是「用時間假裝內容沒變」。
	lastBuildDate := ""
	if len(items) > 0 {
		lastBuildDate = items[0].PubDate
	}
	document := rssDocument{
		Version: "2.0",
		Channel: rssChannel{
			Title:       s.cfg.ForumName,
			Link:        strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/forum",
			Description: description,
			// 介面有十七種語言，但一份 feed 只有一種：站上的內容以繁體中文
			// 為主（預設語系），而語言標籤描述的是內容而不是讀者的介面設定。
			Language:      "zh-TW",
			LastBuildDate: lastBuildDate,
			Items:         items,
		},
	}
	body, err := xml.Marshal(document)
	if err != nil {
		// xml.Marshal 對這些型別不會失敗（全是字串與字串切片），但錯誤
		// 不能吞：吞掉的話回應會是一個空 body 加 200，而閱讀器只會說
		// 「這個 feed 沒有內容」。
		logger.ErrorfContext(r.Context(), "[FORUM] 編碼 RSS 失敗: %v", err)
		internalError(w, "unable to encode feed")
		return
	}
	payload := xml.Header + string(body) + "\n"

	// ETag 由整份文件算出：任何欄位改變（新貼文、編輯、刪除、還原）都會
	// 改變雜湊，因此不會出現「資料變了卻回 304」的錯誤快取。
	sum := sha256.Sum256([]byte(payload))
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	w.Header().Set("ETag", etag)
	// 5 分鐘：公開內容，允許中間快取；閱讀器多半自己也會照這個節奏輪詢。
	w.Header().Set("Cache-Control", "public, max-age=300")
	if r.Header.Get("If-None-Match") == etag {
		// 304 不帶 Content-Type 與 Content-Length：帶了會讓某些客戶端把
		// 304 當成有 body 的回應來讀。
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write([]byte(payload)); err != nil {
		// 寫出失敗（連線已斷開）沒有可補救的副作用，只記一行讓斷線風暴
		// 之外的個案有線索。
		logger.WarnfContext(r.Context(), "[FORUM] 寫出 RSS 失敗: %v", err)
	}
}

// forumRSSTitle 從貼文內容取出標題：折疊所有空白後取開頭，過長時截斷。
//
// 貼文沒有標題欄位（這個站沒有「標題＋內文」的結構），因此標題由內文
// 衍生。折疊空白（strings.Fields 再 join）讓多行貼文的標題是一行正常
// 的句子而不是原樣的換行；60 個字元的上限與後台內容統計的摘要同一個
// 量級，足夠辨識一則貼文，又不會讓訂閱清單被單一長文撐開。
//
// 純函式（不吃 DB）是刻意的：截斷行為是這份輸出的可測試契約，
// 表格測試直接釘住它。
func forumRSSTitle(content string) string {
	const maxTitleRunes = 60
	normalized := strings.Join(strings.Fields(content), " ")
	runes := []rune(normalized)
	if len(runes) <= maxTitleRunes {
		return normalized
	}
	return string(runes[:maxTitleRunes]) + "…"
}
