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

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	cartIDTest      = "cart-123"
	userIDTest      = "user-123"
	userEmailTest   = "test@test.com"
	productIDTest   = "product-123"
	apiCartsURLPath = "/api/carts"
	bearerWithSpace = "Bearer "
)

type mockCartServiceClient struct {
	mock.Mock
}

func (m *mockCartServiceClient) GetCart(ctx context.Context, in *cartv1.GetCartRequest, opts ...grpc.CallOption) (*cartv1.GetCartResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cartv1.GetCartResponse), args.Error(1)
}

func (m *mockCartServiceClient) AddProductToCart(ctx context.Context, in *cartv1.AddProductToCartRequest, opts ...grpc.CallOption) (*cartv1.AddProductToCartResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cartv1.AddProductToCartResponse), args.Error(1)
}

func (m *mockCartServiceClient) RemoveProductFromCart(ctx context.Context, in *cartv1.RemoveProductFromCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *mockCartServiceClient) ClearCart(ctx context.Context, in *cartv1.ClearCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *mockCartServiceClient) DeleteCart(ctx context.Context, in *cartv1.DeleteCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func setupCartTest(t *testing.T) (*httptest.Server, *mockCartServiceClient, *auth.JWTManager) {
	logger := logs.NewSlogLogger()
	mockClient := new(mockCartServiceClient)
	cartHandler := handlers.NewCartHandler(logger, mockClient)

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

	authMW := middlewares.AuthMiddleware(jwtManager, logger)

	mux := http.NewServeMux()
	mux.Handle(apiCartsURLPath, authMW(http.HandlerFunc(cartHandler.GetCartHandler)))
	mux.Handle("POST /api/carts/products", authMW(http.HandlerFunc(cartHandler.AddProductToCartHandler)))
	mux.Handle("DELETE /api/carts/products/{productId}", authMW(http.HandlerFunc(cartHandler.RemoveProductFromCartHandler)))
	mux.Handle("DELETE /api/carts", authMW(http.HandlerFunc(cartHandler.ClearCartHandler)))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, mockClient, jwtManager
}

func TestGetCartHandler(t *testing.T) {
	server, mockClient, jwtManager := setupCartTest(t)

	t.Run("Success", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userIDTest, userEmailTest)
		assert.NoError(t, err)

		mockClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{UserId: userIDTest}).
			Return(&cartv1.GetCartResponse{Id: cartIDTest, UserId: userIDTest}, nil).Once()

		req, _ := http.NewRequest("GET", server.URL+apiCartsURLPath, nil)
		req.Header.Set("Authorization", bearerWithSpace+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockClient.AssertExpectations(t)
	})
}

func TestAddProductToCartHandler(t *testing.T) {
	server, mockClient, jwtManager := setupCartTest(t)

	addProductReq := handlers.AddProductToCartRequest{
		ProductID: productIDTest,
		Quantity:  1,
	}
	body, _ := json.Marshal(addProductReq)

	t.Run("Success", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userIDTest, userEmailTest)
		assert.NoError(t, err)

		mockClient.On("AddProductToCart", mock.Anything, mock.Anything).Return(&cartv1.AddProductToCartResponse{}, nil).Once()

		req, _ := http.NewRequest("POST", server.URL+"/api/carts/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerWithSpace+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockClient.AssertExpectations(t)
	})
}

func TestRemoveProductFromCartHandler(t *testing.T) {
	server, mockClient, jwtManager := setupCartTest(t)

	t.Run("Success", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userIDTest, userEmailTest)
		assert.NoError(t, err)

		mockClient.On("RemoveProductFromCart", mock.Anything, &cartv1.RemoveProductFromCartRequest{UserId: userIDTest, ProductId: productIDTest}).
			Return(&emptypb.Empty{}, nil).Once()

		req, _ := http.NewRequest("DELETE", server.URL+"/api/carts/products/"+productIDTest, nil)
		req.Header.Set("Authorization", bearerWithSpace+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		mockClient.AssertExpectations(t)
	})
}

func TestClearCartHandler(t *testing.T) {
	server, mockClient, jwtManager := setupCartTest(t)

	t.Run("Success", func(t *testing.T) {
		token, err := jwtManager.GenerateToken(userIDTest, userEmailTest)
		assert.NoError(t, err)

		mockClient.On("ClearCart", mock.Anything, &cartv1.ClearCartRequest{UserId: userIDTest}).
			Return(&emptypb.Empty{}, nil).Once()

		req, _ := http.NewRequest("DELETE", server.URL+apiCartsURLPath, nil)
		req.Header.Set("Authorization", bearerWithSpace+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		mockClient.AssertExpectations(t)
	})
}
