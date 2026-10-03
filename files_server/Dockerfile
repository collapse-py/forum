# syntax=docker/dockerfile:1
#
# files_server 的容器映像。
#
# 【這個映像是什麼】
# 一個只有靜態執行檔的檔案服務：接受帶 token 的上傳、刪除檔案、依 Redis 裡的
# 媒體 token 放行讀取。它不碰 MySQL（見 README 部署章節的理由）。
#
# 【與 backend 映像的三個差異，以及每一個的原因】
#   1. 執行階段用 alpine，不用 distroless。
#      原因：這個服務**需要**一個 shell —— 它要處理使用者上傳的檔名，而那個檔名
#      是不可信輸入。見下方「已知限制」：目前沒有處理它的 shell 指令，但一旦
#      需要（例如除錯時想 `ls` 看一眼儲存目錄、或需要用手動工具清掉半成品檔案），
#      沒有 shell 就得另外做一個 sidecar。把 shell 留著的成本（多 7 MB、多一層
#      CVE 曝露面）低於「下次需要時要重做映像」的成本。
#      明確地說：這是為了可診斷性而接受的安全性妥協，而不是疏忽。
#
#   2. 以非 root 執行，且儲存目錄需要一個可寫的 volume。
#      原因：容器內必須有人寫檔案。用 UID 65534（nobody）寫 volume 會在 Linux 上
#      遇到「目錄屬於 root」的問題，因此 compose 的 volume 是以 named volume
#      存在的 —— named volume 的初次建立會繼承映像裡那個目錄的權限。
#
#   3. 沒有 HEALTHCHECK，healthcheck 用執行檔自己。
#      原因同 backend（distroless 沒有 curl）；但這個服務連一個不需要 shell 的
#      探測都沒有 —— 它沒有 /healthz 端點。因此 compose 對這個服務用
#      「程序還在跑就算健康」的弱檢查，理由寫在 compose 的 healthcheck 註解裡。
#
# 【已知限制】
#   - 上傳檔名是使用者可控的輸入，而這個映像裡沒有任何工具能處理它。
#     目前的緩解是應用層：server.go 的 handleUpload 以 filepath.Base 正規化檔名，
#     而實際存檔用的是 newFileName 產生的 UUID（見 storage.go）。因此「檔名不可信」
#     這件事在**目前的**實作下不是問題；但若將來真的需要在 shell 裡處理上傳的檔名，
#     必須先把檔名過一遍這個程式的 UUID 產出流程，不能直接用使用者的字串。
#   - 沒有 HEALTHCHECK 指令的原因是它沒有可探測的端點。加一個 /healthz 會讓
#     「服務活著」與「Redis 可用」綁在一起，而那兩件事在這個服務裡刻意是分開的
#     （Redis 掛掉時它會降級為不驗證媒體 token 而不是停止服務）。

# ---------------------------------------------------------------------------
# 建置階段
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 讓產物是靜態執行檔，可以放在 alpine 的 musl 上執行。
# -trimpath 的理由與 backend 相同：logger 的 runtime.Caller 結果會寫進日誌，
# 而沒有它兩台機器會寫出不同的檔名。
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/files_server \
    .

# ---------------------------------------------------------------------------
# 執行階段
# ---------------------------------------------------------------------------
FROM alpine:3.21

# ca-certificates：上傳時後端會呼叫檔案伺服器（HTTP，但內網也可能是 HTTPS）；
#   這個服務自己**不**對外連 HTTPS（除了 S3 模式下對 S3 的請求），
#   但保留 CA 憑證讓 S3 用一個自簽憑證的環境有調整的空間。
# tzdata：與 backend 同一個理由（container.log 的時間戳）。
# coreutils 刻意不要：alpine 內建的 busybox 已足夠（ls、rm、date）。
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/files_server /app/files_server

# 設定檔掛載點。與 backend 同一個理由：沒有這個目錄，`docker run`（不經過
# compose）會因為工作目錄不存在而失敗，而症狀是「找不到 config.conf」。
RUN mkdir -p /app/config /app/storage

# 預設使用者。刻意建立一個具名使用者（而不是直接用 nobody）：日誌與除錯時
# 「這個檔案是誰寫的」需要一個穩定的名字。
RUN adduser -D -H -u 10001 files && chown -R files:files /app

USER files

ENV TZ=Asia/Taipei

EXPOSE 7070

# storage 是 volume 掛載點。它**必須**是 volume（named volume 或 bind mount），
# 否則容器每次重建都會失去所有上傳的檔案 —— 而症狀是「貼文裡的圖片全都 404」，
# 不會有任何錯誤訊息。
VOLUME ["/app/storage"]

# 刻意不設 HEALTHCHECK：這個服務沒有可探測的端點，而加一個會把
# 「Redis 掛掉時的降級」變成「容器被判定為不健康並重啟」—— 那與這個服務
# 刻意採取的「Redis 不可用時繼續提供上傳」策略直接衝突。
# compose 那邊用「程序還在跑就算健康」的弱檢查，理由寫在那裡。

CMD ["/app/files_server"]