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
- **留言、按讚、檢舉**：留言公開可讀；按讚以 `(post_id, author_email)` 複合主鍵
  限制每人一次；檢舉涵蓋貼文與留言
- **追蹤與私密動態牆**：`/api/forum/following/posts` 只回傳追蹤中的人的貼文
- **Google OAuth2 登入**：僅申請 `userinfo.email`，沒有密碼、沒有本機帳號
- **Redis 版可撤銷 Session**：滑動續期，Redis 掛掉等於全部登入失效
- **管理後台**：貼文／留言 CRUD、檢舉佇列裁決（通過即刪除對象、駁回保留）、
  帳號停權與復原、使用者標籤字典與指派、以使用者身分代發文
- **Elasticsearch 全文搜尋**：`ES_URL` 留空則自動退回 MySQL `LIKE`
- **PWA**：可安裝、有 service worker 與 manifest，頁面 network-first、
  靜態資源 cache-first
- **多語系**：18 個語系（含 `zh-TW` 為基準），阿拉伯文為 RTL
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
| MySQL | 8.x，建一個 `utf8mb4` 資料庫 | 必要（schema 自動建立） |
| Redis | 5+ | 必要（session 與 media token） |
| Elasticsearch | 7.x / 8.x | 選用，留空則用 MySQL `LIKE` 搜尋 |
| Google OAuth2 用戶端 | — | 登入與管理功能需要 |

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
./forum             # Windows: .\forum.exe；或直接 go run .
```

啟動順序是固定的：讀設定 → 初始化 logger → ping Redis → 開 MySQL →
`data.MigrateMySQL` → `auth.Init` → `session.NewManager` → 註冊路由 →
啟動限流清理 → 背景重建 ES 索引 → 監聽。

開瀏覽器到 <http://localhost:8088/forum>（`/` 會 302 到這裡）。

> **一定要在 `backend/` 目錄下執行。** 設定檔路徑是
> `filepath.Join("config", "config.conf")`，相對於工作目錄而不是執行檔所在目錄，
> 在別的目錄跑會直接 fatal。

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
| `PUBLIC_BASE_URL` | `http://localhost` + `SERVER_PORT` | 尾斜線會被自動去掉 |
| `TRUSTED_ORIGINS` | `[PUBLIC_BASE_URL]` | CSRF 來源白名單。**留空等於關閉 CSRF 防護** |
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
| `MEDIA_TOKEN_TTL_SECONDS` | `2592000`（30 天） | 這個兜底值對正式環境明顯過長，請明確設定 |
| `RATE_LIMIT_REQUESTS` | `10` | 內容寫入 |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` | |
| `RATE_LIMIT_UPLOAD_REQUESTS` | `5` | 圖片上傳（最貴，額度最緊） |
| `RATE_LIMIT_UPLOAD_WINDOW_SECONDS` | `60` | |
| `RATE_LIMIT_AUTH_REQUESTS` | `10` | OAuth 端點，含所有 method |
| `RATE_LIMIT_AUTH_WINDOW_SECONDS` | `60` | |
| `GOOGLE_CLIENT_ID` | — | |
| `GOOGLE_CLIENT_SECRET` | — | |
| `GOOGLE_REDIRECT_URL` | — | 必須與 Google Cloud Console 完全一致 |
| `ALLOWED_ADMIN_EMAIL` | 空 | 逗號分隔，大小寫敏感且不做正規化 |
| `LOG_LEVEL` | `INFO` | |
| `LOG_FILE` | `server.log` | |
| `LOG_FORMAT` | `text` | 或 `json` |

三組限流預設值刻意不同，反映各端點的真實成本；`applyDefaults` 對「空值或非正數」
補值，因此無法用設定檔把某個視窗設成 0 秒（那會讓限流失效）。布林值接受
`1` / `true` / `yes` / `on`（不分大小寫）。

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
| `forum_posts` | 貼文 | `image_url` 只存**檔名**，讀取時才用 `FILES_SERVER_PUBLIC_URL` 組網址 |
| `forum_post_comments` | 留言 | 無外鍵，完整性靠應用層 |
| `forum_post_likes` | 按讚 | 複合 PK `(post_id, author_email)`，天然去重 |
| `forum_reports` | 檢舉 | `target_type`（`post`/`comment`）+ `target_id` 的多型參照，無外鍵 |
| `forum_profiles` | 公開資料 | `public_key` = `SHA2(author_email, 256)`；`nickname` 唯一 |
| `forum_users` | 帳號狀態 | `status` 為 `ACTIVE` / `SUSPENDED` |
| `forum_user_tags` | 標籤字典 | `name` 唯一 |
| `forum_user_tag_assignments` | 標籤指派 | 複合 PK `(user_email, tag_id)` |
| `forum_follows` | 追蹤關係 | 複合 PK `(follower_email, target_email)` |

刪除 `forum_reports` 的 `uq_forum_reports_reporter_target` 與
`idx_forum_reports_status_created` 是必要的：前者擋同一使用者重複檢舉同一對象，
後者讓「待處理檢舉」列表不必掃全表。

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
| DELETE | `/api/forum/posts/{id}` | 刪除自己的貼文 |
| GET | `/api/forum/posts/{id}/comments` | 留言列表（公開） |
| POST | `/api/forum/posts/{id}/comments` | 建立留言 |
| POST | `/api/forum/posts/{id}/like` | 按讚 |
| POST | `/api/forum/posts/{id}/report` | 檢舉貼文 |
| POST | `/api/forum/posts/{id}/comments/{cid}/report` | 檢舉留言 |
| POST | `/api/forum/images` | 上傳圖片，轉送 files_server |
| POST | `/api/forum/image-tokens/release` | 釋放媒體存取權杖 |
| GET/PUT | `/api/forum/profile` | 自己的資料 |
| GET/POST | `/api/forum/follows` | 追蹤名單 / 切換追蹤 |
| GET | `/api/forum/following/posts` | 私密動態牆（需登入，不限流） |
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
| `/admin` | `admin.html` | 管理員 |
| `/admin/forum` | `forum-admin.html` | 管理員 |
| `/admin/forum-report` | `forum-report.html` | 管理員 |

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

**MPA，9 個 entry**：`src/entries/` 下的 `forum`、`forum-login`、`forum-new`、
`forum-profile`、`forum-others-profile`、`forum-following`、`forum-admin`、
`forum-report`、`admin`，對應 `frontend/*.html` 與 `vite.config.ts` 的
`rollupOptions.input`。

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

後端有 Go 測試，**前端沒有測試**（package.json 內沒有 `test` 腳本）。

```bash
cd backend
go test ./...          # 全域
go test ./forum/es/... # ES 傳輸層（用 httptest 與 fake server）
```

現有的兩個測試檔：

| 檔案 | 覆蓋 |
| --- | --- |
| `backend/forum/httpapi/site_test.go` | 站名樣板取代、跳脫、manifest、style hash 穩定性 |
| `backend/forum/es/es_test.go` | ES 傳輸層約 16 個案例（`httptest` 假伺服器） |

`go.mod` 有宣告 `miniredis`（供 Redis 相關測試），但目前實際用到的 ES 測試是
`httptest`。`httpapi` 大部分 handler 沒有測試覆蓋。

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

建置指令：

```bash
cd frontend && npm ci && npm run build     # 產出 frontend/dist（已 gitignore）
cd ../files_server && go build -o files-server .
cd ../backend && go build -o forum .
```

`backend/update.sh` 是給 Debian/Ubuntu 的一次性建置腳本（apt 裝 Go 再 `go build`）。

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

# 隨時驗證全部語系目錄
node ../../tools/i18n/verify-catalogs.mjs
```

四種失敗的處理方式：

| 失敗 | 處理 |
| --- | --- |
| 少了鍵 | 拒絕寫檔，補齊後重跑 |
| 多了鍵 | 拒絕寫檔（多出來的鍵不會被匯入） |
| 佔位符不一致（`{id}` 被寫成 `#id`） | **只報錯不修復**。猜測補救會讓錯譯看起來通過驗證 |
| 未翻譯殘留（含漢字，`zh-*` 與 `ja` 除外） | 拒絕寫檔 |

`make-worklist.mjs` 會排除 11 個必須原樣保留的鍵（拉丁字母排版裝飾與產品名，
例如 `NEW POST`、`Admin Console`、`ID`）—— 翻譯會破壞視覺設計，而且本來就不需要翻。

現有語系（`frontend/src/i18n/translations/`）共 17 份，連同基準的 `zh-TW`
（`src/i18n/messages.ts`）共 18 個：
`ar`（RTL）、`de`、`en`、`es`、`fr`、`hi`、`id`、`ja`、`ko`、`pt-BR`、`ru`、
`th`、`vi`、`zh-CN`、`zh-HK`、`zh-MO`。

---

## 已知問題

誠實記錄現況，避免下次 deploy 時踩到：

- **`token_key_prefix` 兩邊不一致的後果很難診斷**。範本已把 `files_server` 的
  `[redis].token_key_prefix` 對齊成 `forum:token:`，但既有的 `config.conf` 若仍留著
  舊值（例如 `hpnm:token:`），症狀是「上傳成功、貼文也存得下，但圖片一律 403/401」。
  兩邊都要是 `forum:token:`，且都要與 session 的 `forum:session:` 前綴區隔。
- **`npm run dev` 開箱即壞**。沒有 `server.proxy`，相對路徑的 API 請求打不到後端。
- **兩個 Go 模組版本不一致**：backend 要 1.25、files_server 要 1.26。
- **`MEDIA_TOKEN_TTL_SECONDS` 的兜底值是 30 天**，對正式環境明顯過長。
- **後端沒有優雅關閉**（沒有 `signal` 處理、沒有 `Server.Shutdown`），也沒有設定
  `ReadHeaderTimeout`，直接用零值 `http.ListenAndServe`。
- **管理端點沒有限流**（刻意如此，但值得知道）。
- **測試覆蓋集中在少數幾個檔**，`httpapi` 的主要 handler 沒有測試；
  `server.go` 的註解提到的 `forum_handlers_test.go` 目前不存在於 repo 中。
- **`ALLOWED_ADMIN_EMAIL` 比對大小寫敏感且不做正規化**。白名單若寫成帶空白或
  大寫會靜默失效（因為 Google 回傳的 email 已是穩定的小寫）。
- **`backend/server.log` 是舊版配置留下的**，裡面寫的是 `frontend/web/dist`，
  這個目錄現在不存在了。
- **schema 沒有 migration 版本控制**，只有 idempotent 步驟，沒有降級路徑。
