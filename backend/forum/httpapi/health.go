package httpapi

/*
本檔案實作單一的健康檢查端點 /healthz（註冊於 server.go 的 Handler）。

回報的依賴
   - mysql：s.db。論壇的貼文、留言、檢舉與使用者狀態都在 MySQL，連不上等於
     讀寫功能全不可用。
   - redis：s.mediaRedis。與 session 共用同一個 Redis 實例（見 main.go），
     負責媒體存取權杖；為 nil 代表該功能未啟用，此時視為正常而不是失敗。
   - es：s.es。Elasticsearch，只負責貼文搜尋與索引。連不上時論壇仍可發文、
     留言，搜尋會退回 MySQL 的 LIKE 比對，因此「es 不健康」不等於
     「整站不就緒」—— 但它仍會被回報，讓維運在搜尋品質變差時有明確線索。
     這是刻意不同於 mysql/redis 的取捨：那兩者掛掉等於核心功能全滅，
     這裡掛掉只影響一個附加功能。

存活（liveness）與就緒（readiness）的區別
  - 這支處理器實作的是「就緒」語意：依賴失效時回 503，讓載入平衡器或
    Kubernetes readiness probe 把這個實體移出服務清單，但行程本身不被殺掉。
  - 存活探針要回答的是「行程是否還能正常運作」，不該因 MySQL 短暫故障就
    回失敗，否則會造成依賴故障 → 重啟風暴。本專案沒有另外註冊 liveness
    端點，因此也沒有把兩者拆開的機會。
  - 狀態碼選擇：200 表示所有已設定的依賴都通過；503（Service Unavailable）
    表示至少一項無法使用，這是負載平衡器與 k8s 慣例會據以停止派送新請求的
    代碼。

設計取向
  - 兩項檢查都會執行完才回應，即使 mysql 已經失敗也要一併回報 redis 的結果，
    讓一次探測就能看到完整的依賴狀態，而不必來回重試。
  - 逾時刻意設為 3 秒：探測端點本身不該被慢速的依賴拖住，否則
    基礎設施的探測逾時會先發生，得到的失敗原因反而不可辨。
  - 回應不含任何憑證或連線字串，只有依賴名稱與錯誤訊息。
*/

import (
	"context"
	"net/http"
	"time"
)

// handleHealth 探測 MySQL 與 Redis 的連線狀態並回傳 JSON。
// 只接受 GET：探測端點不需要主動修改任何狀態，若允許其他方法，惡意或
// 誤觸的請求也能觸發對外部署的連線檢查。
// 探測失敗時回應本文仍是 200 的 JSON 格式描述，只是狀態碼改為 503，
// 讓呼叫端（以及基礎設施）能用狀態碼判斷、用 body 診斷原因。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// 先預設為 200 與全部 "ok"，讓 checks 一定包含每個依賴的欄位；
	// 探測失敗時再覆寫該欄位為實際錯誤訊息。同一個 map 同時負責
	// 「有沒有問題」與「問題是什麼」兩件事。
	// es 固定出現（即使未設定），理由是「查得到但其實是降級模式」與
	// 「壓根沒設定」是兩種維運上要區分的情況，欄位缺席就看不出來。
	status := http.StatusOK
	checks := map[string]string{
		"mysql": "ok",
		"redis": "ok",
		"es":    "ok",
	}

	// 以請求的 context 為基底加上 3 秒逾時：探測必須在基礎設施的逾時內結束，
	// 且用戶端中途離開時不該繼續佔用資料庫連線。defer cancel 確保連線一定
	// 釋放，否則每次探測洩漏一個連線槽。
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// PingContext 是最輕量的存活確認：只驗證連線可取得並回應，不查詢資料表。
	if err := s.db.PingContext(ctx); err != nil {
		checks["mysql"] = err.Error()
		status = http.StatusServiceUnavailable
	}

	// mediaRedis 為 nil 代表建構時沒有注入（媒體功能未啟用），與「連不上」是
	// 不同狀況：前者不應讓整個服務被判為不就緒。
	if s.mediaRedis != nil {
		if err := s.mediaRedis.Ping(ctx).Err(); err != nil {
			checks["redis"] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}

	// ES 不可用同樣只影響搜尋品質，不影響就緒判定，因此只回報不降級狀態碼。
	// 沒設定時以 "disabled" 標示：它不是健康的一種，而是根本沒啟用。
	if !s.esEnabled() {
		checks["es"] = "disabled"
	} else if err := s.es.Ping(ctx); err != nil {
		checks["es"] = err.Error()
	}

	// ok 欄位與狀態碼保持一致，讓只解析 body 的前端也能判斷結果；
	// checks 保留原始錯誤訊息供維運排查（不含連線字串等敏感資訊）。
	writeJSON(w, status, map[string]interface{}{
		"ok":     status == http.StatusOK,
		"checks": checks,
	})
}
