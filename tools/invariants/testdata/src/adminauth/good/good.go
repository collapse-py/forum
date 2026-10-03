// 正確樣本：管理端點都在 method 分派之前授權。
package good

import "net/http"

type Server struct{}

type handlerFunc = func(http.ResponseWriter, *http.Request)

type muxStub struct{}

func (m *muxStub) HandleFunc(pattern string, h handlerFunc) {}

var mux = &muxStub{}

func (s *Server) sessions() *sessionStub { return &sessionStub{} }

type sessionStub struct{}

func (s *sessionStub) IsAdmin(r *http.Request) bool { return true }

// 集合端點：身分檢查在 switch 之前（forum_admin_handlers.go 的形狀）。
func (s *Server) handleAdminPosts(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.list(w, r)
	case http.MethodPost:
		s.create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// 只有一種方法的端點：沒有 switch，因此只要有授權檢查就合格。
func (s *Server) handleAdminMonitor(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	s.report(w, r)
}

// 單一資源端點：仍然要先授權。
func (s *Server) handleAdminPost(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.show(w, r)
	case http.MethodPut:
		s.update(w, r)
	case http.MethodDelete:
		s.destroy(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// 純分流器：它自己沒有授權檢查，但兩支被分流到的 handler 都有。
// 這是 /api/admin/forum/posts/{id}… 的實際形狀 —— ServeMux 無法把 /{id}/pin
// 單獨註冊成另一條路由（它會吃掉整個 /{id} 前綴），因此只能靠分流器。
func (s *Server) handleAdminPostOrPin(w http.ResponseWriter, r *http.Request) {
	if isPinRequest(r) {
		s.handleAdminPin(w, r)
		return
	}
	s.handleAdminPost(w, r)
}

func isPinRequest(r *http.Request) bool { return true }

// 分流器的其中一支：授權檢查在 method 檢查之前。
func (s *Server) handleAdminPin(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminForum(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) requireAdminForum(w http.ResponseWriter, r *http.Request) bool {
	if !s.sessions().IsAdmin(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func (s *Server) list(w http.ResponseWriter, r *http.Request)    {}
func (s *Server) create(w http.ResponseWriter, r *http.Request)  {}
func (s *Server) show(w http.ResponseWriter, r *http.Request)    {}
func (s *Server) update(w http.ResponseWriter, r *http.Request)  {}
func (s *Server) destroy(w http.ResponseWriter, r *http.Request) {}
func (s *Server) report(w http.ResponseWriter, r *http.Request)  {}

func register() {
	mux.HandleFunc("/api/admin/forum/posts", s0.handleAdminPosts)
	mux.HandleFunc("/api/admin/forum/posts/", s0.handleAdminPostOrPin)
	mux.HandleFunc("/api/admin/monitor", s0.handleAdminMonitor)

	// 非後台路由不受這條規則約束：公開讀取端點刻意不需要管理員身分，
	// 一般使用者端點則由 requireLogin / requireLoginForWrite 把關。
	mux.HandleFunc("/api/forum/posts", s0.handleForumPosts)
	mux.HandleFunc("/api/forum/announcement", s0.handleForumAnnouncement)
	mux.HandleFunc("/healthz", s0.handleHealth)

	// 後台**頁面**（/admin/monitor…）不在 /api/admin/ 底下，且它們的 handler
	// 是閉包；兩者都不該被這條規則拖進來。
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {})
}

var s0 = &Server{}

func (s *Server) handleForumPosts(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) handleForumAnnouncement(w http.ResponseWriter, r *http.Request) {}
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request)            {}
