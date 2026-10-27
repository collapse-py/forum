package httpapi

/*
oauth_login.go：OAuth 登入的「發起」端 —— 產生一次性的 state nonce 與 PKCE verifier，
把它們存進 Redis，並寫下一份綁定這個瀏覽器的 cookie。

為什麼 state 不再承載「回跳路徑」

	舊版把 return（登入後要回到哪一頁）直接當成 OAuth 的 state。那讓 state 變成一個
	攻擊者可控制的值：攻擊者用自己的帳號走完登入、把結果寫進受害者的瀏覽器，受害者
	就帶著「攻擊者的登入態」回到本站（login CSRF）。站方原本只防住了開放轉向
	（auth.IsSafeReturnPath），沒有防住「這支登入是誰發起的」。

	現在 state 是 24 位元組的隨機 nonce，值本身不攜帶任何語意：
	  1. nonce → {回跳路徑, code_verifier} 寫進 Redis（FORUM:oauth:state:<nonce>，TTL 10 分鐘）
	  2. 同一個 nonce 寫進 HttpOnly、SameSite=Lax、Path=/auth 的 cookie
	  3. 回呼時 state 與那份 cookie 必須相符，而且 Redis 裡那筆必須還在（讀完即刪）

	攻擊者無法偽造這一對：他完成自己的登入時，nonce 存在他自己瀏覽器的 cookie 裡，
	受害者瀏覽器裡沒有那支 cookie，回呼因此驗證失敗。回跳路徑改存在 Redis，因此它
	不再出現在網址列裡，也不被第三方看到。

PKCE（code_verifier）

	授權網址帶 S256 challenge，換 token 時帶 verifier。授權碼被竊取的幾條路徑
	（瀏覽器擴充、Referer、代理日誌）因此換不出 token。verifier 只存在 Redis，
	從不進授權網址、也不進 cookie。

兩樣東西都是「一次性、短壽命、不在使用者界面出現」的資料，因此沒有清理迴圈：
Redis key 靠 TTL 自動消失，用過的 nonce 在回呼時被 GETDEL 拿走。

Redis 故障時 fail closed：拿不到 nonce 就無法驗證回呼，因此寧可讓登入失敗（500），
也不放行一個未驗證的回呼。這與本站其他 Redis 依賴的取向相同 —— 登入本來就需要
Redis（session），Redis 掛掉時登入不可用不是新的故障。
*/

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"forum/forum/auth"
	"forum/forum/logger"

	"github.com/redis/go-redis/v9"
)

const (
	// oauthStateTTL 是 nonce 的存續時間。10 分鐘是「使用者在 Google 的授權頁上停留」
	// 的合理上限：超過就視為這個流程已被放棄。刻意不設更長 —— nonce 活越久，
	// 「拿舊 nonce 重播」的窗口越大。
	oauthStateTTL = 10 * time.Minute

	// oauthStateCookieName 是綁定 cookie 的名字。它不是 session cookie，也不帶任何
	// 身分資訊；名字與 session 分開，是為了「登入流程」與「登入態」不會互相誤認。
	oauthStateCookieName = "FORUM_oauth_state"

	// oauthStateKeyPrefix 是 Redis 裡的命名空間，與 session（FORUM:session:）和
	// 媒體 token（FORUM:token:）分開，同一個 Redis 實例裡三者的 key 不會互相覆蓋。
	oauthStateKeyPrefix = "FORUM:oauth:state:"

	// oauthVerifierBytes 是 PKCE verifier 的熵（32 位元組）。RFC 7636 要求 43–128 個
	// 字元且只允許未保留的字符，base64url（去 padding）後 32 位元組正好是 43 個字元。
	oauthVerifierBytes = 32
)

// oauthState 是 nonce 對應的那一筆資料。刻意只兩個欄位：nonce 本身不攜帶語意，
// 所有需要跨請求記住的東西都在這裡。
type oauthState struct {
	// ReturnPath 是「登入完成後回到哪一頁」，寫入前已通過 auth.IsSafeReturnPath。
	ReturnPath string `json:"return_path"`
	// CodeVerifier 是 PKCE 的原文，只送給 Google 的 token 端點，絕不出現在網址或 cookie。
	CodeVerifier string `json:"code_verifier"`
}

// oauthStateKey 把 nonce 組成本功能專屬的 Redis key。
func oauthStateKey(nonce string) string {
	return oauthStateKeyPrefix + nonce
}

// newOAuthNonce 產生 24 位元組（192 bits）隨機 nonce，以 hex 回傳。
// 與 session token 同樣的理由：這是流程的憑證，必須不可預測（math/rand 的序列可由
// 少量觀測值還原，那會讓「猜一個還沒用過的 nonce」變成可行）。
func newOAuthNonce() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// newPKCEVerifier 產生 PKCE 的 verifier（base64url、去 padding）。
// 用 crypto/rand 而非 oauth2.GenerateVerifier：那個函式的實作同样是 crypto/rand，
// 但它把熵來源寫在套件內部，這裡用自己的版本可以讓「產生憑證的方式」在這一檔裡
// 一處可見，不需要為了取一個函式而依賴整個 oauth2 套件。
func newPKCEVerifier() (string, error) {
	buf := make([]byte, oauthVerifierBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// handleGoogleLogin 是 GET /auth/google 的 handler。
//
// 流程：產生 nonce 與 verifier → 存 Redis（帶 TTL）→ 寫綁定 cookie → 用 nonce 作為
// state、帶 S256 challenge 導向 Google。
//
// 失敗情境全部回 5xx/4xx 而不導向 Google：憑證或存放不可用時，導過去只會讓使用者
// 卡在 Google 的授權頁（走完流程後回呼必然驗證失敗），那比直接報錯更糟。
//
// 副作用：寫入 Redis 一筆帶 TTL 的 key、追加一個 Set-Cookie 標頭、寫出 307 回應。
// 不寫 MySQL。
func (s *Server) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// OauthConfig 為 nil 代表啟動時忘了呼叫 auth.Init。這是設定錯誤，重試沒有意義，
	// 因此 500 而非 503。
	if auth.OauthConfig == nil {
		logger.ErrorContext(r.Context(), "[AUTH] OAuth config not initialized")
		internalError(w, "OAuth config not initialized")
		return
	}
	// nonce 必須有地方可放。沒有 Redis（或 Redis 這一步失敗）時，回呼就無法驗證
	// state，放行等於把 login CSRF 的防線拆掉 —— fail closed。
	if s.mediaRedis == nil {
		logger.ErrorContext(r.Context(), "[AUTH] 沒有可用的 Redis，無法產生登入用的 state")
		internalError(w, "unable to start login")
		return
	}

	// 回跳路徑：使用者從 /forum/login?return=... 進來。先過 IsSafeReturnPath 才存
	// 進 Redis —— 這一層是「開放轉向」的防線，與 nonce 無關，兩件事各自擋不同的事。
	returnPath := r.URL.Query().Get("return")
	if !auth.IsSafeReturnPath(returnPath) {
		returnPath = "/"
	}

	nonce, err := newOAuthNonce()
	if err != nil {
		logger.ErrorfContext(r.Context(), "[AUTH] 產生 state nonce 失敗: %v", err)
		internalError(w, "unable to start login")
		return
	}
	verifier, err := newPKCEVerifier()
	if err != nil {
		logger.ErrorfContext(r.Context(), "[AUTH] 產生 PKCE verifier 失敗: %v", err)
		internalError(w, "unable to start login")
		return
	}

	payload, err := json.Marshal(oauthState{ReturnPath: returnPath, CodeVerifier: verifier})
	if err != nil {
		// 兩個字串欄位不可能讓 Marshal 失敗；走到這裡代表程式被改壞了，不是使用者問題。
		logger.ErrorfContext(r.Context(), "[AUTH] state 序列化失敗: %v", err)
		internalError(w, "unable to start login")
		return
	}

	// context.Background()：與 session 的寫入同樣的理由 —— 使用者在寫入途中關掉分頁
	// 時，留下一筆「沒有 TTL 的常駐 nonce」比半套流程更糟，因此不讓用戶端取消這一步。
	// Set 帶 TTL，所以沒有清理迴圈；失敗時不導向 Google（下面直接 500）。
	if err := s.mediaRedis.Set(context.Background(), oauthStateKey(nonce), payload, oauthStateTTL).Err(); err != nil {
		logger.ErrorfContext(r.Context(), "[AUTH] 寫入 state 失敗: %v", err)
		internalError(w, "unable to start login")
		return
	}

	// 綁定 cookie：HttpOnly（JS 讀不到，XSS 拿不走它）、SameSite=Lax（回呼是頂層
	// GET 導覽，因此會帶著它；跨站的 POST 帶不到）、Path=/auth（只有 /auth/google 與
	// /auth/callback 需要它，不需要在整個站上來回送）。
	// Secure 沿用 cookieSecure 的設定：正式環境走 HTTPS 時為 true。
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    nonce,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(oauthStateTTL),
	})

	loginURL := auth.AuthorizeURL(nonce, verifier)
	if loginURL == "" {
		// AuthorizeURL 只可能在 OauthConfig 為 nil 時回空字串，上面已經擋掉了；
		// 留這一道是為了「别拿空字串去導向」這件事不會悄悄發生。
		logger.ErrorContext(r.Context(), "[AUTH] 授權網址產生失敗")
		internalError(w, "unable to start login")
		return
	}
	// DEBUG 級別：這一行含 client_id 與 nonce，供排查「登入後跳錯頁」；常駐環境
	// 維持 INFO 以上時不會產生噪音。verifier 不在這行裡（AuthorizeURL 不會把它放進網址）。
	logger.DebugfContext(r.Context(), "[AUTH] 導向 Google 授權頁: url=%s", loginURL)

	// 307 Temporary Redirect：語意上要求保留原本的請求方法與本文，比 302 更精確地
	// 表達「這是登入流程裡的一段轉址，不是資源的永久位置」。此端點只供瀏覽器以 GET
	// 進入，兩者實際行為差異不大，選 307 是為了語意正確。
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

// takeOAuthState 取回並刪除 nonce 對應的那一筆資料（GETDEL：讀與刪在同一個指令裡）。
//
// 為什麼必須「讀完即刪」：nonce 是一次性的。若它可被重複使用，一個被竊取的 state
// 就能在之後的某次回呼裡再被接受一次。GETDEL 也避免了「先 GET 再 DEL」中間被插進
// 另一個請求的競態。
//
// ok=false 代表 nonce 不存在（過期、已被用過、或根本是攻擊者編出來的）；Redis 錯誤
// 同樣收斂成 false —— 呼叫端在授權路徑上，「查不到」一律當「不可信」處理。
func (s *Server) takeOAuthState(ctx context.Context, nonce string) (oauthState, bool) {
	if s.mediaRedis == nil {
		return oauthState{}, false
	}
	// 這裡用 ctx（而非 Background）：這是使用者正在等待的回呼，請求被取消時沒有
	// 必要繼續讀 Redis。
	value, err := s.mediaRedis.GetDel(ctx, oauthStateKey(nonce)).Result()
	if err != nil {
		// redis.Nil（key 不存在）與其他錯誤都當成「不可信」，但只有真的錯誤留一筆日誌，
		// 否則攻擊者大量亂試 state 會把日誌寫滿。
		if err != redis.Nil {
			logger.ErrorfContext(ctx, "[AUTH] 讀取 state 失敗: %v", err)
		}
		return oauthState{}, false
	}
	var state oauthState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		logger.ErrorfContext(ctx, "[AUTH] state 內容無法解析: %v", err)
		return oauthState{}, false
	}
	return state, true
}

// clearOAuthStateCookie 請瀏覽器丟掉綁定 cookie。
// 名稱、Path、HttpOnly、Secure 都要與寫入時完全一致 —— 瀏覽器是按 name+domain+path
// 決定要刪哪一條，對不上時舊的 cookie 會留在瀏覽器裡直到自然到期。
func (s *Server) clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// matchesOAuthState 比對回呼的 state 與綁定 cookie。
// 用 subtle.ConstantTimeCompare：兩者都是憑證，比對長短差異若可被觀察到，
// 攻擊者就能靠回應時間逐段推出一個還沒用過的 nonce。
// 兩者皆為空時不算相符 —— 沒有 cookie 的回呼（舊版流程、或被人直接貼網址）
// 在這裡一律被擋掉。
func matchesOAuthState(state, cookieValue string) bool {
	if state == "" || cookieValue == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(state), []byte(cookieValue)) == 1
}
