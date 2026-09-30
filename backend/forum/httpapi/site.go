/*
站名樣板層（httpapi/site.go）。

為什麼需要這個檔案
  站名以前是寫死的字串，散在三個地方：九個 HTML 殼的 <title>、forum-manifest.json，
  以及前端各頁的 React 文字。要讓部署者只改設定檔就能換站名，就必須讓伺服器在
  送出這些檔案時把佔位符換成設定值 —— 伺服器是唯一同時看得到「設定檔」與
  「前端資產」的地方。

三個佔位符
  {{FORUM_NAME}}         完整站名（FORUM_NAME）
  {{FORUM_SHORT_NAME}}   標誌短名（FORUM_SHORT_NAME，留空時沿用完整站名）
  {{FORUM_DESCRIPTION}}  manifest 說明（FORUM_DESCRIPTION，留空時沿用完整站名）

為什麼走「送出前取代」而不是「前端打 API 拿站名」
  1. 第一次繪製就正確。站名是前端模組層的常數（src/site.ts 直接讀 <meta>），
     不必等任何請求，因此不會出現「先用預設站名畫一帧再換掉」。
  2. 沒有 JavaScript 的爬蟲與讀屏器也拿得到正確的 <title>；走 API 就只有
     跑完 JavaScript 的瀏覽器看得到。
  3. 多一條設定的傳播路徑就多一處可能不同步的真相來源，而這一條完全由啟動時
     的設定快照決定。

一條硬性約束
  佔位符只能出現在樣式區塊之外。style-src 沒有 'unsafe-inline'，各頁的樣式是靠
  「磁碟上那份 HTML 的樣式區塊內容」的 SHA-256 授權的（securityheaders.go 的
  inlineStyleHashes）。一旦取代動到位元組，那組雜湊就對不上，症狀是整頁沒有
  版面且沒有任何錯誤訊息。

  這條規則也包括 HTML 註解：雜湊是純文字掃描（styleBlockContents 找的是
  樣式標籤的開頭字面量），註解裡只要出現那個字面量就會被當成真的開標籤，掃描
  範圍會一路延伸到真正的結束標籤，雜湊同樣對不上。因此九個頁面殼的站名註解
  只寫「樣式區塊」而不寫標籤本身 —— 這不是排版偏好，是這裡的掃描方式決定的。
*/

package httpapi

import (
	"encoding/json"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"

	"forum/forum/logger"
)

// 佔位符字面值。前端樣板（frontend/*.html 與 frontend/forum-manifest.json）
// 寫的就是這三個字串；改動時必須同步那些檔案，否則佔位符會原樣送到瀏覽器。
const (
	placeholderName        = "{{FORUM_NAME}}"
	placeholderShortName   = "{{FORUM_SHORT_NAME}}"
	placeholderDescription = "{{FORUM_DESCRIPTION}}"
)

// renderHTMLTemplate 把佔位符換成設定檔的站名，並以 HTML 跳脫取代結果。
//
// 跳脫是必要的，站名是設定檔裡的任意字串：未跳脫時，站名裡的 " 就能結束
// <meta content="…"> 的屬性並注入自己的標記，設定檔因此變成一份可以改寫 HTML
// 的輸入。html.EscapeString 一併處理 & < > " '，足以讓站名安全地出現在
// <title> 與屬性值中。
//
// 取代是單一趟的（strings.Replacer 不會重新掃描自己剛寫入的內容），所以站名
// 本身若剛好是 "{{FORUM_NAME}}" 也不會造成無限展開。
func (s *Server) renderHTMLTemplate(content string) string {
	return strings.NewReplacer(
		placeholderName, html.EscapeString(s.cfg.ForumName),
		placeholderShortName, html.EscapeString(s.cfg.ForumShortName),
		placeholderDescription, html.EscapeString(s.cfg.ForumDescription),
	).Replace(content)
}

// renderJSONTemplate 與 renderHTMLTemplate 相同，但取代結果是 JSON 字串常量的
// 內容（不含左右的引號）。
//
// 差異在跳脫方式：manifest 是 JSON，站名裡的 " 與 \ 必須以 JSON 的規則跳脫，
// 用 HTML 的跳脫會產生 &#34; 這種 HTML 實體，在 JSON 裡會原樣顯示給瀏覽器。
// 交給 encoding/json 處理同時也涵蓋了換行等控制字元 —— 那些字元直接放進
// manifest 會讓整份 JSON 解析失敗，安裝提示也跟著失效。
func (s *Server) renderJSONTemplate(content string) string {
	return strings.NewReplacer(
		placeholderName, jsonString(s.cfg.ForumName),
		placeholderShortName, jsonString(s.cfg.ForumShortName),
		placeholderDescription, jsonString(s.cfg.ForumDescription),
	).Replace(content)
}

// jsonString 把字串編碼成 JSON 字串常量的內容（不含左右的引號）。
//
// json.Marshal 唯一的失敗原因是遇到不支援的型別或無效的 UTF-8；字串在這兩種
// 情況下都會被替換成 U+FFFD 而不會回傳 error，因此這裡忽略該錯誤是安全的。
func jsonString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded[1 : len(encoded)-1])
}

// serveHTMLFile 送出前端頁面殼，並在送出前把站名佔位符換成設定值。
//
// 為什麼不用 http.ServeFile：它沒有任何機制改寫內容，而換站名正是這裡存在的
// 理由。改用自行讀檔換來三件事 —— 佔位符取代、明確的 Content-Type，以及
// 改設定後立刻反映（設定是啟動時的快照，本來就必須重啟才生效）。
//
// 讀不到檔案時回 404，與 http.ServeFile 的行為一致：前端缺檔會讓那些頁面
// 開不起來，但整套 API 服務仍要能啟動（見 frontendRoot 的說明）。額外補一行
// 警告日誌，因為「頁面全 404」最常見的原因就是找不到 dist，而預設的 404 本身
// 不會說出任何線索。
//
// Cache-Control 只在呼叫端還沒設定時才寫入：handleForumPage 與
// handleForumLoginPage 需要 no-store（內容與登入狀態相關），它們在呼叫本
// 函式之前就會先寫好，這裡不該把它蓋掉。沒人特別指定時沿用 static.go 對
// HTML 的同一條規則 —— 內容更新後使用者必須立刻拿到新版。
func (s *Server) serveHTMLFile(w http.ResponseWriter, r *http.Request, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		logger.Warnf("[HTTP] page shell missing: %s", path)
		http.NotFound(w, r)
		return
	}
	body := s.renderHTMLTemplate(string(raw))
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 明確寫出 Content-Length：取代後的長度與檔案長度不同，而 net/http 只有
	// 在 handler 自己設定時才會算對，否則改以 chunked 編碼送出。
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	// 寫出失敗（連線已斷開）時已經無法改成任何回應，也沒有可補救的副作用，
	// 因此忽略；記錄只會在斷線風暴時淹掉真正的錯誤。
	_, _ = w.Write([]byte(body))
}

// handleForumManifest 送出 PWA manifest，內容中的站名佔位符同樣由設定決定。
//
// 媒體型別是 application/manifest+json 而不是 application/json：瀏覽器以
// 副檔名 .json 猜出的型別不會被當成 manifest 解析，安裝提示也就不會出現。
//
// 快取用 no-cache 而不是 /assets/ 那條 immutable：這份檔案的路徑與內容
// 都不含 hash，改設定就會改內容，因此必須每次重新驗證。
func (s *Server) handleForumManifest(frontendDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := os.ReadFile(frontendAssetPath(frontendDir, "forum-manifest.json"))
		if err != nil {
			logger.Warnf("[HTTP] forum manifest missing under %s", frontendDir)
			http.NotFound(w, r)
			return
		}
		body := s.renderJSONTemplate(string(raw))
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write([]byte(body))
	}
}
