package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/redis/go-redis/v9"
)

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type LocalStorageConfig struct {
	BaseDir  string      `toml:"base_dir"`
	FilesDir string      `toml:"files_dir"`
	DirPerm  os.FileMode `toml:"dir_perm"`
	FilePerm os.FileMode `toml:"file_perm"`
}

type S3StorageConfig struct {
	Endpoint         string `toml:"endpoint"`
	AccessKey        string `toml:"access_key"`
	SecretKey        string `toml:"secret_key"`
	Region           string `toml:"region"`
	Bucket           string `toml:"bucket"`
	ForcePathStyle   bool   `toml:"force_path_style"`
	FilesPrefix      string `toml:"files_prefix"`
	PresignExpireSec int    `toml:"presign_expire_seconds"`
}

type StorageConfig struct {
	Type  string             `toml:"type"`
	Local LocalStorageConfig `toml:"local"`
	S3    S3StorageConfig    `toml:"s3"`
}

type UploadConfig struct {
	Token        string   `toml:"token"`
	MaxSize      int64    `toml:"max_size"`
	AllowedFiles []string `toml:"allowed_files"`
}

type CORSPolicy struct {
	AllowedOrigins []string `toml:"allowed_origins"`
	AllowedMethods []string `toml:"allowed_methods"`
	AllowedHeaders []string `toml:"allowed_headers"`
}

type LoggerConfig struct {
	Level        string `toml:"level"`
	EnableColors bool   `toml:"enable_colors"`
}

type RedisConfig struct {
	Addr           string `toml:"addr"`
	Password       string `toml:"password"`
	DB             int    `toml:"db"`
	TokenKeyPrefix string `toml:"token_key_prefix"`
	TokenTTLSec    int    `toml:"token_ttl_sec"`
	PublicFiles    bool   `toml:"public_files"`
}

type Config struct {
	Server  ServerConfig  `toml:"server"`
	Storage StorageConfig `toml:"storage"`
	Upload  UploadConfig  `toml:"upload"`
	CORS    CORSPolicy    `toml:"cors"`
	Logger  LoggerConfig  `toml:"logger"`
	Redis   RedisConfig   `toml:"redis"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取配置失敗: %w", err)
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, fmt.Errorf("解析配置失敗: %w", err)
	}

	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "local"
	}
	if cfg.Storage.Local.BaseDir == "" {
		cfg.Storage.Local.BaseDir = "./storage"
	}
	if cfg.Storage.Local.FilesDir == "" {
		cfg.Storage.Local.FilesDir = "files"
	}
	if cfg.Storage.Local.DirPerm == 0 {
		cfg.Storage.Local.DirPerm = 0755
	}
	if cfg.Storage.Local.FilePerm == 0 {
		cfg.Storage.Local.FilePerm = 0644
	}
	if cfg.Storage.S3.PresignExpireSec == 0 {
		cfg.Storage.S3.PresignExpireSec = 3600
	}
	if cfg.Upload.MaxSize == 0 {
		cfg.Upload.MaxSize = 50 << 20
	}
	if len(cfg.Upload.AllowedFiles) == 0 {
		cfg.Upload.AllowedFiles = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".opus"}
	}
	if cfg.Logger.Level == "" {
		cfg.Logger.Level = "info"
	}
	if len(cfg.CORS.AllowedOrigins) == 0 {
		cfg.CORS.AllowedOrigins = []string{"*"}
	}
	if len(cfg.CORS.AllowedMethods) == 0 {
		cfg.CORS.AllowedMethods = []string{"GET", "HEAD", "OPTIONS"}
	}
	if len(cfg.CORS.AllowedHeaders) == 0 {
		cfg.CORS.AllowedHeaders = []string{"Origin", "Range", "Accept", "Accept-Language"}
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}
	if cfg.Redis.DB == 0 {
		cfg.Redis.DB = 0
	}
	if cfg.Redis.TokenKeyPrefix == "" {
		cfg.Redis.TokenKeyPrefix = "media:token:"
	}
	if cfg.Redis.TokenTTLSec == 0 {
		cfg.Redis.TokenTTLSec = 300
	}

	return &cfg, nil
}

type storageBackend interface {
	saveFile(dir, filename string, data io.Reader) (string, error)
	deleteFile(path string) error
	getBaseURL() string
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
		os.Remove(dstPath)
		return "", err
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

func isAllowedExt(ext string, allowed []string) bool {
	ext = strings.ToLower(ext)
	for _, a := range allowed {
		if strings.ToLower(a) == ext {
			return true
		}
	}
	return false
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

func resolveDir(filename string, allowed []string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if isAllowedExt(ext, allowed) {
		return "files"
	}
	return ""
}

func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

func addCORSHeaders(w http.ResponseWriter, r *http.Request, cfg *Config) {
	origin := r.Header.Get("Origin")
	if originAllowed(origin, cfg.CORS.AllowedOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		if len(cfg.CORS.AllowedMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.CORS.AllowedMethods, ", "))
		}
		if len(cfg.CORS.AllowedHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.CORS.AllowedHeaders, ", "))
		}
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Content-Length")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
}

func corsMiddleware(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			addCORSHeaders(w, r, cfg)
			if r.Method == http.MethodOptions {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

var redisClient *redis.Client

func newRedisClient(cfg *Config) *redis.Client {
	if cfg.Redis.Addr == "" {
		return nil
	}
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

func mediaTokenKey(cfg *Config, token string) string {
	return cfg.Redis.TokenKeyPrefix + token
}

func validUploadToken(cfg *Config, r *http.Request) bool {
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.Header.Get("X-Upload-Token")
	}
	return cfg.Upload.Token == "" || token == cfg.Upload.Token || token == "Bearer "+cfg.Upload.Token
}

func mediaTokenMiddleware(cfg *Config, next http.Handler) http.Handler {
	if cfg.Redis.PublicFiles || cfg.Redis.Addr == "" || redisClient == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/files/") {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"missing media token"}`)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		key := mediaTokenKey(cfg, token)
		exists, err := redisClient.Exists(ctx, key).Result()
		if err != nil {
			log.Printf("Redis token check failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"error":"media token service unavailable"}`)
			return
		}

		if exists == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid or expired media token"}`)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg, err := loadConfig("config.conf")
	if err != nil {
		log.Fatalf("載入配置失敗: %v", err)
	}

	backend, err := newStorageBackend(cfg)
	if err != nil {
		log.Fatalf("初始化存儲失敗: %v", err)
	}

	redisClient = newRedisClient(cfg)
	if redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Printf("Redis 連線失敗，將跳過靜態檔案 token 驗證: %v", err)
			redisClient = nil
		} else {
			log.Printf("Redis 連線成功 addr=%s db=%d", cfg.Redis.Addr, cfg.Redis.DB)
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !validUploadToken(cfg, r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, cfg.Upload.MaxSize)

		if err := r.ParseMultipartForm(cfg.Upload.MaxSize); err != nil {
			http.Error(w, "Unable to parse form or file exceeds size limit", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Unable to get file from request", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 副檔名決定目錄與 Content-Type，仍以「上傳者給的檔名」為準判斷；
		// 但實際寫進磁碟的檔名是 UUID（見 newFileName），兩者不可混為一談。
		original := filepath.Base(handler.Filename)
		dir := resolveDir(original, cfg.Upload.AllowedFiles)
		if dir == "" {
			http.Error(w, "Unsupported file type", http.StatusBadRequest)
			return
		}

		filename, err := newFileName(original)
		if err != nil {
			log.Printf("產生檔名失敗: %v", err)
			http.Error(w, "Unable to save file", http.StatusInternalServerError)
			return
		}

		rel, err := backend.saveFile(dir, filename, file)
		if err != nil {
			log.Printf("儲存檔案失敗: %v", err)
			http.Error(w, "Unable to save file", http.StatusInternalServerError)
			return
		}

		fileURL := backend.getBaseURL() + rel
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"url":"%s"}`, fileURL)
	})

	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !validUploadToken(cfg, r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		fileURL := r.URL.Query().Get("url")
		if fileURL == "" && r.Method == http.MethodPost {
			_ = r.ParseForm()
			fileURL = r.Form.Get("url")
		}

		if fileURL == "" {
			http.Error(w, "Missing url parameter", http.StatusBadRequest)
			return
		}

		u, err := url.Parse(fileURL)
		if err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		dir := parts[len(parts)-2]
		filename := filepath.Base(parts[len(parts)-1])

		if dir != "files" {
			http.Error(w, "Invalid directory in URL", http.StatusBadRequest)
			return
		}

		rel := "/" + dir + "/" + filename
		if err := backend.deleteFile(rel); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			log.Printf("刪除檔案失敗: %v", err)
			http.Error(w, "Unable to delete file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"deleted"}`)
	})

	if cfg.Storage.Type == "local" {
		base := backend.(*localStorage).baseDir
		staticHandler := http.StripPrefix("/", http.FileServer(http.Dir(base)))
		mux.Handle("/", mediaTokenMiddleware(cfg, staticHandler))
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting files server on %s (storage=%s)", addr, cfg.Storage.Type)
	if cfg.Upload.Token != "" {
		log.Println("Upload token authentication enabled")
	}
	if redisClient != nil {
		log.Println("Static file media token authentication via Redis enabled")
	}
	log.Fatal(http.ListenAndServe(addr, corsMiddleware(cfg)(mux)))
}
