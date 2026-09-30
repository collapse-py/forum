/*
本檔案是論壇「一般使用者端」的 HTTP 處理層，只負責公開 API 與前端頁面的組版；
管理員後台的路由在 forum_admin_handlers.go 與 user_admin_handlers.go，
路由註冊、登入與停權中介層在 server.go，共用回應輔助函式在 response.go，
Origin 檢查在 csrf.go。

一、HTTP 路由對照（註冊於 server.go 的 Handler()）

	GET  /api/forum/posts                     -> listForumPosts          列出文章（分頁）
	POST /api/forum/posts                     -> createForumPost         建立文章
	POST /api/forum/images                    -> handleForumImageUpload  上傳附圖並轉發檔案伺服器
	POST /api/forum/image-tokens/release      -> handleForumImageTokensRelease 釋放圖片存取 token
	*    /api/forum/posts/{id}/comments       -> handleForumComments     留言列表／新增留言
	*    /api/forum/posts/{id}/comments/{cid}/report -> handleForumReport(targetType="comment")
	POST /api/forum/posts/{id}/like           -> handleForumPostLike     按讚／取消按讚（切換）
	POST /api/forum/posts/{id}/report         -> handleForumReport(targetType="post")
	GET  /api/forum/public-profile?key=       -> handleForumPublicProfile 依 publicKey 讀取公開資料
	GET  /api/forum/public-posts?user=        -> handleForumPublicPosts 依 publicKey 讀取該使用者的貼文分頁
	GET|PUT /api/forum/profile                -> handleForumProfile      讀取／更新自己的資料
	GET|POST /api/forum/follows               -> listForumFollows / toggleForumFollow 追蹤清單與切換
	GET  /api/forum/following/posts           -> handleForumFollowingPosts 追蹤者的貼文動態
	GET  /forum/login                         -> handleForumLoginPage    登入頁殼
	GET  /forum, /forum/new,
	     /forum/profile, /forum/others-profile,
	     /forum/following                     -> handleForumPage         論壇前端 SPA 殼

	子資源（comments / like / report）都掛在同一個 /api/forum/posts/ 註冊點上，
	由 handleForumPostAction 依路徑後綴分派，詳見該函式說明。
	追蹤的兩個端點在 forum_follow_handlers.go，不在本檔案。

	/forum/following 沒有另註冊路由：它落在 server.go 的 /forum/ 子樹樣式上，
	而那條路由掛了 requireLogin，因此未登入者會先被導去登入頁。

二、資料表依賴

	    forum_posts                文章主檔（image_url 只存檔名，完整網址由應用層組成）
	forum_post_likes           按讚，主鍵 (post_id, author_email)，天然去重
	forum_post_comments        留言，無外鍵，需自行檢查父文章存在
	forum_reports              檢舉，唯一鍵 (reporter_email, target_type, target_id)
	forum_profiles             暱稱與簡介，主鍵 author_email、public_key 有索引
	forum_follows              追蹤，主鍵 (follower_email, target_email)（在 forum_follow_handlers.go）
	forum_user_tags /
	forum_user_tag_assignments 使用者標籤（由管理員指派），僅在文章列表回傳
	forum_users                帳號狀態；停權檢查由 server.go 的 requireLogin 執行，本檔案不查

三、關鍵設計決策
 1. 權限分層：登入與停權判定放在中介層（requireLogin / requireLoginForWrite），
    本檔案在每個 handler 開頭再做一次輕量檢查，屬縱深防禦，避免日後有人
    註冊新路由時漏掉中介層就意外開放寫入。
 2. 縱深防禦不取代 Origin 檢查：所有會改動資料的分支都另外呼叫 isTrustedOrigin，
    因為中介層的判斷依據是 session cookie，而 cookie 會隨跨站表單 POST 一起送出。
 3. 個資不外洩：email 永不出現在回應中，對外一律以 publicForumKey（email 的
    SHA-256 十六進位）代表身分，顯示名稱則取暱稱或隨機匿名代號。
 4. 圖片不落地論壇服務：檔案交給獨立的檔案伺服器，論壇只保存它的檔名；
    完整網址在回應時才以 FILES_SERVER_PUBLIC_URL 組出，資料庫不存環境相關資訊。
    讀取圖片需要短效 token，token 存在 Redis，由本服務簽發、也由本服務刪除。
 5. 分頁採 OFFSET + hasMore 推測而非游標分頁，因為論壇貼文是 append-only 且
    會持續有新文插入，游標分頁在此情境反而容易漏讀；代價見 listForumPosts 說明。
 6. 只有「讀取狀態再寫入」的切換需要交易（按讚、追蹤），其餘寫入都是單一語句，
    額外開交易只增加往返次數。
 7. 所有 DB 錯誤都先寫入 logger，再對外回傳固定的通用訊息，避免把 SQL 與
    schema 細節洩漏給使用者。

四、已知限制（刻意保留，非本檔案可自行修正）
  - Server.rateLimiter 雖已建立，但 server.go 未把它掛到任何路由上，
    因此本檔案的寫入端點目前沒有速率限制。
  - MySQL schema 未定義外鍵，父資源存在性一律以 SELECT COUNT(*) 檢查，
    交易隔離層級為預設的 REPEATABLE READ，理論上仍可能出現競態（見 handleForumPostLike）。
*/
package httpapi

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"forum/forum/logger"

	"github.com/go-sql-driver/mysql"
)

// forumPost 是文章列表與建立文章回應共用的傳輸物件。
//
// 隱私設計：Author 是「顯示名稱」而非 email，AuthorKey 是 email 的雜湊值，
// 前端以 AuthorKey 當 React/Vue 的 key，並以它呼叫 /api/forum/public-profile，
// 這樣公開 API 全程不需要也不會洩漏 email。
//
// 欄位說明（註解刻意置於型別之外，避免打斷欄位原本的欄位對齊排版）：
//
//	ID           對應 forum_posts.id，由 MySQL AUTO_INCREMENT 產生。
//	Author       顯示名稱：優先取 forum_profiles.nickname，取不到則為隨機匿名代號。
//	AuthorKey    publicForumKey(email) 的結果，長度固定 64 個十六進位字元。
//	AuthorTags   管理員指派的標籤；未指派時為空切片，因 omitempty 而省略欄位。
//	Content      貼文本文，伺服器端已 TrimSpace 且限制在 10000 個 rune 以內。
//	CreatedAt    直接對應 forum_posts.created_at，時區格式化交給前端。
//	LikeCount    該文的按讚總數，由 SQL 的相關子查詢即時 COUNT 而非快取欄位。
//	Liked        「目前這個請求的登入者」是否已按讚；未登入時恆為 false。
//	CommentCount 留言總數；刻意同時回傳，讓前端未展開留言時就能顯示數字，
//	              避免為了顯示計數而額外請求一次留言列表。
//	ImageURL     由庫中檔名組出的公開網址（已附加 token）；無圖時為空字串，
//	             因 omitempty 而省略。
type forumPost struct {
	ID           int64     `json:"id"`
	Author       string    `json:"author"`
	AuthorKey    string    `json:"authorKey"`
	AuthorTags   []string  `json:"authorTags,omitempty"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
	LikeCount    int       `json:"likeCount"`
	Liked        bool      `json:"liked"`
	CommentCount int       `json:"commentCount"`
	ImageURL     string    `json:"imageUrl,omitempty"`
}

// forumComment 是留言列表與新增留言回應共用的傳輸物件。
// 欄位刻意比 forumPost 少：留言不顯示標籤，也不回傳按讚狀態。
//
//	ID        對應 forum_post_comments.id。
//	Author    顯示名稱，語意與 forumPost.Author 相同。
//	AuthorKey email 的雜湊值，供前端作為穩定識別碼。
//	Content   留言本文，已 TrimSpace 且限制在 2000 個 rune 以內。
//	CreatedAt 對應 forum_post_comments.created_at；列表以它遞增排序。
type forumComment struct {
	ID        int64     `json:"id"`
	Author    string    `json:"author"`
	AuthorKey string    `json:"authorKey"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// createForumCommentRequest 是 POST /api/forum/posts/{id}/comments 的請求主體。
// 只有 Content 欄位；留言所屬文章由路徑決定，不接受前端指定。
//
//	Content 去除前後空白後必須為 1 至 2000 個 rune，驗證規則在 handler 內。
type createForumCommentRequest struct {
	Content string `json:"content"`
}

// createForumPostRequest 是 POST /api/forum/posts 的請求主體。
//
//	Content  去除前後空白後必須為 1 至 10000 個 rune。
//	ImageURL 為 /api/forum/images 上傳後回傳的網址；寫入前只做 TrimSpace
//	         並取出檔名，取不到就回 400（不接受第三方網址，見 createForumPost）。
type createForumPostRequest struct {
	Content  string `json:"content"`
	ImageURL string `json:"imageUrl"`
}

// forumReportRequest 是檢舉端點的請求主體，post 與 comment 共用。
//
//	Reason 去除前後空白後長度須介於 1 至 500 個 rune，
//	       上限與 forum_reports.reason VARCHAR(500) 的字元語意一致。
type forumReportRequest struct {
	Reason string `json:"reason"`
}

// forumProfileRequest 是 PUT /api/forum/profile 的請求主體。
//
//	Nickname 去除前後空白後須為 1 至 30 個 rune；
//	         forum_profiles.nickname 有 UNIQUE 索引 uq_forum_profiles_nickname，撞名會回 409。
//	Bio      可為空字串，非空時不得超過 500 個 rune。
type forumProfileRequest struct {
	Nickname string `json:"nickname"`
	Bio      string `json:"bio"`
}

// handleForumPosts 是 /api/forum/posts 的單一進入點，依 HTTP 方法分派。
//
// 登入與停權由 server.go 的 requireLoginForWrite 負責：GET 允許未登入者閱讀
// （但 liked 會是 false），其餘方法必須已登入且未被停權。
func (s *Server) handleForumPosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listForumPosts(w, r)
	case http.MethodPost:
		s.createForumPost(w, r)
	default:
		// 沒有在此處補 Allow 標頭，直接以 405 結束，前端只需處理單一錯誤路徑。
		methodNotAllowed(w)
	}
}

// forumPostProjection 是公開頁所有貼文查詢共用的 SELECT 欄位清單。
//
// 為什麼是常數而不是各查詢各寫一份：動態列表、追蹤動態與搜尋結果三者都會把
// 結果交給同一支前端元件渲染，因此欄位集合必須完全一致。少一欄不會讓編譯器
// 或任何測試失敗，只會讓該欄在某一種來源下恆為零值 —— 例如後來加上
// 「我是否追蹤了這位作者」時漏改搜尋那一處，症狀是「搜尋結果裡的追蹤鈕永遠
// 顯示未追蹤」，而且只在特定頁面出現。寫成單一字串之後，欄位集合不再是三份
// 需要各記各的約定。
//
// 唯一的佔位符是 liked_by_me 的「目前這個請求的登入者」。呼叫端必須把它放在
// 所有 condition 佔位符之前；未登入時傳空字串，該 EXISTS 子查詢自然恆為 false，
// 因此讀取端點不需要額外的登入判斷。
//
// 追蹤狀態刻意不在這裡：它是「頁面層的一次查詢結果」（見 forum_follow_handlers.go
// 的 handleForumFollows），不是每篇貼文的屬性。放進這個投影會讓三個查詢各多一
// 個相關子查詢，卻換不來任何好處 —— 同一頁的追蹤清單只讀一次就夠了。
const forumPostProjection = `fp.id, fp.author_email, fp.content, fp.created_at, fp.image_url,
		(SELECT COUNT(*) FROM forum_post_likes WHERE post_id = fp.id) AS like_count,
		(SELECT COUNT(*) FROM forum_post_comments WHERE post_id = fp.id) AS comment_count,
		CASE WHEN EXISTS (
		    SELECT 1 FROM forum_post_likes
		    WHERE post_id = fp.id AND author_email = ?
		) THEN 1 ELSE 0 END AS liked_by_me`

// errMediaTokenUnavailable 表示簽發圖片存取 token 失敗（通常是 Redis 不可用）。
//
// 存在的理由是狀態碼要分開：Redis 掛掉代表檔案伺服器將拒絕這批圖片，屬上游故障，
// 因此以 502（Bad Gateway）表達才不會被誤讀成「本站的資料庫有問題」而讓維運
// 查錯方向。共用流程把錯誤往上收之後，若不帶著這個區別，呼叫端就只能一律回 500。
var errMediaTokenUnavailable = errors.New("unable to create media token")

// loadForumPosts 執行一條公開頁的貼文查詢並把結果組成可送出的 []forumPost。
//
// condition 與 conditionArgs 是「這一次要看的哪一批貼文」：全站（空字串）、
// 追蹤中的人（`fp.author_email IN (SELECT target_email FROM forum_follows WHERE
// follower_email = ?)`）。刻意讓呼叫端以條件字串表達而不是傳一個函式過來，
// 是因為兩種條件的差異只在 WHERE 子句，而其餘流程（計數、去識別化、圖片 token）
// 完全相同 —— 讓它們共用這一支，欄位與處理順序就不可能漂移。
//
// 三個逐列查詢（暱稱、標籤、圖片 token）是已知的 N+1：主查詢要維持單一、可走
// idx_forum_posts_created_at 的形態，而把標籤 join 進去會因一對多而複製文章列，
// 反而更貴。單頁筆數由 pageSize 封頂，因此最壞情況可控。
func (s *Server) loadForumPosts(r *http.Request, condition string, conditionArgs []interface{}, pageSize, offset int) ([]forumPost, error) {
	ctx := r.Context()
	// 參數順序即 SQL 裡 ? 的出現順序：投影的 currentUser → condition → LIMIT/OFFSET。
	args := make([]interface{}, 0, len(conditionArgs)+3)
	args = append(args, s.sessions.ResolveUser(r))
	args = append(args, conditionArgs...)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, `
		SELECT `+forumPostProjection+`
		FROM forum_posts fp `+condition+`
		ORDER BY fp.created_at DESC, fp.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		// 先記錄原始錯誤再回通用訊息：使用者不需要知道是哪一段 SQL 失敗，
		// 但維運需要看到足以定位的資訊。
		logger.ErrorfContext(ctx, "[FORUM] 載入文章失敗: %v", err)
		return nil, err
	}
	// rows 必須在函式返回前關閉，否則連線不會歸還連線池。
	defer rows.Close()

	// 用 make(..., 0) 而非 var：空結果會序列化成 [] 而不是 null，前端可直接 map 而不必防禦 null。
	posts := make([]forumPost, 0)
	// 整頁共用同一個 media token。前端離開頁面時只呼叫一次 release 就能釋放整頁圖片，
	// 也讓每個列表請求在 Redis 只增加一個 key。
	mediaToken := ""
	for rows.Next() {
		var post forumPost
		// CASE WHEN EXISTS 的結果以 0/1 呈現，MySQL 沒有原生布林型別。
		var liked int
		if err := rows.Scan(&post.ID, &post.Author, &post.Content, &post.CreatedAt, &post.ImageURL, &post.LikeCount, &post.CommentCount, &liked); err != nil {
			logger.ErrorfContext(ctx, "[FORUM] 讀取文章失敗: %v", err)
			return nil, err
		}

		post.Liked = liked == 1
		// 先把庫值（純檔名）取出來，並用它判斷「這頁有沒有圖」：舊資料若存的是
		// 無法辨識的值，這裡會得到空字串，而不會為了它去簽發一個沒人用得到的
		// token。token 延遲到真的有圖片時才簽發，純文字頁完全不碰 Redis。
		imageName := s.forumImageFileName(post.ImageURL)
		if imageName != "" && mediaToken == "" {
			mediaToken, err = s.createMediaToken(ctx)
			if err != nil {
				logger.ErrorfContext(ctx, "[FORUM] 建立圖片 token 失敗: %v", err)
				return nil, fmt.Errorf("%w: %v", errMediaTokenUnavailable, err)
			}
		}
		// 庫值只是檔名，給瀏覽器的完整網址在此現組：公開主機 + /files/ + 檔名
		// + token。任何一步失敗都會得到空字串（前端因此不顯示縮圖），
		// 絕不會把內部位址送出去。
		post.ImageURL = s.forumImageURL(imageName, mediaToken)
		// 先把原始 email 存起來再覆寫 Author：下游都需要 email 來查暱稱、
		// 雜湊與標籤，而 Author 欄位對外只能放顯示名稱。
		authorEmail := post.Author
		post.Author = s.forumAuthor(r, authorEmail)
		post.AuthorKey = publicForumKey(authorEmail)
		post.AuthorTags, err = s.forumAuthorTags(ctx, authorEmail)
		if err != nil {
			// 標籤是裝飾性資訊，但此處選擇讓整個請求失敗而非安靜地省略，
			// 避免同一篇文章在不同請求中出現不同欄位組合而干擾前端渲染。
			logger.ErrorfContext(ctx, "[FORUM] 讀取用戶標籤失敗: %v", err)
			return nil, err
		}
		// 逐筆 INFO 記錄 post_id 與作者（email 僅進日誌，不進 HTTP 回應），
		// 讓異常流量（大量洗文）可以在日誌中以 tag_count 異常偏低等訊號察覺。
		logger.InfofContext(ctx, "[FORUM] post_id=%d author=%s tag_count=%d", post.ID, authorEmail, len(post.AuthorTags))
		posts = append(posts, post)
	}
	// rows.Err() 必須在迴圈結束後另外檢查：迭代過程中斷（例如連線被回收）
	// 不會讓 Next() 回傳 false 以外的訊號，漏掉這一檢查會把部分結果當成完整結果。
	if err := rows.Err(); err != nil {
		logger.ErrorfContext(ctx, "[FORUM] 讀取文章串流失敗: %v", err)
		return nil, err
	}
	return posts, nil
}

// listForumPosts 回傳依 created_at DESC, id DESC 排序的文章分頁（全站動態）。
//
// 分頁設計：呼叫端只給 offset，沒有 page 參數。hasMore 以「回傳筆數是否等於
// pageSize」推導——滿頁即猜測還有下一頁，寧可多給一次空結果，也不要為了確認
// 尾端而多跑一次 COUNT(*)。
//
// 為什麼不用游標分頁（keyset pagination）：論壇貼文是只增不改的，理論上適合
// keyset，但論壇的使用情境包含「回到上一頁重看」與分享特定頁面，游標分頁必須
// 由呼叫端保存游標，實作與前端狀態都更複雜；而貼文表已有 idx_forum_posts_created_at
// 且單頁僅 25 筆，OFFSET 的代價（Server 端仍須掃過並丟棄前 offset 筆）在這個規模
// 可接受。offset 未設上限是刻意的：上限會讓深翻頁靜默失效，不如由呼叫端自律。
func (s *Server) listForumPosts(w http.ResponseWriter, r *http.Request) {
	// 單頁筆數。與前端 FeedPage 的初始抓取量一致，25 筆足以填滿一頁又不至於
	// 讓 loadForumPosts 的 per-row 查詢（暱稱、標籤）爆炸。
	// 追蹤動態（handleForumFollowingPosts）刻意用同一個常數：兩個動態的
	// 單頁成本完全相同，給不同的值只會讓「兩種動態的分頁行為」產生無意義的差異。
	const pageSize = 25
	const pageMore = pageSize

	offset, ok := forumOffsetParam(w, r)
	if !ok {
		return
	}

	posts, err := s.loadForumPosts(r, "", nil, pageSize, offset)
	if err != nil {
		s.writeForumPostLoadError(w, err, "unable to load forum posts")
		return
	}

	writeOK(w, map[string]interface{}{
		"items":   posts,
		"hasMore": len(posts) == pageMore,
	})
}

// writeForumPostLoadError 把 loadForumPosts 的錯誤翻譯成對外的回應。
//
// 唯一需要分流的是 errMediaTokenUnavailable（上游 Redis／檔案伺服器故障 → 502），
// 其餘一律 500 + 通用訊息。抽出來是為了讓追蹤動態與搜尋結果共用同一組判斷，
// 免得三處各自記得「哪一種錯誤該回 502」。
func (s *Server) writeForumPostLoadError(w http.ResponseWriter, err error, message string) {
	if errors.Is(err, errMediaTokenUnavailable) {
		writeError(w, http.StatusBadGateway, "unable to create media token")
		return
	}
	internalError(w, message)
}

// forumOffsetParam 解析 ?offset=，非法值直接寫出 400 回應並回傳 false。
//
// 三個列表端點（動態、追蹤動態、追蹤清單）共用這一段：負數或非數字一律 400，
// 讓 offset 一路帶進 SQL 會產生語意不明的查詢，與其在 DB 層才發現錯誤，
// 不如在入口擋掉。上限刻意不設 —— 上限會讓深翻頁靜默失效，不如由呼叫端自律。
func forumOffsetParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	if r.URL.Query().Get("offset") == "" {
		return 0, true
	}
	parsed, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || parsed < 0 {
		badRequest(w, "invalid offset")
		return 0, false
	}
	return parsed, true
}

// handleForumImageUpload 接收瀏覽器選定的圖片，轉發給獨立的檔案伺服器保存，
// 然後回傳一個「已附加短效 token」的公開網址給前端。
//
// 回傳的網址是「FILES_SERVER_PUBLIC_URL + /files/ + 檔名 + ?token=」，
// 與資料庫的儲存契約（只存檔名）互相配合：前端把這個值送回
// createForumPost 時，後端只會取出其中的檔名寫入庫中。
//
// 為什麼不直接寫本機磁碟：論壇服務只負責業務邏輯，檔案實際存放在
// files_server（可設定為本機磁碟或 S3）。如此一來擴充多個論壇實例時
// 圖片不會被綁死在單一機器的本機檔案系統上，也能把上傳的驗證
// （副檔名白名單、大小）集中在一處。
//
// 前端絕不會看到 s.cfg.FilesServerToken，那個共用密鑰只存在後端與檔案伺服器之間。
func (s *Server) handleForumImageUpload(w http.ResponseWriter, r *http.Request) {
	// 路由雖已用 requireLogin 包住，仍重複檢查方法，理由同其他 handler：
	// 讓 handler 本身自足，日後改動路由註冊時不易出現意外開放。
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	// 跨站表單 POST 會帶著 session cookie，因此上傳這種會消耗磁碟／儲存體的動作
	// 必須另外驗 Origin。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	// 50<<20 是「記憶體門檻」而非硬性大小上限：超過的內容會被 Go 暫存到
	// 系統暫存檔而非直接拒絕。真正的硬上限在檔案伺服器端（MaxBytesReader +
	// 50MB 上限），這裡只是先擋掉明顯過大的表單，避免在解析階段就吃光記憶體。
	// 兩邊的門檻刻意對齊，避免使用者等檔案伺服器回 413 才知道圖片太大。
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		badRequest(w, "invalid image upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		badRequest(w, "image file is required")
		return
	}
	// FormFile 回傳的是 multipart.File，必須關閉；底層可能是實體檔案也可能是記憶體。
	defer file.Close()

	// 把已解析的檔案重新包裝成一份新的 multipart body 再送給檔案伺服器。
	// 這一步不是多餘：它同時做了三件事——
	//   (1) 只轉送 "file" 這一個欄位，攻擊者夾帶的其他表單欄位不會被轉發；
	//   (2) 檔名以 filepath.Base 正規化，剝掉 ../../etc/passwd 這類路徑片段
	//       （檔案伺服器正是以檔名作為儲存鍵）；
	//   (3) 產生新的 boundary 與 Content-Type，避免直接沿用外部請求的邊界。
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(header.Filename))
	if err != nil {
		internalError(w, "unable to prepare image upload")
		return
	}
	if _, err := io.Copy(part, file); err != nil {
		// 讀取失敗代表 client 提前斷線或暫存檔有問題，此時不重試，直接結束。
		internalError(w, "unable to read image upload")
		return
	}
	// 必須 Close 才會寫出結尾邊界，否則檔案伺服器解析不到表單結尾。
	if err := writer.Close(); err != nil {
		internalError(w, "unable to prepare image upload")
		return
	}

	// FilesServerURL 是內網位址（例：192.168.66.5:7070），瀏覽器無法直接連線，
	// 這正是論壇服務充當轉發點的原因。TrimRight 避免設定值尾斜線造成 //upload。
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(s.cfg.FilesServerURL, "/")+"/upload", &body)
	if err != nil {
		internalError(w, "unable to create image upload request")
		return
	}
	// boundary 只能由 writer 產生，因此 Content-Type 必須在建立 body 之後才設定。
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if s.cfg.FilesServerToken != "" {
		// 未設定時不送這個標頭，檔案伺服器端 Upload.Token 為空等同關閉上傳驗證。
		req.Header.Set("X-Upload-Token", s.cfg.FilesServerToken)
	}
	// 60 秒上限涵蓋檔案伺服器連線 S3 的最壞情況；逾時視為上游故障。
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 圖片上傳失敗: %v", err)
		internalError(w, "unable to upload image")
		return
	}
	defer resp.Body.Close()
	// 檔案伺服器可能以 401（token 不符）、413（過大）、415（副檔名不允許）回應。
	// 這些都不該洩漏細節給使用者，但必須和「自己出錯」區隔開，所以用 502。
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.ErrorfContext(r.Context(), "[FORUM] 檔案伺服器回應錯誤 status=%d", resp.StatusCode)
		writeError(w, http.StatusBadGateway, "file server rejected image")
		return
	}
	// 只取 url 欄位，其餘欄位（若日後新增）一律忽略，避免上游擴充欄位就影響本檔案。
	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.URL == "" {
		// 上游回應格式不符預期時視為故障，並在 log 留下 err 供排查。
		internalError(w, "invalid file server response")
		return
	}
	// 檔案伺服器回傳的 url 有兩種形貌：本機儲存時 getBaseURL() 為空字串，
	// url 只有 "/files/xxx"；S3 儲存時則是完整的物件網址。兩者都由
	// forumImageFileName 收斂成純檔名——不接受其他主機，避免有人日後把
	// FILES_SERVER_URL 指向別的服務時，攻擊者就能指定任意網址被存進庫中。
	// 副檔名也在這裡就被擋掉（允許清單見 validForumImageFileName），
	// 不信任上游的驗證結果。
	name := s.forumImageFileName(result.URL)
	if name == "" {
		logger.ErrorfContext(r.Context(), "[FORUM] 檔案伺服器回傳無法辨識的圖片網址: %q", result.URL)
		internalError(w, "invalid uploaded image URL")
		return
	}
	token, err := s.createMediaToken(r.Context())
	if err != nil {
		// 檔案已存檔但取不到 token，此時使用者拿不到網址，會看到錯誤並重新上傳，
		// 孤兒檔案由檔案伺服器端的 TTL／人工清理處理。
		logger.ErrorfContext(r.Context(), "[FORUM] 建立圖片 token 失敗: %v", err)
		writeError(w, http.StatusBadGateway, "unable to create media token")
		return
	}
	// 立刻簽發 token，讓上傳後的預覽可以馬上載入，不必等貼文建立後重新列表。
	// 回傳的是公開網址（FILES_SERVER_PUBLIC_URL），前端拿到的這個值會原樣
	// 送回 createForumPost，而後者只取其中的檔名寫入資料庫。
	writeJSON(w, http.StatusCreated, map[string]string{"url": s.forumImageURL(name, token)})
}

// createMediaToken 產生一組 32 個亂數位元的 token，寫入 Redis 並回傳其十六進位字串。
//
// 這個 token 是檔案伺服器存取 /files/ 資源的憑證：檔案伺服器的中介層會
// 檢查查詢字串的 token 是否存在於 Redis（且兩邊的 key 前綴必須一致）。
// 存 Redis 而非簽章，是為了能主動刪除——見 handleForumImageTokensRelease。
//
// 副作用：在 mediaRedis 建立一個帶 TTL 的 key。TTL 由 MediaTokenTTLSecs 決定
// （實務設定為 60 秒），即使前端從未呼叫釋放，token 也會自動過期。
func (s *Server) createMediaToken(ctx context.Context) (string, error) {
	// mediaRedis 為 nil 代表建構時就沒連上 Redis；此時寧可回錯讓呼叫端顯示
	// 502，也不要發出一個永遠無效的 token 讓前端載入失敗。
	if s.mediaRedis == nil {
		return "", fmt.Errorf("media token service unavailable")
	}
	rawToken := make([]byte, 32)
	// 使用 crypto/rand 而非 math/rand：這是存取憑證，必須密碼學安全且不可預測。
	if _, err := cryptorand.Read(rawToken); err != nil {
		return "", err
	}
	// 32 bytes → 64 個十六進位字元。這個長度同時被釋放端點的格式驗證倚賴。
	token := hex.EncodeToString(rawToken)
	// Redis 是網路 I/O，設 2 秒上限避免拖慢整個列表回應；沿用呼叫端的 ctx
	// 讓上游取消（client 斷線）時也能一併中止。
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.mediaRedis.Set(redisCtx, s.cfg.MediaTokenKeyPrefix+token, "1", time.Duration(s.cfg.MediaTokenTTLSecs)*time.Second).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// handleForumImageTokensRelease 讓前端在「已經看完圖片」之後提早作廢存取 token。
//
// 為什麼需要釋放：token 是在把圖片網址交給瀏覽器時簽發的，但那份 HTML／
// SPA 狀態可能只是被關閉、可能使用者放棄一篇草稿。若只靠 TTL，孤立檔案
// 仍會在 TTL 內保持可被持有該網址者存取。前端在 pagehide 時以 sendBeacon
// 呼叫本端點（forum.js），或在放棄預覽圖片時呼叫（forum-new.js），
// 讓存取權提早失效——屬於 fail-closed：提前失效只會讓圖片看不見，
// 不會讓任何人獲得額外存取。
func (s *Server) handleForumImageTokensRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if s.mediaRedis == nil {
		// 沒有 Redis 就無從談釋放，語意等同全部過期，不該報錯打斷前端流程。
		internalError(w, "media token service unavailable")
		return
	}
	var request struct {
		Tokens []string `json:"tokens"`
	}
	// 64KB 上限：sendBeacon 會夾帶整頁圖片的 token 清單，數十筆綽綽有餘，
	// 上限可擋下惡意塞滿 body 的請求。
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
		badRequest(w, "invalid media token request")
		return
	}
	keys := make([]string, 0, len(request.Tokens))
	for _, token := range request.Tokens {
		// 嚴格驗證：長度必須 64，且去除所有十六進位字元後必須變成空字串。
		// 這是必要的，因為下一行會把字串直接拼成 Redis key。若不驗，
		// 呼叫端就能用任意字串構造出攻擊者選擇的 key，交給 DEL 刪除——
		// 也就是把一個「釋放自己的 token」的端點變成「刪除任意 Redis key」。
		if len(token) != 64 || strings.Trim(token, "0123456789abcdef") != "" {
			continue
		}
		keys = append(keys, s.cfg.MediaTokenKeyPrefix+token)
	}
	if len(keys) == 0 {
		// 全部被格式檢查淘汰時直接回 0，讓前端可以無條件把結果當成功處理。
		writeOK(w, map[string]int{"deleted": 0})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	// 單次 DEL 接受多個 key，比逐筆呼叫省往返；Redis 會回傳實際刪除數，
	// 因此重複釋放同一個 token 不會造成錯誤。
	deleted, err := s.mediaRedis.Del(ctx, keys...).Result()
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 刪除圖片 token 失敗: %v", err)
		writeError(w, http.StatusBadGateway, "unable to release media tokens")
		return
	}
	writeOK(w, map[string]int64{"deleted": deleted})
}

// firstForumImageName 回傳清單中第一張「可對應到本站檔案伺服器」的圖片檔名，
// 整份清單都沒有可用圖片時回傳空字串。
//
// 用途是決定值不值得簽發 media token：呼叫端據此跳過 createMediaToken，
// 讓純文字頁完全不碰 Redis（與 listForumPosts 的延遲簽發同一個考量）。
func (s *Server) firstForumImageName(posts []adminForumPost) string {
	for _, post := range posts {
		if name := s.forumImageFileName(post.ImageURL); name != "" {
			return name
		}
	}
	return ""
}

// forumImageFileName 把 image_url 的值正規化為「純檔名」，無法對應到本站
// 檔案伺服器上的圖片時回傳空字串。
//
// forum_posts.image_url 的儲存契約只放檔名（例如 "9f8c1a.png"），完整的存取
// 網址一律由 forumImageURL 以 FILES_SERVER_PUBLIC_URL 組出來。這樣換網域
// 只需改設定、不必搬資料，也不會讓內部位址（http://192.168.66.5:7070）
// 洩漏進資料庫。
//
// 為了相容改契約之前既有的資料，這裡除了純檔名，也接受三種舊寫法並取其檔名：
// 絕對路徑（/files/x.png）、檔案伺服器的內部位址（FILES_SERVER_URL 或
// localhost／127.0.0.1／::1）、以及公開網域。查詢字串（?token=…）一律丟棄。
// 舊資料因此不必等遷移跑完才正確顯示，兩邊可以各自演進。
//
// 其他主機一律視為「不是本站的圖片」：這同時讓舊版允許的第三方圖片連結失效
// （CSP 的 img-src 本來就擋掉它們，只是當時未回 400 而已）。
func (s *Server) forumImageFileName(imageURL string) string {
	value := strings.TrimSpace(imageURL)
	if value == "" {
		return ""
	}
	// 純檔名：新契約的正常寫法。這裡就擋掉含路徑分隔符與查詢字串的值，
	// 讓帶 "/" 的舊資料一定會走下面的解析路徑，不會被誤認為檔名。
	if !strings.ContainsAny(value, "/?#") {
		return validForumImageFileName(value)
	}

	// url.Parse 對 "://" 這種沒有合法 scheme 前綴的字串會回傳錯誤而不是 panic。
	parsed, err := url.Parse(value)
	// 只有 /files/ 底下的路徑才是我們的媒體；其他路徑（空字串、上傳端點本身）
	// 沒有對應的靜態檔案，不做猜測。
	if err != nil || !strings.HasPrefix(parsed.Path, "/files/") {
		return ""
	}
	// 絕對網址必須屬於自己的檔案伺服器：公開網域、內部上傳位址與本機測試位址。
	if parsed.IsAbs() && !s.isOwnFilesHost(parsed.Hostname()) {
		return ""
	}
	// path.Base 而非 filepath.Base：這裡處理的是 URL 路徑，必須永遠以 "/" 分隔，
	// 用 filepath 會在 Windows 上把 "\" 也當分隔符，語意與部署平台綁定。
	name := path.Base(parsed.Path)
	// 只接受「/files/<單一檔名>」這種形狀：多層路徑與 ".." 一律拒絕，
	// 否則 "/files/../secret.png" 會被默默指向另一個檔案。檔案伺服器存檔時
	// 影像一律直接放在 /files/ 底下，因此這不會擋掉任何合法上傳。
	if parsed.Path != "/files/"+name {
		return ""
	}
	return validForumImageFileName(name)
}

// isOwnFilesHost 判斷某個主機名稱是否為本站檔案伺服器（公開或內部）。
//
// 公開網域與內部上傳位址是同一台服務的兩種稱呼；另含三種本機寫法，因為
// 開發者用本機檔案伺服器測試時，存進庫的會是 localhost:7070 或 127.0.0.1。
// 刻意不比對埠號：PORT 有可能與反向代理不同，且比對主機名稱已足以分辨
// 「是不是自己人」。
func (s *Server) isOwnFilesHost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	for _, raw := range []string{s.cfg.FilesServerPublicURL, s.cfg.FilesServerURL} {
		parsed, err := url.Parse(strings.TrimRight(raw, "/"))
		// 設定值解析失敗或沒有 host 時跳過該筆，讓比對繼續看下一筆。
		if err != nil || parsed.Host == "" {
			continue
		}
		if parsed.Hostname() == host {
			return true
		}
	}
	return false
}

// validForumImageFileName 檢查檔名是否為「單一檔名」且是允許的圖片類型，
// 合法則原樣回傳，否則回傳空字串。
//
// 限制比檔案伺服器實際會存檔的範圍更嚴，是刻意的縱深防禦：handler 收到的是
// 使用者可控的字串，萬一日後放寬別的輸入來源，這裡仍擋得住「../..」這類
// 路徑穿越與非圖片檔名。前導「.」一律拒絕，理由是隱藏檔與點開頭的相對路徑
// 片段都不是合法的圖片名。
func validForumImageFileName(name string) string {
	if name == "" || len(name) > 128 || strings.HasPrefix(name, ".") {
		return ""
	}
	for _, r := range name {
		isAllowed := r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '.' || r == '_' || r == '-'
		if !isAllowed {
			return ""
		}
	}
	// 副檔名白名單刻意不含 svg：SVG 可以內嵌腳本，放行等於開放儲存型 XSS。
	// 這裡的清單必須與檔案伺服器 upload.allowed_files 相同，否則會出現
	// 「寫得進、載不出」或反之的不一致。
	switch {
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"),
		strings.HasSuffix(name, ".png"), strings.HasSuffix(name, ".gif"),
		strings.HasSuffix(name, ".webp"):
		return name
	}
	return ""
}

// forumImageURL 把資料庫存的檔名組成瀏覽器可存取的公開網址。
//
// 完整網址一律在這裡現組：FILES_SERVER_PUBLIC_URL + /files/ + 檔名（+ token）。
// 這是「資料庫不存環境相關資訊」的具體實作 —— 對外網域改版時只要改設定。
//
// 任何無法對應到允許圖片的值都回傳空字串（含附圖本身的空字串），呼叫端因此
// 不必分辨「沒有圖」與「圖壞了」：兩者在 JSON 都是省略 imageUrl、在畫面上
// 都是不顯示縮圖。公開位址設定無效時同樣回傳空字串，寧可看不到圖，也不要
// 產生一個壞掉的網址寫進瀏覽器歷史。
func (s *Server) forumImageURL(imageURL, token string) string {
	name := s.forumImageFileName(imageURL)
	if name == "" {
		return ""
	}
	publicBase, err := url.Parse(strings.TrimRight(s.cfg.FilesServerPublicURL, "/"))
	if err != nil || publicBase.Scheme == "" || publicBase.Host == "" {
		return ""
	}
	// 以 Path 而非字串拼接：String() 會自行做路徑編碼，檔名含空白或非 ASCII
	// 字元時不必另外呼叫 url.PathEscape。設定若帶了子路徑（例如反向代理到
	// /media），該前綴會被保留。
	publicBase.Path = strings.TrimRight(publicBase.Path, "/") + "/files/" + name
	if token != "" {
		query := publicBase.Query()
		query.Set("token", token)
		publicBase.RawQuery = query.Encode()
	}
	return publicBase.String()
}

// handleForumComments 同時處理留言列表（GET）與新增留言（POST）。
//
// 兩者合併在同一個 handler 的原因：它們共用同一段路徑解析（取出 post_id），
// 而新增留言必須先確認這篇文章存在，兩者對 post_id 的語意要求完全一致。
func (s *Server) handleForumComments(w http.ResponseWriter, r *http.Request) {
	// 先剝掉 /comments 後綴再交給 forumPostIDFromPath；如此一來同一支解析函式
	// 可以同時服務 /posts/{id}、/posts/{id}/like、/posts/{id}/report。
	postID, err := forumPostIDFromPath(strings.TrimSuffix(r.URL.Path, "/comments"))
	if err != nil {
		badRequest(w, "invalid post id")
		return
	}
	if r.Method == http.MethodGet {
		// 預設只給 8 則，因為多數留言不會被展開；前端「看更多」才以 limit 放大。
		// 上限壓在 12 是為了防止有人直接請求 limit=100000 拖垮 DB 與回應體積。
		const initialCommentPageSize = 8
		const commentPageSize = 12
		limit := initialCommentPageSize
		offset := 0
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			// 上限同時擋掉「合法但過大」的值，僅檢查 >0 會讓一筆查詢掃過整張留言表。
			if err != nil || parsed <= 0 || parsed > commentPageSize {
				badRequest(w, "invalid comment limit")
				return
			}
			limit = parsed
		}
		if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
			parsed, err := strconv.Atoi(rawOffset)
			if err != nil || parsed < 0 {
				badRequest(w, "invalid comment offset")
				return
			}
			offset = parsed
		}
		// 留言遞增排序（舊的在上），與文章的遞減排序相反：討論串的閱讀順序
		// 是由上而下。idx_forum_post_comments_post_id 讓 WHERE + ORDER BY 走索引。
		// id 作為次要排序鍵是必要的：created_at 是 DATETIME（秒級精度），
		// 同秒寫入的多筆留言若沒有 id 打破平手，分頁結果會不穩定而漏讀或重複。
		rows, err := s.db.QueryContext(r.Context(), `
			SELECT fc.id, fc.author_email, fc.content, fc.created_at
			FROM forum_post_comments fc
			WHERE fc.post_id = ?
			ORDER BY fc.created_at ASC, fc.id ASC
			LIMIT ? OFFSET ?`, postID, limit, offset)
		if err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 載入留言失敗 post_id=%d: %v", postID, err)
			internalError(w, "unable to load comments")
			return
		}
		defer rows.Close()

		comments := make([]forumComment, 0)
		for rows.Next() {
			var comment forumComment
			if err := rows.Scan(&comment.ID, &comment.Author, &comment.Content, &comment.CreatedAt); err != nil {
				logger.ErrorfContext(r.Context(), "[FORUM] 讀取留言失敗 post_id=%d: %v", postID, err)
				internalError(w, "unable to read comments")
				return
			}
			// 與文章列表相同：Author 欄位暫存 email，轉成顯示名稱後再補上雜湊識別碼。
			// 每筆留言各查一次暱稱，屬 N+1，但單次請求上限 12 筆故最壞情況可控。
			authorEmail := comment.Author
			comment.Author = s.forumAuthor(r, authorEmail)
			comment.AuthorKey = publicForumKey(authorEmail)
			comments = append(comments, comment)
		}
		if err := rows.Err(); err != nil {
			internalError(w, "unable to read comments")
			return
		}
		// hasMore 與文章列表同法：湊滿一頁就推測還有下一頁。
		// 此處 limit 由呼叫端決定，因此邊界 false positive 的機率略高於固定 pageSize。
		writeOK(w, map[string]interface{}{"items": comments, "hasMore": len(comments) == limit})
		return
	}

	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	// 只在寫入分支檢查 Origin：GET 是純讀取，且中介層 requireLoginForWrite
	// 對 GET 允許未登入者通過，此時加 origin 檢查只會擋掉正常的公開瀏覽。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	var req createForumCommentRequest
	// 此處刻意不套 MaxBytesReader：留言有 2000 字的業務上限擋在後面，
	// 而限制 body 大小會讓「超長」以 400／413 的另一種形式出現，兩道限制語意重疊。
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// 先 TrimSpace 再驗證，順序有意義：若先驗長度再修剪，一個前後塞滿空白的
	// 剛好超限的留言會被拒；修剪後再驗，空白就不計入字數，
	// 而純空白內容也會被下面的空值檢查擋掉。
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		badRequest(w, "留言內容不可為空")
		return
	}
	// 用 rune 數而非 len(string)：len() 數的是位元組，一則 2000 字的中文留言
	// 會是 6000 個位元組而遭誤拒。rune 數才等於使用者看到的字數，
	// 也與 TEXT 欄位在 utf8mb4 下的實際容量一致。
	if len([]rune(req.Content)) > 2000 {
		badRequest(w, "留言最多 2000 字")
		return
	}

	// Schema 沒有外鍵，因此「文章是否存在」必須自行檢查，否則會留下孤兒留言。
	// 用 COUNT(*) 而非 SELECT 1：兩者成本相同，但 COUNT 語意在這裡更直觀。
	var postCount int
	if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_posts WHERE id = ?`, postID).Scan(&postCount); err != nil {
		internalError(w, "unable to load forum post")
		return
	}
	// 回 404 而非 500：使用者操作沒有錯，是目標不存在，語意要對。
	if postCount == 0 {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}

	// 時間只取一次，插入與回應共用同一值，確保前端拿到的 createdAt
	// 與資料庫中的值完全一致（否則兩次 time.Now() 可能跨越一秒而對不上）。
	createdAt := time.Now()
	// 單一 INSERT 本身即具原子性，不需要交易：包起來只會多出 BEGIN/COMMIT 兩次往返。
	result, err := s.db.ExecContext(r.Context(),
		`INSERT INTO forum_post_comments (post_id, author_email, content, created_at) VALUES (?, ?, ?, ?)`,
		// 再取一次 session 中繼資料作為作者；POST 已通過中介層的登入與停權檢查，
		// 因此不會是空字串。
		postID, s.sessions.ResolveUser(r), req.Content, createdAt)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 新增留言失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to create comment")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to read created comment")
		return
	}
	// 這裡再呼叫一次 ResolveUser 會多一次 Redis 往返；因為上一次的值不在作用域內，
	// 沿用現狀以免改動行為。兩次呼叫讀的是同一個 session cookie，結果必然一致。
	author := s.sessions.ResolveUser(r)
	// 直接回傳建立好的物件，前端可插入列表頭端而不必重新載入整頁。
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok": true,
		"item": forumComment{
			ID: id, Author: s.forumAuthor(r, author), AuthorKey: publicForumKey(author),
			Content: req.Content, CreatedAt: createdAt,
		},
	})
}

// handleForumPostAction 是 /api/forum/posts/ 子樹的單一分派點。
//
// 為什麼用字串後綴判斷而不是在 ServeMux 註冊多條路由：net/http 的
// "/api/forum/posts/" 樣式會匹配整個子樹，無法在其中再區分 comments / like / report。
// 若要註冊 "/api/forum/posts/{id}/like" 這類樣式就得升級到 Go 1.22 的 method-pattern，
// 而目前這個檔案同時服務未登入的讀取與多種寫入，維持單一分派點較易審查。
//
// 判斷順序不可調換，這是本函式最容易出錯的地方：
//
//	/posts/1                         → 刪除本人貼文
//	/posts/1/comments/2/report  → comment 檢舉
//	/posts/1/comments           → 留言
//	/posts/1/like               → 按讚
//	/posts/1/report             → post 檢舉
//
// 「留言檢舉」必須排在「留言列表」之前，因為兩者都以 /comments 開頭；
// 也必須排在「post 檢舉」之前，因為它同樣以 /report 結尾。
// 反過來，任何無法對應到上述子資源的子路徑（例如 /posts/1/abc）落到 404。
func (s *Server) handleForumPostAction(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/forum/posts/"), "/")
	if trimmedPath != "" && !strings.Contains(trimmedPath, "/") {
		if id, err := strconv.ParseInt(trimmedPath, 10, 64); err == nil && id > 0 {
			s.handleForumPostDelete(w, r)
			return
		}
	}
	// 同時檢查後綴是 /report 且路徑中含 /comments/，兩個條件缺一不可：
	// 只看後綴會把 /posts/1/report 誤判為留言檢舉。
	if strings.HasSuffix(r.URL.Path, "/report") && strings.Contains(strings.TrimSuffix(r.URL.Path, "/report"), "/comments/") {
		s.handleForumReport(w, r, "comment")
		return
	}
	if strings.HasSuffix(r.URL.Path, "/comments") {
		s.handleForumComments(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/like") {
		s.handleForumPostLike(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/report") {
		s.handleForumReport(w, r, "post")
		return
	}
	// 明示 404 而非讓子樹樣式一路比對到這裡才由預設 mux 處理，
	// 確保 /api/forum/posts/ 底下不認識的路徑不會誤落到其他 handler。
	http.NotFound(w, r)
}

// handleForumPostDelete 刪除登入者自己的貼文。
func (s *Server) handleForumPostDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	postID, err := forumPostIDFromPath(r.URL.Path)
	if err != nil {
		badRequest(w, "invalid post id")
		return
	}
	// 將作者條件放進 DELETE 本身，避免「先查作者、再刪除」之間的權限競態。
	result, err := s.db.ExecContext(r.Context(), `DELETE FROM forum_posts WHERE id = ? AND author_email = ?`, postID, s.sessions.ResolveUser(r))
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 刪除本人貼文失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to delete forum post")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		// 不區分貼文不存在或屬於他人，避免藉回應差異探測作者權限。
		http.NotFound(w, r)
		return
	}
	s.unindexForumPost(r.Context(), postID)
	writeOK(w, map[string]bool{"ok": true})
}

// handleForumReport 處理文章與留言的檢舉，targetType 為 "post" 或 "comment"。
//
// targetType 只由 handleForumPostAction 以字面值傳入，永遠不是使用者可控的輸入，
// 這是下方能安全地把 table 名稱拼進 SQL 的前提。
func (s *Server) handleForumReport(w http.ResponseWriter, r *http.Request, targetType string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	var req forumReportRequest
	// 8KB 遠大於 500 字的中文內容，卻能擋下夾帶大量資料的請求。
	// 檢舉原因在此加上下限，因為它是管理員審查時唯一線索。
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// 同樣先修剪再驗證：純空白的檢舉理由不應算作有效理由。
	req.Reason = strings.TrimSpace(req.Reason)
	// 500 的上限與 forum_reports.reason VARCHAR(500) 對齊。VARCHAR 在 utf8mb4 下
	// 以字元計數，因此必須用 rune 數檢查，len(string) 會誤拒合法內容。
	if req.Reason == "" || len([]rune(req.Reason)) > 500 {
		badRequest(w, "檢舉原因需為 1 至 500 字")
		return
	}
	// 目標 id 的取法因 post 與 comment 而異，因此抽成獨立純函式。
	targetID, err := forumReportTargetID(r.URL.Path, targetType)
	if err != nil {
		badRequest(w, "invalid report target")
		return
	}
	var count int
	// table 名稱以字串拼接而非字串格式化佔位符（MySQL 不支援），安全性仰賴
	// 上述 targetType 非使用者可控這項前提。
	table := "forum_posts"
	if targetType == "comment" {
		table = "forum_post_comments"
	}
	// 檢查目標存在，否則會產生指向不存在內容的檢舉紀錄，讓管理介面出現幽靈資料。
	if err := s.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM "+table+" WHERE id = ?", targetID).Scan(&count); err != nil {
		internalError(w, "unable to load report target")
		return
	}
	if count == 0 {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	// status 明確寫成 'PENDING' 而非依賴欄位預設值，讓審查佇列的查詢條件
	// 與實際寫入值在同一處可見。
	_, err = s.db.ExecContext(r.Context(), `
		INSERT INTO forum_reports (target_type, target_id, reporter_email, reason, status, created_at)
		VALUES (?, ?, ?, ?, 'PENDING', ?)`, targetType, targetID, s.sessions.ResolveUser(r), req.Reason, time.Now())
	if err != nil {
		// 1062 = ER_DUP_ENTRY。forum_reports 上的
		// UNIQUE KEY uq_forum_reports_reporter_target (reporter_email, target_type, target_id)
		// 是「同一人不能重複檢舉同一內容」的真正防線，而且它同時擋下
		// 使用者送出兩次與兩個分頁同時送出兩次的競態——不需要額外的應用層鎖。
		// 這裡用型別斷言而非 errors.As：go-sql-driver 直接回傳 *mysql.MySQLError，
		// 中間沒有包裝層，斷言足夠；handleForumProfile 用 errors.As 是較保守的寫法。
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			// 409 Conflict 而非 400：請求語意正確，是資源狀態衝突。
			writeError(w, http.StatusConflict, "你已檢舉過此內容")
			return
		}
		logger.ErrorfContext(r.Context(), "[FORUM] 建立檢舉失敗 type=%s target_id=%d: %v", targetType, targetID, err)
		internalError(w, "unable to create report")
		return
	}
	// 只回 ok，不回傳檢舉紀錄 id：公開端沒有任何用途需要它，
	// 少回一個欄位就少洩漏一點內部結構。
	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

// forumReportTargetID 從請求路徑取出被檢舉對象的 id。
// 純函式：不碰資料庫，也不依賴 Server，便于測試與推導。
func forumReportTargetID(path, targetType string) (int64, error) {
	if targetType == "comment" {
		// 留言 id 埋在 /posts/{postID}/comments/{commentID}/report 中段，
		// 無法用前綴去除法解析，因此先定位 /comments/ 再切前後綴。
		marker := "/comments/"
		start := strings.Index(path, marker)
		if start < 0 {
			return 0, fmt.Errorf("missing comment id")
		}
		value := strings.TrimSuffix(strings.TrimPrefix(path[start+len(marker):], "/"), "/report")
		id, err := strconv.ParseInt(value, 10, 64)
		// <= 0 一併拒絕：資料庫的 id 皆為正數 AUTO_INCREMENT，0 與負值只可能
		// 來自惡意或異常輸入，放行等於讓無意义的查詢打到 DB。
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid comment id")
		}
		return id, nil
	}
	// post 與 like 兩條路徑共用同一種「取第一段數字」的形式。
	return forumPostIDFromPath(path)
}

// forumPostIDFromPath 從 /api/forum/posts/{id}[/子資源...] 中取出文章 id。
//
// 前綴 /api/forum/posts/ 與 server.go 的路由註冊是硬耦合的：一旦路由前綴改動，
// 這裡就會對所有帶子資源的路徑解析失敗，因此兩處必須同步修改。
func forumPostIDFromPath(path string) (int64, error) {
	trimmed := strings.TrimPrefix(path, "/api/forum/posts/")
	// 容忍結尾斜線：/api/forum/posts/1/ 與 /api/forum/posts/1 應視為同一篇。
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return 0, fmt.Errorf("missing post id")
	}
	// 在第一個斜線切斷，剩下的就是 /like、/report、/comments 等子資源。
	// 用 Index 而非 LastIndex：子資源之後若還有多餘段（如 /comments/2/report）
	// 也只取文章 id，後續交給各自的解析函式處理。
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	postID, err := strconv.ParseInt(trimmed, 10, 64)
	// 轉型錯誤與非正值都視為無效，讓呼叫端回 400 而不是把奇怪的值送進 SQL。
	if err != nil || postID <= 0 {
		return 0, fmt.Errorf("invalid post id")
	}
	return postID, nil
}

// handleForumPostLike 切換按讚狀態：已讚則取消，未讚則新增。
//
// 刻意只有一個端點而沒有獨立的 DELETE：讚數是單純的計數開關，前端按鈕
// 永遠只需要「切到相反狀態」，這種語意用單一 POST 表達最不容易出錯。
func (s *Server) handleForumPostLike(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	// 縱深防禦：中介層 requireLoginForWrite 對 POST 已做過登入與停權檢查。
	// 這裡再確認一次，讓 handler 被單獨重用或日後改動路由時仍然安全。
	author := s.sessions.ResolveUser(r)
	if author == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}
	// 檢查順序刻意是「身分 → Origin → 路徑」：Origin 驗證成本最低且擋掉
	// 跨站偽造，放在碰資料庫之前可避免無謂的交易開銷。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	postID, err := forumPostIDFromPath(r.URL.Path)
	if err != nil {
		badRequest(w, "invalid post id")
		return
	}

	// 交易邊界涵蓋「讀取目前狀態 → 寫入 → 重算數量 → 送出」。
	// 沒有交易時，「先 SELECT 再 INSERT」這兩步之間有窗口，兩個並發請求
	// 可能都讀到未讚而都送出 INSERT。
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 開始按讚交易失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to begin like transaction")
		return
	}
	// 安全網：本函式所有提早 return 的分支都會走到這裡。
	// Commit 成功之後再次呼叫 Rollback 會回 sql.ErrTxDone，屬預期且被忽略，
	// 因此「無論成功失敗都寫 defer」是安全且必要的。
	defer tx.Rollback()

	// 交易內確認文章存在：schema 沒有外鍵，否則會出現指向不存在文章的讚。
	var postCount int
	if err := tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_posts WHERE id = ?`, postID).Scan(&postCount); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 檢查文章失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to load forum post")
		return
	}
	if postCount == 0 {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}

	// 讀取自己對這篇文章的讚；主鍵是 (post_id, author_email)，
	// 因此這筆查詢是唯一的（最多一列），結果只可能是 0 或 1。
	var likeCount int
	if err := tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ? AND author_email = ?`, postID, author).Scan(&likeCount); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 檢查按讚狀態失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to check like status")
		return
	}
	liked := likeCount > 0
	if liked {
		// DELETE 而非 UPDATE 設成某個停用旗標：讚沒有「歷史」語意，
		// 讓資料列完全消失可以讓唯一的索引同時是複合主鍵，保持結構簡單。
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM forum_post_likes WHERE post_id = ? AND author_email = ?`, postID, author); err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 取消按讚失敗 post_id=%d: %v", postID, err)
			internalError(w, "unable to remove like")
			return
		}
	} else {
		// 就算兩個請求都通過了上面的檢查而同時 INSERT，複合主鍵仍會讓
		// 其中一個收到 1062 而失敗——資料庫層是最終仲裁者。
		// 這一點與 handleForumReport 的 1062 處理不同：本函式沒有為 1062 特別分流，
		// 極端併發下該請求會以 500 結束。之所以可接受，是讚只是計數開關，
		// 使用者再按一次即可，不涉及金流或權限。
		if _, err := tx.ExecContext(r.Context(), `INSERT INTO forum_post_likes (post_id, author_email, created_at) VALUES (?, ?, ?)`, postID, author, time.Now()); err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 新增按讚失敗 post_id=%d: %v", postID, err)
			internalError(w, "unable to add like")
			return
		}
	}

	// 在交易內重算總數：這次讀取會看到自己尚未 commit 的寫入，
	// 因此回傳的數字已包含本次切換，前端可直接採用而不用再請求一次列表。
	// 刻意不加 FOR UPDATE 鎖 forum_posts：那會讓同一篇熱門文章的所有按讚
	// 序列化執行，收益只為了一個計數欄位。
	var totalLikeCount int
	if err := tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM forum_post_likes WHERE post_id = ?`, postID).Scan(&totalLikeCount); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 統計按讚數失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to count likes")
		return
	}

	if err := tx.Commit(); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 儲存按讚失敗 post_id=%d: %v", postID, err)
		internalError(w, "unable to save like")
		return
	}

	writeOK(w, map[string]interface{}{
		// 回傳切換後的新狀態而非原狀態，前端才能直接把 heart 圖示與數字換掉。
		"liked": !liked,
		"count": totalLikeCount,
	})
}

// handleForumPublicProfile 讓任何人依 publicKey 取得某位使用者的公開資料。
//
// 這是論壇中唯一不需要登入就能查詢特定使用者的端點，因此它只接受雜湊後的
// publicKey 而不接受 email——呼叫端手上永遠拿不到原始地址可查。
//
// 回應多帶一個 following（我是否追蹤了這個人）：未登入時為 false。
// 它只反映「請求者」的狀態，不是這位使用者的追蹤者數 —— 追蹤關係刻意完全
// 私有（見 forum_follow_handlers.go 的檔頭），因此這裡沒有任何聚合值。
func (s *Server) handleForumPublicProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	// 長度檢查只為擋掉明顯無效的輸入；真正的比對靠 SQL 的等值條件，
	// 而 public_key 有索引（MigrateMySQL 第 20 步），不會因此變成全表掃描。
	if len(key) != 64 {
		badRequest(w, "invalid profile key")
		return
	}

	// email 只在這裡用於查追蹤狀態與現算 publicKey，不會出現在回應中。
	me := s.sessions.ResolveUser(r)

	var email, nickname, bio string
	// 依 public_key 而非 author_email 查詢。public_key 的值由 MigrateMySQL
	// 以 SHA2(author_email, 256) 回填，與 publicForumKey 的算法一致，兩邊必須對齊。
	err := s.db.QueryRowContext(r.Context(),
		`SELECT author_email, nickname, bio FROM forum_profiles WHERE public_key = ?`, key).Scan(&email, &nickname, &bio)
	if err == sql.ErrNoRows {
		// 找不到時回 200 加預設值而非 404：前端（/forum/others-profile）
		// 因此不必處理錯誤分支，也不會因為狀態碼差異而洩漏
		// 「這個 key 對應的 email 存在過但從未建檔」這類資訊。
		//
		// 刻意不放 following 鍵：沒有對應的 profile 就不存在「可追蹤的對象」
		// （對方可能從未建檔），前端以「無此鍵」判斷不渲染追蹤鈕。
		writeOK(w, map[string]interface{}{
			"nickname": "匿名使用者",
			"bio":      "這位使用者尚未設定公開資料。",
		})
		return
	}
	// 必須把 ErrNoRows 與其他錯誤分開：前者是正常情況，後者要回 500。
	if err != nil {
		internalError(w, "unable to load public profile")
		return
	}

	payload := map[string]interface{}{"nickname": nickname, "bio": bio}
	// 自我追蹤被後端拒絕，因此自己的頁面顯示「追蹤中」是錯的，必須讓前端
	// 知道要把按鈕藏起來。
	if me != "" && me != email {
		var count int
		if err := s.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM forum_follows WHERE follower_email = ? AND target_email = ?`,
			me, email).Scan(&count); err != nil {
			// 追蹤狀態查不到不應該讓整份公開資料開天窗：寧可回 false
			// （按鈕顯示成「追蹤」，使用者按下去就是切換，不會出錯），
			// 也不要把一份讀得到的暱稱與簡介一起拖掉。
			logger.WarnfContext(r.Context(), "[FORUM] 讀取追蹤狀態失敗: %v", err)
		}
		payload["following"] = count > 0
	}
	writeOK(w, payload)
}

// handleForumProfile 讀取（GET）與更新（PUT）自己的暱稱與簡介。
//
// 為什麼不用 publicKey 查詢自己的資料：server.go 的 requireLogin 已保證
// 身分，這裡直接用 session 中的 email 當主鍵查詢，語意最單純，
// 也不必處理「手上的 publicKey 過期」的情況。
func (s *Server) handleForumProfile(w http.ResponseWriter, r *http.Request) {
	email := s.sessions.ResolveUser(r)
	// 路由已用 requireLogin 包住（含停權檢查），此處重複確認是縱深防禦。
	if email == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}

	switch r.Method {
	case http.MethodGet:
		var nickname, bio string
		err := s.db.QueryRowContext(r.Context(),
			`SELECT nickname, bio FROM forum_profiles WHERE author_email = ?`, email).Scan(&nickname, &bio)
		if err == sql.ErrNoRows {
			// 尚未建立資料檔時回空字串（而非預設暱稱），
			// 讓前端可以顯示「尚未設定」並把欄位留空供使用者填寫。
			nickname = ""
		} else if err != nil {
			internalError(w, "unable to load profile")
			return
		}
		writeOK(w, map[string]string{"nickname": nickname, "bio": bio})
	case http.MethodPut:
		if !s.isTrustedOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		var req forumProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			badRequest(w, "invalid request")
			return
		}
		// 兩欄都先修剪。暱稱若保留前後空白，可能與另一個使用者「看起來相同、
		// 實際卻繞過唯一索引」；簡介則會在前端顯示時產生多餘空行。
		req.Nickname = strings.TrimSpace(req.Nickname)
		req.Bio = strings.TrimSpace(req.Bio)
		if req.Nickname == "" {
			badRequest(w, "暱稱不可為空")
			return
		}
		// 30 的上限與 forum_profiles.nickname VARCHAR(30) 對齊；
		// VARCHAR 以字元計，故以 rune 檢查而非 len(string)。
		if len([]rune(req.Nickname)) > 30 {
			badRequest(w, "暱稱最多 30 字")
			return
		}
		// 簡介可以為空（新增個人簡介時先留白是常見動線），但不得超過 500 字。
		if len([]rune(req.Bio)) > 500 {
			badRequest(w, "個人簡介最多 500 字")
			return
		}
		// 以單一 upsert 語句同時處理「首次建立」與「更新」，省掉一次
		// 先 SELECT 判斷再分支寫入的往返。author_email 是主鍵，故
		// ON DUPLICATE KEY 只會在「自己的資料已存在」時觸發。
		// 順帶每次都寫入 publicForumKey，讓早期在該欄位存在前建立的資料
		// 自動補齊公開識別碼。
		_, err := s.db.ExecContext(r.Context(), `
			INSERT INTO forum_profiles (author_email, public_key, nickname, bio, updated_at) VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), bio = VALUES(bio), updated_at = VALUES(updated_at)`,
			email, publicForumKey(email), req.Nickname, req.Bio, time.Now())
		if err != nil {
			// 用 errors.As 而非型別斷言：driver 回傳的錯誤未來若被包裝
			// （例如加上連線重試或額外註解），errors.As 仍能穿透包裝找出
			// *mysql.MySQLError，型別斷言則會失效。1062 = ER_DUP_ENTRY，
			// 在這句 upsert 上唯一可能撞到的是 uq_forum_profiles_nickname
			// （author_email 正是本列自己的鍵，不會衝突）。
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				// 409：暱稱是公開身分的識別方式，撞名會讓他人資料頁產生歧義，
				// 因此在資料庫層禁止重複而非僅靠前端提示。
				writeError(w, http.StatusConflict, "暱稱已被使用")
				return
			}
			internalError(w, "unable to update profile")
			return
		}
		// 直接回傳修剪後的實際值，前端無需再清理一次空白。
		writeOK(w, map[string]string{"nickname": req.Nickname, "bio": req.Bio})
	default:
		// POST／DELETE 一律不支援：語意已由 GET 與 PUT 完整覆蓋，
		// 開一個 POST 別名只會讓呼叫端有兩條路徑通往同一個狀態變更。
		methodNotAllowed(w)
	}
}

// forumAuthorTags 查出某位使用者被指派的標籤名稱，依名稱遞增排序。
//
// 標籤由管理員在 /api/admin/tags 指派，使用者無法自行修改，因此這裡是唯讀查詢。
// 標籤表無關聯到論壇內容，但文章列表會回傳它，讓前端可以在作者名旁顯示身份標示。
func (s *Server) forumAuthorTags(ctx context.Context, email string) ([]string, error) {
	// JOIN forum_user_tags 取名稱；關聯表的主鍵是 (user_email, tag_id) 且另有
	// idx_..._tag_id，兩個方向的存取都不需要額外索引。
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.name
		FROM forum_user_tag_assignments a
		JOIN forum_user_tags t ON t.id = a.tag_id
		WHERE a.user_email = ?
		ORDER BY t.name ASC`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// 回傳非 nil 的空切片：即使欄位帶 omitempty，nil 與空切片在 JSON 上仍不同，
	// 用 make 初始化可讓「沒有標籤」在所有呼叫端表現一致。
	tags := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	// 迭代中途出錯也必須回報，否則呼叫端會把「讀了一半」當成「就這些」。
	return tags, rows.Err()
}

// forumAuthor 決定對外顯示的作者名稱：優先用暱稱，否則給一個匿名代號。
//
// 接受 *http.Request 而非 context.Context，除了能取到 r.Context() 綁住
// 查詢生命週期，也保留了日後「登入者看自己的文章時顯示較多資訊」這類
// 個人化分支的空間；目前刻意不區分，理由見下。
func (s *Server) forumAuthor(r *http.Request, email string) string {
	var nickname string
	err := s.db.QueryRowContext(r.Context(),
		`SELECT nickname FROM forum_profiles WHERE author_email = ?`, email).Scan(&nickname)
	// err == nil 還不夠：暱稱欄位 NOT NULL 但可能是空字串（早期資料或只填了 bio），
	// 空白名稱在畫面上等同沒設定，所以再檢查一次 TrimSpace 後是否為空。
	if err == nil && strings.TrimSpace(nickname) != "" {
		return nickname
	}
	// ErrNoRows 與其他 DB 錯誤都走到同一個分支：任何查詢失敗都退化成顯示匿名代號。
	// 這樣的取捨是刻意的——寧可名字退化成匿名，也不能因為一次 DB 錯誤而
	// 整頁列表失敗（500）或把 email 洩漏出去。
	return publicForumAuthor(email)
}

// publicForumKey 以 email 的 SHA-256 雜湊作為公開識別碼。
//
// 為什麼要雜湊：公開 API 必須能把「同一個人」穩定地辨識出來（給前端當 key、
// 供 /api/forum/public-profile 查詢），但不能直接回傳 email——
// email 等同個人資料，散落在每則貼文的 JSON 裡等同公開名錄。
// SHA-256 不可逆，因此雜湊值本身不會洩漏原字串。
//
// 誠實的界限：雜湊是確定性的，而 email 的取值空間很小，攻擊者仍可枚舉
// 常見信箱與雜湊比對還原。它的作用是「不直接外洩原始位址」，
// 不是讓這個身分無法被推知；真正的保護不應依賴它。
func publicForumKey(email string) string {
	hash := sha256.Sum256([]byte(email))
	// %x 產生 64 個小寫十六進位字元，長度與 handleForumPublicProfile 的檢查一致。
	return fmt.Sprintf("%x", hash[:])
}

// createForumPost 建立一篇文章，並直接回傳組好的物件供前端插入列表。
func (s *Server) createForumPost(w http.ResponseWriter, r *http.Request) {
	// 中介層 requireLoginForWrite 已做登入與停權檢查；此處確認 email 非空
	// 以確保 author_email 欄位 NOT NULL 不會被寫入空值。
	author := s.sessions.ResolveUser(r)
	if author == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}
	// 發文是匿名可做的事，必須驗 Origin。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	var req createForumPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid request")
		return
	}
	// 先 TrimSpace 再檢查長度，理由與留言相同：純空白貼文不該被接受，
	// 而修剪後才量字數才符合使用者看到的內容。
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		badRequest(w, "內容不可為空")
		return
	}
	// 10000 字上限對應 content TEXT 欄位；用 rune 數而非位元組數，
	// 否則全中文貼文會在約三分之一長度就被誤判超限。
	if len([]rune(req.Content)) > 10000 {
		badRequest(w, "內容最多 10000 字")
		return
	}
	// imageUrl 只接受「/api/forum/images 回傳的網址」：由 forumImageFileName
	// 收斂成純檔名後才寫入資料庫，因此庫中不會出現內部位址，也不會出現
	// 使用者自帶的第三方網址。第三方網址在舊版是允許的，但它與 CSP 的
	// img-src 白名單互相矛盾（本來就載不出來），現在於寫入時直接回 400，
	// 讓「貼不上去」變成明確可見的回應，而不是一張永遠破掉的圖。
	imageName := ""
	var mediaToken string
	if strings.TrimSpace(req.ImageURL) != "" {
		imageName = s.forumImageFileName(req.ImageURL)
		if imageName == "" {
			badRequest(w, "附圖必須是上傳到本站的圖片")
			return
		}
		// 重新簽發 token，而沿用請求帶來的那一個不是選項：那是使用者可控的
		// 查詢參數，把它原樣送回等於反射未驗證的輸入。發文成功後前端會導向
		// /forum 並在 pagehide 釋放上傳時那把 token，新簽的這把只出現在回應裡、
		// 沒有被前端使用，會由 MediaTokenTTLSecs 到期作廢。
		token, err := s.createMediaToken(r.Context())
		if err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 建立圖片 token 失敗: %v", err)
			writeError(w, http.StatusBadGateway, "unable to create media token")
			return
		}
		mediaToken = token
	}

	// 單一 INSERT 具原子性，不需要交易。寫入的是檔名，不是網址。
	createdAt := time.Now()
	result, err := s.db.ExecContext(r.Context(),
		`INSERT INTO forum_posts (author_email, content, image_url, created_at) VALUES (?, ?, ?, ?)`,
		author, req.Content, imageName, createdAt)
	if err != nil {
		internalError(w, "unable to create forum post")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		internalError(w, "unable to read created post")
		return
	}
	// 寫入 ES 索引（best-effort，失敗只記日誌；理由見 indexForumPost）。
	// 刻意放在 MySQL 寫入成功之後：順序反過來會讓索引裡出現一篇
	// 資料庫裡不存在的貼文，那種幽靈結果比「搜尋慢一步才出現」更難察覺。
	s.indexForumPost(r.Context(), id, req.Content, author, createdAt)
	// 回讀自己的標籤：作者欄位要與列表中的其他貼文保持一致的呈現方式。
	// 這一次額外查詢取代了「發文後重新載入整頁列表」的往返。
	authorTags, err := s.forumAuthorTags(r.Context(), author)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取新文章作者標籤失敗: %v", err)
		internalError(w, "unable to read forum user tags")
		return
	}
	// 回傳的物件其 LikeCount 與 CommentCount 為零值（新貼文必然為零），
	// ImageURL 則是「檔名 + 剛才那支 token」組出的網址，與列表中的其他貼文
	// 格式一致，前端不必再自己拼一次。
	// CreatedAt 沿用實際寫入資料庫的那個時間值（createdAt），而不是另取一次
	// time.Now()：兩者可能相差不到一秒，而列表與首頁插入的項目必須顯示同一個時間。
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok":   true,
		"item": forumPost{ID: id, Author: s.forumAuthor(r, author), AuthorKey: publicForumKey(author), AuthorTags: authorTags, Content: req.Content, ImageURL: s.forumImageURL(imageName, mediaToken), CreatedAt: createdAt},
	})
}

// publicForumAuthor 產生形如「匿名123456」的隨機代號。
//
// 注意它每次呼叫都會得到不同結果，因此不是穩定的化名：同一個未設暱稱的
// 使用者在兩次請求中會顯示不同的代號。跨請求的身分一致性由 publicForumKey
// 負責，前端應以 AuthorKey（而非 Author）作為識別與比對的依據。
func publicForumAuthor(_ string) string {
	// 參數刻意以底線命名：這函式不需要也不該取得 email——一旦有人誤以為
	// 可以在這裡塞入 email 的衍生資訊（例如雜湊前綴），就會開啟一個
	// 可被還原的識別路徑。需要的雜湊請直接呼叫 publicForumKey。
	number, err := cryptorand.Int(cryptorand.Reader, big.NewInt(1000000))
	if err != nil {
		// 亂數來源失效是極端情況，此時退回固定字串：寧可所有人同名，
		// 也不能讓整個列表因一列而失敗。
		return "匿名000000"
	}
	// %06d 補零到六位，讓前端版面不會因為位數不同而抖動。
	return fmt.Sprintf("匿名%06d", number.Int64())
}

// handleForumPublicPosts 回傳某一位使用者的貼文分頁，供 /forum/others-profile
// 的「貼文」區塊使用。
//
// 與 public-profile 成對：兩者都以 publicKey 當參數、都不需登入、都是「某個人的
// 公開頁面」而不是「我自己的資料」。刻意不做成 /api/forum/posts 的可選參數 ——
// 那支端點是首頁動態，它的分頁語意（回應的 items 形狀、hasMore 推測）被
// useFeed 與搜尋結果共用，混入一個「依某人過濾」的分支會讓那個共用流程多一種
// 形狀，而這裡的形狀其實完全相同（見 loadForumPosts）。
//
// key → email 的解析沿用 forumEmailByPublicKey（定義在 forum_follow_handlers.go）：
// 兩處都需要「先查 forum_profiles、查不到再退回掃曾經發文的作者」這兩段式。
func (s *Server) handleForumPublicPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("user"))
	// 長度檢查與 handleForumPublicProfile 相同：擋掉明顯無效的輸入，真正的
	// 比對靠索引。
	if len(key) != 64 {
		badRequest(w, "invalid profile key")
		return
	}

	author, err := s.forumEmailByPublicKey(r.Context(), key)
	if errors.Is(err, sql.ErrNoRows) {
		// 與 public-profile 同一個理由回 404 而不是 500：金鑰不存在是使用者
		// 看得到的情況（自己打錯連結），而前端對這一頁的處理是「顯示貼文區塊
		// 為空」，因此狀態碼的差別只在診斷上。
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		internalError(w, "unable to resolve forum user")
		return
	}

	// 單頁筆數與兩個既有列表一致（見 listForumPosts 的說明）。
	const pageSize = 25

	offset, ok := forumOffsetParam(w, r)
	if !ok {
		return
	}

	// author_email 有索引（遷移第 21 步），因此這是「索引等值過濾 + 依
	// idx_forum_posts_created_at 排序」。刻意不過濾停權：listForumPosts 不過濾，
	// 這裡額外過濾會讓同一篇貼文在首頁可見、在個人頁消失。
	posts, err := s.loadForumPosts(r, `WHERE fp.author_email = ?`, []interface{}{author}, pageSize, offset)
	if err != nil {
		s.writeForumPostLoadError(w, err, "unable to load public posts")
		return
	}

	writeOK(w, map[string]interface{}{
		"items":   posts,
		"hasMore": len(posts) == pageSize,
	})
}

// handleForumLoginPage 產生登入頁的 handler，回傳前端單一 HTML 殼。
//
// frontendDir 在啟動時解析一次（見 server.go 的 frontendRoot），
// 因此此處不必在每個請求中重新搜尋檔案系統。
func (s *Server) handleForumLoginPage(frontendDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// server.go 對 /forum/login 與 /forum/login/ 各註冊一次；此處再確認一次
		// 是為了防止日後有人改成子樹樣式後，這個 handler 變成整個 /forum/login/*
		// 的守門人——那會讓 /forum/login/abc 也吐出登入頁。
		if r.URL.Path != "/forum/login" && r.URL.Path != "/forum/login/" {
			http.NotFound(w, r)
			return
		}
		// no-store：內容與 session 狀態相關（登入後同一個網址的行為不同），
		// 進了瀏覽器或中介快取會造成已登出者仍看到登入後畫面。
		w.Header().Set("Cache-Control", "no-store")
		// 檔名來自固定字串，不含使用者輸入，因此沒有路徑穿越風險。
		// 走 serveHTMLFile 而非 http.ServeFile：站名佔位符要在送出前換成
		// 設定值（見 site.go），而 Cache-Control 上面已設定過，這裡不會被蓋掉。
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, "forum-login.html"))
	}
}

// handleForumPage 產生論壇前端各頁的 handler。
//
// 這五個頁面其實都是同一支 SPA，只是入口檔不同，送出的 HTML
// 會再由前端 router 依當前路徑決定渲染哪個畫面。
func (s *Server) handleForumPage(frontendDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 以 switch 逐一列出合法路徑，而不是「預設給 forum.html」：
		// /forum/ 是子樹樣式，未列出的子路徑必須明確 404，否則任何
		// /forum/xxx 都會拿到論壇首頁，路由的意圖會變得不可辨識。
		page := "forum.html"
		switch r.URL.Path {
		case "/forum", "/forum/":
		case "/forum/new", "/forum/new/":
			page = "forum-new.html"
		case "/forum/profile", "/forum/profile/":
			page = "forum-profile.html"
		case "/forum/others-profile", "/forum/others-profile/":
			page = "forum-others-profile.html"
		case "/forum/following", "/forum/following/":
			// 追蹤頁由 /forum/ 子樹路由接住（該路由掛 requireLogin），
			// 因此未登入者會先被 303 導去登入頁而不是拿到這份 HTML。
			// 刻意不在 server.go 另註冊一條：ServeMux 取最具體的樣式，
			// 另註一條只會多一份要同步的路由規則。
			page = "forum-following.html"
		default:
			http.NotFound(w, r)
			return
		}
		// 與登入頁相同理由：不可被快取，避免登入／登出後看到過期畫面。
		// 也讓 Service Worker 更新後拿到的是新檔案。
		w.Header().Set("Cache-Control", "no-store")
		s.serveHTMLFile(w, r, filepath.Join(frontendDir, page))
	}
}
