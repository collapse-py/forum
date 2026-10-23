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
- [日誌輸出格式](#日誌輸出格式)

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
| `SHUTDOWN_TIMEOUT_SECONDS` | `15` | 停止時等待在途請求的秒數。之後還有最多 3 秒的收尾，部署環境的停止上限必須大於這個總和 |
| `PUBLIC_BASE_URL` | `http://localhost` + `SERVER_PORT` | 尾斜線會被自動去掉 |
| `TRUSTED_ORIGINS` | `[PUBLIC_BASE_URL]` | CSRF 來源白名單。**留空等於關閉 CSRF 防護** |
| `TRUSTED_PROXY_CIDRS` | 空 | 可信任反向代理的位址段。**容器部署必須填 `172.28.0.0/16`** |
| `DB_DSN` | — | 內含帳密 |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME_MINUTES` | `20` / `5` / `5` | 連線池 |
| `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` | `127.0.0.1:6379` / 空 / `0` | |
| `ES_URL` | 空（停用） | 設定了就啟用 ES 搜尋 |
| `ES_INDEX` | `forum_posts` | |
| `ES_USERNAME` / `ES_PASSWORD` | 空 | ES 的 Basic Auth。留空＝不帶驗證標頭 |
| `ES_API_KEY` | 空 | ES 的 API key。**與 `ES_USERNAME` 同時設定時以此為準** |
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
| `LOG_LEVEL` / `LOG_FILE` / `LOG_FORMAT` | `INFO` / `server.log` / `text` | 格式可選 `json`；輸出行格式見[日誌輸出格式](#日誌輸出格式) |
| `MONITOR_RETENTION_HOURS` | `24` | 監控分鐘彙總在資料庫保留幾小時 |
| `AUDIT_RETENTION_DAYS` | `90` | 稽核紀錄保留幾天 |
| `POST_RETENTION_DAYS` | `90` | **軟刪除的貼文保留幾天**。到期後會被硬刪（連同留言、按讚、收藏、檢舉工單），**不可逆** |
| `LOG_MAX_SIZE_MB` | `50` | 日誌檔輪替的大小上限（MB）。`0` = 不輪替 |
| `LOG_MAX_FILES` | `5` | 保留的輪替檔數（不含目前檔）。至少 1；超過 1000 以 1000 計（避免輪替的 rename 迴圈無界成長） |

### 兩組「設定錯了也不會有症狀」的鍵

`POST_RETENTION_DAYS` 與 `LOG_MAX_SIZE_MB` 的失效**完全沒有症狀**：服務照常啟動、
照常回應、日誌乾淨、HTTP 狀態碼全對。

- `POST_RETENTION_DAYS` 是全站唯一會**不可逆地刪掉使用者內容**的設定。設定成 7 與
  設定成 90 的差別，在啟動之後就看不到 —— 只有背景清理排程知道。因此 `forum -check`
  的報告刻意印出這個值。
- `LOG_MAX_SIZE_MB = 0` 代表「不輪替」，同樣沒有症狀，只是日誌檔會一直長到磁碟滿
  （而磁碟滿之後的失敗模式是「輪替失敗」——那時日誌會退回只寫 stdout，且檔案輸出
  停止；這一條會在 stderr 印一行警告）。

### 站名

`FORUM_NAME` / `FORUM_SHORT_NAME` / `FORUM_DESCRIPTION` 三個鍵是站名。站名會在送出頁面時注入 HTML 與 PWA manifest，因此改 `FORUM_NAME` 不需要重新建置前端。前端含站名的文案以 `{site}`（完整站名）與 `{brand}`（短名）佔位，值來自這裡（見 `frontend/src/i18n/messages.ts` 檔頭註解與 `src/site.ts`）——換站名因此不必動任何語言檔。

### 幾個不看原始碼會猜錯的細節

- `applyDefaults` 對「空值或非正數」補值，因此**無法用設定檔把某個視窗設成 0 秒**（那會讓限流失效）。布林值接受 `1` / `true` / `yes` / `on`（不分大小寫）。
- **ES 的驗證只走設定檔**，不像 `FILES_SERVER_TOKEN` 有環境變數後備 —— 與 `DB_DSN`、`GOOGLE_CLIENT_SECRET` 一致。`ES_API_KEY` 可以貼「未編碼的 `id:api_key`」或「已編碼的 base64」兩種寫法（看有無冒號分辨），程式只挑一種送。
- **ES 三項驗證全留空時不會送出任何 `Authorization` 標頭**。ES 若啟用了安全性，沒有憑證的每個請求都是 401，而搜尋會安靜退回 MySQL `LIKE`（網站看起來正常，只有搜尋品質變差），因此 `forum -check` 會印出這一列的狀態。
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

---

## 日誌輸出格式

`LOG_FORMAT` 只有兩種值，兩種都是「一行一筆」。欄位順序固定，`msg` 永遠在最後。

### `text`（預設，人讀）

```
time=2026-10-13T09:22:12.123+08:00 level=INFO caller="main.go:318" msg="[SERVER] Forum server started at :8088"
time=2026-10-13T09:22:12.456+08:00 level=WARN request_id="9f2c1d0b4a6e5f70" ip="203.0.113.7" user_email="anonymous" msg="[HTTP] GET /api/forum/posts HTTP/1.1 status=404 duration_ms=850.000"
```

- **時間**是 RFC 3339 毫秒精度、帶本機時區位移（`Z` 代表 UTC）。多主機彙總時可直接排序，不需要猜時區；同一時區偏移下字典序等於時間序，混合不同偏移須先解析再比對。
- **等級** `level=` 不加引號，方便 `grep 'level=ERROR'`。
- **`time` / `level` / `caller` / `msg` 之外的值一律引號化**（Go 字串字面值）。IP、email 這類外部可控值即使含空白或 `key=value`，也留在同一對引號內，不會偽造出獨立欄位。
- **`request_id` / `ip` / `user_email`** 只出現在請求範圍的記錄上；背景工作（啟動訊息、遷移結果）整段省略。
- **`caller`** 是「這行是誰打的」。存取記錄的呼叫點在 logger 中介層本身，座標固定、沒有診斷價值，因此省略；業務碼的呼叫一律保留。
- **`msg` 永遠是最後一個欄位**，因此「一行從哪裡開始是訊息」只需要一條規則。
- 訊息裡的換行與控制字元會被清洗成空格（`sanitizeLogValue`），「注入一行假記錄」在這裡被擋下；單一欄位值超過 256 bytes 會截斷並加 `...`。

### `json`（機器讀，給 ELK / Loki）

```json
{"time":"2026-10-13T09:22:12.123+08:00","level":"WARN","request_id":"9f2c1d0b4a6e5f70","ip":"203.0.113.7","user_email":"anonymous","caller":"","msg":"[HTTP] GET /api/forum/posts HTTP/1.1 status=404 duration_ms=850.000"}
```

- 固定七個鍵（`time` / `level` / `request_id` / `ip` / `user_email` / `caller` / `msg`），schema 不隨訊息改變，收集端可以直接建索引。
- 背景工作的三個中繼鍵仍存在、值為空字串（刻意不出現在/缺席鍵，避免收集端型別推斷出錯）；`caller` 被抑制時同樣是空字串。
- 所有值都是字串，因此 `status`、`duration_ms`（固定毫秒、三位小數）目前在 `msg` 內而不是獨立鍵 —— 改版時若要升為頂層欄位，收集端的 parser 需要一併更新。

### 兩個刻意的限制

- **不做檔案切割**：沒有大小上限、沒有時間輪替，`LOG_FILE` 會被無限追加，換檔請靠外部機制（logrotate 或容器 stdout 收集），換檔後需重啟行程才會重新開檔。
- **不做敏感資訊遮蔽**：email 與來源 IP 是明文。`msg` 會不會帶出 token、密碼取決於呼叫端傳了什麼；存取記錄只記路徑、不記查詢字串，就是為了降低這個風險。
