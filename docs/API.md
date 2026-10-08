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

---

## 管理

每個 handler 內部自行呼叫 `requireAdminForum`（**在 method 分派之前**，由 `adminauth` analyzer 強制），**刻意沒有限流**。

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
