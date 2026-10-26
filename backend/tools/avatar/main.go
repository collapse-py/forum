/*
avatar 讓每一位使用者都有頭像。

【它解決什麼】

forum_profiles.avatar_url 為空時，前端只能退回顯示暱稱首字 —— 於是整頁動態裡
大部分的人都沒有圖。這個工具一次補齊：為每位缺頭像的使用者產生一張圖、
上傳到檔案伺服器，再把檔名寫回 avatar_url。

【為什麼是本機產生而不是頭像服務或 Gravatar】

頭像來源若是第三方，就等於把使用者的 email（或其雜湊）送到外部服務，而這個
專案的信箱是 Google 登入的真實信箱。本機產生的圖是 email 的決定性函數：同一
位使用者每次跑出來都是同一張，不需資料庫多存一份對應，也不需網路。

圖形選 5×5 鏡像方塊（identicon）：純 Go 標準庫就能畫（image + image/png），
不新增任何依賴；每個 email 的雜湊決定配色與方塊分佈，因此 300 位使用者不會
看起來一模一樣。

【契約】
  - 檔案伺服器實際寫入的檔名由它自己生成（UUID），呼叫端只能拿回什麼就存什麼
    —— 因此不能先決定檔名再上傳。
  - avatar_url 只存檔名，讀取時由 forumImageURL 以 FILES_SERVER_PUBLIC_URL 現組
    （見 docs/DATABASE.md）。
  - 副檔名限 .png（白名單與 files_server upload.allowed_files 相同，svg 被刻意
    排除，因為 SVG 可內嵌腳本）。
  - forum_media 必須一併插入：那一列是「這個檔名是誰上傳的」，少了它，
    使用者之後就無法透過刪除端點回收自己的頭像檔。

【順序】
  先上傳、後寫入資料庫。反過來的話，資料庫會先指向一個還不存在的檔案，而中間
  失敗的症狀是「頭像 404」。上傳成功但資料庫失敗只留下沒有人引用的檔案，工具
  會把檔名列出來以便清理。

用法：

	DB_DSN='...' FILES_SERVER_URL=http://host:7070 FILES_SERVER_TOKEN='...' \
	  go run ./tools/avatar -dry-run
*/
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dsnFlag  = flag.String("dsn", envOr("DB_DSN", ""), "MySQL DSN（預設取環境變數 DB_DSN）")
	filesURL = flag.String("files-url", envOr("FILES_SERVER_URL", ""),
		"檔案伺服器上傳位址（預設取環境變數 FILES_SERVER_URL；要用內部位址，公開網域擋 POST）")
	tokenFlag = flag.String("upload-token", envOr("FILES_SERVER_TOKEN", ""),
		"上傳 token（預設取環境變數 FILES_SERVER_TOKEN）")
	sizeFlag = flag.Int("size", 256, "頭像邊長（像素）")
	limit    = flag.Int("limit", 0, "只處理前 N 位（0＝全部）")
	prefix   = flag.String("prefix", "", "只處理 email 以此開頭的使用者")
	force    = flag.Bool("force", false, "連已經有頭像的使用者一起換掉（預設只補空的）")
	noCreate = flag.Bool("no-create-missing", false,
		"不為沒有 forum_profiles 列的使用者補列（預設會補，因為頭像只能存在那一列上）")
	dryRun = flag.Bool("dry-run", false, "只列出會處理誰，不上傳也不寫入")
	// 兩個驗證用旗標：把產生出來的頭像存成檔案，讓人用眼看一次圖形是否合理，
	// 而不是只能相信程式說「我畫好了」。指定 -out 時不需要 DSN 與檔案伺服器。
	outFile       = flag.String("out", "", "把頭像寫進這個 PNG 檔後結束（驗證用）")
	previewEmail  = flag.String("preview-email", "seed0001@example.test", "-out 時用來產生頭像的 email")
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// target 是一位需要頭像的使用者。
type target struct {
	email      string
	hasProfile bool
	nickname   string // 既有暱稱；沒有 profile 列時為空
}

// uploaded 是完成上傳、等待寫回資料庫的一列。
type uploaded struct {
	target
	filename string
}

func main() {
	flag.Parse()
	ctx := context.Background()

	if *outFile != "" {
		pngBytes, err := avatarPNG(*previewEmail, *sizeFlag)
		if err != nil {
			fail("產生頭像", err)
		}
		if err := os.WriteFile(*outFile, pngBytes, 0o644); err != nil {
			fail("寫入 "+*outFile, err)
		}
		fmt.Printf("已寫入 %s（%d bytes，email=%s）\n", *outFile, len(pngBytes), *previewEmail)
		return
	}
	if strings.TrimSpace(*dsnFlag) == "" {
		fmt.Fprintln(os.Stderr, "缺少 DSN：請給 -dsn 或設定 DB_DSN")
		os.Exit(2)
	}
	if !*dryRun {
		if strings.TrimSpace(*filesURL) == "" {
			fmt.Fprintln(os.Stderr, "缺少檔案伺服器位址：請給 -files-url 或設定 FILES_SERVER_URL")
			os.Exit(2)
		}
		if strings.TrimSpace(*tokenFlag) == "" {
			fmt.Fprintln(os.Stderr, "缺少上傳 token：請給 -upload-token 或設定 FILES_SERVER_TOKEN")
			os.Exit(2)
		}
	}

	db, err := sql.Open("mysql", *dsnFlag)
	if err != nil {
		fail("開啟連線", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		fail("連線失敗", err)
	}

	targets, err := loadTargets(ctx, db)
	if err != nil {
		fail("讀取使用者", err)
	}
	if *limit > 0 && len(targets) > *limit {
		targets = targets[:*limit]
	}

	fmt.Printf("待處理使用者：%d 位", len(targets))
	if *dryRun {
		fmt.Print("（dry-run）\n")
		for i, t := range targets {
			if i >= 5 && len(targets) > 10 {
				fmt.Printf("  … 省略 %d 位\n", len(targets)-5)
				break
			}
			state := "已有 profile，缺頭像"
			if !t.hasProfile {
				state = "沒有 profile 列（會一併補列）"
			}
			fmt.Printf("  %s  %s\n", t.email, state)
		}
		return
	}
	if len(targets) == 0 {
		fmt.Println("沒有需要處理的使用者。")
		return
	}

	// 既有暱稱：沒有 profile 列的使用者需要一個不撞唯一索引的暱稱。
	taken, err := loadNicknames(ctx, db)
	if err != nil {
		fail("讀取既有暱稱", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	done := make([]uploaded, 0, len(targets))
	for i, t := range targets {
		pngBytes, err := avatarPNG(t.email, *sizeFlag)
		if err != nil {
			fail("產生頭像 "+t.email, err)
		}
		name, err := upload(client, *filesURL, *tokenFlag, fmt.Sprintf("avatar-%d.png", i), pngBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "已上傳 %d 個後中斷；未寫入資料庫的檔案如下（可略過，之後重跑會補上）：\n", len(done))
			for _, d := range done {
				fmt.Fprintln(os.Stderr, "  "+d.filename)
			}
			fail("上傳 "+t.email, err)
		}
		done = append(done, uploaded{target: t, filename: name})
		if (i+1)%50 == 0 {
			fmt.Printf("  已上傳 %d/%d\n", i+1, len(targets))
		}
	}

	if err := persist(ctx, db, done, taken); err != nil {
		fmt.Fprintln(os.Stderr, "資料庫寫入失敗。已上傳但未被引用的檔案：")
		for _, d := range done {
			fmt.Fprintln(os.Stderr, "  " + d.filename)
		}
		fail("寫入資料庫", err)
	}

	created := 0
	for _, d := range done {
		if !d.hasProfile {
			created++
		}
	}
	fmt.Printf("完成：%d 位使用者拿到頭像（其中 %d 位是新建的 profile 列）\n", len(done), created)
}

// ---------------------------------------------------------------------------
// 讀取
// ---------------------------------------------------------------------------

// loadTargets 找出「需要頭像」的使用者：
//   - forum_users 有、forum_profiles 沒有 → 需要補列（否則頭像無處可放）
//   - 有 profile 但 avatar_url 為空 → 只補頭像
//   - -force 時忽略 avatar_url 是否為空
func loadTargets(ctx context.Context, db *sql.DB) ([]target, error) {
	avatarCond := "(p.author_email IS NULL OR p.avatar_url = '')"
	if *force {
		avatarCond = "1=1"
	}
	profileCond := ""
	if *noCreate {
		profileCond = " AND p.author_email IS NOT NULL"
	}
	q := `SELECT u.email, p.author_email IS NOT NULL, COALESCE(p.nickname, '')
	      FROM forum_users u
	      LEFT JOIN forum_profiles p ON p.author_email = u.email
	      WHERE ` + avatarCond + profileCond
	var args []any
	if p := strings.TrimSpace(*prefix); p != "" {
		q += " AND u.email LIKE ?"
		args = append(args, p+"%")
	}
	q += " ORDER BY u.email"

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.email, &t.hasProfile, &t.nickname); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func loadNicknames(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT nickname FROM forum_profiles")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[n] = true
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// 寫入
// ---------------------------------------------------------------------------

func persist(ctx context.Context, db *sql.DB, done []uploaded, taken map[string]bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	now := time.Now()
	for _, d := range done {
		if d.hasProfile {
			// 只補空的頭像（除非 -force）：條件寫進 SQL，讓「跑兩次」不會互相覆寫。
			cond := "avatar_url = ''"
			if *force {
				cond = "1=1"
			}
			res, err := tx.ExecContext(ctx,
				"UPDATE forum_profiles SET avatar_url = ?, updated_at = ? WHERE author_email = ? AND "+cond,
				d.filename, now, d.email)
			if err != nil {
				return fmt.Errorf("更新 %s: %w", d.email, err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return fmt.Errorf("%s 的 avatar_url 在這次執行中被改掉了（條件 %s）", d.email, cond)
			}
		} else {
			nick := uniqueNickname(d.nickname, d.email, taken)
			_, err := tx.ExecContext(ctx,
				`INSERT INTO forum_profiles
					(author_email, public_key, nickname, bio, avatar_url, updated_at)
				 VALUES (?, ?, ?, '', ?, ?)`,
				d.email, publicKeyFor(d.email), nick, d.filename, now)
			if err != nil {
				return fmt.Errorf("建立 %s 的 profile: %w", d.email, err)
			}
			taken[nick] = true
		}
		// 所有權紀錄：少了它，這個檔案之後沒有任何人「擁有」，
		// 使用者的刪除端點會拒絕回收它（見 forum_media_handlers 的所有權模型）。
		if _, err := tx.ExecContext(ctx,
			"INSERT IGNORE INTO forum_media (filename, owner_email, created_at) VALUES (?, ?, ?)",
			d.filename, d.email, now); err != nil {
			return fmt.Errorf("記錄 %s 的檔案所有權: %w", d.email, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// uniqueNickname 為沒有 profile 列的使用者取一個不撞唯一索引的暱稱。
// 基底是 email 的 @ 前段（人類看得懂），撞名時依次加序號。
func uniqueNickname(existing, email string, taken map[string]bool) string {
	local := email
	if at := strings.Index(email, "@"); at > 0 {
		local = email[:at]
	}
	base := truncateRunes(strings.TrimSpace(local), 20)
	if base == "" {
		base = "user"
	}
	cand := base
	for i := 2; taken[cand]; i++ {
		cand = truncateRunes(base, 24) + fmt.Sprint(i)
	}
	return cand
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

func publicKeyFor(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// 圖片
// ---------------------------------------------------------------------------

// avatarPNG 產生一張由 email 決定的 identicon。
//
// 輸出永遠是同一張：同一個 email 重跑會產生位元相同的 PNG，因此重跑只是把
// 同一張圖換一個檔名放回去，不會造成「每個人每次長得不一樣」。
func avatarPNG(email string, size int) ([]byte, error) {
	if size < 32 {
		size = 32
	}
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))

	// 配色：主色由雜湊決定色相，底色是它的淡版、方塊是鄰近色相的深版 ——
	// 同色系比「隨機兩色」穩定，縮成 24px 的小圓時也不會糊成一團。
	base := int(h[0]) * 360 / 256
	bg := hsl(float64(base), 0.55, 0.90)
	fg := hsl(float64((base+24)%360), 0.60, 0.45)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fill(img, bg)

	const grid = 5
	cell := float64(size) / grid
	pad := cell * 0.08
	// 5×5 但只決定左半邊三欄，右邊鏡像 —— 對稱的圖在小尺寸下比較像「一個標誌」，
	// 而非雜訊。
	bits := []byte{h[1], h[2], h[3], h[4], h[5]}
	for y := 0; y < grid; y++ {
		for x := 0; x < 3; x++ {
			if bits[y]&(1<<uint(4-x)) == 0 {
				continue
			}
			for _, cx := range mirrorX(x, grid) {
				x0 := int(float64(cx)*cell + pad)
				y0 := int(float64(y)*cell + pad)
				x1 := int(float64(cx+1)*cell - pad)
				y1 := int(float64(y+1)*cell - pad)
				drawRect(img, x0, y0, x1, y1, fg)
			}
		}
	}

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mirrorX(x, grid int) []int {
	return []int{x, grid - 1 - x}
}

func fill(img *image.RGBA, c color.RGBA) {
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	b := img.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// hsl 轉 sRGB（h 0–360，s/l 0–1）。
func hsl(h, s, l float64) color.RGBA {
	h = h - 360*float64(int(h/360))
	c := (1 - abs(2*l-1)) * s
	x := c * (1 - abs(mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func mod(v, n float64) float64 {
	v = v - n*float64(int(v/n))
	if v < 0 {
		v += n
	}
	return v
}

// ---------------------------------------------------------------------------
// 上傳
// ---------------------------------------------------------------------------

// upload 把 PNG 送到檔案伺服器，回傳**它生成的**檔名。
//
// 檔名刻意不自己決定：newFileName 是檔案伺服器的職責，呼叫端搶先取名只會
// 讓「寫進資料庫的檔名」與「磁碟上的檔名」有機會不一致。
func upload(client *http.Client, base, token, filename string, data []byte) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(base, "/") + "/upload"
	req, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Upload-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("檔案伺服器回 %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("解析上傳回應: %w", err)
	}
	if out.URL == "" {
		return "", fmt.Errorf("上傳回應沒有 url：%s", strings.TrimSpace(string(payload)))
	}
	// url 可能是 "/files/x.png" 或完整網址；資料庫只需要檔名。
	name := path.Base(mustPath(out.URL))
	if name == "." || name == "/" || name == "" {
		return "", fmt.Errorf("無法從 %q 取出檔名", out.URL)
	}
	return name, nil
}

func mustPath(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		return u.Path
	}
	return raw
}

// ---------------------------------------------------------------------------

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s 失敗：%v\n", what, err)
	os.Exit(1)
}

// sortTargets 讓輸出穩定（loadTargets 已排序，這裡保留給測試與重構）。
func sortTargets(ts []target) {
	sort.Slice(ts, func(i, j int) bool { return ts[i].email < ts[j].email })
}
