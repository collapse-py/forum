/*
靜態檔案服務的守門層。

為什麼需要這個檔案：後端與前端原始碼位於同一個 repository
（backend/ 與 frontend/），而 "/" 是 catch-all 路由，任何未被認領的 URL
都會落到本檔的 safeStaticFileServer。若直接用 http.FileServer 暴露
frontend 目錄，攻擊者可以用 ../config.conf 之類的路徑穿越把
backend/config/config.conf、Google client secret、DB DSN 讀出去。

因此本檔採「預設拒絕」的心態：以 blockedExactPaths / blockedPrefixes /
blockedSuffixes 三層規則擋掉敏感路徑，再用 allowedExactPaths 明確放行
少數必須公開的檔案。與其列舉什麼可以給，不如列舉什麼絕對不能給。

三層規則的分工：
  - blockedExactPaths：已知敏感、但不在此目錄的檔名（設定檔、log）。
    擋掉它們只是多一道保險，實際上通常已被前綴或副檔名規則涵蓋。
  - blockedPrefixes：整個目錄樹不可公開，例如 /forum/ 底下的 SPA 原始碼
    與 /範例/ 這類範例資源。
  - blockedSuffixes：副檔名封鎖。最有效的一層，因為設定檔與原始碼的
    副檔名（.go/.conf/.sql/.mod/.sum/.ts）不會出現在瀏覽器需要的資源上，
    一刀就能涵蓋所有「萬一被複製到前端目錄」的敏感檔案。前端全面改用
    TypeScript 之後 .ts 尤其關鍵：frontendRoot 在沒有 dist 時會回退到
    frontend/web，而那個目錄底下除了 HTML/CSS 之外全是原始碼。
*/

package httpapi

import (
	"net/http"
	"path"
	"strings"
)

// blockedExactPaths 為逐筆比對的禁止清單，比對的是清理過的絕對路徑。
// 這些檔案即使被放進前端目錄也不外洩。
var blockedExactPaths = map[string]struct{}{
	"/config.conf": {},
	"/go.mod":      {},
	"/go.sum":      {},
	"/main.go":     {},
	"/main.py":     {},
	"/server.log":  {},
}

// allowedExactPaths 為明確放行的例外。放在比對順序的最前面，因為它必須優先於
// blockedSuffixes 與 blockedExactPaths：/service-worker.js 與 /forum-manifest.json
// 都需要被公開，但不能被 ".json" 這條副檔名規則擋掉。
// 實務上這兩條路徑在 mux 都另有專屬路由，正常情況下不會走到這份清單；
// 保留它們是讓 isBlockedStaticPath 單獨使用時（例如日後被其他 handler 引用）
// 仍維持正確的允許集合。
var allowedExactPaths = map[string]struct{}{
	"/forum-manifest.json": {},
	"/service-worker.js":   {},
}

// blockedPrefixes 為目錄級封鎖，任何以這些前綴開頭的路徑一律 404。
// /forum/ 底下是論壇各頁面的入口，已由 server.go 的 /forum/ 子樹路由與
// handleForumPage 逐一決定要送哪個 HTML，不該再由靜態檔案伺服器兜底；
// 這條規則讓 isBlockedStaticPath 單獨使用時也維持同一個界線。
// /src/ 是前端 React 原始碼所在的子目錄：它整棵都是 .ts/.tsx，副檔名規則
// 擋得到內容，但把整個目錄樹封掉是更直接的做法 —— 未建置時沒有任何一條
// /src/** 需要被公開。
var blockedPrefixes = []string{
	"/forum/",
	"/src/",
	"/範例/",
}

// blockedSuffixes 為副檔名封鎖清單。保留 .json 在其中是為了擋下前端 web 目錄
// 根層的 package.json 與 package-lock.json（建置設定檔，不該公開）。
// .ts / .tsx / .jsx 同理：前端已全面改用 TypeScript + React，未建置時
// frontend/web 底下全是原始碼，而瀏覽器需要的永遠是建置後的 .js —— 沒有任何
// 一條路徑需要公開原始碼，卻能藉它取得 vite.config.ts、全部頁面邏輯，以及
// 貼文建立、留言、標籤指派這些流程的完整實作。
// .tsx 與 .jsx 尤其不能漏：React 版的前端原始碼全部是 .tsx，只擋 .ts 會讓
// 整棵 src/ 變成公開資源（而 /src/ 只是其中一道防線，見 blockedPrefixes）。
// 唯一必須公開的 .json 是 forum-manifest.json，但它有自己的 mux 路由，且
// 已列入 allowedExactPaths；建置後的版本則是 /assets/forum-manifest-<hash>.json，
// 走的是 /assets/ 那條路由，不會落到本檔。
var blockedSuffixes = []string{
	".go",
	".conf",
	".sql",
	".mod",
	".sum",
	".json",
	".ts",
	".tsx",
	".jsx",
}

// isBlockedStaticPath 判斷某個 URL 路徑是否禁止以靜態檔案形式提供。
// 回傳 true 代表應該回 404 而非 403：403 會回應「檔案存在但拒絕提供」，
// 等於向探測者確認路徑有效；404 對外不透露任何額外資訊。
//
// 判斷前先過 path.Clean：這是路徑穿越（Directory Traversal）防護的第一道關卡，
// ../、./、多重斜線與空路徑都會被正規化，之後的比對才不會被 "/./../config.conf"
// 這類寫法繞過。http.FileServer 本身也會做一次清理，兩層並存屬刻意防禦。
func isBlockedStaticPath(urlPath string) bool {
	clean := path.Clean(urlPath)
	// Clean("") 與 Clean(".") 都會得到 "."，統一成 "/" 才能和上方的
	// 絕對路徑清單正確比對。
	if clean == "." {
		clean = "/"
	}
	// 放行清單優先：/service-worker.js 與 /forum-manifest.json 必須公開。
	if _, ok := allowedExactPaths[clean]; ok {
		return false
	}
	if _, ok := blockedExactPaths[clean]; ok {
		return true
	}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(clean, prefix) {
			return true
		}
	}
	// 副檔名比對轉小寫：Windows 與預設設定的 macOS 檔案系統不分大小寫，
	// "config.GO"、"x.CoNf" 在那些環境中仍可被開啟，只比小寫等於一併擋掉。
	lower := strings.ToLower(clean)
	for _, suffix := range blockedSuffixes {
		// 排除根路徑：HasSuffix 對 "/" 為 false，但保險起見仍明寫
		// clean != "/"，避免日後新增 "/" 結尾的字串時意外擋掉首頁。
		if strings.HasSuffix(lower, suffix) && clean != "/" {
			return true
		}
	}
	return false
}

// safeStaticFileServer 包一層安全性與快取策略的 FileServer。
//
// 存在的意義：http.FileServer 只做「把 URL 對應到磁碟檔案」一件事，既不擋
// 路徑穿越也不管快取。這個函式在它前面加了一道黑名單檢查，在它前面補上了
// 依檔案類型分級的 Cache-Control，讓根目錄的靜態檔案不會被無限期快取、
// 也不會在檔案更新後拿到舊內容。
func safeStaticFileServer(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isBlockedStaticPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		// 只在 GET 上設定快取標頭：快取的語意只對「取檔案」有意義，
		// 其他方法不該繼承同一組快取指示。
		if r.Method == http.MethodGet {
			cleanPath := path.Clean(r.URL.Path)
			if strings.HasSuffix(strings.ToLower(cleanPath), ".html") || cleanPath == "/" {
				// HTML 不快取：它是入口檔，內容更新後使用者必須立即拿到新版，
				// 否則舊的 HTML 會去引用已被清掉的 hashed 資產而整頁壞掉。
				w.Header().Set("Cache-Control", "no-cache, must-revalidate")
			} else if strings.HasPrefix(cleanPath, "/assets/") || r.URL.RawQuery != "" {
				// /assets/ 底下的建置資產檔名帶內容 hash，內容變了檔名就變，
				// 可以宣告 immutable。帶 query string 代表呼叫端在做 cache-busting
				// （例如 ?v=時間戳），同一路徑的內容會隨參數而變，因此也視為
				// 內容不會原地改動。
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else if isCacheableStaticPath(cleanPath) {
				// 其他靜態資源給一天快取：這些檔名沒有 hash，必須留一個
				// 重新驗證的窗口給開發期更新。
				w.Header().Set("Cache-Control", "public, max-age=86400")
			}
		}
		fs.ServeHTTP(w, r)
	})
}

// isCacheableStaticPath 判斷某路徑是否屬於「可以短暫快取」的資源型別。
// 這份清單刻意只列前端真正會載入的樣式、腳本與字型／圖片格式：
// 白名單而非黑名單，萬一將來有新副檔名落到這裡，預設是不快取（安全）。
func isCacheableStaticPath(urlPath string) bool {
	for _, suffix := range []string{".css", ".js", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".woff", ".woff2", ".ttf"} {
		// 一律比小寫，理由同 isBlockedStaticPath 的副檔名處理。
		if strings.HasSuffix(strings.ToLower(urlPath), suffix) {
			return true
		}
	}
	return false
}
