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
	"os"
	"path"
	"path/filepath"
	"strings"
)

// blockedExactPaths 為逐筆比對的禁止清單，比對的是清理過的絕對路徑。
// 這些檔案即使被放進前端目錄也不外洩。
//
// 後台的 HTML 殼檔列在這裡：那十個 /admin/* 頁面各自有專屬路由（server.go），
// 且已改成必須是管理員才取得得到。但它們同樣是 catch-all 看得到的「純檔案」，
// 而副檔名清單裡沒有 .html（HTML 是入口檔型別，不能整類擋掉），因此必須逐筆
// 指名。少了這層，GET /admin.html 會繞過路由上的管理員判斷，把後台的介面結構
// 送給任何匿名訪客。
var blockedExactPaths = map[string]struct{}{
	"/config.conf": {},
	"/go.mod":      {},
	"/go.sum":      {},
	"/main.go":     {},
	"/main.py":     {},
	"/server.log":  {},
	"/.env":        {},

	"/admin.html":         {},
	"/forum-admin.html":   {},
	"/forum-report.html":  {},
	"/forum-monitor.html": {},
	"/audit-log.html":     {},
	"/forum-stats.html":   {},
	"/export.html":        {},
	"/sessions.html":      {},
	"/blocks.html":        {},
	"/announcements.html": {},
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
	// /admin/ 底下的任何檔案都不該以純檔案形式送出：那十個後台頁面有專屬路由
	// 並已掛上管理員判斷，而這個目錄本身不該存在任何公開檔案。
	"/admin/",
	// 版控與依賴目錄：一旦它們被複製進前端根目錄（例如 dist 旁邊留了 checkout），
	// .git 底下能讀出完整的 repository 與 commit 歷史。
	"/.git/",
	"/node_modules/",
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
	// 備檔與暫存檔。這一組是針對「檔案落在靜態根內」的補強：.bak / .old / .tmp
	// 是人工改設定時最常見的副檔名（改完留一份舊檔），而它們恰恰帶著完整的憑證
	// —— .conf 擋住了 config.conf，但改設定留下的 config.conf.bak 原本是漏網的。
	// .env 同理：專案根目錄就有 .env，而靜態根是前端目錄，一旦有人把整棵
	// repository 複製進 dist，它就看得得到。
	".bak",
	".old",
	".tmp",
	".env",
	".log",
	// 其他原始碼與組態型別：這裡不存在任何需要公開的這種檔案。
	".sh",
	".ps1",
	".py",
	".ini",
	".yaml",
	".yml",
	".toml",
	".csv",
	".db",
	".sqlite",
}

// blockedAssetSuffixes 是 /assets/ 與 /asset/ 那兩條路由用的封鎖清單。
//
// 為什麼要另開一份而不是重用 blockedSuffixes：那兩條路由指向的是「建置產物」
// 與「一手素材」目錄，而 Vite 會把 PWA manifest 產成
// /assets/forum-manifest-<hash>.json —— 它必須公開。blockedSuffixes 裡的
// ".json" 是為了擋 frontend/web 根層的 package.json 與 package-lock.json，
// 直接套用到 /assets/ 會讓清單變成「擋掉一個必須公開的檔案」。
//
// 這份清單保留了原始碼與組態型別（.ts/.tsx/.go/.conf 等）與上方的備檔型別：
// 那兩目錄的檔案都是 .js/.css/圖片/manifest，沒有任何一條需要公開原始碼。
var blockedAssetSuffixes = []string{
	".go",
	".conf",
	".sql",
	".mod",
	".sum",
	".ts",
	".tsx",
	".jsx",
	".bak",
	".old",
	".tmp",
	".env",
	".log",
	".sh",
	".ps1",
	".py",
	".ini",
	".yaml",
	".yml",
	".toml",
	".db",
	".sqlite",
}

// isBlockedAssetPath 判斷 /assets/ 或 /asset/ 下的某個路徑是否禁止送出。
// 比對的是完整 URL 路徑（含 /assets/ 前綴），讓呼叫端可以直接傳 r.URL.Path。
// 回傳 true 時一樣回 404 而非 403，理由同 isBlockedStaticPath。
func isBlockedAssetPath(urlPath string) bool {
	clean := path.Clean(urlPath)
	if clean == "." {
		clean = "/"
	}
	lower := strings.ToLower(clean)
	for _, suffix := range blockedAssetSuffixes {
		if strings.HasSuffix(lower, suffix) && clean != "/" {
			return true
		}
	}
	return false
}

// staticPathIsDirectory 判斷某個靜態請求對應到的是目錄而非檔案。
//
// 存在的理由：http.FileServer 對「目錄且沒有 index.html」會回應一份 HTML 的
// 目錄清單，把檔名、大小與修改時間列出來。建置產物的檔名帶內容 hash，因此那份
// 清單等於對外公開「這個站建置了哪些模組」；一手素材目錄則會列出 PWA 圖示之外的
// 所有檔案。沒有任何使用者需要那份清單 —— 前端載入的是明確的檔名。
//
// index.html 的例外：FileServer 對目錄的預設行為是「先找 index.html，找到就送它，
// 找不到才列清單」。root 目錄（首頁）正是靠這個行為送 index.html，因此那個例外
// 必須保留，否則首頁會變成 404。
func staticPathIsDirectory(root, urlPath string) bool {
	clean := path.Clean(urlPath)
	if clean == "." {
		clean = "/"
	}
	fi, err := os.Stat(filepath.Join(root, strings.TrimPrefix(clean, "/")))
	if err != nil || !fi.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(root, strings.TrimPrefix(clean, "/"), "index.html"))
	return err != nil
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

// setPublicCacheControl 設定公開快取的 Cache-Control，並移除 session cookie。
//
// 為什麼要移除：session 續期中介層在 handler 之前就會寫入帶 session token 的
// Set-Cookie（見 server.go 的 Refresh 順序），而 public 允許共享快取儲存整份
// 回應；RFC 9111 §3.1 要求快取連同所有回應表頭一起儲存，Set-Cookie 也在內。
// 靜態資產不需要續期 cookie，session 的 server-side TTL 仍會由頁面與 API
// 請求延長。少了這一步，共享快取可能把登入者的 token 送給下一個訪客。
func setPublicCacheControl(w http.ResponseWriter, value string) {
	w.Header().Set("Cache-Control", value)
	w.Header().Del("Set-Cookie")
}

// hashedAssetHandler 服務 /assets/ 底下帶內容 hash 的建置資產（Vite 產物）。
//
// 檔名帶內容 hash，內容變了檔名就變，因此可以宣告一年 immutable；但正因為
// 它是 public，回應上的 Set-Cookie 必須拿掉（見 setPublicCacheControl）。
func hashedAssetHandler(frontendDir string) http.Handler {
	root := frontendAssetsRoot(frontendDir)
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 目錄清單與原始碼型別都不該從這條路由出去（見 staticPathIsDirectory
		// 與 blockedAssetSuffixes）。黑名單比對的是完整 URL 路徑，而目錄判斷要比
		// StripPrefix 之後的相對路徑 —— 那個 root 本身就是 assets 目錄。
		rel := strings.TrimPrefix(path.Clean(r.URL.Path), "/assets")
		if isBlockedAssetPath(r.URL.Path) || staticPathIsDirectory(root, rel) {
			http.NotFound(w, r)
			return
		}
		// 僅 GET 掛 immutable：非 GET 請求不該帶著這份快取指示。
		if r.Method == http.MethodGet {
			setPublicCacheControl(w, "public, max-age=31536000, immutable")
		}
		// StripPrefix 讓底層 FileServer 以 assets/ 之後的路徑去對應磁碟目錄。
		http.StripPrefix("/assets/", fs).ServeHTTP(w, r)
	})
}

// sourceAssetHandler 服務 /asset/ 底下的一手素材（PWA 圖示、logo、字型）。
//
// 這裡沒有明確的 Cache-Control：檔名沒有人工 hash，不能宣告長期快取。但
// http.FileServer 會帶 Last-Modified，共享快取仍可能依啟發式規則暫存；因此
// GET 回應一樣移除 Set-Cookie。真正的快取策略留給反向代理設定。
func sourceAssetHandler(frontendDir string) http.Handler {
	root := frontendAssetRoot(frontendDir)
	fs := http.StripPrefix("/asset/", http.FileServer(http.Dir(root)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isBlockedAssetPath(r.URL.Path) || staticPathIsDirectory(root, strings.TrimPrefix(r.URL.Path, "/asset/")) {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Del("Set-Cookie")
		}
		fs.ServeHTTP(w, r)
	})
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
		// 目錄且沒有 index.html 時回 404，而不是讓 FileServer 列出清單。
		// 有 index.html 的目錄（root 本身）照原樣交給 FileServer，那是首頁的
		// 送出方式，不是清單。
		if staticPathIsDirectory(root, r.URL.Path) {
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
				setPublicCacheControl(w, "public, max-age=31536000, immutable")
			} else if isCacheableStaticPath(cleanPath) {
				// 其他靜態資源給一天快取：這些檔名沒有 hash，必須留一個
				// 重新驗證的窗口給開發期更新。
				setPublicCacheControl(w, "public, max-age=86400")
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
