# syntax=docker/dockerfile:1
#
# 這個專案**唯一的**服務映像：後端 + 前端產物。
#
# 【為什麼是一個映像而不是兩個】
# 後端在啟動時會掃描前端產物、算出每個 HTML 殼裡 <style> 區塊的 SHA-256，
# 然後把它們放進 Content-Security-Policy 的 style-src
# （見 backend/forum/httpapi/securityheaders.go 的 inlineStyleHashes）。
#
# 也就是說：**被提供的檔案與被授權的雜湊必須來自同一份建置輸出**。它們分開的
# 症狀是「整頁沒有版面」—— style.css 仍會作用（它是被 <link> 載入的外部檔案，
# style-src 'self' 已經涵蓋），所以畫面看起來「有樣式但怪怪的」，
# 而瀏覽器主控台只有一行 Refused to apply inline style。
#
# 分成兩個映像時這個不變量要靠「部署者記得用對版本的 dist/」來維持。
# 合成一個映像時它由建置流程**結構性**保證：不可能提供一份與雜湊不符的檔案，
# 因為兩者來自同一個 stage 的輸出。
#
# 另一個好處（ROADMAP.md Phase 4.5 的「runtime 慣例」）：部署時需要啟動的
# 容器只有兩個 —— 這個與 files_server。沒有第三個「前端」服務，也就不會有
# 「啟動了 nginx 但它指向後端，而後端還沒起來」這個啟動順序問題。
#
# 【建置情境是倉庫根目錄】
#   docker build -f backend/Dockerfile .
# 這是必要的：前端的原始碼在 frontend/，而 Go 的原始碼在 backend/，
# 兩者在同一個映像裡就必須共享一個建置情境。backend/.dockerignore 因此**不是**
# 這個建置流程使用的檔案 —— 那一份是給「在本機 `go build`」時的心智模型用的，
# 真正的 .dockerignore 必須在**建置情境的根**（也就是倉庫根），見根目錄的
# .dockerignore。
#
# 【為什麼 CGO_ENABLED=0】
# 這個專案用 database/sql + go-sql-driver/mysql（純 Go）與 go-redis（純 Go），
# 沒有任何需要 cgo 的相依。關掉 cgo 讓產物不依賴動態連結器版本 ——
# 這在 distroless 上是唯一可能，而且少一層 libc 的 CVE 曝露面。
#
# 【為什麼用 distroless 而不是 alpine】
# 執行階段不需要 shell（沒有「進去看一眼」是刻意的取捨）、不需要套件管理員。
# distroless 只剩執行檔 + CA 憑證，約 20 MB，而 alpine 是 7 MB 但有 shell、
# 有 apk、有 libc —— 而一個有 shell 的正式環境容器，距離「有人用 exec 進去手動
# 修東西」只有一步。
#
# 代價（唯一一個值得知道的）：debugging 不能靠 exec 進去看檔案。診斷方式是把
# LOG_FORMAT 設成 json 從 stdout 收集 —— 那也是 compose 的預設。
#
# 【TZ 的處理】
# 日誌時間戳取 time.Now()（本機時區），而容器預設是 UTC。只設定 TZ 環境變數而
# 沒有 tzdata 與 /etc/localtime 的話，那個變數完全沒有效果 —— 而沒有人會去看
# 容器的環境是什麼。因此這裡兩者都處理。

# ---------------------------------------------------------------------------
# 階段 1：建置前端產物
# ---------------------------------------------------------------------------
FROM node:22-alpine AS frontend

WORKDIR /src

# 先只複製 lock 相關的檔案：只要 package-lock.json 沒變，修改任何 .tsx
# 都不會重跑 npm ci —— 而那是整個建置裡最慢的一步。
COPY frontend/package.json frontend/package-lock.json ./

# npm ci 而不是 npm install：前者依 lock 建置，且在 lock 與 package.json 不同步
# 時直接失敗。後者會靜默更新 lock，讓「建置結果與上次不同」沒有任何訊號。
RUN npm ci

COPY frontend/ ./

# typecheck 獨立跑一次，而不是只看 build 會不會失敗。
#
# 為什麼需要：`Record<MessageKey, string>` 這組型別讓「某個語系漏了一個鍵」與
# 「某個元件用了不存在的鍵」**在編譯期**就失敗。這是這個專案最好的設計之一，
# 因此它不該被「Vite 剛好也會失敗」順帶覆蓋 —— 一旦有人改了 build 設定讓它
# 不再做型別檢查，漏翻譯就會變成一個只在使用者看到空白文字時才發現的問題。
RUN npm run typecheck

RUN npm run build

# ---------------------------------------------------------------------------
# 階段 2：建置後端執行檔
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS backend-builder

# git 與 ca-certificates：go mod download 需要它們驗證 https 的 module proxy。
# build-base（gcc）刻意**不要**：CGO_ENABLED=0 不需要編譯器，少一個相依就少
# 一層需要更新的東西。
RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# -trimpath：讓 panic traceback 與 runtime.Caller 的路徑不含建置機的路徑。
#   這一點對這個專案特別重要，因為 logger 把 runtime.Caller 的結果寫進每一條
#   日誌 —— 沒有 -trimpath 的話，兩台機器建出的同一個版本會寫出不同的檔名，
#   而「日誌裡的路徑」正是維運拿來對照程式碼的線索。
# -s -w：去掉符號表與 DWARF 資料，縮小執行檔（不影響 panic traceback 的可讀性）。
#
# 刻意不加 -race：那會讓執行檔大三倍且慢十倍，而 race 的用途是 CI
# （見 .github/workflows/ci.yml 的 backend job），不是正式環境。
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/forum \
    .

# ---------------------------------------------------------------------------
# 階段 3：執行
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

# CA 憑證是**必要**的：這個行程要對外連 Google（OAuth token 與 userinfo）與
# Elasticsearch，而兩者都是 HTTPS。沒有 CA 憑證時症狀是
# "x509: certificate signed by unknown authority" —— 那個錯誤完全不會指向
# 「你的映像缺了 CA 憑證」。
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# tzdata：讓 TZ 環境變數真的生效（見檔頭）。
# 貼成一個 symlink 而不複製整個 /usr/share/zoneinfo：只需要一個時區，
# 而整個 zoneinfo 有 3 MB 多。
COPY --from=backend-builder /usr/share/zoneinfo/UTC /usr/share/zoneinfo/UTC

WORKDIR /app

COPY --from=backend-builder /out/forum /app/forum

# 前端產物。**從 frontend stage 拿，而不是從 build context 拿** —— 這是
# 「被提供的檔案與被授權的雜湊來自同一份建置輸出」的實作。
COPY --from=frontend /src/dist /app/frontend/dist

# PWA 圖示與 manifest 是一手來源（不是 Vite 的產物），後端以 http.Dir 直接提供
# 它們。若 dist/ 裡沒有它們，容器的 PWA 安裝會拿到 404，而症狀是
# 「安裝了圖示但打開是空白」。
COPY backend/frontend/asset /app/frontend/asset

# 設定檔掛載點。刻意建立這個空目錄（而不是只靠 compose 的 volume）——
# 沒有它，`docker run`（不經過 compose）會因為工作目錄不存在而失敗，
# 而症狀是「找不到 config/config.conf」，那不會讓人想到要先建立目錄。
RUN mkdir -p /app/config /app/logs

# 非 root 執行。這個專案不需要任何 root 權限：它不對外 listen 80、不寫
# /etc、不建立系統使用者。以 root 執行的唯一好處是「除錯方便」，那不值得。
USER nonroot:nonroot

ENV TZ=Asia/Taipei

EXPOSE 8088

# healthcheck 呼叫執行檔自己的 -healthz。
#
# 為什麼不寫成 curl -f http://localhost:8088/healthz：distroless 沒有 curl。
# 替代方案是「不設 healthcheck」，那則會讓 compose 無法用 depends_on 的
# service_healthy 條件等待後端就緒 —— 而那正是「檔案伺服器在後端之前啟動、
# 第一次上傳失敗」這個競態的來源。
HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
    CMD ["/app/forum", "-healthz"]

# 預設就是正常啟動；-check 與 -healthz 是同一支執行檔的另外兩種模式
# （見 backend/main.go）。刻意不用 shell 形式的 CMD：那需要 /bin/sh，
# 而 distroless 沒有。
CMD ["/app/forum"]