package httpapi

/*
本檔案是全站 HTTP 回應的統一出口，所有 JSON 回應都應經由這裡的函式送出。

統一回應格式
  - 失敗時的結構固定為 {"ok":false,"error":...,"message":...}（apiError）。
  - error 是由 HTTP 狀態碼推得的固定英文片語（http.StatusText），適合程式
    判斷與 log；message 是給人看的具體原因，可能為中文，也可能帶入上游
    錯誤文字。
  - 成功時由呼叫端自訂 payload（writeOK 直接透傳），許多端點另外帶自己的
    "ok" 欄位。換言之「失敗」有保證的結構，「成功」則保持各端點的自由度。

設計意圖
  - 讓前端只需要學會一種失敗形狀，不必對每個端點撰寫錯誤處理分支。
  - error 與 message 分工：狀態碼負責傳達結果種類，error 負責機器可讀的
    原因片語，message 負責診斷細節；三者都不承載敏感資料的遮蔽責任 ——
    writeError 不會過濾 message，呼叫端若把 err.Error() 塞進來就會外洩。
    目前僅 auth_handlers 的 Google 登入失敗路徑如此，屬刻意的除錯資訊取捨。

Content-Type 固定為 application/json
  - 格式正確與否由這裡把關，呼叫端不必記得設定表頭。
  - 不加 charset 參數是正確的：JSON 依 RFC 8259 只能是 UTF-8，瀏覽器與
    fetch 一律以 UTF-8 解讀；Go 的 encoder 也會把 <、>、& 與 U+2028/U+2029
    轉成 \uXXXX，因此內嵌中文不會造成解析問題。

狀態碼與 statusText 的分工
  - 狀態碼（數字）才是契約：負載平衡器、監控與前端分流都依它判斷。
  - http.StatusText 只是該狀態碼的標準原因片語，會原樣放進 error 欄位；
    它的存在讓回應自我描述（即使只看到 body 也知道大概原因），且不需要
    為每個狀態碼維護對照表。
*/

import (
	"encoding/json"
	"net/http"
)

// apiError 是所有失敗回應的固定結構，欄位說明如下：
//
//	OK      恆為 false。保留這個欄位是為了讓前端能用同一個鍵判斷成功或失敗，
//	        不必先看有沒有 error 欄位。
//	Error   依 HTTP 狀態碼推得的原因片語，適合程式判斷與 log 對照。
//	Message 具體原因。omitempty 讓沒有補充說明的錯誤不會多出一個空欄位。
type apiError struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// writeJSON 送出 JSON 回應，是本套件所有回應的共同出口。
// 參數 status 會原樣成為 HTTP 狀態碼，payload 則交給 encoding/json 序列化；
// payload 必須是 JSON 可序列化的型別，否則會得到空 body（Encode 的錯誤被忽略，
// 理由見函式內說明）。
// 副作用：寫入表頭並提交狀態碼，因此呼叫之後不可再改變這兩者。
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	// 必須在 WriteHeader 之前設定：一旦狀態碼送出，表頭就鎖定了。
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// 忽略 Encode 的錯誤：此時狀態碼與表頭都已經送到用戶端，即使序列化失敗
	// 或連線中途斷開，也無法再改成錯誤回應；唯一剩下的選項是在 HTTP handler
	// 中 panic，那會被 net/http 記成伺服器錯誤並可能洩漏堆疊，弊多於利。
	// 附帶效果：Encoder 會在結尾補一個換行，對 JSON 解析無影響。
	_ = json.NewEncoder(w).Encode(payload)
}

// writeOK 以 200 送出 payload。
// 成功回應的內容由各端點自訂，因此這裡不做任何包裝，維持既有 API 契約。
func writeOK(w http.ResponseWriter, payload interface{}) {
	writeJSON(w, http.StatusOK, payload)
}

// writeError 以指定狀態碼送出標準失敗結構。
// error 欄位取自 http.StatusText(status)，因此 message 可以自由填寫而不會
// 影響機器可讀的那一份；若傳入的 status 不是標準代碼，StatusText 會回傳
// 空字串，此時 error 將是空字串（message 仍會保留）。
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiError{OK: false, Error: http.StatusText(status), Message: message})
}

// methodNotAllowed 回應 405，供只接受特定 HTTP 方法的端點在方法不符時使用。
// 注意：依 RFC 9110 應一併送出 Allow 表頭說明可用的方法，目前未實作；對瀏覽器
// 與 fetch 而言 405 本身已足以判斷，不影響正常使用。
func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// unauthorized 回應 401，並在 message 為空時填入預設文字。
// 補預設值是為了讓「忘記傳 message」的呼叫端仍能產生可顯示的內容，避免
// message 欄位因 omitempty 而整個消失，前端因而拿到空的錯誤訊息。
func unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "unauthorized"
	}
	writeError(w, http.StatusUnauthorized, message)
}

// badRequest 回應 400，用於輸入格式或參數錯誤。
// 它存在的價值是讓呼叫端一眼看出意圖，status 與 message 的一致性也由
// 這一層統一維護。
func badRequest(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, message)
}

// internalError 回應 500。
// 呼叫端應傳入不含內部細節的靜態文字（例如 "unable to generate session"），
// 真正的錯誤只寫進 log；此函式本身不做過濾，因此無法阻止敏感資訊被寫入
// message。500 語意是「伺服器端問題、與請求內容無關」。
func internalError(w http.ResponseWriter, message string) {
	writeError(w, http.StatusInternalServerError, message)
}
