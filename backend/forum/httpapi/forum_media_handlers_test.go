/*
舊寫法收斂表達式的形狀測試（httpapi/forum_media_handlers_test.go）。

forumImageReferenceCount 與 forumFileAttachable 對「舊資料的寫法」都做同一件事：
先把值收斂成純檔名，再拿它跟目標檔名做**等值**比對。收斂是三段 SQL 字串函式
（丟掉 `?` 之後的查詢字串 → 取最後一段路徑 → 丟掉 `]`），而它的失效方式是安靜的：
少任何一步、或把順序顛倒，查詢仍然回一份合法的數字，只是那個數字可能來自別的
值 —— 症狀是「一張還在用的圖被刪掉」或「別人的舊檔被當成自己引用過的」，兩者
不會出現在任何日誌裡。

因此把三段步驟釘成形狀測試（不需要 MySQL，這個專案沒有資料庫替身，
見 docs/KNOWN_ISSUES.md）。真實資料的端到端行為在
forum_media_reference_mysql_test.go（需要 FORUM_TEST_MYSQL_DSN）。
*/

package httpapi

import (
	"strings"
	"testing"
)

func TestForumImageLegacyExprShape(t *testing.T) {
	cases := []struct {
		name   string
		expr   string
		column string
	}{
		{"貼文附圖", forumImageLegacyPostsExpr, "image_url"},
		{"頭像", forumImageLegacyAvatarExpr, "avatar_url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.expr, "TRIM("+tc.column+")") {
				t.Errorf("%s 的表達式沒有對 %s 先 TRIM（舊資料可能帶空白）", tc.name, tc.column)
			}
			// 三步的順序是有意義的：先剝掉 query（完整網址帶 ?token=…），才取
			// 最後一段路徑（否則會得到 "a.png?token=…的殘段"），最後才剝掉
			// 舊版遷移黏在尾段的 `]`。顛倒任何兩步都會讓收斂不出正確檔名。
			query := strings.Index(tc.expr, "'?'")
			sep := strings.Index(tc.expr, "'/'")
			bracket := strings.Index(tc.expr, "']'")
			if query < 0 || sep < 0 || bracket < 0 {
				t.Fatalf("%s 的表達式缺少其中一步: %q", tc.name, tc.expr)
			}
			if !(query < sep && sep < bracket) {
				t.Errorf("%s 的收斂順序錯誤（query=%d, 路徑=%d, 括號=%d），應為 query → 路徑 → 括號: %q",
					tc.name, query, sep, bracket, tc.expr)
			}
		})
	}

	// 兩份表達式只能差在欄名：同一套收斂必須套在兩個欄位上，否則「貼文還在用
	// 這張圖」與「頭像還在用這張圖」會變成兩種不同的語意。
	posts := strings.Replace(forumImageLegacyPostsExpr, "image_url", "COL", 1)
	avatar := strings.Replace(forumImageLegacyAvatarExpr, "avatar_url", "COL", 1)
	if posts != avatar {
		t.Errorf("兩份收斂表達式除了欄名之外不同:\n posts=%s\n avatar=%s", posts, avatar)
	}
}
