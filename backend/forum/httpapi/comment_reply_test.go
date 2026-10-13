/*
留言的層級：父留言、層級上限，以及對前端的承諾（comment_reply_test.go）。

這個功能有四件事會在壞掉時**不出錯**，因此每一件都需要被釘死：

  1. 層級上限（forumCommentMaxDepth = 3）
       由 forumParentDepthAllowsReply 的表格測試把守。上限失效時的症狀不是
       錯誤，是討論串安靜地長到第 4 層 —— 前端那時已經不再渲染回覆鈕，於是那一則
       沒有任何辦法被再回覆，也沒有任何訊息解釋為什麼。

  2. parentId 的輸入邊界
       負數必須在碰到資料庫之前就被擋掉。它是一個「查得到」的形狀（負數也是合法
       的 int64），放進去的話 SELECT 只會找不到東西，然後使用者看到 404「parent
       comment not found」—— 一句與他做的事（按回覆）毫無關係的話。

  3. 回傳給前端的 JSON 形狀
       parentId 在第一層時必須省略、depth 必須永遠存在。這兩件是前端組樹與決定
       「還能不能回覆」的唯一依據，而它們的失效症狀同樣不是錯誤：省略 depth 會讓
       回覆鈕的判斷退回 undefined，而前端因此可能在第 3 層仍渲染一顆必然 400 的
       按鈕。

  4. 刪除的子樹 SQL（forumCommentSubtreeJoin / …Condition）
       需要真的 MySQL 才會執行（見 comment_reply_mysql_test.go），因此這裡只檢查
       它的**形狀**：三條路徑對應三層上限（TestForumCommentSubtreeJoinCoversMaxDepth
       釘住這個對應，上限改動就不會只改數字而忘了加路徑），沒有任何子查詢
       （ERROR 1093），而 post_id 必須是第一個條件（索引在此）。

刻意沒有測的：實際的 depth 讀取與 INSERT（需要 MySQL）。授權邊界由既有的
TestPostAndCommentMutationsRequireLogin 守住，而 SQL 本身是單一陳述句。
*/
package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

/* ==========================================================================
   層級上限
   ========================================================================== */

/*
父留言在第 parentDepth 層時還能不能回覆。

表格刻意涵蓋 0、1、2、3、4 而不是只測「3 不行」：0 與負數代表「資料裡發生了
沒有預期過的事」（父留言不存在程式的哪裡會算出 0 depth），而那時的答案是
「當成可以」—— 後端接下來的 forumParentCommentDepth 會回 ErrNoRows 而變成
404，這個函式不該是那道防線，但它也不該在意外輸入時 panic 或說不可以。
*/
func TestForumParentDepthAllowsReply(t *testing.T) {
	cases := []struct {
		parentDepth int
		want        bool
	}{
		{0, true},  // 讀取失敗時的退化值：放行，讓後面的 404 說話
		{1, true},  // 回覆第一層 → 第二層
		{2, true},  // 回覆第二層 → 第三層（最後一層）
		{3, false}, // 第三層已經是最深，不能再長
		{4, false}, // 超過上限的資料同樣不能再長
	}
	for _, tc := range cases {
		if got := forumParentDepthAllowsReply(tc.parentDepth); got != tc.want {
			t.Errorf("forumParentDepthAllowsReply(%d) = %v, want %v", tc.parentDepth, got, tc.want)
		}
	}
}

// 上限是產品規則，而它同時被前端（MAX_COMMENT_DEPTH）與 schema 的 depth 上限
// 引用。改這個數字是一次產品決定，不該是三個檔案各自的一個字面值。
func TestForumCommentMaxDepth(t *testing.T) {
	if forumCommentMaxDepth != 3 {
		t.Errorf("forumCommentMaxDepth = %d, want 3（前端 MAX_COMMENT_DEPTH 與 "+
		 "docs/API.md 的說明都寫著三層）", forumCommentMaxDepth)
	}
}

/* ==========================================================================
   parentId 的輸入邊界
   ========================================================================== */

/*
負的 parentId 必須是 400，而且必須在碰到資料庫之前。

走 newPostEditTestServer（沒有 db）正是這條測試的一半意義：這個 server 的
db 是 nil，任何真的去查詢留言的路徑都會 panic —— 因此「回應是 400 而不是
panic」同時證明了那道檢查發生在查詢之前。
*/
func TestForumCommentRepliesRejectBadParentID(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{"負數的 parentId", `{"content":"同意","parentId":-1}`, "invalid parent comment"},
		{"極端負數", `{"content":"同意","parentId":-9223372036854775808}`, "invalid parent comment"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newPostEditTestServer(t, "http://localhost:8088")
			recorder := httptest.NewRecorder()
			request := loggedInRequest(t, server, http.MethodPost, "/api/forum/posts/1/comments", tc.body)
			request.Header.Set("Origin", "http://localhost:8088")
			server.handleForumComments(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("回應 = %d, want 400", recorder.Code)
			}
			if !strings.Contains(recorder.Body.String(), tc.wantMsg) {
				t.Errorf("body = %q，應包含 %q", recorder.Body.String(), tc.wantMsg)
			}
		})
	}
}

/*
parentId = 0 與省略必須是同一件事。

兩者都要一路走到「文章不存在」的 404 —— 那個 server 沒有 db，因此這裡預期的
正是 nil db 上的 panic：它證明這兩種輸入都沒有被任何 parentId 專屬的檢查擋下
（0 是「第一層」，不該有任何父留言檢查）。刻意用 recover 把 panic 轉成斷言，
否則這條測試只會以 panic 的形式失敗，看不出它想表達的是哪一條規則。
*/
func TestForumCommentTopLevelIgnoresParentCheck(t *testing.T) {
	for _, body := range []string{`{"content":"第一層留言"}`, `{"content":"第一層留言","parentId":0}`} {
		func() {
			defer func() {
				// nil db 上的第一個查詢就是文章存在性檢查；走到那裡代表
				// parentId 沒有被額外檢查擋下。
				if recovered := recover(); recovered == nil {
					t.Errorf("body %s 沒有走到資料庫查詢（%v），第一層留言不該被 parentId 檢查擋下", body, recovered)
				}
			}()
			server := newPostEditTestServer(t, "http://localhost:8088")
			recorder := httptest.NewRecorder()
			request := loggedInRequest(t, server, http.MethodPost, "/api/forum/posts/1/comments", body)
			request.Header.Set("Origin", "http://localhost:8088")
			server.handleForumComments(recorder, request)
		}()
	}
}

/* ==========================================================================
   對前端的 JSON 承諾
   ========================================================================== */

/*
forumComment 的 JSON 形狀就是前端組樹的契約。

兩條規則各自對應一個前端依賴：
  - parentId 在第一層省略（omitempty）：前端把「讀不到」與 0 視為同一個根節點。
    這裡釘死省略，是因為「送出 parentId: 0」不會壞，但它讓每一則第一層留言的
    回應都多一個沒有資訊量的欄位 —— 而第一層是多數。
  - depth 永遠存在：前端用它決定「還能不能回覆」。省略（例如改成 omitempty）
    會讓 Number(undefined) || level 永遠退回 level，而那在正常資料下恰好正確 ——
    於是沒有任何測試會紅，直到層級上限真的被推翻為止。
*/
func TestForumCommentJSONShape(t *testing.T) {
	top := forumComment{ID: 1, ParentID: 0, Depth: 1}
	raw, err := json.Marshal(top)
	if err != nil {
		t.Fatalf("marshal 失敗: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal 失敗: %v", err)
	}
	if _, ok := decoded["parentId"]; ok {
		t.Errorf("第一層留言不該帶 parentId：%s", raw)
	}
	if decoded["depth"] != float64(1) {
		t.Errorf("depth 必須永遠送出：%s", raw)
	}

	reply := forumComment{ID: 2, ParentID: 1, Depth: 2}
	raw, err = json.Marshal(reply)
	if err != nil {
		t.Fatalf("marshal 失敗: %v", err)
	}
	decoded = map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal 失敗: %v", err)
	}
	if decoded["parentId"] != float64(1) || decoded["depth"] != float64(2) {
		t.Errorf("回覆必須帶上 parentId 與 depth：%s", raw)
	}
}

/* ==========================================================================
   子樹刪除的 SQL 形狀
   ========================================================================== */

/*
forumCommentSubtreeJoin 的結構對應三層上限，以及「DELETE 的目標表不能掛子查詢」。

為什麼值得測一段 SQL 字串：它壞掉時不會有任何錯誤。少一路徑（例如忘了孫子）的
症狀是「刪掉一則第一層留言，它的孫回覆留在樹外」—— 而前端組樹會把那個孤兒放回
第一層，看起來就像一則普通的留言。改用子查詢的版本則有另一個失效模式：
MySQL 的 ERROR 1093（DELETE 的目標表不能出現在子查詢裡），而繞過它的衍生表寫法
只在最佳化器選擇具現化時合法。JOIN 寫法兩邊的好處都拿到，因此這裡逐項確認它。
*/
func TestForumCommentSubtreeJoinShape(t *testing.T) {
	join := forumCommentSubtreeJoin
	condition := forumCommentSubtreeCondition

	// 三層路徑：自己、父、祖父。
	if !strings.Contains(join, "FROM forum_post_comments c") {
		t.Errorf("刪除目標必須有別名 c：\n%s", join)
	}
	if !strings.Contains(join, "LEFT JOIN forum_post_comments parent1 ON parent1.id = c.parent_id") {
		t.Errorf("少了 parent1（回覆）：\n%s", join)
	}
	if !strings.Contains(join, "LEFT JOIN forum_post_comments parent2 ON parent2.id = parent1.parent_id") {
		t.Errorf("少了 parent2（回覆的回覆）：\n%s", join)
	}
	// 目標表不能掛子查詢：那是 MySQL ERROR 1093，而它的繞法只在最佳化器具現化
	// 衍生表時才合法。這條斷言讓「改回子查詢」的意圖必須先改測試。
	if strings.Contains(join+condition, "SELECT") {
		t.Errorf("子樹骨幹不該有子查詢（DELETE 的目標表掛子查詢會撞 ERROR 1093）：\n%s\n%s", join, condition)
	}
	// post_id 的等值條件讓 c 走 idx_forum_post_comments_post_id：沒有它，
	// 這句 DELETE 會掃描整張留言表。
	if !strings.HasPrefix(condition, "c.post_id = ?") {
		t.Errorf("子樹條件必須以 c.post_id = ? 開頭（索引在此）：\n%s", condition)
	}
	// 佔位符數目：post_id + 三個同一個留言 id。呼叫端依這個順序傳值，數目錯了
	// 不會是 SQL 錯誤，而是值被綁到別的欄位。
	if got := strings.Count(condition, "?"); got != 4 {
		t.Errorf("條件佔位符數目 = %d, want 4：\n%s", got, condition)
	}
}

/*
路徑數必須跟著層級上限走（forumCommentSubtreeJoin ↔ forumCommentMaxDepth）。

子樹刪除靠「沿著 parent_id 往上走」認出子孫，因此 parent{N} 的路徑數等於
「上限 - 1」：第一層的子孫最多兩層（回覆、回覆的回覆）。上限改成 4 而忘了加
一路徑時，症狀不是錯誤 —— 刪掉第一層留言，第四層的回覆安靜地留在樹外，而前端
組樹會把那個孤兒放回第一層，看起來像一句沒有上下文的話。TestForumCommentMaxDepth
釘的是數字本身，這一支釘的是「SQL 與那個數字同步」。

逐項比對 parent{N}.id = parent{N-1}.parent_id 而不只是數個數：鏈條斷開
（例如 parent2 指回 c.parent_id）時路徑數一樣是 3，但認出的子孫完全不同。
*/
func TestForumCommentSubtreeJoinCoversMaxDepth(t *testing.T) {
	join := forumCommentSubtreeJoin
	wantParents := forumCommentMaxDepth - 1

	if got := strings.Count(join, "LEFT JOIN forum_post_comments parent"); got != wantParents {
		t.Errorf("子樹 JOIN 的父路徑數 = %d, want %d（forumCommentMaxDepth=%d 減掉自己）",
			got, wantParents, forumCommentMaxDepth)
	}
	for i := 1; i <= wantParents; i++ {
		var on string
		if i == 1 {
			on = "parent1.id = c.parent_id"
		} else {
			on = fmt.Sprintf("parent%d.id = parent%d.parent_id", i, i-1)
		}
		want := fmt.Sprintf("LEFT JOIN forum_post_comments parent%d ON %s", i, on)
		if !strings.Contains(join, want) {
			t.Errorf("少了第 %d 條父路徑（%q）：\n%s", i, want, join)
		}
	}
	// 條件裡除了自己（c.id）之外，每一條父路徑都要有一個等值比對 ——
	// 少一個的症狀是「那一層的子孫不會被刪」，同樣不出錯。
	for i := 1; i <= wantParents; i++ {
		want := fmt.Sprintf("parent%d.id = ?", i)
		if !strings.Contains(forumCommentSubtreeCondition, want) {
			t.Errorf("子樹條件少了 %q：\n%s", want, forumCommentSubtreeCondition)
		}
	}
}
