# 資料庫

MySQL，`utf8mb4` / InnoDB。**Schema 在啟動時自動建立**：`data.MigrateMySQL` 是一連串 idempotent 步驟（`CREATE TABLE IF NOT EXISTS`、`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`、`INSERT IGNORE`），所以空資料庫直接啟動即可，不需手動匯入 SQL。沒有版本化的 migration，也沒有降級路徑。

- [資料表](#資料表)
- [沒有 session 表](#沒有-session-表)
- [稽核的三個性質](#稽核的三個性質)

---

## 資料表

| 資料表 | 用途 | 關鍵結構 |
| --- | --- | --- |
| `forum_posts` | 貼文 | `image_url` 只存**檔名**，讀取時才用 `FILES_SERVER_PUBLIC_URL` 組網址；`updated_at` 可空＝從未被作者編輯過 |
| `forum_post_comments` | 留言 | 無外鍵，完整性靠應用層；`updated_at` 語意同上 |
| `forum_post_likes` | 按讚 | 複合 PK `(post_id, author_email)`，天然去重 |
| `forum_reports` | 檢舉 | `target_type`（`post`/`comment`）+ `target_id` 的多型參照，無外鍵 |
| `forum_profiles` | 公開資料 | `public_key` = `SHA2(author_email, 256)`；`nickname` 唯一；`avatar_url` 只存**檔名**（讀取時組網址，語意指 `forum_posts.image_url`） |
| `forum_users` | 帳號狀態 | `status` 為 `ACTIVE` / `SUSPENDED`。**沒有 role 欄位** |
| `forum_user_tags` | 標籤字典 | `name` 唯一 |
| `forum_user_tag_assignments` | 標籤指派 | 複合 PK `(user_email, tag_id)` |
| `forum_follows` | 追蹤關係 | 複合 PK `(follower_email, target_email)` |
| `forum_request_metrics` | 分鐘級請求統計 | PK `(bucket_minute)`；純衍生資料，可整表刪除 |
| `forum_admin_actions` | 管理員操作稽核 | `changes` 存欄位級 diff JSON；只能由時間清理 |
| `forum_announcements` | 站內公告 | 同時只有一列 `active`；`expires_at` 可空＝永不自動過期 |

---

## 沒有 session 表

Session 完全存在 Redis 的 `FORUM:session:<token>` hash 裡（欄位 `email` + `is_admin`）。這一點是整個安全模型的關鍵 —— cookie 只是查詢鍵，能讀 Redis 就等於能登入任何人 —— 見 [`SECURITY.md`](SECURITY.md)。Session 模型的設計理由與「為什麼這樣設計」見 [`ARCHITECTURE.md`](ARCHITECTURE.md#session-模型)。

---

## 稽核的三個性質

設計理由（「為什麼這樣設計」與代價）見 [`ARCHITECTURE.md`](ARCHITECTURE.md#管理員操作稽核)。

1. **稽核與操作在同一個交易裡。** `audit.Record` 回傳錯誤，而每個後台寫入 handler 都在 `Commit()` **之前**檢查它 —— 因此「操作發生但沒有紀錄」在資料庫層不可能發生。這件事很容易被無聲破壞：Go 的 `database/sql` **不會**因為交易內某個語句失敗就中止交易，所以呼叫了 `recordAdminAction` 卻不看回傳值，等於把稽核降級成「盡力記錄」。由 `auditcheck` analyzer 在編譯期強制（見 [`DEVELOPMENT.md`](DEVELOPMENT.md#不變條件-analyzer)）。
2. **欄位級 diff 存在 `changes`（JSON），每個值截到 200 bytes。** 完整內容仍在 `forum_posts` / `forum_post_comments`，稽核表不重複一份可能含有個資的長文字。因此稽核紀錄不能回答「被改掉的那段原文是什麼」，只能回答「原本的前 200 bytes」。
3. **沒有任何刪除紀錄的 API。** 唯一清理途徑是時間式的 `audit.Pruner`（保留 `AUDIT_RETENTION_DAYS` 天，預設 90 天，每小時清一次）。一個能刪除自己紀錄的稽核日誌等於沒有稽核日誌。
