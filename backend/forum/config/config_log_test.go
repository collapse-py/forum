/*
日誌輪替設定的解析測試（backend/forum/config/config_log_test.go）。

LOG_MAX_SIZE_MB 的 0 有明確語意（不輪替），而負數與非數字必須被拒絕並保留
預設值 —— 三者在症狀上完全不同：

	0      → 沿用「無限追加」的舊行為（合法）
	-1     → 若被接受，maxBytes 為負代表「每行都輪替」，日誌只剩 .1 一份
	非數字  → 若被接受，int 零值會被誤解成「不輪替」

LOG_MAX_FILES 則只接受正數：0 個輪替檔等於「每次輪替直接丟掉舊檔」，
那是比不輪替更糟的資料損失。

parseNonNegativeInt 的三態在此一併釘住（它是這個檔唯一為 0 開門的解析器）。
*/

package config

import "testing"

func TestParseNonNegativeInt(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"0", 0, true},
		{"7", 7, true},
		{" 12", 0, false}, // 前導空白：strconv 拒絕，刻意不猜
		{"-1", 0, false},
		{"-0", 0, true},  // -0 數學上就是 0，是合法的不輪替
		{"abc", 0, false},
		{"", 0, false},
		{"1.5", 0, false},
	}
	for _, c := range cases {
		got, ok := parseNonNegativeInt(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("parseNonNegativeInt(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestLogRotationSettingsParse(t *testing.T) {
	// 0 是合法的「不輪替」。
	cfg := loadString(t, "LOG_MAX_SIZE_MB=0\nLOG_MAX_FILES=3\n")
	if cfg.LogMaxSizeMB != 0 {
		t.Errorf("LOG_MAX_SIZE_MB=0 被改成 %d（0 是合法的不輪替語意）", cfg.LogMaxSizeMB)
	}
	if cfg.LogMaxFiles != 3 {
		t.Errorf("LOG_MAX_FILES = %d, want 3", cfg.LogMaxFiles)
	}

	// 正常值。
	cfg = loadString(t, "LOG_MAX_SIZE_MB=7\nLOG_MAX_FILES=10\n")
	if cfg.LogMaxSizeMB != 7 || cfg.LogMaxFiles != 10 {
		t.Errorf("解析結果 = (%d, %d), want (7, 10)", cfg.LogMaxSizeMB, cfg.LogMaxFiles)
	}

	// 負數與非數字：保留預設值，而不是把「每行都輪替」或「不輪替」當成結果。
	cfg = loadString(t, "LOG_MAX_SIZE_MB=-1\nLOG_MAX_FILES=abc\n")
	if cfg.LogMaxSizeMB != 50 {
		t.Errorf("LOG_MAX_SIZE_MB=-1 未被拒絕，得到 %d（預設 50）", cfg.LogMaxSizeMB)
	}
	if cfg.LogMaxFiles != 5 {
		t.Errorf("LOG_MAX_FILES=abc 未被拒絕，得到 %d（預設 5）", cfg.LogMaxFiles)
	}

	// LOG_MAX_FILES=0：保留 0 個輪替檔等於每次輪換直接丟舊檔，必須被拒絕。
	cfg = loadString(t, "LOG_MAX_FILES=0\n")
	if cfg.LogMaxFiles != 5 {
		t.Errorf("LOG_MAX_FILES=0 未被拒絕，得到 %d（那是比不輪替更糟的資料損失）", cfg.LogMaxFiles)
	}
}
