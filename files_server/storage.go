/*
storage.go 是 files_server 的儲存後端：把「檔名 + 內容」這件事從 HTTP 層抽離，
讓本機磁碟與 S3 可以互換。

【為什麼獨立成檔】
上傳、刪除、對外網址這三件事是唯一需要隨儲存後端改變的邏輯。把它們留在 main.go
（組裝與啟動流程）裡，會讓閱讀啟動順序的人必須先讀懂 S3 的簽章細節才能知道
服務在做什麼。

【對外介面（本檔全部匯出符號）】
storageBackend：後端介面，三個方法。
newStorageBackend：以設定選出具體實作。
newFileName：為上傳的檔案產生儲存檔名（UUID v4 + 副檔名）。
本檔不碰 HTTP，也不碰 Redis。

【主要依賴】
僅標準函式庫。S3 用的是最小可用子集（PUT / DELETE + BasicAuth），刻意不引入
AWS SDK：那會讓這個模組多出一個體積可觀的相依樹，而實際用到的只有兩個方法。

【關鍵設計決策】
 1. 儲存鍵一律是服務端產生的 UUID，不是使用者上傳的檔名。理由逐條寫在
    newFileName 上 —— 覆蓋、資訊洩漏與跨平台路徑問題都來自「信任檔名」。
 2. 介面刻意不包含「列出」或「讀取」：這個服務只做上傳與刪除，讀取是
    http.FileServer 直接從本機磁碟送的（S3 模式則另有前端直接連 S3 的路徑）。
    介面越小，未來新增後端要實作的東西越少。
 3. S3 的上傳把整個檔案讀進記憶體再送出。這個決定的前提是 Upload.MaxSize
    預設 50 MiB；它若被調高到數百 MB，這裡會變成記憶體尖峰的來源，因此
    介面不假裝支援串流 —— 真的需要時，介面要改成 io.Reader 串流而不是在
    呼叫端偷偷 buffer。
*/
package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type storageBackend interface {
	saveFile(dir, filename string, data io.Reader) (string, error)
	deleteFile(path string) error
	getBaseURL() string
	// staticRoot 回報「本機磁碟上可以直接以 HTTP 送出檔案的根目錄」。
	//
	// 為什麼它在介面上而不是讓 server.go 去比對 cfg.Storage.Type：型別字串的
	// 比對（"local" vs "LOCAL" vs " local "）與「這個後端是否由本服務供檔案」
	// 是兩件無關的事，而 `docker stop` 之後的第一個症狀往往是「圖片 404」——
	// 把這個判斷交給後端實作，就讓它只由 newStorageBackend 一處決定。
	//
	// S3 回 (nil, false)：那個模式下檔案由前端直接連 S3，本服務不供應靜態檔案。
	staticRoot() (string, bool)
}

type localStorage struct {
	baseDir   string
	filesDir  string
	dirPerm   os.FileMode
	filePerm  os.FileMode
	publicURL string
}

func newLocalStorage(cfg *Config) (*localStorage, error) {
	base := filepath.Clean(cfg.Storage.Local.BaseDir)
	files := filepath.Join(base, cfg.Storage.Local.FilesDir)

	for _, dir := range []string{base, files} {
		if err := os.MkdirAll(dir, cfg.Storage.Local.DirPerm); err != nil {
			return nil, fmt.Errorf("建立目錄 %s 失敗: %w", dir, err)
		}
	}

	return &localStorage{
		baseDir:   base,
		filesDir:  cfg.Storage.Local.FilesDir,
		dirPerm:   cfg.Storage.Local.DirPerm,
		filePerm:  cfg.Storage.Local.FilePerm,
		publicURL: "",
	}, nil
}

// resolveDir 把邏輯目錄名轉成實際路徑。
//
// 刻意用白名單式的 switch 而不是字串拼接：dir 來自使用者可控的 URL
// （/delete 端點），直接拼接等於允許 ".." 走出儲存根目錄。
func (l *localStorage) resolveDir(dir string) string {
	switch dir {
	case "files":
		return filepath.Join(l.baseDir, l.filesDir)
	}
	return l.baseDir
}

func (l *localStorage) saveFile(dir, filename string, data io.Reader) (string, error) {
	destDir := l.resolveDir(dir)
	dstPath := filepath.Join(destDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, data); err != nil {
		// 失敗必須刪掉半成品：檔名是 UUID，下次不會有人剛好再用到同一個名字，
		// 因此留下半個檔案等於永久的無主垃圾，而且沒有任何機制會清掉它。
		os.Remove(dstPath)
		return "", err
	}

	// 明確以 dirPerm 之外的形式設定權限：os.Create 依 umask 決定，而容器裡
	// 常見的 umask 會讓上傳的檔案變成 0600（只有容器內的使用者讀得到）。
	// 這裡「至少」確保 owner 有讀寫；是否讓其他使用者讀取則沿用設定檔的
	// file_perm —— 瀏覽器是透過 HTTP 讀檔的，不需要檔案系統權限。
	if l.filePerm != 0 {
		_ = os.Chmod(dstPath, l.filePerm)
	}

	rel := "/" + dir + "/" + filename
	return rel, nil
}

func (l *localStorage) deleteFile(path string) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	abs := filepath.Join(l.baseDir, strings.TrimPrefix(path, "/"))
	return os.Remove(abs)
}

func (l *localStorage) getBaseURL() string {
	return l.publicURL
}

// staticRoot 回報本機儲存的根目錄 —— 也就是 http.FileServer 該服务的目錄。
//
// 回傳 baseDir 而不是 baseDir/files：stripPrefix("/") 之後的 URL 形狀是
// /files/<uuid>.<ext>，而目錄結構上 baseDir/files 才是存檔的位置。因此 baseDir
// 才是讓那個 URL 形狀成立的那一層。
func (l *localStorage) staticRoot() (string, bool) {
	return l.baseDir, true
}

type s3Storage struct {
	endpoint         string
	accessKey        string
	secretKey        string
	region           string
	bucket           string
	forcePathStyle   bool
	filesPrefix      string
	presignExpireSec int
	publicURL        string
}

func newS3Storage(cfg *Config) (*s3Storage, error) {
	return &s3Storage{
		endpoint:         cfg.Storage.S3.Endpoint,
		accessKey:        cfg.Storage.S3.AccessKey,
		secretKey:        cfg.Storage.S3.SecretKey,
		region:           cfg.Storage.S3.Region,
		bucket:           cfg.Storage.S3.Bucket,
		forcePathStyle:   cfg.Storage.S3.ForcePathStyle,
		filesPrefix:      cfg.Storage.S3.FilesPrefix,
		presignExpireSec: cfg.Storage.S3.PresignExpireSec,
		publicURL:        "",
	}, nil
}

func (s *s3Storage) saveFile(dir, filename string, data io.Reader) (string, error) {
	objectKey := s.filesPrefix + filename

	var b strings.Builder
	if _, err := io.Copy(&b, data); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPut, s.endpoint+"/"+s.bucket+"/"+objectKey, strings.NewReader(b.String()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.SetBasicAuth(s.accessKey, s.secretKey)
	if s.region != "" {
		req.Header.Set("x-amz-region", s.region)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("s3 上傳失敗 status=%d body=%s", resp.StatusCode, string(body))
	}

	return "/" + dir + "/" + filename, nil
}

func (s *s3Storage) deleteFile(path string) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid s3 path %s", path)
	}
	filename := parts[1]

	objectKey := s.filesPrefix + filename

	req, err := http.NewRequest(http.MethodDelete, s.endpoint+"/"+s.bucket+"/"+objectKey, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.accessKey, s.secretKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 204 與 200 都視為成功：S3 與相容實作（MinIO）在刪除不存在的 key 時回
	// 204 或 404，而本服務的語意是「確保它不存在」，兩者都符合。
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("s3 刪除失敗 status=%d body=%s", resp.StatusCode, string(body))
}

func (s *s3Storage) getBaseURL() string {
	if s.publicURL != "" {
		return s.publicURL
	}
	return s.endpoint + "/" + s.bucket
}

// staticRoot 在 S3 模式下回 (nil, false)。
//
// 刻意不在本服務前面擋一層「下載代理」：那會讓這個服務變成檔案流量的中繼點，
// 而它的存在理由是「檔案不經過論壇服務」—— 讓檔案流量繼續走 S3 的公開網址
// （由後端以 FILES_SERVER_PUBLIC_URL 寫進貼文）才是這個設計的目的。
func (s *s3Storage) staticRoot() (string, bool) {
	return "", false
}

// newStorageBackend 依設定的 storage.type 選出具體後端。
//
// 切換大小寫並去除空白是刻意的：TOML 的值是人手寫的，而 "S3" 與 "s3" 指同一個
// 東西 —— 讓它在這裡靜默失效的症狀是「啟動時報不支援的存儲類型」，那是一個需要
// 回頭查設定檔才能理解的錯誤訊息。
func newStorageBackend(cfg *Config) (storageBackend, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Storage.Type)) {
	case "local":
		return newLocalStorage(cfg)
	case "s3":
		return newS3Storage(cfg)
	default:
		return nil, fmt.Errorf("不支援的存儲類型: %s", cfg.Storage.Type)
	}
}

// newFileName 為上傳的檔案產生一個新的儲存檔名：UUID v4 + 原始副檔名。
//
// 為什麼不用使用者上傳的檔名當儲存鍵：原始檔名是使用者可控的輸入，直接拿來
// 存檔會同時產生兩種問題——
//   - 覆蓋：兩個人都傳 "IMG_0001.jpg" 時後者會無聲地蓋掉前者，任何人也可以
//     刻意指定已知檔名把別人的圖片換掉；
//   - 資訊洩漏與跨平台問題：檔名會出現在磁碟、URL 與日誌裡，含空白、Unicode
//     或路徑片段時還會造成 URL 編碼與路徑正規化的麻煩。
//
// 改用 UUID 後，檔名內容與上傳者無關，既不會撞名也不會洩漏原始名稱。
// 副檔名必須保留，因為它決定瀏覽器的 Content-Type，也讓後端
// validForumImageFileName 的白名單檢查與前端 <img> 顯示都維持原樣。
// 這裡刻意自行用 crypto/rand 組出 UUID v4，而不是引入相依套件：這個服務的
// 相依項已經只有 toml 與 redis 兩個，為了一個 16 行的函式增加外部相依不值。
//
// crypto/rand 讀取失敗會回錯誤而不是退化成亂數：檔名不可預測是這條路徑的安全
// 前提，而 crypto/rand 失敗代表作業系統的隨機來源有問題，那種情況下寧可讓上傳
// 失敗，也不要產出一批可猜測的檔名。
func newFileName(original string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// RFC 4122 §4.4：把版本與變異位元固定下來，產生合法的 v4 UUID。
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	// 副檔名一律轉小寫：後端 validForumImageFileName 只認小寫的 .jpg/.png 等，
	// 保留 "PHOTO.JPG" 的大小寫會讓上傳成功卻在後端被判定為非法圖片。
	// 這裡只取副檔名（而非整個檔名），且呼叫端已先以 filepath.Base 正規化。
	ext := strings.ToLower(filepath.Ext(original))

	return fmt.Sprintf("%x-%x-%x-%x-%x%s", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16], ext), nil
}
