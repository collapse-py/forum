# 資料庫

MySQL，`utf8mb4` / InnoDB。**Schema 在啟動時自動建立**：`data.MigrateMySQL` 是一連串 idempotent 步驟（`CREATE TABLE IF NOT EXISTS`、`INSERT IGNORE`，以及補欄位／補索引的 `information_schema` 探測 —— MySQL 沒有 `ADD COLUMN IF NOT EXISTS`，那是 MariaDB 專屬語法），所以空資料庫直接啟動即可，不需手動匯入 SQL。沒有版本化的 migration，也沒有降級路徑。

- [資料表](#資料表)
- [主鍵是雪花 ID](#主鍵是雪花-id)
- [沒有 session 表](#沒有-session-表)
- [稽核的三個性質](#稽核的三個性質)

---

## 資料表

| 資料表 | 用途 | 關鍵結構 |
| --- | --- | --- |
| `forum_posts` | 貼文 | `image_url` 只存**檔名**，讀取時才用 `FILES_SERVER_PUBLIC_URL` 組網址；`updated_at` 可空＝從未被作者編輯過；`deleted_at` 可空＝未被刪除（軟刪除，見 [`API.md`](API.md#貼文的軟刪除soft-delete)） |
| `forum_post_comments` | 留言 | 無外鍵，完整性靠應用層；`updated_at` 語意同上；`parent_id` 指回應的另一則留言（`0`＝第一層）、`depth` 是層級 1..3，見 [`API.md`](API.md#留言的層級回覆) |
| `forum_post_likes` | 按讚 | 複合 PK `(post_id, author_email)`，天然去重 |
| `forum_post_bookmarks` | 收藏 | 複合 PK `(post_id, author_email)`，天然去重；`created_at` 是收藏時間（目前不用於排序）；收藏數**不公開**，見 [`API.md`](API.md#收藏) |
| `forum_media` | 上傳檔案的所有權 | 主鍵是**檔名**（檔案伺服器以此為鍵）；`owner_email` 是上傳者，刪圖端點靠它回答「這是不是我的」。只記錄這個功能上線**之後**上傳的檔案 —— 更早的檔案沒有列，見 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md#這次改動之前留下的孤兒檔無法清除) |
| `forum_media_deletions` | 待刪除的圖片檔名佇列 | 主鍵 `filename`；與貼文硬刪**同一個交易**寫入，`purge.Prune` 成功刪檔或確認仍被引用後才移除。詳見 [`ARCHITECTURE.md`](ARCHITECTURE.md#清理機制) |
| `forum_reports` | 檢舉 | `target_type`（`post`/`comment`）+ `target_id` 的多型參照，無外鍵 |
| `forum_profiles` | 公開資料 | **兩把**對外識別碼：`public_key_hmac` = `HMAC-SHA256(AUTHOR_KEY_SECRET, email)`（第 39 步，有索引；未設秘密時恆為空字串）、`public_key` = `SHA2(author_email, 256)`（第 16 步，有索引，舊形態）；值都在應用層算（`httpapi.DeriveAuthorKey`），不在 SQL 裡算 —— 秘密會進 MySQL 的 general log 與日誌。`nickname` 唯一；`avatar_url` 只存**檔名**（讀取時組網址，語意指 `forum_posts.image_url`） |
| `forum_users` | 帳號狀態 | `status` 為 `ACTIVE` / `SUSPENDED`。**沒有 role 欄位** |
| `forum_user_tags` | 標籤字典 | `name` 唯一 |
| `forum_user_tag_assignments` | 標籤指派 | 複合 PK `(user_email, tag_id)` |
| `forum_follows` | 追蹤關係 | 複合 PK `(follower_email, target_email)` |
| `forum_notifications` | 站內通知 | `recipient_email` / `actor_email` 皆為 email（無外鍵，應用層維護）；`kind` 為 `comment` / `reply` / `follow`；`read_at` 可空＝未讀；`post_id` / `comment_id` 為 0 ＝該事件沒有這個目標 |
| `forum_daily_stats` | 日別新增量彙總 | PK `(day)`；純衍生資料。由 `data.RefreshDailyStats` 每 10 分鐘重算最近三天，啟動時 `BackfillDailyStats` 補缺漏日期；記的是「那天產生了多少」的歷史量，軟刪除與清理都不改變它 |
| `forum_request_metrics` | 分鐘級請求統計 | PK `(bucket_minute)`；純衍生資料，可整表刪除 |
| `forum_admin_actions` | 管理員操作稽核 | `changes` 存欄位級 diff JSON；只能由時間清理 |
| `forum_announcements` | 站內公告 | 同時只有一列 `active`；`expires_at` 可空＝永不自動過期 |

---

## 主鍵是雪花 ID

有 `id` 主鍵的七張表（`forum_posts`、`forum_post_comments`、`forum_reports`、`forum_user_tags`、`forum_announcements`、`forum_notifications`、`forum_admin_actions`）由**應用層**產生 id，資料庫沒有 `AUTO_INCREMENT`。寫入時必須把 id 放進 INSERT 的第一個欄位；漏掉的症狀是 MySQL 以「Field 'id' doesn't have a default value」拒絕，而不是安靜地寫入 0。

實作在 `forum/idgen`，位元配置刻意壓在 **53 位元**：

```
41 位元毫秒時間戳（自 2024-01-01T00:00:00Z）
 6 位元節點編號（0–63）
 6 位元同毫秒序號（0–63）
```

壓在 53 位元內是因為 id 以 JSON **數字**送給瀏覽器，而 JavaScript 的 `number` 只能精確表示到 `2^53-1`。標準 64 位元雪花在 2026 年就超過這個範圍，症狀會是「某些 id 點進去是 404」—— 只有部分資料壞掉，因此很難聯想到精度。代價是每毫秒最多 64 個 id、節點上限 64 台；這個站是論壇，遠低於那個量級。

- 同一行程內 id 嚴格遞增，時鐘回退時沿用上一個毫秒，因此「依 id 排序」等於「依建立順序排序」。
- 多實例部署時每個實例必須設定不同的 [`SNOWFLAKE_NODE_ID`](CONFIGURATION.md)，否則同一毫秒可能發出相同 id。單一實例維持 `0`。
- **既有資料會被重新編號**：`MigrateMySQL` 的第 38 步在偵測到 `AUTO_INCREMENT` 時，把七張表的既有列依舊 id 遞增換成雪花 id，並同步所有引用欄位（留言的 `post_id`／`parent_id`、按讚／收藏、檢舉的 `target_id`、通知的 `post_id`／`comment_id`、標籤指派、以及稽核紀錄的 `target_id`）。每張表一個交易、整個步驟在 MySQL named lock 下序列化；可重複執行。
- 重新編號的已知成本：舊的**分享連結與書籤**（`/forum/post/<id>`）會失效；ES 索引由啟動時的全量重建換成新 id（舊文件由定期對帳清除）；稽核 `changes` 裡歷史 diff 的文字值（例如 `postId`）不會改寫。

---

## 沒有 session 表

Session 完全存在 Redis 的 `FORUM:session:<token>` hash 裡（欄位 `email` + `is_admin`）。這一點是整個安全模型的關鍵 —— cookie 只是查詢鍵，能讀 Redis 就等於能登入任何人 —— 見 [`SECURITY.md`](SECURITY.md)。Session 模型的設計理由與「為什麼這樣設計」見 [`ARCHITECTURE.md`](ARCHITECTURE.md#session-模型)。

---

## 作者識別碼的回填（第 39 步）

第 39 步只**加** `public_key_hmac` 欄位與索引，值不在 SQL 裡算 —— 派生靠部署秘密，
而 SQL 文字會進 MySQL 的 general log 與日誌。因此啟動時由 `data.BackfillProfileAuthorKeys`
在應用層逐批（每批 500 列）補齊 `public_key_hmac = ''` 的列，派生的唯一來源是
`httpapi.DeriveAuthorKey`（`main.go` 以它組出 `keyFor` 閉包）。

- **未設 `AUTHOR_KEY_SECRET` 時這一步不做任何事**：那種部署的對外識別碼本來就是第 16
  步的 SHA-256，新欄位保持空字串是正確狀態。
- **冪等**：只動空字串的列。已回填的列不重算，所以**換秘密不會被這一步靜默覆蓋** ——
  換秘密時既有對外 key 失效是部署者明示的代價，舊連結仍由 `public_key` 欄位解析。
- 解析（`/api/forum/public-profile`、`/api/forum/public-posts`、追蹤對象）用
  `WHERE public_key_hmac = ? OR public_key = ?`，兩欄都有索引；新形態排在前面是
  刻意讓設了秘密後的請求走第一次命中。

設計與代價見 [`SECURITY.md`](SECURITY.md#匿名性的邊界)。

---

## 稽核的三個性質

設計理由（「為什麼這樣設計」與代價）見 [`ARCHITECTURE.md`](ARCHITECTURE.md#管理員操作稽核)。

1. **稽核與操作在同一個交易裡。** `audit.Record` 回傳錯誤，而每個後台寫入 handler 都在 `Commit()` **之前**檢查它 —— 因此「操作發生但沒有紀錄」在資料庫層不可能發生。這件事很容易被無聲破壞：Go 的 `database/sql` **不會**因為交易內某個語句失敗就中止交易，所以呼叫了 `recordAdminAction` 卻不看回傳值，等於把稽核降級成「盡力記錄」。由 `auditcheck` analyzer 在編譯期強制（見 [`DEVELOPMENT.md`](DEVELOPMENT.md#不變條件-analyzer)）。
2. **欄位級 diff 存在 `changes`（JSON），每個值截到 200 bytes。** 完整內容仍在 `forum_posts` / `forum_post_comments`，稽核表不重複一份可能含有個資的長文字。因此稽核紀錄不能回答「被改掉的那段原文是什麼」，只能回答「原本的前 200 bytes」。
3. **沒有任何刪除紀錄的 API。** 唯一清理途徑是時間式的 `audit.Pruner`（保留 `AUDIT_RETENTION_DAYS` 天，預設 90 天，每小時清一次）。一個能刪除自己紀錄的稽核日誌等於沒有稽核日誌。
