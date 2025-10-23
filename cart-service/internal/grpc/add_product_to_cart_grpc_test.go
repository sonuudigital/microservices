package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddProductToCart(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	server := grpc_server.NewGRPCServer(mockQuerier, mockProductFetcher, nil)

	req := &cartv1.AddProductToCartRequest{
		UserId:    uuidTest,
		ProductId: productIDTest,
		Quantity:  1,
	}

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{}, pgx.ErrNoRows)
	mockQuerier.On("CreateCart", mock.Anything, userUUID).Return(repository.Cart{ID: pgtype.UUID{}}, nil)
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]grpc_server.Product{
		productIDTest: {ID: productIDTest, Price: 99.99},
	}, nil)
	mockQuerier.On("AddOrUpdateProductInCart", mock.Anything, mock.Anything).Return(repository.CartsProduct{ID: pgtype.UUID{}}, nil)

	_, err := server.AddProductToCart(context.Background(), req)

	assert.NoError(t, err)
}
