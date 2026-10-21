/*
ES 傳輸層的失敗分支測試（backend/forum/es/es_coverage_test.go）。

es_test.go 覆蓋正常路徑（索引建立、寫入、搜尋、刪除、驗證標頭），這個檔補
「ES 那邊壞掉時呼叫端拿到什麼」：

 1. HTTPError 的兩種文案（有 type / 無 type）。無 type 那條是 ES 在某些錯誤
    （例如連線層）回的形狀，漏掉它會讓錯誤訊息變成 "http status 0: "。
 2. nil client（＝設定檔沒填 ES_URL）的每一個方法：全部回 ErrDisabled，
    讓搜尋整條降級到 MySQL LIKE，而不是 panic。
 3. _bulk 的「部分失敗」：ES 對它回 200 + errors:true —— 只看狀態碼會把
    寫入失敗的文件當成成功，重建後的索引因此少了一截資料。
 4. ListAllIDs 的三種壞回應：HTTP 錯誤、壞 JSON、_id 不是數字（重建索引時
    混進非數字 _id 的唯一訊號就在這裡）。以及 search_after 的分頁迴圈：
    滿頁時必須翻下一頁，否則對帳只看得到前 200 筆。
*/

package es

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

/* ==========================================================================
   HTTPError 文案
   ========================================================================== */

func TestHTTPErrorMessageCoversBothShapes(t *testing.T) {
	withType := (&HTTPError{Status: 400, Type: "parsing_exception", Reason: "bad query"}).Error()
	if withType != "es: http status 400 (parsing_exception): bad query" {
		t.Errorf("有 type 的文案 = %q", withType)
	}
	withoutType := (&HTTPError{Status: 502, Reason: "bad gateway"}).Error()
	if withoutType != "es: http status 502: bad gateway" {
		t.Errorf("無 type 的文案 = %q（ES 某些錯誤不帶 type）", withoutType)
	}
	if StatusOf(fmt.Errorf("plain error")) != 0 {
		t.Error("非 HTTP 錯誤的 StatusOf 應為 0")
	}
}

/* ==========================================================================
   nil client：全部降級
   ========================================================================== */

// 設定檔沒填 ES_URL 時 NewServer 会让 s.es 保持 nil —— 每個方法都必須優雅
// 地回 ErrDisabled，讓搜尋整條走 MySQL LIKE。
func TestNilClientEveryMethodDegrades(t *testing.T) {
	var c *Client
	ctx := context.Background()

	if c.Enabled() {
		t.Error("nil client 的 Enabled 應為 false")
	}
	if got := c.AuthKind(); got != AuthKindNone {
		t.Errorf("nil client 的 AuthKind = %q, want %q", got, AuthKindNone)
	}
	if got := c.IndexName(); got != "" {
		t.Errorf("nil client 的 IndexName = %q, want 空", got)
	}
	if err := c.Ping(ctx); !errors.Is(err, ErrDisabled) {
		t.Errorf("Ping = %v, want ErrDisabled", err)
	}
	if err := c.EnsureIndex(ctx); !errors.Is(err, ErrDisabled) {
		t.Errorf("EnsureIndex = %v, want ErrDisabled", err)
	}
	if err := c.IndexPost(ctx, PostDocument{ID: 1}); !errors.Is(err, ErrDisabled) {
		t.Errorf("IndexPost = %v, want ErrDisabled", err)
	}
	if err := c.DeletePost(ctx, 1); !errors.Is(err, ErrDisabled) {
		t.Errorf("DeletePost = %v, want ErrDisabled", err)
	}
	if _, err := c.ListAllIDs(ctx); !errors.Is(err, ErrDisabled) {
		t.Errorf("ListAllIDs = %v, want ErrDisabled", err)
	}
	if _, err := c.Search(ctx, "關鍵字", 0, 10, false); !errors.Is(err, ErrDisabled) {
		t.Errorf("Search = %v, want ErrDisabled", err)
	}
	// 空文件清單的批次寫入：不打任何網路，直接成功。
	if err := c.IndexPosts(ctx, nil); err != nil {
		t.Errorf("IndexPosts(空) = %v, want nil", err)
	}
}

/* ==========================================================================
   HTTP 錯誤與 bulk 部分失敗
   ========================================================================== */

func TestIndexPostSurfacesHTTPError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONBody(w, http.StatusInternalServerError, `{"error":{"type":"server_error","reason":"disk full"}}`)
	})

	err := client.IndexPost(context.Background(), PostDocument{ID: 9, Content: "x"})
	if err == nil {
		t.Fatal("500 回應的 IndexPost 應回錯誤")
	}
	if got := StatusOf(err); got != http.StatusInternalServerError {
		t.Errorf("StatusOf = %d, want 500", got)
	}
}

// _bulk 的部分失敗是 200 + errors:true：只看狀態碼會把寫入失敗當成成功。
func TestIndexPostsRejectsPartialFailure(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/_bulk") {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeJSONBody(w, http.StatusOK, `{"took":5,"errors":true,"items":[{"index":{"_id":"1","status":500,"error":{"type":"server_error","reason":"rejected"}}}]}`)
	})

	err := client.IndexPosts(context.Background(), []PostDocument{{ID: 1, Content: "x"}})
	if err == nil {
		t.Fatal("_bulk errors:true 必須回錯誤（部分失敗不是成功）")
	}
	if !strings.Contains(err.Error(), "bulk") {
		t.Errorf("錯誤應指出來自 _bulk 的部分失敗: %v", err)
	}
}

/* ==========================================================================
   ListAllIDs 的壞回應與分頁
   ========================================================================== */

func TestListAllIDsRejectsBadResponses(t *testing.T) {
	t.Run("HTTP 錯誤", func(t *testing.T) {
		client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSONBody(w, http.StatusServiceUnavailable, `{"error":{"reason":"shard failed"}}`)
		})
		if _, err := client.ListAllIDs(context.Background()); err == nil {
			t.Fatal("503 的 ListAllIDs 應回錯誤")
		}
	})

	t.Run("壞 JSON", func(t *testing.T) {
		client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSONBody(w, http.StatusOK, `{not json`)
		})
		if _, err := client.ListAllIDs(context.Background()); err == nil {
			t.Fatal("壞 JSON 的 ListAllIDs 應回錯誤")
		}
	})

	t.Run("_id 不是數字", func(t *testing.T) {
		client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSONBody(w, http.StatusOK, `{"hits":{"hits":[{"_id":"not-a-number"}]}}`)
		})
		_, err := client.ListAllIDs(context.Background())
		if err == nil || !strings.Contains(err.Error(), "不是數字") {
			t.Fatalf("非數字 _id 應回明確錯誤, 得到 %v", err)
		}
	})
}

// 滿一頁（bulkChunkSize 筆）時必須用 search_after 翻下一頁 —— 對帳只看得到
// 前 200 筆的話，坐在後面的幽靈文件永遠不會被找到。
func TestListAllIDsPaginatesWithSearchAfter(t *testing.T) {
	pages := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		body, _ := io.ReadAll(r.Body)
		var b strings.Builder
		b.WriteString(`{"hits":{"hits":[`)
		if strings.Contains(string(body), "search_after") {
			// 第二頁：只回一筆，結束分頁。
			b.WriteString(`{"_id":"201"}`)
		} else {
			for i := 0; i < bulkChunkSize; i++ {
				if i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, `{"_id":"%d"}`, i)
			}
		}
		b.WriteString(`]}}`)
		writeJSONBody(w, http.StatusOK, b.String())
	})

	ids, err := client.ListAllIDs(context.Background())
	if err != nil {
		t.Fatalf("ListAllIDs: %v", err)
	}
	if pages != 2 {
		t.Errorf("請求次數 = %d, want 2（滿頁後應翻頁）", pages)
	}
	if len(ids) != bulkChunkSize+1 {
		t.Errorf("ID 數量 = %d, want %d", len(ids), bulkChunkSize+1)
	}
	if ids[0] != 0 || ids[len(ids)-1] != 201 {
		t.Errorf("ID 內容不對: 首=%d 尾=%d", ids[0], ids[len(ids)-1])
	}
}

/* ==========================================================================
   Search 的失敗分支
   ========================================================================== */

func TestSearchSurfacesHTTPAndDecodeErrors(t *testing.T) {
	t.Run("HTTP 錯誤", func(t *testing.T) {
		client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSONBody(w, http.StatusBadRequest, `{"error":{"type":"parsing_exception","reason":"bad"}}`)
		})
		_, err := client.Search(context.Background(), "x", 0, 10, false)
		if err == nil {
			t.Fatal("400 的 Search 應回錯誤")
		}
		if got := StatusOf(err); got != http.StatusBadRequest {
			t.Errorf("StatusOf = %d, want 400", got)
		}
	})

	t.Run("壞 JSON", func(t *testing.T) {
		client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSONBody(w, http.StatusOK, `<html>not json</html>`)
		})
		if _, err := client.Search(context.Background(), "x", 0, 10, false); err == nil {
			t.Fatal("壞 JSON 的 Search 應回錯誤")
		}
	})
}

// 負 from 與零 size 必須被收斂：ES 對 from<0 與 size=0 各自回 400，
// 而「翻太深」（from+size > max_result_window）也要夾住而不是讓 ES 拒絕。
func TestSearchClampsPagingArguments(t *testing.T) {
	var gotBody string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		writeJSONBody(w, http.StatusOK, `{"hits":{"total":0,"hits":[]}}`)
	})

	if _, err := client.Search(context.Background(), "x", -5, 0, false); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !strings.Contains(gotBody, `"from":0`) {
		t.Errorf("負 from 未收斂成 0: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"size":1`) {
		t.Errorf("size=0 未收斂成 1: %s", gotBody)
	}

	if _, err := client.Search(context.Background(), "x", maxResultWindow+500, 10, false); err != nil {
		t.Fatalf("深分頁 Search: %v", err)
	}
	if !strings.Contains(gotBody, fmt.Sprintf(`"from":%d`, maxResultWindow-10)) {
		t.Errorf("超過 max_result_window 的 from 未夾住: %s", gotBody)
	}
}
