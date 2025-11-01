package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jackc/pgx/v5/pgtype"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRemoveProductFromCart(t *testing.T) {
	t.Setenv("CART_TTL_HOURS", "24")

	mockQuerier := new(MockQuerier)
	redisClient, _ := redismock.NewClientMock()
	server := grpc_server.NewGRPCServer(mockQuerier, nil, redisClient, nil)

	req := &cartv1.RemoveProductFromCartRequest{
		UserId:    uuidTest,
		ProductId: productIDTest,
	}

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	var cartUUID pgtype.UUID
	_ = cartUUID.Scan("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")
	var productUUID pgtype.UUID
	_ = productUUID.Scan(productIDTest)

	existingCart := repository.Cart{
		ID:        cartUUID,
		UserID:    userUUID,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(existingCart, nil)
	mockQuerier.On("RemoveProductFromCart", mock.Anything, repository.RemoveProductFromCartParams{UserID: userUUID, ProductID: productUUID}).Return(nil)

	_, err := server.RemoveProductFromCart(context.Background(), req)

	assert.NoError(t, err)
}
