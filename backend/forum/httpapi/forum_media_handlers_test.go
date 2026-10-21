/*
圖片 LIKE 樣式的測試（httpapi/forum_media_handlers_test.go）。

forumImageLikePattern 是「引用計數」（forumImageReferenceCount）與「舊制所有權
退路」（forumFileAttachable）共用的比對樣式，而它的失效方式是**安靜的**：不跳脫
檔名裡的底線時，樣式仍然是一份合法的 LIKE 樣式，查詢也仍然會回一個數字 —— 只是
那個數字可能來自別的圖片。症狀因此是「一張沒人用的圖刪不掉」（引用計數多算）或
「別人的舊檔被當成自己引用過的」（所有權誤判），兩者都不會出現在任何日誌裡。

刻意只測純函式這一層：這兩個函式的其餘部分需要真的 MySQL（這個專案沒有替身，
見 docs/KNOWN_ISSUES.md），而樣式本身的形狀不需要資料庫就能釘住。
*/
package httpapi

import "testing"

func TestForumImageLikePattern(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		want     string
	}{
		{"一般檔名只加前綴", "9f8c1a2b.png", `%/9f8c1a2b.png`},
		{"底線要跳脫", "a_b.png", `%/a\_b.png`},
		{"百分號要跳脫", "100%.png", `%/100\%.png`},
		{"反斜線要跳脫", `a\b.png`, `%/a\\b.png`},
		{"跳脫字元與萬用字元相鄰", `a\%_`, `%/a\\\%\_`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := forumImageLikePattern(tc.filename)
			if got != tc.want {
				t.Errorf("forumImageLikePattern(%q) = %q, want %q", tc.filename, got, tc.want)
			}
			// 不變條件：除了前綴的那一個 %，樣式裡不能再有**有作用**的萬用字元。
			// 這一條就是「LIKE 樣式不會被檔名撐寬」的定義，比逐一比對字串更
			// 不容易隨著實作改寫而失效。
			active := unescapedWildcards(got)
			if len(active) != 1 || active[0] != "%" {
				t.Errorf("forumImageLikePattern(%q) = %q 的有效萬用字元 = %v，應只有前綴的 %%",
					tc.filename, got, active)
			}
		})
	}
}

// unescapedWildcards 數出樣式裡真正有作用的 LIKE 萬用字元。
//
// 跳脫的判定與 MySQL 相同：一個反斜線讓下一個字元變成字面值，反斜線自己也要
// 能被跳脫（因此掃描是逐字元前進，而不是用 strings.Split 之類的近似）。
func unescapedWildcards(pattern string) []string {
	var out []string
	escaped := false
	for _, r := range pattern {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '%' || r == '_':
			out = append(out, string(r))
		}
	}
	return out
}
