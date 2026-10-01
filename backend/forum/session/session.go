package session

/*
Package session 負責論壇的登入態（session）簽發、驗證與撤銷。

對外介面
  - NewManager：建構 Manager。
  - Manager.Create / SetCookie / ClearCookie / Refresh / RemoveByRequest：登入、續期與登出。
  - Manager.ResolveUser / IsAdmin / IsLoggedIn：供 httpapi 的中介層做身分判斷。
  - Manager.StartCleanup：保留的相容性空實作。

關鍵設計決策
  1. cookie 內「沒有」簽章，也沒有金鑰。cookie 存放的是 24 位元組 crypto/rand
     隨機數的 hex 字串（192 bits 熵），它只是 Redis 的查詢鍵，自身不攜帶任何
     可解讀的使用者資料。竄改 cookie 頂多得到一個查不到的鍵（＝未登入），
     不會讓攻擊者偽造出他人身分。若日後要改為「簽章 cookie（無狀態）」，
     就必須讓 cookie 不再攜帶 email，並自行維護 HMAC 金鑰與簽章有效期。
  2. 真正的權威狀態放在 Redis（FORUM:session:<token> 這個 hash，含 email 與
     is_admin），所以可以立即撤銷（RemoveByRequest 直接刪 key），不必等 token
     自然過期；這是選用 Redis 而非自簽 JWT 的主要理由。
  3. Redis key 以 FORUM:session: 前綴與同實例的媒體 token（FORUM:token:）及其他
     資料分隔，避免多個功能共用同一個 Redis DB 時互相覆蓋。
  4. 過期採滑動視窗：server.go 在最外層對每個請求呼叫 Refresh，把 TTL 重設為
     完整的 expire。使用者操作期間不會被登出，只有靜置超過 expire 才失效。

安全假設與限制
  - Manager 建構後所有欄位皆不再變更，因此可安全地被多個 goroutine 併發使用；
    本型別不需要 mutex，go-redis 的 *redis.Client 本身也具備併發安全性。
  - 這些方法只能在該請求自己的 handler goroutine 內呼叫：Refresh 會寫入
    Set-Cookie 表頭，而 net/http 規定 ResponseWriter 在 handler 回傳後不可再
    使用，因此 Refresh 必須在回應開始前執行（server.go 的最外層包裝已如此做）。
  - 所有 Redis 呼叫一律使用 context.Background() 而非 r.Context()，讓續期與
    撤銷不會因用戶端中途斷線而被取消；代價是放棄請求層級的逾時控制。
  - token 在 session 存續期間不會輪替，也不綁定 IP 或 UA，因此遭竊的 cookie
    在過期前可被完整重用；唯一能提前失效的手段是登出或直接清除 Redis key。
    後臺的「列出 session」與「強制登出」是為了讓這件事可管理，見 session_list.go。

  4. 建立時間寫進 hash（created_at），但**不回傳完整 token 給任何介面** ——
    token 就是憑證，因此後臺只顯示前 8 個字元。詳見 session_list.go 檔頭。
*/

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Info 描述一支 session 在 Redis hash 中的語意欄位，欄位說明如下：
//
//	Expiry     這支 session 的到期時間點。Redis 以 key 的 TTL 為準，寫入時不會
//	           另外存時間戳，因此要查到期時間必須讀 TTL，而不是這個欄位。
//	IsAdmin    對應 hash 欄位 is_admin。管理員與否在登入當下由
//	           config.IsAdminEmail 判定後寫死，不是每次請求即時重算。
//	Email      對應 hash 欄位 email，即論壇唯一的身分識別。
//	CreatedAt  對應 hash 欄位 created_at（Unix 秒）。這是後來才加入的欄位，
//	           因此既有 session 沒有它；讀取時缺欄位要回零值而不是猜一個
//	           （見 session_list.go 的 Record.CreatedAt 說明）。
//
// 本型別與 List 回傳的 Record 並存，差別在用途：Info 是「查單一支 session」
// 的結果，Record 是「掃描出很多支」的結果。保留 Info 是為了不動既有呼叫端。
type Info struct {
	Expiry  time.Time
	IsAdmin bool
	Email   string
}

// Manager 集中持有簽發與驗證 session 所需的設定與 Redis 連線，欄位說明如下：
//
//	cookieName   session cookie 的名稱，來自設定檔 COOKIE_NAME，預設
//	             FORUM_forum；改名等同讓全體使用者被登出。
//	expire       session 的存續時間，同時作為 Redis key 的 TTL 與 cookie 的
//	             到期時間。預設 72 小時，並會在每個請求被 Refresh 重設。
//	cookieSecure 決定 cookie 是否加上 Secure 屬性。正式環境走 HTTPS 時必須為
//	             true，本機以 HTTP 開發才關閉，否則瀏覽器不會送回 cookie。
//	rdb          與主程式共用的 Redis 連線（見 main.go），session 與媒體
//	             token 使用同一個實例。
//
// 建構後所有欄位皆唯讀，因此不需要鎖；同一個 *Manager 可以安全地同時被多個
// HTTP handler goroutine 使用。
type Manager struct {
	cookieName   string
	expire       time.Duration
	cookieSecure bool
	rdb          *redis.Client
}

// sessionKeyPrefix 是 session key 的命名空間前綴，作用有二：讓同實例的其他
// 功能（媒體 token 為 FORUM:token:）不會撞到相同的 key，也讓 Redis 內的 key
// 一眼可辨識歸屬。
const sessionKeyPrefix = "FORUM:session:"

// NewManager 建立一個 session Manager。
// cookieName 與 cookieSecure 直接決定瀏覽器如何保存與送回憑證；expire 同時
// 影響 Redis TTL 與 cookie 到期時間，兩者不一致會造成「cookie 還在但 session
// 已過期」或反之的困惑狀態，因此只由設定檔提供單一值。
// rdb 由呼叫端（main.go）注入，讓本套件不需自行連線與處理連線錯誤。
// 這個函式不做任何 I/O，也不會失敗；回傳的 Manager 從此保持不變。
func NewManager(cookieName string, expire time.Duration, cookieSecure bool, rdb *redis.Client) *Manager {
	return &Manager{
		cookieName:   cookieName,
		expire:       expire,
		cookieSecure: cookieSecure,
		rdb:          rdb,
	}
}

// StartCleanup 原本預期啟動一個定期清除過期 session 的背景 goroutine。
// 現在 session 的生命週期完全交給 Redis 的 key TTL 自動處理，記憶體與磁碟都不
// 會殘留過期資料，因此不需要任何清理迴圈；簽章保留是為了維持 API 相容性。
// interval 參數因而被刻意忽略，呼叫端（若有的話）不需因為清理功能而改動。
func (m *Manager) StartCleanup(interval time.Duration) {
	// Redis expires session keys automatically; retained for API compatibility.
}

// Create 為已通過身分驗證的使用者建立一筆 session，並回傳要寫進 cookie 的
// token。email 與 isAdmin 由呼叫端（Google 驗證回呼）決定，後續變更（例如
// 移出管理員名單）不會回寫到既有 session，必須等其過期或另行撤銷。
// 錯誤情況：隨機數產生失敗（僅在作業系統亂數來源失效時）會直接回傳，
// 絕不回傳弱隨機 token；Redis 寫入失敗則代表無法建立 session，呼叫端應
// 視為登入失敗而非降級成免驗證。
// 副作用：在 Redis 建立 FORUM:session:<token> 的 hash，並設定 TTL。
func (m *Manager) Create(email string, isAdmin bool) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}

	// token 直接當作 key 的一部分，未另加雜湊：攻擊者若能猜中 token 就等同
	// 取得該 session，但 192 bits 熵使猜測不可行，且省掉一層不必要的計算。
	key := sessionKey(token)
	// 使用 context.Background()：這是登入後的第一個請求，若因用戶端在
	// 寫入途中斷線而取消，會留下無法預期的半套 session 狀態。
	ctx := context.Background()
	// isAdmin 存成字串 "true"/"false"：Redis hash 的值只有字串型別，
	// 用 strconv.FormatBool 與 Read 時的字串比較保持對稱。
	// HSet 而非 Set，是為了之後擴充欄位時不必改變 key 的資料型別。
	//
	// created_at 是後來才加入的欄位（見 List 的說明）：它讓後臺的 session 列表
	// 能顯示「這支 session 是什麼時候建立的」。既有 session 沒有這個欄位，
	// 而那一類 session 在介面上會顯示為「未知」—— 補寫回去是不可能的
	// （建立時間已經過去了），而猜一個值只會產生比「不知道」更糟的資料。
	//
	// 存 Unix 秒而非 RFC3339：Unix 秒是固定長度的數字，比較與顯示都直接，
	// 而且不帶時區資訊（UTC 這個事實由寫入端與讀取端各自固定，見
	// session_list.go 的 CreatedAt 處理）。
	if err := m.rdb.HSet(ctx, key,
		"email", email,
		"is_admin", strconv.FormatBool(isAdmin),
		createdAtField, strconv.FormatInt(time.Now().Unix(), 10),
	).Err(); err != nil {
		return "", err
	}
	// HSet 與 Expire 分成兩道指令而非交易或 pipeline：兩者之間若當機，會留下
	// 一個沒有 TTL 的常駐 session，等同永久後門。因此 Expire 失敗時要主動
	// 刪除 key，讓「設定 TTL 失敗」一律以登入失敗收場（fail closed）。
	if err := m.rdb.Expire(ctx, key, m.expire).Err(); err != nil {
		_ = m.rdb.Del(ctx, key).Err()
		return "", err
	}

	return token, nil
}

// SetCookie 把 token 寫成瀏覽器 cookie。
// 屬性設定的理由：
//   - Path "/"：前端、/forum、/admin 與 /api 都共用同一張 session cookie，
//     限制路徑會讓部分頁面讀不到登入態。
//   - HttpOnly：JavaScript 無法讀取，阻擋以 XSS 竊取 session 的路徑。
//   - Secure：由設定決定；正式環境為 true 時 cookie 只在 HTTPS 傳輸。
//   - SameSite=Lax：允許使用者從外部連結「點進來」時仍帶著登入態（頂層導覽），
//     但跨站 POST 不會附帶 cookie，這是防止 CSRF 的第一道防線；csrf.go 的
//     Origin/Referer 比對則是第二道。
//   - 用 Expires 而非 MaxAge：寫成絕對時間點，與 Redis 的 TTL 起算方式無關，
//     但兩者都由同一個 expire 推導，語意一致。
//
// 副作用：向 w 追加一個 Set-Cookie 表頭，必須在寫出回應之前呼叫。
func (m *Manager) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(m.expire),
	})
}

// ClearCookie 請瀏覽器立即刪除 session cookie。
// MaxAge 設為 -1 是 net/http 送出「立即到期」的唯一寫法；瀏覽器比對的是
// name + domain + path 三者，所以 Path 必須與 SetCookie 完全一致，否則舊的
// cookie 會被保留，使用者看起來「登出後仍登入」。
// 這只清除瀏覽器端；伺服器端的 key 需由呼叫端另外呼叫 RemoveByRequest 刪除。
func (m *Manager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// RemoveByRequest 依請求帶來的 cookie 刪除對應的 Redis key，這是登出時真正
// 生效的撤銷動作（單方面刪 cookie 不足以登出，使用者仍可重送舊 token）。
// 沒有 cookie 就直接返回，因為沒有 token 也就沒有東西可撤銷。
// 刪除結果刻意忽略：撤銷是冪等的，失敗時下一個請求仍會因 key 不存在而視為
// 未登入，無須把錯誤往上丟到使用者看得見的地方。
func (m *Manager) RemoveByRequest(r *http.Request) {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil {
		return
	}
	_ = m.rdb.Del(context.Background(), sessionKey(cookie.Value)).Err()
}

// Refresh 在每個請求進入業務邏輯之前被呼叫，負責把滑動過期時間往後推。
// 之所以要對「每個」請求做，是因為後續的驗證方法（ResolveUser 等）並不做
// 續期；一旦漏掉任何路徑，使用者就會在瀏覽過程中被登出。
// 回傳 true 表示這支 session 目前有效且 TTL 已延長；false 代表 cookie 缺失
// 或 session 已過期／已被撤銷，此時不會重發 cookie，讓瀏覽器自然過期。
// 錯誤一律收斂成 false：續期失敗不該讓整個請求失敗，但呼叫端也無法據此
// 區分「未登入」與「Redis 暫時故障」。
// 副作用：會寫入 Set-Cookie 表頭，因此必須在回應尚未開始前呼叫；
// 呼叫端（server.go）把它放在最外層包裝，確保早於任何 handler 輸出。
func (m *Manager) Refresh(r *http.Request, w http.ResponseWriter) bool {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil {
		return false
	}

	key := sessionKey(cookie.Value)
	// 同樣用 Background：續期是「順帶」做的維護工作，不該被用戶端斷線取消，
	// 否則使用者快速切換分頁就可能讓 session 意外過期。
	ctx := context.Background()
	// 先確認 key 存在再續期：對不存在的 key 下 Expire 不會回報錯誤，只會回傳 0，
	// 若省略這個檢查，已過期的 session 仍會被重發 cookie 而看起來「永遠有效」。
	if m.rdb.Exists(ctx, key).Val() != 1 {
		return false
	}
	if err := m.rdb.Expire(ctx, key, m.expire).Err(); err != nil {
		return false
	}

	// 只有成功續期才重發 cookie，讓瀏覽器端的到期時間與 Redis TTL 同步；
	// 沿用原本的 token，不在續期時輪替（見檔頭安全假設的最後一點）。
	m.SetCookie(w, cookie.Value)
	return true
}

// ResolveUser 回傳目前請求對應的登入者 email，未登入時回傳空字串。
// 每次呼叫都重新讀 Redis 而不在請求內快取，好處是管理端停權或刪除 session
// 會立刻對所有進行中的請求生效；代價是每個需要身分的請求都要付一次
// Redis 往返延遲。
// key 不存在與 Redis 故障都回傳空字串，呼叫端（如 requireLogin）因此會把兩者
// 一律當成未登入：寧可導回登入頁，也不要在 Redis 不穩時放行未驗證的請求。
func (m *Manager) ResolveUser(r *http.Request) string {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil {
		return ""
	}

	email, err := m.rdb.HGet(context.Background(), sessionKey(cookie.Value), "email").Result()
	if err == nil {
		return email
	}
	return ""
}

// IsAdmin 判斷目前請求是否為管理員。
// 管理員身分不是從 cookie 或 email 推算，而是登入當下由
// config.IsAdminEmail 比對設定檔後寫進 session hash（is_admin 欄位），
// 因此這裡只做字串比對；cookie 內容本身不影響結果，攻擊者無法藉由改寫
// cookie 取得管理權限。
// 與 ResolveUser 相同，任何 Redis 錯誤都視為「非管理員」（fail closed）。
// 注意此方法不做「非管理員的停權檢查」，那是 httpapi.requireLogin 的責任。
func (m *Manager) IsAdmin(r *http.Request) bool {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil {
		return false
	}

	// 與 Create 時 strconv.FormatBool 的輸出比對，字串必須完全相同才為 true。
	isAdmin, err := m.rdb.HGet(context.Background(), sessionKey(cookie.Value), "is_admin").Result()
	return err == nil && isAdmin == "true"
}

// IsLoggedIn 判斷 cookie 帶來的 token 是否對應一筆仍存在的 session。
// 只檢查 key 是否存在，不讀欄位，是最便宜的身分確認方式；「/api/check」這類
// 只想知道登入狀態的端點用它即可，不必多打一次 HGet。
// 過期、已登出或 token 遭竄改都會讓 key 不存在而回傳 false。
func (m *Manager) IsLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil {
		return false
	}

	return m.rdb.Exists(context.Background(), sessionKey(cookie.Value)).Val() == 1
}

// sessionKey 把 token 組成本套件專屬的 Redis key。
// 這是整個「token 即憑證」設計的關鍵：cookie 的值只被當成查詢鍵使用，
// 任何驗證都回到 Redis 完成，因此不需要簽章金鑰，也不存在「竄改 cookie 內容
// 就能改變身分」的路徑。
func sessionKey(token string) string {
	return sessionKeyPrefix + token
}

// newToken 產生 24 位元組（192 bits）的密碼學隨機 token，並以 hex 回傳。
// 選用 crypto/rand 而非 math/rand：這段輸出就是 session 憑證本身，必須不可
// 預測；math/rand 的序列可由少量觀測值還原，等同開放門戶。
// 24 位元組足以讓生日攻擊的碰撞機率約為 2^-144，可忽略；hex 編碼讓結果是
// 固定 48 個 [0-9a-f] 字元，天然符合 cookie value 的字元集，無需跳脫，
// 且每個位元組固定佔 2 個字元。
// 錯誤只在作業系統亂數來源不可用時發生；此時回傳錯誤而絕不回傳弱 token。
func newToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
