package handlers_test

import (
	"api-gateway/internal/handlers"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"shared/auth"
	"shared/logs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	emailTest = "test@example.com"
	loginURL  = "/api/auth/login"
)

func TestLoginHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	jwtManager := auth.NewJWTManager("test-secret", "test-issuer", "test-audience", 15*time.Minute)

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
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid credentials"))
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
		assert.Equal(t, "invalid credentials\n", rr.Body.String())
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
