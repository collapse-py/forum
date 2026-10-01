# 程式碼審查報告

- **審查對象**：`backend/`（Go 1.25 模組 `forum`）、`frontend/`（Vite + React 19 + TS）
- **範圍**：目前工作區未提交的變更（40 個修改檔 + 8 個新套件 / 24 個新前端頁面與端點）
  ，涵蓋 audit / metrics / ipban / session_list 四個新 Go 套件，以及 monitor、stats、
  audit-log、CSV 匯出、批次操作、session 管理、IP 封鎖、站內公告、貼文置頂等
  新功能。
- **驗證基準**（本次審查實際執行）：

  | 指令 | 結果 |
  | --- | --- |
  | `go build ./...` | 通過 |
  | `go vet ./...` | 通過 |
  | `go test ./...` | 全部 ok（audit / es / httpapi / ipban / metrics / session） |
  | `npm run typecheck` | 通過（`tsc -p tsconfig.json` + `tsc -p tsconfig.sw.json`） |
  | `go test -race ./...` | **無法執行** — 本機無 C 編譯器（`gcc not found`），資料競爭未被驗證 |

> 審查方式：主線逐檔閱讀 audit / metrics / ipban / session_list 四個新套件，
> 以及 announcement、audit_log、blocklist、monitoring、session_admin、
> server 路由與 mysql 遷移；另有兩條平行審查線分別負責 Go 的
> stats / csv_export / batch_handlers 與新前端頁面。

---

## 摘要

| 嚴重度 | 數量 |
| --- | --- |
| Critical | 0 |
| High | 5 |
| Medium | 14 |
| Low | 7 |

整體品質相當高，值得先說明：這批變更在多數地方把「為什麼這樣做」寫進了程式碼註解，
而且那些註解描述的取捨是真的有被實作出來（稽核與操作同交易、SCAN 帶游標、pipeline
一次撈回、CSV 單一實作點、遞減索引與 `ORDER BY` 對齊）。`go vet` 與 `npm run typecheck`
全綠，18 個語系的 key 完整性由 `Record<MessageKey, string>` 這個型別在編譯期強制
（`npm run typecheck` 通過即代表沒有漏翻譯的 key）。

下列問題都不是風格意見，而是可以指出具體失敗症狀的缺陷。

---

## High

### H0. 使用者 CSV 匯出：50,000 列 × `forum_post_comments` 全表掃描

**檔案**：`backend/forum/httpapi/csv_export.go:179-185`；
`backend/forum/data/mysql.go:178-190`

```go
rows, err := s.db.QueryContext(r.Context(), `
    SELECT u.email, u.status, u.created_at, u.updated_at,
           (SELECT COUNT(*) FROM forum_posts         WHERE author_email = u.email),
           (SELECT COUNT(*) FROM forum_post_comments WHERE author_email = u.email)
    FROM forum_users u
    ORDER BY u.email ASC
    LIMIT `+strconv.Itoa(csvExportMaxRows))    // 50000
```

`forum_post_comments` 的索引只有 `post_id` 與 `created_at`（`mysql.go:185-186`），
**沒有 `author_email`**。因此 MySQL 對每一列輸出都做一次該表的全表掃描。
`csvExportMaxRows` 只能限制**回傳**列數，不能限制相關子查詢**讀取**的列數 ——
這正是 `csv_export.go:344-347` 把「有 ORDER BY + LIMIT 就不是全表掃描」列為設上限
理由時沒有考慮到的情況。

**症狀**：留言累積到數萬筆後，按一次「匯出使用者」會讓 MySQL 執行數萬次全表掃描。
症狀是「匯出按鈕轉圈很久、然後逾時」，沒有任何錯誤訊息。這條路由還刻意不掛限流
（`server.go:558-565`），而全站沒有任何 statement timeout。

（同一個缺索引也讓 `/admin/stats` 的熱門作者排行變成 N+1，見 M0(b)。）

**修法**：新增一個冪等遷移步驟補索引（仿照第 21 步的 `information_schema` 探測）：

```sql
ALTER TABLE forum_post_comments ADD INDEX idx_forum_post_comments_author_email (author_email)
```

並把相關子查詢改成預先聚合的衍生表 join（每個子表只掃一次）：

```sql
LEFT JOIN (SELECT author_email, COUNT(*) c FROM forum_post_comments GROUP BY author_email) cc
       ON cc.author_email = u.email
```

---

### H1. 發佈「不啟用」的公告會靜默下架目前生效的公告

**檔案**：`backend/forum/httpapi/announcement_handlers.go:255-268`

```go
// 先停用舊的再插新的。...
if _, err := tx.ExecContext(r.Context(),
    `UPDATE forum_announcements SET active = 0, updated_at = ?, updated_by = ? WHERE active = 1`,
    now, actor); err != nil { ... }        // ← 無條件停用

result, err := tx.ExecContext(r.Context(), `
    INSERT INTO forum_announcements (body, active, created_at, ...)
    VALUES (?, ?, ?, ?, ?, ?, ?)`,
    body, boolToInt(req.Active), now, actor, now, actor, expires)  // ← 這裡才看 req.Active
```

`announcementRequest.Active` 的文件（`announcement_handlers.go:195-196`）明寫：

> Active 為 false 代表「發佈但不顯示」。存在的理由是一則要寫好、稍後才生效的公告很常見。

前端 `AnnouncementsPage.tsx:222-230` 也確實提供這個 checkbox。但停用舊公告的那句
沒有受 `req.Active` 保護，於是：

**症狀**：管理員在有公告生效時，勾掉「顯示」後按發佈 → 舊公告被停用、新公告以
`active=0` 插入 → 全站橫幅消失，且沒有任何錯誤訊息。這正是該欄位要支援的「先寫好、
稍後才生效」情境，結果變成「順手把線上的公告關掉」。

**修法**：把停用舊公告的條件綁到 `req.Active`。

```go
if req.Active {
    if _, err := tx.ExecContext(r.Context(),
        `UPDATE forum_announcements SET active = 0, updated_at = ?, updated_by = ? WHERE active = 1`,
        now, actor); err != nil { ... }
}
```

（檔頭「這個不變條件由三件事保證」的說明也要一併調整：目前第 1 點在 `active=false`
時不成立。）

---

### H2. 從後臺列表**無法取消置頂**——後端從不回傳 `pinned`

**檔案**：`backend/forum/httpapi/forum_admin_handlers.go:83-92`、`:1018-1022`、`:1039`；
`backend/forum/httpapi/search.go:402-405`；`frontend/src/admin/ForumAdminPage.tsx:269,613,617`

`adminForumPost` 沒有 `Pinned` 欄位：

```go
type adminForumPost struct {
    ID           int64               `json:"id"`
    ...
    CommentCount int                 `json:"commentCount"`
    Comments     []adminForumComment `json:"comments"`
}
```

而 `listAdminForumPosts` 與 `loadAdminForumPostsByIDs`（後臺搜尋）兩條查詢的
SELECT 都沒有 `pinned`：

```go
rows, err := s.db.QueryContext(r.Context(), `
    SELECT id, author_email, content, created_at, image_url,
           (SELECT COUNT(*) FROM forum_post_likes WHERE post_id = forum_posts.id),
           (SELECT COUNT(*) FROM forum_post_comments WHERE post_id = forum_posts.id)
    FROM forum_posts ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, ...)
```

前端卻依賴這個欄位：

```tsx
const togglePin = async (item: AdminPost) => {
  const next = item.pinned !== true;          // 永遠是 true
  ...
  await adminApi(`/api/admin/forum/posts/${item.id}/pin`, { method: 'POST', body: { pinned: next } });
```

**症狀鏈**：
1. 後臺每一列的按鈕永遠顯示「置頂」，永遠不會變成「取消置頂」（`item.pinned` 恆為 `undefined`）。
2. 對一篇已置頂的文章按下去 → `next = true` → 送 `{"pinned": true}` → 後端
   `announcement_handlers.go:456` 判定 `(before == 1) == req.Pinned` 成立 → **直接回
   200 且不寫入、不記稽核**。畫面顯示「置頂成功」但狀態完全沒變。
3. 結果是：置頂只能用一次，之後無法從介面取消，且稽核紀錄裡也不會有 `post.unpin`。

`types.ts:127` 已經宣告了 `pinned?: boolean`，說明前端是照著「後端會回這個欄位」寫的。

**修法**：把 `pinned` 加進 `adminForumPost` 與那兩條 SELECT（以及對應的 `Scan`），
或（若後臺刻意不顯示）移除 `togglePin` 對 `item.pinned` 的依賴。目前兩邊的契約不一致。

---

### H3. 從列表切換公告的啟用狀態會把到期時間清成「永不過期」

**檔案**：`frontend/src/admin/AnnouncementsPage.tsx:152-155`；
`backend/forum/httpapi/announcement_handlers.go:218-223`

```tsx
await adminApi(`/api/admin/announcements/${item.id}`, {
  method: 'PATCH',
  body: { body: item.body, active: next, hoursUntilExpiry: 0 },   // ← 固定送 0
});
```

後端 `normalizeAnnouncement`：

```go
if hours < 0 { return ..., &requestError{message: "有效時間不可為負數"} }
if hours == 0 { return trimmed, sql.NullTime{}, nil }   // ← NULL = 永不過期
```

**症狀**：一則設定了 7 天有效期的公告被停用後，管理員在列表按「重新啟用」→ 它的
`expires_at` 被改成 NULL，變成**永久顯示**。README 對這個 UI 的設計描述是
「已過期（藍，要重新設定時間）」，但按鈕實際做的事是「設成永不過期」，而那個語意
在畫面上完全看不到。

同一個函式的 `startEdit` 已經很小心地把絕對時間換算成剩餘小時（`hoursUntil()`），
但 `setAnnouncementActive` 繞過了這個換算。

**修法**：`setAnnouncementActive` 帶上 `hoursUntil(item.expiresAt) ?? 0`（或明確傳一個
要套用的時長），並讓 `normalizeAnnouncement` 對 `active=true` 且 `hours=0` 的情況
拒絕而不是默默設成永久。

---

### H4. IP 封鎖與稽核紀錄的 IP 都建立在他人可自由偽造的標頭上

**檔案**：`backend/forum/httpapi/ratelimit.go:355-379`、`backend/forum/logger/middleware.go:366-381`；
使用點 `backend/forum/httpapi/blocklist.go:97`、`backend/forum/httpapi/audit_log.go:124`

```go
func clientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        parts := splitComma(xff)
        if len(parts) > 0 { return parts[0] }   // ← 無條件採信最左一項
    }
    if xri := r.Header.Get("X-Real-IP"); xri != "" { return xri }
    ...
}
```

專案自己在 `ratelimit.go:349-353` 已經把這個限制寫得很清楚（限流可被偽造來源繞過）。
但這批變更新增了兩個**把 clientIP 當成可信識別值**的功能，讓這個既有限制的後果
從「限流可以被繞過」升級成兩個更嚴重的問題：

1. **IP 封鎖可被單一標頭繞過。** `withBlocklistHandler` 用 `clientIP(r)` 查封鎖名單。
   被封鎖者只要在每個請求加上 `X-Forwarded-For: 1.2.3.4`，就會取得一份全新的額度，
   封鎖形同不存在。這直接抵消了 README 宣稱的買點「封鎖在部署之後仍然有效」——
   它撐得過重啟，但撐不過一個 header。
2. **稽核紀錄的 `ip` 欄位不可作為稽核依據。** `recordAdminAction` 把它寫進
   `forum_admin_actions.ip`，而 `audit_log.go:183-190` 強調稽核紀錄的全部價值在於它是
   真實的。攻擊者（或被控的管理員 session）可以讓自己操作的來源 IP 顯示成任意值。

`logger/middleware.go:359-361` 的警告有寫，但它只針對「日誌」；這次新增的兩個使用點
把同一個不可信的值提升成了一個安全控制的輸入。

**修法**（擇一或並行）：
- 依 `RemoteAddr` 或設定檔新增的 `TRUSTED_PROXY_CIDRS` 白名單決定要不要採信
  `X-Forwarded-For`；不在白名單就退回 `RemoteAddr`。這是標準做法。
- 短期緩解：`blocklist.go` 改用 `r.RemoteAddr`（或兩者都查：XFF 命中**或** RemoteAddr
  命中就視為封鎖），讓偽造標頭無法解除封鎖。
- 稽核的 `ip` 欄位加上 `ip_source`（`xff` / `peer`）標記，讓讀者知道可信度。

---

## Medium

### M0. schema 缺 `forum_post_comments.author_email` 索引（H0 與 `/admin/stats` N+1 的共同根因）

**檔案**：`backend/forum/data/mysql.go:178-190`（缺索引）、`:634-640`（錯誤的判斷前提）

`forum_post_comments` 目前只有兩個索引：

```sql
INDEX idx_forum_post_comments_post_id (post_id),
INDEX idx_forum_post_comments_created_at (created_at)
```

沒有 `author_email`。這次變更新增了兩條以 `author_email` 為條件的查詢，都因此
退化成全表掃描：**使用者 CSV 匯出**（已列為 H0）與 **熱門作者排行的 N+1**
（`stats_handlers.go:390-396`）：

```go
for i := range items {
    if err := s.db.QueryRowContext(ctx,
        `SELECT COUNT(*) FROM forum_post_comments WHERE author_email = ? AND created_at >= ?`,
        items[i].Email, start).Scan(&items[i].Comments); err != nil {
```

10 次額外往返，每次都是「掃 90 天的 `created_at` 區間再逐列過濾 email」——
同一段區間被重掃 10 次。

這與該檔案檔頭自己宣稱的設計原則（`stats_handlers.go:39-43`「相關子查詢…行為不隨
資料量變糟」）不一致——那個推理對 `post_id` 有索引的子查詢成立，對 `author_email`
不成立。

**修法**：在 `MigrateMySQL` 新增一步，仿照第 21 步的 `information_schema` 探測寫法：

```sql
ALTER TABLE forum_post_comments ADD INDEX idx_forum_post_comments_author_email (author_email)
```

並把熱門作者的那段合併成單一查詢（兩個 `GROUP BY author_email` 的衍生表各掃一次
後 join），把 10 次往返降到 1 次。

---

### M1. `loadStatsTotals` 每次載入 `/admin/stats` 都全表掃描 `forum_post_likes`

**檔案**：`backend/forum/httpapi/stats_handlers.go:273-279`；`backend/forum/data/mysql.go:161-171`

```go
(SELECT COUNT(*) FROM forum_post_likes WHERE created_at >= ?)
```

`forum_post_likes` 的主鍵是 `(post_id, author_email)`，另有 `idx_forum_post_likes_post_id`，
**`created_at` 上沒有索引**。這是整支端點裡唯一一個沒有可用索引的子查詢，
而成本隨「全站累積按讚數」線性成長（不是「最近 N 天」，因為沒有索引可用）。

**這個查詢是本次變更新增的**，而遷移第 24 步的註解（`mysql.go:638-640`）明確寫著
「我沒有為 `forum_post_likes.created_at` 加索引（那條查詢根本沒用到它）」——
前提已被這支 handler 推翻，但註解沒有跟著更新，下一個讀到它的人會得到錯誤結論。

**修法**：新增遷移步驟加 `idx_forum_post_likes_created_at (created_at)`，並修正第 24 步
的註解說明（改為「`/admin/stats` 的視窗內按讚合計需要它」）。

---

### M2. `handleAdminBatchTags` 的 `tagIds` 沒有上限，單一交易可產生近 160 萬次 INSERT

**檔案**：`backend/forum/httpapi/batch_handlers.go:61`、`:184`、`:229-235`、`:257-275`

`emails` 有 `batchMaxEmails = 200` 的防禦性上限（`:101-103`），但 `tagIds` 只受
16 KiB body 上限（`:170`）約束。`[1,2,3,…]` 每個 id 兩個位元組 → **約 8,000 個 id**
可以塞進一個請求，而 `validateBatchTagIDs` 只檢查正數與重複，**沒有長度檢查**
（`:257-275`）。

接著寫入迴圈是雙層巢狀：

```go
for _, email := range emails {                 // 最多 200
    ...
    for _, id := range tagIDs {                // 最多約 8000
        tx.ExecContext(r.Context(),
            `INSERT INTO forum_user_tag_assignments (user_email, tag_id) VALUES (?, ?)`, email, id)
```

最壞情況是 **200 × 8000 = 160 萬次單列 INSERT，全部在同一個交易裡**，
所有 row lock 持有到 `Commit`，且這條路由刻意不掛限流（`server.go:566-570`）。
另外它還會組出一個約 16 KB 的 `IN (?,?,…)` 子句（`:277-283`）。

**註解自己也暴露了這個缺口**：`:276` 寫「一次查出所有名稱。逐個查的話是 N 次往返，
而 N ≤ 200」——那個 `N` 指的是 `tagIds`，但 `200` 是 `batchMaxEmails` 的值，
兩者無關。`:55-60` 的防禦性上限論證（「一個惡意請求可以送 10 萬個 email 讓交易撐爆
記憶體」）對 `tagIds` 同樣成立，只是沒有套用。

**修法**：

```go
const batchMaxTags = 20   // 與 audit.maxChangesPerEntry 同量級

// validateBatchTagIDs 開頭（在重複檢查之前）
if len(raw) > batchMaxTags {
    return nil, nil, errBatchTooLarge
}
```

並把寫入改成單一多列 `INSERT ... VALUES (?,?),(?,?)…`，一次交易內的往返數從
O(emails × tags) 降到 O(1)。

---

### M3. `handleAdminBatchTags` 對「沒有變化」的寫入也記稽核，與同專案的處理不一致

**檔案**：`backend/forum/httpapi/batch_handlers.go:218-242`（對照 `:382-388`）

姊妹端點 `handleAdminBatchStatus` 有明確的守衛：

```go
if before == status {
    // 已經是目標狀態。不記稽核 —— 稽核紀錄記的是「發生了什麼改變」
    result.Counts["unchanged"]++
    continue
}
```

`handleAdminBatchTags` 沒有對應的判斷：無論標籤集是否相同，都會 `DELETE` 全部綁定、
重新 `INSERT`、並寫下一筆 `user.tags.update` 稽核，`Before` 與 `After` 內容相同
（`"a, b"` → `"a, b"`）。

**症狀**：管理員對同一批使用者重複套用相同標籤，稽核紀錄就會累積 N 筆宣稱
「標籤被改過」但前後相同的紀錄。這正好污染稽核紀錄存在的理由
（「這個帳號的標籤被動過幾次」），而且 `audit_log.go:167` 的 `onlyChanged()` 就是
為此而存在，這裡卻沒用。

觸發條件不需要任何花招：MySQL 預設的 `utf8mb4` 排序規則**大小寫不敏感**，
所以 `A@B.com` 與 `a@b.com` 在資料庫裡是同一列，卻是去重後 `emails` slice 裡的兩筆。

**修法**：

```go
before := strings.Join(beforeNames, ", ")
if before == after {
    result.Counts["unchanged"]++   // 記得在 :205 的 map 加上 "unchanged": 0
    continue
}
```

---

### M4. `COALESCE(reviewed_at, '')` 混合 DATETIME 與字串，檢舉匯出可能整份回 500

**檔案**：`backend/forum/httpapi/csv_export.go:299`、`:317-318`

```go
SELECT id, target_type, target_id, reporter_email, reason, status,
       created_at, COALESCE(reviewed_at, ''), reviewed_by
FROM forum_reports ORDER BY id DESC LIMIT 50000
...
rows.Scan(&item.id, ..., &item.created, &item.reviewedAt, &item.reviewedBy)
```

`reviewed_at` 是 `DATETIME NULL`（`mysql.go:211`），`''` 是字串常值，`COALESCE`
必須對兩個型別做聚合。DSN 帶 `parseTime=True`，所以若 MySQL 把結果解析成
`MYSQL_TYPE_DATETIME`，driver 會走日期解析分支並對空字串 `''` 呼叫
`parseDateTime` 而失敗。因為 `database/sql` 只 prepare 一次、欄位型別對整個
result set 固定下來，**只要有一列 `PENDING` 檢舉就會讓整份 `reports.csv` 回 500**。

而每一筆新檢舉都是從 `PENDING` 開始的（`mysql.go:206` 的欄位預設），所以這不會是
邊緣情況。

**誠實說明**：本次審查環境沒有可用的 MySQL，無法實際確認 MySQL 8.0 對
「DATETIME + 字串常值」的型別聚合結果，因此這個**後果**是條件性的。但無論 MySQL
怎麼解析，這種寫法對一個**經常性為 NULL** 的欄位來說都很脆弱，而且修法成本很低。

對照組不能證明安全：`user_admin_handlers.go:378` 的 `COALESCE(fp.nickname, '')`
之所以沒事，是因為 `nickname` 本來就是 `VARCHAR`。

**修法**：不要在 SQL 裡混型別，把 NULL 處理搬到 Go：

```go
var reviewedAt sql.NullTime
... SELECT ..., reviewed_at, reviewed_by ...
rows.Scan(&item.id, ..., &reviewedAt, &item.reviewedBy)
item.reviewedAt = ""
if reviewedAt.Valid {
    item.reviewedAt = reviewedAt.Time.Format("2006-01-02 15:04:05")
}
```

順帶好處：時間格式會與另外兩份匯出一致（目前那兩份是靠 `database/sql` 的
`time.Time → *string` 路徑拿到 RFC3339，而這裡會是 SQL 字串格式，
三份匯出同一個欄位的格式不一致本身就是對帳時的混淆來源）。

---

### M5. `probeDatabase` 在 goroutine 裡無防護地解參考 `s.db`

**檔案**：`backend/forum/httpapi/monitoring.go:240-249`

```go
func (s *Server) probeDatabase(parent context.Context) dependencyStatus {
	ctx, cancel := context.WithTimeout(parent, dependencyProbeTimeout)
	defer cancel()
	start := time.Now()
	status := dependencyStatus{State: "ok"}
	if err := s.db.PingContext(ctx); err != nil {     // ← 沒有 nil 檢查
```

這個 handler 裡的每���個依賴都有防護：`s.metrics == nil`（`monitoring.go:171`）、
`s.mediaRedis == nil`（`:272`）、`s.es` 走 `esEnabled()`（`:312`），
`metricsMiddleware` 與 `limitStats` 也都明確 nil-safe。只有這一個沒有。

差異在於**崩潰的後果**：`probeDependencies` 在 `monitoring.go:219-229` 用
`go func()` 啟動這三個探測，而 goroutine 裡的 panic **不會**被 `net/http` 的
per-connection `recover` 接住（那個只涵蓋 handler 本身的 goroutine），
所以 `s.db` 為 nil 時會讓整個行程崩潰，而不是回 500。

以目前的 `main.go` 而言不可達（`OpenMySQL` 失敗時會直接 `Fatal`），但它與三個
姊妹探針的寫法不一致，未來任何「監控要能在資料庫沒接上時仍然開得起來」的改動都會
踩到。

**修法**：在函式開頭加上

```go
if s.db == nil {
    return dependencyStatus{State: "disabled", Detail: map[string]any{"engine": "none"}}
}
```

---

### M6. 批次標籤的存在性檢查在交易之外，有產生孤兒綁定的 TOCTOU

**檔案**：`backend/forum/httpapi/batch_handlers.go:184`（對照 `:198`）

```go
tagIDs, tagNameByID, err := validateBatchTagIDs(r, s.db, req.TagIDs)   // s.db，無交易
...
tx, err := s.beginAdminTx(r)                                          // 交易在這裡才開
```

`validateBatchTagIDs` 收的是 `*sql.DB`，所以「標籤存在」的檢查發生在交易開啟**之前**。
在兩者之間，`handleAdminTag` 的 `DELETE`（`user_admin_handlers.go:231-275`）可以
把該標籤連同它的綁定刪掉；而 `forum_user_tag_assignments` **沒有外鍵**
（`mysql.go:346-355`），所以後續的 `INSERT` 會成功，留下指向不存在標籤的孤兒綁定
—— 正是 `user_admin_handlers.go:616` 的註解說明這個驗證要防止的那件事。

**修法**：把驗證移進交易。`loadAdminUserTagNames`（`user_admin_handlers.go:748`）
已經示範了正確作法——它收的是 `queryer` 介面，`*sql.DB` 與 `*sql.Tx` 都滿足。
把 `validateBatchTagIDs` 的參數型別從 `*sql.DB` 改成 `queryer`，
呼叫端改傳 `tx`，並把這行移到 `beginAdminTx` 之後。

---

### M7. CSV 寫出錯誤被完全丟棄，被截斷的下載檔與完整檔無法區分

**檔案**：`backend/forum/httpapi/csv_export.go:143`、`:217`、`:278`、`:336`

```go
_ = writer.Write(sanitized)      // :143
...
writer.Flush()                   // :217, :278, :336 —— writer.Error() 從未檢查
```

客戶端中途斷線（或磁碟寫滿）會讓管理員拿到一份被截斷的 CSV、HTTP 200、而且
**沒有任何一行日誌**。而這個端點自己在 `csv_export.go:346-351` 的理由是
「頁面上必須寫出上限，讓管理者知道匯出檔不是完整的資料集」——靜默的 I/O 截斷
讓對帳同樣無法進行。

**修法**：每個 `writer.Flush()` 之後補一行

```go
if err := writer.Error(); err != nil {
    logger.WarnfContext(r.Context(), "[EXPORT] 寫出失敗（檔案可能不完整）: %v", err)
}
```

---

### M8. `normalizeBatchEmails` 不檢查長度，會讓稽核寫入回 500 而不是 400

**檔案**：`backend/forum/httpapi/batch_handlers.go:116`；`backend/forum/audit/audit.go:203-205`

```go
if !strings.Contains(email, "@") || strings.ContainsAny(email, " \t\r\n") {
    return nil, &requestError{message: "email 格式不正確：" + email}
}
```

只檢查「有 @ 且無空白」。這個 email 會被 `recordAdminAction` 當成 `targetID`
傳入，而 `audit.Record` 只截斷 `TargetLabel`（`audit.go:203`），
`TargetID` 是原樣綁定的（`audit.go:205`），目標欄位是 `target_id VARCHAR(320) NOT NULL`。

管理員送出一個約 400 字元的「email」（遠在 16 KiB body 上限之內）就會讓 MySQL
回 error 1406（Data too long）→ `audit.Record` 往上傳 → **整批 200 人的交易回滾、
回 500**。正確的答案應該是 400（輸入不合法），而不是 500。

`TargetLabel` 有截斷而 `TargetID` 沒有，這個不對稱本身就是問題：
同一個值在一欄被截斷、在另一欄不截。

**修法**：在邊界擋掉（欄位是 `VARCHAR(255)`）

```go
if len(email) > 255 {
    return nil, &requestError{message: "email 長度超過 255 字元"}
}
```

並讓 `audit.Record` 對 `TargetID` 套用與 `TargetLabel` 相同的截斷
（`target_id` 是 `VARCHAR(320)`，截到 `maxValueLength` = 200 是安全的）。

---

### M9. `MONITOR_RETENTION_HOURS` 對監控時間軸長度**完全沒有作用**（文件與實作不符）


**檔案**：`backend/forum/metrics/metrics.go:694-699`、`backend/main.go:224`；
`README.md:288-289`、`:473`

README 宣稱：

> `MONITOR_RETENTION_HOURS` 決定 `forum_request_metrics` 留多久（…）**以及監控頁時間軸的長度**。

但時間軸長度只由 `windowMinutes` 決定：

```go
func (r *Registry) timelineLocked(now time.Time) []TimelinePoint {
    currentMinute := minuteIndex(now)
    length := r.windowMinutes          // ← 固定 120，來自 NewServer 的 Options
    if length > maxTimelinePoints { length = maxTimelinePoints }   // 144
```

`Snapshot(retentionHours)` 只把該值原樣放進回應的 `RetentionHours` 欄位，
沒有任何地方用它決定要畫幾格。

**後果**：
- 設定 `MONITOR_RETENTION_HOURS=72` 不會讓時間軸變長，管理員會以為設定沒生效。
- README:473「時間軸上一格 24 小時前的資料是從資料庫讀回來的」這句在實作上不可能發生
  （時間軸只有 120 格 = 2 小時）。
- `main.go:224` 的 `since = now - MonitorRetentionHours` 會從資料庫讀回最多 1440 列
  歷史（24 小時），接著 `pruneLocked`（`metrics.go:911-923`）在 20 秒內把它裁到 120 列。
  其餘 1320 列純屬白讀、白佔記憶體、白耗一次 DB 往返。

**修法**：二選一。要嘛讓 `timelineLocked` 依 `retentionHours` 決定長度（並相應調大
`maxTimelinePoints`），要嘛改 README 說明「此設定只影響資料庫保留，時間軸固定 120 分鐘」，
並讓 `LoadHistory` 的 `since` 對齊實際會被顯示的區間。

---

### M10. 公告的 `expiresAt` 帶著假的 `Z`，`Effective` 判定混用兩個時鐘

**檔案**：`backend/forum/httpapi/announcement_handlers.go:101`、`:144`、`:188`、`:348`、`:378`

```go
// SELECT 端
COALESCE(DATE_FORMAT(expires_at, '%Y-%m-%dT%H:%i:%sZ'), '')
// 判定端
item.Effective = item.Active &&
    (item.ExpiresAt == "" || item.ExpiresAt > time.Now().UTC().Format(time.RFC3339))
```

`DATE_FORMAT` 用的是 **MySQL session 的時區**（預設 `SYSTEM`），而
`time.Now().UTC()` 是 UTC。兩者被當成同一個時鐘做字串比較，字尾卻都硬寫成 `Z`。
`expires_at` 本身是以 `time.Now()`（本機時間）寫入的（`announcement_handlers.go:229-230`），
所以 `DATE_FORMAT` 輸出的是本機掛鐘時間，卻對外宣稱是 UTC。

**症狀**（站台不在 UTC 時，例如 README 情境的 UTC+8）：
- 一則 3 小時前就過期的公告仍顯示為「顯示中」，且持續顯示到 UTC 偏移過後為止
  （此時公開端點 `expires_at > NOW()` 已經不顯示橫幅了 → 後臺說有、公開頁沒有）。
- 後臺列表的到期時間比實際早/晚一個偏移量。
- 同一份 payload 裡 `createdAt` 是正確的（DATETIME 掃描成 `time.Time` 後以
  `RFC3339Nano` 輸出，帶真實偏移），`expiresAt` 卻是假的 `Z` → 兩者自相矛盾。

**連帶影響**：`announcement_handlers.go:378` 的稽核 diff 用
`Before: beforeExpires`（`DATE_FORMAT` 字串）比對 `After: formatExpiry(expires)`
（UTC RFC3339 或「（永不過期）」），格式不同 → `onlyChanged` 幾乎永遠判定
`expiresAt`「有變更」，稽核紀錄因此出現大量假變更。

**修法**：統一在應用層做時區換算。要嘛把 `expires_at` 也以 UTC 寫入並讓 SQL 端
`DATE_FORMAT(..., '%Y-%m-%dT%H:%i:%s')` 去掉假的 `Z`、由前端當作 UTC 解析；要嘛
乾脆在 Go 端把 `expires_at` 掃成 `time.Time` 再自行 `Format(time.RFC3339)`，
完全不要在 SQL 裡組時間字串（同一份檔案裡 `recordAdminAction` 已經是這樣做的，
見 `audit_log.go:129`）。

---

### M11. 稽核查詢頁每次都對整張稽核表做兩次 `SELECT DISTINCT`

**檔案**：`backend/forum/httpapi/audit_log.go:231-240`；`backend/forum/audit/query.go:223-268`

```go
result, err := audit.List(r.Context(), s.db, filter)      // 內部還有 COUNT(*)
...
actions, err := audit.DistinctActions(r.Context(), s.db)   // SELECT DISTINCT action ...
actors,  err := audit.DistinctActors(r.Context(), s.db)    // SELECT DISTINCT actor_email ...
```

三個查詢都在 `/api/admin/log` 每次打開時執行，後兩個沒有 `LIMIT`、沒有上界，
而 `DistinctActors` 用的 `actor_email` 沒有涵蓋索引（只有
`idx_forum_admin_actions_actor (actor_email, created_at)`，MySQL 只能做 loose index scan）。

`AUDIT_RETENTION_DAYS` 預設 90 天，而稽核表的寫入量來自後臺所有寫入操作
（停權、改標籤、代發文、批次每人一筆）。表格長大之後，這兩支查詢的成本會隨
資料量線性上升，而且失敗時只降級成空清單（`audit_log.go:233`）——症狀是篩選器
「突然少了幾個選項」，看起來像沒有那些操作發生過。

**修法**：
- 把 action / actor 清單做成記憶體快取（異動時失效，或 5 分鐘 TTL），
- 或改成「先取前 N 筆（依 `created_at` 排序）再從中 distinct」，
- 或直接加一張 `forum_admin_action_facets` 對照表，由後臺寫入時順帶 upsert。

另外 `audit.List` 每次都跑 `SELECT COUNT(*)`（`query.go:153`），而 `Offset` 沒有
上限（第 118-129 行的 `normalize` 只夾負數）—— 深度 offset 會變成全表掃描，
建議也加上 `MaxOffset`。

---

### M12. `csrf.go` 檔頭對現況的描述已過期，會誤導後續維護

**檔案**：`backend/forum/httpapi/csrf.go:19-20`

```go
目前只有 /api/logout 掛上這層保護（見 server.go）；其餘寫入端點僅依賴
SameSite 與 requireLogin。
```

實上現在**每個**會改變資料的端點都自行呼叫 `isTrustedOrigin`（本次變更新增的
公告、置頂、IP 封鎖、session 撤銷、批次操作都在內，共 12 處）。這段過期說明正好
寫在「防線分層」這個最需要準確的地方，而 `forum_admin_handlers.go:41-42` 的說明
（「每一條寫入路徑都必須同時通過這兩道檢查」）與它互相矛盾。

**修法**：把該段改成描述現況（handler 內顯式呼叫，非中介層），並點出為什麼不能
改成中介層（同一條路由同時服務 GET）。

---

### M13. `createAdminForumPost` 等路徑在 `Commit` 之後才更新 ES 索引，失敗時索引會指向已刪除的資料

**檔案**：`backend/forum/httpapi/forum_admin_handlers.go:1182-1196`、`:1289-1303`、`:1344-1356`

`createAdminForumPost` 與 `handleAdminForumPost`（PUT / DELETE）都在
`tx.Commit()` 之後呼叫 `s.indexForumPost` / `s.reindexForumPostByID` /
`s.unindexForumPost`。這個順序本身是對的（註解也解釋得很清楚：避免索引指向被回滾的
資料）。但 `handleAdminForumPost` 的 **DELETE** 分支在 `Commit()` 之後、
`s.unindexForumPost` 之前就已經回應不了錯誤——`unindexForumPost` 是 best-effort。

這點本身是可接受的取捨，問題在於 `adminActionPostDelete` 的稽核紀錄**不包含
「索引是否同步成功」**，而稽核紀錄是這個專案用來回答「使用者看到的是什麼」的依據。
刪文之後若 ES 索引殘留，搜尋結果會出現點進去 404 的幽靈貼文（程式碼註解自己
指出了這個症狀），而稽核紀錄不會留下任何痕跡。

**修法**（低成本的緩解）：`unindexForumPost` 回傳 error，在失敗時追加一筆
`post.index_failed` 的稽核紀錄（走 `s.db`，因為此時交易已提交）。

---

## Low

### L1. 根目錄有一個 0 位的 `vite.config.ts`

`D:\Projet\forum\forum\vite.config.ts` 是未追蹤的空檔案。真實設定在
`frontend/vite.config.ts`。任何在 repo 根目錄執行 `npx vite build` 的動作（或日後
有人在根目錄加 npm script）都會讀到這個空設定。

**修法**：刪除它。

---

### L2. `_code_statistics.csv` 是產生的統計檔，未被 gitignore

未追蹤的 `_code_statistics.csv`（語言 / 檔數 / 程式行數統計）目前在 `.gitignore`
之外，隨時可能被 `git add .` 帶進版控。

**修法**：刪除或加進 `.gitignore`。

---

### L3. `server.go` 有一段重複的「IP 封鎖名單」註解

`backend/forum/httpapi/server.go:576-577` 與 `:584-586` 是同一段註解兩份：
第一份被放在公告路由上方（錯位），第二份才對應到 `mux.HandleFunc("/api/admin/blocks", ...)`。

**修法**：刪掉 `:576-577`。

---

### L4. `session.revoke` 的動作名稱與它的 `target_type` 不符約定

`audit_log.go:42-43` 宣稱「格式為 `"<target_type>.<動作>"`，前綴必須與 audit 的
TargetType 常數一致」，但 `adminActionSessionRevoke = "session.revoke"`
（`audit_log.go:91`）搭配的是 `audit.TargetUser`（`session_admin_handlers.go:213`）。

`/admin/log` 的 action 篩選器是從 `DistinctActions` 動態產生的，所以功能上不會壞；
但約定不一致會讓「依 target_type 前綴分組」的未來功能（以及人工查詢時的直覺）出錯。

**修法**：要嘛改名為 `user.sessions.revoke`，要嘛在約定敘述裡明確列出這個例外。

---

### L5. `isMissingValueError` 用 `==` 比對 `redis.Nil`

`backend/forum/session/session_list.go:392-394`

```go
func isMissingValueError(err error) bool {
    return err == redis.Nil
}
```

同檔案其他地方的慣例是 `errors.Is`（見 `ipban.go:186`）。目前 go-redis 會直接回傳
`redis.Nil` 哨兵值所以不會出錯，但一旦上游改為包裝回傳，`pipe.Exec` 的錯誤處理就會
從「略過過期 session」變成「中止整批掃描」。

**修法**：改用 `errors.Is(err, redis.Nil)`。

---

### L6. README 的 MySQL 版本需求寫得太寬，新遷移步驟實際需要 8.0.29+

`README.md:111` 寫「MySQL 8.x」。但這次新增的第 26 步（`data/mysql.go:714-717`）
同時用了兩個各自有版本門檻的語法：

- `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` — 需要 **MySQL 8.0.29+**
  （8.0.29 之前只有 `CREATE TABLE IF NOT EXISTS` 與 `DROP COLUMN IF EXISTS`）
- `ADD INDEX (pinned DESC, created_at DESC, id DESC)` — 遞減索引需要 **MySQL 8.0+**

`ADD COLUMN IF NOT EXISTS` 在既有的第 14、15 步（`mysql.go:149,382,389`）就已存在，
所以門檻不是這次引入的；但 README 的「8.x」從來沒有被修正成實際可運作的版本，
而第 26 步把它從「理論上」變成「這次部署就會踩到」。症狀是遷移直接回
`ERROR 1064 (42000): You have an error in your SQL syntax`，
`MigrateMySQL` 回錯 → `main` 呼叫 `Fatalf` → 行程在啟動時就死。

**修法**：把 `README.md:111` 與部署章節的版本需求改成 `MySQL 8.0.29+`
（若要支援 8.0.29 以下，第 26 步的欄位新增得改用 `information_schema` 探測 +
`ADD COLUMN`，與第 17、20、21 步處理索引的作法一致）。

---

### L7. `fetchRecords` 在 pipeline 部分失敗時會把 TTL 讀成 0

`backend/forum/session/session_list.go:353-381`

若 `pipe.Exec` 因為**指令層**錯誤（而非連線層）回錯，程式會繼續往下走
（`!isMissingValueError(err)` 只擋連線層錯誤）。此時該批中失敗的 `fields[i].Result()`
會回錯而被 `continue` 跳過，但 `ttls[i]` 若只是部分失敗就會回傳零值
`DurationCmd`，`Val()` 得到 `0` → `ExpiresAt = now` → 介面顯示「馬上要到期」。

**症狀**：低頻且需要 pipeline 部分失敗，實際發生機率低，但顯示的是**錯誤的**
到期時間（而不是「未知」）。

**修法**：`if err := ttls[i].Err(); err != nil { record.TTL = -1 }`，讓
`ExpiresAt` 保持零值、由呼叫端呈現為「未知」。

---

## 檢查過但確認正確的部分

以下看起來可疑但實際沒問題，記錄下來以免下次重查：

- **SQL 注入面**：本次新增的 SQL 全部使用佔位符。僅有的字串插值有兩處，都安全：
  `stats_handlers.go:249` 的 `+" "+table`（只由三個字串常數餵入）、
  `batch_handlers.go:283` 與 `audit/query.go:79-80` 的 `IN (?,?,…)` 佔位符數量
  （由 `len()` 決定，不是使用者控制的結構字串）。沒有任何 HTTP 參數能到達
  表名、欄位名或 `ORDER BY` 位置。
- **授權**：所有新的 `/api/admin/*` handler 都在 method 分派**之前**呼叫
  `requireAdminForum`（未登入者不會用「方法不支援」反覆探測路由是否存在），
  且所有寫入分支都有 `isTrustedOrigin`。這些 handler 沒有使用路徑參數，因此沒有
  IDOR 面。
- **稽核與操作同生共死**：`beginAdminTx` 的**每一個**呼叫端都在 `tx.Commit()` 之前
  檢查 `recordAdminAction` 的錯誤，且 `defer tx.Rollback()` 到位。已逐一確認
  `forum_admin_handlers.go`（post/comment/report 的建立、修改、刪除、裁定）、
  `user_admin_handlers.go`（tag 建立/刪除/改名、user 狀態、user tags、代發文/代留言）、
  `announcement_handlers.go`、`batch_handlers.go` 兩支、`blocklist.go` 的
  `recordBlockAction`、`session_admin_handlers.go` 的 revoke —— 全部符合這個不變條件。
  包裝成 24 個呼叫點，沒有一個漏檢。
- **CSV 公式注入防護**：完整。危險前綴涵蓋 `= + - @ TAB CR`（`csv_export.go:66`），
  三份匯出的每一格都經過 `writeCSVRow` → `sanitizeCSVField`（`:138`）；
  BOM 在 `csv.NewWriter` 之前寫出（`:121`）；`filename` 由兩個套件常數組成（`:129`），
  `Content-Disposition` 無法被注入。由 `batch_export_test.go` 的六前綴表格測試守住。
- **資料競爭**：`metrics.Registry` 由 mutex / atomics 完整保護；
  `RateLimiter.Stats` / `TrackedKeys` 有加鎖，`Limit` / `Window` 讀的是建構後不變的欄位；
  `probeDependencies`（`monitoring.go:207-233`）寫進有鎖的 map，而 `wg.Wait()`
  建立了 happens-before 邊界後才讀。沒有 goroutine 洩漏（三個探測都受 2 秒
  context deadline 約束）。**但注意：本次審查無法執行 `go test -race`
  （環境缺 C 編譯器），所以這是靜態審查結論，未經 race detector 實證。**
- **SCAN 用法**：`session_list.go:290-327` 的游標正確帶回續掃；預算超出時切斷並標記
  `truncated`；`RevokeByEmail` 在截斷時回錯（不是回 0）；`seenKeys` 處理了
  SCAN 不保證不重複回傳的問題；每批用 pipeline 一次撈回 HGETALL + TTL。
- **ServeMux 衝突**：`/api/admin/forum/posts/{id}` 與 `/{id}/pin` 沒有重複註冊同一樣式
  （改用 `handleAdminPostOrPin` 依尾綴分流），避免了 Go 1.22+ 啟動時 panic。
  16 個前端殼檔名與 `securityheaders_test.go` 的同步測試一致，
  `frontend/vite.config.ts` 的 16 個 input 也對得上。
- **`DATE(created_at)` 掃描成 `time.Time`**：`stats_handlers.go` 的 `dailyCounts`
  依賴 `parseTime=True` 讓 driver 走日期解析分支，這在
  go-sql-driver/mysql 的 `packets.go:869-878` 有對應實作，成立。
- **i18n 完整性**：18 個語系由 `Record<MessageKey, string>` 這個型別在**編譯期**強制，
  `npm run typecheck` 通過即代表沒有任何語系漏 key（也沒有任何元件用了不存在的 key）。
- **XSS**：新前端沒有任何 `dangerouslySetInnerHTML` 的**實際呼叫**（只在註解中提到），
  公告內文、稽核 diff、CSV 欄位全部以 React 文字節點或 `textContent` 渲染。

---

## 建議的修復順序

1. **H0 + M0 + M1**（補 `forum_post_comments.author_email` 與
   `forum_post_likes.created_at` 兩個索引）—— 一個冪等遷移步驟就同時解掉三個
   全表掃描／N+1，成本最低、效益最大。
2. **H2**（後端加 `pinned`）—— 目前置頂功能是半壞的狀態，會產生「以為改了但沒改」
   的操作，而且連稽核紀錄都不會有。
3. **H1 + H3**（公告）—— 兩者都會讓管理員的正常操作造成非預期的公開可見狀態變更，
   而公開橫幅是全站使用者都會看到的。
4. **M2**（`tagIds` 上限）—— 單一請求可在單一交易裡產生近 160 萬次 INSERT，
   這是資源耗盡而非效能問題。
5. **H4**（clientIP 信任模型）—— 影響範圍是整個限流 / 封鎖 / 稽核體系的正確性，
   但修法牽涉部署假設，需要先確認生產環境的代理拓撲。
6. **M4**（`COALESCE` 型別混用）—— 先在測試環境用真實 MySQL 確認行為再修；
   修法本身很低成本，無論確認結果如何都該改。
7. **M9、M10**（文件與時區）—— 會讓維運做出錯誤判斷，修法明確。
8. **M11**（稽核頁查詢成本）、**M3**（批次標籤的重複稽核）—— 目前資料量下不會痛，
   可排入下次重構。
9. **L1–L7** —— 可併入任何一次提交。
