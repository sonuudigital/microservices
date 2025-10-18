package handlers_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"

	"github.com/stretchr/testify/assert"
)

const (
	emailTest = "test@example.com"
	loginURL  = "/api/auth/login"
)

func TestLoginHandler(t *testing.T) {
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

	t.Run("Successful Login", func(t *testing.T) {
		mockUserService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(handlers.UserResponse{
				ID:    "user-123",
				Email: emailTest,
			})
		}))
		defer mockUserService.Close()
		os.Setenv("USER_SERVICE_URL", mockUserService.URL)

		authHandler := handlers.NewAuthHandler(logger, jwtManager)

		loginReq := handlers.LoginRequest{Email: emailTest, Password: "password"}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		authHandler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp handlers.LoginResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)

		assert.NotEmpty(t, resp.Token)
	})

	t.Run("Unauthorized from user-service", func(t *testing.T) {
		mockUserService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(web.ProblemDetail{
				Title:  "Unauthorized",
				Status: http.StatusUnauthorized,
				Detail: "invalid credentials",
			})
		}))
		defer mockUserService.Close()
		os.Setenv("USER_SERVICE_URL", mockUserService.URL)

		authHandler := handlers.NewAuthHandler(logger, jwtManager)

		loginReq := handlers.LoginRequest{Email: emailTest, Password: "wrong-password"}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		authHandler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		var problem web.ProblemDetail
		err := json.NewDecoder(rr.Body).Decode(&problem)
		assert.NoError(t, err)
		assert.Equal(t, "invalid credentials", problem.Detail)
	})

	t.Run("user-service is down", func(t *testing.T) {
		os.Setenv("USER_SERVICE_URL", "http://localhost:12345")
		authHandler := handlers.NewAuthHandler(logger, jwtManager)

		loginReq := handlers.LoginRequest{Email: emailTest, Password: "password"}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		authHandler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	})
}