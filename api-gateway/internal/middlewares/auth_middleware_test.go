package middlewares_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"

	"github.com/stretchr/testify/assert"
)

const (
	protectedURL = "/protected"
)

func TestAuthMiddleware(t *testing.T) {
	logger := logs.NewSlogLogger()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	privKeyBytes, err := x509.MarshalECPrivateKey(privKey)
	assert.NoError(t, err)
	privKeyPem := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privKeyBytes})

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	assert.NoError(t, err)
	pubKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})

	jwtManager, err := auth.NewJWTManager(privKeyPem, pubKeyPem, "test-issuer", "test-audience", 15*time.Minute)
	assert.NoError(t, err)

	middleware := middlewares.AuthMiddleware(jwtManager, logger)

	mockNextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	handlerToTest := middleware(mockNextHandler)

	t.Run("Valid Token", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user-123", "test@example.com")
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
		var problem web.ProblemDetail
		err := json.NewDecoder(rr.Body).Decode(&problem)
		assert.NoError(t, err)
		assert.Equal(t, "Missing authorization header.", problem.Detail)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", protectedURL, nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var problem web.ProblemDetail
		err := json.NewDecoder(rr.Body).Decode(&problem)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid or expired token.", problem.Detail)
	})

	t.Run("Malformed Header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", protectedURL, nil)
		req.Header.Set("Authorization", "NotBearer some-token")
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var problem web.ProblemDetail
		err := json.NewDecoder(rr.Body).Decode(&problem)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid authorization header format.", problem.Detail)
	})

	t.Run("Expired Token", func(t *testing.T) {
		shortLivedJwtManager, err := auth.NewJWTManager(privKeyPem, pubKeyPem, "test-issuer", "test-audience", -1*time.Minute)
		assert.NoError(t, err)
		token, err := shortLivedJwtManager.GenerateToken("user-123", "test@example.com")
		assert.NoError(t, err)

		req, _ := http.NewRequest("GET", protectedURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var problem web.ProblemDetail
		err = json.NewDecoder(rr.Body).Decode(&problem)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid or expired token.", problem.Detail)
	})
}