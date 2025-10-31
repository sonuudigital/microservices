package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
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

func TestGetProductsByIDs(t *testing.T) {
	req := &productv1.GetProductsByIDsRequest{
		Ids: []string{uuidTest, uuidTest2},
	}

	t.Run("Success - All from Cache", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		cachedProduct1 := map[string]string{
			"id":            uuidTest,
			"categoryId":    categoryUID,
			"name":          "Cached Product 1",
			"description":   "From Cache",
			"price":         "99.99",
			"stockQuantity": "100",
			"createdAt":     "1698624000",
			"updatedAt":     "1698624000",
		}
		cachedProduct2 := map[string]string{
			"id":            uuidTest2,
			"categoryId":    categoryUID,
			"name":          "Cached Product 2",
			"description":   "From Cache 2",
			"price":         "149.99",
			"stockQuantity": "50",
			"createdAt":     "1698624000",
			"updatedAt":     "1698624000",
		}

		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetVal(cachedProduct1)
		redisMock.ExpectHGetAll(productCachePrefix + uuidTest2).SetVal(cachedProduct2)

		res, err := server.GetProductsByIDs(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Products, 2)
		mockQuerier.AssertNotCalled(t, "GetProductsByIDs", mock.Anything, mock.Anything)
	})

	t.Run("Success - Partial Cache, Rest from DB", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		cachedProduct := map[string]string{
			"id":            uuidTest,
			"categoryId":    categoryUID,
			"name":          "Cached Product",
			"description":   "From Cache",
			"price":         "99.99",
			"stockQuantity": "100",
			"createdAt":     "1698624000",
			"updatedAt":     "1698624000",
		}

		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetVal(cachedProduct)
		redisMock.ExpectHGetAll(productCachePrefix + uuidTest2).SetVal(map[string]string{})

		var pgUUID2 pgtype.UUID
		_ = pgUUID2.Scan(uuidTest2)

		mockQuerier.On("GetProductsByIDs", mock.Anything, []pgtype.UUID{pgUUID2}).
			Return([]repository.Product{
				{ID: pgUUID2, Name: "DB Product"},
			}, nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectHSet(mock.Anything, mock.Anything).SetVal(1)
		redisMock.ExpectExpire(mock.Anything, 10*time.Minute).SetVal(true)

		res, err := server.GetProductsByIDs(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Products, 2)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Success - All from DB", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetVal(map[string]string{})
		redisMock.ExpectHGetAll(productCachePrefix + uuidTest2).SetVal(map[string]string{})

		var pgUUID1, pgUUID2 pgtype.UUID
		_ = pgUUID1.Scan(uuidTest)
		_ = pgUUID2.Scan(uuidTest2)

		mockQuerier.On("GetProductsByIDs", mock.Anything, []pgtype.UUID{pgUUID1, pgUUID2}).
			Return([]repository.Product{
				{ID: pgUUID1, Name: "DB Product 1"},
				{ID: pgUUID2, Name: "DB Product 2"},
			}, nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectHSet(mock.Anything, mock.Anything).SetVal(1)
		redisMock.ExpectExpire(mock.Anything, 10*time.Minute).SetVal(true)
		redisMock.ExpectHSet(mock.Anything, mock.Anything).SetVal(1)
		redisMock.ExpectExpire(mock.Anything, 10*time.Minute).SetVal(true)

		res, err := server.GetProductsByIDs(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Products, 2)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Empty IDs", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		emptyReq := &productv1.GetProductsByIDsRequest{Ids: []string{}}

		res, err := server.GetProductsByIDs(context.Background(), emptyReq)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Products, 0)
		mockQuerier.AssertNotCalled(t, "GetProductsByIDs", mock.Anything, mock.Anything)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetVal(map[string]string{})

		var pgUUID1 pgtype.UUID
		_ = pgUUID1.Scan(uuidTest)

		mockQuerier.On("GetProductsByIDs", mock.Anything, []pgtype.UUID{pgUUID1}).
			Return([]repository.Product{}, assert.AnError).Once()

		singleReq := &productv1.GetProductsByIDsRequest{Ids: []string{uuidTest}}

		res, err := server.GetProductsByIDs(context.Background(), singleReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Redis Pipeline Error - Fallback to DB", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		redisMock.ExpectHGetAll(productCachePrefix + uuidTest).SetErr(redis.Nil)

		var pgUUID1 pgtype.UUID
		_ = pgUUID1.Scan(uuidTest)

		mockQuerier.On("GetProductsByIDs", mock.Anything, []pgtype.UUID{pgUUID1}).
			Return([]repository.Product{
				{ID: pgUUID1, Name: "DB Product"},
			}, nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectHSet(mock.Anything, mock.Anything).SetVal(1)
		redisMock.ExpectExpire(mock.Anything, 10*time.Minute).SetVal(true)

		singleReq := &productv1.GetProductsByIDsRequest{Ids: []string{uuidTest}}

		res, err := server.GetProductsByIDs(context.Background(), singleReq)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Products, 1)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid UUID Format", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		invalidReq := &productv1.GetProductsByIDsRequest{
			Ids: []string{"invalid-uuid"},
		}

		redisMock.ExpectHGetAll("product:invalid-uuid").SetVal(map[string]string{})

		res, err := server.GetProductsByIDs(context.Background(), invalidReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		mockQuerier.AssertNotCalled(t, "GetProductsByIDs", mock.Anything, mock.Anything)
	})
}
