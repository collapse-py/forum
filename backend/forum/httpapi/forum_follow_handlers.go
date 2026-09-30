/*
追蹤使用者（backend/forum/httpapi/forum_follow_handlers.go）

一、HTTP 路由對照（註冊於 server.go 的 Handler()）

	GET  /api/forum/follows          -> handleForumFollows        我追蹤中的使用者（需登入）
	POST /api/forum/follows          -> handleForumFollows        切換追蹤／取消追蹤
	GET  /api/forum/following/posts  -> handleForumFollowingPosts 追蹤中的人的貼文動態（需登入）

二、為什麼是「追蹤使用者」而不是別的對象

	論壇沒有 thread 結構（貼文與留言是兩張表，沒有回覆串的層級），因此「追蹤一篇
	討論」沒有可對應的實體；使用者則是全站唯一有穩定識別（publicForumKey）、有內容
	（forum_posts.author_email）又有個人頁的對象。

三、資料表依賴

	forum_follows   追蹤關聯，複合主鍵 (follower_email, target_email)
	forum_posts     追蹤動態的資料來源（condition 由 handleForumFollowingPosts 組出）
	forum_profiles  清單端點取暱稱用（LEFT JOIN，讓沒建檔的對象仍出現在清單裡）

四、四條貫穿全檔的設計決策

 1. 完全私有。沒有任何端點回傳「誰追蹤了誰」或「某人有幾個追蹤者」。這不只是
    少做功能：公開 API 一律以 publicForumKey 代表身分，開放追蹤關係等於新增一條
    可用來還原 email 的關聯路徑（追蹤表存的就是 email）。回傳給本人的只有
    「我追蹤了哪些 key」與「我是否追蹤了這個人」。
 2. 關聯鍵是 email，API 表面只有 64 個十六進位元的 publicKey。理由見 MigrateMySQL
    第 19 步：存 key 會讓追蹤動態的條件變成 SHA2(author_email,256) IN (...)，
    MySQL 無法為運算式建索引。
 3. 追蹤狀態不放進 forumPost。動態、追蹤動態與搜尋結果三種來源都渲染同一支
    PostCard，但追蹤狀態是「一頁一次」而非「一篇一次」：GET /api/forum/follows
    一次回全部，貼文卡以 authorKey 比對。若把它做成每篇貼文的欄位，三個查詢都要
    各加一個相關子查詢，而結果完全相同。
 4. 切換用單一 POST 而沒有獨立的 DELETE，理由與按讚完全相同
    （見 handleForumPostLike）：前端按鈕永遠只需要「切到相反狀態」，
    這種語意用單一 POST 表達最不容易出錯。
*/
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"forum/forum/logger"
)

// forumFollowRequest 是 POST /api/forum/follows 的請求主體。
//
//	Key 是被追蹤者的 publicForumKey（email 的 SHA-256，64 個十六進位字元）。
//
// 刻意不接受 email：公開 API 全程拿不到原始位址，這是本站個資設計的底線
// （見 publicForumKey 的說明）。追蹤對象的身分由金鑰決定，伺服器自行解析。
type forumFollowRequest struct {
	Key string `json:"key"`
}

// forumFollowTarget 是 GET /api/forum/follows 回傳的單一對象。
//
//	Key         publicForumKey(target_email)，由伺服器現算 —— 資料庫裡沒有也不需要
//	            存這一欄。email 永不出現在回應中。
//	Nickname    forum_profiles.nickname 原樣回傳（可能為空字串）。刻意不用
//	            forumAuthor()：那個函式對沒有暱稱的人回的是「每次請求都不同」的
//	            匿名代號，會讓這份清單在重新載入時整列跳動。空字串交給前端顯示
//	            t('publicProfile.anonymous')，那才是穩定的呈現。
//	FollowedAt  追蹤建立的時間，供前端顯示「加入追蹤於…」。
type forumFollowTarget struct {
	Key        string    `json:"key"`
	Nickname   string    `json:"nickname"`
	FollowedAt time.Time `json:"followedAt"`
}

// forumFollowToggleResponse 是 POST /api/forum/follows 的回應。
//
//	Following 切換「後」的狀態而非切換前。連點兩下的結果（例如第二下被合併成
//	          一次取消）只有後端知道，因此前端不該在本地猜。
type forumFollowToggleResponse struct {
	Key       string `json:"key"`
	Following bool   `json:"following"`
}

// handleForumFollows 是 /api/forum/follows 的單一進入點，依 HTTP 方法分派。
//
// 路由已用 requireLogin 包住，因此兩種方法都必須登入且未被停權；handler 內仍
// 重複確認 email 非空，讓它被單獨重用或日後改動路由時仍然安全。
func (s *Server) handleForumFollows(w http.ResponseWriter, r *http.Request) {
	email := s.sessions.ResolveUser(r)
	if email == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listForumFollows(w, r, email)
	case http.MethodPost:
		s.toggleForumFollow(w, r, email)
	default:
		// 沒有在此處補 Allow 標頭，直接以 405 結束，前端只需處理單一錯誤路徑。
		methodNotAllowed(w)
	}
}

// listForumFollows 回傳「我追蹤中的使用者」分頁。
//
// 這是一個純私有的讀取端點：回應裡的每一個 key 都是自己按過的鈕，
// 不含任何「被誰追蹤」的資訊。
func (s *Server) listForumFollows(w http.ResponseWriter, r *http.Request, follower string) {
	// 單頁上限 50。與貼文動態的 25 不同是因為這一列的成本只有一筆 join，
	// 但使用者通常追蹤得不多，給大一點可以讓多數情況一次載完、不必顯示
	// 「載入更多」。超過 50 則要求呼叫端分頁 —— 不設上限的話，一個惡意的
	// 追蹤數（唯一能變大的方式是把每篇文章的作者都追蹤一遍）會讓這支查詢
	// 變成無上限的全表掃描。
	const pageSize = 50

	offset, ok := forumOffsetParam(w, r)
	if !ok {
		return
	}

	// LEFT JOIN 而非 JOIN：追蹤對象可能從未建立 forum_profiles（早期使用者，
	// 見 MigrateMySQL 第 13 步的回填）。用 INNER JOIN 會讓這筆追蹤從清單裡消失，
	// 但他的貼文仍然出現在追蹤動態裡 —— 清單與動態的對象集合必須一致。
	// 排序以 created_at 為主、target_email 為次：同一毫秒內追蹤兩個人時
	// 仍然有確定順序，避免重新載入後順序跳動。
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT f.target_email, COALESCE(p.nickname, ''), f.created_at
		FROM forum_follows f
		LEFT JOIN forum_profiles p ON p.author_email = f.target_email
		WHERE f.follower_email = ?
		ORDER BY f.created_at DESC, f.target_email ASC
		LIMIT ? OFFSET ?`, follower, pageSize, offset)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 載入追蹤清單失敗: %v", err)
		internalError(w, "unable to load forum follows")
		return
	}
	defer rows.Close()

	items := make([]forumFollowTarget, 0)
	for rows.Next() {
		var targetEmail, nickname string
		var followedAt time.Time
		if err := rows.Scan(&targetEmail, &nickname, &followedAt); err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 讀取追蹤清單失敗: %v", err)
			internalError(w, "unable to read forum follows")
			return
		}
		// key 在應用層現算：publicForumKey 是 email 的確定性雜湊，
		// 因此這個值與貼文卡上的 authorKey 必然一致（那是同一個函式算出來的）。
		// 也因為是現算，email 從不離開後端。
		items = append(items, forumFollowTarget{
			Key:        publicForumKey(targetEmail),
			Nickname:   nickname,
			FollowedAt: followedAt,
		})
	}
	if err := rows.Err(); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取追蹤清單串流失敗: %v", err)
		internalError(w, "unable to read forum follows")
		return
	}

	writeOK(w, map[string]interface{}{
		"items": items,
		// 與貼文列表同一套推測：滿頁即猜還有下一頁。
		"hasMore": len(items) == pageSize,
		// selfKey 是「我自己的 publicKey」。前端需要它來做兩件事：
		//   - 貼文卡上遇到自己的文章時不渲染追蹤鈕（自我追蹤沒有意義）
		//   - 個人頁判斷「這個對象是不是我」
		// 之所以由這支端點順便回傳而不動 /api/check：/api/check 是首頁與
		// 所有殼層共用的端點，改它的回應形狀會牽動每一頁；而每一個渲染
		// PostCard 或顯示追蹤鈕的頁面都已經需要呼叫這支端點了。
		"selfKey": publicForumKey(follower),
	})
}

// toggleForumFollow 切換追蹤狀態：已追蹤則取消，未追蹤則新增。
//
// 檢查順序刻意是「身分 → Origin → 輸入」，與 handleForumPostLike 相同：
// Origin 驗證成本最低且擋掉跨站偽造，放在碰資料庫之前可避免無謂的交易開銷。
func (s *Server) toggleForumFollow(w http.ResponseWriter, r *http.Request, follower string) {
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	var req forumFollowRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		badRequest(w, "invalid follow request")
		return
	}
	key := strings.TrimSpace(req.Key)
	target, err := s.resolveFollowTarget(r.Context(), follower, key)
	if err != nil {
		s.writeFollowTargetError(w, r, err)
		return
	}

	// 交易邊界涵蓋「讀取目前狀態 → 寫入」，理由與 handleForumPostLike 相同：
	// 沒有交易時兩個並發請求可能都讀到未追蹤而都送出 INSERT。
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 開始追蹤交易失敗: %v", err)
		internalError(w, "unable to begin follow transaction")
		return
	}
	// 安全網：本函式所有提早 return 的分支都會走到這裡。Commit 成功之後再次
	// 呼叫 Rollback 會回 sql.ErrTxDone，屬預期且被忽略。
	defer tx.Rollback()

	// 主鍵是 (follower_email, target_email)，因此這筆查詢是唯一的（最多一列），
	// 結果只可能是 0 或 1。
	var existing int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM forum_follows WHERE follower_email = ? AND target_email = ?`,
		follower, target).Scan(&existing); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 檢查追蹤狀態失敗: %v", err)
		internalError(w, "unable to check follow status")
		return
	}

	following := existing > 0
	if following {
		// DELETE 而非 UPDATE 設成某個停用旗標：追蹤沒有「歷史」語意，
		// 讓資料列完全消失可以讓唯一的索引同時是複合主鍵，保持結構簡單
		// （與 forum_post_likes 同樣的取捨）。
		if _, err := tx.ExecContext(r.Context(),
			`DELETE FROM forum_follows WHERE follower_email = ? AND target_email = ?`,
			follower, target); err != nil {
			logger.ErrorfContext(r.Context(), "[FORUM] 取消追蹤失敗: %v", err)
			internalError(w, "unable to remove follow")
			return
		}
	} else if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO forum_follows (follower_email, target_email, created_at) VALUES (?, ?, ?)`,
		follower, target, time.Now()); err != nil {
		// 並發下兩個請求都可能通過上面的檢查，複合主鍵會讓其中一個收到
		// 1062。刻意不為它特別分流（handleForumReport 就有分流）：追蹤只是
		// 一個開關，使用者再按一次即可，與金流或權限無關。
		logger.ErrorfContext(r.Context(), "[FORUM] 新增追蹤失敗: %v", err)
		internalError(w, "unable to add follow")
		return
	}

	if err := tx.Commit(); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 儲存追蹤失敗: %v", err)
		internalError(w, "unable to save follow")
		return
	}

	writeOK(w, forumFollowToggleResponse{Key: key, Following: !following})
}

/*
 * followTargetError 是「這組追蹤輸入能不能處理」的三種失敗。
 *
 * 抽出型別而不是回傳 (int, string) 配對，是為了讓呼叫端必須用 switch 窮舉 ——
 * 未來新增一種失敗時漏掉一個分支，編譯器就會指出來。配對回傳值不會有這種保護。
 */
type followTargetError int

const (
	// followTargetBadKey：金鑰長度不對，400。
	followTargetBadKey followTargetError = iota
	// followTargetNotFound：金鑰格式正確但沒有對應的使用者，404。
	followTargetNotFound
	// followTargetSelf：追蹤自己，400。
	followTargetSelf
	// followTargetLookupFailed：金鑰格式正確但查詢失敗（不是「找不到」），500。
	//
	// 刻意與 followTargetNotFound 分開：查詢失敗時回 404 會讓使用者以為對方不存在，
	// 而實際上是站台故障 —— 兩者的下一步動作完全不同。
	followTargetLookupFailed
)

// Error 讓 followTargetError 滿足 error 介面，因此可以把「哪一種失敗」當成
// error 往上傳，而不必在每一層都多帶一個型別參數。
func (e followTargetError) Error() string {
	switch e {
	case followTargetBadKey:
		return "invalid profile key"
	case followTargetNotFound:
		return "user not found"
	case followTargetSelf:
		return "cannot follow yourself"
	default:
		return "unable to resolve forum user"
	}
}

/*
 * checkFollowTargetInput 檢查金鑰格式與「不是自己」兩條規則。
 *
 * 這兩條都不依賴資料庫狀態，因此可以在任何交易之前擋掉 —— 自我追蹤尤其重要：
 * 讓它開一個交易再回 400 只是在白花一次往返。
 *
 * 會被擋掉的原因：
 *   - 長度不對：真正的比對靠 public_key 的索引，長度檢查只是提早拒絕明顯無效的輸入。
 *   - 追蹤自己：沒有任何意義（自己的貼文本來就在首頁），而複合主鍵
 *     (follower_email, target_email) 無法擋掉 ('x','x') 這種合法的重複組合。
 *     這是「唯一只能靠應用層守住」的資料完整性規則。
 *
 * 回傳 nil 代表這條輸入可以拿去解析。
 */
func checkFollowTargetInput(follower, key string) error {
	if len(key) != 64 {
		return followTargetBadKey
	}
	if key == publicForumKey(follower) {
		// 直接比對 follower 自己的雜湊，不等於「解析之後才比較」：那會讓自我
		// 追蹤多花一次查詢（而且解析走的是退路掃描時，是全表掃描）。
		return followTargetSelf
	}
	return nil
}

// resolveFollowTarget 檢查輸入規則並把 publicKey 還原成 email。
func (s *Server) resolveFollowTarget(ctx context.Context, follower, key string) (string, error) {
	if problem := checkFollowTargetInput(follower, key); problem != nil {
		return "", problem
	}
	email, err := s.forumEmailByPublicKey(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", followTargetNotFound
	}
	if err != nil {
		return "", followTargetLookupFailed
	}
	return email, nil
}

// writeFollowTargetError 把 followTargetError 翻譯成對外的回應。
//
// 404 與 500 的分界理由見 followTargetError 的定義；400 的兩種訊息分開寫，
// 是為了讓「金鑰格式錯」與「不能追蹤自己」在日誌與使用者端都可辨識。
func (s *Server) writeFollowTargetError(w http.ResponseWriter, r *http.Request, err error) {
	var problem followTargetError
	if !errors.As(err, &problem) {
		// 不是這三種之一就當作查詢失敗處理。
		problem = followTargetLookupFailed
	}
	switch problem {
	case followTargetBadKey:
		badRequest(w, "invalid profile key")
	case followTargetSelf:
		badRequest(w, "cannot follow yourself")
	case followTargetNotFound:
		writeError(w, http.StatusNotFound, "user not found")
	default:
		logger.ErrorfContext(r.Context(), "[FORUM] 解析追蹤對象失敗: %v", err)
		internalError(w, "unable to resolve forum user")
	}
}

// forumEmailByPublicKey 把 publicKey 還原成 email。
//
// 兩段查詢，理由是「哪些人算得出 publicKey」與「哪些人有 public_key 欄位」
// 這兩件事並不對等：
//
//	第一段走 forum_profiles.public_key（ MigrateMySQL 第 20 步已補上索引）。
//	這涵蓋所有設定過暱稱的使用者，是絕大多數的情況。
//
//	第二段是退路。貼文卡上的 authorKey 對「每一個」作者都算得出來
//	（publicForumKey(email) 不需要任何資料表），但 public_key 只存在於
//	forum_profiles。早期使用者（帳號由 MigrateMySQL 第 13 步從既有貼文回填、
//	從未建過檔）因此會有 authorKey 卻查不到 email。只做第一段的話，他們的
//	追蹤鈕會恆回 404 —— 一個沒有任何錯誤訊息、只有按不動的按鈕。
//
//	為什麼不能用「順手建檔」取代退路：forum_profiles.nickname 是 NOT NULL 且有
//	唯一索引 uq_forum_profiles_nickname，多筆空暱稱會撞 1062，要自動建檔就必須
//	先改那個索引 —— 那是一個影響「暱稱不得重複」語意的結構變更，不該由這個功能
//	順手決定。因此退路保留為一段有界的掃描：SELECT DISTINCT 的結果是「曾經發過文
//	的人數」，遠小於貼文總數，且這個端點掛在內容寫入限流（預設 10 次 / 60 秒）之下。
//
//	回傳 sql.ErrNoRows 表示兩段都找不到，呼叫端據此回 404。
func (s *Server) forumEmailByPublicKey(ctx context.Context, key string) (string, error) {
	var email string
	err := s.db.QueryRowContext(ctx,
		`SELECT author_email FROM forum_profiles WHERE public_key = ? LIMIT 1`, key).Scan(&email)
	if err == nil {
		return email, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		// 查詢本身失敗（例如連線中斷）不是「找不到」，必須往上回報，
		// 否則會靜默地落到退路並得到一個誤導的 404。
		return "", err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT author_email FROM forum_posts`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var candidate string
		if err := rows.Scan(&candidate); err != nil {
			return "", err
		}
		// 雜湊比對在應用層做：SHA-256 是確定性的，兩邊演算法一致
		// （與第 16 步的 SQL SHA2 回填位元級相同）。
		if publicForumKey(candidate) == key {
			return candidate, rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "", sql.ErrNoRows
}

// handleForumFollowingPosts 回傳「我追蹤中的人」的貼文動態。
//
// 與 listForumPosts 的差別只有 WHERE 子句，其餘（欄位、計數、去識別化、圖片 token、
// 分頁推測）全部走同一支 loadForumPosts —— 兩種動態因此不會出現「一邊顯示讚數、
// 另一邊不顯示」這種漂移。
//
// 這是純私有讀取端點（沒有登入就沒有追蹤清單），因此路由掛 requireLogin 而不掛
// 限流：GET 端點限流對匿名訪客才有意義，這裡本來就擋掉了匿名。
//
// 刻意不過濾停權帳號：listForumPosts 不過濾，追蹤動態額外過濾會讓同一篇貼文
// 在一處可見、另一處消失。
func (s *Server) handleForumFollowingPosts(w http.ResponseWriter, r *http.Request) {
	follower := s.sessions.ResolveUser(r)
	// 路由已用 requireLogin 包住（含停權檢查），此處重複確認是縱深防禦。
	if follower == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}

	// 與 listForumPosts 同一個單頁筆數：兩個動態的單頁成本完全相同。
	const pageSize = 25

	offset, ok := forumOffsetParam(w, r)
	if !ok {
		return
	}

	// 條件走 forum_follows 的複合主鍵最左前綴（follower_email），
	// 因此這是索引查詢而不是全表掃描。沒追蹤任何人時子查詢回空集合，
	// 整個條件恆為 false，得到 0 筆而不是錯誤。
	posts, err := s.loadForumPosts(r,
		`WHERE fp.author_email IN (
		     SELECT target_email FROM forum_follows WHERE follower_email = ?
		 )`, []interface{}{follower}, pageSize, offset)
	if err != nil {
		s.writeForumPostLoadError(w, err, "unable to load followed posts")
		return
	}

	writeOK(w, map[string]interface{}{
		"items":   posts,
		"hasMore": len(posts) == pageSize,
	})
}
