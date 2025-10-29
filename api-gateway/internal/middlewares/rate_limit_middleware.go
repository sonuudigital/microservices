package middlewares

import (
	"net"
	"net/http"
	"sync"

	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"golang.org/x/time/rate"
)

const (
	UnknownClient = iota
	AuthenticatedClient
)

type RateLimit struct {
	Rate  rate.Limit
	Burst int
}

type RateLimiterMiddleware struct {
	logger     logs.Logger
	clients    map[string]*rate.Limiter
	rateLimits map[int]RateLimit
	isEnabled  bool
	mu         sync.Mutex
}

func NewRateLimiterMiddleware(logger logs.Logger, rateLimits map[int]RateLimit, isEnabled bool) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		logger:     logger,
		clients:    make(map[string]*rate.Limiter),
		rateLimits: rateLimits,
		isEnabled:  isEnabled,
	}
}

func (rl *RateLimiterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.isEnabled {
			next.ServeHTTP(w, r)
			return
		}

		identifier, clientHaveClaims, err := rl.getUserClaimsOrIPAddress(r)
		if err != nil {
			rl.logger.Error("could not parse IP from remote address", "error", err)
			web.RespondWithError(w, rl.logger, r, http.StatusInternalServerError, "Internal Server Error", "Could not process request.")
			return
		}

		rl.mu.Lock()
		limiter, exists := rl.clients[identifier]
		if !exists {
			limiter = rl.createLimiterForClient(identifier, clientHaveClaims)
		}
		rl.mu.Unlock()

		if !limiter.Allow() {
			web.RespondWithError(w, rl.logger, r, http.StatusTooManyRequests, "Too Many Requests", "You have exceeded the request limit.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiterMiddleware) createLimiterForClient(identifier string, isAuthenticated bool) *rate.Limiter {
	var rlConfig RateLimit
	if isAuthenticated {
		rlConfig = rl.rateLimits[AuthenticatedClient]
	} else {
		rlConfig = rl.rateLimits[UnknownClient]
	}

	limiter := rate.NewLimiter(rlConfig.Rate, rlConfig.Burst)
	rl.clients[identifier] = limiter
	return limiter
}

func (rl *RateLimiterMiddleware) getUserClaimsOrIPAddress(r *http.Request) (string, bool, error) {
	claims, ok := GetUserClaims(r)
	if ok {
		return claims.Subject, ok, nil
	}

	ip, err := rl.extractIPAddress(r)
	if err != nil {
		return "", false, err
	}

	return ip, false, nil
}

func (rl *RateLimiterMiddleware) extractIPAddress(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	return ip, nil
}
