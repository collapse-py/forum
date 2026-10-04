# 論壇

免註冊、以 Google 帳號登入的匿名論壇。登入後的對外身分是 `SHA256(email)` 算出來的
`public_key`，公開頁面與 URL 裡都不會出現 email。管理員身分不看資料庫角色表，只看
`ALLOWED_ADMIN_EMAIL` 白名單。

後端是只用標準函式庫寫的 Go HTTP 服務（`net/http` + `database/sql`，沒有 web
framework、沒有 ORM），前端是 Vite + React 19 + TypeScript 的多頁應用（PWA），
另有獨立的 Go 檔案服務處理媒體存儲。

---

## 目錄

- [文件地圖](#文件地圖)
- [功能](#功能)
- [架構](#架構)
- [環境需求](#環境需求)
- [快速開始](#快速開始)
- [設定檔](#設定檔)
- [資料庫](#資料庫)
- [API](#api)
- [頁面路由](#頁面路由)
- [前端開發](#前端開發)
- [測試與不變條件](#測試與不變條件)
- [部署](#部署)
- [安全模型](#安全模型)
- [目前最需要注意的三件事](#目前最需要注意的三件事)

---

## 文件地圖

本檔是**入口**：怎麼跑起來、設定有哪些、端點有哪些。設計理由與維運細節拆到 `docs/`，
因為它們值得寫、但不該擋住「先把它跑起來」這件事。

| 文件 | 回答什麼問題 | 讀它的時機 |
| --- | --- | --- |
| `README.md`（本檔） | 怎麼跑起來？設定有哪些？ | 第一次接觸這個專案 |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | 每個子系統為什麼長成這樣？刻意不做哪些事？ | 要改程式碼之前 |
| [`docs/SECURITY.md`](docs/SECURITY.md) | 信任邊界在哪？哪些東西被刻意信任了？ | 上線前、或做安全評估時 |
| [`docs/OPERATIONS.md`](docs/OPERATIONS.md) | 怎麼部署？後台各頁在回答什麼問題？ | 維運這個站的時候 |
| [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) | 現在還有哪些沒處理完的？ | 下次 deploy 前、或接手維護時 |
| [`tools/i18n/README.md`](tools/i18n/README.md) | 批次翻譯的流程與驗證 | 要加語系時 |

原始碼的檔頭註解自成一份規格 —— 這個專案多數非顯而易見的決定都寫在**它發生的地方**，
而不在本檔。

---

## 功能

**論壇**

- **匿名發文**：文字貼文，可選一張圖片；編輯與刪除自己的貼文（卡片上就地編輯，
  並標示「已編輯」；編輯後同步更新搜尋索引，不會出現「改了字、搜尋還是舊字」）
- **永久連結**：`/forum/post/{id}` 單篇頁刻意不需登入 —— 分享連結的人不必先跟
  收連結的人說「請先登入」
- **草稿自動儲存**：新增貼文頁的內文 debounce 寫進 localStorage（只有文字；
  選好的圖片無法序列化，因此不保留）
- **留言、按讚、檢舉**：留言公開可讀，作者可自行編輯或刪除；按讚以
  `(post_id, author_email)` 複合主鍵限制每人一次；檢舉涵蓋貼文與留言
- **追蹤與私密動態牆**：`/api/forum/following/posts` 只回傳追蹤中的人的貼文
- **Google OAuth2 登入**：僅申請 `userinfo.email`，沒有密碼、沒有本機帳號
- **Elasticsearch 全文搜尋**：`ES_URL` 留空則自動退回 MySQL `LIKE`
- **PWA**：可安裝、有 service worker 與 manifest
- **多語系**：17 個語系（含 `zh-TW` 為基準），阿拉伯文為 RTL

**後台**（`/admin`，需管理員）

- **貼文／留言 CRUD**、**檢舉佇列裁決**（通過即刪除對象、駁回保留）、**帳號停權與復原**、
  **使用者標籤**字典與指派、**以使用者身分代發文**
- **系統監控** `/admin/monitor`：MySQL／Redis／搜尋引擎狀態、Go 執行期用量、依正規化
  路由統計的請求量與延遲分位數、限流器計數、來源位址排行（可一鍵封鎖，並標示位址是
  取自連線對端還是可偽造的標頭）、分鐘級流量圖
- **稽核日誌** `/admin/log`：後台每一項改變資料的操作，含欄位級 diff。稽核寫入與操作在
  同一個交易裡，不會出現「操作發生但沒有紀錄」；保留 90 天，**沒有**任何刪除紀錄的 API
- **內容趨勢** `/admin/stats`：每日新增的使用者／文章／留言、視窗內合計、熱門文章、
  熱門標籤、活躍作者。視窗 7～90 天可切換
- **匯出與批次** `/admin/export`：三份 CSV（對公式注入有防護）；用戶管理頁可多選後
  批次停權、批次套標籤
- **登入與 Session 管理** `/admin/sessions`：列出仍有效的登入狀態（只顯示 token 前 8 個
  字元），可強制登出某帳號的所有裝置
- **IP 封鎖** `/admin/blocks`：把濫用來源加進存在 Redis 的封鎖名單。刻意**不會**自動封鎖
- **站內公告與置頂** `/admin/announcements`：全站橫幅（同時只有一則生效）、文章逐篇置頂

**跨站共通**

- **Redis 版可撤銷 Session**：滑動續期，Redis 掛掉等於全部登入失效
- **三段式 IP 限流**：內容寫入（10/60s）、圖片上傳（5/60s）、OAuth 跳轉（10/60s）各自
  獨立額度，依端點成本區分
- **媒體存取權杖**：上傳時發短 TTL 的 Redis token 綁定圖片網址，可主動釋放

各後台頁面「在回答什麼問題」與其取捨，見 [`docs/OPERATIONS.md`](docs/OPERATIONS.md)。

---

## 架構

```
                     ┌───────────────────────────────────────────────┐
   瀏覽器 ──────────▶ │  backend/  (Go, :8088)                        │
   PWA                │                                               │
   /forum             │   httpapi/   路由、中介層、security headers   │
   /admin             │   auth/      Google OAuth2                   │
                      │   session/   Redis session                   │
                      │   ipban/     IP 封鎖名單（Redis sorted set）  │
                      │   metrics/   請求統計、分鐘彙總持久化         │
                      │   audit/     管理員操作稽核、依時間清理        │
                      │   data/      MySQL + MigrateMySQL            │
                      │   es/        Elasticsearch（選用）            │
                      │   ── 服務 frontend/dist/ 的靜態檔案 ──       │
                      └────────┬─────────────────────┬───────────────┘
                               │ SQL                 │ HTTP + token
                               ▼                     ▼
                     ┌──────────────────┐   ┌─────────────────────────┐
                     │ MySQL  :3306     │   │ files_server/ (Go,      │
                     │ utf8mb4 / InnoDB │   │ :7070) 檔案服務          │
                     ├──────────────────┤   │  local / S3 存儲         │
                     │ Redis  :6379     │◀─▶│  Redis media token      │
                     │ ES     :9200     │   └─────────────────────────┘
                     │ （選用）          │
                     └──────────────────┘
```

反向代理（nginx / Caddy / Cloudflare Tunnel）**不在此 repo 內**，TLS 由它終結。

兩個 Go 模組彼此獨立，各自的 `go.mod` 與設定檔：

| 目錄 | 模組名 | Go | 設定檔 | 埠 | 職責 |
| --- | --- | --- | --- | --- | --- |
| `backend/` | `forum` | 1.26 | `config/config.conf` | 8088 | 全部 API、認證、頁面與靜態檔 |
| `files_server/` | `files_server` | 1.26 | `config.conf` | 7070 | 媒體上傳／刪除／讀取 |

第三個 Go 模組 `tools/invariants/` 只在 CI 使用（見「[測試與不變條件](#測試與不變條件)」）。

---

## 環境需求

| 需求 | 版本 | 必要？ |
| --- | --- | --- |
| Go | 1.26+ | 必要 |
| Node.js | 20+ | 建置前端時必要 |
| MySQL | **8.0.29+**，建一個 `utf8mb4` 資料庫 | 必要（schema 自動建立） |
| Redis | 5+ | 必要（session 與 media token） |
| Elasticsearch | 7.x / 8.x | 選用，留空則用 MySQL `LIKE` 搜尋 |
| Google OAuth2 用戶端 | — | 登入與管理功能需要 |

MySQL 的下限是 **8.0.29** 而不是籠統的「8.x」，因為 `MigrateMySQL` 用到
`ALTER TABLE ... ADD COLUMN IF NOT EXISTS` —— 這個語法在 8.0.29 之前不存在，而
`forum_posts.pinned` 正是這樣加上去的。在較舊的 8.0 上啟動會直接拿到
`ERROR 1064 (42000): You have an error in your SQL syntax`，遷移回錯、`main` 呼叫
`Fatalf`，行程在啟動時就死。

---

## 快速開始

### 1. 準備設定檔

兩個 `config.conf` 都在 `.gitignore` 裡，必須各自從範本複製出來。範本已提交且不含
任何真實憑證：

```bash
# Linux / macOS
cp backend/config/config.conf.example backend/config/config.conf
cp files_server/config.conf.example  files_server/config.conf

# Windows PowerShell
Copy-Item backend\config\config.conf.example backend\config\config.conf
Copy-Item files_server\config.conf.example  files_server\config.conf
```

兩份範本都附有逐項註解，以下是最小可用設定。

**`backend/config/config.conf`**

```ini
FORUM_NAME=SB 論壇
FORUM_SHORT_NAME=SB
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

> `token_key_prefix` 兩邊必須一致，且要與後端 session 的 `forum:session:` 前綴區隔。
> 不一致的話**症狀是上傳成功、貼文也存得下，但圖片 GET 一律 401**。

### 2. 啟動檔案服務

```bash
cd files_server
go mod download
go run .            # 或 ./start.ps1（會先 go build 再執行）
```

### 3. 建置前端

```bash
cd frontend
npm install
npm run build       # 內含 typecheck，產出 frontend/dist/
```

`npm run build` = `tsc -p tsconfig.json && tsc -p tsconfig.sw.json && vite build`。
沒有 `test` / `lint` 腳本；型別檢查（`strict` + `noUncheckedIndexedAccess` +
`exactOptionalPropertyTypes`）就是這個專案的 lint。

### 4. 啟動後端

```bash
cd backend
go build .

./forum -check      # 先驗證設定：輸出報告，有問題時 exit 1
./forum             # 正常啟動（Windows: .\forum.exe；或直接 go run .）
```

`-check` **不連線任何外部服務**，所以它回答的是「這個行程會以什麼組態啟動」，而不是
「相依服務能不能連上」。報告刻意不印任何憑證（只印「已設定（N 個字元）」），因為它會
出現在 CI 日誌、issue 回報與截圖裡。它點名的都是**啟動之後完全沒有症狀**的那一類問題：
正式環境的 `COOKIE_SECURE`、三組限流的實際額度、媒體 token 的 TTL、管理員白名單的
**實際儲存值**（去除空白並轉小寫之後 —— 這正是部署者要確認的東西）。

啟動順序是固定的：讀設定 → 初始化 logger → **驗證設定（`config.Validate`）** → ping
Redis → 開 MySQL → `data.MigrateMySQL` → `auth.Init` → `session.NewManager` → 註冊路由
→ 啟動限流清理 → 背景重建 ES 索引 → 監聽。

開瀏覽器到 <http://localhost:8088/forum>（`/` 會 302 到這裡）。

> **一定要在 `backend/` 目錄下執行。** 設定檔路徑是 `config/config.conf`（相對於
> 工作目錄而不是執行檔所在目錄），在別的目錄跑會直接 fatal。用 `-config <路徑>`
> 可以覆寫它。

後端執行檔有三種模式，都是同一支執行檔：

| 旗標 | 做什麼 | 連線任何相依服務？ | 退出碼 |
| --- | --- | --- | --- |
| （無） | 正常啟動服務 | 是 | 0（正常停止）／1（監聽失敗） |
| `-check` | 輸出設定報告後結束 | **否** | 0（通過）／1（有問題） |
| `-healthz` | 探測 MySQL 與 Redis 後結束 | 是（僅探測） | 0（健康）／1（不健康） |

停止後端按 Ctrl-C（或 `kill`／`systemctl stop`／`docker stop`）即可：收到
`SIGINT` / `SIGTERM` 之後會停止接受新連線、等在途請求完成（上限
`SHUTDOWN_TIMEOUT_SECONDS`）、寫出最後一次分鐘彙總，然後以 exit code 0 結束。
容器部署要一併把停止上限調大（見「[部署](#部署)」）。

---

## 設定檔

### `backend/config/config.conf`

格式是樸素的 `KEY=VALUE`，一行一組，`#` 或 `;` 開頭是註解，逗號分隔表示清單。
**沒有熱重載** —— 一次啟動時讀成快照。

| 鍵 | 預設值 | 說明 |
| --- | --- | --- |
| `FORUM_NAME` | `FORUM 論壇` | 站名。會被注入 HTML 與 PWA manifest |
| `FORUM_SHORT_NAME` | 同 `FORUM_NAME` | 精簡站名，給標誌與窄螢幕 |
| `FORUM_DESCRIPTION` | 同 `FORUM_NAME` | manifest 描述 |
| `COOKIE_NAME` | `FORUM_forum` | Session cookie 名稱 |
| `SESSION_EXPIRE_HOURS` | `72` | **滑動**續期，所以是「閒置多久失效」 |
| `COOKIE_SECURE` | `false` | 走 HTTPS 就必須 `true`（見 [SECURITY](docs/SECURITY.md)） |
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

幾個不看原始碼會猜錯的細節：

- `applyDefaults` 對「空值或非正數」補值，因此**無法用設定檔把某個視窗設成 0 秒**
  （那會讓限流失效）。布林值接受 `1` / `true` / `yes` / `on`（不分大小寫）。
- **刻意「設成 30 天」是允許的** —— 那代表部署者知道自己在做什麼，與「忘記設定」
  是不同的情況。
- `MONITOR_RETENTION_HOURS` 決定「寫多少進資料庫」，**不決定監控頁時間軸的長度**
  （那是 `metrics.Options.WindowMinutes`，目前固定 120 分鐘）。混為一談的後果是設定
  retention 的人以為圖會變長。
- `ALLOWED_ADMIN_EMAIL` 刻意**不**做 Gmail 點號與 `+` 別名的正規化 —— 白名單的重點
  是「只有這些明確的信箱可以當管理員」，放寬比對只會讓它更容易被誤加。

### `files_server/config.conf`

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

---

## 資料庫

MySQL，`utf8mb4` / InnoDB。**Schema 在啟動時自動建立**：`data.MigrateMySQL` 是一連串
idempotent 步驟（`CREATE TABLE IF NOT EXISTS`、`ALTER TABLE ... ADD COLUMN IF NOT
EXISTS`、`INSERT IGNORE`），所以空資料庫直接啟動即可，不需手動匯入 SQL。沒有版本化的
migration，也沒有降級路徑。

| 資料表 | 用途 | 關鍵結構 |
| --- | --- | --- |
| `forum_posts` | 貼文 | `image_url` 只存**檔名**，讀取時才用 `FILES_SERVER_PUBLIC_URL` 組網址；`updated_at` 可空＝從未被作者編輯過 |
| `forum_post_comments` | 留言 | 無外鍵，完整性靠應用層；`updated_at` 語意同上 |
| `forum_post_likes` | 按讚 | 複合 PK `(post_id, author_email)`，天然去重 |
| `forum_reports` | 檢舉 | `target_type`（`post`/`comment`）+ `target_id` 的多型參照，無外鍵 |
| `forum_profiles` | 公開資料 | `public_key` = `SHA2(author_email, 256)`；`nickname` 唯一 |
| `forum_users` | 帳號狀態 | `status` 為 `ACTIVE` / `SUSPENDED`。**沒有 role 欄位** |
| `forum_user_tags` | 標籤字典 | `name` 唯一 |
| `forum_user_tag_assignments` | 標籤指派 | 複合 PK `(user_email, tag_id)` |
| `forum_follows` | 追蹤關係 | 複合 PK `(follower_email, target_email)` |
| `forum_request_metrics` | 分鐘級請求統計 | PK `(bucket_minute)`；純衍生資料，可整表刪除 |
| `forum_admin_actions` | 管理員操作稽核 | `changes` 存欄位級 diff JSON；只能由時間清理 |
| `forum_announcements` | 站內公告 | 同時只有一列 `active`；`expires_at` 可空＝永不自動過期 |

**沒有 session 表** —— session 完全存在 Redis 的 `FORUM:session:<token>` hash 裡。
這一點是整個安全模型的關鍵，見 [`docs/SECURITY.md`](docs/SECURITY.md)。

稽核的三個性質（細節見 [ARCHITECTURE](docs/ARCHITECTURE.md#管理員操作稽核)）：

1. **稽核與操作在同一個交易裡。** `audit.Record` 回傳錯誤，而每個後台寫入 handler 都在
   `Commit()` **之前**檢查它 —— 因此「操作發生但沒有紀錄」在資料庫層不可能發生。
2. **欄位級 diff 存在 `changes`（JSON），每個值截到 200 bytes。** 完整內容仍在
   `forum_posts` / `forum_post_comments`，稽核表不重複一份可能含有個資的長文字。
3. **沒有任何刪除紀錄的 API。** 唯一清理途徑是時間式的 `audit.Pruner`。一個能刪除
   自己紀錄的稽核日誌等於沒有稽核日誌。

---

## API

回應格式一律 JSON，錯誤以非 2xx 狀態碼回傳。各端點的語意細節與取捨見
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)。

### 基礎

| Method | Path | 說明 |
| --- | --- | --- |
| GET | `/healthz` | ping MySQL / Redis / ES，全通 200 否則 503 |
| GET | `/api/check` | 永遠 200；回 `{ ok, isAdmin }` |
| GET | `/auth/google` | 307 導向 Google（**限流，所有 method**） |
| GET | `/auth/callback` | OAuth 交換 token 並種 session |
| POST | `/api/logout` | 需要可信 Origin |

### 論壇

讀取匿名可達；寫入需要登入並受限流。

| Method | Path | 說明 |
| --- | --- | --- |
| GET | `/api/forum/posts?offset=&limit=` | 貼文列表 |
| POST | `/api/forum/posts` | 建立貼文 |
| GET | `/api/forum/posts/{id}` | 單篇貼文（永久連結頁；匿名可讀，找不到回 404） |
| PUT | `/api/forum/posts/{id}` | 編輯自己的貼文本文（**只有 `content`**，見 [已知問題](docs/KNOWN_ISSUES.md)） |
| DELETE | `/api/forum/posts/{id}` | 刪除自己的貼文 |
| GET | `/api/forum/posts/{id}/comments` | 留言列表（公開） |
| POST | `/api/forum/posts/{id}/comments` | 建立留言 |
| PUT / DELETE | `/api/forum/posts/{id}/comments/{cid}` | 編輯／刪除自己的留言 |
| POST | `/api/forum/posts/{id}/like` | 按讚 |
| POST | `/api/forum/posts/{id}/report` | 檢舉貼文 |
| POST | `/api/forum/posts/{id}/comments/{cid}/report` | 檢舉留言 |
| POST | `/api/forum/images` | 上傳圖片，轉送 files_server |
| POST | `/api/forum/image-tokens/release` | 釋放媒體存取權杖 |
| GET / PUT | `/api/forum/profile` | 自己的資料 |
| GET / POST | `/api/forum/follows` | 追蹤名單 / 切換追蹤 |
| GET | `/api/forum/following/posts` | 私密動態牆（需登入，不限流） |
| GET | `/api/forum/my-posts` | 自己的貼文（需登入；未發過文回空清單而非 404） |
| GET | `/api/forum/public-profile?key=` | 以 `public_key` 查公開資料（無需登入） |
| GET | `/api/forum/public-posts?user=` | 以 `public_key` 查貼文（無需登入） |
| GET | `/api/forum/search?q=&offset=&limit=` | 全文搜尋；ES 不可用時退回 `LIKE` |

> 兩個 `public-*` 端點的參數名**不一致**（`key` vs `user`），這是現況不是筆誤。

### 管理

每個 handler 內部自行呼叫 `requireAdminForum`（**在 method 分派之前**，由 analyzer
強制），**刻意沒有限流**。

| Method | Path |
| --- | --- |
| GET / POST | `/api/admin/forum/posts` |
| GET / PUT / DELETE | `/api/admin/forum/posts/{id}` |
| POST | `/api/admin/forum/posts/{id}/pin` （body `{"pinned":true\|false}`） |
| GET / POST | `/api/admin/forum/comments` |
| GET / PUT / DELETE | `/api/admin/forum/comments/{id}` |
| GET / POST | `/api/admin/forum/reports?status=PENDING\|RESOLVED\|REJECTED` |
| GET / PUT / PATCH / DELETE | `/api/admin/forum/reports/{id}` |
| GET | `/api/admin/forum/search?page=` （含作者 email，僅管理員可見） |
| GET | `/api/admin/users` |
| PATCH | `/api/admin/users/{email}` （body `{"status":"ACTIVE"\|"SUSPENDED"}`） |
| GET | `/api/admin/users/{email}/content` |
| GET / PUT | `/api/admin/users/{email}/tags` |
| POST | `/api/admin/users/{email}/posts` **（以該使用者身分發文）** |
| POST | `/api/admin/users/{email}/comments` **（同上）** |
| GET / POST | `/api/admin/tags` |
| PATCH / DELETE | `/api/admin/tags/{id}` |
| GET | `/api/admin/monitor` |
| GET | `/api/admin/log?actor=&action=&targetType=&targetId=&from=&to=&offset=` |
| GET | `/api/admin/stats?days=` （預設 30 天，上限 90 天） |
| GET | `/api/admin/export/users.csv` · `posts.csv` · `reports.csv` |
| POST | `/api/admin/batch/tags` （body `{"emails":[…],"tagIds":[…]}`） |
| POST | `/api/admin/batch/status` （body `{"emails":[…],"status":"SUSPENDED"\|"ACTIVE"}`） |
| GET | `/api/admin/sessions?email=&limit=` |
| POST | `/api/admin/sessions/revoke` （body `{"email":…}`） |
| GET / POST | `/api/admin/blocks` （body `{"ip":…,"minutes":1440,"reason":"…"}`；`minutes <= 0` 代表解除） |
| GET | `/api/forum/announcement` （公開；沒有生效中的公告時回 `announcement: null`） |
| GET / POST | `/api/admin/announcements` |
| PATCH | `/api/admin/announcements/{id}` （body `{"body":…,"active":…,"hoursUntilExpiry":…}`） |

---

## 頁面路由

後端把多個 HTML 檔直接送給瀏覽器（這是 MPA 不是 SPA 的原因）。**17 個頁面殼**，
對應 `src/entries/` 的 17 個 entry 與 `vite.config.ts` 的 `rollupOptions.input`。

| Path | HTML 檔 | 需登入 |
| --- | --- | --- |
| `/` `/index.html` | — | 302 → `/forum` |
| `/forum` | `forum.html` | 否 |
| `/forum/new` | `forum-new.html` | 否（送出時需登入） |
| `/forum/login` | `forum-login.html` | 否 |
| `/forum/profile` | `forum-profile.html` | 是 |
| `/forum/others-profile` | `forum-others-profile.html` | 是 |
| `/forum/following` | `forum-following.html` | 是 |
| `/forum/post/{id}` | `forum-post.html` | 否（互動時需登入） |
| `/admin` | `admin.html` | 管理員 |
| `/admin/forum` | `forum-admin.html` | 管理員 |
| `/admin/forum-report` | `forum-report.html` | 管理員 |
| `/admin/monitor` | `forum-monitor.html` | 管理員 |
| `/admin/log` | `audit-log.html` | 管理員 |
| `/admin/stats` | `forum-stats.html` | 管理員 |
| `/admin/export` | `export.html` | 管理員 |
| `/admin/sessions` | `sessions.html` | 管理員 |
| `/admin/blocks` | `blocks.html` | 管理員 |
| `/admin/announcements` | `announcements.html` | 管理員 |

另有 `/service-worker.js`（附 `Service-Worker-Allowed: /`）、`/forum-manifest.json`、
`/assets/*`（內容雜湊，cache 一年）、`/asset/*`（PWA 圖示等原始檔，後端以
`http.Dir` 直接提供 `frontend/asset/`）。

站名會在送出頁面時注入 HTML 與 manifest，因此改 `FORUM_NAME` 不需要重新建置前端。

---

## 前端開發

```bash
cd frontend
npm install
npm run dev          # Vite dev server（預設 5173）
npm run typecheck    # tsc x2：主程式 + service worker
npm run build        # typecheck + vite build → dist/
npm run preview      # 預覽 dist/
```

`vite.config.ts` 有三個自訂行為：

- **`copyForumRuntimeAssets`**（`enforce: 'post'`）：用 esbuild 編譯 `service-worker.ts`
  成 IIFE，輸出**不帶 hash** 的 `/service-worker.js` 與 `/forum-manifest.json`
  （service worker 必須如此），並把 HTML 裡的 manifest 連結改回未雜湊版，讓後端可以在
  送出時替換 `{{FORUM_NAME}}`。
- **`vite-plugin-javascript-obfuscator`**（只在 `build`）：字串陣列旋轉 + control flow
  flattening。`debugProtection` / `selfDefending` 維持關閉，因為它們需要 `eval`，而 CSP
  是 `script-src 'self'`。
- **沒有設定 `server.proxy`**，見下。

### `npm run dev` 開箱即壞

前端所有 `fetch` 都是同源相對路徑（`/api/...`），但沒有 proxy，所以 dev server 起在
5173 時請求會打到 Vite 而不是 8088 的後端。要用 dev server + HMR 開發，先補上：

```ts
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8088',
    '/auth': 'http://localhost:8088',
    '/healthz': 'http://localhost:8088',
  },
},
```

不改的話就照正式模式走：`npm run build` 產出 `dist/`，再啟動後端由它提供靜態檔案。

### 批次翻譯

`tools/i18n/` 是一組 Node ESM 腳本，負責批次翻譯的切單與合併；代理模型只負責中間那一段
（讀 TSV、寫 TSV）。因此**翻譯品質的驗證不依賴代理模型自報** —— 鍵集、佔位符、未翻譯
殘留、字元損毀都由腳本把關。流程與設計理由見
[`tools/i18n/README.md`](tools/i18n/README.md)。

```bash
cd frontend    # 全部腳本必須從 frontend/ 執行

# 1. 切工作單：messages.ts → key<TAB>原文，切成每份約 96 行
node ../../tools/i18n/make-worklist.mjs <輸出目錄> <語言> [份數，預設 4]

# 2. 交給代理模型，一個任務一份：讀 <語言>-N.tsv，寫 <語言>-N.out.tsv

# 3. 合併：鍵集 / 佔位符 / 未翻譯殘留都過關才會寫出 TS 目錄檔
node ../../tools/i18n/merge-translations.mjs <輸出目錄> <語言> <匯出名稱> <檔名>
#    沒收齊會輸出 <語言>-todo.tsv，遞迴回第 1 或第 2 步
node ../../tools/i18n/split-todo.mjs <輸出目錄> <語言> [份數]

# 隨時驗證全部語系目錄（可從任何目錄執行）
node tools/i18n/verify-catalogs.mjs
```

現有語系（`frontend/src/i18n/translations/`）共 16 份，連同基準的 `zh-TW`
（`src/i18n/messages.ts`）共 **17 個**：`ar`（RTL）、`de`、`en`、`es`、`fr`、`hi`、`id`、
`ja`、`ko`、`pt-BR`、`ru`、`th`、`vi`、`zh-CN`、`zh-HK`、`zh-MO`。

驗證擋下的五種失敗之中，**`U+FFFD`（REPLACEMENT CHARACTER）那一條與其他四條性質不同**：
它不是任何一種語言的字，出現在目錄裡只代表某個字元在一次編碼往返中被解碼失敗並換掉
—— **原始位元組當場就丟了**，所以它既不能被翻譯修好，也不會因為換語系而消失。佔位符、
鍵集、順序在這種情況下全部正常，因此沒有其他檢查抓得到。修法是照 `messages.ts` 的原文
重寫整句，而不是去猜那個丟掉的字元。

### 新增一個頁面要動三處

`src/entries/` · `vite.config.ts` 的 `rollupOptions.input` · 後端
`httpapi.frontendShellFiles`（CSP 的 `<style>` SHA-256 授權）。**少一處的症狀不是
錯誤訊息，而是整頁沒有版面** —— `style.css` 仍會作用（它是 `<link>` 載入的外部檔案），
所以畫面看起來「有樣式但怪怪的」，瀏覽器主控台只有一行
`Refused to apply inline style`。`frontend_shells_test.go` 會指名缺哪一處。

---

## 測試與不變條件

後端與檔案服務都有 Go 測試，**前端沒有單元測試** —— 前端的測試是**型別檢查**加上
編譯期保證的翻譯完整性（`Record<MessageKey, string>` 讓漏翻譯與用錯鍵在編譯期就失敗）。

```bash
# 後端
cd backend
go test ./...                # 全域
go test ./forum/config/...   # 設定解析（限流兜底、管理員白名單正規化、TTL 守門）
go test ./forum/auth/...     # OAuth（開放轉向防護、state 往返、四種上游失敗）
go test ./forum/logger/...   # 欄位清洗（含日誌注入）、兩種格式、分級、併行安全
go test ./forum/session/...  # Session 列舉與撤銷（miniredis）
go test ./forum/ipban/...    # IP 封鎖名單（miniredis）
go test ./forum/audit/...    # 稽核（CSV 防護、佔位符、截斷）
go test ./forum/es/...       # ES 傳輸層（httptest 假伺服器）
go test .                    # 監聽逾時、優雅停止、-check 報告（package main）

# 檔案服務（上傳／刪除／token 驗證／路徑穿越）
cd files_server && go test ./...

# 需要真實 MySQL 的遷移測試（未設定 DSN 時會自動跳過）
FORUM_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/?parseTime=true&loc=UTC&multiStatements=true' \
  go test ./backend/forum/data/...

# 翻譯目錄與本文件的語系宣稱（可從任何目錄執行）
node tools/i18n/verify-catalogs.mjs
```

`verify-catalogs.mjs` 同時比對**本文件宣稱的語系數量與清單** —— 新增或刪除語系而沒
更新上面那兩處數字時，它會指名 README 的哪一處過期（CI 會跑這一步）。

### 不變條件 analyzer

這個專案有四條**專屬**的不變條件。它們約束的不是「型別對不對」或「有沒有測試」，
因此通用工具抓不到 —— 一個 handler 從 0% 變成 30% 覆蓋，與「`Commit` 之前有沒有
檢查稽核錯誤」毫無關係。

| 不變條件 | 由什麼守住 |
| --- | --- |
| 稽核與操作同生共死：`beginAdminTx` → **檢查** `recordAdminAction` 的錯誤 → `tx.Commit()` | `auditcheck` analyzer |
| `/api/admin/*` 的 handler 在 method 分派**之前**呼叫 `requireAdminForum` | `adminauth` analyzer |
| `withBlocklistHandler` 只在 `applyRateLimit` 內被呼叫（單一掛載點） | `blocklistmount` analyzer |
| 十七個 HTML 殼在三處同步 | `frontend_shells_test.go` |

前三條是 `tools/invariants` 這個獨立 Go 模組裡的 `go/analysis` analyzer。
**失敗模式是編譯失敗**（`go vet` 回報），而不是「有人沒注意到」。

```bash
cd tools/invariants && go build -o invariants .
cd ../../backend && go vet -vettool=<abs>/tools/invariants/invariants ./...
cd ../tools/invariants && go test ./...   # 含負向測試：刻意寫違規程式碼，確認會被報

# 覆蓋率地板（backend/tools/coveragefloor）
cd ../backend && go run ./tools/coveragefloor <合併後的 coverage.out> 25.0
```

它刻意是**獨立的 Go 模組**：它需要 `golang.org/x/tools`，而 backend 的相依項刻意維持
在四個（miniredis、mysql、go-redis、oauth2）。把分析工具的相依塞進 runtime 相依裡，
等於讓每個部署環境都多下載一份只有 CI 需要的程式碼。

第四條跨越 Go / TypeScript / HTML 三種語言，`go/analysis` 表達不了，因此它是**一個
測試**而不是 analyzer。兩種失敗模式（編譯失敗 vs 測試失敗）刻意不混在同一個工具裡，
以免搞混哪一條失效了。

### CI

`.github/workflows/ci.yml` 有七個 job：

| job | 做什麼 |
| --- | --- |
| `backend` | build、`go vet`、`go test -race`（CGO 開啟就是為了它） |
| `files_server` | 同上 |
| `frontend` | `npm ci`、`typecheck`、`build`、`verify-catalogs` |
| `coverage-floor` | 合併兩個模組的覆蓋率，檢查是否 ≥ **25.0** |
| `invariants` | 建置並執行分析器，加上分析器自測 |
| `migrations` | 起真實 MySQL 8.0.29，跑 `forum/data` 的遷移測試 |
| `secrets` | 確認 `config.conf` 從未被追蹤、範本檔不含明文憑證 |

兩件事值得知道：

- `secrets` 用 `fetch-depth: 0` 掃**整份歷史** —— 只掃工作區的話，一個「刪掉檔案」的
  commit 就足以讓真正的憑證留在倉庫裡。
- **覆蓋率地板的作用是防止退化，不是製造動機。** 刻意不設一個虛高的目標（例如 70%），
  因為那會誘發無行為斷言的測試來拉數字，而那種測試是負債：它會在重構時壞掉，卻沒有
  指出任何東西壞了。目前合併覆盖率約 **26.2%**；`forum/httpapi` 約 2,500 行、覆蓋
  率最低，是主要缺口。

### 靜態檢查與格式

這個專案**刻意沒有** gofmt / golangci-lint 的閘門。理由不是懶，而是它們會在第一次
執行時擋下 25 個既有檔案，而那些格式差異是**行尾（CRLF）**造成的 —— 在一個以 Windows
為主的開發環境裡，那會讓每一個開發者第一次推上去時都被擋，而他們完全無法理解原因。

`go vet` 有閘門（它找的是真問題，不是排版）。

---

## 部署

```mermaid
flowchart LR
  C[瀏覽器] -->|HTTPS| P[反向代理<br/>TLS 終結]
  P --> F[/forum /admin → :8088]
  P --> FS[/files/* → :7070]
  F --> DB[(MySQL)]
  F --> R[(Redis)]
  F -.選用.-> ES[(Elasticsearch)]
  F -->|上傳 + token| FS
  FS --> R
  FS --> S[(本機磁碟 或 S3)]
```

反向代理不在此 repo 內。務必確認：

1. **`COOKIE_SECURE=true`**，並且代理有處理 HTTPS。
2. **`PUBLIC_BASE_URL`、`GOOGLE_REDIRECT_URL` 與 `TRUSTED_ORIGINS` 對齊**實際對外網址。
3. **`FILES_SERVER_PUBLIC_URL` 指向對外可讀的位址**；`FILES_SERVER_URL` 可以是內部位址。
   兩者相同代表沒有內外網分離。
4. **Google Cloud Console 的授權 redirect URI** 與 `GOOGLE_REDIRECT_URL` 完全一致。
5. **代理必須放行 `Service-Worker-Allowed` 與 `/service-worker.js`**，否則 PWA 不會更新。
6. `/files/*` 若走代理並由瀏覽器直接取圖，代理**不得**吃掉 `?token=` 查詢參數。
7. **Redis 只能綁內網** —— 它是本站的信任根（見 [SECURITY](docs/SECURITY.md)）。
8. **停止訊號要送得對，而且要給夠時間。** 後端收到 `SIGTERM` / `SIGINT` 後會排空在途
   請求，上限是 `SHUTDOWN_TIMEOUT_SECONDS`（預設 15 秒）。部署環境的停止上限必須
   **大於**它：

   | 環境 | 預設停止上限 | 怎麼調 |
   | --- | --- | --- |
   | systemd | `TimeoutStopSec=90s` | 通常已足夠 |
   | Docker | `--stop-timeout=10` | `docker run --stop-timeout=20 …` 或 compose 的 `stop_grace_period` |
   | Kubernetes | `terminationGracePeriodSeconds=30` | 部署 manifest 內設定 |
   | 裸執行 | — | 會送出 `SIGINT`，Ctrl-C 即可 |

   另一個方向是讓反向代理**先**把流量抽掉再轉送停止訊號，那樣排空會是瞬間完成的事。

建置指令：

```bash
cd frontend && npm ci && npm run build     # 產出 frontend/dist（已 gitignore）
cd ../files_server && go build -o files-server .
cd ../backend && go build -o forum .
```

### 容器部署（docker compose）

`docker-compose.yml` 提供四個服務：`mysql`、`redis`、`backend`、`files_server`。
**沒有「前端」服務** —— 那是這個架構最刻意的一個決定：後端啟動時掃描前端產物、算出每個
HTML 殼裡 `<style>` 的 SHA-256 寫進 CSP，因此**被提供的檔案與被授權的雜湊必須來自
同一份建置輸出**。合成一個映像讓那個不變量由建置流程結構性保證。理由見
[ARCHITECTURE](docs/ARCHITECTURE.md#為什麼前端沒有獨立服務)。

```bash
# 0. 打包兩個服務映像（Windows：./deploy/build-images.ps1）
./deploy/build-images.ps1

# 1. 準備三份設定檔
cp deploy/settings.conf.example deploy/settings.conf       # 容器編排的環境變數
cp backend/config/config.conf.example backend/config/config.conf
cp files_server/config.conf.example files_server/config.conf

# 2. 部署前先驗證設定 —— 這是整份 compose 裡最值得做的一步
docker compose --env-file deploy/settings.conf run --rm backend /app/forum -check

# 3. 起服務
docker compose --env-file deploy/settings.conf up -d
docker compose ps && docker compose logs -f backend
```

**為什麼每一步都帶 `--env-file deploy/settings.conf`**：compose 檔裡的密碼寫成
`${MYSQL_ROOT_PASSWORD:?…}`，而 compose 只會自動讀專案根目錄的 `.env`。
`deploy/settings.conf` 不是 `.env`，因此不傳就會在**開始建置之前**結束。

**兩個 config.conf 掛載到不同位置**：`backend` 讀 `/app/config/config.conf`，
`files_server` 讀 `/app/config.conf`（後者是工作目錄下的檔名）。compose 檔裡已經
分開寫好，不要照抄成同一個。

容器部署**必須額外設定** `TRUSTED_PROXY_CIDRS=172.28.0.0/16`（寫在
`backend/config/config.conf`）。留空會讓「無條件採信 `X-Forwarded-For`」生效，而
`forum-net` 內的**任何容器**都能直接連 8088 —— 它只要自己塞一個標頭就能讓限流計數記到
別的 IP 上：**症狀是「限流看起來有開，只是擋不住任何人」**。那個 subnet 寫死在
`docker-compose.yml` 的 `ipam` 設定裡，所以這裡可以直接抄。

停止行為與 systemd 部署的細節見 [`docs/OPERATIONS.md`](docs/OPERATIONS.md#停止行為)。

---

## 安全模型

完整威脅模型見 [`docs/SECURITY.md`](docs/SECURITY.md)。這裡是最需要先知道的幾條：

- **Redis 就是信任根。** Session 不在資料庫，而在 Redis 的 `FORUM:session:<token>`
  hash（欄位 `email` + `is_admin`）。cookie 內**沒有簽章** —— 它只是一個查詢鍵，因此
  **能讀 Redis 就等於能登入任何人，能寫 Redis 就等於能自己開管理員 session**。
  密碼必須獨立、Redis 必須綁內網。這不是理論：`SCAN FORUM:session:*` 會直接列出
  當下所有有效 session token。
- **管理員身分不看資料庫。** 沒有 role 欄位；`is_admin` 在**登入當下**由
  `ALLOWED_ADMIN_EMAIL` 比對 Google 回傳的 email 決定後寫死進 session。把 email 從
  白名單移走**不會撤銷既有 session** —— 必須用 `POST /api/admin/sessions/revoke`
  或等 72 小時滑動 TTL 到期。
- **CSP**：`script-src 'self'`，沒有 `unsafe-inline`、沒有 `unsafe-eval`。HTML 內的
  `<style>` 區塊在啟動時算 SHA-256 寫進 `style-src`，所以樣板不能隨意新增內嵌樣式。
- **CSRF**：寫入端點檢查 `Origin` 是否在 `TRUSTED_ORIGINS` 內。**但 `Origin` 與
  `Referer` 都沒有時會刻意放行**，因此這道防護對非瀏覽器客戶端（curl、腳本）無效 ——
  真正的底線是 `SameSite=Lax` cookie 加上後端的身分檢查。
- **匿名性**：email 不出現在任何 URL，對外一律用 `public_key = SHA256(email)`。
- **檔名**：落地一律 UUID v4 + 原副檔名，不使用使用者提供的檔名；副檔名白名單
  **不含 svg**（SVG 可內嵌腳本）。
- **媒體權杖**：圖片網址綁短 TTL 的 Redis token，前端離開頁面時可主動釋放。
- **Slowloris**：`READ_HEADER_TIMEOUT_SECONDS`（預設 10 秒）+ `IdleTimeout`。
  `ReadTimeout` / `WriteTimeout` **刻意不設**，理由見 [已知問題](docs/KNOWN_ISSUES.md)。
- **密鑰不進版控**：`config.conf` 被 gitignore。唯一例外是 `FILES_SERVER_TOKEN`
  可用環境變數注入。

---

## 目前最需要注意的三件事

完整清單（約 35 條，含每一條的取捨理由）見
[`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md)。最容易在維運時咬人的三條：

1. **`TRUSTED_PROXY_CIDRS` 留空時，限流與封鎖都可被單一偽造標頭繞過。** 症狀是
   「限流看起來有開，只是擋不住任何人」—— 輪替 `X-Forwarded-For` 就重置額度。
   監控頁的 `clientIpTrust.mode` 會顯示 `legacy-headers`，那就是沒設定的訊號。
2. **`httpapi` 約 2,500 行、仍有約 88% 沒有測試覆蓋。** 優先順序建議依「壞掉時的
   爆炸半徑」排：批次／匯出 → 貼文 CRUD → 公告 → 統計。
3. **`docker compose` 尚未在本機實測過**（撰寫本文時的開發環境沒有可用的 Docker
   daemon）。映像與 compose 檔只經過靜態檢查。第一次實際部署時請特別留意
   `docker compose run --rm backend -check` 的輸出。

---

## 授權

見 [`LICENSE`](LICENSE)。
