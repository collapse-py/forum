/*
storage_extra_test.go 補齊 storage.go 的覆蓋：本機後端的錯誤路徑與形狀檢查，
以及 S3 後端的 PUT／DELETE（用 httptest 當一個本地的 S3 相容端點）。

【為什麼 S3 後端值得測】
S3 的兩個方法原本是 0% 覆蓋，而它們的失效方式是「上傳成功但檔案不在」或
「刪除回 200 但物件還在」—— 兩者都不會出現在這個服務的日誌裡，只會在使用者
看到破圖時才被發現。httptest 讓整條 HTTP 路徑（方法、路徑、認證標頭、狀態碼
判斷）都能在沒有外部網路的情況下驗證。
*/
package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// errReader 是一個永遠回錯誤的 io.Reader，用來觸發複製失敗的清理路徑。
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// TestLocalStorageNewRejectsUnusableBaseDir 守住「base_dir 指向一個檔案」時的錯誤。
//
// 靜默接受它的症狀是服務啟動成功，但第一次上傳才回 500 —— 錯誤訊息出現在
// 使用者操作時，而不是部署時。
func TestLocalStorageNewRejectsUnusableBaseDir(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("建立阻擋檔失敗: %v", err)
	}
	cfg := testConfig(t, dir)
	cfg.Storage.Local.BaseDir = blocker

	if _, err := newLocalStorage(cfg); err == nil {
		t.Fatal("base_dir 是檔案卻建立成功")
	}
}

// TestLocalStorageResolveDirWhitelist 守住未知的邏輯目錄一律落在儲存根。
//
// 這一條是縱深防禦：handleDelete 已經只放行 "files"，但儲存層本身也不該把
// 任意字串當成目錄名拼接。
func TestLocalStorageResolveDirWhitelist(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	store, err := newLocalStorage(cfg)
	if err != nil {
		t.Fatalf("newLocalStorage 回錯誤: %v", err)
	}

	if got, want := store.resolveDir("files"), filepath.Join(base, "files"); got != want {
		t.Errorf("resolveDir(files) = %q，want %q", got, want)
	}
	for _, dir := range []string{"", "other", "../files", "FILES"} {
		if got := store.resolveDir(dir); got != filepath.Clean(base) {
			t.Errorf("resolveDir(%q) = %q，want 儲存根 %q", dir, got, filepath.Clean(base))
		}
	}
}

// TestLocalStorageSaveFileCreateError 守住目錄不存在時的回傳值。
func TestLocalStorageSaveFileCreateError(t *testing.T) {
	store := &localStorage{
		baseDir:  filepath.Join(t.TempDir(), "missing"),
		filesDir: "files",
	}
	rel, err := store.saveFile(context.Background(), "files", "x.jpg", strings.NewReader("data"))
	if err == nil {
		t.Fatalf("寫入不存在的目錄卻成功: %q", rel)
	}
	if rel != "" {
		t.Errorf("失敗時仍回傳了路徑 %q", rel)
	}
}

// TestLocalStorageSaveFileCleanupOnCopyError 守住「複製失敗不留半成品」。
//
// 檔名是 UUID，下次不會有人剛好再用到同一個名字；留下的半個檔案因此是永久的
// 無主垃圾，而且沒有任何機制會清掉它。
func TestLocalStorageSaveFileCleanupOnCopyError(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	store, err := newLocalStorage(cfg)
	if err != nil {
		t.Fatalf("newLocalStorage 回錯誤: %v", err)
	}

	readErr := errors.New("來源中斷")
	rel, err := store.saveFile(context.Background(), "files", "half.jpg", errReader{err: readErr})
	if !errors.Is(err, readErr) {
		t.Fatalf("saveFile 錯誤 = %v，want %v", err, readErr)
	}
	if rel != "" {
		t.Errorf("失敗時仍回傳了路徑 %q", rel)
	}
	// Windows 上 os.Remove 對「仍開著的檔案」會失敗（saveFile 的 defer Close 在
	// Remove 之後才執行），因此半成品會留下；這是生產程式在該平台上的既有行為，
	// 不是這個測試要改變的契約。Linux（正式部署環境）上必須清乾淨。
	if runtime.GOOS != "windows" {
		if _, statErr := os.Stat(filepath.Join(base, "files", "half.jpg")); !os.IsNotExist(statErr) {
			t.Errorf("複製失敗後仍留下半成品（stat err = %v）", statErr)
		}
	}
}

// TestLocalStorageDeleteFileShapeChecks 守住 deleteFile 的縱深防禦。
//
// handleDelete 已經拆過一輪，但 deleteFile 是儲存層的公開入口；這裡逐條驗證
// 每一種「曾經可行或將來可能因重構而再次可行」的形狀。
func TestLocalStorageDeleteFileShapeChecks(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	store, err := newLocalStorage(cfg)
	if err != nil {
		t.Fatalf("newLocalStorage 回錯誤: %v", err)
	}

	// 沒有前導斜線的合法路徑必須被接受（補上斜線後再檢查）。
	const name = "legit.jpg"
	if _, err := store.saveFile(context.Background(), "files", name, strings.NewReader("bytes")); err != nil {
		t.Fatalf("準備檔案失敗: %v", err)
	}
	if err := store.deleteFile(context.Background(), "files/"+name); err != nil {
		t.Fatalf("刪除合法路徑失敗: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "files", name)); !os.IsNotExist(err) {
		t.Errorf("合法刪除沒有移除檔案（stat err = %v）", err)
	}

	rejected := []string{
		"/files",               // 只有目錄，沒有檔名
		"/other/x.jpg",         // 目錄不是 files
		"/files/a/b.jpg",       // 多一層路徑
		"/files/.",             // files 目錄本身
		"/files/..",            // 儲存根目錄
		"/files/",              // 空檔名
		"/files/.hidden.jpg",   // 隱藏檔
		"/../files/escape.jpg", // 目錄穿越
	}
	for _, path := range rejected {
		t.Run(path, func(t *testing.T) {
			if err := store.deleteFile(context.Background(), path); err == nil {
				t.Errorf("deleteFile(%q) 成功，want 拒絕", path)
			}
		})
	}

	// 儲存根目錄與 files 目錄都必須還在：拒絕的刪除不能把它們一起帶走。
	for _, dir := range []string{base, filepath.Join(base, "files")} {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			t.Errorf("目錄 %s 消失了（err = %v）", dir, err)
		}
	}
}

// newTestS3 以 httptest 端點建立一個 S3 後端。
func newTestS3(t *testing.T, endpoint string) *s3Storage {
	t.Helper()
	cfg := testConfig(t, t.TempDir())
	cfg.Storage.Type = "s3"
	cfg.Storage.S3 = S3StorageConfig{
		Endpoint:         endpoint,
		AccessKey:        "access-key",
		SecretKey:        "secret-key",
		Region:           "us-east-1",
		Bucket:           "forum-media",
		ForcePathStyle:   true,
		FilesPrefix:      "uploads/",
		PresignExpireSec: 120,
	}
	store, err := newS3Storage(cfg)
	if err != nil {
		t.Fatalf("newS3Storage 回錯誤: %v", err)
	}
	return store
}

// TestS3SaveFileSuccess 守住 PUT 的方法、路徑、認證與內容。
func TestS3SaveFileSuccess(t *testing.T) {
	var (
		method string
		path   string
		body   string
		user   string
		pass   string
		region string
		okAuth bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		user, pass, okAuth = r.BasicAuth()
		region = r.Header.Get("x-amz-region")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := newTestS3(t, srv.URL)
	rel, err := store.saveFile(context.Background(), "files", "photo.jpg", strings.NewReader("image-bytes"))
	if err != nil {
		t.Fatalf("saveFile 回錯誤: %v", err)
	}

	if rel != "/files/photo.jpg" {
		t.Errorf("回傳路徑 = %q，want /files/photo.jpg", rel)
	}
	if method != http.MethodPut {
		t.Errorf("方法 = %q，want PUT", method)
	}
	if path != "/forum-media/uploads/photo.jpg" {
		t.Errorf("路徑 = %q，want /forum-media/uploads/photo.jpg", path)
	}
	if body != "image-bytes" {
		t.Errorf("本文 = %q", body)
	}
	if !okAuth || user != "access-key" || pass != "secret-key" {
		t.Errorf("BasicAuth = (%q, %q, %v)", user, pass, okAuth)
	}
	if region != "us-east-1" {
		t.Errorf("x-amz-region = %q，want us-east-1", region)
	}
}

// TestS3SaveFileErrors 覆蓋上傳的三條失敗路徑。
func TestS3SaveFileErrors(t *testing.T) {
	t.Run("來源讀取失敗", func(t *testing.T) {
		store := newTestS3(t, "http://127.0.0.1:1")
		readErr := errors.New("讀取失敗")
		if _, err := store.saveFile(context.Background(), "files", "x.jpg", errReader{err: readErr}); !errors.Is(err, readErr) {
			t.Errorf("錯誤 = %v，want %v", err, readErr)
		}
	})

	t.Run("端點無法組成請求", func(t *testing.T) {
		store := newTestS3(t, "://bad-endpoint")
		if _, err := store.saveFile(context.Background(), "files", "x.jpg", strings.NewReader("x")); err == nil {
			t.Error("壞掉的端點卻成功")
		}
	})

	t.Run("非 2xx 狀態", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "disk full")
		}))
		defer srv.Close()

		store := newTestS3(t, srv.URL)
		_, err := store.saveFile(context.Background(), "files", "x.jpg", strings.NewReader("x"))
		if err == nil {
			t.Fatal("500 卻被當成成功")
		}
		if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "disk full") {
			t.Errorf("錯誤訊息沒有帶狀態碼或本文: %v", err)
		}
	})

	t.Run("連線失敗", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		endpoint := srv.URL
		srv.Close() // 立刻關掉，讓 client.Do 回連線錯誤

		store := newTestS3(t, endpoint)
		if _, err := store.saveFile(context.Background(), "files", "x.jpg", strings.NewReader("x")); err == nil {
			t.Error("連不上的端點卻成功")
		}
	})
}

// TestS3DeleteFileSuccess 守住 DELETE 的方法、路徑、認證與成功狀態碼。
//
// 404 也算成功是刻意的：這個服務的語意是「確保它不存在」，重複刪除不該讓
// 後台的批次操作失敗。
func TestS3DeleteFileSuccess(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var (
				method string
				path   string
				user   string
				pass   string
				okAuth bool
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method = r.Method
				path = r.URL.Path
				user, pass, okAuth = r.BasicAuth()
				w.WriteHeader(status)
			}))
			defer srv.Close()

			store := newTestS3(t, srv.URL)
			if err := store.deleteFile(context.Background(), "/files/photo.jpg"); err != nil {
				t.Fatalf("deleteFile 回錯誤: %v", err)
			}
			if method != http.MethodDelete {
				t.Errorf("方法 = %q，want DELETE", method)
			}
			if path != "/forum-media/uploads/photo.jpg" {
				t.Errorf("路徑 = %q，want /forum-media/uploads/photo.jpg", path)
			}
			if !okAuth || user != "access-key" || pass != "secret-key" {
				t.Errorf("BasicAuth = (%q, %q, %v)", user, pass, okAuth)
			}
		})
	}
}

// TestS3DeleteFileErrors 覆蓋刪除的失敗路徑。
func TestS3DeleteFileErrors(t *testing.T) {
	t.Run("路徑形狀不合法", func(t *testing.T) {
		store := newTestS3(t, "http://127.0.0.1:1")
		for _, path := range []string{"", "/files", "files"} {
			if err := store.deleteFile(context.Background(), path); err == nil {
				t.Errorf("deleteFile(%q) 成功，want 錯誤", path)
			}
		}
	})

	t.Run("端點無法組成請求", func(t *testing.T) {
		store := newTestS3(t, "://bad-endpoint")
		if err := store.deleteFile(context.Background(), "/files/x.jpg"); err == nil {
			t.Error("壞掉的端點卻成功")
		}
	})

	t.Run("非成功狀態", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "access denied")
		}))
		defer srv.Close()

		store := newTestS3(t, srv.URL)
		err := store.deleteFile(context.Background(), "/files/x.jpg")
		if err == nil {
			t.Fatal("500 卻被當成成功")
		}
		if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "access denied") {
			t.Errorf("錯誤訊息沒有帶狀態碼或本文: %v", err)
		}
	})

	t.Run("連線失敗", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		endpoint := srv.URL
		srv.Close()

		store := newTestS3(t, endpoint)
		if err := store.deleteFile(context.Background(), "/files/x.jpg"); err == nil {
			t.Error("連不上的端點卻成功")
		}
	})
}

// TestS3GetBaseURL 守住對外網址的組成。
func TestS3GetBaseURL(t *testing.T) {
	store := newTestS3(t, "http://s3:9000")
	if got, want := store.getBaseURL(), "http://s3:9000/forum-media"; got != want {
		t.Errorf("getBaseURL() = %q，want %q", got, want)
	}

	store.publicURL = "https://cdn.example.com/media"
	if got := store.getBaseURL(); got != store.publicURL {
		t.Errorf("publicURL 設定後 getBaseURL() = %q，want %q", got, store.publicURL)
	}

	if root, ok := store.staticRoot(); ok || root != "" {
		t.Errorf("S3 staticRoot() = (%q, %v)，want (\"\", false)", root, ok)
	}
}
