package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	product_service_mock "github.com/sonuudigital/microservices/product-service/internal/mock"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetProduct(t *testing.T) {
	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	req := &productv1.GetProductRequest{Id: uuidTest}

	t.Run("Success - Cache Miss", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetVal(map[string]string{})

		mockQuerier.On("GetProduct", mock.Anything, pgUUID).
			Return(repository.Product{ID: pgUUID, Name: "Test Product"}, nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectHSet(mock.Anything, mock.Anything).SetVal(1)
		redisMock.ExpectExpire(mock.Anything, 10*time.Minute).SetVal(true)

		res, err := server.GetProduct(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, uuidTest, res.Id)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Success - Cache Hit", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		cachedProduct := map[string]string{
			"id":            uuidTest,
			"categoryId":    "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22",
			"name":          "Cached Product",
			"description":   "From Cache",
			"price":         "99.99",
			"stockQuantity": "100",
			"createdAt":     "1698624000",
			"updatedAt":     "1698624000",
		}
		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetVal(cachedProduct)

		res, err := server.GetProduct(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "Cached Product", res.Name)
		assert.Equal(t, "From Cache", res.Description)
		mockQuerier.AssertNotCalled(t, "GetProduct", mock.Anything, mock.Anything)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetVal(map[string]string{})

		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(repository.Product{}, pgx.ErrNoRows).Once()

		res, err := server.GetProduct(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := server.GetProduct(ctx, req)

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "GetProduct", mock.Anything, mock.Anything)
	})
}
