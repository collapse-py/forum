# 論壇

免註冊、以 Google 帳號登入的匿名論壇。登入後的對外身分是 `SHA256(email)` 算出來的
`public_key`，公開頁面與 URL 裡都不會出現 email。管理員身分不看資料庫角色表，只看
`ALLOWED_ADMIN_EMAIL` 白名單。

後端是只用標準函式庫寫的 Go HTTP 服務（`net/http` + `database/sql`，沒有 web
framework、沒有 ORM），前端是 Vite + React 19 + TypeScript 的多頁應用（PWA），
另有獨立的 Go 檔案服務處理媒體存儲。

[文件](docs/) · [Docker Compose](docker-compose.yml) · [LICENSE](LICENSE)

## 為什麼

- **沒有密碼、沒有本機帳號**：只有 Google OAuth2，且僅申請 `userinfo.email`。
- **匿名性由構造保證**：email 不進入任何 URL 或公開頁面，對外身分只有雜湊值。
- **可立即撤銷的 session**：權威狀態在 Redis，不在簽章 cookie —— 登出或強制登出立即生效。

## 核心特色

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

**多語系**：17 個語系（含 `zh-TW` 為基準），阿拉伯文為 RTL。現有語系
（`frontend/src/i18n/translations/`）共 16 份，連同基準的 `zh-TW`
（`src/i18n/messages.ts`）共 17 個：`ar`（RTL）、`de`、`en`、`es`、`fr`、`hi`、`id`、
`ja`、`ko`、`pt-BR`、`ru`、`th`、`vi`、`zh-CN`、`zh-HK`、`zh-MO`。批次翻譯流程見
[`docs/FRONTEND.md`](docs/FRONTEND.md#批次翻譯)。

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

## 架構

```
                      ┌───────────────────────────────────────────────┐
    瀏覽器 ──────────▶ │  backend/  (Go, :8088)                        │
    PWA                │   httpapi/ 路由、中介層、security headers      │
    /forum             │   auth/ Google OAuth2 · session/ Redis session │
    /admin             │   ipban/ · metrics/ · audit/ · data/ · es/     │
                       │   ── 服務 frontend/dist/ 的靜態檔案 ──         │
                       └────────┬─────────────────────┬───────────────┘
                                │ SQL                 │ HTTP + token
                                ▼                     ▼
                      ┌──────────────────┐   ┌─────────────────────────┐
                      │ MySQL  :3306     │   │ files_server/ (Go,      │
                      │ Redis  :6379     │◀─▶│ :7070)  local / S3 存儲  │
                      │ ES     :9200     │   │ Redis media token        │
                      │ （ES 選用）       │   └─────────────────────────┘
                      └──────────────────┘
```

- `backend/`：全部 API、認證、頁面與靜態檔（Go，標準函式庫）
- `MySQL`：持久化資料（貼文、留言、稽核…），schema 啟動時自動建立
- `Redis`：session、媒體權杖、IP 封鎖名單 —— 全站的信任根
- `Elasticsearch`：全文搜尋，選用（留空則退回 MySQL `LIKE`）
- `files_server/`：媒體上傳／刪除／讀取（local 或 S3）

反向代理（nginx / Caddy / Cloudflare Tunnel）**不在此 repo 內**，TLS 由它終結。

兩個 Go 模組彼此獨立，各自的 `go.mod` 與設定檔；第三個 Go 模組
`tools/invariants/` 只在 CI 使用（見 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)）：

| 目錄 | 模組名 | 設定檔 | 埠 | 職責 |
| --- | --- | --- | --- | --- |
| `backend/` | `forum` | `config/config.conf` | 8088 | 全部 API、認證、頁面與靜態檔 |
| `files_server/` | `files_server` | `config.conf` | 7070 | 媒體上傳／刪除／讀取 |

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
`ALTER TABLE ... ADD COLUMN IF NOT EXISTS` —— 這個語法在 8.0.29 之前不存在。在較舊的
8.0 上啟動會在遷移時拿到 `ERROR 1064 (42000)`，行程在啟動時就死。

## 快速開始

```bash
# 1. 準備設定檔（兩份都在 .gitignore，範本已提交且不含真實憑證）
cp backend/config/config.conf.example backend/config/config.conf
cp files_server/config.conf.example  files_server/config.conf
#    範本附有逐項註解；最小可用設定見 docs/CONFIGURATION.md

# 2. 啟動檔案服務
cd files_server && go mod download && go run .

# 3. 建置前端
cd frontend && npm install && npm run build

# 4. 啟動後端（一定要在 backend/ 目錄下執行；設定檔路徑是相對於工作目錄）
cd backend && go build .
./forum -check      # 先驗證設定：輸出報告，有問題時 exit 1
./forum             # 正常啟動（Windows: .\forum.exe；或直接 go run .）
```

開瀏覽器到 <http://localhost:8088/forum>（`/` 會 302 到這裡）。

`-check` **不連線任何外部服務**，回答的是「這個行程會以什麼組態啟動」而不是「相依
服務能不能連上」。報告刻意不印任何憑證，點名的都是**啟動之後完全沒有症狀**的那一類
問題（正式環境的 `COOKIE_SECURE`、三組限流的實際額度、媒體 token 的 TTL、管理員
白名單的實際儲存值）。後端執行檔另有 `-healthz` 旗標：探測 MySQL 與 Redis 後結束
（0 健康／1 不健康）。

## Docker 部署

`docker-compose.yml` 提供四個服務：`mysql`、`redis`、`backend`、`files_server`。
**沒有「前端」服務** —— 後端啟動時算出每個 HTML 殼 `<style>` 的 SHA-256 寫進 CSP，
被提供的檔案與被授權的雜湊必須來自同一份建置輸出（理由見
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md#為什麼前端沒有獨立服務)）。

```bash
./deploy/build-images.ps1                                  # 0. 打包兩個服務映像
cp deploy/settings.conf.example deploy/settings.conf       # 1. 容器編排的環境變數
docker compose --env-file deploy/settings.conf run --rm backend /app/forum -check   # 2. 部署前驗證
docker compose --env-file deploy/settings.conf up -d       # 3. 起服務
```

容器部署**必須額外設定** `TRUSTED_PROXY_CIDRS=172.28.0.0/16`（寫在
`backend/config/config.conf`；該 subnet 寫死在 compose 的 `ipam` 設定裡）。留空會讓
限流可被單一偽造標頭繞過。逐步細節、部署前檢查表與停止上限的調整見
[`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)。

## 設定

兩份設定檔都是啟動時讀成快照，沒有熱重載。最重要的幾個：

- `DB_DSN`、`REDIS_ADDR` —— 相依服務連線
- `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_REDIRECT_URL` —— OAuth2
- `ALLOWED_ADMIN_EMAIL` —— 管理員白名單（雙邊 TrimSpace + ToLower 後比對）
- `TRUSTED_ORIGINS` —— CSRF 來源白名單，**留空等於關閉 CSRF 防護**
- `TRUSTED_PROXY_CIDRS` —— 可信任反向代理的位址段
- `COOKIE_SECURE` —— 走 HTTPS 就必須 `true`
- `FILES_SERVER_TOKEN`（唯一支援環境變數的設定）與 files_server 的 `[upload] token`
  必須相同；`MEDIA_TOKEN_KEY_PREFIX` 與 files_server 的 `token_key_prefix` 必須相同

完整參數表與「不看原始碼會猜錯的細節」見 [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md)。

## 安全

- OAuth2 登入，Redis 支撐的可撤銷 session（滑動續期）
- **Redis 就是信任根**：cookie 只是查詢鍵，能讀 Redis 等於能登入任何人、能寫 Redis
  等於能自己開管理員 session —— 密碼必須獨立、Redis 必須綁內網
- CSP（`script-src 'self'`，無 `unsafe-inline` / `unsafe-eval`）、CSRF 來源檢查、
  三段式限流
- 匿名性：email 不出現在 URL，公開身分是 `SHA256(email)`
- 媒體：UUID v4 檔名、副檔名白名單（不含 svg）、短 TTL 存取權杖

完整威脅模型、實測記錄與上線檢查表見 [`docs/SECURITY.md`](docs/SECURITY.md)。

## 文件

| 文件 | 回答什麼問題 | 讀它的時機 |
| --- | --- | --- |
| `README.md`（本檔） | 這是什麼？怎麼跑起來？ | 第一次接觸這個專案 |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | 每個子系統為什麼長成這樣？刻意不做哪些事？ | 要改程式碼之前 |
| [`docs/SECURITY.md`](docs/SECURITY.md) | 信任邊界在哪？哪些東西被刻意信任了？ | 上線前、或做安全評估時 |
| [`docs/OPERATIONS.md`](docs/OPERATIONS.md) | 怎麼維運？後台各頁在回答什麼問題？ | 維運這個站的時候 |
| [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) | 正式環境怎麼部署？ | 第一次部署時 |
| [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) | 每個設定鍵做什麼？ | 調設定時 |
| [`docs/DATABASE.md`](docs/DATABASE.md) | 資料庫長什麼樣？ | 改 schema 之前 |
| [`docs/API.md`](docs/API.md) | 端點有哪些？ | 寫用戶端或整合時 |
| [`docs/FRONTEND.md`](docs/FRONTEND.md) | 前端怎麼工作？怎麼加頁面、加語系？ | 改前端時 |
| [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) | 怎麼測試？CI 跑什麼？ | 貢獻程式碼時 |
| [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) | 現在還有哪些沒處理完的？ | 下次 deploy 前 |
| [`tools/i18n/README.md`](tools/i18n/README.md) | 批次翻譯的流程與驗證 | 要加語系時 |

原始碼的檔頭註解自成一份規格 —— 這個專案多數非顯而易見的決定都寫在**它發生的地方**，
而不在文件裡。

## 開發

```bash
cd backend && go test ./...              # 後端測試
cd files_server && go test ./...         # 檔案服務測試
cd frontend && npm run typecheck && npm run build
node tools/i18n/verify-catalogs.mjs      # 翻譯目錄與語系宣稱
```

前端沒有單元測試 —— 型別檢查（`strict` + `noUncheckedIndexedAccess` +
`exactOptionalPropertyTypes`）加上編譯期保證的翻譯完整性就是它的測試。測試架構、
四條專屬不變條件 analyzer 與 CI 的完整說明見
[`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)。

## 已知問題

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

## 授權

見 [`LICENSE`](LICENSE)。
