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

// TestClientKeyStripsPort 守住計價鍵是「來源 IP，不含埠號」。
//
// 含埠的鍵是這個計價器最容易靜默失效的方式：桶是「每來源一個」，而每條新 TCP
// 連線都帶一個新的臨時埠，因此「每發請求都開新連線」的呼叫端永遠落在一個全新的
// 桶裡 —— 限流仍在跑、仍在回 429 的程式碼，卻一次也不會觸發。
func TestClientKeyStripsPort(t *testing.T) {
	for _, tc := range []struct {
		remoteAddr string
		want       string
	}{
		{"203.0.113.9:52415", "203.0.113.9"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		// 沒有可切分的冒號對：保留原值（它們本身就是可用的鍵）。
		{"::1", "::1"},
		{"203.0.113.9", "203.0.113.9"},
	} {
		request := httptest.NewRequest(http.MethodGet, "/files/x.jpg", nil)
		request.RemoteAddr = tc.remoteAddr
		if got := clientKey(request); got != tc.want {
			t.Errorf("clientKey(%q) = %q，want %q", tc.remoteAddr, got, tc.want)
		}
	}
}

// TestRateLimiterSameIPDifferentPortsShareBucket 是上一條的行為版：同一個 IP
// 換了埠，額度必須繼續被消耗，而不是重新計。
func TestRateLimiterSameIPDifferentPortsShareBucket(t *testing.T) {
	limiter := newRateLimiter(1, 1, 16)
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/files/x.jpg", nil)
	request.RemoteAddr = "203.0.113.9:40000"
	handler.ServeHTTP(first, request)
	if first.Code != http.StatusOK {
		t.Fatalf("第一個請求 = %d，want 200", first.Code)
	}

	second := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/files/x.jpg", nil)
	request.RemoteAddr = "203.0.113.9:40001" // 同一個 IP、新的連線埠
	handler.ServeHTTP(second, request)
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("同一 IP 換埠後 = %d，want 429（兩個埠各拿到一份額度）", second.Code)
	}
}
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
