package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	product_service_mock "github.com/sonuudigital/microservices/product-service/internal/mock"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	uint8ArrayType = "[]uint8"
)

func TestUpdateStockBatch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		req := &productv1.UpdateStockBatchRequest{
			Updates: []*productv1.StockUpdate{
				{Id: uuidTest, Quantity: 5},
				{Id: uuidTest2, Quantity: 10},
			},
		}

		mockQuerier.On("UpdateStockBatch", mock.Anything, mock.AnythingOfType(uint8ArrayType)).
			Return(nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectDel(productCachePrefix + uuidTest).SetVal(1)
		redisMock.ExpectDel(productCachePrefix + uuidTest2).SetVal(1)

		res, err := server.UpdateStockBatch(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Success - Single Update", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		req := &productv1.UpdateStockBatchRequest{
			Updates: []*productv1.StockUpdate{
				{Id: uuidTest, Quantity: 3},
			},
		}

		mockQuerier.On("UpdateStockBatch", mock.Anything, mock.AnythingOfType(uint8ArrayType)).
			Return(nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectDel(productCachePrefix + uuidTest).SetVal(1)

		res, err := server.UpdateStockBatch(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Empty Updates", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		req := &productv1.UpdateStockBatchRequest{
			Updates: []*productv1.StockUpdate{},
		}

		res, err := server.UpdateStockBatch(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "no updates provided")
		mockQuerier.AssertNotCalled(t, "UpdateStockBatch", mock.Anything, mock.Anything)
	})

	t.Run("Nil Updates", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		req := &productv1.UpdateStockBatchRequest{
			Updates: nil,
		}

		res, err := server.UpdateStockBatch(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "no updates provided")
		mockQuerier.AssertNotCalled(t, "UpdateStockBatch", mock.Anything, mock.Anything)
	})

	t.Run("Database Error", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		req := &productv1.UpdateStockBatchRequest{
			Updates: []*productv1.StockUpdate{
				{Id: uuidTest, Quantity: 5},
			},
		}

		mockQuerier.On("UpdateStockBatch", mock.Anything, mock.AnythingOfType(uint8ArrayType)).
			Return(errors.New("database error")).Once()

		res, err := server.UpdateStockBatch(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to update stock batch")
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		req := &productv1.UpdateStockBatchRequest{
			Updates: []*productv1.StockUpdate{
				{Id: uuidTest, Quantity: 5},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := server.UpdateStockBatch(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "UpdateStockBatch", mock.Anything, mock.Anything)
	})

	t.Run("Large Batch Update", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)

		updates := make([]*productv1.StockUpdate, 10)
		for i := range int32(10) {
			updates[i] = &productv1.StockUpdate{
				Id:       uuidTest,
				Quantity: i + 1,
			}
		}

		req := &productv1.UpdateStockBatchRequest{
			Updates: updates,
		}

		mockQuerier.On("UpdateStockBatch", mock.Anything, mock.AnythingOfType(uint8ArrayType)).
			Return(nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		for range 10 {
			redisMock.ExpectDel(productCachePrefix + uuidTest).SetVal(1)
		}

		res, err := server.UpdateStockBatch(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		mockQuerier.AssertExpectations(t)
	})
}
