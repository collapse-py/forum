package httpapi

/*
本檔案實作全站的安全標頭中介層，補上 Content-Security-Policy 與
X-Frame-Options / X-Content-Type-Options / Referrer-Policy 等防護標頭。

為什麼需要這個檔案
  - 論壇是典型的 XSS 高風險面：貼文內文、留言、自我簡介、使用者暱稱都是
    其他使用者會看到的字串。前端已改用 React，渲染時本來就會跳脫文字
    （因此不再有手寫的 escapeHTML），但那份保證只覆蓋「用 JSX 產生」的節點：
    dangerouslySetInnerHTML 一旦被誤用就會完全繞過它。CSP 提供的
    是「就算跳脫寫錯了，惡意腳本仍然跑不起來」的第二道縱深防禦。
  - csrf.go 的檔頭註解已明確寫出「與本站同源的 XSS 必須靠 CSP、輸出跳脫與
    HttpOnly cookie 阻擋」。在此之前 CSP 並不存在，那句註解是不成立的。
  - 靜態檔案服務（static.go）只處理路徑與快取，完全不管瀏覽器該不該信任
    這份回應的內容型態。

套用在誰身上
  整棵 mux 之外（見 server.go 的 Handler），因此 HTML、JSON、靜態資產與
  Service Worker 全部都會帶上這些標頭。標頭本身對非文件型回應無害，
  而「全站一致」比「只在某些路徑才正確」更不容易在日後改動時被破壞。

CSP 的組成與取捨
  這份政策刻意不放寬任何一個指令，每一項都是照著本專案實際會載入的資源列出：

	default-src 'self'          預設拒絕，只允許同源。
	base-uri 'self'            阻擋 <base> 標籤劫持（見下方說明）。
	object-src 'none'          阻擋 <object>/<embed> 載入外掛內容。本專案沒有用到，
	                           但預設允許等於留了一個已知會被用來做混淆攻擊的面。
	frame-ancestors 'none'     阻擋點擊劫持（clickjacking）。
	form-action 'self'         限制表單只能送回本站。目前前端沒有任何 <form>
	                           （全部用 fetch 送），因此這是純防護。
	script-src 'self' https://static.cloudflareinsights.com
	                           前端本身只用外部模組；唯一例外是 Cloudflare 的
	                           Browser Insights —— 它在邊緣（Cloudflare edge）改寫
	                           HTML，插入 <script src="https://static.cloudflareinsights.com/
	                           beacon.min.js">，這段程式碼不在本 repo 裡，但確實
	                           會在每個頁面執行，因此必須放行，否則主控台固定出現
	                           一行 Refused to load、且該頁的 RUM 完全收不到資料。
	style-src 'self' https://fonts.googleapis.com
	                           前端以 <link> 載入 Google Fonts 的 CSS；頁面專屬的
	                           樣式寫在各 HTML 的 <style> 區塊裡，靠啟動時從磁碟
	                           HTML 算出來的 'sha256-…' 雜湊逐一放行（見
	                           inlineStyleHashes）。刻意不用 'unsafe-inline'：
	                           那一項一旦存在，攻擊者只要能注入一個 <style> 就能
	                           覆蓋整頁外觀（把登入表單藏起來、蓋掉錯誤訊息），
	                           而雜湊只放行「我們自己寫在檔案裡的」那些區塊。
	                           style="" 屬性同樣禁止，JS 一律只能用 class。
	font-src 'self' https://fonts.gstatic.com
	                           Google Fonts 的 CSS 會再指向 fonts.gstatic.com 取字型
	                           檔；自製的 NotoSansTC 字型走 /asset/（同源）。
	img-src 'self' data: <FILES_SERVER_PUBLIC_URL>
	                           貼文附圖存在於獨立的檔案伺服器，因此必須依設定把
	                           它的對外網域加進來，否則所有圖片都會被擋下。
	                           data: 是為了讓前端能用 data URI 顯示預覽。
	connect-src 'self' https://cloudflareinsights.com
	                           fetch / sendBeacon 只能連回本站。前端沒有任何
	                           對外 API 呼叫（OAuth 是整頁導向，不是 XHR）。
	                           唯一的例外同樣是 Browser Insights：beacon.min.js
	                           載入後會把量測資料 POST 到
	                           https://cloudflareinsights.com/cdn-cgi/rum，
	                           connect-src 沒放行的話指令碼雖然會跑，但每次回報
	                           都會被擋下，等於拿到一份沒有資料的 RUM。
	manifest-src 'self'        PWA manifest 來自本站。

  刻意不使用的指令：
	  'unsafe-inline' / 'unsafe-eval'  一旦寫了等於放棄 XSS 防護，因此寧可先改
	                                  前端也不要寫進來。頁面 <style> 用雜湊解決，
	                                  不需要為了它開這條。
	  'unsafe-hashes'                  同上，只會讓內聯腳本重新變成可用。
	  report-uri / report-to           需要蒐集端點才有意义，本專案沒有；
	                                  且會把使用者造訪的網址送給第三方，屬隱私取捨。

  特例：/service-worker.js 不送 CSP。Service Worker 在 worker 環境中求值，
  而它會主動 fetch 多個網址做預快取；對它套一份為文件設計的 default-src
  只會造成難以診斷的失敗，且不會提升任何安全性（SW 本身不是攻擊面 ——
  真正的風險是它能攔截哪些請求，那由 /service-worker.js 本身的內容決定）。
*/

import (
	"crypto/sha256"
	"encoding/base64"
	"forum/forum/config"
	"forum/forum/logger"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

// securityHeader 是一組「標頭名稱 → 值」。以結構而非 http.Header 表示，是因為
// 這份清單在建構後就完全唯讀，且名稱可能重複（Content-Security-Policy 只需要
// 一個，但多個 Set-Cookie 這類情況由各自的 handler 自行處理）。
type securityHeader struct {
	name  string
	value string
}

// securityHeaders 是預先算好的標頭清單，於啟動時建構一次並在所有請求間共用。
//
// 為什麼不每次請求都重算：Content-Security-Policy 的字串組裝（尤其解析
// FILES_SERVER_PUBLIC_URL）會配置記憶體，而它對同一個行程永遠是相同的值。
// 在請求路徑上重算是純粹的浪費，且會讓這段程式碼出現在每個請求的 profile 裡。
type securityHeaders struct {
	headers []securityHeader
}

// frontendShellFiles 是「頁面專屬樣式要放行」的檔案清單。
//
// 名稱必須與 vite.config.ts 的 rollup input 一致，也與 server.go 的
// handleForumPage / handleForumLoginPage 會送出的檔名一致。三處若不同步，症狀
// 是某一頁的 <style> 被 CSP 擋下（整頁沒有版面），而不是任何錯誤訊息 —— 所以
// 這份清單在 server.go 與 vite.config.ts 的註解裡都留了指回來的路。
//
// 這裡不列出 style.css：它是 <link> 載入的外部檔案，style-src 'self' 已經涵蓋。
var frontendShellFiles = []string{
	"forum.html",
	"forum-login.html",
	"forum-new.html",
	"forum-profile.html",
	"forum-others-profile.html",
	"forum-following.html",
	"admin.html",
	"forum-admin.html",
	"forum-report.html",
	"forum-monitor.html",
	"audit-log.html",
	"forum-stats.html",
	"export.html",
	"sessions.html",
	"blocks.html",
	"announcements.html",
}

// inlineStyleHashes 掃出各頁面 HTML 裡的 <style> 區塊內容，算出 CSP 需要的
// 'sha256-…' 來源運算式。
//
// 為什麼需要它：各頁的樣式寫在 HTML 的 <style> 區塊裡（調版面時不必在 CSS 與
// HTML 之間跳），而 style-src 沒有 'unsafe-inline'，因此這些區塊必須被逐一授權。
// 雜湊授權比 'unsafe-inline' 嚴格得多：只有「內容剛好等於磁碟上那份」的區塊會
// 被套用，任何被注入的 <style> 都匹配不到任何一項。
//
// 為什麼在啟動時算而不是把雜湊寫死在原始碼裡：寫死等於每改一次 CSS 就要同時
// 改兩個檔案，而漏改的那一次症狀是「樣式整片消失、主控台只有一行 Refused to
// load」。從磁碟上的 HTML 直接算，兩邊不可能不同步。
//
// 但「不可能不同步」有前提：伺服器啟動之後磁碟上的檔案不能再變。改完某一頁的
// <style> 之後必須重啟本服務，雜湊才會重新計算。忘記重啟的症狀非常不直觀：
//   - 只有「被改的那一頁」失去樣式，其餘八頁完全正常（因為它們的雜湊仍然對得上）
//   - 失去的是整個區塊，不只是你剛加的那條規則，所以頁面會退化成一堆看起來
//     毫不相關的問題（清單長出項目符號、flex 消失、元素擠在一起）
//   - style.css 照常作用（走 style-src 'self'），因此畫面看起來「有樣式但怪怪的」
//
// 診斷方式（不用開 devtools）：把每個 dist/*.html 的第一個 <style> 區塊算一次
// base64 SHA-256，看有沒有出現在線上 CSP 標頭的 style-src 清單裡。哪一頁不在，
// 就是那一頁需要重啟。
//
// 送出的內容並非逐位元組等於磁碟上那份：handleForumPage 走 serveHTMLFile，
// 會把站名佔位符換成設定值（見 site.go）。這不影響雜湊正確性，因為那些佔位符
// 只允許出現在 <style> 區塊之外（site.go 檔頭的硬性約束）；一旦有人把佔位符
// 寫進樣式區塊，雜湊就會對不上，症狀仍是整頁沒有版面。
//
// 讀不到檔案時只警告不中止：前端缺檔本來就讓那些頁面 404（frontendRoot 的
// 既有行為），CSP 缺雜湊只是讓樣式也被擋下，兩者是同一個部署問題的兩種症狀。
func inlineStyleHashes(frontendDir string) []string {
	seen := make(map[string]struct{})
	var missing []string

	for _, name := range frontendShellFiles {
		path := frontendAssetPath(frontendDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			missing = append(missing, name)
			continue
		}
		for _, css := range styleBlockContents(string(raw)) {
			sum := sha256.Sum256([]byte(css))
			seen["'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'"] = struct{}{}
		}
	}

	if len(missing) > 0 {
		logger.Warnf("[HTTP] inline <style> hashes missing for %s under %s; "+
			"those pages will render without their page styles (CSP blocks them)",
			strings.Join(missing, ", "), frontendDir)
	}
	if len(seen) == 0 {
		return nil
	}

	// 排序的理由是讓 CSP 字串在相同輸入下產生相同輸出：啟動日誌、測試與
	// 實際送出的標頭才不會因為 map 的迭代順序而每次都不一樣。
	hashes := make([]string, 0, len(seen))
	for hash := range seen {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	return hashes
}

// styleBlockContents 取出 HTML 中每個 <style> 區塊的內容（不含標籤本身）。
//
// 為什麼自己掃而不是用 encoding/xml 或 golang.org/x/net/html：這兩者都會把
// 文件重新序列化或糾錯，而 CSP 雜湊必須對「瀏覽器實際收到的那幾個位元組」計算
// —— 只要掃描過程動到任何一個空白，雜湊就對不上，症狀是樣式無聲消失。因此這裡
// 只做「找出開標籤、跳過屬性、取出到結束標籤為止的原文」這一件事。
//
// 只認小寫的 <style：HTML 標籤在瀏覽端不區分大小寫，但這個 repo 的樣板全部是
// 小寫，而為了涵蓋 <STYLE> 就得在比對時對整份文件做_lower()，那會配置一份與
// 文件等大的副本。真的出現大寫寫法時，症狀是那一頁沒有樣式 —— 明顯且好修。
func styleBlockContents(html string) []string {
	const (
		openTag  = "<style"
		closeTag = "</style>"
	)

	var blocks []string
	for rest := html; ; {
		start := strings.Index(rest, openTag)
		if start < 0 {
			return blocks
		}
		rest = rest[start+len(openTag):]

		// 邊界檢查：<style 後面必須是屬性、空白或 '/'，否則這是 <stylesheet>
		// 這種以相同字面開頭的標籤／文字，把它當成樣式區塊會算出一個永遠不會
		// 對應到任何元素的雜湊，白白放行一個沒有意義的來源。
		if rest != "" && !isStyleTagBoundary(rest[0]) {
			continue
		}

		// 跳過屬性：<style> 與 <style nonce="…"> 都要能處理，因此不能假設
		// 開標籤結束在第一個 '>'。
		bodyStart := strings.Index(rest, ">")
		if bodyStart < 0 {
			// 沒有結束的 '>' 代表這不是 <style> 開標籤，繼續往後找而不是
			// 回傳空內容。
			continue
		}
		rest = rest[bodyStart+1:]

		end := strings.Index(rest, closeTag)
		if end < 0 {
			// 區塊沒有被關閉：送出這份 HTML 的頁面本來就是壞的，靜靜忽略
			// 剩下的內容比塞一個空雜湊進 CSP 好。
			return blocks
		}
		blocks = append(blocks, rest[:end])
		rest = rest[end+len(closeTag):]
	}
}

// isStyleTagBoundary 判斷 <style> 之後的字元是否仍屬於同一個標籤。
//
// 合法的是標籤結束（'>'）、屬性分隔（空白或 '/'）；字母、數字與底線代表那是
// 另一個標籤的名稱（<stylesheet>），必須排除。
func isStyleTagBoundary(c byte) bool {
	switch c {
	case '>', ' ', '\t', '\n', '\r', '\f', '/':
		return true
	}
	return false
}

// newSecurityHeaders 依設定檔建構安全標頭清單。
//
// styleHashes 是各頁 <style> 區塊的 'sha256-…' 來源（見 inlineStyleHashes），
// 呼叫端負責算出來傳入：它需要檔案系統，而 buildContentSecurityPolicy 刻意
// 保持成「純粹由設定檔與參數決定」以便單獨測試。
//
// 會寫一行日誌說明 CSP 最終長相：瀏覽器只會用一句「Refused to load」來回報
// 違規，不會告訴你缺的是哪一個來源。啟動時把政策寫進日誌，是日後排查
// 「為什麼圖片／字型／版面不見了」唯一可靠的線索。
func newSecurityHeaders(cfg config.Config, styleHashes []string) *securityHeaders {
	headers := []securityHeader{
		// 禁止瀏覽器嗅探內容型態。這條同時補掉 static.go 副檔名白名單的
		// 殘餘風險：就算某個副檔名被誤判而以 text/plain 送出，瀏覽器也不會
		// 把它當成 HTML 或 JS 執行。
		{"X-Content-Type-Options", "nosniff"},
		// frame-ancestors 的舊版後援。舊瀏覽器不支援 CSP 時，仍需要它擋住
		// 點擊劫持。DENY 比 SAMEORIGIN 更嚴：後台與論壇頁面都沒有被嵌入的
		// 需求，因此不給例外。
		{"X-Frame-Options", "DENY"},
		// 只在跨源時送出 origin、不帶路徑與查詢字串。論壇的網址帶著
		// ?user=<public_key> 這類參數，連到外部網站時不該順手把它送出去。
		// 對同源請求則保留完整 Referer（等同瀏覽器預設行為），以免影響
		// 站內資源的相對路徑解析。
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		// 關閉本站沒有用到的瀏覽器能力。設定相機／麥克風／定位為明確的
		// 拒絕（而非省略），因此就算日後不小心在某個頁面要求權限，使用者
		// 也不會被無提示地授權。
		{"Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()"},
		// 把本站文件與跨源文件切成不同的 browsing context group，攻擊者
		// 就無法用 window.opener 從自己開的分頁反向操作本站（tabnabbing）。
		// 安全性理由：Google OAuth 走的是 <a href> 的整頁導向而非彈出視窗，
		// 全站沒有任何 window.open，因此不會有程式被這條切斷。
		{"Cross-Origin-Opener-Policy", "same-origin"},
	}

	// HSTS 只在確定對外是 HTTPS 時才送。
	//
	// 判斷依據是 COOKIE_SECURE 而非 r.TLS != nil：TLS 幾乎都由前置反向代理
	// 終結，後端只看到 http，此時 r.TLS 永遠是 nil，判斷會永遠為假。
	// COOKIE_SECURE 的既定語意就是「對外以 HTTPS 服務時應為 true」，因此
	// 它是這個部署環境下唯一可靠的訊號。
	//
	// includeSubDomains 有一個前提：本專案的所有子網域都必須有 TLS。若未來
	// 有子網域走 http（例如內部工具），HSTS 會讓它無法再被存取 —— 屆時
	// 應把 includeSubDomains 拿掉，而不是整條刪除。
	if cfg.CookieSecure {
		headers = append(headers, securityHeader{
			"Strict-Transport-Security", "max-age=31536000; includeSubDomains",
		})
	}

	headers = append(headers, securityHeader{
		"Content-Security-Policy", buildContentSecurityPolicy(cfg, styleHashes),
	})

	logger.Debugf("[HTTP] security headers: %d directives, hsts=%v, inline style hashes=%d",
		len(headers), cfg.CookieSecure, len(styleHashes))
	return &securityHeaders{headers: headers}
}

// buildContentSecurityPolicy 組出 CSP 字串。
//
// 拆成獨立函式（而非塞在 newSecurityHeaders 裡）是為了讓它能單獨測試：
// 這份字串是純粹的設定檔與參數衍生結果，不依賴任何請求或檔案系統。
//
// styleHashes 為各頁 <style> 區塊的雜湊（見 inlineStyleHashes）；傳 nil 或空
// 切片會得到一份不含任何內聯樣式授權的政策，那份政策仍然安全，只是那些頁面
// 會沒有版面 —— 這是啟動時讀不到前端檔案的症狀，而不是政策本身的漏洞。
func buildContentSecurityPolicy(cfg config.Config, styleHashes []string) string {
	// style-src 的來源順序刻意固定為「'self' → fonts.googleapis.com → 雜湊」：
	// 政策字串會被寫進啟動日誌，前兩項是人類看得懂的意圖，雜湊只是清單的尾巴。
	styleSources := append([]string{"'self'", "https://fonts.googleapis.com"}, styleHashes...)

	directives := []string{
		"default-src 'self'",
		"base-uri 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"form-action 'self'",
		// 兩個 cloudflareinsights.com 是全站唯一放行的第三方來源，而且兩者
		// 缺一不可：只有 script-src 時 beacon 會載入但每次回報都被 connect-src
		// 擋下（症狀是主控台乾淨、但 RPM 永遠是 0，比直接報錯更難察覺）。
		// 來源來自 Cloudflare 對 Browser Insights 的官方說明，站台是固定的
		// 兩組主機，因此直接寫死而不做成設定項：若 Cloudflare 未啟用該功能，
		// 邊緣不會注入 beacon，這兩個來源就是無人使用的空條目。
		"script-src 'self' https://static.cloudflareinsights.com",
		"style-src " + strings.Join(styleSources, " "),
		"font-src 'self' https://fonts.gstatic.com",
		"connect-src 'self' https://cloudflareinsights.com",
		"manifest-src 'self'",
	}

	// 貼文附圖來自獨立的檔案伺服器，必須把它的對外網域放進 img-src，
	// 否則 CSP 會擋下論壇上每一張圖片。
	//
	// 這裡只取 scheme 與 host（丟棄路徑與查詢字串），因為 CSP 的來源
	// 運算式本來就以來源為粒度；把整條 URL（含路徑）寫進去既不會更嚴格，
	// 也會在對外路徑改版時需要同步更新設定。
	//
	// 解析失敗或未設定時不加這一段：此時 img-src 只剩 'self' 與 data:，
	// 圖片會載不出來，但這是「明確不允許」而非「默默放行」，且啟動日誌與
	// 瀏覽器的 CSP 違規訊息都會指出缺口，屬於可見、可修的失敗。
	imgSources := []string{"'self'", "data:"}
	if imageSource := cspSourceFromURL(cfg.FilesServerPublicURL); imageSource != "" {
		imgSources = append(imgSources, imageSource)
	} else if cfg.FilesServerPublicURL != "" {
		logger.Warnf("[HTTP] FILES_SERVER_PUBLIC_URL %q is not a usable absolute URL; "+
			"post images will be blocked by Content-Security-Policy img-src",
			cfg.FilesServerPublicURL)
	}
	directives = append(directives, "img-src "+strings.Join(imgSources, " "))

	// 指令之間以 "; " 分隔。保留單行的理由不是美觀，而是實務考量：啟動日誌
	// 會把這整行寫出來，換行會讓它被切成多筆 log、難以用單一關鍵字撈出來；
	// 而 CSP 對多餘空白並不敏感，只是解析時會忽略。
	return strings.Join(directives, "; ")
}

// cspSourceFromURL 把一個絕對 URL 轉成 CSP 的來源運算式（scheme + host）。
//
// 轉換失敗時回傳空字串而非原字串：把 "https://example.com/path?x=1" 這種
// 帶路徑的值塞進 img-src 雖然語法合法，但意圖錯誤且會在換版時靜默失效。
// 只接受有 scheme 與 host 的形式；純路徑或相對值（例如 "/files"）無法構成
// 來源運算式，同樣回傳空字串。
func cspSourceFromURL(raw string) string {
	// 設定檔載入時已去掉尾斜線，但這裡再保險一次：url.Parse 對
	// "https://example.com/" 與 "https://example.com" 會給出不同的 Path，
	// 組出來的字串也就差一個斜線。
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	// 沒有 scheme（u.Scheme 為空）代表這是相對路徑而非絕對 URL。
	// 沒有 host 同理，例如 "mailto:someone@example.com" 對 img-src 沒有意義。
	if u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// apply 把標頭寫進回應。
//
// 必須在 handler 寫出任何內容之前呼叫：一旦 WriteHeader 送出，表頭就鎖定。
// 這是為什麼本檔案以中介層形式使用，而不是在各 handler 裡手動呼叫。
func (h *securityHeaders) apply(w http.ResponseWriter, r *http.Request) {
	// Service Worker 是唯一例外：它在自己的 worker 環境求值並會主動 fetch
	// 多個網址，對它套用為文件設計的 default-src 只會造成難以診斷的失敗。
	// 判斷用 r.URL.Path 而非比對完整 RequestURI，因此帶查詢字串的
	// /service-worker.js?v=20260922 仍會被辨識出來 —— service-worker.js 正是
	// 以帶版本參數的形式被註冊的，若漏掉這個分支，重新載入 SW 時就會被套上
	// 一份不該有的 CSP。
	// 這裡刻意只比對這一個路徑而非前綴，未來若出現 /service-worker.js.map
	// 之類的兄弟檔案，它仍會拿到（此情況下有害的）CSP 而非被一併跳過。
	isServiceWorker := r.URL.Path == "/service-worker.js"

	for _, header := range h.headers {
		if isServiceWorker && header.name == "Content-Security-Policy" {
			continue
		}
		// 用 Set 而非 Add：這裡要的是「這份回應該有這個值」，不是累加。
		// 若某個下游 handler 自己也設定了同名標頭（例如日後的靜態資源
		// 處理器），Set 會讓 ours 覆蓋它，語意明確。
		w.Header().Set(header.name, header.value)
	}
}

// withSecurityHeaders 是安全標頭中介層。
//
// 簽章刻意是 http.Handler 而非 http.HandlerFunc：它的套用位置在 mux 之外
// （見 server.go），作用對象是整棵路由樹，不只是某幾條路由。
//
// styleHashes 由呼叫端從磁碟上的前端 HTML 算出（見 inlineStyleHashes），
// 因為這是唯一需要檔案系統的步驟，而這裡不該自己去猜前端根目錄。
func (s *Server) withSecurityHeaders(next http.Handler, styleHashes []string) http.Handler {
	// 在建構中介層時就把標頭算好，而不是每個請求算一次。
	headers := newSecurityHeaders(s.cfg, styleHashes)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers.apply(w, r)
		next.ServeHTTP(w, r)
	})
}
