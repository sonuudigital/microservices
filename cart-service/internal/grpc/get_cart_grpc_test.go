package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetCart(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	server := grpc_server.NewGRPCServer(mockQuerier, mockProductFetcher, nil)

	req := &cartv1.GetCartRequest{UserId: uuidTest}

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: pgtype.UUID{}}, nil)
	mockQuerier.On("GetCartProductsByCartID", mock.Anything, mock.Anything).Return([]repository.GetCartProductsByCartIDRow{}, nil)
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, mock.Anything).Return(map[string]grpc_server.Product{}, nil)

	_, err := server.GetCart(context.Background(), req)

	assert.NoError(t, err)
}
