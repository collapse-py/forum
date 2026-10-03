// 違規樣本：管理端點漏掉授權，或把授權放在 method 分派之後。
package bad

import "net/http"

type Server struct{}

type handlerFunc = func(http.ResponseWriter, *http.Request)

type muxStub struct{}

func (m *muxStub) HandleFunc(pattern string, h handlerFunc) {}

var mux = &muxStub{}

var srv = &Server{}

// 沒有任何身分檢查：任何人都能呼叫這個後台端點。而且因為未通過時不會寫
// 稽核紀錄，這種探測在稽核日誌裡完全看不到。
//
// 診斷位置在函式宣告（fn.Pos()），因此 want 註解與宣告同一行。
func (s *Server) handleAdminPosts(w http.ResponseWriter, r *http.Request) { // want "沒有呼叫 requireAdminForum"
	switch r.Method {
	case http.MethodGet:
		s.list(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// 身分檢查在 method 分派之後：未登入者可以用 405 反覆探測路由是否存在。
// 診斷位置在那個 if 敘述，因此 want 註解與它同一行。
func (s *Server) handleAdminReports(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.list(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	if !s.requireAdminForum(w, r) { // want "的 requireAdminForum 呼叫在 method 分派之後"
		return
	}
}

// 單一方法端點也沒有身分檢查。
func (s *Server) handleAdminLog(w http.ResponseWriter, r *http.Request) { // want "沒有呼叫 requireAdminForum"
	s.list(w, r)
}

// 分流器把其中一支導到未授權的 handler。
// 診斷位置是被分流到的那一支的**函式宣告**，而不是呼叫點 ——
// 那才是漏掉授權、而路由實際會到達的地方。
func (s *Server) handleAdminExport(w http.ResponseWriter, r *http.Request) {
	if isCSV(r) {
		s.handleAdminTagExport(w, r)
		return
	}
	s.handleAdminMonitor(w, r)
}

func isCSV(r *http.Request) bool { return true }

// 兩支都有授權，因此分流器自己不需要 —— 這一組刻意**不**被報出來。
// 它是那個誤報的對照組：若規則只認「分流器自己必須有 guard」，
// 這個函式就會被報，而它其實完全安全。
func (s *Server) handleAdminSessionDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		s.handleAdminMonitor(w, r)
		return
	}
	s.handleAdminMonitor(w, r)
}

func (s *Server) requireAdminForum(w http.ResponseWriter, r *http.Request) bool {
	return true
}

func (s *Server) handleAdminMonitor(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	s.list(w, r)
}

func (s *Server) handleAdminTagExport(w http.ResponseWriter, r *http.Request) { // want "沒有呼叫 requireAdminForum"
	s.list(w, r)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {}

func register() {
	mux.HandleFunc("/api/admin/forum/posts", srv.handleAdminPosts)
	mux.HandleFunc("/api/admin/forum/reports", srv.handleAdminReports)
	mux.HandleFunc("/api/admin/log", srv.handleAdminLog)
	mux.HandleFunc("/api/admin/export/users.csv", srv.handleAdminExport)
	mux.HandleFunc("/api/admin/sessions", srv.handleAdminSessionDispatch)
}
