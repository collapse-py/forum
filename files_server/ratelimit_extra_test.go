/*
ratelimit_extra_test.go 補齊 ratelimit.go 剩下的兩條分支：burst 下限與
「清除真的會回收閒置的桶」。

【為什麼需要這一支】
既有測試守住了「活躍的桶不被清掉」，但沒有守住「閒置的桶**會被**清掉」。
少了後者，maxBuckets 這個上界只存在於註解裡：記憶體會隨著每個新 IP 單向成長，
而症狀要到服務跑很久之後才會出現。
*/
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewRateLimiterClampsBurst 守住 burst < 1 時的下限。
//
// burst=0 若被原樣保留，第一個請求會拿到 tokens=-1，於是連一次都放行不了 ——
// 設定檔打錯一個數字就讓整個端點永久 429。下限 1 讓它至少能服務單一請求，
// 而「每秒幾個」由 rate 決定。
func TestNewRateLimiterClampsBurst(t *testing.T) {
	limiter := newRateLimiter(1, 0, 16)
	if limiter == nil {
		t.Fatal("rate=1 卻回傳 nil")
	}
	if limiter.burst != 1 {
		t.Errorf("burst = %v，want 1（下限）", limiter.burst)
	}
	if !limiter.allow("1.2.3.4") {
		t.Error("burst=0 被夾成 1 之後，第一個請求仍被擋")
	}

	negative := newRateLimiter(1, -5, 16)
	if negative == nil || negative.burst != 1 {
		t.Errorf("burst=-5 沒有被夾成 1: %+v", negative)
	}
	if !negative.allow("1.2.3.4") {
		t.Error("burst=-5 被夾成 1 之後，第一個請求仍被擋")
	}
}

// TestRateLimiterSweepRemovesIdleBuckets 守住清除會回收閒置的桶。
//
// 直接把一個「兩小時前」的桶放進 map，再讓下一個請求觸發清除。這不依賴
// sleep：桶的 lastSeen 是我們自己寫進去的。
func TestRateLimiterSweepRemovesIdleBuckets(t *testing.T) {
	limiter := newRateLimiter(1, 1, 1)
	limiter.buckets["stale.example"] = &bucket{
		tokens:   1,
		lastSeen: time.Now().Add(-2 * time.Hour),
	}

	// len(buckets) >= maxBuckets，因此這一個請求會先觸發 sweepLocked。
	if !limiter.allow("fresh.example") {
		t.Fatal("新鍵的第一個請求應該放行")
	}

	if _, ok := limiter.buckets["stale.example"]; ok {
		t.Error("閒置兩小時的桶沒有被回收 —— maxBuckets 不是一個真正的上界")
	}
	if _, ok := limiter.buckets["fresh.example"]; !ok {
		t.Error("剛建立的桶不見了")
	}
}

// TestRateLimiterMiddlewareRetryAfterIsCeil 守住 Retry-After 是「向上取整的
// 每秒補充數」，且永遠至少 1。
//
// 0 會讓呼叫端立刻重試，正好製造出這個中介層想擋掉的行為。
func TestRateLimiterMiddlewareRetryAfterIsCeil(t *testing.T) {
	for _, rate := range []float64{0.25, 1, 2.5} {
		limiter := newRateLimiter(rate, 1, 16)
		handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// 第一個請求消耗 burst；第二個被擋。
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/files/x.jpg", nil))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/x.jpg", nil))

		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("rate=%v 的第二個請求 = %d，want 429", rate, rec.Code)
		}
		want := map[float64]string{0.25: "1", 1: "1", 2.5: "3"}[rate]
		if got := rec.Header().Get("Retry-After"); got != want {
			t.Errorf("rate=%v 的 Retry-After = %q，want %q", rate, got, want)
		}
	}
}
