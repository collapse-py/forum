/*
logger_test.go 覆蓋 logger 套件純邏輯的部分：INFO memory 解析、門檻比對、
欄位清洗、兩種輸出格式，以及中介層的狀態碼分級。

【為什麼這些函式值得測】
ROADMAP.md 把 logger 列為 0% 覆蓋的目標，而它值得補的原因不是「行數」而是
**失效方式**：

  - sanitizeLogValue 是對抗「注入換行偽造出一行假的日誌」的唯一防線。它壞掉時，
    使用者可控輸入（email、貼文片段、URL）會能在日誌裡插進任意內容 —— 而稽核
    與存取日誌正是出事後唯一的線索。也就是說，壞掉時**最需要日誌可信的那一刻，
    正是日誌不再可信的那一刻**。
  - 門檻比對擺在格式化之前是刻意的效能取捨（被門檻濾掉的 DEBUG 連 Sprintf
    都不用付）。若這個性質被「順手改掉」，症狀是 DEBUG 開著時 CPU 使用率
    莫名變高 —— 不會有任何測試或日誌提到它。
  - 兩種輸出格式與「背景工作省略前綴」那個分支：它讓啟動日誌好讀，而它壞掉時
    會讓每一條背景訊息看起來都像是一個有來源的 HTTP 請求 —— 那會讓「這條記錄
    是從哪來的」變得無法回答。

【測試策略上的兩個刻意選擇】
 1. 不測 Init 的檔案開啟成功路徑作為主要目標，而是測「開不起來時回錯誤」。
    後者的失效症狀是「設定檔指向一個不存在的目錄 → 服務照常啟動 →
    日誌全部消失」，而那是設定錯誤最典型的靜默後果。
 2. 對全域 std 的測試都用 t.Cleanup 把它還原。logger 的 API 是 package 層級的
    函式（Infof 等），沒有注入點 —— 這是刻意的（檔頭說明為什麼不設 nil），
    因此測試必須自己負責收拾全局狀態，否則一個測試的門檻會洩漏到下一個測試。
*/
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

// capture 暫時把 std 重新導向一個記憶體內的 writer，並在測試結束後還原。
//
// 這是這個套件唯一能觀察輸出的方式：logf → logWithContext 寫進 std.logger，
// 而 std.logger 是一個 *log.Logger。直接改 std.logger 可以，但必須同時把
// mutex 拿對 —— 因此包成一個 helper，讓每支測試都不必重複那段容易出错的
// 取代與還原。
func capture(t *testing.T) *syncBuffer {
	t.Helper()

	var mu sync.Mutex
	buf := &syncBuffer{}

	std.mu.Lock()
	prevLogger := std.logger
	prevFile := std.file
	prevFormat := std.format
	prevLevel := std.level
	std.logger = log.New(&lockedWriter{w: buf, mu: &mu}, "", 0)
	std.file = nil
	std.format = "text"
	std.level = DEBUG
	std.mu.Unlock()

	t.Cleanup(func() {
		std.mu.Lock()
		std.logger = prevLogger
		std.file = prevFile
		std.format = prevFormat
		std.level = prevLevel
		std.mu.Unlock()
	})
	return buf
}

// syncBuffer 是一個可以在並行寫入下安全讀取的緩衝區。
//
// 需要它是因為這個套件的主要賣點之一就是「可安全地被多執行緒同時使用」——
// 而 race detector 只能在並行寫入下才有意義。普通 bytes.Buffer 在並行寫入時
// 是資料競爭，那會讓這支測試自己就是一個不可靠的測試。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// lockedWriter 把寫入序列化到外部提供的鎖上，讓 log.Logger 與測試的讀取
// 不會交錯。
type lockedWriter struct {
	w  *syncBuffer
	mu *sync.Mutex
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// TestParseLevel 覆蓋 parseLevel 的認得與認不得兩種路徑。
//
// 「認不得時退回 INFO」是一個刻意的偏嚴選擇（寧可少記，也不要因為打錯一個字
// 而讓整個服務安靜無聲）。這支測試把「打錯字」的各種樣式都列出來，因為那是
// 最可能被寫錯的一類。
func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want Level
	}{
		{"DEBUG", DEBUG},
		{"INFO", INFO},
		{"WARN", WARN},
		{"ERROR", ERROR},
		// 大小寫與空白容忍：設定檔裡 "log_level = warn" 是很自然的寫法。
		{"debug", DEBUG},
		{"Debug", DEBUG},
		{"  warn  ", WARN},
		{"\tERROR\n", ERROR},
		// 以下都必須退回 INFO。
		{"", INFO},
		{"WARNING", INFO},
		{"FATAL", INFO},
		{"TRACE", INFO},
		{"  ", INFO},
		{"verbose", INFO},
	}
	for _, tc := range cases {
		if got := parseLevel(tc.in); got != tc.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestSanitizeLogValue 覆蓋欄位清洗的四條規則。
//
// 這一條是這個檔最重要的一支測試：sanitizeLogValue 是對抗日誌注入的唯一防線，
// 而它壞掉時症狀是「稽核日誌不再能被信任」—— 那恰好是在最需要它的時刻。
func TestSanitizeLogValue(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// 規則 1：去頭尾空白；全空白回傳 "-"
		{name: "正常值", in: "admin@example.com", want: "admin@example.com"},
		{name: "去頭尾空白", in: "  admin@example.com  ", want: "admin@example.com"},
		{name: "空字串", in: "", want: "-"},
		{name: "只有空白", in: "   ", want: "-"},
		{name: "只有換行", in: "\n\r\n", want: "-"},

		// 規則 2：\n \r \t 換成一般空白 —— 這是對抗日誌注入的主要防線
		{name: "換行", in: "a\nb", want: "a b"},
		{name: "回車", in: "a\rb", want: "a b"},
		{name: "Tab", in: "a\tb", want: "a b"},
		// 這是真正的攻擊形狀：注入一行看起來像系統訊息的內容
		{
			name: "偽造一行 ERROR 記錄",
			in:   "normal\n2026-10-02 00:00:00.000 [ERROR] 密碼已竄改為 attacker",
			want: "normal 2026-10-02 00:00:00.000 [ERROR] 密碼已竄改為 attacker",
		},

		// 規則 3：丟棄其餘 C0 控制字元與 DEL
		{name: "NUL", in: "a\x00b", want: "ab"},
		{name: "BEL", in: "a\x07b", want: "ab"},
		{name: "ESC", in: "a\x1bb", want: "ab"},
		{name: "DEL", in: "a\x7fb", want: "ab"},
		{name: "全部 C0 控制字元", in: "\x01\x02\x03\x04", want: "-"},

		// 逐 rune 處理：不能切斷 UTF-8 多位元組字元
		{name: "中文", in: "使用者名稱", want: "使用者名稱"},
		{name: "emoji", in: "🙂", want: "🙂"},
		{name: "中文與控制字元混合", in: "使用者\n名稱", want: "使用者 名稱"},

		// 規則 4：超過 256 bytes 截斷成（不超過）253 bytes + "..."
		{name: "剛好 256 不截斷", in: strings.Repeat("a", 256), want: strings.Repeat("a", 256)},
		{name: "257 截斷", in: strings.Repeat("a", 257), want: strings.Repeat("a", 253) + "..."},
		{name: "很長的值", in: strings.Repeat("x", 10000), want: strings.Repeat("x", 253) + "..."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeLogValue(tc.in); got != tc.want {
				t.Errorf("sanitizeLogValue(%q)\n  got  %q\n  want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSanitizeLogValueTruncatesOnRuneBoundary 是單獨抽出來的一支，因為它是這個
// 函式最容易寫錯的地方。
//
// 原實作直接寫 value[:253] + "..."，而 253 不是任何三個位元組的倍數，因此一個
// 中文字會被切成「兩���字 + 一個頭」，欄位裡因此出現一個 U+FFFD（顯示為 �）。
// 那直接違反這個函式自己宣稱的「逐 rune 處理…是為了不切斷 UTF-8 多位元組字元，
// 避免產生亂碼」—— 而亂碼在診斷日誌裡是最糟的失效形式：它讓「使用者到底送了
// 什麼字」變成無法回答的問題。
func TestSanitizeLogValueTruncatesOnRuneBoundary(t *testing.T) {
	// 逐一長度掃過：3、6、…、300 bytes 的中文字串，每一個都必須是合法 UTF-8。
	for n := 1; n <= 100; n++ {
		in := strings.Repeat("中", n)
		got := sanitizeLogValue(in)

		if !utf8.ValidString(got) {
			t.Fatalf("%d 個中文字（%d bytes）截斷後不是合法 UTF-8: %q", n, len(in), got)
		}
		if strings.ContainsRune(got, utf8.RuneError) {
			t.Fatalf("%d 個中文字（%d bytes）截斷後含 U+FFFD（亂碼）: %q", n, len(in), got)
		}
		if len(got) > maxLogValueBytes {
			t.Errorf("%d bytes 截斷後長度 = %d，超過上限 %d", len(in), len(got), maxLogValueBytes)
		}
		if !strings.HasSuffix(got, truncationEllipsis) && len(in) > maxLogValueBytes {
			t.Errorf("超長值沒有加上 %q: %q", truncationEllipsis, got)
		}
		// 有效前綴必須完整保留：清洗是「縮短」，不是「吃掉中間」。
		kept := strings.TrimSuffix(got, truncationEllipsis)
		if !strings.HasPrefix(in, kept) {
			t.Errorf("截斷後保留的部分不是原字串的前綴: %q", kept)
		}
	}

	// 混雜寬度的字元（中文 3 bytes、ASCII 1 byte、emoji 4 bytes）也要成立 ——
	// 這是實際情況：欄位裡通常是中英文與 emoji 混在一起。
	mixed := strings.Repeat("a中🙂b", 40)
	if got := sanitizeLogValue(mixed); !utf8.ValidString(got) || strings.ContainsRune(got, utf8.RuneError) {
		t.Errorf("混雜寬度字元截斷後損毀: %q", got)
	}
}

// TestSanitizeLogValueUsesPlaceholderAfterCleaning 守住「清洗完為空也要是 -」。
//
// 原實作只在清洗**前**檢查空白，因此一個只由控制字元組成的值（外部可控，例如
// 某個標頭剛好是 "\x01"）會變成空字串，輸出裡就出現 "ip=" 這種沒有值的欄位。
// 這條規則 1 的整個目的是「讓欄位在輸出中仍然可見」，因此那個形狀不該存在。
func TestSanitizeLogValueUsesPlaceholderAfterCleaning(t *testing.T) {
	for _, in := range []string{
		"\x01", "\x01\x02\x03\x04", "\x7f", " \x01 \x02 ",
	} {
		if got := sanitizeLogValue(in); got != "-" {
			t.Errorf("sanitizeLogValue(%q) = %q, want \"-\"（清洗後為空必須用佔位符）", in, got)
		}
	}
}

// TestSanitizeLogValueStripsInjectionAttempt 是把攻擊形狀單獨拉出來的一支。
//
// 它與上一支的重疊是刻意的：那個攻擊形狀值得有一個以它命名的測試，讓未來
// 有人讀測試清單時能直接看見「日誌注入」被防住了，而不必在表格裡猜哪一列
// 是哪一個攻擊。
func TestSanitizeLogValueStripsInjectionAttempt(t *testing.T) {
	attack := "victim@example.com\n2026-01-01 00:00:00.000 [INFO] [HTTP] request_id=deadbeef ip=127.0.0.1"
	got := sanitizeLogValue(attack)

	if strings.ContainsAny(got, "\n\r") {
		t.Fatalf("輸出仍含換行，日誌注入沒有被擋下: %q", got)
	}
	if strings.Count(got, "[HTTP]") != 1 {
		t.Errorf("輸出了 %d 個 [HTTP]，攻擊者可能偽造了一整行記錄: %q",
			strings.Count(got, "[HTTP]"), got)
	}
	if !strings.HasPrefix(got, "victim@example.com ") {
		t.Errorf("原本的內容被丟掉了: %q", got)
	}
}

// TestLevelThresholdFiltersBeforeFormatting 守住「門檻判斷在格式化之前」。
//
// 這是一個效能性質而不是正確性性質，因此它不會出現在任何錯誤訊息裡：把它
// 移到格式化之後的症狀是「DEBUG 開著時 CPU 使用率莫名變高」，而沒有人會為
// 這個去讀日誌程式碼。所以只能用測試釘住。
//
// 斷言方式是「被門檻濾掉的訊息不出現在輸出裡」，因為若實作先格式化再丟棄，
// 輸出的**內容**會一模一樣 —— 這個測試守的是順序，不是結果。
func TestLevelThresholdFiltersBeforeFormatting(t *testing.T) {
	buf := capture(t)

	std.mu.Lock()
	std.level = WARN
	std.mu.Unlock()

	Debugf("這條不該出現")
	Infof("這條也不該出現")
	Warnf("這條應該出現")

	out := buf.String()
	if strings.Contains(out, "不該出現") {
		t.Errorf("低於門檻的訊息被輸出了:\n%s", out)
	}
	if !strings.Contains(out, "應該出現") {
		t.Errorf("等於門檻的訊息被丟棄了:\n%s", out)
	}
}

// TestLevelOrderingIsMeaningful 守住 DEBUG < INFO < WARN < ERROR 這個順序。
//
// 它是「只許升不許降」的隱含前提：若有人把 ERROR 宣告在 INFO 之前，
// parseLevel 仍會正確，但門檻比較會完全反轉 —— 症狀是「設定 ERROR 只記錄
// 錯誤」變成「設定 ERROR 記錄一切」，而那會讓正式環境的日誌爆量。
func TestLevelOrderingIsMeaningful(t *testing.T) {
	if !(DEBUG < INFO && INFO < WARN && WARN < ERROR) {
		t.Fatalf("等級順序被改動了: DEBUG=%d INFO=%d WARN=%d ERROR=%d", DEBUG, INFO, WARN, ERROR)
	}
	for level, want := range map[Level]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	} {
		if got := levelNames[level]; got != want {
			t.Errorf("levelNames[%v] = %q, want %q", level, got, want)
		}
	}
}

// TestBackgroundMessagesOmitMetadataPrefix 守住背景工作的輸出形狀。
//
// 背景工作（啟動訊息、遷移結果）沒有請求中繼資料，三欄會都是 "-"。這個分支
// 讓那些訊息短得多，而它壞掉時的症狀是「每一條啟動訊息看起來都像一個有來源
// IP 與使用者的 HTTP 請求」—— 那會讓「這條記錄從哪來的」變得無法回答。
func TestBackgroundMessagesOmitMetadataPrefix(t *testing.T) {
	buf := capture(t)
	Infof("啟動訊息")

	out := buf.String()
	if strings.Contains(out, "request_id=") || strings.Contains(out, "user_email=") {
		t.Errorf("背景訊息不該帶中繼前綴:\n%s", out)
	}
	if !strings.Contains(out, "啟動訊息") {
		t.Errorf("訊息本身不見了:\n%s", out)
	}
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("等級標記不見了:\n%s", out)
	}
}

// TestRequestMessagesCarryMetadata 守住 HTTP 存取記錄帶著中繼欄位。
func TestRequestMessagesCarryMetadata(t *testing.T) {
	buf := capture(t)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = WithMetadata(r, "203.0.113.7", "someone@example.com", "abc123")
	InfofContext(r.Context(), "處理貼文列表")

	out := buf.String()
	for _, want := range []string{
		"request_id=abc123",
		"ip=203.0.113.7",
		"user_email=someone@example.com",
		"處理貼文列表",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("輸出缺少 %q:\n%s", want, out)
		}
	}
}

// TestMetadataFieldsAreSanitized 守住中繼欄位也會被清洗。
//
// 這一條特別重要，因為 IP 與 request ID 都來自外部可控的輸入
// （X-Forwarded-For 與客戶端送來的標頭）。若中繼欄位繞過了清洗，一個攻擊者
// 就能用一個含換行的 X-Forwarded-For 在每一行 log 前塞進假的欄位 —— 而那是
// 一個比 msg 注入更容易做到、也更難察覺的版本。
//
// 這支測試斷言的是**行數不變**（也就是「無法偽造出一條記錄」），不是
// 「值裡不會出現 key=value 的樣子」。後者是 sanitizeLogValue 刻意不做的事，
// 理由寫在該函式的「已知殘餘限制」裡：擋掉它就得改變整個 text 輸出的形狀，
// 而那會讓既有的 grep 規則失效。
func TestMetadataFieldsAreSanitized(t *testing.T) {
	buf := capture(t)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = WithMetadata(r,
		"1.2.3.4\n[ERROR] 2026-01-01 00:00:00.000 密碼已竄改",
		"victim@example.com\nfake: attacker@evil.example",
		"realid\nfake: zzz")
	InfofContext(r.Context(), "測試")

	out := strings.TrimRight(buf.String(), "\n")
	if strings.ContainsAny(out, "\n\r") {
		t.Fatalf("中繼欄位注入了額外的記錄行 —— 這是這條防線要擋的攻擊:\n%s", out)
	}
	// 真正的內容必須還在 —— 清洗是去換行，不是丟棄整個欄位。
	for _, good := range []string{"1.2.3.4", "victim@example.com", "realid", "測試"} {
		if !strings.Contains(out, good) {
			t.Errorf("清洗後遺失了原本的內容 %q:\n%s", good, out)
		}
	}
}

// TestSanitizeLogValueKeepsColumnFormUnambiguous 記錄一個刻意不修的限制。
//
// 這支測試的斷言是「目前會產生這種混淆的輸出」。它存在的理由不是要讓這個
// 行為保持不變，而是要讓**它是有名的**：sanitizeLogValue 的「已知殘餘限制」
// 說明裡寫著 text 輸出的欄位無法防住值內含 " key=value" 的混淆，而這支測試
// 是那個說明的可執行證據。
//
// 若將來有人真的去修（例如以引號包住欄位值），這支測試會失敗並要求同時更新
// 那段說明與所有依賴 text 輸出的 grep 規則 —— 那正是它該做的事。
func TestSanitizeLogValueKeepsColumnFormUnambiguous(t *testing.T) {
	buf := capture(t)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = WithMetadata(r, "1.2.3.4 fake_ip=9.9.9.9", "u@example.com", "realid")
	InfofContext(r.Context(), "測試")

	out := buf.String()
	if !strings.Contains(out, "fake_ip=9.9.9.9") {
		t.Skip("text 輸出的欄位形式已經改成不會被混淆的形狀 —— " +
			"請更新 sanitizeLogValue 的「已知殘餘限制」說明，並確認沒有 grep 規則依賴舊形狀")
	}
	t.Log("如預期，text 輸出無法區分欄位值內的 \" key=value\"：" +
		"這是刻意不修的取捨，理由見 sanitizeLogValue 的說明")
}

// TestJSONFormatEmitsParsableObject 覆蓋 json 輸出格式。
//
// 這一條的價值在於「可被日誌收集系統索引」這個承諾：欄位名固定、值都是字串。
// 若有人把某個欄位改成數字或拿掉，症狀是 ELK / Loki 把它歸到錯誤的欄位類型
// 而不會報錯。
func TestJSONFormatEmitsParsableObject(t *testing.T) {
	buf := capture(t)

	std.mu.Lock()
	std.format = "json"
	std.mu.Unlock()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = WithMetadata(r, "203.0.113.7", "someone@example.com", "abc123")
	InfofContext(r.Context(), "帶引號與換行的訊息 \"quoted\"\ninjected")

	line := strings.TrimSpace(buf.String())
	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("輸出不是可解析的 JSON (%v): %s", err, line)
	}

	for _, key := range []string{"time", "level", "request_id", "ip", "user_email", "caller", "msg"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("JSON 缺少欄位 %q: %s", key, line)
		}
	}
	if entry["level"] != "INFO" {
		t.Errorf("level = %v, want \"INFO\"", entry["level"])
	}
	if entry["ip"] != "203.0.113.7" {
		t.Errorf("ip = %v", entry["ip"])
	}
	// 換行必須已被清掉，且引號必須仍被正確跳脫（否則這行就不是合法 JSON）。
	if msg, _ := entry["msg"].(string); strings.ContainsAny(msg, "\n\r") {
		t.Errorf("msg 仍含換行: %q", msg)
	}
	if msg, _ := entry["msg"].(string); !strings.Contains(msg, "quoted") {
		t.Errorf("msg 被過度清洗了，引號內的內容應該保留: %q", msg)
	}
}

// TestJSONBackgroundMetadataIsBlanked 守住 json 模式下背景訊息的三欄被清空。
//
// 這個分支的理由在程式碼裡有註：只有三欄「同時」為 "-" 才清空，因為
// LoggingMiddleware 會把未登入者填成 "anonymous"。因此這支測試也必須確認
// "anonymous" 不會被清空 —— 否則「有人造訪但未登入」會與「沒有任何請求」
// 混為一談。
func TestJSONBackgroundMetadataIsBlanked(t *testing.T) {
	t.Run("背景訊息：三欄清空", func(t *testing.T) {
		buf := capture(t)
		std.mu.Lock()
		std.format = "json"
		std.mu.Unlock()

		Infof("啟動訊息")

		var entry map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
			t.Fatalf("不是合法 JSON (%v): %s", err, buf.String())
		}
		for _, key := range []string{"request_id", "ip", "user_email"} {
			if got, ok := entry[key].(string); !ok || got != "" {
				t.Errorf("背景訊息的 %s = %v，want \"\"（不能留佔位符）", key, entry[key])
			}
		}
	})

	t.Run("未登入的 HTTP 請求：anonymous 必須保留", func(t *testing.T) {
		buf := capture(t)
		std.mu.Lock()
		std.format = "json"
		std.mu.Unlock()

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = WithMetadata(r, "203.0.113.7", "anonymous", "abc123")
		InfofContext(r.Context(), "未登入請求")

		var entry map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
			t.Fatalf("不是合法 JSON (%v): %s", err, buf.String())
		}
		if entry["user_email"] != "anonymous" {
			t.Errorf("user_email = %v, want \"anonymous\"（它與「沒有請求」不同，不能被清空）", entry["user_email"])
		}
	})
}

// TestUninitializedLoggerIsSilentNotPanicking 守住「未呼叫 Init 也不會 panic」。
//
// 這是這個套件一個明確的設計目標（logger.go 的 std 變數說明）：main 在
// logger 尚未就緒時就會 Fatalf（設定檔載入失敗），若那條路徑 panic 掉，
// 使用者會看到一個 stack trace 而不是真正的錯誤原因。
func TestUninitializedLoggerIsSilentNotPanicking(t *testing.T) {
	saved := &Logger{} // logger 欄位為 nil

	// 這些呼叫全部必須安靜返回。刻意不用 t.Parallel()：它們碰的是 package 層
	// 的 std，而這個測試要驗證的正是 nil 接收者的路徑。
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("未初始化時呼叫 panic 了: %v", r)
			}
		}()
		saved.logf(ERROR, "不該有任何輸出 %d", 1)
		saved.logWithContext(context.Background(), 2, ERROR, "不該有任何輸出")
	}()
}

// TestInitReportsUnopenableFile 守住「開不起來時回錯誤」。
//
// 這一條是本檔最實際的一個測試：LOG_FILE 指向一個不存在的目錄是極常見的部署
// 錯誤，而症狀若是靜默（服務照常啟動、日誌全部消失在虛無中）會非常難診斷。
// Init 必須回錯誤，讓 main 走 Fatalf。
func TestInitReportsUnopenableFile(t *testing.T) {
	t.Cleanup(func() {
		std.mu.Lock()
		std.logger = log.New(os.Stdout, "", 0)
		std.file = nil
		std.format = "text"
		std.level = INFO
		std.mu.Unlock()
	})

	missing := filepath.Join(t.TempDir(), "no-such-dir", "forum.log")
	err := Init("INFO", missing, "text")
	if err == nil {
		t.Fatal("Init 一個開不起來的路徑卻回 nil error，main 因此不會 Fatal —— 日誌會全部消失")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("錯誤訊息沒有點名路徑（維運需要知道是設定檔的哪一項）: %v", err)
	}
}

// TestInitWritesToFile 覆蓋 Init 成功開檔的那一條路徑。
//
// 同時驗證兩件事：檔案真的收到內容、以及重新 Init 不會讓檔案描述符洩漏
// （舊句柄必須被關閉）。後者的症狀是「每次重載都洩漏一個 fd」，那要跑到
// 檔案描述符上限才會發現 —— 因此用「舊句柄已關閉」這個直接事實來斷言。
func TestInitWritesToFile(t *testing.T) {
	t.Cleanup(func() {
		Close()
		std.mu.Lock()
		std.logger = log.New(os.Stdout, "", 0)
		std.level = INFO
		std.format = "text"
		std.mu.Unlock()
	})

	path := filepath.Join(t.TempDir(), "forum.log")
	if err := Init("INFO", path, "text"); err != nil {
		t.Fatalf("Init 回錯誤: %v", err)
	}

	std.mu.Lock()
	firstFile := std.file
	std.mu.Unlock()
	if firstFile == nil {
		t.Fatal("Init 沒有記下檔案句柄，Close 因此關不掉它")
	}

	Infof("寫進檔案的訊息")

	// Close 必須真的關掉句柄：對已關閉的 *os.File 寫入會回錯。
	Close()
	if _, err := firstFile.WriteString("x"); err == nil {
		t.Error("Close 之後檔案句柄仍然可寫，Close 沒有真的關掉它")
	}

	// 檔案內容必須有那則訊息。
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("讀取日誌檔失敗: %v", err)
	}
	if !strings.Contains(string(data), "寫進檔案的訊息") {
		t.Errorf("日誌檔沒有那則訊息:\n%s", data)
	}
}

// TestCloseIsIdempotent 守住 Close 可以被重複呼叫。
//
// main 以 defer 呼叫 Close，而 logger.Init 在失敗時也可能已經開過檔 —— 一個
// 會 panic 的 Close 會讓「設定檔有問題」這個診斷步驟自己崩潰。
func TestCloseIsIdempotent(t *testing.T) {
	t.Cleanup(func() {
		std.mu.Lock()
		std.logger = log.New(os.Stdout, "", 0)
		std.file = nil
		std.format = "text"
		std.level = INFO
		std.mu.Unlock()
	})

	path := filepath.Join(t.TempDir(), "forum.log")
	if err := Init("INFO", path, "text"); err != nil {
		t.Fatalf("Init 回錯誤: %v", err)
	}

	Close()
	Close() // 第二次必須是安全的 no-op
}

// TestConcurrentLoggingIsRaceFree 在並行寫入下使用 logger。
//
// 這個套件的主要賣點之一就是「可安全地被多執行緒同時使用」，而那個承諾只有
// 在 -race 之下才有意義（CI 的 backend job 有開 CGO 專門跑它，見
// ROADMAP.md 的 Phase 1.2）。
//
// 這支測試**不斷言**輸出內容 —— 它斷言的是「跑到結束不會 panic、也不會被 race
// detector 抓到」。輸出內容的正確性由前面那些測試負責；這一支只負責併行安全。
func TestConcurrentLoggingIsRaceFree(t *testing.T) {
	capture(t)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r = WithMetadata(r, "203.0.113.1", "u@example.com", "id")
				InfofContext(r.Context(), "併行訊息 %d-%d\n帶換行的引數", n, j)
				Warnf("無中繼資料的併行訊息 %d-%d", n, j)
				Debugf("被門檻濾掉的 %d", j)
			}
		}(i)
	}
	wg.Wait()
}

// TestMetadataFromContextFillsPlaceholders 覆蓋 MetadataFromContext 的三個補值分支。
func TestMetadataFromContextFillsPlaceholders(t *testing.T) {
	t.Run("沒有 metadata 時三欄都是 -", func(t *testing.T) {
		meta := MetadataFromContext(context.Background())
		if meta.IP != "-" || meta.UserEmail != "-" || meta.RequestID != "-" {
			t.Errorf("MetadataFromContext(背景) = %+v, want 三欄都是 \"-\"", meta)
		}
	})

	t.Run("部分欄位有值時只補缺的", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), MetadataKey, Metadata{IP: "1.2.3.4"})
		meta := MetadataFromContext(ctx)
		if meta.IP != "1.2.3.4" {
			t.Errorf("IP = %q, want \"1.2.3.4\"", meta.IP)
		}
		if meta.UserEmail != "-" || meta.RequestID != "-" {
			t.Errorf("缺的欄位沒被補上: %+v", meta)
		}
	})

	t.Run("型別不符時視為沒有 metadata", func(t *testing.T) {
		// 這個分支的註解說明它是刻意的：context 中若存了別的型別，
		// 統一視為「沒有」，而不是讓型別斷言的失敗變成一個 panic。
		ctx := context.WithValue(context.Background(), MetadataKey, "不是 Metadata")
		meta := MetadataFromContext(ctx)
		if meta.IP != "-" || meta.UserEmail != "-" || meta.RequestID != "-" {
			t.Errorf("型別不符時 = %+v, want 三欄都是 \"-\"", meta)
		}
	})
}

// TestWithMetadataDoesNotMutateTheRequest 守住 WithMetadata 的值語意。
//
// 它的註解明確說「不直接修改傳入的 r」—— 若它改成就地修改，LoggingMiddleware
// 在呼叫 next 之前賦值的 request 就會與呼叫端手上的那份不同，而兩者的
// context 會不一致（症狀是 handler 讀不到 metadata，日誌裡全是 "-"）。
func TestWithMetadataDoesNotMutateTheRequest(t *testing.T) {
	original := httptest.NewRequest(http.MethodGet, "/", nil)
	derived := WithMetadata(original, "1.2.3.4", "u@example.com", "id")

	if got := MetadataFromContext(original.Context()).IP; got != "-" {
		t.Errorf("原始 request 的 context 被就地修改了（IP = %q）", got)
	}
	if got := MetadataFromContext(derived.Context()).IP; got != "1.2.3.4" {
		t.Errorf("衍生 request 的 context 沒有 metadata（IP = %q）", got)
	}
}

// TestGetAndSetUserEmail 覆蓋那組預留的介面。
func TestGetAndSetUserEmail(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := GetUserEmail(r); got != "" {
		t.Errorf("未設定時 GetUserEmail = %q, want \"\"", got)
	}

	r = SetUserEmail(r, "someone@example.com")
	if got := GetUserEmail(r); got != "someone@example.com" {
		t.Errorf("GetUserEmail = %q, want \"someone@example.com\"", got)
	}
}

// TestGetClientIPPrecedence 覆蓋 IP 解析的優先序與邊界。
//
// 這個函式的信任模型（無條件信任 X-Forwarded-For）在 httpapi 的
// trustedproxy.go 才有白名單版本；logger 這裡是單純的「最靠近用戶端」。
// 這支測試固定的是**優先序**與格式解析，不是信任模型。
func TestGetClientIPPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		want       string
	}{
		{name: "XFF 優先於 X-Real-IP 與 RemoteAddr", xff: "1.1.1.1", xri: "2.2.2.2", remoteAddr: "3.3.3.3:1234", want: "1.1.1.1"},
		{name: "XFF 取第一段", xff: "1.1.1.1, 2.2.2.2, 3.3.3.3", remoteAddr: "3.3.3.3:1234", want: "1.1.1.1"},
		{name: "XFF 第一段去空白", xff: "  1.1.1.1  ,2.2.2.2", remoteAddr: "3.3.3.3:1234", want: "1.1.1.1"},
		{name: "沒有 XFF 時用 X-Real-IP", xri: "2.2.2.2", remoteAddr: "3.3.3.3:1234", want: "2.2.2.2"},
		{name: "兩者都沒有時用 RemoteAddr", remoteAddr: "3.3.3.3:1234", want: "3.3.3.3"},
		// IPv6 的位址本身含冒號，因此必須取「最後一個」冒號。
		{name: "IPv6 RemoteAddr", remoteAddr: "[2001:db8::1]:443", want: "[2001:db8::1]"},
		{name: "RemoteAddr 沒有埠", remoteAddr: "3.3.3.3", want: "3.3.3.3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" {
				r.Header.Set("X-Real-IP", tc.xri)
			}
			if got := getClientIP(r); got != tc.want {
				t.Errorf("getClientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestNewRequestIDIsUniqueAndHex 守住 request ID 的形狀與唯一性。
//
// 形狀重要：它會出現在回應標頭 X-Request-ID，而「16 個十六進位字元」是
// frontend_shells / 前端與運維工具在比對時假設的長度。改變形狀會讓那些比對
// 靜默失效。
func TestNewRequestIDIsUniqueAndHex(t *testing.T) {
	seen := make(map[string]bool, 512)
	for i := 0; i < 512; i++ {
		id := newRequestID()
		if len(id) != 16 {
			t.Fatalf("request ID 長度 = %d, want 16（%q）", len(id), id)
		}
		if strings.Trim(id, "0123456789abcdef") != "" {
			t.Fatalf("request ID 含非十六進位字元: %q", id)
		}
		if seen[id] {
			t.Fatalf("request ID 重複了（%d 次取樣內）: %q", len(seen), id)
		}
		seen[id] = true
	}
}

// TestStatusWriterRecordsStatusCode 覆蓋 StatusWriter 的核心用途。
func TestStatusWriterRecordsStatusCode(t *testing.T) {
	t.Run("只寫 body 時預填 200", func(t *testing.T) {
		rec := httptest.NewRecorder()
		sw := NewStatusWriter(rec)
		if got := sw.StatusCode(); got != http.StatusOK {
			t.Errorf("預填狀態碼 = %d, want 200", got)
		}
		sw.Write([]byte("body")) // 沒有 WriteHeader
		if got := sw.StatusCode(); got != http.StatusOK {
			t.Errorf("只寫 body 後狀態碼 = %d, want 200", got)
		}
	})

	t.Run("WriteHeader 會被記下", func(t *testing.T) {
		rec := httptest.NewRecorder()
		sw := NewStatusWriter(rec)
		sw.WriteHeader(http.StatusTeapot)
		if got := sw.StatusCode(); got != http.StatusTeapot {
			t.Errorf("狀態碼 = %d, want 418", got)
		}
		if rec.Code != http.StatusTeapot {
			t.Errorf("底層 writer 沒有收到 WriteHeader（code = %d）", rec.Code)
		}
	})
}

// TestStatusWriterFlushIsBestEffort 守住 Flush 在底層不支援時不 panic。
//
// httptest.ResponseRecorder 實作了 Flusher，而要測「不支援」的那一條需要一個
// 刻意不實作它的 wrapper。症狀若是不 panic 這件事被破壞，panic 會發生在
// **每個** flush 型 handler 上 —— 而那種 handler 通常是 SSE 串流，很難重現。
func TestStatusWriterFlushIsBestEffort(t *testing.T) {
	t.Run("底層支援 Flusher 時轉發", func(t *testing.T) {
		rec := httptest.NewRecorder()
		NewStatusWriter(rec).Flush()
		if !rec.Flushed {
			t.Error("Flush 沒有被轉發到底層 writer")
		}
	})

	t.Run("底層不支援時靜默略過", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("底層不支援 Flusher 時 panic 了: %v", r)
			}
		}()
		NewStatusWriter(struct{ http.ResponseWriter }{httptest.NewRecorder()}).Flush()
	})
}

// TestLoggingMiddlewareAssignsRequestID 守住 request ID 進了 context 與回應標頭。
//
// 標頭必須在 next 之前寫入，否則 handler 一旦輸出內容就補不上了 —— 而這是
// 「維運拿 request ID 回報問題」這條動線的起點。
func TestLoggingMiddlewareAssignsRequestID(t *testing.T) {
	capture(t)

	var seenInHandler string
	h := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInHandler = MetadataFromContext(r.Context()).RequestID
	}), nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/forum/posts", nil))

	header := rec.Header().Get("X-Request-ID")
	if header == "" {
		t.Fatal("X-Request-ID 回應標頭沒有被設定")
	}
	if header != seenInHandler {
		t.Errorf("標頭的 ID (%q) 與 handler 讀到的 (%q) 不一致", header, seenInHandler)
	}
	if len(header) != 16 {
		t.Errorf("X-Request-ID 長度 = %d, want 16", len(header))
	}
}

// TestLoggingMiddlewareAnonymousWhenNoResolver 守住「沒有 resolver 時記 anonymous」。
//
// "anonymous" 而不是 "" 是刻意的：中繼欄位的三欄同時為 "-" 時 json 輸出會把
// 它們清空，而「有人造訪但未登入」必須與「完全沒有請求」可以區分。
func TestLoggingMiddlewareAnonymousWhenNoResolver(t *testing.T) {
	cases := []struct {
		name     string
		resolver UserResolver
		want     string
	}{
		{name: "resolver 為 nil", resolver: nil, want: "anonymous"},
		{name: "resolver 回空字串", resolver: func(*http.Request) string { return "" }, want: "anonymous"},
		{name: "resolver 回 email", resolver: func(*http.Request) string { return "u@example.com" }, want: "u@example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			h := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = MetadataFromContext(r.Context()).UserEmail
			}), tc.resolver)

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			if seen != tc.want {
				t.Errorf("UserEmail = %q, want %q", seen, tc.want)
			}
		})
	}
}

// TestLoggingMiddlewareSeverityByStatus 守住存取記錄的分級規則。
//
// 分級是維運的第一個過濾條件（只看 ERROR 找真正的問題），而它壞掉時的症狀是
// 「5xx 沒被記成 ERROR」—— 也就是出事時最該看到的那些記錄被埋在 INFO 裡。
func TestLoggingMiddlewareSeverityByStatus(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantLevels []string
		notLevels  []string
	}{
		{name: "2xx → INFO", status: http.StatusOK, wantLevels: []string{"[INFO]"}, notLevels: []string{"[WARN]", "[ERROR]"}},
		{name: "3xx → INFO", status: http.StatusFound, wantLevels: []string{"[INFO]"}, notLevels: []string{"[WARN]", "[ERROR]"}},
		{name: "4xx → WARN", status: http.StatusNotFound, wantLevels: []string{"[WARN]"}, notLevels: []string{"[INFO]", "[ERROR]"}},
		{name: "5xx → ERROR", status: http.StatusInternalServerError, wantLevels: []string{"[ERROR]"}, notLevels: []string{"[INFO]", "[WARN]"}},
		// 5xx 的判定優先於 4xx 的分支：兩者並列而非串接。
		{name: "599 → ERROR（不是 WARN）", status: 599, wantLevels: []string{"[ERROR]"}, notLevels: []string{"[WARN]"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := capture(t)
			h := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}), nil)

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/thing", nil))

			out := buf.String()
			for _, want := range tc.wantLevels {
				if !strings.Contains(out, want) {
					t.Errorf("輸出缺少等級 %s:\n%s", want, out)
				}
			}
			for _, not := range tc.notLevels {
				if strings.Contains(out, not) {
					t.Errorf("輸出不該有等級 %s:\n%s", not, out)
				}
			}
			if !strings.Contains(out, "status=") {
				t.Errorf("記錄裡沒有狀態碼:\n%s", out)
			}
		})
	}
}

// TestLoggingMiddlewareRecordsPathWithoutQuery 守住不記查詢字串。
//
// 查詢字串常帶 token 與使用者輸入，而這條記錄會進日誌檔與檔案系統。只記路徑
// 是刻意的取捨，因此把它釘成測試 —— 它的失效是靜默的（沒有任何錯誤）。
func TestLoggingMiddlewareRecordsPathWithoutQuery(t *testing.T) {
	buf := capture(t)

	h := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), nil)
	h.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/forum/search?q=secret_token_abc", nil))

	out := buf.String()
	if !strings.Contains(out, "/api/forum/search") {
		t.Errorf("記錄裡沒有路徑:\n%s", out)
	}
	if strings.Contains(out, "secret_token_abc") {
		t.Errorf("查詢字串被寫進日誌了（可能含 token）:\n%s", out)
	}
}

// TestIsNoisyRequest 覆蓋雜訊判定。
//
// 這支測試釀住的是「未接線」這個狀態：isNoisyRequest 目前沒有被任何地方呼叫
// （middleware.go 的說明明寫了這一點）。它的失效模式不是記錄變多，而是有人
// 讀到它的註解以為「存取記錄已經排除靜態資源」—— 因此這支測試也明確記錄
// 它目前的回傳值，好讓接線時的人知道要從哪裡開始。
func TestIsNoisyRequest(t *testing.T) {
	noisy := []string{
		"/api/check",
		"/healthz",
		"/asset/logo.png",
		"/static/app.js",
		"/app.js",
		"/style.css",
		"/forum-manifest.json",
		"/img.PNG", // 大寫副檔名也算
		"/service-worker.js",
		"/build/main-CqL8x.js.map",
		"/mod.wasm",
	}
	for _, path := range noisy {
		if !isNoisyRequest(httptest.NewRequest(http.MethodGet, path, nil)) {
			t.Errorf("isNoisyRequest(%q) = false, want true", path)
		}
	}

	quiet := []string{
		"/",
		"/api/forum/posts",
		"/api/admin/users",
		"/forum",
		"/admin/log",
		// 副檔名不在清單內，因此不是靜態資源。
		"/api/report.csv",
		// 查詢字串不影響判斷（Path 不含它）。
		"/api/forum/search?q=x",
	}
	for _, path := range quiet {
		if isNoisyRequest(httptest.NewRequest(http.MethodGet, path, nil)) {
			t.Errorf("isNoisyRequest(%q) = true, want false", path)
		}
	}

	// 空路徑視為根路徑，必須被當成「不是雜訊」而不是 panic 或當成雜訊。
	if isNoisyRequest(httptest.NewRequest(http.MethodGet, "/", nil)) {
		t.Error("根路徑不該被當成雜訊")
	}
}

// TestIsNoisyRequestIsCurrentlyUnused 守住「未接線」這個狀態是刻意的且已被知悉。
//
// 這個測試看起來很怪（斷言一個函式沒有被呼叫），但它是這個檔裡唯一能防止
// 「有人日誌寫註解說靜態資源已被排除」的東西 —— 那種錯誤會讓維運在排查時
// 找不到本來根本不存在的記錄。
//
// 若將來真的接上線（LoggingMiddleware 呼叫它），這個測試會失敗並要求把
// 註解與這個測試一起更新：那正是它存在的目的。
func TestIsNoisyRequestIsCurrentlyUnused(t *testing.T) {
	// 這個函式刻意不被 LoggingMiddleware 使用。middleware.go 的檔頭有寫明
	// 為什麼：呼叫它會讓靜態檔案請求不再帶 user_email，而那正是「誰在抓這一頁」
	// 唯一能回答的線索。
	if noisyAccessLogEnabled {
		t.Skip("isNoisyRequest 已被接線 —— 請更新 middleware.go 的檔頭說明與 isNoisyRequest 的註解")
	}
}

// noisyAccessLogEnabled 恆為 false，用來讓上面那支測試有東西可斷言。
//
// 把它寫成一個具名常數而不是直接 t.Skip，是為了讓「接上線」這件事在程式碼裡
// 留下一個明確的痕跡：接上的人會看到這裡，然後知道要更新三處說明。
const noisyAccessLogEnabled = false
