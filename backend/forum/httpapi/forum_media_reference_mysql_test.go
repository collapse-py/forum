/*
圖片引用計數與舊制所有權退路的真實 MySQL 測試
（backend/forum/httpapi/forum_media_reference_mysql_test.go）。

為什麼需要它：forumImageReferenceCount（換圖／刪圖的回收判斷）與
forumFileAttachable（「這個檔案是不是自己的」的舊制退路）對「舊資料的寫法」
比對的是**收斂之後的等值**，而不是 LIKE 樣式。這兩條路徑的誤判都是安靜的：
查詢仍然回一份合法的數字，只是那個數字可能來自別的值或根本是 0。

這裡釘住四種寫法都必須被算成「有人在用」：
  1. 新契約：image_url 是 JSON 陣列字串（`["shared.png","other.png"]`）。
     這一種是資料毀損的入口 —— 只問 `image_url = ?` 時它永遠是 0，
     換圖回收與保留期清理就會刪掉別人還在用的圖。
  2. 舊寫法：`/files/x.png` 與帶 `?token=…` 的完整網址。
  3. 舊版遷移的損壞值：`x.png"]`（先包成 JSON 再切路徑的產物，
     見 data.MigrateMySQL 第 2b/18 步）。
  4. 頭像的舊寫法（forum_profiles.avatar_url 沒有遷移，舊值可能仍是完整網址）。

約定同 comment_reply_mysql_test.go：FORUM_TEST_MYSQL_DSN 未設定時 skip。
*/
package httpapi

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// insertForumPostImage 直接寫一列帶附圖的貼文（不走 handler），目的是讓「四種
// 寫法」能原樣出現在欄位裡 —— 透過 handler 寫入的資料只會是新契約那一種。
func insertForumPostImage(t *testing.T, db *sql.DB, id int64, email, imageURL string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO forum_posts (id, author_email, content, image_url, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, email, "測試內容 "+fmt.Sprint(id), imageURL, time.Now().UTC()); err != nil {
		t.Fatalf("寫入測試貼文 %d 失敗: %v", id, err)
	}
}

func insertForumProfileAvatar(t *testing.T, db *sql.DB, email, nickname, avatar string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO forum_profiles (author_email, nickname, bio, updated_at, avatar_url)
		VALUES (?, ?, ?, ?, ?)`,
		email, nickname, "", time.Now().UTC(), avatar); err != nil {
		t.Fatalf("寫入測試個人資料 %s 失敗: %v", email, err)
	}
}

func TestForumImageReferenceCountMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	srv := newCoreForumServer(t, db)
	ctx := context.Background()

	owner := "owner@example.test"
	// 四種都指向同一個檔名：任何一種收斂不出來，總數就會少一。
	insertForumPostImage(t, db, 1, owner, `["shared.png","other.png"]`)
	insertForumPostImage(t, db, 2, owner, "/files/shared.png")
	insertForumPostImage(t, db, 3, owner, "http://localhost:7070/files/shared.png?token=abc")
	insertForumPostImage(t, db, 4, owner, `shared.png"]`)
	// 一個不一樣的檔名，用来確認「收斂」不會把別的檔案算成這張圖。
	insertForumPostImage(t, db, 5, owner, `["nothere.png"]`)

	counts := []struct {
		name string
		want int
	}{
		{"新契約 + 三種舊寫法都要算到", 4},
		{"other.png 只有第 1 列引用", 1},
		{"沒有人用的檔名", 0},
	}
	names := []string{"shared.png", "nothere.png", "missing.png"}
	for i, c := range counts {
		got, err := srv.forumImageReferenceCount(ctx, names[i])
		if err != nil {
			t.Fatalf("forumImageReferenceCount(%q) 失敗: %v", names[i], err)
		}
		if got != c.want {
			t.Errorf("forumImageReferenceCount(%q) = %d, want %d（%s）", names[i], got, c.want, c.name)
		}
	}

	// 檔名含底線時不能匹配到 aXb.png（LIKE 的 _ 是萬用字元；收斂後等值不會）。
	insertForumPostImage(t, db, 6, owner, `["a_b.png"]`)
	if got, err := srv.forumImageReferenceCount(ctx, "a_b.png"); err != nil || got != 1 {
		t.Errorf("底線檔名的引用計數 = (%d, %v), want (1, nil)", got, err)
	}
	if got, err := srv.forumImageReferenceCount(ctx, "aXb.png"); err != nil || got != 0 {
		t.Errorf("aXb.png 不該被 a_b.png 那列算成引用 = (%d, %v), want (0, nil)", got, err)
	}
}

func TestForumFileAttachableLegacyMySQL(t *testing.T) {
	db := subtreeTempDatabase(t)
	srv := newCoreForumServer(t, db)
	ctx := context.Background()

	owner := "owner@example.test"
	other := "other@example.test"
	// 舊制：沒有 forum_media 的上傳紀錄，所有權只能靠「已經是自己內容的一部分」
	// 這個退路。頭像帶 token 的舊網址必須仍能认出是自己的，否则使用者换头像时
	// 會被 403 擋掉（舊的 LIKE 樣式匹配不到帶 token 的值）。
	insertForumProfileAvatar(t, db, owner, "owner", "http://localhost:7070/files/mine.png?token=abc")
	insertForumPostImage(t, db, 1, owner, "http://localhost:7070/files/post.png?token=abc")
	insertForumProfileAvatar(t, db, other, "other", "")

	own := []struct {
		name string
		file string
	}{
		{"舊寫法的頭像", "mine.png"},
		{"舊寫法的貼文附圖", "post.png"},
	}
	for _, c := range own {
		ok, err := srv.forumFileAttachable(ctx, c.file, owner)
		if err != nil {
			t.Fatalf("forumFileAttachable(%q) 失敗: %v", c.file, err)
		}
		if !ok {
			t.Errorf("%s：自己的舊檔應可掛回自己的內容，got false", c.name)
		}
	}

	// 別人的舊檔不能被掛到自己的內容上（這是最危險的誤判方向：掛上之後換圖
	// 回收只問引用計數，那個檔案真的會被刪掉）。
	ok, err := srv.forumFileAttachable(ctx, "mine.png", other)
	if err != nil {
		t.Fatalf("forumFileAttachable 失敗: %v", err)
	}
	if ok {
		t.Errorf("別人的舊檔不應可掛到 other 的內容上，got true")
	}
	// 沒有上傳紀錄、也沒有任何內容引用過的檔案：一律不可掛。
	if ok, err := srv.forumFileAttachable(ctx, "nothere.png", owner); err != nil || ok {
		t.Errorf("沒有引用的檔案不應可掛 = (%v, %v), want (false, nil)", ok, err)
	}
}
