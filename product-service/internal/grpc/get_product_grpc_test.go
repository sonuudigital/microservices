package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetProduct(t *testing.T) {
	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	req := &productv1.GetProductRequest{Id: uuidTest}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewGRPCServer(mockQuerier)
		mockQuerier.On("GetProduct", mock.Anything, pgUUID).
			Return(repository.Product{ID: pgUUID, Name: "Test Product"}, nil).Once()

		res, err := server.GetProduct(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, uuidTest, res.Id)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewGRPCServer(mockQuerier)
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
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewGRPCServer(mockQuerier)
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
