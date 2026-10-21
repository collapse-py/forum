/*
server_extra_test.go 補齊 server.go 剩下的分支：儲存層失敗時的 500、
刪除網址無法解析、以及媒體 token 中介層「不該介入」的兩條路徑。

【為什麼要注入一個會失敗的儲存後端】
handleUpload 與 handleDelete 的錯誤分支原本沒有覆蓋，因為真實的本機後端在
測試環境裡幾乎不會失敗（磁碟沒滿、權限正確）。但那兩條分支正是「失敗時回什麼」
的契約：上傳必須回 500 而不是假裝成功，刪除的**非**「不存在」錯誤也必須是 500
而不是 404 —— 404 會讓後台把一次權限或磁碟問題當成「已經刪掉了」。
*/
package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// failingStorage 是一個可以指定 saveFile／deleteFile 失敗的後端。
type failingStorage struct {
	saveErr   error
	deleteErr error
}

func (f *failingStorage) saveFile(ctx context.Context, dir, filename string, data io.Reader) (string, error) {
	if f.saveErr != nil {
		return "", f.saveErr
	}
	return "/" + dir + "/" + filename, nil
}

func (f *failingStorage) deleteFile(ctx context.Context, path string) error {
	return f.deleteErr
}

func (f *failingStorage) getBaseURL() string { return "http://files.example" }

func (f *failingStorage) staticRoot() (string, bool) { return "", false }

// TestUploadReturns500WhenStorageFails 守住儲存失敗時的回應。
func TestUploadReturns500WhenStorageFails(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	handler := NewServer(cfg, &failingStorage{saveErr: errors.New("disk full")}, nil).Handler()

	req := multipartUpload(t, "/upload", "test-upload-token", "photo.jpg", "image/jpeg", []byte("bytes"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("狀態碼 = %d，want 500: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "disk full") {
		t.Errorf("回應洩漏了內部錯誤細節: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"url"`) {
		t.Errorf("失敗的回應仍然帶了檔案網址: %s", rec.Body.String())
	}
}

// TestDeleteReturns500WhenStorageFails 守住「非 NotExist 的刪除錯誤 → 500」。
//
// 這條界線很重要：把權限錯誤或磁碟錯誤回成 404，會讓後台的批次刪除把
// 「刪不掉」記成「已經不存在」，於是沒有人會去修那個問題。
func TestDeleteReturns500WhenStorageFails(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	handler := NewServer(cfg, &failingStorage{deleteErr: errors.New("permission denied")}, nil).Handler()

	req := httptest.NewRequest(http.MethodDelete, "/delete?url=%2Ffiles%2Fphoto.jpg", nil)
	req.Header.Set("X-Upload-Token", "test-upload-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("狀態碼 = %d，want 500: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "permission denied") {
		t.Errorf("回應洩漏了內部錯誤細節: %s", rec.Body.String())
	}
}

// TestDeleteRejectsUnparseableURL 守住 url.Parse 失敗的分支。
//
// %7F（DEL）是 net/url 明確拒絕的控制字元。這個分支的重點是「回 400 而不是
// 讓後續的 parts 解析去處理一個半壞的字串」。
func TestDeleteRejectsUnparseableURL(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	handler := NewServer(cfg, &failingStorage{}, nil).Handler()

	req := httptest.NewRequest(http.MethodDelete, "/delete?url=%7F", nil)
	req.Header.Set("X-Upload-Token", "test-upload-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("狀態碼 = %d，want 400: %s", rec.Code, rec.Body.String())
	}
}

// TestMediaTokenMiddlewareSkipsIrrelevantRequests 守住中介層的作用範圍。
//
// 它只該處理 GET/HEAD 的 /files/ 路徑。若它開始檢查其他路徑或方法，症狀是
// 「後端上傳突然 401」或「刪除請求被要求帶媒體 token」—— 兩者都會讓後端
// 的寫入路徑整個停擺，而錯誤訊息指向一個跟權限無關的地方。
func TestMediaTokenMiddlewareSkipsIrrelevantRequests(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	backend, err := newStorageBackend(cfg)
	if err != nil {
		t.Fatalf("newStorageBackend 回錯誤: %v", err)
	}
	// found=0：任何查詢都會回 401，因此「有被呼叫」與「被跳過」可以分辨。
	store := &fakeTokenStore{found: 0}
	middleware := NewServer(cfg, backend, store).mediaTokenMiddleware

	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middleware(next)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"非 /files/ 路徑", http.MethodGet, "/upload"},
		{"非 GET/HEAD 方法", http.MethodPost, "/files/photo.jpg"},
		{"DELETE 方法", http.MethodDelete, "/files/photo.jpg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := called
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))

			if rec.Code != http.StatusNoContent {
				t.Errorf("狀態碼 = %d，want 204（應直接交給 next）", rec.Code)
			}
			if called != before+1 {
				t.Error("請求沒有被交給我 next handler")
			}
		})
	}

	if len(store.queried) != 0 {
		t.Errorf("被跳過的請求仍然查了 Redis: %v", store.queried)
	}
}
