/*
收藏與圖片刪除的路由關卡（httpapi/forum_bookmark_test.go）。

這一批端點有兩條「不會有任何錯誤訊息」的性質，測試集中在它們：

  1. 路由分派
     收藏的切換掛在 /api/forum/posts/{id}/bookmark 這個**尾綴**上。尾綴分派
     寫錯的症狀不是 500，而是請求掉進「單篇文章」分支：GET 會回一篇貼文、
     POST 會回 404 —— 兩種都不像路由壞掉，看起來像前端送錯。因此這裡直接
     餵請求給 handleForumPostAction，釘住「/bookmark 不會被別的尾綴吃掉」。

  2. 授權邊界
     收藏清單是**私有**資料（誰收藏了什麼不公開），未登入必須 401；而切換
     與刪圖都是寫入，未登入同樣必須 401。少了這些檢查的症狀是「匿名請求
     悄悄寫入某一列」或「清單回一份空陣列」—— 前者是資料汙染，後者是
     「我的收藏不見了」的客服問題，兩者都不會出現在日誌裡。

刻意沒有測的：交易內的 INSERT／DELETE 與檔案伺服器的刪除呼叫。這個專案沒有
MySQL 替身（見 docs/KNOWN_ISSUES.md 的已知問題），而那兩段需要真的資料庫與
檔案伺服器；它們的形狀與按讚、追蹤完全相同（同一個 BeginTx + defer Rollback）。
*/

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

/* ==========================================================================
   路由分派
   ========================================================================== */

// GET 到 /api/forum/posts/1/bookmark 必須被交給收藏處理器，並在那裡因方法
// 不對而 405。
//
// 若尾綴分派漏了 /bookmark，這一條會落到單篇文章的 GET 分支，得到 200 與
// 一篇貼文 —— 而這就是「看起來正常、其實打錯地方」的失敗形狀。
func TestForumPostActionDispatchesBookmarkSuffix(t *testing.T) {
	server := newAnnouncementTestServer(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/forum/posts/1/bookmark", nil)
	server.handleForumPostAction(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /api/forum/posts/1/bookmark 回應 = %d, want 405（分派到收藏處理器）", recorder.Code)
	}
}

/* ==========================================================================
   授權邊界
   ========================================================================== */

// 未登入切換收藏 → 401（而不是寫入一筆沒有主人的收藏）。
func TestForumPostBookmarkRequiresLogin(t *testing.T) {
	server := newAnnouncementTestServer(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/forum/posts/1/bookmark", nil)
	server.handleForumPostBookmark(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("未登入的回應 = %d, want 401", recorder.Code)
	}
}

// 收藏清單只接受 GET：POST 若被放行，它會是一個沒有語意的「清單寫入」。
func TestForumBookmarksRejectsNonGet(t *testing.T) {
	server := newAnnouncementTestServer(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/api/forum/bookmarks", nil)
		server.handleForumBookmarks(recorder, request)
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/forum/bookmarks 回應 = %d, want 405", method, recorder.Code)
		}
	}
}

// 未登入讀收藏清單 → 401（而不是空清單：空清單會讓前端顯示「你還沒有收藏」
// 而誤導使用者以為資料不見了）。
func TestForumBookmarksRequiresLogin(t *testing.T) {
	server := newAnnouncementTestServer(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/forum/bookmarks", nil)
	server.handleForumBookmarks(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("未登入的回應 = %d, want 401", recorder.Code)
	}
}

// 刪除圖片是唯讀以外的動作，只接受 POST。
func TestForumImageDeleteRejectsNonPost(t *testing.T) {
	server := newAnnouncementTestServer(t)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/api/forum/images/delete", nil)
		server.handleForumImageDelete(recorder, request)
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/forum/images/delete 回應 = %d, want 405", method, recorder.Code)
		}
	}
}

func TestForumImageDeleteRequiresLogin(t *testing.T) {
	server := newAnnouncementTestServer(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/forum/images/delete", nil)
	server.handleForumImageDelete(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("未登入的回應 = %d, want 401", recorder.Code)
	}
}
