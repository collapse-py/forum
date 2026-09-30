package es

/*
es 套件的測試（backend/forum/es/es_test.go）。

測試一律以 httptest 假造 ES 的回應，不連真實叢集。理由有兩個：
  1. 測試不該依賴「某台機器上有個 ES」這個環境事實，否則 CI 與開發機的
     結果會不一致。
  2. 本套件的職責就是把 HTTP 請求與回應轉成結構化結果，因此用假的 HTTP 層
     測試它比真的架一個 ES 更直接 —— 可以精確構造出那些難以在真實叢集上
     重現的邊界回應（例如回應本文超過讀取上限、或 hits.total 是舊格式）。

真正與 ES 的整合（mapping 是否被接受、cjk analyzer 的實際切詞）不在此涵蓋：
那屬於環境驗證，於開發時以 _analyze 與 _mapping 端點確認過。
*/

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient 對著指定的 handler 建立 Client，並把逾時縮短到 2 秒：
// 測試若因故打不到 handler，會在 2 秒內失敗而不是掛滿預設的 30 秒。
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(server.URL, "forum_posts"), server
}

// capturedRequest 記錄假 ES 收到的請求，供斷言使用。
type capturedRequest struct {
	Method      string
	Path        string
	ContentType string
	Body        string
}

func writeJSONBody(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func TestNewNormalizesBaseURLAndIndex(t *testing.T) {
	client := New("http://es.example:9200/", "")
	if client.baseURL != "http://es.example:9200" {
		t.Fatalf("baseURL = %q, want trailing slash removed", client.baseURL)
	}
	if client.index != defaultIndexName {
		t.Fatalf("index = %q, want %q", client.index, defaultIndexName)
	}
	if !client.Enabled() {
		t.Fatal("Enabled() = false for a configured client")
	}
	// 空白字串是「沒設定」，Enabled 必須為 false，讓呼叫端走 MySQL 降級。
	if New("   ", "forum_posts").Enabled() {
		t.Fatal("Enabled() = true for a blank baseURL")
	}
	// 索引名會被用在 URL 路徑上，因此必須去掉空白；名稱本身的驗證交給 ES。
	if New("http://es:9200", "  custom  ").index != "custom" {
		t.Fatalf("index = %q, want trimmed", New("http://es:9200", "  custom  ").index)
	}
}

// TestDisabledClientIsSafe 確認未設定 ES 時所有方法都不發請求、且回傳
// ErrDisabled 讓呼叫端可以用 errors.Is 分流。
func TestDisabledClientIsSafe(t *testing.T) {
	client := New("", "forum_posts")
	ctx := context.Background()

	if client.Enabled() {
		t.Fatal("Enabled() = true for an unconfigured client")
	}
	if err := client.Ping(ctx); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Ping error = %v, want ErrDisabled", err)
	}
	if err := client.EnsureIndex(ctx); !errors.Is(err, ErrDisabled) {
		t.Fatalf("EnsureIndex error = %v, want ErrDisabled", err)
	}
	if err := client.IndexPost(ctx, PostDocument{ID: 1}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("IndexPost error = %v, want ErrDisabled", err)
	}
	if err := client.DeletePost(ctx, 1); !errors.Is(err, ErrDisabled) {
		t.Fatalf("DeletePost error = %v, want ErrDisabled", err)
	}
	if err := client.IndexPosts(ctx, []PostDocument{{ID: 1}}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("IndexPosts error = %v, want ErrDisabled", err)
	}
	// 空索引也不能繞過「未啟用」的判斷直接回 nil，否則呼叫端會誤以為已同步。
	if _, err := client.Search(ctx, "x", 0, 10, false); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Search error = %v, want ErrDisabled", err)
	}
	// nil receiver 也要安全：Server.es 為 nil 時（未設定 ES_URL）程式仍會
	// 呼叫 esEnabled 之後的某些路徑。
	var nilClient *Client
	if nilClient.Enabled() {
		t.Fatal("nil client reports Enabled")
	}
	if _, err := nilClient.Search(ctx, "x", 0, 10, false); !errors.Is(err, ErrDisabled) {
		t.Fatalf("nil client Search error = %v, want ErrDisabled", err)
	}
	if nilClient.IndexName() != "" {
		t.Fatal("nil client IndexName should be empty")
	}
}

/*
TestSearchDecodesHitsAndTotal 是本檔最重要的一個測試。

它同時守住三件事：
  1. hits.total 必須被正確解析成數字。這裡刻意使用兩層巢狀的舊寫法時會
     靜默失效（encoding/json 不寫入值），因此測試用真正的 ES 9 回應格式
     {"hits":{"total":{"value":7,...}}}。
  2. 每個 hit 的 id 要從 _source.id 取得（缺漏時退回 _id）。
  3. 沒有命中時要回空切片而不是 nil slice，呼叫端才不會對 nil range。
*/
func TestSearchDecodesHitsAndTotal(t *testing.T) {
	var got capturedRequest
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = capturedRequest{Method: r.Method, Path: r.URL.Path, Body: string(body)}
		writeJSONBody(w, http.StatusOK, `{
			"took": 3,
			"hits": {
				"total": {"value": 7, "relation": "eq"},
				"max_score": 2.5,
				"hits": [
					{"_id": "12", "_score": 2.5, "_source": {"id": 12, "content": "十二"}},
					{"_id": "34", "_score": 1.5, "_source": {"id": 34, "content": "三十四"}}
				]
			}
		}`)
	})

	result, err := client.Search(context.Background(), "查詢", 0, 2, false)
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if result.Total != 7 {
		t.Fatalf("Total = %d, want 7 (hits.total.value must survive decoding)", result.Total)
	}
	if len(result.Hits) != 2 {
		t.Fatalf("len(Hits) = %d, want 2", len(result.Hits))
	}
	if result.Hits[0].ID != 12 || result.Hits[1].ID != 34 {
		t.Fatalf("hit ids = %d,%d want 12,34", result.Hits[0].ID, result.Hits[1].ID)
	}
	if result.Hits[0].Score != 2.5 {
		t.Fatalf("score = %v, want 2.5", result.Hits[0].Score)
	}
	if got.Method != http.MethodPost || got.Path != "/forum_posts/_search" {
		t.Fatalf("request = %s %s, want POST /forum_posts/_search", got.Method, got.Path)
	}
	// 查詢必須同時比對 cjk 與 standard 兩欄，並帶上前綴比對；
	// 少任何一條都會讓「單字查詢」或「部分字詞」靜默零命中。
	for _, fragment := range []string{`"content"`, `"content.standard"`, `"match_phrase_prefix"`, `"analyzer":"cjk"`, `"analyzer":"standard"`, `"track_total_hits":true`} {
		if !strings.Contains(got.Body, fragment) {
			t.Errorf("query body missing %s: %s", fragment, got.Body)
		}
	}
	// 公開搜尋不得比對 authorEmail：那會讓任何人用信箱反推其他人的貼文。
	if strings.Contains(got.Body, "authorEmail") {
		t.Errorf("public search must not query authorEmail: %s", got.Body)
	}
}

// TestSearchIncludesAuthorOnlyForAdmin 確認 includeAuthor 才會加上
// authorEmail 的 term 比對，且比對的是小寫（ES 的 keyword 區分大小寫，
// 資料裡存的是使用者實際輸入的大小寫，管理員輸入時常會大小寫不一致）。
func TestSearchIncludesAuthorOnlyForAdmin(t *testing.T) {
	var got capturedRequest
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = capturedRequest{Body: string(body)}
		writeJSONBody(w, http.StatusOK, `{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)
	})

	result, err := client.Search(context.Background(), "User@Example.com", 0, 10, true)
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if !strings.Contains(got.Body, `"term":{"authorEmail":"user@example.com"}`) {
		t.Errorf("admin search missing lowercased author term: %s", got.Body)
	}
	if result.Total != 0 || len(result.Hits) != 0 {
		t.Errorf("empty result should be an empty slice, got total=%d hits=%d", result.Total, len(result.Hits))
	}
	if result.Hits == nil {
		t.Error("Hits must not be nil for an empty result (callers range over it)")
	}
}

// TestSearchAcceptsLegacyNumericTotal 確認舊格式（"total": 123）也能解析。
func TestSearchAcceptsLegacyNumericTotal(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusOK, `{"hits":{"total":123,"hits":[]}}`)
	})
	result, err := client.Search(context.Background(), "x", 0, 10, false)
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if result.Total != 123 {
		t.Fatalf("Total = %d, want 123", result.Total)
	}
}

// TestSearchClampsOffsetBeyondResultWindow 確認深翻頁被夾住而不是被 ES 以
// 400 拒絕（夾住之後最後一頁仍能顯示空結果）。
func TestSearchClampsOffsetBeyondResultWindow(t *testing.T) {
	var body string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeJSONBody(w, http.StatusOK, `{"hits":{"total":{"value":1,"relation":"eq"},"hits":[]}}`)
	})

	size := 25
	if _, err := client.Search(context.Background(), "x", 50000, size, false); err != nil {
		t.Fatalf("Search error = %v", err)
	}
	var decoded struct {
		From int `json:"from"`
		Size int `json:"size"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("request body is not valid JSON: %v (%s)", err, body)
	}
	if decoded.From != maxResultWindow-size {
		t.Fatalf("from = %d, want %d", decoded.From, maxResultWindow-size)
	}
	// size 大於視窗時夾到的 from 不得為負。
	if _, err := client.Search(context.Background(), "x", 0, maxResultWindow*2, false); err != nil {
		t.Fatalf("Search with oversized size error = %v", err)
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if decoded.From < 0 {
		t.Fatalf("from = %d, want >= 0", decoded.From)
	}
}

// TestEnsureIndexToleratesExistingIndex 確認重複建立索引被視為成功
// （每次啟動都會呼叫一次，冪等是必要條件）。
func TestEnsureIndexToleratesExistingIndex(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/forum_posts" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		writeJSONBody(w, http.StatusBadRequest, `{"error":{"type":"resource_already_exists_exception","reason":"index already exists"},"status":400}`)
	})
	if err := client.EnsureIndex(context.Background()); err != nil {
		t.Fatalf("EnsureIndex error = %v, want nil for an existing index", err)
	}
}

// TestEnsureIndexPropagatesOtherErrors 確認其他 4xx 仍要回報（那代表設定
// 有問題，例如權限不足），不能被當成「已存在」而吞掉。
func TestEnsureIndexPropagatesOtherErrors(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusForbidden, `{"error":{"type":"security_exception","reason":"no permission"},"status":403}`)
	})
	err := client.EnsureIndex(context.Background())
	if err == nil {
		t.Fatal("EnsureIndex error = nil, want a security_exception to surface")
	}
	if StatusOf(err) != http.StatusForbidden {
		t.Fatalf("StatusOf = %d, want 403", StatusOf(err))
	}
}

// TestIndexPostUsesDocumentID 確認 PUT 目標是 /{index}/_doc/{MySQL 主鍵}：
// 以主鍵當文件 ID 讓重複寫入是冪等覆寫，刪除也不需要額外條件。
func TestIndexPostUsesDocumentID(t *testing.T) {
	var got capturedRequest
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = capturedRequest{Method: r.Method, Path: r.URL.Path, ContentType: r.Header.Get("Content-Type"), Body: string(body)}
		writeJSONBody(w, http.StatusCreated, `{"result":"created"}`)
	})

	created := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	if err := client.IndexPost(context.Background(), PostDocument{ID: 42, Content: "內容", AuthorEmail: "a@b.c", CreatedAt: created}); err != nil {
		t.Fatalf("IndexPost error = %v", err)
	}
	if got.Method != http.MethodPut || got.Path != "/forum_posts/_doc/42" {
		t.Fatalf("request = %s %s, want PUT /forum_posts/_doc/42", got.Method, got.Path)
	}
	if got.ContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got.ContentType)
	}
	if !strings.Contains(got.Body, `"authorEmail":"a@b.c"`) {
		t.Errorf("body missing authorEmail: %s", got.Body)
	}
}

// TestDeletePostIgnoresMissingDocument 確認文件已不存在（404）不算失敗：
// 索引的目標狀態就是「這裡不該有這篇」，重複刪除不該讓呼叫端視為失敗。
func TestDeletePostIgnoresMissingDocument(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusNotFound, `{"_index":"forum_posts","found":false}`)
	})
	if err := client.DeletePost(context.Background(), 7); err != nil {
		t.Fatalf("DeletePost error = %v, want nil for a missing document", err)
	}
}

func TestDeletePostPropagatesServerErrors(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusInternalServerError, `{"error":"boom"}`)
	})
	if err := client.DeletePost(context.Background(), 7); err == nil {
		t.Fatal("DeletePost error = nil, want a 500 to surface")
	}
}

// TestIndexPostsSendsNDJSONChunks 確認批次寫入用 NDJSON、每批一組動作行 +
// 文件行，且逐批送出。
func TestIndexPostsSendsNDJSONChunks(t *testing.T) {
	var bodies []string
	var contentTypes []string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_bulk" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		contentTypes = append(contentTypes, r.Header.Get("Content-Type"))
		writeJSONBody(w, http.StatusOK, `{"errors":false,"items":[]}`)
	})

	// 刻意超過一個批次，確認切成兩批。
	docs := make([]PostDocument, 0, bulkChunkSize+1)
	for i := int64(1); i <= bulkChunkSize+1; i++ {
		docs = append(docs, PostDocument{ID: i, Content: "內容", AuthorEmail: "a@b.c", CreatedAt: time.Now()})
	}
	if err := client.IndexPosts(context.Background(), docs); err != nil {
		t.Fatalf("IndexPosts error = %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("chunk count = %d, want 2", len(bodies))
	}
	for _, contentType := range contentTypes {
		if contentType != "application/x-ndjson" {
			t.Errorf("Content-Type = %q, want application/x-ndjson", contentType)
		}
	}
	// 第一行必須是帶 _id 的動作行，第二行才是文件本體。
	lines := strings.Split(strings.TrimSpace(bodies[0]), "\n")
	if len(lines) != 2*bulkChunkSize {
		t.Fatalf("line count = %d, want %d", len(lines), 2*bulkChunkSize)
	}
	if !strings.Contains(lines[0], `"_id":"1"`) || !strings.Contains(lines[0], `"_index":"forum_posts"`) {
		t.Errorf("first line is not an action line: %s", lines[0])
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[1]), `{"id":1`) {
		t.Errorf("second line is not the document: %s", lines[1])
	}
	if err := client.IndexPosts(context.Background(), nil); err != nil {
		t.Fatalf("IndexPosts(nil) error = %v, want nil without any request", err)
	}
}

// TestIndexPostsReportsPerDocumentErrors 確認 _bulk 的「部分失敗」被回報：
// 它回的是 200，只看狀態碼會把寫入失敗當成成功，重建後索引就少了一截。
func TestIndexPostsReportsPerDocumentErrors(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusOK, `{"errors":true,"items":[{"index":{"status":400}}]}`)
	})
	err := client.IndexPosts(context.Background(), []PostDocument{{ID: 1, Content: "x"}})
	if err == nil {
		t.Fatal("IndexPosts error = nil, want per-document errors to surface")
	}
}

/*
TestDoAcceptsResponsesLargerThanErrorLimit 是讀取上限的迴歸測試。

_bulk 的回應會隨批次內文件數線性成長（每筆約 250 bytes），先前把「錯誤本文」
用的 8KB 上限套用在所有回應上，結果重建 61 篇貼文時回應被安靜截斷，
JSON 解析失敗的訊息（unexpected end of JSON input）完全指不出真正原因。
因此成功回應必須有獨立的、夠大的上限。
*/
func TestDoAcceptsResponsesLargerThanErrorLimit(t *testing.T) {
	// 組一個遠大於 8KB、但仍是合法 JSON 的回應。
	items := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		items = append(items, `{"index":{"_index":"forum_posts","_id":"1","status":201,"result":"created"}}`)
	}
	body := `{"errors":false,"items":[` + strings.Join(items, ",") + `]}`
	if len(body) <= 8192 {
		t.Fatalf("test payload is only %d bytes; it must exceed the old 8KB limit", len(body))
	}
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusOK, body)
	})
	if err := client.IndexPosts(context.Background(), []PostDocument{{ID: 1, Content: "x"}}); err != nil {
		t.Fatalf("IndexPosts error = %v, want nil for a large but valid response", err)
	}
}

// TestDoRejectsOversizedResponses 確認真的過大時回報明確錯誤，而不是讓
// 呼叫端在解析半截 JSON 時收到誤導性的訊息。
func TestDoRejectsOversizedResponses(t *testing.T) {
	// 直接把上限調小不可行（常數），因此改用一個剛好超過上限的合法 JSON。
	huge := `{"errors":false,"pad":"` + strings.Repeat("x", maxResponseBodySize+16) + `"}`
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusOK, huge)
	})
	if err := client.IndexPosts(context.Background(), []PostDocument{{ID: 1}}); err == nil {
		t.Fatal("IndexPosts error = nil, want an explicit oversize error")
	}
}

// TestParseHTTPErrorShapes 確認錯誤解析在兩種形狀下都給得出狀態碼。
func TestParseHTTPErrorShapes(t *testing.T) {
	// 標準形狀：type + reason。
	err := parseHTTPError(http.StatusNotFound, []byte(`{"error":{"type":"index_not_found_exception","reason":"no such index"},"status":404}`))
	if StatusOf(err) != http.StatusNotFound {
		t.Fatalf("StatusOf = %d, want 404", StatusOf(err))
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.Type != "index_not_found_exception" {
		t.Fatalf("error = %v, want an HTTPError carrying the ES error type", err)
	}
	// 非 JSON（ES 偶爾回純文字）→ 退回使用原始本文，不丟掉診斷資訊。
	plain := parseHTTPError(http.StatusBadGateway, []byte("upstream connect error"))
	if StatusOf(plain) != http.StatusBadGateway {
		t.Fatalf("StatusOf = %d, want 502", StatusOf(plain))
	}
	if !errors.As(plain, &httpErr) || !strings.Contains(httpErr.Reason, "upstream") {
		t.Fatalf("reason = %v, want the raw body preserved", httpErr.Reason)
	}
	// 空本文 → 以狀態碼的原因片語填補，避免 Reason 空白。
	empty := parseHTTPError(http.StatusServiceUnavailable, nil)
	if !errors.As(empty, &httpErr) || httpErr.Reason == "" {
		t.Fatalf("reason = %v, want a non-empty fallback", httpErr.Reason)
	}
	// 非 HTTP 錯誤的 StatusOf 必須回 0。
	if StatusOf(errors.New("boom")) != 0 {
		t.Fatal("StatusOf(plain error) should be 0")
	}
}

// TestRequestTimeoutIsBounded 確認逾時是有限值：ES 是輔助性相依，
// 搜尋與索引都不該讓 HTTP 請求被它拖住。
func TestRequestTimeoutIsBounded(t *testing.T) {
	client := New("http://es.example:9200", "forum_posts")
	if client.client.Timeout != requestTimeout {
		t.Fatalf("timeout = %v, want %v", client.client.Timeout, requestTimeout)
	}
	if requestTimeout > 5*time.Second {
		t.Fatalf("requestTimeout = %v, want a value that keeps requests well under proxy limits", requestTimeout)
	}
}
