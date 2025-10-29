package middlewares

import (
	"net"
	"net/http"
	"sync"

	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"golang.org/x/time/rate"
)

type RateLimiterMiddleware struct {
	logger    logs.Logger
	limiter   *rate.Limiter
	clients   map[string]*rate.Limiter
	rate      rate.Limit
	burst     int
	isEnabled bool
	mu        sync.Mutex
}

func NewRateLimiterMiddleware(logger logs.Logger, r rate.Limit, b int, isEnabled bool) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		logger:    logger,
		clients:   make(map[string]*rate.Limiter),
		rate:      r,
		burst:     b,
		isEnabled: isEnabled,
	}
}

func (rl *RateLimiterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.isEnabled {
			next.ServeHTTP(w, r)
			return
		}

		var identifier string
		claims, ok := GetUserClaims(r)
		if ok {
			identifier = claims.Subject
		} else {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				rl.logger.Error("could not parse IP from remote address", "error", err)
				web.RespondWithError(w, rl.logger, r, http.StatusInternalServerError, "Internal Server Error", "Could not process request.")
				return
			}
			identifier = ip
		}

		rl.mu.Lock()
		limiter, exists := rl.clients[identifier]
		if !exists {
			limiter = rate.NewLimiter(rl.rate, rl.burst)
			rl.clients[identifier] = limiter
		}
		rl.mu.Unlock()

		if !limiter.Allow() {
			web.RespondWithError(w, rl.logger, r, http.StatusTooManyRequests, "Too Many Requests", "You have exceeded the request limit.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
