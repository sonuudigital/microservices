package middlewares

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
)

const (
	UnknownClient = iota
	AuthenticatedClient
)

type RateLimitConfig struct {
	RatePerSecond int
	Burst         int
}

type RateLimiterMiddleware struct {
	logger     logs.Logger
	rateLimits map[int]RateLimitConfig
	limiter    *redis_rate.Limiter
	isEnabled  bool
}

func NewRateLimiterMiddleware(
	logger logs.Logger,
	rateLimits map[int]RateLimitConfig,
	limiter *redis_rate.Limiter,
	isEnabled bool,
) *RateLimiterMiddleware {
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
			web.RespondWithError(w, rl.logger, r, http.StatusInternalServerError,
				"Internal Server Error", "Could not process request.")
			return
		}

		rlConfig := rl.getRateLimitConfig(clientHaveClaims)

		limit := redis_rate.Limit{
			Rate:   rlConfig.RatePerSecond,
			Period: time.Second,
			Burst:  rlConfig.Burst,
		}

		res, err := rl.limiter.Allow(r.Context(), identifier, limit)
		if err != nil {
			rl.logger.Error("could not check rate limit", "error", err)
			web.RespondWithError(w, rl.logger, r, http.StatusInternalServerError,
				"Internal Server Error", "Could not process request.")
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(res.RetryAfter).Unix(), 10))

		if res.Allowed == 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(res.RetryAfter.Seconds())))
			web.RespondWithError(w, rl.logger, r, http.StatusTooManyRequests,
				"Too Many Requests", "You have exceeded the request limit.")
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
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0]), nil
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip, nil
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	return ip, nil
}
