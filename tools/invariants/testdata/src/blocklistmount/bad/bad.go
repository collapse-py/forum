// 違規樣本：第二個掛載點。
package bad

import "net/http"

type Server struct{}

type RateLimiter struct{}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc { return next }

type store struct{}

func (s *Server) withBlocklistHandler(rl *RateLimiter, st *store, next http.HandlerFunc) http.HandlerFunc {
	return next
}

func (s *Server) applyRateLimit(rl *RateLimiter, next http.HandlerFunc, skipGet bool) http.HandlerFunc {
	return s.withBlocklistHandler(rl, nil, next)
}

// 第二個掛載點：這條路由會有封鎖檢查，但不在那個唯一的組裝函式裡，因此
// 「先查封鎖、再查限流」的順序與涵蓋範圍就不再由單一處決定。壞掉時沒有任何
// 錯誤訊息：限流照常運作、封鎖名單照常運作、稽核照常寫入。
func (s *Server) rateLimitAllMethods(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return s.withBlocklistHandler(rl, nil, next) // want "在 applyRateLimit 之外呼叫 withBlocklistHandler"
}

// 把封鎖檢查直接掛在某條路由上，繞過 applyRateLimit。
func (s *Server) registerUploadRoute(rl *RateLimiter, next http.HandlerFunc) {
	_ = s.withBlocklistHandler(rl, nil, next) // want "在 applyRateLimit 之外呼叫 withBlocklistHandler"
}
