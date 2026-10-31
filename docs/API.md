# API 參考

回應格式一律 JSON，錯誤以非 2xx 狀態碼回傳。各端點的語意細節與取捨見 [`ARCHITECTURE.md`](ARCHITECTURE.md)。

- [基礎](#基礎)
- [論壇](#論壇)
- [收藏](#收藏)
- [貼文附圖與頭像的替換](#貼文附圖與頭像的替換)
- [圖片的所有權與刪除](#圖片的所有權與刪除)
- [RSS](#rss)
- [管理](#管理)

---

## 基礎

| Method | Path | 說明 |
| --- | --- | --- |
| GET | `/healthz` | ping MySQL / Redis / ES，全通 200 否則 503 |
| GET | `/api/check` | 永遠 200；回 `{ ok, isAdmin }` |
| GET | `/auth/google` | 307 導向 Google（**限流，所有 method**） |
| GET | `/auth/callback` | OAuth 交換 token 並種 session |
| POST | `/api/logout` | 需要可信 Origin |

---

## 論壇

讀取匿名可達；寫入需要登入並受限流。分頁型 GET 另受**讀取額度**（預設 60 次／分鐘，
`RATE_LIMIT_READ_*`）約束 —— 它專門擋掉爬蟲、自動翻頁腳本與失控的連鎖載入，被拒時回
429 帶 `Retry-After`。刻意不吃那份額度的端點：`/api/forum/public-profile`（設計給匿名
訪客）、`/api/forum/notifications/unread-count`（每個頁面載入都打一次）、
`/api/forum/notifications/ws`（長連線）、`/feed.xml`，以及全部後台路由。

| Method | Path | 說明 |
| --- | --- | --- |
| GET | `/api/forum/posts?offset=&limit=` | 貼文列表（吃讀取額度） |
| POST | `/api/forum/posts` | 建立貼文 |
| GET | `/api/forum/posts/{id}` | 單篇貼文（永久連結頁；匿名可讀，找不到回 404） |
| PUT | `/api/forum/posts/{id}` | 編輯自己的貼文：`content` 必填，`imageUrl` 選填（省略／`null`＝不動、`""`＝移除、有值＝換圖，見 [附圖的替換](#貼文附圖與頭像的替換)） |
| DELETE | `/api/forum/posts/{id}` | 軟刪除自己的貼文（`deleted_at` 寫入時間，資料保留） |
| GET | `/api/forum/posts/{id}/comments` | 留言列表（公開；含回覆，最多三層；吃讀取額度） |
| POST | `/api/forum/posts/{id}/comments` | 建立留言；body 可帶 `parentId` 回覆某一則留言 |
| PUT / DELETE | `/api/forum/posts/{id}/comments/{cid}` | 編輯／刪除自己的留言（刪除連同它的回覆子樹；回應帶 `removed`） |
| POST | `/api/forum/posts/{id}/like` | 按讚 |
| POST | `/api/forum/posts/{id}/bookmark` | 切換收藏（見 [收藏](#收藏)） |
| GET | `/api/forum/bookmarks` | 我收藏的貼文（需登入；吃讀取額度；單頁 25 筆，與動態相同） |
| POST | `/api/forum/posts/{id}/report` | 檢舉貼文 |
| POST | `/api/forum/posts/{id}/comments/{cid}/report` | 檢舉留言 |
| POST | `/api/forum/images` | 上傳圖片，轉送 files_server |
| POST | `/api/forum/images/delete` | 刪除自己上傳且未被引用的圖片（body `{url}`，見 [附圖的替換](#貼文附圖與頭像的替換)） |
| POST | `/api/forum/image-tokens/release` | 釋放媒體存取權杖 |
| GET / PUT | `/api/forum/profile` | 自己的資料（暱稱、簡介、頭像；PUT 為整份覆寫） |
| GET / POST | `/api/forum/follows` | 追蹤名單 / 切換追蹤（名單是分頁的，GET 吃讀取額度） |
| GET | `/api/forum/following/posts` | 私密動態牆（需登入；吃讀取額度） |
| GET | `/api/forum/my-posts` | 自己的貼文（需登入；吃讀取額度；未發過文回空清單而非 404） |
| GET / POST | `/api/forum/notifications` | 我的通知 / 標記已讀（GET 吃讀取額度；POST 只是一次索引更新，不限流。POST 帶 `{id}` 標一則，`id` 為 0 或缺省標全部） |
| GET | `/api/forum/notifications/unread-count` | 未讀通知數（導覽列鈴鐺用；需登入；**刻意不限流** —— 每個頁面載入都打一次，而它只取一個整數） |
| GET | `/api/forum/notifications/ws` | 即時通知的 WebSocket（需登入；**刻意不限流** —— 它是升級成長連線，連線數上限在 handler 內；訊息格式見下） |
| GET | `/api/forum/public-profile?key=` | 以 `public_key` 查公開資料（無需登入；**刻意不限流** —— 設計給匿名訪客，且不是分頁端點） |
| GET | `/api/forum/public-posts?user=` | 以 `public_key` 查貼文（無需登入；吃讀取額度） |
| GET | `/api/forum/search?q=&offset=&limit=` | 全文搜尋；ES 不可用時退回 `LIKE`（吃讀取額度） |

> 兩個 `public-*` 端點的參數名**不一致**（`key` vs `user`），這是現況不是筆誤。

### 即時通知（WebSocket）

`GET /api/forum/notifications/ws` 在同源且已登入的頁面上建立一條長連線，把「剛產生的通知」
推給瀏覽器，前端因此不必等下一次頁面載入。訊息全部是 JSON，也全部由伺服器送出：

| 訊息 | 內容 |
| --- | --- |
| `ready` | `{"type":"ready","unreadCount":3}` —— 連線建立時送一次 |
| `notification` | `{"type":"notification","unreadCount":4,"item":{…}}` —— 有新的通知時送 |

- `item` 的欄位與 `GET /api/forum/notifications` 的單一項目**完全相同**（後端共用同一份組裝
  程式碼），因此前端可以直接把它插進清單最前面。
- 只推**連線期間**產生的通知；斷線期間的不補送 —— 重新連上時前端會重新載入清單（`ready`
  送過第二次即代表這是重新連線）。
- 瀏覽器不需要送任何訊息，送來的內容一律忽略（但仍會持續讀取，pong 是對方還活著的證據）。
- 上限：同一使用者 8 條連線（超過時關掉**最舊**的一條，不是拒絕新的）、全站 1000 條（滿了
  直接拒絕新連線）。兩者被關掉時都會收到一個帶原因的 close frame。單一連線的傳送佇列滿了
  （對方讀得比我們寫得慢）也是關掉那條連線，而不是讓它拖住其他推播。
- close frame 的應用層 code 是**契約的一部分**（前端靠它決定要不要重連）：

  | code | 意思 | 前端 |
  | --- | --- | --- |
  | 4000 | 被同一使用者的較新連線取代 | **不重連**（重連只會讓兩條連線互相汰換） |
  | 4001 | 全站連線數已達上限 | 照退避重連 |
  | 4002 | 讀得比伺服器寫得慢，被關掉 | 照退避重連 |
  | 4003 | session 失效或帳號被停權 | 照退避重連 |
- 身分每分鐘重新確認一次：session 被撤銷或被停權就關閉連線。
- 跨站來源回 403（與其他寫入端點共用 `TRUSTED_ORIGINS`）。
- 反向代理必須放行 `Upgrade`／`Connection` 標頭，見[部署](DEPLOYMENT.md#部署前檢查表)。

### 頭像

上傳沒有獨立端點：頭像沿用 `/api/forum/images`，寫入走 `PUT /api/forum/profile`。

| 用途 | 怎麼做 |
| --- | --- |
| 換頭像 | `POST /api/forum/images` 上傳取得網址 → `PUT /api/forum/profile` 帶 `{nickname, bio, avatarUrl}` |
| 移除頭像 | `PUT /api/forum/profile` 帶 `avatarUrl: ""` |

- `avatarUrl` 與 `bio` 同一個語意：**整份覆寫**。省略等同送空字串，因此呼叫端每次都要把自己現在的值一起送回來。
- 回應欄位：`GET /api/forum/profile`、`GET /api/forum/public-profile` 回 `avatarUrl`（已附加 media token）；貼文與留言多帶 `authorAvatar`；追蹤清單的每一項多帶 `avatarUrl`。三處都省略空值。
- 讀取端點一共有四個讀者：`/forum/profile`（編輯自己的頭像）、`/forum/others-profile`（看別人的）、`/forum/following`（追蹤清單），以及 `/forum/new` 的 composer 那一顆「我」。因此 `GET /api/forum/profile` 在已登入的頁面上會被重複呼叫 —— 這是有意的：四個頁面各只需要自己那一份，而共用一個快取會讓「剛改了頭像」在某一頁還是舊的。
 - `PUT /api/forum/profile` **不回** `avatarUrl` —— 呼叫端送進去的值就是寫入的值，多回一份只會讓它有第二個來源。
- 被替換或移除的舊檔，會在**不再被任何貼文或頭像引用**時一併刪除（見 [附圖的替換](#貼文附圖與頭像的替換)）。

## 收藏

收藏是**私人**的備忘，不是公開的按讚：沒有任何回應會說一則貼文被幾個人收藏。

| Method | Path | 說明 |
| --- | --- | --- |
| POST | `/api/forum/posts/{id}/bookmark` | 切換收藏（已收藏→取消、未收藏→新增）；回 `{"bookmarked":…}` |
| GET | `/api/forum/bookmarks?offset=` | 我收藏過的貼文（需登入、吃讀取額度；單頁 25 筆） |

- 回應只有 `bookmarked`（切換**後**的狀態）：連點兩下的合併結果只有後端的交易知道，前端不猜。
- 清單只列**還活著**的貼文 —— 一則被作者刪除的貼文不再出現，即使有人收藏過它（與所有公開列表同一條規則）。還原之後它會回到清單裡。
- 排序與全站動態相同（置頂優先、再依發表時間），不是依收藏時間：置頂的貼文在每個列表都看得到。
- 對已刪除或不存在的貼文收藏回 404；收藏自己的貼文是允許的。取消收藏是真的 `DELETE`（沒有歷史語意，與按讚、追蹤相同）。

## 貼文附圖與頭像的替換

兩者共用同一個流程：**先上傳、再寫入**（`POST /api/forum/images`，然後各自的 `PUT`），差別只在欄位名。

| 動作 | 貼文附圖（`PUT /api/forum/posts/{id}`） | 頭像（`PUT /api/forum/profile`） |
| --- | --- | --- |
| 不動 | 欄位**省略**（或 `null`） | 沒有這個選項 —— 頭像每次都是整份覆寫 |
| 換成新圖 | `imageUrl` 帶新的網址 | `avatarUrl` 帶新的網址 |
| 移除 | `imageUrl: ""` | `avatarUrl: ""` |

- 貼文附圖是三態（`省略／null`／`""`／有值），`content` 仍為必填。刻意不用 `null` 表示「移除」：JSON 的 `null` 與「欄位不存在」解碼後都是 `nil`，無法區分。
- **換圖會回收舊檔**：更新成功之後，舊檔名若不再被任何貼文或頭像引用，就從檔案伺服器刪除。回收是 best-effort（失敗只留日誌），因此它不會把一次成功的編輯變成失敗。
- **軟刪除的貼文仍然算引用**：被刪除的貼文還原之後必須還看得到它的圖，所以「刪文」不會把圖刪掉。
- 寫入時只取網址裡的**檔名**（資料庫只存檔名的既有契約），且該檔名必須是本站的檔案、**上傳者是自己**（見下）。

## 圖片的所有權與刪除

「這個檔名是誰上傳的」記在 `forum_media`，上傳成功時寫入。它回答兩個問題：這張圖能不能掛到我的內容上、以及能不能被我刪掉。

| Method | Path | 說明 |
| --- | --- | --- |
| POST | `/api/forum/images/delete` | 刪除自己上傳、且沒有被任何貼文或頭像引用的圖片（body `{url}`） |

| 情況 | 結果 |
| --- | --- |
| 是我上傳的，且沒有任何引用 | 200 `{"ok":true}`，檔案刪除 |
| 檔名無效（第三方網址、路徑穿越、不在副檔名白名單裡） | 400 `{"error":"invalid image url"}` |
| 不是我上傳的，或**沒有上傳紀錄**（舊檔案，含從未上傳過的檔名） | 404 —— 與「這個檔名不存在」共用同一個回應，因此無法用它探測別人上傳過什麼 |
| 仍被某一篇貼文或某個頭像引用 | 409（呼叫端要先把引用改掉） |
| 檔案伺服器說檔案已經不在 | 視為成功（重試安全） |

- 這條端點是給「上傳了但沒用到」的情況（選錯圖、發文失敗後重試）；正常換圖／移除由上面的 `PUT` 自動回收。
- 上傳時若寫不進所有權紀錄，該次上傳回 500 並把剛上傳的檔案刪掉：寧可失敗重傳，也不要留下一個沒有人能刪的檔。
- 舊檔案（`forum_media` 存在之前上傳的）沒有紀錄，因此**不能被刪除**；它們唯一還能做的是被掛回**自己既有的**內容上（例如個人資料頁把目前的舊頭像原樣送回來）。見 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md#這次改動之前留下的孤兒檔無法清除)。

### 檔案層對上傳內容的規則（`POST /api/forum/images`）

圖片的型別、尺寸與中繼資料是在 **files_server** 決定的，不是在後端（實作在
`files_server/imagepolicy.go`，政策值來自 `files_server/config.conf` 的 `[upload]`，
見 [設定](CONFIGURATION.md#files_serverconfigconf)）。對呼叫端而言這改變了三件事：

| 條件 | 結果 |
| --- | --- |
| 副檔名在 `[upload].allowed_files` 裡，但**位元組不是那四種圖片之一**（HTML／腳本／ZIP／音訊被改名成 `photo.jpg`） | files_server 回 400 → 後端轉 502 `{"error":"file server rejected image"}`。副檔名不是事實，位元組才是 |
| 內容與副檔名不一致（叫 `a.png` 的 JPEG） | 存下來，但落地檔名與回應網址的副檔名**由內容決定**（`.jpg`）—— 這樣 `Content-Type` 才不會與實際內容矛盾 |
| JPEG／PNG 超過 `max_width`／`max_height` | 存下來，**等比縮小**（不放大）。長寬比保持 |
| GIF／WebP 超過 `max_width`／`max_height` | 400 → 後端 502。這兩種格式不被縮小（標準庫沒有 WebP 解碼器，`image/gif` 只解第一影格） |
| 來源解析度超過 `max_source_pixels`（寬×高） | 400 → 後端 502。這一道在任何解碼**之前**檢查，因此一個幾 KB 的「解壓炸彈」不會吃掉記憶體 |
| 簽章對但內容壞了（標頭缺 SOF、位元流被截斷、半張 GIF） | 400 → 後端 502。被拒的上傳**不留檔案** |

中繼資料的代價要寫清楚：**存下來的是重新編碼過的檔案**，因此原檔不再可取回，雜湊與檔大小
也與上傳者手上的那份不同（讀取端只讀這份存下來的檔案，對論壇而言「原檔」沒有必要）。
EXIF 含 GPS、相機型號、拍攝時間與軟體名稱，這些不會進儲存 —— 那是這個站匿名性立論的一部分
（見 [安全](SECURITY.md#檔案上傳)）。`[upload].reencode = false` 會把它們留下來，
而行程啟動時會印一行 `WARNING` 說這件事。

後端把 files_server 的任何非 2xx 統一成 **502**，因此「被縮小」與「被拒」在回應形狀上無法
區分（前者不會發生 —— 縮小是 200）。日誌裡的區别在 files_server 那一邊，後端不帶原因。

---

## RSS

| Method | Path | 說明 |
| --- | --- | --- |
| GET / HEAD | `/feed.xml` | 全站貼文的 RSS 2.0 摘要（公開、不限流） |

- 掛在**網站的根目錄**而不是 `/forum/` 之下：那棵子樹有登入閘門，而訂閱來源必須匿名可達。公開頁的 `<head>` 都帶 `<link rel="alternate" type="application/rss+xml">`，閱讀器在任何一頁都能發現它。
- 最新 50 則，**依發表時間新到舊**（與動態的「置頂優先」不同：摘要要回答的是「最近發生什麼」）。
- 每一則有標題、連結（即永久連結）、全文、發佈時間與 GUID。**不含作者、不含圖片**：作者 email 不離開後端，而附圖需要分鐘級的 media token —— RSS 是存起來慢慢讀的格式，簽一個很快失效的網址只會得到破圖。
- 標題由內文衍生：折疊空白後取開頭 60 個字（更長時補 `…`）。**沒有貼文時仍是一份合法的空 feed**，不是 404。
- 回應帶 `ETag`（整份文件的 SHA-256）與 `Cache-Control: public, max-age=300`；內容沒變時回 304，因此閱讀器的輪詢幾乎不傳輸內容。`lastBuildDate` 由**最新一則貼文的時間**推導而非產生回應的時刻 —— 文件裡只要有一個會自己走動的欄位，雜湊就會每秒不同，304 永遠不會發生（而那個失效沒有任何錯誤訊息）。沒有貼文時整份 feed 省略 `lastBuildDate`。
- 只列未刪除的貼文（與所有公開端點同一條規則）；還原之後它會回到摘要裡。

---

## 管理

每個 handler 內部自行呼叫 `requireAdminForum`（**在 method 分派之前**，由 `adminauth` analyzer 強制），**刻意沒有限流**。

| Method | Path |
| --- | --- |
| GET / POST | `/api/admin/forum/posts` （GET 可帶 `deleted=1`：只列已刪除，依刪除時間新到舊） |
| GET / PUT / DELETE | `/api/admin/forum/posts/{id}` （DELETE 為軟刪除） |
| POST | `/api/admin/forum/posts/{id}/pin` （body `{"pinned":true\|false}`） |
| POST | `/api/admin/forum/posts/{id}/restore` （還原軟刪除；冪等，回 `{"ok":true,"restored":…}`） |
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

## 貼文的軟刪除（soft delete）

貼文刪除（使用者的 `DELETE /api/forum/posts/{id}` 與後臺的 `DELETE /api/admin/forum/posts/{id}`）不會把資料列從 `forum_posts` 移除，而是把 `deleted_at` 寫上當下時間。

| 問題 | 答案 |
| --- | --- |
| 刪掉之後還看得到嗎？ | 看不到。所有公開的讀取路徑（動態、追蹤動態、收藏、個人頁、永久連結、**留言列表**、搜尋、RSS、後臺列表、統計、CSV 匯出）都帶 `deleted_at IS NULL`。唯一的例外是後臺的 `?deleted=1` 清單，它要的正是那一批。 |
| 重複刪除同一篇？ | 回 404（與「這篇不存在」同一種回應，不區分以避免探測作者權限）。 |
| 留言、按讚、檢舉呢？ | 資料都還在表裡，但**讀不到也不能再互動**：`GET /comments` 回 404，編輯／刪除自己的留言、按讚、收藏、留言、檢舉、置頂同樣一律 404。 |
| 對一篇已刪除的貼文還能做什麼？ | 只有管理員能在後臺還原它（`POST /api/admin/forum/posts/{id}/restore`）。除此之外什麼都不能：它對外等同不存在，因此所有子資源端點都以母文章的可見性把關，而不是只看子資源自己還在不在。 |
| 搜尋引擎呢？ | 刪除時移除 ES 文件（`unindexForumPost`）；全量重建也跳過已刪除的貼文。若移除失敗而留下幽靈文件，搜尋端會再用 MySQL 的可見性把它從結果與筆數中挑掉（必要時向後補齊該頁），因此回應宣稱的筆數不會多於畫面上看得到的。刪除是 best-effort，還原則會重新索引。 |
| 可以救回來嗎？ | 可以。後臺的「只顯示已刪除」清單（`?deleted=1`）上有「還原」：`POST /api/admin/forum/posts/{id}/restore` 把 `deleted_at` 設回 `NULL` 並補一筆 `post.restore` 稽核。已經不是刪除狀態時回 200 但 `restored:false`（冪等，不記稽核）。剩下的限制見 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md#軟刪除的貼文會在保留期後被清理)。 |
| 資料什麼時候真正消失？ | 超過 `POST_RETENTION_DAYS`（預設 90 天）之後會被**硬刪**：貼文本身，加上它的留言、按讚、收藏與指向它的檢舉工單，在同一個交易裡一起消失（`backend/forum/purge`，每小時跑一次、每個 tick 最多清 2,000 列）。那篇貼文的附圖在「不再被任何貼文或頭像引用」時會被回收。**保留期是硬期限**：到期後就不可逆了，`/restore` 對已硬刪的列回 404。詳見 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md#軟刪除的貼文會在保留期後被清理)。 |

---

## 留言的層級（回覆）

留言可以是另一則留言的回覆，**最多三層**：貼文 → 留言（第 1 層）→ 回覆（第 2 層）→ 回覆的回覆（第 3 層）。第 3 層不能再被回覆，後端回 400（`回覆已達最深層級`）。

| 問題 | 答案 |
| --- | --- |
| 怎麼發出一則回覆？ | `POST /api/forum/posts/{id}/comments` 的 body 帶 `parentId`（所回覆的留言 id）。省略或 `0` 代表第一層留言。 |
| 回覆的對象有條件嗎？ | 必須是**同一篇貼文下**的留言，否則 404（`parent comment not found`）。這讓「回覆了卻在自己的貼文頁面上看不到」不可能發生。 |
| 列表端有什麼不同？ | `GET /api/forum/posts/{id}/comments` 仍然回**扁平**的一串（沿用 `items`／`hasMore` 分頁），每則多帶 `parentId`（第一層省略）與 `depth`（1..3，一定存在）。組樹由前端負責。 |
| 刪除一則留言會怎樣？ | 連同它底下的所有回覆一起刪除（使用者與後台的路徑相同）。確認框說的就是這件事。回應帶 `removed`（實際刪掉的筆數），前端的留言數減的是它而不是 1。 |
| 為什麼刪除要連坐？ | 父留言消失而回覆留下，會出現沒有父留言的孤兒 —— 它會在樹裡漂到第一層，看起來像一句沒有上下文的話。 |
| 後臺能指定 `parentId` 嗎？ | 不能。後台新增的留言一律是第一層（`POST /api/admin/forum/comments`），但刪除同樣連坐，列表也回 `parentId`／`depth`。 |
| 留言數怎麼算？ | 回覆計入貼文的 `commentCount`（它是 `forum_post_comments` 的 COUNT，沒有區分層級）。 |
