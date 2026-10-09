// 違規樣本：recordAdminAction 的回傳值被丟棄，因此稽核是「盡力記錄」。
//
// 這是 docs/DEVELOPMENT.md 點名的失效模式：使用者照樣拿到 200、沒有任何日誌、沒有任何
// 狀態碼差異，而「誰動了這筆資料」從此查不到。
//
// 注意：analysistest 的期望註解必須與診斷被回報的那一行**同一行**。
// 這裡的診斷位置是 tx.Commit()（因為那是「提交發生而稽核沒有」這個事實
// 成立的地方）。
package bad

import "errors"

type Server struct{}

type request struct{}

type tx struct{}

func (t *tx) Commit() error   { return nil }
func (t *tx) Rollback() error { return nil }

func (s *Server) beginAdminTx(r *request) (*tx, error) { return &tx{}, nil }
func (s *Server) recordAdminAction(r *request, t *tx, action string) error {
	return errors.New("audit failed")
}

// 稽核寫了但沒看錯誤：表達式被丟棄，等於稽核降級為「盡力記錄」。
func (s *Server) discardAuditError(r *request) {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	s.recordAdminAction(r, tx, "update")
	tx.Commit() // want "以 beginAdminTx 開啟交易並提交"
}

// 完全沒有稽核：後台寫入路徑沒有留下任何稽核紀錄。
func (s *Server) noAuditAtAll(r *request) {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	tx.Commit() // want "以 beginAdminTx 開啟交易並提交"
}

// 稽核在 Commit 之後：即使有檢查，順序仍舊讓稽核失去意義。
func (s *Server) auditAfterCommit(r *request) {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	tx.Commit() // want "以 beginAdminTx 開啟交易並提交"
	if err := s.recordAdminAction(r, tx, "update"); err != nil {
		return
	}
}

// 稽核在另一個交易裡：該函式的第一個 Commit 之前沒有稽核，因此仍應被報。
// 這是刻意的保守判定 —— 分析器不做跨交易的資料流追蹤，而漏報比誤報更糟。
func (s *Server) auditInAnotherTx(r *request) {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	other, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer other.Rollback()
	tx.Commit() // want "以 beginAdminTx 開啟交易並提交"
	if err := s.recordAdminAction(r, other, "update"); err != nil {
		return
	}
	other.Commit()
}

// 交易被回滾：沒有任何東西被提交，因此不需要稽核。
//
// 這一段是負向的：它**不應該**被報出來，否則這條規則會在「提早返回」的
// 路徑上製造無法修正的誤報。
func (s *Server) rollsBackWithoutAudit(r *request) {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if err != nil {
		return
	}
}

// 自由函式形狀的違規：beginAdminTx 被重構成自由函式之後，規則仍然要抓得到。
//
// 少了這一筆，把交易改用自由函式就能讓整條規則安靜失效 —— 而 CI 全綠。
// 見 isNamedCall 的說明。自由函式與方法同名在 Go 裡是合法的（方法名不佔用
// 套件作用域），因此這個樣本可以同時有兩種形狀。
func discardAuditErrorWithFreeFunctions(r *request) {
	tx, err := beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	recordAdminAction(r, tx, "update")
	tx.Commit() // want "以 beginAdminTx 開啟交易並提交"
}

func beginAdminTx(r *request) (*tx, error) { return &tx{}, nil }
func recordAdminAction(r *request, t *tx, action string) error {
	return errors.New("audit failed")
}
