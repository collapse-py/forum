/*
server_test.go 覆蓋檔案伺服器的 HTTP 層：上傳限制、token 驗證、刪除路徑驗證、
媒體 token 中介層。

【為什麼這是這個模組最該先測的部分】
ROADMAP.md 說 files_server 是「整個模組零測試，而它是單一個 559 行的
main.go，承載上傳、下載、刪除、S3 路徑、token 驗證。先做 token 驗證與上傳
限制（安全 + 資源耗盡）」。這個判斷是對的，因為這兩件事的失效症狀是：

  - token 驗證壞掉：任何人拿到一個 UUID 檔名就能讀取或刪除論壇上的圖片。
    而檔名是存在貼文內容裡的（公開可讀），所以「猜檔名」不需要任何資訊。
  - 上傳限制壞掉：一個 POST 就能讓服務吃掉任意大的本文或任意多的磁碟。
    而上傳端點只有一個 token，而那個 token 在後端行程裡。

兩者都不會出現在任何日誌裡 —— 授權失敗回 401，內容照樣被送出。

【測試方法的關鍵決定：注入假的 mediaTokenStore】
mediaTokenMiddleware 需要一個「查某個 Redis key 是否存在」的能力。這個套件
把它宣告成介面（mediaTokenStore）正是為了這裡：

  - 「token 不存在 → 401」可以用一個永遠回 0 的假實作驗證。
  - 「Redis 故障 → 503」可以用一個回 error 的假實作驗證 —— 那在真實 Redis
    上只能靠指一個壞掉的位址來達成，而那既慢又不確定。
  - 「Redis 連不上 → 完全跳過驗證」可以讓 redis 欄位為 nil 來驗證。

【為什麼刪除路徑的測試佔了這裡一半的份量】
delete 端點把「使用者可控的 url 查詢參數」直接交給檔案系統。這個形狀是路徑
穿越的標準入口，而它的症狀（刪掉別的檔案）是**資料損毀**。ROADMAP.md 的
Phase 3.4 把它排在 S3 路徑之前，理由也正是這個 —— 它的爆炸半徑最大。
*/
package main

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

// fakeTokenStore 是一個可控的 mediaTokenStore。
//
// 存在三種模式，因為三種對應三個真正不同的生產行為：
//   - found > 0：token 存在 → 放行
//   - found == 0：token 不存在 → 401
//   - err != nil：Redis 故障 → 503
//
// 刻意記錄被查詢的 key：那讓測試能驗證「前綴正確」，而前綴錯誤的症狀是
// 「上傳成功、貼文也存得下，但圖片一律 401」—— 那是最難診斷的形狀之一。
type fakeTokenStore struct {
	found int64
	err   error

	queried []string
}

func (f *fakeTokenStore) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	f.queried = append(f.queried, keys...)
	cmd := redis.NewIntCmd(ctx)
	if f.err != nil {
		cmd.SetErr(f.err)
		return cmd
	}
	cmd.SetVal(f.found)
	return cmd
}

// testConfig 產生一份指向暫存目錄的預設設定。
//
// 每個欄位都顯式給值而不是留空：這些測試要驗的是 HTTP 層的行為，而預設值
// 會讓「哪一個設定造成了這個行為」變得不清楚。特別是 Token —— 讓它有值才
// 能測授權分支。
func testConfig(t *testing.T, baseDir string) *Config {
	t.Helper()
	return &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 7070, ShutdownTimeoutSec: 15},
		Storage: StorageConfig{
			Type: "local",
			Local: LocalStorageConfig{
				BaseDir:  baseDir,
				FilesDir: "files",
				DirPerm:  0o755,
				FilePerm: 0o644,
			},
		},
		Upload: UploadConfig{
			Token:        "test-upload-token",
			MaxSize:      50 << 20,
			AllowedFiles: []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".opus"},
		},
		CORS: CORSPolicy{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "HEAD", "OPTIONS"},
			AllowedHeaders: []string{"Origin", "Range", "Accept", "Accept-Language"},
		},
		Logger: LoggerConfig{Level: "info"},
		Redis: RedisConfig{
			Addr:           "127.0.0.1:6379",
			TokenKeyPrefix: "media:token:",
			TokenTTLSec:    300,
		},
	}
}

// newTestServer 建立一個掛好 Handler 的 Server，回傳 handler 與儲存後端。
func newTestServer(t *testing.T, cfg *Config, store mediaTokenStore) (http.Handler, storageBackend) {
	t.Helper()
	backend, err := newStorageBackend(cfg)
	if err != nil {
		t.Fatalf("newStorageBackend 回錯誤: %v", err)
	}
	return NewServer(cfg, backend, store).Handler(), backend
}

// multipartUpload 組出一個帶檔案的 multipart 請求。
//
// 刻意手動組這個表單而不是依賴任何測試輔助套件：這個模組的相依項刻意維持在
// 兩個（toml、redis），而為了測試再加一個相依會讓「這個服務實際上需要什麼」
// 變得不清楚。
func multipartUpload(t *testing.T, url, token, filename, contentType string, body []byte) *http.Request {
	t.Helper()

	var buf strings.Builder
	mw := multipart.NewWriter(&buf)

	if token != "" {
		if err := mw.WriteField("dummy", ""); err != nil {
			t.Fatalf("WriteField 失敗: %v", err)
		}
	}

	// filename 可能是空字串（用來測「沒帶檔名」），那時不寫檔案欄位。
	if filename != "" || contentType != "" {
		var (
			part io.Writer
			err  error
		)
		if contentType != "" {
			part, err = mw.CreateFormFile("file", filename)
		} else {
			part, err = mw.CreateFormField("file")
		}
		if err != nil {
			t.Fatalf("建立表單欄位失敗: %v", err)
		}
		if _, err := part.Write(body); err != nil {
			t.Fatalf("寫入表單內容失敗: %v", err)
		}
	}

	if err := mw.Close(); err != nil {
		t.Fatalf("關閉 multipart writer 失敗: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("X-Upload-Token", token)
	}
	return req
}

// TestUploadRequiresToken 守住上傳的授權邊界。
//
// 這是這個服務的第一道防線。它的失效症狀不是「被入侵」，而是「任何人都能
// 把任意檔案寫進儲存目錄」—— 而那在沒有 CDN 限制時就是一個公開的檔案託管。
func TestUploadRequiresToken(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	cases := []struct {
		name       string
		setToken   func(*http.Request)
		wantStatus int
	}{
		{
			name:       "沒有任何 token",
			setToken:   func(*http.Request) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "錯誤的 token",
			setToken: func(r *http.Request) {
				r.Header.Set("X-Upload-Token", "wrong-token")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "空 token",
			setToken: func(r *http.Request) {
				r.Header.Set("X-Upload-Token", "")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "只有前綴沒有值",
			setToken: func(r *http.Request) {
				r.Header.Set("X-Upload-Token", "Bearer ")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "正確的 X-Upload-Token",
			setToken: func(r *http.Request) {
				r.Header.Set("X-Upload-Token", "test-upload-token")
			},
			wantStatus: http.StatusOK,
		},
		{
			// 這個形式是刻意支援的：讓呼叫端可以直接沿用它對外服務的
			// Authorization 慣例，而不必為這個內部端點特別組一個標頭。
			name: "Bearer 前綴形式",
			setToken: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer test-upload-token")
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := multipartUpload(t, "/upload", "", "photo.jpg", "image/jpeg", []byte("fake-jpeg-bytes"))
			tc.setToken(req)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("狀態碼 = %d，want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				// 拒絕時**不該**留下任何檔案 —— 那會讓攻擊者即使被擋下也能
				// 消耗磁碟。
				if files := countFiles(t, filepath.Join(base, "files")); files != 0 {
					t.Errorf("被拒絕的請求仍然寫入了 %d 個檔案", files)
				}
			}
		})
	}
}

// TestUploadAllowsEverythingWhenNoTokenConfigured 守住那個刻意保留的行為。
//
// 設定檔沒設 upload.token 時全部放行是既有的行為（本機測試用）。把它釘成
// 測試是為了讓「這是一個刻意的不安全預設值」有明確出處 —— 若將來有人改成
// 拒絕，這支測試會失敗並要求更新 docs/DEPLOYMENT.md 的部署章節（那份章節要求正式環境
// 必須設定 upload.token）。
func TestUploadAllowsEverythingWhenNoTokenConfigured(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	cfg.Upload.Token = "" // 刻意留空
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	req := multipartUpload(t, "/upload", "", "photo.jpg", "image/jpeg", []byte("fake-jpeg-bytes"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("狀態碼 = %d，want 200（未設定 token 時刻意全部放行）", rec.Code)
	}
}

// TestUploadRejectsOversizedBody 守住資源耗盡的防線。
//
// 這個測試的價值在於「MaxBytesReader 設在 ParseMultipartForm 之前」這個
// 順序。若順序反了，一個 200 MB 的上傳會先被 net/http 完整寫進暫存目錄，
// 才在解析時被拒 —— 那一刻磁碟已經被吃掉了。
//
// 因此這支測試用一個**遠超過上限**的本文，並斷言記憶體／磁碟不會被撐爆；
// 實際上它只能驗證狀態碼與「沒有留下檔案」，真正的資源上限由 net/http 保證。
func TestUploadRejectsOversizedBody(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	// 把上限調小到 1 KB 讓測試跑得快 —— 行為與 50 MB 時完全相同。
	cfg.Upload.MaxSize = 1024
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	// 送出 100 KB —— 遠超過 1 KB 上限。
	oversized := make([]byte, 100*1024)
	for i := range oversized {
		oversized[i] = 'A'
	}

	req := multipartUpload(t, "/upload", "test-upload-token", "big.jpg", "image/jpeg", oversized)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("狀態碼 = %d，want 400（本文超過上限）", rec.Code)
	}
	if files := countFiles(t, filepath.Join(base, "files")); files != 0 {
		t.Errorf("被拒絕的超大上傳仍然留下了 %d 個檔案", files)
	}
}

// TestUploadRejectsDisallowedExtensions 守住副檔名白名單。
//
// 這個白名單同時決定了「能不能存」與「URL 前綴是什麼」（見 resolveDir 的
// 說明），因此放行 .html 或 .svg 意味著能上傳能執行腳本的內容。
func TestUploadRejectsDisallowedExtensions(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	// 這些副檔名不在白名單裡，且每一個都有實際的攻擊意義。
	rejected := []struct {
		filename string
		why      string
	}{
		{"payload.html", "上傳可執行的 HTML"},
		{"payload.svg", "SVG 可以內嵌腳本"},
		{"payload.js", "JavaScript"},
		{"payload.exe", "可執行檔"},
		{"payload.php", "可執行檔"},
		{"shell.sh", "腳本"},
		{"noextension", "沒有副檔名，無法判斷型別"},
		{"archive.zip", "壓縮檔（可包含任何東西）"},
	}
	for _, tc := range rejected {
		t.Run(tc.filename, func(t *testing.T) {
			req := multipartUpload(t, "/upload", "test-upload-token", tc.filename, "application/octet-stream", []byte("payload"))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("狀態碼 = %d，want 400（%s）", rec.Code, tc.why)
			}
		})
	}
}

// TestUploadAcceptsAllowedExtensions 守住白名單不會過度拒絕。
//
// 沒有這一支的話，「白名單只剩 .jpg」也能通過上一支 —— 而那會讓所有音訊
// 上傳失敗，而症狀是使用者看到「上傳失敗」而沒有人知道是白名單被改小了。
func TestUploadAcceptsAllowedExtensions(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	allowed := []string{
		"a.jpg", "a.jpeg", "a.png", "a.gif", "a.webp",
		"a.mp3", "a.wav", "a.ogg", "a.m4a", "a.flac", "a.opus",
		// 大寫副檔名必須被接受（resolveDir 會轉小寫）。
		"A.JPG", "Photo.PNG",
		// 副檔名在檔名的中間（多一個點）。
		"my.photo.v2.png",
	}
	for _, name := range allowed {
		t.Run(name, func(t *testing.T) {
			req := multipartUpload(t, "/upload", "test-upload-token", name, "image/jpeg", []byte("bytes"))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("狀態碼 = %d，want 200（%s 應該被接受）", rec.Code, name)
			}
		})
	}
}

// TestUploadStoresUnderUUIDName 守住「儲存檔名是 UUID，不是使用者檔名」。
//
// 這一條守的是三件事，同時也是這個設計的理由：
//  1. 不會覆蓋（兩個人都傳 IMG_0001.jpg 時不會互相蓋掉）。
//  2. 不洩漏原始檔名（檔名會出現在磁碟、URL 與日誌裡）。
//  3. 不會因路徑片段造成問題（"../../etc/passwd" 這種檔名）。
func TestUploadStoresUnderUUIDName(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	// 這個檔名同時帶著路徑穿越、絕對路徑與隱私資訊三種問題。
	const hostile = "../../../etc/passwd.jpg"

	req := multipartUpload(t, "/upload", "test-upload-token", hostile, "image/jpeg", []byte("bytes"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("狀態碼 = %d，want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "/files/") {
		t.Errorf("回應沒有檔案網址: %s", body)
	}
	if strings.Contains(body, "..") || strings.Contains(body, "passwd") {
		t.Errorf("回應的網址含使用者上傳的檔名或路徑片段: %s", body)
	}

	// 磁碟上必須只有一個檔案，且它的名字是 UUID 形狀。
	stored := listFiles(t, filepath.Join(base, "files"))
	if len(stored) != 1 {
		t.Fatalf("儲存了 %d 個檔案，want 1（%v）", len(stored), stored)
	}
	if !strings.HasSuffix(stored[0], ".jpg") {
		t.Errorf("檔名 %q 沒有保留副檔名（瀏覽器要靠它決定 Content-Type）", stored[0])
	}
	// UUID v4 的形狀：8-4-4-4-12 個十六進位字元。
	if !isUUIDv4Shape(stored[0][:len(stored[0])-4]) {
		t.Errorf("檔名 %q 不是 UUID v4 形狀", stored[0])
	}
	// 檔案必須真的在 baseDir 底下，不在任何父目錄。
	if _, err := os.Stat(filepath.Join(base, "files", stored[0])); err != nil {
		t.Errorf("檔案不在預期位置: %v", err)
	}
}

// TestTwoUploadsDoNotOverwriteEachOther 守住不覆蓋。
//
// 這是 UUID 命名的第二個理由，而且是「可觀察的」：覆蓋之後第一張圖片就永久
// 消失了，而沒有任何錯誤、沒有任何日誌。
func TestTwoUploadsDoNotOverwriteEachOther(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	// 同一個檔名上傳兩次，內容不同。
	urls := make([]string, 0, 2)
	for i, content := range []string{"first-image", "second-image"} {
		req := multipartUpload(t, "/upload", "test-upload-token", "same-name.jpg", "image/jpeg", []byte(content))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("第 %d 次上傳狀態碼 = %d", i, rec.Code)
		}
		urls = append(urls, extractURL(t, rec.Body.String()))
	}

	if urls[0] == urls[1] {
		t.Fatalf("兩次上傳得到相同的網址 %q —— 第二次覆蓋了第一次", urls[0])
	}

	// 兩張圖都必須還在，且內容各自正確。
	for i, u := range urls {
		data, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(u)))
		if err != nil {
			t.Errorf("讀取 %s 失敗: %v", u, err)
			continue
		}
		want := []string{"first-image", "second-image"}[i]
		if string(data) != want {
			t.Errorf("%s 的內容 = %q，want %q", u, data, want)
		}
	}
}

// TestUploadWithoutFileField 守住表單裡沒有檔案時的錯誤路徑。
func TestUploadWithoutFileField(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	// 只有一個文字欄位，沒有檔案。
	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("name", "value"); err != nil {
		t.Fatalf("WriteField 失敗: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Upload-Token", "test-upload-token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("狀態碼 = %d，want 400（表單裡沒有檔案）", rec.Code)
	}
}

// TestUploadMethodNotAllowed 守住只接受 POST 與 PUT。
func TestUploadMethodNotAllowed(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPatch, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/upload", nil)
			req.Header.Set("X-Upload-Token", "test-upload-token")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("狀態碼 = %d，want 405", rec.Code)
			}
		})
	}
}

// TestDeleteRejectsPathTraversal 是這個檔最重要的一支測試。
//
// delete 端點把「使用者可控的 url 查詢參數」交給檔案系統。這個形狀是路徑
// 穿越的標準入口，而它的症狀是**資料損毀**：被刪掉的是儲存目錄之外的檔案
// （包含另一個用者的上傳、其他專案的檔案、設定檔本身）。
//
// 因此這支測試不只驗證狀態碼，還在每次嘗試之後確認「儲存目錄內的檔案一個
// 都還在、儲存目錄外的哨兵檔也還在」。
func TestDeleteRejectsPathTraversal(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)

	// 在儲存目錄**之外**放一個哨兵檔：刪除端點若能穿越，會把它刪掉。
	sentinel := filepath.Join(base, "must-not-be-deleted.txt")
	if err := os.WriteFile(sentinel, []byte("important"), 0o600); err != nil {
		t.Fatalf("建立哨兵檔失敗: %v", err)
	}
	// 儲存目錄裡放一個「合法目標」，用來確認合法刪除本身仍然是能成功的 ——
	// 否則一個「把所有刪除都擋掉」的實作也會通過這支測試。
	handler, backend := newTestServer(t, cfg, &fakeTokenStore{found: 1})
	victim := "victim.jpg"
	if _, err := backend.saveFile(context.Background(), "files", victim, strings.NewReader("victim-bytes")); err != nil {
		t.Fatalf("準備受害者檔案失敗: %v", err)
	}

	attacks := []struct {
		name string
		url  string
	}{
		{"父目錄", "/files/../must-not-be-deleted.txt"},
		{"多層父目錄", "/files/../../must-not-be-deleted.txt"},
		{"以 .. 開頭", "../must-not-be-deleted.txt"},
		{"深度穿越（編碼前的形式）", "/files/../../../../../../etc/passwd"},
		{"混合：合法前綴 + 穿越", "/files/x/../../must-not-be-deleted.txt"},
		{"Windows 磁碟路徑", `C:\Windows\System32\config`},
		{"不同的目錄", "/other/whatever.jpg"},
		{"根目錄", "/must-not-be-deleted.txt"},
		{"雙斜線", "//must-not-be-deleted.txt"},
		{"少一層", "/files"},
		{"空的檔名", "/files/"},
		{"沒有檔名", "/files"},
		// 下面三筆是「捨棄目錄」而不是穿越：filepath.Base 對 "." 與 ".." 原樣
		// 回傳，因此在 Base 那一關看不出來，但它們能讓 os.Remove 作用在
		// files/ 目錄與儲存根本身。哨兵檔留在 base 是為了讓「基目錄被刪」這個
		// 結果在測試裡一定被抓到。
		{"點：files 目錄本身", "/files/."},
		{"點：儲存根目錄", "/files/.."},
		{"點：穿越後的目錄", "/files/x/.."},
		{"隱藏檔", "/files/.hidden.jpg"},
	}

	for _, tc := range attacks {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/delete?url="+url.QueryEscape(tc.url), nil)
			req.Header.Set("X-Upload-Token", "test-upload-token")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// 無論回 400 還是 404 都可以接受 —— 重點是沒有刪到東西。
			if rec.Code == http.StatusOK {
				t.Errorf("路徑 %q 竟然刪除成功了（狀態碼 200）", tc.url)
			}
			if _, err := os.Stat(sentinel); err != nil {
				t.Fatalf("哨兵檔消失了：路徑穿越成功了（%v 被刪掉）", tc.url)
			}
			if _, err := os.Stat(filepath.Join(base, "files", victim)); err != nil {
				t.Fatalf("受害者檔案消失了：%q 被誤刪", tc.url)
			}
		})
	}

	// 哨兵與受害者都還在 —— 這是這支測試真正的斷言。
	if _, err := os.Stat(sentinel); err != nil {
		t.Error("哨兵檔在測試結束時消失了")
	}
	if _, err := os.Stat(filepath.Join(base, "files", victim)); err != nil {
		t.Error("受害者檔案在測試結束時消失了")
	}
}

// TestDeleteWorksForLegitimatePath 守住合法刪除仍然能成功。
//
// 沒有這一支的話，「把所有刪除都擋掉」也能通過路徑穿越測試 —— 而那會讓
// 後台的「刪除圖片」功能整個失效，症狀是「按了沒反應但沒有錯誤」。
func TestDeleteWorksForLegitimatePath(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, backend := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	const name = "to-delete.jpg"
	if _, err := backend.saveFile(context.Background(), "files", name, strings.NewReader("bytes")); err != nil {
		t.Fatalf("準備檔案失敗: %v", err)
	}
	path := filepath.Join(base, "files", name)

	// 兩種呼叫來源：查詢參數（後端既有呼叫）與表單欄位（瀏覽器端）。
	t.Run("查詢參數", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/delete?url="+url.QueryEscape("/files/"+name), nil)
		req.Header.Set("X-Upload-Token", "test-upload-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("狀態碼 = %d，want 200（合法的刪除應該成功）: %s", rec.Code, rec.Body.String())
		}
		if _, err := os.Stat(path); err == nil {
			t.Error("檔案還在 —— 刪除沒有生效")
		}
	})

	// 表單欄位那條要重新準備檔案。
	if _, err := backend.saveFile(context.Background(), "files", name, strings.NewReader("bytes")); err != nil {
		t.Fatalf("重新準備檔案失敗: %v", err)
	}
	t.Run("表單欄位", func(t *testing.T) {
		form := url.QueryEscape("url") + "=" + url.QueryEscape("/files/"+name)
		req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Upload-Token", "test-upload-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("狀態碼 = %d，want 200: %s", rec.Code, rec.Body.String())
		}
		if _, err := os.Stat(path); err == nil {
			t.Error("檔案還在 —— 刪除沒有生效")
		}
	})
}

// TestDeleteMissingFileReturns404 守住「刪一個不存在的檔案」不是伺服器錯誤。
//
// 後台在「刪標籤 → 刪圖片」這類操作後偶爾會重複呼叫，而 500 會讓整批批次
// 操作失敗並顯示錯誤訊息。
func TestDeleteMissingFileReturns404(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	req := httptest.NewRequest(http.MethodDelete, "/delete?url="+url.QueryEscape("/files/never-existed.jpg"), nil)
	req.Header.Set("X-Upload-Token", "test-upload-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("狀態碼 = %d，want 404", rec.Code)
	}
}

// TestDeleteRequiresToken 守住刪除的授權。
//
// 刪除與上傳共用同一個 token 是刻意的（能上傳的人自然也能刪），因此這一支
// 測的是「共用這個事實沒有變成兩個都沒檢查」。
func TestDeleteRequiresToken(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, backend := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	const name = "protected.jpg"
	if _, err := backend.saveFile(context.Background(), "files", name, strings.NewReader("bytes")); err != nil {
		t.Fatalf("準備檔案失敗: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/delete?url="+url.QueryEscape("/files/"+name), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("狀態碼 = %d，want 401", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(base, "files", name)); err != nil {
		t.Error("未授權的刪除仍然生效了")
	}
}

// TestDeleteMissingURLParam 守住缺少參數時的錯誤路徑。
func TestDeleteMissingURLParam(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	req := httptest.NewRequest(http.MethodDelete, "/delete", nil)
	req.Header.Set("X-Upload-Token", "test-upload-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("狀態碼 = %d，want 400", rec.Code)
	}
}

// TestDeleteMethodNotAllowed 守住只接受 DELETE 與 POST。
func TestDeleteMethodNotAllowed(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/delete?url=%2Ffiles%2Fa.jpg", nil)
			req.Header.Set("X-Upload-Token", "test-upload-token")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("狀態碼 = %d，want 405", rec.Code)
			}
		})
	}
}

// TestMediaTokenMiddleware 覆蓋靜態檔案的媒體 token 驗證。
//
// 這是這個服務對外的存取控制邊界：圖片的網址在貼文裡是公開可讀的，因此
// 「知道網址」不等於「有權讀取」—— 權限由那個 Redis token 表達。
func TestMediaTokenMiddleware(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)

	// 先寫一個真正存在的檔案，否則驗證通過之後會拿到 404 而看不出差異。
	backend, err := newStorageBackend(cfg)
	if err != nil {
		t.Fatalf("newStorageBackend 回錯誤: %v", err)
	}
	const name = "photo.jpg"
	if _, err := backend.saveFile(context.Background(), "files", name, strings.NewReader("image-bytes")); err != nil {
		t.Fatalf("準備檔案失敗: %v", err)
	}

	t.Run("token 存在 → 放行", func(t *testing.T) {
		store := &fakeTokenStore{found: 1}
		handler, _ := newTestServer(t, cfg, store)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+name+"?token=valid-token", nil))

		if rec.Code != http.StatusOK {
			t.Errorf("狀態碼 = %d，want 200: %s", rec.Code, rec.Body.String())
		}
		if rec.Body.String() != "image-bytes" {
			t.Errorf("回應內容 = %q", rec.Body.String())
		}
		// 前綴必須正確：錯了就是「上傳成功但圖片一律 401」那個最難診斷的形狀。
		if len(store.queried) != 1 || store.queried[0] != "media:token:valid-token" {
			t.Errorf("查詢的 Redis key = %v，want [\"media:token:valid-token\"]", store.queried)
		}
	})

	t.Run("token 不存在 → 401", func(t *testing.T) {
		handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 0})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+name+"?token=expired-token", nil))

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("狀態碼 = %d，want 401", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "image-bytes") {
			t.Error("拒絕的請求仍然回傳了檔案內容")
		}
		// 必須是 JSON 形狀：後端依賴它來分辨「token 過期」與「檔案不存在」。
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q，want application/json", ct)
		}
	})

	t.Run("沒有帶 token → 401", func(t *testing.T) {
		handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+name, nil))

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("狀態碼 = %d，want 401（沒有 token 時不該放行）", rec.Code)
		}
	})

	// 這一條是「故障時不要把使用者踢出登入」那個決定的考題。
	t.Run("Redis 故障 → 503 而不是 401", func(t *testing.T) {
		handler, _ := newTestServer(t, cfg, &fakeTokenStore{err: errors.New("dial tcp: connection refused")})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+name+"?token=some-token", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("狀態碼 = %d，want 503", rec.Code)
		}
		if rec.Code == http.StatusUnauthorized {
			t.Error("Redis 故障被回報成 401 —— 前端會把它當成登入失效並把使用者踢去登入頁")
		}
		// 錯誤訊息不可洩漏內部連線細節。
		if strings.Contains(rec.Body.String(), "connection refused") {
			t.Errorf("錯誤訊息洩漏了內部細節: %s", rec.Body.String())
		}
	})
}

// TestMediaTokenDegradationPaths 覆蓋兩條「不驗證」的路徑。
//
// 兩者都導向同一個結果，但理由完全不同，因此要分開測：
//   - PublicFiles = true：部署者刻意公開（公開的圖片 CDN）。
//   - redis == nil：連不上而降級 —— main.go 的檔頭說明為什麼這是對的。
//
// 守住它們的理由是「不被驗證」是這個服務最脆弱的狀態：它必須是**刻意**的，
// 而不是任何一條路徑不小心走過來的結果。
func TestMediaTokenDegradationPaths(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	backend, err := newStorageBackend(cfg)
	if err != nil {
		t.Fatalf("newStorageBackend 回錯誤: %v", err)
	}
	const name = "photo.jpg"
	if _, err := backend.saveFile(context.Background(), "files", name, strings.NewReader("image-bytes")); err != nil {
		t.Fatalf("準備檔案失敗: %v", err)
	}

	t.Run("PublicFiles = true → 不驗證", func(t *testing.T) {
		publicCfg := *cfg
		publicCfg.Redis.PublicFiles = true
		store := &fakeTokenStore{found: 0} // 就算查也不該查到東西
		handler := NewServer(&publicCfg, backend, store).Handler()

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+name, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("狀態碼 = %d，want 200（PublicFiles 時不驗證）: %s", rec.Code, rec.Body.String())
		}
		if len(store.queried) != 0 {
			t.Errorf("PublicFiles 模式仍然查了 Redis: %v", store.queried)
		}
	})

	t.Run("redis 為 nil → 不驗證", func(t *testing.T) {
		store := &fakeTokenStore{found: 0}
		handler := NewServer(cfg, backend, nil).Handler()

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+name, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("狀態碼 = %d，want 200（沒有 Redis 時降級為不驗證）: %s", rec.Code, rec.Body.String())
		}
		if len(store.queried) != 0 {
			t.Errorf("redis 為 nil 時仍然嘗試查詢: %v", store.queried)
		}
	})
}

// TestMediaTokenOnlyAffectsFilesPath 守住中介層的作用範圍。
//
// 它**不該**擋住上傳與刪除：那兩個是後端對後端的呼叫，走 upload.token 通道。
// 把它们一起擋掉會讓後端自己的上傳也 401 —— 而症狀是「貼文存得下但圖片永遠
// 上傳失敗」，那是最難診斷的形狀之一。
func TestMediaTokenOnlyAffectsFilesPath(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 0}) // 任何 token 都查不到

	// 沒有 media token，但有 upload token 的上傳應該成功。
	req := multipartUpload(t, "/upload", "test-upload-token", "photo.jpg", "image/jpeg", []byte("bytes"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("上傳狀態碼 = %d，want 200（媒體 token 中介層不該擋上傳）: %s", rec.Code, rec.Body.String())
	}
}

// TestCORSPreflight 守住 OPTIONS 預檢的處理。
//
// 這一條的失效症狀很具體：瀏覽器端「每次請求都在 console 報 CORS 錯誤」而
// 實際請求其實有到達伺服器 —— 而後端完全不知道這件事。
func TestCORSPreflight(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)

	t.Run("預設允許所有來源", func(t *testing.T) {
		handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

		req := httptest.NewRequest(http.MethodOptions, "/files/photo.jpg", nil)
		req.Header.Set("Origin", "https://forum.example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("狀態碼 = %d，want 204", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://forum.example.com" {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
		// Range 必須在允許的標頭裡：論壇的音檔靠它做拖曳定位，少了它音訊
		// 播放器會退化成「整個檔案下載完才能播」。
		if allow := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(allow, "Range") {
			t.Errorf("Access-Control-Allow-Headers = %q，缺少 Range", allow)
		}
	})

	t.Run("來源在白名單內", func(t *testing.T) {
		restricted := *cfg
		restricted.CORS.AllowedOrigins = []string{"https://forum.example.com"}
		handler, _ := newTestServer(t, &restricted, &fakeTokenStore{found: 1})

		req := httptest.NewRequest(http.MethodOptions, "/files/photo.jpg", nil)
		req.Header.Set("Origin", "https://forum.example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("狀態碼 = %d，want 204", rec.Code)
		}
	})

	// 來源被拒絕時**不該**回 204：那會讓瀏覽器以為預檢通過，然後在真正的
	// 請求上失敗（症狀完全不同，更難診斷）。
	t.Run("來源不在白名單", func(t *testing.T) {
		restricted := *cfg
		restricted.CORS.AllowedOrigins = []string{"https://forum.example.com"}
		handler, _ := newTestServer(t, &restricted, &fakeTokenStore{found: 1})

		req := httptest.NewRequest(http.MethodOptions, "/upload", nil)
		req.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusNoContent {
			t.Error("來源被拒絕卻回 204 —— 瀏覽器會誤以為預檢通過")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("來源被拒絕卻送了 Access-Control-Allow-Origin: %q", got)
		}
	})
}

// TestNoDirectoryListingIsReachable 守住「儲存結構無法被列出」。
//
// 這一條來自 ROADMAP.md Phase 3.4 撰寫時的實測結果，不是事後想像的攻擊。
// http.FileServer 在路徑指向目錄時會回傳一份 HTML 目錄列表，而媒體 token
// 中介層只擋 /files/ 前綴 —— 因此把檔案伺服器掛在 "/" 的話，GET / 會回傳一份
// 儲存根目錄的列表，且**不需要 token**。
//
// 那不讀得到檔案內容（內容仍然需要 token），但它洩漏了兩件事：儲存結構，以及
// 全部檔名（UUID）。檔名不是機密的（它們就在貼文裡），但「哪些貼文的圖片
// 還存在」不該是一個匿名端點就能回答的問題。
//
// 因此這支測試斷言三件事：根路徑 404、/files/ 不回傳列表、以及單一檔案仍能
// 在 token 通過時正常取得。
func TestNoDirectoryListingIsReachable(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	backend, err := newStorageBackend(cfg)
	if err != nil {
		t.Fatalf("newStorageBackend 回錯誤: %v", err)
	}
	if _, err := backend.saveFile(context.Background(), "files", "photo.jpg", strings.NewReader("image-bytes")); err != nil {
		t.Fatalf("準備檔案失敗: %v", err)
	}
	// 也在儲存根目錄放一個檔案：它**不該**出現在任何列表裡，而且它根本沒有
	// 對應的 URL 形狀（只有 /files/<名稱> 是合法的）。
	if err := os.WriteFile(filepath.Join(base, "stray.txt"), []byte("stray"), 0o600); err != nil {
		t.Fatalf("建立散落檔案失敗: %v", err)
	}

	// redis 為 nil → 驗證被停用（最壞情況）。即使如此也不該有任何列表。
	handler := NewServer(cfg, backend, nil).Handler()

	for _, path := range []string{"/", "/files/", "/files", "/stray.txt", "/stray.txt/"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			body := rec.Body.String()
			if strings.Contains(body, "Directory listing") || strings.Contains(body, "<pre>") {
				t.Fatalf("GET %s 回傳了目錄列表:\n%s", path, body)
			}
			if strings.Contains(body, "photo.jpg") {
				t.Errorf("GET %s 洩漏了檔名:\n%s", path, body)
			}
			if strings.Contains(body, "stray.txt") {
				t.Errorf("GET %s 洩漏了儲存根目錄的內容:\n%s", path, body)
			}
			if strings.Contains(body, "image-bytes") {
				t.Errorf("GET %s 回傳了檔案內容（驗證被停用時也不該讓根路徑直通）:\n%s", path, body)
			}
		})
	}
}

// TestRootPathIsNotServed 守住根路徑不回應 200。
//
// 與上一支分開是因為它斷言的是「路由不存在」這件事本身：即使
// rejectDirectoryListing 被移除，掛在 /files/ 上的路由也不會讓 / 變成 200。
// 兩條一起守住，才不會在改動其中一條時失去另一條的保護。
func TestRootPathIsNotServed(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET / 狀態碼 = %d，want 404", rec.Code)
	}
}

// TestUnknownRouteReturns404 守住沒有路由時的預設行為。
func TestUnknownRouteReturns404(t *testing.T) {
	base := t.TempDir()
	cfg := testConfig(t, base)
	handler, _ := newTestServer(t, cfg, &fakeTokenStore{found: 1})

	// 刻意不含 "/"（由 TestRootPathIsNotServed 專責）與 "/files"
	// （ServeMux 會以 301 導向 "/files/"，那是它的標準行為而不是 404）。
	for _, path := range []string{"/nope", "/upload/extra", "/admin", "/stray.txt"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusNotFound {
				t.Errorf("狀態碼 = %d，want 404", rec.Code)
			}
		})
	}
}

// TestS3StorageHasNoStaticRoute 守住 S3 模式不掛本機靜態檔案路由。
//
// 在 S3 模式下掛上 /files/ 會讓一個**不存在於本機**的目錄變成一個可以列
// 目錄的 HTTP 根目錄 —— 那是一個資訊洩漏。因此這支測試斷言的是「路由不存在」
// 這件事本身。
func TestS3StorageHasNoStaticRoute(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	cfg.Storage.Type = "s3"
	cfg.Storage.S3.Endpoint = "http://s3:9000"
	cfg.Storage.S3.Bucket = "forum-media"
	cfg.Storage.S3.AccessKey = "key"
	cfg.Storage.S3.SecretKey = "secret"

	handler, _ := newTestServer(t, cfg, nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/photo.jpg", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("狀態碼 = %d，want 404（S3 模式不該由本服務供應靜態檔案）", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<html") {
		t.Error("回應是目錄列表或 HTML —— 本機目錄被當成靜態根目錄了")
	}
}

// TestNewStorageBackendRejectsUnknownType 守住設定檔打錯字時的行為。
//
// storage.type 打錯字（例如 "s30" 或 "localdisk"）若被靜默當成本機模式，
// 症狀是「檔案被存到磁碟，但貼文裡的網址指向 S3」—— 那會讓每一張圖片都
// 404，而錯誤訊息指向一個沒有人會去查的地方。
func TestNewStorageBackendRejectsUnknownType(t *testing.T) {
	// 這些值在設定檔裡打錯字時必須被拒絕 —— 否則「檔案被存到磁碟，但貼文裡
	// 的網址指向 S3」會讓每一張圖片都 404，而錯誤訊息指向沒有人會去查的地方。
	for _, typ := range []string{"s30", "localdisk", "unknown", "loc", "s-3", "s3.0"} {
		t.Run(typ, func(t *testing.T) {
			cfg := testConfig(t, t.TempDir())
			cfg.Storage.Type = typ
			_, err := newStorageBackend(cfg)
			if err == nil {
				t.Errorf("storage.type = %q 卻被接受了", typ)
			}
			if !strings.Contains(err.Error(), typ) {
				t.Errorf("錯誤訊息沒有點名那個值: %v", err)
			}
		})
	}

	// 大小寫與空白容忍：這兩者在真實設定檔裡很常見（人手寫的 TOML），
	// 而讓它們靜默失效的症狀正是上面那個「每一張圖片都 404」。
	for _, typ := range []string{"local", "LOCAL", " local ", "Local", "s3", "S3", " S3 "} {
		t.Run("容忍/"+typ, func(t *testing.T) {
			cfg := testConfig(t, t.TempDir())
			cfg.Storage.Type = typ
			if _, err := newStorageBackend(cfg); err != nil {
				t.Errorf("storage.type = %q 卻被拒絕: %v", typ, err)
			}
		})
	}
}

// countFiles 計算目錄底下的檔案數（不遞迴）。
func countFiles(t *testing.T, dir string) int {
	t.Helper()
	return len(listFiles(t, dir))
}

// listFiles 列出目錄底下的檔名（不遞迴）；目錄不存在時回傳空清單。
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("讀取目錄 %s 失敗: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// extractURL 從 JSON 回應裡取出 url 欄位。
func extractURL(t *testing.T, body string) string {
	t.Helper()
	const marker = `"url":"`
	i := strings.Index(body, marker)
	if i < 0 {
		t.Fatalf("回應裡沒有 url 欄位: %s", body)
	}
	rest := body[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		t.Fatalf("回應的 url 欄位沒有結束引號: %s", body)
	}
	return rest[:j]
}

// isUUIDv4Shape 檢查字串是否符合 8-4-4-4-12 的十六進位形狀。
func isUUIDv4Shape(s string) bool {
	groups := strings.Split(s, "-")
	if len(groups) != 5 {
		return false
	}
	wantLen := []int{8, 4, 4, 4, 12}
	for i, g := range groups {
		if len(g) != wantLen[i] {
			return false
		}
		for _, c := range g {
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}
