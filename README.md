## 目錄結構

```
forum/
├── backend/          Go HTTP 伺服器（論壇 API + 靜態檔案服務）
│   ├── main.go
│   ├── config/config.conf
│   └── module/
│       ├── auth/     Google OAuth 登入
│       ├── config/   設定檔載入
│       ├── data/     MySQL 存取與論壇資料表遷移
│       ├── es/       Elasticsearch 傳輸層（貼文搜尋與索引維護）
│       ├── httpapi/  論壇路由與處理器、限流、安全標頭中介層、站名樣板代入
│       ├── logger/   請求日誌中介層
│       └── session/  Redis Session 管理
├── frontend/
│   ├── asset/        logo、PWA 圖示、中文字型
│   └── web/          Vite 前端專案（React 19 + TypeScript，輸出至 web/dist）
└── files_server/     圖片檔案微服務（論壇附圖上傳的相依服務）
```


## 系統需求

- Go 1.25+
- Node.js 18+ / npm
- MySQL 8（InnoDB、`utf8mb4`）
- Redis 6+
- Elasticsearch 7+（選用；未設定 `ES_URL` 時搜尋降級為 MySQL 關鍵字比對，其餘功能不受影響）
- 圖片檔案伺服器（`files_server/`，選用；未啟用時貼文附圖功能無法使用）

## 設定

編輯 `backend/config/config.conf`：

| 設定值 | 說明 |
| --- | --- |
| `FORUM_NAME` / `FORUM_SHORT_NAME` / `FORUM_DESCRIPTION` | 站台名稱（詳見「站名」一節） |
| `SERVER_PORT` | 監聽位址與埠（預設 `:8088`） |
| `PUBLIC_BASE_URL` / `TRUSTED_ORIGINS` | 站台網址與允許的 CSRF Origin |
| `DB_DSN` | MySQL 連線字串 |
| `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` | Redis（Session 與圖片存取 token） |
| `ES_URL` / `ES_INDEX` | Elasticsearch（貼文搜尋）。`ES_URL` 留空＝不啟用，搜尋改用 MySQL 關鍵字比對；`ES_INDEX` 預設 `forum_posts` |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_REDIRECT_URL` | Google OAuth 設定 |
| `ALLOWED_ADMIN_EMAIL` | 可進入後臺的管理員 Email（逗號分隔） |
| `FILES_SERVER_URL` / `FILES_SERVER_PUBLIC_URL` / `FILES_SERVER_TOKEN` | 圖片檔案伺服器 |
| `MEDIA_TOKEN_KEY_PREFIX` / `MEDIA_TOKEN_TTL_SECONDS` | 圖片存取 token 的 Redis 鍵前綴與存活秒數 |
| `RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW_SECONDS` | 內容寫入的限流次數與視窗秒數 |
| `RATE_LIMIT_UPLOAD_REQUESTS` / `RATE_LIMIT_UPLOAD_WINDOW_SECONDS` | 圖片上傳的限流次數與視窗秒數 |
| `RATE_LIMIT_AUTH_REQUESTS` / `RATE_LIMIT_AUTH_WINDOW_SECONDS` | OAuth 登入跳轉的限流次數與視窗秒數 |
| `COOKIE_SECURE` | 對外以 HTTPS 服務時應為 `true`；同時是 HSTS 標頭的送出依據 |
| `LOG_LEVEL` / `LOG_FILE` / `LOG_FORMAT` | 日誌設定（`INFO` / `DEBUG` 等；`WARN` 以上預設輸出到檔案） |

## 站名

站名是設定值，不是寫死的字串。三行設定決定它在全站的三種長度：

| 設定值 | 用途 | 留空時 |
| --- | --- | --- |
| `FORUM_NAME` | 完整站名：各頁 `<title>`、登入頁說明、導覽列與後臺的 aria 標籤、manifest 的 `name` | 預設 `FORUM 論壇` |
| `FORUM_SHORT_NAME` | 短名：導覽列標誌、後臺 rail 與 topbar 麵包屑、manifest 的 `short_name` | 沿用 `FORUM_NAME` |
| `FORUM_DESCRIPTION` | manifest 的 `description`（瀏覽器安裝提示與桌面捷徑說明） | 沿用 `FORUM_NAME` |

換站名的完整流程只有兩步：**改設定檔並重啟後端**。不必重建前端，也不必改任何
翻譯（見下）。

### 它怎麼從設定檔走到畫面上

1. 九個頁面殼的 `<head>` 帶著 `{{FORUM_NAME}}` / `{{FORUM_SHORT_NAME}}` 佔位符，
   `<title>` 與 `forum-manifest.json` 裡也是。
2. 後端在**送出**這些檔案時把佔位符換成設定值（`backend/forum/httpapi/site.go`），
   並以 HTML／JSON 兩種跳脫規則分別處理。
3. 前端 `src/site.ts` 在**模組層**讀 `<head>` 裡的兩個 `<meta name="forum-…">`，
   因此第一次繪製的標題與標誌就是正確站名，不會先用預設值畫一帧再換掉。

走「伺服器送出前取代」而不是「前端打 API 取站名」，是因為後者要等一個 RTT
（畫面上會看到一幀閃爍），而且沒有 JavaScript 的爬蟲與讀屏器連 `<title>` 都拿不到。

### 三條容易踩到的規則

- **佔位符必須在樣式區塊之外。** `style-src` 沒有 `'unsafe-inline'`，各頁的內嵌
  樣式是靠「磁碟上那份 HTML 的樣式內容」的 SHA-256 授權的（`securityheaders.go` 的
  `inlineStyleHashes`）。站名取代改動了位元組，雜湊就對不上 —— 症狀是**整頁沒有
  版面，且沒有任何錯誤訊息**。連 HTML 註解都不能寫出樣式標籤的字面量：雜湊是純
  文字掃描，註解裡的標籤會被當成真的開標籤。
- **翻譯目錄裡的站名是佔位符。** 13 條文案含站名，全部寫成 `{site}`（完整站名）
  或 `{brand}`（短名）：`title.*` 十條、`admin.railBrandAria`、`login.body`、
  `users.signinEyebrow`。`usePageTitle` 統一注入兩個參數，其餘三處由呼叫端傳入。
  `tools/i18n/verify-catalogs.mjs` 會比對各語言的佔位符集合，因此漏傳參數是
  編譯期與驗收期都抓得到的錯誤，而不是畫面上殘留 `{site}`。
- **站名不隨語言切換。** 它是設定檔裡的一個值，語言只改寫它周圍的介面文字
  （`登入｜{site}` 的「登入」）。這是刻意的取捨：一個可設定的專有名詞沒有譯文。

### manifest 的路徑不是 Vite 能決定的

`vite.config.ts` 會在 `generateBundle` 把各頁 `<link rel="manifest">` 的 href
改回 `/forum-manifest.json` 並刪掉 Vite 自己 emit 的 `assets/forum-manifest-<hash>.json`
複本。Vite 預設會把 `link[href]` 當成建置資產（emit 一份帶 hash 的複本並改寫
href），那份複本由 `/assets/` 的靜態檔案伺服器直接送出、帶 `immutable` 快取，站名
佔位符永遠不會被取代 —— PWA 於是會用字面量 `{{FORUM_NAME}}` 當應用程式名稱。
manifest 必須留在後端，因為它是站名唯一需要在執行期決定的前端資產。

換站名後已安裝的 PWA 仍會顯示舊名稱：那是瀏覽器端已安裝應用的資料，必須先
移除再安裝，網站端無法更新。

站名**不會**連動任何內部識別碼：cookie 名稱（`COOKIE_NAME`、語言 cookie
`forum_lang`）、Redis key 前綴與 Service Worker 的快取名（`forum-forum-v1`）都維持
原樣。改站名不影響登入狀態與圖片 token；反過來說，識別碼的命名慣例與顯示名稱是
兩件事，不要順手一起改。

## 建置與啟動

### 前端

```powershell
cd frontend/web
npm install
npm run typecheck    # tsc --noEmit，兩個 project：頁面碼與 service worker
npm run build        # 先 typecheck 再 vite build，輸出至 frontend/web/dist
npm run dev          # 開發用（Fast Refresh）；正式建置請用 build
```

`npm run build` 會先跑 `typecheck`，因此建置失敗即代表型別有問題，不會產出壞掉的
`dist`。型別設定分兩個 project，理由見「Service Worker 的型別檢查」一節。

### 後端

```powershell
cd backend
go build ./...
go vet ./...
go test ./...
go run .
```

啟動後：

- `/` 轉址至 `/forum`
- 靜態前端自 `../frontend/web/dist` 提供（若尚未建置則回退到 `../frontend/web`）
- 靜態資源自 `../frontend/asset` 以 `/asset/` 提供

前端已全面改用 TypeScript，**未建置就等於不能跑**：回退到原始碼目錄時 HTML 與 CSS
仍可載入，但頁面邏輯不會執行 —— 瀏覽器無法執行 `*.ts`，而 `static.go` 也刻意把 `.ts`
列入 `blockedSuffixes`（不公開原始碼）。開發時請先 `npm run build`。

### 圖片檔案伺服器（選用）

```powershell
cd files_server
go run .
```

## 資料表

啟動時自動建立／遷移（皆為 `CREATE TABLE IF NOT EXISTS` 與可重複執行的欄位新增）：

`forum_posts`、`forum_post_likes`、`forum_post_comments`、`forum_reports`、
`forum_users`、`forum_profiles`、`forum_user_tags`、`forum_user_tag_assignments`、
`forum_follows`

追蹤的三條設計約束（細節見 `backend/forum/httpapi/forum_follow_handlers.go` 的檔頭）：

- **關聯鍵是 `email`，不是 `public_key`。** 若以 `public_key`（`SHA2(email,256)`）
  當關聯鍵，追蹤動態的條件就得寫成 `SHA2(fp.author_email, 256) IN (...)`，
  MySQL 無法為運算式建索引，等於每次請求掃全表。存 email 則
  `IN (SELECT target_email ...)` 走複合主鍵最左前綴。API 表面仍然只有 64 個
  十六進位元的金鑰，email 不外流。
- **完全私有。** 沒有任何端點回傳「誰追蹤了誰」或「某人有幾個追蹤者」——
  追蹤表存的就是 email，開放這兩件事等於新增一條還原 email 的關聯路徑。
  回給本人的只有「我追蹤了哪些金鑰」與「我是否追蹤了這個人」。
- **複合主鍵擋不住自我追蹤**（`('x','x')` 是合法主鍵值），該規則由應用層守，
  見 `checkFollowTargetInput`（有單元測試）。

## 搜尋

貼文全文搜尋走 Elasticsearch，**搜尋只涵蓋 `forum_posts`**（留言不進索引）。

### 分工：ES 只找 ID，內容一律回 MySQL 讀

`module/es` 的 `Search` 回傳的是「符合條件的貼文 ID 與分數」，不是文件內容。
`module/httpapi/search.go` 拿著這串 ID 回 MySQL 取真正的貼文，包含按讚數、留言數、
附圖 token 與去識別化的作者顯示名。三個好處：

1. 索引裡不出現 `authorEmail`，公開搜尋不可能洩漏 email。
2. 索引與 MySQL 不一致時，使用者看到的是 MySQL 的版本。
3. 搜尋結果與貼文列表是同一種結構（`forumPost` / `adminForumPost`），
   前端可以用同一個元件渲染。

### 中文分詞

`content` 用內建的 `cjk` analyzer（重疊二字組合：「檢舉管理」→ 檢舉／舉管／管理），
另建 `content.standard` 子欄位補上單字召回 —— bigram 分析器不產生單字 token，
因此單字查詢（「的」「有」）只靠 `cjk` 會零命中。查詢以 `bool.should` 並聯
`content`（cjk）、`content.standard`（整詞）、`content.standard`（前綴，
讓 `elastic` 能命中 `elasticsearch`）三條；後臺另外加一條
`authorEmail` 的 `term` 精確比對，才能「找出某個信箱的所有貼文」。
`authorEmail` 只在後臺查詢裡出現，公開搜尋無法用信箱反推他人。

### 降級：ES 壞掉不等於不能搜尋

`ES_URL` 留空，或查詢時連不上／失敗，搜尋會自動改用 MySQL 的
`content LIKE '%關鍵字%'`（萬用字元有跳脫），回應中的 `engine` 欄位
標示 `elasticsearch` 或 `mysql`，前端據此顯示說明。ES 的寫入也是
best-effort：索引失敗只寫日誌，絕不讓「發文」這件事失敗。`/healthz` 會
回報 ES 的狀態但不因此判為不就緒 —— 掛掉的只有搜尋，不是核心功能。

### 索引維護

- 建立貼文（使用者發文、兩種後臺發文）→ 寫入索引
- 編輯貼文（後臺 `PUT /api/admin/forum/posts/:id`）→ 依 id 回讀整列後覆寫
- 刪除貼文（後臺 `DELETE`）→ 移除文件（檢舉的「通過（刪文）」也走這條路徑）
- **服務啟動時於背景 goroutine 內全量重建一次**，以消除停機期間的落差

ES 的 mapping 一旦要改（改分析器、加欄位）必須重建索引，服務不會自動做：

```sh
curl -X DELETE http://<es-host>:9200/forum_posts   # 下次啟動時會以新 mapping 重建
```

## 頁面與 API

### 頁面

| 路徑 | 說明 |
| --- | --- |
| `/forum` | 論壇首頁（文章動態、搜尋、載入更多、回應、讚、檢舉） |
| `/forum/new` | 新增貼文（支援附圖） |
| `/forum/login` | 登入提醒與 Google 登入入口 |
| `/forum/profile` | 個人資料（暱稱、自我介紹、「我的追蹤」入口） |
| `/forum/others-profile` | 公開個人資料（以加密 `?user=` 金鑰、追蹤鈕、該使用者的貼文列表） |
| `/forum/following` | 追蹤：追蹤中的人 + 這些人的貼文動態（需登入） |
| `/admin` | 後臺：用戶管理與標籤管理（僅管理員） |
| `/admin/forum` | 論壇文章管理（僅管理員，含關鍵字搜尋） |
| `/admin/forum-report` | 檢舉管理（僅管理員） |

### API

| 方法 | 路徑 | 說明 |
| --- | --- | --- |
| `GET` | `/healthz` | MySQL、Redis 與 Elasticsearch 健康檢查（ES 未啟用時回報 `disabled`，不影響就緒判定） |
| `GET` | `/auth/google` | 導向 Google OAuth（支援 `?return=` 回傳路徑） |
| `GET` | `/auth/callback` | OAuth 回调，建立 Session |
| `POST` | `/api/logout` | 登出 |
| `GET` | `/api/check` | 查詢登入與管理員狀態 |
| `GET`/`POST` | `/api/forum/posts` | 文章列表（`?offset=`）/ 新增文章 |
| `GET` | `/api/forum/search` | 貼文搜尋（`?q=&offset=&limit=`；去識別化，ES 不可用時降級 MySQL LIKE） |
| `GET` | `/api/forum/posts/:id/comments` | 留言列表 |
| `POST` | `/api/forum/posts/:id/like` | 讚／收回讚 |
| `POST` | `/api/forum/posts/:id/comments` | 新增留言 |
| `POST` | `/api/forum/posts/:id/report` | 檢舉文章 |
| `GET`/`PUT` | `/api/forum/profile` | 讀取／更新個人資料 |
| `GET`/`POST` | `/api/forum/follows` | 我的追蹤清單／切換追蹤（需登入） |
| `GET` | `/api/forum/following/posts` | 追蹤中的人的貼文動態（需登入） |
| `GET` | `/api/forum/public-profile?key=` | 以金鑰讀取公開個人資料（含「我是否追蹤了他」） |
| `GET` | `/api/forum/public-posts?user=` | 以金鑰讀取該使用者的貼文分頁（公開、限流不加，因為走索引） |
| `POST` | `/api/forum/images` | 上傳貼文附圖（需登入） |
| `POST` | `/api/forum/image-tokens/release` | 釋放未使用的圖片 token |
| `GET` | `/api/admin/forum/posts` | 文章列表（含留言） |
| `GET` | `/api/admin/forum/search` | 貼文搜尋（含作者 Email 精確比對，結果含留言以供就地治理） |
| `POST` | `/api/admin/forum/posts`、`PUT`/`DELETE` `/api/admin/forum/posts/:id` | 新增／編輯／刪除文章 |
| `GET`/`POST` | `/api/admin/forum/comments`、`PUT`/`DELETE` `/api/admin/forum/comments/:id` | 留言管理 |
| `GET` | `/api/admin/forum/reports`、`GET`/`PUT`/`DELETE` `/api/admin/forum/reports/:id`、`PATCH` 狀態 | 檢舉管理 |
| `GET`/`POST` | `/api/admin/users`、`GET`/`PATCH`/`DELETE` `/api/admin/users/:email`、`GET`/`POST` `/api/admin/users/:email/posts`、`/comments`、`/content` | 用戶與內容管理 |
| `PUT` | `/api/admin/users/:email/tags` | 指派用戶標籤 |
| `GET`/`POST` | `/api/admin/tags`、`PATCH`/`DELETE` `/api/admin/tags/:id` | 標籤管理 |

## 前端架構

前端全部是 **React 19 + TypeScript**（`strict` + `noUncheckedIndexedAccess` +
`exactOptionalPropertyTypes` + `verbatimModuleSyntax`）。原始碼在 `frontend/web/src/`，
Vite 以九個 HTML 為入口建置；`?v=` 參數仍照原樣保留作 cache-busting。

### 為什麼仍是多頁，而不是單一 SPA

後端 `server.go` 的 `handleForumPage` / `handleForumLoginPage` 把 `/forum`、
`/forum/new`、`/forum/profile`、`/forum/others-profile`、`/forum/following`、
`/forum/login` 與三個
`/admin*` 路徑逐一對應到**各自檔名的 HTML**，而那段 `switch` 刻意讓未列出的子路徑
回 404。改成 SPA fallback 等於把這條白名單換成「任何路徑都吐同一份 index.html」，
放寲了一條刻意的路由規則。因此九個 HTML 保留原檔名，每個檔案只剩：

```html
<div id="root"></div>
<script type="module" src="/src/entries/forum.tsx"></script>
```

`script-src 'self'` 沒有 `'unsafe-inline'`，所以掛載點不能靠內聯腳本建立 ——
這也是每頁必須各自有一支入口模組的原因。

### 原始碼結構

| 檔案 | 職責 |
| --- | --- |
| `src/core.ts` | 全站底層：跟著語言走的日期／數字格式化、`text()`、`errorMessage` / `errorText`、`requestJSON`（公開頁唯一的 fetch 出口，含 `LoginRequiredError` 的 303/401 判斷）、`readJSONObject`、`goToLogin`、`logout`。純函式與 I/O 邊界，不含任何 React（只匯入 `i18n/runtime`，那也是純模組）|
| `src/i18n/*` | 多語系：語言清單與偵測、cookie、翻譯目錄、Provider、語言切換器（見「多語系」一節）|
| `src/types.ts` | 前端可見的後端 API 契約（用戶、標籤、文章、留言、檢舉、列表包裝） |
| `src/site.ts` | 站名：在模組層讀頁面殼的 `<meta name="forum-…">`（值由後端依設定檔代入），導出完整站名、短名與兩字標誌 |
| `src/api/admin.ts` | 後臺的 fetch 封裝：統一 401 導頁、把後端 `message` 包成 `ApiError`、`checkAdmin()` |
| `src/icons.tsx` | 圖示。路徑是寫死的常數，因此直接以 JSX 表達，**不需要** `dangerouslySetInnerHTML` |
| `src/admin/provider.tsx` | 後臺共用的狀態層：toast、`<dialog>`（`confirm` / `promptText` / `pickTags` 三種互動）、rail 上的用戶數徽章 |
| `src/admin/shell.tsx` | rail / topbar / 登入按鈕。active 項目與頁面標題是 props，沒有第二個真相來源 |
| `src/admin/ui.tsx` | 表格四種狀態（載入中 / 有資料 / 沒資料 / 載入失敗）、忙碌按鈕、表單狀態列 |
| `src/admin/UsersPage.tsx` | `/admin`：登入閘門、用戶統計與清單、標籤管理、逐篇內容治理 |
| `src/admin/ForumAdminPage.tsx` | `/admin/forum`：文章 CRUD、留言治理、關鍵字搜尋（取代分頁）、待處理檢舉快速裁定 |
| `src/admin/ReportAdminPage.tsx` | `/admin/forum-report`：逐筆裁定（通過並刪文／不成立）、紀錄更正與狀態篩選 |
| `src/forum/shell.tsx` | 公開頁共用的頂部／底部導覽、`useAuth`、PWA 安裝流程 |
| `src/forum/useFeed.ts` | 文章動態的分頁、重新載入、按讚 |
| `src/forum/useSearch.ts` | 貼文搜尋：關鍵字、分頁，以及 `engine`（Elasticsearch 或降級的 MySQL）|
| `src/forum/useComments.ts` | 每一篇文章的留言層狀態（展開、已載入幾筆、還有沒有更多、草稿） |
| `src/forum/useReport.ts` | 檢舉目標與理由 |
| `src/forum/PostCard.tsx` | 單篇貼文與其留言層。沒有 fetch，只負責把狀態畫成標記 |
| `src/forum/*.tsx` | 對應六個公開頁的頁面元件 |
| `src/entries/*.tsx` | 九個入口模組 |

公開頁的 import 圖裡不會出現任何 `admin/*` 模組：舊版靠註解記錄「公開頁不必為了
`escapeHTML` 而下載 `<dialog>` 樣板」，現在那是結構上的事實。

### 舊版 DOM 操作的對應物

| 舊版 | React 版 |
| --- | --- |
| `escapeHTML()` + `innerHTML` | 不需要了：React 在建立文字節點時就會跳脫。`text()` 只保留「後端欄位可能是 `undefined` 而舊版要顯示空字串」這個規則 |
| `$(...)` / `need(...)` | useState + JSX。骨架壞掉時是編譯期或 render 期錯誤，不是執行到某個呼叫點才炸 |
| `setBusy(button, true)` | `busy` state + `.is-busy` class（spinner 不改文字，因此按鈕寬度不跳動） |
| `createList()` | `ListState<T>` 型別（`loading` / `ready` / `error`）與 `<ListBody>` / `<EmptyState>` |
| `initShell()` 事後改 DOM | `active` / `pageTitle` 直接是 props |
| `toast()` 找 `#toasts` | `AdminProvider` 的 state |
| `confirmAction` / `promptText` / `pickTags`（單例 `<dialog>`） | provider 內的三個 Promise 函式，共用一支 `<dialog>` |

`src/entries/*.tsx` 刻意**不**包 `<StrictMode>`：本站頁面都是 effect 驅動的資料載入，
StrictMode 會在開發模式把 mount 跑兩次，等於每個頁面第一次開啟都發兩輪請求
（查詢端點是公開的，寫入端點會吃額度）。

### Service Worker 的型別檢查

`service-worker.ts` 跑在 WebWorker 環境，與頁面碼的 lib 互斥
（`lib.dom` 與 `lib.webworker` 對 `self` 的宣告會衝突）。因此型別檢查分兩個 project：

| 檔案 | lib | 內容 |
| --- | --- | --- |
| `tsconfig.json` | `ES2022` + `DOM` + `DOM.Iterable` | 全部頁面碼（`src/**`）與 `vite.config.ts` |
| `tsconfig.sw.json` | `ES2022` + `WebWorker` | 只有 `service-worker.ts` |

`service-worker.ts` 另外用 `const sw = self as unknown as ServiceWorkerGlobalScope`：
`lib.webworker.d.ts` 把 `self` 宣告成 `WorkerGlobalScope`，因此 `skipWaiting()`、
`clients`、`ExtendableEvent` 與 `FetchEvent` 全部看不到，而那些只掛在
`ServiceWorkerGlobalScope` 上。

建置時 `vite.config.ts` 會單獨編譯它成根層的 `service-worker.js`（IIFE，因為
service worker 以傳統 script 註冊）—— 這個路徑不能被 Vite 加上 hash，否則瀏覽器
註冊不到，而它的 scope 決定了 `/forum` 底下所有頁面是否在控制範圍內。

### 建置指令

```bash
npm run dev        # vite dev server（Fast Refresh）
npm run typecheck  # 頁面碼 + service worker 兩份 tsconfig
npm run build      # typecheck 後再 vite build → dist/
npm run preview    # 預覽 dist/
```

### 樣式的放置規則

CSS 分成兩層，分界線是「有幾個頁面在用」：

| 檔案 | 職責 |
| --- | --- |
| `style.css` | 九個頁面共用，或同一家族內被兩頁以上共用的樣式：reset、公開頁殼層（導覽列、底部導覽、膠囊按鈕）、**貼文卡與留言串**（`/forum` 與 `/forum/following` 共用）、追蹤鈕、後臺 design tokens、rail/topbar/canvas、面板、表格、對話框、toast |
| 各頁 HTML 的 `<style>` | 只被那一頁渲染的樣式：搜尋藥丸、追蹤頁的清單列、登入閘門、某一張後臺表格、某一個篩選器… |

調某一頁的版面時只需要開那個 HTML，不必在 CSS 與 HTML 之間來回跳；
要改導覽列或後臺殼層才動 `style.css`。這條分界線與 React 的模組邊界一致：
`src/forum/shell.tsx`、`src/admin/shell.tsx`、`src/admin/ui.tsx` 對應 `style.css`，
各頁 `*Page.tsx` 對應該頁的 `<style>`。

兩套設計系統共用一支 `style.css`，因此有兩件事靠載入順序解決：

- 同名 class 有兩種不同意義（`.eyebrow`、`.search-field`、`.google-btn`、`.loading`）
  一律**不**放 `style.css`，改由頁內 `<style>` 各自宣告。
- CSS 變數同名不同值（`--line`、`--accent`）：`style.css` 宣告的是後臺那一組，
  公開頁在頁內 `<style>` 的 `:root` 重新宣告。因為 `<link>` 一定在 `<style>`
  之前，同優先級下後者勝出。

### 後臺頁面

三個後臺頁面共用同一組檔案，這是刻意的：過去 `/admin` 用 `admin.css` + `admin.js`、
`/admin/forum` 與 `/admin/forum-report` 用 `forum-admin.css` + 各自腳本，兩套設計系統
的色票、類名語彙、內容最大寬度（1440 vs 1180）與響應式策略都不同，且同一個元件
（例如側欄）在兩邊是不同 class，維護時必然漂移。

| 檔案 | 職責 |
| --- | --- |
| `style.css` 第 3~16 節 | 唯一的 design token 來源 + 三頁共用的殼層與樣板元件（深色 rail、淺色 canvas、表格、對話框、toast、語言切換器、響應式斷點） |
| `admin.html` / `forum-admin.html` / `forum-report.html` | 各頁自己的元件樣式（用戶表與登入閘門、文章表與留言摺疊、檢舉表單與篩選器） |
| `src/admin/*` | 三頁共用的 provider、殼層與樣板元件（見上表） |

Vite 會把三個入口共用的那部分打成一組共享 chunk（`core` 與 `admin` UI），
`style.css` 也打包成一份由九個入口共用的檔案。

三頁共用的硬性約束：

- `script-src 'self'` 沒有 `'unsafe-inline'`：禁止內聯 `<script>`，掛載必須由外部
  模組完成。顯示與隱藏一律用 `hidden` 或 class（例如登出鈕用
  `.logout-button--visible`，不用 `element.style.display`）。
- `style-src` 沒有 `'unsafe-inline'`，但**允許** `<style>` 區塊：各頁 HTML 裡的
  `<style>` 內容由後端在啟動時直接從磁碟上的 HTML 算出 SHA-256，逐一放行進
  `style-src`（見 `backend/forum/httpapi/securityheaders.go` 的 `inlineStyleHashes`）。
  雜湊值不必手動維護。`style="..."` 屬性仍然禁止。
- **但雜湊是在「啟動時」算的 —— 改完某一頁的 `<style>` 必須重啟後端。**
  忘記重啟的症狀相當不直觀，值得記下來：只有被改的那一頁失去樣式，其餘八頁完全
  正常（它們的雜湊仍對得上）；失去的是**整個區塊**而不只是你剛加的那條規則，所以
  畫面會退化成一堆看起來互不相關的問題（清單長出項目符號、`display: flex` 失效、
  元素擠在一起）；而 `style.css` 照常作用（走 `style-src 'self'`），因此看起來
  「有樣式但怪怪的」。診斷不必開 devtools：把每個 `dist/*.html` 的第一個 `<style>`
  區塊算一次 base64 SHA-256，看有沒有出現在線上 CSP 標頭的 `style-src` 清單裡，
  缺哪一頁就是哪一頁需要重啟。
- 後臺頁面**不**註冊 Service Worker（`usePwaInstall()` 只在公開頁使用）。預快存清單
  與 manifest 的 scope 都只涵蓋 `/forum`，後臺不在可安裝應用範圍內。

## 多語系

支援 **十七種語言**，公開頁與後臺九個頁面全部覆蓋。右上角顯示兩字母代碼
（`TW` / `HK` / `MO` / `CN` / `EN` / `JP` / `KO` / `VN` / `TH` / `ID` / `FR` /
`DE` / `ES` / `BR` / `RU` / `AR` / `HI`），點開後是全部語言的下拉清單。

| 語言 | 代碼 | 檔案 | 匯出 |
| --- | --- | --- | --- |
| 繁體中文（台灣） | `zh-TW` | `messages.ts` | `zhTW` |
| 繁體中文（香港） | `zh-HK` | `translations/zh-HK.ts` | `zhHK` |
| 繁體中文（澳門） | `zh-MO` | `translations/zh-MO.ts` | `zhMO` |
| 简体中文 | `zh-CN` | `translations/zh-CN.ts` | `zhCN` |
| English | `en` | `translations/en.ts` | `en` |
| 日本語 | `ja` | `translations/ja.ts` | `ja` |
| 한국어 | `ko` | `translations/ko.ts` | `ko` |
| Tiếng Việt | `vi` | `translations/vi.ts` | `vi` |
| ไทย | `th` | `translations/th.ts` | `th` |
| Bahasa Indonesia | `id` | `translations/id.ts` | `id` |
| Français | `fr` | `translations/fr.ts` | `fr` |
| Deutsch | `de` | `translations/de.ts` | `de` |
| Español | `es` | `translations/es.ts` | `es` |
| Português (Brasil) | `pt-BR` | `translations/pt-BR.ts` | `ptBR` |
| Русский | `ru` | `translations/ru.ts` | `ru` |
| العربية | `ar` | `translations/ar.ts` | `ar` |
| हिन्दी | `hi` | `translations/hi.ts` | `hi` |

阿拉伯文是唯一的**右至左（RTL）**語言。

| 檔案 | 職責 |
| --- | --- |
| `src/i18n/locales.ts` | 語言清單（代碼、名稱、`<html lang>`、**書寫方向**）與「瀏覽器語言 → 站台語言」的對應表。純模組，不含 React |
| `src/i18n/messages.ts` | **型別基準**：繁體中文原文。`MessageKey` 由它推導 |
| `src/i18n/translations/*.ts` | 其餘十六種語言，型別是 `Record<MessageKey, string>` |
| `src/i18n/runtime.ts` | 偵測、cookie、`t()`、延後翻譯的 `msg()` / `tr()`。純模組 |
| `src/i18n/index.tsx` | `I18nProvider`、`useI18n()`、`usePageTitle()` |
| `src/i18n/LanguageSwitcher.tsx` | 切換器元件。`variant` 決定外觀，互動邏輯兩邊共用 |

### 右至左（阿拉伯文）

`<html dir>` 由 `LocaleMeta.dir` 決定，與 `lang` 一起在 `useLayoutEffect` 寫入
（繪製前完成，讀螢幕軟體不會讀錯語言、版面也不會先以 LTR 畫一帧）。

版面鏡像分兩部分：

- **不需要寫程式**：`style.css` 與各頁 `<style>` 裡的 grid 欄位、flex 排列、
  `justify-self`、`text-align: start|end` 全部跟著 `dir` 翻。
- **需要改成邏輯屬性**：實體值不會翻。已改的有 `inset-inline-*`（rail、toast、
  skip link、搜尋框圖示、語言選單）、`margin-inline-*`（貼文縮排、thread 的按鈕列、
  check-list 的 ID）、`text-align: start|end`（表格）、`border-inline-start`
  （toast 的狀態色條）。
- **邏輯屬性表達不了的三處**，因此在 `style.css` 第 17 節用 `[dir="rtl"]` 覆寫：
  1. `transform: translateX(-102%)` —— 收起的抽屜要改成 `+102%`。
  2. `box-shadow: inset 3px 0 0` —— 這個 `0` 是偏移量不是位移。
  3. `▸` / `▾` —— 字形方向固定，展開三角要換成 `◂`。
  另外把「返回」與「登出」兩個方向性圖示鏡像；底部導覽的 home / add / person
  是對稱圖形，翻了反而奇怪，因此明確排除。

字體：`--font` 堆疊的尾端放了 `Noto Sans Arabic` / `Noto Sans Devanagari` /
`Noto Sans Thai`。Inter 與 Noto Sans TC 都沒有這三種文字的字形，沒有它們的系統
會掉到預設字體，字重與字距因而與介面其他部分對不上。它們**沒有**被當成 webfont
載入（本站的 `<link>` 只有 Inter / Noto Sans TC / Material Symbols），這幾項是
「作業系統有的話就用」的意思。

### 語言決定的三個步驟

1. **偵測**（`resolveInitialLocale`）依序試：cookie → `navigator.languages` →
   `<html lang>` → `zh-TW`。順序代表「誰的意圖比較明確」：cookie 是使用者在本站
   明確選過的，瀏覽器語言是使用者設定的偏好，`<html lang>` 是樣板的預設。
   BCP 47 的比對用前綴（`zh-Hant-*` / `en-*` / `ja-*` 都收），並剝掉 `-u-` 底層的
   副語言擴充。
2. **套用**（`applyLocale`）一次做三件事：更新模組狀態（`t()` 讀的）、寫
   `<html lang>`、寫 cookie。cookie 名稱 `hpnm_lang`，`Path=/; Max-Age=1y;
   SameSite=Lax`，僅在 https 下加 `Secure`（localhost 是 http，硬加會被丟棄）。
3. **持久化**：每次載入都會寫回 cookie，**包含第一次由瀏覽器語言推導出來的那一次**。
   這是需求「自動切換到瀏覽器語言」加「保存 cookie」合起來的行為 —— 使用者第一次
   進站看到自己瀏覽器的語言，此後不論他怎麼調整瀏覽器偏好，站內介面都維持他看到的
   那一種。只在手動切換時寫入的話，改了瀏覽器語言隔天整個介面就會自己變掉，而且
   沒有任何提示。

### 三條容易踩到的規則

- **不要在事件處理器裡呼叫 `t()` 然後存進 state。** 狀態列、摘要列、錯誤訊息這類東西
  是「已載入資料的純函式結果」，在 setter 裡就翻譯的話，使用者切換語言時畫面會停在舊
  語言。存 `msg('key', { params })`，render 時用 `tr()` 解析。
- **後端回來的訊息不翻譯。** 我們沒有那些字串的譯文，也不該有。因此 `tr()` 同時接受
  `Message` 與 `string`，而 `core.ts` 的 `errorText(error, msg('key'))` 就是為了把兩種
  來源分開：`errorMessage()` 是給一次性顯示的（toast、錯誤處理）用的舊版，
  需要存進 state 的錯誤訊息走 `errorText()`。
- **模組層的常數不能放要翻譯的字串。** `const TABS = [{ label: '首頁' }]` 會在
  import 時把當下的語言固定住，整個 session 都不會跟著換。改成函式
  （`bottomTabs()` / `navItems()` / `filters()`）。

### 加一條文案的流程

1. 在 `src/i18n/messages.ts` 加一個鍵（繁體中文是原文，抄現有介面上的字串）。
2. `npm run typecheck` 會列出十七個還沒翻的語言。
3. 在 `translations/*.ts` 各補一條。

漏翻會是**編譯期錯誤**（`Record<MessageKey, string>` 少一個欄位），不會有人到介面上
才發現某顆按鈕還是中文。佔位符 `{name}` 同樣要逐字保留。

### 批次翻譯的作法（新增語言／補翻時用）

十七種語言、409 條文案要靠代理模型批次產生時，**不要**讓代理模型直接寫 TS 檔案。
本地模型的 context 很小，讀 25KB 的 `messages.ts` 再輸出 409 筆 TS 字串會直接
overflow（實際測過：12 個語言的任務全部失敗）。可行的流程是把工作拆成三段，中間用
純文字檔交換：

1. **切工作單**：把 `messages.ts` 的鍵與原文抽成 `key<TAB>原文` 的 TSV，切成每份
   約 96 行。必須原樣保留的 11 個鍵（拉丁字母排版裝飾與產品名）不放進去，由合併
   階段直接填原值。
2. **翻譯**：一個代理任務只做一件事 —— 讀一份 TSV，寫出同樣格式的 TSV。不碰引號
   跳脫、結尾逗號、型別，也不用跑任何驗證。
3. **合併**：腳本以「鍵」為準把任意數量的 TSV 合併成 TS 檔。分段只是代理模型的
   輸入單位，對輸出格式沒有影響，因此「某段重跑」不需要考慮其他段。

合併腳本同時是品質關卡，四種失敗是實際發生過的：

| 失敗 | 處理 |
| --- | --- |
| 少寫了行、整段沒回來 | 報 missing，並輸出 `-todo.tsv` 讓下一輪只補缺的（可反覆迭代到收齊） |
| 分隔符被寫成空格或句點 | 對照預期的鍵前綴自動修復（已知：印尼語 3 行、西班牙語 1 行） |
| 佔位符被丟掉或改名 | **不修復、直接報錯** —— 猜測語意比壞掉更糟（已知：越南語 8 條把 `{id}` 寫成 `#id`） |
| 整段原樣抄回中文 | 報錯（只比對漢字；全形 `｜` 與 `（）` 是刻意保留的，不算） |

另外兩個容易踩的坑：

- **別用字面 CJK 字元寫 regex 範圍。** 經過編碼往返會悄悄錯位，實際上因此把韓文
  誤判成「未翻譯」。用 `\u3400-\u4DBF` 這種明確逸出。
- **「日文與中文可以用漢字判斷未翻譯」是錯的** —— 日文介面裡出現 `確認`、`削除` 是
  正常的。中文與日文要排除在這項檢查之外。

**只新增十幾條文案時不要走上面的流程。** `merge-translations.mjs` 會要求
`-todo` 收齊為止，也就是**整份目錄 409 條都要重新產出**；為了 16 個新鍵重翻
16 種語言的全文既慢又容易讓既有用詞漂移。實際作法是：在 `messages.ts` 加鍵 →
`npm run typecheck` 讓編譯器列出每個語言缺哪幾個 → 在各 `translations/*.ts`
對應位置插入那幾筆（注意各檔的排版不一致：`en` / `ja` / `zh-CN` 是單行
`'key': '值',`，其餘 13 檔是 `  'key':` 換行 `    '值',`）→
`node ../tools/i18n/verify-catalogs.mjs` 驗收。


### 已知未涵蓋

`forum-manifest.json` 的 `name` / `description` 來自設定檔的單一值，**不跟著語言
走**（站名是可設定的專有名詞，本來就沒有譯文）。Manifest 是瀏覽器在「安裝」當下
另外抓的檔案，要跟著語言走需要後端依 cookie 提供不同版本（另一組端點）。
`<html lang>` 本身在掛載後即時更新（`useLayoutEffect`，繪製前完成，讀螢幕軟體
不會讀錯語言），但**送出 HTML 時**它還是樣板的 `zh-Hant` —— 對沒有 JavaScript 的
爬蟲或讀屏器而言，語言標記與內容不符。

## 存取控制

- 讀取文章列表、留言與公開個人資料不需登入；其餘寫入操作需登入。
- 追蹤清單（`/api/forum/follows`）與追蹤動態（`/api/forum/following/posts`）是
  **私有讀取**，兩者都需登入；`/forum/following` 頁面本身也需登入。
- 因為 `requireLogin` 對 API 路徑同樣回 `303` 轉址到登入頁，前端**不得**在未確認
  登入前就請求那兩支端點 —— `requestJSON` 會把轉址後的登入頁 HTML 判成
  `LoginRequiredError` 而把整頁導走，未登入者連首頁都開不了。`useFollow` 因此
  接受 `autoLoad`，由呼叫端綁 `useAuth()` 的結果決定。
- 未登入存取受保護頁面或 API 會以 `303` 導向 `/forum/login?return=<原路徑>`。
- 非管理員存取 `/api/admin/*` 回傳 `401`；停權帳號回傳 `401`「此帳號已被停權」。
- 變更狀態的請求會檢查 `Origin` 是否在 `TRUSTED_ORIGINS` 內。
- 後臺 API 刻意不掛限流：操作本身需管理員身分，且管理員有合理的批次操作需求。

## 速率限制

以用戶端 IP 為單位，採行程內滑動視窗日誌（預設 10 次 / 分鐘一組）。分三組是因為各端點的單次成本差異極大，混用同一份額度的結果必然是「要嘛放行上傳轟炸、要嘛把留言一起擋掉」。

| 組別 | 套用端點 | 預設 |
| --- | --- | --- |
| 內容寫入 | `POST /api/forum/posts`、`/api/forum/posts/{id}/*`（留言、按讚、檢舉）、`POST /api/forum/follows`、`PUT /api/forum/profile` | 10 次 / 60 秒 |
| 圖片上傳 | `/api/forum/images`、`/api/forum/image-tokens/release` | 5 次 / 60 秒 |
| OAuth | `/auth/google`、`/auth/callback` | 10 次 / 60 秒 |

- **內容端點只擋非 GET**：讀取端點對匿名訪客開放，限流它們會直接壞掉首頁與「載入更多」。
- **OAuth 端點限流所有方法**：登入跳轉本身就是 `GET`，若沿用「放行 GET」的規則，
  限流會變成完全無作用的死設定。
- 超限時回 `429` 並附 `Retry-After`（秒數），讓瀏覽器與爬蟲能自動退避。
- 限流排在登入檢查**之後**：未登入的垃圾流量不會消耗已登入使用者的額度。
- 限流器只存在行程記憶體，重啟後計數歸零；背景 goroutine 會定期回收已過期的
  用戶端紀錄，避免 map 隨時間只增不減。
- **已知限制**：多執行個體部署時實際上限是設定值 × 執行個體數，需全站一致時
  應改用 Redis 計數。以 IP 為 key 代表同一個 NAT 出口後的使用者共享額度。

## 安全標頭

`httpapi/securityheaders.go` 在整棵路由樹之外統一套用，HTML、JSON、靜態資產與
Service Worker 皆一致：

`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、
`Referrer-Policy: strict-origin-when-cross-origin`、
`Permissions-Policy`（關閉相機／麥克風／定位／支付／USB）、
`Cross-Origin-Opener-Policy: same-origin`，
以及 `Strict-Transport-Security`（僅在 `COOKIE_SECURE=true` 時送出）。

`Content-Security-Policy` 採嚴格白名單：`default-src 'self'`、`base-uri 'self'`、
`object-src 'none'`、`frame-ancestors 'none'`、`form-action 'self'`、
`manifest-src 'self'`、
`style-src` 另加 `https://fonts.googleapis.com`（Google Fonts）、
`font-src` 另加 `https://fonts.gstatic.com`、
`img-src` 另加 `data:` 與 `FILES_SERVER_PUBLIC_URL`（貼文附圖）。

全站唯一的第三方來源是 Cloudflare Browser Insights，由 Cloudflare 邊緣注入、
不在本 repo 內：`script-src` 放行 `https://static.cloudflareinsights.com`
（`beacon.min.js`），`connect-src` 放行 `https://cloudflareinsights.com`
（量測資料 POST 到 `/cdn-cgi/rum`）。兩者缺一不可 —— 只加 `script-src` 會得到
「主控台乾淨、但 RPM 永遠是 0」的靜默失敗。未啟用該功能時邊緣不會注入，
這兩條就是沒人使用的空條目。

- **不使用 `'unsafe-inline'` / `'unsafe-eval'`**：後臺與論壇各頁的邏輯都在外部模組
  （`src/entries/*.tsx` 與其 import 圖），掛載點是 HTML 裡的 `<div id="root">`，
  因此不需要放寬。新增前端功能時必須維持「只用外部腳本、不用內聯 style 屬性」，
  否則後臺會整頁失效。
- `/service-worker.js` 是唯一例外，不送 CSP：它在 worker 環境求值並會主動
  fetch 多個網址做預快取。
- 新增外部資源（CDN、字型、圖床）時，必須同步更新
  `buildContentSecurityPolicy`，否則瀏覽器只會顯示一行 `Refused to load`。

### 貼文附圖與 `img-src`

`forum_posts.image_url` **只存檔名**（例如 `9f8c1a2b.png`），完整網址一律由
`Server.forumImageURL` 現組：`FILES_SERVER_PUBLIC_URL` + `/files/` + 檔名
（+ `?token=`）。因此：

- 換對外網域只要改設定，不必 `UPDATE` 全表，也不會讓內部位址
  （`http://192.168.66.5:7070`）留在資料庫裡。
- 對外網域必須是**檔案伺服器自己**的路徑。`forumImageFileName` 只認
  `FILES_SERVER_PUBLIC_URL`、`FILES_SERVER_URL` 與
  `localhost`／`127.0.0.1`／`::1` 四種主機，其他主機一律視為「不是本站的圖片」。

寫入端（`POST /api/forum/posts`）只接受本站上傳的圖片，第三方網址回 400 ——
舊版雖允許存任意網址，但 CSP 的 `img-src` 本來就擋掉它們，實際只會得到破圖。

`MigrateMySQL` 的第 18 步會把既有的完整網址轉成檔名
（`WHERE image_url LIKE '%/%'`，冪等）。應用層同時容忍尚未轉換的舊寫法，
兩者不必同步；只有「舊資料是第三方網址」那一種情況會讓圖片消失（回空字串），
若要保留須先把檔案搬到自己的檔案伺服器。

`img-src` 被擋下時先確認瀏覽器主控台的實際來源：白名單裡的
`FILES_SERVER_PUBLIC_URL` 卻仍被擋，代表請求沒帶 `?token=`（Redis 沒有那把
key，見 `MEDIA_TOKEN_*` 設定與「圖片檔案伺服器」一節），而不是 CSP 設定漏了
來源。想確認資料庫是否已轉成檔名：

```sql
-- 應只看到檔名（不含 "/"）；仍看到網址表示遷移沒跑或曾被人工改回
SELECT id, image_url FROM forum_posts WHERE image_url LIKE '%/%' LIMIT 20;
```

## PWA

`frontend/forum-manifest.json` 與 `service-worker.ts` 由建置流程編譯／複製到 `dist`，
`/service-worker.js` 與 `/forum-manifest.json` 由後端直接提供（manifest 另會以設定檔
的站名代入佔位符，見「站名」一節）。兩者的路徑都被外部世界綁死（瀏覽器註冊 URL、
各頁的 `<link rel="manifest">`），因此在 `vite.config.ts` 中以 plugin 直接 emit，
不走 Rollup 的 hashed 入口；同一個 plugin 也會把 Vite 自動產生的 manifest 雜湊複本
改回根層路徑並刪除，理由見「站名」一節的 manifest 小節。
