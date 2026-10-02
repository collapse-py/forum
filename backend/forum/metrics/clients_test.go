/*
每 IP 統計的測試（metrics/clients_test.go）。

這一組測的是三個會在真實流量下出問題的邊界，而不是「計數有沒有加對」：

	1. 基數上限之後**驅逐最舊的**而不是丟棄新的 —— 丟棄的版本會讓輪換位址的
	   攻擊者把真實使用者擠出清單，而症狀是「監控頁看起來很乾淨」。
	2. 閒置剪枝不影響仍在活動的來源。
	3. 快照的排序是穩定的，否則自動刷新會讓表格整列跳動。
*/

package metrics

import (
	"testing"
	"time"
)

// newClientTestRegistry 建立一個有假時鐘、上限可控的 Registry。
func newClientTestRegistry(t *testing.T, maxClients, idleMinutes int) (*Registry, *fixedClock) {
	t.Helper()
	clock := &fixedClock{current: time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)}
	registry := New(Options{
		WindowMinutes:     120,
		MaxClients:        maxClients,
		ClientIdleMinutes: idleMinutes,
		Now:               clock.now,
	})
	return registry, clock
}

func observeClient(registry *Registry, ip, route string, status int) {
	registry.ObserveClient(ip, ClientSourcePeer, route, status, false)
}

// 同一個位址的多次請求必須累積成一份統計，而不是各自成列。
func TestObserveClientAccumulatesPerAddress(t *testing.T) {
	registry, _ := newClientTestRegistry(t, 0, 0)
	observeClient(registry, "203.0.113.9", "/api/forum/posts", 201)
	observeClient(registry, "203.0.113.9", "/api/forum/posts", 429)
	observeClient(registry, "203.0.113.9", "/api/forum/posts/1/comments", 500)

	clients := registry.Snapshot(0).Clients
	if len(clients) != 1 {
		t.Fatalf("clients = %d, want 1", len(clients))
	}
	got := clients[0]
	if got.Total != 3 {
		t.Errorf("total = %d, want 3", got.Total)
	}
	// 429 與 500 同時屬於錯誤分類，但 rateLimited 是獨立欄位 —— 它必須能
	// 與 clientErrors 並存，否則「被限流」這個可行動訊號會被算掉。
	if got.ClientErrors != 1 || got.RateLimited != 1 || got.ServerErrors != 1 {
		t.Errorf("errors: client=%d rateLimited=%d server=%d, want 1/1/1",
			got.ClientErrors, got.RateLimited, got.ServerErrors)
	}
	// LastRoute 是「最近打到哪裡」，因此必須是第三次那一條。
	if got.LastRoute != "/api/forum/posts/:id/comments" {
		t.Errorf("lastRoute = %q, want /api/forum/posts/:id/comments", got.LastRoute)
	}
}

// 被封鎖擋下的請求仍然要計入 total：那正是「有人在硬闖」的證據。
func TestObserveClientCountsBannedRequests(t *testing.T) {
	registry, _ := newClientTestRegistry(t, 0, 0)
	registry.ObserveClient("203.0.113.9", ClientSourceXFF, "/api/forum/posts", 403, true)
	registry.ObserveClient("203.0.113.9", ClientSourceXFF, "/api/forum/posts", 403, true)
	// 第三個請求不是封鎖擋下的，而是「不是管理員」的 401 —— 它不該被算進
	// banned，但仍是一個 4xx，因此要算進 clientErrors。
	registry.ObserveClient("203.0.113.9", ClientSourceXFF, "/api/admin/blocks", 401, false)

	got := registry.Snapshot(0).Clients[0]
	if got.Banned != 2 {
		t.Errorf("banned = %d, want 2", got.Banned)
	}
	if got.Total != 3 {
		t.Errorf("total = %d, want 3（被擋下的請求也要計入）", got.Total)
	}
	if got.ClientErrors != 3 {
		t.Errorf("clientErrors = %d, want 3（三次都是 4xx）", got.ClientErrors)
	}
}

// 空位址不能變成一個統計槽：那會讓所有拿不到來源的請求共用一份數字，
// 畫面上看起來像「一個神秘來源」。
func TestObserveClientIgnoresEmptyAddress(t *testing.T) {
	registry, _ := newClientTestRegistry(t, 0, 0)
	registry.ObserveClient("", ClientSourcePeer, "/api/forum/posts", 200, false)

	if got := len(registry.Snapshot(0).Clients); got != 0 {
		t.Fatalf("clients = %d, want 0", got)
	}
}

// 關鍵行為：達到上限時驅逐最久沒出現的那一個，而不是丟棄新來的。
//
// 丟棄的版本在這裡會失敗：第三個位址（唯一還在活動的）會被丟掉，而清單裡
// 留著兩個已經安靜很久的假位址 —— 那正是「監控頁被攻擊者清空」的症状。
func TestObserveClientEvictsOldestAtCapacity(t *testing.T) {
	registry, clock := newClientTestRegistry(t, 2, 0)
	observeClient(registry, "203.0.113.1", "/a", 200)
	clock.advance(time.Minute)
	observeClient(registry, "203.0.113.2", "/a", 200)

	// 第三個位址到來時應驅逐 203.0.113.1（最後活動時間最舊）。
	clock.advance(time.Minute)
	observeClient(registry, "203.0.113.3", "/a", 200)

	clients := registry.Snapshot(0).Clients
	if len(clients) != 2 {
		t.Fatalf("clients = %d, want 2（上限之內）", len(clients))
	}
	for _, client := range clients {
		if client.IP == "203.0.113.1" {
			t.Error("最舊的來源應該被驅逐，但它還在清單裡")
		}
	}
	if clients[0].IP != "203.0.113.3" {
		t.Errorf("clients[0] = %s, want 203.0.113.3（請求數最多的排最前）", clients[0].IP)
	}
}

// 驅逐必須可觀察：沒有 clientsDropped，被截斷的清單會看起來只是「今天沒人來」。
func TestObserveClientReportsDroppedCount(t *testing.T) {
	registry, clock := newClientTestRegistry(t, 1, 0)
	observeClient(registry, "203.0.113.1", "/a", 200)
	clock.advance(time.Minute)
	observeClient(registry, "203.0.113.2", "/a", 200)

	snapshot := registry.Snapshot(0)
	if snapshot.ClientsDropped != 1 {
		t.Errorf("clientsDropped = %d, want 1", snapshot.ClientsDropped)
	}
	if snapshot.MaxClients != 1 {
		t.Errorf("maxClients = %d, want 1", snapshot.MaxClients)
	}
}

// 閒置剪枝：超過 idle 分鐘沒出現的來源被移除，仍在活動的留著。
func TestObserveClientPrunesIdleSources(t *testing.T) {
	registry, clock := newClientTestRegistry(t, 0, 30)
	observeClient(registry, "203.0.113.1", "/a", 200)
	clock.advance(10 * time.Minute)
	observeClient(registry, "203.0.113.2", "/a", 200)

	// 過了 25 分鐘之後：.1 已經 35 分鐘沒出現，.2 只有 15 分鐘。
	clock.advance(25 * time.Minute)
	observeClient(registry, "203.0.113.2", "/b", 200)

	clients := registry.Snapshot(0).Clients
	if len(clients) != 1 {
		t.Fatalf("clients = %d, want 1（.1 應被剪枝）", len(clients))
	}
	if clients[0].IP != "203.0.113.2" {
		t.Errorf("留下的是 %s, want 203.0.113.2", clients[0].IP)
	}
}

// 剪枝只能用「最後活動」，不能用「建立時間」：一個三小時前來過、剛剛又回來
// 的來源仍然值得顯示。
func TestObserveClientKeepsLongLivedSource(t *testing.T) {
	registry, clock := newClientTestRegistry(t, 0, 30)
	observeClient(registry, "203.0.113.1", "/a", 200)
	for range 10 {
		clock.advance(20 * time.Minute)
		observeClient(registry, "203.0.113.1", "/a", 200)
	}
	// 這個位址存在了 200 分鐘，遠超過 30 分鐘的閒置期。
	clients := registry.Snapshot(0).Clients
	if len(clients) != 1 {
		t.Fatalf("clients = %d, want 1（持續活動的來源不該被剪掉）", len(clients))
	}
}

// 相同請求數時依位址字串排序，讓每一輪自動刷新都得到相同的列順序。
func TestObserveClientSortIsStable(t *testing.T) {
	// 刻意不推進時鐘：三個位址因此有相同的請求數與相同的最後活動時刻，
	// 排序落到最後一級的位址字串比對 —— 那才是「自動刷新時整列不跳動」真正
	// 依賴的那一級。（最後活動時刻不同時本來就該按新舊排，那是另一個規則。）
	registry, _ := newClientTestRegistry(t, 0, 0)
	for _, ip := range []string{"203.0.113.3", "203.0.113.1", "203.0.113.2"} {
		observeClient(registry, ip, "/a", 200)
	}

	want := []string{"203.0.113.1", "203.0.113.2", "203.0.113.3"}
	for range 5 {
		clients := registry.Snapshot(0).Clients
		for i, ip := range want {
			if clients[i].IP != ip {
				t.Fatalf("第 %d 列 = %s, want %s（相同數量時應依字串排序）", i, clients[i].IP, ip)
			}
		}
	}
}

// source 必須原樣送到前端：管理員要能分辨「連線對端」與「使用者自己送的標頭」。
func TestObserveClientKeepsSource(t *testing.T) {
	registry, _ := newClientTestRegistry(t, 0, 0)
	registry.ObserveClient("203.0.113.9", ClientSourceXFF, "/a", 200, false)
	registry.ObserveClient("203.0.113.10", ClientSourcePeer, "/a", 200, false)

	sources := map[string]string{}
	for _, client := range registry.Snapshot(0).Clients {
		sources[client.IP] = client.Source
	}
	if sources["203.0.113.9"] != ClientSourceXFF {
		t.Errorf("source = %q, want xff", sources["203.0.113.9"])
	}
	if sources["203.0.113.10"] != ClientSourcePeer {
		t.Errorf("source = %q, want peer", sources["203.0.113.10"])
	}
}

// 剪枝必須在沒有資料庫的情況下也會發生。
//
// StartFlusher 在拿不到 *sql.DB 時不會啟動（見它的說明），因此「pruneLocked 只
// 由 FlushPending 呼叫」會讓那種部署的閒置來源永遠留著。這支測試直接讀 map，
// 因為 Snapshot 自己也會剪枝，用它斷言會測不出「誰呼叫了剪枝」。
func TestPruneClientsRunsWithoutFlusher(t *testing.T) {
	registry, clock := newClientTestRegistry(t, 0, 30)
	observeClient(registry, "203.0.113.1", "/a", 200)
	clock.advance(time.Hour)

	registry.mu.Lock()
	registry.pruneClientsLocked(registry.now())
	remaining := len(registry.clients)
	registry.mu.Unlock()

	if remaining != 0 {
		t.Fatalf("clients = %d, want 0（閒置一小時的來源應被剪掉）", remaining)
	}
}