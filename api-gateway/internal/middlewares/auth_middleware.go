package middlewares

import (
	"context"
	"net/http"
	"os"

	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
)

type contextKey string

const userClaimsKey contextKey = "userClaims"

func AuthMiddleware(jwtManager *auth.JWTManager, logger logs.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(os.Getenv("COOKIE_AUTH_NAME"))
			if err != nil {
				if err == http.ErrNoCookie {
					web.RespondWithError(w, logger, r, http.StatusUnauthorized, "Unauthorized", "Missing authentication cookie.")
					return
				}
				web.RespondWithError(w, logger, r, http.StatusBadRequest, "Bad Request", "Invalid authentication cookie.")
				return
			}

			tokenString := cookie.Value

			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				logger.Warn("invalid token", "error", err)
				web.RespondWithError(w, logger, r, http.StatusUnauthorized, "Unauthorized", "Invalid or expired token.")
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
