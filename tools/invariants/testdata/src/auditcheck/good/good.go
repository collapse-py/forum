// 正確樣本：每一種合法寫法都不該被報。
package good

import "errors"

type Server struct{}

type request struct{}

type tx struct{}

func (t *tx) Commit() error   { return nil }
func (t *tx) Rollback() error { return nil }

func (s *Server) beginAdminTx(r *request) (*tx, error) { return &tx{}, nil }
func (s *Server) recordAdminAction(r *request, t *tx, a string) error {
	return errors.New("audit failed")
}

// if 的初始敘述（這個專案 22 處呼叫端全部是這個形狀）。
func (s *Server) checkedInIfInit(r *request) {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if err := s.recordAdminAction(r, tx, "update"); err != nil {
		return
	}
	tx.Commit()
}

// 先賦值再檢查。
func (s *Server) assignedThenChecked(r *request) {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return
	}
	defer tx.Rollback()
	auditErr := s.recordAdminAction(r, tx, "update")
	if auditErr != nil {
		return
	}
	tx.Commit()
}

// 直接回傳稽核錯誤（blocklist.go 的 recordBlockAction 就是這個形狀）。
func (s *Server) returnedAuditError(r *request) error {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.recordAdminAction(r, tx, "block"); err != nil {
		return err
	}
	return tx.Commit()
}

// 把稽核錯誤送到別處（仍然算「有使用」）。
func (s *Server) forwardedAuditError(r *request) error {
	tx, err := s.beginAdminTx(r)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := auditGuard(s.recordAdminAction(r, tx, "update")); err != nil {
		return err
	}
	tx.Commit()
	return nil
}

func auditGuard(err error) error { return err }

// 完全不碰 beginAdminTx 的交易：不是後台操作，規則不適用。
func (s *Server) plainTransaction(t *tx) {
	t.Commit()
}

// var 宣告形狀。
func (s *Server) declaredWithVar(r *request) {
	tx, _ := s.beginAdminTx(r)
	defer tx.Rollback()
	if err := s.recordAdminAction(r, tx, "update"); err != nil {
		return
	}
	tx.Commit()
}
