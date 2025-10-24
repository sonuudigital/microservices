package grpc_test

import (
	"context"
	"testing"

	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestListProducts(t *testing.T) {
	req := &productv1.ListProductsRequest{Limit: 10, Offset: 0}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewServer(mockQuerier)
		mockQuerier.On("ListProductsPaginated", mock.Anything, mock.AnythingOfType("repository.ListProductsPaginatedParams")).
			Return([]repository.Product{{Name: "Product 1"}, {Name: "Product 2"}}, nil).Once()

		res, err := server.ListProducts(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Products, 2)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewServer(mockQuerier)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := server.ListProducts(ctx, req)

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "ListProductsPaginated", mock.Anything, mock.Anything)
	})
}
