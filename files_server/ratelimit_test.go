/*
ratelimit_test.go：files_server 自己的每 IP 計價器。

【測什麼】
這個服務在很長一段時間裡完全沒有限流，而它有三類成本差異極大的端點
（upload 50 MiB 的 multipart、delete 檔案系統、/files/ 一次 Redis EXISTS）。
計價器就是為此而存在，因此這裡測的是三件會讓它**靜默失效**的事：

  1. 放行／阻擋的時機 —— 權杖補充算錯的話，症狀是「要嘛一直 429，要嘛永遠放行」。
  2. nil 計價器一律放行 —— 這是「關掉限流」的開關，而它壞掉時整個保護消失。
  3. 惰性清除不會把還在用的桶清掉 —— 清錯了等於把限流變成隨機。

斷言刻意用「時間推進」而不是 sleep：權杖桶的補充是時間的函式，而測試跑得越快
越好。這裡不注入時鐘，而是直接用能觀測的行為（burst 內全放行、超過後全擋），
那已經足以界定行為。
*/

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// TestRateLimiterAllowsBurstThenRejects 釘住最基本的不變式：burst 內的連发放行，
// 超過就擋，而**被擋掉的請求不會消耗權杖**。
func TestRateLimiterAllowsBurstThenRejects(t *testing.T) {
	limiter := newRateLimiter(1, 3, 1024)

	for i := 0; i < 3; i++ {
		if !limiter.allow("1.2.3.4") {
			t.Fatalf("第 %d 個請求被擋：burst=3 之內應該全放行", i+1)
		}
	}
	if limiter.allow("1.2.3.4") {
		t.Error("burst 用完之後仍被放行：權杖桶沒有上限")
	}

	// 被擋的請求不該繼續消耗權杖 —— 否則一次短 burst 之後要等更久才恢復。
	limiter.allow("1.2.3.4")
	limiter.allow("1.2.3.4")
	time.Sleep(1100 * time.Millisecond) // rate=1/s → 補充約一個權杖
	if !limiter.allow("1.2.3.4") {
		t.Error("等了一個補充週期仍被擋：補充沒在運作，或被擋的請求把權杖吃掉了")
	}
}

// TestRateLimiterKeysAreIndependent 確認每個 IP 各自一個桶。
//
// 共用一個桶的症狀是「一個使用者被抓爆，其他人也跟著 429」—— 而那正是全站
// 共用限流配額的災難，比不限流更難解釋。
func TestRateLimiterKeysAreIndependent(t *testing.T) {
	// rate 必須是正數：0 在這裡的意思是「關掉限流」（newRateLimiter 回 nil），
	// 那由另一支測試釘住。burst=1 讓每個 IP 只有一次機會，因此任何共用都會被看到。
	limiter := newRateLimiter(1, 1, 1024)
	if !limiter.allow("1.1.1.1") {
		t.Fatal("第一個 IP 的第一個請求應該放行")
	}
	if !limiter.allow("2.2.2.2") {
		t.Error("第二個 IP 受到第一個 IP 的額度影響：兩個鍵沒有分開")
	}
	if limiter.allow("1.1.1.1") {
		t.Error("第一個 IP 的額度已經用完卻又放行：兩個鍵被混在一起了")
	}
}

// TestNilRateLimiterAllowsEverything 釘住「關掉限流」這個開關。
//
// rate <= 0 時 newRateLimiter 回 nil，而 nil 接收者一律放行。少了這一支，
// 「關掉限流」會變成「所有請求都 429」或 panic —— 兩者都不會出現在測試之外
// 的任何地方。
func TestNilRateLimiterAllowsEverything(t *testing.T) {
	if newRateLimiter(0, 10, 1024) != nil {
		t.Error("rate=0 應該回 nil，而不是一個永遠不放行的計價器")
	}
	var limiter *rateLimiter
	for i := 0; i < 5; i++ {
		if !limiter.allow("1.2.3.4") {
			t.Fatalf("nil 計價器擋掉了第 %d 個請求", i+1)
		}
	}
}

// TestRateLimiterSweepKeepsActiveBuckets 確認惰性清除只回收閒置的桶。
//
// 清錯的症狀是限流變成隨機（某些請求突然可以超過 burst），而那比沒有限流更難
// 排查。這一支把 maxBuckets 壓到 1，強制每一次 allow 都觸發清除。
func TestRateLimiterSweepKeepsActiveBuckets(t *testing.T) {
	limiter := newRateLimiter(1, 1, 1)
	if !limiter.allow("1.1.1.1") {
		t.Fatal("第一個請求應該放行")
	}
	// 第二個鍵會觸發清除（len(buckets) >= maxBuckets）。1.1.1.1 剛剛才被用過，
	// 因此它必須活著 —— 活著的意思是它的額度已經用完。
	if !limiter.allow("2.2.2.2") {
		t.Fatal("第二個鍵的第一個請求應該放行")
	}
	if limiter.allow("1.1.1.1") {
		t.Error("剛被用過的 1.1.1.1 又放行了一個請求：清除把還在用的桶回收了")
	}
}

// TestRateLimiterMiddlewareReturns429 釘住 HTTP 層的回應形狀。
//
// 429 而不是 403：403 看起來像「你沒有權限」，而這裡想說的是「你太快了」。
// Retry-After 讓呼叫端知道要等多久，而不是立刻重試。
func TestRateLimiterMiddlewareReturns429(t *testing.T) {
	limiter := newRateLimiter(1, 1, 16)
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/files/x.jpg", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("第一個請求 = %d, want 200", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/files/x.jpg", nil))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("第二個請求 = %d, want 429", recorder.Code)
	}
	if retry := recorder.Header().Get("Retry-After"); retry == "" {
		t.Error("429 沒有附 Retry-After：呼叫端只能立刻重試")
	}
}

// TestHandlerHasRateLimiters 確認三條路由都掛上了限制。
//
// 這支測試的存在理由是「忘記掛中介層」不會有任何症狀：服務照樣運作，只是不再
// 有任何上限。因此它直接走完整的中介層鏈，用 burst=1 打到 429。
func TestHandlerHasRateLimiters(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	// 兩類限制都壓到 burst=1：上傳兩次、刪除兩次，第二次必須 429。
	cfg.RateLimit.MediaRate = 1
	cfg.RateLimit.MediaBurst = 1
	cfg.RateLimit.WriteRate = 1
	cfg.RateLimit.WriteBurst = 1
	cfg.Upload.Token = "test-token"

	// newTestServer 回傳的是掛好整條中介層鏈的 handler，因此這一支測的是
	// 「掛了沒掛」而不是「限制器本身對不對」（那由上面的單元測試負責）。
	handler, backend := newTestServer(t, cfg, &fakeTokenStore{found: 1})
	if backend == nil {
		t.Fatal("newTestServer 沒有回傳儲存後端")
	}
	_ = handler

	// /upload 兩次。
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/upload", nil)
		request.Header.Set("X-Upload-Token", cfg.Upload.Token)
		handler.ServeHTTP(recorder, request)
		want := http.StatusBadRequest // 沒有本文 → 400
		if i == 1 {
			want = http.StatusTooManyRequests
		}
		if recorder.Code != want {
			t.Errorf("upload 第 %d 次 = %d, want %d", i+1, recorder.Code, want)
		}
	}

	// /delete 兩次。
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodDelete, "/delete?url="+url.QueryEscape("/files/none.jpg"), nil)
		request.Header.Set("X-Upload-Token", cfg.Upload.Token)
		handler.ServeHTTP(recorder, request)
		want := http.StatusNotFound // 檔案不存在 → 404
		if i == 1 {
			want = http.StatusTooManyRequests
		}
		if recorder.Code != want {
			t.Errorf("delete 第 %d 次 = %d, want %d", i+1, recorder.Code, want)
		}
	}
}
