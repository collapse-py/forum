# 前端

前端是 Vite + React 19 + TypeScript 的**多頁應用**（MPA，不是 SPA），同時是 PWA。

- [頁面路由](#頁面路由)
- [開發](#開發)
- [批次翻譯](#批次翻譯)
- [新增一個頁面要動三處](#新增一個頁面要動三處)

---

## 頁面路由

後端把多個 HTML 檔直接送給瀏覽器（這是 MPA 不是 SPA 的原因）。**17 個頁面殼**，對應 `src/entries/` 的 17 個 entry 與 `vite.config.ts` 的 `rollupOptions.input`。

| Path | HTML 檔 | 需登入 |
| --- | --- | --- |
| `/` `/index.html` | — | 302 → `/forum` |
| `/forum` | `forum.html` | 否 |
| `/forum/new` | `forum-new.html` | 否（送出時需登入） |
| `/forum/login` | `forum-login.html` | 否 |
| `/forum/profile` | `forum-profile.html` | 是 |
| `/forum/others-profile` | `forum-others-profile.html` | 是 |
| `/forum/following` | `forum-following.html` | 是 |
| `/forum/post/{id}` | `forum-post.html` | 否（互動時需登入） |
| `/admin` | `admin.html` | 管理員 |
| `/admin/forum` | `forum-admin.html` | 管理員 |
| `/admin/forum-report` | `forum-report.html` | 管理員 |
| `/admin/monitor` | `forum-monitor.html` | 管理員 |
| `/admin/log` | `audit-log.html` | 管理員 |
| `/admin/stats` | `forum-stats.html` | 管理員 |
| `/admin/export` | `export.html` | 管理員 |
| `/admin/sessions` | `sessions.html` | 管理員 |
| `/admin/blocks` | `blocks.html` | 管理員 |
| `/admin/announcements` | `announcements.html` | 管理員 |

另有 `/service-worker.js`（附 `Service-Worker-Allowed: /`）、`/forum-manifest.json`、`/assets/*`（內容雜湊，cache 一年）、`/asset/*`（PWA 圖示等原始檔，後端以 `http.Dir` 直接提供 `frontend/asset/`）。

站名會在送出頁面時注入 HTML 與 manifest，因此改 `FORUM_NAME` 不需要重新建置前端（見 [`CONFIGURATION.md`](CONFIGURATION.md#站名)）。

---

## 開發

```bash
cd frontend
npm install
npm run dev          # Vite dev server（預設 5173）
npm run typecheck    # tsc x2：主程式 + service worker
npm run build        # typecheck + vite build → dist/
npm run preview      # 預覽 dist/
```

`npm run build` = `tsc -p tsconfig.json && tsc -p tsconfig.sw.json && vite build`。沒有 `test` / `lint` 腳本；型別檢查（`strict` + `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes`）就是這個專案的 lint。

`vite.config.ts` 有三個自訂行為：

- **`copyForumRuntimeAssets`**（`enforce: 'post'`）：用 esbuild 編譯 `service-worker.ts` 成 IIFE，輸出**不帶 hash** 的 `/service-worker.js` 與 `/forum-manifest.json`（service worker 必須如此），並把 HTML 裡的 manifest 連結改回未雜湊版，讓後端可以在送出時替換 `{{FORUM_NAME}}`。
- **`vite-plugin-javascript-obfuscator`**（只在 `build`）：字串陣列旋轉 + control flow flattening。`debugProtection` / `selfDefending` 維持關閉，因為它們需要 `eval`，而 CSP 是 `script-src 'self'`。
- **沒有設定 `server.proxy`**，見下。

### `npm run dev` 開箱即壞

前端所有 `fetch` 都是同源相對路徑（`/api/...`），但沒有 proxy，所以 dev server 起在 5173 時請求會打到 Vite 而不是 8088 的後端。要用 dev server + HMR 開發，先補上：

```ts
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8088',
    '/auth': 'http://localhost:8088',
    '/healthz': 'http://localhost:8088',
  },
},
```

不改的話就照正式模式走：`npm run build` 產出 `dist/`，再啟動後端由它提供靜態檔案。

---

## 批次翻譯

`tools/i18n/` 是一組 Node ESM 腳本，負責批次翻譯的切單與合併；代理模型只負責中間那一段（讀 TSV、寫 TSV）。因此**翻譯品質的驗證不依賴代理模型自報** —— 鍵集、佔位符、未翻譯殘留、字元損毀都由腳本把關。流程與設計理由見 [`tools/i18n/README.md`](../tools/i18n/README.md)。

```bash
cd frontend    # 全部腳本必須從 frontend/ 執行

# 1. 切工作單：messages.ts → key<TAB>原文，切成每份約 96 行
node ../../tools/i18n/make-worklist.mjs <輸出目錄> <語言> [份數，預設 4]

# 2. 交給代理模型，一個任務一份：讀 <語言>-N.tsv，寫 <語言>-N.out.tsv

# 3. 合併：鍵集 / 佔位符 / 未翻譯殘留都過關才會寫出 TS 目錄檔
node ../../tools/i18n/merge-translations.mjs <輸出目錄> <語言> <匯出名稱> <檔名>
#    沒收齊會輸出 <語言>-todo.tsv，遞迴回第 1 或第 2 步
node ../../tools/i18n/split-todo.mjs <輸出目錄> <語言> [份數]

# 隨時驗證全部語系目錄（可從任何目錄執行）
node tools/i18n/verify-catalogs.mjs
```

合併腳本的四種失敗與處理：

| 失敗 | 檢查 | 處理 |
| --- | --- | --- |
| 鍵集不符 | 與 `messages.ts` 的鍵集與順序比對（缺鍵、多餘鍵、順序不同） | 寫出前擋下：缺鍵補譯、多餘鍵刪除 |
| 佔位符不符 | `{name}` 集合與原文比對 | **不修復只報錤** —— 猜著補上會讓一條翻錯的譯文看起來通過驗證 |
| 未翻譯殘留 | 非中文／日文語系出現漢字 | 判定為未翻譯，必須補譯（中文與日文本身以漢字書寫，對它們只會產生噪音，因此排除） |
| 字元損毀（U+FFFD） | 掃 REPLACEMENT CHARACTER | **不放行**：見下 |

現有語系（`frontend/src/i18n/translations/`）的完整清單與數量宣稱在 README 的功能列表，由 `verify-catalogs.mjs` 對照目錄實際內容驗證（新增或刪除語系而沒更新 README 時，CI 會指名哪一處過期）。

**U+FFFD（REPLACEMENT CHARACTER）那一條與其他三條性質不同**：它不是任何一種語言的字，出現在目錄裡只代表某個字元在一次編碼往返中被解碼失敗並換掉 —— **原始位元組當場就丟了**，所以它既不能被翻譯修好，也不會因為換語系而消失。佔位符、鍵集、順序在這種情況下全部正常，因此沒有其他檢查抓得到。修法是照 `messages.ts` 的原文重寫整句，而不是去猜那個丟掉的字元。

---

## 新增一個頁面要動三處

`src/entries/` · `vite.config.ts` 的 `rollupOptions.input` · 後端 `httpapi.frontendShellFiles`（CSP 的 `<style>` SHA-256 授權）。**少一處的症狀不是錯誤訊息，而是整頁沒有版面** —— `style.css` 仍會作用（它是 `<link>` 載入的外部檔案），所以畫面看起來「有樣式但怪怪的」，瀏覽器主控台只有一行 `Refused to apply inline style`。`frontend_shells_test.go` 會指名缺哪一處。
