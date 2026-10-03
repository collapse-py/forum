# SB 論壇

一個免註冊、以 Google 帳號登入的匿名論壇。後端是只用標準函式庫寫的 Go HTTP 服務
（`net/http` + `database/sql`，沒有 web framework、沒有 ORM），前端是 Vite + React 19 +
TypeScript 的多頁應用（PWA），另有獨立的 Go 檔案服務處理媒體存儲。

論壇本身匿名：登入後的對外身份是 `SHA256(email)` 算出來的 `public_key`，公開頁面與
URL 裡都不會出現 email。管理員的身分不看資料庫角色表，只看 `ALLOWED_ADMIN_EMAIL`
白名單。

---

## 目錄

- [功能](#功能)
- [架構](#架構)
- [環境需求](#環境需求)
- [快速開始](#快速開始)
- [設定檔](#設定檔)
- [資料庫](#資料庫)
- [API](#api)
- [頁面路由](#頁面路由)
- [前端開發](#前端開發)
- [測試](#測試)
- [部署](#部署)
- [安全設計](#安全設計)
- [批次翻譯的作法](#批次翻譯的作法)
- [已知問題](#已知問題)

---

## 功能

- **匿名發文**：文字貼文，可選一張圖片；刪除自己的貼文
- **編輯自己的貼文與留言**：兩者都在卡片上就地編輯（不用跳頁），並標示
  「已編輯」；編輯後的貼文會同步更新搜尋索引，因此不會出現「改了字、
  搜尋還是舊字」。替換附圖刻意**不在**這個端點的語意裡（見已知問題）
- **永久連結**：`/forum/post/{id}` 是單篇貼文頁，貼文卡的 ⋯ 選單可直接前往。
  該頁刻意不需登入 —— 分享連結的人不會先跟收連結的人說「請先登入」
- **草稿自動儲存**：新增貼文頁的內文會 debounce 寫進 localStorage，
  關掉分頁再回來仍在（只有文字；選好的圖片無法序列化，因此不保留）
- **留言、按讚、檢舉**：留言公開可讀，也可由作者自己編輯或刪除；按讚以
  `(post_id, author_email)` 複合主鍵限制每人一次；檢舉涵蓋貼文與留言
- **追蹤與私密動態牆**：`/api/forum/following/posts` 只回傳追蹤中的人的貼文
- **Google OAuth2 登入**：僅申請 `userinfo.email`，沒有密碼、沒有本機帳號
- **Redis 版可撤銷 Session**：滑動續期，Redis 掛掉等於全部登入失效
- **管理後台**：貼文／留言 CRUD、檢舉佇列裁決（通過即刪除對象、駁回保留）、
  帳號停權與復原、使用者標籤字典與指派、以使用者身分代發文
- **系統監控**：`/admin/monitor` 顯示 MySQL／Redis／搜尋引擎狀態、Go 執行期用量、
  依正規化路由統計的請求量與延遲分位數、限流器計數、依請求量排序的來源位址清單
  （可一鍵封鎖，並標示位址是取自連線對端還是可偽造的標頭），以及分鐘級流量圖
  （記憶體內 120 分鐘 + 資料庫保留 24 小時，重啟後時間軸不會歸零）
- **管理員操作稽核**：`/admin/log` 記錄後臺每一項會改變資料的操作，包含欄位級
  diff（哪個欄位從什麼變成什麼）。稽核寫入與操作在同一個交易裡，因此不會出現
  「操作發生但沒有紀錄」；保留 90 天，沒有任何刪除紀錄的 API
- **內容趨勢統計**：`/admin/stats` 顯示每日新增的使用者／文章／留言、視窗內合計，
  以及熱門文章、熱門標籤、活躍作者三份排行。視窗 7～90 天可切換
- **匯出與批次**：`/admin/export` 提供使用者／文章／檢舉三份 CSV（對公式注入
  有防護），用戶管理頁可多選後批次停權、批次套標籤
- **登入與 Session 管理**：`/admin/sessions` 列出目前仍有效的登入狀態（只顯示
  token 前 8 個字元），可強制登出某個帳號的所有裝置。Session 的建立時間從
  這個功能開始才會記錄，既有 session 顯示為「未知」
- **IP 封鎖**：`/admin/blocks` 把確認濫用的來源位址加進存在 Redis 的封鎖名單
  （跨行程、跨重啟都存活），並可隨時解除。刻意**不會**自動封鎖
- **站內公告與置頂**：`/admin/announcements` 發佈全站橫幅（同時只有一則生效），
  公開頁首頁最上方顯示；文章可在「論壇文章」管理頁逐篇置頂
- **Elasticsearch 全文搜尋**：`ES_URL` 留空則自動退回 MySQL `LIKE`
- **PWA**：可安裝、有 service worker 與 manifest，頁面 network-first、
  靜態資源 cache-first
- **多語系**：17 個語系（含 `zh-TW` 為基準），阿拉伯文為 RTL
- **三段式 IP 限流**：內容寫入、上傳、OAuth 各自獨立額度
- **媒體存取權杖**：上傳時發短 TTL 的 Redis token 綁定圖片網址，可主動釋放

---

## 架構

```
                   ┌──────────────────────────────────────────┐
   瀏覽器 ────────▶ │ backend/  (Go, :8088)                    │
   PWA             │                                          │
   /forum          │  httpapi/  路由、中介層、security headers  │
   /admin          │  auth/     Google OAuth2                  │
│  session/  Redis session                  │
                    │  ipban/    IP 封鎖名單（Redis sorted set）  │
                    │  metrics/  請求統計、分鐘彙總持久化         │
                    │  audit/    管理員操作稽核、依時間清理        │
                    │  data/     MySQL + MigrateMySQL           │
                   │  es/       Elasticsearch（選用）           │
                   │  ─ 服務 frontend/dist/ 的靜態檔案 ─      │
                   └───────┬──────────────────┬───────────────┘
                           │ SQL              │ HTTP + token
                           ▼                  ▼
                  ┌────────────────┐   ┌──────────────────────┐
                  │ MySQL  :3306   │   │ files_server/ (Go,   │
                  │ utf8mb4/InnoDB │   │ :7070) 檔案服務       │
                  ├────────────────┤   │  local / S3 存儲      │
                  │ Redis  :6379   │◀─▶│  Redis media token   │
                  │ ES     :9200   │   └──────────────────────┘
                  │ （選用）        │
                  └────────────────┘
```

兩個 Go 模組是獨立的，各自的 `go.mod` 與設定檔：

| 目錄 | 模組名 | Go 版本 | 設定檔 | 連接埠 | 職責 |
| --- | --- | --- | --- | --- | --- |
| `backend/` | `forum` | 1.25.0 | `config/config.conf` | 8088 | 全部 API、認證、頁面與靜態檔 |
| `files_server/` | `files_server` | 1.26.0 | `config.conf` | 7070 | 媒體上傳／刪除／讀取 |

---

## 環境需求

| 需求 | 版本 | 必要？ |
| --- | --- | --- |
| Go | 1.25+（files_server 需 1.26） | 必要 |
| Node.js | 20+ | 建置前端時必要 |
| MySQL | **8.0.29+**，建一個 `utf8mb4` 資料庫 | 必要（schema 自動建立） |
| Redis | 5+ | 必要（session 與 media token） |
| Elasticsearch | 7.x / 8.x | 選用，留空則用 MySQL `LIKE` 搜尋 |
| Google OAuth2 用戶端 | — | 登入與管理功能需要 |

MySQL 的下限是 8.0.29 而不是籠統的「8.x」，因為 `MigrateMySQL` 用到
`ALTER TABLE ... ADD COLUMN IF NOT EXISTS` —— 這個語法在 8.0.29 之前不存在
（8.0.29 之前只有 `CREATE TABLE IF NOT EXISTS` 與 `DROP COLUMN IF EXISTS`），
而 `forum_posts.pinned` 正是這樣加上去的。在較舊的 8.0 上啟動會直接拿到
`ERROR 1064 (42000): You have an error in your SQL syntax`，遷移回錯、
`main` 呼叫 `Fatalf`，行程在啟動時就死。遞減索引（置頂那條）本身只要 8.0+。

---

## 快速開始

### 1. 準備設定檔

兩個 `config.conf` 都在 `.gitignore` 裡，必須各自從範本複製出來。範本已提交，
且不含任何真實憑證：

```bash
# Linux / macOS
cp backend/config/config.conf.example backend/config/config.conf
cp files_server/config.conf.example  files_server/config.conf

# Windows PowerShell
Copy-Item backend\config\config.conf.example backend\config\config.conf
Copy-Item files_server\config.conf.example  files_server\config.conf
```

兩份範本都附有逐項註解（`backend/config/config.conf.example` 167 行、
`files_server/config.conf.example` 129 行），以下是最小可用設定。

**`backend/config/config.conf`**

```ini
FORUM_NAME=SB 論壇
FORUM_SHORT_NAME=SB
SERVER_PORT=:8088
PUBLIC_BASE_URL=http://localhost:8088
TRUSTED_ORIGINS=http://localhost:8088
# 只有在確實擋在一道反向代理後面時才填；見「來源 IP 的信任模型」
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

> `token_key_prefix` 兩邊必須一致，且要與後端 session 的 `forum:session:`
> 前綴區隔。不一致的話上傳會成功，但圖片 GET 會一律 401。

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

啟動順序是固定的：讀設定 → 初始化 logger → **驗證設定（`config.Validate`）**
→ ping Redis → 開 MySQL → `data.MigrateMySQL` → `auth.Init` →
`session.NewManager` → 註冊路由 → 啟動限流清理 → 背景重建 ES 索引 → 監聽。

開瀏覽器到 <http://localhost:8088/forum>（`/` 會 302 到這裡）。

> **一定要在 `backend/` 目錄下執行。** 設定檔路徑是 `config/config.conf`
> （相對於工作目錄而不是執行檔所在目錄），在別的目錄跑會直接 fatal。
> 用 `-config <路徑>` 可以覆寫它。

後端執行檔有三種模式，都是同一支執行檔：

| 旗標 | 做什麼 | 連線任何相依服務？ | 退出碼 |
| --- | --- | --- | --- |
| （無） | 正常啟動服務 | 是 | 0（正常停止）／1（監聽失敗） |
| `-check` | 輸出設定報告後結束 | **否** | 0（通過）／1（有問題） |
| `-healthz` | 探測 MySQL 與 Redis 後結束 | 是（僅探測） | 0（健康）／1（不健康） |

停止後端按 Ctrl-C（或 `kill`／`systemctl stop`／`docker stop`）即可：收到
`SIGINT` / `SIGTERM` 之後會停止接受新連線、等在途請求完成（上限
`SHUTDOWN_TIMEOUT_SECONDS`）、寫出最後一次分鐘彙總，然後以 exit code 0 結束。
超過上限仍有請求沒完成時會記警告並強制中斷 —— 那時 `docker stop` 預設 10 秒的
停止上限通常會先到，所以容器部署時要一併把 `stop_grace_period` 調大（見「部署」）。

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
| `COOKIE_SECURE` | `false` | 走 HTTPS 就必須 `true` |
| `SERVER_PORT` | `:8088` | `net/http` 的 `host:port` |
| `READ_HEADER_TIMEOUT_SECONDS` | `10` | 讀完**請求標頭**的期限。只卡標頭不卡本文（貼文附圖本文可達 50 MB 且要轉送檔案伺服器）。0 與負值不被接受 |
| `SHUTDOWN_TIMEOUT_SECONDS` | `15` | 收到停止訊號後等待在途請求的秒數，超過即強制中斷。必須小於部署環境的停止上限 |
| `PUBLIC_BASE_URL` | `http://localhost` + `SERVER_PORT` | 尾斜線會被自動去掉 |
| `TRUSTED_ORIGINS` | `[PUBLIC_BASE_URL]` | CSRF 來源白名單。**留空等於關閉 CSRF 防護** |
| `TRUSTED_PROXY_CIDRS` | 空 | 可信任反向代理的位址段。留空 = 舊的「標頭優先」行為（限流與封鎖可被偽造標頭繞過）。**容器部署必須填 `172.28.0.0/16`**，理由見「容器部署」 |
| `DB_DSN` | — | 內含帳密 |
| `DB_MAX_OPEN_CONNS` | — | 連線池上限 |
| `DB_MAX_IDLE_CONNS` | — | 連線池閒置數 |
| `DB_CONN_MAX_LIFETIME_MINUTES` | — | 連線回收時間 |
| `REDIS_ADDR` | `127.0.0.1:6379` | |
| `REDIS_PASSWORD` | 空 | |
| `REDIS_DB` | `0` | |
| `ES_URL` | 空（停用） | 設定了就啟用 ES 搜尋 |
| `ES_INDEX` | `forum_posts` | |
| `FILES_SERVER_URL` | `http://localhost:7070` | 內部上傳目標 |
| `FILES_SERVER_PUBLIC_URL` | 同上 | 寫進資料庫的對外網址 |
| `FILES_SERVER_TOKEN` | 讀環境變數 `FILES_SERVER_TOKEN` | **唯一支援環境變數的設定** |
| `MEDIA_TOKEN_KEY_PREFIX` | `forum:token:` | 必須與 files_server 的 `token_key_prefix` 相同 |
| `MEDIA_TOKEN_TTL_SECONDS` | `2592000`（30 天） | 這個兜底值對正式環境明顯過長，因此**正式環境採用兜底值會讓啟動被拒絕**（`-check` 會指出；開發環境不受影響）。寫出你想要的數字即可 —— 刻意設成 2592000 會被視為知情同意 |
| `RATE_LIMIT_REQUESTS` | `10` | 內容寫入 |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` | |
| `RATE_LIMIT_UPLOAD_REQUESTS` | `5` | 圖片上傳（最貴，額度最緊） |
| `RATE_LIMIT_UPLOAD_WINDOW_SECONDS` | `60` | |
| `RATE_LIMIT_AUTH_REQUESTS` | `10` | OAuth 端點，含所有 method |
| `RATE_LIMIT_AUTH_WINDOW_SECONDS` | `60` | |
| `GOOGLE_CLIENT_ID` | — | |
| `GOOGLE_CLIENT_SECRET` | — | |
| `GOOGLE_REDIRECT_URL` | — | 必須與 Google Cloud Console 完全一致 |
| `ALLOWED_ADMIN_EMAIL` | 空 | 逗號分隔。**載入時會去除每項前後空白並轉小寫，比對時對傳入值做同樣的正規化** —— 因此帶空白或大寫不會再靜默失效 |
| `LOG_LEVEL` | `INFO` | |
| `LOG_FILE` | `server.log` | |
| `LOG_FORMAT` | `text` | 或 `json` |
| `MONITOR_RETENTION_HOURS` | `24` | 監控分鐘彙總在資料庫保留幾小時；非正值會退回 24。**不影響時間軸長度**（見下方） |
| `AUDIT_RETENTION_DAYS` | `90` | 稽核紀錄保留幾天；非正值會退回 90 |

三組限流預設值刻意不同，反映各端點的真實成本；`applyDefaults` 對「空值或非正數」
補值，因此無法用設定檔把某個視窗設成 0 秒（那會讓限流失效）。布林值接受
`1` / `true` / `yes` / `on`（不分大小寫）。

`MONITOR_RETENTION_HOURS` 決定 `forum_request_metrics` 在資料庫留多久（過期列由
`metrics` 套件的背景 goroutine 每 20 秒清一次）。給一個有限的值是刻意的：這張表
每分鐘都會長出一列，只增不減的監控資料在幾個月後就會變成資料庫裡最大的一張表，
而它的價值隨時間急遽下降 —— 維運看的是「最近幾小時」。

**它不決定監控頁時間軸的長度。** 時間軸固定是 `metrics.Options.WindowMinutes`
分鐘（`NewServer` 目前設 120，也就是兩小時），再被 `maxTimelinePoints = 144`
收斂；啟動時讀回歷史的範圍也對齊這個長度，因此不會有多讀進來又被丟掉的資料。
兩個設定刻意分開：保留期是「寫多少進資料庫」的維運政策，時間軸長度是「畫幾格」
的呈現選擇 —— 混為一談的後果是設定 retention 的人以為圖會變長。不設這個值時，
記憶體裡仍然有 120 分鐘的即時資料，時間軸只是不會有重啟前的歷史。

`AUDIT_RETENTION_DAYS` 決定 `forum_admin_actions` 留多久。給 90 天是因為稽核
紀錄的用途是「上季有人動過什麼」，跨月保留才有稽核意義；但不能無限保留 ——
這張表保存了使用者的文字片段（截斷後），留得愈久愈像一份沒有用途的個人資料
備份。與 `MONITOR_RETENTION_HOURS` 相反，這個值給得愈長愈好。

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
| `[upload] allowed_files` | 副檔名白名單 |
| `[redis] addr, password, db` | |
| `[redis] token_key_prefix, token_ttl_sec` | 媒體權杖驗證用，須與後端一致 |
| `[redis] public_files` | `true` 則 `/files/*` 不需要 token |
| `[logger]`, `[cors]` | |

檔案落地時一律用 **UUID v4 + 原副檔名（小寫）**，永遠不用使用者提供的檔名。

---

## 資料庫

MySQL，`utf8mb4` / InnoDB。**Schema 在啟動時自動建立**：
`data.MigrateMySQL` 是一連串 idempotent 步驟（`CREATE TABLE IF NOT EXISTS`、
`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`、`INSERT IGNORE`），
所以空資料庫直接啟動即可，不需手動匯入 SQL。沒有版本化的 migration，也沒有降級路徑。

| 資料表 | 用途 | 關鍵結構 |
| --- | --- | --- |
| `forum_posts` | 貼文 | `image_url` 只存**檔名**，讀取時才用 `FILES_SERVER_PUBLIC_URL` 組網址；`updated_at` 可空＝從未被作者編輯過 |
| `forum_post_comments` | 留言 | 無外鍵，完整性靠應用層；`updated_at` 的語意同上 |
| `forum_post_likes` | 按讚 | 複合 PK `(post_id, author_email)`，天然去重 |
| `forum_reports` | 檢舉 | `target_type`（`post`/`comment`）+ `target_id` 的多型參照，無外鍵 |
| `forum_profiles` | 公開資料 | `public_key` = `SHA2(author_email, 256)`；`nickname` 唯一 |
| `forum_users` | 帳號狀態 | `status` 為 `ACTIVE` / `SUSPENDED` |
| `forum_user_tags` | 標籤字典 | `name` 唯一 |
| `forum_user_tag_assignments` | 標籤指派 | 複合 PK `(user_email, tag_id)` |
| `forum_follows` | 追蹤關係 | 複合 PK `(follower_email, target_email)` |
| `forum_request_metrics` | 分鐘級請求統計 | PK `(bucket_minute)`；純衍生資料，可整表刪除 |
| `forum_admin_actions` | 管理員操作稽核 | `changes` 存欄位級 diff JSON；只能由時間清理 |
| `forum_announcements` | 站內公告 | 同時只有一列 `active`；`expires_at` 可空＝永不自動過期 |

刪除 `forum_reports` 的 `uq_forum_reports_reporter_target` 與
`idx_forum_reports_status_created` 是必要的：前者擋同一使用者重複檢舉同一對象，
後者讓「待處理檢舉」列表不必掃全表。

`forum_request_metrics` 沒有欄位明細表，只有「每分鐘的總量、4xx、5xx、耗時總和」。
這是刻意的：依路由分拆的明細會隨路由數量線性成長，而維運真正會回頭查
「三小時前那波流量來自哪條路由」的機率很低 —— 記憶體裡的 120 分鐘即時資料
已經涵蓋那個情境。存耗時「總和」而非「平均」是因為平均在累加多個執行個體的
分鐘時無法正確合併（`(10+20)/2 ≠ (30+40)/2`），而總和可以在查詢時用
`SUM()/COUNT()` 得到正確的加權平均。

`forum_admin_actions` 記錄後臺**每一項會改變資料的操作**：停權／恢復、改標籤、
代發文／代留言、貼文與留言的建立／修改／刪除、檢舉的建立／裁定／修改／刪除、
標籤字典的建立／改名／刪除。存取日誌做不到這件事 —— 它只記到「某個 IP 對
`/api/admin/forum/reports/9` 送出 DELETE」，不包含「那筆檢舉被裁定為成立」還是
「被刪掉的是哪一篇文」。

三個性質值得知道：

- **稽核與操作在同一個交易裡。** `audit.Record` 回傳錯誤，而每個後臺寫入
  handler 都在 `Commit()` **之前**檢查它 —— 因此「操作發生但沒有紀錄」在
  資料庫層不可能發生。這件事很容易被無聲破壞：Go 的 `database/sql` 不會因為
  交易內某個語句失敗就中止交易，所以呼叫了 `recordAdminAction` 卻不看回傳
  值，等於把稽核降級成「盡力記錄」。搜尋 `beginAdminTx` 會列出全部需要稽核
  的寫入點（搜尋 `s.db.ExecContext` 則會混進二十幾個唯讀查詢）。
- **欄位級 diff 存在 `changes`（JSON），每個值截到 200 bytes。** 截斷是刻意的：
  稽核要回答「這段文字被改成了什麼」，但不是為了備份全文。完整內容仍在
  `forum_posts` / `forum_post_comments`，稽核表不重複一份可能含有個資的長文字。
  介面上會標示「已截斷」—— 不標的話，讀者會以為看到的就是全文，而那正是
  稽核紀錄最不能出現的誤解。
- **沒有任何刪除紀錄的 API。** 唯一的清理途徑是時間式的 `audit.Pruner`
  （保留 `AUDIT_RETENTION_DAYS` 天，預設 90，每小時清一次）。一個能刪除自己
  紀錄的稽核日誌等於沒有稽核日誌。

---

## API

回應格式一律 JSON。錯誤以非 2xx 狀態碼回傳。

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
| PUT | `/api/forum/posts/{id}` | 編輯自己的貼文本文（只有 `content`） |
| DELETE | `/api/forum/posts/{id}` | 刪除自己的貼文 |
| GET | `/api/forum/posts/{id}/comments` | 留言列表（公開） |
| POST | `/api/forum/posts/{id}/comments` | 建立留言 |
| PUT | `/api/forum/posts/{id}/comments/{cid}` | 編輯自己的留言 |
| DELETE | `/api/forum/posts/{id}/comments/{cid}` | 刪除自己的留言 |
| POST | `/api/forum/posts/{id}/like` | 按讚 |
| POST | `/api/forum/posts/{id}/report` | 檢舉貼文 |
| POST | `/api/forum/posts/{id}/comments/{cid}/report` | 檢舉留言 |
| POST | `/api/forum/images` | 上傳圖片，轉送 files_server |
| POST | `/api/forum/image-tokens/release` | 釋放媒體存取權杖 |
| GET/PUT | `/api/forum/profile` | 自己的資料 |
| GET/POST | `/api/forum/follows` | 追蹤名單 / 切換追蹤 |
| GET | `/api/forum/following/posts` | 私密動態牆（需登入，不限流） |
| GET | `/api/forum/my-posts` | 自己的貼文（需登入，不限流；未發過文回空清單而非 404） |
| GET | `/api/forum/public-profile?key=` | 以 `public_key` 查公開資料（無需登入，不限流） |
| GET | `/api/forum/public-posts?user=` | 以 `public_key` 查貼文 |
| GET | `/api/forum/search?q=&offset=&limit=` | 全文搜尋；ES 不可用時退回 `LIKE` |

### 管理

每個 handler 內部自行呼叫 `requireAdminForum`，**刻意沒有限流**。

| Method | Path |
| --- | --- |
| GET/POST | `/api/admin/forum/posts` |
| GET/PUT/DELETE | `/api/admin/forum/posts/{id}` |
| GET/POST | `/api/admin/forum/comments` |
| GET/PUT/DELETE | `/api/admin/forum/comments/{id}` |
| GET/POST | `/api/admin/forum/reports?status=PENDING\|RESOLVED\|REJECTED` |
| GET/PUT/PATCH/DELETE | `/api/admin/forum/reports/{id}` |
| GET | `/api/admin/forum/search?page=` （含作者 email，僅限管理員可見的欄位） |
| GET | `/api/admin/users` |
| PATCH | `/api/admin/users/{email}` （body `{"status":"ACTIVE"\|"SUSPENDED"}`） |
| GET | `/api/admin/users/{email}/content` |
| GET/PUT | `/api/admin/users/{email}/tags` |
| POST | `/api/admin/users/{email}/posts` （以該使用者身分發文） |
| POST | `/api/admin/users/{email}/comments` |
| GET/POST | `/api/admin/tags` |
| PATCH/DELETE | `/api/admin/tags/{id}` |
| GET | `/api/admin/monitor` |
| GET | `/api/admin/log?actor=&action=&targetType=&targetId=&from=&to=&offset=` |
| GET | `/api/admin/stats?days=` （預設 30 天，上限 90 天） |
| GET | `/api/admin/export/users.csv` |
| GET | `/api/admin/export/posts.csv` |
| GET | `/api/admin/export/reports.csv` |
| POST | `/api/admin/batch/tags` （body `{"emails":[…],"tagIds":[…]}`） |
| POST | `/api/admin/batch/status` （body `{"emails":[…],"status":"SUSPENDED"\|"ACTIVE"}`） |
| GET | `/api/admin/sessions?email=&limit=` |
| POST | `/api/admin/sessions/revoke` （body `{"email":…}`） |
| GET | `/api/admin/blocks` |
| POST | `/api/admin/blocks` （body `{"ip":…,"minutes":1440,"reason":"…"}`；`minutes <= 0` 代表解除） |
| GET | `/api/forum/announcement` （公開；沒有生效中的公告時回 `announcement: null`） |
| GET/POST | `/api/admin/announcements` |
| PATCH | `/api/admin/announcements/{id}` （body `{"body":…,"active":…,"hoursUntilExpiry":…}`） |
| POST | `/api/admin/forum/posts/{id}/pin` （body `{"pinned":true\|false}`） |

### 系統監控

`GET /api/admin/monitor` 是監控頁（`/admin/monitor`）唯一的資料來源，一次回傳：

| 區塊 | 內容 |
| --- | --- |
| `stats.runtime` | Go 版本、goroutine 數、CPU 核心數、堆積／堆內用量、GC 次數與最近一次暫停 |
| `stats.requests` | 本次啟動以來的請求總數、4xx／5xx、進行中與峰值並行數、平均與 P50／P95／P99／最慢 |
| `stats.requests.routes` | 依正規化路由分組的同樣統計（數字與 email 段換成 `:id`，上限 200 條） |
| `stats.timeline` | 近 120 分鐘的分鐘級彙總，每格標示 `live`（本次啟動）或 `history`（資料庫讀回） |
| `stats.clients` | 依請求量排序的來源位址清單：位址、`source`（位址怎麼來的）、4xx／5xx、429 次數、被封鎖名單擋下的次數、最近打到哪條路由 |
| `stats.rateLimits` | 三組限流器的額度、放行／阻擋累計、目前追蹤中的來源數 |
| `dependencies` | MySQL 連線池水位與 ping 延遲、Redis 鍵數／記憶體／連線池、搜尋引擎狀態 |
| `clientIpTrust` | 目前生效的來源 IP 信任模型：`mode`、`trusted`（實際生效的位址段）、`declared`（設定檔原文） |

三件讀這個端點時要知道的事：

- **延遲分位數是直方圖的桶上界，不是精確值。** 邊界固定為
  1/2/5/10/25/50/100/250/500ms 與 1/2/5/10s，因此 P95 只會落在這些刻度上。
  這是刻意的取捨：精確分位數要保存每次請求的耗時，記憶體會隨流量線性成長。
- **總量只算本次啟動。** 記憶體裡沒有跨行程的累計，時間軸上「重啟前」的資料是從
  資料庫讀回來的（`source: "history"`），與重啟後的資料（`source: "live"`）在圖上以
  虛線框區分。時間軸固定 `stats.windowMinutes` 分鐘（目前 120），與
  `MONITOR_RETENTION_HOURS` 無關 —— 讀回歷史的範圍也對齊這個長度。
- **`stats.clients` 的位址不一定可信，而且它只活在記憶體裡。** 兩件事要分開看：

  1. **位址的來源**由 `source` 標明：`peer` 取自 `RemoteAddr`（TCP 連線對端），
     `xff` 與 `real-ip` 取自 `X-Forwarded-For`／`X-Real-IP`。後兩者**只有在
     `TRUSTED_PROXY_CIDRS` 已設定且對端確實在白名單裡時**才代表真人；留空時它們
     可被使用者偽造，而限流、封鎖與稽核紀錄用的是同一個值，也就是偽造一個標頭
     等於同時繞過三者。同時被標成 `xff` 且 `clientIpTrust.mode` 是
     `legacy-headers`，就是「這台站還沒設定白名單」的訊號。
  2. **這份資料不寫進資料庫。** 分鐘彙總會落盤（`forum_request_metrics`），每 IP
     統計不會 —— 「某個位址在什麼時候打了本站」是一筆個人資料，落盤等於多一份永久
     訪客紀錄，而這一頁需要的只是「現在正在怎樣」。重啟後歸零是刻意的。

  記憶體上限 200 個來源，超過時**驅逐最久沒再出現的那一個**（不是丟棄新的：丟棄會
  讓輪換位址的攻擊者把真實使用者擠出清單，而畫面看起來只是「今天沒什麼人來」）。
  被驅逐的次數記在 `stats.clientsDropped`，介面會把它顯示出來 —— 沒有那個數字，
  一份被截斷的清單看起來會和一份完整的一模一樣。

依賴探測每個 2 秒逾時、三者並行，總耗時回在 `probesMs`。探測失敗不會讓整個
請求回 5xx —— 依賴狀態放在 body 裡、狀態碼維持 200，因為管理員正是最需要看到
「哪一個掛了」的時候。

### 內容趨勢統計

`GET /api/admin/stats?days=` 補足後臺其他三頁都答不出的那個問題：「站正在長成
什麼樣子」。那三頁都是「現在是什麼」（總數、清單、待裁決的佇列），這裡是
「變成這樣多久了」。

一次回傳四塊資料：

| 區塊 | 內容 |
| --- | --- |
| `series` | 視窗內每日新增的使用者／文章／留言，**固定 `days` 長度**（沒有資料的日子是 0 而非缺漏） |
| `totals` | 視窗內合計（新增的使用者／文章／留言／按讚） |
| `topPosts` | 視窗內發表的文章，依「留言 + 按讚」排序，含前 80 字摘要 |
| `topTags` | 依綁定人數排序的標籤；**不設時間範圍**（標籤是身分分類而不是事件） |
| `topAuthors` | 視窗內發文最多的作者，留言數另外列出 |

三件讀這支端點時要知道的事：

- **每日份量以伺服器主機的本地時區切分。** `created_at` 由應用層寫入
  `time.Now()`（不是資料庫的 `NOW()`），因此站台部署在 UTC 而管理員在 UTC+8
  時，「今天」的數字看起來會少一截。那是時區差，不是流量掉了 —— 介面上也
  照實寫了這一句。
- **`topTags` 刻意不受 `days` 限制。** 一個人三個月前被標成「高雄」今天仍然
  是高雄；放進 30 天視窗會讓「熱門標籤」變成「最近剛好被套上的標籤」，那是
  完全不同的一個問題。
- **每次讀取都即時計算，沒有預先彙總的統計表。** 與監控頁的分鐘彙總
  （`forum_request_metrics`）相反：這裡的查詢被 `WHERE created_at >= ?` 限制在
  視窗內，掃描量是「最近 N 天的文章數」而不是全部。三張表的 `created_at`
  都有索引（`forum_users` 的在遷移第 24 步補上，那是這批查詢裡唯一原本沒有
  索引的）。什麼時候該改：文章總量成長到讓「最近 N 天」也不再是有界的時候 ——
  那時正確做法是加一張每日彙總表，而不是把視窗縮到 7 天（縮視窗會讓圖上看不出
  季節性的起伏，而那正是這一頁存在的理由）。

圖上的三條線**共用同一個縱軸刻度**。各自正規化看起來比較「好看」—— 三條線都會
有明顯的起伏 —— 但那會讓讀者誤以為三者的量級相當，而實情是新使用者通常比新
留言少一到兩個數量級。共用刻度之後貼地的那條線就是事實。

刻意**沒有**為 `forum_posts.author_email` 補複合索引：「活躍作者排行」那條查詢
會 `GROUP BY author_email` 而沒有可用索引，但它被 `created_at` 限制在視窗內；
為了一條有界的查詢在「最熱的表」上加索引，是把成本放在每次發文而不是偶爾開
一次後臺。

### 匯出與批次操作

三份 CSV 匯出（`/api/admin/export/{users,posts,reports}.csv`）與兩支批次端點
（`/api/admin/batch/{tags,status}`）。批次按鈕在**用戶管理頁**（`/admin`）的
核取方塊列上，而 `/admin/export` 只有下載入口與說明 —— 因為選取狀態只存在於
用戶管理頁，在匯出頁放批次按鈕只會得到一個永遠按不動的誘餌。

#### CSV 注入防護

以 `=`、`+`、`-`、`@`、Tab 或 CR 開頭的欄位值，Excel / Google Sheets /
LibreOffice 會當成**公式**評估。而這個專案匯出的欄位幾乎全是使用者可控的
自由文字（暱稱、貼文內容、**檢舉原因**），所以一條 `=cmd|'/C calc'!A0` 的
檢舉原因在管理員的機器上就會被執行。

`sanitizeCSVField` 因此在危險值前面加一個單引號（Excel 認得的「以下是文字」
前綴，顯示時不會出現）。三件刻意的取捨：

- **不跳過危險值。** 跳過會讓匯出少掉資料，而管理者不會知道少了什麼 ——
  那比顯示成文字更糟，因為它讓匯出結果不可信。
- **不改寫第一個字元。** 匯出的用途是「與線上內容逐字比對」，把 `=` 換成
  別的字元等於改資料。
- **接受前綴的可見代價**：在文字編輯器裡打開匯出檔會看到 `'。介面上寫明了
  這是刻意行為、不要要求移除 —— 否則某個管理員會以為那是資料損壞。

這個防護只有一份實作（`writeCSVRow`），因此沒有某一個匯出會漏掉它 ——
而漏掉的症狀不會出現在任何日誌、任何 HTTP 狀態碼、任何測試失敗裡。
`batch_export_test.go` 有一組涵蓋全部六個前綴的表格測試專門守這件事。

另外兩個容易被忽略的細節也一併處理了：檔案以 **UTF-8 BOM** 開頭（沒有它
Excel 開啟中文全是亂碼），以及 `Content-Disposition: attachment`（沒有它瀏覽器
會把匯出內容直接內嵌顯示，而那是全站使用者的 email 與站內言論）。

#### 批次操作的兩個保證

1. **每個受影響的對象各自記一筆稽核。** 「一次停權 50 個帳號」若只記一筆，
   稽核紀錄就答不出「這個 email 什麼時候被停權的」—— 而那正是稽核紀錄存在
   的理由。
2. **整批生效或整批不動。** 整批放在單一交易裡，因此不會出現「第 30 個失敗、
   前 29 個已經停權」這種沒有任何地方記錄部分結果的狀態。

回應帶 `counts`（`updated` / `unchanged`）與 `skipped`（逐項列出 email 與
原因）。逐項帶原因的理由：管理員需要知道跳過的 4 個是「帳號不存在」還是
「格式錯誤」—— 兩者的下一步完全不同。

刻意不做的事：批次標籤採**覆寫**語意（與單一使用者的 `PUT` 相同），沒有
「加標籤」與「設定標籤」兩種並存的模式 —— 猜錯「加」對「設」的後果是使用者
無預期地失去標籤。

### 登入與 Session 管理

`GET /api/admin/sessions` 與 `POST /api/admin/sessions/revoke` 回答後臺原本答不
不出來的兩個問題：「現在有誰在線上」與「把某個人的所有裝置都登出」。它存在的
理由是「cookie 即憑證」設計（見 `session.go` 檔頭第一點）留下的一個洞：token
不輪替、不綁 IP、不綁 UA，因此遭竊的 cookie 在過期前可被完整重用，而後臺原本
沒有任何手段提前止血。

#### 掃描必須用 SCAN，且每批用 pipeline 撈回

key 是隨機 token，**無法由 email 反推 token**，因此沒有「只掃某個 email 的 key」
這種可能 —— 只能掃整個 `FORUM:session:` 命名空間再逐一比對 hash 裡的 email。

- **不能用 KEYS。** 它會讓 Redis 在掃完全部 keyspace 之前阻塞整個實例。
  session 庫與媒體 token 共用同一個 Redis，因此一個後臺頁面的 KEYS 會讓整站
  同時無法登入、無法讀圖 —— 包括「按一下頁面來解除封鎖」這件事本身。
- **游標必須帶著回傳值繼續。** 寫成「每次都傳 0」會讓迴圈永遠只掃第一頁，
  而症狀是「看得到少數幾支 session，看起來一切正常」。
- **每批的 HGETALL + TTL 用 pipeline 一次撈回。** 逐個呼叫是每支 session 兩次
  往返；列出 200 支就是 400 次往返，跨網路時是一段好幾秒的空白頁。
- **SCAN 不保證一次遍歷內不重複**，因此有一層以 token 去重的過濾。
- **key 數有上限（20000），達到就截斷並回報 `truncated: true`。** 掃描成本是
  O(session 總數)；若同時有十萬支 session，這個請求會變成跑好幾秒的 SCAN，
  而它是管理員按一下就發出的請求。截斷的結果對「找出那個帳號的 session」仍然
  有用，而且「沒掃完」會被明確告知 —— 少了那個旗標，「沒列出來」看起來就會像
  「不存在」。截斷時強制登出**回 409 而不是靜默回 0 筆**：那個函式的契約是
  「刪掉所有」，掃描沒完就做不到。

#### 不回傳完整 token

token 就是憑證本身。介面上只給前 8 個字元（`tokenPrefix`），足以讓管理員分辨
「是不是同一支」而不足以還原。批次撤銷也只接受 email、不接受 token —— 讓介面
傳 token 才能撤銷，等於那個 token 已經離開伺服器了。

頁面上有一段明寫「這是刻意的限制，不是尚未完成的功能」：少了它，第一個管理員
會以為是 bug 而要求把完整 token 顯示出來。

#### 建立時間是「新的 session 才會有」

`created_at` 從這個功能開始才寫進 hash。既有 session 沒有它，而建立時間已經
過去了、補寫不可能，猜一個值只會產生比「不知道」更糟的資料 —— 因此介面顯示
「未知」，後端回空字串而不是 `0001-01-01T00:00:00Z`（那看起來像真實時間）。

#### 強制登出的稽核寫在 MySQL，動作在 Redis

兩個資料庫之間沒有共同交易，因此這個操作**不可能**像其他後臺寫入那樣「操作
與稽核同生共死」。這裡選擇的方向是**先刪 Redis（動作），再寫稽核（紀錄）**：
「動手之後忘了記錄」比「記錄了但還沒動手」安全 —— 後者會讓稽核紀錄宣稱「已撤銷」
而 session 還活著，那是一個**不實的紀錄**，而稽核紀錄的價值全在於它是真實的。

稽核寫入失敗時回 500 並在訊息裡說明「session 已撤銷但紀錄失敗」。那會讓管理員
困惑（他會想重試，而重試會顯示「已撤銷 0 筆」），但那個困惑好過一個不實的
稽核紀錄。

#### 刻意不做的事

- **不顯示每支 session 的 IP 與 User-Agent。** session hash 裡沒有存這兩項，
  而為了顯示它們就得新增欄位 —— 那是為了診斷而擴大憑證的儲存面。token 不綁
  IP/UA 這件事本身就是既有的已知限制，補上這兩項也不會讓它消失。
- **不提供「登出所有 session」。** 那是一個非常容易誤按的按鈕，而且按下之後
  管理員自己也被踢出，症狀是「我按了登出全部，結果我也登出了，而且沒有辦法
  再進來登出所有人」。

### IP 封鎖名單

`GET /api/admin/blocks` 與 `POST /api/admin/blocks`（`minutes <= 0` 代表解除）。
名單是 Redis sorted set：`FORUM:ipban`，member 是 IP、score 是到期 Unix 秒。
過期項目由背景 goroutine 每小時清理一次，而查詢時（`ipban.IsBanned`）也會正確
地把已過期的項目視為未封鎖 —— 因此**清理間隔只影響「ZRANGE 結果裡有多少雜訊」
而不影響正確性**。

#### 為什麼需要它（限流不夠的兩個地方）

`ratelimit.go` 的限流是**行程內**的純記憶體滑動視窗，而那是刻意的取捨
（見該檔檔頭：把計數器放進 Redis 會讓每個請求多一次網路往返）。但純記憶體有
兩個限制，它們都不是「可接受的取捨」而是「擋不住攻擊」：

1. **重啟即失效** —— 攻擊者只要等一次部署或崩潰就重新拿到滿額度，而且沒有
   任何人收到通知。
2. **不跨行程** —— 負載平衡器後面有 N 個執行個體時，實際額度是 `limit × N`，
   而輪到哪一台不是使用者能控制的。

把封鎖做成「額度設成 0」是錯的：限流的記憶體狀態有上面這兩個限制，所以用限流
實作的封鎖會在一次維護窗口之後自動解除。封鎖必須存在於行程之外。

#### 熱路徑的成本（這是本功能最需要被評估的一件事）

`IsBanned` 是一次 `ZSCORE`，加上它的時機是每個「非 GET 且被限流器攔到」的
請求：發文、留言、按讚、檢舉、上傳圖片、OAuth 登入跳轉。成本的實際大小：

| 面向 | 評估 |
| --- | --- |
| 指令本身 | `ZSCORE` 是 O(log N)，N 是封鎖筆數（這個站上通常是 0 到數十）；單一 key、單一 member，回應約 8 bytes |
| 往返 | 與既有 Redis 呼叫同一條連線。`session` 套件在**每個**請求上都已經做了一次 `HGET`（`ResolveUser`），而它只為了「知道有沒有登入」；`IsBanned` 多出來的那次與那一次性質相同。因此在本架構下它是「又多一次已經在做的事」，不是「新的延遲來源」 |
| 相對該請求的工作量 | 建立一則貼文要做一次 INSERT、可能一次 ES 索引（又一次 HTTP 往返）。`ZSCORE` 相對之下是雜音 |
| 讀取端點 | **完全不受影響**。限流器只掛在寫入型路由上（見 `ratelimit.go` 檔頭的掛載位置說明），所以匿名訪客的瀏覽不會多付這一次 |

結論：這個成本可以接受，而它買到的是「封鎖在部署之後仍然有效」。

掛載位置由 `withBlocklistHandler` 這**一個**函式決定，讓「先查封鎖、再查限流」
不會有第二個掛載點而不同步。順序不可交換：反過來會讓被封鎖的 IP 先累積限流
計數，而那個計數會在解除封鎖之後仍然生效 —— 一個沒有管理員動作卻持續存在的
隱藏狀態。

#### 來源 IP 的信任模型

限流、封鎖與稽核紀錄的 `ip` 欄位都用同一個判定：「這次請求來自哪個 IP」。
`X-Forwarded-For` 與 `X-Real-IP` 是**任何用戶端都能自己設定的標頭**，因此怎麼
對待它們決定了這三件事的可信度：

| `TRUSTED_PROXY_CIDRS` | 對端在白名單裡 | 對端不在白名單裡 |
| --- | --- | --- |
| 留空（預設） | 採信 XFF 最左項 → X-Real-IP → 對端 | 同左（**不檢查白名單**） |
| 已設定 | 從 XFF **右**往左掃過所有可信代理，取第一個不在白名單的位址 | 完全忽略轉送標頭，一律用對端位址 |

三件事要分開看：

1. **封鎖查所有候選位址，命中任何一個就算被封鎖。** 候選包含解析出的用戶端位址
   與 TCP 對端，因此被封鎖者加多少個 `X-Forwarded-For` 都還是會被查到 —— 這條
   不需要先設定白名單就成立。常見情況（直連、或 XFF 與對端相同）候選只有一個，
   Redis 往返數不變。
2. **白名單已設定時偽造標頭完全無效。** 對端不可信就整個忽略轉送標頭；對端可信
   時是從右往左掃，因此攻擊者在 XFF 左邊多塞幾項也不會改變判定（最左項恰恰是
   舊寫法裡最容易被控制的位置）。
3. **稽核紀錄的 `ip` 欄位寫的是同一個判定結果**，所以它與限流、封鎖看到的
   一定是同一個值 —— 不會出現「日誌說是 A、封鎖說是 B」。

留空時維持舊的標頭優先行為是刻意的相容性選擇：不採信 XFF 會讓限流與封鎖的分桶
鍵變成**代理的位址**，一個濫用者就會讓整站共用同一個代理的人一起被擋。兩種模式
的差別會顯示在監控頁的 `clientIpTrust`（`mode: "trusted-proxies"` 或
`"legacy-headers"`）；宣告了白名單卻一筆都解析不出來時，會看到 `mode` 是
`legacy-headers` 而 `declared` 非空 —— 那就是「設定寫錯了」的可見訊號。

#### 失敗時 fail open

Redis 故障時封鎖檢查會失敗，而呼叫端**放行**。理由：讓封鎖檢查失敗就擋掉所有
人，會把一次 Redis 抖動變成「整站不能發文」，而那比「Redis 掛掉期間封鎖失效」
嚴重得多（Redis 掛掉時本站的登入本來就已經受影響 —— session 查不到等於未登入）。

可觀察代價：攻擊者只要製造 Redis 壓力，就能暫時解除對自己的封鎖。監控頁的
Redis 探測會變紅，那是這個狀態唯一的提示。因此記錄是**節流的**（每 90 秒一行），
否則 Redis 故障會在幾秒內灌滿 log。

#### 刻意不做的事

- **不會自動封鎖。** 超過限流額度的位址只會收到 429，不會被自動加進名單。
  同一個出口位址可能是一整間辦公室或一整個 NAT，而自動封鎖會誤傷他們 ——
  誤封正常使用者的後果比多讓一個腳本多打幾次嚴重得多。**封鎖必須是管理員的決定。**
- **不支援 CIDR 範圍。** 封鎖名單的 key 是單一 IP，接受「一段範圍」會讓
  「這個 IP 被封了嗎」變成一個需要逐一比對的問題。範圍封鎖是另一個功能。
- **沒有「永久封鎖」。** score 是到期秒數，「永久」只能寫成一個極大的數字，
  而那種封鎖沒有辦法靠時間自動解除 —— 它會一直查到有人來解除為止，而管理員
  在幾個月後已經不記得自己封過誰。上限是 365 天，輸入更長的會被**就地收斂**
  （而不是回錯），介面上寫出了這個上限。
- **不把原因存進 Redis。** 只寫進稽核紀錄。sorted set 的 member 只能是 IP，
  而 reason 若塞進 member 就無法再用 IP 查詢；另開一個 hash 又多了一個要保持
  同步的資料結構。管理員填的原因是給「事後查」看的，而稽核紀錄正好是那個地方。

### 站內公告與文章置頂

`forum_announcements`（一列一則）與 `forum_posts.pinned`（每篇一個旗標）。
公告在 `/admin/announcements` 管理，置頂按鈕在「論壇文章」管理頁的每一列上 ——
它們分成兩處是因為「單元」不同：一個是全站唯一資源，一個是每篇文章的屬性。

#### 同一時間只有一則公告

公告在公開頁上是一條橫幅。兩條同時生效會互相衝突（使用者看到一個被橫幅佔掉
的上半頁，其中還可能是「活動改期」與「活動照常」這種打架的內容）。表格保留多列
是為了留下「這則公告是什麼時候、經誰發布、之後被誰關掉」的歷史，而那正是稽核
紀錄之外的另一半脈絡。

這個不變條件由**三件**事保證：

1. 發佈新公告時，在同一個交易裡把舊的設為不啟用。
2. 改為啟用時，同樣先把其他全部關掉。
3. 公開查詢回 `announcement: null` 而不是 404 —— 那是壓倒性的常見情況，若回 404，
   前端每個頁面載入都得區分「正常的沒有」與「端點壞了」。

`expires_at` 允許 NULL（永不自動過期），因為「過期」與「手動停用」是兩件事：
活動結束的公告會自然過期，而放錯一則需要立刻消失 —— 前者不該要求管理員記得回來
關，後者不該等到過期時間。

#### 置頂的索引必須是遞減的

動態的排序是 `pinned DESC, created_at DESC, id DESC`。MySQL 的「反向掃描索引」
會把**所有**欄位一起反向，因此一個 `(pinned, created_at)` 的遞增索引無法服務這個
排序 —— 查詢最佳化器會判定它不能用而退回 filesort（正確但慢，那正是加索引要避免
的）。所以 `idx_forum_posts_feed` 宣告成 `(pinned DESC, created_at DESC, id DESC)`
（MySQL 8.0+ 支援，README 的環境需求正是 MySQL 8.0.29+）。既有的
`idx_forum_posts_created_at` 保留：搜尋結果、profile 的貼文列表與管理端的多處
查詢都只按 `created_at` 排序，那些地方不該為了支援置頂而多付一個用不上的索引。

#### 什麼翻譯、什麼不翻譯

這是這個功能唯一需要寫成規格的事：

- **不翻譯** —— 公告的**內文**。它是管理員寫的純文字，翻譯它需要一套翻譯資料庫，
  而本站的多語系是純介面層的（見 `src/i18n`）。所以內文以原文顯示，換行以 `\n`
  保留（本站沒有富文字編輯器，textarea 的換行是唯一的排版資訊）。
- **翻譯** —— 圍著它的每一個字：標題、關閉鈕的 aria-label、發佈與到期的時間格式，
  以及文章卡上的「置頂」徽章。把標題寫死成「公告」會讓英文介面裡出現兩個中文字，
  而那是最容易被發現也最難以辯解的一類 bug。

#### 三狀態而不是兩狀態

後臺的公告列表顯示 `active` 與 `effective` 兩個不同的值：前者是開關，後者是
「實際上會不會顯示」。一則 `active` 但已過期的公告在列表裡看起來是開著的，而它
實際上什麼都不顯示 —— 合成一個欄位會讓管理員以為橫幅還在。三個狀態因此是
「顯示中」（綠）／「已停用」（琥珀，可重新啟用）／「已過期」（藍，要重新設定時間），
而顏色區分的是「能不能按」而不是嚴重度。

#### 刻意不做的事

- **不刪除公告。** 只有停用。保留歷史是為了回答「這則是什麼時候、經誰發布的」，
  而那正是稽核紀錄之外的另一半脈絡。頁面上明寫了這件事，因為「按了刪掉之後
  就找不回來」是使用者在按下去之前應該知道的。
- **有效時間不超過 365 天**，且介面上是有限清單而不是數字輸入框（理由與 IP 封鎖
  的時長相同：數字欄位最常見的填法是把「7」當成「7 分鐘」）。

---

## 頁面路由

後端把多個 HTML 檔直接送給瀏覽器（這是 MPA 不是 SPA 的原因）。

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

**MPA，16 個 entry**：`src/entries/` 下的 `forum`、`forum-login`、`forum-new`、
`forum-profile`、`forum-others-profile`、`forum-following`、`forum-admin`、
`forum-report`、`forum-monitor`、`audit-log`、`forum-stats`、`forum-export`、
`forum-sessions`、`forum-blocks`、`forum-announcements`、`admin`，對應
`frontend/*.html` 與 `vite.config.ts` 的 `rollupOptions.input`。同一組檔名也出現在
後端 `httpapi.frontendShellFiles`（CSP 的 `<style>` SHA-256 授權）—— 三處少一個
檔名，那一頁就會整頁沒有版面，因此 `securityheaders_test.go` 有一支測試守住
這兩份清單的同步。

`vite.config.ts` 另有兩個自訂行為：

- `copyForumRuntimeAssets`（`enforce: 'post'`）：用 esbuild 編譯 `service-worker.ts`
  成 IIFE，輸出**不帶 hash** 的 `/service-worker.js` 與 `/forum-manifest.json`
  （service worker 必須如此），並把 HTML 裡的 manifest 連結改回未雜湊版，
  讓後端可以在送出時替換 `{{FORUM_NAME}}`。
- `vite-plugin-javascript-obfuscator`（只在 `build`）：字串陣列旋轉 +
  control flow flattening。`debugProtection` / `selfDefending` 維持關閉，
  因為它們需要 `eval`，而 CSP 是 `script-src 'self'`。

### 開發時的 API 請求

前端所有 `fetch` 都是同源相對路徑（`/api/...`），但 `vite.config.ts` **沒有設定
`server.proxy`**，所以 `npm run dev` 起在 5173 時，請求會打到 Vite 而不是 8088 的
後端。要用 dev server + HMR 開發，先補上 proxy：

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

不改 `vite.config.ts` 的話，就照正式模式走：`npm run build` 產出 `dist/`，
再啟動後端由它提供靜態檔案。

---

## 測試

後端與檔案服務都有 Go 測試，**前端沒有單元測試**（package.json 內沒有 `test`
腳本）—— 前端的測試是**型別檢查**加上編譯期保證的翻譯完整性，理由見
`src/i18n/messages.ts` 的說明（`Record<MessageKey, string>` 讓漏翻譯與用錯鍵
在編譯期就失敗）。

```bash
# 後端
cd backend
go test ./...                # 全域
go test ./forum/config/...   # 設定解析（三組限流的兜底、管理員白名單正規化、TTL 守門）
go test ./forum/auth/...     # OAuth（開放轉向防護、state 往返、四種上游失敗）
go test ./forum/logger/...   # 欄位清洗（含日誌注入）、兩種格式、分級、併行安全
go test ./forum/metrics/...  # 請求統計（注入假時鐘，斷言快照）
go test ./forum/audit/...    # 稽核（CSV 防護、佔位符、截斷）
go test ./forum/session/...  # Session 列舉與撤銷（miniredis）
go test ./forum/ipban/...    # IP 封鎖名單（miniredis；過期、清理、長度上下限）
go test ./forum/es/...       # ES 傳輸層（用 httptest 與 fake server）
go test .                    # 監聽逾時、優雅停止、-check 報告（package main）

# 檔案服務（上傳／刪除／token 驗證／路徑穿越）
cd files_server
go test ./...

# 需要真實 MySQL 的遷移測試（未設定 DSN 時會自動跳過）
FORUM_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/?parseTime=true&loc=UTC&multiStatements=true' \
  go test ./backend/forum/data/...

# 翻譯目錄與本文件的語系宣稱（路徑以腳本自身位置推導，可從任何目錄執行）
node tools/i18n/verify-catalogs.mjs
```

### 測試檔清單

| 檔案 | 覆蓋 |
| --- | --- |
| `backend/forum/config/config_test.go` | 三組限流與兩組逾時**拒絕 0 與負數**、管理員白名單的 TrimSpace + ToLower（雙邊正規化）、`MEDIA_TOKEN_TTL_SECONDS` 的正式環境守門與「刻意設成 30 天」豁免、尾斜線裁切、`GOOGLE_REDIRECT_URL` 不裁切、DSN 的 `=` 切分、未知 key 靜默忽略 |
| `backend/forum/auth/auth_test.go` | **開放轉向的全家族**（絕對網址／protocol-relative／反斜線／五種編碼繞過）、`HandleLogin` → `ReturnPath` 的 state 編碼往返、授權網址形狀（scope 正確、client_secret 不外洩）、`GetUserEmail` 的成功路徑與四種上游失敗、**上游內容不洩漏進錯誤訊息** |
| `backend/forum/logger/logger_test.go` | `sanitizeLogValue` 的四條規則（含 **UTF-8 rune 邊界截斷**與「清洗後為空要用佔位符」）、門檻過濾、text 與 json 兩種格式、背景訊息省略前綴、**中繼欄位也會被清洗**、存取記錄的分級、併行寫入（`-race`） |
| `backend/forum/httpapi/frontend_shells_test.go` | **十七個 HTML 殼在三處同步**（`src/entries/`、`vite.config.ts`、`frontendShellFiles`）—— 新增一頁而不動另外兩處會指名缺哪一處 |
| `backend/forum/httpapi/security_invariants_test.go` | CSRF 來源檢查（含「兩個標頭都沒有」這個刻意放行）、限流器的設定轉換與 per-client 計數、CSP 來源正規化、**樣式區塊內容的位元組精確性** |
| `backend/forum/data/migration_test.go` | **真實 MySQL**：13 張表全部建立、冪等（跑兩次結構不變且**資料不被清掉**）、舊 schema 升級路徑、中文與 emoji 往返、三個併行遷移不死鎖 |
| `files_server/server_test.go` | 上傳的 token 驗證與大小限制、副檔名白名單、**UUID 命名（不覆蓋、不洩漏檔名）**、**刪除的路徑穿越全家族**、媒體 token（不存在 → 401、Redis 故障 → **503 而非 401**）、兩條降級路徑、CORS 預檢、**沒有任何路徑能列出儲存結構** |
| `files_server/shutdown.go`（`drainUntilTimeout` 的回傳值） | 由 `files_server` 的測試與 `backend/shutdown_test.go` 共同覆蓋 |
| `tools/invariants/analyzer_test.go` | 三條不變條件分析器的**負向測試**（刻意寫違規程式碼，確認會被報）與正向測試（確認不誤報），加上 `walkBody` 的 stack 配對 |
| `backend/forum/httpapi/site_test.go` | 站名樣板取代、跳脫、manifest、style hash 穩定性 |
| `backend/forum/httpapi/securityheaders_test.go` | `frontendShellFiles` 與實際頁面殼的同步、雜湊的完整與穩定 |
| `backend/forum/httpapi/monitoring_test.go` | 統計中介層的路由／狀態碼／in-flight、**每 IP 統計的接線與封鎖標記**、限流器計數、監控端點的授權、記憶體解析 |
| `backend/forum/httpapi/stats_handlers_test.go` | 統計視窗的收斂與起點、摘要的字元計算、端點授權 |
| `backend/forum/httpapi/batch_export_test.go` | **CSV 注入的六個危險前綴**、BOM 與下載標頭、email 清單正規化、匯出與批次的授權 |
| `backend/forum/httpapi/announcement_test.go` | 公告內文的邊界（空白／300 字／有效時間上下限）、後臺端點的授權、公開端點的 null 路徑 |
| `backend/forum/httpapi/post_edit_test.go` | 貼文／留言編輯與刪除的授權邊界（未登入 401、來源不可信 403）、**內文驗證在碰資料庫之前就擋下**（空白／超長／非 JSON）、`/api/forum/posts/` 子樹的方法分派（405 與 404 的分界）、兩個路徑解析純函式的表格測試 |
| `backend/forum/httpapi/trustedproxy_test.go` | `TRUSTED_PROXY_CIDRS` 的解析與信任模型（標頭優先 vs 白名單） |
| `backend/forum/metrics/metrics_test.go` | 請求統計：路由正規化、延遲分桶、分鐘桶守恆、歷史讀回、時間軸連續性 |
| `backend/forum/metrics/clients_test.go` | 每 IP 統計的累加、被封鎖擋下的計數、上限時驅逐最舊的、`clientsDropped` 可觀察、閒置剪枝、排序穩定 |
| `backend/forum/session/session_list_test.go` | 掃描走遍整個 keyspace、**絕不回傳完整 token**、SCAN 去重、上限截斷、撤銷只刪目標帳號 |
| `backend/forum/ipban/ipban_test.go` | 過期封鎖視為未封鎖、重複封鎖延長到期、`Ban` 順手清理、CIDR 被拒、長度上下限、nil 連線安全 |
| `backend/forum/audit/audit_test.go` | 稽核寫入的參數順序與錯誤傳遞、UTF-8 邊界截斷、變更筆數上限、查詢條件的佔位符與分頁收斂 |
| `backend/forum/es/es_test.go` | ES 傳輸層約 16 個案例（`httptest` 假伺服器） |
| `backend/shutdown_test.go` | 逾時設定（**含「刻意留白」的 `ReadTimeout` / `WriteTimeout`**）、停止訊號與監聽錯誤的分流、**`Shutdown` 會等在途請求完成** |

### 覆蓋率

2026-10-04 的實測（合併兩個 Go 模組，**26.2%**）：

| 套件 | 覆蓋率 | 說明 |
| --- | --- | --- |
| `forum/auth` | 92.0% | OAuth 的防護面幾乎全覆蓋 |
| `forum/config` | 89.1% | 全部純函式 + 每個兜底分支 |
| `forum/logger` | 85.5% | 含併行安全與注入防護 |
| `forum/es` | 84.3% | 原本就有 |
| `forum/ipban` | 82.4% | 原本就有 |
| `forum/metrics` | 72.6% | 原本就有 |
| `forum/session` | 67.1% | 原本就有 |
| `forum/audit` | 50.0% | 原本就有 |
| `files_server` | 48.9% | 從 **0%** 開始（Phase 3.4） |
| `forum/httpapi` | 12.0% | 約 2,500 行；已覆蓋 CSRF、限流、CSP、殼檔同步、貼文／留言編輯的授權與驗證邊界 |
| `forum/data` | 0%（本機）/ 高（有 MySQL） | 需要真實資料庫，CI 的 `migrations` job 提供 |

CI 有一道**覆蓋率地板**（`.github/workflows/ci.yml` 的 `coverage-floor` job，
地板值 25.0）。它的作用是**防止退化**，不是製造動機 —— 刻意不設一個虛高的
目標（例如 70%），因為那會誘發無行為斷言的測試來拉數字，而那種測試是負債：
它會在重構時壞掉，卻沒有指出任何東西壞了。

### 不變條件 analyzer

這個專案有四條**專屬**的不變條件。它們約束的不是「型別對不對」或「有沒有
測試」，因此通用工具抓不到 —— 一個 handler 從 0% 變成 30% 覆蓋，與「Commit
之前有沒有檢查稽核錯誤」毫無關係。

| 不變條件 | 由什麼守住 |
| --- | --- |
| 稽核與操作同生共死：`beginAdminTx` → **檢查** `recordAdminAction` 的錯誤 → `tx.Commit()` | `auditcheck` analyzer |
| `/api/admin/*` 的 handler 在 method 分派**之前**呼叫 `requireAdminForum` | `adminauth` analyzer |
| `withBlocklistHandler` 只在 `applyRateLimit` 內被呼叫（單一掛載點） | `blocklistmount` analyzer |
| 十六個 HTML 殼在三處同步 | `frontend_shells_test.go` |

前三條是 `tools/invariants` 這個獨立 Go 模組裡的 `go/analysis` analyzer。
**失敗模式是編譯失敗**（`go vet` 回報），而不是「有人沒注意到」。

```bash
# 建置並執行
cd tools/invariants && go build -o invariants .
cd ../../backend && go vet -vettool=<abs>/tools/invariants/invariants ./...

# 分析器自己也要被測試（含負向測試：刻意寫違規程式碼，確認會被報）
cd tools/invariants && go test ./...
```

它刻意是**獨立的 Go 模組**：它需要 `golang.org/x/tools`，而 backend 的相依項
刻意維持在四個（miniredis、mysql、go-redis、oauth2）。把分析工具的相依塞進
runtime 相依裡，等於讓每個部署環境都多下載一份只有 CI 需要的程式碼。

第四條跨越 Go / TypeScript / HTML 三種語言，`go/analysis` 表達不了，因此它是
一個**測試**而不是 analyzer。兩種失敗模式（編譯失敗 vs 測試失敗）刻意不混在
同一個工具裡，以免搞混哪一條失效了。

### CI

`.github/workflows/ci.yml` 有八個 job：

| job | 做什麼 |
| --- | --- |
| `backend` | build、`go vet`、`go test -race`（CGO 開啟就是為了它） |
| `files_server` | 同上 |
| `frontend` | `npm ci`、`typecheck`、`build`、`verify-catalogs` |
| `coverage-floor` | 合併兩個模組的覆蓋率，檢查是否 ≥ 25.0 |
| `invariants` | 建置並執行分析器，加上分析器自測 |
| `migrations` | 起真實 MySQL 8.0.29，跑 `forum/data` 的遷移測試 |
| `secrets` | 確認 `config.conf` 從未被追蹤、範本檔不含明文憑證 |

`secrets` 那個 job 用 `fetch-depth: 0` 掃**整份歷史** —— 只掃工作區的話，一個
「刪掉檔案」的 commit 就足以讓真正的憑證留在倉庫裡。

### 靜態檢查與格式

這個專案**刻意沒有** gofmt / golangci-lint 的閘門。理由不是懶，而是它們會在
第一次執行時擋下 25 個既有檔案，而那些格式差異是**行尾（CRLF）**造成的 ——
在一個以 Windows 為主的開發環境裡，那會讓每一個開發者第一次推上去時都被擋，
而他們完全無法理解原因。

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

反向代理（nginx / Caddy）不在此 repo 內。務必確認：

1. **後端的 `COOKIE_SECURE=true`**，並且代理有處理 HTTPS。
2. **`PUBLIC_BASE_URL`、`GOOGLE_REDIRECT_URL` 與 `TRUSTED_ORIGINS` 對齊**實際對外網址。
3. **`FILES_SERVER_PUBLIC_URL` 指向對外可讀的位址**；`FILES_SERVER_URL` 可以是內部位址。
   兩者相同代表沒有內外網分離。
4. **Google Cloud Console 的授權_redirect URI** 與 `GOOGLE_REDIRECT_URL` 完全一致。
5. **代理必須放行 `Service-Worker-Allowed` 與 `/service-worker.js`**，否則 PWA 不會更新。
6. `/files/*` 若走代理並由瀏覽器直接取圖，代理**不得**吃掉 `?token=` 查詢參數。
7. **停止訊號要送得對，而且要給夠時間**。後端收到 `SIGTERM` / `SIGINT` 後會排空
   在途請求，上限是 `SHUTDOWN_TIMEOUT_SECONDS`（預設 15 秒）；超過就強制中斷。
   因此部署環境的停止上限必須**大於**它，否則部署端會先動手殺掉行程：

   | 環境 | 預設停止上限 | 怎麼調 |
   | --- | --- | --- |
   | systemd | `TimeoutStopSec=90s` | 通常已足夠 |
   | Docker | `--stop-timeout=10` | `docker run --stop-timeout=20 …` 或 compose 的 `stop_grace_period` |
   | Kubernetes | `terminationGracePeriodSeconds=30` | 部署 manifest 內設定 |
   | 裸執行 | — | 會送出 `SIGINT`，Ctrl-C 即可 |

   另一個方向是讓反向代理**先**把流量抽掉再轉送停止訊號給後端，那樣後端收到
   訊號時本來就沒有新請求要接，排空會是瞬間完成的事。

建置指令：

```bash
cd frontend && npm ci && npm run build     # 產出 frontend/dist（已 gitignore）
cd ../files_server && go build -o files-server .
cd ../backend && go build -o forum .
```

`backend/update.sh` 是給 Debian/Ubuntu 的一次性建置腳本（apt 裝 Go 再 `go build`）。

### 容器部署（docker compose）

`docker-compose.yml` 提供四個服務：`mysql`、`redis`、`backend`、`files_server`。
**沒有「前端」服務** —— 那是這個架構最刻意的一個決定，理由見下。

```bash
# 1. 準備三份設定檔
cp deploy/settings.conf.example deploy/settings.conf       # 容器編排的環境變數
cp backend/config/config.conf.example backend/config/config.conf
cp files_server/config.conf.example files_server/config.conf
# 三份都改成自己的值

# 2. 部署前先驗證設定 —— 這是整份 compose 裡最值得做的一步
docker compose run --rm backend -check

# 3. 起服務
docker compose up -d
docker compose ps
docker compose logs -f backend
```

#### 為什麼前端沒有獨立服務

因為後端在啟動時會掃描前端產物、算出每個 HTML 殼裡 `<style>` 區塊的 SHA-256，
然後把它們放進 CSP 的 `style-src`（`backend/forum/httpapi/securityheaders.go`）。
也就是說：**被提供的檔案與被授權的雜湊必須來自同一份建置輸出**。

它們分開的症狀是「整頁沒有版面」——`style.css` 仍會作用（它是 `<link>` 載入的
外部檔案，`style-src 'self'` 已經涵蓋），所以畫面看起來「有樣式但怪怪的」，
而瀏覽器主控台只有一行 `Refused to apply inline style`。

因此 `backend/Dockerfile` 的建置情境是**倉庫根目錄**，它先跑 `npm ci` +
`npm run typecheck` + `npm run build`，再編譯 Go，最後把 `dist/` 從同一個 stage
COPY 進最終映像。那個不變量由建置流程**結構性**保證 —— 不可能提供一份與雜湊
不符的檔案，因為兩者來自同一次建置。

另一個好處：部署時需要啟動的容器只有兩個。沒有第三個「前端」服務，也就不會有
「啟動了靜態檔案伺服器，而它指向後端，但後端還沒起來」這個啟動順序問題。

#### 容器部署**必須**額外設定的兩個值

1. **`TRUSTED_PROXY_CIDRS=172.28.0.0/16`**（寫在 `backend/config/config.conf`）

   這是容器部署下唯一必須額外設定的安全項目。留空會讓「無條件採信
   `X-Forwarded-For`」生效，而 `forum-net` 內的**任何容器**都能直接連 8088 ——
   它只要自己塞一個標頭就能讓限流計數記到別的 IP 上：那等於限流完全失效，
   而症狀是「限流看起來有開，只是擋不住任何人」。

   為什麼那個 subnet 是寫死的：`docker-compose.yml` 明確指定了
   `172.28.0.0/16`，讓這裡可以直接抄。留空讓 Docker 自己挑的話，每台機器都
   不同，而那個差異只在啟動之後才會以上述症狀出現。確認實際生效的值可以看
   後臺監控頁的「來源 IP 信任模型」。

2. **`MEDIA_TOKEN_TTL_SECONDS`**（同樣寫在 `config.conf`）

   這不是容器專屬的，但 `-check` 會在正式環境擋下忘記設定的部署
   （見下面的「設定檢查」）。

#### 停止行為

`backend` 與 `files_server` 都有 `stop_grace_period: 20s`，而 `SHUTDOWN_TIMEOUT_SECONDS`
的預設是 15 秒。那 5 秒的間隙是刻意留的：給後端寫最後一次監控彙總與關閉資料庫
連線的時間。

`docker compose stop` 送出 `SIGTERM` 後，兩個容器都會走各自的排空流程然後以
**exit code 0** 結束（見 `backend/shutdown.go` 與 `files_server/shutdown.go`）。
一個回 1 會讓部署端以為有東西壞了。

MySQL 的 `stop_grace_period` 是 20 秒、`Redis` 是 10 秒 —— 分開設定是因為它們
的停止成本完全不同（InnoDB 需要時間 checkpoint，Redis 的 AOF 重寫很快）。

#### `-check`：部署前先驗證設定

```bash
cd backend
go run . -check
docker compose run --rm backend -check    # 不啟動任何服務
```

它讀設定檔、套用兜底值、跑一次 `Validate`，然後輸出一份報告。**它不連線任何
外部服務**，所以它回答的是「這個行程會以什麼組態啟動」，而不是「相依服務能不能
連上」。

報告刻意**不印任何憑證**（只印「已設定（N 個字元）」）——`-check` 的輸出會出現在
CI 日誌、issue 回報與截圖裡，而那些地方經常沒有存取控制。

它會點名的項目，都是**啟動之後完全沒有症狀**的那一類：

- `COOKIE_SECURE` / `PUBLIC_BASE_URL` 判定為開發環境，而設定裡有正式環境的痕跡
- 三組限流的實際額度（它們在設定檔打錯字時會沿用預設值，而那看起來「有開」）
- 媒體 token 的 TTL 與**它是明確設定還是採用兜底值**（30 天）
- 管理員白名單的**實際儲存值**（已去除空白並轉小寫）—— 這正是部署者要確認的
  東西：「我寫的那個信箱，真的在名單裡嗎」
- 逾時值（拿來和部署端的停止上限對照）
- 正式環境下未設定 `FILES_SERVER_TOKEN` 的警告

#### 這個專案在容器上的三個刻意取捨

| 決定 | 理由 | 代價 |
| --- | --- | --- |
| 後端映像是 **distroless** | 沒有 shell 與套件管理員；一個有 shell 的正式環境容器距離「有人 exec 進去手動修東西」只有一步 | 診斷不能靠 `exec` 進去看檔案；改用 `LOG_FORMAT=json` 從 stdout 收集（compose 的預設） |
| 檔案服務映像是 **alpine** | 這個服務處理使用者上傳的檔名（不可信輸入），需要一個 shell 作為未來的緩解手段 | 多 7 MB、多一層 libc 的 CVE 曝露面 |
| 不用 **Docker secrets** | secrets 檔案是明文（base64），靠檔案權限保護 —— 與 `settings.conf` 的保護機制**相同**。而真正值得保護的 `GOOGLE_CLIENT_SECRET` 與 `FILES_SERVER_TOKEN` 在應用層的設定檔裡，從不進版控、也從不進 build context | `settings.conf` 必須靠 gitignore 保護 |

#### systemd 部署

容器不是唯一的選擇。`docker-compose.yml` 與 systemd 是**並存**的兩條路
（ROADMAP.md 的 Phase 4.7 刻意保留 systemd 單元檔）。

systemd 的部署條件與容器相同：反向代理在別處、`COOKIE_SECURE=true`、
`TRUSTED_ORIGINS` 對齊、`TimeoutStopSec` > `SHUTDOWN_TIMEOUT_SECONDS`、
`TRUSTED_PROXY_CIDRS` 填真正在前面那道代理的位址段。

---

## 安全設計

- **CSP**：`script-src 'self'`，沒有 `unsafe-inline`、沒有 `unsafe-eval`。
  HTML 內的 `<style>` 區塊在啟動時算 SHA-256，寫進 `style-src` 的 hash 清單 ——
  所以樣板不能隨意新增內嵌樣式。
- **CSRF**：`POST` 寫入端點檢查 `Origin` 是否在 `TRUSTED_ORIGINS` 內。
- **匿名性**：email 不出現在任何 URL；對外一律用 `public_key = SHA256(email)`。
- **Session**：存 Redis，可撤銷；cookie 滑動續期。
- **限流**：三段獨立額度，依端點成本區分。啟動時有背景清理 goroutine。
- **檔名**：落地一律 UUID v4 + 原副檔名，不使用使用者提供的檔名；副檔名白名單。
- **媒體權杖**：圖片網址綁短 TTL 的 Redis token，前端離開頁面時可主動釋放。
- **Slowloris**：`READ_HEADER_TIMEOUT_SECONDS`（預設 10 秒）限制讀完請求標頭的
  時間；`IdleTimeout` 讓閒置的 keep-alive 連線在一分鐘內被回收。`ReadTimeout` /
  `WriteTimeout` 刻意不設，理由見「已知問題」
- **資源大小**：上傳有 `max_size` 上限。
- **密鑰不進版控**：`config.conf` 被 gitignore。唯一例外是 `FILES_SERVER_TOKEN`
  可用環境變數注入（供 CI/CD 或容器使用）。

---

## 批次翻譯的作法

`tools/i18n/` 是一組 Node ESM 腳本，負責批次翻譯的切單與合併；代理模型只負責中間
那一段（讀 TSV、寫 TSV）。因此**翻譯品質的驗證不依賴代理模型自報** —— 鍵集、
佔位符、未翻譯殘留都由腳本把關。

流程細節與設計理由見 [`tools/i18n/README.md`](tools/i18n/README.md)。全部腳本
必須從 `frontend/` 執行：

```bash
cd frontend

# 1. 切工作單：messages.ts → key<TAB>原文，切成每份約 96 行
node ../../tools/i18n/make-worklist.mjs <輸出目錄> <語言> [份數，預設 4]

# 2. 交給代理模型，一個任務一份：讀 <語言>-N.tsv，寫 <語言>-N.out.tsv

# 3. 合併：鍵集 / 佔位符 / 未翻譯殘留都過關才會寫出 TS 目錄檔
node ../../tools/i18n/merge-translations.mjs <輸出目錄> <語言> <匯出名稱> <檔名>
#    沒收齊會輸出 <語言>-todo.tsv，遞迴回第 1 或第 2 步

# 附帶：把 todo 再切小
node ../../tools/i18n/split-todo.mjs <輸出目錄> <語言> [份數]

# 隨時驗證全部語系目錄（路徑以腳本自身位置推導，可從任何目錄執行）
node ../../tools/i18n/verify-catalogs.mjs
#    這一項同時比對本文件宣稱的語系數量與清單：新增或刪除語系而沒更新
#    上面兩處數字時，它會指名 README 的哪一處過期（CI 會跑這一步）。
```

四種失敗的處理方式：

| 失敗 | 處理 |
| --- | --- |
| 少了鍵 | 拒絕寫檔，補齊後重跑 |
| 多了鍵 | 拒絕寫檔（多出來的鍵不會被匯入） |
| 佔位符不一致（`{id}` 被寫成 `#id`） | **只報錯不修復**。猜測補救會讓錯譯看起來通過驗證 |
| 未翻譯殘留（含漢字，`zh-*` 與 `ja` 除外） | 拒絕寫檔 |
| **字元損毀（值裡出現 `U+FFFD`）** | 拒絕寫檔，並指名是哪個鍵 |

最後那一條與語言無關，且與其他四條性質不同：`U+FFFD`（REPLACEMENT CHARACTER）
不是任何一種語言的字，它出現在目錄裡只代表某個字元在某一次編碼往返中被解碼
失敗並換成替代符號 —— 原始位元組當場就丟了，所以它既不能被翻譯修好，也不會
因為換語系而消失。佔位符、鍵集、順序在這種情況下全部正常，因此先前沒有任何
檢查抓得到。實測抓到過兩處（`th` 的 `export.batchNote` 與 `block.reasonHint`），
兩者都因為「整句還看得出大致意思」而被當成可接受的翻譯擱置了；修法是照
`messages.ts` 的原文重寫整句，而不是去猜那個丟掉的字元。

`make-worklist.mjs` 會排除 11 個必須原樣保留的鍵（拉丁字母排版裝飾與產品名，
例如 `NEW POST`、`Admin Console`、`ID`）—— 翻譯會破壞視覺設計，而且本來就不需要翻。

現有語系（`frontend/src/i18n/translations/`）共 16 份，連同基準的 `zh-TW`
（`src/i18n/messages.ts`）共 17 個：
`ar`（RTL）、`de`、`en`、`es`、`fr`、`hi`、`id`、`ja`、`ko`、`pt-BR`、`ru`、
`th`、`vi`、`zh-CN`、`zh-HK`、`zh-MO`。

---

## 已知問題

誠實記錄現況，避免下次 deploy 時踩到：

- **`httpapi` 約 2,500 行仍有約 88% 沒有測試覆蓋**（Phase 3.3 補了 CSRF、限流、
  CSP 與殼檔同步；貼文／留言編輯補了授權與驗證邊界，其餘的 handler 邏輯仍未
  涵蓋）。優先順序建議依「壞掉時的爆炸半徑」排：批次／匯出 → 貼文 CRUD →
  公告 → 統計。
- **編輯貼文刻意不接受替換附圖**。`PUT /api/forum/posts/{id}` 只有 `content`
  欄位：換圖不只是改一個欄位，舊檔會變成沒有任何資料列指向的孤兒檔，而本站
  沒有「使用者刪除自己圖片」的端點（`/api/forum/image-tokens/release` 只作廢
  存取權杖，不刪檔）。在補上那條路徑之前，選錯圖只能刪掉整篇重發。
- **留言與檢舉的 handler 沒有逐支重複的身分檢查**。`handleForumPostUpdate`、
  `handleForumCommentUpdate`、`handleForumCommentDelete` 與
  `handleForumPostDelete` 都在開頭確認一次登入狀態（縱深防禦），但
  `handleForumComments`（POST 留言）與 `handleForumReport` 沒有 —— 它們的安全性
  完全來自中介層的 `requireLoginForWrite`。這兩條路由目前不會被未登入者呼叫到，
  所以不是可利用的缺口；但若日後有人把它們註冊到別的路徑上，就會同時失去
  中介層與 handler 兩層。
- **`forum/data` 的遷移測試需要真實 MySQL**，本機執行時會全部跳過
  （未設 `FORUM_TEST_MYSQL_DSN`）。CI 的 `migrations` job 會跑它們 —— 但那也
  意味著**本機的 `go test ./...` 不會驗證 schema 遷移**。
- **`docker compose` 尚未在本機實測過**：這個環境沒有 Docker，因此映像與
  compose 檔只經過靜態檢查（YAML 結構驗證 + 每個值與程式碼的設定鍵對照）。
  第一次實際部署時請特別留意 `docker compose run --rm backend -check` 的輸出。
- **`files_server` 的 S3 模式未經測試**。測試覆蓋的是本機模式；S3 後端是刻意
  保持「最小可用子集」（PUT / DELETE + BasicAuth），而它的 `saveFile` 會把整個
  檔案讀進記憶體 —— 因此 `upload.max_size` 被調到數百 MB 時那會成為記憶體尖峰。
- **`token_key_prefix` 兩邊不一致的後果很難診斷**。範本已把 `files_server` 的
  `[redis].token_key_prefix` 對齊成 `forum:token:`，但既有的 `config.conf` 若仍留著
  舊值（例如 `hpnm:token:`），症狀是「上傳成功、貼文也存得下，但圖片一律 403/401」。
  兩邊都要是 `forum:token:`，且都要與 session 的 `forum:session:` 前綴區隔。
- **`npm run dev` 開箱即壞**。沒有 `server.proxy`，相對路徑的 API 請求打不到後端。
- **兩個 Go 模組版本不一致**：backend 要 1.25、files_server 要 1.26。
- **`MEDIA_TOKEN_TTL_SECONDS` 的兜底值是 30 天**，對正式環境明顯過長。已緩解：
  正式環境（`COOKIE_SECURE=true` 或 `PUBLIC_BASE_URL` 為 https）未明確設定時，
  `config.Validate` 會讓啟動被拒絕，並由 `-check` 在部署前指出。
  刻意「設成 30 天」是允許的 —— 那代表部署者知道自己在做什麼。
- **後端刻意不設 `ReadTimeout` 與 `WriteTimeout`**（`shutdown.go`）。這是刻意的：
  前者會連請求本文一起計時，而貼文附圖的本文可達 50 MB 且還要轉送給檔案伺服器；
  後者會在回應寫完前砍斷連線，而 CSV 匯出的耗時就落在這段裡。補上這兩項時必須
  同時評估上傳與匯出路徑，而不是照常見範例填 30 秒。`ReadHeaderTimeout` 與
  `IdleTimeout` 有設，Slowloris 與 keep-alive 連線累積這兩個曝露面是關掉的。
- **`files_server` 仍然沒有優雅停止**。它用的是零值 `http.ListenAndServe`
  （`files_server/main.go:644`），收到 `SIGTERM` 會直接死，沒有逾時也沒有排空。
  後端轉送上傳的那一條連線因此可能在停止時被中斷 —— 症狀是「貼文存得下但圖片
  上傳失敗」。它的請求都是單一的短命上傳／讀取，影響面遠小於後端，但這是兩個
  Go 模組之間唯一還沒處理的停止流程差異。
- **管理端點沒有限流**（刻意如此，但值得知道）。`/api/admin/monitor` 尤其
  不限流：它是監控頁每十秒打一次的合法流量，限流它只會在真正出事時多一條
  混淆的訊息。
- **監控統計只活在一個行程裡**。請求總量、依路由統計、延遲直方圖與每 IP 統計都是
  記憶體內的累計，重啟歸零；分鐘桶會寫進 `forum_request_metrics`（已結束的分鐘、
  每分鐘只寫一次、用累加語意支援多執行個體），因此時間軸在重啟後不會變空白，
  但「上個行程一共服務了多少請求」這種問題答不出來。要長期資料得看存取日誌。
- **每 IP 統計刻意不落盤**（`metrics/clients.go`）。落盤會讓「某個位址在什麼時候
  打了本站」變成一份永久的訪客紀錄，而這一頁只需要「現在正在怎樣」；重啟後歸零
  是這個取捨的代價。若部署上真的需要長期的來源統計，那應該是存取日誌的工作
  （那裡本來就有 IP），而不是從監控頁擴張出去。
- **來源位址的清單上限 200 個，會被驅逐**。位址是攻擊者可控的維度（可偽造
  `X-Forwarded-For`），因此 map 一定要有上限。驅逐掉的那個位址，其計數會**整筆
  消失**（沒有併進任何彙總槽 —— IP 在這個頁面上是可操作的，彙總槽裡的假位址無法
  操作也沒有意義）；`stats.clientsDropped` 是唯一的線索，介面會把它顯示出來。
- **被強制殺掉的行程仍會遺失當前分鐘的監控資料**（`SIGKILL`、容器被硬殺、或
  排空超過 `SHUTDOWN_TIMEOUT_SECONDS` 之後被部署端殺掉）。優雅停止會同步寫出
  最後一次分鐘彙總，因此「正常重啟」不再有這個缺口；剩下的只有非正常結束，
  那與「監控本來就是取樣」相符，但確實是一個已知的資料缺口。
- **延遲分位數是分桶上界**，P95 只會落在 1/2/5/10/25/50/100/250/500ms 或
  1/2/5/10s 上。介面上照實標示，但拿它跟精確的 APM 數字比較會失望。
- **多執行個體時監控頁只顯示打到這個行程的流量**。限流器的 hits map 本來就是
  行程內的（見 `ratelimit.go`），請求統計沿用同一個範圍；資料庫的分鐘彙總會把
  各個個體加總，但即時統計不會。
- **稽核紀錄的 INSERT 與操作共用交易，因此稽核表故障會讓後臺寫入全部失敗**。
  這是刻意的取捨：寧可「停權按了沒反應」，也不要「停權成功了但沒人知道是誰停的」。
  若 `forum_admin_actions` 出了問題（例如磁碟滿），後臺的每一個寫入操作都會回
  500，必須先修好稽核表才能繼續管理。唯讀頁面不受影響。
- **稽核紀錄保存的是截斷後的文字**（每個值 200 bytes）。要還原完整內容得回
  `forum_posts` / `forum_post_comments`，而那兩張表的內容本身已經被覆寫了 ——
  稽核紀錄不能回答「被改掉的那段原文是什麼」，只能回答「原本的前 200 bytes 是什麼」。
- **稽核紀錄沒有防止同一管理員互相掩蓋的機制**。要查到「誰動的」需要該管理員
  的存取日誌與稽核紀錄一起看；而 access log 沒有輪替設定，會無限期增長。
- **內容趨勢的每日份量以主機本地時區切分**，站台在 UTC、管理員在 UTC+8 時
  「今天」會看起來少一截。這是 `created_at` 寫入 `time.Now()`（而非資料庫的
  `NOW()`）的必然結果，介面上已寫出這一句，但它是需要管理員自己記住的背景知識。
- **內容趨勢的「熱門標籤」不受時間視窗限制**，而「熱門文章」與「活躍作者」受
  限制。這個不一致是刻意的（標籤是身分分類、其餘是事件），但讀排行榜時若不
  注意到就會拿三份不同範圍的資料互相比較。
- **`/admin/stats` 每次讀取都即時計算**，沒有預先彙總的統計表。掃描量被視窗
  限制住，但當文章總量成長到讓「最近 N 天」也不再有界時，這個查詢會開始變慢 ——
  那時該加每日彙總表，而不是縮短視窗。
- **CSV 匯出每份上限 5 萬列，到達上限時靜默截斷**（串流的 CSV 不能中途插入
  一行提示，那會讓欄位數不一致）。介面上寫出了上限，但匯出檔本身沒有標記。
- **匯出檔裡的單引號前綴在試算表中不可見、在文字編輯器中可見。** 這是刻意的
  取捨（見「CSV 注入防護」），但它會被誤認為資料損壞 —— 因此頁面上明寫
  「不要要求移除它」。
- **Session 列表的建立時間只對新建立的 session 有意義。** `created_at` 是這個
  功能之後才寫進 hash 的，既有 session 顯示「未知」且永遠不會補上。
- **強制登出的稽核紀錄可能在動作之後才寫，且兩者不在同一個交易裡**（一個在
  Redis、一個在 MySQL）。稽核寫入失敗時操作已經生效，介面會明確說明這一點 ——
  它看起來像矛盾，但那正是「寧可少一筆紀錄，也不要一筆不實的紀錄」這個取捨
  的樣子。
- **Session 掃描有 20000 個 key 的上限**，達到時清單不完整（介面會標示）。
  在同時登入人數遠低於這個數字的站上不會發生；一旦發生（例如爬蟲大量登入），
  強制登出会回 409 而不是靜默宣稱成功。
- **Session 不記錄 IP 與 User-Agent**，因此無法從後臺看出「這支 session 是從
  哪裡來的」。這是「token 即憑證」設計的已知限制（見 `session.go` 檔頭安全
  假設的最後一點），而不是這個功能漏了。
- **IP 封鎖在 Redis 故障時失效（fail open）。** 擋掉所有人會把一次 Redis 抖動
  變成「整站不能發文」，那比封鎖失效嚴重得多。代價是攻擊者只要製造 Redis 壓力
  就能暫時解除對自己的封鎖；監控頁的 Redis 探測是這個狀態唯一的提示。
- **IP 封鎖在超過 365 天時被就地收斂**，不會回錯。介面上寫出了上限，但匯出的
  稽核紀錄只會記到收斂後的到期時間。
- **IP 封鎖的「原因」不存進 Redis**，只存在稽核紀錄裡。因此從 Redis 直接查
  （redis-cli）會看不到原因，而那些紀錄在稽核保留期（90 天）到期後就消失了。
- **站內公告的內文不翻譯**，只翻譯圍著它的介面文字（標題、時間格式、置頂徽章）。
  這是刻意的：翻譯管理員寫的文字需要一套翻譯資料庫，而本站的多語系是純介面層的。
  介面上是英文的使用者會看到中文公告 —— 那正確嗎？取決於你的讀者，而這個決定
  應該由站方明確做出，而不是由實作細節決定。
- **公告只有一則生效，且不會被刪除**（只有停用）。因此 `forum_announcements` 會
  隨時間累積列數；若要清理過期且已停用的公告，目前得直接動資料庫。
- **測試覆蓋集中在少數幾個檔**，`httpapi` 的主要 handler 沒有測試；
  `server.go` 的註解提到的 `forum_handlers_test.go` 目前不存在於 repo 中。
- **`ALLOWED_ADMIN_EMAIL` 的正規化**：比對**兩邊**都做 TrimSpace + ToLower，
  因此帶空白或大寫的寫法不再靜默失效。刻意「不」做的正規化是 Gmail 的點號與
  `+` 別名（`a.b+c@example.com` 與 `abc@example.com` 是同一信箱）—— 那會把
  「同一個信箱」變成一個需要業務規則的判斷，而白名單的重點是「只有這些明確的
  信箱可以當管理員」，放寬比對只會讓它更容易被誤加。
- **`backend/server.log` 是舊版配置留下的**，裡面寫的是 `frontend/web/dist`，
  這個目錄現在不存在了。
- **schema 沒有 migration 版本控制**，只有 idempotent 步驟，沒有降級路徑。
