/*
checkConfig 的分支測試（backend/main_config_report_test.go）。

-check 的價值在於「設定寫錯了但服務不會報錯」的那些組合。這些組合的共通
特徵是：每一個都只在報告的某一、兩行裡出現，而且彼此不影響 —— 因此測試
必須一個一個把設定檔寫成那個形狀，再斷言對應的那一行真的印出來了。

這裡刻意不釘住報告全文：文案會隨維運需求調整，釘全文只會讓每次改字都
變成三處修改。斷言的是「這個分支的結論句」與「憑證不得出現在輸出裡」。

唯一刻意沒有覆蓋的分支是「TRUSTED_ORIGINS 為空」的警告：config.Load 的
applyDefaults 一定會把空清單補成 PublicBaseURL，所以透過真實設定檔無法
製造出那個狀態。留著它是為了讓 checkConfig 在未來改成不吃 applyDefaults
時仍然正確，而不是死碼。
*/
package main

import (
	"strings"
	"testing"
)

// TestCheckConfigReportsProductionRisks 用一份「正式環境但多項沒設好」的
// 設定檔，一次走完所有只在正式環境出現的警告與錯誤。
func TestCheckConfigReportsProductionRisks(t *testing.T) {
	// applyDefaults 會在 FILES_SERVER_TOKEN 為空時讀同名環境變數；不先清掉
	// 的話，「正式環境未設定 token」這條警告會隨執行機器的環境而消失。
	t.Setenv("FILES_SERVER_TOKEN", "")

	cfgPath := writeConfigFile(t, strings.Join([]string{
		"COOKIE_SECURE=true",
		"PUBLIC_BASE_URL=https://forum.example.com",
		"SERVER_PORT=:9443",
		"GOOGLE_CLIENT_ID=client-id",
		"GOOGLE_CLIENT_SECRET=client-secret",
		"FILES_SERVER_TOKEN=",
		"MEDIA_TOKEN_KEY_PREFIX=forum:session:media",
		"ALLOWED_ADMIN_EMAIL=b@example.com, A@Example.com",
		"ES_URL=http://es.internal:9200",
		"ES_USERNAME=elastic",
		"ES_PASSWORD=es-secret",
		"LOG_FILE=logs/forum.log",
	}, "\n")+"\n")

	out := captureStdout(t, func() {
		err := checkConfig(cfgPath)
		if err == nil {
			t.Error("正式環境採用 30 天兜底 TTL，checkConfig 應回錯誤（exit 1）")
		}
	})

	wants := []string{
		"未設定 TRUSTED_PROXY_CIDRS",    // 空代理白名單
		"有 client id 卻沒有回呼網址",        // GOOGLE_CLIENT_ID 有、REDIRECT 無
		"正式環境未設定 FILES_SERVER_TOKEN", // 上傳端點會接受任何來源
		"正式環境不接受兜底值",                 // Validate 的結論
		`不得含 "session"`,              // 媒體 token 前綴與 session 重疊
		"結論：不通過",
		"驗證 basic", // ES_URL 有值時一併報驗證種類
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("報告缺少 %q:\n%s", want, out)
		}
	}

	// 白名單已正規化（去空白、轉小寫）並排序；報告要印的是「真的存進去的值」。
	if !strings.Contains(out, "a@example.com") || !strings.Contains(out, "b@example.com") {
		t.Errorf("報告沒有列出正規化後的完整白名單:\n%s", out)
	}
	if strings.Index(out, "a@example.com") > strings.Index(out, "b@example.com") {
		t.Errorf("白名單應排序後輸出:\n%s", out)
	}

	// 報告會出現在 CI 日誌與 issue 裡，因此任何憑證都只能報「有設定」。
	for _, secret := range []string{"client-secret", "es-secret"} {
		if strings.Contains(out, secret) {
			t.Errorf("報告洩漏了憑證 %q:\n%s", secret, out)
		}
	}
}

// TestCheckConfigWarnsProxyAndInsecureCookieWithoutProductionFlags 釘住兩條
// 「跟環境判斷無關」的警告。它們原本都掛在 IsProduction() 上，而那個判斷的依據
// 是 COOKIE_SECURE 與 PUBLIC_BASE_URL —— 一個實際被代理接到公網、卻把這兩個值
// 留成開發預設的部署（2026-10-27 線上那份就是這個形狀）會被判成非正式環境，
// 於是那兩行提示完全不出現，而它們恰恰是那個部署上真正仍在生效的暴露。
func TestCheckConfigWarnsProxyAndInsecureCookieWithoutProductionFlags(t *testing.T) {
	t.Setenv("FILES_SERVER_TOKEN", "")

	t.Run("網址寫著 http 也照樣警告代理白名單", func(t *testing.T) {
		cfgPath := writeConfigFile(t, strings.Join([]string{
			"COOKIE_SECURE=false",
			"PUBLIC_BASE_URL=http://forum.internal:8088",
			"SERVER_PORT=:8088",
			"MEDIA_TOKEN_TTL_SECONDS=600",
		}, "\n")+"\n")

		out := captureStdout(t, func() { _ = checkConfig(cfgPath) })
		if !strings.Contains(out, "未設定 TRUSTED_PROXY_CIDRS") {
			t.Errorf("非正式環境的報告沒有警告代理白名單:\n%s", out)
		}
	})

	t.Run("對外寫著 HTTPS 而 cookie 沒標 Secure → 警告", func(t *testing.T) {
		cfgPath := writeConfigFile(t, strings.Join([]string{
			"COOKIE_SECURE=false",
			"PUBLIC_BASE_URL=https://forum.example.com",
			"SERVER_PORT=:8088",
			"MEDIA_TOKEN_TTL_SECONDS=600",
			"TRUSTED_PROXY_CIDRS=127.0.0.1/32",
		}, "\n")+"\n")

		out := captureStdout(t, func() { _ = checkConfig(cfgPath) })
		for _, want := range []string{"COOKIE_SECURE=false", "Strict-Transport-Security"} {
			if !strings.Contains(out, want) {
				t.Errorf("報告缺少 %q:\n%s", want, out)
			}
		}
	})

	t.Run("代理白名單已設定時不警告", func(t *testing.T) {
		cfgPath := writeConfigFile(t, strings.Join([]string{
			"COOKIE_SECURE=true",
			"PUBLIC_BASE_URL=https://forum.example.com",
			"TRUSTED_PROXY_CIDRS=192.168.66.7/32",
			"MEDIA_TOKEN_TTL_SECONDS=600",
		}, "\n")+"\n")

		out := captureStdout(t, func() { _ = checkConfig(cfgPath) })
		if strings.Contains(out, "未設定 TRUSTED_PROXY_CIDRS") {
			t.Errorf("已設定白名單卻仍警告未設定:\n%s", out)
		}
		if strings.Contains(out, "不會送 Strict-Transport-Security") {
			t.Errorf("COOKIE_SECURE=true 時不該警告 HSTS:\n%s", out)
		}
	})
}

// TestCheckConfigReportsAuthorKeySecret 釘住 F-03 那行警告：對外的作者識別碼
// 能不能被離線還原，取決於 AUTHOR_KEY_SECRET 有沒有設定。
//
// 它刻意不進 Validate（本機開發與尚未導入秘密的部署都還在舊形態，讓啟動失敗
// 會把一個「部署者要做的決定」變成「這支行程跑不了」）。因此唯一讓它被看見的
// 地方就是 `-check` 的報告 —— 那一列不在，這個暴露就沒有人知道。
func TestCheckConfigReportsAuthorKeySecret(t *testing.T) {
	t.Setenv("FILES_SERVER_TOKEN", "")

	t.Run("未設定時提出警告", func(t *testing.T) {
		cfgPath := writeConfigFile(t, strings.Join([]string{
			"COOKIE_SECURE=true",
			"PUBLIC_BASE_URL=https://forum.example.com",
			"SERVER_PORT=:9443",
			"MEDIA_TOKEN_TTL_SECONDS=600",
		}, "\n")+"\n")

		out := captureStdout(t, func() { _ = checkConfig(cfgPath) })
		for _, want := range []string{"未設定 AUTHOR_KEY_SECRET", "SHA-256(email)"} {
			if !strings.Contains(out, want) {
				t.Errorf("報告缺少 %q:\n%s", want, out)
			}
		}
		if strings.Contains(out, "AUTHOR_KEY_SECRET · 已設定") {
			t.Errorf("未設定時憑證列不應報已設定:\n%s", out)
		}
		if !strings.Contains(out, "AUTHOR_KEY_SECRET · 未設定") {
			t.Errorf("憑證列沒有報未設定:\n%s", out)
		}
	})

	// 設定值本身不能進報告：它出現在 CI 日誌與 issue 裡就等于洩漏。
	t.Run("已設定時不警告且不洩漏內容", func(t *testing.T) {
		cfgPath := writeConfigFile(t, strings.Join([]string{
			"COOKIE_SECURE=true",
			"PUBLIC_BASE_URL=https://forum.example.com",
			"SERVER_PORT=:9443",
			"MEDIA_TOKEN_TTL_SECONDS=600",
			"AUTHOR_KEY_SECRET=author-key-secret-value",
		}, "\n")+"\n")

		out := captureStdout(t, func() { _ = checkConfig(cfgPath) })
		if strings.Contains(out, "未設定 AUTHOR_KEY_SECRET") {
			t.Errorf("已設定卻仍警告:\n%s", out)
		}
		if strings.Contains(out, "author-key-secret-value") {
			t.Errorf("報告洩漏了秘密內容:\n%s", out)
		}
		if !strings.Contains(out, "AUTHOR_KEY_SECRET · 已設定（") {
			t.Errorf("憑證列沒有報有無與長度:\n%s", out)
		}
	})
}

// TestCheckConfigReportsRetentionAndRotation 確認報告裡有那些「失效完全
// 沒有症狀」的設定。
//
// 兩個例子：
//   - POST_RETENTION_DAYS 是全站唯一會**不可逆地刪掉使用者內容**的設定，
//     而服務照常啟動、照常回應 —— 只有背景排程會在 N 天後硬刪。
//   - LOG_MAX_SIZE_MB = 0 代表「不輪替」，同樣沒有任何症狀，只是日誌檔會
//     一直長到磁碟滿。
//
// 這道閘門存在的理由是「回答這個行程會以什麼組態啟動」，而這兩個值決定
// 了行程啟動之後會做什麼 —— 因此它們必須在報告裡。
func TestCheckConfigReportsRetentionAndRotation(t *testing.T) {
	t.Setenv("FILES_SERVER_TOKEN", "")

	cfgPath := writeConfigFile(t, strings.Join([]string{
		"COOKIE_SECURE=false",
		"POST_RETENTION_DAYS=7",
		"LOG_MAX_SIZE_MB=0",
		"LOG_MAX_FILES=3",
	}, "\n")+"\n")

	out := captureStdout(t, func() {
		_ = checkConfig(cfgPath)
	})

	wants := []string{
		"軟刪除貼文保留",
		"7 天",
		"不輪替",
		"3 檔",
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("報告缺少 %q:\n%s", want, out)
		}
	}
}

// TestCheckConfigReportsDevDefaultsAndExplicitTTL 走非正式環境的路徑：
// 明示的 TTL、非空的代理白名單、合法的 ES 基本驗證，結論應為通過。
func TestCheckConfigReportsDevDefaultsAndExplicitTTL(t *testing.T) {
	cfgPath := writeConfigFile(t, strings.Join([]string{
		"PUBLIC_BASE_URL=http://localhost:8080",
		"TRUSTED_PROXY_CIDRS=10.0.0.0/8,192.168.0.0/16",
		"MEDIA_TOKEN_TTL_SECONDS=60",
		"ES_URL=http://es.internal:9200",
		"ES_USERNAME=elastic",
		"ES_PASSWORD=p",
		"ALLOWED_ADMIN_EMAIL=only@example.com",
	}, "\n")+"\n")

	var err error
	out := captureStdout(t, func() {
		err = checkConfig(cfgPath)
	})
	if err != nil {
		t.Fatalf("非正式環境 + 明示 TTL 的 checkConfig = %v, want nil", err)
	}

	wants := []string{
		"非正式環境下會採用開發用的兜底值",
		"10.0.0.0/8,192.168.0.0/16", // 代理白名單有值時原樣印出
		"設定檔明確設定",                   // TTL 來源不是兜底值
		"60 秒 · 1 分鐘",
		"only@example.com",
		"結論：通過",
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("報告缺少 %q:\n%s", want, out)
		}
	}
}

// TestCheckConfigReportsESAuthMismatches 逐一走完 ES 驗證的五種不對稱組合。
//
// 每一種組合的症狀都不同（401、不送標頭、送錯標頭），但全部都是靜默的：
// 搜尋只是安靜地退回 MySQL LIKE。因此報告必須逐項點名，而不是合成一句
// 「ES 驗證設定有問題」。
func TestCheckConfigReportsESAuthMismatches(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "有帳密但沒有 ES_URL",
			body: "ES_USERNAME=elastic\n",
			want: "有 ES 驗證資訊但 ES_URL 留空",
		},
		{
			name: "只有帳號沒有密碼",
			body: "ES_URL=http://es.internal:9200\nES_USERNAME=elastic\n",
			want: "只設了 ES_USERNAME 沒有 ES_PASSWORD",
		},
		{
			name: "只有密碼沒有帳號",
			body: "ES_URL=http://es.internal:9200\nES_PASSWORD=p\n",
			want: "組不成一組 Basic Auth",
		},
		{
			name: "API key 與帳號同時設定",
			body: "ES_URL=http://es.internal:9200\nES_USERNAME=elastic\nES_API_KEY=abc\n",
			want: "實際送出的是 API key",
		},
		{
			name: "有 ES_URL 但沒有驗證",
			body: "ES_URL=http://es.internal:9200\n",
			want: "沒有任何驗證資訊",
		},
		{
			name: "沒有 ES_URL 也沒有驗證",
			body: "SERVER_PORT=:18091\n",
			want: "搜尋功能不會啟用",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfgPath := writeConfigFile(t, c.body)
			var err error
			out := captureStdout(t, func() {
				err = checkConfig(cfgPath)
			})
			if err != nil {
				t.Fatalf("非正式環境的 ES 設定不該讓 Validate 失敗: %v", err)
			}
			if !strings.Contains(out, c.want) {
				t.Errorf("報告缺少 %q:\n%s", c.want, out)
			}
		})
	}
}
