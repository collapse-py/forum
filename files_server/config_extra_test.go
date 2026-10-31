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

	// 預設的副檔名白名單**只有圖片**。這一份清單必須與後端的 validForumImageFileName
	// 相同（見 config.go 那一處的理由）；音訊在預設值裡是刻意**不**存在的：
	// 那六種副檔名在後端永遠不認得，所以「能存、掛不上、也刪不掉」。
	// 這一支斷言把那個方向釘住 —— 若哪天有人把音訊加回預設值，這裡會紅并要求
	// 同時在 docs/KNOWN_ISSUES.md 裡寫出「後端認得它們嗎」。
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		if !isAllowedExt(ext, cfg.Upload.AllowedFiles) {
			t.Errorf("預設 allowed_files 缺少 %s", ext)
		}
	}
	for _, ext := range []string{".mp3", ".wav", ".ogg", ".m4a", ".flac", ".opus", ".svg", ".html"} {
		if isAllowedExt(ext, cfg.Upload.AllowedFiles) {
			t.Errorf("預設 allowed_files 多了 %s —— 後端不認得它，那是一個存得下但永遠用不到的檔案", ext)
		}
	}

	// 圖片正規化的兜底值必須是「有限」的那一個：0 在這四個欄位上都不是「不設上限」，
	// 而是「沒設定」。特別是 jpeg_quality：0 會被 jpeg.Encode 夾到 **1**（最差的那一個），
	// 因此一個「沒設定」的部署會產出品質 1 的圖 —— 沒有任何錯誤。
	p := cfg.imagePolicy()
	if !p.Reencode {
		t.Error("空設定檔把圖片正規化關掉了 —— 那個預設值應該是「開」（EXIF 的 GPS 會被移除）")
	}
	if p.MaxWidth != 2048 || p.MaxHeight != 2048 {
		t.Errorf("輸出尺寸上限 = %dx%d，want 2048x2048", p.MaxWidth, p.MaxHeight)
	}
	if p.MaxSourcePixels != 24_000_000 {
		t.Errorf("來源解析度上限 = %d，want 24000000（那是解壓炸彈的記憶體上限）", p.MaxSourcePixels)
	}
	if p.JPEGQuality < 50 || p.JPEGQuality > 100 {
		t.Errorf("JPEG 品質 = %d，不在 50–100 這個「看得出差異」的區間裡", p.JPEGQuality)
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
reencode = false
max_width = 1600
max_height = 1200
max_source_pixels = 8000000
jpeg_quality = 70

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

	// 正規化的四個鍵必須能被寫出來並原樣帶出來。這裡的 reencode = **false** 是刻意選的：
	// 這一支測試走的是 TOML 路徑，而 *bool 的「明確 false」只有走這條路徑才驗得到
	// （直接構造 Config 的測試裡 nil 與 false 是兩個不同的值，见 TestApplyDefaultsPreservesExplicitValues）。
	normalized := cfg.imagePolicy()
	if normalized.Reencode {
		t.Error("TOML 裡寫了 reencode = false，卻被解成開啟")
	}
	if normalized.MaxWidth != 1600 || normalized.MaxHeight != 1200 {
		t.Errorf("輸出尺寸上限 = %dx%d，want 1600x1200", normalized.MaxWidth, normalized.MaxHeight)
	}
	if normalized.MaxSourcePixels != 8_000_000 {
		t.Errorf("來源解析度上限 = %d，want 8000000", normalized.MaxSourcePixels)
	}
	if normalized.JPEGQuality != 70 {
		t.Errorf("JPEG 品質 = %d，want 70", normalized.JPEGQuality)
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
			// 明確關掉正規化。這是這個模組裡唯一用 *bool 的欄位，而它存在的理由就是
			// 這一支測試要驗的東西：「明確關掉」不能被兜底值蓋成「開」
			// （bool 的零值是 false＝「明確關掉」與「沒設定」在同一個值上）。
			Reencode: imagePolicyEnabled(false),
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
	// 這一條是 *bool 那條取捨的**全部**價值：若兜底值把「明確關掉」蓋成「開」，
	// 那個部署會以為自己關掉了正規化，而 EXIF（含 GPS）仍然被移除 —— 而
	// main.go 的啟動警告會說「normalization enabled」，那是一條**對的**日誌
	// 對應一個錯的预期。這比「沒設定」更糟，因為它看起來像被驗證過了。
	if cfg.imagePolicy().Reencode {
		t.Error("reencode=false 被兜底值蓋成開啟 —— 這個設定值語意失效")
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

// imagePolicyEnabled 產生 UploadConfig.Reencode 的那個 *bool。
//
// 用函式而不是直接取字面量的位址（&false 在 Go 裡不合法）：這個欄位需要**兩種**顯式的值
// —— 「明確關掉」與「明確開啟」—— 而「沒設定」（nil）是第三種。三者必須都能被寫出來，
// 否則這個 *bool 的意義（見 config.go）在測試裡就只剩兩種可驗證的形状。
//
// 它放在這裡而不是 server_test.go，因為這個欄位的語意屬於 config.go；server_test.go
// 與 config_extra_test.go 都在同一個 package main 裡，因此都能用到它。
func imagePolicyEnabled(enabled bool) *bool { return &enabled }

// TestImagePolicyFromZeroConfigIsUsable 守住「直接建立的 Config 也能得到可用的政策」。
//
// server_test.go 的 testConfig 是直接構造 Config 而不是經過 loadConfig/applyDefaults，
// 因此 imagePolicy 必須自己處理零值。這裡的斷言不是「與 applyDefaults 一致」而是
// 「每個值都在可用的範圍內」，因為可用的定義在 imagePolicy 這一邊。
//
// 最關鍵的一條是 jpeg_quality：零值經過 jpeg.Encode 會被夾到 **1**（最差的那一個，
// 不是「用預設」）。若 imagePolicy 不處理 0，那些測試會產出品質 1 的圖片，
// 而沒有任何錯誤或日誌會說這件事。
func TestImagePolicyFromZeroConfigIsUsable(t *testing.T) {
	p := (&Config{}).imagePolicy()
	if !p.Reencode {
		t.Error("零值 Config 的正規化是關的 —— 預設應該是開（EXIF 的 GPS 該被移除）")
	}
	if p.MaxWidth <= 0 || p.MaxHeight <= 0 {
		t.Errorf("輸出尺寸上限 = %dx%d，必须是正數（0 會讓每一張圖被縮成 0）", p.MaxWidth, p.MaxHeight)
	}
	if p.MaxSourcePixels <= 0 {
		t.Errorf("來源解析度上限 = %d，必须是正數（0 會被 normalizeImage 當成「不設上限」）", p.MaxSourcePixels)
	}
	if p.JPEGQuality < 1 || p.JPEGQuality > 100 {
		t.Errorf("JPEG 品質 = %d，不在 1–100 內；jpeg.Encode 會把 <1 的值夾到 1", p.JPEGQuality)
	}
}

// TestImagePolicyClampsAbsurdValues 守住「設定值超出範圍時被夾而不是被照單全收」。
//
// 這一支的意義是 main.go 的啟動摘要（imagePolicyWarning）印的是**生效的**值：
// 若這裡不夾，那行日誌會寫「JPEG 品質 200」而編碼器實際用的是 100 ——
// 那條日誌就變成一件不實的事，而它是維運唯一能看到「現在用什麼政策」的地方。
func TestImagePolicyClampsAbsurdValues(t *testing.T) {
	cfg := &Config{Upload: UploadConfig{
		Reencode:        imagePolicyEnabled(true),
		MaxWidth:        -5,
		MaxHeight:       0,
		MaxSourcePixels: -1,
		JPEGQuality:     500,
	}}
	p := cfg.imagePolicy()
	if p.MaxWidth <= 0 || p.MaxHeight <= 0 {
		t.Errorf("負/零的尺寸上限沒有被夾: %dx%d", p.MaxWidth, p.MaxHeight)
	}
	if p.MaxSourcePixels <= 0 {
		t.Errorf("負的來源解析度上限沒有被夾: %d", p.MaxSourcePixels)
	}
	if p.JPEGQuality != 100 {
		t.Errorf("JPEG 品質 = %d，want 被夾到 100（設定裡寫 500，而編碼器只用 100）", p.JPEGQuality)
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
