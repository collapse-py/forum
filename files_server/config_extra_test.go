/*
config_extra_test.go 補齊 config.go 的覆蓋：讀檔失敗、TOML 語法錯誤、空設定檔的
全部兜底值，以及「明確設定的值不會被兜底蓋掉」。

【為什麼需要這一支】
loadConfig 與 applyDefaults 原本是 0% 覆蓋。這兩個函式的失效方式是**靜默的**：
兜底值少了一項，服務照樣啟動，只是某個欄位變成零值 —— 例如 Upload.MaxSize 為 0
時上傳沒有上限，而沒有任何錯誤訊息。因此這裡逐項斷言預設值，而不是只驗「有沒有
回錯誤」。
*/
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfigFile 把內容寫進暫存目錄裡的設定檔並回傳路徑。
func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.conf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("寫入設定檔失敗: %v", err)
	}
	return path
}

// TestLoadConfigMissingFile 守住「設定檔不存在」是一條錯誤而不是空設定。
//
// 若這裡被改成回傳預設值，症狀是「設定檔路徑寫錯，服務照樣用預設值啟動」——
// 而部署者會以為自己的設定生效了。
func TestLoadConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.conf")
	cfg, err := loadConfig(path)
	if err == nil {
		t.Fatalf("讀取不存在的設定檔卻成功: %+v", cfg)
	}
	if !strings.Contains(err.Error(), "讀取配置失敗") {
		t.Errorf("錯誤訊息沒有說明是讀取失敗: %v", err)
	}
}

// TestLoadConfigMalformedTOML 守住 TOML 語法錯誤會回錯誤。
func TestLoadConfigMalformedTOML(t *testing.T) {
	path := writeConfigFile(t, "[server\nhost = \"broken\"\n")
	cfg, err := loadConfig(path)
	if err == nil {
		t.Fatalf("解析壞掉的 TOML 卻成功: %+v", cfg)
	}
	if !strings.Contains(err.Error(), "解析配置失敗") {
		t.Errorf("錯誤訊息沒有說明是解析失敗: %v", err)
	}
}

// TestLoadConfigEmptyFileAppliesAllDefaults 逐項釘住空設定檔的兜底值。
func TestLoadConfigEmptyFileAppliesAllDefaults(t *testing.T) {
	cfg, err := loadConfig(writeConfigFile(t, ""))
	if err != nil {
		t.Fatalf("解析空設定檔失敗: %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("server.host = %q，want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("server.port = %d，want 8080", cfg.Server.Port)
	}
	if cfg.Server.ShutdownTimeoutSec != 15 {
		t.Errorf("server.shutdown_timeout_seconds = %d，want 15", cfg.Server.ShutdownTimeoutSec)
	}

	if cfg.Storage.Type != "local" {
		t.Errorf("storage.type = %q，want local", cfg.Storage.Type)
	}
	if cfg.Storage.Local.BaseDir != "./storage" {
		t.Errorf("storage.local.base_dir = %q，want ./storage", cfg.Storage.Local.BaseDir)
	}
	if cfg.Storage.Local.FilesDir != "files" {
		t.Errorf("storage.local.files_dir = %q，want files", cfg.Storage.Local.FilesDir)
	}
	if cfg.Storage.Local.DirPerm != 0o755 {
		t.Errorf("storage.local.dir_perm = %o，want 755", cfg.Storage.Local.DirPerm)
	}
	if cfg.Storage.Local.FilePerm != 0o644 {
		t.Errorf("storage.local.file_perm = %o，want 644", cfg.Storage.Local.FilePerm)
	}
	if cfg.Storage.S3.PresignExpireSec != 3600 {
		t.Errorf("storage.s3.presign_expire_seconds = %d，want 3600", cfg.Storage.S3.PresignExpireSec)
	}

	// 上傳上限必須是有限值：0 等於「不設上限」，那是一個資源耗盡的入口。
	if cfg.Upload.MaxSize != 50<<20 {
		t.Errorf("upload.max_size = %d，want %d", cfg.Upload.MaxSize, int64(50<<20))
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".opus"} {
		if !isAllowedExt(ext, cfg.Upload.AllowedFiles) {
			t.Errorf("預設 allowed_files 缺少 %s", ext)
		}
	}

	if cfg.Logger.Level != "info" {
		t.Errorf("logger.level = %q，want info", cfg.Logger.Level)
	}

	if !originAllowed("https://anything.example", cfg.CORS.AllowedOrigins) {
		t.Errorf("預設 CORS 沒有放行所有來源: %v", cfg.CORS.AllowedOrigins)
	}
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "DELETE"} {
		if !containsString(cfg.CORS.AllowedMethods, method) {
			t.Errorf("預設 allowed_methods 缺少 %s", method)
		}
	}
	for _, header := range []string{"Origin", "Range", "Accept", "Accept-Language", "Content-Type", "X-Upload-Token", "Authorization"} {
		if !containsString(cfg.CORS.AllowedHeaders, header) {
			t.Errorf("預設 allowed_headers 缺少 %s", header)
		}
	}

	if cfg.RateLimit.MediaRate != 50 {
		t.Errorf("ratelimit.media_requests_per_sec = %v，want 50", cfg.RateLimit.MediaRate)
	}
	if cfg.RateLimit.MediaBurst != 200 {
		t.Errorf("ratelimit.media_burst = %d，want 200", cfg.RateLimit.MediaBurst)
	}
	if cfg.RateLimit.WriteRate != 2 {
		t.Errorf("ratelimit.write_requests_per_sec = %v，want 2", cfg.RateLimit.WriteRate)
	}
	if cfg.RateLimit.WriteBurst != 20 {
		t.Errorf("ratelimit.write_burst = %d，want 20", cfg.RateLimit.WriteBurst)
	}
	// enabled 的零值是 false，而刻意不把它兜底成 true：關掉限流必須是一個
	// 明確寫出來的決定。
	if cfg.RateLimit.Enabled {
		t.Error("空設定檔把 ratelimit.enabled 變成了 true")
	}

	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("redis.addr = %q，want localhost:6379", cfg.Redis.Addr)
	}
	if cfg.Redis.TokenKeyPrefix != "forum:token:" {
		t.Errorf("redis.token_key_prefix = %q，want forum:token:", cfg.Redis.TokenKeyPrefix)
	}
	// 0 不能被當成「永不過期」。
	if cfg.Redis.TokenTTLSec != 300 {
		t.Errorf("redis.token_ttl_sec = %d，want 300", cfg.Redis.TokenTTLSec)
	}
	if cfg.Redis.PublicFiles {
		t.Error("空設定檔把 redis.public_files 變成了 true")
	}
}

// TestLoadConfigReadsEverySection 守住 TOML 欄位到結構體的對應。
//
// 少一個 tag 的症狀是「設定檔寫了但沒生效」，而 applyDefaults 會把該欄位補成
// 預設值 —— 從日誌完全看不出來。
func TestLoadConfigReadsEverySection(t *testing.T) {
	const content = `
[server]
host = "127.0.0.1"
port = 1234
shutdown_timeout_seconds = 7

[storage]
type = "s3"

[storage.local]
base_dir = "/var/lib/forum"
files_dir = "media"
dir_perm = 488
file_perm = 384

[storage.s3]
endpoint = "http://s3:9000"
access_key = "access"
secret_key = "secret"
region = "us-east-1"
bucket = "forum-media"
force_path_style = true
files_prefix = "uploads/"
presign_expire_seconds = 120

[upload]
token = "configured-token"
max_size = 1048576
allowed_files = [".png", ".webp"]

[cors]
allowed_origins = ["https://forum.example.com"]
allowed_methods = ["GET", "POST"]
allowed_headers = ["Range", "X-Upload-Token"]

[logger]
level = "debug"
enable_colors = true

[redis]
addr = "127.0.0.1:6380"
password = "redis-password"
db = 3
token_key_prefix = "custom:token:"
token_ttl_sec = 60
public_files = true

[ratelimit]
enabled = true
media_requests_per_sec = 5.5
media_burst = 7
write_requests_per_sec = 1.5
write_burst = 2
`
	cfg, err := loadConfig(writeConfigFile(t, content))
	if err != nil {
		t.Fatalf("解析設定檔失敗: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 1234 || cfg.Server.ShutdownTimeoutSec != 7 {
		t.Errorf("server = %+v", cfg.Server)
	}
	if cfg.Storage.Type != "s3" {
		t.Errorf("storage.type = %q", cfg.Storage.Type)
	}
	local := cfg.Storage.Local
	if local.BaseDir != "/var/lib/forum" || local.FilesDir != "media" || local.DirPerm != 0o750 || local.FilePerm != 0o600 {
		t.Errorf("storage.local = %+v", local)
	}
	s3 := cfg.Storage.S3
	if s3.Endpoint != "http://s3:9000" || s3.AccessKey != "access" || s3.SecretKey != "secret" {
		t.Errorf("storage.s3 認證欄位 = %+v", s3)
	}
	if s3.Region != "us-east-1" || s3.Bucket != "forum-media" || !s3.ForcePathStyle {
		t.Errorf("storage.s3 位置欄位 = %+v", s3)
	}
	if s3.FilesPrefix != "uploads/" || s3.PresignExpireSec != 120 {
		t.Errorf("storage.s3 前綴/期限 = %+v", s3)
	}

	if cfg.Upload.Token != "configured-token" || cfg.Upload.MaxSize != 1048576 {
		t.Errorf("upload = %+v", cfg.Upload)
	}
	if len(cfg.Upload.AllowedFiles) != 2 || cfg.Upload.AllowedFiles[0] != ".png" || cfg.Upload.AllowedFiles[1] != ".webp" {
		t.Errorf("upload.allowed_files = %v", cfg.Upload.AllowedFiles)
	}

	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "https://forum.example.com" {
		t.Errorf("cors.allowed_origins = %v", cfg.CORS.AllowedOrigins)
	}
	if len(cfg.CORS.AllowedMethods) != 2 || cfg.CORS.AllowedMethods[1] != "POST" {
		t.Errorf("cors.allowed_methods = %v", cfg.CORS.AllowedMethods)
	}
	if len(cfg.CORS.AllowedHeaders) != 2 || cfg.CORS.AllowedHeaders[0] != "Range" {
		t.Errorf("cors.allowed_headers = %v", cfg.CORS.AllowedHeaders)
	}

	if cfg.Logger.Level != "debug" || !cfg.Logger.EnableColors {
		t.Errorf("logger = %+v", cfg.Logger)
	}

	redisCfg := cfg.Redis
	if redisCfg.Addr != "127.0.0.1:6380" || redisCfg.Password != "redis-password" || redisCfg.DB != 3 {
		t.Errorf("redis 連線欄位 = %+v", redisCfg)
	}
	if redisCfg.TokenKeyPrefix != "custom:token:" || redisCfg.TokenTTLSec != 60 || !redisCfg.PublicFiles {
		t.Errorf("redis token 欄位 = %+v", redisCfg)
	}

	rl := cfg.RateLimit
	if !rl.Enabled || rl.MediaRate != 5.5 || rl.MediaBurst != 7 || rl.WriteRate != 1.5 || rl.WriteBurst != 2 {
		t.Errorf("ratelimit = %+v", rl)
	}
}

// TestApplyDefaultsPreservesExplicitValues 守住兜底值不會覆蓋掉明確設定。
//
// 這一支直接呼叫 applyDefaults（而不是經過 TOML），因為要驗的就是「非零值不動」
// 這件事本身。最關鍵的是 MaxSize、TokenTTLSec 與 shutdown_timeout：它們的 0 值
// 都有安全或可用性上的意義。
func TestApplyDefaultsPreservesExplicitValues(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Host: "10.0.0.5", Port: 9090, ShutdownTimeoutSec: 1},
		Storage: StorageConfig{
			Type: "s3",
			Local: LocalStorageConfig{
				BaseDir:  "/data",
				FilesDir: "media",
				DirPerm:  0o700,
				FilePerm: 0o600,
			},
			S3: S3StorageConfig{PresignExpireSec: 30},
		},
		Upload: UploadConfig{
			Token:        "tok",
			MaxSize:      1,
			AllowedFiles: []string{".png"},
		},
		CORS: CORSPolicy{
			AllowedOrigins: []string{"https://only.example"},
			AllowedMethods: []string{"GET"},
			AllowedHeaders: []string{"Range"},
		},
		Logger: LoggerConfig{Level: "error", EnableColors: true},
		Redis: RedisConfig{
			Addr:           "redis:6379",
			Password:       "pw",
			DB:             9,
			TokenKeyPrefix: "custom:",
			TokenTTLSec:    1,
			PublicFiles:    true,
		},
		RateLimit: RateLimitConfig{
			Enabled:    true,
			MediaRate:  1,
			MediaBurst: 2,
			WriteRate:  3,
			WriteBurst: 4,
		},
	}

	cfg.applyDefaults()

	if cfg.Server.Host != "10.0.0.5" || cfg.Server.Port != 9090 || cfg.Server.ShutdownTimeoutSec != 1 {
		t.Errorf("server 被改寫: %+v", cfg.Server)
	}
	if cfg.Storage.Type != "s3" || cfg.Storage.Local.BaseDir != "/data" || cfg.Storage.Local.FilesDir != "media" {
		t.Errorf("storage 被改寫: %+v", cfg.Storage)
	}
	if cfg.Storage.Local.DirPerm != 0o700 || cfg.Storage.Local.FilePerm != 0o600 {
		t.Errorf("storage.local 權限被改寫: %+v", cfg.Storage.Local)
	}
	if cfg.Storage.S3.PresignExpireSec != 30 {
		t.Errorf("storage.s3.presign_expire_seconds 被改寫: %d", cfg.Storage.S3.PresignExpireSec)
	}
	if cfg.Upload.Token != "tok" || cfg.Upload.MaxSize != 1 || len(cfg.Upload.AllowedFiles) != 1 {
		t.Errorf("upload 被改寫: %+v", cfg.Upload)
	}
	if cfg.CORS.AllowedOrigins[0] != "https://only.example" || cfg.CORS.AllowedMethods[0] != "GET" || cfg.CORS.AllowedHeaders[0] != "Range" {
		t.Errorf("cors 被改寫: %+v", cfg.CORS)
	}
	if cfg.Logger.Level != "error" || !cfg.Logger.EnableColors {
		t.Errorf("logger 被改寫: %+v", cfg.Logger)
	}
	if cfg.Redis.Addr != "redis:6379" || cfg.Redis.DB != 9 || cfg.Redis.TokenKeyPrefix != "custom:" || cfg.Redis.TokenTTLSec != 1 || !cfg.Redis.PublicFiles {
		t.Errorf("redis 被改寫: %+v", cfg.Redis)
	}
	if !cfg.RateLimit.Enabled || cfg.RateLimit.MediaRate != 1 || cfg.RateLimit.MediaBurst != 2 || cfg.RateLimit.WriteRate != 3 || cfg.RateLimit.WriteBurst != 4 {
		t.Errorf("ratelimit 被改寫: %+v", cfg.RateLimit)
	}
}

// containsString 是 slice 的小工具；不引入 slices 是為了讓測試的相依與
// 生產程式一致（只用標準函式庫，且不挑新加入的套件）。
func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
