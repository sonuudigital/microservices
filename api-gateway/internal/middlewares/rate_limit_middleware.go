package middlewares

import (
	"net"
	"net/http"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"golang.org/x/time/rate"
)

const (
	UnknownClient = iota
	AuthenticatedClient
)

type RateLimitConfig struct {
	Rate  rate.Limit
	Burst int
}

type RateLimiterMiddleware struct {
	logger     logs.Logger
	rateLimits map[int]RateLimitConfig
	limiter    *redis_rate.Limiter
	isEnabled  bool
}

func NewRateLimiterMiddleware(logger logs.Logger, rateLimits map[int]RateLimitConfig, limiter *redis_rate.Limiter, isEnabled bool) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		logger:     logger,
		rateLimits: rateLimits,
		limiter:    limiter,
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

		rlConfig := rl.getRateLimitConfig(clientHaveClaims)
		limit := redis_rate.Limit{
			Rate:   int(rlConfig.Rate),
			Period: time.Second,
			Burst:  rlConfig.Burst,
		}

		res, err := rl.limiter.Allow(r.Context(), identifier, limit)
		if err != nil {
			rl.logger.Error("could not check rate limit", "error", err)
			web.RespondWithError(w, rl.logger, r, http.StatusInternalServerError, "Internal Server Error", "Could not process request.")
			return
		}

		if res.Allowed == 0 {
			web.RespondWithError(w, rl.logger, r, http.StatusTooManyRequests, "Too Many Requests", "You have exceeded the request limit.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiterMiddleware) getRateLimitConfig(isAuthenticated bool) RateLimitConfig {
	if isAuthenticated {
		return rl.rateLimits[AuthenticatedClient]
	}
	return rl.rateLimits[UnknownClient]
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
