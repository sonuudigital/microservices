package middlewares_test

import (
	"api-gateway/internal/middlewares"
	"net/http"
	"net/http/httptest"
	"shared/auth"
	"shared/logs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	protectedURL = "/protected"
)

func TestAuthMiddleware(t *testing.T) {
	logger := logs.NewSlogLogger()
	jwtManager := auth.NewJWTManager("test-secret", "test-issuer", "test-audience", 15*time.Minute)
	middleware := middlewares.AuthMiddleware(jwtManager, logger)

	mockNextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	handlerToTest := middleware(mockNextHandler)

	t.Run("Valid Token", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("test@example.com")
		assert.NoError(t, err)

		req, _ := http.NewRequest("GET", protectedURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "Success", rr.Body.String())
	})

	t.Run("No Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", protectedURL, nil)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "missing authorization header\n", rr.Body.String())
	})

	t.Run("Invalid Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", protectedURL, nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "invalid or expired token\n", rr.Body.String())
	})

	t.Run("Malformed Header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", protectedURL, nil)
		req.Header.Set("Authorization", "NotBearer some-token")
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "invalid authorization header format\n", rr.Body.String())
	})

	t.Run("Expired Token", func(t *testing.T) {
		shortLivedJwtManager := auth.NewJWTManager("test-secret", "test-issuer", "test-audience", -1*time.Minute)
		token, err := shortLivedJwtManager.GenerateToken("test@example.com")
		assert.NoError(t, err)

		req, _ := http.NewRequest("GET", protectedURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "invalid or expired token\n", rr.Body.String())
	})
}
