package data

import (
	"context"
	"errors"
	"strings"
	"testing"
)

/* ==========================================================================
   BackfillProfileAuthorKeys（第 39 步的 public_key_hmac 回填）
   ========================================================================== */

const authorKeySelectSQL = "SELECT author_email FROM forum_profiles WHERE public_key_hmac = ''"
const authorKeyUpdateSQL = "UPDATE forum_profiles SET public_key_hmac = ? WHERE author_email = ? AND public_key_hmac = ''"

func TestBackfillProfileAuthorKeysNilDatabase(t *testing.T) {
	if err := BackfillProfileAuthorKeys(context.Background(), nil, func(string) string { return "k" }); err == nil {
		t.Fatal("nil database must be refused")
	}
}

// keyFor == nil 是「沒有設秘密」的部署：一列都不動，欄位保持空字串是正確狀態。
func TestBackfillProfileAuthorKeysNilKeyFor(t *testing.T) {
	db, src := covDataDB(t)
	if err := BackfillProfileAuthorKeys(context.Background(), db, nil); err != nil {
		t.Fatalf("BackfillProfileAuthorKeys(nil keyFor): %v", err)
	}
	if src.countSeen(authorKeySelectSQL) != 0 {
		t.Fatalf("must not read profiles without a secret, saw %d", src.countSeen(authorKeySelectSQL))
	}
	if src.countSeen(authorKeyUpdateSQL) != 0 {
		t.Fatal("must not write profiles without a secret")
	}
}

// 沒有待回填的列時不回空值以外的東西：既有值不被靜默覆蓋（換秘密的代價是
// 部署者明示的，不由這一步悄悄改寫）。
func TestBackfillProfileAuthorKeysSkipsFilledRows(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{substr: authorKeySelectSQL, cols: []string{"author_email"}})

	if err := BackfillProfileAuthorKeys(context.Background(), db, func(email string) string {
		t.Fatalf("keyFor must not run when there is nothing to fill")
		return ""
	}); err != nil {
		t.Fatalf("BackfillProfileAuthorKeys: %v", err)
	}
	if src.countSeen(authorKeyUpdateSQL) != 0 {
		t.Fatal("must not write when the batch is empty")
	}
}

func TestBackfillProfileAuthorKeysWritesEveryRow(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: authorKeySelectSQL,
		cols:   []string{"author_email"},
		rows: [][]covDataValue{
			{"one@example.com"},
			{"two@example.com"},
			{"three@example.com"},
		},
	})

	key := func(email string) string { return "hmac-" + email }
	if err := BackfillProfileAuthorKeys(context.Background(), db, key); err != nil {
		t.Fatalf("BackfillProfileAuthorKeys: %v", err)
	}
	if got := src.countSeen(authorKeySelectSQL); got != 1 {
		t.Fatalf("select round trips = %d, want 1", got)
	}
	// 預言式語句：同一條 UPDATE 對每列各執行一次（SQL 文字不變，參數帶值）。
	if got := src.countSeen(authorKeyUpdateSQL); got != 3 {
		t.Fatalf("update executions = %d, want 3", got)
	}
	// 秘密不在 SQL 文字裡（它會進 general log 與日誌）。
	for _, query := range src.seen {
		if strings.Contains(query, "hmac-") || strings.Contains(query, "secret") {
			t.Fatalf("SQL must not carry derived keys or secrets: %s", query)
		}
	}
}

// 滿一批時繼續下一批：記憶體與單次往返都有上界。
func TestBackfillProfileAuthorKeysLoopsOverBatches(t *testing.T) {
	db, src := covDataDB(t)

	rows := make([][]covDataValue, authorKeyBackfillBatch)
	for i := range rows {
		rows[i] = []covDataValue{"user" + string(rune('a'+i%26)) + "@example.com"}
	}
	src.addQuery(covDataQueryRule{
		substr:  authorKeySelectSQL,
		cols:    []string{"author_email"},
		rows:    rows,
		consume: true,
	})

	if err := BackfillProfileAuthorKeys(context.Background(), db, func(email string) string {
		return "hmac-" + email
	}); err != nil {
		t.Fatalf("BackfillProfileAuthorKeys: %v", err)
	}
	if got := src.countSeen(authorKeySelectSQL); got != 2 {
		t.Fatalf("select round trips = %d, want 2", got)
	}
	if got := src.countSeen(authorKeyUpdateSQL); got != authorKeyBackfillBatch {
		t.Fatalf("update executions = %d, want %d", got, authorKeyBackfillBatch)
	}
}

// 派生回空字串只可能是 keyFor 的 bug；寧可停下來也不要寫進查不到的空值。
// 錯誤訊息刻意不含 email：那會跟著日誌把個資寫出去。
func TestBackfillProfileAuthorKeysRejectsEmptyDerivedKey(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: authorKeySelectSQL,
		cols:   []string{"author_email"},
		rows:   [][]covDataValue{{"private@example.com"}},
	})

	err := BackfillProfileAuthorKeys(context.Background(), db, func(string) string { return "" })
	if err == nil {
		t.Fatal("empty derived key must stop the backfill")
	}
	if !strings.Contains(err.Error(), "derive author key returned empty value") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "private@example.com") {
		t.Fatalf("error must not leak the email: %v", err)
	}
	if src.countSeen(authorKeyUpdateSQL) != 0 {
		t.Fatal("must not write an empty key")
	}
}

func TestBackfillProfileAuthorKeysPropagatesWriteError(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: authorKeySelectSQL,
		cols:   []string{"author_email"},
		rows:   [][]covDataValue{{"one@example.com"}},
	})
	src.addExec(covDataExecRule{substr: "UPDATE forum_profiles SET public_key_hmac", err: errors.New("disk full")})

	err := BackfillProfileAuthorKeys(context.Background(), db, func(email string) string { return "hmac-" + email })
	if err == nil {
		t.Fatal("write failure must surface")
	}
	if !strings.Contains(err.Error(), "write author key") || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("error = %v", err)
	}
}

func TestBackfillProfileAuthorKeysPropagatesReadError(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{substr: authorKeySelectSQL, err: errors.New("connection reset")})

	err := BackfillProfileAuthorKeys(context.Background(), db, func(email string) string { return "hmac-" + email })
	if err == nil {
		t.Fatal("read failure must surface")
	}
	if !strings.Contains(err.Error(), "read profiles without author key") {
		t.Fatalf("error = %v", err)
	}
	if src.countSeen(authorKeyUpdateSQL) != 0 {
		t.Fatal("must not write after a failed read")
	}
}

// 列數不符時 Scan 報錯：欄位被改（第 39 步以外的迁移動了這張表）不會靜默略過。
func TestBackfillProfileAuthorKeysPropagatesPrepareError(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: authorKeySelectSQL,
		cols:   []string{"author_email"},
		rows:   [][]covDataValue{{"one@example.com"}},
	})
	src.addPrepare(covDataExecRule{substr: "UPDATE forum_profiles SET public_key_hmac", err: errors.New("no such column: public_key_hmac")})

	err := BackfillProfileAuthorKeys(context.Background(), db, func(email string) string { return "hmac-" + email })
	if err == nil {
		t.Fatal("prepare failure must surface")
	}
	if !strings.Contains(err.Error(), "prepare author key update") {
		t.Fatalf("error = %v", err)
	}
}

// 讀列中途斷掉（連線重置）要報錯，不能把「讀到一半」當成「讀完了」而回 nil。
func TestBackfillProfileAuthorKeysPropagatesRowsErr(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr:  authorKeySelectSQL,
		cols:    []string{"author_email"},
		rows:    [][]covDataValue{{"one@example.com"}},
		iterErr: errors.New("connection reset mid-result"),
	})

	err := BackfillProfileAuthorKeys(context.Background(), db, func(email string) string { return "hmac-" + email })
	if err == nil {
		t.Fatal("mid-result error must not be treated as a clean finish")
	}
	if !strings.Contains(err.Error(), "scan profile emails") {
		t.Fatalf("error = %v", err)
	}
	if src.countSeen(authorKeyUpdateSQL) != 0 {
		t.Fatal("must not write after a mid-result error")
	}
}

func TestBackfillProfileAuthorKeysPropagatesScanError(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: authorKeySelectSQL,
		cols:   []string{"author_email", "unexpected"},
		rows:   [][]covDataValue{{"one@example.com", "x"}},
	})

	err := BackfillProfileAuthorKeys(context.Background(), db, func(email string) string { return "hmac-" + email })
	if err == nil {
		t.Fatal("scan failure must surface")
	}
	if !strings.Contains(err.Error(), "scan profile email") {
		t.Fatalf("error = %v", err)
	}
}

func TestBackfillProfileAuthorKeysHonoursContext(t *testing.T) {
	db, src := covDataDB(t)
	src.addQuery(covDataQueryRule{
		substr: authorKeySelectSQL,
		cols:   []string{"author_email"},
		rows:   [][]covDataValue{{"one@example.com"}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := BackfillProfileAuthorKeys(ctx, db, func(email string) string { return "hmac-" + email })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if src.countSeen(authorKeyUpdateSQL) != 0 {
		t.Fatal("must not write after cancellation")
	}
}

/* ==========================================================================
   第 39 步的註解要說明「值不在 SQL 裡算」：派生只來自 keyFor 一個來源
   ========================================================================== */

func TestAuthorKeyBackfillBatchIsBounded(t *testing.T) {
	if authorKeyBackfillBatch <= 0 || authorKeyBackfillBatch > 1000 {
		t.Fatalf("authorKeyBackfillBatch = %d, want a bounded batch (1..1000)", authorKeyBackfillBatch)
	}
}
