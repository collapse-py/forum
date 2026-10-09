/*
config 是 files_server 的設定檔（TOML）解析層。

【為什麼獨立成檔】
原本這些型別與 loadConfig 擠在 main.go 裡，而 main.go 已經是「組裝所有相依
元件並決定啟動順序」的地方。設定解析是另一個主題（這份設定長什麼樣、哪些值有
兜底、哪些值不該有），混在啟動流程裡會讓閱讀順序被無關的欄位細節打斷。

【對外介面（本檔全部匯出符號）】
Config：設定容器，以指標傳遞（內含 slice，值傳遞的複製語意會造成誤解）。
loadConfig：讀檔、解析、套用兜底值。
本檔不註冊 HTTP 路由，也不碰磁碟上的實際檔案（除了解設定檔本身）。

【主要依賴】
github.com/BurntSushi/toml。刻意不引入其他解析器：這個模組的相依項目標準是
「能用預設值跑起來的站不需要額外安裝任何東西」。

【關鍵設計決策】
 1. 兜底值全部集中在 loadConfig 尾端，理由與 backend 的 applyDefaults 相同：
    一份沒有任何 [section] 的空設定檔仍然要能得到可用的設定。
 2. 只對「值為 0 / 空字串 / 空清單」補值，不對「格式錯誤」報錯。代價是打錯字與
    沒設定無法區分（見 Validate 對真正有風險的那幾項做的補償）。
 3. 設定路徑固定為工作目錄下的 "config.conf"（見 main.go 的說明），與 backend
    的 "config/config.conf" 是相對的兩種慣例 —— 這是既有的部署約束，不是這裡
    能統一掉的。
*/
package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
	// ShutdownTimeoutSec 是收到停止訊號後、強制結束前等待在途請求完成的秒數
	// （見 shutdown.go）。刻意只接受正數，因此 0 不是「立即關閉」這個選項 ——
	// 那等於把優雅停止整個關掉卻又不會有任何提示。0 會沿用預設值。
	ShutdownTimeoutSec int `toml:"shutdown_timeout_seconds"`
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

// RateLimitConfig 是每 IP 的計價設定。
//
// 兩個獨立的限制而不是一個：這三類端點的成本差了一個數量級（一次 50 MiB 的
// multipart 解析 vs 一次 Redis EXISTS）。共用一個上限的結果是要嘛保護不到寫入，
// 要嘛把正常載入圖片的頁面一起擋掉。理由見 ratelimit.go 的檔頭。
type RateLimitConfig struct {
	// Enabled 為 false 時完全不限制。留這個開關是為了「這個服務被放在另一個
	// 已經有限流的反向代理之後」的部署 —— 兩層限制疊加會讓上限難以推理。
	Enabled bool `toml:"enabled"`
	// MediaRate 是 /files/ 每秒放行的請求數（每 IP）。
	MediaRate float64 `toml:"media_requests_per_sec"`
	// MediaBurst 是 /files/ 的連發容量。
	MediaBurst int `toml:"media_burst"`
	// WriteRate 是 /upload 與 /delete 每秒放行的請求數（每 IP）。
	WriteRate float64 `toml:"write_requests_per_sec"`
	// WriteBurst 是 /upload 與 /delete 的連發容量。
	WriteBurst int `toml:"write_burst"`
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
	Server    ServerConfig    `toml:"server"`
	Storage   StorageConfig   `toml:"storage"`
	Upload    UploadConfig    `toml:"upload"`
	CORS      CORSPolicy      `toml:"cors"`
	Logger    LoggerConfig    `toml:"logger"`
	Redis     RedisConfig     `toml:"redis"`
	RateLimit RateLimitConfig `toml:"ratelimit"`
}

// loadConfig 讀取並解析 path 指向的 TOML 設定檔，回傳套用兜底值後的設定。
//
// 錯誤條件：檔案讀不到、或 TOML 語法錯誤。個別欄位的格式錯誤不會產生錯誤，而是
// 沿用兜底值（與 backend 的 key=value 解析風格一致）。
//
// 副作用：讀取檔案。不寫檔案、不發 HTTP 請求、不碰 Redis 與儲存後端 —— 那些都在
// main 裡、且各自帶自己的失敗處理。
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取配置失敗: %w", err)
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, fmt.Errorf("解析配置失敗: %w", err)
	}

	cfg.applyDefaults()

	return &cfg, nil
}

// applyDefaults 為「值為空或為 0」的欄位補上兜底值。
//
// 判斷一律以零值為準（""、nil、0、空清單），而不是「這個欄位有沒有出現過」：
// 這個設定格式不值得為此維護一份「已設定欄位集合」，而代價（無法用 0 明確停用
// 一個功能）已在各欄位的說明裡逐條寫出 —— 這裡唯一值得留意的是 MaxSize 與
// TokenTTLSec：0 會讓上傳不設上限、token 永不過期，兩者都是安全相關，因此
// 它們的兜底值必須是「有限」的那個。
func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ShutdownTimeoutSec <= 0 {
		// 15 秒，與後端的 SHUTDOWN_TIMEOUT_SECONDS 預設一致。
		//
		// 為什麼是 15 而不是更短：這個服務最長的單一請求是圖片上轉（本文可達
		// 50 MB），在慢速連線上寫完磁碟需要時間，而排空逾時會讓它變成「貼文存得
		// 下但圖片上傳失敗」—— 症狀出現在使用者身上，而原因在部署端。
		//
		// 為什麼不是更長：部署端（compose 的 stop_grace_period、systemd 的
		// TimeoutStopSec）不會等更久，因此設得再長也不會讓請求有更多時間完成，
		// 只會讓「部署看起來卡住」的情況拉長。
		c.Server.ShutdownTimeoutSec = 15
	}
	if c.Storage.Type == "" {
		c.Storage.Type = "local"
	}
	if c.Storage.Local.BaseDir == "" {
		c.Storage.Local.BaseDir = "./storage"
	}
	if c.Storage.Local.FilesDir == "" {
		c.Storage.Local.FilesDir = "files"
	}
	if c.Storage.Local.DirPerm == 0 {
		c.Storage.Local.DirPerm = 0755
	}
	if c.Storage.Local.FilePerm == 0 {
		c.Storage.Local.FilePerm = 0644
	}

	// 限流預設是「開」，而且刻意用「無法用 0 關掉」的形式：這個欄位的零值不是
	// 「關掉」，而是「沒設定」（見 applyDefaults 的說明）。真的想關掉請把
	// enabled 明確寫成 false —— 那是一次需要被看見的決定。
	//
	// 50/s + burst 200：一篇論壇貼文最多一張圖，而一頁動態大約 10 篇；一個 IP
	// 在頁面載入時發出的圖片請求遠低於此。寫入 2/s + burst 20：後端上傳是人的
	// 動作觸發的，一個人不可能每秒發兩次貼文。
	if c.RateLimit.MediaRate == 0 {
		c.RateLimit.MediaRate = 50
	}
	if c.RateLimit.MediaBurst == 0 {
		c.RateLimit.MediaBurst = 200
	}
	if c.RateLimit.WriteRate == 0 {
		c.RateLimit.WriteRate = 2
	}
	if c.RateLimit.WriteBurst == 0 {
		c.RateLimit.WriteBurst = 20
	}
	if c.Storage.S3.PresignExpireSec == 0 {
		c.Storage.S3.PresignExpireSec = 3600
	}
	if c.Upload.MaxSize == 0 {
		// 50 MiB，與後端 ParseMultipartForm 的上限一致。
		//
		// 這兩個數字必須相同：後端先在它的路由上設一個上限再轉送過來，若本服務
		// 的上限比較大，多出來的那一段會變成「後端已把本文收進記憶體、這裡拒收」
		// 的 400 —— 症狀是上傳失敗，而錯誤訊息來自一個使用者看不到的內部服務。
		// 反過來（這裡比較小）則是乾淨的 413，比較好診斷，但仍屬於設定不一致。
		c.Upload.MaxSize = 50 << 20
	}
	if len(c.Upload.AllowedFiles) == 0 {
		c.Upload.AllowedFiles = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".opus"}
	}
	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}
	if len(c.CORS.AllowedOrigins) == 0 {
		// 預設放行所有來源。為什麼這是合理的：這個服務只做兩件事 ——
		// 接受帶 token 的上傳、回傳已簽發的媒體 token 過濾過的檔案。兩者都
		// 需要先持有後端簽發的憑證，因此「知道網址」不等於「能讀取內容」。
		// 真正的保護是 upload.token 與 Redis 裡的 media token，不是 CORS。
		// 反過來，收緊預設值會讓本機開發（Vite 的 5173）開箱即壞。
		c.CORS.AllowedOrigins = []string{"*"}
	}
	if len(c.CORS.AllowedMethods) == 0 {
		// 刻意包含寫入方法：上傳與刪除是這個服務存在的主要理由，而它們都是
		// 跨來源的瀏覽器請求（後端轉传 FormData、後台刪圖）。只放行 GET/HEAD 會
		// 讓預檢在 204 上「看起來成功」，真正的請求才失敗 —— 那是最難診斷的一種。
		c.CORS.AllowedMethods = []string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "DELETE"}
	}
	if len(c.CORS.AllowedHeaders) == 0 {
		// 刻意包含 Range：論壇的音檔靠它做拖曳定位，少了它音訊播放器會退化成
		// 「整個檔案下載完才能播」。
		//
		// 另三項是上傳與刪除的實際標頭：multipart 的 Content-Type、後端的
		// X-Upload-Token，以及部分呼叫端沿用的 Authorization。少了任何一個，
		// 預檢一樣是 204 而真正的請求失敗。
		c.CORS.AllowedHeaders = []string{
			"Origin", "Range", "Accept", "Accept-Language",
			"Content-Type", "X-Upload-Token", "Authorization",
		}
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}
	// 刻意不對 DB 做兜底（0 就是 0）：Redis 的邏輯資料庫編號 0 是正常值，
	// 而「寫成 3 卻被當成未設定而歸 0」會讓它與後端的 session 資料混在同
	// 一個邏輯庫裡 —— 那個後果（誤執行 FLUSHDB 會把登入狀態一起清掉）遠比
	// 少了這個兜底嚴重。
	if c.Redis.TokenKeyPrefix == "" {
		// 必須與後端的 MEDIA_TOKEN_KEY_PREFIX 相同，且都要與 session 的
		// "forum:session:" 前綴區隔。不一致的症狀是「上傳成功、貼文也存得下，
		// 但圖片一律 401/403」—— 而那正是難以診斷的形狀。
		//
		// 兜底值刻意與 backend/forum/config 的 applyDefaults 用同一個字面值
		// "forum:token:"：兩邊都「沒設定」時仍然一致，而「一邊漏設」這個最常見
		// 的部署失誤就不會再產生上述症狀。故意不選一個只有本服務會用的前綴
		// —— 那等於把「漏設」從設定錯誤升級成「所有圖片永遠 401」。
		c.Redis.TokenKeyPrefix = "forum:token:"
	}
	if c.Redis.TokenTTLSec == 0 {
		// 300 秒。
		//
		// 這個值目前**沒有被任何程式碼讀取**：TTL 是由後端在簽發 token 時寫進
		// Redis 的（MEDIA_TOKEN_TTL_SECONDS，見 backend/forum/httpapi 的
		// mediaRedis），本服務只做「查這個 key 還在不在」。之所以仍然給它兜底值
		// 並在這裡寫下理由，是因為它出現在設定檔裡就會有人照著設定：若將來真的
		// 在本服務用上它，而它與後端不一致，症狀是「圖片在某些情況下突然 401」，
		// 而那正是最難診斷的一類。
		//
		// 0 刻意不當成「永不過期」：那會讓任何一張被轉貼出去的圖片連結永久有效。
		c.Redis.TokenTTLSec = 300
	}
}
