/*
Package logger 提供論壇後端的結構化日誌能力：一組等級門檻的 logger 與 HTTP 存取記錄中介層。

職責

  - 單一全域 logger（std），啟動時由 Init 依 config 的 LOG_LEVEL / LOG_FILE / LOG_FORMAT 設定一次。
  - 兩種輸出格式：LOG_FORMAT=text（人眼可讀）與 LOG_FORMAT=json（供日誌收集系統解析）。
  - 透過 context 夾帶每個請求的中繼資料（request_id、來源 IP、使用者 email），
    讓任何深度的 logger.xxxContext 呼叫都能自動附上同一組欄位。

log 輸出格式與寫入目的地

	目的地：永遠寫入 os.Stdout；若 LOG_FILE 非空，另外以 io.MultiWriter 同時寫入該檔案。
	        多寫一份的目的，是讓容器環境可以直接收 stdout，同時保留本機排查用的檔案。
	檔名格式：檔名完全取自 config 的 LOG_FILE 值（config.conf 的 LOG_FILE 鍵），
	        程式端不做任何組裝。開檔旗標為 O_CREATE|O_WRONLY|O_APPEND，
	        0644 權限：以 append 模式寫入，進程重啟不覆寫既有內容。
	切割（rotation）：LOG_FILE 超過 LOG_MAX_SIZE_MB 時，本實作自己換檔：
		目前檔改名為 <LOG_FILE>.1，既有 .1 改名 .2，依此推移，超過
		LOG_MAX_FILES 個輪替檔的最舊一筆被刪除，然後重新開啟 LOG_FILE。
		计数以「寫入的位元組數」累加，Init 時以檔案現有大小為起點，因此
		「舊版無限增長留下的超大檔案」在第一次寫入後就會被輪替。
		取捨：輪替是**行程內**的計數，多個行程共用同一 LOG_FILE 時各自
		輪替各自的計數——一個日誌檔請只讓一個行程寫。刻意不呼叫外部
		logrotate：那需要部署端安裝與排程，而「日誌無限增長」是本專案
		自己可以解決的缺口（見 docs/KNOWN_ISSUES.md）。
	時間格式：2006-01-02T15:04:05.000Z07:00 —— RFC 3339、毫秒精度、帶本機時區位移
	        （例如 2026-10-13T09:22:12.123+08:00；UTC 主機上以 Z 結尾）。取自 time.Now()。
	        帶偏移是為了讓跨主機彙整的紀錄可以直接排序與比對，不需要另外猜每台機器
	        落在哪個時區。同一偏移下欄位的字典序等於時間序；混合不同偏移的主機仍須
	        先解析再比對 —— 而「到底有沒有偏移」在舊格式裡根本無從判斷。

輸出格式（LOG_FORMAT=text，預設）

	每一行都是 key=value 序列，欄位順序固定：

	    time=<時間> level=<LEVEL> request_id="<16 進位>" ip="<來源>" user_email="<email>" caller="<檔名:行號>" msg="<訊息>"

	三條省略規則，全部是為了「一行的寬度只花在有意義的欄位上」：
	  - request_id / ip / user_email：背景工作（啟動訊息、遷移結果）沒有任何請求中繼資料，
	    此時整段省略，行首直接是 time=level=caller=msg=。
	  - caller：呼叫點就在 logger 套件內時省略（例如 LoggingMiddleware 自己呼叫 InfoContext，
	    caller 會指向 middleware.go 而不是任何業務碼）。業務碼的呼叫一律保留。
	  - msg 永遠是最後一個欄位，讓「一行從哪裡開始是訊息」只需要一條規則。
	time 與 level 是程式產生的固定形狀，不加引號，方便 `grep level=ERROR`；
	其餘欄位一律以 Go 字串字面值（strconv.Quote）包覆 —— 見 sanitizeLogValue 的說明。

輸出格式（LOG_FORMAT=json）

	固定七個鍵，schema 不隨訊息而變，收集端（ELK、Loki）可以直接建立索引：

	    {"time":"<時間>","level":"<LEVEL>","request_id":"…","ip":"…","user_email":"…","caller":"…","msg":"…"}

	背景工作的三個中繼鍵仍存在、值為空字串（不出現/缺席的鍵會讓收集端的型別推斷
	更容易出錯，因此刻意保留空值而非省略）。caller 未知或被抑制時同樣是空字串。

日誌等級門檻

	DEBUG < INFO < WARN < ERROR，門檻由 LOG_LEVEL 決定，default 為 INFO。
	寫入前先比對 level，未達門檻就直接 return —— 這道判斷刻意放在格式化之前，
	讓被門檻濾掉的 DEBUG 呼叫連 fmt.Sprintf 與 runtime.Caller 的成本都不用付。

敏感資訊

	本套 logger「不做」欄位遮蔽。sanitizeLogValue 只做三件事：去頭尾空白、移除控制字元、
	截斷到 256 bytes。因此 email、來源 IP 會以明文寫入日誌與檔案；呼叫端若把 token、
	密碼或 Cookie 傳進格式字串，也會被一併寫出。呼叫端必須自行決定傳入哪些欄位。

關鍵設計決策

  - 使用標準庫 log.Logger 作為實際輸出層，並以 flags=0 關閉它自帶的時間戳前綴，
    改由本套件自行組裝完整格式，好讓 text 與 json 兩種輸出一致且可控。
  - 以 sync.Mutex 保護整段「組裝 + 寫入」。除確保同一行不被多執行緒打斷外，
    也能讓 Init 執行緒安全地替換 logger 與檔案句柄。
  - 欄位值一律先過 sanitizeLogValue 再進入 log，這是為了讓使用者可控輸入（email、URL、訊息）
    無法注入換行破壞日誌結構或竄改欄位。
*/
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Level 表示日誌等級，同時充當門檻值：門檻以下的等級會被丟棄。
// 宣告為 int 而非 uint，方便直接與運算子比較。
type Level int

// 日誌等級列舉。順序不可調動，DEBUG = 0 是最低門檻（記錄最多）。
const (
	// DEBUG：開發用的細部診斷資訊，預設門檻（INFO）之下，正式環境不會輸出。
	DEBUG Level = iota
	// INFO：一般流程事件（伺服器啟動、登入成功、存取記錄）。
	INFO
	// WARN：非致命但值得注意，例如 4xx 回應或處理時間偏長。
	WARN
	// ERROR：錯誤路徑被觸發，或 HTTP 5xx。
	ERROR
)

// levelNames 為等級到輸出字串的對照表，決定 text 與 json 輸出中的 level 欄位值。
// 刻意使用 map 而非陣列：未定義的 Level 值（若有）不會被誤編碼成空白或越界。
var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

/*
Logger 是本套件的日誌寫入器，可安全地被多執行緒同時使用。欄位語意如下：

	mu     序列化「組裝訊息 + 寫入」這整段臨界區，也保護 Init 對其餘欄位的替換。
	       log.Logger 本身已有內部鎖，但這裡仍需自己的鎖：level/format 的讀取與
	       中繼欄位的組裝也必須與寫入保持一致，否則同一實例的輸出可能交錯。
	level  門檻值；level 低於門檻的呼叫直接被丟棄，僅在 Init 時寫入。
	logger 實際輸出層；可能是單純 stdout，也可能是 stdout + 檔案的 MultiWriter。
	       為 nil 時 logWithContext 會安全地靜默返回，讓「未呼叫 Init」不致於 panic。
	file   LOG_FILE 對應的檔案句柄，僅供 Close 與重新 Init 時關閉／替換之用；
	       寫入一律透過 logger（MultiWriter），不直接使用此句柄。
	format 輸出格式："text" 或 "json"。空字串在 Init 時會被正規化為 "text"，
	       其他任何值都視為 text（比對採嚴格相等，不做模糊比對）。
*/
type Logger struct {
	mu     sync.Mutex
	level  Level
	logger *log.Logger
	file   *os.File
	format string
	// 輪替狀態。logPath 是 LOG_FILE 的原始路徑（不含 .N 尾巴），maxBytes
	// 是「目前檔寫到多少位元組就換檔」的上界（0 代表不輪替），maxFiles 是
	// 保留的輪替檔數，written 是目前檔已寫入的位元組數——用累加而不是每次
	// Stat，因為每次請求都要問「檔案多大」的 stat 系統呼叫不值得。
	logPath  string
	maxBytes int64
	maxFiles int
	written  int64
}

// std 是全檔唯一的預設 logger 實例。
// 刻意不設 nil，而是在沒有呼叫 Init 的情況下就寫入 os.Stdout（格式 text、門檻 INFO），
// 這樣即使 main 在 Init 之前（例如載入 config 失敗）呼叫 Fatalf，也仍然看得到日誌。
// 因為它是 package 層級變數，Init 與 Close 都必須透過 mu 來同步。
var std = &Logger{
	level:  INFO,
	logger: log.New(os.Stdout, "", 0),
	format: "text",
}

// Init 依設定檔初始化全域 logger。通常在 main 讀完 config 後呼叫一次。
//
// 參數：
//   - levelStr：LOG_LEVEL，可為 DEBUG / INFO / WARN / ERROR，忽略大小寫與前後空白；
//     無法辨識時 parseLevel 會退回 INFO。
//   - logFile：LOG_FILE，空字串代表只輸出到 stdout。
//   - format：LOG_FORMAT，"json" 走結構化輸出，其他值（含空字串）皆視為 text。
//   - maxBytesMB：LOG_MAX_SIZE_MB，單檔輪替的大小上限（MB）。0 或負值代表
//     不輪替——沿用舊的「無限追加」行為。
//   - maxFiles：LOG_MAX_FILES，保留的輪替檔數（不含目前檔）。非正值時以 1
//     處理：「輪替但不留任何舊檔」等于「每次換檔直接丟掉舊檔」，那比
//     不輪替更糟，所以至少保留一個。
//
// 回傳值：只有開啟檔案失敗時才回傳錯誤（例如目錄不存在或權限不足）；
// 設定不正確本身不視為錯誤，會安靜地退回預設值。
//
// 副作用：會改寫 std 的 level、format、logger 與 file（以及輪替狀態）。
// 若原本已有開啟的檔案，會先關閉舊句柄再開新檔案，避免檔案描述符洩漏。
// 已知取捨：Init 以 mu 保護，但 logWithContext 讀取 l.level 是在鎖外；
// 因此執行期間再呼叫 Init 會造成 level 的資料競爭。
// 這在實務上不成問題，因為 Init 只在啟動階段、開始服務之前執行一次。
func Init(levelStr, logFile, format string, maxBytesMB, maxFiles int) error {
	std.mu.Lock()
	defer std.mu.Unlock()

	// 先設定門檻與格式，即使後續開檔失敗也不影響已套用部分（呼叫端會 Fatal 結束行程）。
	std.level = parseLevel(levelStr)
	std.format = strings.TrimSpace(format)
	if std.format == "" {
		std.format = "text"
	}

	if logFile != "" {
		// O_APPEND：多行程或重複呼叫時都只會往檔尾追加，符合日誌語意。
		// 0644：讓同一主機上的其他帳號（如 log 收集帳號）能讀取。
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", logFile, err)
		}
		if std.file != nil {
			// 重新 Init 時先關閉舊句柄；此時舊 logger 仍指向已關閉的檔案，
			// 但下行程式敘句會立刻用新的 MultiWriter 取代，不會寫入已關閉的檔案。
			std.file.Close()
		}
		std.file = f
		std.logPath = logFile
		if maxBytesMB > 0 {
			std.maxBytes = int64(maxBytesMB) << 20
		} else {
			std.maxBytes = 0
		}
		if maxFiles < 1 {
			maxFiles = 1
		}
		std.maxFiles = maxFiles
		// 起點用檔案現有大小：舊版「無限追加」留下的超大檔案，第一次寫入
		// 後就會被輪替，而不是從零開始計數讓它再長一個 maxBytes。
		if stat, statErr := f.Stat(); statErr == nil {
			std.written = stat.Size()
		} else {
			std.written = 0
		}
		// MultiWriter：stdout 與檔案兩邊都寫。任一邊失敗 MultiWriter 都會回傳錯誤，
		// 但此處忽略它 —— 日誌失敗不應該讓呼叫端（每個 HTTP 請求）收到錯誤。
		std.logger = log.New(io.MultiWriter(os.Stdout, f), "", 0)
	} else {
		// 未指定檔案：退回純 stdout。刻意把 std.file 保留為原值不歸零，
		// 讓 Close 仍能關閉先前 Init 開啟的句柄。
		std.logger = log.New(os.Stdout, "", 0)
	}

	return nil
}

// parseLevel 把設定檔字串轉成 Level，忽略大小寫與前後空白。
// 認不出來時一律退回 INFO：寧可少記，也不要因為打錯一個字而讓整個服務安靜無聲。
func parseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// sanitizeLogValue 為要寫入日誌的欄位值做正規化，避免使用者可控輸入破壞日誌結構。
//
// 處理規則：
//  1. 去掉頭尾空白；清洗後為空的值回傳 "-"，讓欄位在輸出中仍然可見而不留空缺。
//  2. 把 \n、\r、\t 換成一般空白 —— 這是對抗「注入換行偽造出一行假的日誌」的主要防線。
//  3. 丟棄其餘 C0 控制字元與 DEL（127），它們會讓終端機或日誌檢視器產生怪異顯示。
//  4. 長度超過 256 bytes 時截斷，且**截在 rune 邊界上**。
//
// 逐 rune 處理而非逐 byte，是為了不切斷 UTF-8 多位元組字元，避免產生亂碼。
// 這一點在規則 3 與規則 4 都成立 —— 規則 4 特別容易漏掉：直接寫
// value[:253] + "..." 會把一個中文字切成三個位元組加上那個字的頭，
// 而欄位裡因此出現一個 U+FFFD。那正是這條規則要防的亂碼。
//
// 規則 1 的「清洗後再判斷」不可省：只檢查原值是否為空白的話，一個只由控制
// 字元組成的值（例如某個標頭剛好是 "\x01"）會變成空字串，而欄位在輸出裡就變成
// 一個沒有值的 key=—— 那正是規則 1 要避免的「不留空缺」。
//
// 已知殘餘限制（本函式刻意不做的事）：
//   - **不做敏感資訊遮蔽**。email 與 IP 會以明文輸出。
//   - **不跳脫引號本身**。這不是漏洞：text 輸出的每個欄位值都由 quoteField 以
//     strconv.Quote 包住（JSON 輸出的跳脫由 encoding/json 負責），因此值裡即使含
//     " fake_ip=9.9.9.9" 這種內容，也留在同一個引號區段內，不會與真正的欄位混淆。
//     text 輸出「無法區分欄位邊界」是早期版本的已知限制，已隨欄位引號化消失。
//     sanitizeLogValue 真正守住的是「不能多出第 N 行」—— 那才是能偽造出一條記錄的
//     層級，也是稽核日誌真正需要守住的那一層。
func sanitizeLogValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}

	// 用 strings.Builder 累積可列印的字元，避免在迴圈中反覆配置字串。
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '\n', '\r', '\t':
			b.WriteRune(' ')
		default:
			// 32 以下為 C0 控制字元、127 為 DEL，兩者都無法列印，直接丟棄。
			if r < 32 || r == 127 {
				continue
			}
			b.WriteRune(r)
		}
	}
	value = strings.TrimSpace(b.String())
	if value == "" {
		// 規則 1 的後半段：原始值非空但清洗後空了（全是控制字元）。
		// 沒有這個分支的話，一個只含 \x01 的標頭會讓輸出出現 "ip=" 這種
		// 語意不明的空欄位。
		return "-"
	}
	if len(value) > maxLogValueBytes {
		// 截斷點必須落在 rune 邊界上，否則會在欄位裡留下一個 U+FFFD。
		//
		// 邊界的找法是從 253 往前退到最近的一個「不是延續位元組」的位置：
		// UTF-8 的延續位元組形狀是 10xxxxxx，而一個字元的**第一個**位元組不是。
		// 因此往前退到第一個非延續位元組就是字元邊界。
		//
		// 這裡刻意用位元組層級的檢查而不是 strings.ToValidUTF8：後者會把壞掉的
		// 位元組換成 U+FFFD（增加 3 bytes），那會讓上限變成「不確定的 256+」，
		// 而這個函式的整個長度規則就是為了讓上限可預測。
		cut := maxLogValueBytes - truncationEllipsisLen
		for cut > 0 && isUTF8Continuation(value[cut]) {
			cut--
		}
		value = value[:cut] + truncationEllipsis
	}
	return value
}

const (
	// maxLogValueBytes 是單一欄位值的長度上限。
	//
	// 256 這個數字是「在 text 輸出裡一眼看不出被截斷、但足以容納一個貼文片段或
	// 一個完整 email + 上下文」之間的折衷。刻意不是更小（那會讓診斷資訊被切掉）
	// 也不是更大（那會讓單一欄位灌爆檔案）。
	maxLogValueBytes = 256
	// truncationEllipsis 是超長時附加的標記。
	//
	// 刻意寫出「有被截斷」這件事：一句沒有標記的半句話，會讓讀者以為那就是
	// 完整的內容，而那正是最容易被誤判成「使用者送了這個字串」的情況。
	truncationEllipsis = "..."
	// truncationEllipsisLen 是上面那個標記的長度。
	//
	// 抽出成常數是為了讓「總長度 ≤ maxLogValueBytes」這個不變條件寫在一處：
	// 標記若被改長，不會有人記得回來調 cut。
	truncationEllipsisLen = len(truncationEllipsis)
)

// isUTF8Continuation 判斷一個位元組是否為 UTF-8 的延續位元組（10xxxxxx）。
func isUTF8Continuation(b byte) bool { return b&0xC0 == 0x80 }

// timeLayout 是每行日誌的時間格式：RFC 3339、毫秒精度、帶本機時區位移
// （例如 2026-10-13T09:22:12.123+08:00；UTC 主機上以 Z 結尾）。
//
// 選它的三個理由：
//  1. 帶時區位移：跨主機彙整時不必推測每台機器在哪個時區 —— 舊格式
//     （2006-01-02 15:04:05.000）把位移丟掉了，彙整後的時間序是不可信的。
//  2. 可被 time.Parse(time.RFC3339, …) 與任何日誌收集端直接解析，
//     不需要自訂 parser。
// 3. 固定寬度，位移格式統一（Z 或 ±HH:MM）。同一時區偏移的多行紀錄，字典序
//    等於時間序，sort / awk 因此可用；混合不同偏移的主機仍須先解析再比對
//    —— 那本來就是帶位移要解決的問題，而不是它引入的問題。
const timeLayout = "2006-01-02T15:04:05.000Z07:00"

/*
loggerPackageDir 是本套件原始碼所在的目錄（正規化成斜線），只用來辨識
「呼叫點就在 logger 套件內」的堆疊帧。

為什麼要省略這種 caller：LoggingMiddleware 自己呼叫 InfoContext / WarnContext /
ErrorContext 記存取記錄，而那些函式的 callerSkip 是固定的 2，所以 caller 一定
指向 middleware.go 的某一行 —— 對讀日誌的人沒有任何價值，只是在每一行存取記錄裡
重複同一個無意義的座標。業務碼的呼叫（caller 指到 handler、blocklist.go 等）
一律保留，那才是出事時要的「這行是誰打的」。

比對的是**目錄**而不是檔名：不同套件存在同名檔案（middleware.go、logger.go）
完全可能，只靠 basename 比對會誤傷。這裡先用完整路徑比對目錄，命中才略過，
之後仍然只把 basename 寫進日誌。

初始化時計算一次：runtime.Caller 的成本只在這裡付一次。
*/
var loggerPackageDir = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return path.Dir(filepath.ToSlash(file))
}()

// quoteField 把要寫進 text 輸出的欄位值包成 Go 字串字面值。
//
// 這是 text 輸出「欄位不會被值混淆」的機制：值裡即使含空白、等號或看起來像
// 另一個欄位的 " fake_ip=9.9.9.9"，也在同一對引號內。sanitizeLogValue 已經
// 移除控制字元，因此 strconv.Quote 實際只處理引號、反斜線與非列印字元。
func quoteField(value string) string { return strconv.Quote(value) }

// logf 是不帶 context 的內部寫入路徑。
// callerSkip 固定為 3：呼叫堆疊為 logWithContext → logf → Debugf/Infof/... → 業務程式碼，
// 所以要往上跳 3 層才能指到真正的呼叫端，runtime.Caller(0) 才是 logWithContext 自己。
func (l *Logger) logf(level Level, format string, args ...interface{}) {
	l.logWithContext(context.Background(), 3, level, format, args...)
}

// logWithContext 是所有寫入的單一實作，負責門檻判斷、欄位組裝與格式化。
//
// 參數：
//   - ctx：若帶有 Metadata（見 middleware.go 的 WithMetadata），會把 request_id /
//     IP / user_email 附到每一條日誌上，讓後端呼叫鏈的記錄可以依 request_id 串起來。
//     傳入 context.Background() 時（走 logf 的路徑）只會得到三個 "-" 的佔位值，
//     輸出時整段中繼欄位被省略（見檔案頭的 text 輸出格式）。
//   - callerSkip：要顯示的呼叫端深度，見 logf 的說明。
//
// caller 欄位的省略規則：呼叫點就在 logger 套件內時不輸出 caller。唯一的呼叫者是
// LoggingMiddleware 自己（記存取記錄），它的座標是固定的 middleware.go，對診斷
// 沒有價值；業務碼的呼叫一律保留。理由見 loggerPackageDir 的說明。
//
// 錯誤處理：json 編碼失敗時不讓錯誤向外傳播（否則每個請求都會因日誌而失敗），
// 而是降級輸出一行保底的文字記錄。
//
// 效能考量：門檻判斷與 args 格式化都放在取得鎖之前，縮短臨界區；
// 但 runtime.Caller 與中繼資料的清洗在鎖內執行（runtime.Caller 會讀取 goroutine 堆疊，
// 需與其他寫入者序列化以免輸出交錯）。
func (l *Logger) logWithContext(ctx context.Context, callerSkip int, level Level, format string, args ...interface{}) {
	// 雙重防呆：允許在未呼叫 Init 的情況下使用（logger 或 receiver 為 nil 時靜默返回），
	// 避免初始化階段的 Fatalf 反過來觸發 panic 而看不到真正的原因。
	if l == nil || l.logger == nil {
		return
	}
	// 門檻判斷。刻意不套 mu：這裡只需要讀取一個 int，成本遠低於鎖的開銷；
	// 代價是若在服務期間呼叫 Init，會與 Init 的寫入產生資料競爭（實務上不會發生）。
	if level < l.level {
		return
	}

	// 從這裡開始進入臨界區，確保「組裝 + 寫入」對其他 goroutine 是原子的。
	l.mu.Lock()
	defer l.mu.Unlock()

	// 毫秒精度足夠排序又不至於讓每行過長；帶本機時區位移（見 timeLayout）。
	now := time.Now().Format(timeLayout)
	// 先組成訊息再清洗一次：format 的引數常來自使用者輸入（email、URL、錯誤內容），
	// 必須在寫入前統一去掉換行與控制字元。
	msg := sanitizeLogValue(fmt.Sprintf(format, args...))

	// 從 context 取出本請求的中繼資料。注意這是 struct 值（value type），
	// 取出後的修改只影響本地副本，不會污染 context 內已存的值。
	meta := MetadataFromContext(ctx)
	// 中繼欄位同樣過清洗：IP 來自 X-Forwarded-For 標頭，是外部可控輸入。
	meta.IP = sanitizeLogValue(meta.IP)
	meta.UserEmail = sanitizeLogValue(meta.UserEmail)
	meta.RequestID = sanitizeLogValue(meta.RequestID)

	// 背景工作（main 的啟動訊息、遷移結果）三個中繼欄位都會是 "-"，此時 text 輸出
	// 整段省略它們、JSON 輸出把它們清成空字串。註：只有三欄「同時」為 "-" 時才算
	// 背景；LoggingMiddleware 會把未登入者填成 "anonymous"，因此 HTTP 存取記錄
	// 走不到這個分支。
	background := meta.RequestID == "-" && meta.IP == "-" && meta.UserEmail == "-"

	// runtime.Caller 的 skip 必須與呼叫者的實際堆疊深度一致，否則會記錄到本套件自己的檔名。
	// 保留 ok 的判斷：部分環境（如已裁剪的堆疊）可能取不到，這時留空字串而非中斷輸出。
	_, file, line, ok := runtime.Caller(callerSkip)
	caller := ""
	if ok && (loggerPackageDir == "" || path.Dir(filepath.ToSlash(file)) != loggerPackageDir) {
		// runtime 回傳的是建置機的完整路徑；正規化成斜線後只取檔名，
		// 避免把部署路徑寫進日誌（一次處理同時涵蓋 Unix 與 Windows 的分隔符號）。
		base := filepath.ToSlash(file)
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		caller = sanitizeLogValue(fmt.Sprintf("%s:%d", base, line))
	}

	if l.format == "json" {
		// 結構化輸出：固定欄位名，方便日誌收集系統（ELK、Loki 等）建立索引。
		entry := map[string]string{
			"time":       now,
			"level":      levelNames[level],
			"request_id": meta.RequestID,
			"ip":         meta.IP,
			"user_email": meta.UserEmail,
			"caller":     caller,
			"msg":        msg,
		}
		// 背景工作沒有任何請求中繼資料，三欄會都是 "-"。此時改寫成空字串，
		// 避免佔位符號被誤認為真實值；鍵仍然保留，讓每行的 schema 一致。
		// 註：只有三欄「同時」為 "-" 時才會清空；LoggingMiddleware 會把未登入者填成
		// "anonymous"，因此 HTTP 存取記錄走不到這個分支。
		if background {
			entry["request_id"] = ""
			entry["ip"] = ""
			entry["user_email"] = ""
		}
		encoded, err := json.Marshal(entry)
		if err != nil {
			// 走到這裡代表 json.Marshal 對 map[string]string 失敗，屬於極度罕見情況。
			// 不向外回傳錯誤（呼叫端是每個 HTTP 請求，會因此全部失敗），
			// 改輸出一行仍具備時間與等級資訊的文字記錄，並省略 msg 以免再觸發問題。
			// 已知取捨：這行是用 fmt 手組的 JSON，欄位值未做引號跳脫；
			// 由於 sanitizeLogValue 已移除換行與控制字元，殘留風險僅限引號字元。
			l.writeLocked(fmt.Sprintf(`{"time":"%s","level":"%s","ip":"%s","user_email":"%s","caller":"%s","msg":"failed to encode log entry"}`,
				now, levelNames[level], meta.IP, meta.UserEmail, caller))
			return
		}
		l.writeLocked(string(encoded))
	} else {
		// 純文字輸出：key=value 序列，msg 永遠在最後（見檔案頭的格式說明）。
		// time 與 level 是固定形狀，不加引號，方便 grep level=ERROR；
		// 其餘欄位一律引號化，讓值裡的空白與 " key=value" 不會混淆欄位邊界。
		var b strings.Builder
		b.WriteString("time=")
		b.WriteString(now)
		b.WriteString(" level=")
		b.WriteString(levelNames[level])
		if !background {
			b.WriteString(" request_id=")
			b.WriteString(quoteField(meta.RequestID))
			b.WriteString(" ip=")
			b.WriteString(quoteField(meta.IP))
			b.WriteString(" user_email=")
			b.WriteString(quoteField(meta.UserEmail))
		}
		if caller != "" {
			b.WriteString(" caller=")
			b.WriteString(quoteField(caller))
		}
		b.WriteString(" msg=")
		b.WriteString(quoteField(msg))
		l.writeLocked(b.String())
	}
}

// writeLocked 寫出一行並更新輪替計數。必须在 mu 已鎖定的臨界區內呼叫：
// written 與 file 的更新、以及換檔本身，都與「組裝 + 寫入」共用同一把鎖。
//
// 計數用「行長度 + 換行」累加，而不是每次 Stat 檔案：log.Logger.Print 會
// 補一個 \n，所以這個累加與檔案實際增長一致（MultiWriter 的 stdout 部分
// 不佔磁盘，只有檔案部分需要計數——而我們計的是「打算寫入的位元組數」，
// 兩邊同值）。
func (l *Logger) writeLocked(line string) {
	l.logger.Print(line)
	if l.file == nil || l.maxBytes <= 0 {
		return
	}
	l.written += int64(len(line)) + 1
	if l.written >= l.maxBytes {
		l.rotateLocked()
	}
}

// rotateLocked 換檔：目前檔改名 .1，既有 .1 改名 .2，依此推移；超過
// maxFiles 個的最舊輪替檔被刪除，然後重新開啟目前檔。必须在 mu 已鎖定時
// 呼叫。
//
// 順序不可簡化：不先删掉 .maxFiles，Rename 到它時會覆寫（POSIX）或失敗
// （某些平台），而「先删最舊、再逐檔往前推」是兩邊都安全的写法。
//
// 換檔失敗（例如目錄被移除、磁碟滿）的處置是「退回純 stdout」：日誌不能
// 因為檔案系統故障而讓每個請求變慢或失敗，而 stdout 在容器環境本來就是
// 主要目的地。这里刻意不重試——下一次寫入會再尝试開檔吗？不会：句柄已
// 為 nil，直到下一次 Init。這是可接受的退化：日誌仍完整地到 stdout，
// 只是少了檔案副本。
func (l *Logger) rotateLocked() {
	if l.file == nil || l.maxFiles < 1 {
		return
	}
	l.file.Close()

	// 最舊的一檔（.maxFiles）先刪：不刪的話，下一步「.maxFiles-1 推移成
	// .maxFiles」的 rename 在 POSIX 會靜默覆寫、在 Windows 會因目標已存在
	// 而失敗 —— 「先删最舊、再逐檔往前推」是兩邊都安全的順序。
	_ = os.Remove(l.logPath + "." + strconv.Itoa(l.maxFiles))
	for i := l.maxFiles - 1; i >= 1; i-- {
		source := l.logPath + "." + strconv.Itoa(i)
		if _, err := os.Stat(source); err != nil {
			continue
		}
		// 目標是 i+1 而不是 i：.1 要推移成 .2、.2 推移成 .3，依此推移。
		// 寫成與 source 相同的 i 會讓 Rename 變成 no-op（同名改名），
		// 症狀是永遠只存在一個輪替檔、更舊的紀錄靜默消失。
		_ = os.Rename(source, l.logPath+"."+strconv.Itoa(i+1))
	}
	// 最舊的一檔（.maxFiles）在最前面已被刪掉，這裡把目前檔推移成 .1。

	_ = os.Rename(l.logPath, l.logPath+".1")

	f, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		l.file = nil
		l.written = 0
		l.logger = log.New(os.Stdout, "", 0)
		return
	}
	l.file = f
	l.written = 0
	l.logger = log.New(io.MultiWriter(os.Stdout, f), "", 0)
}

// Debugf 輸出 DEBUG 等級的格式化日誌，使用 printf 風格格式化。
// 屬於「非 request scope」的呼叫，因此不會附帶任何 request 中繼資料。
func Debugf(format string, args ...interface{}) {
	std.logf(DEBUG, format, args...)
}

// Infof 輸出 INFO 等級的格式化日誌，參數意義同 Debugf。
func Infof(format string, args ...interface{}) {
	std.logf(INFO, format, args...)
}

// Warnf 輸出 WARN 等級的格式化日誌，參數意義同 Debugf。
// 用途為「非致命但需要留意」的情況，例如用戶輸入不合法或依賴服務變慢。
func Warnf(format string, args ...interface{}) {
	std.logf(WARN, format, args...)
}

// Errorf 輸出 ERROR 等級的格式化日誌，參數意義同 Debugf。
// 注意：只記錄、不改變程式流程；若需要中止執行請改用 Fatalf。
func Errorf(format string, args ...interface{}) {
	std.logf(ERROR, format, args...)
}

// DebugfContext 與 Debugf 相同，但會從 ctx 取出 request_id / IP / user_email 一併輸出。
// 應用在 HTTP 處理路徑中優先選用這組 *Context 版本，才能把記錄串到同一個請求上。
// callerSkip 為 2：堆疊為 logWithContext → DebugfContext → 業務程式碼。
func DebugfContext(ctx context.Context, format string, args ...interface{}) {
	std.logWithContext(ctx, 2, DEBUG, format, args...)
}

// InfofContext 輸出帶請求中繼資料的 INFO 日誌。
func InfofContext(ctx context.Context, format string, args ...interface{}) {
	std.logWithContext(ctx, 2, INFO, format, args...)
}

// WarnfContext 輸出帶請求中繼資料的 WARN 日誌。
func WarnfContext(ctx context.Context, format string, args ...interface{}) {
	std.logWithContext(ctx, 2, WARN, format, args...)
}

// ErrorfContext 輸出帶請求中繼資料的 ERROR 日誌，是本專案處理路徑最常用的日誌進入點。
func ErrorfContext(ctx context.Context, format string, args ...interface{}) {
	std.logWithContext(ctx, 2, ERROR, format, args...)
}

// Debug 以 fmt.Sprint 的方式（引數以空白串接）輸出 DEBUG 日誌，不需自行排格式。
func Debug(args ...interface{}) {
	std.logf(DEBUG, "%s", fmt.Sprint(args...))
}

// Info 以 fmt.Sprint 的方式輸出 INFO 日誌。
func Info(args ...interface{}) {
	std.logf(INFO, "%s", fmt.Sprint(args...))
}

// Warn 以 fmt.Sprint 的方式輸出 WARN 日誌。
func Warn(args ...interface{}) {
	std.logf(WARN, "%s", fmt.Sprint(args...))
}

// Error 以 fmt.Sprint 的方式輸出 ERROR 日誌。
func Error(args ...interface{}) {
	std.logf(ERROR, "%s", fmt.Sprint(args...))
}

// DebugContext 輸出帶請求中繼資料的 DEBUG 日誌，引數以 fmt.Sprint 串接。
func DebugContext(ctx context.Context, args ...interface{}) {
	std.logWithContext(ctx, 2, DEBUG, "%s", fmt.Sprint(args...))
}

// InfoContext 輸出帶請求中繼資料的 INFO 日誌，引數以 fmt.Sprint 串接。
func InfoContext(ctx context.Context, args ...interface{}) {
	std.logWithContext(ctx, 2, INFO, "%s", fmt.Sprint(args...))
}

// WarnContext 輸出帶請求中繼資料的 WARN 日誌。
// LoggingMiddleware 在收到 4xx 或請求逾時時走的就是這個進入點。
func WarnContext(ctx context.Context, args ...interface{}) {
	std.logWithContext(ctx, 2, WARN, "%s", fmt.Sprint(args...))
}

// ErrorContext 輸出帶請求中繼資料的 ERROR 日誌。
// LoggingMiddleware 在收到 5xx 時走的就是這個進入點。
func ErrorContext(ctx context.Context, args ...interface{}) {
	std.logWithContext(ctx, 2, ERROR, "%s", fmt.Sprint(args...))
}

// Fatal 以 ERROR 等級寫出一條日誌後立即以狀態碼 1 結束行程。
// 用途：啟動階段的不可復原錯誤（設定檔載入失敗、資料庫連不上）。
// 因為 os.Exit 不會執行 defer，呼叫端掛著的 defer（例如 logger.Close）不會被執行，
// 所以這類訊息務必在寫出後就結束行程，日誌檔案句柄由作業系統回收。
func Fatal(args ...interface{}) {
	std.logf(ERROR, "%s", fmt.Sprint(args...))
	os.Exit(1)
}

// Fatalf 以 printf 風格格式化寫出日誌後結束行程，參數意義同 Debugf。
func Fatalf(format string, args ...interface{}) {
	std.logf(ERROR, format, args...)
	os.Exit(1)
}

// Close 關閉 LOG_FILE 對應的檔案句柄，通常由 main 以 defer 呼叫。
// 將 file 設為 nil 而非保留，可讓後續 Close 成為安全的冪等操作。
// 關閉後 std.logger 仍指向含該檔案的 MultiWriter，因此 Close 之後的寫入會失敗於檔案部分；
// 正常流程中 Close 一定發生在行程結束前的最後一刻，不會有後續寫入。
func Close() {
	std.mu.Lock()
	defer std.mu.Unlock()
	if std.file != nil {
		std.file.Close()
		std.file = nil
	}
}
