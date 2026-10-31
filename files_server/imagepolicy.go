/*
imagepolicy.go 是上傳內容的圖片正規化層：以「檔案的位元組」而不是「副檔名」判斷這是不是
圖片，重新編碼以移除 EXIF／中繼資料，並把輸出尺寸收斂到設定給定的上限。

【為什麼獨立成檔】
這三件事（判斷型別、剝中繼資料、收斂尺寸）是一套**圖片格式**的知識，而不是 HTTP 的知識。
放進 server.go 會讓那個檔同時承擔「路由與授權」和「JPEG 的 APP1 區塊怎麼走」，而後者的
細節量比前者還大。分開之後 server.go 只在 handleUpload 裡問一句「這份位元組能存嗎」。

【對外介面（本檔全部匯出符號）】
ImagePolicy：一份正規化政策（是否重編碼、輸出上限、來源解析度上限、JPEG 品質）。
imageKind / detectImageKind / isImageExtension：以內容判斷型別，以及它對應的副檔名。
normalizeImage：把上傳的位元組轉成要寫進儲存的那份位元組。
本檔不碰 HTTP、不碰 Redis、不碰磁碟。

【主要依賴】
只有標準函式庫的 image／image/jpeg／image/png／image/gif。刻意不引入 golang.org/x/image：
  - WebP 的**解碼器**不在標準庫裡（只有 x/image 有），因此 WebP 走「容器層剝中繼資料」而
    不重編碼。代價寫在關鍵決策第 4 點。
  - 這個模組的相依標準是「能用預設值跑起來的站不需要額外安裝任何東西」（見 config.go 的檔頭）。

【關鍵設計決策】
 1. **型別由位元組決定，不由副檔名決定。** 副檔名是上傳者可控的字串，而 `[]byte("fake-jpeg-bytes")`
    從前會被當成圖片存進儲存（實測：2026-10-29 的 `server_test.go` 就是這樣跑的）。那不是圖片，
    它只是被掛在圖片的 URL 形狀上 —— 瀏覽器拿到一個 Content-Type 說「這是圖」的 non-圖。
    收斂之後，輸出副檔名由**檢測到的型別**產生（見 server.go 的 handleUpload），所以
    「檔名叫 a.png 但內容是 JPEG」會存成 `.jpg`，而不是存一份被誤標的檔案。
 2. **重編碼是移除中繼資料唯一可靠的做法。** EXIF（含 GPS、相機型號、拍攝時間、軟體名稱）在
    JPEG 的 APP1 區塊裡，PNG 在 tEXt／iTXt／eXIf 區塊裡。逐區塊跳過再存回原格式需要對每種
    格式維護一份「哪些區塊是隱私」的清單，而那份清單會隨著軟體新增區塊而過期。重新編碼把
    「保留什麼」變成一個短白名單：只有像素與格式本身。
    代價要清楚寫下來：**原檔不再可取回**，且重編碼會改變雜湊與檔大小；對一個論壇而言
    「原檔」從本來就沒有必要（讀取端只讀這份存下来的檔案）。
 3. **來源解析度上限，而不是「先解碼再縮小」。** 一個 1 MB 的 PNG 可以宣告 20000×20000 的
    畫布，解碼後是 1.6 GB 的 RGBA —— 那是解壓炸彈的形狀，而 `upload.max_size` 完全管不到它。
    因此先讀**標頭**的尺寸（`image.DecodeConfig` 對 JPEG／PNG／GIF 都不解碼），超過
    `max_source_pixels` 就直接拒收（400），而不是解開再縮小。
    拒收而非縮小的理由是：縮小這個動作本身要先完整解碼，而完整解碼正是這條路徑唯一的
    記憶體尖峰來源（見 config.go 對 `[upload] max_source_pixels` 的說明）。
 4. **GIF 與 WebP 不重編碼，只剝容器層的中繼資料。** 重編碼 GIF 會把動畫變成單影格
    （`image/gif` 只解第一影格），重編碼 WebP 需要外部相依。兩者的中繼資料都在容器區塊裡
    （GIF 的 Comment／Application 區塊、WebP 的 `EXIF` 區塊），因此逐區塊挑掉就够了：
    動畫、影格順序、無損編碼都原樣保留。GIF 那一邊只丟 Comment 與非 NETSCAPE 的 Application
    —— Plain Text（顯示用的文字）與 Graphic Control（影格延遲）保留，因為它們是內容而不是身分。
    代價是這兩種格式**不會被縮小** —— 超過輸出上限時
    被拒（400）而不是被縮小，這與 JPEG／PNG 的行為不同，已寫在 docs/CONFIGURATION.md。
 5. **JPEG 的品質是設定值，PNG 一律無損。** PNG 是有損轉無損沒有意義的那一種（格式沒有
    「品質」這個維度），因此不提供 `png_quality`：多一個設定鍵但只有一個能用的值，比沒有
    這個鍵更糟。
 6. **中繼資料的判別是「白名單」而不是「黑名單」。** 未知區塊一律丟掉（WebP 的 `VPRM` 這類
    私有區塊、GIF 的非 NETSCAPE Application 區塊）。用黑名單（只挑掉 `EXIF`）的失效方式是
    「某個軟體潢了一個區塊名，隱私又回來了」—— 而那種失效沒有任何症狀。
*/
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	// 這個 import 只是為了**註冊** GIF 的解碼器：image.DecodeConfig 走全域註冊表，
	// 少了它，GIF 會得到 "image: unknown format" —— 而那個錯誤看起來像「這個檔案壞了」，
	// 實際上是這個檔忘了 import。像素資料由 stripGIFMetadata 自己走，不用 gif.Decode。
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"strings"
)

// imageKind 是一種圖片格式。值是內部標識，不會出現在任何回應裡 —— 對外的形貌是副檔名。
type imageKind string

const (
	kindJPEG imageKind = "jpeg"
	kindPNG  imageKind = "png"
	kindGIF  imageKind = "gif"
	kindWebP imageKind = "webp"
)

// imageExtensions 是四種圖片格式對應的副檔名。刻意把 JPEG 收斂成 `.jpg` 一個值：
// 白名單裡 `.jpg` 與 `.jpeg` 都放行（兩者都是人手上常見的寫法），但**輸出**只需要一種，
// 而 `.jpg` 是後端 validForumImageFileName、前端 `<img>` 與既有檔案最常見的那一個。
var imageExtensions = map[imageKind]string{
	kindJPEG: ".jpg",
	kindPNG:  ".png",
	kindGIF:  ".gif",
	kindWebP: ".webp",
}

// extensionFor 回傳格式對應的輸出副檔名；未知格式回空字串（呼叫端據此拒收）。
func extensionFor(kind imageKind) string {
	return imageExtensions[kind]
}

// isImageExtension 判斷副檔名是不是一種圖片。
//
// 它決定的不是「能不能存」——那是 upload.allowed_files 的工作 —— 而是「這個上傳要不要走
// 正規化」。白名單裡的其他副檔名（音訊）走原樣存檔，見 server.go 的 handleUpload。
func isImageExtension(ext string) bool {
	ext = strings.ToLower(ext)
	for _, e := range imageExtensions {
		if e == ext {
			return true
		}
	}
	return false
}

var (
	pngSignature  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	gifHeader     = []byte{'G', 'I', 'F', '8'}
	webpContainer = []byte("RIFF")
	webpFourCC    = []byte("WEBP")
	webpVP8Sync   = []byte{0x90, 0x64, 0x20}

	// 刻意宣告成 byte：它們是「一個位元組的區塊類型」，而寫成 int（Go 的無型別常數）
	// 會在 `switch data[pos]` 上得到 "mismatched types int and byte" —— 那個錯誤
	// 看起來像「這個變數的類型選錯了」，而不是「我寫了一個無型別常數」。
	gifTrailer    = byte(0x3b)
	gifExtension  = byte(0x21)
	gifImageDesc  = byte(0x2c)
	gifCommentExt = byte(0xfe)
	gifAppExt     = byte(0xff)
	// Plain Text（0x01）是**要顯示的**文字，不是中繼資料，因此保留；它有一個固定 13 位元組
	// 的欄位區（見 stripGIFMetadata 那一段）。Graphic Control（0xf9）不需要對應的處理：
	// 它的固定區塊是「一個長度位元組（恒為 4）＋ 4 位元組」，那與「長度前綴的子區塊」
	// 是同一個形狀，因此 gifSubBlocks 天生就走對 —— 這是巧合，但不是錯誤的巧合（規格說
	// 那個長度必須是 4，而所有寫 GIF 的軟體都寫 4）。
	gifPlainTextExt = byte(0x01)
	webpVP8LSig     = byte(0x2f)

	// WebP 裡被保留的非圖片區塊：動畫（ANIM/ANFO/FRAM）與色彩描述（ICCP）。
	webpLoopChunks = map[string]bool{"ANIM": true, "ANFO": true, "FRAM": true, "ICCP": true}
)

// errNotAnImage 是「這份位元組不是這四種圖片之一」。
//
// 它刻意是哨兵錯誤而不是 fmt.Errorf 的訊息：呼叫端要把它導向 400，而「圖片格式」的細節
// 不該出現在給使用者的回應裡（後端把非 2xx 統一成 502，見 forum_handlers.go）。
var errNotAnImage = errors.New("內容不是合法的圖片")

// ImagePolicy 是 handleUpload 用到的正規化政策。值由 config.go 的 Config.imagePolicy 組出，
// 因此上限的兜底與範圍收斂只在 config 那一處發生 —— 本檔假拿到的是可用的值。
type ImagePolicy struct {
	// Reencode 為 false 時完全不做正規化：位元組原樣存檔，中繼資料（含 JPEG 的 GPS）保留。
	// 那是一個需要被明確寫出來的決定，因此 main.go 的 imagePolicyWarning 會在啟動時講它。
	Reencode bool
	// MaxWidth／MaxHeight 是**輸出**的尺寸上限（像素）。JPEG／PNG 超過時被等比縮小；
	// GIF／WebP 超過時被拒（見檔頭第 4 點）。
	MaxWidth int
	MaxHeight int
	// MaxSourcePixels 是**來源**的解析度上限（寬×高）。超過一律拒收，不縮小 —— 理由見檔頭第 3 點。
	MaxSourcePixels int
	// JPEGQuality 是 jpeg.Encode 的品質（0–100）。PNG 沒有對應的設定（見檔頭第 5 點）。
	JPEGQuality int
}

// detectImageKind 以位元組而不是副檔名判斷圖片格式。回 false 代表「不是這四種之一」。
//
// 判斷只看簽章的前幾個位元組，因此它不是「驗證這個檔案完好」——那是解碼的工作。它的用途
// 是**拒掉被誤標成圖片的非圖片內容**（HTML、腳本、音訊），也就是檔頭第 1 點。
func detectImageKind(data []byte) (imageKind, bool) {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return kindJPEG, true
	case len(data) >= 8 && bytes.Equal(data[:8], pngSignature):
		return kindPNG, true
	case len(data) >= 6 && bytes.Equal(data[:4], gifHeader) && (data[4] == '7' || data[4] == '9') && data[5] == 'a':
		// GIF87a 與 GIF89a。其他 GIF8xx 變體不存在於瀏覽器裡，因此不放行。
		return kindGIF, true
	case len(data) >= 12 && bytes.Equal(data[:4], webpContainer) && bytes.Equal(data[8:12], webpFourCC):
		return kindWebP, true
	}
	return "", false
}

// imageBounds 讀來源的尺寸。只讀標頭：
//   - JPEG／PNG／GIF 走 image.DecodeConfig，它不解碼像素（這正是它存在的理由）
//   - WebP 走 parseWebP（標準庫沒有 WebP 解碼器，因此沒有 DecodeConfig 可用）
func imageBounds(kind imageKind, data []byte) (int, int, error) {
	switch kind {
	case kindWebP:
		info, err := parseWebP(data)
		if err != nil {
			return 0, 0, err
		}
		return info.width, info.height, nil
	default:
		// DecodeConfig 回 (Config, 格式名, error)。格式名是我們已經由簽章檢測過的，
		// 因此這裡丟棄它；留下它會讓讀者以為這個檔依賴解碼器回報的格式，而那個
		// 回報在「簽章對了但標頭壞了」的檔案上可能與 detectImageKind 不一致。
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return 0, 0, fmt.Errorf("讀取圖片尺寸失敗: %w", err)
		}
		if cfg.Width <= 0 || cfg.Height <= 0 {
			return 0, 0, fmt.Errorf("圖片尺寸無效: %dx%d", cfg.Width, cfg.Height)
		}
		return cfg.Width, cfg.Height, nil
	}
}

// normalizeImage 把上傳的位元組轉成要寫進儲存的那份位元組，並回報它實際是哪種格式。
//
// 順序刻意是「檢測 → 讀尺寸 → 來源上限 → 才做正規化」：來源上限必須在**任何**解碼之前
// 生效（檔頭第 3 點），而正規化是全記憶體的操作。
//
// 回傳的副檔名由 detectImageKind 決定，不由上傳者給的檔名決定 —— 見檔頭第 1 點。
func normalizeImage(data []byte, p ImagePolicy) ([]byte, imageKind, error) {
	kind, ok := detectImageKind(data)
	if !ok {
		return nil, "", errNotAnImage
	}

	width, height, err := imageBounds(kind, data)
	if err != nil {
		return nil, kind, err
	}
	if p.MaxSourcePixels > 0 && width*height > p.MaxSourcePixels {
		return nil, kind, fmt.Errorf("來源解析度 %dx%d 超過 max_source_pixels=%d", width, height, p.MaxSourcePixels)
	}

	if !p.Reencode {
		// 停用正規化：原樣存。這一條分支存在是為了讓「我需要同步保留原檔」的部署有出路，
		// 而它的代價（中繼資料保留）由 main.go 的啟動警告說出來。
		return data, kind, nil
	}

	switch kind {
	case kindGIF:
		if width > p.MaxWidth || height > p.MaxHeight {
			return nil, kind, fmt.Errorf("GIF 尺寸 %dx%d 超過輸出上限 %dx%d（GIF 不會被縮小，見 imagepolicy.go 檔頭第 4 點）",
				width, height, p.MaxWidth, p.MaxHeight)
		}
		stripped, err := stripGIFMetadata(data)
		if err != nil {
			return nil, kind, err
		}
		return stripped, kind, nil
	case kindWebP:
		if width > p.MaxWidth || height > p.MaxHeight {
			return nil, kind, fmt.Errorf("WebP 尺寸 %dx%d 超過輸出上限 %dx%d（WebP 不會被縮小，見 imagepolicy.go 檔頭第 4 點）",
				width, height, p.MaxWidth, p.MaxHeight)
		}
		info, err := parseWebP(data)
		if err != nil {
			return nil, kind, err
		}
		return info.stripped, kind, nil
	case kindJPEG, kindPNG:
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, kind, fmt.Errorf("解析圖片失敗: %w", err)
		}
		if width > p.MaxWidth || height > p.MaxHeight {
			img = scaleDown(img, fitSize(width, height, p.MaxWidth, p.MaxHeight))
		}
		out, err := encodeImage(kind, img, p.JPEGQuality)
		if err != nil {
			return nil, kind, err
		}
		return out, kind, nil
	}
	return nil, kind, fmt.Errorf("不支援的圖片格式 %q", kind)
}

// fitSize 把來源尺寸等比縮到「寬與高都不超過上限」。不放大：縮小只在來源超過上限時發生。
//
// 等比而不是「各自夾到上限」的理由：各自夾會改變長寬比，症狀是使用者上傳的橫幅變成方圖。
func fitSize(width, height, maxWidth, maxHeight int) image.Point {
	if width <= maxWidth && height <= maxHeight {
		return image.Point{X: width, Y: height}
	}
	scale := float64(maxWidth) / float64(width)
	if sh := float64(maxHeight) / float64(height); sh < scale {
		scale = sh
	}
	w := int(float64(width)*scale + 0.5)
	h := int(float64(height)*scale + 0.5)
	// 加 0.5 是四捨五入：截斷會得到 2047×1365 這類比上限小一像素的尺寸，
	// 而 0 是一個不能存的尺寸（一個極小的圖被縮到 0 就消失了 —— 那是「上傳成功、
	// 圖不見了」的形狀，沒有任何錯誤）。
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if w > maxWidth {
		w = maxWidth
	}
	if h > maxHeight {
		h = maxHeight
	}
	return image.Point{X: w, Y: h}
}

// scaleDown 以面積平均把圖縮小。
//
// 為什麼是面積平均而不是「取最近的那個像素」：最近鄰在縮 2 倍以上時會讓細線消失、
// 文字邊緣出現鋸齒，而且它不處理 alpha —— 一個半透明的邊界像素會被整颗保留或整颗丟掉。
// 面積平均把框內的像素加權平均，細線與半透明邊界都變成「灰」而不是「消失」。
//
// 刻意不引入 golang.org/x/image/draw（它有 Catmuc4Kernel）：標準庫沒有縮放，而為一個
// 40 行的實作加一個相依，與 config.go 的檔頭「最小相依」那個標準不符。
//
// 這裡一律回 *image.RGBA：兩個編碼器都接受 image.Image，但 RGBA 讓「縮小之後的格式」
// 只有一種，PNG 與 JPEG 走同一條轉換路徑。alpha 保留（PNG 需要它；JPEG 的來源本來沒有 alpha）。
func scaleDown(src image.Image, target image.Point) *image.RGBA {
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	dstW, dstH := target.X, target.Y
	if srcW <= 0 || srcH <= 0 || dstW <= 0 || dstH <= 0 {
		return image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	}
	out := image.NewRGBA(image.Rect(0, 0, dstW, dstH))

	for y := 0; y < dstH; y++ {
		// 每個輸出像素對應的來源範圍。用整數除法而不是浮點座標，是因為
		// 「(y+1)*srcH/dstH」在整數下就是這個像素應該覆蓋的右界，而浮點會 introduce
		// 一個 0.5 的邊界誤差 —— 那在尺寸不整除時會讓某列被平均兩次、另一列一次。
		sy0 := y * srcH / dstH
		sy1 := (y + 1) * srcH / dstH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < dstW; x++ {
			sx0 := x * srcW / dstW
			sx1 := (x + 1) * srcW / dstW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, a, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					// color.Color.RGBA() 是 **預乘 alpha 的 16 位元**值 —— 這是 Go 的契約，
					// 而把它當成 8 位元來平均會得到偏暗的結果（16 位元除以 257 才是 8 位元）。
					cr, cg, cb, ca := src.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
					r += uint64(cr)
					g += uint64(cg)
					b += uint64(cb)
					a += uint64(ca)
					n++
				}
			}
			if n == 0 {
				continue
			}
			avgR, avgG, avgB, avgA := r/n, g/n, b/n, a/n
			if avgA == 0 {
				// 完全透明：寫一個 alpha=0 的像素。不能寫 (0,0,0,255)，那會在 JPEG
				// 那一邊變成一個看不見的黑色圖 —— 而 PNG 需要它真的是透明的。
				out.SetRGBA(x, y, color.RGBA{A: 0})
				continue
			}
			// 預乘值除以平均 alpha 就還原成非預乘的通道值。avgR ≤ avgA 恆成立（預乘的定義），
			// 因此這個除法不可能超過 255，不需要夾最大值。
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(avgR * 255 / avgA),
				G: uint8(avgG * 255 / avgA),
				B: uint8(avgB * 255 / avgA),
				A: uint8(avgA >> 8), // 16 位元 → 8 位元：除以 257，>>8 是同一個近似
			})
		}
	}
	return out
}

// encodeImage 把解析出來的圖編成要存的那種格式。
//
// JPEG 的品質來自設定值；PNG 用預設壓縮層級（格式沒有品質這個維度，見檔頭第 5 點）。
func encodeImage(kind imageKind, img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	switch kind {
	case kindJPEG:
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("編碼 JPEG 失敗: %w", err)
		}
		return buf.Bytes(), nil
	case kindPNG:
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("編碼 PNG 失敗: %w", err)
		}
		return buf.Bytes(), nil
	}
	return nil, fmt.Errorf("格式 %q 不需要重新編碼", kind)
}

// stripGIFMetadata 保留像素資料，丟掉 Comment 區塊與非 NETSCAPE 的 Application 區塊。
//
// 兩個丟掉的理由不同：
//   - Comment（0x21 0xFE）是文字中繼資料，內容完全由產生軟體決定（常見的是「這張圖由 X 產生」）。
//   - Application（0x21 0xFF）裡只有 NETSCAPE2.0/2.1 是**功能**（動畫的循環次數），少了它
//     瀏覽器會把多影格 GIF 播一次就停。其他識別碼（Adobe Photoshop、Adobe_XMCC 這類）是
//     軟體指紋，屬於要剝的那一邊。
//
// 逐區塊走一遍而不是搜尋位元組模式：GIF 的像素資料裡可能出现看起來像區塊標頭的字串，
// 搜尋會把資料當成區塊並改寫它 —— 那是一個「圖壞了但沒有任何錯誤」的失效方式。
func stripGIFMetadata(data []byte) ([]byte, error) {
	if len(data) < 13 {
		return nil, errors.New("GIF 標頭不完整")
	}
	out := make([]byte, 0, len(data))
	out = append(out, data[:13]...) // Header（6）＋ Logical Screen Descriptor（7）
	pos := 13

	// 全域色彩表：packed 位元 0x80 表示存在，大小是 2^(bits+1) 個色彩 entries × 3 位元組。
	// 它必須保留 —— 影格的索引指向它。
	if data[6+4]&0x80 != 0 {
		gct := 3 * (1 << (int(data[6+4]&0x07) + 1))
		if pos+gct > len(data) {
			return nil, errors.New("GIF 的全域色彩表超出檔案長度")
		}
		out = append(out, data[pos:pos+gct]...)
		pos += gct
	}

	for pos < len(data) {
		switch data[pos] {
		case gifTrailer:
			out = append(out, data[pos:]...)
			return out, nil
		case gifExtension:
			if pos+2 > len(data) {
				return nil, errors.New("GIF 的延伸區塊缺少標籤")
			}
			label := data[pos+1]
			// Plain Text（0x01）在標籤之後有一個**固定 13 位元組**的區塊（位置、尺寸、
			// 字元屬性），之後才是文字的子區塊。那 13 位元組的第一個是 x 的低位元組，
			// 因此把它當成子區塊的長度前綴會讀到 0x00（左上角的 x 就是 0）→ 「子區塊結束了」
			// → 走位停在這個區塊的中間 → 下一個位元組被當成區塊類型（通常 0x00）→ 整個檔案被拒。
			// 標準庫的 reader 對這個標籤讀的就是 13 位元組（eText），因此這裡讀同樣的數量。
			start := pos + 2
			if label == gifPlainTextExt {
				if pos+15 > len(data) {
					return nil, errors.New("GIF 的 Plain Text 區塊缺少 13 位元組的欄位")
				}
				start = pos + 15
			}
			end, err := gifSubBlocks(data, start)
			if err != nil {
				return nil, err
			}
			// 只留 NETSCAPE 的 Application 區塊（動畫循環）。Comment 與其他 Application 一律丟。
			if label == gifAppExt && gifIsLoopApp(data, pos+2, end) {
				out = append(out, data[pos:end]...)
			} else if label != gifCommentExt && label != gifAppExt {
				// 其他延伸區塊（0x01 Plain Text 這類）保留：它們不是中繼資料的常見載體，
				// 而丟掉一個不認識的延伸區塊可能改變資料的意義。
				out = append(out, data[pos:end]...)
			}
			pos = end
		case gifImageDesc:
			// 影格描述是「0x2C + 10 位元組」共 11 位元組：x(2) y(2) 寬(2) 高(2)
			// packed(1) flags(1)。packed 在 pos+9 而不是 pos+8 —— 寫成 pos+8 會讀到
			// 「高度」的高位元組，而那在真實的圖上幾乎一定有 0x80 那一位，
			// 症狀是「宣稱有一個局部色彩表」然後整個走位偏移，變成一個
			// 「檔案壞了」的錯誤而檔案其實完好。
			if pos+11 > len(data) {
				return nil, errors.New("GIF 的影格描述不完整")
			}
			n := 11
			if data[pos+9]&0x80 != 0 {
				n += 3 * (1 << (int(data[pos+9]&0x07) + 1)) // 局部色彩表：2^(bits+1) 個 entries × 3 位元組
			}
			end, err := gifSubBlocks(data, pos+n)
			if err != nil {
				return nil, err
			}
			out = append(out, data[pos:end]...)
			pos = end
		default:
			return nil, fmt.Errorf("GIF 有未知的區塊類型 %#x（offset %d）", data[pos], pos)
		}
	}
	return nil, errors.New("GIF 缺少結尾區塊（0x3B）")
}

// gifSubBlocks 走過一段以長度前綴的資料子區塊，回傳終止位元組（0x00）之後的位置。
//
// 每一個長度都在檔案長度內被驗證：一個宣告超出檔案長度的子區塊會被拒而不是被截斷，
// 因為截斷一份半壞的 GIF 會得到一個「存在但播放不出來」的檔案。
func gifSubBlocks(data []byte, pos int) (int, error) {
	for pos < len(data) {
		n := int(data[pos])
		if n == 0 {
			return pos + 1, nil
		}
		if pos+1+n > len(data) {
			return 0, fmt.Errorf("GIF 子區塊宣告 %d 位元組但檔案只剩 %d 位元組", n, len(data)-pos-1)
		}
		pos += 1 + n
	}
	return 0, errors.New("GIF 的子區塊缺少 0x00 終止")
}

// gifIsLoopApp 判斷一個 Application 區塊的識別碼是不是 NETSCAPE（動畫循環）。
// start..end 是子區塊鏈的範圍；識別碼是第一個子區塊的內容。
func gifIsLoopApp(data []byte, start, end int) bool {
	if start >= end || data[start] == 0 {
		return false
	}
	n := int(data[start])
	if n == 0 || start+1+n > end {
		return false
	}
	return bytes.HasPrefix(data[start+1:start+1+n], []byte("NETSCAPE"))
}

// webpInfo 是一次 parseWebP 的結果：畫布尺寸與「剝掉中繼資料之後的那份位元組」。
type webpInfo struct {
	width    int
	height   int
	stripped []byte
}

// parseWebP 走過 RIFF 的區塊鏈，只保留圖片與動畫區塊，並重算 RIFF 的長度。
//
// 保留的清單是白名單：VP8／VP8L／VP8X（圖片）、ANIM／ANFO／FRAM（動畫）、ICCP（色彩描述，
// 那是顏色而不是身分）。丟掉的是 EXIF（含 GPS）與任何不認識的區塊 —— 理由見檔頭第 6 點。
//
// RIFF 宣告的長度**不被信任**：它只用來做「宣告的長度超過實際長度」這一道驗證，
// 區塊鏈的走法以 len(data) 為界。輸出裡的長度由實際產物重算 —— 保留的區塊少了一塊，
// 舊的長度就是一個「宣告比實際長」的檔案，而瀏覽器對它的處理是靜默截斷。
func parseWebP(data []byte) (webpInfo, error) {
	if len(data) < 12 {
		return webpInfo{}, errors.New("WebP 標頭不完整")
	}
	if declared := binary.LittleEndian.Uint32(data[4:8]); uint64(declared)+8 > uint64(len(data)) {
		return webpInfo{}, fmt.Errorf("WebP 的 RIFF 長度宣告 %d，實際檔案只有 %d 位元組", declared, len(data))
	}

	out := make([]byte, 0, len(data))
	out = append(out, data[:12]...)
	var width, height int
	hasImage := false

	for pos := 12; pos < len(data); {
		if pos+8 > len(data) {
			return webpInfo{}, fmt.Errorf("WebP 在 offset %d 有一個不完整區塊標頭", pos)
		}
		fourCC := string(data[pos : pos+4])
		size := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		payload := pos + 8
		if payload+int(size) > len(data) {
			return webpInfo{}, fmt.Errorf("WebP 區塊 %q 宣告 %d 位元組但檔案只剩 %d 位元組", fourCC, size, len(data)-payload)
		}
		body := data[payload : payload+int(size)]
		// RIFF 規定區塊以偶數長度對齊，奇數長度後面多一個填充位元組。它屬於這個區塊，
		// 因此「丟掉這個區塊」時也必須丟掉那個填充 —— 留著會得到一個錯位的下一個區塊。
		padded := int(size) + int(size%2)

		switch fourCC {
		case "VP8X":
			// 旗元 1 + 保留 3 + 寬-1 3 + 高-1 3（小端 24 位元）。
			if len(body) < 10 {
				return webpInfo{}, errors.New("WebP 的 VP8X 區塊太短")
			}
			width = int(le24(body[4:7])) + 1
			height = int(le24(body[7:10])) + 1
			hasImage = true
			out = append(out, data[pos:payload+padded]...)
		case "VP8 ":
			// 簡單影格標頭：同步碼 3 + 影格類型 1 + 尺寸 6（各 14 位元）。
			// 非關鍵影格不能被獨立解出來，因此拒收 —— 那種檔案在瀏覽器裡本來也播不出來。
			if len(body) < 10 || !bytes.Equal(body[:3], webpVP8Sync) || body[3]&0x01 != 1 {
				return webpInfo{}, errors.New("WebP 的 VP8 區塊不是關鍵影格")
			}
			width = int(binary.LittleEndian.Uint16(body[6:8])) & 0x3fff
			height = int(binary.LittleEndian.Uint16(body[8:10])) & 0x3fff
			hasImage = true
			out = append(out, data[pos:payload+padded]...)
		case "VP8L":
			// 無損標頭：0x2f + 32 位元（寬-1 14 位元、高-1 14 位元、alpha 1、量化 3）。
			if len(body) < 5 || body[0] != webpVP8LSig {
				return webpInfo{}, errors.New("WebP 的 VP8L 區塊標頭無效")
			}
			v := binary.LittleEndian.Uint32(body[1:5])
			width = int(v&0x3fff) + 1
			height = int((v >> 14) & 0x3fff) + 1
			hasImage = true
			out = append(out, data[pos:payload+padded]...)
		default:
			if webpLoopChunks[fourCC] {
				out = append(out, data[pos:payload+padded]...)
			}
			// EXIF 與一切不認識的區塊：丟掉（連同它的填充位元組）。
		}
		pos += 8 + padded
	}

	if !hasImage {
		return webpInfo{}, errNotAnImage
	}
	if width <= 0 || height <= 0 {
		return webpInfo{}, fmt.Errorf("WebP 尺寸無效: %dx%d", width, height)
	}
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return webpInfo{width: width, height: height, stripped: out}, nil
}

// le24 讀一個小端 24 位元無號數（VP8X 的畫布尺寸就是這個形式）。
func le24(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}
