# API 參考

回應格式一律 JSON，錯誤以非 2xx 狀態碼回傳。各端點的語意細節與取捨見 [`ARCHITECTURE.md`](ARCHITECTURE.md)。

- [基礎](#基礎)
- [論壇](#論壇)
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

讀取匿名可達；寫入需要登入並受限流。

| Method | Path | 說明 |
| --- | --- | --- |
| GET | `/api/forum/posts?offset=&limit=` | 貼文列表 |
| POST | `/api/forum/posts` | 建立貼文 |
| GET | `/api/forum/posts/{id}` | 單篇貼文（永久連結頁；匿名可讀，找不到回 404） |
| PUT | `/api/forum/posts/{id}` | 編輯自己的貼文本文（**只有 `content`**，見 [已知問題](KNOWN_ISSUES.md)） |
| DELETE | `/api/forum/posts/{id}` | 軟刪除自己的貼文（`deleted_at` 寫入時間，資料保留） |
| GET | `/api/forum/posts/{id}/comments` | 留言列表（公開） |
| POST | `/api/forum/posts/{id}/comments` | 建立留言 |
| PUT / DELETE | `/api/forum/posts/{id}/comments/{cid}` | 編輯／刪除自己的留言 |
| POST | `/api/forum/posts/{id}/like` | 按讚 |
| POST | `/api/forum/posts/{id}/report` | 檢舉貼文 |
| POST | `/api/forum/posts/{id}/comments/{cid}/report` | 檢舉留言 |
| POST | `/api/forum/images` | 上傳圖片，轉送 files_server |
| POST | `/api/forum/image-tokens/release` | 釋放媒體存取權杖 |
| GET / PUT | `/api/forum/profile` | 自己的資料（暱稱、簡介、頭像；PUT 為整份覆寫） |
| GET / POST | `/api/forum/follows` | 追蹤名單 / 切換追蹤 |
| GET | `/api/forum/following/posts` | 私密動態牆（需登入，不限流） |
| GET | `/api/forum/my-posts` | 自己的貼文（需登入；未發過文回空清單而非 404） |
| GET | `/api/forum/public-profile?key=` | 以 `public_key` 查公開資料（無需登入） |
| GET | `/api/forum/public-posts?user=` | 以 `public_key` 查貼文（無需登入） |
| GET | `/api/forum/search?q=&offset=&limit=` | 全文搜尋；ES 不可用時退回 `LIKE` |

> 兩個 `public-*` 端點的參數名**不一致**（`key` vs `user`），這是現況不是筆誤。

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
- 被替換掉的舊檔沒有刪除端點，會成為孤兒檔（與貼文附圖相同，見 `forumProfileRequest` 的說明）。

---

## 管理

每個 handler 內部自行呼叫 `requireAdminForum`（**在 method 分派之前**，由 `adminauth` analyzer 強制），**刻意沒有限流**。

| Method | Path |
| --- | --- |
| GET / POST | `/api/admin/forum/posts` |
| GET / PUT / DELETE | `/api/admin/forum/posts/{id}` （DELETE 為軟刪除） |
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

## 貼文的軟刪除（soft delete）

貼文刪除（使用者的 `DELETE /api/forum/posts/{id}` 與後臺的 `DELETE /api/admin/forum/posts/{id}`）不會把資料列從 `forum_posts` 移除，而是把 `deleted_at` 寫上當下時間。

| 問題 | 答案 |
| --- | --- |
| 刪掉之後還看得到嗎？ | 看不到。所有讀取路徑（動態、追蹤動態、個人頁、永久連結、**留言列表**、搜尋、後臺列表、統計、CSV 匯出）都帶 `deleted_at IS NULL`。 |
| 重複刪除同一篇？ | 回 404（與「這篇不存在」同一種回應，不區分以避免探測作者權限）。 |
| 留言、按讚、檢舉呢？ | 資料都還在表裡，但**讀不到也不能再互動**：`GET /comments` 回 404，編輯／刪除自己的留言、按讚、留言、檢舉、置頂同樣一律 404。 |
| 對一篇已刪除的貼文還能做什麼？ | 什麼都不能。它對外等同不存在，因此所有子資源端點都以母文章的可見性把關，而不是只看子資源自己還在不在。 |
| 搜尋引擎呢？ | 刪除時移除 ES 文件（`unindexForumPost`）；全量重建也跳過已刪除的貼文。若移除失敗而留下幽靈文件，搜尋端會再用 MySQL 的可見性把它從結果與筆數中挑掉（必要時向後補齊該頁），因此回應宣稱的筆數不會多於畫面上看得到的。 |
| 可以救回來嗎？ | 目前沒有恢復入口（API 或介面都沒有）。要救回只能對 `forum_posts` 下 `UPDATE ... SET deleted_at = NULL`，見 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)。 |
| 資料什麼時候真正消失？ | 目前不會。軟刪除把「回收」從 handler 移出去，需要另外設計清理機制（見 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)）。 |
