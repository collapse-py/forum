/*
圖片的所有權與刪除（backend/forum/httpapi/forum_media_handlers.go）。

一、這個檔案解決什麼

	圖片上傳到獨立的檔案伺服器之後，本站對「這個檔名是誰上傳的」原本沒有
	任何紀錄 —— forum_posts.image_url 與 forum_profiles.avatar_url 只回答
	「誰現在用它」。這個檔案補上兩件事：

	  1. 上傳時寫一筆所有權紀錄（forum_media，見 MigrateMySQL 第 33 步）。
	  2. POST /api/forum/images/delete —— 讓使用者刪除「自己上傳、而且已經
	     沒有任何地方在引用」的圖片。

	「換圖」（編輯貼文或頭像時換掉舊檔）不需要這支端點：那兩條路徑在更新
	成功後會自動呼叫 releaseForumImage 回收舊檔。刪除端點負責的是另一種
	情況 —— 上傳了但沒有用到的檔案（選錯圖、發文失敗後重試、放棄草稿）。

二、所有權模型（為什麼不是「引用即所有權」）

	一個檔名可以有三種狀態：

	  新制   forum_media 有列：擁有者以上傳者為準。attach 與 delete 都只認
	         這一列，刻意不接受「引用關係」的救援 —— 否則「先把別人的圖片
	         掛到自己的貼文上」就會變成「可以刪除別人的檔案」。
	  舊制   forum_media 沒有列（這個表存在之前上傳的檔案）：沒有歷史可查，
	         退路是「它已經是自己內容的一部分」—— 自己的貼文引用它，或它
	         就是自己的頭像。不足以證明者一律拒絕。
	  未引用 沒有任何貼文或頭像引用它：只有新制的擁有者可以刪。

	為什麼引用計數把軟刪除的貼文也算進去（forumImageReferenceCount 不過濾
	deleted_at）：貼文可以還原（後台的 /restore），而還原後圖片必須還在。
	把軟刪文的引用排除會讓「還原」與「刪圖」互相破壞，而那個破壞只在
	「先刪圖、後還原」的順序下才看得到。

三、檔案與資料列的刪除順序

	一律「先刪檔案伺服器上的檔案，成功後才刪 forum_media 的列」。反過來的話
	（先刪列、檔案刪除失敗）那個檔案會永遠失去擁有者 —— 端點與換圖流程都
	再也無法回收它。而檔案刪除成功、列刪除失敗只留下一筆「擁有者還在、
	檔案已不存在」的列，重新呼叫刪除即可收斂（deleteForumFile 把 404 視為
	成功，正是為了讓重試是安全的）。

	回收（releaseForumImage）刪列時多一個 owner_email 條件：那一列必須是
	發出請求的人的紀錄。刪檔案本身只看引用計數（那是「舊檔」的定義），
	但紀錄屬於誰是另一回事 —— 兩件事不該互換。

四、這條路徑刻意不記稽核

	與刪除本人貼文同一個理由（見 handleForumPostDelete）：稽核紀錄的 actor
	是管理員，而這是使用者處理自己的檔案。索引與檔案伺服器的失敗都只留
	日誌。
*/
package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"forum/forum/logger"
)

// forumImageDeleteRequest 是 POST /api/forum/images/delete 的請求主體。
//
//	URL 是任何一種「本站檔案的寫法」——上傳回應的完整網址、資料庫裡存的
//	檔名、或舊式的 /files/x.png 路徑。forumImageFileName 會收斂成純檔名，
//	收斂不出來的值（第三方網址、路徑穿越、不允許的副檔名）一律 400。
type forumImageDeleteRequest struct {
	URL string `json:"url"`
}

// handleForumImageDelete 刪除一張自己上傳、而且沒有被任何貼文或頭像引用的圖片。
//
// 路由已用 requireLogin 包住；handler 內重複確認 email 非空，讓它被單獨
// 重用或日後改動路由時仍然安全（與 handleForumImageUpload 同一個形狀）。
//
// 回應刻意不多說：刪除失敗的兩種原因（不是你的 / 還在使用中）分別回 404
// 與 409，而 404 與「這個檔名不存在」共用同一個回應 —— 讓呼叫端無法用
// 狀態碼差異探測「某個檔名是否存在於本站」。
func (s *Server) handleForumImageDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	email := s.sessions.ResolveUser(r)
	if email == "" {
		unauthorized(w, "請先使用 Google 登入")
		return
	}
	// 這條路徑會刪除外部系統上的檔案，因此與上傳共用同一條不變條件：
	// 跨站請求必須在碰檔案伺服器之前被擋下。
	if !s.isTrustedOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}

	var req forumImageDeleteRequest
	// 4 KB：body 只有一個網址（上限 128 字元的檔名 + 公開網址前綴），
	// 這個上限是灌水防線而不是業務校驗。
	if err := decodeLimitedJSON(w, r, &req, 4<<10); err != nil {
		badRequest(w, "invalid request")
		return
	}
	name := s.forumImageFileName(req.URL)
	if name == "" {
		badRequest(w, "invalid image url")
		return
	}

	owner, err := s.forumMediaOwner(r.Context(), name)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 讀取圖片紀錄失敗 filename=%s: %v", name, err)
		internalError(w, "unable to load image record")
		return
	}
	// owner 為空字串代表沒有紀錄（舊制檔案），此時無法證明所有權，與
	// 「紀錄存在但擁有者是別人」回同一個 404：兩者對呼叫端是同一件事。
	if owner == "" || owner != email {
		http.NotFound(w, r)
		return
	}

	references, err := s.forumImageReferenceCount(r.Context(), name)
	if err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 檢查圖片引用失敗 filename=%s: %v", name, err)
		internalError(w, "unable to check image usage")
		return
	}
	if references > 0 {
		// 409 而不是 400：請求本身合法（這是你的檔案），只是目前的狀態
		// 不允許刪除。使用者能採取的動作是先去改掉引用它的內容。
		writeError(w, http.StatusConflict, "image is still in use")
		return
	}

	if err := s.deleteForumFile(r.Context(), name); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 刪除圖片檔案失敗 filename=%s: %v", name, err)
		// 502：檔案伺服器是上游，與上傳路徑對同樣故障的處理一致。
		writeError(w, http.StatusBadGateway, "unable to delete image")
		return
	}
	if _, err := s.db.ExecContext(r.Context(),
		`DELETE FROM forum_media WHERE filename = ? AND owner_email = ?`, name, email); err != nil {
		logger.ErrorfContext(r.Context(), "[FORUM] 刪除圖片紀錄失敗 filename=%s: %v", name, err)
		internalError(w, "unable to delete image record")
		return
	}
	writeOK(w, map[string]bool{"ok": true})
}

// forumImageLikePattern 是「以 /檔名 結尾」的 LIKE 樣式，供引用計數與舊制
// 所有權退路共用。
//
// 兩個部分各有理由：
//
//   - 前綴 %/：舊資料可能存著 /files/x.png 或完整網址（見 forumImageFileName），
//     因此除了完全相等也接受「最後一段是這個檔名」的值。
//   - escapeLikePattern：檔名允許「_」（合法檔名的白名單裡有它），而 LIKE 的
//     _ 是「任意一個字元」。不跳脫的話 %/a_b.png 也會匹配 /aXb.png，症狀是
//     引用計數多算（刪不掉自己的圖）或所有權退路誤判（把別人的舊檔當成自己
//     引用過的）。搜尋的 content LIKE 用的是同一支跳脫函式。
func forumImageLikePattern(name string) string {
	return "%/" + escapeLikePattern(name)
}

// forumMediaOwner 回傳某個檔名的上傳者；沒有紀錄（舊制檔案）時回空字串。
//
// 刻意用「空字串＝沒有紀錄」而不是 sql.ErrNoRows：呼叫端（所有權判斷與
// 刪除端點）對這兩種情況的處理都是「沒有可用的所有權資訊」，讓呼叫端各自
// 判斷 errors.Is 只會多一種要記得處理的錯誤形狀。
func (s *Server) forumMediaOwner(ctx context.Context, name string) (string, error) {
	var owner string
	err := s.db.QueryRowContext(ctx,
		`SELECT owner_email FROM forum_media WHERE filename = ?`, name).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return owner, nil
}

// forumImageReferenceCount 回傳有幾個地方在引用這個檔名（貼文附圖與頭像合計）。
//
// 兩個刻意的決定：
//
//  1. 含軟刪除的貼文（不過濾 deleted_at）。貼文可以還原（後台的
//     /api/admin/forum/posts/{id}/restore），還原後圖片必須還在 —— 把軟刪文
//     排除會讓「先刪圖、後還原」得到一張破掉的圖，而那個順序是合法的使用
//     者行為。代價是「刪文之後圖片不會自動釋放」，但那本來就不是這條路徑
//     的職責（真正的回收是保留期清理那條線的事）。
//  2. LIKE 退路。資料庫的契約是純檔名（MigrateMySQL 第 18 步），但舊資料
//     可能存著 /files/x.png 或完整網址（見 forumImageFileName 的相容說明），
//     因此除了完全相等也接受「以 /檔名 結尾」的值。
//
// 檔名先過 escapeLikePattern（與搜尋的 content LIKE 同一支）：檔名允許
// 「_」，而 LIKE 的 _ 是「任意一個字元」—— 不跳脫的話 %/a_b.png 也會匹配
// /aXb.png。方向是保守的（多算一個引用只會讓刪除被拒絕），但那個「多算」
// 會讓使用者看到一個不存在的「圖片仍在使用中」，因此仍然不是可接受的行為。
func (s *Server) forumImageReferenceCount(ctx context.Context, name string) (int, error) {
	pattern := forumImageLikePattern(name)
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM forum_posts WHERE image_url = ? OR image_url LIKE ?) +
			(SELECT COUNT(*) FROM forum_profiles WHERE avatar_url = ? OR avatar_url LIKE ?)`,
		name, pattern, name, pattern).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// forumFileAttachable 回答「這個檔案可不可以被這個人掛到自己的內容上」。
//
// 使用時機：建立貼文帶附圖、編輯貼文換圖、儲存個人資料換頭像。三條路徑
// 都必須先問過這一題，否則會出現兩個漏洞：
//
//   - 把別人的檔案掛進自己的內容（熱連結），以及
//   - 更糟的：掛進來之後「換圖」的自動回收會把那個檔案從檔案伺服器上刪掉
//     —— 檔案要不要刪只看引用計數，不問擁有者（見 releaseForumImage）。
//
// 規則見檔頭「所有權模型」：新制只認上傳紀錄，舊制退路是「已經是自己內容
// 的一部分」。空檔名一律拒絕（呼叫端在更早的步驟就會擋掉，這裡是縱深防禦）。
func (s *Server) forumFileAttachable(ctx context.Context, name, email string) (bool, error) {
	if name == "" {
		return false, nil
	}
	owner, err := s.forumMediaOwner(ctx, name)
	if err != nil {
		return false, err
	}
	if owner != "" {
		return owner == email, nil
	}
	// 舊制檔案（沒有上傳紀錄）：退路查詢與引用計數同一個形狀，但多兩個
	// 「必須是自己的內容」的條件。檔名同樣先跳脫 LIKE 的萬用字元 —— 這一條
	// 的誤判方向比引用計數更危險：模式裡的 _ 讓「/files/aXbei.png」也算成
	// 「自己引用過 a_bei.png」，於是別人的舊檔就能被掛到自己的內容上
	//（掛上之後，換圖回收只問引用計數，那個檔案就真的會被刪掉）。
	pattern := forumImageLikePattern(name)
	var count int
	err = s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM forum_posts WHERE author_email = ? AND (image_url = ? OR image_url LIKE ?)) +
			(SELECT COUNT(*) FROM forum_profiles WHERE author_email = ? AND (avatar_url = ? OR avatar_url LIKE ?))`,
		email, name, pattern, email, name, pattern).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// releaseForumImage 在檔案不再被引用時回收它：刪除檔案伺服器上的檔案，
// 然後刪除 forum_media 的列。
//
// 呼叫端有兩處（編輯貼文的換圖／移除圖片、換頭像），全部在使用者請求的
// 流程上，因此這支函式刻意不回傳錯誤、只記日誌：
//
//   - 它跑在主操作（UPDATE）已經成功之後，失敗無法也不該讓那個操作回滾；
//   - 最壞情況是留下一個孤兒檔（與這個功能存在之前一模一樣），而那個狀態
//     可以由檔案伺服器端的清理或將來的保留期機制收斂。
//
// 引用計數查詢失敗時直接放棄（不刪）：把「查不到」當成「沒人用」是最糟糕
// 的誤判方向。
//
// owner 是發出這次請求的人（回收的觸發者），用來把最後那條 DELETE 限制在
// 自己的紀錄上。引用計數只回答「還有沒有人在用」，不問擁有者，因此一顆
// 檔名如果以別人的列存在（正常情況下不會發生：掛上別人的檔案在 attach 就被
// 拒絕了），不該由這裡把那一列刪掉。代價是那種情況下會留下一列指向已刪除
// 檔案的紀錄 —— 它無害（擁有者重新呼叫刪除端點即可收斂，deleteForumFile
// 把 404 視為成功），而「不破壞別人的紀錄」比「讓自己的狀態收斂」重要。
func (s *Server) releaseForumImage(ctx context.Context, name, owner string) {
	if name == "" {
		return
	}
	count, err := s.forumImageReferenceCount(ctx, name)
	if err != nil {
		logger.WarnfContext(ctx, "[FORUM] 檢查圖片引用失敗，略過回收 filename=%s: %v", name, err)
		return
	}
	if count > 0 {
		return
	}
	if err := s.deleteForumFile(ctx, name); err != nil {
		logger.WarnfContext(ctx, "[FORUM] 回收圖片失敗 filename=%s: %v", name, err)
		return
	}
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM forum_media WHERE filename = ? AND owner_email = ?`, name, owner); err != nil {
		logger.WarnfContext(ctx, "[FORUM] 刪除圖片紀錄失敗 filename=%s: %v", name, err)
	}
}

// deleteForumFile 呼叫檔案伺服器的 DELETE /delete 刪除一個檔案。
//
// 檔案伺服器回 404（檔案已經不在）視為成功：呼叫端要的語意是「確保它不
// 存在」，而重試安全正是刪除端點與回收流程都需要的一個性質。
//
// url 參數帶的是 /files/<檔名>：檔案伺服器從路徑的第二段取目錄、最後一段
// 取 Base 作為檔名，且只接受 "files" 這個目錄（見 files_server 的
// handleDelete）。檔名先過本站的 validForumImageFileName 再送出 —— 那一份
// 白名單比檔案伺服器的檢查更嚴，是縱深防禦，不是唯一防線。
//
// 30 秒上限：刪除不涉及 S3 的多段上傳，比上傳的 60 秒短。逾時視為上游故障。
func (s *Server) deleteForumFile(ctx context.Context, name string) error {
	if validForumImageFileName(name) == "" {
		return fmt.Errorf("invalid image filename")
	}
	endpoint := strings.TrimRight(s.cfg.FilesServerURL, "/") +
		"/delete?url=" + url.QueryEscape("/files/"+name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	if s.cfg.FilesServerToken != "" {
		// 未設定時不送這個標頭，與上傳路徑同一條契約（檔案伺服器端
		// Upload.Token 為空等同關閉上傳驗證）。
		req.Header.Set("X-Upload-Token", s.cfg.FilesServerToken)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("file server delete failed: status %d", resp.StatusCode)
	}
	return nil
}

// DeleteFile 是 purge.FileDeleter 的實作：讓背景清理排程能回收檔案伺服器
// 上的檔案，而不必把 FilesServerURL / token 的組裝複製到 purge 套件。
//
// 語意與 deleteForumFile 逐字相同（404 視為成功，因此重試安全），这里只是
// 把未匯出的內部函式暴露成一個稳定的契約：排程的失敗處理（只記日誌）由
// 呼叫端决定，而「怎麼連到檔案伺服器」只有一份實作。
func (s *Server) DeleteFile(ctx context.Context, name string) error {
	return s.deleteForumFile(ctx, name)
}

// discardForumUpload 刪除一個「剛上傳但無法完成登記」的檔案（best-effort）。
//
// 呼叫端有兩處（上傳取得 token 失敗、所有權紀錄寫入失敗），兩處的共同點是
// 檔案已經存在於檔案伺服器，但使用者永遠拿不到它的網址 —— 留著它只是孤兒。
// 與 releaseForumImage 的差異：這裡**不問引用計數**。剛上傳的檔案不可能被
// 引用（它連網址都還沒回給任何人），而「查不到引用就刪」在這條路徑上是多餘
// 的一次資料庫往返。
func (s *Server) discardForumUpload(ctx context.Context, name string) {
	if err := s.deleteForumFile(ctx, name); err != nil {
		logger.WarnfContext(ctx, "[FORUM] 丟棄未完成登記的上傳失敗 filename=%s: %v", name, err)
	}
}

// forumRecordUploadedImage 在上傳成功後寫入所有權紀錄。
//
// 寫入失敗時呼叫端會嘗試把剛上傳的檔案刪掉再回 500（見
// handleForumImageUpload）：沒有紀錄的檔案無法被它的上傳者刪除，也無法被
// 掛進任何內容（attach 檢查會拒絕沒有紀錄的檔名），因此讓它繼續存在只是
// 製造一個永遠回收不了的孤兒 —— 寧可讓上傳失敗，讓使用者重新上傳一次。
func (s *Server) forumRecordUploadedImage(ctx context.Context, name, email string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO forum_media (filename, owner_email, created_at) VALUES (?, ?, ?)`,
		name, email, time.Now())
	return err
}
