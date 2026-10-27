/*
對外作者識別碼的測試（httpapi/author_key_test.go）。

這一份釘的是 F-03 修完之後「對外看到什麼」：

  - 設了 AUTHOR_KEY_SECRET：任何回應裡的作者識別碼都是 HMAC 形態，舊的
    SHA-256 值不再出現在任何回應裡（它是可離線枚舉還原的，那正是 F-03）。
  - 沒設秘密：退回舊形態（email 的 SHA-256），既有連結與既有部署不受影響。
  - 解析（個人頁、追蹤、退路掃描）同時接受兩把 key：舊連結要能點開。
  - 「是不是自己」比的是目前的形態，解析完再比一次 email：舊形態的 key
    仍解析得出同一個人，不能繞過自我追蹤的規則。
  - 秘密只走參數，永遠不進 SQL 文字（SQL 會進 general log 與日誌）。

函式以 TestAuthorKey 開頭，與既有 TestCov* 的覆蓋測試區隔。
*/
package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const authorKeySecret = "test-author-key-secret"

const authorKeyEmail = "author@example.com"
const authorKeyOtherEmail = "other@example.com"

// authorKeyHarness 建一個 Server，並把作者識別碼的秘密設成傳入的值。
// 直接改 srv.cfg 是因為「用哪種形態」只取決於 cfg.AuthorKeySecret 一個來源。
func authorKeyHarness(t *testing.T, secret string) *covHarness {
	t.Helper()
	h := newCovHarness(t)
	h.srv.cfg.AuthorKeySecret = secret
	return h
}

// wantHMAC 獨立算出 HMAC-SHA256(secret, email) 的小寫十六進位。
// 刻意不呼叫 DeriveAuthorKey：那樣這個測試只是把實作複述一遍，釘不住形態。
func wantHMAC(secret, email string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(email))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// wantHexKey 檢查識別碼是 64 個小寫十六進位字元 —— 前端與長度檢查的契約。
func wantHexKey(t *testing.T, key, label string) {
	t.Helper()
	if len(key) != 64 {
		t.Fatalf("%s 長度 = %d, want 64（%q）", label, len(key), key)
	}
	if _, err := hex.DecodeString(key); err != nil {
		t.Fatalf("%s 不是十六進位：%v（%q）", label, err, key)
	}
	for _, r := range key {
		if r >= 'A' && r <= 'F' {
			t.Fatalf("%s 應為小寫十六進位：%q", label, key)
		}
	}
}

/* ==========================================================================
   派生本身
   ========================================================================== */

func TestAuthorKeyDeriveWithSecretIsHMAC(t *testing.T) {
	key := DeriveAuthorKey(authorKeySecret, authorKeyEmail)
	wantHexKey(t, key, "有秘密的識別碼")

	if key != wantHMAC(authorKeySecret, authorKeyEmail) {
		t.Fatalf("DeriveAuthorKey = %q, want HMAC-SHA256(secret, email) = %q", key, wantHMAC(authorKeySecret, authorKeyEmail))
	}
	if key == publicForumKey(authorKeyEmail) {
		t.Fatal("有秘密時不應等於舊的 SHA-256 —— 那正是 F-03 要修掉的形態")
	}
	if key != DeriveAuthorKey(authorKeySecret, authorKeyEmail) {
		t.Fatal("同一個秘密與 email 必須每次得到同一個值（前端以它當 key）")
	}
	if key == DeriveAuthorKey(authorKeySecret, authorKeyOtherEmail) {
		t.Fatal("不同 email 必須得到不同的識別碼")
	}
	if key == DeriveAuthorKey("another-secret", authorKeyEmail) {
		t.Fatal("換秘密必須改變識別碼（換秘密是部署者明示的代價）")
	}
	// 識別碼裡不含原文：它不是「把 email 包起來加個前綴」這種做法。
	if strings.Contains(key, authorKeyEmail) || strings.Contains(key, authorKeySecret) {
		t.Fatalf("識別碼洩漏了原文：%q", key)
	}
}

func TestAuthorKeyDeriveWithoutSecretIsLegacy(t *testing.T) {
	key := DeriveAuthorKey("", authorKeyEmail)
	wantHexKey(t, key, "未設秘密的識別碼")
	if key != publicForumKey(authorKeyEmail) {
		t.Fatalf("未設秘密時應退回舊形態：got %q want %q", key, publicForumKey(authorKeyEmail))
	}
	// 空白秘密在 config 層就被 TrimSpace 成空字串（見 config.Load），因此
	// 這裡的「未設定」就是空字串本身；DeriveAuthorKey 不重複那道正規化。
	if DeriveAuthorKey("", authorKeyOtherEmail) == DeriveAuthorKey("", authorKeyEmail) {
		t.Fatal("未設秘密時不同 email 仍必須得到不同的值")
	}
}

// 形態只取決於 cfg.AuthorKeySecret 一個來源：所有端點都走 forumAuthorKey。
func TestAuthorKeyServerUsesSingleConfigSource(t *testing.T) {
	h := newCovHarness(t)
	if got := h.srv.forumAuthorKey(authorKeyEmail); got != publicForumKey(authorKeyEmail) {
		t.Fatalf("未設秘密時 forumAuthorKey = %q，want 舊形態", got)
	}

	h.srv.cfg.AuthorKeySecret = authorKeySecret
	if got := h.srv.forumAuthorKey(authorKeyEmail); got != wantHMAC(authorKeySecret, authorKeyEmail) {
		t.Fatalf("設了秘密後 forumAuthorKey = %q，want HMAC 形態", got)
	}
	if got := h.srv.forumAuthorKey(authorKeyEmail); got == publicForumKey(authorKeyEmail) {
		t.Fatal("設了秘密後不應仍對外發布舊形態")
	}
}

/* ==========================================================================
   對外的回應裡只出現 HMAC 形態
   ========================================================================== */

func TestAuthorKeyPublicPostsPublishHMAC(t *testing.T) {
	hmacKey := wantHMAC(authorKeySecret, authorKeyEmail)
	legacyKey := publicForumKey(authorKeyEmail)

	t.Run("列表", func(t *testing.T) {
		h := authorKeyHarness(t, authorKeySecret)
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(1, authorKeyEmail, nil, "內容", "", 0, nil, 0, 0, 0, 0))
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)

		body := recorder.Body.String()
		if !strings.Contains(body, `"authorKey":"`+hmacKey+`"`) {
			t.Errorf("列表的 authorKey 不是 HMAC 形態；body = %s", body)
		}
		if strings.Contains(body, legacyKey) {
			t.Errorf("列表仍出現舊的 SHA-256（可離線枚舉還原）；body = %s", body)
		}
		if strings.Contains(body, authorKeyEmail) {
			t.Errorf("列表洩漏 email；body = %s", body)
		}
	})

	t.Run("單篇", func(t *testing.T) {
		h := authorKeyHarness(t, authorKeySecret)
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(7, authorKeyEmail, nil, "永久連結內容", "", 0, nil, 1, 2, 0, 0))
		recorder := h.do(h.srv.handleForumPostDetail, http.MethodGet, "/api/forum/posts/7", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)

		body := recorder.Body.String()
		if !strings.Contains(body, `"authorKey":"`+hmacKey+`"`) {
			t.Errorf("單篇的 authorKey 不是 HMAC 形態；body = %s", body)
		}
		if strings.Contains(body, legacyKey) {
			t.Errorf("單篇仍出現舊的 SHA-256；body = %s", body)
		}
	})

	// 未設秘密的部署：對外仍是舊形態，既有前端與連結不受影響。
	t.Run("未設秘密時仍是舊形態", func(t *testing.T) {
		h := authorKeyHarness(t, "")
		h.rows("FROM forum_posts fp", covPostsMainColumns,
			covPostsMainRow(1, authorKeyEmail, nil, "內容", "", 0, nil, 0, 0, 0, 0))
		recorder := h.do(h.srv.handleForumPosts, http.MethodGet, "/api/forum/posts", "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), `"authorKey":"`+legacyKey+`"`) {
			t.Errorf("未設秘密時應保持舊形態；body = %s", recorder.Body.String())
		}
	})
}

func TestAuthorKeySearchResultsPublishHMAC(t *testing.T) {
	// ES 未設定時走 MySQL LIKE；組裝回應的那一段用 forumAuthorKey，
	// 與貼文卡必須同一種形態，否則前端從搜尋結果連出去的個人頁會查不到人。
	h := authorKeyHarness(t, authorKeySecret)
	h.rows("COUNT(*) FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
		[]string{"total"}, []driverValue{int64(1)})
	h.rows("SELECT id FROM forum_posts WHERE deleted_at IS NULL AND content LIKE",
		[]string{"id"}, []driverValue{int64(7)})
	h.rows("fp.id IN (", covSearchBatchPostColumns(), []driverValue{
		int64(7), authorKeyEmail, nil, "貼文內容",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		"", int64(0), nil, int64(2), int64(3), int64(0), int64(0),
	})

	recorder := h.do(h.srv.handleForumSearch, http.MethodGet, "/api/forum/search?q=hello", "", "", false)
	covWantStatus(t, recorder, http.StatusOK)

	body := recorder.Body.String()
	hmacKey := wantHMAC(authorKeySecret, authorKeyEmail)
	if !strings.Contains(body, `"authorKey":"`+hmacKey+`"`) {
		t.Errorf("搜尋結果的 authorKey 不是 HMAC 形態；body = %s", body)
	}
	if strings.Contains(body, publicForumKey(authorKeyEmail)) {
		t.Errorf("搜尋結果仍出現舊的 SHA-256；body = %s", body)
	}
	if strings.Contains(body, authorKeyEmail) {
		t.Errorf("搜尋結果洩漏 email；body = %s", body)
	}
}

// 通知是私密的，但它帶的是「別人的」識別碼，而且前端用它連到公開個人頁。
func TestAuthorKeyNotificationsUseCurrentForm(t *testing.T) {
	h := authorKeyHarness(t, authorKeySecret)
	item := h.srv.forumNotificationItemFromRow(forumNotificationRow{
		ID:        11,
		Kind:      forumNotificationFollow,
		ActorMail: authorKeyEmail,
		Nickname:  "小明",
		CreatedAt: time.Now(),
	}, "")

	if item.ActorKey != wantHMAC(authorKeySecret, authorKeyEmail) {
		t.Fatalf("actorKey = %q，want HMAC 形態", item.ActorKey)
	}
	if item.ActorKey == publicForumKey(authorKeyEmail) {
		t.Fatal("通知不應對外發布可離線枚舉的 SHA-256")
	}
	if item.Actor != "小明" {
		t.Fatalf("actor = %q，want 顯示名稱", item.Actor)
	}

	noSecret := newCovHarness(t)
	if got := noSecret.srv.forumNotificationItemFromRow(forumNotificationRow{ActorMail: authorKeyEmail}, "").ActorKey; got != publicForumKey(authorKeyEmail) {
		t.Fatalf("未設秘密時 actorKey = %q，want 舊形態", got)
	}
}

/* ==========================================================================
   解析：兩把 key 都接受，舊連結不破
   ========================================================================== */

const publicProfileSelect = "SELECT author_email, nickname, bio, avatar_url FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1"

func TestAuthorKeyPublicProfileResolvesBothForms(t *testing.T) {
	for _, form := range []struct {
		name string
		key  string
	}{
		{"目前形態（HMAC）", wantHMAC(authorKeySecret, authorKeyEmail)},
		{"舊形態（SHA-256，第 16 步的 public_key 欄位）", publicForumKey(authorKeyEmail)},
	} {
		t.Run(form.name, func(t *testing.T) {
			h := authorKeyHarness(t, authorKeySecret)
			h.rows(publicProfileSelect,
				[]string{"author_email", "nickname", "bio", "avatar_url"},
				[]driverValue{authorKeyEmail, "作者", "簡介", ""})

			recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet,
				"/api/forum/public-profile?key="+form.key, "", "", false)
			covWantStatus(t, recorder, http.StatusOK)
			if !strings.Contains(recorder.Body.String(), `"nickname":"作者"`) {
				t.Errorf("body = %s", recorder.Body.String())
			}
			if !h.sawQuery("public_key_hmac = ? OR public_key = ?") {
				t.Error("沒有用兩把識別碼欄位查詢")
			}
			// 兩欄都用同一個 key 比對：不是「一欄查 HMAC、另一欄查舊 SHA」。
			args := h.argsFor(publicProfileSelect)
			if len(args) != 2 {
				t.Fatalf("參數數 = %d, want 2（%v）", len(args), args)
			}
			if args[0] != form.key || args[1] != form.key {
				t.Fatalf("參數 = %v, want 兩個都是 %q", args, form.key)
			}
			// 回應裡不出現 email。
			if strings.Contains(recorder.Body.String(), authorKeyEmail) {
				t.Errorf("個人頁洩漏 email；body = %s", recorder.Body.String())
			}
		})
	}
}

// 退路掃描（沒有建檔的使用者）用目前的形態比對。
func TestAuthorKeyFallbackScanMatchesCurrentForm(t *testing.T) {
	h := authorKeyHarness(t, authorKeySecret)
	// 不登記個人資料的查詢 → 兩欄都查不到 → 走到退路掃描。
	h.rows("SELECT DISTINCT author_email FROM forum_posts",
		[]string{"author_email"},
		[]driverValue{authorKeyOtherEmail},
		[]driverValue{authorKeyEmail},
	)

	t.Run("目前的形態命中", func(t *testing.T) {
		email, err := h.srv.forumEmailByPublicKey(context.Background(), wantHMAC(authorKeySecret, authorKeyEmail))
		if err != nil {
			t.Fatalf("forumEmailByPublicKey: %v", err)
		}
		if email != authorKeyEmail {
			t.Fatalf("email = %q, want %q", email, authorKeyEmail)
		}
	})

	// 設了秘密後，舊形態在退路裡不再命中：對外已不發布舊值，而這條退路是
	// 「拿掃描到的 email 現算比對」，沒有秘密就反推不出來。
	t.Run("舊形態在退路裡不命中", func(t *testing.T) {
		_, err := h.srv.forumEmailByPublicKey(context.Background(), publicForumKey(authorKeyEmail))
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})

	// 未設秘密時，退路仍用舊形態命中（既有部署的行為不變）。
	t.Run("未設秘密時舊形態命中", func(t *testing.T) {
		fallback := authorKeyHarness(t, "")
		fallback.rows("SELECT DISTINCT author_email FROM forum_posts",
			[]string{"author_email"}, []driverValue{authorKeyEmail})
		email, err := fallback.srv.forumEmailByPublicKey(context.Background(), publicForumKey(authorKeyEmail))
		if err != nil || email != authorKeyEmail {
			t.Fatalf("forumEmailByPublicKey = (%q, %v)", email, err)
		}
	})
}

/* ==========================================================================
   追蹤：自我追蹤在兩種形態下都擋得住
   ========================================================================== */

const followProfileSelect = "SELECT author_email FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1"

func TestAuthorKeyFollowSelfBlockedInBothForms(t *testing.T) {
	t.Run("目前形態：在任何交易之前擋掉", func(t *testing.T) {
		h := authorKeyHarness(t, authorKeySecret)
		recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
			`{"key":"`+wantHMAC(authorKeySecret, "me@example.com")+`"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "cannot follow yourself") {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if h.sawQuery(followProfileSelect) {
			t.Error("自我追蹤不應花一次解析查詢")
		}
		if h.sawQuery("INSERT INTO forum_follows") {
			t.Error("自我不應寫進 forum_follows")
		}
	})

	// 舊形態的 key 仍解析得出同一個人（public_key 欄位裡存的是舊值）。
	// 設了秘密之後它不再對外發布，但仍可能從舊連結進來 —— 第二道檢查比的是
	// 解析出來的 email。
	t.Run("舊形態：解析完再擋一次", func(t *testing.T) {
		h := authorKeyHarness(t, authorKeySecret)
		h.rows(followProfileSelect, []string{"author_email"}, []driverValue{"me@example.com"})
		recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
			`{"key":"`+publicForumKey("me@example.com")+`"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "cannot follow yourself") {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if h.sawQuery("INSERT INTO forum_follows") {
			t.Error("自我不應寫進 forum_follows（複合主鍵擋不掉 ('x','x')）")
		}
	})

	// 未設秘密時仍走第一道檢查（舊形態就是目前的形態）。
	t.Run("未設秘密時仍擋住", func(t *testing.T) {
		h := authorKeyHarness(t, "")
		recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
			`{"key":"`+publicForumKey("me@example.com")+`"}`, "me@example.com", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "cannot follow yourself") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})
}

func TestAuthorKeyFollowResolvesOtherUserWithHMAC(t *testing.T) {
	h := authorKeyHarness(t, authorKeySecret)
	h.rows(followProfileSelect, []string{"author_email"}, []driverValue{authorKeyEmail})
	// 原本沒有追蹤：現有的計數必須回 0，否則替身的空結果集會讓 Scan 失敗。
	h.rows("SELECT COUNT(*) FROM forum_follows WHERE follower_email",
		[]string{"count"}, []driverValue{int64(0)})

	recorder := h.do(h.srv.handleForumFollows, http.MethodPost, "/api/forum/follows",
		`{"key":"`+wantHMAC(authorKeySecret, authorKeyEmail)+`"}`, "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)

	args := h.argsFor("INSERT INTO forum_follows")
	if len(args) != 3 {
		t.Fatalf("INSERT 參數數 = %d, want 3（%v）", len(args), args)
	}
	if args[0] != "me@example.com" || args[1] != authorKeyEmail {
		t.Fatalf("追蹤寫的是 (%v, %v)，want (me@example.com, %q)", args[0], args[1], authorKeyEmail)
	}
}

// 追蹤清單上的 key 與貼文卡上的 authorKey 必然一致：同一支函式算出來的。
func TestAuthorKeyFollowListKeysMatchPostCards(t *testing.T) {
	h := authorKeyHarness(t, authorKeySecret)
	h.rows("FROM forum_follows",
		[]string{"target_email", "nickname", "avatar", "followed_at"},
		[]driverValue{authorKeyEmail, "被追蹤者", "", time.Now()})
	recorder := h.do(h.srv.handleForumFollows, http.MethodGet, "/api/forum/follows", "", "me@example.com", false)
	covWantStatus(t, recorder, http.StatusOK)

	body := recorder.Body.String()
	if !strings.Contains(body, `"key":"`+wantHMAC(authorKeySecret, authorKeyEmail)+`"`) {
		t.Errorf("追蹤清單的 key 不是 HMAC 形態；body = %s", body)
	}
	if !strings.Contains(body, `"selfKey":"`+wantHMAC(authorKeySecret, "me@example.com")+`"`) {
		t.Errorf("selfKey 不是 HMAC 形態；body = %s", body)
	}
	if strings.Contains(body, publicForumKey(authorKeyEmail)) {
		t.Errorf("追蹤清單仍出現舊的 SHA-256；body = %s", body)
	}
}

/* ==========================================================================
   個人資料：兩把識別碼都寫進資料庫，且秘密不進 SQL
   ========================================================================== */

const profileUpsert = "INSERT INTO forum_profiles (author_email, public_key, public_key_hmac,"

func TestAuthorKeyProfileUpsertWritesBothColumns(t *testing.T) {
	t.Run("有秘密時兩欄各寫自己的形態", func(t *testing.T) {
		h := authorKeyHarness(t, authorKeySecret)
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile",
			`{"nickname":"暱稱"}`, authorKeyEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery(profileUpsert) {
			t.Fatal("沒有寫入個人資料的兩把識別碼")
		}

		args := h.argsFor(profileUpsert)
		if len(args) < 3 {
			t.Fatalf("INSERT 參數數 = %d, want 至少 3（%v）", len(args), args)
		}
		if args[0] != authorKeyEmail {
			t.Fatalf("author_email = %v, want %q", args[0], authorKeyEmail)
		}
		// 舊欄位保持舊形態（第 16 步的 SQL SHA2 值），新欄位寫目前的形態：
		// 舊連結靠舊欄位解析，新連結靠新欄位走索引。
		if args[1] != publicForumKey(authorKeyEmail) {
			t.Fatalf("public_key = %v，want 舊形態 %q", args[1], publicForumKey(authorKeyEmail))
		}
		if args[2] != wantHMAC(authorKeySecret, authorKeyEmail) {
			t.Fatalf("public_key_hmac = %v，want HMAC 形態 %q", args[2], wantHMAC(authorKeySecret, authorKeyEmail))
		}
	})

	t.Run("未設秘密時兩欄都是舊形態", func(t *testing.T) {
		h := authorKeyHarness(t, "")
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile",
			`{"nickname":"暱稱"}`, authorKeyEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		args := h.argsFor(profileUpsert)
		if len(args) < 3 {
			t.Fatalf("INSERT 參數數 = %d, want 至少 3（%v）", len(args), args)
		}
		if args[1] != publicForumKey(authorKeyEmail) || args[2] != publicForumKey(authorKeyEmail) {
			t.Fatalf("未設秘密時兩欄都應是舊形態：%v / %v", args[1], args[2])
		}
	})

	// 秘密不進 SQL 文字：SQL 會進 MySQL 的 general log 與日誌（docs/SECURITY.md）。
	// 它只走參數，因此替身記錄的 SQL 裡不應出現它。
	t.Run("秘密不進 SQL 文字", func(t *testing.T) {
		h := authorKeyHarness(t, authorKeySecret)
		if _, err := h.srv.forumEmailByPublicKey(context.Background(), wantHMAC(authorKeySecret, authorKeyEmail)); err != nil && !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("forumEmailByPublicKey: %v", err)
		}
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile",
			`{"nickname":"暱稱"}`, authorKeyEmail, false)
		covWantStatus(t, recorder, http.StatusOK)

		for _, query := range h.queriesSeen() {
			if strings.Contains(query, authorKeySecret) {
				t.Fatalf("SQL 文字裡出現秘密：%s", query)
			}
		}
	})
}

// 派生只有一個來源：呼叫端不直接算 SHA。
func TestAuthorKeyPublicForumKeyStillAvailableForLegacyRows(t *testing.T) {
	// publicForumKey 仍然存在（它是 public_key 欄位裡的值，舊連結要能解析），
	// 但不再是任何對外回應的來源 —— 那由上面那些測試釘住。
	key := publicForumKey(authorKeyEmail)
	wantHexKey(t, key, "舊形態")
	if key != publicForumKey(authorKeyEmail) {
		t.Fatal("publicForumKey 必須穩定（資料庫裡已存的值靠它比對）")
	}
}
