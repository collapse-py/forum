# 設定參考

兩個 `config.conf` 都在 `.gitignore` 裡，必須各自從範本複製出來。範本已提交且不含任何真實憑證，並附有逐項註解：

```bash
# Linux / macOS
cp backend/config/config.conf.example backend/config/config.conf
cp files_server/config.conf.example  files_server/config.conf

# Windows PowerShell
Copy-Item backend\config\config.conf.example backend\config\config.conf
Copy-Item files_server\config.conf.example  files_server\config.conf
```

- [backend/config/config.conf](#backendconfigconfigconf)
- [files_server/config.conf](#files_serverconfigconf)
- [最小可用設定](#最小可用設定)

---

## `backend/config/config.conf`

格式是樸素的 `KEY=VALUE`，一行一組，`#` 或 `;` 開頭是註解，逗號分隔表示清單。**沒有熱重載** —— 一次啟動時讀成快照。

| 鍵 | 預設值 | 說明 |
| --- | --- | --- |
| `FORUM_NAME` | `FORUM 論壇` | 站名。會被注入 HTML 與 PWA manifest |
| `FORUM_SHORT_NAME` | 同 `FORUM_NAME` | 精簡站名，給標誌與窄螢幕 |
| `FORUM_DESCRIPTION` | 同 `FORUM_NAME` | manifest 描述 |
| `COOKIE_NAME` | `FORUM_forum` | Session cookie 名稱 |
| `SESSION_EXPIRE_HOURS` | `72` | **滑動**續期，所以是「閒置多久失效」 |
| `COOKIE_SECURE` | `false` | 走 HTTPS 就必須 `true`（見 [SECURITY](SECURITY.md)） |
| `SERVER_PORT` | `:8088` | `net/http` 的 `host:port` |
| `READ_HEADER_TIMEOUT_SECONDS` | `10` | 讀完**請求標頭**的期限。只卡標頭不卡本文 |
| `SHUTDOWN_TIMEOUT_SECONDS` | `15` | 停止時等待在途請求的秒數。必須小於部署環境的停止上限 |
| `PUBLIC_BASE_URL` | `http://localhost` + `SERVER_PORT` | 尾斜線會被自動去掉 |
| `TRUSTED_ORIGINS` | `[PUBLIC_BASE_URL]` | CSRF 來源白名單。**留空等於關閉 CSRF 防護** |
| `TRUSTED_PROXY_CIDRS` | 空 | 可信任反向代理的位址段。**容器部署必須填 `172.28.0.0/16`** |
| `DB_DSN` | — | 內含帳密 |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME_MINUTES` | `20` / `5` / `5` | 連線池 |
| `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` | `127.0.0.1:6379` / 空 / `0` | |
| `ES_URL` | 空（停用） | 設定了就啟用 ES 搜尋 |
| `ES_INDEX` | `forum_posts` | |
| `FILES_SERVER_URL` | `http://localhost:7070` | 內部上傳目標 |
| `FILES_SERVER_PUBLIC_URL` | 同上 | 對外可讀的位址；與上一項相同代表沒有內外網分離 |
| `FILES_SERVER_TOKEN` | 讀環境變數 `FILES_SERVER_TOKEN` | **唯一支援環境變數的設定** |
| `MEDIA_TOKEN_KEY_PREFIX` | `forum:token:` | 必須與 files_server 的 `token_key_prefix` 相同 |
| `MEDIA_TOKEN_TTL_SECONDS` | `2592000`（30 天） | 這個兜底值對正式環境明顯過長，因此**正式環境採用兜底值會讓啟動被拒絕** |
| `RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW_SECONDS` | `10` / `60` | 內容寫入 |
| `RATE_LIMIT_UPLOAD_REQUESTS` / `RATE_LIMIT_UPLOAD_WINDOW_SECONDS` | `5` / `60` | 圖片上傳（最貴，額度最緊） |
| `RATE_LIMIT_AUTH_REQUESTS` / `RATE_LIMIT_AUTH_WINDOW_SECONDS` | `10` / `60` | OAuth 端點，含所有 method |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | — | |
| `GOOGLE_REDIRECT_URL` | — | 必須與 Google Cloud Console 逐字相符 |
| `ALLOWED_ADMIN_EMAIL` | 空 | 逗號分隔。載入與比對**雙邊**都做 TrimSpace + ToLower |
| `LOG_LEVEL` / `LOG_FILE` / `LOG_FORMAT` | `INFO` / `server.log` / `text` | 格式可選 `json` |
| `MONITOR_RETENTION_HOURS` | `24` | 監控分鐘彙總在資料庫保留幾小時 |
| `AUDIT_RETENTION_DAYS` | `90` | 稽核紀錄保留幾天 |

### 站名

`FORUM_NAME` / `FORUM_SHORT_NAME` / `FORUM_DESCRIPTION` 三個鍵是站名。站名會在送出頁面時注入 HTML 與 PWA manifest，因此改 `FORUM_NAME` 不需要重新建置前端。前端含站名的文案以 `{site}`（完整站名）與 `{brand}`（短名）佔位，值來自這裡（見 `frontend/src/i18n/messages.ts` 檔頭註解與 `src/site.ts`）——換站名因此不必動任何語言檔。

### 幾個不看原始碼會猜錯的細節

- `applyDefaults` 對「空值或非正數」補值，因此**無法用設定檔把某個視窗設成 0 秒**（那會讓限流失效）。布林值接受 `1` / `true` / `yes` / `on`（不分大小寫）。
- **刻意「設成 30 天」是允許的** —— 那代表部署者知道自己在做什麼，與「忘記設定」是不同的情況。
- `MONITOR_RETENTION_HOURS` 決定「寫多少進資料庫」，**不決定監控頁時間軸的長度**（那是 `metrics.Options.WindowMinutes`，目前固定 120 分鐘）。混為一談的後果是設定 retention 的人以為圖會變長。
- `ALLOWED_ADMIN_EMAIL` 刻意**不**做 Gmail 點號與 `+` 別名的正規化 —— 白名單的重點是「只有這些明確的信箱可以當管理員」，放寬比對只會讓它更容易被誤加。

---

## `files_server/config.conf`

TOML 格式，同樣是相對於工作目錄的 `config.conf`，所以要從 `files_server/` 執行。

| 區段 / 鍵 | 說明 |
| --- | --- |
| `[server] host, port` | 監聽位址，預設埠 7070 |
| `[storage] type` | `local` 或 `s3` |
| `[storage.local] base_dir, files_dir, ...` | 本機存儲根目錄，執行時自動建立 |
| `[storage.s3] endpoint, keys, region, bucket, files_prefix, presign_expire_seconds` | S3 / MinIO 相容後端 |
| `[upload] token` | 上傳／刪除權杖，必須等於後端的 `FILES_SERVER_TOKEN` |
| `[upload] max_size` | 位元組上限，預設 50 MB |
| `[upload] allowed_files` | 副檔名白名單（**不含 svg**） |
| `[redis] addr, password, db` | |
| `[redis] token_key_prefix, token_ttl_sec` | 媒體權杖驗證用，須與後端一致 |
| `[redis] public_files` | `true` 則 `/files/*` 不需要 token |
| `[logger]`, `[cors]` | |

檔案落地時一律用 **UUID v4 + 原副檔名（小寫）**，永遠不用使用者提供的檔名。

> `token_key_prefix` 兩邊必須一致，且要與後端 session 的 `forum:session:` 前綴區隔。
> 不一致的話**症狀是上傳成功、貼文也存得下，但圖片 GET 一律 401**。

---

## 最小可用設定

**`backend/config/config.conf`**

```ini
FORUM_NAME=論壇
FORUM_SHORT_NAME=FORUM
SERVER_PORT=:8088
PUBLIC_BASE_URL=http://localhost:8088
TRUSTED_ORIGINS=http://localhost:8088
# 只有在確實擋在一道反向代理後面時才填；見 docs/SECURITY.md 的信任模型
TRUSTED_PROXY_CIDRS=

DB_DSN=forum:forum_password@tcp(127.0.0.1:3306)/forum?charset=utf8mb4&parseTime=True&loc=Local
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=

# Google OAuth2（authorization code flow）
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=http://localhost:8088/auth/callback
ALLOWED_ADMIN_EMAIL=you@example.com

# 檔案服務
FILES_SERVER_URL=http://localhost:7070
FILES_SERVER_PUBLIC_URL=http://localhost:7070
FILES_SERVER_TOKEN=change_me

LOG_LEVEL=INFO
LOG_FORMAT=text
```

**`files_server/config.conf`**（TOML）

```toml
[server]
host = "0.0.0.0"
port = 7070

[storage]
type = "local"

[storage.local]
base_dir = "./storage"

[upload]
token = "change_me"      # 必須與後端 FILES_SERVER_TOKEN 相同
max_size = 52428800
allowed_files = ['.jpg', '.jpeg', '.png', '.gif', '.webp']

[redis]
addr = "127.0.0.1:6379"
password = ""
db = 0
token_key_prefix = "forum:token:"   # 必須與後端 MEDIA_TOKEN_KEY_PREFIX 相同
token_ttl_sec = 60
public_files = false
```
