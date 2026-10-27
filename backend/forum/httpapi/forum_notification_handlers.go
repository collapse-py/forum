/*
站內通知（backend/forum/httpapi/forum_notification_handlers.go）

一、HTTP 路由對照（註冊於 server.go 的 Handler()）

	GET  /api/forum/notifications              -> handleForumNotifications    我的通知分頁（需登入）
	POST /api/forum/notifications              -> handleForumNotifications    標記為已讀（需登入）
	GET  /api/forum/notifications/unread-count -> handleForumNotificationUnreadCount 未讀數（需登入）
	GET  /api/forum/notifications/ws           -> handleForumNotificationStream 即時推播（需登入，見 forum_notification_stream.go）

二、通知是怎麼產生的

	本檔案只負責「讀取」與「標記已讀」。通知的**寫入**由觸發它的動作就地完成，
	因為只有那些 handler 才知道「這件事的通知該寄給誰」：

	  - 新增留言（forum_handlers.go 的 handleForumComments POST 分支）：
	    頂層留言通知貼文作者（kind=comment），回覆則通知父留言作者（kind=reply）。
	  - 切換追蹤（forum_follow_handlers.go 的 toggleForumFollow）：
	    新增追蹤時通知被追蹤者（kind=follow）。

	三處都呼叫同一個 createForumNotification，且都是 best-effort：通知寫不進去
	只記一筆日誌，絕不讓底層的留言／追蹤跟著失敗。理由見該函式的說明。

三、身分一律是 email，對外一律是 publicKey

	與追蹤功能同一個設計（見 forum_follow_handlers.go 的檔頭）：資料表以 email
	當關聯鍵（可以走索引），而任何對外的 JSON 都只出現 forumAuthorKey(email)。
	actor_email 從不出現在回應裡，回應帶的是 actor（顯示名稱）與 actorKey。

	actorKey 也走 forumAuthorKey 而不是舊的 SHA-256：通知是私密的，但它帶的是
	「別人的」識別碼，而且前端用它連到那個人的公開個人頁。兩件事都要求它與
	貼文卡上的 key 同一種形態 —— 否則設了秘密的部署上，個人頁連結會查不到人，
	而舊形態仍是可離線枚舉還原的 SHA-256（見 DeriveAuthorKey）。

四、三個端點都掛 requireLogin

	通知是純私人資料，「這則通知是我的」這個判斷在每一條查詢的 WHERE 裡
	（recipient_email = ?）。路由層的 requireLogin 只是縱深防禦，不是唯一的關卡。
*/
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/forum/idgen"
	"forum/forum/logger"
)

/*
 * 通知種類常數。
 *
 * 與資料庫的 kind 欄位是同一個字串空間：DB 端刻意用 VARCHAR 而非 ENUM
 * （見 MigrateMySQL 第 32 步），因此這裡的常數就是那份「合法值」的唯一權威。
 * 前端據此決定要顯示哪一句文案，因此值本身是契約的一部分，不可隨意更動。
 */
const (
	// forumNotificationComment：有人在「我的貼文」下留了頂層留言。
	forumNotificationComment = "comment"
	// forumNotificationReply：有人回覆了「我的留言」。
	forumNotificationReply = "reply"
	// forumNotificationFollow：有人追蹤了我。
	forumNotificationFollow = "follow"
)

// forumNotificationItem 是 GET /api/forum/notifications 回傳的單一通知。
//
//	ID          通知編號。標記已讀時送回後端的就是這個值。
//	Kind        comment / reply / follow，見上面的常數。
//	Actor       觸發者的顯示名稱（暱稱，沒有則為匿名代號）。與貼文卡上的
//	            作者名稱走同一個函式，因此同一個人在兩處顯示一致。
//	ActorKey    觸發者的 forumAuthorKey，供前端連結到他的公開個人頁。
//	ActorAvatar 觸發者頭像網址（已附加 token）；沒有頭像時省略。
//	PostID      相關貼文編號（kind=follow 時為 0，omitempty 因此省略）。
//	CommentID   相關留言編號：comment／reply 都是「這一則新留言」，follow 為 0
//	            （omitempty 因此省略）。前端據此把連結指向那則留言所在的討論串。
//	Read        是否已讀。由 read_at 是否為 NULL 推得，是回應唯一需要曝露的
//	            讀取狀態 —— read_at 的實際時間點前端用不到。
//	CreatedAt   通知產生的時間。
type forumNotificationItem struct {
	ID          int64     `json:"id"`
	Kind        string    `json:"kind"`
	Actor       string    `json:"actor"`
	ActorKey    string    `json:"actorKey"`
	ActorAvatar string    `json:"actorAvatar,omitempty"`
	PostID      int64     `json:"postId,omitempty"`
	CommentID   int64     `json:"commentId,omitempty"`
	Read        bool      `json:"read"`
	CreatedAt   time.Time `json:"createdAt"`
}

// forumNotificationRow 是「一列 forum_notifications」（含 LEFT JOIN forum_profiles
// 取來的暱稱與頭像），也就是組成一則對外通知所需的全部欄位。
//
// 存在的理由：清單端點與即時推播都要把同一列組成 forumNotificationItem，而那組
// 規則（顯示名稱的退路、頭像 token 的政策、哪些欄位在哪些 kind 下出現）只能有
// 一份。兩份會漂移，而漂移的症狀不是錯誤 —— 是「剛推播來的那一則」和「重新載入
// 之後的那一則」長得不太一樣。
type forumNotificationRow struct {
	ID        int64
	Kind      string
	ActorMail string
	// Nickname 與 Avatar 來自 forum_profiles，未建檔時為空字串
	// （LEFT JOIN，見 listForumNotifications）。
	Nickname  string
	Avatar    string
	PostID    int64
	CommentID int64
	// Read 由 read_at 是否為 NULL 推得；推播路徑讀到的必然是新寫入的列，
	// 因此那裡是 false。
	Read      bool
	CreatedAt time.Time
}

// forumNotificationItemFromRow 把一列通知組成對外的項目。
//
// mediaToken 是呼叫端簽好的頭像存取 token（沒有頭像、或簽不出來時為空字串）：
// 沒有 token 的圖片網址在瀏覽器裡是一張破圖，因此沒有 token 時整欄留空
// （omitempty 讓它從 JSON 裡消失），而不是送出一個註定壞掉的網址。
func (s *Server) forumNotificationItemFromRow(row forumNotificationRow, mediaToken string) forumNotificationItem {
	avatar := row.Avatar
	if avatar == "" || mediaToken == "" {
		avatar = ""
	}
	// 顯示名稱與貼文卡、追蹤清單三處一致：暱稱優先，空暱稱退化成匿名代號。
	// 這裡刻意不用 forumAuthor()（那會再查一次 DB，而暱稱已經在這一列上）。
	actor := row.Nickname
	if strings.TrimSpace(actor) == "" {
		actor = publicForumAuthor(row.ActorMail)
	}
	return forumNotificationItem{
		ID:          row.ID,
		Kind:        row.Kind,
		Actor:       actor,
		ActorKey:    s.forumAuthorKey(row.ActorMail),
		ActorAvatar: s.forumImageURL(avatar, mediaToken),
		PostID:      row.PostID,
		CommentID:   row.CommentID,
		Read:        row.Read,
		CreatedAt:   row.CreatedAt,
	}
}

// forumNotificationMarkRequest 是 POST /api/forum/notifications 的請求主體。
//
//	ID  > 0：只把這一則標記為已讀（且必須是自己的通知）。
//	ID <= 0：把「我的全部未讀」標記為已讀。
//
// 用同一個欄位的值域表達兩種操作，而不是另開 /mark-all 端點：兩者的差異只有
// WHERE 子句，而前端「點一則」與「全部已讀」是同一支函式的兩個呼叫。
type forumNotificationMarkRequest struct {
	ID int64 `json:"id"`
}

// createForumNotification 寫入一則通知，成功時回傳新通知的 id（跳過時為 0）。
//
// 回傳 id 是為了推播：呼叫端拿到 id 之後可以呼叫 notifyForumNotificationCreated
// 把這一則即時推給收件者開著的頁面。id 在寫入前由應用層的雪花演算法產生
// （資料庫沒有 AUTO_INCREMENT），因此它就是「這一則通知」的識別碼 ——
// 推播訊息裡帶的也是它。
//
// 刻意 best-effort：任何失敗都只記日誌並回傳 error 給呼叫端「參考」，而呼叫端
// 一律忽略它。理由與 forumAuthor 對 DB 錯誤退化成匿名代號相同 —— 通知是一個
// 「附帶」的效果，它壞掉不該讓使用者真正想做的事（留言、追蹤）跟著失敗。
// 一則漏掉的通知只會讓收件者少看到一個紅點，而一次 500 會讓留言直接消失。
//
// 三種情況直接跳過而不寫入：
//   - recipient 或 actor 為空：沒有收件者或沒有觸發者的通知沒有意義，
//     而空字串寫進 NOT NULL 欄位只會在 DB 端才炸開。
//   - recipient == actor：自己回覆自己的貼文、或（不可能但防禦性地）追蹤自己
//     時，通知自己是一件只有在 bug 裡才會發生的事。
//
// postID / commentID 為 0 代表「這一種通知沒有這個目標」，與資料表的
// NOT NULL DEFAULT 0 契約一致。
func (s *Server) createForumNotification(ctx context.Context, recipient, actor, kind string, postID, commentID int64) (int64, error) {
	recipient = strings.TrimSpace(recipient)
	actor = strings.TrimSpace(actor)
	if recipient == "" || actor == "" || recipient == actor {
		return 0, nil
	}
	id := idgen.Next()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO forum_notifications (id, recipient_email, actor_email, kind, post_id, comment_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, recipient, actor, kind, postID, commentID, time.Now()); err != nil {
		return 0, err
	}
	return id, nil
}

// notifyForumCommentCreated 依「這是頂層留言還是回覆」決定通知的收件者。
//
// 抽成獨立函式的原因是它同時需要「父留言作者」（回覆時）與「貼文作者」
// （頂層留言時），而這兩者在 handleForumComments 的作用域裡是分開取得的：
//
//	回覆（parentAuthor 非空）  -> 通知父留言作者，kind=reply。
//	                             父留言作者由 handleForumComments 從
//	                             forumParentCommentInfo 取得，已在手邊。
//	頂層留言（parentAuthor 空）-> 通知貼文作者，kind=comment。貼文作者需要
//	                             額外一次查詢（forumPostAuthorEmail）。
//
// parentAuthor 空字串代表頂層留言，因此呼叫端不需再傳 bool。
func (s *Server) notifyForumCommentCreated(r *http.Request, postID, commentID int64, actor, parentAuthor string) {
	kind := forumNotificationComment
	recipient := parentAuthor
	if strings.TrimSpace(recipient) != "" {
		kind = forumNotificationReply
	} else {
		var err error
		recipient, err = s.forumPostAuthorEmail(r.Context(), postID)
		if err != nil {
			// 連貼文作者都查不到時，通知無從建立。這不是罕見路徑：貼文在
			// 「檢查可見」與「插入留言」之間被刪掉就會走到這裡，而留言本身
			// 已經成功地寫入了。
			logger.WarnfContext(r.Context(), "[FORUM] 找不到留言通知收件者 post_id=%d comment_id=%d: %v", postID, commentID, err)
			return
		}
	}
	// 收件者等於觸發者（自己的貼文／自己的留言）時，createForumNotification
	// 會自行跳過，這裡不必再判斷一次。
	id, err := s.createForumNotification(r.Context(), recipient, actor, kind, postID, commentID)
	if err != nil {
		logger.WarnfContext(r.Context(), "[FORUM] 建立留言通知失敗 post_id=%d comment_id=%d kind=%s: %v", postID, commentID, kind, err)
		return
	}
	// 推播給收件者開著的頁面（非同步，理由見 notifyForumNotificationCreated）。
	// 放在寫入成功之後：沒有寫進資料庫的通知不值得推。
	s.notifyForumNotificationCreated(recipient, id)
}

// handleForumNotifications 是 /api/forum/notifications 的單一進入點。
//
// 路由已用 requireLogin 包住，因此兩種方法都必須登入；handler 內仍重複確認
// email 非空，理由與 handleForumFollows 相同（縱深防禦）。
func (s *Server) handleForumNotifications(w http.ResponseWriter, r *http.Request) {
	email := s.sessions.ResolveUser(r)
	if email == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listForumNotifications(w, r, email)
	case http.MethodPost:
		s.markForumNotificationsRead(w, r, email)
	default:
		methodNotAllowed(w)
	}
}

// listForumNotifications 回傳統「我的通知」分頁。
//
//	GET /api/forum/notifications?limit=&offset=
//
// 回應帶 unreadCount 讓前端在同一次請求裡更新鈴鐺上的紅點，不必再問一次
// unread-count —— 「重新載入列表」與「修正未讀數」永遠同時發生。
func (s *Server) listForumNotifications(w http.ResponseWriter, r *http.Request, recipient string) {
	// 單頁 20 則：更大沒有效益，因為列表本身就提供「載入更多」。
	// 上限壓在 50 與追蹤清單一致，避免 limit=100000 一次掃過整個收件匣。
	const initialPageSize = 20
	const maxPageSize = 50

	limit := initialPageSize
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > maxPageSize {
			badRequest(w, "invalid limit")
			return
		}
		limit = parsed
	}
	offset, ok := forumOffsetParam(w, r)
	if !ok {
		return
	}

	// 排序用 id DESC 而非 created_at DESC：created_at 是秒級精度，同一秒寫入
	// 的多則通知若用時間排序會沒有確定順序，分頁時可能漏讀或重複。id 是
	// 雪花 id，本身就是「產生的先後」，且 idx_forum_notifications_recipient
	// 的二級索引尾端就接著主鍵，這個排序不必額外排序。
	//
	// LEFT JOIN forum_profiles 取暱稱與頭像：觸發者可能從未建檔（早期使用者），
	// 用 INNER JOIN 會讓那則通知整個消失。與 listForumFollows 同一個取捨。
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT n.id, n.kind, n.actor_email, COALESCE(p.nickname, ''), COALESCE(p.avatar_url, ''),
		       n.post_id, n.comment_id, n.read_at, n.created_at
		FROM forum_notifications n
		LEFT JOIN forum_profiles p ON p.author_email = n.actor_email
		WHERE n.recipient_email = ?
		ORDER BY n.id DESC
		LIMIT ? OFFSET ?`, recipient, limit, offset)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 載入通知清單失敗: %v", err)
		internalError(w, "unable to load notifications")
		return
	}
	defer rows.Close()

	// 頭像 token 與 listForumFollows 同一個模式：整頁共用一把，簽發失敗則
	// 整頁退化成沒有頭像（而不是整支端點回 500）。avatarTokenFailed 讓放棄
	// 成為一次性決定，避免每列都重試一次 Redis 逾時。
	mediaToken := ""
	avatarTokenFailed := false
	items := make([]forumNotificationItem, 0)
	for rows.Next() {
		var (
			row    forumNotificationRow
			readAt sql.NullTime
		)
		if err := rows.Scan(&row.ID, &row.Kind, &row.ActorMail, &row.Nickname, &row.Avatar, &row.PostID, &row.CommentID, &readAt, &row.CreatedAt); err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 讀取通知清單失敗: %v", err)
			internalError(w, "unable to read notifications")
			return
		}
		if row.Avatar != "" && mediaToken == "" && !avatarTokenFailed {
			token, tokenErr := s.createMediaToken(r.Context())
			if tokenErr != nil {
				avatarTokenFailed = true
				logger.WarnfContext(r.Context(), "[FORUM] 建立通知清單頭像 token 失敗: %v", tokenErr)
			} else {
				mediaToken = token
			}
		}
		row.Read = readAt.Valid
		items = append(items, s.forumNotificationItemFromRow(row, mediaToken))
	}
	if err := rows.Err(); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取通知清單串流失敗: %v", err)
		internalError(w, "unable to read notifications")
		return
	}

	// 未讀數與清單一起回：前端在同一次請求裡更新紅點。滿頁即猜還有下一頁
	// （與其他列表端點同一套推測）。未讀數用 COUNT 查詢，走
	// idx_forum_notifications_unread 只碰索引。
	unread, err := s.forumUnreadNotificationCount(r.Context(), recipient)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取未讀通知數失敗: %v", err)
		internalError(w, "unable to read notifications")
		return
	}

	writeOK(w, map[string]interface{}{
		"items":       items,
		"hasMore":     len(items) == limit,
		"unreadCount": unread,
	})
}

// markForumNotificationsRead 把一則（或全部）通知標記為已讀。
//
//	POST /api/forum/notifications  body {id}
//
// 回應只回剩下的未讀數：前端拿它更新鈴鐺紅點，而列表本身已經在前端手上
// （剛點開的那一則就地變灰），不必重載整份清單。
func (s *Server) markForumNotificationsRead(w http.ResponseWriter, r *http.Request, recipient string) {
	// 與其他寫入端點相同：跨站偽造防護。路由掛了 requireLogin，但 requireLogin
	// 不含 Origin 檢查。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	var req forumNotificationMarkRequest
	// 讀不出 body 或 body 為空都當成「全部已讀」（id 維持 0）—— 前端「全部已讀」
	// 就是送一個空物件或空 body，不值得為它多開一條路徑。
	// 只有「body 存在但不是合法 JSON」才回 400：io.EOF 代表沒有 body，那是
	// 「全部已讀」而不是錯誤。
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		badRequest(w, "invalid request")
		return
	}

	now := time.Now()
	var err error
	if req.ID > 0 {
		// recipient_email 放進 WHERE 是不可省略的一關：少了它，任何人只要猜
		// 得到別人的通知 id 就能把它標成已讀。
		// read_at IS NULL 讓已讀的列不被重複更新（read_at 記錄的是「第一次
		// 讀到的時間」）。id > 0 且沒有影響任何列代表「不是我的通知」或
		// 「本來就已讀」—— 兩種都不是錯誤（重複點擊是正常行為），直接回
		// 目前的未讀數即可。
		_, err = s.db.ExecContext(r.Context(),
			`UPDATE forum_notifications SET read_at = ? WHERE id = ? AND recipient_email = ? AND read_at IS NULL`,
			now, req.ID, recipient)
	} else {
		_, err = s.db.ExecContext(r.Context(),
			`UPDATE forum_notifications SET read_at = ? WHERE recipient_email = ? AND read_at IS NULL`,
			now, recipient)
	}
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 標記通知已讀失敗: %v", err)
		internalError(w, "unable to update notifications")
		return
	}

	unread, err := s.forumUnreadNotificationCount(r.Context(), recipient)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取未讀通知數失敗: %v", err)
		internalError(w, "unable to read notifications")
		return
	}
	writeOK(w, map[string]interface{}{"unreadCount": unread})
}

// handleForumNotificationUnreadCount 只回未讀數，供導覽列鈴鐺紅點使用。
//
//	GET /api/forum/notifications/unread-count
//
// 這是獨立的端點而不是從列表端點的參數切成「只要數字」：導覽列的鈴鐺在每個
// 頁面載入時都會問一次，而它不該為了一個整數拉回整份清單。
func (s *Server) handleForumNotificationUnreadCount(w http.ResponseWriter, r *http.Request) {
	email := s.sessions.ResolveUser(r)
	if email == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	unread, err := s.forumUnreadNotificationCount(r.Context(), email)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取未讀通知數失敗: %v", err)
		internalError(w, "unable to read notifications")
		return
	}
	writeOK(w, map[string]interface{}{"unreadCount": unread})
}

// forumUnreadNotificationCount 查詢某人的未讀通知數。
//
// 條件走 idx_forum_notifications_unread (recipient_email, read_at)，
// 因此這是一支只碰索引的查詢 —— 它是整個功能裡呼叫最頻繁的 SQL
// （每一頁載入都問一次），成本必須壓在索引層。
//
// 參數是 context.Context：除了 HTTP handler（傳 r.Context()）之外，通知串流的
// 推播與初始訊息也在沒有請求的 goroutine 裡呼叫它（見
// forum_notification_stream.go 的檔頭）。
func (s *Server) forumUnreadNotificationCount(ctx context.Context, recipient string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM forum_notifications WHERE recipient_email = ? AND read_at IS NULL`,
		recipient).Scan(&count)
	return count, err
}
