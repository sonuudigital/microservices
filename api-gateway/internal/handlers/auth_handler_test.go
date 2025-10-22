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
	"testing"
	"time"

	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
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

		mockClient.On("AuthorizeUser", mock.Anything, mock.Anything).Return(&userv1.User{Id: "user-123", Email: emailTest}, nil).Once()

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