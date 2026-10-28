/*
個人資料、公開貼文與圖片的覆蓋測試（httpapi/cov_forum_profile_media_test.go）。

這一份只覆蓋 forum_handlers.go 的個人資料／個人貼文區塊、
forum_bookmark_handlers.go 的收藏清單，以及 forum_media_handlers.go 的
圖片所有權與刪除流程。全部使用 cov_harness_test.go 的資料庫替身，因此不需要
MySQL；檔案伺服器一律以 httptest.Server 代替，不會連外網。

測試名稱一律以 TestCovForumProfile 開頭，helper 一律以 covProfile 開頭，避免
與同套件其他測試檔的符號衝突。
*/
package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

const (
	covProfileEmail      = "cov-profile@example.com"
	covProfilePublicBase = "https://cdn.cov.test"
	covProfileFilesBase  = "http://files.cov.internal:7070"
	covProfileKeyPrefix  = "cov:profile:token:"
)

// covProfileConfigureMedia 把 harness 的預設空設定補成有意義的檔案伺服器設定。
// NewServer 不會套用 config.Load 的預設值，因此 harness 的 cfg 這些欄位是零值。
func covProfileConfigureMedia(h *covHarness) {
	h.srv.cfg.FilesServerURL = covProfileFilesBase
	h.srv.cfg.FilesServerPublicURL = covProfilePublicBase
	h.srv.cfg.MediaTokenKeyPrefix = covProfileKeyPrefix
	h.srv.cfg.MediaTokenTTLSecs = 60
}

func covProfileJSONBody(t *testing.T, payload interface{}) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(raw)
}

func covProfileJSONMap(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("回應不是 JSON：%v；body = %s", err, recorder.Body.String())
	}
	return payload
}

// covProfilePostCols 必須與 forumPostProjection 的欄位順序一致。
func covProfilePostCols() []string {
	return []string{
		"id", "author_email", "author_avatar", "content", "created_at", "image_url",
		"pinned", "updated_at", "like_count", "comment_count", "liked_by_me", "bookmarked_by_me",
	}
}

func covProfilePostRow(id int64, email, content, imageURL string) []driverValue {
	return []driverValue{
		id, email, nil, content, time.Now(), imageURL,
		int64(0), nil, int64(2), int64(3), int64(0), int64(0),
	}
}

func covProfileMultipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("寫入 multipart 內容: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("關閉 multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func covProfileUploadRequest(t *testing.T, h *covHarness, field, filename string, content []byte, origin, email string) *http.Request {
	t.Helper()
	body, contentType := covProfileMultipartBody(t, field, filename, content)
	req := httptest.NewRequest(http.MethodPost, "/api/forum/images", body)
	req.Header.Set("Content-Type", contentType)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if email != "" {
		req.AddCookie(h.login(email, false))
	}
	return req
}

func covProfileDoOrigin(h *covHarness, handler http.HandlerFunc, method, target, body, email, origin string) *httptest.ResponseRecorder {
	req := h.request(method, target, body, email, false)
	req.Header.Set("Origin", origin)
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

/* ==========================================================================
   公開個人資料
   ========================================================================== */

func TestCovForumProfilePublicProfile(t *testing.T) {
	t.Run("方法不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodPut, "/api/forum/public-profile?key="+strings.Repeat("a", 64), "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("金鑰長度不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet, "/api/forum/public-profile?key=short", "", "", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("查無資料回預設值", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet, "/api/forum/public-profile?key="+strings.Repeat("b", 64), "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		body := recorder.Body.String()
		for _, want := range []string{"匿名使用者", "這位使用者尚未設定公開資料。"} {
			if !strings.Contains(body, want) {
				t.Errorf("body 缺少 %q：%s", want, body)
			}
		}
		if strings.Contains(body, "following") {
			t.Errorf("沒有 profile 時不該帶 following：%s", body)
		}
	})

	t.Run("有資料且有頭像且追蹤中", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT author_email, nickname, bio, avatar_url FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email", "nickname", "bio", "avatar_url"},
			[]driverValue{"target@example.com", "目標", "簡介", "avatar.png"},
		)
		h.rows("SELECT COUNT(*) FROM forum_follows WHERE follower_email",
			[]string{"count"}, []driverValue{int64(1)})

		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet,
			"/api/forum/public-profile?key="+strings.Repeat("c", 64), "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if payload["nickname"] != "目標" || payload["bio"] != "簡介" {
			t.Fatalf("payload = %v", payload)
		}
		avatar, _ := payload["avatarUrl"].(string)
		if !strings.Contains(avatar, "/files/avatar.png?token=") {
			t.Errorf("avatarUrl = %q，應帶 media token", avatar)
		}
		if payload["following"] != true {
			t.Errorf("following = %v，want true", payload["following"])
		}
	})

	t.Run("看自己的頁面不帶 following", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT author_email, nickname, bio, avatar_url FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email", "nickname", "bio", "avatar_url"},
			[]driverValue{covProfileEmail, "自己", "", ""},
		)
		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet,
			"/api/forum/public-profile?key="+strings.Repeat("d", 64), "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if _, ok := payload["following"]; ok {
			t.Errorf("自己的頁面不該帶 following：%v", payload)
		}
		if payload["avatarUrl"] != "" {
			t.Errorf("avatarUrl = %v，want 空字串", payload["avatarUrl"])
		}
	})

	t.Run("頭像 token 失敗仍回資料", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.srv.mediaRedis = nil
		h.rows("SELECT author_email, nickname, bio, avatar_url FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email", "nickname", "bio", "avatar_url"},
			[]driverValue{"target@example.com", "目標", "", "avatar.png"},
		)
		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet,
			"/api/forum/public-profile?key="+strings.Repeat("e", 64), "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if payload["avatarUrl"] != "" {
			t.Errorf("頭像 token 失敗時 avatarUrl = %v，want 空字串", payload["avatarUrl"])
		}
	})

	t.Run("追蹤狀態查詢失敗退化成 false", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email, nickname, bio, avatar_url FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email", "nickname", "bio", "avatar_url"},
			[]driverValue{"target@example.com", "目標", "", ""},
		)
		h.queryError("SELECT COUNT(*) FROM forum_follows WHERE follower_email", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet,
			"/api/forum/public-profile?key="+strings.Repeat("f", 64), "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if payload["following"] != false {
			t.Errorf("following = %v，want false", payload["following"])
		}
	})

	t.Run("資料庫錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_profiles WHERE public_key_hmac = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPublicProfile, http.MethodGet,
			"/api/forum/public-profile?key="+strings.Repeat("1", 64), "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

/* ==========================================================================
   自己的個人資料：GET 與 PUT
   ========================================================================== */

func TestCovForumProfileSelfProfileGet(t *testing.T) {
	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumProfile, http.MethodGet, "/api/forum/profile", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("方法不支援", func(t *testing.T) {
		h := newCovHarness(t)
		for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPatch} {
			recorder := h.do(h.srv.handleForumProfile, method, "/api/forum/profile", "", covProfileEmail, false)
			covWantStatus(t, recorder, http.StatusMethodNotAllowed)
		}
	})

	t.Run("尚未建立資料", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumProfile, http.MethodGet, "/api/forum/profile", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		for _, key := range []string{"nickname", "bio", "avatarUrl"} {
			if payload[key] != "" {
				t.Errorf("%s = %v，want 空字串", key, payload[key])
			}
		}
	})

	t.Run("資料庫錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT nickname, bio, avatar_url FROM forum_profiles WHERE author_email = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleForumProfile, http.MethodGet, "/api/forum/profile", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("有頭像", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT nickname, bio, avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"nickname", "bio", "avatar_url"},
			[]driverValue{"我的暱稱", "我的簡介", "me.png"},
		)
		recorder := h.do(h.srv.handleForumProfile, http.MethodGet, "/api/forum/profile", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		avatar, _ := payload["avatarUrl"].(string)
		if !strings.Contains(avatar, "/files/me.png?token=") {
			t.Errorf("avatarUrl = %q", avatar)
		}
	})

	t.Run("頭像 token 失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.srv.mediaRedis = nil
		h.rows("SELECT nickname, bio, avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"nickname", "bio", "avatar_url"},
			[]driverValue{"我的暱稱", "", "me.png"},
		)
		recorder := h.do(h.srv.handleForumProfile, http.MethodGet, "/api/forum/profile", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if payload["avatarUrl"] != "" {
			t.Errorf("avatarUrl = %v，want 空字串", payload["avatarUrl"])
		}
	})
}

func TestCovForumProfileSelfProfilePutValidation(t *testing.T) {
	t.Run("來源不受信任", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covProfileDoOrigin(h, h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile",
			`{"nickname":"暱稱"}`, covProfileEmail, "https://evil.example")
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("JSON 不合法", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", `{"nickname":`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("暱稱空白", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", `{"nickname":"   "}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "暱稱不可為空") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("暱稱過長", func(t *testing.T) {
		h := newCovHarness(t)
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": strings.Repeat("名", 31)})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("簡介過長", func(t *testing.T) {
		h := newCovHarness(t)
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": "ok", "bio": strings.Repeat("b", 501)})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("頭像不是本站檔案", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": "ok", "avatarUrl": "https://evil.example/files/x.png"})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "頭像必須是上傳到本站的圖片") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("讀取目前頭像失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT avatar_url FROM forum_profiles WHERE author_email = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", `{"nickname":"ok"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("所有權查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.queryError("FROM forum_posts WHERE author_email = ?", errors.New("db down"))
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": "ok", "avatarUrl": "new.png"})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("頭像是別人的檔案", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{"other@example.com"})
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": "ok", "avatarUrl": "new.png"})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
		if !strings.Contains(recorder.Body.String(), "頭像必須是自己上傳的圖片") {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("舊制檔案沒有任何引用", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		// owner 查詢沒有規則 → ErrNoRows，走「舊制檔案」的引用退路；0 筆代表不可掛。
		h.rows("FROM forum_posts WHERE author_email = ?", []string{"count"}, []driverValue{int64(0)})
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": "ok", "avatarUrl": "new.png"})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("暱稱撞名", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_profiles", &mysql.MySQLError{Number: 1062, Message: "duplicate"})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", `{"nickname":"撞名"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusConflict)
	})

	t.Run("寫入失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.execError("INSERT INTO forum_profiles", errors.New("db down"))
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", `{"nickname":"暱稱"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovForumProfileSelfProfilePutSuccess(t *testing.T) {
	t.Run("換頭像並回收舊檔", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		var deleted []string
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deleted = append(deleted, r.URL.Query().Get("url"))
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL

		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"avatar_url"}, []driverValue{"old.png"})
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		// 引用計數是「新契約先查、查到 0 才查舊寫法」的兩段，因此兩段都要登記。
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(0)})
		h.rows("FROM forum_posts WHERE image_url IS NOT NULL", []string{"count"}, []driverValue{int64(0)})

		body := covProfileJSONBody(t, map[string]interface{}{
			"nickname":  " 新暱稱 ",
			"bio":       " 新簡介 ",
			"avatarUrl": covProfilePublicBase + "/files/new.png",
		})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if payload["nickname"] != "新暱稱" || payload["bio"] != "新簡介" {
			t.Errorf("payload = %v，want 修剪後的值", payload)
		}
		if !h.sawQuery("INSERT INTO forum_profiles") {
			t.Error("沒有執行 profile upsert")
		}
		if !h.sawQuery("DELETE FROM forum_media") {
			t.Error("換頭像後沒有刪除舊檔的 ownership 紀錄")
		}
		if len(deleted) != 1 || !strings.Contains(deleted[0], "/files/old.png") {
			t.Errorf("檔案伺服器收到的刪除 = %v，want /files/old.png", deleted)
		}
	})

	t.Run("原樣重送同一個頭像", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"avatar_url"}, []driverValue{"same.png"})
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": "暱稱", "avatarUrl": "same.png"})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		if h.sawQuery("SELECT owner_email FROM forum_media") {
			t.Error("同一個檔名不該再做所有權查詢")
		}
	})

	t.Run("移除頭像但檔案仍被引用", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"avatar_url"}, []driverValue{"old.png"})
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(2)})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", `{"nickname":"暱稱","avatarUrl":""}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		if h.sawQuery("DELETE FROM forum_media") {
			t.Error("仍被引用的檔案不該刪除 ownership 紀錄")
		}
	})

	t.Run("舊制檔案已有自己內容引用", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		// owner 查詢沒有規則 → 走舊制退路；1 筆代表自己的貼文引用過它。
		h.rows("FROM forum_posts WHERE author_email = ?", []string{"count"}, []driverValue{int64(1)})
		body := covProfileJSONBody(t, map[string]interface{}{"nickname": "暱稱", "avatarUrl": "legacy.png"})
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile", body, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery("INSERT INTO forum_profiles") {
			t.Error("沒有執行 profile upsert")
		}
	})

	t.Run("沒有頭像", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumProfile, http.MethodPut, "/api/forum/profile",
			`{"nickname":"只有暱稱"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
	})
}

/* ==========================================================================
   公開貼文與自己的貼文
   ========================================================================== */

func TestCovForumProfilePublicPosts(t *testing.T) {
	const key = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	t.Run("方法不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodPost, "/api/forum/public-posts?user="+key, "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("金鑰長度不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user=short", "", "", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("找不到使用者", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user="+key, "", "", false)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT author_email FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user="+key, "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("offset 不合法", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email"}, []driverValue{"author@example.com"})
		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user="+key+"&offset=-3", "", "", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("成功列出貼文", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email"}, []driverValue{"author@example.com"})
		h.rows("FROM forum_posts fp", covProfilePostCols(), covProfilePostRow(7, "author@example.com", "內容", ""))
		h.rows("FROM forum_user_tag_assignments", []string{"name"}, []driverValue{"標籤"})

		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user="+key, "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		items, _ := payload["items"].([]interface{})
		if len(items) != 1 {
			t.Fatalf("items = %v，want 1 筆", payload["items"])
		}
		if payload["hasMore"] != false {
			t.Errorf("hasMore = %v，want false", payload["hasMore"])
		}
		if !h.sawQuery("fp.author_email = ?") {
			t.Error("沒有以作者 email 過濾")
		}
	})

	t.Run("退路掃描貼文作者", func(t *testing.T) {
		h := newCovHarness(t)
		target := "legacy@example.com"
		key := publicForumKey(target)
		h.rows("SELECT DISTINCT author_email FROM forum_posts",
			[]string{"author_email"},
			[]driverValue{"someone@example.com"},
			[]driverValue{target},
		)
		h.rows("FROM forum_posts fp", covProfilePostCols(), covProfilePostRow(8, target, "舊文", ""))

		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user="+key, "", "", false)
		covWantStatus(t, recorder, http.StatusOK)
		if !h.sawQuery("SELECT DISTINCT author_email FROM forum_posts") {
			t.Error("沒有走 public key 的作者退路掃描")
		}
	})

	t.Run("建立圖片 token 失敗回 502", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.srv.mediaRedis = nil
		h.rows("SELECT author_email FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email"}, []driverValue{"author@example.com"})
		h.rows("FROM forum_posts fp", covProfilePostCols(), covProfilePostRow(9, "author@example.com", "有圖", "pic.png"))
		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user="+key, "", "", false)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})

	t.Run("載入貼文失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT author_email FROM forum_profiles WHERE public_key_hmac = ? OR public_key = ? LIMIT 1",
			[]string{"author_email"}, []driverValue{"author@example.com"})
		h.queryError("FROM forum_posts fp", errors.New("db down"))
		recorder := h.do(h.srv.handleForumPublicPosts, http.MethodGet, "/api/forum/public-posts?user="+key, "", "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

func TestCovForumProfileMyPosts(t *testing.T) {
	t.Run("方法不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodPost, "/api/forum/my-posts", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodGet, "/api/forum/my-posts", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("offset 不合法", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodGet, "/api/forum/my-posts?offset=abc", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("成功列出自己的貼文", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts fp", covProfilePostCols(), covProfilePostRow(11, covProfileEmail, "自己的文", ""))
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodGet, "/api/forum/my-posts", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		items, _ := payload["items"].([]interface{})
		if len(items) != 1 {
			t.Fatalf("items = %v，want 1 筆", payload["items"])
		}
	})

	t.Run("標籤查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts fp", covProfilePostCols(), covProfilePostRow(12, covProfileEmail, "自己的文", ""))
		h.queryError("FROM forum_user_tag_assignments", errors.New("db down"))
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodGet, "/api/forum/my-posts", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("掃描貼文失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_posts fp", []string{"only"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodGet, "/api/forum/my-posts", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("圖片 token 失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.srv.mediaRedis = nil
		h.rows("FROM forum_posts fp", covProfilePostCols(), covProfilePostRow(13, covProfileEmail, "有圖", "pic.png"))
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodGet, "/api/forum/my-posts", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_posts fp", errors.New("db down"))
		recorder := h.do(h.srv.handleForumMyPosts, http.MethodGet, "/api/forum/my-posts", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

/* ==========================================================================
   收藏清單
   ========================================================================== */

func TestCovForumProfileBookmarks(t *testing.T) {
	t.Run("方法不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumBookmarks, http.MethodPost, "/api/forum/bookmarks", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumBookmarks, http.MethodGet, "/api/forum/bookmarks", "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("offset 不合法", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumBookmarks, http.MethodGet, "/api/forum/bookmarks?offset=-1", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("成功列出收藏", func(t *testing.T) {
		h := newCovHarness(t)
		row := covProfilePostRow(21, "someone@example.com", "收藏的文", "")
		row[11] = int64(1) // bookmarked_by_me
		h.rows("FROM forum_posts fp", covProfilePostCols(), row)
		recorder := h.do(h.srv.handleForumBookmarks, http.MethodGet, "/api/forum/bookmarks", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		items, _ := payload["items"].([]interface{})
		if len(items) != 1 {
			t.Fatalf("items = %v，want 1 筆", payload["items"])
		}
		if !strings.Contains(recorder.Body.String(), `"bookmarked":true`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
		if !h.sawQuery("FROM forum_post_bookmarks") {
			t.Error("沒有以收藏表過濾")
		}
	})

	t.Run("查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_posts fp", errors.New("db down"))
		recorder := h.do(h.srv.handleForumBookmarks, http.MethodGet, "/api/forum/bookmarks", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("圖片 token 失敗", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.srv.mediaRedis = nil
		h.rows("FROM forum_posts fp", covProfilePostCols(), covProfilePostRow(22, "someone@example.com", "有圖", "pic.png"))
		recorder := h.do(h.srv.handleForumBookmarks, http.MethodGet, "/api/forum/bookmarks", "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})
}

/* ==========================================================================
   作者名稱與頭像的輔助函式
   ========================================================================== */

func TestCovForumProfileAuthorHelpers(t *testing.T) {
	t.Run("publicForumKey 穩定且為 64 位十六進位", func(t *testing.T) {
		key := publicForumKey("someone@example.com")
		if len(key) != 64 {
			t.Fatalf("len(publicForumKey) = %d，want 64", len(key))
		}
		if _, err := hex.DecodeString(key); err != nil {
			t.Fatalf("publicForumKey 不是十六進位：%v", err)
		}
		sum := sha256.Sum256([]byte("someone@example.com"))
		if key != hex.EncodeToString(sum[:]) {
			t.Errorf("publicForumKey = %q，want SHA-256", key)
		}
		if key != publicForumKey("someone@example.com") {
			t.Error("同一個 email 兩次呼叫得到不同 key")
		}
	})

	t.Run("publicForumAuthor 形狀", func(t *testing.T) {
		author := publicForumAuthor("someone@example.com")
		if !strings.HasPrefix(author, "匿名") {
			t.Errorf("publicForumAuthor = %q，want 匿名前綴", author)
		}
		if utf8.RuneCountInString(author) != utf8.RuneCountInString("匿名")+6 {
			t.Errorf("publicForumAuthor = %q，want 六位數字", author)
		}
	})

	t.Run("forumAuthor 使用暱稱", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT nickname FROM forum_profiles WHERE author_email = ?",
			[]string{"nickname"}, []driverValue{"暱稱"})
		if got := h.srv.forumAuthor(h.request(http.MethodGet, "/", "", "", false), "x@example.com"); got != "暱稱" {
			t.Errorf("forumAuthor = %q，want 暱稱", got)
		}
	})

	t.Run("forumAuthor 空白暱稱退成匿名", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT nickname FROM forum_profiles WHERE author_email = ?",
			[]string{"nickname"}, []driverValue{"   "})
		if got := h.srv.forumAuthor(h.request(http.MethodGet, "/", "", "", false), "x@example.com"); !strings.HasPrefix(got, "匿名") {
			t.Errorf("forumAuthor = %q，want 匿名代號", got)
		}
	})

	t.Run("forumAuthor 查無資料退成匿名", func(t *testing.T) {
		h := newCovHarness(t)
		if got := h.srv.forumAuthor(h.request(http.MethodGet, "/", "", "", false), "x@example.com"); !strings.HasPrefix(got, "匿名") {
			t.Errorf("forumAuthor = %q，want 匿名代號", got)
		}
	})

	t.Run("forumAuthor 資料庫錯誤退成匿名", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT nickname FROM forum_profiles WHERE author_email = ?", errors.New("db down"))
		if got := h.srv.forumAuthor(h.request(http.MethodGet, "/", "", "", false), "x@example.com"); !strings.HasPrefix(got, "匿名") {
			t.Errorf("forumAuthor = %q，want 匿名代號", got)
		}
	})

	t.Run("forumAuthorTags 成功", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_user_tag_assignments", []string{"name"},
			[]driverValue{"alpha"}, []driverValue{"beta"})
		tags, err := h.srv.forumAuthorTags(context.Background(), "x@example.com")
		if err != nil {
			t.Fatalf("forumAuthorTags: %v", err)
		}
		if len(tags) != 2 || tags[0] != "alpha" || tags[1] != "beta" {
			t.Fatalf("tags = %v", tags)
		}
	})

	t.Run("forumAuthorTags 查詢錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("FROM forum_user_tag_assignments", errors.New("db down"))
		if _, err := h.srv.forumAuthorTags(context.Background(), "x@example.com"); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("forumAuthorTags 掃描錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("FROM forum_user_tag_assignments", []string{"name", "extra"},
			[]driverValue{"a", "b"})
		if _, err := h.srv.forumAuthorTags(context.Background(), "x@example.com"); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("forumAuthorAvatarURL 查無資料", func(t *testing.T) {
		h := newCovHarness(t)
		url, token, err := h.srv.forumAuthorAvatarURL(context.Background(), "x@example.com", "carry")
		if err != nil || url != "" || token != "carry" {
			t.Fatalf("url=%q token=%q err=%v", url, token, err)
		}
	})

	t.Run("forumAuthorAvatarURL 資料庫錯誤", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT avatar_url FROM forum_profiles WHERE author_email = ?", errors.New("db down"))
		if _, _, err := h.srv.forumAuthorAvatarURL(context.Background(), "x@example.com", ""); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("forumAuthorAvatarURL 沿用既有 token", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"avatar_url"}, []driverValue{"a.png"})
		url, token, err := h.srv.forumAuthorAvatarURL(context.Background(), "x@example.com", "carry")
		if err != nil {
			t.Fatalf("forumAuthorAvatarURL: %v", err)
		}
		if token != "carry" || !strings.Contains(url, "?token=carry") {
			t.Fatalf("url=%q token=%q", url, token)
		}
	})

	t.Run("forumAuthorAvatarURL 自行簽發 token", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"avatar_url"}, []driverValue{"a.png"})
		url, token, err := h.srv.forumAuthorAvatarURL(context.Background(), "x@example.com", "")
		if err != nil {
			t.Fatalf("forumAuthorAvatarURL: %v", err)
		}
		if len(token) != 64 || !strings.Contains(url, "?token="+token) {
			t.Fatalf("url=%q token=%q", url, token)
		}
	})

	t.Run("forumAuthorAvatarURL 無法辨識的檔名", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"avatar_url"}, []driverValue{"not-an-image"})
		url, token, err := h.srv.forumAuthorAvatarURL(context.Background(), "x@example.com", "carry")
		if err != nil || url != "" || token != "carry" {
			t.Fatalf("url=%q token=%q err=%v", url, token, err)
		}
	})

	t.Run("forumAuthorAvatarURL 無法簽發 token", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.srv.mediaRedis = nil
		h.rows("SELECT avatar_url FROM forum_profiles WHERE author_email = ?",
			[]string{"avatar_url"}, []driverValue{"a.png"})
		if _, _, err := h.srv.forumAuthorAvatarURL(context.Background(), "x@example.com", ""); err == nil {
			t.Fatal("want error")
		}
	})
}

/* ==========================================================================
   圖片檔名、主機與網址
   ========================================================================== */

func TestCovForumProfileMediaHelpers(t *testing.T) {
	t.Run("forumImageFileName 各種寫法", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		cases := []struct{ in, want string }{
			{"", ""},
			{"  pic.png  ", "pic.png"},
			{"pic.jpeg", "pic.jpeg"},
			{"pic.svg", ""},
			{".hidden.png", ""},
			{"a b.png", ""},
			{"sub/pic.png", ""},
			{"/files/pic.png", "pic.png"},
			{"/files/pic.png?token=x", "pic.png"},
			{"/files/sub/pic.png", ""},
			{"/files/../pic.png", ""},
			{"/other/pic.png", ""},
			{covProfilePublicBase + "/files/pic.png", "pic.png"},
			{covProfileFilesBase + "/files/pic.png", "pic.png"},
			{"http://localhost:7070/files/pic.png", "pic.png"},
			{"http://127.0.0.1:7070/files/pic.png", "pic.png"},
			{"http://[::1]:7070/files/pic.png", "pic.png"},
			{"https://evil.example/files/pic.png", ""},
			{"http://[::1", ""},
			{"/files/pic.PNG", ""},
			{"/files/" + strings.Repeat("a", 129) + ".png", ""},
		}
		for _, tc := range cases {
			if got := h.srv.forumImageFileName(tc.in); got != tc.want {
				t.Errorf("forumImageFileName(%q) = %q，want %q", tc.in, got, tc.want)
			}
		}
	})

	t.Run("validForumImageFileName", func(t *testing.T) {
		cases := []struct{ in, want string }{
			{"", ""},
			{".png", ""},
			{strings.Repeat("a", 129) + ".png", ""},
			{"ok.png", "ok.png"},
			{"ok.jpg", "ok.jpg"},
			{"ok.jpeg", "ok.jpeg"},
			{"ok.gif", "ok.gif"},
			{"ok.webp", "ok.webp"},
			{"bad!.png", ""},
			{"ok.PNG", ""},
		}
		for _, tc := range cases {
			if got := validForumImageFileName(tc.in); got != tc.want {
				t.Errorf("validForumImageFileName(%q) = %q，want %q", tc.in, got, tc.want)
			}
		}
	})

	t.Run("isOwnFilesHost", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		for _, host := range []string{"localhost", "127.0.0.1", "::1", "cdn.cov.test", "files.cov.internal"} {
			if !h.srv.isOwnFilesHost(host) {
				t.Errorf("isOwnFilesHost(%q) = false，want true", host)
			}
		}
		if h.srv.isOwnFilesHost("evil.example") {
			t.Error("isOwnFilesHost(evil.example) = true，want false")
		}

		broken := newCovHarness(t)
		broken.srv.cfg.FilesServerPublicURL = "http://[bad"
		broken.srv.cfg.FilesServerURL = ""
		if broken.srv.isOwnFilesHost("cdn.cov.test") {
			t.Error("設定值無法解析時不該認得任何主機")
		}
	})

	t.Run("forumImageURL", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)

		if got := h.srv.forumImageURL("", "tok"); got != "" {
			t.Errorf("空檔名 = %q，want 空字串", got)
		}
		if got := h.srv.forumImageURL("bad.svg", "tok"); got != "" {
			t.Errorf("不允許的副檔名 = %q，want 空字串", got)
		}
		if got := h.srv.forumImageURL("pic.png", ""); got != covProfilePublicBase+"/files/pic.png" {
			t.Errorf("forumImageURL = %q", got)
		}
		if got := h.srv.forumImageURL("pic.png", "tok"); got != covProfilePublicBase+"/files/pic.png?token=tok" {
			t.Errorf("forumImageURL = %q", got)
		}

		h.srv.cfg.FilesServerPublicURL = covProfilePublicBase + "/media/"
		if got := h.srv.forumImageURL("pic.png", ""); got != covProfilePublicBase+"/media/files/pic.png" {
			t.Errorf("帶子路徑的 forumImageURL = %q", got)
		}

		h.srv.cfg.FilesServerPublicURL = ""
		if got := h.srv.forumImageURL("pic.png", "tok"); got != "" {
			t.Errorf("公開位址為空時 = %q，want 空字串", got)
		}
		h.srv.cfg.FilesServerPublicURL = "cdn.cov.test"
		if got := h.srv.forumImageURL("pic.png", "tok"); got != "" {
			t.Errorf("公開位址沒有 scheme 時 = %q，want 空字串", got)
		}
		h.srv.cfg.FilesServerPublicURL = "http://[bad"
		if got := h.srv.forumImageURL("pic.png", "tok"); got != "" {
			t.Errorf("公開位址無法解析時 = %q，want 空字串", got)
		}
	})

	t.Run("DeleteFile 包裝 deleteForumFile", func(t *testing.T) {
		h := newCovHarness(t)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("方法 = %s，want DELETE", r.Method)
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		if err := h.srv.DeleteFile(context.Background(), "pic.png"); err != nil {
			t.Fatalf("DeleteFile(404) = %v，want nil", err)
		}
		if err := h.srv.DeleteFile(context.Background(), "bad.svg"); err == nil {
			t.Fatal("DeleteFile(非法檔名) = nil，want error")
		}
	})

	t.Run("releaseForumImage 的三條早退路徑", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.releaseForumImage(context.Background(), "", "owner")

		h.queryError("FROM forum_posts WHERE JSON_VALID(image_url)", errors.New("db down"))
		h.srv.releaseForumImage(context.Background(), "pic.png", "owner")

		h2 := newCovHarness(t)
		h2.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(1)})
		h2.srv.releaseForumImage(context.Background(), "pic.png", "owner")
	})
}

/* ==========================================================================
   媒體 token 簽發與釋放
   ========================================================================== */

func TestCovForumProfileMediaToken(t *testing.T) {
	t.Run("成功簽發並寫入 Redis", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		token, err := h.srv.createMediaToken(context.Background())
		if err != nil {
			t.Fatalf("createMediaToken: %v", err)
		}
		if len(token) != 64 {
			t.Fatalf("len(token) = %d，want 64", len(token))
		}
		if _, err := hex.DecodeString(token); err != nil {
			t.Fatalf("token 不是十六進位：%v", err)
		}
		exists, err := h.client.Exists(context.Background(), covProfileKeyPrefix+token).Result()
		if err != nil || exists != 1 {
			t.Fatalf("Redis key exists = %d, err = %v", exists, err)
		}
		ttl, err := h.client.TTL(context.Background(), covProfileKeyPrefix+token).Result()
		if err != nil || ttl <= 0 || ttl > time.Minute {
			t.Fatalf("TTL = %v, err = %v", ttl, err)
		}
	})

	t.Run("沒有 mediaRedis", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.mediaRedis = nil
		if _, err := h.srv.createMediaToken(context.Background()); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestCovForumProfileImageTokensRelease(t *testing.T) {
	const path = "/api/forum/image-tokens/release"

	t.Run("方法不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageTokensRelease, http.MethodGet, path, "", "", false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("來源不受信任", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covProfileDoOrigin(h, h.srv.handleForumImageTokensRelease, http.MethodPost, path, `{"tokens":[]}`, "", "https://evil.example")
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("沒有 mediaRedis", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.mediaRedis = nil
		recorder := h.do(h.srv.handleForumImageTokensRelease, http.MethodPost, path, `{"tokens":[]}`, "", false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("JSON 不合法", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageTokensRelease, http.MethodPost, path, `{`, "", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("body 過大", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageTokensRelease, http.MethodPost, path, strings.Repeat("a", 70<<10), "", false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("格式不合的 token 全部略過", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		body := covProfileJSONBody(t, map[string]interface{}{"tokens": []string{
			"short",
			strings.Repeat("A", 64),       // 大寫不在允許字元內
			strings.Repeat("a", 63) + "z", // 非十六進位
			strings.Repeat("a", 65),       // 長度不對
		}})
		recorder := h.do(h.srv.handleForumImageTokensRelease, http.MethodPost, path, body, "", false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if payload["deleted"] != float64(0) {
			t.Errorf("deleted = %v，want 0", payload["deleted"])
		}
	})

	t.Run("刪除合法 token 與略過非法 token", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		token, err := h.srv.createMediaToken(context.Background())
		if err != nil {
			t.Fatalf("createMediaToken: %v", err)
		}
		body := covProfileJSONBody(t, map[string]interface{}{"tokens": []string{"bad", token}})
		recorder := h.do(h.srv.handleForumImageTokensRelease, http.MethodPost, path, body, "", false)
		covWantStatus(t, recorder, http.StatusOK)
		payload := covProfileJSONMap(t, recorder)
		if payload["deleted"] != float64(1) {
			t.Errorf("deleted = %v，want 1", payload["deleted"])
		}
		exists, err := h.client.Exists(context.Background(), covProfileKeyPrefix+token).Result()
		if err != nil || exists != 0 {
			t.Fatalf("釋放後 key exists = %d, err = %v", exists, err)
		}
	})

	t.Run("Redis 刪除失敗回 502", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		token, err := h.srv.createMediaToken(context.Background())
		if err != nil {
			t.Fatalf("createMediaToken: %v", err)
		}
		if err := h.client.Close(); err != nil {
			t.Fatalf("關閉 Redis client: %v", err)
		}
		body := covProfileJSONBody(t, map[string]interface{}{"tokens": []string{token}})
		recorder := h.do(h.srv.handleForumImageTokensRelease, http.MethodPost, path, body, "", false)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})
}

/* ==========================================================================
   圖片上傳
   ========================================================================== */

func TestCovForumProfileImageUpload(t *testing.T) {
	const path = "/api/forum/images"

	t.Run("方法不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageUpload, http.MethodGet, path, "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageUpload, http.MethodPost, path, "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("來源不受信任", func(t *testing.T) {
		h := newCovHarness(t)
		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), "https://evil.example", covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("不是 multipart", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageUpload, http.MethodPost, path, "not-multipart", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("缺少 file 欄位", func(t *testing.T) {
		h := newCovHarness(t)
		req := covProfileUploadRequest(t, h, "other", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("成功上傳", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		var gotToken string
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/upload" {
				t.Errorf("路徑 = %s，want /upload", r.URL.Path)
			}
			gotToken = r.Header.Get("X-Upload-Token")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"url":"/files/uploaded.png"}`))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNGDATA"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusCreated)
		payload := covProfileJSONMap(t, recorder)
		url, _ := payload["url"].(string)
		if !strings.Contains(url, "/files/uploaded.png?token=") {
			t.Errorf("url = %q", url)
		}
		if gotToken != "" {
			t.Errorf("未設定 token 卻送出 X-Upload-Token = %q", gotToken)
		}
		if !h.sawQuery("INSERT INTO forum_media") {
			t.Error("沒有寫入 forum_media 所有權紀錄")
		}
	})

	t.Run("轉送時帶上共用密鑰", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		var gotToken string
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotToken = r.Header.Get("X-Upload-Token")
			_, _ = w.Write([]byte(`{"url":"/files/secret.png"}`))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		h.srv.cfg.FilesServerToken = "shared-secret"

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusCreated)
		if gotToken != "shared-secret" {
			t.Errorf("X-Upload-Token = %q，want shared-secret", gotToken)
		}
	})

	t.Run("檔案伺服器回應 500", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})

	t.Run("檔案伺服器回應不是 JSON", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("檔案伺服器回應空網址", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"url":""}`))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("檔案伺服器回應別人的網址", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"url":"https://evil.example/files/uploaded.png"}`))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("檔案伺服器回應不允許的副檔名", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"url":"/files/uploaded.svg"}`))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("建立 token 失敗就丟棄上傳檔案", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		var deletes int
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deletes++
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_, _ = w.Write([]byte(`{"url":"/files/tokenless.png"}`))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		h.srv.mediaRedis = nil

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusBadGateway)
		if deletes != 1 {
			t.Errorf("丟棄檔案的 DELETE 次數 = %d，want 1", deletes)
		}
	})

	t.Run("寫入所有權紀錄失敗就丟棄上傳檔案", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		var deletes int
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deletes++
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_, _ = w.Write([]byte(`{"url":"/files/recordless.png"}`))
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		h.execError("INSERT INTO forum_media", errors.New("db down"))

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusInternalServerError)
		if deletes != 1 {
			t.Errorf("丟棄檔案的 DELETE 次數 = %d，want 1", deletes)
		}
	})

	t.Run("檔案伺服器位址無法建立請求", func(t *testing.T) {
		h := newCovHarness(t)
		covProfileConfigureMedia(h)
		h.srv.cfg.FilesServerURL = "://bad"

		req := covProfileUploadRequest(t, h, "file", "photo.png", []byte("PNG"), covOrigin, covProfileEmail)
		recorder := httptest.NewRecorder()
		h.srv.handleForumImageUpload(recorder, req)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})
}

/* ==========================================================================
   圖片刪除
   ========================================================================== */

func TestCovForumProfileImageDelete(t *testing.T) {
	const path = "/api/forum/images/delete"

	t.Run("方法不對", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodGet, path, "", covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusMethodNotAllowed)
	})

	t.Run("未登入", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, "", "", false)
		covWantStatus(t, recorder, http.StatusUnauthorized)
	})

	t.Run("來源不受信任", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := covProfileDoOrigin(h, h.srv.handleForumImageDelete, http.MethodPost, path,
			`{"url":"pic.png"}`, covProfileEmail, "https://evil.example")
		covWantStatus(t, recorder, http.StatusForbidden)
	})

	t.Run("JSON 不合法", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("body 過大", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, strings.Repeat("a", 5000), covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("網址不是本站檔案", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"not a file"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadRequest)
	})

	t.Run("讀取擁有者失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.queryError("SELECT owner_email FROM forum_media WHERE filename = ?", errors.New("db down"))
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("沒有所有權紀錄", func(t *testing.T) {
		h := newCovHarness(t)
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("擁有者是別人", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{"other@example.com"})
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusNotFound)
	})

	t.Run("引用計數查詢失敗", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		h.queryError("FROM forum_posts WHERE JSON_VALID(image_url)", errors.New("db down"))
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("檔案仍被引用", func(t *testing.T) {
		h := newCovHarness(t)
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(1)})
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusConflict)
	})

	t.Run("檔案伺服器刪除失敗", func(t *testing.T) {
		h := newCovHarness(t)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		// 引用計數是「新契約先查、查到 0 才查舊寫法」的兩段，因此兩段都要登記。
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(0)})
		h.rows("FROM forum_posts WHERE image_url IS NOT NULL", []string{"count"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})

	t.Run("刪除資料列失敗", func(t *testing.T) {
		h := newCovHarness(t)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		// 引用計數是「新契約先查、查到 0 才查舊寫法」的兩段，因此兩段都要登記。
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(0)})
		h.rows("FROM forum_posts WHERE image_url IS NOT NULL", []string{"count"}, []driverValue{int64(0)})
		h.execError("DELETE FROM forum_media", errors.New("db down"))
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusInternalServerError)
	})

	t.Run("檔案伺服器位址無法建立請求", func(t *testing.T) {
		h := newCovHarness(t)
		h.srv.cfg.FilesServerURL = "://bad"
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		// 引用計數是「新契約先查、查到 0 才查舊寫法」的兩段，因此兩段都要登記。
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(0)})
		h.rows("FROM forum_posts WHERE image_url IS NOT NULL", []string{"count"}, []driverValue{int64(0)})
		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusBadGateway)
	})

	t.Run("成功刪除", func(t *testing.T) {
		h := newCovHarness(t)
		var gotURL string
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("方法 = %s，want DELETE", r.Method)
			}
			gotURL = r.URL.Query().Get("url")
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		// 引用計數是「新契約先查、查到 0 才查舊寫法」的兩段，因此兩段都要登記。
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(0)})
		h.rows("FROM forum_posts WHERE image_url IS NOT NULL", []string{"count"}, []driverValue{int64(0)})

		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
		if gotURL != "/files/pic.png" {
			t.Errorf("檔案伺服器收到 url = %q，want /files/pic.png", gotURL)
		}
		if !h.sawQuery("DELETE FROM forum_media") {
			t.Error("沒有刪除 forum_media 紀錄")
		}
		payload := covProfileJSONMap(t, recorder)
		if payload["ok"] != true {
			t.Errorf("payload = %v", payload)
		}
	})

	t.Run("檔案已不存在視為成功", func(t *testing.T) {
		h := newCovHarness(t)
		files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(files.Close)
		h.srv.cfg.FilesServerURL = files.URL
		h.rows("SELECT owner_email FROM forum_media WHERE filename = ?",
			[]string{"owner_email"}, []driverValue{covProfileEmail})
		// 引用計數是「新契約先查、查到 0 才查舊寫法」的兩段，因此兩段都要登記。
		h.rows("FROM forum_posts WHERE JSON_VALID(image_url)", []string{"count"}, []driverValue{int64(0)})
		h.rows("FROM forum_posts WHERE image_url IS NOT NULL", []string{"count"}, []driverValue{int64(0)})

		recorder := h.do(h.srv.handleForumImageDelete, http.MethodPost, path, `{"url":"pic.png"}`, covProfileEmail, false)
		covWantStatus(t, recorder, http.StatusOK)
	})
}
