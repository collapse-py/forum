/*
JSON 請求主體的大小上限（body_limit_test.go）。

這一批斷言釘的是 H5 修好後的形狀：每個寫入端點都經過 http.MaxBytesReader，
而「業務邏輯有 2000 字上限」不再是 body 的保護。兩件事必須分開 ——
json.Decoder 會在**任何長度檢查之前**把整個 JSON value 讀進記憶體，所以一個
沒有 body 上限的端點能用一份數百 MB 的本文換一次完整的解析（記憶體放大），
而它後面的 rune 檢查只會決定那份記憶體之後被丟掉還是被寫進資料庫。

兩個方向都要測，因為「把所有 body 都擋掉」也能通過其中任何一個方向：

  超限 → 400 invalid request
      MaxBytesReader 讓 Decode 回傳錯誤，handler 在碰到資料庫之前就回 400。
     這條路徑不需要 MySQL，因此在沒有資料庫的環境裡也能跑。

  未超限但內容不合法 → 400 帶著**業務**訊息
      用「簡介超過 500 字」而不是「內容合法」：合法的 body 會繼續走到 upsert，
     而這個測試伺服器沒有 db（見 newPostEditTestServer 的說明）。業務訊息出現
     即證明 request 通過了大小那一關、確實進到了 handler 的驗證邏輯。
*/
package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWriteRejectsOversizedJSONBody 確認超大的 JSON 主體在解碼階段就被擋下。
func TestWriteRejectsOversizedJSONBody(t *testing.T) {
	server := newPostEditTestServer(t, "http://localhost:8088")

	// 32768 + 4096 > profile 端點的 4 KB 上限。單位元組數刻意遠大於上限，
	// 這樣即使將來上限被調高幾倍，這支測試仍然測得到「有上限」這件事。
	recorder := httptest.NewRecorder()
	request := loggedInRequest(t, server, "PUT", "/api/forum/profile",
		`{"nickname":" Alic e","bio":"`+strings.Repeat("a", 32768)+`"}`)
	request.Header.Set("Origin", "http://localhost:8088")
	server.handleForumProfile(recorder, request)

	if recorder.Code != 400 {
		t.Fatalf("狀態碼 = %d, want 400（body 上限沒有生效）", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "invalid request") {
		t.Errorf("訊息 = %q, want 含 invalid request（回應不該透露內部細節）", body)
	}
}

// TestWriteAcceptsBodyUnderLimit 確認大小閘門不會擋掉合法尺寸的請求。
//
// 斷言的關鍵是錯誤訊息來自**業務**驗證（簡介長度）而不是大小閘門：這證明
// 請求確實通過了 MaxBytesReader 並進到 handler 的驗證邏輯。少了這一支，
// 「把所有 PUT 都回 400 invalid request」也能通過上面那支測試。
func TestWriteAcceptsBodyUnderLimit(t *testing.T) {
	server := newPostEditTestServer(t, "http://localhost:8088")

	recorder := httptest.NewRecorder()
	request := loggedInRequest(t, server, "PUT", "/api/forum/profile",
		`{"nickname":"Alic e","bio":"`+strings.Repeat("a", 501)+`"}`)
	request.Header.Set("Origin", "http://localhost:8088")
	server.handleForumProfile(recorder, request)

	if recorder.Code != 400 {
		t.Fatalf("狀態碼 = %d, want 400", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "簡介最多 500 字") {
		t.Errorf("訊息 = %q, want 含「簡介最多 500 字」（代表請求通過了大小閘門、進到業務驗證）", body)
	}
}
