# 開發路線圖

這份文件記錄**接下來要做的順序，以及那個順序的理由**。它不是願望清單：每個
Phase 都有量測過的現況作為前提，順序本身有依據（見「為什麼是這個順序」）。

本文件不取代 `README.md`。`README.md` 描述**這個專案現在是什麼**（架構、
設定、API、資料庫、部署），本文件描述**它接下來要去哪裡**。兩者刻意分開：
現況說明要能對照程式碼驗證，計畫則一定會過期。

---

## 目錄

- [量測基準](#量測基準)
- [已裁決的事](#已裁決的事)
- [為什麼是這個順序](#為什麼是這個順序)
- [Phase 0 — 衛生](#phase-0--衛生)
- [Phase 1 — CI 閘門](#phase-1--ci-閘門)
- [Phase 2 — 把紀律變成工具](#phase-2--把紀律變成工具)
- [Phase 3 — 覆蓋風險面](#phase-3--覆蓋風險面)
- [Phase 4 — Docker 化](#phase-4--docker-化)
- [Phase 5 — 觸發式備忘錄](#phase-5--觸發式備忘錄)
- [Phase 6 — 功能](#phase-6--功能)
- [刻意不做的事](#刻意不做的事)
- [未驗證的前提](#未驗證的前提)

---

## 量測基準

建立於 **2026-10-02**，全部為當日實測，不是估計值。重跑指令寫在每項下方。

### 規模

| 項目 | 數值 | 量測方式 |
| --- | --- | --- |
| Go 檔案總數 | 53 檔 / 約 22,000 行 | `Get-ChildItem -Recurse -Filter *.go` |
| `httpapi`（業務邏輯核心） | 30 檔 / **12,678 行** | 同上 |
| `httpapi` handler 數 | **51** | `Select-String -Pattern '^func \(s \*Server\) handle'` |
| `beginAdminTx` 呼叫點 | **25** | `Select-String -Pattern 'beginAdminTx\('` |
| 限流掛載點 | **8** | `server.go:528,529,543,548,550,556,561,566` |
| `files_server` | **1 個 Go 檔、559 行** | `files_server/main.go` |
| 前端 TS/TSX | 71 檔 / 31,633 行 | `Get-ChildItem -Recurse -Include *.ts,*.tsx` |
| ├ 其中 i18n 目錄 | 20,500 行（**64.8%**） | `frontend/src/i18n/` |
| └ 其中 UI 程式碼 | 11,133 行（35.2%） | 差值 |
| HTML 殼（entry） | 16 | `frontend/*.html` |

i18n 佔前端行數的 65% 這件事本身是個訊號：這個專案前端的「體量」有三分之二
是翻譯資料，不是邏輯。評估前端複雜度時不該被總行數嚇到。

### 測試覆蓋

`cd backend && go test ./... -covermode=count`（總計 **18.7%**）

| 套件 | 覆蓋 | 判讀 |
| --- | --- | --- |
| `es` | 84.3% | 純傳輸層，容易測 |
| `ipban` | 82.4% | 有界、無 DB |
| `metrics` | 72.6% | 注入假時鐘，設計良好 |
| `session` | 67.1% | miniredis |
| `audit` | 50.0% | |
| `forum`（package main） | 12.9% | 優雅停止等 |
| **`httpapi`** | **9.5%** | **12,678 行的商業邏輯幾乎沒測** |
| `auth` | **0%** | **OAuth 零覆蓋** |
| `data` | **0%** | **遷移層零覆蓋，失敗即啟動即死** |
| `config` | **0%** | |
| `logger` | **0%** | |
| **`files_server`** | **0%** | **整個模組零測試** |

**覆蓋與風險完全倒置**：被測最好的是最容易測的（純函式、有界、可用
miniredis），風險最高的三塊（遷移、OAuth、handler）接近零。

### 自動化閘門

| 項目 | 現況 | 影響 |
| --- | --- | --- |
| CI | **無**（無 `.github/`、無 Makefile/taskfile） | 所有驗證靠人手 |
| 前端測試 | **無**（`package.json` 內無 `test` 腳本） | 11,133 行 UI 零保護 |
| `go test -race` | **跑不了**（`CGO_ENABLED=0`，本機缺 gcc） | 並行程質是靜態審查結論，未經實證 |
| 測試相依 | 僅 `miniredis v2.35.0` | 無 testcontainers / sqlmock |
| 容器產物 | **無**（無 Dockerfile、compose、`.dockerignore`） | |

`CODE_REVIEW.md`（另一個 worktree 內）那一輪審查找出 26 個問題、0 個
Critical、5 個 High，證明**人工審查在這個專案上是有效的**。但它同樣證明
有效是有價格的 —— 而那 26 個問題現在全部已修。下一輪沒有任何機制保證會有
第二輪。

### 開發環境（本機實測）

| 項目 | 狀態 | 對計畫的影響 |
| --- | --- | --- |
| Go | **1.26.0** | `go.mod` 版本不一致可立即統一 |
| Node | 22.13.0 | |
| MySQL | **已在 :3306 執行** | **Phase 3.1 遷移測試不需 Docker 即可開始** |
| Redis | 未執行 | 無妨：Redis 測試用 `miniredis` 走行程內 |
| Docker | **未安裝** | Phase 4 產物可寫但**本機無法驗證構建** |

### 順帶查出的文件缺陷

量測時發現 `README.md` 的多語系數量有 off-by-one，**兩處都多算了 1**：

| 位置 | 宣稱 | 實際 |
| --- | --- | --- |
| `README.md:63` | 「18 個語系（含 `zh-TW` 為基準）」 | **17** |
| `README.md:1069` | 「共 17 份，連同基準的 `zh-TW` 共 18 個」 | **16 份，共 17 個** |

`frontend/src/i18n/translations/` 實際有 16 個檔（`ar de en es fr hi id ja ko
pt-BR ru th vi zh-CN zh-HK zh-MO`），加上基準的 `src/i18n/messages.ts` = 17。
README 自己列出的語系清單也只有 16 個，與它宣稱的數字對不上。

這不是排版小錯 —— 它是「文件與實作漂移」的第一個實例，而那正是本路線圖
Phase 0 與 Phase 2 要處理的同一類問題。

---

## 已裁決的事

2026-10-02 由站方裁決，這兩點決定了本路線圖的形狀。

### 部署方式：Docker

單一 `docker-compose` 堆疊（MySQL + Redis + files_server + backend +
選用反向代理）。

**這個裁決順帶解決了本專案最大的架構懸而未決的問題。** 行程內滑動視窗限流
（`ratelimit.go` 檔頭的取捨）在負載平衡背後會讓實際額度變成 `limit × N`，
而 N 不是使用者能控制的 —— 這是 `ipban` 存在的原因（限流實作的封鎖會隨
部署自動解除）。單容器部署下，**行程內限流是完全正確的，不是妥協**。

因此 Phase 5 的「跨行程限流」從架構決題降級成備忘錄。

### 上線時程：近期不上線

這是**比看起來更好的情況**，不是更壞的。沒有搶救性的排序壓力，代表可以
把最枯燥但最重要的部分（CI 閘門、不變條件工具化）做完整，而不是做一半先
趕功能。唯一的直接後果：Phase 5 的三件事全部不實作。

---

## 為什麼是這個順序

一句話：**這個專案的瓶頸不是功能，是「不變條件只靠紀律維持」，而目前沒有
任何自動化閘門會攔下違規。**

這個專案有四組很棒的不變條件：

1. **稽核與操作同生共死** —— 25 處 `beginAdminTx` 呼叫端都必須在 `Commit()`
   **之前**檢查 `recordAdminAction` 的錯誤。漏檢等於把稽核降級成「盡力
   記錄」，而症狀不會出現在任何日誌、任何狀態碼、任何測試失敗裡。
2. **`sanitizeCSVField` 是 CSV 防護的唯一實作點** —— 漏掉一個匯出，
   同樣沒有任何症狀。
3. **`withBlocklistHandler` 是單一掛載點** —— 順序不可交換（反過來會讓被
   封鎖的 IP 先累積限流計數，解封後仍生效）。
4. **16 個 HTML 殼檔名在三處同步** —— 前端 entry、`vite.config.ts` 的
   `rollupOptions.input`、後端 `httpapi.frontendShellFiles`（CSP style
   hash 授權）。少一處，那一頁整頁沒有版面。

**第 1 與第 2 項完全沒有工具保護，第 4 項有測試但只有一個。** 在這種情況下
加功能，等於擴大一個沒有人會自動檢查的面。

所以順序是：

```
Phase 0（衛生，1-2 天）
   └─ 移除會誤導後人的東西
Phase 1（CI，2-3 天）          ← 其餘一切的保護網
   └─ 先有閘門，之後的變更才不是裸奔上線
Phase 2（不變條件工具化，4-6 天）  ← 最高槓桿
   └─ 把「靠紀律」變成「靠工具」，讓 Phase 3 之後不必依賴有人記得看
Phase 3（覆蓋風險面，8-12 天）  ← 依爆炸半徑排序，不依覆蓋率百分比
Phase 4（Docker，6-8 天）        ← compose 先行、Dockerfile 殿後
Phase 5 / 6（擱置）
```

兩個刻意的排序決定：

- **Phase 2 在 Phase 3 之前。** 先有工具才補測試，補的測試才不會在之後被
  下一輪變更悄悄破掉。反過來做會得到兩份要一起維護的東西。
- **Phase 4 的 compose 先行、Dockerfile 殿後。** compose 是**測試基礎設施**
  優先、是**部署產物**其次。理由見 Phase 4。

---

## Phase 0 — 衛生

**1–2 人日。每項獨立、可單獨合併。**

這一階段刻意排在 CI 之前：這些都是「看一眼就知道對不對」的問題，成本最低，
而它們每一個都在浪費後續所有工作 —— 一段指向錯誤前提的註解，會讓下一個人
重新推理一個已經解決的問題。

### 0.1 修掉會誤導的註解

`backend/forum/httpapi/forum_handlers.go:65-66`：

> Server.rateLimiter 雖已建立，但 server.go 未把它掛到任何路由上，
> 因此本檔案的寫入端點目前沒有速率限制。

**這是假的。** `server.go` 在 8 條路由上掛了限流（`:528` `/auth/google`、
`:529` `/auth/callback`、`:543` 貼文、`:548` 上傳、`:550` 釋放 token、
`:556` 貼文動作、`:561` 個人資料、`:566` 追蹤）。

危害不在於它錯，而在於它寫在「已知限制」這個最需要準確的位置，且與
`server.go:245-246` 的說明互相矛盾。`CODE_REVIEW.md` 的 M12 發現過同型問題
（`csrf.go` 檔頭），本項是同型的第二例。

**驗收**：全庫 sweep 一輪 `沒有速率限制|未掛|尚未實作|不存在的|沒有實作`，
逐條確認每一則仍然成立。已確認乾淨的部分不必動。

### 0.2 統一 `go.mod` 版本

`backend/go.mod` 是 `go 1.25.0`，`files_server/go.mod` 是 `go 1.26.0`。
本機工具鏈是 **go1.26.0**，兩者都滿足，因此統一到 `1.26.0` 不需要任何
額外安裝。

理由不只是整潔：`files_server` 需要 1.26 的理由目前是隱性的，而兩條指令
需要不同版本本身就是一個持續的環境地雷。

**驗收**：`cd backend; go build ./...` 與 `cd files_server; go build ./...`
都通過。

### 0.3 `files_server` 優雅停止

`files_server/main.go:644` 仍是 `log.Fatal(http.ListenAndServe(...))` ——
收到 `SIGTERM` 直接死，沒有逾時也沒有排空。

**Docker 決策讓這件事從「註解裡的注意事項」升級成「compose 裡必須寫對的
數值」**：

| 項目 | 預設 |
| --- | --- |
| Docker `--stop-timeout` | **10 秒** |
| 後端 `SHUTDOWN_TIMEOUT_SECONDS` | **15 秒** |

後端排空需要最多 15 秒，但 Docker 只等 10 秒 → **後端會在排空到一半時被
SIGKILL**。症狀是「貼文存得下但圖片上傳失敗」（後端轉送上傳的那條連線被
中斷）。README 有記這件事，但記錄不等於防護；`stop_grace_period` 必須寫進
compose（Phase 4.2）。

`files_server` 自己也要補上同樣的流程：它是另一個 Go 模組，卻是唯一還沒
處理停止流程的。

**驗收**：`docker stop` 送出 SIGTERM 後，兩個容器都在 grace period 內以
exit code 0 結束。

### 0.4 `ALLOWED_ADMIN_EMAIL` 正規化

`README.md` 已知問題：比對大小寫敏感且不做正規化。白名單若寫成帶空白或
大寫會**靜默失效** —— 症狀是管理員被鎖在後台外，且沒有任何錯誤訊息
（Google 回傳的 email 已是穩定的小寫）。

這是靜默失效，而靜默失效是這一階段要消滅的東西。

**驗收**：載入時 `strings.TrimSpace` + `strings.ToLower`；測試涵蓋帶空白、
大寫、混合三種輸入。

### 0.5 `MEDIA_TOKEN_TTL_SECONDS` 的兜底值

預設 30 天的兜底值對正式環境明顯過長，而**採用這個值的動作是靜默的**。

**驗收**：正式環境（`COOKIE_SECURE=true` 或 `PUBLIC_BASE_URL` 為 https）
未明確設定時，啟動明確報錯或警告；開發環境仍可採用兜底值。

### 0.6 補 `vite.config.ts` 的 `server.proxy`

`npm run dev` **開箱即壞** —— 沒有 `server.proxy`，前端所有相對路徑的
`/api/*` 請求會打到 Vite(5173) 而不是後端(8088)。

這條擋住所有前端開發，是整條路線圖上最便宜的品質提升。

```ts
server: {
  proxy: {
    '/api': 'http://localhost:8088',
    '/auth': 'http://localhost:8088',
    '/healthz': 'http://localhost:8088',
  },
},
```

**注意**：`/files/*` **不要**加進 proxy。前端在開發模式下取圖走的是
`FILES_SERVER_PUBLIC_URL`（預設 `:7070` 的直接跨來源請求），把它轉進 8088
會與媒體 token 的驗證路徑互動出意料之外的行為。

**驗收**：`npm run dev` 連同 `go run .` 起在 8088，登入、發文、上傳圖片
全數可用。

### 0.7 修 `README.md` 的多語系數字

見「順帶查出的文件缺陷」。兩處 18 → 17，`README.md:1069` 的 17 份 → 16 份。

**驗收**：README 的數字與 `frontend/src/i18n/translations/` 的實際檔案數
一致。建議同時在 `tools/i18n/verify-catalogs.mjs` 加一個斷言讓它不會再漂。

---

## Phase 1 — CI 閘門

**2–3 人日。這是其餘所有階段的前提。**

沒有了上線 deadline，唯一能防回歸的就是 CI。這一階段在「近期不上線」的
裁決下價值**上升**。

### 1.1 三個 job

| Job | 內容 |
| --- | --- |
| `backend` | `go build ./...` → `go vet ./...` → `go test ./... -race` |
| `files_server` | `go build ./...` → `go vet ./...` → `go test ./... -race` |
| `frontend` | `npm ci` → `npm run typecheck` → `npm run build` |

`npm run build` 內含 `typecheck`，所以前端 job 沒有漏掉型別檢查
（這個專案的 lint 就是型別檢查：`strict` + `noUncheckedIndexedAccess` +
`exactOptionalPropertyTypes`）。

### 1.2 開 CGO 跑 race detector

**成本近乎零，收益是把一個假設變成事實。**

`CODE_REVIEW.md` 對資料競爭的結論是「沒有發現問題，**但注意：本次審查
無法執行 `go test -race`（環境缺 C 編譯器），所以這是靜態審查結論，未經
race detector 實證**」。

這個專案大量依賴 goroutine 與記憶體共享：`metrics.Registry`（mutex +
atomics）、三個 `RateLimiter`、`probeDependencies` 的三個並行探測、
`beginAdminTx` 邊界的 `defer tx.Rollback()`。CI 映像裝 gcc 即可，成本是
一個 apt line。

**驗收**：`go test -race ./...` 在 CI 通過，且測試確實執行了（不是因為
沒有 `_test.go` 而空跑）。

### 1.3 覆蓋率地板

從當前 **18.7% 起算，只許升不許降**。

起點刻意不是 0，也不是「越高越好」：18.7% 如實反映「工具性套件測得好、
業務邏輯沒測」這個狀況。地板的作用是**防止退化**，不是製造動機。

**刻意不做**：不設一個虛高的目標（例如 70%）。那會誘發無行為斷言的測試
來拉數字，而那種測試是負債 —— 它會在重構時壞掉，卻沒有指出任何東西壞了。

### 1.4 i18n 目錄驗證

`node tools/i18n/verify-catalogs.mjs` 納入 CI。

i18n 完整性目前**已經**在編譯期由 `Record<MessageKey, string>` 這個型別
強制（`npm run typecheck` 通過即代表沒有任何語系漏 key，也沒有任何元件用
了不存在的 key）—— 這是這個專案最好的設計之一。但 `verify-catalogs.mjs`
檢查的是 build 之外的事（未翻譯殘留、佔位符一致性），不該缺席。

### 1.5 Secret scanning

`config.conf` 被 `.gitignore` 擋住，但那是**唯一**的防線。CI 加 secret
scanning，並在偵測到 `config.conf` 被追蹤時直接失敗。

**刻意不做**：不把 `FILES_SERVER_TOKEN` 改成只能走環境變數。它已經支援
環境變數（README 記錄了這是唯一的例外），compose 與 systemd 都會用變數
注入。

---

## Phase 2 — 把紀律變成工具

**4–6 人日。本路線圖槓桿最高的一段。**

別的專案在這裡會寫「多寫測試」。但這個專案的不變條件是**專屬的**，
通用覆蓋率抓不到它們 —— 一個 handler 從 0% 變成 30% 覆蓋，與「`Commit()`
前有沒有檢查稽核錯誤」毫無關係。

自訂 `go/analysis` analyzer 成本低、誤報可控，而且失敗模式是**編譯失敗**
而不是「有人沒注意到」。

### 2.1 稽核不變條件（最高價值）

`beginAdminTx` 的 **25 個**呼叫端，每一個都必須在 `tx.Commit()` **之前**
檢查 `recordAdminAction` 的錯誤。

`README.md` 自己說明這個不變條件有多脆弱：

> 這件事很容易被無聲破壞：Go 的 `database/sql` 不會因為交易內某個語句失敗
> 就中止交易，所以呼叫了 `recordAdminAction` 卻不看回傳值，等於把稽核降級
> 成「盡力記錄」。

**Analyzer 設計**：追蹤 `beginAdminTx(...)` 回傳的 `*sql.Tx`，檢查在同一
函式內 `tx.Commit()` 被呼叫之前，`recordAdminAction(...)` 的回傳值有被
當成 error 使用。漏檢即編譯失敗。

**驗收**：`httpapi` 在 analyzer 開啟下乾淨編譯；刻意移除一個檢查會讓
`go build` 失敗（用一個測試鎖住這個行為）。

### 2.2 管理端點授權

每個 `/api/admin/*` handler 必須在 method 分派**之前**呼叫
`requireAdminForum`。順序有意義：反過來會讓未登入者用「方法不支援」反覆
探測路由是否存在。

**Analyzer 設計**：比對 `server.go` 裡的路由註冊與對應 handler 的函式本體，
確認授權檢查在 method 分派前。

**驗收**：同上，含負向測試。

### 2.3 `withBlocklistHandler` 單一掛載點

`ipban` 的 `IsBanned` 時機由**一個**函式決定，讓「先查封鎖、再查限流」
不會有第二個掛載點而不同步。順序不可交換。

**Analyzer 設計**：拒絕第二個掛載點。

### 2.4 HTML 殼三處同步

沿用 `securityheaders_test.go` 已有的模式（它已經守住後端清單與實際殼檔
的同步，並驗證 hash 的完整與穩定），**補上「新增殼檔必須三處齊備」**：
`src/entries/`、`vite.config.ts` 的 `rollupOptions.input`、
`httpapi.frontendShellFiles`。

**驗收**：新增一個 HTML 殼而不動另外兩處，測試失敗並指名缺哪一處。

### 2.5 為什麼這一段排在測試之前

補的測試會被下一輪變更悄悄破掉。先有 analyzer，補的測試才有一層工具層
擋著。

---

## Phase 3 — 覆蓋風險面

**8–12 人日。依「壞掉時的爆炸半徑」排序，不依覆蓋率百分比。**

按百分比補會得到一個漂亮但沒有防禦價值的數字。

### 3.1 `forum/data` 遷移層（0% → 70%）｜最高優先

**為什麼排第一**：遷移失敗 → `MigrateMySQL` 回錯 → `main` 呼叫 `Fatalf`
→ **行程在啟動時就死**。而 `README.md` 兩處承認：**沒有版本化的 migration，
也沒有降級路徑**。

這是全專案唯一一個「失敗時沒有任何部分成功」的地方 —— 其他地方的失敗最多
是功能不可用，這裡是站開不起來。

**為什麼需要真實 MySQL**：`go-sqlmock` 無法驗證 DDL 真的能執行；
`miniredis` 對 MySQL 沒有對應物。而 `ADD COLUMN IF NOT EXISTS`（需要
MySQL 8.0.29+）與遞減索引這兩個語法正是要驗的東西。

**本機 MySQL 已在 :3306 執行 —— 這一項現在就能開始，不等 Docker。**

測試重點：

- **冪等性** —— 同一個遷移連跑兩次，schema 與資料不變
- **部分失敗後的重啟** —— 第 N 步失敗時，前 N-1 步已套用；下一次啟動
  應該補齊剩餘步驟（README 的檔頭宣稱這個行為，**沒有任何測試**）
- **從舊版 schema 升級** —— 步驟 6/7/15/16/17/20/21 各自處理一種舊結構
- **下限版本的可見性** —— 8.0.29 以下會拿到 `ERROR 1064`；測試應該讓這個
  診斷訊息可被理解

**策略**：用一個拋棄式的資料庫（測試開頭 `CREATE DATABASE`、
結尾 `DROP`）跑在 CI 提供的 MySQL service 上，而不是在程式裡起容器 ——
CI 已經是容器裡了，`services:` 區塊比 testcontainers 更簡單也更可靠。

### 3.2 `forum/auth`（0% → 80%）

**為什麼排第二**：安全關鍵且完全沒測，而它只有 251 行。

- `isSafeReturnPath` —— **開放重導向的守門**。繞過它意味著
  `?return_path=https://evil.example` 會在 OAuth 完成後把使用者送去任意外站
- `ReturnPath` —— OAuth 流程的狀態載體，處理不當是 CSRF
- `GetUserEmail` —— 決定整個身分模型
- `Init` / `HandleLogin` —— 用 `httptest` 假 Google 授權端點

**成本低、價值高**：`es` 套件的 84.3% 覆蓋就是用 `httptest` 假伺服器
達成的，同一個手法可以直接套在 auth 上。

**驗收**：至少一個測試斷言 `return_path=https://evil.example` 被拒。

### 3.3 `httpapi`（9.5% → 40%）

51 個 handler。以**表格測試**為主（這個專案已在
`batch_export_test.go` 示範了六前綴的表格測試風格）。

優先序：

1. **25 處 `beginAdminTx` 的寫入路徑** —— 風險 × 爆炸半徑最高
2. 授權邊界（未登入打 `/api/admin/*` 一律被拒）
3. CSRF（`isTrustedOrigin` 有 **30 處**呼叫點，橫跨 9 個檔案，**零測試覆蓋**）
4. 公開讀取端點
5. 限流行為（429、IP 分桶、封鎖優先於限流）

**刻意不做**：不追求把 51 個 handler 全部覆蓋到高比例。目標 40% 是為了
讓**有風險的那部分**被守住，不是為了數字。

### 3.4 `files_server`（0% → 60%）

整個模組零測試，而它是**單一個 559 行的 `main.go`**，承載上傳、下載、
刪除、S3 路徑、token 驗證。

**為什麼在 Docker 化之前做**：容器化會改動訊號處理（Phase 0.3）與檔案
權限（非 root、`storage/` 掛 volume）。先測掉純邏輯，容器化時才不會同時
引入兩個變數。

先做 token 驗證與上傳限制（安全 + 資源耗盡），再做 S3 路徑（需要
`httptest` 假 S3 或 MinIO）。

### 3.5 `config`（0% → 80%）

`applyDefaults` / `IsAdminEmail` / `parseBool` / `parseList` /
`parsePositiveInt` / `parsePositiveSeconds` —— 全部是純函式，**最便宜的高
價值覆蓋**。

設定解析失敗的症狀是部署後才浮現的**靜默失效**：限流視窗設成 0 秒會讓
限流失效、TTL 兜底成 30 天、admin 白名單大小寫不匹配。每一個都有明確的
症狀，沒有一個會報錯。

### 3.6 `logger`（0% → 60%）

24 + 13 個未覆蓋函式。範圍是「INFO memory 解析」這類純邏輯
（`monitoring_test.go` 已覆蓋一部分），補齊成本低。

---

## Phase 4 — Docker 化

**6–8 人日。**

順序刻意是 **compose 先行、Dockerfile 殿後**：compose 首先是**測試基礎
設施**，其次才是部署產物。這個順序讓 Phase 3 的遷移測試與 files_server
測試有地方跑，也讓整堆的行為在開發機上可一鍵驗證。

**前提**：本機 Docker 未安裝（見「未驗證的前提」）。

### 4.1 `docker-compose.yml`（測試基礎設施優先）

服務：MySQL 8.0.29+、Redis、files_server、backend、選用 nginx。

**含 `healthcheck` 與 `depends_on: condition: service_healthy`** ——
這讓 `/healthz` 從「管理頁用的一個端點」變成「本機一鍵可驗證整堆健康」的
機制，同時是後端啟動順序（讀設定 → ping Redis → 開 MySQL → 遷移 →
`auth.Init` → session → 註冊路由 → 監聽）的自然表達。

Volume：MySQL 資料目錄、`files_server/storage/`（本機存儲）。

### 4.2 `stop_grace_period`（**必須寫對**）

| 項目 | 值 |
| --- | --- |
| compose `stop_grace_period` | **20s** |
| 後端 `SHUTDOWN_TIMEOUT_SECONDS` | 15s（預設） |
| Docker 預設 `--stop-timeout` | 10s ← **不夠** |

必須在 compose 寫死並加註解說明為什麼。README 有記這件事，但記錄不等於
防護 —— 這是 `README.md` 反覆出現的同一個型態：知道了，沒做到。

### 4.3 `backend -check` 模式

新增 `-check` 旗標：載入並驗證所有設定後 exit non-zero，不啟動任何服務。

**容器環境下價值最高**，因為容器正是靜默設定錯誤最常發生的地方：

- 兩份設定檔**格式不同** —— backend 是樸素的 `KEY=VALUE`，files_server
  是 TOML
- `MEDIA_TOKEN_KEY_PREFIX` 與 files_server 的 `token_key_prefix` **必須
  相同**，且都要與 session 的 `forum:session:` 前綴區隔
- 不一致的症狀是 **`README.md` 自承「很難診斷」**的那一個：上傳成功、
  貼文也存得下，但圖片一律 403/401
- `COOKIE_SECURE=false` 走 HTTPS → cookie 根本不會被送回
- `PUBLIC_BASE_URL` / `GOOGLE_REDIRECT_URL` / `TRUSTED_ORIGINS` 三者不對齊

**這類錯必須在部署時失敗，不該靠人診斷。** Compose 把它做成
`backend -check && backend` 的形式，或獨立的一次性執行。

### 4.4 `Dockerfile` ×3

前端需要多階段建置：`node` 階段跑 `npm ci && npm run build` 產出 `dist/`，
再交給 Go 階段，最後 runtime 階段只留執行檔。

**`.dockerignore` 必須排除 `node_modules`。** 本機那個目錄是數百 MB；
不排除的後果不只是 build context 爆炸，而是 `COPY . .` 會讓映像內安裝好的
依賴被本機版本覆蓋 —— 在 Windows 上還會帶進 symlink 問題。多階段建置
是唯一正解，不是最佳化。

同時排除：`dist/`（必須在映像內建置，不可從 build context 帶入）、
`*.exe`（`files_server` 有 `files_server.exe` 與 `files-server.exe` 兩個
殘留產物）、`config.conf`、`.git/`、`_code_statistics.csv`。

### 4.5 runtime 慣例

- **非 root** 執行
- `storage/` 掛 volume 而非寫進映像層
- `config.conf` **掛載**，不烘焙（設定檔含憑證，且 `FILES_SERVER_TOKEN`
  應該走環境變數）
- `dist/` 烘焙進映像（它被 `.gitignore` 掉，build context 裡沒有）
- **站名注入不需重建** —— `FORUM_NAME` 是送出頁面時注入 HTML 與
  manifest，所以改站名只需重啟，不需要 rebuild

### 4.6 `TRUSTED_PROXY_CIDRS`

Compose 下代理是 Docker 網路上的另一個容器，所以這個值**是可知的**
（該網段的 CIDR）。但近期不上線，就不必現在定死。

**維持 `legacy-headers` 現況**，並在 compose 裡釘一句註解說明：若加入反向
代理，必須設此值，否則限流、封鎖與稽核紀錄可被偽造的
`X-Forwarded-For` 繞過（README 的「來源 IP 的信任模型」章節有完整說明）。

### 4.7 保留 systemd 路徑

`backend/update.sh` 是既有的 Debian/Ubuntu 建置腳本。
**不要為了 Docker 刪掉它。** 換部署方式不該等於移除既有選項。

---

## Phase 5 — 觸發式備忘錄

**近期不上線 + 單容器 = 下面三件事現在都不會痛。記下觸發條件，不實作。**

記下來的理由不是怕忘，而是怕**未來有人在不知道代價的情況下重新發明它們**。

### 5.1 `/admin/stats` 每日彙總表

**觸發條件**：文章總量成長到讓「最近 N 天」不再是**有界**的掃描量。

`README.md` 已寫明正確做法：加一張每日彙總表，**不是**縮短視窗。
縮視窗會讓圖上看不出季節性的起伏，而那正是這一頁存在的理由。

### 5.2 稽核 facet 對照表

**觸發條件**：`/admin/log` 每次載入對整張稽核表做兩次無 `LIMIT` 的
`SELECT DISTINCT`（`AUDIT_RETENTION_DAYS` 預設 90 天），累積到明顯變慢。

症狀值得記下：失敗時降級成空清單，所以管理員看到的是**「篩選器突然少了
幾個選項」**，看起來像那些操作從來沒發生過。

替代方案按成本排序：記憶體快取（5 分鐘 TTL）／「先取前 N 筆再 distinct」
／`forum_admin_action_facets` 對照表。

### 5.3 跨行程限流

**觸發條件**：backend 容器數 > 1。

那時的實際額度是 `limit × N`，而輪到哪一台不是使用者能控制的。
三條路與其代價：

| 選項 | 代價 |
| --- | --- |
| Redis 滑動視窗 | 每個寫入請求 +1 次網路往返（這個專案刻意不做，見 `ratelimit.go` 檔頭） |
| 交給 LB 層限流 | 卸載，但與應用的 IP 信任判定脫鉤，會出現兩套「什麼是這個 IP」 |
| 維持現狀 + 文件化 | 多執行個體下實際限流是 `limit × N`，必須寫進 README |

**記下 `limit × N` 這個數字** —— 它是將來做這個決定時唯一需要的輸入。

---

## Phase 6 — 功能

**明確擱置。**

在 Phase 2 完成前加功能 = 擴大一個沒有任何工具會檢查的面。理由回到本
路線圖的核心診斷：這個專案的不變條件目前靠紀律維持，而紀律的上限就是
「有人記得看」。

---

## 刻意不做的事

這些看起來像待辦事項，但每一個都有理由。

- **不引入 web framework 或 ORM。** 現況是刻意且合理的，Phase 0–4 全部
  可以在 Go 標準函式庫內完成。引入框架不會讓上面任何一項變容易。
- **不追求覆蓋率數字本身。** 18.7% → 40% 有意義；40% → 70% 若靠無行為
  斷言的測試達成，那是負債 —— 它會在重構時壞掉，卻沒指出任何東西壞了。
  Phase 3 已按風險排序，不按數字。
- **不刪除 `forum_request_metrics` 的多執行個體累加語意。** 單容器下它是
  過剩的，但完全無害，而刪掉它會讓將來擴充時要重新設計。
- **不為多執行個體重寫監控統計。** `README.md` 已記錄這是刻意的：每 IP
  統計刻意不落盤，因為「某個位址在什麼時候打了本站」是一筆個人資料。
  若真的需要長期來源統計，那是**存取日誌的工作**，不是從監控頁擴張出去。
- **不新增刪除稽核紀錄的 API。** 專案現有設計的理由成立：一個能刪掉自己
  紀錄的稽核日誌等於沒有稽核日誌。
- **不為了 Docker 刪掉 systemd 路徑。**
- **不讓 Phase 0 的項目合併成一個大變更。** 七項各自獨立、都是低風險的
  修正，合成一個 diff 會讓 review 變難，而 review 正是這些修改存在的原因。

---

## 未驗證的前提

誠實記錄，避免下次 deploy 時踩到 —— 沿用 `README.md`「已知問題」的寫法。

- **Phase 4 的容器產物在本機無法驗證。** Docker 未安裝於開發機
  （2026-10-02 量測）。Dockerfile / compose / `.dockerignore` 會寫成可
  review 的獨立產物，但**構建驗證需要 Docker Desktop**。Phase 0–3 完全
  不需要 Docker，Phase 3.1 現在就能開始。
- **資料庫的量體是未知的。** 本路線圖的排序假設這個站的資料量還不需要
  Phase 5 的任何一項。若資料量已經很大，Phase 5.1 與 5.2 要提前。
- **race detector 的結論會是新的。** 目前「無資料競爭」是靜態審查結論
  （`CODE_REVIEW.md` 明寫未經實證）。Phase 1.2 第一次真正跑 `-race` 時，
  **可能會找到真的資料競爭**。那不是 Phase 1 失敗，是 Phase 1 開始有
  用 —— 預先知道這件事，才不會在 CI 第一次紅掉時誤判成環境問題。
- **多語系數量已修正，但「文件與實作漂移」沒有被根治。** Phase 0.7 只是
  修掉查得到的那一處。根治手段是 Phase 1.4 把 `verify-catalogs.mjs` 納入
  CI，讓可自動檢查的部分不再漂。
