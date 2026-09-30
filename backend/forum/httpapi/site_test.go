/*
站名樣板層的測試（httpapi/site_test.go）。

這裡測的是三件事，任何一件壞掉都不會有明確的錯誤訊息：

  1. 取代有沒有真的發生。佔位符拼錯或設定沒讀到時，送出的是字面量
     "{{FORUM_NAME}}" —— 頁面看起來只是名稱怪怪的，不會報錯。
  2. 跳脫。站名是設定檔裡的任意字串，未跳脫就等於讓設定檔可以注入 HTML
     屬性或破壞 manifest 的 JSON 結構。
  3. <style> 區塊逐位元組不變。這是 CSP 的雜湊授權成立的前提，壞掉的症狀是
     整頁沒有版面。

狀態隔離：與同套件其他測試一致，每個測試自行建立 Server 與暫存目錄，
不使用 package 層級共用變數。
*/

package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"forum/forum/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// newSiteTestServer 建立只帶站名設定的 Server。
//
// 刻意不呼叫 NewServer（理由同 newImageTestServer）：站名樣板只讀 cfg 的三個
// 欄位，nil 的 db / sessions 在這條路徑上安全。
func newSiteTestServer(name, shortName, description string) *Server {
	return &Server{cfg: config.Config{
		ForumName:        name,
		ForumShortName:   shortName,
		ForumDescription: description,
	}}
}

// TestRenderHTMLTemplateSubstitutesOnlySitePlaceholders 驗證三個佔位符都會被換掉，
// 而其他看起來像佔位符的文字不會被動到。
//
// 「其他文字不被動到」這一條是刻意的：HTML 殼裡可能出現 i18n 風格的 {name}
// 或 CSS 內容，那些不是站名佔位符。取代器若寫成「把 {{…}} 裡的內容當鍵查表」
// 反而更寬鬆，那種寬鬆只會讓樣板裡多出一個沒有來源的佔位符。
func TestRenderHTMLTemplateSubstitutesOnlySitePlaceholders(t *testing.T) {
	server := newSiteTestServer("某某大學論壇", "某某", "某某大學的匿名論壇")

	const template = `<meta name="forum-name" content="{{FORUM_NAME}}" />` +
		`<meta name="forum-short-name" content="{{FORUM_SHORT_NAME}}" />` +
		`<p>{{FORUM_DESCRIPTION}}</p><span>{{NOT_A_SITE_KEY}}</span><i>{name}</i>`

	got := server.renderHTMLTemplate(template)

	for _, want := range []string{
		`content="某某大學論壇"`,
		`content="某某"`,
		`<p>某某大學的匿名論壇</p>`,
		// 未知的佔位符必須原樣留下，方便一眼看出是哪個鍵打錯。
		`{{NOT_A_SITE_KEY}}`,
		`<i>{name}</i>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered template missing %q\ngot: %s", want, got)
		}
	}
	if strings.Contains(got, placeholderName) || strings.Contains(got, placeholderShortName) {
		t.Errorf("rendered template still contains a site placeholder: %s", got)
	}
}

// TestRenderHTMLTemplateEscapesSiteName 驗證站名裡的引號與標記不會原樣進入
// HTML 屬性。
//
// 這是設定檔唯一能影響標記結構的入口，因此必須跳脫。html.EscapeString 產生的是
// 實體而非反斜線轉義，這裡因此比對實體字面值。
func TestRenderHTMLTemplateEscapesSiteName(t *testing.T) {
	server := newSiteTestServer(`Ev"il <script>alert(1)</script>`, `E"v`, "")

	got := server.renderHTMLTemplate(
		`<meta name="forum-name" content="{{FORUM_NAME}}" />` +
			`<meta name="forum-short-name" content="{{FORUM_SHORT_NAME}}" />`)

	if strings.Contains(got, `"Ev"il`) {
		t.Errorf("site name broke out of the attribute: %s", got)
	}
	if strings.Contains(got, `"E"v`) {
		t.Errorf("short name broke out of the attribute: %s", got)
	}
	if !strings.Contains(got, "&#34;") {
		t.Errorf("site name was not HTML-escaped: %s", got)
	}
	if strings.Contains(got, "<script>") {
		t.Errorf("site name injected a raw tag: %s", got)
	}
	// 兩個佔位符都必須被換掉：留下任何一個字面量，頁面上就會出現 "{{FORUM_NAME}}"。
	if strings.Contains(got, placeholderName) || strings.Contains(got, placeholderShortName) {
		t.Errorf("a site placeholder survived the render: %s", got)
	}
}

// TestRenderJSONTemplateProducesParsableManifest 驗證含引號與換行的站名放進
// manifest 後，JSON 仍然可以解析，而且取回的字串與設定值逐字相同。
//
// 為什麼要真的 Unmarshal：直接比對輸出字串只能證明「看起來對」，證明不了
// manifest 沒有被站名破壞 —— 而 manifest 壞掉的症狀是瀏覽器不提供安裝提示，
// 不會有任何錯誤訊息。
func TestRenderJSONTemplateProducesParsableManifest(t *testing.T) {
	const name = "Ev\"il\n論壇"
	server := newSiteTestServer(name, "Ev\"il", "第一行\n第二行")

	const template = `{"name": "{{FORUM_NAME}}", "short_name": "{{FORUM_SHORT_NAME}}", ` +
		`"description": "{{FORUM_DESCRIPTION}}", "start_url": "/forum"}`

	var parsed struct {
		Name        string `json:"name"`
		ShortName   string `json:"short_name"`
		Description string `json:"description"`
		StartURL    string `json:"start_url"`
	}
	if err := json.Unmarshal([]byte(server.renderJSONTemplate(template)), &parsed); err != nil {
		t.Fatalf("rendered manifest is not valid JSON: %v", err)
	}

	if parsed.Name != name {
		t.Errorf("name = %q, want %q", parsed.Name, name)
	}
	if parsed.ShortName != "Ev\"il" {
		t.Errorf("short_name = %q, want %q", parsed.ShortName, "Ev\"il")
	}
	if parsed.Description != "第一行\n第二行" {
		t.Errorf("description = %q, want the two-line description", parsed.Description)
	}
	// 未含佔位符的欄位必須完全不受影響。
	if parsed.StartURL != "/forum" {
		t.Errorf("start_url = %q, want /forum", parsed.StartURL)
	}
}

// TestServeHTMLFileRendersAndSetsHeaders 驗證頁面殼的送出行為：取代、型別、
// 快取標頭，以及呼叫端已設定的 Cache-Control 不被蓋掉。
func TestServeHTMLFileRendersAndSetsHeaders(t *testing.T) {
	server := newSiteTestServer("某某論壇", "某某", "")
	shell := filepath.Join(t.TempDir(), "forum.html")
	body := "<html><head><title>登入｜{{FORUM_NAME}}</title></head><body></body></html>"
	if err := os.WriteFile(shell, []byte(body), 0o600); err != nil {
		t.Fatalf("write shell: %v", err)
	}

	recorder := httptest.NewRecorder()
	server.serveHTMLFile(recorder, httptest.NewRequest(http.MethodGet, "/forum/login", nil), shell)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
	// 沒人指定時沿用 static.go 對 HTML 的規則。
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache, must-revalidate" {
		t.Errorf("Cache-Control = %q, want the HTML default", got)
	}
	if !strings.Contains(recorder.Body.String(), "<title>登入｜某某論壇</title>") {
		t.Errorf("body was not rendered: %s", recorder.Body.String())
	}
	// Content-Length 必須等於取代後的長度：內容被改寫過，沿用檔案長度會讓
	// 瀏覽器在讀到一半時斷線。
	if want := strconv.Itoa(len(recorder.Body.String())); recorder.Header().Get("Content-Length") != want {
		t.Errorf("Content-Length = %q, want %q", recorder.Header().Get("Content-Length"), want)
	}

	// 呼叫端指定的 Cache-Control（handleForumPage 的 no-store）必須保留。
	noStore := httptest.NewRecorder()
	noStore.Header().Set("Cache-Control", "no-store")
	server.serveHTMLFile(noStore, httptest.NewRequest(http.MethodGet, "/forum", nil), shell)
	if got := noStore.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want the caller's no-store preserved", got)
	}

	// 缺檔必須是 404（前端未建置時的既有行為），而不是 500 或 panic。
	missing := httptest.NewRecorder()
	server.serveHTMLFile(missing, httptest.NewRequest(http.MethodGet, "/forum", nil), shell+".missing")
	if missing.Code != http.StatusNotFound {
		t.Errorf("missing shell status = %d, want 404", missing.Code)
	}
}

// TestHandleForumManifestServesRenderedJSON 驗證 manifest 的型別、內容與缺檔行為。
func TestHandleForumManifestServesRenderedJSON(t *testing.T) {
	server := newSiteTestServer("某某論壇", "某某", "某某的匿名論壇")
	dir := t.TempDir()
	manifest := `{"name": "{{FORUM_NAME}}", "short_name": "{{FORUM_SHORT_NAME}}",` +
		`"description": "{{FORUM_DESCRIPTION}}", "start_url": "/forum"}`
	if err := os.WriteFile(filepath.Join(dir, "forum-manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	recorder := httptest.NewRecorder()
	server.handleForumManifest(dir)(recorder, httptest.NewRequest(http.MethodGet, "/forum-manifest.json", nil))

	// 型別不是 application/json 時瀏覽器不會把它當 manifest 解析，安裝提示直接
	// 消失且沒有任何錯誤訊息，因此這條比對不能放寬。
	if got := recorder.Header().Get("Content-Type"); got != "application/manifest+json" {
		t.Errorf("Content-Type = %q, want application/manifest+json", got)
	}
	var parsed map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if parsed["name"] != "某某論壇" || parsed["short_name"] != "某某" || parsed["description"] != "某某的匿名論壇" {
		t.Errorf("manifest fields = %v, want the configured site name", parsed)
	}

	// 空目錄（未建置前端）必須回 404。
	empty := httptest.NewRecorder()
	server.handleForumManifest(t.TempDir())(empty, httptest.NewRequest(http.MethodGet, "/forum-manifest.json", nil))
	if empty.Code != http.StatusNotFound {
		t.Errorf("missing manifest status = %d, want 404", empty.Code)
	}
}

// TestRenderKeepsStyleBlocksByteIdentical 驗證取代不會動到 <style> 區塊的內容。
//
// 這是整套機制最容易踩的坑：style-src 沒有 'unsafe-inline'，各頁樣式是靠
// 磁碟上那份 HTML 的 SHA-256 授權的（inlineStyleHashes）。佔位符一旦被寫進
// 樣式區塊，送出的位元組就與雜湊的輸入不同，症狀是整頁沒有版面。
//
// 驗證方式刻意比對「雜湊」而不是「字串」：雜湊落在 CSP 授權清單裡才真的代表
// 瀏覽器會套用該樣式。授權清單由磁碟上的檔案算出，與送出的內容完全無關 —
// 這正是兩邊可能不同步、而這條測試要抓的情況。
func TestRenderKeepsStyleBlocksByteIdentical(t *testing.T) {
	server := newSiteTestServer("某某論壇", "某某", "某某的匿名論壇")
	const css = "\n.brand { color: #111; }\n"
	shell := "<html><head><title>{{FORUM_NAME}}</title>\n" +
		`<meta name="forum-name" content="{{FORUM_NAME}}" />` + "\n" +
		"<style>" + css + "</style></head><body>{{FORUM_SHORT_NAME}}</body></html>"

	// 授權清單只從磁碟上的檔案算，因此把九個殼都以同一份內容寫進暫存目錄。
	dir := t.TempDir()
	for _, name := range frontendShellFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(shell), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	authorized := inlineStyleHashes(dir)

	blocks := styleBlockContents(server.renderHTMLTemplate(shell))
	if len(blocks) != 1 {
		t.Fatalf("rendered page has %d <style> blocks, want 1", len(blocks))
	}
	sum := sha256.Sum256([]byte(blocks[0]))
	hash := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if !slices.Contains(authorized, hash) {
		t.Errorf("rendered style block hash %s is not in the CSP allowlist %v", hash, authorized)
	}

	// 佔位符在樣式區塊之外必須真的被換掉，否則這條測試會在「取代根本沒發生」
	// 的情況下也通過。
	rendered := server.renderHTMLTemplate(shell)
	if !strings.Contains(rendered, "<title>某某論壇</title>") {
		t.Errorf("placeholders outside <style> were not substituted: %s", rendered)
	}
}
