package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUpdateProduct(t *testing.T) {
	req := &productv1.UpdateProductRequest{
		Id:            uuidTest,
		CategoryId:    "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22",
		Name:          "Updated Product",
		Description:   "Updated Description",
		Price:         129.99,
		StockQuantity: 50,
	}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewServer(mockQuerier)
		mockQuerier.On("UpdateProduct", mock.Anything, mock.AnythingOfType("repository.UpdateProductParams")).
			Return(repository.Product{ID: pgtype.UUID{Bytes: [16]byte{}, Valid: true}, Name: req.Name}, nil).Once()

		res, err := server.UpdateProduct(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.Name, res.Name)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewServer(mockQuerier)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := server.UpdateProduct(ctx, req)

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "UpdateProduct", mock.Anything, mock.Anything)
	})
}
