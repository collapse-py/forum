/*
security_hardening_test.go 釘住 2026-10-27 那一輪安全收尾的行為界線。

每一組測試對應一條當時仍在線上生效的暴露（audit-workspace/reports/audit-report.md）：

  - 目錄清單（F-05）：/assets/、/asset/ 與 catch-all 都不該把目錄列出來。
  - 後台 HTML 殼（F-04）：頁面路由要管理員，而 raw 檔名（/admin.html）要 404。
  - 備檔與依賴目錄（L-07）：.bak / .env / .git 這類路徑不該從靜態根出去。
  - /healthz 的錯誤文字（L-06）：回應只有固定值，錯誤留在日誌。
  - JSON 回應可被快取（L-08）：每一份 JSON 都標 no-store。

刻意只斷言「對外看到什麼」，不斷言內部實作：這些條的價值就在於回應本身。
*/

package httpapi

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/* ==========================================================================
   L-07：黑名單的覆蓋面
   ========================================================================== */

func TestHardeningStaticBlacklistCoversBackupAndAdminShells(t *testing.T) {
	blocked := map[string]bool{
		// 原本就擋住的（確認新增條目沒有把既有規則弄壞）。
		"/config.conf":         true,
		"/package.json":        true,
		"/service-worker.js":   false,
		"/forum-manifest.json": false,
		"/app.js":              false,
		"/logo.png":            false,

		// 新增：備檔與暫存檔。
		"/config.conf.bak": true,
		"/config.old":      true,
		"/secret.tmp":      true,
		"/.env":            true,
		"/backup.env":      true,
		"/server.log":      true,
		"/notes.md.bak":    true,
		"/run.sh":          true,
		"/deploy.ps1":      true,
		"/schema.yml":      true,
		"/users.csv":       true,
		"/forum.db":        true,

		// 新增：版控與依賴目錄。
		"/.git/HEAD":              true,
		"/.git/config":            true,
		"/node_modules/foo/x.js":  true,
		"/node_modules/foo/x.png": true,

		// 新增：後台 HTML 殼（副檔名清單沒有 .html，因此必須逐筆指名）。
		"/admin.html":         true,
		"/forum-admin.html":   true,
		"/forum-report.html":  true,
		"/forum-monitor.html": true,
		"/audit-log.html":     true,
		"/forum-stats.html":   true,
		"/export.html":        true,
		"/sessions.html":      true,
		"/blocks.html":        true,
		"/announcements.html": true,
		"/admin/monitor":      true,

		// 使用者頁面與一般 HTML 不受影響：它們是入口檔，不是後台殼。
		"/forum.html":         false,
		"/forum-post.html":    false,
		"/index.html":         false,
		"/forum-login.html":   false,
		"/forum/post/1":        true,
		"/forum/not-a-block":   true, // /forum/ 是目錄級封鎖，子路徑一律 404
		"/src/App.tsx":         true,
		"/範例/logo.png":         true,
		"/admin-notes.txt":     false, // 不是 /admin/ 前綴，也不是後台殼

		// 副檔名「在中間出現」也算命中：結尾不是敏感型別時，HasSuffix 會漏掉
		// 這雨種真實存在的路徑（改設定留下的備檔再加後綴、把原始碼包成 .js）。
		"/config.conf.bak.pre-readlimit": true,
		"/provider.tsx.js":               true,
		"/notes.env.js":                  true,

		// 但「中間出現」不能變成「檔名裡有這幾個字就擋」：這些是合法的前端資源，
		// 它们的點只出现在真正的副檔名前。
		"/env.js":       false,
		"/blog.js":      false,
		"/settings.js":  false,
		"/react-dom.js": false,
		"/tmp/x.js":     false, // /tmp 是目錄段，檔本身不是 .tmp
		"/sitemap.xml":  false, // 帶 .map 這三個字的目錄名不是來源對應檔
	}
	for path, want := range blocked {
		if got := isBlockedStaticPath(path); got != want {
			t.Errorf("isBlockedStaticPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestHardeningAssetBlacklistKeepsManifestWhileBlockingSource(t *testing.T) {
	// /assets/ 是建置產物目錄，那裡必須公開 .json（PWA manifest），因此它用
	// 另一份清單。這條測試釘的就是「兩份清單的差別只有一個 .json」。
	blocked := map[string]bool{
		"/assets/forum-manifest-abc123.json": false,
		"/assets/index-abc123.js":            false,
		"/assets/style-abc123.css":           false,
		"/asset/logo.png":                    false,

		"/assets/package.json":      false, // .json 在/dist 裡是 manifest，必須公開 —— 這正是另開清單的理由
		"/assets/App.tsx":           true,
		"/assets/main.go":           true,
		"/assets/config.conf":       true,
		"/assets/config.conf.bak":   true,
		"/assets/config.conf.bak.pre-readlimit": true, // .conf 在中間也算
		"/assets/provider.tsx.js":   true,             // .tsx 在中間也算
		"/assets/forum-CHhv90hV.js": false,            // 真實的建置產物名稱
		"/assets/foo.js.map":        true,             // source map 帶着完整原始碼
		"/assets/.env":              true,
		"/assets/schema.sql":        true,
		"/assets/x.yaml":            true,
		"/asset/notes.log":          true,
		"/asset/build.sh":           true,
		"/":                         false,
		"/assets/":                  false, // 目錄由 staticPathIsDirectory 處理，不是副檔名規則
	}
	for path, want := range blocked {
		if got := isBlockedAssetPath(path); got != want {
			t.Errorf("isBlockedAssetPath(%q) = %v, want %v", path, got, want)
		}
	}
}

/* ==========================================================================
   F-05：目錄清單
   ========================================================================== */

// hardeningAssetTree 建立一個「有 assets/ 與同層 asset/，沒有 index.html」的
// 前端目錄，正是 dist 的實際形狀。
func hardeningAssetTree(t *testing.T) string {
	t.Helper()
	parent := t.TempDir()
	frontendDir := filepath.Join(parent, "dist")
	if err := os.MkdirAll(filepath.Join(frontendDir, "assets"), 0o755); err != nil {
		t.Fatalf("建立 assets 目錄失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "assets", "index-abc.js"), []byte("js"), 0o644); err != nil {
		t.Fatalf("寫入建置資產失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "assets", "forum-manifest-abc.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("寫入 manifest 失敗: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "asset"), 0o755); err != nil {
		t.Fatalf("建立一手素材目錄失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(parent, "asset", "logo.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("寫入 logo 失敗: %v", err)
	}
	return filepath.Join(parent, "dist")
}

func TestHardeningAssetHandlersRejectDirectoryListings(t *testing.T) {
	frontendDir := hardeningAssetTree(t)

	t.Run("/assets/ 根目錄不列清單", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		hashedAssetHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/", nil))
		// FileServer 的原始行為是「先 301 到尾斜線、再回 HTML 清單」。
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("GET /assets/ = %d, want %d（body = %s）", recorder.Code, http.StatusNotFound, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "index-abc.js") {
			t.Errorf("回應裡出現了資產檔名，代表這是一份目錄清單：\n%s", recorder.Body.String())
		}
	})

	t.Run("/assets/ 子目錄不列清單", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(frontendDir, "assets", "fonts"), 0o755); err != nil {
			t.Fatalf("建立子目錄失敗: %v", err)
		}
		recorder := httptest.NewRecorder()
		hashedAssetHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/fonts/", nil))
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("/assets/ 的檔案照舊送出，manifest 不被副檔名規則擋掉", func(t *testing.T) {
		for _, path := range []string{"/assets/index-abc.js", "/assets/forum-manifest-abc.json"} {
			recorder := httptest.NewRecorder()
			hashedAssetHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d", path, recorder.Code, http.StatusOK)
			}
			if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
				t.Errorf("%s 的 Cache-Control = %q，want immutable 一年", path, got)
			}
		}
	})

	t.Run("/asset/ 根目錄不列清單", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		sourceAssetHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/asset/", nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("GET /asset/ = %d, want %d（body = %s）", recorder.Code, http.StatusNotFound, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "logo.png") {
			t.Errorf("回應裡出現了素材檔名，代表這是一份目錄清單：\n%s", recorder.Body.String())
		}
	})

	t.Run("/asset/ 的檔案照舊送出", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		sourceAssetHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/asset/logo.png", nil))
		covWantStatus(t, recorder, http.StatusOK)
	})

	t.Run("/asset/ 底下的備檔不送出", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(filepath.Dir(frontendDir), "asset", "config.conf.bak"), []byte("secret=1"), 0o644); err != nil {
			t.Fatalf("寫入備檔失敗: %v", err)
		}
		recorder := httptest.NewRecorder()
		sourceAssetHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/asset/config.conf.bak", nil))
		covWantStatus(t, recorder, http.StatusNotFound)
	})
}

func TestHardeningStaticRootBlocksListingsButKeepsIndex(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html>首頁</html>"), 0o644); err != nil {
		t.Fatalf("寫入 index.html 失敗: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "uploads"), 0o755); err != nil {
		t.Fatalf("建立 uploads 目錄失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "uploads", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("寫入檔案失敗: %v", err)
	}

	handler := safeStaticFileServer(root)

	t.Run("沒有 index 的目錄回 404 而不是清單", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/uploads/", nil))
		covWantStatus(t, recorder, http.StatusNotFound)
		if strings.Contains(recorder.Body.String(), "a.txt") {
			t.Errorf("回應列出了目錄內容：\n%s", recorder.Body.String())
		}
	})

	t.Run("有 index.html 的目錄照舊送出 index（首頁依賴這個行為）", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), "首頁") {
			t.Errorf("首頁沒有送出 index.html：%s", recorder.Body.String())
		}
	})
}

/* ==========================================================================
   F-04：後台頁面要管理員，且不能當成純檔案取得
   ========================================================================== */

func TestHardeningRequireAdminPageOnlyLetsAdminsThrough(t *testing.T) {
	h := newCovHarness(t)
	passed := false
	handler := h.srv.requireAdminPage(func(http.ResponseWriter, *http.Request) { passed = true })

	t.Run("沒有 session → 303 導向登入頁並帶 return", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, "/admin/monitor?window=1h", nil))
		covWantStatus(t, recorder, http.StatusSeeOther)
		want := "/forum/login?return=" + url.QueryEscape("/admin/monitor?window=1h")
		if got := recorder.Header().Get("Location"); got != want {
			t.Errorf("Location = %q, want %q", got, want)
		}
		if passed {
			t.Error("未登入卻進入了受保護的 handler")
		}
	})

	t.Run("已登入但不是管理員 → 同樣導向登入頁", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/monitor", nil)
		req.AddCookie(h.login("user@example.com", false))
		handler(recorder, req)
		covWantStatus(t, recorder, http.StatusSeeOther)
		if passed {
			t.Error("非管理員卻進入了受保護的 handler")
		}
	})

	t.Run("管理員 → 通過", func(t *testing.T) {
		passed = false
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/monitor", nil)
		req.AddCookie(h.login("admin@example.com", true))
		handler(recorder, req)
		if !passed {
			t.Error("管理員被擋在中介層外面")
		}
		if recorder.Code != http.StatusOK {
			t.Errorf("中介層不該自己寫狀態碼，got %d", recorder.Code)
		}
	})
}

func TestHardeningAdminShellRoutesAndRawFiles(t *testing.T) {
	h := newCovHarness(t)
	handler := h.srv.Handler()

	// Handler() 在前端沒有 dist 時會得到一個不存在的路徑，因此管理員那條路徑
	// 「通過授權但拿不到檔案」（404）。這正是要斷言的分工：授權判斷在檔案之前。
	t.Run("匿名造訪 /admin → 導向登入頁，不是 HTML", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin", nil))
		covWantStatus(t, recorder, http.StatusSeeOther)
		if !strings.HasPrefix(recorder.Header().Get("Location"), "/forum/login") {
			t.Errorf("Location = %q，沒有導向登入頁", recorder.Header().Get("Location"))
		}
	})

	t.Run("已登入的非管理員造訪後台頁面 → 導向登入頁", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/sessions", nil)
		req.AddCookie(h.login("user@example.com", false))
		handler.ServeHTTP(recorder, req)
		covWantStatus(t, recorder, http.StatusSeeOther)
	})

	t.Run("管理員造訪後台頁面 → 已經不在授權層被擋（拿到 404 而非 303）", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/monitor", nil)
		req.AddCookie(h.login("admin@example.com", true))
		handler.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusSeeOther {
			t.Fatal("管理員被導去登入頁，授權判斷錯了")
		}
	})

	// 殼檔的 raw 路徑：那十個 HTML 檔案是 catch-all 看得到的檔案，而路由上的
	// 判斷對它們完全不起作用 —— 所以黑名單必須自己蓋住這一路徑。
	t.Run("GET /admin.html 這類 raw 路徑 → 404", func(t *testing.T) {
		for _, path := range []string{
			"/admin.html", "/forum-admin.html", "/forum-report.html", "/forum-monitor.html",
			"/audit-log.html", "/forum-stats.html", "/export.html", "/sessions.html",
			"/blocks.html", "/announcements.html",
		} {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusNotFound {
				t.Errorf("GET %s = %d, want %d（raw 檔案路徑不該送出後台殼）", path, recorder.Code, http.StatusNotFound)
			}
			// 就算前端目錄裡真的有這個檔案（正式部署會），也不能送出。
			if strings.Contains(recorder.Body.String(), "<html") {
				t.Errorf("GET %s 送出了 HTML：\n%s", path, recorder.Body.String())
			}
		}
	})

	t.Run("前端目錄裡真的有這些殼檔時，靜態根仍然不送出", func(t *testing.T) {
		// 上面那條在「前端沒建置」的開發環境裡會是 404，但那個 404 可能只是
		// 檔案不存在。這一條把檔案放進去，證明 404 來自黑名單而不是缺檔。
		root := t.TempDir()
		shells := []string{"admin.html", "forum-admin.html", "sessions.html", "forum-monitor.html", "audit-log.html"}
		for _, name := range shells {
			if err := os.WriteFile(filepath.Join(root, name), []byte("<html>後台內容</html>"), 0o644); err != nil {
				t.Fatalf("寫入 %s 失敗: %v", name, err)
			}
		}
		static := safeStaticFileServer(root)
		for _, name := range shells {
			recorder := httptest.NewRecorder()
			static.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/"+name, nil))
			if recorder.Code != http.StatusNotFound {
				t.Errorf("GET /%s = %d, want %d", name, recorder.Code, http.StatusNotFound)
			}
			if strings.Contains(recorder.Body.String(), "後台內容") {
				t.Errorf("GET /%s 送出了後台殼檔：%s", name, recorder.Body.String())
			}
		}
	})

	t.Run("管理員也讀不到 raw 路徑 —— 那條路徑不存在，不是「被擋住」", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin.html", nil)
		req.AddCookie(h.login("admin@example.com", true))
		handler.ServeHTTP(recorder, req)
		covWantStatus(t, recorder, http.StatusNotFound)
	})
}

/* ==========================================================================
   L-06：/healthz 不帶依賴的錯誤文字
   ========================================================================== */

func TestHardeningHealthDoesNotExposeDependencyErrors(t *testing.T) {
	h := newCovHarness(t)
	covAdminUserCloseDB(t, h)
	recorder := h.do(h.srv.handleHealth, http.MethodGet, "/healthz", "", "", false)

	covWantStatus(t, recorder, http.StatusServiceUnavailable)
	body := recorder.Body.String()
	if !strings.Contains(body, `"mysql":"error"`) {
		t.Errorf("body 缺少固定的 error 值：%s", body)
	}
	// 驅動與客戶端的錯誤文字（「sql: database is closed」、dial 的位址、
	// Redis 的 AUTH 措辭）都不該出現在免登入的回應裡。
	for _, forbidden := range []string{"sql:", "closed", "dial", "connect", " refused", "127.0.0.1"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Errorf("body 洩露了依賴的錯誤文字（含 %q）：%s", forbidden, body)
		}
	}
}

/* ==========================================================================
   L-09：上傳的硬性上限與串流轉發
   ========================================================================== */

// hardeningInfinite 是不斷產生 chunk 的來源，讓「過大的上傳」不需要在記憶體裡
// 存一份 50MB 的副本 —— 上限是由 MaxBytesReader 在讀到第 50MB 時中止的，因此
// 測試只需要讓來源「永遠讀不完」，而不是真的準備一個超大檔案。
type hardeningInfinite struct{ chunk []byte }

func (r *hardeningInfinite) Read(p []byte) (int, error) {
	n := copy(p, r.chunk)
	return n, nil
}

func TestHardeningUploadRejectsOverSizeBody(t *testing.T) {
	h := newCovHarness(t)
	covProfileConfigureMedia(h)

	// 假檔案伺服器：如果它被呼叫到，代表上限沒有在後端這一側生效。
	var called bool
	files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"url":"/files/late.png"}`))
	}))
	t.Cleanup(files.Close)
	h.srv.cfg.FilesServerURL = files.URL

	body, contentType := hardeningOversizeMultipart(t, forumUploadMaxBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/api/forum/images", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Origin", covOrigin)
	req.AddCookie(h.login(covProfileEmail, false))

	recorder := httptest.NewRecorder()
	h.srv.handleForumImageUpload(recorder, req)
	covWantStatus(t, recorder, http.StatusRequestEntityTooLarge)
	if !strings.Contains(recorder.Body.String(), "too large") {
		t.Errorf("413 的訊息應該說明「太大」而不是其他原因：%s", recorder.Body.String())
	}
	if called {
		t.Error("過大的上傳被轉發給檔案伺服器了 —— 上限沒有在後端生效")
	}
	if h.sawQuery("INSERT INTO forum_media") {
		t.Error("過大的上傳寫入了所有權紀錄")
	}
}

// hardeningOversizeMultipart 產生一份「比上限還大」的 multipart 本文，以串流
// 的方式送出來（而不是先在記憶體裡組好）。
func hardeningOversizeMultipart(t *testing.T, size int64) (io.Reader, string) {
	t.Helper()
	reader, writer := io.Pipe()
	mw := multipart.NewWriter(writer)
	t.Cleanup(func() { _ = reader.Close() })
	go func() {
		part, err := mw.CreateFormFile("file", "huge.png")
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, io.LimitReader(&hardeningInfinite{chunk: make([]byte, 1<<20)}, size)); err != nil {
			writer.CloseWithError(err)
			return
		}
		if err := mw.Close(); err != nil {
			writer.CloseWithError(err)
			return
		}
		writer.Close()
	}()
	return reader, mw.FormDataContentType()
}

func TestHardeningUploadForwardsOnlyTheFileField(t *testing.T) {
	h := newCovHarness(t)
	covProfileConfigureMedia(h)

	// 攻擊者可以在表單裡夾帶任意欄位。轉發層必須只送出 "file" —— 否則那些欄位
	// 會帶著攻擊者的內容進到內網的檔案伺服器。
	var fields []string
	var fileContent string
	files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("轉發的表單解析失敗: %v", err)
			return
		}
		for name := range r.Form {
			fields = append(fields, name)
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("轉發的表單裡沒有 file 欄位: %v", err)
			return
		}
		defer f.Close()
		buf := new(strings.Builder)
		_, _ = io.Copy(buf, f)
		fileContent = buf.String()
		_, _ = w.Write([]byte(`{"url":"/files/ok.png"}`))
	}))
	t.Cleanup(files.Close)
	h.srv.cfg.FilesServerURL = files.URL

	body, contentType := hardeningMultipartWithExtraField(t, "注入的內容", "photo.png", []byte("PNGDATA"))
	req := httptest.NewRequest(http.MethodPost, "/api/forum/images", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Origin", covOrigin)
	req.AddCookie(h.login(covProfileEmail, false))

	recorder := httptest.NewRecorder()
	h.srv.handleForumImageUpload(recorder, req)
	covWantStatus(t, recorder, http.StatusCreated)
	if fileContent != "PNGDATA" {
		t.Errorf("轉發的檔案內容 = %q, want PNGDATA（串流轉發不能截斷或重複內容）", fileContent)
	}
	for _, name := range fields {
		if name != "file" {
			t.Errorf("轉發了額外的表單欄位 %q（應該只轉 file）", name)
		}
	}
}

func hardeningMultipartWithExtraField(t *testing.T, extraValue, filename string, content []byte) (io.Reader, string) {
	t.Helper()
	body, writer := io.Pipe()
	mw := multipart.NewWriter(writer)
	t.Cleanup(func() { _ = body.Close() })
	go func() {
		_ = mw.WriteField("note", extraValue)
		part, err := mw.CreateFormFile("file", filename)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		if _, err := part.Write(content); err != nil {
			writer.CloseWithError(err)
			return
		}
		if err := mw.Close(); err != nil {
			writer.CloseWithError(err)
			return
		}
		writer.Close()
	}()
	return body, mw.FormDataContentType()
}

/* ==========================================================================
   L-10：兩個來源表頭都缺席時看「有沒有帶憑證」
   ========================================================================== */

func TestHardeningCSRFRequiresOriginWhenCookiePresent(t *testing.T) {
	h := newCovHarness(t)
	cookie := h.login("victim@example.com", false)

	t.Run("帶 cookie、兩個表頭都沒有 → 不受信任", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/forum/posts", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		if h.srv.isTrustedOrigin(req) {
			t.Error("帶著 cookie 卻沒有來源訊號的請求被放行了 —— 那是跨站偽造請求的形狀")
		}
	})

	t.Run("不帶 cookie、兩個表頭都沒有 → 放行（curl／探測／tools/seed）", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/forum/posts", strings.NewReader("{}"))
		if !h.srv.isTrustedOrigin(req) {
			t.Error("不帶憑證的呼叫不該被來源檢查擋掉")
		}
	})

	t.Run("空值的 cookie 不算帶憑證", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/forum/posts", strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: h.srv.sessionCookieName(), Value: ""})
		if !h.srv.isTrustedOrigin(req) {
			t.Error("空 cookie 被當成有憑證了")
		}
	})

	t.Run("帶 cookie + 可信 Origin → 通過", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/forum/posts", strings.NewReader("{}"))
		req.AddCookie(cookie)
		req.Header.Set("Origin", covOrigin)
		if !h.srv.isTrustedOrigin(req) {
			t.Error("正常的同站寫入請求被擋掉了")
		}
	})

	t.Run("帶 cookie + 可信 Referer（Origin 缺席）→ 通過", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/forum/posts", strings.NewReader("{}"))
		req.AddCookie(cookie)
		req.Header.Set("Referer", covOrigin+"/forum/new")
		if !h.srv.isTrustedOrigin(req) {
			t.Error("只有 Referer 的合法請求被擋掉了")
		}
	})

	t.Run("帶 cookie + 不可信 Origin／不可信 Referer → 拒絕", func(t *testing.T) {
		for _, tc := range []struct{ name, origin, referer string }{
			{"攻擊者的 Origin", "https://evil.example", ""},
			{"攻擊者的 Referer", "", "https://evil.example/attack.html"},
			{"解析不了的 Referer", "", "not-a-url"},
		} {
			req := httptest.NewRequest(http.MethodPost, "/api/forum/posts", strings.NewReader("{}"))
			req.AddCookie(cookie)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			if h.srv.isTrustedOrigin(req) {
				t.Errorf("%s 被當成可信來源", tc.name)
			}
		}
	})

	t.Run("中介層端點也一樣：帶 cookie 無 Origin → 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/check", nil)
		req.AddCookie(cookie)
		recorder := httptest.NewRecorder()
		h.srv.requireTrustedOrigin(func(http.ResponseWriter, *http.Request) {
			t.Error("來源不受信任卻進入了 next")
		})(recorder, req)
		covWantStatus(t, recorder, http.StatusForbidden)
	})
}

/* ==========================================================================
   L-08：JSON 回應不可被快取
   ========================================================================== */

func TestHardeningJSONResponsesAreNotCacheable(t *testing.T) {
	h := newCovHarness(t)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		email  string
		admin  bool
	}{
		{"健康檢查", http.MethodGet, "/healthz", "", "", false},
		{"未授權的後台呼叫", http.MethodGet, "/api/admin/forum/posts", "", "", false},
		{"身分檢查", http.MethodGet, "/api/check", "", "", false},
		{"不存在的標籤查詢", http.MethodGet, "/api/admin/tags", "", "", false},
	} {
		recorder := h.do(h.srv.Handler().ServeHTTP, tc.method, tc.path, tc.body, tc.email, tc.admin)
		if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s 的 Cache-Control = %q，want no-store", tc.name, got)
		}
	}
}

/* ==========================================================================
   L-03：管理員身分的 session 快照在「人離開白名單」之後必須失效
   ========================================================================== */

// is_admin 是登入當下寫進 Redis 的一筆快照，因此收回權限原本要等那支 session 自然
// 過期（預設 72 小時）。這裡釘住的是「下一個請求就失效」：401、cookie 被要求丟掉、
// Redis 裡那支 key 消失。
func TestAdminSessionRevokedWhenEmailLeavesWhitelist(t *testing.T) {
	h := newCovHarness(t)
	h.srv.cfg.AllowedAdminEmail = []string{"admin@example.com"}

	inWhitelist := httptest.NewRequest(http.MethodGet, "/api/admin/forum/posts", nil)
	inWhitelist.AddCookie(h.login("admin@example.com", true))
	recorder := httptest.NewRecorder()
	if !h.srv.requireAdminForum(recorder, inWhitelist) {
		t.Fatalf("白名單內的 admin session 被擋 = %d", recorder.Code)
	}

	// 部署把這個人移出白名單（測試用直接改 cfg 表達，等同改設定檔後重啟）。
	h.srv.cfg.AllowedAdminEmail = []string{"someone-else@example.com"}

	out := httptest.NewRequest(http.MethodGet, "/api/admin/forum/posts", nil)
	out.AddCookie(h.login("admin@example.com", true))
	out.Header.Set("Origin", covOrigin)
	recorder2 := httptest.NewRecorder()
	if h.srv.requireAdminForum(recorder2, out) {
		t.Fatal("已移出白名單的舊 session 仍被當成管理員")
	}
	if recorder2.Code != http.StatusUnauthorized {
		t.Errorf("移出白名單後的回應 = %d, want 401", recorder2.Code)
	}
	if h.srv.sessions.IsLoggedIn(out) {
		t.Error("session 仍在 Redis 裡 —— 撤銷沒有發生")
	}
	var cleared bool
	for _, cookie := range recorder2.Result().Cookies() {
		if cookie.Name == h.srv.sessionCookieName() && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("回應沒有要求瀏覽器丟掉 session cookie")
	}
}

// 白名單為空時，is_admin 欄位就是唯一的權威來源（本機開發與測試的形態）。這一條釘住
// 「不能因為白名單是空的就把所有 admin 擋掉」。
func TestAdminSessionStillValidWithEmptyWhitelist(t *testing.T) {
	h := newCovHarness(t)
	h.srv.cfg.AllowedAdminEmail = nil

	request := httptest.NewRequest(http.MethodGet, "/api/admin/forum/posts", nil)
	request.AddCookie(h.login("admin@example.com", true))
	request.Header.Set("Origin", covOrigin)
	recorder := httptest.NewRecorder()
	if !h.srv.requireAdminForum(recorder, request) {
		t.Errorf("白名單為空時 admin session 被擋 = %d", recorder.Code)
	}
}

// /api/check 是前端決定「要不要顯示後台入口」的依據：過期的 true 會讓已被收回權限的
// 人繼續看到後台。
func TestCheckReportsAdminFalseAfterWhitelistRemoval(t *testing.T) {
	h := newCovHarness(t)
	h.srv.cfg.AllowedAdminEmail = []string{"admin@example.com"}
	adminCookie := h.login("admin@example.com", true)

	h.srv.cfg.AllowedAdminEmail = []string{"someone-else@example.com"}
	recorder := httptest.NewRecorder()
	check := httptest.NewRequest(http.MethodGet, "/api/check", nil)
	check.AddCookie(adminCookie)
	h.srv.handleCheck(recorder, check)

	if strings.Contains(recorder.Body.String(), `"isAdmin":true`) {
		t.Errorf("移出白名單後 /api/check 仍回報 admin: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"ok":true`) {
		t.Errorf("移出白名單不等同未登入: %s", recorder.Body.String())
	}
}

// 後台 HTML 殼：同一支過期的 admin session 不能再拿到頁面（走 F-04 那層導向）。
func TestAdminPageRedirectsAfterWhitelistRemoval(t *testing.T) {
	h := newCovHarness(t)
	h.srv.cfg.AllowedAdminEmail = []string{"admin@example.com"}
	adminCookie := h.login("admin@example.com", true)

	handler := h.srv.requireAdminPage(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h.srv.cfg.AllowedAdminEmail = []string{"someone-else@example.com"}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.AddCookie(adminCookie)
	handler(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Errorf("過期 admin session 拿到後台頁 = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); !strings.HasPrefix(loc, "/forum/login") {
		t.Errorf("導向目標 = %q", loc)
	}
}

/* ==========================================================================
   Service Worker 的快取標頭：部署新版之後不能還是舊版
   ========================================================================== */

// TestHardeningServiceWorkerIsNotLongCached 釘住 /service-worker.js 的回應不帶
// 任何長期快取或 public。
//
// 這一支的存在理由不是「快取越快越好」的反面，而是 SW 的更新機制：瀏覽器載入頁面
// 時會對這支檔案做一次更新檢查，而共享快取帶著 max-age 時那次檢查直接拿到快取的舊版。
// 症狀是「部署完成了，但使用者仍跑舊的攔截邏輯」，而且沒有使用者端的手段能讓它失效
// （舊 SW 就是那份快取的讀取者）。2026-10-28 的部署就踩過：邊緣節點帶著
// max-age=14400，新 SW 四小時內對使用者不可見。
func TestHardeningServiceWorkerIsNotLongCached(t *testing.T) {
	frontendDir := t.TempDir()
	script := "self.addEventListener('install', () => {});"
	if err := os.WriteFile(filepath.Join(frontendDir, "service-worker.js"), []byte(script), 0o644); err != nil {
		t.Fatalf("寫入 service-worker.js 失敗: %v", err)
	}

	recorder := httptest.NewRecorder()
	serviceWorkerHandler(frontendDir).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service-worker.js", nil))
	covWantStatus(t, recorder, http.StatusOK)
	if got := recorder.Body.String(); got != script {
		t.Errorf("回應本文 = %q，want 那支腳本本身", got)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/javascript" {
		t.Errorf("Content-Type = %q, want application/javascript", got)
	}
	if got := recorder.Header().Get("Service-Worker-Allowed"); got != "/" {
		t.Errorf("Service-Worker-Allowed = %q, want /", got)
	}

	cache := recorder.Header().Get("Cache-Control")
	// public / max-age / immutable 是 hashed 資產用的那一組：它們假設「檔名帶 hash，
	// 內容不會原地改動」。SW 的檔名沒有 hash，而它的內容正是會原地改動的那一份。
	for _, forbidden := range []string{"public", "max-age", "immutable"} {
		if strings.Contains(cache, forbidden) {
			t.Errorf("Cache-Control = %q，含 %s —— 舊版會被共享快取留住，部署的新 SW 對使用者不可見", cache, forbidden)
		}
	}
	if !strings.Contains(cache, "no-cache") {
		t.Errorf("Cache-Control = %q，want 含 no-cache（使用前必須回源驗證）", cache)
	}
}
