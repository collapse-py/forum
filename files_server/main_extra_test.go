/*
main_extra_test.go 補齊 main.go 裡「可以被單獨呼叫而不啟動整個行程」的部分：
Redis 探測（含降級）與 *redis.Client → mediaTokenStore 的轉換。

【為什麼不測 main 本身】
main 會讀工作目錄下的 config.conf、綁定真實埠並在錯誤時 os.Exit —— 在測試
行程裡呼叫它等於讓測試套件自殺。它的組裝順序由各零件的測試與 shutdown.go 的
測試共同守住；這裡只釘住那個最容易在重構時被改壞的 nil 介面陷阱。
*/
package main

import (
	"net"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

// TestMediaTokenStoreOrNil 守住「nil 指標不能變成非 nil 介面」。
//
// 這個陷阱的症狀是：Redis 連不上時 connectRedis 回 nil，但把 nil 塞進介面後
// 介面本身不是 nil，於是 server.go 的 `s.redis == nil` 判斷失效，中介層仍然
// 對一個 nil 用戶端呼叫 Exists —— 每次讀圖都 panic。而它只會在 Redis 故障時
// 發生，正常部署永遠看不到。
func TestMediaTokenStoreOrNil(t *testing.T) {
	var nilClient *redis.Client
	if got := mediaTokenStoreOrNil(nilClient); got != nil {
		t.Errorf("nil client 變成了非 nil 介面 (%T)", got)
	}

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() })

	store := mediaTokenStoreOrNil(client)
	if store == nil {
		t.Fatal("非 nil client 被轉成了 nil 介面")
	}
	if _, ok := store.(*redis.Client); !ok {
		t.Errorf("介面底層型別 = %T，want *redis.Client", store)
	}
}

// TestUploadAuthWarningIsExplicitWhenTokenEmpty 守住「留空＝沒有驗證」必須被講出來。
//
// validUploadToken 在 token 為空時放行所有請求（既有的本機測試就靠它），那是一個
// 靜默的 fail-open 狀態：沒有錯誤、沒有症狀，日誌裡也沒有任何一行說「這台服務沒有
// 驗證」。這一條釘的是那個狀態會印 WARNING，而不是只印「已啟用」或什麼都不印。
func TestUploadAuthWarningIsExplicitWhenTokenEmpty(t *testing.T) {
	empty := uploadAuthWarning(&Config{})
	if !strings.Contains(empty, "WARNING") {
		t.Errorf("token 留空時的啟動紀錄 = %q，want 含 WARNING", empty)
	}
	for _, mention := range []string{"/upload", "/delete"} {
		if !strings.Contains(empty, mention) {
			t.Errorf("啟動紀錄沒有提到受影響的端點 %s: %q", mention, empty)
		}
	}

	set := uploadAuthWarning(&Config{Upload: UploadConfig{Token: "tok"}})
	if strings.Contains(set, "WARNING") {
		t.Errorf("token 已設定卻印 WARNING: %q", set)
	}
	if !strings.Contains(set, "enabled") {
		t.Errorf("token 已設定時啟動紀錄 = %q，want 說「已啟用」", set)
	}
}

// TestImagePolicyWarningIsExplicitWhenDisabled 守住「reencode=false 會留下 GPS」被講出來。
//
// 這個設定值是一個**安靜的隱私決定**：關掉之後 EXIF（含 GPS、相機型號、拍攝時間）會原樣
// 存進儲存並掛在公開的圖片網址上，而沒有任何錯誤、沒有任何回應上的差別。對一個立論是
// 「匿名性由構造保證」的站（README），那是一個與立論矛盾的狀態，而日誌是唯一能看出來
// 的地方 —— 與 uploadAuthWarning 同一個道理（見那個注解）。
func TestImagePolicyWarningIsExplicitWhenDisabled(t *testing.T) {
	off := &Config{Upload: UploadConfig{Reencode: imagePolicyEnabled(false)}}
	line := imagePolicyWarning(off)
	if !strings.Contains(line, "WARNING") {
		t.Errorf("正規化被關掉時啟動紀錄 = %q，want 含 WARNING", line)
	}
	for _, mention := range []string{"reencode", "EXIF", "GPS"} {
		if !strings.Contains(line, mention) {
			t.Errorf("啟動紀錄沒有提到 %s —— 那個決定影響的是什麼？: %q", mention, line)
		}
	}

	// 開啟時不該印 WARNING，而且要印出**生效的**上限與品質：那行日誌是維運唯一能看到
	// 「這台現在用什麼政策」的地方，而它必須是對的（見 config.go 的 imagePolicy 夾雜值）。
	on := imagePolicyWarning(&Config{Upload: UploadConfig{Reencode: imagePolicyEnabled(true), MaxWidth: 1200, MaxHeight: 800, JPEGQuality: 90}})
	if strings.Contains(on, "WARNING") {
		t.Errorf("正規化開啟卻印 WARNING: %q", on)
	}
	for _, mention := range []string{"1200x800", "90"} {
		if !strings.Contains(on, mention) {
			t.Errorf("啟動紀錄沒有印出生效的 %s: %q", mention, on)
		}
	}
}

// TestImagePolicyWarningUsesClampedValues 守住「日誌印的是生效值而不是設定值」。
//
// jpeg_quality = 500 在編碼器上會被夾到 100。若日誌印 500，那是一條不實的啟動紀錄，
// 而維運會拿它去對設定檔 —— 對不出來因為兩個數字都「看起來對」。
func TestImagePolicyWarningUsesClampedValues(t *testing.T) {
	line := imagePolicyWarning(&Config{Upload: UploadConfig{Reencode: imagePolicyEnabled(true), JPEGQuality: 500}})
	if strings.Contains(line, "quality 500") {
		t.Errorf("啟動紀錄印了設定裡的 500 而不是生效的 100: %q", line)
	}
	if !strings.Contains(line, "quality 100") {
		t.Errorf("啟動紀錄 = %q，want 含生效的品質 100", line)
	}
}

//
// 這條路徑不該嘗試連線：位址是空的代表 Redis 沒有被設定，而不是「連
// localhost」。若它真的去連，啟動會多等一個逾時。
func TestConnectRedisWithoutAddr(t *testing.T) {
	cfg := &Config{}
	cfg.Redis.Addr = ""

	if client := connectRedis(cfg); client != nil {
		_ = client.Close()
		t.Fatal("addr 是空字串卻建立了 Redis 用戶端")
	}
}

// TestConnectRedisUnreachableDegrades 守住「連不上 → nil、不阻止啟動」。
//
// 用一個剛被關掉的本地埠：連線會被拒絕，不需要外部網路，也不會等滿逾時。
// 回 nil 是刻意的降級（見 main.go 檔頭第 1 點）—— 上傳與刪除不依賴 Redis。
func TestConnectRedisUnreachableDegrades(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("建立暫存監聽失敗: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("關閉暫存監聽失敗: %v", err)
	}

	cfg := &Config{}
	cfg.Redis.Addr = addr

	client := connectRedis(cfg)
	if client != nil {
		_ = client.Close()
		t.Fatalf("連不上 %s 卻回傳了非 nil 用戶端", addr)
	}
}
