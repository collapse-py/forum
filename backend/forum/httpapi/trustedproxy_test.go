package httpapi

/*
可信任代理的信任模型測試（httpapi/trustedproxy_test.go）。

這一支測試存在的理由：H4 描述的症狀是「IP 封鎖可被單一標頭繞過」與「稽核
紀錄的來源不可信」，而這兩者都**不會**讓任何既有的測試失敗 —— 在沒有
TRUSTED_PROXY_CIDRS 的情況下舊行為完全一樣，症狀要到有人在真實流量裡偽造
標頭才會出現。因此這裡把信任模型的三個分支各自釘住。

覆蓋的三件事：

  1. 未設定白名單時維持舊行為（標頭優先）—— 這是既有部署的相容性保證。
  2. 已設定白名單時，對端不可信就完全忽略轉送標頭（這是修掉漏洞的那一步）。
  3. 已設定白名單時，對端可信則從右往左掃過所有可信代理。
  4. 封鎖的候選清單包含對端，因此偽造標頭無法解除封鎖。
*/

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// forwardedRequest 造一個帶有代理標頭的請求。
func forwardedRequest(remoteAddr, xff string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/forum/posts", nil)
	request.RemoteAddr = remoteAddr
	if xff != "" {
		request.Header.Set("X-Forwarded-For", xff)
	}
	return request
}

func TestResolveClientIPWithoutTrustedProxiesKeepsLegacyOrder(t *testing.T) {
	// 未設定白名單 → 舊行為。這一條守住「升級不會讓既有部署突然共用一個
	// 限流額度」：分桶鍵仍然是可以被偽造的，但至少不會變成代理的位址。
	server := &Server{}

	ip, source := server.clientIPDetail(forwardedRequest("10.0.0.1:44321", "198.51.100.7"))
	if ip != "198.51.100.7" || source != "xff" {
		t.Errorf("ip/source = %q/%q, want 198.51.100.7/xff", ip, source)
	}

	// 沒有標頭時退回對端，這一點兩種模式都必須成立。
	ip, source = server.clientIPDetail(forwardedRequest("10.0.0.1:44321", ""))
	if ip != "10.0.0.1" || source != "peer" {
		t.Errorf("ip/source = %q/%q, want 10.0.0.1/peer", ip, source)
	}
}

func TestResolveClientIPIgnoresHeadersFromUntrustedPeer(t *testing.T) {
	// 這是修掉 H4 的那一步：白名單已設定，但對端不在名單裡，因此那個標頭
	// 沒有資格替自己發言。攻擊者直連（或打到任何一台不是白名單的機器）時，
	// 偽造 XFF 不再改變任何判定。
	server := &Server{trustedProxies: parseTrustedProxyCIDRs("10.0.0.0/8")}

	ip, source := server.clientIPDetail(forwardedRequest("203.0.113.50:44321", "198.51.100.7"))
	if ip != "203.0.113.50" || source != "peer" {
		t.Errorf("ip/source = %q/%q, want 203.0.113.50/peer（不可信對端的標頭必須被忽略）", ip, source)
	}
}

func TestResolveClientIPWalksForwardedChainRightToLeft(t *testing.T) {
	// 多層代理（CDN → 負載平衡 → 本站）時，最左項是最上游填的、也是攻擊者
	// 最想控制的位置。正確的演算法是從右往左跳過可信代理。
	server := &Server{trustedProxies: parseTrustedProxyCIDRs("10.0.0.0/8")}

	ip, source := server.clientIPDetail(forwardedRequest(
		"10.0.0.1:44321", "198.51.100.7, 10.0.0.5"))
	if ip != "198.51.100.7" || source != "xff" {
		t.Errorf("ip/source = %q/%q, want 198.51.100.7/xff", ip, source)
	}

	// 三層：使用者 → CDN（可信）→ 本站。三段 XFF 中右兩段是可信代理，
	// 剩下最左那一段才是發起者。
	ip, _ = server.clientIPDetail(forwardedRequest(
		"10.0.0.1:44321", "203.0.113.9, 10.0.0.9, 10.0.0.5"))
	if ip != "203.0.113.9" {
		t.Errorf("ip = %q, want 203.0.113.9", ip)
	}

	// 攻擊者在 XFF 左邊**多塞**一項：判定必須完全不變。那正是從右往左掃
	// 要買到的東西 —— 多塞的值位於「我們自己信任的代理已經簽證的內容」之外，
	// 因此它對「這是誰」這個問題沒有發言權。（舊的取最左一項寫法在這裡會
	// 直接回 1.2.3.4，讓攻擊者取得一份全新的額度與一份乾淨的封鎖查詢結果。）
	ip, _ = server.clientIPDetail(forwardedRequest(
		"10.0.0.1:44321", "1.2.3.4, 203.0.113.9, 10.0.0.5"))
	if ip != "203.0.113.9" {
		t.Errorf("ip = %q, want 203.0.113.9（在左側多塞一項不該改變判定）", ip)
	}
}

func TestResolveClientIPFallsBackWhenWholeChainIsTrusted(t *testing.T) {
	// XFF 每一段都是可信代理：沒有可指認的發起者。此時退回對端，而不是硬
	// 取最左一項（那一項仍然只是最上游填的字串）。
	server := &Server{trustedProxies: parseTrustedProxyCIDRs("10.0.0.0/8")}

	ip, source := server.clientIPDetail(forwardedRequest("10.0.0.1:44321", "10.0.0.5, 10.0.0.9"))
	if ip != "10.0.0.1" || source != "peer" {
		t.Errorf("ip/source = %q/%q, want 10.0.0.1/peer", ip, source)
	}
}

func TestParseTrustedProxyCIDRsAcceptsBareIPAndRejectsGarbage(t *testing.T) {
	// 裸 IP 是最常見的設定寫法（單一 sidecar 代理），不該要求人寫出 /32。
	set := parseTrustedProxyCIDRs("127.0.0.1, 10.1.0.0/16, ::1")
	if !set.configured || len(set.nets) != 3 {
		t.Fatalf("configured=%v nets=%d, want true / 3", set.configured, len(set.nets))
	}
	if !set.isTrusted("127.0.0.1") || !set.isTrusted("10.1.2.3") {
		t.Error("127.0.0.1 與 10.1.2.3 應該都算可信")
	}
	if set.isTrusted("10.2.0.1") {
		t.Error("10.2.0.1 不在 10.1.0.0/16 裡，不該算可信")
	}
	if len(set.invalid) != 0 {
		t.Errorf("全部合法時 invalid 應為空, got %v", set.invalid)
	}

	// 全是垃圾 → 視為「未設定」。這個區分讓監控頁能把「設定寫錯」顯示成一個
	// 看得見的狀態，而不是靜默退回舊行為。
	broken := parseTrustedProxyCIDRs("not-an-ip, 999.999.999.999/8")
	if broken.configured {
		t.Errorf("configured = true, want false（寫錯的值不能被當成有效的白名單）")
	}
	if broken.declared == "" {
		t.Error("declared 應保留原文，讓管理員看得到自己寫了什麼")
	}
	if len(broken.invalid) != 2 {
		t.Errorf("invalid = %v, want 兩項", broken.invalid)
	}
}

// 「部分寫錯」是這一整套信任模型裡最容易被忽略的失敗模式：一筆打對的位址段
// 讓 configured 變成 true，於是監控頁走「已設定可信代理」分支，只列出真正
// 生效的那些 —— 寫錯的那條永遠不生效，而畫面看起來完全正常。
//
// 這個測試釘住「寫錯的項目會被保留下來」，因為它是那個盲點唯一的補救。
func TestParseTrustedProxyCIDRsKeepsPartiallyInvalidEntries(t *testing.T) {
	// 第二項遮罩多打一個字（/16 → /64），是典型的「打對大部分」寫法。
	set := parseTrustedProxyCIDRs("10.0.0.0/8, 172.17.0.0/64")

	if !set.configured {
		t.Fatal("有一項合法就應該算 configured（否則已設定的代理會退回舊行為）")
	}
	if len(set.nets) != 1 || set.nets[0].String() != "10.0.0.0/8" {
		t.Fatalf("nets = %v, want 只有 10.0.0.0/8", set.nets)
	}
	if len(set.invalid) != 1 || set.invalid[0] != "172.17.0.0/64" {
		t.Errorf("invalid = %v, want [172.17.0.0/64]（要保留管理員寫的原值）", set.invalid)
	}
	// 白名單裡沒有的位址必須不可信 —— 寫錯的項目的效果是「不採信」，
	// 而不是「放行」。
	if set.isTrusted("172.17.0.1") {
		t.Error("無法解析的項目不該讓該位址變成可信代理")
	}

	// trustReport 是監控頁唯一的資料來源，因此它必須把無效項一起帶出去。
	server := &Server{trustedProxies: set}
	report := server.trustReport()
	if report.Mode != "trusted-proxies" {
		t.Errorf("mode = %q, want trusted-proxies", report.Mode)
	}
	if len(report.Trusted) != 1 || report.Trusted[0] != "10.0.0.0/8" {
		t.Errorf("trusted = %v, want [10.0.0.0/8]", report.Trusted)
	}
	if len(report.Invalid) != 1 || report.Invalid[0] != "172.17.0.0/64" {
		t.Errorf("invalid = %v, want [172.17.0.0/64]", report.Invalid)
	}

	// trustReport 不得洩漏內部 slice：呼叫端若 append 進去，會改到
	// parseTrustedProxyCIDRs 產生的那一份（那是建構後不可變的欄位）。
	report.Invalid = append(report.Invalid, "10.9.9.9/32")
	if len(server.trustReport().Invalid) != 1 {
		t.Error("trustReport 的 Invalid 不得與內部 slice 共享底層陣列")
	}
}

// 未設定時 Invalid 必須是空陣列而不是 null：前端對它做 .length 與 .join()。
func TestTrustReportAlwaysReturnsNonNilSlices(t *testing.T) {
	for _, raw := range []string{"", "10.0.0.0/8", "nonsense"} {
		report := (&Server{trustedProxies: parseTrustedProxyCIDRs(raw)}).trustReport()
		if report.Trusted == nil || report.Invalid == nil {
			t.Errorf("declared=%q: Trusted=%v Invalid=%v, want 非 nil 的切片", raw, report.Trusted, report.Invalid)
		}
	}
	// 測試以 struct literal 構造 Server 時 trustedProxies 會是 nil。
	report := (&Server{}).trustReport()
	if report.Trusted == nil || report.Invalid == nil {
		t.Errorf("nil trustedProxies 時切片必須仍非 nil: %+v", report)
	}
}

func TestClientIPCandidatesAlwaysIncludePeer(t *testing.T) {
	// 這條是「封鎖無法被偽造標頭繞過」的保證：無論信任模型怎麼設定，對端
	// 都在候選清單裡，因此一個被封鎖的直連使用者加多少個 XFF 都還是會被查到。
	proxies := parseTrustedProxyCIDRs("10.0.0.0/8")
	candidates := clientIPCandidates(forwardedRequest("10.0.0.1:44321", "198.51.100.7"), proxies)
	if len(candidates) != 2 || candidates[0] != "198.51.100.7" || candidates[1] != "10.0.0.1" {
		t.Errorf("candidates = %v, want [198.51.100.7 10.0.0.1]", candidates)
	}

	// 直連（沒有任何可信代理）時兩者相同，因此只查一次 —— 這是常見情況下
	// 不增加 Redis 往返的理由。
	direct := clientIPCandidates(forwardedRequest("203.0.113.50:44321", ""), proxies)
	if len(direct) != 1 || direct[0] != "203.0.113.50" {
		t.Errorf("candidates = %v, want [203.0.113.50]", direct)
	}

	// 未設定白名單時：解析值就是 XFF，而對端必須**另外**被查到，否則舊模式下
	// 偽造標頭仍然能解除封鎖。
	legacy := clientIPCandidates(forwardedRequest("203.0.113.50:44321", "198.51.100.7"), nil)
	if len(legacy) != 2 || legacy[0] != "198.51.100.7" || legacy[1] != "203.0.113.50" {
		t.Errorf("candidates = %v, want [198.51.100.7 203.0.113.50]", legacy)
	}
}

func TestTrustReportExposesBrokenConfiguration(t *testing.T) {
	// 宣告了卻一筆都解析不出來：Mode 必須是 legacy-headers 而 Declared 非空。
	// 這個組合在舊行為下完全不可見 —— 它就是「限流與封鎖目前可被偽造標頭
	// 繞過」這件事唯一的可觀察訊號。
	server := &Server{trustedProxies: parseTrustedProxyCIDRs("nonsense")}
	report := server.trustReport()
	if report.Mode != "legacy-headers" {
		t.Errorf("mode = %q, want legacy-headers", report.Mode)
	}
	if report.Declared != "nonsense" {
		t.Errorf("declared = %q, want nonsense", report.Declared)
	}

	// 正確設定時必須如實回報實際生效的位址段。
	server = &Server{trustedProxies: parseTrustedProxyCIDRs("127.0.0.1")}
	report = server.trustReport()
	if report.Mode != "trusted-proxies" {
		t.Errorf("mode = %q, want trusted-proxies", report.Mode)
	}
	if len(report.Trusted) != 1 || report.Trusted[0] != "127.0.0.1/32" {
		t.Errorf("trusted = %v, want [127.0.0.1/32]", report.Trusted)
	}
}
