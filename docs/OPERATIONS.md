# 維運手冊

部署細節、後台各頁在回答什麼問題、以及怎麼處理常見狀況。設計理由見
[`ARCHITECTURE.md`](ARCHITECTURE.md)，安全相關見 [`SECURITY.md`](SECURITY.md)。

- [部署前驗證](#部署前驗證)
- [停止行為](#停止行為)
- [容器部署的取捨](#容器部署的取捨)
- [systemd 部署](#systemd-部署)
- [後台各頁](#後台各頁)
- [常見狀況](#常見狀況)

---

## 部署前驗證

```bash
cd backend
go run . -check                              # 本機
docker compose run --rm backend -check      # 容器內，不啟動任何服務
```

它讀設定檔、套用兜底值、跑一次 `Validate`，然後輸出一份報告。**它不連線任何外部服務** ——
所以它回答的是「這個行程會以什麼組態啟動」，而不是「相依服務能不能連上」。

報告**刻意不印任何憑證**（只印「已設定（N 個字元）」），因為它會出現在 CI 日誌、issue
回報與截圖裡，而那些地方經常沒有存取控制。

它點名的都是**啟動之後完全沒有症狀**的那一類：

| 檢查 | 抓什麼 |
| --- | --- |
| `COOKIE_SECURE` / `PUBLIC_BASE_URL` 判定為開發環境，卻有正式環境的痕跡 | 忘了開 Secure cookie |
| 三組限流的實際額度 | 設定檔打錯字時會沿用預設值，而那「看起來有開」 |
| 媒體 token 的 TTL，以及**它是明確設定還是採用兜底值** | 兜底值 30 天的 TTL 對正式環境明顯過長 |
| 管理員白名單的**實際儲存值**（去空白、轉小寫之後） | 這正是部署者要確認的東西：「我寫的那個信箱，真的在名單裡嗎」 |
| 逾時值 | 拿來和部署端的停止上限對照 |
| `FILES_SERVER_TOKEN` 未設定的警告（正式環境） | 上傳會全部失敗 |
| Redis key 前綴 | 含 `session` 會被拒絕（前綴相撞會讓兩種資料互相覆蓋） |

**這是整份部署流程裡最值得做的一步。** 它花不到一秒，擋掉的是那些不會報錯、只在正式
流量上才會出現的問題。

---

## 停止行為

後端與 files_server 都實作了優雅停止。收到 `SIGTERM` / `SIGINT` 之後：

1. 停止接受新連線
2. 等在途請求完成，上限 `SHUTDOWN_TIMEOUT_SECONDS`（預設 15 秒）
3. 寫出最後一次分鐘彙總
4. 關閉資料庫連線
5. 以 **exit code 0** 結束

**超過上限仍有請求沒完成時會記警告並強制中斷。** 一個回 1 會讓部署端以為有東西壞了。

### 部署端的停止上限必須大於它

否則部署端會先動手殺掉行程，看到的就是「排空被中斷」而不是乾淨的停止。

| 環境 | 預設停止上限 | 怎麼調 |
| --- | --- | --- |
| systemd | `TimeoutStopSec=90s` | 通常已足夠 |
| Docker | `--stop-timeout=10` | `docker run --stop-timeout=20 …` 或 compose 的 `stop_grace_period` |
| Kubernetes | `terminationGracePeriodSeconds=30` | 部署 manifest 內設定 |

compose 的設定：

| 服務 | `stop_grace_period` | 為什麼是這個值 |
| --- | --- | --- |
| `backend` | 20s | 比 `SHUTDOWN_TIMEOUT_SECONDS`（15s）多 5 秒，那 5 秒給最後一次監控彙總與關閉連線 |
| `files_server` | 20s | 與 backend 相同。**對這個服務更重要**：一次 50 MB 的圖片上傳被砍斷時，後端已經把貼文寫進 MySQL 了，而檔案沒有落地 —— 症狀是資料庫裡有一篇文章引用一張不存在的圖片，**沒有任何日誌、沒有任何錯誤碼** |
| `mysql` | 20s | InnoDB 需要時間 checkpoint |
| `redis` | 10s | AOF 重寫很快 |

### 更好的做法

讓反向代理**先**把流量抽掉再轉送停止訊號給後端。那樣後端收到訊號時本來就沒有新請求要接，
排空會是瞬間完成的事，也就不需要 20 秒的寬限。

### 優雅停止會做與不會做的事

會：寫出最後一分鐘的監控彙總 → 所以「正常重啟」不會留下當前分鐘的資料缺口。

不會：處理**非正常結束**（`SIGKILL`、容器被硬殺、排空超時後被部署端殺掉）—— 那種情況下
當前分鐘的監控資料仍會遺失。

---

## 容器部署的取捨

| 決定 | 理由 | 代價 |
| --- | --- | --- |
| 後端映像是 **distroless** | 沒有 shell 與套件管理員；一個有 shell 的正式環境容器距離「有人 exec 進去手動修東西」只有一步 | 診斷不能靠 `exec` 進去看檔案；改用 `LOG_FORMAT=json` 從 stdout 收集 |
| 檔案服務映像是 **alpine** | 這個服務處理使用者上傳的檔名（不可信輸入），需要一個 shell 作為未來的緩解手段 | 多 7 MB、多一層 libc 的 CVE 曝露面 |
| **沒有前端服務** | 被提供的檔案與被授權的 CSP 雜湊必須來自同一次建置 | 每次改前端都要重建 backend 映像 |
| **不用 Docker secrets** | secrets 檔案是明文（base64），靠檔案權限保護 —— 與 `settings.conf` 的保護機制**相同**。而真正值得保護的 `GOOGLE_CLIENT_SECRET` 與 `FILES_SERVER_TOKEN` 在應用層的設定檔裡，從不進版控、也從不進 build context | `settings.conf` 必須靠 gitignore 保護 |

### 沒有前端服務的理由（完整版）

後端啟動時掃描前端產物、算出每個 HTML 殼裡 `<style>` 的 SHA-256 寫進 CSP 的 `style-src`。
分開部署的症狀是**整頁沒有版面**而主控台只有一行
`Refused to apply inline style`（`style.css` 是外部檔案所以仍會作用，看起來「有樣式但
怪怪的」）。

`backend/Dockerfile` 的建置情境因此是**倉庫根目錄**，把 `npm ci` + typecheck + build 與
Go 編譯放進同一個 stage。詳細理由見 [ARCHITECTURE](ARCHITECTURE.md#為什麼前端沒有獨立服務)。

### 建置腳本會驗證映像

`./deploy/build-images.ps1`（Windows）在建置之後會驗證映像的檔案配置 —— 前端產物有沒有
進去、`/asset/` 的素材在不在、掛載點的擁有權是不是 nonroot。

**這三件事壞掉時容器照樣會起來**，症狀是全站 404 或「安裝了圖示但打開是空白」。

### healthcheck

- `backend` 呼叫自己的 `/app/forum -healthz`（distroless 沒有 shell 與 curl）。`timeout 5s`
  必須大於 `-healthz` 內部的 2 秒探測期限，否則結果是「探測被中斷」，那與「相依不可用」
  無法區別。
- `mysql` 用 `mysqladmin status` 而**不是** `ping` —— `ping` 會在伺服器還在初始化時就
  成功，那一刻它對任何資料庫都回 "Access denied"。
- **`files_server` 刻意不設 healthcheck。** 它沒有可探測的端點，而加一個會把「Redis 掛掉
  時降級為不驗證媒體 token」變成「容器被判定為不健康並重啟」，直接衝突。

---

## systemd 部署

容器不是唯一的選擇，systemd 與 docker compose 是**並存**的兩條路。

部署條件與容器相同：反向代理在別處、`COOKIE_SECURE=true`、`TRUSTED_ORIGINS` 對齊、
`TimeoutStopSec` > `SHUTDOWN_TIMEOUT_SECONDS`、`TRUSTED_PROXY_CIDRS` 填真正在前面
那道代理的位址段。

`backend/update.sh` 是給 Debian/Ubuntu 的一次性建置腳本（apt 裝 Go 再 `go build`）。

---

## 後台各頁

每頁回答一個其他頁答不出來的問題。設計理由見
[ARCHITECTURE](ARCHITECTURE.md#登入與-session-管理) 等對應段落。

| 頁面 | 回答什麼 |
| --- | --- |
| `/admin` | 帳號管理、停權、標籤指派、批次操作 |
| `/admin/forum` | 貼文／留言 CRUD、**置頂**、**以使用者身分代發文** |
| `/admin/forum-report` | 檢舉佇列裁決（通過即刪除對象、駁回保留） |
| `/admin/monitor` | **現在**正在怎樣：相依服務、執行期用量、請求量與延遲、來源位址 |
| `/admin/log` | **誰**動了**什麼**：欄位級 diff，保留 90 天 |
| `/admin/stats` | 站**正在長成什麼樣子**：每日新增、熱門文章、活躍作者 |
| `/admin/export` | 把資料帶走：三份 CSV |
| `/admin/sessions` | **現在有誰在線上**，以及怎麼強制登出 |
| `/admin/blocks` | IP 封鎖名單 |
| `/admin/announcements` | 全站橫幅公告 |

### 監控頁要看的三件事

**1. `clientIpTrust.mode` 是 `legacy-headers` → 限流可被偽造標頭繞過。**
`declared` 非空卻解析不出任何位址時，`mode` 也是 `legacy-headers` 而 `declared` 非空 ——
那就是「設定寫錯了」的可見訊號。

**2. `stats.clientsDropped` 非 0 → 來源清單被截斷了。**
上限 200 個來源，超過時驅逐最久沒再出現的那一個。**沒有那個數字，一份被截斷的清單看起來
會和一份完整的一模一樣。**

**3. `stats.clients` 裡 `source: "xff"` 且 mode 是 `legacy-headers` → 那些位址是假的。**
此時這一欄不能當成證據。

### 監控頁的三個已知限制

| 限制 | 影響 |
| --- | --- |
| **延遲分位數是分桶上界** | P95 只會落在 1/2/5/10/25/50/100/250/500ms 或 1/2/5/10s 上。跟精確的 APM 數字比較會失望 |
| **總量只算本次啟動** | 「上個行程一共服務了多少請求」答不出來。分鐘彙總會落盤，所以時間軸不會歸零；要長期資料得看存取日誌 |
| **多執行個體時只顯示打到這個行程的流量** | 資料庫的分鐘彙總會把各個個體加總，但即時統計不會 |

### 批次操作的兩個保證

- 每個受影響的對象**各自**記一筆稽核（「一次停權 50 個帳號」若只記一筆，稽核就答不出
  「這個 email 什麼時候被停權的」）。
- **整批生效或整批不動**（單一交易），不會出現「第 30 個失敗、前 29 個已經停權」。

回應的 `skipped` 逐項帶原因，讓你知道跳過的是「帳號不存在」還是「格式錯誤」。

### 強制登出的稽核順序

`POST /api/admin/sessions/revoke` 的稽核寫在 MySQL，動作在 Redis，兩者沒有共同交易。

順序是**先刪 Redis（動作），再寫稽核（紀錄）** —— 「動手之後忘了記錄」比「記錄了但還沒
動手」安全，後者會讓稽核宣稱「已撤銷」而 session 還活著。

稽核寫入失敗時回 **500 並說明「session 已撤銷但紀錄失敗」**。介面上這看起來像矛盾，
但那正是這個取捨的樣子 —— **寧可少一筆紀錄，也不要一筆不實的紀錄。**

---

## 常見狀況

### 圖片上傳成功，但貼文裡的圖一律 403 / 401

`MEDIA_TOKEN_KEY_PREFIX`（後端）與 `token_key_prefix`（files_server）不一致。兩邊都
要是 `forum:token:`，且都要與 session 的 `forum:session:` 前綴區隔。

**範本已對齊，既有設定檔可能還留著舊值。**

### 整頁沒有版面，`style.css` 好像有作用但怪怪的

瀏覽器主控台會有一行 `Refused to apply inline style`。代表 HTML 殼不在 CSP 的
`style-src` hash 清單裡 —— 三處清單不同步：

1. `frontend/src/entries/`
2. `frontend/vite.config.ts` 的 `rollupOptions.input`
3. `backend/forum/httpapi` 的 `frontendShellFiles`

`frontend_shells_test.go` 會指名缺哪一處。

### 頁面載入後完全沒有反應（React 從來沒掛載）

`src/entries/X.tsx` 不存在 —— 建置產物裡沒有那一頁的入口。症狀是頁面 404 或主控台
沒有錯誤。

### PWA 不會更新

代理沒有放行 `Service-Worker-Allowed` 與 `/service-worker.js`。

### 停權按了沒反應（後台寫入全部回 500）

稽核表的 INSERT 與操作共用交易，所以**稽核表故障會讓後台寫入全部失敗**。這是刻意的取捨：
寧可「停權按了沒反應」，也不要「停權成功了但沒人知道是誰停的」。

先確認 `forum_admin_actions` 狀態（磁碟滿是最常見的原因）。唯讀頁面不受影響。

### 上傳到一半被部署打斷，貼文存得下但圖片沒有

`files_server` 的停止上限不夠。50 MB 的上傳被砍斷時後端已經把貼文寫進 MySQL 了。
把 `files_server` 的 `stop_grace_period` 調到 ≥ 20 秒。

### 登入後又被登出

Redis 不可用。session 查不到 key 就等於未登入（fail closed），而 72 小時的滑動 TTL 意味
著 Redis 掛掉超過一段時間後所有人都需要重新登入。

這是刻意的：寧可導回登入頁，也不要在 Redis 不穩時放行未驗證的請求。

### 圖片一直載不出來，但上傳是成功的

確認 `FILES_SERVER_PUBLIC_URL` 是**瀏覽器可達**的位址（不是內部的 `192.168.x.x`），
以及代理**沒有**吃掉 `/files/*` 的 `?token=` 查詢參數。

### `-check` 說「正式環境未設定 MEDIA_TOKEN_TTL_SECONDS」

這是**擋你，不是 bug**。兜底值 30 天對正式環境明顯過長。寫出你想要的數字即可 ——
刻意設成 30 天是被允許的（那代表你知道自己在做什麼）。
