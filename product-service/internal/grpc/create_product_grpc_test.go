package grpc_test

import (
	"context"
	"errors"
	"testing"

	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateProduct(t *testing.T) {
	req := &productv1.CreateProductRequest{
		Name:          "Test Product",
		Description:   "Test Description",
		Price:         99.99,
		StockQuantity: 100,
	}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewServer(mockQuerier)
		mockQuerier.On("CreateProduct", mock.Anything, mock.AnythingOfType("repository.CreateProductParams")).
			Return(repository.Product{Name: req.Name}, nil).Once()

		res, err := server.CreateProduct(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.Name, res.Name)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewServer(mockQuerier)
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
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewServer(mockQuerier)
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
