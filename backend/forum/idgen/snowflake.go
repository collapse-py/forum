/*
套件 idgen 產生雪花識別碼（snowflake ID），供資料庫主鍵使用。

【為什麼需要它】

資料庫原本以 AUTO_INCREMENT 產生主鍵，而自增主鍵有兩個問題：

 1. 識別碼的產生順序綁在單一資料庫實例上。多實例或分庫之後，「誰先誰後」
    不再是應用層能回答的問題，而這個專案的排序（動態、通知、稽核）都依賴
    「id 大致等於建立時間」。
 2. 寫入時必須先 INSERT 再 LastInsertId 回讀。那是一次額外的往返，而且在
    交易裡「先寫入才知道 id」讓「先組好完整物件、再落庫」變得不可能。

雪花演算法把時間、機器與序號編進同一個整數，由應用層在寫入前產生 id。
本專案只保留它需要的部分：

	41 位元  毫秒時間戳（自 2024-01-01T00:00:00Z 起算）
	 6 位元  節點編號（0–63）
	 6 位元  同一毫秒內的序號（0–63）
	------
	53 位元

【為什麼是 53 位元而不是標準的 64 位元】

這個 API 的 id 以 JSON 數字送給瀏覽器，而 JavaScript 的 number 只能精確
表示到 2^53-1。標準雪花（41+10+12 = 63 位元）在 2026 年就已經遠超這個
範圍：前端拿到的是被四捨五入過的數字，而症狀是「某些貼文點進去是 404」
—— 只有部分 id 會壞，因此極難聯想是精度問題。

把位元數壓在 53 位元內，前端與既有 API 契約完全不必改。代價是同一毫秒
只能發 64 個 id、節點上限 64 台。這個站是論壇，單一節點的寫入量遠低於
每秒 64,000 個 id；真正需要更高吞吐時，該換的是 API 契約（id 改字串），
而不是把精度問題留給前端。

【時間戳只有 41 位元，夠用嗎】

2^41 毫秒約等於 69.7 年，從 2024-01-01 起算可用到 2093 年。這比絕大多數
服務的壽命長；真的走到那一天時，該做的是換 epoch 並遷移資料，而不是
現在就犧牲 JS 相容性。

【與既有資料的關係】

既有資料的 AUTO_INCREMENT id（1、2、3…）全部小於新的雪花 id，因此
「id 遞增」的語意在新舊資料之間仍然成立：舊資料排在前面，新資料排在
後面。遷移會把既有資料重新編號（見 data.MigrateMySQL 的第 38 步），
因此實際上不會長期混用兩種 id。
*/
package idgen

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// EpochMillis 是時間戳的起點：2024-01-01T00:00:00Z。
	// 選在這個專案開始之前，讓可用的時間範圍涵蓋整個專案生命週期。
	EpochMillis int64 = 1704067200000

	// timestampBits / nodeBits / sequenceBits 是位元配置。
	// 三者相加必須等於 53（見檔頭），測試會釘住這個不變式。
	timestampBits = 41
	nodeBits      = 6
	sequenceBits  = 6
	totalBits     = timestampBits + nodeBits + sequenceBits

	// MaxNodeID 是節點編號的上限。多實例部署時每個實例必須不同。
	MaxNodeID = (1 << nodeBits) - 1

	maxSequence  = (1 << sequenceBits) - 1
	maxTimestamp = (1 << timestampBits) - 1

	// sequenceShift 是序號在 id 中的位移；nodeShift 是節點編號的位移。
	sequenceShift  = 0
	nodeShift      = sequenceBits
	timestampShift = nodeBits + sequenceBits
)

// Generator 是一個執行緒安全的雪花 id 產生器。
//
// 同一個行程只應該有一個 Generator（見套件層級的 Next 與 Configure）：
// 兩個 Generator 共用同一個節點編號時，會在時鐘回退或同一毫秒內產生
// 重複的 id。這個型別仍可獨立建構，是為了讓測試能注入時鐘與節點編號。
type Generator struct {
	mu sync.Mutex
	// nodeID 是這台實例的編號（0–63），建立後不變。
	nodeID int64
	// lastMillis 是上一次發號使用的毫秒時間戳。
	// 時鐘回退時用它把 id 維持在單調遞增，而不是產生一個比舊 id 小的值
	// （那會讓「依 id 排序」的查詢把新資料排到舊資料前面）。
	lastMillis int64
	// sequence 是 lastMillis 這一毫秒內已發出的序號。
	sequence int64
	// now 讓測試能注入時鐘。正式路徑一律是 time.Now。
	now func() time.Time
}

// New 建立一個指定節點編號的產生器。
//
// nodeID 必須在 0–63 之間。超出範圍時回傳錯誤而不是夾到邊界：兩個實例
// 因為設定打錯而共用同一個節點編號時，症狀會是「偶爾有請求撞主鍵」，
// 而那比啟動失敗難查得多。
func New(nodeID int64) (*Generator, error) {
	if nodeID < 0 || nodeID > MaxNodeID {
		return nil, fmt.Errorf("snowflake node id %d 超出範圍（0–%d）", nodeID, MaxNodeID)
	}
	return &Generator{nodeID: nodeID, now: time.Now}, nil
}

// Next 產生下一個 id。
//
// 保證：
//   - 同一行程內嚴格遞增（時鐘回退時沿用 lastMillis，不產生較小的 id）。
//   - 53 位元以內，可直接以 JSON 數字送給 JavaScript。
//   - 同一毫秒內最多 64 個；超過時會等到下一毫秒，而不是回傳重複值。
//
// 副作用：同一毫秒內序號用盡時會短暫沉睡。正常情況下約 1 毫秒；若系統時鐘
// 大幅回退（例如 VM 快照還原），會一直等到時鐘追上 lastMillis 才繼續發號 ——
// 這是「不產生重複 id」與「不產生未來時間戳」之間的取捨。
//
// 會 panic 的條件只有兩個，且都代表部署環境壞掉而不是呼叫端用法錯誤：
// 系統時鐘早於 epoch（2024-01-01），或時間戳超出 41 位元可表示範圍。
// 兩者都無法在執行期「修復」，而回傳 0 會讓呼叫端寫入一個看似合法的 id。
func (g *Generator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	ms := g.now().UnixMilli()
	if ms < EpochMillis {
		panic(fmt.Sprintf("idgen: 系統時鐘 (%s) 早於雪花 epoch (%s)", time.UnixMilli(ms).UTC(), time.UnixMilli(EpochMillis).UTC()))
	}
	if ms < g.lastMillis {
		// 時鐘回退（NTP 校正、VM 快照還原、手動改時間）。沿用上一個
		// 時間戳繼續發號，讓 id 維持遞增；序號用盡時才真的等時鐘追上。
		ms = g.lastMillis
	}

	if ms == g.lastMillis {
		g.sequence++
		if g.sequence > maxSequence {
			// 這一毫秒的 64 個序號用完了。等到時間真的前進為止。
			// 用 Sleep 而不是忙迴圈：這個等待最多 1 毫秒，但忙迴圈會
			// 在整個等待期間佔住一個 CPU 核心。
			for ms <= g.lastMillis {
				time.Sleep(100 * time.Microsecond)
				ms = g.now().UnixMilli()
				if ms < g.lastMillis {
					// 時鐘在一次短暫的等待後仍然落後：保持單調，繼續等。
					ms = g.lastMillis
				}
			}
			g.sequence = 0
		}
	} else {
		g.sequence = 0
	}

	elapsed := ms - EpochMillis
	if elapsed > maxTimestamp {
		panic(fmt.Sprintf("idgen: 時間戳 %d 超出 %d 位元可表示範圍（epoch 需要更新）", elapsed, timestampBits))
	}

	g.lastMillis = ms
	return (elapsed << timestampShift) | (g.nodeID << nodeShift) | (g.sequence << sequenceShift)
}

// std 是行程層級的產生器。零值不可用，因此以 init 建立一個節點 0 的實例。
//
// 為什麼是套件層級的單例：雪花 id 的節點編號是「一個行程一份」的事實，
// 而呼叫端（handler、稽核、遷移）散落在多個套件。把 Generator 一路 injection
// 下去會讓每一個函式簽章多一個參數，而那個參數在正式路徑上永遠是同一個值。
var std atomic.Pointer[Generator]

func init() {
	g, err := New(0)
	if err != nil {
		// 節點 0 一定合法；走到這裡代表常數被改壞了。
		panic(err)
	}
	std.Store(g)
}

// Configure 以指定的節點編號取代行程層級的產生器。
//
// 呼叫時機：main 在讀完設定、開始服務之前呼叫一次。之後的 Next 都使用
// 新的節點編號。節點編號錯誤時回傳錯誤，呼叫端應視為啟動失敗。
func Configure(nodeID int64) error {
	g, err := New(nodeID)
	if err != nil {
		return err
	}
	std.Store(g)
	return nil
}

// Next 以行程層級的產生器產生下一個 id。
//
// 這是正式路徑唯一需要呼叫的函式；它與 (*Generator).Next 有相同的保證。
func Next() int64 {
	return std.Load().Next()
}
