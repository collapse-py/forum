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
cd ../backend && go run ./tools/coveragefloor <合併後的 coverage.out> 25.0
```

它刻意是**獨立的 Go 模組**：它需要 `golang.org/x/tools`，而 backend 的相依項刻意維持在四個（miniredis、mysql、go-redis、oauth2）。把分析工具的相依塞進 runtime 相依裡，等於讓每個部署環境都多下載一份只有 CI 需要的程式碼。

第四條跨越 Go / TypeScript / HTML 三種語言，`go/analysis` 表達不了，因此它是**一個測試**而不是 analyzer。兩種失敗模式（編譯失敗 vs 測試失敗）刻意不混在同一個工具裡，以免搞混哪一條失效了。

---

## CI

`.github/workflows/ci.yml` 有七個 job：

| job | 做什麼 |
| --- | --- |
| `backend` | build、`go vet`、`go test -race`（CGO 開啟就是為了它） |
| `files_server` | 同上 |
| `frontend` | `npm ci`、`typecheck`、`build`、`verify-catalogs` |
| `coverage-floor` | 合併兩個模組的覆蓋率，檢查是否 ≥ **25.0** |
| `invariants` | 建置並執行分析器，加上分析器自測 |
| `migrations` | 起真實 MySQL 8.0.29，跑 `forum/data` 的遷移測試 |
| `secrets` | 確認 `config.conf` 從未被追蹤、範本檔不含明文憑證 |

兩件事值得知道：

- `secrets` 用 `fetch-depth: 0` 掃**整份歷史** —— 只掃工作區的話，一個「刪掉檔案」的 commit 就足以讓真正的憑證留在倉庫裡。
- **覆蓋率地板的作用是防止退化，不是製造動機。** 刻意不設一個虛高的目標（例如 70%），因為那會誘發無行為斷言的測試來拉數字，而那種測試是負債：它會在重構時壞掉，卻沒有指出任何東西壞了。目前合併覆盖率約 **26.2%**；`forum/httpapi` 約 2,500 行、覆蓋率最低，是主要缺口。

---

## 靜態檢查與格式

這個專案**刻意沒有** gofmt / golangci-lint 的閘門。理由不是懶，而是它們會在第一次執行時擋下 25 個既有檔案，而那些格式差異是**行尾（CRLF）**造成的 —— 在一個以 Windows 為主的開發環境裡，那會讓每一個開發者第一次推上去時都被擋，而他們完全無法理解原因。

`go vet` 有閘門（它找的是真問題，不是排版）。
