# 安全模型

本檔回答三個問題：**這個系統的信任邊界在哪裡**、**哪些東西被刻意信任了**、**上線前
必須確認什麼**。設計理由見 [`ARCHITECTURE.md`](ARCHITECTURE.md)。

- [信任邊界](#信任邊界)
- [已驗證的風險：Redis 就是信任根](#已驗證的風險redis-就是信任根)
- [管理員授權](#管理員授權)
- [CSRF](#csrf)
- [XSS 與 CSP](#xss-與-csp)
- [輸入驗證與 SQL](#輸入驗證與-sql)
- [檔案上傳](#檔案上傳)
- [匿名性的邊界](#匿名性的邊界)
- [刻意不做的事](#刻意不做的事)
- [上線檢查表](#上線檢查表)
- [已知的低風險問題](#已知的低風險問題)

---

## 信任邊界

```
        網際網路                可信任區（家裡的 LAN / 內網）
  ─────────────────  ────────────────────────────────────────────
  瀏覽器  ──TLS──▶  反向代理  ──▶  backend :8088
                                  ├─ MySQL   （內容、稽核）
                                  ├─ Redis   （★ 憑證與權限都在這裡）
                                  └─ files_server :7070
```

**唯一真正的授權資料在 Redis。** `forum_users` 沒有 role 欄位；`is_admin` 只存在於
`FORUM:session:<token>` hash。因此信任邊界不是「MySQL 沒被入侵」，而是「**沒有人能讀寫
Redis**」。

這個設計本身是好的（可立即撤銷、不需要簽章金鑰），但它把後果集中到一個點上：Redis 可達
就等於全站沦陷。

| 假設被打破時 | 後果 |
| --- | --- |
| Redis **可讀** | 列出所有有效 session token → 以任何登入者身分操作 |
| Redis **可寫** | 自行建立 `is_admin=true` 的 session → 完整管理員權限 |
| Redis **不可用** | 全站登入失效（fail closed），IP 封鎖同時失效（fail open） |

---

## 已驗證的風險：Redis 就是信任根

**這是實測確認的，不是理論推演。** 記錄下來是為了讓下一次有人質疑這個設計時，有證據
可以對照。

### 實測條件

- 攻擊者位置：與服務**同一個 LAN**（不需要從網際網路可達）
- 攻擊者持有的東西：`backend/config/config.conf` 的內容（部署者的檔案；該檔已
  gitignore，未進版控）

### 實測結果

**第一步 —— 憑證是弱密碼，且與 MySQL root 共用：**

```ini
# backend/config/config.conf
REDIS_PASSWORD=collapse
DB_DSN=root:collapse@tcp(192.168.66.5:3306)/forum?…
```

`192.168.66.5:6379` 對整個 LAN 開放，`AUTH collapse` 直接回 `OK`。

**第二步 —— 只讀就足以接管所有管理員：**

```
SCAN 0 MATCH FORUM:session:* COUNT 500
→ 3 keys
    email=cllapse.money@gmail.com  is_admin=true
    email=cllapse.money@gmail.com  is_admin=true
    email=cllapse.money@gmail.com  is_admin=true
```

**token 就是 Redis key 的名稱**，所以把任一個填進 `forum_forum` cookie 就是管理員。
**不需要寫入權限。**

**第三步 —— 寫入可以自己開管理員：**

```
HSET FORUM:session:<自己選的 24 位元組 hex> email cllapse.money@gmail.com is_admin true
EXPIRE FORUM:session:<同上> 259200
→ HGETALL → {'email': 'cllapse.money@gmail.com', 'is_admin': 'true'}
```

cookie 值**不需要是應用發出來的那個** —— 後端只把它當查詢鍵。驗證：

```
GET /api/check   Cookie: forum_forum=<自選 token>
→ {"isAdmin":true,"ok":true}

POST /api/forum/posts   （Origin 為站點網域）
→ {"ok":true,"item":{"id":4,"authorKey":"8d26441d…"}}
```

`8d26441d…` = `SHA-256("cllapse.money@gmail.com")`，確認作者就是管理員本人。

**清理**：刪掉 `FORUM:session:<自選 token>` 之後，同一個 cookie 立即回
`{"isAdmin":false,"ok":false}`。原有的 3 支 session 未受影響。

### 為什麼 web 應用本身擋不住這件事

`is_admin` 全站**只有一個寫入點**（`session/session.go` 的 `Manager.Create`），而
`Create` 只從 Google OAuth 回呼呼叫。要讓它在**不動 Redis**的情況下被寫成 `true`，必須
讓 Google 為 `cllapse.money@gmail.com` 簽發一組授權碼給本站的 OAuth client ——
攻擊者沒有那個 Google 帳號就做不到。

實測排除過的路徑：

| 嘗試 | 結果 |
| --- | --- |
| 未登入打 `/api/admin/*`（posts / users / sessions / stats / users/{email}/content） | 全部 401 |
| 帶偽造 session cookie 打 `/api/admin/*` | 401（`HGET` 查不到 key → fail closed） |
| 找 Redis 的其他寫入點 | 全站 `HSet` 只有一處 |
| 讓媒體 token 改寫 session key | 前綴 `forum:token:` 固定，且啟動檢查拒絕含 `session` 的前綴 |
| 貼文內容 XSS | 前端是 React，全站 `dangerouslySetInnerHTML` **實際出現 0 次** |
| **後台圖片 `href` XSS**（`admin/ForumAdminPage.tsx` 把 `imageUrl` 放進 `href`） | **擋住了** —— DB 只存純檔名，讀取時才用 `FILES_SERVER_PUBLIC_URL` 現組；白名單排除 `:` `/` 與 svg |

最後那一條是唯一一條純網際網路可能走到管理員的路（後台頁的 `href` 若可控就是儲存型
XSS），但驗證是完整的。

**結論：管理員權限本身在 web 應用層是守住的。整個邊界在 Redis 上。**

### 修正

按重要性：

1. **Redis 綁內網介面。** `bind 127.0.0.1` 或防火牆只允許 backend 與 files_server 的
   來源。**這比換密碼重要**：弱密碼只擋得住暴力破解，綁網段擋得住整個 LAN。
2. **用 Redis ACL，不要用全域 `requirepass`。** ACL 可以只給 `session.*` 與 `FORUM:token:*`
   需要的權限，並且**不允許 `CONFIG`、`FLUSHALL`、`EVAL`**：

   ```
   ACL SETUSER forum on >~FORUM:session:* ~FORUM:token:* ~FORUM:ipban +@read +@write -@dangerous
   ```

   `-@dangerous` 擋掉 `KEYS` / `FLUSHALL` / `CONFIG` 這類會被用來一步取得全部 token 的指令。
3. **密碼獨立且夠長。** 不要與 MySQL root 共用 —— 共用密碼代表任一處洩漏等於兩處淪陷，
   而且兩處的 log 與設定檔會互相佐證。
4. **不要把 `REDIS_PASSWORD` 與 `GOOGLE_CLIENT_SECRET` 放在同一個檔案裡。** 目前七種
   憑證都在 `config.conf`（含 ES 的帳密與 API key），任何一次檔案外洩就是全部。
   若要分開，可讀的是「把 ES 指到同一份設定檔之外的部署端」而不是抽環境變數 ——
   見 [CONFIGURATION](CONFIGURATION.md) 的 ES 驗證段。
5. **設定 `TRUSTED_PROXY_CIDRS`**（見下），並確認 `X-Forwarded-For` 在最外層就被剝掉。
6. **網路層加一條**：即使 Redis 只在內網，也值得確認後端**沒有**把 `192.168.x.x` 這類
   私有位址暴露成可從外部連線的服務（例如誤設的 port forward）。

---

## 管理員授權

### 判定鏈

```
Google userinfo → email
  → cfg.IsAdminEmail(email)          # 精確比對，雙邊 TrimSpace + ToLower
  → session.Create(email, isAdmin)   # 寫死進 Redis
  → 之後每個請求：IsAdmin(r) → HGET is_admin → isAdmin == "true"
```

- **email 只來自 Google**，不是任何 request 欄位、header 或 cookie。
- 比對是**精確字串比對**，不是正規化後的模糊比對。刻意**不**處理 Gmail 點號與 `+`
  別名 —— 白名單的重點是「只有這些明確的信箱可以當管理員」。
- `IsAdmin` 的任何 Redis 錯誤都回 false（fail closed）。

### 所有權與內容歸屬

- 作者一律取自 `ResolveUser(r)`，不接受 body 傳入的 authorEmail。
- 編輯／刪除自己的內容，權限判斷在**SQL predicate 裡**而不是先 SELECT 再比對：

  ```sql
  UPDATE forum_posts SET content = ?, updated_at = ? WHERE id = ? AND author_email = ?
  ```

  0 列同時代表「不存在」與「不是你的」，**不區分**以免藉回應差異探測作者。

- 留言用三段式 predicate：`id AND author_email AND post_id`。

### 刻意不做的事

**後台可以冒用任意使用者發文。** `POST /api/admin/users/{email}/posts` 直接把**路徑中**
的 email 寫成 `author_email`，沒有 email 格式驗證、沒有二次確認，也沒有速率限制
（後台路由刻意不掛限流）。

這是**設計功能**（「以使用者身分代發文」用於檢舉裁決），且有稽核紀錄
（`user.posts.create`）。但要意識到：**一個管理員 session 被取得，就同時等於一個
「用任何人的身分說任何話」的原語。** 這讓 Redis 入侵的影響遠大於一般的 session 劫持。

### 移除白名單不會撤銷既有 session

`is_admin` 是登入當下寫死的。要真正撤銷必須呼叫
`POST /api/admin/sessions/revoke {"email":…}`，或等 72 小時滑動 TTL 到期。

**後台沒有提供「刪除稽核紀錄」的 API，也沒有「撤銷自己以外所有人」以外的批次撤銷。**

---

## CSRF

三層防護：

| 層 | 說明 |
| --- | --- |
| `SameSite=Lax` cookie | 擋跨站 POST。**主要防線** |
| Origin / Referer 白名單 | 30 個呼叫點，每一條改變資料的路徑都要呼叫 |
| CSP | 偏向 XSS |

`csrf_guard_test.go` 用 AST 走訪每個 handler 的呼叫圖，找出缺守衛的路徑。**這支測試抓到
過一個真實缺口**（圖片 token 釋放端點）。

### 兩件必須知道的事

**「兩個標頭都沒有」會刻意放行。**

```go
if origin == "" {
    referer := strings.TrimSpace(r.Header.Get("Referer"))
    if referer == "" {
        // 兩者都沒有就放行…
        // 殘餘風險：刻意移除兩個標頭的隱私設定或代理會讓跨站請求也通過，
        // 這時本防護失效，必須仰賴 SameSite cookie 與後端的身分檢查。
        return true
    }
```

**因此 `403 invalid origin` 不是可靠的閘門** —— curl、腳本、某些 SDK 都不送這兩個標頭，
會直接跳過這道檢查。真正回答「這個請求有沒有身分」的是後端的身分檢查。

**沒有 synchronizer token。** 只靠 `SameSite=Lax` 撐住跨站 POST這一條。這在現代瀏覽器
上是可接受的，但它意味著 **`SameSite` 屬性是被依賴的，不是可有可無的**。

比對是精確 `EqualFold`，**不做預設埠或尾斜線正規化** —— `https://x` 與 `https://x:443`
是不同的 origin，寫錯會讓全部寫入被擋（或全部通過）。

### 登入 CSRF 未被緩解

OAuth 的 `state` 參數被**重複利用為「登入後要回到哪一頁」**，而不是標準建議的 CSRF
隨機 nonce，因此 `state` 未綁定發起登入的瀏覽器。

攻擊者可以讓受害者完成一次**攻擊者自己帳號**的登入（login CSRF）。這**不是提權**，
但它是可用的釣魚／信任降級原點：受害者之後在這個站上輸入的內容，攻擊者用同一個 OAuth
token 都讀得到。

唯一的防線是 `isSafeReturnPath` 這道開放轉向檢查（會先解碼、拒絕 `//` 與反斜線）。

---

## XSS 與 CSP

### CSP

`script-src 'self'` —— 沒有 `unsafe-inline`、沒有 `unsafe-eval`。另加
`X-Frame-Options: DENY`、`nosniff`、`Referrer-Policy`、COOP，以及
`frame-ancestors 'none'`。

HTML 內的 `<style>` 區塊在啟動時算 SHA-256 寫進 `style-src` 的 hash 清單 —— 所以
樣板**不能隨意新增內嵌樣式**。症狀是整頁沒有版面而主控台只有一行
`Refused to apply inline style`。

`HSTS` **受 `COOKIE_SECURE` 控制** —— `COOKIE_SECURE=false` 時不會送出。

### 渲染路徑

前端是 React，文字節點自動跳脫。`dangerouslySetInnerHTML` 在 `.tsx` 檔中**實際出現
0 次**（`icons.tsx` 的註解說明它不需要）。

### 圖片網址是唯一的 href sink

後台貼文列表把 `imageUrl` 放進 `<a href>`：

```tsx
<a className="post-figure" href={item.imageUrl} target="_blank" rel="noreferrer noopener">
```

若 `imageUrl` 能是 `javascript:`，那就是後台頁的儲存型 XSS → 管理員接管。這條路被三層
擋住：

1. **寫入時** `forumImageFileName` 把輸入收斂成純檔名，收斂失敗直接回 400。
2. **檔名白名單**只接受 `[A-Za-z0-9._-]`，副檔名在 `{.jpg,.jpeg,.png,.gif,.webp}`，
   **不含 svg**（SVG 可內嵌腳本）。
3. **資料庫只存檔名**，`imageUrl` 在讀取時才用 `FILES_SERVER_PUBLIC_URL` 現組 —— 因此
   就算資料庫被塞進髒值，呈現出去的还是設定檔決定的網域。

> 這一條是**縱深防禦**，而縱深防禦的條件是每一層都獨立成立。若日後有人為了「支援外部
> 圖片」而放寬 `forumImageFileName`，`javascript:` 就會立刻變得可用 —— 那個改動必須
> 同時檢查後台的 `href`。

---

## 輸入驗證與 SQL

- 全部查詢走 `database/sql` 的**參數化查詢**。沒有字串拼接 SQL。
- 表格、欄位名稱全部是常數；`targetType` 是由 dispatcher 傳入的字面值，不是使用者輸入。
- `ORDER BY` / 欄位名稱若需動態選擇，走白名單比對而非直接插入。
- 內文長度以 `len([]rune(...))` 計算（不是位元組數），否則全中文貼文會在約三分之一
  長度就被判超限。
- 登出時回 `0 列` 的分支回 **404 而非 403**，以免藉回應差異探測某篇內容是否存在或屬於誰。

---

## 檔案上傳

| 防線 | 內容 |
| --- | --- |
| 檔名 | 落地一律 **UUID v4 + 原副檔名**，永遠不用使用者提供的檔名 |
| 副檔名 | 白名單 `{.jpg,.jpeg,.png,.gif,.webp}`，**不含 svg** |
| 大小 | `upload.max_size`，預設 50 MB |
| 路徑穿越 | 刪除端點對 `..`、多層路徑等做完整家族測試 |
| 存取 | 綁短 TTL 的 Redis token（`FORUM:token:*`），不是公開網址 |
| 授權 | 上傳需要登入 + 限流 + 來源檢查 |
| 結構 | **沒有任何路徑能列出儲存結構** |

`files_server` 的媒體 token 驗證在 Redis 故障時回 **503 而非 401** —— 兩者的差別是
「憑證無效」與「無法判斷」。

---

## 匿名性的邊界

**email 不出現在任何 URL 與公開頁面**，對外一律用 `public_key = SHA2(author_email, 256)`
（`publicForumKey`；遷移第 13 步會把既有貼文的作者回填，`public_key` 只存在於
`forum_profiles`，且有索引）。

但這個雜湊是**確定性**的，而 email 的取值空間很小，所以：

> **任何使用者的完整發文紀錄可以被枚舉還原。**

```bash
# 不需登入
GET /api/forum/public-posts?user=<SHA-256(email)>
```

實測以管理員 email 的雜湊取回該帳號的全部 4 篇貼文。攻擊者枚舉常見信箱、雜湊比對即可。
`/api/forum/public-profile?key=` 同理（注意參數名與前者不一致：`key` vs `user`）。

**這不是授權漏洞**（那些貼文本來就是公開的），但它是**身分隱私的漏洞**：匿名性承諾的是
「別人不知道那是你的信箱」，而這個承諾在有耐心的攻擊者面前不成立。

要真正修掉需要加 salt 或改成不可逆但不可枚舉的識別方式，兩者都有代價 —— 這是一個產品
決策，不是實作細節。

---

## 刻意不做的事

| 不做 | 理由 |
| --- | --- |
| **per-account 限流** | 目前只有 per-IP，共用 NAT 使用者會互相影響。讀取端點是匿名的，限流對它沒有意義 |
| **登入 CAPTCHA / 速率限制的帳號維度** | 目前 OAuth 端點有 IP 限流（10/60s），足以擋轟炸 |
| **自動 IP 封鎖** | 同一出口位址可能是一整間辦公室或一整個 NAT。**誤封正常使用者的後果比多讓一個腳本多打幾次嚴重得多。封鎖必須是管理員的決定。** |
| **CDR / 範圍封鎖** | 範圍封鎖會讓「這個 IP 被封了嗎」變成一個需要逐一比對的問題 |
| **session 綁 IP 或 UA** | 換 IP 的使用者會被登出；token 竊取者也能在同 IP 下重放 |
| **刪除稽核紀錄的 API** | 一個能刪除自己紀錄的稽核日誌等於沒有稽核日誌 |
| **後台限流** | 所有操作都需要管理員身分，限流只會在真正出事時多一條混淆的訊息（`/admin/monitor` 尤其每十秒打一次） |
| **`ReadTimeout` / `WriteTimeout`** | 前者會連請求本文一起計時，而貼文附圖本文可達 50 MB 且要轉送；後者會在回應寫完前砍斷連線，而 CSV 匯出的耗時就落在這段裡 |

---

## 上線檢查表

`./forum -check` 會自動檢查其中一部分，但下面幾項它看不到：

**憑證與網路**

- [ ] `COOKIE_SECURE=true`（走 HTTPS 就必須），並確認 HSTS 有送出
- [ ] Redis **只綁內網**，且用 ACL 而非全域 `requirepass`
- [ ] Redis 密碼獨立於 MySQL root，且足夠長
- [ ] MySQL root 不從外部可達
- [ ] `TRUSTED_PROXY_CIDRS` 已填成**真正在前面那道代理**的位址段
- [ ] 最外層的代理有**剝掉**外部送來的 `X-Forwarded-For`，而不是照單全收後追加
- [ ] Elasticsearch 若未啟用，`ES_URL` 是留空的
- [ ] Elasticsearch 若啟用了安全性，`ES_USERNAME`/`ES_PASSWORD` 或 `ES_API_KEY`
      有填（兩者皆空時每個 ES 請求都是 401，而搜尋會安靜退回 MySQL）

**設定一致性**

- [ ] `PUBLIC_BASE_URL`、`GOOGLE_REDIRECT_URL`、`TRUSTED_ORIGINS` 三者對齊實際對外網址
- [ ] `TRUSTED_ORIGINS` 裡**沒有**殘留的 `localhost` / `127.0.0.1`（它們讓非瀏覽器客戶端
      可以聲稱任何來源）
- [ ] `MEDIA_TOKEN_TTL_SECONDS` 有**明確**設定（正式環境用兜底值會讓啟動被拒絕）
- [ ] `ALLOWED_ADMIN_EMAIL` 用 `-check` 確認過**實際儲存值**（去空白、轉小寫之後）
- [ ] `FILES_SERVER_TOKEN` 兩邊一致；`MEDIA_TOKEN_KEY_PREFIX` 與 files_server 的
      `token_key_prefix` 一致
- [ ] `FILES_SERVER_PUBLIC_URL` 與 `FILES_SERVER_URL` 不同（相同代表沒有內外網分離）

**部署環境**

- [ ] 停止上限 > `SHUTDOWN_TIMEOUT_SECONDS`（Docker `stop_grace_period` / systemd
      `TimeoutStopSec`）
- [ ] 代理放行 `Service-Worker-Allowed` 與 `/service-worker.js`
- [ ] 代理**沒有**吃掉 `/files/*` 的 `?token=` 查詢參數
- [ ] `config.conf` 沒被 git 追蹤（`git log --all --full-history -- '**/config.conf'`）

---

## 已知的低風險問題

這三項是實測確認、但**不需要內網位置或憑證**就能利用的。它們不是管理員提權，但都是
純網際網路可達的問題。

### 1. 限流可被單一偽造標頭繞過

`TRUSTED_PROXY_CIDRS` 留空時（**預設**），限流的分桶鍵是 `X-Forwarded-For` 的最左項：

```go
if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
    parts := splitComma(xff)
    if len(parts) > 0 {
        return parts[0], metrics.ClientSourceXFF
    }
}
```

輪替這個標頭就能重置 10 次/分鐘的寫入額度。**症狀是「限流看起來有開，只是擋不住
任何人」**，而且沒有任何錯誤。

IP 封鎖**不受**此影響（它檢查所有候選位址，命中任何一個就算被封）。受影響的是限流與
稽核紀錄的來源位址 —— 後者因此**不能當作證據**。

監控頁的 `clientIpTrust.mode` 顯示 `legacy-headers` 就是沒設定的訊號。

### 2. 任意使用者的發文紀錄可枚舉

見[匿名性的邊界](#匿名性的邊界)。未登入，`/api/forum/public-posts?user=<hash>`。

### 3. 登入 CSRF

見[CSRF](#csrf)那一節。`state` 是返回路徑而非 nonce。

### 順帶：`public-profile` 與 `public-posts` 的參數名不一致

`/api/forum/public-profile` 用 `?key=`，`/api/forum/public-posts` 用 `?user=`。
兩者都是 64 字元長度檢查。這是現況不是筆誤，但兩個端點並排時很容易記錯。
