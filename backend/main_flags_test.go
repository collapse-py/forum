/*
main 的旗標與離開路徑測試（backend/main_flags_test.go）。

main() 依三條路徑離開：-check（exit 0 或 1）、-healthz（exit 0 或 1）、
以及正常啟動後收到停止訊號（exit 0）。後者需要 Redis／MySQL 與監聽埠，
不在這裡測；前兩條的旗標解析可以在同一個行程裡測到「會走進哪個分支」，
而真正的 exit code 則用子行程測 —— os.Exit 無法在同一個行程裡觀察。

【為什麼 -healthz 的同行程測試只測「不健康」的子行程版本】
reportHealth 會探測 Redis，而 config.Load 的 applyDefaults 保證 RedisAddr
永遠非空（預設 127.0.0.1:6379）。也就是說「兩個相依都沒設定就跳過探測」
那條路徑在真實設定檔下不可達；在測試行程裡呼叫 -healthz 只會有兩種結果：
本機剛好有 Redis（通過）或沒有（os.Exit(1) 帶走整個測試行程）。兩者都不
適合當成同行程的斷言，因此 -healthz 只走子行程。
*/
package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// TestMainCheckModePrintsReportAndReturns 確認 -check 走完報告後正常返回
// （不啟動任何相依、不進入監聽），而且報告的頭尾都在 stdout。
func TestMainCheckModePrintsReportAndReturns(t *testing.T) {
	cfgPath := writeConfigFile(t, "SERVER_PORT=:18088\nCOOKIE_SECURE=false\n")

	out := captureStdout(t, func() {
		callMain(t, "-check", "-config", cfgPath)
	})

	if !strings.Contains(out, "設定檔檢查："+cfgPath) {
		t.Errorf("報告沒有帶出被檢查的路徑:\n%s", out)
	}
	if !strings.Contains(out, "結論：通過") {
		t.Errorf("合法設定檔的報告結論應為通過:\n%s", out)
	}
	// -check 的整個目的就是「不連任何外部服務」；報告若有連線相關的失敗
	// 訊息，代表它偏離了這個模式。
	if strings.Contains(out, "無法讀取或解析設定檔") {
		t.Errorf("合法檔案不應出現讀取失敗的結論:\n%s", out)
	}
}

// TestMainCheckModeHonoursConfigFlag 確認 -config 真的被採用，而不是永遠讀
// 預設的 config/config.conf：路徑寫錯時使用者必須在報告第一行看到。
func TestMainCheckModeHonoursConfigFlag(t *testing.T) {
	cfgPath := writeConfigFile(t, "SERVER_PORT=:18089\nCOOKIE_SECURE=false\n")

	out := captureStdout(t, func() {
		callMain(t, "-check", "-config", cfgPath)
	})
	if !strings.Contains(out, cfgPath) {
		t.Errorf("報告應顯示 -config 指定的路徑 %q:\n%s", cfgPath, out)
	}
}

/* ==========================================================================
   子行程：觀察真實的 exit code
   ========================================================================== */

// TestMainHelperProcess 是子行程的進入點，由 TestMainSubprocessExitCodes
// 以 -test.run 單獨啟動。
//
// 沒有 FORUM_MAIN_MODE 時跳過：同一個測試二進位檔也會在父行程的一般測試
// 執行中被跑到，那時它不該做任何事。
func TestMainHelperProcess(t *testing.T) {
	mode := os.Getenv("FORUM_MAIN_MODE")
	if mode == "" {
		t.Skip("只在子行程模式執行")
	}
	cfgPath := os.Getenv("FORUM_MAIN_CONFIG")

	switch mode {
	case "check-ok", "check-missing":
		os.Args = []string{"forum", "-check", "-config", cfgPath}
	case "health-unhealthy", "health-ok":
		os.Args = []string{"forum", "-healthz", "-config", cfgPath}
	case "fatal-config-missing", "fatal-logger", "fatal-validate", "fatal-redis", "fatal-mysql":
		// 走完整啟動流程的第一段：main 會依序讀設定、初始化日誌、跑
		// Validate、連 Redis、連 MySQL，並在第一個失敗點以 exit 1 結束。
		os.Args = []string{"forum", "-config", cfgPath}
	default:
		t.Fatalf("未知的子行程模式: %q", mode)
	}
	main()
}

// TestMainSubprocessExitCodes 以子行程驗證 main 的 exit code。
//
// 這三個案例刻意涵蓋「成功 0」與兩種「失敗 1」：
//   - -check 成功：0，部署端才知道設定檢查通過。
//   - -check 讀不到檔案：1，且理由在 stdout 的報告裡。
//   - -healthz 相依不健康：1，且理由在 stderr 的 [HEALTH] 行。
//
// 用 127.0.0.1:1 當 Redis 位址：loopback 上沒有任何東西會監聽那個埠，
// 連線會立刻被拒絕，因此不需要真的伺服器，也不需要等待逾時。
func TestMainSubprocessExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("子行程測試在 -short 下略過")
	}

	// fatal-mysql 這個案例需要一個「Redis 可以連上」的啟動環境：main 的
	// 順序是 Redis 先於 MySQL，沒有可用的 Redis 就永遠走不到 MySQL 的
	// 失敗點。miniredis 是行程內的測試替身，位址是 loopback，不是真伺服器。
	mini := miniredis.RunT(t)

	okCfg := writeConfigFile(t, "SERVER_PORT=:18090\nCOOKIE_SECURE=false\n")
	unhealthyCfg := writeConfigFile(t, "DB_DSN=\nREDIS_ADDR=127.0.0.1:1\n")
	loggerCfg := writeConfigFile(t, "SERVER_PORT=:18092\nLOG_FILE="+
		filepath.Join(t.TempDir(), "no-such-dir", "forum.log")+"\n")
	validateCfg := writeConfigFile(t, "COOKIE_SECURE=true\n")
	redisCfg := writeConfigFile(t, "REDIS_ADDR=127.0.0.1:1\n")
	mysqlCfg := writeConfigFile(t, "REDIS_ADDR="+mini.Addr()+"\n"+
		"DB_DSN=forum:secret@tcp(127.0.0.1:1)/forum\n")
	healthyCfg := writeConfigFile(t, "DB_DSN=\nREDIS_ADDR="+mini.Addr()+"\n")

	cases := []struct {
		name       string
		cfgPath    string
		mode       string
		wantExit   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "-check 合法設定檔",
			cfgPath:    okCfg,
			mode:       "check-ok",
			wantExit:   0,
			wantStdout: "結論：通過",
		},
		{
			name:       "-check 設定檔不存在",
			cfgPath:    filepath.Join(t.TempDir(), "no-such.conf"),
			mode:       "check-missing",
			wantExit:   1,
			wantStdout: "結論：不通過",
		},
		{
			name:       "-healthz Redis 不可用",
			cfgPath:    unhealthyCfg,
			mode:       "health-unhealthy",
			wantExit:   1,
			wantStderr: "[HEALTH] unhealthy",
		},
		{
			name:       "-healthz 相依可用",
			cfgPath:    healthyCfg,
			mode:       "health-ok",
			wantExit:   0,
			wantStdout: "[HEALTH] Redis: OK",
		},
		{
			name:       "啟動：設定檔不存在",
			cfgPath:    filepath.Join(t.TempDir(), "no-such.conf"),
			mode:       "fatal-config-missing",
			wantExit:   1,
			wantStdout: "[CONFIG] Failed to load config.conf",
		},
		{
			name:       "啟動：日誌檔無法開啟",
			cfgPath:    loggerCfg,
			mode:       "fatal-logger",
			wantExit:   1,
			wantStdout: "[LOGGER] Failed to initialize logger",
		},
		{
			name:       "啟動：Validate 不通過",
			cfgPath:    validateCfg,
			mode:       "fatal-validate",
			wantExit:   1,
			wantStdout: "設定檢查失敗",
		},
		{
			name:       "啟動：Redis 不可用",
			cfgPath:    redisCfg,
			mode:       "fatal-redis",
			wantExit:   1,
			wantStdout: "[REDIS] Connection failed",
		},
		{
			name:       "啟動：MySQL 不可用",
			cfgPath:    mysqlCfg,
			mode:       "fatal-mysql",
			wantExit:   1,
			wantStdout: "[DB] 資料庫連線開啟失敗",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMainHelperProcess$")
			cmd.Env = append(os.Environ(),
				"FORUM_MAIN_MODE="+c.mode,
				"FORUM_MAIN_CONFIG="+c.cfgPath,
			)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if ctx.Err() != nil {
				t.Fatalf("子行程逾時: %v", ctx.Err())
			}

			gotExit := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("子行程無法執行: %v（stderr=%s）", err, stderr.String())
				}
				gotExit = exitErr.ExitCode()
			}
			if gotExit != c.wantExit {
				t.Errorf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s",
					gotExit, c.wantExit, stdout.String(), stderr.String())
			}
			if c.wantStdout != "" && !strings.Contains(stdout.String(), c.wantStdout) {
				t.Errorf("stdout 缺少 %q:\n%s", c.wantStdout, stdout.String())
			}
			if c.wantStderr != "" && !strings.Contains(stderr.String(), c.wantStderr) {
				t.Errorf("stderr 缺少 %q:\n%s", c.wantStderr, stderr.String())
			}
		})
	}
}
