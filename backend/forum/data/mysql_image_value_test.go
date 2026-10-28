/*
附圖欄位的值收斂（mysql_image_value_test.go）。

第 2b 步（normalizeForumImageValues）把 forum_posts.image_url 從「單一檔名或完整
網址」改成新契約的 JSON 陣列字串，並修掉舊版遷移**順序**錯誤留下的損壞值。這裡的
每一種輸入形狀都出現在真實資料裡：新契約的檔名、`/files/x.png`、帶 `?token=…` 的
完整網址，以及「先包成 JSON 再切路徑」留下的 `x.png"]`。

為什麼值得單獨測：收斂規則同時存在三份 —— data 的 Go（本檔測試的對象）、httpapi
與 purge 的 SQL SUBSTRING_INDEX 表達式。三份必須逐字對應，否則「遷移後存的值」與
「查詢時收斂出的值」會分歧，而那個分歧的症狀是「縮圖看得見，但引用計數查不到」：
清理排程把仍然有人在用的圖當沒有引用而刪掉。
*/

package data

import (
	"errors"
	"testing"
)

func TestLegacyForumImageName(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"a_b.png", "a_b.png"},
		{"/files/a_b.png", "a_b.png"},
		{"http://localhost:7070/files/a_b.png", "a_b.png"},
		{"http://localhost:7070/files/a_b.png?token=xyz", "a_b.png"},
		{"/files/a_b.png?token=xyz", "a_b.png"},
		// 舊版遷移順序錯誤的損壞值：尾段是 `"]` 兩個字元，只丟 `]` 會剩下一個引號。
		{`a_b.png"]`, "a_b.png"},
		{`["http://localhost:7070/files/a_b.png?token=xyz"]`, "a_b.png"},
		// 空白是舊資料裡常見的雜訊（欄位沒有 TRIM 過時）。
		{"  a_b.png  ", "a_b.png"},
		{"", ""},
		{"   ", ""},
		{"/files/", ""},
	}
	for _, c := range cases {
		if got := legacyForumImageName(c.raw); got != c.want {
			t.Errorf("legacyForumImageName(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestForumImageValueToCanonical(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		// 空值一律變成 `[]`：新契約的「沒有附圖」是空陣列，不是空字串。
		{"", "[]"},
		{"   ", "[]"},
		{"[]", "[]"},
		// 單一值：收斂之後包成單元素陣列。
		{"a.png", `["a.png"]`},
		{"/files/a.png", `["a.png"]`},
		{"http://192.168.66.5:7070/files/a.png?token=t", `["a.png"]`},
		{`a.png"]`, `["a.png"]`},
		// 已是 JSON 陣列：逐元素收斂（舊版可能把完整網址包進陣列裡）。
		{`["a.png","b.png"]`, `["a.png","b.png"]`},
		{`["/files/a.png","http://host/files/b.png?token=t"]`, `["a.png","b.png"]`},
		// 陣列裡收斂不出來的元素被丟掉；全丟光時是 `[]` 而不是 `[""]`。
		{`["https://evil.test/x.png"]`, `["x.png"]`},
		{`["/files/"]`, "[]"},
		// 不合法的 JSON（舊資料被截斷過）走單值那條路：它不是檔名，但收斂不出來
		// 的東西必須仍被包成合法 JSON，否則 JSON_VALID 的查詢會在這列上永遠失敗。
		{`[broken`, `["[broken"]`},
	}
	for _, c := range cases {
		if got := forumImageValueToCanonical(c.raw); got != c.want {
			t.Errorf("forumImageValueToCanonical(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

// TestNormalizeForumImageValuesRewritesLegacyRows 走完整的「數到舊值 → 逐列收斂 →
// 回寫」流程，並確認只有**真的需要改**的列會被 UPDATE。
//
// 第二列（已是新契約）不會出現在 UPDATE 裡：冪等是靠這個條件成立的，第二次啟動
// 必須不動任何一列。
func TestNormalizeForumImageValuesRewritesLegacyRows(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: "SELECT COUNT(*) FROM forum_posts WHERE image_url LIKE",
		cols:   []string{"COUNT(*)"},
		rows:   [][]covDataValue{{int64(2)}},
	})
	src.addQuery(covDataQueryRule{
		substr: "SELECT id, image_url FROM forum_posts WHERE image_url LIKE",
		cols:   []string{"id", "image_url"},
		rows: [][]covDataValue{
			{int64(7), `/files/a.png?token=t`},
			{int64(8), `["b.png"]`},
		},
	})

	if err := normalizeForumImageValues(db); err != nil {
		t.Fatalf("normalizeForumImageValues: %v", err)
	}
	if !src.saw(`UPDATE forum_posts SET image_url = ? WHERE id = ?`) {
		t.Fatal("沒有把收斂後的值回寫 —— 舊寫法會永遠留在庫裡")
	}
	if got := src.countSeen(`UPDATE forum_posts SET image_url = ? WHERE id = ?`); got != 1 {
		t.Errorf("回寫次數 = %d, want 1（已是新契約的那一列不該被再寫）", got)
	}
}

// TestNormalizeForumImageValuesSkipsWhenNothingLegacy 釘住「沒有舊值就不碰資料庫」
// 的另一條路：計數為 0 時既不 SELECT 逐列，也不 UPDATE。
//
// 這一條的意義是啟動成本：一個已經遷移完的庫，每次啟動都要跑這條探測，若它還逐列
// SELECT + 回寫，遷移就變成每次啟動都做一次全表寫入。
func TestNormalizeForumImageValuesSkipsWhenNothingLegacy(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: "SELECT COUNT(*) FROM forum_posts WHERE image_url LIKE",
		cols:   []string{"COUNT(*)"},
		rows:   [][]covDataValue{{int64(0)}},
	})

	if err := normalizeForumImageValues(db); err != nil {
		t.Fatalf("normalizeForumImageValues: %v", err)
	}
	if src.saw("SELECT id, image_url FROM forum_posts") {
		t.Error("計數為 0 卻仍逐列 SELECT")
	}
	if src.saw("UPDATE forum_posts SET image_url") {
		t.Error("計數為 0 卻仍回寫 —— 每次啟動都寫全表")
	}
}

// TestNormalizeForumImageValuesPropagatesErrors 覆蓋三條會中途中止的分支：計數失敗、
// SELECT 失敗、回寫失敗。任何一條都必須讓 MigrateMySQL 在啟動時失敗（而不是默默
// 留著舊寫法），因為舊寫法的值會持續被引用計數誤讀。
func TestNormalizeForumImageValuesPropagatesErrors(t *testing.T) {
	t.Run("計數失敗", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{
			substr: "SELECT COUNT(*) FROM forum_posts WHERE image_url LIKE",
			err:    errors.New("db down"),
		})
		if err := normalizeForumImageValues(db); err == nil {
			t.Fatal("計數失敗時必須回傳錯誤")
		}
	})

	t.Run("逐列 SELECT 失敗", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{
			substr: "SELECT COUNT(*) FROM forum_posts WHERE image_url LIKE",
			cols:   []string{"COUNT(*)"},
			rows:   [][]covDataValue{{int64(1)}},
		})
		src.addQuery(covDataQueryRule{
			substr: "SELECT id, image_url FROM forum_posts WHERE image_url LIKE",
			err:    errors.New("db down"),
		})
		if err := normalizeForumImageValues(db); err == nil {
			t.Fatal("SELECT 失敗時必須回傳錯誤")
		}
	})

	t.Run("回寫失敗", func(t *testing.T) {
		db, src := covDataDB(t)
		src.addQuery(covDataQueryRule{
			substr: "SELECT COUNT(*) FROM forum_posts WHERE image_url LIKE",
			cols:   []string{"COUNT(*)"},
			rows:   [][]covDataValue{{int64(1)}},
		})
		src.addQuery(covDataQueryRule{
			substr: "SELECT id, image_url FROM forum_posts WHERE image_url LIKE",
			cols:   []string{"id", "image_url"},
			rows:   [][]covDataValue{{int64(9), `/files/a.png`}},
		})
		src.addExec(covDataExecRule{substr: "UPDATE forum_posts SET image_url", err: errors.New("write failed")})
		if err := normalizeForumImageValues(db); err == nil {
			t.Fatal("回寫失敗時必須回傳錯誤")
		}
	})
}
