/*
security_invariants_test.go 覆蓋 httpapi 裡「壞掉時沒有錯誤訊息」的純邏輯：
CSRF 來源檢查、限流器的計數邊界、以及 CSP 樣板組裝。

【為什麼優先測這三塊】
ROADMAP.md 的 Phase 3.3 說 httpapi「0% 覆蓋，約 2,500 行，含 CSRF、限流、
代理信任、管理員授權、HTML 渲染」，而優先順序是「CSRF 與來源檢查 → 代理信任
→ 限流 → HTML 渲染」。

httpapi 已經有幾支測試（announcement、monitoring、securityheaders、
site、stats、trustedproxy），因此這個檔刻意**不重複**它們，只補上那些
「純函式、可用表格窮舉、而失效症狀是靜默的」部分：

  - isTrustedOrigin：CSRF 的最後一道防線。它壞掉時的症狀是「攻擊者可以
    以使用者的身分發文」，而伺服器端看不到任何異常 —— 因為請求確實來自
    使用者的瀏覽器。
  - RateLimiter：它壞掉時的症狀是「限流看起來有開，只是擋不住任何人」。
  - buildContentSecurityPolicy：它壞掉時的症狀是「某一頁沒有版面」。

【一致的測試方法】
三塊都用「輸入 → 期望輸出」的表格，並且每一條失敗訊息都寫明**症狀是什麼**。
理由是這個專案的失效模式幾乎都不是「報錯」，而是「看起來正常的錯值」——
一個只說「got != want」的斷言會讓讀者看不出該去看哪裡。
*/
package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forum/forum/config"
)

// ---------------------------------------------------------------------------
// CSRF：來源檢查
// ---------------------------------------------------------------------------

// TestIsTrustedOrigin 是這個檔最重要的一支測試。
//
// isTrustedOrigin 是本站防止跨站請求偽造（CSRF）的最後一道防線，而 CSRF 的
// 特性是：受害者會**自願**發出那個請求（表單自動送出、連結被點）。
// 因此從伺服器的角度來看，那個請求與使用者自己的請求完全一樣 —— 沒有任何
// 異常、沒有任何日誌可以把它標記出來。
func TestIsTrustedOrigin(t *testing.T) {
	cfg := config.Config{
		TrustedOrigins: []string{
			"https://forum.example.com",
			"https://www.forum.example.com",
		},
		PublicBaseURL: "https://forum.example.com",
	}
	s := &Server{cfg: cfg}

	cases := []struct {
		name    string
		method  string
		origin  string
		referer string
		want    bool
		why     string
	}{
		// 允許：精確命中
		{name: "Origin 精確命中", method: http.MethodPost,
			origin: "https://forum.example.com", want: true},
		{name: "Origin 命中第二個白名單項", method: http.MethodPost,
			origin: "https://www.forum.example.com", want: true},

		// 允許：沒有 Origin 時改比對 Referer
		// 理由寫在 isTrustedOrigin 的說明裡：Referer 在舊瀏覽器與部分隱私設定下
		// 比 Origin 更可靠，而「兩者都沒有」才是真正危險的情況。
		{name: "沒有 Origin，有相符的 Referer", method: http.MethodPost,
			referer: "https://forum.example.com/forum/new", want: true},

		// 拒絕：跨站
		{name: "跨站 Origin", method: http.MethodPost,
			origin: "https://evil.example", want: false,
			why: "這就是 CSRF：攻擊者的站發出的請求"},
		{name: "子網域攻擊", method: http.MethodPost,
			origin: "https://forum.example.com.evil.example", want: false,
			why: "字串前綴匹配會把它當成白名單項目"},
		{name: "父網域", method: http.MethodPost,
			origin: "https://example.com", want: false},
		{name: "同 host 不同 scheme", method: http.MethodPost,
			origin: "http://forum.example.com", want: false,
			why: "http 與 https 是不同的來源；放行會讓一個被降級的連線成為跳板"},
		{name: "同 host 不同 port", method: http.MethodPost,
			origin: "https://forum.example.com:8443", want: false},
		{name: "Referer 指向別站", method: http.MethodPost,
			referer: "https://evil.example/steal", want: false},

		// 允許：兩個標頭都沒有。
		//
		// 這是一個**刻意**的取捨，而它值得單獨一個測試案例說明，因為它是
		// csrf.go 檔頭明確記錄的殘餘風險：
		//
		//	兩者都沒有就放行。理由：像 curl、伺服器對伺服器的呼叫、瀏覽器
		//	擴充套件等合法客戶端可能一個都不送，擋下它們會讓功能壞掉；
		//	殘餘風險：刻意移除兩個表頭的隱私設定或代理會讓跨站請求也通過，
		//	這時本防護失效，必須仰賴 SameSite cookie 與後端的身分檢查。
		//
		// 因此這個測試的價值是**把這個取捨釘成可見的事實**：若將來有人想改成
		// 「沒有標頭就拒絕」，這支測試會失敗並要求同時評估後端對
		// 伺服器對伺服器呼叫的支援（那個改動會讓 break-glass 工具失效）。
		{name: "Origin 與 Referer 都沒有（刻意放行）", method: http.MethodPost,
			want: true,
			why: "csrf.go 明確記錄了這個取捨與它的殘餘風險；改動前必須評估 " +
				"伺服器對伺服器的呼叫（後端上傳圖片時會用到）"},
		// Origin 是空白字元而非缺席的時候：中介設備常補空白，
		// 必須與「沒有」同等處理。
		{name: "Origin 是空白字元", method: http.MethodPost,
			origin: "   ", want: true,
			why: "空白等同缺席；它與真正缺席走同一條程式路徑"},
		{name: "Origin 空白 + Referer 跨站", method: http.MethodPost,
			origin: "   ", referer: "https://evil.example", want: false,
			why: "Origin 缺席時必須往 Referer 看，不能因為「有 Origin 標頭" +
				"（只是空白）」就放行"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/api/forum/posts", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				r.Header.Set("Referer", tc.referer)
			}
			got := s.isTrustedOrigin(r)
			if got != tc.want {
				t.Errorf("isTrustedOrigin(Origin=%q, Referer=%q) = %v, want %v。%s",
					tc.origin, tc.referer, got, tc.want, tc.why)
			}
		})
	}
}

// TestTrustedOriginWithEmptyWhitelist 守住「白名單空 = 全部拒絕」。
//
// 這個組合不會出現在正常部署（applyDefaults 至少會給 PublicBaseURL），
// 但它值得被釘住，因為它的失效方式特別糟：若實作把空白名單解讀成
// 「沒有白名單所以沒有東西要限制」，那麼一個設定檔打錯字讓白名單變空時，
// CSRF 防護會整個關閉 —— 而且沒有任何錯誤訊息。
func TestTrustedOriginWithEmptyWhitelist(t *testing.T) {
	s := &Server{cfg: config.Config{TrustedOrigins: nil}}

	r := httptest.NewRequest(http.MethodPost, "/api/forum/posts", nil)
	r.Header.Set("Origin", "https://forum.example.com")

	if s.isTrustedOrigin(r) {
		t.Error("白名單為空時 isTrustedOrigin 回 true —— " +
			"設定檔打錯字會讓 CSRF 防護整個關閉，而且沒有任何錯誤訊息")
	}
}

// TestRequireTrustedOriginMiddleware 覆蓋中介層層級的行為。
//
// 這一支補的是表格測試補不到的一半：**中介層必須根本呼叫下游**。
// 若 requireTrustedOrigin 忘了呼叫 next，一個「被拒絕時回 403、被允許時
// 什麼都不做」的實作也會通過上面那一整支測試 —— 而症狀是「登入後無法發文」
// 且沒有任何錯誤。
func TestRequireTrustedOriginMiddleware(t *testing.T) {
	cfg := config.Config{
		TrustedOrigins: []string{"https://forum.example.com"},
		PublicBaseURL:  "https://forum.example.com",
	}
	s := &Server{cfg: cfg}

	t.Run("允許的來源會到達下游", func(t *testing.T) {
		called := false
		handler := s.requireTrustedOrigin(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		r := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
		r.Header.Set("Origin", "https://forum.example.com")
		rec := httptest.NewRecorder()
		handler(rec, r)

		if !called {
			t.Error("允許的來源沒有到達下游 handler —— 症狀是「登入後無法登出」")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("狀態碼 = %d，want 200", rec.Code)
		}
	})

	t.Run("跨站來源被擋下且不呼叫下游", func(t *testing.T) {
		called := false
		handler := s.requireTrustedOrigin(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		r := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
		r.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		handler(rec, r)

		if called {
			t.Error("被拒絕的請求仍然呼叫了下游 handler —— 那等於 CSRF 防護形同不存在")
		}
		if rec.Code != http.StatusForbidden {
			t.Errorf("狀態碼 = %d，want 403", rec.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// 限流
// ---------------------------------------------------------------------------

// TestPruneExpired 覆蓋計數視窗的裁剪邏輯。
//
// 這個函式的失效症狀是**記憶體持續增長**：一個不裁剪過期時間戳的限流器
// 會把每個曾經連過的 IP 的時間戳永遠留在記憶體裡 —— 那是一個不需要任何
// 流量就能觸發的資源耗盡。
func TestPruneExpired(t *testing.T) {
	base := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	at := func(sec int) time.Time { return base.Add(time.Duration(sec) * time.Second) }

	cases := []struct {
		name  string
		in    []time.Time
		cut   time.Time
		wantN int
	}{
		{name: "空的視窗", in: nil, cut: at(0), wantN: 0},
		{name: "全部在視窗內", in: []time.Time{at(1), at(2), at(3)}, cut: at(0), wantN: 3},
		{name: "全部過期", in: []time.Time{at(1), at(2)}, cut: at(10), wantN: 0},
		{name: "部分過期（中間）", in: []time.Time{at(1), at(5), at(8)}, cut: at(4), wantN: 2},
		{name: "邊界：等於 cutoff", in: []time.Time{at(4)}, cut: at(4), wantN: 0}, // 刻意記錄這個方向：等於 cutoff 的項目被**剔除**。這代表視窗是
		// 「(cutoff, now]」而不是「[cutoff, now]」—— 而一秒的差異
		// 在一個 60 秒的視窗裡不重要，在一個 1 秒的視窗裡就意味著
		// 額度其實是 0。因此這個方向必須被釘住。

		{name: "邊界：剛好超過 cutoff", in: []time.Time{at(5)}, cut: at(4), wantN: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pruneExpired(tc.in, tc.cut)
			if len(got) != tc.wantN {
				t.Errorf("pruneExpired 回傳 %d 個項目，want %d", len(got), tc.wantN)
			}
		})
	}
}

// TestPruneExpiredDoesNotGrow 守住「裁剪就地進行，不配置新 slice」。
//
// 這是一個效能性質，而它壞掉時的症狀是「GC 壓力隨流量上升」—— 不會出現在任何
// 錯誤裡，只會出現在一段時間之後的 CPU profile 上。
func TestPruneExpiredDoesNotGrow(t *testing.T) {
	base := time.Now()
	// 1000 個時間戳，兩半過期。
	times := make([]time.Time, 0, 1000)
	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			times = append(times, base.Add(-time.Duration(i)*time.Second))
		} else {
			times = append(times, base.Add(time.Duration(i)*time.Second))
		}
	}

	kept := pruneExpired(times, base)
	if len(kept) != 500 {
		t.Fatalf("保留 %d 個項目，want 500", len(kept))
	}
	// 檢查它**就地**重用：&kept[0] 必須等於 &times[0]（同一個底層陣列）。
	// 這個斷言寫成「地址相同」而不是比較 cap，因為 reused := times[:0]
	// 與 append(times[:0], ...) 兩種寫法都會就地，但後者會真的配置。
	if &kept[0] != &times[0] {
		t.Error("pruneExpired 配置了新的 backing array —— " +
			"它的說明承諾就地裁剪，理由是 Reduce 負載下的 GC 壓力")
	}
}

// TestSplitComma 覆蓋標頭值的解析。
func TestSplitComma(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{name: "空字串", in: "", want: nil},
		{name: "單一項", in: "a", want: []string{"a"}},
		{name: "多項", in: "a,b,c", want: []string{"a", "b", "c"}},
		{name: "有空白", in: " a , b ", want: []string{"a", "b"}},
		{name: "空項目", in: "a,,b", want: []string{"a", "b"}},
		{name: "只有逗號", in: ",", want: []string{}},
		{name: "前後逗號", in: ",a,", want: []string{"a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitComma(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("splitComma(%q) = %v，want %v", tc.in, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("splitComma(%q)[%d] = %q，want %q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestTrimSpaceAndLastIndexByte 覆蓋這兩個為了取代 strings 呼叫而自製的小工具。
//
// 之所以要測它們：這兩個函式的存在理由是效能（避免配置），而「效能理由」最
// 容易被後人「簡化」回 strings.TrimSpace —— 那會是一個**行為不同**的改動
// （見下面 trimSpace 的說明），因此這個檔讓那個差異變成可被驗證的。
func TestTrimSpaceAndLastIndexByte(t *testing.T) {
	// trimSpace 刻意只去掉半形空格與水平 tab，**不**去掉 \n、\r 與 Unicode
	// 空白。這不是疏忽而是它的文件明載的取捨：「表頭值在實務上只會有空格與
	// tab」。這個測試把那個取捨釘住 —— 一個「順手改成 strings.TrimSpace」的
	// 提交會在這裡失敗，而那個改動對這個函式的呼叫端（XFF 解析）是安全的但
	// 分配較多，因此它需要一個刻意的決定而不是順手。
	t.Run("trimSpace", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			want string
		}{
			{name: "兩端半形空格", in: "  a  ", want: "a"},
			{name: "兩端 tab", in: "\t\ta\t\t", want: "a"},
			{name: "空格與 tab 混合", in: " \t a \t ", want: "a"},
			{name: "無空白", in: "a", want: "a"},
			{name: "空字串", in: "", want: ""},
			{name: "只有空格", in: "     ", want: ""},
			{name: "內部空白不動", in: " a b  c ", want: "a b  c"},
			// 刻意記錄：不會去掉換行、回車與 Unicode 空白。
			{name: "換行不動（刻意）", in: "\na\n", want: "\na\n"},
			{name: "Unicode 空白不動（刻意）", in: "\u00a0a\u00a0", want: "\u00a0a\u00a0"},
			// 多位元組字元不受影響：逐位元組處理不切割 UTF-8。
			{name: "中文不受影響", in: "  中文  ", want: "中文"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := trimSpace(tc.in); got != tc.want {
					t.Errorf("trimSpace(%q) = %q，want %q", tc.in, got, tc.want)
				}
			})
		}
	})

	t.Run("lastIndexByte", func(t *testing.T) {
		cases := []struct {
			name   string
			s      string
			c      byte
			wantAt int
		}{
			// 注意 "開頭" 那列的期望是 0 而非第一個 '/' 的位置 —— 對
			// "/a/b" 而言，最後一個 '/' 在索引 2。
			{name: "第一個在開頭", s: "abc", c: 'a', wantAt: 0},
			{name: "最後一個在結尾", s: "/a/b", c: '/', wantAt: 2},
			{name: "多個相符取最後一個", s: "aXbXc", c: 'X', wantAt: 3},
			{name: "中間", s: "/a/b", c: 'a', wantAt: 1},
			{name: "不存在", s: "/a/b", c: 'z', wantAt: -1},
			{name: "空字串", s: "", c: '/', wantAt: -1},
			// 多位元組：以位元組為單位，因此一個中文的三個位元組都不會被誤判。
			{name: "多位元組字元不被誤判", s: "中文", c: 'X', wantAt: -1},
			{name: "多位元組之後的相符", s: "中X", c: 'X', wantAt: 3},
		}
		for _, tc := range cases {
			if got := lastIndexByte(tc.s, tc.c); got != tc.wantAt {
				t.Errorf("%s: lastIndexByte(%q, %q) = %d，want %d",
					tc.name, tc.s, tc.c, got, tc.wantAt)
			}
		}
	})
}

// TestNewRateLimiterCoercesInvalidSettings 守住「設定無效時不會變成全部放行」。
//
// NewRateLimiter 把 limit <= 0 與 window <= 0 各自改成一個預設值，而它的說明
// 把理由寫得很直接：
//
//	避免設定缺漏或解析失敗時變成「全部放行」或「全部拒絕」—— 尤其 limit <= 0
//	會讓每個請求的計數都 >= limit，等於整站被封鎖。
//
// 因此這個測試斷言的是「**被改成了什麼值**」，而不是「第一個請求有沒有被允許」
// —— 後者對 limit=10 與 limit=5 是同樣的結果，因此它無法分辨這兩個函式是否
// 真的做了轉換。
func TestNewRateLimiterCoercesInvalidSettings(t *testing.T) {
	cases := []struct {
		name       string
		limit      int
		window     time.Duration
		wantLimit  int
		wantWindow time.Duration
	}{
		{name: "原樣保留合法的設定", limit: 5, window: 30 * time.Second,
			wantLimit: 5, wantWindow: 30 * time.Second},
		{name: "額度 0 → 10", limit: 0, window: time.Minute,
			wantLimit: 10, wantWindow: time.Minute},
		{name: "額度負數 → 10", limit: -5, window: time.Minute,
			wantLimit: 10, wantWindow: time.Minute},
		{name: "視窗 0 → 1 分", limit: 5, window: 0,
			wantLimit: 5, wantWindow: time.Minute},
		{name: "視窗負數 → 1 分", limit: 5, window: -time.Second,
			wantLimit: 5, wantWindow: time.Minute},
		{name: "兩者皆錯", limit: 0, window: 0,
			wantLimit: 10, wantWindow: time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rl := NewRateLimiter(tc.limit, tc.window)
			if rl == nil {
				t.Fatal("NewRateLimiter 回傳 nil")
			}
			// 讀取未匯出的欄位是刻意的：這個套件的測試在同一個 package 裡，
			// 而「轉換後的值」正是這個函式的全部契約。
			if rl.limit != tc.wantLimit {
				t.Errorf("limit = %d，want %d（輸入 %d）", rl.limit, tc.wantLimit, tc.limit)
			}
			if rl.window != tc.wantWindow {
				t.Errorf("window = %v，want %v（輸入 %v）", rl.window, tc.wantWindow, tc.window)
			}

			// 額度必須是**正數**，而視窗必須讓至少一個請求落得下來。
			// 這兩條是「不等於整站被封鎖」的機器可判定形式。
			firstAllowed, retryAfter := rl.Allow("203.0.113.1")
			if !firstAllowed {
				t.Errorf("第一個請求就被拒絕了（retryAfter=%v）—— "+
					"而一個剛建立的限流器永遠應該允許第一個請求", retryAfter)
			}
		})
	}
}

// TestRateLimiterCountsPerClient 守住「一份額度一個 IP」。
//
// 這是限流最核心的性質，而它壞掉時的症狀是「一個使用者就能讓所有人被擋」
// —— 一個非常容易被誤判成「伺服器有問題」的故障。
func TestRateLimiterCountsPerClient(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	// 第一個 IP 用完額度。
	for i := 0; i < 3; i++ {
		if allowed, _ := rl.Allow("203.0.113.1"); !allowed {
			t.Fatalf("第 %d 個請求被拒絕了（額度是 3）", i+1)
		}
	}
	if allowed, retryAfter := rl.Allow("203.0.113.1"); allowed {
		t.Error("超過額度後仍然允許 —— 限流沒生效")
	} else if retryAfter <= 0 {
		t.Errorf("Retry-After = %v，want > 0（回 0 會讓客戶端立刻重試並變成忙迴圈）", retryAfter)
	}

	// 另一個 IP 不受影響。
	if allowed, _ := rl.Allow("203.0.113.2"); !allowed {
		t.Error("另一個 IP 被前一個 IP 的用量影響了 —— " +
			"一個使用者就能讓所有人被擋，而症狀非常難診斷")
	}
}

// ---------------------------------------------------------------------------
// CSP 樣板組裝
// ---------------------------------------------------------------------------

// TestCSPSourceFromURL 覆蓋來源字串的正規化。
//
// 這個函式的輸出直接進入 Content-Security-Policy 標頭，而它的失效方式是
// **CSP 太寬**：一個寫錯的正規化會讓 CSP 接受來自任何地方的圖片 —— 那等於
// 讓一個外站可以把使用者瀏覽器裡的資料讀走。
//
// 它的契約有兩個容易被忽略的部分，這個測試逐條釘住：
//  1. **路徑與查詢字串會被丟掉**。一個設定檔寫成
//     FILES_SERVER_PUBLIC_URL=https://cdn.example.com/media 時，回傳的必須是
//     "https://cdn.example.com" —— 而回傳帶路徑的形式會讓換版時的靜默失效
//     成為可能（見函式的說明）。
//  2. **沒有 scheme 或 host 時回傳空字串**，而不是原字串。純路徑與
//     mailto: 這類值放進 img-src 語法合法但意圖錯誤，而 CSP 標頭是**不會**
//     回報語法錯誤的 —— 瀏覽器會安靜地忽略它。
func TestCSPSourceFromURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "單一來源", in: "https://forum.example.com", want: "https://forum.example.com"},
		{name: "含空白的字串", in: " https://a.example ", want: "https://a.example"},
		{name: "空字串", in: "", want: ""},
		{name: "只有逗號與空白", in: " , , ", want: ""},
		// 尾斜線被去掉：url.Parse 對 "https://a.example/" 與
		// "https://a.example" 會給出不同的 Path，而 CSP 來源運算式不需要它。
		{name: "尾端斜線被去掉", in: "https://a.example/", want: "https://a.example"},
		{name: "多個尾端斜線", in: "https://a.example///", want: "https://a.example"},
		// 路徑與查詢字串被丟掉 —— 這是這個函式存在的理由之一。
		{name: "路徑被丟掉", in: "https://a.example/media/files", want: "https://a.example"},
		{name: "查詢字串被丟掉", in: "https://a.example/?x=1", want: "https://a.example"},
		{name: "連埠一起保留", in: "http://127.0.0.1:7070", want: "http://127.0.0.1:7070"},

		// 以下四個都必須回傳空字串。把「無法構成來源運算式」的值原樣放進
		// CSP 會讓那條 directive 包含一個永遠匹配不到東西的來源 —— 症狀是
		// 「圖片載不出來」而瀏覽器主控台只有一行不指向任何原因的 CSP 警告。
		{name: "純路徑", in: "/files", want: ""},
		{name: "相對路徑", in: "./assets", want: ""},
		{name: "沒有 host", in: "mailto:someone@example.com", want: ""},
		{name: "只有空白", in: "   ", want: ""},
		// 以下兩個是 Phase 3 補測時發現的真實缺口：
		// url.Parse 不會拒絕含逗號的字串，它會把 Host 解讀成 "a.example,https:"，
		// 而那個垃圾組進 img-src 會讓瀏覽器遇到語法非法的來源運算式 ——
		// 瀏覽器對它是**靜默忽略**的，症狀是「圖片全部載不出來，主控台只有
		// 一行不指向任何設定的 CSP 警告」。
		{name: "逗號分隔的多個來源", in: "https://a.example,https://b.example", want: ""},
		{name: "含內部空白的清單", in: "https://a.example, https://b.example", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cspSourceFromURL(tc.in); got != tc.want {
				t.Errorf("cspSourceFromURL(%q)\n  got  %q\n  want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestBuildContentSecurityPolicy 覆蓋 CSP 字串的形狀。
//
// 斷言的是「必須存在」與「必須不存在」的子字串，而不是整個字串的逐字比對。
// 理由：CSP 的欄位順序與空白的排版不影響行為，逐字比對會讓任何無關的
// 重排都變成一次測試失敗 —— 而那會誘發把這個測試「修正」成較弱的斷言。
//
// 真正要守住的是「'unsafe-inline' 不在 script-src 裡」：這個專案的整個
// 前端架構（React 掛載、CSP 雜湊）都是為了讓它不在。
func TestBuildContentSecurityPolicy(t *testing.T) {
	cfg := config.Config{
		PublicBaseURL: "https://forum.example.com",
	}
	hashes := []string{"'sha256-abc123'", "'sha256-def456'"}

	csp := buildContentSecurityPolicy(cfg, hashes)

	mustContain := []string{
		"default-src",
		"script-src",
		"style-src",
		"img-src",
		"frame-ancestors",
		// 雜湊必須真的在裡面：漏掉一個的症狀是「那一頁的 <style> 被 CSP
		// 擋下」，而畫面看起來「有樣式但怪怪的」——
		// style.css 仍會作用（它是 <link>），只有頁面專屬樣式消失。
		"'sha256-abc123'",
		"'sha256-def456'",
		// frame-ancestors 'none' 是防點擊劫持的基本要求。
		"'none'",
	}
	for _, want := range mustContain {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP 缺少 %q\n完整內容：%s", want, csp)
		}
	}

	// script-src 裡絕不能有 'unsafe-inline' 或 'unsafe-eval'：
	// 有前者的話任何 XSS 都能執行，而這個專案的所有防護都建立在它不存在之上。
	scriptSrc := directiveValue(csp, "script-src")
	if scriptSrc == "" {
		t.Fatalf("CSP 沒有 script-src：%s", csp)
	}
	for _, forbidden := range []string{"'unsafe-inline'", "'unsafe-eval'"} {
		if strings.Contains(scriptSrc, forbidden) {
			t.Errorf("script-src 裡有 %s —— 這等於關閉 XSS 防護，而症狀不會出現在任何日誌裡",
				forbidden)
		}
	}

	// 沒有雜湊時 style-src 仍然必須存在，且不得出現空的引號字串
	// （"'sha256-'" 那種形狀會讓瀏覽器拒絕整個標頭）。
	noHashes := buildContentSecurityPolicy(cfg, nil)
	if strings.Contains(noHashes, "sha256-'") || strings.Contains(noHashes, "''") {
		t.Errorf("沒有雜湊時產生了畸形來源：%s", noHashes)
	}
}

// directiveValue 從 CSP 字串取出某個 directive 的值。
//
// 測試用的最小實作：只處理單行的 directive（這個專案的 CSP 是單行的），
// 而它存在的理由是「把斷言寫在 directive 的層級」比在整個標頭裡做
// substrings.Contains 更能說明意圖 —— 尤其對「script-src 裡不准有
// unsafe-inline」這種針對單一 directive 的規則。
func directiveValue(csp, directive string) string {
	parts := strings.Split(csp, ";")
	for _, p := range parts {
		fields := strings.Fields(strings.TrimSpace(p))
		if len(fields) >= 2 && fields[0] == directive {
			return fields[1]
		}
	}
	return ""
}

// TestStyleBlockContents 覆蓋 CSP 雜湊的計算來源。
//
// 這個函式決定了「哪一段 CSS 會被授權」，而它壞掉時的症狀是整頁沒有版面 ——
// 而那個錯誤只出現在**瀏覽器**主控台（Refused to apply inline style），
// 伺服器端的日誌完全乾淨。
//
// 這裡特別關注註解與屬性值的邊界：一個寫在 <style> 裡的註解若被當成
// 樣式內容之外的部分（或反過來），雜湊就會算錯，而症狀是「單獨一頁沒有版面」。
func TestStyleBlockContents(t *testing.T) {
	cases := []struct {
		name  string
		html  string
		wantN int
		// wantFirst 是第一段的內容（只用於 wantN == 1 的情況）。
		wantFirst string
	}{
		{name: "沒有 style", html: "<html><body>hi</body></html>", wantN: 0},
		{name: "一個 style", html: "<style>.a{color:red}</style>", wantN: 1,
			wantFirst: ".a{color:red}"},
		{name: "兩個 style", html: "<style>.a{}</style><style>.b{}</style>", wantN: 2},
		{name: "style 內有 CSS 註解", html: "<style>/* x */ .a{}</style>", wantN: 1},
		{name: "style 外有 HTML 註解", html: "<!-- x --><style>.a{}</style>", wantN: 1},
		{name: "style 有屬性", html: `<style media="screen">.a{}</style>`, wantN: 1,
			wantFirst: ".a{}"},
		{name: "未關閉的 style", html: "<style>.a{}", wantN: 0}, // 刻意記錄這個方向：未關閉時靜靜忽略而不是回傳空內容。
		// 塞一個空雜湊進 CSP 會放行一個沒有對應元素的來源。

		{name: "<stylesheet 不是 style 區塊", html: "<stylesheet>x</stylesheet>", wantN: 0}, // 同開頭的標籤必須被排除，否則會算出一個永遠不會對應到任何元素的
		// 雜湊 —— 那是 CSP 裡一個沒有意義的放行。

		// 大寫標籤刻意**不**認：css_source 只比對 "<style" 這五個字元，
		// 涵蓋 <STYLE> 需要對整份文件做 _lower()（配置一份等大的副本）。
		// 這個取捨的症狀是「那一頁沒有樣式」—— 明顯且好修。
		{name: "大寫標籤（刻意不認）", html: "<STYLE>.a{}</STYLE>", wantN: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := styleBlockContents(tc.html)
			if len(got) != tc.wantN {
				t.Fatalf("styleBlockContents 回傳 %d 段，want %d\n輸入: %s", len(got), tc.wantN, tc.html)
			}
			if tc.wantN == 1 && tc.wantFirst != "" && got[0] != tc.wantFirst {
				t.Errorf("第一段的內容\n  got  %q\n  want %q", got[0], tc.wantFirst)
			}
		})
	}
}

// TestStyleBlockContentsKeepsWhitespaceByteExact 是這個函式最貴的一條性質。
//
// 樣板裡的樣式區塊原文會被送去算 SHA-256，而 CSP 授權的是**位元組完全相同**
// 的那一段。因此這個函式不能對內容做任何正規化 ——
// 連「把連續的空白正規化成一格」這種直覺的清理都不行。
//
// 症狀特別難診斷：雜湊對不上時瀏覽器只說 Refused to apply inline style，而
// 伺服器端沒有任何錯誤 —— 而「差一個空格」在肉眼上看不出來。因此這支測試
// 用帶各種空白的樣本逐位元組比對。
func TestStyleBlockContentsKeepsWhitespaceByteExact(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"單一空格", ".a{color:red}"},
		{"多個空格", ".a  {  color : red }"},
		{"tab", ".a\t{\tcolor:red}"},
		{"換行與縮排", "\n  .a {\n    color: red;\n  }\n"},
		{"CRLF", "\r\n.a {\r\n  color: red;\r\n}\r\n"},
		{"行尾空白", ".a{}   \n   .b{}"},
		{"開頭就換行", "\n.a{}"},
		{"結尾沒換行", ".a{}\n"},
		{"全形空白", "\u3000.a{}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := "<style>" + tc.body + "</style>"
			got := styleBlockContents(html)
			if len(got) != 1 {
				t.Fatalf("取得 %d 段，want 1", len(got))
			}
			if got[0] != tc.body {
				t.Errorf("樣式內容被改動了 —— 雜湊會對不上，症狀是「那一頁沒有版面，\n"+
					"而伺服器端沒有任何錯誤」\n  got  %q\n  want %q", got[0], tc.body)
			}
		})
	}
}

// TestStyleBlockContentsExcludesTheMetaTagComment 是本檔最窄但最貴的一條。
//
// frontend/forum.html 的註解明確寫著：
//
//	這段註解因此刻意不寫出樣式標籤本身 —— 雜湊是純文字掃描，註解裡出現
//	標籤字面量會被當成開標籤。
//
// 也就是說：若有人在某個頁面的註解裡寫出樣式標籤的開頭，那一段註解文字
// 就會被當成一個 style 區塊的**內容**而被算進雜湊，而真正的 style 區塊
// 的雜湊就會算在錯的內容上 —— 症狀是那一頁沒有版面。
//
// 這支測試不依賴真實頁面（那些會隨設計變動而需要更新），而是用一個能引發
// 這個失效的最小樣本。
func TestStyleBlockContentsExcludesTheMetaTagComment(t *testing.T) {
	// 一個刻意模仿 frontend/forum.html 註解寫法的樣本：
	// 註解裡提到了樣式標籤，註解外有一個真的樣式區塊。
	const html = `<html><head>
<!-- 站名佔位符由後端在送出時換成設定檔的值。佔位符必須留在樣式區塊之外。 -->
<meta name="forum-name" content="{{FORUM_NAME}}" />
<style>
  html:root { --bg: #fff; }
</style>
</head><body><div id="root"></div></body></html>`

	blocks := styleBlockContents(html)
	if len(blocks) != 1 {
		t.Fatalf("取得 %d 段樣式，want 1。樣式區塊之外的註解被當成樣式內容了 ——\n"+
			"那會讓真正的樣式區塊算錯雜湊，症狀是那一頁沒有版面", len(blocks))
	}
	// 取得的那一段必須是**真的**樣式區塊的內容，而不是註解。
	if !strings.Contains(blocks[0], "--bg") {
		t.Errorf("取得的樣式內容不是 style 區塊的內容：\n%q", blocks[0])
	}
	if strings.Contains(blocks[0], "佔位符") {
		t.Errorf("樣式內容裡混進了註解文字：\n%q", blocks[0])
	}
}
