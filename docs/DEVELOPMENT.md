# 開發與測試

怎麼跑測試、專屬的不變條件 analyzer、CI 跑什麼。

- [測試](#測試)
- [不變條件 analyzer](#不變條件-analyzer)
- [CI](#ci)
- [靜態檢查與格式](#靜態檢查與格式)

---

## 測試

後端與檔案服務都有 Go 測試，**前端沒有單元測試** —— 前端的測試是**型別檢查**加上編譯期保證的翻譯完整性（`Record<MessageKey, string>` 讓漏翻譯與用錯鍵在編譯期就失敗）。

```bash
# 後端
cd backend
go test ./...                # 全域
go test ./forum/config/...   # 設定解析（限流兜底、管理員白名單正規化、TTL 守門）
go test ./forum/auth/...     # OAuth（開放轉向防護、state 往返、四種上游失敗）
go test ./forum/logger/...   # 欄位清洗（含日誌注入）、兩種格式、分級、併行安全
go test ./forum/session/...  # Session 列舉與撤銷（miniredis）
go test ./forum/ipban/...    # IP 封鎖名單（miniredis）
go test ./forum/audit/...    # 稽核（CSV 防護、佔位符、截斷）
go test ./forum/es/...       # ES 傳輸層（httptest 假伺服器）
go test ./forum/httpapi/...  # HTTP 層（用假的資料庫替身，不需要 MySQL）
go test .                    # 監聽逾時、優雅停止、-check 報告（package main）

# 檔案服務（上傳／刪除／token 驗證／路徑穿越）
cd files_server && go test ./...

# 需要真實 MySQL 的遷移測試（未設定 DSN 時會自動跳過）
FORUM_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/?parseTime=true&loc=UTC&multiStatements=true' \
  go test ./backend/forum/data/...

# 翻譯目錄與 README 語系宣稱（可從任何目錄執行）
node tools/i18n/verify-catalogs.mjs
```

`verify-catalogs.mjs` 同時比對**README 宣稱的語系數量與清單** —— 新增或刪除語系而沒更新 README 那兩處數字時，它會指名 README 的哪一處過期（CI 會跑這一步）。

### 沒有 MySQL 怎麼測 handler

`forum/httpapi` 有一個**假的資料庫**（`cov_harness_test.go`）：它是一個註冊在
`database/sql/driver` 上的極簡驅動，查詢預設回空結果集、寫入預設成功、交易
可以開也可以結束。要讓某一條查詢回資料時，用 `h.rows("FROM forum_posts", 欄位,
列)` 登記結果集即可（欄位順序必須與 handler 的 `Scan` 對應）。

它與 `go-sqlmock` 的差別是**寬鬆**：不要求每一筆查詢都被預期，也不檢查順序。
因此 handler 只要走到某條路徑，就不會因為「少登記一筆查詢」而失敗；代價是它
不適合用來斷言 SQL 形狀 —— 那種斷言請用 `go-sqlmock` 或既有的 `*_mysql_test.go`。

`package main` 那一側有同樣的思路：啟動流程（OAuth → Session → Server → 背景
工作）抽成 `startServices` 之後，可以在同一個行程裡用替身（假的 `*sql.DB`、
miniredis）直接執行，不需要真的監聽任何連接埠。

---

## 不變條件 analyzer

這個專案有四條**專屬**的不變條件。它們約束的不是「型別對不對」或「有沒有測試」，因此通用工具抓不到 —— 一個 handler 從 0% 變成 30% 覆蓋，與「`Commit` 之前有沒有檢查稽核錯誤」毫無關係。

| 不變條件 | 由什麼守住 |
| --- | --- |
| 稽核與操作同生共死：`beginAdminTx` → **檢查** `recordAdminAction` 的錯誤 → `tx.Commit()` | `auditcheck` analyzer |
| `/api/admin/*` 的 handler 在 method 分派**之前**呼叫 `requireAdminForum` | `adminauth` analyzer |
| `withBlocklistHandler` 只在 `applyRateLimit` 內被呼叫（單一掛載點） | `blocklistmount` analyzer |
| 十七個 HTML 殼在三處同步 | `frontend_shells_test.go` |

前三條是 `tools/invariants` 這個獨立 Go 模組裡的 `go/analysis` analyzer。**失敗模式是編譯失敗**（`go vet` 回報），而不是「有人沒注意到」。

```bash
cd tools/invariants && go build -o invariants .
cd ../../backend && go vet -vettool=<abs>/tools/invariants/invariants ./...
cd ../tools/invariants && go test ./...   # 含負向測試：刻意寫違規程式碼，確認會被報

# 覆蓋率地板（backend/tools/coveragefloor）
cd ../backend && go run ./tools/coveragefloor <合併後的 coverage.out> 75.0
cd ../backend && go run ./tools/coveragefloor <單一模組的 coverage.out> 75.0 --per-package
```

它刻意是**獨立的 Go 模組**：它需要 `golang.org/x/tools`，而 backend 的相依項刻意維持在四個（miniredis、mysql、go-redis、oauth2）。把分析工具的相依塞進 runtime 相依裡，等於讓每個部署環境都多下載一份只有 CI 需要的程式碼。

第四條跨越 Go / TypeScript / HTML 三種語言，`go/analysis` 表達不了，因此它是**一個測試**而不是 analyzer。兩種失敗模式（編譯失敗 vs 測試失敗）刻意不混在同一個工具裡，以免搞混哪一條失效了。

---

## CI

`.github/workflows/ci.yml` 有七個 job：

| job | 做什麼 |
| --- | --- |
| `backend` | build、`go vet`、`go test -race`（CGO 開啟就是為了它）、逐套件覆蓋率地板 |
| `files_server` | 同上 |
| `frontend` | `npm ci`、`typecheck`、`build`、`verify-catalogs` |
| `coverage-floor` | 合併兩個模組的覆蓋率，檢查是否 ≥ **75.0** |
| `invariants` | 建置並執行分析器，加上分析器自測 |
| `migrations` | 起真實 MySQL 8.0.29，跑 `forum/data` 的遷移測試 |
| `secrets` | 確認 `config.conf` 從未被追蹤、範本檔的憑證鍵仍是佔位符（附反向自我測試） |

兩件事值得知道：

- `secrets` 用 `fetch-depth: 0` 掃**整份歷史** —— 只掃工作區的話，一個「刪掉檔案」的 commit 就足以讓真正的憑證留在倉庫裡。
- **覆蓋率地板的作用是防止退化，不是製造動機。** 地板是 **75.0**，而且是**逐套件**量的：每個套件都要 ≥ 75.0%，`backend` 與 `files_server` 各自的 job 各量一次，`coverage-floor` job 另外量合併後的總數。兩道閘門擋的不是同一種退化 —— 總數看不見「某個套件被掏空」，而那正是會出事的地方。

  目前實測（2026-10-21，本機 `go test ./...`，不含真實 MySQL）：

  | 套件 | 覆蓋率 | 套件 | 覆蓋率 |
  | --- | --- | --- | --- |
  | `forum/forum/data` | 98.6% | `forum/forum/config` | 89.0% |
  | `forum/forum/audit` | 97.0% | `forum/forum/session` | 88.3% |
  | `forum/forum/purge` | 93.5% | `forum/forum/httpapi` | 87.0% |
  | `forum/forum/ipban` | 92.9% | `forum`（main） | 78.5% |
  | `forum/forum/es` | 92.8% | `forum/forum/metrics` | 76.9% |
  | `forum/forum/logger` | 91.1% | `forum/tools/coveragefloor` | 98.3% |
  | `forum/forum/auth` | 92.0% | `files_server` | 91.6% |

  backend 合計 **87.4%**、files_server **91.6%**、合併 **87.7%**（`forum/forum/logcaller` 沒有敘述，不列入）。

---

## 靜態檢查與格式

這個專案**刻意沒有** gofmt / golangci-lint 的閘門。理由不是懶，而是它們會在第一次執行時擋下 25 個既有檔案，而那些格式差異是**行尾（CRLF）**造成的 —— 在一個以 Windows 為主的開發環境裡，那會讓每一個開發者第一次推上去時都被擋，而他們完全無法理解原因。

`go vet` 有閘門（它找的是真問題，不是排版）。
