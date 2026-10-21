/*
shutdown.go 的補充測試（backend/shutdown_extra_test.go）。

shutdown_test.go 已經釘住逾時設定、訊號與監聽錯誤的分流、以及 Shutdown
會等在途請求；這裡只補 notifyOnSignal 本身，因為它是 shutdown.go 裡唯一
沒有被任何測試執行到的函式。

刻意不在這裡「假裝送一個訊號」：Windows 上沒有可靠的跨平台方式把
os.Interrupt 送給自己（os.Process.Signal 對它回 not supported），而
syscall.Kill 不存在。真正要驗的是「註冊與撤銷都不會 panic、也不會把
呼叫端的通道填滿」—— 那正是這個函式唯一會做的事，其餘行為由
serveUntilSignal 的測試以直接寫入通道的方式驗證。
*/
package main

import (
	"os"
	"os/signal"
	"testing"
)

// TestNotifyOnSignalRegistersAndStops 執行一次註冊與撤銷。
//
// 撤銷（signal.Stop）是必要的收尾：signal.Notify 會讓執行期保留一個對
// 通道的參照，沒有 Stop 的測試會留下一個永遠沒人讀的通道。同時它也確認
// 「停止」不會 panic —— 部署流程若在啟動失敗時撤銷訊號，不該反而崩潰。
func TestNotifyOnSignalRegistersAndStops(t *testing.T) {
	ch := make(chan os.Signal, 1)
	notifyOnSignal(ch)
	signal.Stop(ch)

	select {
	case sig := <-ch:
		t.Fatalf("沒有送出訊號卻收到 %v", sig)
	default:
	}
}
