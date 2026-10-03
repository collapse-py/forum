// 正確樣本：所有封鎖中介層的組裝都集中在 applyRateLimit 這一個函式裡。
package good

import "net/http"

type Server struct{ trustedProxies int }

type RateLimiter struct{}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc { return next }

type store struct{}

func (s *Server) withBlocklistHandler(rl *RateLimiter, st *store, next http.HandlerFunc) http.HandlerFunc {
	return next
}

func (s *Server) isAnyCandidateBanned(r *http.Request, st *store, p int) (bool, int, error) {
	return false, 0, nil
}

// 唯一允許的組裝點。
func (s *Server) applyRateLimit(rl *RateLimiter, next http.HandlerFunc, skipGet bool) http.HandlerFunc {
	if !skipGet {
		return s.withBlocklistHandler(rl, nil, next)
	}
	limited := s.withBlocklistHandler(rl, nil, next)
	return func(w http.ResponseWriter, r *http.Request) {
		limited(w, r)
	}
}

// 兩個薄包裝都走 applyRateLimit，不自己組裝 —— 這正是這條規則要守住的形狀。
func (s *Server) rateLimit(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return s.applyRateLimit(rl, next, true)
}

func (s *Server) rateLimitAllMethods(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return s.applyRateLimit(rl, next, false)
}
