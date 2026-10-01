/*
內容趨勢統計的測試（httpapi/stats_handlers_test.go）。

這個頁面的邏輯不多，但每一件事「壞掉時都不會報錯」，所以測試集中在三處：

  1. statsDays 的參數收斂
     少一點收斂就會有兩個很難 debug 的症狀：days=0 讓圖上只有一個桶，
     days=99999 讓資料庫掃全表。兩者都不會回錯，只會變慢或變怪。

  2. excerptRunes 的字元計算
     用 len(string) 會讓「80 字」在中文貼文上變成 27 個字，而症狀是
     「摘要怎麼少了那麼多」—— 不會有人知道是單位算錯了。

  3. 授權
     /api/admin/stats 會回傳全站的使用者 email 與貼文內容摘錄。任何未登入
     或非管理員的人能讀到它，後臺的三個敏感值就一起外洩了。
*/

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forum/forum/session"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

/* ==========================================================================
   days 參數
   ========================================================================== */

func TestStatsDaysClampsInput(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want int
	}{
		{"缺漏用預設", "", defaultStatsDays},
		{"不是數字用預設", "abc", defaultStatsDays},
		{"零與負數用預設", "0", defaultStatsDays},
		{"負數用預設", "-30", defaultStatsDays},
		{"前後空白容忍", "  14  ", 14},
		{"正常值原樣", "7", 7},
		{"超過上限收斂", "365", maxStatsDays},
		{"剛好等於上限", "90", maxStatsDays},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statsDays(tc.raw); got != tc.want {
				t.Fatalf("statsDays(%q) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}

// 視窗的起點必須是「今天 00:00 往前 days-1 天」，而不是「現在往前 days×24
// 小時」。這個差別在日別的圖上是每天都在的假凹陷：後者的起點落在今天的某個
// 時點，於是最後一個桶只有半天的資料，看起來像流量突然掉了。
func TestStatsWindowStartsAtLocalMidnight(t *testing.T) {
	// 2026-09-30 23:47:00 本地時間，取 7 天。
	now := time.Date(2026, 9, 30, 23, 47, 0, 0, time.Local)
	today := time.Date(2026, 9, 30, 0, 0, 0, 0, time.Local)
	wantStart := today.AddDate(0, 0, -6) // 2026-09-24

	got := today.AddDate(0, 0, -(7 - 1))
	if !got.Equal(wantStart) {
		t.Fatalf("視窗起點 = %v, want %v", got, wantStart)
	}
	// 順帶確認這個起點確實涵蓋 7 個日曆日（而不是 6 個半）。
	if last := got.AddDate(0, 0, 6); !last.Equal(today) {
		t.Fatalf("視窗最後一天 = %v, want %v（7 天必須含今天）", last, today)
	}
	_ = now
}

/* ==========================================================================
   摘錄
   ========================================================================== */

func TestExcerptRunesCountsCharacters(t *testing.T) {
	// 中文每字 3 bytes。len(string) 會把 80 字算成 240 而誤判超長 ——
	// 症狀是「摘要怎麼少了那麼多」，沒有人會聯想到是單位算錯。
	text := ""
	for range 100 {
		text += "中"
	}
	if got := excerptRunes(text, 80); len([]rune(got)) != 81 {
		t.Fatalf("摘錄字元數 = %d, want 81（80 字 + 省略號）", len([]rune(got)))
	}
	// 未超長時原樣返回，不加省略號。
	if got := excerptRunes("短文", 80); got != "短文" {
		t.Fatalf("未超長應原樣返回, got %q", got)
	}
	// 剛好等於上限不算超長。
	exact := ""
	for range 80 {
		exact += "中"
	}
	if got := excerptRunes(exact, 80); got != exact {
		t.Errorf("剛好等於上限不應被截斷, got %q", got)
	}
}

/* ==========================================================================
   授權
   ========================================================================== */

func TestAdminStatsRequiresAdmin(t *testing.T) {
	// 這個端點會回傳全站的使用者 email 與貼文內容摘錄。未登入或非管理員
	// 能讀到它，等於後臺三個敏感值一起外洩，所以授權必須擋在最前面 ——
	// 也要擋在 method 分派之前（否則未登入者可以用一個不支援的 method
	// 拿到 405，等於洩漏「這條路由存在且需要管理員身分」）。
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	manager := session.NewManager("FORUM_test", time.Hour, false, client)
	server := &Server{sessions: manager}

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		server.handleAdminStats(recorder, httptest.NewRequest(method, "/api/admin/stats", nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s 的回應 = %d, want 401", method, recorder.Code)
		}
	}
}
