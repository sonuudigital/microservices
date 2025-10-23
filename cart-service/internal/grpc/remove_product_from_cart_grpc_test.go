package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRemoveProductFromCart(t *testing.T) {
	mockQuerier := new(MockQuerier)
	server := grpc_server.NewGRPCServer(mockQuerier, nil, nil)

	req := &cartv1.RemoveProductFromCartRequest{
		UserId:    uuidTest,
		ProductId: productIDTest,
	}

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	var productUUID pgtype.UUID
	_ = productUUID.Scan(productIDTest)

	mockQuerier.On("RemoveProductFromCart", mock.Anything, repository.RemoveProductFromCartParams{UserID: userUUID, ProductID: productUUID}).Return(nil)

	_, err := server.RemoveProductFromCart(context.Background(), req)

	assert.NoError(t, err)
}
