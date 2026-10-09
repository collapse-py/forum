# 部署

正式環境怎麼部署。維運細節（停止行為、systemd、常見狀況）見 [`OPERATIONS.md`](OPERATIONS.md)，設定參數見 [`CONFIGURATION.md`](CONFIGURATION.md)，上線前安全檢查見 [`SECURITY.md`](SECURITY.md#上線檢查表)。

- [部署前檢查表](#部署前檢查表)
- [建置](#建置)
- [容器部署（docker compose）](#容器部署docker-compose)

---

## 部署前檢查表

```mermaid
flowchart LR
  C[瀏覽器] -->|HTTPS| P[反向代理<br/>TLS 終結]
  P --> F[/forum /admin → :8088]
  P --> FS[/files/* → :7070]
  F --> DB[(MySQL)]
  F --> R[(Redis)]
  F -.選用.-> ES[(Elasticsearch)]
  F -->|上傳 + token| FS
  FS --> R
  FS --> S[(本機磁碟 或 S3)]
```

反向代理不在此 repo 內。務必確認：

1. **`COOKIE_SECURE=true`**，並且代理有處理 HTTPS。
2. **`PUBLIC_BASE_URL`、`GOOGLE_REDIRECT_URL` 與 `TRUSTED_ORIGINS` 對齊**實際對外網址。
3. **`FILES_SERVER_PUBLIC_URL` 指向對外可讀的位址**；`FILES_SERVER_URL` 可以是內部位址。
   兩者相同代表沒有內外網分離。
4. **Google Cloud Console 的授權 redirect URI** 與 `GOOGLE_REDIRECT_URL` 完全一致。
5. **代理必須放行 `Service-Worker-Allowed` 與 `/service-worker.js`**，否則 PWA 不會更新。
6. `/files/*` 若走代理並由瀏覽器直接取圖，代理**不得**吃掉 `?token=` 查詢參數。
7. **Redis 只能綁內網** —— 它是本站的信任根（見 [SECURITY](SECURITY.md)）。
8. **`files_server` 的 `upload.token` 必須設定。** 留空是程式明確允許的狀態（本機測試
   用），而那等於「任何找得到這個埠的人都能上傳與刪除檔案」—— 沒有任何錯誤訊息會
   提醒你。這個服務的 `/delete` 也接受 `.` 與 `..` 以外的所有檔名（見
   `files_server/storage.go` 的 deleteFile）。
9. **停止訊號要送得對，而且要給夠時間。** 後端收到 `SIGTERM` / `SIGINT` 後會排空在途
   請求，上限是 `SHUTDOWN_TIMEOUT_SECONDS`（預設 15 秒）。部署環境的停止上限必須
   **大於**它：

   | 環境 | 預設停止上限 | 怎麼調 |
   | --- | --- | --- |
   | systemd | `TimeoutStopSec=90s` | 通常已足夠 |
   | Docker | `--stop-timeout=10` | `docker run --stop-timeout=20 …` 或 compose 的 `stop_grace_period` |
   | Kubernetes | `terminationGracePeriodSeconds=30` | 部署 manifest 內設定 |
   | 裸執行 | — | 會送出 `SIGINT`，Ctrl-C 即可 |

   另一個方向是讓反向代理**先**把流量抽掉再轉送停止訊號，那樣排空會是瞬間完成的事。

## 建置

```bash
cd frontend && npm ci && npm run build     # 產出 frontend/dist（已 gitignore）
cd ../files_server && go build -o files-server .
cd ../backend && go build -o forum .
```

---

## 容器部署（docker compose）

`docker-compose.yml` 提供四個服務：`mysql`、`redis`、`backend`、`files_server`。
**沒有「前端」服務** —— 那是這個架構最刻意的一個決定：後端啟動時掃描前端產物、算出每個
HTML 殼裡 `<style>` 的 SHA-256 寫進 CSP，因此**被提供的檔案與被授權的雜湊必須來自
同一份建置輸出**。合成一個映像讓那個不變量由建置流程結構性保證。理由見
[ARCHITECTURE](ARCHITECTURE.md#為什麼前端沒有獨立服務)。

```bash
# 0. 打包兩個服務映像（Windows：./deploy/build-images.ps1）
./deploy/build-images.ps1

# 1. 準備三份設定檔
cp deploy/settings.conf.example deploy/settings.conf       # 容器編排的環境變數
cp backend/config/config.conf.example backend/config/config.conf
cp files_server/config.conf.example files_server/config.conf

# 2. 部署前先驗證設定 —— 這是整份 compose 裡最值得做的一步
docker compose --env-file deploy/settings.conf run --rm backend /app/forum -check

# 3. 起服務
docker compose --env-file deploy/settings.conf up -d
docker compose ps && docker compose logs -f backend
```

**為什麼每一步都帶 `--env-file deploy/settings.conf`**：compose 檔裡的密碼寫成
`${MYSQL_ROOT_PASSWORD:?…}`，而 compose 只會自動讀專案根目錄的 `.env`。
`deploy/settings.conf` 不是 `.env`，因此不傳就會在**開始建置之前**結束。

**兩個 config.conf 掛載到不同位置**：`backend` 讀 `/app/config/config.conf`，
`files_server` 讀 `/app/config.conf`（後者是工作目錄下的檔名）。compose 檔裡已經
分開寫好，不要照抄成同一個。

容器部署**必須額外設定** `TRUSTED_PROXY_CIDRS=172.28.0.0/16`（寫在
`backend/config/config.conf`）。留空會讓「無條件採信 `X-Forwarded-For`」生效，而
`forum-net` 內的**任何容器**都能直接連 8088 —— 它只要自己塞一個標頭就能讓限流計數記到
別的 IP 上：**症狀是「限流看起來有開，只是擋不住任何人」**。那個 subnet 寫死在
`docker-compose.yml` 的 `ipam` 設定裡，所以這裡可以直接抄。

停止行為與 systemd 部署的細節見 [`OPERATIONS.md`](OPERATIONS.md#停止行為)。
