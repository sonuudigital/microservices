package handlers_test

import (
	"bytes"
	"context"
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
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emailTest = "test@example.com"
	loginURL  = "/api/auth/login"
)

type mockUserServiceClient struct {
	mock.Mock
}

func (m *mockUserServiceClient) AuthorizeUser(ctx context.Context, in *userv1.AuthorizeUserRequest, opts ...grpc.CallOption) (*userv1.User, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.User), args.Error(1)
}

func (m *mockUserServiceClient) CreateUser(ctx context.Context, in *userv1.CreateUserRequest, opts ...grpc.CallOption) (*userv1.User, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.User), args.Error(1)
}

func (m *mockUserServiceClient) GetUserByID(ctx context.Context, in *userv1.GetUserByIDRequest, opts ...grpc.CallOption) (*userv1.User, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.User), args.Error(1)
}

func TestLoginHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	os.Setenv("COOKIE_AUTH_NAME", "auth_token")
	t.Cleanup(func() {
		os.Unsetenv("COOKIE_AUTH_NAME")
	})

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
		mockClient := new(mockUserServiceClient)
		authHandler := handlers.NewAuthHandler(logger, jwtManager, mockClient)

		mockClient.On("AuthorizeUser", mock.Anything, mock.Anything).Return(&userv1.User{Id: "user-123", Email: emailTest, Username: "testuser"}, nil).Once()

		loginReq := handlers.LoginRequest{Email: emailTest, Password: "password"}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		authHandler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp handlers.UserResponse
		err := json.NewDecoder(rr.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, "user-123", resp.ID)
		assert.Equal(t, emailTest, resp.Email)
		assert.Equal(t, "testuser", resp.Username)

		cookie := rr.Result().Cookies()
		assert.Len(t, cookie, 1)
		assert.Equal(t, "auth_token", cookie[0].Name)
		assert.NotEmpty(t, cookie[0].Value)
		assert.True(t, cookie[0].HttpOnly)
		assert.Equal(t, "/", cookie[0].Path)
		assert.Equal(t, http.SameSiteStrictMode, cookie[0].SameSite)

		mockClient.AssertExpectations(t)
	})

	t.Run("Unauthorized from user-service", func(t *testing.T) {
		mockClient := new(mockUserServiceClient)
		authHandler := handlers.NewAuthHandler(logger, jwtManager, mockClient)

		mockClient.On("AuthorizeUser", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Unauthenticated, "invalid credentials")).Once()

		loginReq := handlers.LoginRequest{Email: emailTest, Password: "wrong-password"}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		authHandler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockClient.AssertExpectations(t)
	})

	t.Run("user-service is down", func(t *testing.T) {
		mockClient := new(mockUserServiceClient)
		authHandler := handlers.NewAuthHandler(logger, jwtManager, mockClient)

		mockClient.On("AuthorizeUser", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Unavailable, "service unavailable")).Once()

		loginReq := handlers.LoginRequest{Email: emailTest, Password: "password"}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", loginURL, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		authHandler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
		mockClient.AssertExpectations(t)
	})
}

func TestLogoutHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	os.Setenv("COOKIE_AUTH_NAME", "auth_token")
	t.Cleanup(func() {
		os.Unsetenv("COOKIE_AUTH_NAME")
		os.Unsetenv("ENV")
	})

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

	tests := []struct {
		name   string
		env    string
		secure bool
	}{
		{name: "EnvDevSecureFalse", env: "dev", secure: false},
		{name: "EnvProdSecureTrue", env: "prod", secure: true},
		{name: "EnvEmptySecureFalse", env: "", secure: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				os.Setenv("ENV", tt.env)
			} else {
				os.Unsetenv("ENV")
			}

			mockClient := new(mockUserServiceClient)
			authHandler := handlers.NewAuthHandler(logger, jwtManager, mockClient)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			rr := httptest.NewRecorder()

			req.AddCookie(&http.Cookie{
				Name:     "auth_token",
				Value:    "token",
				Path:     "/",
				HttpOnly: true,
			})

			authHandler.LogoutHandler(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)

			var clearedCookie *http.Cookie
			cookies := rr.Result().Cookies()
			for i := len(cookies) - 1; i >= 0; i-- {
				if cookies[i].Name == "auth_token" {
					clearedCookie = cookies[i]
					break
				}
			}

			assert.NotNil(t, clearedCookie)
			assert.Equal(t, "", clearedCookie.Value)
			assert.Equal(t, "/", clearedCookie.Path)
			assert.True(t, clearedCookie.HttpOnly)
			assert.Equal(t, http.SameSiteStrictMode, clearedCookie.SameSite)
			assert.Equal(t, -1, clearedCookie.MaxAge)
			assert.True(t, clearedCookie.Expires.Before(time.Now()))
			assert.Equal(t, tt.secure, clearedCookie.Secure)
		})
	}
}
