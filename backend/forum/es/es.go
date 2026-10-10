/*
es 套件：Elasticsearch 的最小傳輸層（backend/forum/es/es.go）。

【為什麼自己打 HTTP 而不用官方 SDK】

  go.mod 刻意不新增任何相依套件。本專案對 ES 的需求只有四種動作
  （確保索引、寫入單筆、刪除單筆、搜尋），這四種都是 ES 的標準 REST 端點，
  官方 SDK 帶來的是編碼、指標、快照與版本相容性矩陣，而本站的 ES 是自己
  控管的一個單節點實例。少一個相依就少一份升級與安全更新的負擔。


【本套件的邊界：只做索引的讀寫，不做「內容的真相」】

  搜尋結果回傳的是「符合條件的貼文 ID 與分數」，不是文件內容本身。
  呼叫端（httpapi.search.go）拿著這串 ID 回 MySQL 取真正的貼文，包含按讚數、
  留言數、附圖 token 與去識別化的作者顯示名。

  這不是偷懶，而是刻意的三個好處：
    1. 索引裡不會出現 email。公開搜尋若直接回文件內容，等於讓 ES 成為
       一份「email + 全文」的副本，去識別化的邊界就多了一個要防守的地方。
    2. 索引與 MySQL 不一致時（寫入時序、重建失敗），使用者看到的是
       MySQL 的版本，而不是可能過期的副本。
    3. 搜尋的輸出形狀與貼文列表完全一致（同一個 forumPost 結構），
       前端因此可以用同一張卡片元件渲染搜尋結果與動態。

【設定與降級】

  baseURL 為空字串時整個 Client 視為未啟用（Enabled() 為 false），
  所有方法都回 ErrDisabled 而不發出任何請求。呼叫端因此可以把
  「沒有設定 ES」與「ES 連不上」走同一條降級路徑（改用 MySQL LIKE 搜尋）。

【驗證】

   ES 啟用安全性（xpack.security.enabled）時，未帶驗證的每個請求都會拿到
   401 security_exception。本套件支援 ES 原生支援的兩種 realm：Basic Auth 與
   API key（見 Auth）。兩者都不會出現在日誌或錯誤訊息裡 —— 日誌只記錄
   AuthKind() 的三個值之一。

【逾時】

  單一 HTTP.Client 的 Timeout 固定 3 秒，與 health.go 的探測逾時一致。
  理由：ES 是輔助性相依，搜尋與索引都不該讓 HTTP 請求被它拖住。
  貼文列表的預設 SLA 是「MySQL 查詢 + 25 次 per-row 查詢」，
  3 秒的 ES 逾時讓總時間仍然落在反向代理常見的 10 秒上限內。
*/

package es

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 逾時與大小上限的常數。刻意是常數而非設定項：逾時是「本站可接受的等待時間」，
// 大小上限則是「本功能單次回應的合理天花板」。兩者調大都不會讓功能變好，
// 只会讓故障時的請求變慢或佔用更多記憶體。
const (
	requestTimeout   = 3 * time.Second
	defaultIndexName = "forum_posts"
	bulkChunkSize    = 200
	maxResultWindow  = 10000
	// maxErrorBodySize 限制「錯誤回應」讀進來的位元組數。錯誤本文只用於診斷，
	// 不需要全文，截斷也無妨（parseHTTPError 會退化成使用原始本文）。
	maxErrorBodySize = 8 << 10
	// maxResponseBodySize 限制「成功回應」的大小。上限抓得鬆是必要的：
	// _bulk 的回應會隨著批次內的文件數線性成長（每筆約 250 bytes，200 筆約 50KB），
	// 把所有回應都用 8KB 截斷會在重建貼文時讓 JSON 解析失敗（實測：61 篇就超過了
	// 8KB，錯誤訊息是令人摸不著頭緒的 "unexpected end of JSON input"）。
	maxResponseBodySize = 8 << 20
)

// ErrDisabled 是「未設定 ES_URL」時所有方法回傳的 sentinel error。
//
// 用 error 而不是 (bool, error) 的回傳值，是為了讓呼叫端可以用
// errors.Is 單獨辨識「這台站本來就沒開 ES」與「開了但連不上」：
// 前者不必記警告日誌（它是預期中的部署選項），後者必須記。
var ErrDisabled = errors.New("es: not configured")

// 驗證種類。這三個值是日誌與報告顯示用的標籤，不含任何機密內容。
const (
	AuthKindNone   = "none"
	AuthKindBasic  = "basic"
	AuthKindAPIKey = "apikey"
)

// Auth 是連線 ES 時使用的驗證資訊，對應 ES 原生支援的兩種 realm。
//
//	Username / Password	HTTP Basic Auth（ES 的 native realm）
//	APIKey		ES 的 API key，以 Authorization: ApiKey <base64> 送出
//
// 三者都留空代表「這個 ES 沒有啟用安全性」，此時完全不送 Authorization 標頭。
//
// 兩者同時設定時以 APIKey 為主：API key 可以只授權單一索引的讀寫，而基本帳密
// 通常是權限更大的內建帳號（elastic）。這個優先序只在 Kind 裡決定一次，
// 因此「日誌說用哪一種」與「實際送出哪一種」不可能不一致。
//
// 之所以不做「401 就退回無驗證」這種自動降級：那會把「密碼寫錯」與
// 「ES 沒開安全性」混在一起，而兩者的處置完全相反（前者要修設定，後者
// 本來就不該送憑證）。寧可讓搜尋退回 MySQL LIKE（那條路徑本來就存在），
// 也不要偷偷改送驗證標頭。
type Auth struct {
	Username string
	Password string
	APIKey   string
}

// Kind 報告這個 Auth 會用哪一種驗證。沒有任何資訊時是 AuthKindNone。
func (a Auth) Kind() string {
	switch {
	case strings.TrimSpace(a.APIKey) != "":
		return AuthKindAPIKey
	case strings.TrimSpace(a.Username) != "":
		return AuthKindBasic
	default:
		return AuthKindNone
	}
}

// HeaderValue 回傳 Authorization 標頭的值；Kind 為 none 時是空字串。
func (a Auth) HeaderValue() string {
	switch a.Kind() {
	case AuthKindAPIKey:
		return "ApiKey " + apiKeyCredential(a.APIKey)
	case AuthKindBasic:
		// net/http 的 SetBasicAuth 只作用在已建立的 Request 上，而本套件的
		// 標頭在建構 Client 時就決定；兩者產生的是同一個值。
		credentials := strings.TrimSpace(a.Username) + ":" + a.Password
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
	default:
		return ""
	}
}

// apiKeyCredential 把設定值轉成 ES 期待的憑證本體。
//
// ES 的 apikey realm 收的是 base64("id:api_key")，而操作員手上常見的值有兩種：
// console 與 curl 範例給的已編碼字串，以及 _security/api_key 回傳的
// {"id": ..., "api_key": ...} 兩截拚成的 "id:key"。分辨方式是看冒號 ——
// std base64 的字元集（A-Za-z0-9+/=）永遠不含冒號，而未編碼的兩截式一定含有一個。
// 因此含冒號視為原始格式並在此編碼，其餘原樣送出。
//
// 為什麼要容忍兩種：ES 的錯誤訊息區分不了「你給了未編碼的值」與「key 過期」，
// 兩者都是一句 security_exception。讓部署者貼哪一種都能用，勝過要求他先自己
// 確認編碼狀態。
func apiKeyCredential(raw string) string {
	credential := strings.TrimSpace(raw)
	if !strings.Contains(credential, ":") {
		return credential
	}
	return base64.StdEncoding.EncodeToString([]byte(credential))
}

// PostDocument 是論壇貼文在 ES 中的文件格式。
//
//	ID	對應 forum_posts.id，同時作為 ES 的 _id。
//		以 MySQL 主鍵當文件 ID 有兩個好處：重複寫入是冪等的覆寫
//		（不需要先查再決定新增或更新），刪除也不需要任何查詢條件。
//	Content	貼文內文，唯一的可搜尋欄位。cjk analyzer 見 EnsureIndex。
//	AuthorEmail	作者信箱，只供後臺以「term」精確比對（見 Search 的 includeAuthor）。
//		公開搜尋的結果永遠不會帶出這個欄位，見檔頭說明。
//	CreatedAt	建立時間，同時作為同分時的次要排序依據。
type PostDocument struct {
	ID          int64     `json:"id"`
	Content     string    `json:"content"`
	AuthorEmail string    `json:"authorEmail"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Hit 是一筆搜尋結果。只帶 ID 與分數，理由見檔頭說明。
type Hit struct {
	ID    int64
	Score float64
}

// Result 是一次搜尋的結果。
//
//	Hits	依相關性排序的 ID 列表（已套用呼叫端要求的 from / size）。
//	Total	符合條件的總筆數。ES 8 之後 total 預設只算到 10000，
//		這裡以 track_total_hits=true 取得真實值。
type Result struct {
	Hits  []Hit
	Total int
}

// HTTPError 是 ES 回應 4xx / 5xx 時的錯誤型別，保留狀態碼與 ES 的錯誤類型，
// 讓呼叫端能分辨「索引不存在」（需要重建）與「查詢語法錯誤」這類不同處置。
type HTTPError struct {
	Status int
	Type   string
	Reason string
}

func (e *HTTPError) Error() string {
	if e.Type == "" {
		return fmt.Sprintf("es: http status %d: %s", e.Status, e.Reason)
	}
	return fmt.Sprintf("es: http status %d (%s): %s", e.Status, e.Type, e.Reason)
}

// StatusOf 取出錯誤的 HTTP 狀態碼；不是 HTTP 錯誤時回傳 0。
//
// 呼叫端（例如刪除單篇貼文時容忍 404）需要這個分流，而
// errors.As 搭配 type assertion 會讓每個呼叫處都寫三行。
func StatusOf(err error) int {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status
	}
	return 0
}

// Client 是 Elasticsearch 的傳輸層。值不可變（建構後不再修改），
// 因此可以被多個 goroutine 同時使用：http.Client 本身是安全可重用的。
type Client struct {
	baseURL string
	index   string
	client  *http.Client
	// authKind 與 authHeader 在建構時算一次，之後不再改變。
	// authHeader 為空字串代表這個 ES 不需要驗證，do 因此不必為每個請求重新
	// 判斷「哪一種模式、要不要編碼」。
	authKind   string
	authHeader string
}

// New 建立 Client。baseURL 為空字串時 Client 仍可建構，但所有方法都回
// ErrDisabled（不 panic、不發請求），因此呼叫端不必在每個呼叫點判斷
// 「設定檔裡到底有沒有這一行」。
//
// auth 是驗證資訊；三者皆為空時以無驗證方式連線（見 Auth）。
func New(baseURL, index string, auth Auth) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	index = strings.TrimSpace(index)
	if index == "" {
		index = defaultIndexName
	}
	return &Client{
		baseURL:    baseURL,
		index:      index,
		client:     &http.Client{Timeout: requestTimeout},
		authKind:   auth.Kind(),
		authHeader: auth.HeaderValue(),
	}
}

// Enabled 報告這個 Client 是否設定了 ES 位址。
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

// AuthKind 報告實際使用的驗證種類（none / basic / apikey），供啟動日誌與
// 設定檔檢查報告顯示。回傳值不含任何機密內容。
func (c *Client) AuthKind() string {
	if c == nil {
		return AuthKindNone
	}
	return c.authKind
}

// IndexName 回傳實際使用的索引名，僅供日誌顯示。
func (c *Client) IndexName() string {
	if c == nil {
		return ""
	}
	return c.index
}

// do 送出一次請求並回傳回應本文。
//
// 逾時由 c.client.Timeout 負責，不在這裡另外設 deadline：caller 傳進來的
// ctx（例如 HTTP 請求的 r.Context()）才是「使用者已經離開」的唯一訊號。
func (c *Client) do(ctx context.Context, method, path, contentType string, body []byte) ([]byte, error) {
	if !c.Enabled() {
		return nil, ErrDisabled
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	// 驗證標頭在這裡加，因此每一個端點（Ping、EnsureIndex、搜尋、_bulk、刪除）
	// 都一致地帶上憑證 —— 漏掉其中一個的症狀是「大部分功能正常，只有某一項
	// 一直 401」，而那比全部 401 更難查。
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 錯誤回應與成功回應用不同的讀取上限，理由見常數的說明（bulk 的回應會很大，
	// 而 8KB 的上限只適合診斷用的錯誤本文）。
	if resp.StatusCode >= 400 {
		payload, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		if readErr != nil {
			return nil, readErr
		}
		return nil, parseHTTPError(resp.StatusCode, payload)
	}
	// 多讀 1 byte 並據此判斷是否超限：若只是安靜地截斷，呼叫端會在解析
	// 半截 JSON 時收到與真正原因無關的錯誤訊息。明確回報「回應過大」，
	// 遠勝於一個指向錯誤方向的解析錯誤。
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > maxResponseBodySize {
		return nil, fmt.Errorf("es: response exceeds %d bytes", maxResponseBodySize)
	}
	return payload, nil
}

// parseHTTPError 把 ES 的錯誤本文轉成 HTTPError。
//
// ES 的錯誤格式是 {"error": {"type": ..., "reason": ...}, "status": 400}，
// 但不同端點的形狀並不完全一致（例如 index not found 是
// {"error": {"root_cause": [...], "type": ...}}），因此解析失敗時
// 退回使用原始本文作為 Reason，絕不因為解析不了就丟掉診斷資訊。
func parseHTTPError(status int, payload []byte) error {
	var envelope struct {
		Error struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Error.Type == "" {
		reason := strings.TrimSpace(string(payload))
		if reason == "" {
			reason = http.StatusText(status)
		}
		return &HTTPError{Status: status, Reason: reason}
	}
	return &HTTPError{Status: status, Type: envelope.Error.Type, Reason: envelope.Error.Reason}
}

// indexMapping 是建立索引時使用的設定與欄位對應。
//
// 四個決定值得說明：
//
//  1. content 的 analyzer 選 cjk（內建的 CJK bigram 分析器），把中文切成重疊的
//     二字組合：「檢舉管理」→ 檢舉／舉管／管理。對照預設的 standard，後者把
//     中文切成單字，搜尋「檢舉」時會同時命中「檢」與「舉」而帶出大量無關結果。
//     實測（ES _analyze）：cjk 保留英文單詞（iphone、OK→ok）。
//
//  2. content.standard 這個子欄位是為了補上 cjk 的唯一缺陷：bigram 過濾器
//     不產生單字 token，因此「的」「有」這種單字查詢逐字比對得到「的」這個
//     token，卻沒有任何文件以它作為索引 term（實測 cjk 對「好的，像這樣」
//     只產出 好的／像這／這樣），於是單字查詢必定零命中。standard 分析器會把
//     中文切成單字，補上這一層召回；搜尋時兩欄以 should 並聯，cjk 那條負責
//     短語的精準度，standard 那條負責單字與邊緣情形。
//
//  3. authorEmail 用 keyword：它是「精確比對作者」的欄位，後臺搜尋時用
//     term 而非 analyzed match，因此不該被切成 token。內容欄位則是 text。
//
//  4. number_of_replicas = 0：部署形態是單節點 Docker（主站原本的
//     history_posts 索引也是 rep=0）。留 1 會讓索引恆為 yellow health，
//     徒增健康檢查的雜訊。文件數量成長到需要多節點時，這裡要一起改。
const indexMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "id": { "type": "long" },
      "content": {
        "type": "text",
        "analyzer": "cjk",
        "fields": {
          "standard": { "type": "text", "analyzer": "standard" }
        }
      },
      "authorEmail": { "type": "keyword" },
      "createdAt": { "type": "date" }
    }
  }
}`

// EnsureIndex 建立索引（若尚未存在）。
//
// 冪等：索引已存在時 ES 回 400 resource_already_exists_exception，這裡視為成功。
// 這讓「每次啟動都呼叫一次」是安全的，與 MySQL 遷移的 CREATE TABLE IF NOT EXISTS
// 是同一個設計意圖。
//
// 刻意不做 mapping 更新：ES 不允許把已存在的文字欄位換掉分析器（只能加欄位、
// 不能改既有欄位的分析方式），而「改 mapping」在 ES 的設計裡是一律重建索引。
// 因此欄位定義有變更時，部署步驟是手動刪掉舊索引再讓啟動流程重建：
//
//	curl -X DELETE http://<es>:9200/forum_posts
//
// 這個取捨是刻意的：讓服務在啟動時偷偷重建索引，會讓「我改了設定檔，為什麼
// 還沒生效」變成一個難以追查的問題。
func (c *Client) EnsureIndex(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodPut, "/"+c.index, "application/json", []byte(indexMapping))
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.Type == "resource_already_exists_exception" {
			return nil
		}
		return err
	}
	return nil
}

// Ping 確認 ES 可連線且回應正常（GET / 回傳版本資訊）。
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodGet, "/", "", nil)
	return err
}

// IndexPost 寫入或覆寫單篇貼文的文件。
//
// 用 PUT /{index}/_doc/{id} 而非 POST：_id 由 MySQL 主鍵決定，因此重送同一篇
// 貼文（例如管理員連按兩次儲存）是冪等的覆寫，不會產生重複文件。
//
// 不帶 refresh 參數：預設 1 秒的 refresh 間隔代表「剛發的文要等一下才搜得到」，
// 這對論壇是正確的取捨 —— 為了讓每篇貼文的寫入都同步可搜尋而開
// refresh=wait_for，會讓每次發文多花掉一個 refresh 週期的延遲。
func (c *Client) IndexPost(ctx context.Context, doc PostDocument) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = c.do(ctx, http.MethodPut,
		"/"+c.index+"/_doc/"+strconv.FormatInt(doc.ID, 10), "application/json", body)
	return err
}

// DeletePost 刪除單篇貼文的文件。文件已不存在（404）視為成功：
// 索引的目標狀態就是「這裡不該有這篇文」，重複刪除不該讓呼叫端視為失敗。
func (c *Client) DeletePost(ctx context.Context, id int64) error {
	_, err := c.do(ctx, http.MethodDelete, "/"+c.index+"/_doc/"+strconv.FormatInt(id, 10), "", nil)
	if StatusOf(err) == http.StatusNotFound {
		return nil
	}
	return err
}

// IndexPosts 以 _bulk 批次寫入，供啟動時的全量重建使用。
//
// 分批送（bulkChunkSize 篇一批）而不是一次送出全部：單一請求過大時
// ES 會回 413，而錯誤裡不會指出是哪一批，逐批送出才能回報進度、
// 也能在記憶體峰值上守住上限。
func (c *Client) IndexPosts(ctx context.Context, docs []PostDocument) error {
	if len(docs) == 0 {
		return nil
	}
	for start := 0; start < len(docs); start += bulkChunkSize {
		end := start + bulkChunkSize
		if end > len(docs) {
			end = len(docs)
		}
		if err := c.bulkChunk(ctx, docs[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// bulkChunk 送出一批文件並檢查回應中的 errors 旗標。
//
// 檢查 errors 旗標是必要的：_bulk 對「部分失敗」回 200，
// 只看狀態碼會把寫入失敗的文件當成成功，重建後的索引因此少了一截資料。
func (c *Client) bulkChunk(ctx context.Context, docs []PostDocument) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, doc := range docs {
		// NDJSON 的第一行是動作行（決定 index/delete 與 _id），第二行才是文件本體。
		// 這兩行都必須是完整 JSON 且以換行結尾，否則 ES 會回 400。
		meta := map[string]map[string]string{
			"index": {"_index": c.index, "_id": strconv.FormatInt(doc.ID, 10)},
		}
		if err := encoder.Encode(meta); err != nil {
			return err
		}
		if err := encoder.Encode(doc); err != nil {
			return err
		}
	}
	payload, err := c.do(ctx, http.MethodPost, "/_bulk", "application/x-ndjson", buf.Bytes())
	if err != nil {
		return err
	}
	var result struct {
		Errors bool `json:"errors"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return err
	}
	if result.Errors {
		// 只回報「有錯誤」而不列出是哪幾筆：_bulk 的失敗明細是整個 items 陣列，
		// 在數百筆的重建裡逐一轉成錯誤訊息只會淹掉真正的問題。
		// 診斷方式是看 ES 自己的日誌，或針對單篇貼文呼叫 IndexPost。
		return fmt.Errorf("es: bulk index reported per-document errors")
	}
	return nil
}

// Search 搜尋貼文。
//
// query	使用者輸入的關鍵字（呼叫端負責 TrimSpace 與長度限制）
// from	跳過的筆數；size	取回的筆數
// includeAuthor	為 true 時額外比對 authorEmail 的完整值（後臺治理搜尋用）。
//
// 查詢語意：bool.should 放四個條件（content 的 cjk、content.standard 的整詞、
// content.standard 的前綴、authorEmail 的 term）加上 minimum_should_match=1，
// 代表「命中任一條件即可」。
//
// 兩條 content 整詞條件各自指定 analyzer，讓查詢端的分詞與欄位定義必然一致 ——
// 兩邊不一致是中文搜尋最常見的「明明有這篇卻搜不到」。相關性由 BM25 決定，
// 不做任何人工加權：cjk 那條命中完整 bigram 的分數本來就會高於 standard
// 的單字命中，因此多字查詢自然偏向精準的結果。
//
// 排序：先 _score（相關性）再 createdAt desc。同分時取新的，與貼文列表的
// 直覺一致；純按分數排序會讓結果順序在每次重建索引後微微變動。
func (c *Client) Search(ctx context.Context, query string, from, size int, includeAuthor bool) (Result, error) {
	if !c.Enabled() {
		return Result{}, ErrDisabled
	}
	if from < 0 {
		from = 0
	}
	if size < 1 {
		size = 1
	}
	// 超過 max_result_window 的 from 會被 ES 以 400 拒絕（illegal_argument_exception）。
	// 夾住而不是回錯，是因為「翻太深」對使用者而言等同「沒有更多了」，
	// 夾住讓最後一頁仍能正常顯示空結果。
	if from+size > maxResultWindow {
		from = maxResultWindow - size
		if from < 0 {
			from = 0
		}
	}

	should := []interface{}{
		map[string]interface{}{
			"match": map[string]interface{}{"content": map[string]interface{}{
				"query": query, "operator": "and", "analyzer": "cjk",
			}},
		},
		map[string]interface{}{
			"match": map[string]interface{}{"content.standard": map[string]interface{}{
				"query": query, "operator": "and", "analyzer": "standard",
			}},
		},
		/*
		 * 前綴比對。沒有這一條的話，「elastic」只會以整詞比對，而索引裡的詞是
		 * 「elasticsearch」，結果是零命中 —— 使用者實際想要找的明明就在那裡。
		 * 用 match_phrase_prefix 而不是 wildcard：它只把「最後一個詞」當前綴，
		 * 因此 "elastic search" 仍要求 elastic 開頭的詞後面接 search，不會變成
		 * 任意位置命中。放在 should 最後只是影響加權順序，不影響能否命中。
		 */
		map[string]interface{}{
			"match_phrase_prefix": map[string]interface{}{"content.standard": map[string]interface{}{
				"query": query, "analyzer": "standard",
			}},
		},
	}
	if includeAuthor {
		// authorEmail 是 keyword，term 為完全比對（大小寫敏感）。
		// 使用者輸入的 email 因此必須與資料庫中的寫法逐字相同。
		should = append(should, map[string]interface{}{
			"term": map[string]interface{}{"authorEmail": strings.ToLower(query)},
		})
	}
	body, err := json.Marshal(map[string]interface{}{
		"from":             from,
		"size":             size,
		"track_total_hits": true,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               should,
				"minimum_should_match": 1,
			},
		},
		"sort": []interface{}{"_score", map[string]interface{}{"createdAt": "desc"}},
	})
	if err != nil {
		return Result{}, err
	}
	payload, err := c.do(ctx, http.MethodPost, "/"+c.index+"/_search", "application/json", body)
	if err != nil {
		return Result{}, err
	}
	/*
		刻意把 total 宣告成單層的 json.RawMessage，而不是
		`Total struct{ Total json.RawMessage \`json:"total"\` } \`json:"total"\`` 兩層巢狀。

		兩層巢狀在 encoding/json 下會靜默失效：同名的欄位與相同的 tag 在巢狀兩層
		出現時，解碼器不會把值寫進去（err 為 nil、欄位保持零值），於是 hits.total
		永遠被讀成 0。實測（Go 1.25）確認如此；單層寫法同一份回應可正確解出
		{"value":7,"relation":"eq"}。這種「不報錯但拿到零值」的失效模式正是
		最難察覺的一種，所以在此留下說明，避免日後有人「整理」成巢狀版本。
	*/
	var decoded struct {
		Hits struct {
			Total json.RawMessage `json:"total"`
			Hits  []struct {
				ID     string  `json:"_id"`
				Score  float64 `json:"_score"`
				Source struct {
					ID int64 `json:"id"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return Result{}, err
	}

	result := Result{Hits: make([]Hit, 0, len(decoded.Hits.Hits))}
	// total 同時支援 {"value":N,"relation":"eq"}（ES 7 以後）與純數字（舊格式，
	// track_total_hits 關閉時）兩種形狀。兩種都解析失敗時保持 0：
	// 「不確定有多少筆」比編一個數字誠實，呼叫端會因此顯示「共 0 筆」而不是
	// 假裝資料完整。
	var totalObject struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal(decoded.Hits.Total, &totalObject); err == nil {
		result.Total = totalObject.Value
	} else if value, convErr := strconv.Atoi(strings.TrimSpace(string(decoded.Hits.Total))); convErr == nil {
		result.Total = value
	}
	for _, hit := range decoded.Hits.Hits {
		id := hit.Source.ID
		if id == 0 {
			// 兜底：_source 缺 id 時用 _id。正常情況兩者必然相同，
			// 但索引若由其他工具寫入（只有 _id 沒有 id 欄位）仍要能運作。
			if parsed, convErr := strconv.ParseInt(hit.ID, 10, 64); convErr == nil {
				id = parsed
			}
		}
		result.Hits = append(result.Hits, Hit{ID: id, Score: hit.Score})
	}
	return result, nil
}
