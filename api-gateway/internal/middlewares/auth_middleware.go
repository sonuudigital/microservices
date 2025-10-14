package middlewares

import (
	"context"
	"net/http"
	"shared/auth"
	"shared/logs"
	"strings"
)

type contextKey string

const userClaimsKey contextKey = "userClaims"

func AuthMiddleware(jwtManager *auth.JWTManager, logger logs.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				logger.Warn("invalid token", "error", err)
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserClaims(r *http.Request) (*auth.Claims, bool) {
	claims, ok := r.Context().Value(userClaimsKey).(*auth.Claims)
	return claims, ok
}
