/*
imagepolicy_test.go 釘住 imagepolicy.go 的六個決定（見該檔的【關鍵設計決策】）。

它刻意不透過 HTTP 打：這六條都是「位元組進、位元組出」的純函式行為，走 httptest 只會
讓每一條都要先組一份 multipart 表單，而那個形狀屬於 server_test.go 的工作。

【這裡為什麼自己產圖片而不是放二進位固定檔】
Go 的標準庫能編 JPEG／PNG／GIF，因此「一張真的圖」可以在這裡現場產生，不需要把
幾十 KB 的二進位檔放進版控。WebP 編不出來（標準庫沒有 WebP 編碼器，見 imagepolicy.go
的檔頭），因此 WebP 那幾支用**合成但結構合法**的容器：這支解析器讀的是 RIFF 的區塊
結構，不是位元流，所以合成檔對它完全是足夠的證據 —— 而這個事實必須寫下來，否則讀者
會以為這裡測的是「真的 WebP」。

【六條對應】
  1. 內容而不是副檔名 → TestDetectImageKindRejectsNonImageContent
  2. 重編碼才剝得掉中繼資料 → TestNormalizeJPEGStripsEXIF / TestNormalizePNGStripsMetadata /
     TestNormalizeDisabledKeepsMetadata
  3. 來源解析度上限在解碼之前 → TestNormalizeRejectsSourceAbovePixelCap
  4. GIF／WebP 不重編碼、只剝容器 → TestNormalizeGIFKeepsAnimationAndStripsComments /
     TestNormalizeWebPStripsMetadataChunk
  5. 尺寸收斂等比 → TestNormalizeDownscalesPreservingAspectRatio
  6. 白名單而不是黑名單 → TestNormalizeWebPStripsUnknownChunk
*/
package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// sampleJPEG 產一張 w×h 的漸層 JPEG。
//
// 用漸層而不是單色：單色的 JPEG 經重新編碼後可能與原檔位元組相同（DCT 係數全為 0 的
// 平塊），那會讓「輸出是重新編碼過的」這種斷言變成斷言一件恰好沒發生的事。
func sampleJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 7) % 256),
				G: uint8((y * 11) % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("編 JPEG 失敗: %v", err)
	}
	return buf.Bytes()
}

// samplePNG 產一張 w×h 的 PNG，帶 alpha（半透明）。
//
// alpha 必須是真的：PNG 的重編碼如果偷偷把半透明變成不透明，這條測試就會看到輸出
// 裡所有像素的 alpha=255 —— 而那是一個「圖看起來正常但透明度沒了」的靜默缺陷。
func samplePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 40, B: 90, A: uint8(60 + (x%200))})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("編 PNG 失敗: %v", err)
	}
	return buf.Bytes()
}

// sampleGIF 產一張帶多影格與循環的 GIF（LoopCount 讓 encoder 寫出 NETSCAPE 區塊）。
//
// Config（畫布尺寸＋色彩模型）必須給：`gif.EncodeAll` 只有在 `Config.ColorModel` 是 Palette
// 時才寫全域色彩表（writer.go 的那個 `ok` 判斷）。沒給的該這份檔案是「標頭＋局部色彩表」
// 的形狀 —— 真實的 GIF 幾乎都有全域色彩表，而下面那幾支「走位」的測試是按 GCT 的長度
// 算位置的（見 truncatedGIF），因此那個差異會讓它們從「測區塊不完整」變成「測走位讀到
// 了標頭中間的位元組」—— 兩者都回錯誤，但只有前者是格式知識。
func sampleGIF(t *testing.T, w, h, frames int) []byte {
	t.Helper()
	palette := []color.Color{color.RGBA{R: 255, G: 0, B: 0, A: 255}, color.RGBA{R: 0, G: 255, B: 0, A: 255}}
	images := make([]*image.Paletted, 0, frames)
	delays := make([]int, 0, frames)
	for i := 0; i < frames; i++ {
		p := image.NewPaletted(image.Rect(0, 0, w, h), palette)
		// Paletted 只有 Set(x, y, color.Color)（沒有 SetIndex）—— 直接寫位元組會得到
		// 「uint8 does not implement color.Color」，那個錯誤看起來像palette 建立錯了。
		colorIndex := palette[i%2]
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				p.Set(x, y, colorIndex)
			}
		}
		images = append(images, p)
		delays = append(delays, 10)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &gif.GIF{
		// Config.ColorModel 的類型是 color.Model（不是 []color.Color），而 writer 讀的是
		// `Config.ColorModel.(color.Palette)` —— 因此這裡必須給 color.Palette(...)：
		// 它同時是 Model（有 Convert）和 Palette，兩個條件都满足。直接給 palette
		// （[]color.Color）會得到「does not implement color.Model」，那個錯誤看起來像
		// palette 的類型選錯了，而不是「這個欄位是 Model 而不是切片」。
		Config:    image.Config{Width: w, Height: h, ColorModel: color.Palette(palette)},
		Image:     images,
		Delay:     delays,
		LoopCount: 0,
	}); err != nil {
		t.Fatalf("編 GIF 失敗: %v", err)
	}
	return buf.Bytes()
}

// insertJPEGEXIF 把一個 APP1（EXIF）區塊插在 SOI 之後。
//
// 這是「相機把 GPS 寫在哪裡」的真實位置，而不是隨意塞在檔案尾端：塞在尾端會被 jpeg.Decode
// 當成垃圾忽略，那這條測試就只證明了「標準庫不在意尾端的垃圾」，而不是「重編碼剝掉了它」。
func insertJPEGEXIF(t *testing.T, data []byte, payload string) []byte {
	t.Helper()
	if len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
		t.Fatalf("這不是一份 JPEG（前兩個位元組 %#x %#x）", data[0], data[1])
	}
	segLen := len(payload) + 2 // APP1 的長度欄位包含它自己那 2 個位元組
	out := make([]byte, 0, len(data)+len(payload)+4)
	out = append(out, data[0], data[1], 0xff, 0xe1, byte(segLen>>8), byte(segLen&0xff))
	out = append(out, payload...)
	out = append(out, data[2:]...)
	return out
}

// insertPNGChunk 在 IEND 之前插入一個 chunk（type + payload + CRC）。
//
// 必須插在 IEND 之前而不是檔案尾端：PNG 的 chunk 是「依順序讀取」的，插在 IEND 之後
// 的 chunk 在真實實作裡可能被忽略（Go 的解碼器就是這樣），那會讓「它被剝掉了」變成
// 一件本來就不會被讀到的事。CRC 必須正確，否則 Go 的 png.Decode 會在讀取時就失敗。
func insertPNGChunk(t *testing.T, data []byte, chunkType string, payload []byte) []byte {
	t.Helper()
	iend := bytes.Index(data, []byte("IEND"))
	if iend < 4 {
		t.Fatalf("找不到 IEND 區塊（index=%d）", iend)
	}
	var chunk []byte
	chunk = binary.BigEndian.AppendUint32(chunk, uint32(len(payload)))
	chunk = append(chunk, chunkType...)
	chunk = append(chunk, payload...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
	out := make([]byte, 0, len(data)+len(chunk))
	out = append(out, data[:iend-4]...)
	out = append(out, chunk...)
	out = append(out, data[iend-4:]...)
	return out
}

// insertGIFExtension 在結尾區塊（0x3B）之前插入一個延伸區塊。
// label 是 0xFE（Comment）或 0xFF（Application）；identifier 是 Application 的識別碼，
// Comment 傳空字串。
func insertGIFExtension(t *testing.T, data []byte, label byte, identifier string, text string) []byte {
	t.Helper()
	trailer := bytes.IndexByte(data, 0x3b)
	if trailer < 0 {
		t.Fatal("找不到 GIF 的結尾區塊")
	}
	var block []byte
	block = append(block, 0x21, label)
	if identifier != "" {
		block = append(block, byte(len(identifier)))
		block = append(block, identifier...)
	}
	block = append(block, byte(len(text)))
	block = append(block, text...)
	block = append(block, 0x00)

	out := make([]byte, 0, len(data)+len(block))
	out = append(out, data[:trailer]...)
	out = append(out, block...)
	out = append(out, data[trailer:]...)
	return out
}

// webpChunk 組一個 RIFF 區塊（含奇數長度時的填充位元組）。
func webpChunk(fourCC string, payload []byte) []byte {
	var chunk []byte
	chunk = append(chunk, fourCC...)
	chunk = binary.LittleEndian.AppendUint32(chunk, uint32(len(payload)))
	chunk = append(chunk, payload...)
	if len(payload)%2 == 1 {
		chunk = append(chunk, 0) // RIFF 的偶數對齊填充
	}
	return chunk
}

// riffWrap 組出 RIFF 的容器頭（"RIFF" + 長度 + 形式碼）。長度是「整個檔案減 8」，
// 因此它必須在 body 組完之後才算 —— 算錯的長度是一個「宣告比實際長」的檔案，
// 而瀏覽器對它的處理是靜默截斷。
func riffWrap(body []byte) []byte {
	out := append([]byte("RIFF"), binary.LittleEndian.AppendUint32(nil, uint32(len(body)))...)
	return append(out, body...)
}

// syntheticWebP 組一份結構合法的 WebP：VP8 簡單影格標頭＋可選的额外區塊。
//
// 為什麼可以合成：標準庫沒有 WebP 編碼器（見 imagepolicy.go 檔頭），而 parseWebP 讀的是
// RIFF 的區塊結構與影格標頭裡的尺寸欄位 —— 它不解位元流。因此這是一份「容器正確、
// 位元流是假的」的檔案，對這支解析器是足夠的證據，但**不是**對「瀏覽器播得出來」的證據。
func syntheticWebP(t *testing.T, width, height int, extra ...[2]string) []byte {
	t.Helper()
	frame := make([]byte, 10)
	frame[0], frame[1], frame[2] = webpVP8Sync[0], webpVP8Sync[1], webpVP8Sync[2]
	frame[3] = 0x01 // 關鍵影格
	frame[6], frame[7] = byte(width&0xff), byte((width>>8)&0x3f)
	frame[8], frame[9] = byte(height&0xff), byte((height>>8)&0x3f)

	body := []byte("WEBP")
	body = append(body, webpChunk("VP8 ", frame)...)
	for _, e := range extra {
		body = append(body, webpChunk(e[0], []byte(e[1]))...)
	}
	return riffWrap(body)
}

// policyWith 是一份顯式的 ImagePolicy —— 刻意不用 Config.imagePolicy：這些測試要的是
// 「這幾個上限值造成這個行為」，而經 Config 走一遍會多一個「哪一層的兜底值生效了」的變數。
func policyWith(reencode bool, maxW, maxH, maxPixels, quality int) ImagePolicy {
	return ImagePolicy{
		Reencode:        reencode,
		MaxWidth:        maxW,
		MaxHeight:       maxH,
		MaxSourcePixels: maxPixels,
		JPEGQuality:     quality,
	}
}

var defaultTestPolicy = policyWith(true, 2048, 2048, 24_000_000, 85)

// TestDetectImageKindRejectsNonImageContent 守住決定 1：副檔名不是事實，位元組才是。
//
// 這一支的失效方式是「任何被誤標成圖片的內容都能進儲存」：那是一個公開的檔案託管
// （URL 形狀是 /files/<uuid>.jpg，Content-Type 說它是圖），而且後端會把它寫進貼文。
func TestDetectImageKindRejectsNonImageContent(t *testing.T) {
	accepted := []struct {
		name string
		data []byte
		want imageKind
	}{
		{"JPEG", sampleJPEG(t, 16, 12), kindJPEG},
		{"PNG", samplePNG(t, 16, 12), kindPNG},
		{"GIF", sampleGIF(t, 8, 8, 2), kindGIF},
		{"WebP", syntheticWebP(t, 320, 240), kindWebP},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			kind, ok := detectImageKind(tc.data)
			if !ok || kind != tc.want {
				t.Errorf("detectImageKind = (%q, %v)，want (%q, true)", kind, ok, tc.want)
			}
			if ext := extensionFor(kind); ext == "" {
				t.Errorf("格式 %q 沒有對應的副檔名", kind)
			}
		})
	}

	rejected := []struct {
		name string
		data []byte
	}{
		{"從前測試用的假位元組", []byte("fake-jpeg-bytes")},
		{"HTML", []byte("<!DOCTYPE html><script>alert(1)</script>")},
		{"JavaScript", []byte("document.location='http://evil'")},
		{"SVG（後端刻意不放行，因為它能內嵌腳本）", []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"><script>x</script></svg>")},
		{"ZIP", []byte("PK\x03\x04\x00\x00\x00\x00")},
		{"MP3（音訊沒有圖片的重新編碼可用）", []byte("ID3\x03\x00\x00\x00\x00\x00")},
		// 簽章被截到一半：一個「上傳中被斷線」的半個檔案不應該被當成圖片存下來。
		{"截斷的 JPEG 簽章", []byte{0xff, 0xd8}},
		{"截斷的 PNG 簽章", []byte{0x89, 'P', 'N', 'G'}},
		{"截斷的 WebP 標頭", []byte("RIFF\x10\x00\x00\x00")},
		{"RIFF 但不是 WebP", []byte("RIFF\x10\x00\x00\x00AVIX")},
		// GIF89a 之外的「GIF8xx」在瀏覽器裡不存在；放行它等於放行一個播不出來的檔案。
		{"不存在的 GIF 變體", []byte("GIF88a\x00\x00")},
		{"空檔案", []byte{}},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			kind, ok := detectImageKind(tc.data)
			if ok {
				t.Errorf("%q 被當成圖片（%q）—— 那正是決定 1 要擋掉的東西", tc.name, kind)
			}
		})
	}
}

// TestNormalizeRejectsNonImageContent 是同一件事在 normalizeImage 那一層的出口：
// 它必須導向 errNotAnImage（server.go 據此回 400）。
func TestNormalizeRejectsNonImageContent(t *testing.T) {
	_, _, err := normalizeImage([]byte("<html>not an image</html>"), defaultTestPolicy)
	if err != errNotAnImage {
		t.Errorf("錯誤 = %v，want errNotAnImage（server.go 靠它分類 400）", err)
	}
}

// TestNormalizeJPEGStripsEXIF 守住決定 2 在 JPEG 上的那一個。
//
// 這是這個功能存在的首要理由：**GPS 在 EXIF 裡**，而本站的立論是「匿名性由構造保證」
// （README）。附圖帶 GPS 時，那個立論對「你貼的這張圖」不成立 —— 而沒有任何頁面
// 會說這件事，因此它只能由測試釘住。
func TestNormalizeJPEGStripsEXIF(t *testing.T) {
	original := insertJPEGEXIF(t, sampleJPEG(t, 40, 30), "Exif\x00\x00"+
		"II*\x00"+ // 一個假的 TIFF 標頭 —— 標準庫不在意內容，這支測試在意的是「它還在不在」
		"GPSLocation\x00"+
		"Canon EOS R5\x00"+
		"2026:10:29 13:40:00\x00")

	if !bytes.Contains(original, []byte("GPSLocation")) {
		t.Fatal("合成失敗：EXIF 沒有被插進去了")
	}

	out, kind, err := normalizeImage(original, defaultTestPolicy)
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	if kind != kindJPEG {
		t.Errorf("格式 = %q，want jpeg", kind)
	}
	for _, marker := range []string{"Exif", "GPSLocation", "Canon EOS R5", "2026:10:29"} {
		if bytes.Contains(out, []byte(marker)) {
			t.Errorf("重新編碼之後還留下 %q —— 那個位置就是隱私的落點", marker)
		}
	}

	// 輸出必須是一份**真的**能讀的 JPEG，尺寸與原檔相同：剝中繼資料不能把圖也剝掉。
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("輸出不是可讀的圖片: %v", err)
	}
	if cfg.Width != 40 || cfg.Height != 30 {
		t.Errorf("尺寸 = %dx%d，want 40x30（沒有超過上限時不該被縮小）", cfg.Width, cfg.Height)
	}
}

// TestNormalizePNGStripsMetadata 守住決定 2 在 PNG 上的那一個。
//
// tEXt 是 PNG 的中繼資料載體（Software、Description、Author 這類）。它插在 IEND 之前：
// 插在尾端的話 Go 的解碼器根本不會讀到它，那這條測試會證明一件無關的事。
func TestNormalizePNGStripsMetadata(t *testing.T) {
	original := insertPNGChunk(t, samplePNG(t, 24, 18), "tEXt", []byte("Software\x00SecretTool 12.3"))
	if !bytes.Contains(original, []byte("SecretTool")) {
		t.Fatal("合成失敗：tEXt 沒有被插進去了")
	}

	out, kind, err := normalizeImage(original, defaultTestPolicy)
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	if kind != kindPNG {
		t.Errorf("格式 = %q，want png", kind)
	}
	if bytes.Contains(out, []byte("SecretTool")) {
		t.Error("重新編碼之後還留下 tEXt 的 Software")
	}

	// alpha 必須還在。重新編碼把半透明變成不透明是一個「圖看起來正常」的缺陷，
	// 因此這一條不能靠「它是一份可讀的 PNG」通過。
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("輸出不是可讀的 PNG: %v", err)
	}
	_, _, _, alpha := img.At(3, 3).RGBA()
	if alpha == 0xffff {
		t.Error("輸出裡 (3,3) 是完全不透明的 —— 原檔的 alpha 在重新編碼時被丢掉了")
	}
	if alpha == 0 {
		t.Error("輸出裡 (3,3) 是完全透明的 —— 原檔的像素被抹掉了")
	}
}

// TestNormalizeDisabledKeepsMetadata 守住「reencode = false」的語意。
//
// 這一支測試的價值不在於「關掉會怎樣」，而在於**關掉會留下 EXIF**：那是 main.go 的
// imagePolicyWarning 要警告的內容，而它必須有可觀察的後果才值得被警告。
func TestNormalizeDisabledKeepsMetadata(t *testing.T) {
	original := insertJPEGEXIF(t, sampleJPEG(t, 20, 16), "Exif\x00\x00GPSLocation\x00")
	out, kind, err := normalizeImage(original, policyWith(false, 2048, 2048, 24_000_000, 85))
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	if kind != kindJPEG {
		t.Errorf("格式 = %q，want jpeg", kind)
	}
	if !bytes.Equal(out, original) {
		t.Error("reencode=false 時輸出被改寫了 —— 那個設定值的語意是「原樣存檔」")
	}
	if !bytes.Contains(out, []byte("GPSLocation")) {
		t.Error("reencode=false 卻剝掉了 EXIF —— 那表示這個開關不再只是「不重編碼」")
	}
}

// TestNormalizeRejectsSourceAbovePixelCap 守住決定 3。
//
// 這一支的失效方式是「解壓炸彈」：一份 40 位元組的 PNG 宣告 5000×5000 的畫布，
// 解碼後是 100 MB 的 RGBA，而 upload.max_size 完全管不到它（40 位元組遠小於上限）。
// 因此拒收必須發生在**任何解碼之前** —— 這裡用一份只有 IHDR 的假 PNG，正好證明
// 這條路徑不需要位元流就能做決定。
func TestNormalizeRejectsSourceAbovePixelCap(t *testing.T) {
	bomb := pngSignature
	bomb = binary.BigEndian.AppendUint32(bomb, 13) // IHDR 長度
	bomb = append(bomb, "IHDR"...)
	bomb = binary.BigEndian.AppendUint32(bomb, 5000)
	bomb = binary.BigEndian.AppendUint32(bomb, 5000)
	bomb = append(bomb, 8, 6, 0, 0, 0) // 位深度/顏色類型/压缩/滤镜/_interlace
	bomb = binary.BigEndian.AppendUint32(bomb, crc32.ChecksumIEEE(bomb[len(pngSignature)+4:]))
	bomb = binary.BigEndian.AppendUint32(bomb, 0) // IEND 長度
	bomb = append(bomb, "IEND"...)
	bomb = binary.BigEndian.AppendUint32(bomb, crc32.ChecksumIEEE(bomb[len(bomb)-4:]))

	_, _, err := normalizeImage(bomb, policyWith(true, 2048, 2048, 24_000_000, 85))
	if err == nil {
		t.Fatal("來源 5000×5000（25M px）沒有被拒 —— 那是解壓炸彈的形狀")
	}
	if !strings.Contains(err.Error(), "max_source_pixels") {
		t.Errorf("錯誤訊息沒有指名 max_source_pixels: %v", err)
	}
}

// TestNormalizeDownscalesPreservingAspectRatio 守住決定 5。
//
// 斷言的是「兩個邊都不超過上限、長寬比不變」而不是「一個確定的尺寸」：確定的尺寸
// 把四捨五入寫進斷言裡，那會讓「改一個 rounding」變成一個紅的測試，而它不是缺陷。
// 這裡另外斷言輸出**小於原檔** —— 縮小的存在理由就是傳輸量，若它變大這條路徑就沒有意義。
func TestNormalizeDownscalesPreservingAspectRatio(t *testing.T) {
	original := sampleJPEG(t, 3000, 2000)
	out, _, err := normalizeImage(original, policyWith(true, 2048, 2048, 24_000_000, 85))
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("輸出不是可讀的圖片: %v", err)
	}
	if cfg.Width > 2048 || cfg.Height > 2048 {
		t.Fatalf("尺寸 = %dx%d，超過上限 2048x2048", cfg.Width, cfg.Height)
	}
	// 長寬比：3000×2000 是 3:2，因此 width*2 應該等於 height*3。四捨五入可能差 1 個像素，
	// 因此容許 ±3 —— 那是一個「不是 3:2」的斷言而不是「恰好等於某個數」的斷言，
	// 後者會讓改一個 rounding 變成一個紅的測試，而它不是缺陷。
	if diff := cfg.Width*2 - cfg.Height*3; diff < -3 || diff > 3 {
		t.Errorf("長寬比跑掉了：%dx%d（原檔 3000x2000 是 3:2；width*2-height*3 = %d）", cfg.Width, cfg.Height, diff)
	}
	if len(out) >= len(original) {
		t.Errorf("縮小之後的檔案（%d bytes）不比原檔（%d bytes）小 —— 這條路徑的存在理由消失了", len(out), len(original))
	}
}

// TestNormalizeDoesNotUpscale 守住「縮小只在超過上限時發生」。
//
// 放大的失效方式是「上傳一張 500px 的圖得到一張 2048px 的圖」：檔案變大、畫質變差，
// 而沒有任何錯誤。上一支測試只測了「超過會被縮」，這一條測的是反向的那一邊。
func TestNormalizeDoesNotUpscale(t *testing.T) {
	original := sampleJPEG(t, 500, 400)
	out, _, err := normalizeImage(original, policyWith(true, 2048, 2048, 24_000_000, 85))
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("輸出不是可讀的圖片: %v", err)
	}
	if cfg.Width != 500 || cfg.Height != 400 {
		t.Errorf("尺寸 = %dx%d，want 500x400（不該放大）", cfg.Width, cfg.Height)
	}
}

// TestNormalizeGIFKeepsAnimationAndStripsComments 守住決定 4 的 GIF 那一邊。
//
// 兩個方向都要斷言，因為這兩個方向的失效都安靜：
//   - 剝掉太多 → 動畫變成單影格（少了 NETSCAPE 區塊，瀏覽器只播一次就停），圖看起來
//     「正常」而它不再動。
//   - 剝太少 → Comment／Application 裡的中繼資料留在一個公開的圖片網址上。
func TestNormalizeGIFKeepsAnimationAndStripsComments(t *testing.T) {
	original := sampleGIF(t, 16, 16, 3)
	if !bytes.Contains(original, []byte("NETSCAPE")) {
		t.Fatal("合成失敗：這份 GIF 沒有 NETSCAPE 循環區塊（LoopCount 沒有生效？）")
	}
	original = insertGIFExtension(t, original, gifCommentExt, "", "Created by SecretTool on 2026-10-29")
	original = insertGIFExtension(t, original, gifAppExt, "Adobe Photoshop 24.0", "\x00\x03\x00")

	out, kind, err := normalizeImage(original, defaultTestPolicy)
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	if kind != kindGIF {
		t.Errorf("格式 = %q，want gif", kind)
	}
	for _, marker := range []string{"SecretTool", "Adobe Photoshop"} {
		if bytes.Contains(out, []byte(marker)) {
			t.Errorf("還留下 %q —— GIF 的 Comment／Application 區塊是決定 4 要剝的那一邊", marker)
		}
	}
	if !bytes.Contains(out, []byte("NETSCAPE")) {
		t.Error("NETSCAPE 循環區塊被剝掉了 —— 這份 GIF 現在只會播一次（那是「圖不動了」的靜默缺陷）")
	}

	// 影格數與循環必須還在。讀的是 DecodeAll 而不是 DecodeConfig：影格被吃掉時
	// DecodeConfig 仍然會通過（畫布尺寸在標頭裡），因此「它是一份可讀的 GIF」不夠。
	all, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("輸出不是可讀的 GIF: %v", err)
	}
	if len(all.Image) != 3 {
		// 欄位名是 Image（不是 Frames）—— 看錯它會得到 "undefined" 而不是「影格數不對」，
		// 那個錯誤看起來像測試寫壞了，而不是像正規化吃掉了影格。
		t.Errorf("影格數 = %d，want 3（原檔是 3 影格）", len(all.Image))
	}
	if all.LoopCount != 0 {
		t.Errorf("LoopCount = %d，want 0（0＝無限循環；-1 表示 NETSCAPE 區塊被剝掉了，"+
			"那是「圖不動了」的靜默缺陷）", all.LoopCount)
	}
	if all.Config.Width != 16 || all.Config.Height != 16 {
		t.Errorf("尺寸 = %dx%d，want 16x16", all.Config.Width, all.Config.Height)
	}
}

// TestNormalizeGIFRejectsOversizedAnimation 寫出決定 4 的代價：GIF 不會被縮小，因此
// 超過輸出上限時是被**拒**而不是被縮。這一條存在的理由是「那個差異必須被看見」——
// 若哪天有人改成縮小，這支測試會失敗并要求同時改 docs/CONFIGURATION.md。
func TestNormalizeGIFRejectsOversizedAnimation(t *testing.T) {
	original := sampleGIF(t, 2600, 2600, 2)
	_, _, err := normalizeImage(original, policyWith(true, 2048, 2048, 24_000_000, 85))
	if err == nil {
		t.Fatal("2600x2600 的 GIF 沒有被拒（它超過 2048 的輸出上限，而 GIF 不會被縮小）")
	}
	if !strings.Contains(err.Error(), "不會被縮小") {
		t.Errorf("錯誤訊息沒有講出「GIF 不會被縮小」這個取捨: %v", err)
	}
}

// TestNormalizeWebPStripsMetadataChunk 守住決定 4 的 WebP 那一邊，以及決定 3 的尺寸來源。
//
// WebP 的 EXIF 在一個獨立的 `EXIF` 區塊裡（不是影格資料），因此「剝掉它」是容器层的
// 工作。這一條同時斷言 RIFF 的長度被重算：丟掉一個區塊之後，舊的宣告長度是一個
// 「宣告比實際長」的檔案，而瀏覽器對它的處理是**靜默截斷**。
func TestNormalizeWebPStripsMetadataChunk(t *testing.T) {
	original := syntheticWebP(t, 640, 480,
		[2]string{"EXIF", "Exif fake GPS 2026"},
		[2]string{"ICCP", "\x00\x00\x01fake ICC profile"}, // 奇數長度 → 帶一個填充位元組
	)
	out, kind, err := normalizeImage(original, defaultTestPolicy)
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	if kind != kindWebP {
		t.Errorf("格式 = %q，want webp", kind)
	}
	if bytes.Contains(out, []byte("Exif")) {
		t.Error("EXIF 區塊還在 —— WebP 的 GPS 就在那裡")
	}
	if !bytes.Contains(out, []byte("ICCP")) {
		t.Error("ICCP（色彩描述）被剝掉了 —— 它不是身分資訊，而剝掉它會讓顏色偏掉")
	}
	if got := binary.LittleEndian.Uint32(out[4:8]); int(got) != len(out)-8 {
		t.Errorf("RIFF 長度宣告 %d，實際檔案 %d 位元組（宣告-8 應該等於實際長度）", got, len(out))
	}
	if len(out)%2 != 0 {
		t.Errorf("輸出長度 %d 是奇數 —— RIFF 要求偶數對齊，填充位元組必須跟著區塊一起處理", len(out))
	}

	info, err := parseWebP(out)
	if err != nil {
		t.Fatalf("剝完之後 parseWebP 讀不到: %v", err)
	}
	if info.width != 640 || info.height != 480 {
		t.Errorf("尺寸 = %dx%d，want 640x480", info.width, info.height)
	}
}

// TestNormalizeWebPStripsUnknownChunk 守住決定 6：保留清單是**白名單**。
//
// 用黑名單（只挑掉 EXIF）的失效方式是「某個軟體潢了一個區塊名，隱私又回來了」——
// 而那種失效沒有任何症狀。因此這裡放一個不是 EXIF 的私有區塊，斷言它也進了垃圾桶。
func TestNormalizeWebPStripsUnknownChunk(t *testing.T) {
	original := syntheticWebP(t, 320, 240,
		[2]string{"VPRM", "some-proprietary-thing"},
		[2]string{"XMP ", "<xmp:CreatorTool>Tool</xmp:CreatorTool>"},
	)
	out, _, err := normalizeImage(original, defaultTestPolicy)
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	for _, marker := range []string{"VPRM", "some-proprietary-thing", "xmp:CreatorTool"} {
		if bytes.Contains(out, []byte(marker)) {
			t.Errorf("還留下未知的區塊 %q —— 決定 6 說的是白名單，不是「只挑掉 EXIF」", marker)
		}
	}
	if !bytes.Contains(out, []byte("VP8 ")) {
		t.Error("VP8 影格區塊被剝掉了 —— 這份檔案沒有圖片了")
	}
}

// TestParseWebPRejectsMalformedContainers 守住「半壞的檔案被拒而不是被截斷存下來」。
//
// 截斷一份半壞的 WebP 會得到一個「存在但播不出來」的檔案，而它掛在貼文上時症狀是
// 破圖 —— 沒有錯誤、沒有日誌差異。與 storage.go 裡「失敗必須刪掉半成品」同一個道理。
func TestParseWebPRejectsMalformedContainers(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"RIFF 宣告的長度比實際長", func() []byte {
			good := syntheticWebP(t, 320, 240)
			binary.LittleEndian.PutUint32(good[4:8], uint32(len(good)+100))
			return good
		}()},
		{"VP8 不是關鍵影格", func() []byte {
			good := syntheticWebP(t, 320, 240)
			good[15] = 0x00 // 影格類型的 bit 0 = 關鍵影格
			return good
		}()},
		{"只有 EXIF 區塊（沒有圖片）", riffWrap(append([]byte("WEBP"), webpChunk("EXIF", []byte("exif only"))...))},
		{"區塊標頭被截到一半", []byte("RIFF\x20\x00\x00\x00WEBPVP")},
		{"區塊宣告的長度超出檔案", func() []byte {
			body := append([]byte("WEBP"), webpChunk("VP8 ", make([]byte, 10))...)
			body = append(body, "XXXX"...)
			body = append(body, 0xff, 0xff, 0xff, 0xff) // 宣告 4 GB，實際後面什麼都沒有
			return riffWrap(body)
		}()},
		{"VP8 影格標頭比 10 位元組短", riffWrap(append([]byte("WEBP"), webpChunk("VP8 ", []byte{0x90, 0x64, 0x20, 0x01})...))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseWebP(tc.data); err == nil {
				t.Error("parseWebP 接受了這個檔案 —— 它應該被拒，而不是被截斷存進儲存")
			}
		})
	}
}

// TestScaleDownAveragesInsteadOfNearest 守住縮放用的是面積平均。
//
// 用最近鄰的失效方式是「細線消失」與「半透明的邊界像素被整颗保留或整颗丟掉」。
// 這裡的證據是一個紅藍棋盤格縮成 1×1：最近鄰會只留下一種顏色，面積平均兩種都留。
func TestScaleDownAveragesInsteadOfNearest(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if (x+y)%2 == 0 {
				src.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			} else {
				src.SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
			}
		}
	}

	out := scaleDown(src, image.Point{X: 1, Y: 1})
	r, g, b, a := out.At(0, 0).RGBA()
	if r == 0 || b == 0 {
		t.Errorf("1×1 的輸出 = rgba(%d,%d,%d,%d)，red 或 blue 消失了 —— 那是最近鄰的行為，不是面積平均", r, g, b, a)
	}
	if r > 0 && b > 0 && (r+b)/2 > 65535 {
		t.Errorf("red+blue 的平均超出來源的總能量（r=%d b=%d）—— 加權寫错了", r, b)
	}
	if a != 0xffff {
		t.Errorf("不透明的來源縮完 alpha = %d，want 65535（alpha 不該被平均掉）", a)
	}
}

// TestScaleDownKeepsTransparency 守住「完全透明的框」不會變成黑色不透明。
//
// 這是 JPEG 那一邊最難診斷的形狀：一個透明的框在 JPEG 裡是黑的（格式沒有 alpha），
// 而「黑邊」在圖片上看起來像一個設計決定，不像一個 bug。
func TestScaleDownKeepsTransparency(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4)) // 全透明（零值）
	out := scaleDown(src, image.Point{X: 2, Y: 2})
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if _, _, _, a := out.At(x, y).RGBA(); a != 0 {
				t.Fatalf("(%d,%d) 的 alpha = %d，want 0（全透明的來源縮完必須還是透明的）", x, y, a)
			}
		}
	}
}

// TestExtensionForMatchesBackendWhitelist 把輸出副檔名的集合釘成「後端認得的那四個」。
//
// 後端的 validForumImageFileName（backend/forum/httpapi/forum_handlers.go）只認
// .jpg/.jpeg/.png/.gif/.webp。這裡多一種，或輸出 `.jpeg` 之外的寫法，都會造成
// 「檔案存得下、後端判定為非法圖片、貼文掛不上圖」—— 而那個症狀出现在使用者身上，
// 原因卻在一個使用者看不到的內部服務。
func TestExtensionForMatchesBackendWhitelist(t *testing.T) {
	for _, kind := range []imageKind{kindJPEG, kindPNG, kindGIF, kindWebP} {
		ext := extensionFor(kind)
		if !isImageExtension(ext) {
			t.Errorf("格式 %q 的副檔名 %q 不被 isImageExtension 認為是圖片", kind, ext)
		}
		if !strings.HasPrefix(ext, ".") {
			t.Errorf("副檔名 %q 缺少前導的點（後端比對的是帶點的寫法）", ext)
		}
	}
	// JPEG 收斂成一個值：白名單放行 .jpg 與 .jpeg，但輸出只需要一種。
	if ext := extensionFor(kindJPEG); ext != ".jpg" {
		t.Errorf("JPEG 的輸出副檔名 = %q，want .jpg", ext)
	}
	// 非圖片的副檔名必須被 isImageExtension 拒掉 —— 那是「要不要走正規化」的判別。
	for _, ext := range []string{".mp3", ".svg", ".html", ".txt", ""} {
		if isImageExtension(ext) {
			t.Errorf("isImageExtension(%q) = true —— 那會讓非圖片走圖片的正規化路徑", ext)
		}
	}
}

// le24Put 寫一個小端 24 位元（VP8X 的畫布尺寸就是這個形式）。
//
// 不能寫成 `binary.LittleEndian.PutUint32(body[4:8], v)`：那會多寫一個位元組，
// 而那個位元組正是高度第一個位元組（VP8X 的欄位是緊貼著排的，沒有對齊填充）。
func le24Put(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
}

// syntheticWebPVP8X 組一份「擴充容器」的 WebP：尺寸來自 VP8X 而不是 VP8 影格標頭。
//
// 為什麼要另開一支：VP8（有損）與 VP8X（擴充，含動畫與無損）的尺寸欄位在不同的位置 ——
// VP8X 是小端 24 位元且**減 1**，VP8 是 14 位元且**不減 1**。只測 VP8 的話，
// 「+1」這個轉換、le24 這支函式，以及動畫區塊的保留清單，全部沒有被執行過。
func syntheticWebPVP8X(t *testing.T, width, height int, extra ...[2]string) []byte {
	t.Helper()
	// 旗元 1 + 保留 3 + 寬-1 3 + 高-1 3 = 10 位元組（偶數，因此沒有填充）。
	flags := make([]byte, 10)
	flags[0] = 0x01 // 宣告「有影格」的旗元；解析器只看尺寸欄位
	le24Put(flags[4:7], uint32(width-1))
	le24Put(flags[7:10], uint32(height-1))

	body := []byte("WEBP")
	body = append(body, webpChunk("VP8X", flags)...)
	for _, e := range extra {
		body = append(body, webpChunk(e[0], []byte(e[1]))...)
	}
	return riffWrap(body)
}

// TestParseWebPVP8XCanvasAndAnimation 守住 VP8X 的尺寸讀法與動畫區塊的保留。
//
// 兩條斷言各自對應一個靜默的失效：
//   - 忘了「-1」：宣告 639×479 的畫布會被讀成 639×479 而不是 640×480 —— 尺寸差一個像素
//     在頁面上看不出來，但它會改變 max_source_pixels 的判定邊界（剛好卡在上限的圖被放行）。
//   - 丟掉 ANIM/ANFO/FRAM：這份檔案有影格但不再動，而 parseWebP 仍然回「成功」。
func TestParseWebPVP8XCanvasAndAnimation(t *testing.T) {
	original := syntheticWebPVP8X(t, 640, 480,
		[2]string{"ANIM", "\x00\x00\x00\x02\x00\x00\x00"}, // 循環次數 3（小端 24）＋影格數 2
		[2]string{"ANFO", "\x0a\x00\x00\x00\x00\x00\x00"}, // 延遲 10（4）＋影格索引 0（3）＝奇數長度
		[2]string{"FRAM", "\x00\x00\x00\x00\x00\x02\x00\x02\x00"}, // 旗元＋位置＋尺寸＝奇數長度
		[2]string{"EXIF", "Exif fake GPS 2026"},
		[2]string{"XMP ", "<xmp:CreatorTool>Tool</xmp:CreatorTool>"},
	)

	out, kind, err := normalizeImage(original, defaultTestPolicy)
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	if kind != kindWebP {
		t.Errorf("格式 = %q，want webp", kind)
	}
	info, err := parseWebP(out)
	if err != nil {
		t.Fatalf("剝完之後 parseWebP 讀不到: %v", err)
	}
	if info.width != 640 || info.height != 480 {
		t.Errorf("尺寸 = %dx%d，want 640x480（VP8X 的欄位存的是寬-1／高-1）", info.width, info.height)
	}
	for _, keep := range []string{"VP8X", "ANIM", "ANFO", "FRAM"} {
		if !bytes.Contains(out, []byte(keep)) {
			t.Errorf("%s 區塊被剝掉了 —— 這份 WebP 現在有影格但不動（而 parseWebP 仍然回成功）", keep)
		}
	}
	for _, drop := range []string{"Exif", "xmp:CreatorTool"} {
		if bytes.Contains(out, []byte(drop)) {
			t.Errorf("還留下 %q —— 決定 6 說的是白名單，不是「只挑掉 EXIF」", drop)
		}
	}
	if len(out)%2 != 0 {
		t.Errorf("輸出長度 %d 是奇數 —— ANFO／FRAM 是奇數長度的**保留**區塊，填充位元組必須跟著留下", len(out))
	}
	if got := binary.LittleEndian.Uint32(out[4:8]); int(got) != len(out)-8 {
		t.Errorf("RIFF 長度宣告 %d，實際檔案 %d 位元組", got, len(out))
	}
}

// TestParseWebPVP8LLosslessHeader 守住無損（VP8L）的尺寸讀法。
//
// VP8L 的尺寸是「一個 32 位元packed 值裡的兩個 14 位元欄位」，而且是**減 1** 的寫法。
// 沒有這一條的話，一張無損 PNG 轉出來的 WebP 會得到一個錯的尺寸 —— 尺寸決定「要不要縮小」
// 與「是否超過來源上限」，而讀錯它沒有任何症狀：檔案照樣存下來、照樣顯示。
func TestParseWebPVP8LLosslessHeader(t *testing.T) {
	// 0x2f ＋ packed：寬-1 在 bits 0–13、高-1 在 bits 14–27、alpha 在 bit 28。
	packed := uint32(319) | uint32(239)<<14 | 1<<28
	payload := append([]byte{webpVP8LSig}, binary.LittleEndian.AppendUint32(nil, packed)...)
	body := append([]byte("WEBP"), webpChunk("VP8L", payload)...)
	body = append(body, webpChunk("EXIF", []byte("Exif fake GPS"))...)
	original := riffWrap(body)

	out, _, err := normalizeImage(original, defaultTestPolicy)
	if err != nil {
		t.Fatalf("normalizeImage 回錯誤: %v", err)
	}
	info, err := parseWebP(out)
	if err != nil {
		t.Fatalf("剝完之後 parseWebP 讀不到: %v", err)
	}
	if info.width != 320 || info.height != 240 {
		t.Errorf("尺寸 = %dx%d，want 320x240（packed 的兩個 14 位元欄位都是「減 1」的）", info.width, info.height)
	}
	if bytes.Contains(out, []byte("Exif")) {
		t.Error("EXIF 還在 —— 無損的 WebP 一樣帶 GPS")
	}
	if !bytes.Contains(out, []byte("VP8L")) {
		t.Error("VP8L 被剝掉了 —— 這份檔案沒有圖片了")
	}
	// VP8L 的 payload 是 5 位元組（奇數），因此它帶一個填充位元組，而那個填充屬於這個區塊。
	if len(out)%2 != 0 {
		t.Errorf("輸出長度 %d 是奇數 —— 保留的奇數長度區塊少了它的填充位元組", len(out))
	}
}

// TestParseWebPRejectsMalformedVP8XAndVP8L 補上標頭那三種「太短／簽章不對」的拒收。
//
// 這一條補的是上一支 MalformedContainers 沒涵蓋的格式分支：VP8X 與 VP8L 各自有標頭驗證，
// 而「讀到 0 的尺寸」是第三種結果 —— 它會被存成一張 0×0 的圖，而 0×0 的圖片在瀏覽器裡
// 是一塊看不見的空白（上傳成功、圖不見了，沒有錯誤）。
func TestParseWebPRejectsMalformedVP8XAndVP8L(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"VP8X 只有 3 位元組（讀不到畫布欄位）", riffWrap(append([]byte("WEBP"), webpChunk("VP8X", []byte{0x01, 0x00, 0x00})...))},
		{"VP8L 的簽章不是 0x2f", riffWrap(append([]byte("WEBP"), webpChunk("VP8L", []byte{0x2e, 0x00, 0x00, 0x00, 0x00})...))},
		{"VP8L 只有 4 位元組（packed 讀不完）", riffWrap(append([]byte("WEBP"), webpChunk("VP8L", []byte{0x2f, 0x00, 0x00, 0x00})...))},
		{"VP8 宣告 0×0（尺寸欄位全零）", syntheticWebP(t, 0, 0)},
		{"標頭被截到 8 位元組（連形式碼都不完整）", []byte("RIFFWEBP")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWebP(tc.data)
			if err == nil {
				t.Error("parseWebP 接受了這個檔案 —— 它應該被拒，而不是被截斷存進儲存")
			}
			if _, _, err := normalizeImage(tc.data, defaultTestPolicy); err == nil {
				t.Errorf("normalizeImage 卻接受了這份檔案 —— 它會被存進儲存（400 的出口在 server.go）")
			}
		})
	}

	// 反向的那一邊：全零的 VP8X 是一份**合法的 1×1**（那個格式存的是「寬-1」與「高-1」），
	// 因此它必須被接受。把它拒掉是「上傳成功、圖不見了」的形狀，而它與上一條的
	// 「0×0 必須被拒」只差一個格式欄位的讀法 —— 這就是為什麼兩邊都要寫。
	t.Run("全零的 VP8X 是合法的 1×1（必須被接受）", func(t *testing.T) {
		one := riffWrap(append([]byte("WEBP"), webpChunk("VP8X", make([]byte, 10))...))
		info, err := parseWebP(one)
		if err != nil {
			t.Fatalf("parseWebP 拒絕了一份 1×1 的 VP8X: %v", err)
		}
		if info.width != 1 || info.height != 1 {
			t.Errorf("尺寸 = %dx%d，want 1x1（VP8X 的欄位是「減 1」的，全零就是 1）", info.width, info.height)
		}
		if _, _, err := normalizeImage(one, defaultTestPolicy); err != nil {
			t.Errorf("normalizeImage 拒絕了一份 1×1 的 WebP: %v", err)
		}
	})
}

// TestNormalizeWebPRejectsOversizedCanvas 寫出決定 4 在 WebP 上的代價（與 GIF 那一支對稱）。
//
// 這一條存在的理由與 GIF 那一支相同：「超過輸出上限時被拒而不是被縮」是一個**與 JPEG 不同**
// 的行為，它必須被釘住 —— 若有人改成「WebP 也縮小」，那需要一個解碼器相依，而 docs 裡
// 寫的是現在這個行為。
func TestNormalizeWebPRejectsOversizedCanvas(t *testing.T) {
	original := syntheticWebP(t, 3000, 2000)
	_, _, err := normalizeImage(original, policyWith(true, 2048, 2048, 24_000_000, 85))
	if err == nil {
		t.Fatal("3000x2000 的 WebP 沒有被拒（它超過 2048 的輸出上限，而 WebP 不會被縮小）")
	}
	if !strings.Contains(err.Error(), "不會被縮小") {
		t.Errorf("錯誤訊息沒有講出「WebP 不會被縮小」這個取捨: %v", err)
	}
}

// TestNormalizeRejectsCorruptJPEG 守住「簽章對但內容壞了」的兩種拒收。
//
// 兩個分支對應兩個不同的失效位置，因此兩份檔案各自要證明自己那一條：
//   - 標頭壞了：imageBounds（讀尺寸）就失敗，因此它**不會**走到解碼 —— 那這條也證明
//     「來源上限那一道不會先解碼」。
//   - 位元流被截斷：標頭讀得到，解碼失敗。若這一條被放行，存下來的是半張圖，
//     而它的症狀是破圖而不是錯誤。
func TestNormalizeRejectsCorruptJPEG(t *testing.T) {
	t.Run("標頭壞了（只有 SOI 與 APP0）", func(t *testing.T) {
		broken := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x04, 'J', 'F', 'I', 'F', 0x00, 0x01}
		if _, ok := detectImageKind(broken); !ok {
			t.Fatal("這一份應該被 detectImageKind 認為是 JPEG（簽章是对的）—— 那才是要測的形狀")
		}
		_, _, err := normalizeImage(broken, defaultTestPolicy)
		if err == nil {
			t.Fatal("一份只有標頭的 JPEG 被存下来了")
		}
		if !strings.Contains(err.Error(), "讀取圖片尺寸失敗") {
			t.Errorf("錯誤 = %v，want 由 imageBounds 那一层發出（讀尺寸就失敗，不需要解碼）", err)
		}
	})

	t.Run("標頭完整但位元流被截斷", func(t *testing.T) {
		whole := sampleJPEG(t, 64, 48)
		truncated := whole[:len(whole)/2]
		if _, _, err := image.DecodeConfig(bytes.NewReader(truncated)); err != nil {
			t.Fatalf("這份的標頭應該讀得到（%v）—— 否則這一條測的是標頭壞了而不是位元流截斷", err)
		}
		_, _, err := normalizeImage(truncated, defaultTestPolicy)
		if err == nil {
			t.Fatal("半張 JPEG 被存下来了 —— 那是一個「掛在貼文上但顯示不出來」的檔案")
		}
		if err == errNotAnImage {
			t.Error("位元流截斷被歸類成「不是圖片」—— 它確實是圖片，只是壞了；兩者都回 400，但訊息必須對得上")
		}
	})
}

// TestNormalizeRejectsCorruptPNG 是同一件事在 PNG 上的那一個。
//
// 用一份「只有 IHDR」的假 PNG（上一支解壓炸彈那條用的就是它）：尺寸讀得到，
// 因此拒收必須由解碼那一層發出。這一條釘的是「IHDR 合法但沒有 IDAT」的檔案不會被存下來。
func TestNormalizeRejectsCorruptPNG(t *testing.T) {
	body := append([]byte{}, pngSignature...)
	body = binary.BigEndian.AppendUint32(body, 13)
	body = append(body, "IHDR"...)
	body = binary.BigEndian.AppendUint32(body, 8)
	body = binary.BigEndian.AppendUint32(body, 8)
	body = append(body, 8, 2, 0, 0, 0)
	body = binary.BigEndian.AppendUint32(body, crc32.ChecksumIEEE(body[len(pngSignature)+4:]))
	body = binary.BigEndian.AppendUint32(body, 0)
	body = append(body, "IEND"...)
	body = binary.BigEndian.AppendUint32(body, crc32.ChecksumIEEE(body[len(body)-4:]))

	if _, _, err := image.DecodeConfig(bytes.NewReader(body)); err != nil {
		t.Fatalf("IHDR 應該讀得到: %v", err)
	}
	_, _, err := normalizeImage(body, defaultTestPolicy)
	if err == nil {
		t.Fatal("一份沒有像素資料的 PNG 被存下来了")
	}
	if !strings.Contains(err.Error(), "解析圖片失敗") {
		t.Errorf("錯誤 = %v，want 由解碼那一層發出", err)
	}
}

// truncatedGIF 保留「標頭＋全域色彩表」，並在那之後放一段合成的區塊資料、丟掉其餘的檔案。
//
// 位置是算出來的而不是搜尋出來的：在一份真實的 GIF 裡搜尋 0x2c 會找到**像素資料裡的**
// 那個位元組（LZW 的輸出是任意位元組），那這條測的是「走位走到資料中間」而不是
// 「這個區塊本身不完整」—— 兩者都回錯誤，但只有後者是格式知識。
//
// 「丟掉其餘的檔案」也是必要的：保留它們的話，那段合成資料後面接著的是一份合法的 GIF，
// 走位會繼續走下去而可能**成功** —— 那這條就變成一件不會發生的事。
func truncatedGIF(data []byte, block []byte) []byte {
	gct := 0
	if data[10]&0x80 != 0 {
		gct = 3 * (1 << (int(data[10]&0x07) + 1))
	}
	return append(append([]byte{}, data[:13+gct]...), block...)
}

// gifDescriptorBytes 是一個「完整但空的」影格描述：0x2c ＋ 10 位元組（共 11）。
//
// 寫成 10 而不是 11 是最容易踩的那一個：影格描述是「標頭位元組＋10 位元欄位」，
// 因此總長 11。少了那個位元組，走位從這裡開始偏移，後面每一步都讀到錯的位元組 ——
// 而症狀是「一份完好被剝成壞掉的檔案」，不是「這個檔案壞了」。
func gifDescriptorBytes() []byte {
	return append([]byte{gifImageDesc}, make([]byte, 10)...)
}

// TestStripGIFMetadataRejectsMalformed 逐項釘住 GIF 走位的每一種拒收。
//
// 這一條的價值是「被拒而不是被截斷」：一份半壞的 GIF 若被截斷存下來，它掛在貼文上時
// 的症狀是「不動的圖」或破圖，而沒有任何错误。與 storage.go 的「失敗必須刪掉半成品」
// 同一個道理 —— 一個被拒的上傳不留檔案。
func TestStripGIFMetadataRejectsMalformed(t *testing.T) {
	good := sampleGIF(t, 16, 16, 2)

	cases := []struct {
		name   string
		data   []byte
		wanted string
	}{
		{"標頭不到 13 位元組", []byte("GIF89a"), "標頭不完整"},
		{"標頭宣告的全域色彩表超出檔案", func() []byte {
			// packed 位元 0x80（有 GCT）＋ bits=7（256 個 entries＝768 位元組）：
			// 這份真實的 GIF 只有 12 位元組的 GCT，因此宣告的 768 位元組超出檔案。
			bad := append([]byte{}, good...)
			bad[10] = 0x80 | 0x07
			return bad
		}(), "全域色彩表超出檔案長度"},
		{"延伸區塊只有標頭沒有標籤", func() []byte {
			i := bytes.IndexByte(good, gifExtension)
			if i < 0 {
				t.Fatal("這份 GIF 沒有延伸區塊（NETSCAPE 呢？）")
			}
			return append([]byte{}, good[:i+1]...) // 截在 0x21 之後，標籤位元組不见了
		}(), "延伸區塊缺少標籤"},
		{"未知的區塊類型", truncatedGIF(good, []byte{0x33, 0x00}), "未知的區塊類型"},
		{"影格描述被截到一半（不到 11 位元組）", truncatedGIF(good, []byte{gifImageDesc, 0, 0, 0, 0}), "影格描述不完整"},
		{"影格的像素子區塊宣告超出檔案", truncatedGIF(good, append(gifDescriptorBytes(), 0xff)), "子區塊宣告"},
		{"影格的子區塊沒有 0x00 終止", truncatedGIF(good, append(gifDescriptorBytes(), 0x04, 'a', 'b', 'c', 'd')), "缺少 0x00 終止"},
		{"缺少結尾區塊（0x3B）", func() []byte {
			i := bytes.LastIndexByte(good, gifTrailer)
			if i < 0 {
				t.Fatal("這份 GIF 沒有結尾區塊")
			}
			return append([]byte{}, good[:i]...)
		}(), "缺少結尾區塊"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := stripGIFMetadata(tc.data)
			if err == nil {
				t.Error("stripGIFMetadata 接受了這份檔案 —— 它應該被拒")
			}
			if !strings.Contains(err.Error(), tc.wanted) {
				t.Errorf("錯誤 = %v，want 含 %q", err, tc.wanted)
			}
			// 同一份檔案走 normalizeImage 也必須被拒：那是 400 的出口（server.go）。
			if _, _, err := normalizeImage(tc.data, defaultTestPolicy); err == nil {
				t.Errorf("normalizeImage 卻接受了這份檔案 —— 它會被存進儲存")
			}
		})
	}
}

// TestStripGIFMetadataKeepsUnrecognisedNonMetadataExtension 守住「不認識的延伸區塊被保留」。
//
// 丟掉一個不認識的延伸區塊的失效方式是「改變資料的意義」：Plain Text（0x01）是**要顯示的**
// 文字，不是中繼資料，而剝掉它會讓圖少一塊內容而沒有任何錯誤。Comment 與 Application
// 是兩個已知要丟的（上一支測了），因此這裡測的是第三種：既不在丟的清單裡、也不是影格。
//
// 那份 13 位元組的固定區塊是**規格要求的**：標準庫的 reader 在 0x21 0x01 之後讀 13 位元組
// （x／y／寬／高／旗元／字元屬性等），少了它 DecodeAll 會得到「reading extension: unexpected
// EOF」。從前這裡只放了「0x21 0x01 ＋ 子區塊鏈」，那這條測的是「標準庫讀不到我合成的檔案」
// 而不是「保留的那塊仍在檔案裡」—— 兩者看起來一樣，但後者才是這支要證明的。
func TestStripGIFMetadataKeepsUnrecognisedNonMetadataExtension(t *testing.T) {
	original := sampleGIF(t, 16, 16, 2)
	trailer := bytes.IndexByte(original, gifTrailer)
	var block []byte
	block = append(block, gifExtension, 0x01) // Plain Text Extension
	block = append(block, make([]byte, 13)...) // 規格要求的固定欄位（位置與尺寸，全零＝左上角）
	block = append(block, 4)
	block = append(block, []byte("abcd")...)
	block = append(block, 0x00)
	original = append(append(append([]byte{}, original[:trailer]...), block...), original[trailer:]...)

	out, err := stripGIFMetadata(original)
	if err != nil {
		t.Fatalf("stripGIFMetadata 回錯誤: %v", err)
	}
	if !bytes.Contains(out, []byte("abcd")) {
		t.Error("Plain Text 延伸區塊被剝掉了 —— 它是要顯示的內容，不是中繼資料")
	}

	all, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("剝完之後 DecodeAll 讀不到（Plain Text 區塊被留著時應該還是可讀的）: %v", err)
	}
	if len(all.Image) != 2 {
		t.Errorf("影格數 = %d，want 2", len(all.Image))
	}
}

// TestStripGIFMetadataDropsApplicationWithNoSubBlocks 守住「空延伸區塊不算 NETSCAPE」。
//
// 這一條走的是 gifIsLoopApp 的第一個分支（子區塊鏈是空的）。它存在的理由：
// `0x21 0xff 0x00` 是一份合法的「Application 區塊但沒有識別碼」，若把「標籤是 0xff」
// 當作「這是循環區塊」，那個檔案會被**保留**，而真正的循環資訊根本不在裡面 ——
// 保留它沒有害處，但把它當成循環會讓下一條（真正的 NETSCAPE）被誤判為重複。
func TestStripGIFMetadataDropsApplicationWithNoSubBlocks(t *testing.T) {
	original := sampleGIF(t, 16, 16, 2)
	trailer := bytes.IndexByte(original, gifTrailer)
	var block []byte
	block = append(block, gifExtension, gifAppExt, 0x00) // 標籤 0xff，沒有任何子區塊
	original = append(append(append([]byte{}, original[:trailer]...), block...), original[trailer:]...)

	out, err := stripGIFMetadata(original)
	if err != nil {
		t.Fatalf("stripGIFMetadata 回錯誤: %v", err)
	}
	if !bytes.Contains(out, []byte("NETSCAPE")) {
		t.Error("原本的 NETSCAPE 循環區塊被剝掉了 —— 這份 GIF 現在只播一次")
	}
	// 那個空的 Application 區塊必須被丟掉（它不是循環，也不是影格）。
	if all, err := gif.DecodeAll(bytes.NewReader(out)); err != nil {
		t.Fatalf("輸出不是可讀的 GIF: %v", err)
	} else if len(all.Image) != 2 {
		t.Errorf("影格數 = %d，want 2", len(all.Image))
	}
}

// TestFitSizeNeverProducesZero 守住「縮小不會把圖縮到不存在」。
//
// 這一支的失效方式是「上傳成功、圖不見了」：一個極端長寬比的來源（10000×3）被等比縮到
// 高度時，高度會算出 0.6 → 截斷成 0，而 0 是一個不能編碼的尺寸。四捨五入加夾到 1 是
// 那一條的防禦；這裡直接呼叫 fitSize 而不是走整條管線，因為那個極端的尺寸要用
// sampleJPEG 做出來太貴（10000×3 的圖是一份几 MB 的檔案）。
func TestFitSizeNeverProducesZero(t *testing.T) {
	for _, tc := range []struct {
		name        string
		w, h        int
		maxW, maxH  int
	}{
		{"超寬的橫幅（10000×3 → 高被縮到不足 1）", 10000, 3, 2048, 2048},
		{"超高的直式圖（3×10000 → 寬被縮到不足 1）", 3, 10000, 2048, 2048},
		{"上限本身是 1×1", 2000, 2000, 1, 1},
		{"來源恰好等於上限（不該被縮）", 100, 100, 100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := fitSize(tc.w, tc.h, tc.maxW, tc.maxH)
			if p.X < 1 || p.Y < 1 {
				t.Errorf("fitSize(%d×%d, %d×%d) = %v —— 0 是一個不能存的尺寸（上傳成功、圖不見了）",
					tc.w, tc.h, tc.maxW, tc.maxH, p)
			}
			if p.X > tc.maxW || p.Y > tc.maxH {
				t.Errorf("fitSize(%d×%d, %d×%d) = %v —— 超過輸出上限", tc.w, tc.h, tc.maxW, tc.maxH, p)
			}
			if tc.w <= tc.maxW && tc.h <= tc.maxH {
				if p.X != tc.w || p.Y != tc.h {
					t.Errorf("來源沒超過上限時被改了：%v（want %dx%d —— 不放大）", p, tc.w, tc.h)
				}
			}
			if tc.w > tc.maxW || tc.h > tc.maxH {
				// 等比：長寬比必須保持 —— 但**只在那兩邊都沒被夾到 1 的時候**。
				// 3×10000 縮到高 2048 時寬度是 0.0009，夾到 1 之後長寬比必然偏掉，
				// 而那是「不消失」這個要求的代價：寧可讓比例偏掉，也不要給出一個 0。
				// 沒有這個條件的該這支會在一個正確的行為上紅，而症狀讀起來像「等比壞了」。
				if p.X > 1 && p.Y > 1 {
					if p.X*tc.h > p.Y*tc.w+tc.w || p.X*tc.h < p.Y*tc.w-tc.w {
						t.Errorf("長寬比跑掉了：%v from %dx%d", p, tc.w, tc.h)
					}
				}
			}
		})
	}
	// 那兩個「夾到上限」的分支（fitSize 裡的 w > maxWidth 與 h > maxHeight）在這裡是
	// 走不到的：scale 是從兩個上限裡取較小的那個，因此 w*scale ≤ maxWidth 恆成立。
	// 它們是防禦性的（改四捨五入時它們就會有意義），因此留著而不是刪掉。
}

// TestEncodeImageRejectsNonReencodedKinds 守住「GIF／WebP 不走編碼器」。
//
// 這一條不是裝飾：它把「這兩種格式被剝而不是被重編碼」這個決定釘在一個函式上。
// 若哪天有人把 GIF 接進 gif.Encode（單影格），那第一個症状是動畫消失，而這裡會先紅。
func TestEncodeImageRejectsNonReencodedKinds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for _, kind := range []imageKind{kindGIF, kindWebP, imageKind("bmp")} {
		if _, err := encodeImage(kind, img, 85); err == nil {
			t.Errorf("encodeImage(%q) 回錯誤 = nil —— 那個格式不該走重編碼（見 imagepolicy.go 檔頭第 4 點）", kind)
		} else if !strings.Contains(err.Error(), "不需要重新編碼") {
			t.Errorf("錯誤 = %v，want 講出「這個格式不需要重新編碼」", err)
		}
	}
}

// TestScaleDownGuardsEmptyGeometry 守住「零尺寸的幾何不 panic」。
//
// scaleDown 的第一個分支是一條防禦：fitSize 保證 target 至少 1×1，而來源的尺寸
// 由 imageBounds 保證是正的，因此正常管線走不到這裡。留著這條的意義是「一個壞的
// 來源不會讓這個行程死掉」—— 一個 panic 在上傳路徑裡是 500，而 500 對使用者
// 是「上傳失敗」而對維運是「不知道為什麼」。
func TestScaleDownGuardsEmptyGeometry(t *testing.T) {
	empty := image.NewRGBA(image.Rect(0, 0, 0, 0))
	if out := scaleDown(empty, image.Point{X: 4, Y: 4}); out.Bounds().Dx() != 4 || out.Bounds().Dy() != 4 {
		t.Errorf("來源為空時輸出尺寸 = %v，want 4x4（來源沒有像素可以平均，但目標必須是那個尺寸）", out.Bounds())
	}
	if out := scaleDown(image.NewRGBA(image.Rect(0, 0, 4, 4)), image.Point{X: 0, Y: 0}); out.Bounds().Dx() != 0 {
		t.Errorf("目標為 0 時輸出尺寸 = %v，want 空的（不能 panic，也不能憑空造出像素）", out.Bounds())
	}
}
