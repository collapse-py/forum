package httpapi

/*
可信任反向代理（httpapi/trustedproxy.go）。

這個檔案回答一個問題：X-Forwarded-For 能不能被採信？

為什麼需要它

	X-Forwarded-For 與 X-Real-IP 都是**任何用戶端都能自己設定的標頭**。
	無條件採信它們的後果不是只有「日誌裡的 IP 不準」，而是：

	  - IP 封鎖名單形同不存在。被封鎖者只要在每個請求加上
	    X-Forwarded-For: 1.2.3.4，就會取得一份全新的額度與一份「乾淨」的
	    封鎖查詢結果。封鎖撐得過重啟，撐不過一個標頭。
	  - 稽核紀錄的 ip 欄位不再是稽核依據。攻擊者（或被控的管理員 session）
	    可以讓自己操作的來源 IP 顯示成任意值，而稽核紀錄的全部價值就在於
	    它是真的。

	標準做法是「依 TCP 對端決定要不要採信轉送標頭」：只有當對端本身是我們
	信任的代理（同一台機器的 loopback、Kubernetes 的 sidecar、一段叢集
	內網）時，那個標頭才代表某個不可偽造的事實；對端不可信就整個忽略。

為什麼預設（未設定）時維持舊行為，而不是「預設不採信」

	不採信 XFF 的後果是把限流與封鎖的分桶鍵變成**代理的位址** ——
	一個 abusive 使用者就會讓整站所有人（共用同一個代理）一起被限流甚至
	被封鎖。那是把一個安全弱點換成一個可用性災難，而它會在第一次部署時
	就發生。

	因此這裡採「明確設定才啟用」：TRUSTED_PROXY_CIDRS 留空時行為與過去完全
	相同（採信 XFF），設定之後才取得正確的信任模型。兩種模式的差別寫在
	monitoring 的回應裡（trustedProxies），讓管理員從後臺就看得出自己處在
	哪一種，而不是靠讀設定檔猜。

從右往左掃而不是直接取最左一項

	多層代理（CDN → 負載平衡 → 本站）時，XFF 是「每一層把自己的對端附加到
	右邊」。因此可信的演算法是：從**最右邊**開始，跳過所有屬於白名單的位址，
	遇到的第一個不在白名單的就是原始發起者。

	直接取最左一項（過去的寫法）在這種拓撲下是錯的：最左邊那一項是任何
	上游都可以自由填入的，而它恰好是攻擊者最想控制的位置。
*/

import (
	"net"
	"net/http"
	"strings"

	"forum/forum/metrics"
)

// trustedProxySet 是可信任代理的位址集合。
//
// 建構之後**不可變**，因此讀取時不需要鎖 —— 這與 RateLimiter 的
// Limit / Window 讀建構後不變欄位的做法是同一個理由。
type trustedProxySet struct {
	// nets 是設定檔裡宣告且解析成功的位址段。空長度代表「未設定」或
	// 「寫了但一筆都用不了」，兩者的差別由 declared / invalid 說明。
	nets []*net.IPNet
	// configured 與 len(nets) 分開存，因為「設定了但全部寫錯」與「沒有設定」
	// 對呼叫端的意義完全不同：前者是設定錯誤（監控該報警），後者是刻意選擇。
	configured bool
	// declared 保留設定檔原文，供監控頁顯示（admin 看到空清單時需要知道
	// 「你宣告了什麼卻沒一筆能用」）。
	declared string
	// invalid 保留宣告了卻無法解析的項目。
	//
	// 為什麼需要它：只有「全部寫錯」時才看得見設定有問題，而**部分**寫錯
	// （10.0.0.0/8 打對、172.17.0.0/16 少打一個字）是更常見也更容易被忽略的
	// 情況 —— 那條位址段會靜默地永遠不生效，而管理員在畫面上看到的是一份
	// 看起來正常的白名單。把無法解析的原文帶出去，監控頁才能把
	// 「我寫了什麼」與「程式認得什麼」的落差顯示出來。
	invalid []string
}

// isTrusted 判斷某個位址是否屬於可信任代理。
func (s *trustedProxySet) isTrusted(ip string) bool {
	if s == nil || len(s.nets) == 0 {
		return false
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		// 不是一個可解析的位址（例如 XFF 被塞了 "unknown"）：無法證明它在
		// 白名單裡，因此視為不可信。這是「預設拒絕」而非「預設允許」。
		return false
	}
	for _, network := range s.nets {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

// parseTrustedProxyCIDRs 解析設定檔的 TRUSTED_PROXY_CIDRS。
//
// 接受的每一項可以是「IP/遮罩」（10.0.0.0/8）或裸 IP（127.0.0.1）。裸 IP
// 視為 /32 或 /128 —— 設定檔裡寫單一代理的位址是最常見的用法，強制人寫出
// /32 只是增加設定出錯的機率。
//
// 無法解析的項目不会被丟掉，而是記在 invalid 裡（見 trustedProxySet 的說明）。
// 它們的效果是「不採信」—— 白名單永遠拒絕優先，因此一條寫錯的位址段表現為
// 「那台代理的轉送標頭不被採信」，症狀是限流與封鎖的分桶鍵變成代理的位址。
// 讓它安靜發生正是這個函式要記下來的理由。
//
// 回傳的 trustedProxySet 永遠非 nil（解析失敗時 nets 為空），因此呼叫端
// 不需要 nil 檢查；真正需要 nil 檢查的是**元素**。
func parseTrustedProxyCIDRs(raw string) *trustedProxySet {
	set := &trustedProxySet{declared: strings.TrimSpace(raw)}
	for _, entry := range strings.Split(set.declared, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			// 裸 IP 補成 /32 或 /128，讓後續一律走 ParseCIDR。
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			set.nets = append(set.nets, network)
			continue
		}
		// 記下**原始**宣告值而不是被補過 /32 的版本：管理員要看到的是自己
		// 寫了什麼，而不是程式猜測它「本來可能是什麼」。
		set.invalid = append(set.invalid, strings.TrimSpace(entry))
	}
	set.configured = set.declared != "" && len(set.nets) > 0
	return set
}

// peerIP 回傳 TCP 連線的對端位址（去掉埠號）。
//
// 它與 clientIPDetail 的最後一個分支是同一段程式碼，但**不接受任何標頭**：
// 這是判斷「要不要採信轉送標頭」的唯一依據，因此不能建立在被採信的來源上。
func peerIP(r *http.Request) string {
	// net/http 保證 RemoteAddr 是 "host:port"。從「最後一個冒號」切斷，
	// 因此 [::1]:8080 這種含冒號的 IPv6 也能正確去掉埠號。限制：若
	// RemoteAddr 是不含埠號的裸 IPv6（例如 "::1"），會被截成 ":"。
	// 這個值只作比較與 map 的鍵使用，只要一致即可。
	ip := r.RemoteAddr
	if idx := lastIndexByte(ip, ':'); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// resolveClientIP 決定本次請求的用戶端識別值與它的來源。
//
// 三段信任模型：
//
//  1. 未設定 TRUSTED_PROXY_CIDRS → 舊行為：XFF 最左項 → X-Real-IP → 對端。
//     這個模式在「本站確實被一道會覆寫這些標頭的代理擋在後面、且使用者
//     無法繞過它直連」時是正確的；否則限流與封鎖都可被單一標頭繞過。
//     這是既有部署的相容性選擇，不是「安全」的選擇。
//
//  2. 已設定且對端在白名單裡 → 從 XFF 最右項往左掃，跳過白名單位址，
//     第一個不在白名單的就是發起者；全部都是白名單時退回 X-Real-IP，
//     再退回對端。
//
//  3. 已設定但對端**不在**白名單 → 完全忽略轉送標頭，回對端位址。
//     這一步是整個機制的重點：來路不明的對端沒有資格替自己發言。
func resolveClientIP(r *http.Request, proxies *trustedProxySet) (string, string) {
	peer := peerIP(r)
	if proxies == nil || !proxies.configured {
		// 未設定：維持舊的採信順序，避免既有部署在升級後突然全部共用一個
		// 限流額度。安全性取決於部署，見檔頭。
		return legacyClientIP(r, peer)
	}
	if !proxies.isTrusted(peer) {
		return peer, metrics.ClientSourcePeer
	}
	if parts := splitComma(r.Header.Get("X-Forwarded-For")); len(parts) > 0 {
		for i := len(parts) - 1; i >= 0; i-- {
			if proxies.isTrusted(parts[i]) {
				continue
			}
			return parts[i], metrics.ClientSourceXFF
		}
	}
	// XFF 每一段都是可信代理：這代表從本站往外數每一跳都是自己人，
	// 沒有可指認的發起者。此時退到第二順位而不是硬取最左一項（那個值仍是
	// 最上游填的，而「最上游是我們自己」不代表它是發起者）。
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri, metrics.ClientSourceRealIP
	}
	return peer, metrics.ClientSourcePeer
}

// legacyClientIP 是未設定白名單時的舊行為，逐字保留過去的採信順序。
func legacyClientIP(r *http.Request, peer string) (string, string) {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取最左一項：XFF 是由左往右「代理逐層附加」，最左邊才是原始的發起者；
		// 越靠右越接近本站、可信度越高。
		parts := splitComma(xff)
		// 若 XFF 全是逗號或空白，splitComma 會回傳長度 0 的切片，這時
		// 不應回傳空字串當作 key，否則所有這類請求會共用同一個額度。
		if len(parts) > 0 {
			return parts[0], metrics.ClientSourceXFF
		}
	}
	// 部分反向代理（Nginx 預設）會設定 X-Real-IP，作為第二順位來源。
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri, metrics.ClientSourceRealIP
	}
	return peer, metrics.ClientSourcePeer
}

// clientIPCandidates 列出「這一次請求應該被視為哪些位址」。
//
// 存在的原因是封鎖：只查一個鍵時，被封鎖者可以用一個偽造標頭取得一份
// 乾淨的查詢結果 —— 那正是 IP 封鎖形同不存在的成因。因此封鎖路徑查**所有**
// 候選值，命中任何一個就算被封鎖。
//
// 為什麼不會誤傷：真正被封鎖的是「這個位址」，而當 XFF 不可信時它就是
// 對端位址；兩者一致時只查一次。這一條不需要管理員先設定白名單就有效。
func clientIPCandidates(r *http.Request, proxies *trustedProxySet) []string {
	resolved, _ := resolveClientIP(r, proxies)
	peer := peerIP(r)
	if resolved == "" || resolved == peer {
		if resolved == "" {
			return nil
		}
		return []string{resolved}
	}
	return []string{resolved, peer}
}
