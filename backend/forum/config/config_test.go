/*
config_test.go 覆蓋設定解析層的全部純函式。

【為什麼這個檔是「最便宜的高價值覆蓋」】
設定解析失敗的症狀是部署後才浮現的**靜默失效**，而且每一個都有明確的症狀、
沒有一個會報錯：

  - ALLOWED_ADMIN_EMAIL 寫成帶空白或大寫 → 管理員被鎖在後台外，沒有任何
    錯誤訊息（Google 回傳的 email 已是穩定的小寫）
  - 限流視窗寫成 0 或負數 → 限流形同不存在
  - MEDIA_TOKEN_TTL_SECONDS 漏設 → 圖片存取權杖存活 30 天
  - SHUTDOWN_TIMEOUT_SECONDS 寫成 0 → 優雅停止被整個關掉，卻沒有任何提示

這些函式全部是純函式（有界、無 I/O），因此測試它們的成本接近零，而它們守住
的東西是整個部署能否以正確的行為運作。

【這個檔的測試方法：表格測試】
每一組都用「輸入字串 → 期望結果」的表格。理由不是慣例，而是這個設定格式裡
有大量「同樣是壞的、但壞法不同」的輸入（打錯字、大小寫、空白、負數、非數字），
而它們的期望結果往往相同 —— 把它們列在一張表裡，那份清單本身就是規格，
而逐個寫成獨立的 if 會讓「漏掉哪一種壞法」看不出來。

【順帶守住的一條設計決定】
「打錯字」與「沒設定」無法區分這件事是刻意且危險的（見 config.go 的檔頭）。
因此測試刻意**不**把「RATE_LIMIT_WINDOW_SECONDS=abc 應該報錯」寫成期望 ——
那會把一個刻意的設計決策變成一個待修的缺陷。取而代之的是測試「它會沿用預設值」
這個實際行為，並在測試名稱裡說明為什麼。
*/
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeConfig 寫出一份暫時的設定檔並回傳路徑。
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.conf")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("寫入暫時設定檔失敗: %v", err)
	}
	return path
}

// loadString 解析一份設定檔內容並回傳 Config。
func loadString(t *testing.T, body string) Config {
	t.Helper()
	cfg, err := Load(writeConfig(t, body))
	if err != nil {
		t.Fatalf("Load 回傳錯誤（這份設定應該是合法的）: %v", err)
	}
	return cfg
}

// TestParseBool 覆蓋 parseBool。
//
// 方向刻意是「明確為真才成立」：flase（打錯字）必須是 false。這個方向的代價是
// 打錯字會讓安全旗標失效，因此每一個把它設成 true 的設定都要有明確的值 ——
// 這個測試是那個假設的守門人。
func TestParseBool(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		// 這些是「常見的寫錯」，全部必須是 false。
		{"flase", false}, // 打錯字
		{"t", false},
		{"y", false},
		{"enabled", false},
		{"", false},
		{"0", false},
		{"no", false},
		{"off", false},
		// 空白容忍：設定檔裡 "COOKIE_SECURE= true" 是很容易打出來的。
		{"  true  ", true},
		{"\ttrue\t", true},
		{"  flase  ", false},
	}
	for _, tc := range cases {
		if got := parseBool(tc.in); got != tc.want {
			t.Errorf("parseBool(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestParseList 覆蓋 parseList 的三個分支：空輸入、有內容、全是空項目。
func TestParseList(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
		// wantNil 區分「回傳 nil」與「回傳空切片」—— 兩者對呼叫端的
		// len() 檢查一樣，但對 %v 輸出不一樣，而 debug 時看得出差別。
		wantNil bool
	}{
		{name: "空字串", in: "", want: nil, wantNil: true},
		{name: "單一項", in: "a@example.com", want: []string{"a@example.com"}},
		{name: "多項", in: "a@example.com,b@example.com", want: []string{"a@example.com", "b@example.com"}},
		{name: "逗號後有空白", in: "a@example.com, b@example.com", want: []string{"a@example.com", "b@example.com"}},
		{name: "多餘的空白", in: "  a@example.com  ,  b@example.com  ", want: []string{"a@example.com", "b@example.com"}},
		{name: "結尾逗號", in: "a@example.com,", want: []string{"a@example.com"}},
		{name: "開頭逗號", in: ",a@example.com", want: []string{"a@example.com"}},
		{name: "連續逗號", in: "a,,b", want: []string{"a", "b"}},
		{name: "只有逗號", in: ",,,", want: []string{}},
		{name: "只有空白", in: "   ", want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseList(tc.in)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("parseList(%q) = %v, want nil", tc.in, got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("parseList(%q) = %v, want %v", tc.in, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("parseList(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestParsePositiveInt 覆蓋 parsePositiveInt 拒絕 0 與負數的行為。
//
// 0 與負數被拒絕不是細節：次數為 0 會讓「每個請求的計數都 >= limit」，
// 等於整站被封鎖；負數則語意不明。兩者都是靜默的嚴重錯誤。
func TestParsePositiveInt(t *testing.T) {
	cases := []struct {
		in     string
		want   int
		wantOK bool
	}{
		{in: "5", want: 5, wantOK: true},
		{in: "1", want: 1, wantOK: true},
		// 刻意記錄「空白容忍發生在 Load 而不在這裡」：Load 對每個值都做了
		// TrimSpace，所以 parsePositiveInt 永遠收不到帶空白的輸入。
		// 若將來有人把 Load 的 TrimSpace 拿掉，這一條會變成一道防線 ——
		// 而在那之前，它說明的是這個函式的邊界在哪。
		{in: " 7 ", want: 0, wantOK: false},
		{in: "0", want: 0, wantOK: false},
		{in: "-1", want: 0, wantOK: false},
		{in: "-100", want: 0, wantOK: false},
		{in: "abc", want: 0, wantOK: false},
		{in: "5x", want: 0, wantOK: false},
		{in: "", want: 0, wantOK: false},
		{in: "3.5", want: 0, wantOK: false},
	}
	for _, tc := range cases {
		got, ok := parsePositiveInt(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("parsePositiveInt(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

// TestParsePositiveSeconds 覆蓋 parsePositiveSeconds，並說明它拒絕 0 的理由。
//
// 0 語意是「視窗長度為零」，那會讓 cutoff 等於 now，所有時間戳都被視為過期
// → 限流形同不存在。負值則讓 cutoff 落在未來 → 每個請求都在視窗內 → 整站
// 被立刻封鎖。兩個方向都是靜默的，因此這個函式寧可拒絕也不猜。
func TestParsePositiveSeconds(t *testing.T) {
	cases := []struct {
		in     string
		want   time.Duration
		wantOK bool
	}{
		{in: "30", want: 30 * time.Second, wantOK: true},
		{in: "1", want: time.Second, wantOK: true},
		// 空白由 Load 處理，不在這裡（理由同 parsePositiveInt 的那一條）。
		{in: " 60 ", want: 0, wantOK: false},
		{in: "0", want: 0, wantOK: false},
		{in: "-5", want: 0, wantOK: false},
		{in: "abc", want: 0, wantOK: false},
		{in: "", want: 0, wantOK: false},
	}
	for _, tc := range cases {
		got, ok := parsePositiveSeconds(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("parsePositiveSeconds(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

// TestRateLimitValuesFallBackWhenUnusable 覆蓋最危險的一組靜默失效。
//
// 三組限流設定在 applyDefaults 裡都沒有兜底 —— 它們的值是在 Load 開頭預先
// 寫入的。因此「設定檔把它們寫成壞值」這個路徑必須靠「看 ok 再賦值」的寫法
// 擋住，而不是靠套預設。這支測試把那個寫法的後果釘住。
//
// 順帶固定一個刻意的取捨：打錯字（SESSION_EXPIRE_HOURS=abc）與沒設定無法
// 區分，結果是沿用預設值。它**不是**待修的缺陷 —— 見 config.go 的檔頭。
func TestRateLimitValuesFallBackWhenUnusable(t *testing.T) {
	t.Run("打錯字時沿用預設值（刻意，不是缺陷）", func(t *testing.T) {
		cfg := loadString(t, "RATE_LIMIT_WINDOW_SECONDS=abc\n")
		if cfg.RateLimitWindow != time.Minute {
			t.Errorf("RateLimitWindow = %v，want 1 分鐘（Load 開頭的預設值）", cfg.RateLimitWindow)
		}
	})

	cases := []struct {
		name string
		body string
	}{
		{"視窗為 0", "RATE_LIMIT_WINDOW_SECONDS=0\n"},
		{"視窗為負數", "RATE_LIMIT_WINDOW_SECONDS=-60\n"},
		{"上傳視窗為 0", "RATE_LIMIT_UPLOAD_WINDOW_SECONDS=0\n"},
		{"OAuth 視窗為 0", "RATE_LIMIT_AUTH_WINDOW_SECONDS=0\n"},
		{"次數為 0", "RATE_LIMIT_REQUESTS=0\n"},
		{"次數為負數", "RATE_LIMIT_REQUESTS=-1\n"},
		{"上傳次數為 0", "RATE_LIMIT_UPLOAD_REQUESTS=0\n"},
		{"OAuth 次數為 0", "RATE_LIMIT_AUTH_REQUESTS=0\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadString(t, tc.body)
			// 任何一個為零都代表「限流失效」或「整站被封鎖」，
			// 而兩者都不會出現在任何日誌或狀態碼裡。
			if cfg.RateLimitRequests <= 0 || cfg.RateLimitWindow <= 0 ||
				cfg.RateLimitUploadRequests <= 0 || cfg.RateLimitUploadWindow <= 0 ||
				cfg.RateLimitAuthRequests <= 0 || cfg.RateLimitAuthWindow <= 0 {
				t.Errorf("壞的設定值被接受了：write=%d/%v upload=%d/%v auth=%d/%v",
					cfg.RateLimitRequests, cfg.RateLimitWindow,
					cfg.RateLimitUploadRequests, cfg.RateLimitUploadWindow,
					cfg.RateLimitAuthRequests, cfg.RateLimitAuthWindow)
			}
		})
	}
}

// TestValidRateLimitsAreAccepted 是上一支測試的另一半：合法的值必須真的被採用。
//
// 沒有這一支的話，「一律沿用預設值」也能通過上一支 —— 而那正是限流設定
// 完全失效的形式（所有額度都變成預設，站方改設定沒有任何效果）。
func TestValidRateLimitsAreAccepted(t *testing.T) {
	cfg := loadString(t, strings.Join([]string{
		"RATE_LIMIT_REQUESTS=42",
		"RATE_LIMIT_WINDOW_SECONDS=90",
		"RATE_LIMIT_UPLOAD_REQUESTS=7",
		"RATE_LIMIT_UPLOAD_WINDOW_SECONDS=120",
		"RATE_LIMIT_AUTH_REQUESTS=3",
		"RATE_LIMIT_AUTH_WINDOW_SECONDS=30",
	}, "\n"))

	want := Config{
		RateLimitRequests:       42,
		RateLimitWindow:         90 * time.Second,
		RateLimitUploadRequests: 7,
		RateLimitUploadWindow:   120 * time.Second,
		RateLimitAuthRequests:   3,
		RateLimitAuthWindow:     30 * time.Second,
	}
	got := []struct {
		name      string
		got, want any
	}{
		{"RATE_LIMIT_REQUESTS", cfg.RateLimitRequests, want.RateLimitRequests},
		{"RATE_LIMIT_WINDOW_SECONDS", cfg.RateLimitWindow, want.RateLimitWindow},
		{"RATE_LIMIT_UPLOAD_REQUESTS", cfg.RateLimitUploadRequests, want.RateLimitUploadRequests},
		{"RATE_LIMIT_UPLOAD_WINDOW_SECONDS", cfg.RateLimitUploadWindow, want.RateLimitUploadWindow},
		{"RATE_LIMIT_AUTH_REQUESTS", cfg.RateLimitAuthRequests, want.RateLimitAuthRequests},
		{"RATE_LIMIT_AUTH_WINDOW_SECONDS", cfg.RateLimitAuthWindow, want.RateLimitAuthWindow},
	}
	for _, c := range got {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v（合法的設定值必須被採用）", c.name, c.got, c.want)
		}
	}
}

// TestTimeoutsRejectNonPositive 守住兩���逾時設定不能被設成 0。
//
// 0 語意是「不設期限」或「立即強制關閉」。前者是 Slowloris 的解藥被關掉，
// 後者等於把「優雅停止」整個關掉卻又不會有任何提示 —— 兩個都是靜默的。
func TestTimeoutsRejectNonPositive(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"0", "READ_HEADER_TIMEOUT_SECONDS=0\n"},
		{"負數", "READ_HEADER_TIMEOUT_SECONDS=-1\n"},
		{"打錯字", "READ_HEADER_TIMEOUT_SECONDS=abc\n"},
	}
	for _, tc := range cases {
		t.Run("READ_HEADER_TIMEOUT_SECONDS="+tc.name, func(t *testing.T) {
			cfg := loadString(t, tc.body)
			if cfg.ReadHeaderTimeout != 10*time.Second {
				t.Errorf("ReadHeaderTimeout = %v, want 10 秒（預設值必須被保留）", cfg.ReadHeaderTimeout)
			}
		})
		t.Run("SHUTDOWN_TIMEOUT_SECONDS="+tc.name, func(t *testing.T) {
			cfg := loadString(t, tc.body)
			if cfg.ShutdownTimeout != 15*time.Second {
				t.Errorf("ShutdownTimeout = %v, want 15 秒（預設值必須被保留）", cfg.ShutdownTimeout)
			}
		})
	}
}

// TestAdminEmailNormalization 覆蓋 ROADMAP.md Phase 0.4 的修正。
//
// 這是整個設定檔裡最典型的靜默失效：白名單寫成帶空白或大寫時，管理員被鎖在
// 後台外，而症狀是「沒有任何錯誤訊息」（Google 回傳的 email 已是穩定的小寫）。
//
// 測試涵蓋規格要求的三種輸入（帶空白、大寫、混合），並額外釘住「兩邊都正規化」
// 這個方向：只正規化白名單而不正規化比對值，同樣會靜默失效。
func TestAdminEmailNormalization(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"基準", "admin@example.com"},
		{"帶空白", " admin@example.com "},
		{"全大寫", "ADMIN@EXAMPLE.COM"},
		{"混合大小寫", "AdMiN@ExAmPlE.CoM"},
		{"多項且各帶空白", " admin@example.com , root@example.com "},
		{"多項且大小寫不一", "Admin@example.com, ROOT@EXAMPLE.COM"},
		{"換行與 Tab 混雜", "\tAdmin@example.com\t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadString(t, "ALLOWED_ADMIN_EMAIL="+tc.value+"\n")

			// 載入後的值必須已經正規化 —— 這是「只正規化一邊」的陷阱：
			// 若這裡斷言的是比對結果而沒有斷言儲存的內容，
			// 一個把正規化放在 IsAdminEmail 裡的實作也會通過，
			// 而那會讓 debug 時看到未正規化的設定值而困惑。
			for _, stored := range cfg.AllowedAdminEmail {
				if stored != strings.ToLower(stored) {
					t.Errorf("白名單項目 %q 仍是混合大小寫：載入時就該轉小寫", stored)
				}
				if stored != strings.TrimSpace(stored) {
					t.Errorf("白名單項目 %q 仍有頭尾空白：載入時就該去除", stored)
				}
			}

			// 比對的三個方向都要成立。
			for _, probe := range []string{
				"admin@example.com",
				" admin@example.com ",
				"ADMIN@EXAMPLE.COM",
				"Admin@Example.com",
			} {
				if !cfg.IsAdminEmail(probe) {
					t.Errorf("IsAdminEmail(%q) = false, want true（白名單為 %v）", probe, cfg.AllowedAdminEmail)
				}
			}
		})
	}
}

// TestAdminEmailNotInWhitelist 守住「正規化不會讓白名單變成萬用字元」。
func TestAdminEmailNotInWhitelist(t *testing.T) {
	cfg := loadString(t, "ALLOWED_ADMIN_EMAIL= admin@example.com , root@example.com \n")

	rejected := []struct {
		name  string
		email string
	}{
		{"不同信箱", "other@example.com"},
		{"空字串", ""},
		{"只有空白", "   "},
		// 刻意收窄的樣本：正規化只做 TrimSpace + ToLower，因此前綴／子字串
		// 都不該命中。若將來有人加入 Gmail 點號之類的規則，這一條要重新評估。
		{"前綴相同", "admin@example.com.evil.com"},
		{"前綴有加號", "admin+tag@example.com"},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			if cfg.IsAdminEmail(tc.email) {
				t.Errorf("IsAdminEmail(%q) = true，want false（白名單 %v 不該命中）",
					tc.email, cfg.AllowedAdminEmail)
			}
		})
	}
}

// TestAdminEmailEmptyWhitelistMatchesNothing 守住空白名單不會誤授權。
//
// 這一條特別重要：parseList 丟掉空項目，而 normalizeEmails 再丟一次。
// 若其中一層漏了「空字串」這個處理，「ALLOWED_ADMIN_EMAIL=,," 就會讓
// IsAdminEmail("") 為 true —— 而空 email 正是未登入請求會帶進來的。
func TestAdminEmailEmptyWhitelistMatchesNothing(t *testing.T) {
	for _, body := range []string{
		"ALLOWED_ADMIN_EMAIL=\n",
		"ALLOWED_ADMIN_EMAIL=   \n",
		"ALLOWED_ADMIN_EMAIL=,,,\n",
		"ALLOWED_ADMIN_EMAIL= , , \n",
	} {
		t.Run(strings.TrimSpace(body), func(t *testing.T) {
			cfg := loadString(t, body)
			for _, probe := range []string{"", " ", "admin@example.com"} {
				if cfg.IsAdminEmail(probe) {
					t.Errorf("白名單為空時 IsAdminEmail(%q) = true，want false（欄位 %v）",
						probe, cfg.AllowedAdminEmail)
				}
			}
		})
	}
}

// TestIsProduction 覆蓋「這份設定是不是正式環境」的判斷。
//
// 這個判斷是 Validate 的前提條件，而 Validate 會決定服務是否啟動 —— 因此它
// 判錯的方向會直接變成「正式環境沒被擋」或「開發環境被擋掉」。
func TestIsProduction(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"COOKIE_SECURE=true", "COOKIE_SECURE=true\n", true},
		{"COOKIE_SECURE=1", "COOKIE_SECURE=1\n", true},
		{"COOKIE_SECURE=on", "COOKIE_SECURE=on\n", true},
		{"PUBLIC_BASE_URL 為 https", "PUBLIC_BASE_URL=https://forum.example.com\n", true},
		{"https 且大小寫不同", "PUBLIC_BASE_URL=HTTPS://forum.example.com\n", true},
		// 兩個條件同時成立也只是一個布林結果，不該有任何額外行為。
		{"兩者都成立", "COOKIE_SECURE=true\nPUBLIC_BASE_URL=https://f.example.com\n", true},

		{"開發環境的預設", "", false},
		{"COOKIE_SECURE=false", "COOKIE_SECURE=false\n", false},
		{"COOKIE_SECURE 打錯字", "COOKIE_SECURE=flase\n", false},
		{"PUBLIC_BASE_URL 為 http", "PUBLIC_BASE_URL=http://localhost:8088\n", false},
		{"PUBLIC_BASE_URL 為 http 且 cookie 不安全", "PUBLIC_BASE_URL=http://f.example.com\nCOOKIE_SECURE=false\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadString(t, tc.body)
			if got := cfg.IsProduction(); got != tc.want {
				t.Errorf("IsProduction() = %v, want %v（設定 %q）", got, tc.want, tc.body)
			}
		})
	}
}

// TestValidateRejectsFallbackMediaTokenTTLInProduction 覆蓋 ROADMAP.md Phase 0.5。
//
// 這是「正式環境採用 30 天兜底值」的攔截。它必須只在正式環境生效 —— 在開發
// 環境強制它會讓「忘記設定也能跑」這個刻意的便利消失，而便利正是那個兜底值
// 存在的理由。
func TestValidateRejectsFallbackMediaTokenTTLInProduction(t *testing.T) {
	t.Run("正式環境且未設定 → 報錯", func(t *testing.T) {
		for _, body := range []string{
			"COOKIE_SECURE=true\n",
			"PUBLIC_BASE_URL=https://forum.example.com\n",
		} {
			cfg := loadString(t, body)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("設定 %q 下 Validate() = nil，want 報錯（媒體 token TTL 沿用 30 天兜底值）", body)
			}
			// 訊息必須點名是哪一個設定，以及要怎麼修 —— 一個只說
			// 「設定錯誤」的訊息會讓人回到逐項比對設定檔。
			if !strings.Contains(err.Error(), "MEDIA_TOKEN_TTL_SECONDS") {
				t.Errorf("錯誤訊息沒有點名 MEDIA_TOKEN_TTL_SECONDS: %v", err)
			}
			if !strings.Contains(err.Error(), "2592000") {
				t.Errorf("錯誤訊息沒有說明實際採用的兜底值: %v", err)
			}
		}
	})

	t.Run("正式環境且明確設定 → 通過", func(t *testing.T) {
		cfg := loadString(t, "COOKIE_SECURE=true\nMEDIA_TOKEN_TTL_SECONDS=60\n")
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil（已明確設定 TTL）", err)
		}
	})

	// 「明確設成 30 天」必須與「忘記設定」區分開來 —— 否則 CDN 後面、
	// 圖片本來就該長期可取的部署者會被擋，而他們沒有辦法表達「我知道」。
	t.Run("正式環境且明確設成 30 天 → 通過", func(t *testing.T) {
		cfg := loadString(t, "COOKIE_SECURE=true\nMEDIA_TOKEN_TTL_SECONDS=2592000\n")
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil（刻意設成 30 天是知情同意）", err)
		}
	})

	t.Run("開發環境未設定 → 通過", func(t *testing.T) {
		cfg := loadString(t, "")
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil（開發環境允許採用兜底值）", err)
		}
	})

	// 寫成壞值等同於沒設定 —— 這一條很重要，因為「寫了但沒生效」比「沒寫」
	// 更難察覺：部署者看到設定檔裡有那一行，會合理地以為 TTL 是他設的值。
	t.Run("正式環境但 TTL 寫成壞值 → 報錯", func(t *testing.T) {
		for _, value := range []string{"0", "-1", "abc", ""} {
			cfg := loadString(t, "COOKIE_SECURE=true\nMEDIA_TOKEN_TTL_SECONDS="+value+"\n")
			if err := cfg.Validate(); err == nil {
				t.Errorf("MEDIA_TOKEN_TTL_SECONDS=%q 時 Validate() = nil，want 報錯", value)
			}
		}
	})
}

// TestValidateIsCleanForAProductionConfig 是上一支測試的另一半：一份完整的正式
// 設定必須乾淨通過。
//
// 沒有這一支的話，「Validate 永遠回錯誤」也能通過上一支 —— 而那會讓任何正式
// 部署都開不起來。因此兩支必須同時存在。
func TestValidateIsCleanForAProductionConfig(t *testing.T) {
	cfg := loadString(t, strings.Join([]string{
		"COOKIE_SECURE=true",
		"PUBLIC_BASE_URL=https://forum.example.com",
		"MEDIA_TOKEN_TTL_SECONDS=60",
		"SHUTDOWN_TIMEOUT_SECONDS=15",
		"READ_HEADER_TIMEOUT_SECONDS=10",
		"RATE_LIMIT_REQUESTS=10",
		"RATE_LIMIT_WINDOW_SECONDS=60",
		"RATE_LIMIT_UPLOAD_REQUESTS=5",
		"RATE_LIMIT_UPLOAD_WINDOW_SECONDS=60",
		"RATE_LIMIT_AUTH_REQUESTS=10",
		"RATE_LIMIT_AUTH_WINDOW_SECONDS=60",
		"AUDIT_RETENTION_DAYS=90",
		"MONITOR_RETENTION_HOURS=24",
	}, "\n"))
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil（這是一份完整的正式環境設定）", err)
	}
}

// TestApplyDefaultsCoversEveryEmptyField 覆蓋 applyDefaults 的每一個分支。
//
// 逐欄位斷言「空設定檔 → 有預設值」，理由是這個函式的失敗模式是**靜默的**：
// 漏掉一個分支，那個欄位就是零值，而零值對某些欄位意味著「功能被關掉」
// （SessionExpire = 0 會讓每個 session 立刻過期、TrustedOrigins 空會讓
// 所有來源都視為可信 = 關閉 CSRF 防護）。
func TestApplyDefaultsCoversEveryEmptyField(t *testing.T) {
	cfg := loadString(t, "")

	if cfg.CookieName != "FORUM_forum" {
		t.Errorf("CookieName = %q, want \"FORUM_forum\"", cfg.CookieName)
	}
	if cfg.SessionExpire != 72*time.Hour {
		t.Errorf("SessionExpire = %v, want 72h", cfg.SessionExpire)
	}
	if cfg.ServerPort != ":8088" {
		t.Errorf("ServerPort = %q, want \":8088\"", cfg.ServerPort)
	}
	if cfg.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 10s", cfg.ReadHeaderTimeout)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
	}
	if cfg.PublicBaseURL != "http://localhost:8088" {
		t.Errorf("PublicBaseURL = %q, want \"http://localhost:8088\"（隱含相依於 ServerPort）", cfg.PublicBaseURL)
	}
	// 這個預設值是 CSRF 防護的最後一道防線：白名單空 = 所有來源可信。
	if len(cfg.TrustedOrigins) != 1 || cfg.TrustedOrigins[0] != cfg.PublicBaseURL {
		t.Errorf("TrustedOrigins = %v, want [%s]", cfg.TrustedOrigins, cfg.PublicBaseURL)
	}
	if cfg.FilesServerURL != "http://localhost:7070" {
		t.Errorf("FilesServerURL = %q", cfg.FilesServerURL)
	}
	if cfg.FilesServerPublicURL != cfg.FilesServerURL {
		t.Errorf("FilesServerPublicURL = %q, want %q（未設定時沿用內部位址）",
			cfg.FilesServerPublicURL, cfg.FilesServerURL)
	}
	// 與 session 的 "forum:session:" 前綴必須區隔。
	if cfg.MediaTokenKeyPrefix != "forum:token:" {
		t.Errorf("MediaTokenKeyPrefix = %q, want \"forum:token:\"", cfg.MediaTokenKeyPrefix)
	}
	if cfg.MediaTokenTTLSecs != 2592000 {
		t.Errorf("MediaTokenTTLSecs = %d, want 2592000（30 天開發用兜底值）", cfg.MediaTokenTTLSecs)
	}
	if cfg.RedisAddr != "127.0.0.1:6379" {
		t.Errorf("RedisAddr = %q, want \"127.0.0.1:6379\"", cfg.RedisAddr)
	}
	// ESURL 刻意**不**有預設值：設定了就代表要啟用搜尋，給一個內網預設值會
	// 讓人以為有搜尋功能其實打到別台機器。
	if cfg.ESURL != "" {
		t.Errorf("ESURL = %q, want \"\"（留空＝不啟用 Elasticsearch）", cfg.ESURL)
	}
	if cfg.ESIndex != "forum_posts" {
		t.Errorf("ESIndex = %q, want \"forum_posts\"", cfg.ESIndex)
	}
	if cfg.LogLevel != "INFO" {
		t.Errorf("LogLevel = %q, want \"INFO\"", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want \"text\"", cfg.LogFormat)
	}
	if cfg.AuditRetentionDays != 90 {
		t.Errorf("AuditRetentionDays = %d, want 90", cfg.AuditRetentionDays)
	}
	if cfg.MonitorRetentionHours != 24 {
		t.Errorf("MonitorRetentionHours = %d, want 24", cfg.MonitorRetentionHours)
	}
	// 站名三欄的相依順序：短名與說明預設都取完整站名。
	if cfg.ForumName != "FORUM 論壇" {
		t.Errorf("ForumName = %q", cfg.ForumName)
	}
	if cfg.ForumShortName != cfg.ForumName {
		t.Errorf("ForumShortName = %q, want %q", cfg.ForumShortName, cfg.ForumName)
	}
	if cfg.ForumDescription != cfg.ForumName {
		t.Errorf("ForumDescription = %q, want %q", cfg.ForumDescription, cfg.ForumName)
	}
}

// TestApplyDefaultsFieldOrderingIsLoadBearing 釘住 applyDefaults 裡的順序相依。
//
// 兩處相依是有意的，而它們的失效方式都是「看起來正常的錯值」：
//   - PublicBaseURL 讀 ServerPort。順序反了會得到
//     "http://localhost"（沒有埠）。
//   - ForumShortName / ForumDescription 讀 ForumName。順序反了會讓標誌
//     整顆消失 —— 而設定檔裡明明有站名。
func TestApplyDefaultsFieldOrderingIsLoadBearing(t *testing.T) {
	t.Run("自訂 SERVER_PORT 會帶進 PUBLIC_BASE_URL", func(t *testing.T) {
		cfg := loadString(t, "SERVER_PORT=:9090\n")
		if cfg.PublicBaseURL != "http://localhost:9090" {
			t.Errorf("PublicBaseURL = %q, want \"http://localhost:9090\"", cfg.PublicBaseURL)
		}
		if len(cfg.TrustedOrigins) != 1 || cfg.TrustedOrigins[0] != cfg.PublicBaseURL {
			t.Errorf("TrustedOrigins = %v, want [%s]（必須等 PublicBaseURL 補完）",
				cfg.TrustedOrigins, cfg.PublicBaseURL)
		}
	})

	t.Run("自訂 FORUM_NAME 會帶進短名與說明", func(t *testing.T) {
		cfg := loadString(t, "FORUM_NAME=某某論壇\n")
		if cfg.ForumShortName != "某某論壇" {
			t.Errorf("ForumShortName = %q, want \"某某論壇\"", cfg.ForumShortName)
		}
		if cfg.ForumDescription != "某某論壇" {
			t.Errorf("ForumDescription = %q, want \"某某論壇\"", cfg.ForumDescription)
		}
	})

	t.Run("短名與說明可以獨立覆寫", func(t *testing.T) {
		cfg := loadString(t, "FORUM_NAME=某某論壇\nFORUM_SHORT_NAME=某某\nFORUM_DESCRIPTION=一個論壇\n")
		if cfg.ForumShortName != "某某" || cfg.ForumDescription != "一個論壇" {
			t.Errorf("短名/說明被覆寫成了 %q / %q", cfg.ForumShortName, cfg.ForumDescription)
		}
	})
}

// TestTrailingSlashIsTrimmed 覆蓋那些「去掉尾斜線」的欄位。
//
// 尾斜線留下的症狀是 "//forum_posts" 這種語意不對但仍可請求的網址 ——
// 它不會報錯，只會讓 log 很難看，而且 ES 索引會變成一個莫名其妙的名字。
func TestTrailingSlashIsTrimmed(t *testing.T) {
	cfg := loadString(t, strings.Join([]string{
		"PUBLIC_BASE_URL=https://forum.example.com/",
		"FILES_SERVER_URL=http://files:7070/",
		"FILES_SERVER_PUBLIC_URL=https://cdn.example.com/",
		"ES_URL=http://es:9200/",
	}, "\n"))

	for _, c := range []struct{ name, got, want string }{
		{"PUBLIC_BASE_URL", cfg.PublicBaseURL, "https://forum.example.com"},
		{"FILES_SERVER_URL", cfg.FilesServerURL, "http://files:7070"},
		{"FILES_SERVER_PUBLIC_URL", cfg.FilesServerPublicURL, "https://cdn.example.com"},
		{"ES_URL", cfg.ESURL, "http://es:9200"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestGoogleRedirectURLIsNotTrimmed 守住一個反直覺的例外。
//
// GOOGLE_REDIRECT_URL 刻意**不**去掉結尾斜線：Google 對 redirect_uri 的比對
// 是逐字的，這裡若擅自去掉，反而會造成 redirect_uri_mismatch —— 而那個錯誤
// 訊息不會告訴你真正的原因是斜線。
func TestGoogleRedirectURLIsNotTrimmed(t *testing.T) {
	cfg := loadString(t, "GOOGLE_REDIRECT_URL=https://forum.example.com/auth/callback/\n")
	if cfg.GoogleRedirectURL != "https://forum.example.com/auth/callback/" {
		t.Errorf("GoogleRedirectURL = %q，尾端斜線必須原樣保留", cfg.GoogleRedirectURL)
	}
}

// TestDSNKeepsEverythingAfterTheFirstEquals 守住設定檔的切分規則。
//
// 值裡經常含有 "="（MySQL DSN 的 "?charset=utf8mb4&parseTime=True"），因此
// 只切第一個 "="。用 Split 而不是 SplitN 會把 DSN 截斷，而症狀是「連不上
// 資料庫」—— 一個完全看不出根因的錯誤。
func TestDSNKeepsEverythingAfterTheFirstEquals(t *testing.T) {
	const dsn = "user:pa=ss@tcp(127.0.0.1:3306)/forum?charset=utf8mb4&parseTime=True&loc=Local"
	cfg := loadString(t, "DB_DSN="+dsn+"\n")
	if cfg.DbDSN != dsn {
		t.Errorf("DbDSN = %q, want %q", cfg.DbDSN, dsn)
	}
}

// TestUnknownKeysAreSilentlyIgnored 守住「未知 key 被忽略」這個刻意的寬鬆。
//
// 這個行為讓不同環境能共用同一份範本檔（本機不設 ES_URL，正式環境設）。
// 它的代價是打錯一個 key 也不會有提示 —— 而那正是 config_test.go 的檔頭
// 所說的、必須靠人留意的取捨。把這個行為釘住，是為了讓將來有人想改成「報錯」
// 時必須先想清楚那個代價。
func TestUnknownKeysAreSilentlyIgnored(t *testing.T) {
	cfg, err := Load(writeConfig(t, strings.Join([]string{
		"TYPO_IN_THIS_KEY=value",
		"RATE_LIMIT_REQUESTS=11",
		"",
		"# 註解行",
		"; 另一種註解",
		"沒有等號的一行",
		"SERVER_PORT=:8088",
	}, "\n")))
	if err != nil {
		t.Fatalf("含未知 key 的設定不該讓 Load 失敗: %v", err)
	}
	if cfg.RateLimitRequests != 11 {
		t.Errorf("RateLimitRequests = %d, want 11（未知 key 不該影響其他設定）", cfg.RateLimitRequests)
	}
	if cfg.ServerPort != ":8088" {
		t.Errorf("ServerPort = %q, want \":8088\"", cfg.ServerPort)
	}
}

// TestLastValueWins 覆蓋 applyDefaults 必須在整份檔案解析完之後才執行的理由。
//
// 副作用是使用者無法用空字串停用某個功能（空字串與未設定無法區分）——
// 這個取捨在 config.go 的 applyDefaults 有記錄，這裡只釘住「後面的值贏」這件事。
func TestLastValueWins(t *testing.T) {
	cfg := loadString(t, "SERVER_PORT=:1111\nSERVER_PORT=:2222\n")
	if cfg.ServerPort != ":2222" {
		t.Errorf("ServerPort = %q, want \":2222\"（最後一次出現的值必須贏）", cfg.ServerPort)
	}
}

// TestLoadMissingFileIsFatal 守住「檔案讀不到是不可降級的錯誤」。
//
// main 依賴這個行為做 logger.Fatalf。若 Load 在檔案不存在時回一個可用的
// Config 且 err 為 nil，站會以一組預設值啟動 —— 症狀是「連錯的資料庫」或
// 「Redis 不通」，而不是「設定檔不見了」。
func TestLoadMissingFileIsFatal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.conf")
	cfg, err := Load(path)
	if err == nil {
		t.Fatal("Load 一個不存在的檔案卻回 nil error，必須讓呼叫端能走 Fatal")
	}
	// 即使在錯誤路徑上 cfg 仍是可讀的結構 —— main 因此能在 Fatal 訊息裡
	// 引用它。這個形狀刻意保留，因此測試固定它。
	if cfg.ServerPort != "" {
		t.Errorf("錯誤路徑上的 cfg.ServerPort = %q, want \"\"（applyDefaults 尚未執行）", cfg.ServerPort)
	}
}

// TestLoadRateLimitPreDefaultsSurviveError 守住「錯誤路徑上三組限流仍有值」。
//
// Load 在開頭就預先寫入三組限流設定，正是為了讓它們在檔案不存在時仍然存在
// —— 若將來有人把這些預先寫入移到 applyDefaults，這支測試會失敗，而那個改動
// 會讓「設定檔遺失」時的錯誤訊息少掉三個數字。
func TestLoadRateLimitPreDefaultsSurviveError(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.conf"))
	if err == nil {
		t.Fatal("預期 Load 回錯誤")
	}
	if cfg.RateLimitRequests <= 0 || cfg.RateLimitWindow <= 0 {
		t.Errorf("錯誤路徑上的限流預設值不完整: write=%d/%v", cfg.RateLimitRequests, cfg.RateLimitWindow)
	}
}

// TestRetentionSettingsRejectZero 守住稽核與監控的保留期不能是 0。
//
// 「保留 0 天」等於「不保存任何稽核紀錄」，那會讓這張表形同虛設，而稽核頁
// 照樣顯示「沒有紀錄」—— 看起來完全正常。靜默接受它比拒絕它危險得多。
func TestRetentionSettingsRejectZero(t *testing.T) {
	cases := []struct {
		key string
		get func(Config) int
	}{
		{"AUDIT_RETENTION_DAYS", func(c Config) int { return c.AuditRetentionDays }},
		{"MONITOR_RETENTION_HOURS", func(c Config) int { return c.MonitorRetentionHours }},
	}
	for _, tc := range cases {
		for _, value := range []string{"0", "-1", "abc"} {
			t.Run(tc.key+"="+value, func(t *testing.T) {
				cfg := loadString(t, tc.key+"="+value+"\n")
				if got := tc.get(cfg); got <= 0 {
					t.Errorf("%s=%q 被接受了（%d），必須沿用預設值", tc.key, value, got)
				}
			})
		}
	}
}

// TestTrustedProxyCIDRsStaysRaw 守住 TRUSTED_PROXY_CIDRS 不在 config 層解析。
//
// 解析成位址段是 httpapi 的責任（它才是決定「這個值該不該被採信」的地方）。
// 在此處解析會讓 config 依賴 net 套件，並且無法區分「寫錯了」與「沒設定」——
// 那兩者對管理員的意義完全不同：前者要修，後者代表維持舊的標頭優先行為。
func TestTrustedProxyCIDRsStaysRaw(t *testing.T) {
	const raw = " 10.0.0.0/8 , 192.168.1.1 "
	cfg := loadString(t, "TRUSTED_PROXY_CIDRS="+raw+"\n")
	if cfg.TrustedProxyCIDRs != strings.TrimSpace(raw) {
		t.Errorf("TrustedProxyCIDRs = %q, want %q（只去頭尾空白，不解析、不切逗號）",
			cfg.TrustedProxyCIDRs, strings.TrimSpace(raw))
	}
}

// TestFilesServerTokenEnvFallback 覆蓋唯一支援環境變數的欄位。
//
// 這是「唯一應該在部署時注入、而不是寫進版控檔案的設定」的注入點。優先序是
// 設定檔的值優先、環境變數次之 —— 而這個優先序只寫在一個地方（applyDefaults），
// 因此測試要同時釘住兩條路徑。
func TestFilesServerTokenEnvFallback(t *testing.T) {
	t.Run("設定檔有值時優先", func(t *testing.T) {
		t.Setenv("FILES_SERVER_TOKEN", "from-env")
		cfg := loadString(t, "FILES_SERVER_TOKEN=from-file\n")
		if cfg.FilesServerToken != "from-file" {
			t.Errorf("FilesServerToken = %q, want \"from-file\"（設定檔的值必須優先）", cfg.FilesServerToken)
		}
	})
	t.Run("設定檔沒有值時取環境變數", func(t *testing.T) {
		t.Setenv("FILES_SERVER_TOKEN", "from-env")
		cfg := loadString(t, "COOKIE_NAME=x\n")
		if cfg.FilesServerToken != "from-env" {
			t.Errorf("FilesServerToken = %q, want \"from-env\"", cfg.FilesServerToken)
		}
	})
	t.Run("兩者都沒有時為空（後端仍會送出無驗證標頭）", func(t *testing.T) {
		t.Setenv("FILES_SERVER_TOKEN", "")
		cfg := loadString(t, "COOKIE_NAME=x\n")
		if cfg.FilesServerToken != "" {
			t.Errorf("FilesServerToken = %q, want \"\"", cfg.FilesServerToken)
		}
	})
}

// TestNormalizeEmail 單獨覆蓋正規化的兩個步驟。
//
// 抽出來測是因為它同時被 parseList 之後的路徑與 IsAdminEmail 使用，而兩邊
// 都必須做 —— 少一邊就會靜默失效（這個函式的檔頭有說明）。
func TestNormalizeEmail(t *testing.T) {
	cases := []struct{ in, want string }{
		{"admin@example.com", "admin@example.com"},
		{"  admin@example.com  ", "admin@example.com"},
		{"ADMIN@EXAMPLE.COM", "admin@example.com"},
		{"\t\nAdmin@Example.Com \r\n", "admin@example.com"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range cases {
		if got := normalizeEmail(tc.in); got != tc.want {
			t.Errorf("normalizeEmail(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNormalizeEmails 覆蓋清單版本的正規化，含丟空項目。
func TestNormalizeEmails(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil 維持 nil", in: nil, want: nil},
		{name: "正規化每一項", in: []string{" A@B.com ", "c@d.com"}, want: []string{"a@b.com", "c@d.com"}},
		{name: "丟掉全空白的項目", in: []string{"a@b.com", "   ", "", "c@d.com"}, want: []string{"a@b.com", "c@d.com"}},
		{name: "全部被丟掉", in: []string{"", "  "}, want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeEmails(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("normalizeEmails(%v) = %v, want %v", tc.in, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("normalizeEmails(%v)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}
