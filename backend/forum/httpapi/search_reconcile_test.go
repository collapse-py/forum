/*
索引對帳的測試（backend/forum/httpapi/search_reconcile_test.go）。

ReconcileSearchIndex 的失效模式是「幽靈文件留在索引裡」：unindexForumPost
是 best-effort，ES 短暫不可用時貼文刪除成功而索引留下已不存在的文件。對帳
要根除的正是那些「索引裡有、MySQL 裡已不存在」的文件。

需要真實 MySQL（FORUM_TEST_MYSQL_DSN，與 comment_reply_mysql_test.go 同一
個約定）：對帳的正確性是「DELETE 之後 ES 裡看不到」，而 ES 端用 httptest
假造——es 套件本身就是 HTTP 傳輸層，假 ES 對它而言就是全部（見 es_test.go
的說明）。MySQL 端不能假造，因為「哪些貼文可見」是對帳的另一半。
*/
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"forum/forum/es"
	"forum/forum/idgen"
)

// fakeES 是一份記憶體中的「索引」：支援 _search（match_all + search_after
// 分頁）與 _doc/{id} 的 DELETE。它刻意只實作對帳用到的端點，其他請求回 400
// —— 對帳程式若打到别的端點，測試會直接失敗而不是靜默通過。
type fakeES struct {
	docs map[int64]bool
}

func (f *fakeES) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/forum_posts")
	switch {
	case r.Method == http.MethodPost && path == "/_search":
		var q struct {
			Size        int     `json:"size"`
			SearchAfter []int64 `json:"search_after"`
		}
		// 只接受陣列（與真實 ES 同形）：純數字在線上是 400 parsing_exception。
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Size <= 0 {
			writeRawJSON(w, http.StatusBadRequest, `{"error":{"type":"bad_request"}}`)
			return
		}
		after := int64(0)
		if len(q.SearchAfter) > 0 {
			after = q.SearchAfter[0]
		}
		ids := make([]int64, 0, q.Size)
		for id := range f.docs {
			if id > after {
				ids = append(ids, id)
			}
		}
		sortInt64Asc(ids)
		if len(ids) > q.Size {
			ids = ids[:q.Size]
		}
		hits := make([]string, 0, len(ids))
		for _, id := range ids {
			hits = append(hits, `{"_id":"`+strconv.FormatInt(id, 10)+`"}`)
		}
		writeRawJSON(w, http.StatusOK, `{"hits":{"hits":[`+strings.Join(hits, ",")+`]}}`)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/_doc/"):
		id, err := strconv.ParseInt(path[len("/_doc/"):], 10, 64)
		if err != nil {
			writeRawJSON(w, http.StatusBadRequest, `{"error":{"type":"bad_request"}}`)
			return
		}
		if !f.docs[id] {
			writeRawJSON(w, http.StatusNotFound, `{"error":{"type":"not_found"}}`)
			return
		}
		delete(f.docs, id)
		writeRawJSON(w, http.StatusOK, `{"result":"deleted"}`)
	default:
		writeRawJSON(w, http.StatusBadRequest, `{"error":{"type":"bad_request","reason":"unexpected endpoint "+r.Method+" "+r.URL.Path}}`)
	}
}

func writeRawJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func sortInt64Asc(ids []int64) {
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j] < ids[j-1]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
}

func TestReconcileSearchIndexDisabled(t *testing.T) {
	srv := &Server{}
	if _, err := srv.ReconcileSearchIndex(context.Background()); !errors.Is(err, es.ErrDisabled) {
		t.Fatalf("未設定 ES 時對帳應回 ErrDisabled，得到 %v", err)
	}
}

func TestReconcileSearchIndexMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)

	now := time.Now()
	// id 由應用層給（schema 沒有 AUTO_INCREMENT），後面的斷言必須用同一個值。
	liveID, deletedID, ghostID := idgen.Next(), idgen.Next(), idgen.Next()
	if _, err := db.Exec(`INSERT INTO forum_posts (id, author_email, content, image_url, created_at) VALUES (?, 'a@example.com', '活的文章', '', ?)`, liveID, now); err != nil {
		t.Fatalf("插入活貼文失敗: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO forum_posts (id, author_email, content, image_url, created_at, deleted_at) VALUES (?, 'b@example.com', '已軟刪的文章', '', ?, ?)`, deletedID, now.Add(-time.Hour), now); err != nil {
		t.Fatalf("插入軟刪貼文失敗: %v", err)
	}

	// 索引裡刻意多留兩筆：已軟刪的貼文與一筆根本不存在的 id（正是
	// unindexForumPost 失敗留下的幽靈形狀）。
	fake := &fakeES{docs: map[int64]bool{liveID: true, deletedID: true, ghostID: true}}
	esServer := httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(esServer.Close)

	srv := &Server{db: db, es: es.New(esServer.URL, "forum_posts", es.Auth{})}

	removed, err := srv.ReconcileSearchIndex(context.Background())
	if err != nil {
		t.Fatalf("對帳失敗: %v", err)
	}
	if removed != 2 {
		t.Fatalf("應移除 2 筆幽靈，實際 %d", removed)
	}
	if len(fake.docs) != 1 || !fake.docs[liveID] {
		t.Fatalf("索引應只剩活貼文 %d，實際 %v", liveID, fake.docs)
	}

	// 冪等：再跑一次沒有東西可刪。
	again, err := srv.ReconcileSearchIndex(context.Background())
	if err != nil || again != 0 {
		t.Fatalf("第二次對帳應回 (0, nil)，得到 (%d, %v)", again, err)
	}
}
