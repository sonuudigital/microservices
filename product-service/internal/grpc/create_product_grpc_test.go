package grpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
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

func TestCreateProduct(t *testing.T) {
	req := &productv1.CreateProductRequest{
		CategoryId:    "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22",
		Name:          "Test Product",
		Description:   "Test Description",
		Price:         99.99,
		StockQuantity: 100,
	}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)
		mockQuerier.On("CreateProduct", mock.Anything, mock.AnythingOfType("repository.CreateProductParams")).
			Return(repository.Product{Name: req.Name}, nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectHSet(mock.Anything, mock.Anything).SetVal(1)
		redisMock.ExpectExpire(mock.Anything, 10*time.Minute).SetVal(true)

		res, err := server.CreateProduct(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.Name, res.Name)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)
		mockQuerier.On("CreateProduct", mock.Anything, mock.AnythingOfType("repository.CreateProductParams")).
			Return(repository.Product{}, errors.New("db error")).Once()

		res, err := server.CreateProduct(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(product_service_mock.MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewServer(logs.NewSlogLogger(), mockQuerier, redisClient)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := server.CreateProduct(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything)
	})
}
