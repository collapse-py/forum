package httpapi

/*
本檔案實作以 Origin / Referer 白名單為基礎的跨站請求偽造（CSRF）防護。

威脅模型
  - session 存在 SameSite=Lax 的 cookie 裡，瀏覽器會在跨站請求中自動附帶
    這張 cookie。攻擊者只要讓已登入 victim 造訪自己控制的頁面，就能在不知
    情的情況下對本站發出帶有合法憑證的狀態變更請求。
  - 假設：攻擊者能控制一個外部網站，並誘導 victim 的瀏覽器發出請求，卻
    讀不回本站的回應內容（跨源讀取會被同源政策阻擋）。非瀏覽器的直接呼叫
    （curl、腳本）不在此模型內 —— 那類呼叫本來就不依賴 cookie 憑證。

防線分層
  1. session cookie 設定 SameSite=Lax（見 session 套件），擋掉大部分跨站
     狀態變更請求。
  2. 本檔案比對 Origin / Referer 是否屬於設定檔的 TRUSTED_ORIGINS，屬於
     縱深防禦的第二層，也補上 SameSite 支援不完整或 cookie 屬性被放寬時的缺口。

第 2 層是**每個寫入 handler 自行呼叫** isTrustedOrigin，而不是一個中介層。

  這不是疏漏，是刻意的：多數後臺路由是「同一條路徑、同時服務 GET 與寫入方法」
  （/api/admin/users 是 GET 列表、POST 停權、PUT 改資料共用一個樣式），而
  isTrustedOrigin 只該擋**狀態變更**的方法。若把它做成中介層，就只有兩條路可走
  —— 要嘛把 GET 也擋掉（後臺整頁開不起來），要嘛在中介層裡內部做 method 判斷
  （那其實就是把 handler 的分支搬到另一個檔案，換不到任何東西）。

  依賴這個決定的是一條不變條件：**每一條會改變資料的路徑都必須呼叫它**。
  稽核端點也要寫入（稽核紀錄本身是資料），因此同樣要求來源檢查。目前全站共
  30 處 `!s.isTrustedOrigin(r)` 守衛（28 處獨立成行的 `if !s.isTrustedOrigin(r) {`，
  另有 2 處與 method 檢查合併成同一個條件），涵蓋公告、置頂、IP 封鎖、session
  撤銷、批次操作、關注、圖片 token 釋放與後臺 CRUD。
  新增寫入端點時漏掉這一行的症狀是「CSRF 防護失效但沒有任何錯誤」—— 因此
  這個清單靠程式碼審查維持，而不是靠型別。

能擋住什麼
  - 跨站表單送出、fetch/XHR、img 與 iframe 觸發的狀態變更請求：瀏覽器會
    附上攻擊者站點的 Origin（或其網頁的 Referer），比對失敗即以 403 拒絕。
  - 被注入惡意 Referer 值的請求（Referer 內容可由攻擊者頁面的 URL 決定）。

擋不住什麼
  - 與本站同源（same-origin）的 XSS：XSS 發出的請求其 Origin 就是本站，
    會直接通過。這類攻擊必須靠 CSP、輸出跳脫與 HttpOnly cookie 阻擋。
  - 同一個 TrustedOrigins 內部的子網域被入侵後發出的請求。
  - 不帶 cookie 的請求；本機的 curl 與後端對後端的呼叫需要能直接使用，
    因此不能要求每個請求都帶 Origin。

實務限制
  - 比對採整串完全相符（僅忽略大小寫），不支援萬用字元或前綴比對；
    設定值必須寫成 "https://example.com" 這種 origin 形式，不含結尾斜線
    與路徑。預設埠（https 對應 443、http 對應 80）不會被自動正規化，
    寫 "https://example.com:443" 會被視為不同來源。
  - 非瀏覽器客戶端可輕易偽造這兩個表頭；本防護的有效性完全建立在
    「攻擊者是網頁內容而非可自由送表頭的程式」這個假設上。
*/

import (
	"net/http"
	"net/url"
	"strings"
)

// requireTrustedOrigin 是 http.HandlerFunc 風格的中介層：來源不在白名單時
// 直接回 403 且不呼叫 next，否則原樣放行。
// 放在最外層（而非深層業務函式）的好處是「不通過就不碰任何業務邏輯」，
// 包含不查資料庫、不建立 session。
func (s *Server) requireTrustedOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isTrustedOrigin(r) {
			// 錯誤訊息刻意不指出「哪裡不對」也不回傳白名單內容，避免讓攻擊者
			// 透過回應逐步試出可信來源。403 而非 400：請求本身格式正確，
			// 是它的來源不被接受。
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		next(w, r)
	}
}

// isTrustedOrigin 判斷請求的來源是否可信，判斷順序為 Origin → Referer →
// 兩者皆無則放行。
// 這是純比對函式，不做 I/O、不寫回應。
func (s *Server) isTrustedOrigin(r *http.Request) bool {
	// 空白用字元可能被中介設備補上，先 Trim 再判斷是否為空字串。
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	// 為什麼 Origin 優先於 Referer：
	//   - 瀏覽器在跨源請求上一律會送 Origin，且其值由瀏覽器決定，網頁內容
	//     無法改寫；Referer 則可被 Referrer-Policy 完全抑制。
	//   - Referer 含有完整路徑與查詢字串，直接拿來比對容易因為尾斜線、
	//     查詢參數而誤判。
	// 因此只有在 Origin 缺席時才退而使用 Referer。
	if origin == "" {
		referer := strings.TrimSpace(r.Header.Get("Referer"))
		if referer == "" {
			// 兩者都沒有就放行。理由：像 curl、伺服器對伺服器的呼叫、瀏覽器
			// 擴充套件等合法客戶端可能一個都不送，擋下它們會讓功能壞掉；
			// 而且這些呼叫並不是「瀏覽器自動附帶 cookie」的威脅模型。
			// 殘餘風險：刻意移除兩個表頭的隱私設定或代理會讓跨站請求也通過，
			// 這時本防護失效，必須仰賴 SameSite cookie 與後端的身分檢查。
			return true
		}
		// 解析失敗一律拒絕（fail closed）：合法瀏覽器送出的 Referer 不可能
		// 解析失敗，解析不了代表這是偽造或被竄改的值，不該猜測它想表達什麼。
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		// 取出 scheme://host 這一節，與 Origin 表頭的格式對齊；路徑、查詢
		// 字串與 fragment 都丟棄（Referer 本來就不含 fragment）。
		origin = u.Scheme + "://" + u.Host
	}

	// 逐一比對設定檔的 TRUSTED_ORIGINS。未設定時 config.applyDefaults 會填入
	// PUBLIC_BASE_URL，所以這裡通常至少有一個可比的項目，來源存在但不在名單
	// 內一律視為不可信。
	for _, allowed := range s.cfg.TrustedOrigins {
		// EqualFold：scheme 與 host 依 RFC 3986 是不分大小寫的，瀏覽器多半
		// 會轉小寫，但經過代理或手動輸入的設定值未必。以不分大小寫比對可
		// 避免 "https://Example.com" 這類等價寫法被誤擋。
		// 限制：這裡不做預設埠與尾斜線的正規化，兩者仍須寫法一致。
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}
