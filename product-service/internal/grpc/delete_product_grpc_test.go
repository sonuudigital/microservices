package grpc_test

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	product_service_mock "github.com/sonuudigital/microservices/product-service/internal/mock"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDeleteProduct(t *testing.T) {
	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	req := &productv1.DeleteProductRequest{Id: uuidTest}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(mockQuerier, redisClient)
		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(repository.Product{}, nil).Once()
		mockQuerier.On("DeleteProduct", mock.Anything, pgUUID).Return(nil).Once()

		redisMock.ExpectDel("product:" + uuidTest).SetVal(1)

		_, err := server.DeleteProduct(context.Background(), req)

		assert.NoError(t, err)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(mockQuerier, redisClient)
		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(repository.Product{}, pgx.ErrNoRows).Once()

		_ = redisMock
		_, err := server.DeleteProduct(context.Background(), req)

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(mockQuerier, redisClient)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := server.DeleteProduct(ctx, req)

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "GetProduct", mock.Anything, mock.Anything)
		mockQuerier.AssertNotCalled(t, "DeleteProduct", mock.Anything, mock.Anything)
	})
}
