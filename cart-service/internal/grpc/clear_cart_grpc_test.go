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
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClearCart(t *testing.T) {
	t.Setenv("CART_TTL_HOURS", "24")

	mockQuerier := new(MockQuerier)
	redisClient, redisMock := redismock.NewClientMock()
	server := grpc_server.NewGRPCServer(mockQuerier, nil, redisClient, logs.NewSlogLogger())

	req := &cartv1.ClearCartRequest{UserId: uuidTest}

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	var cartUUID pgtype.UUID
	_ = cartUUID.Scan(cartUUIDTest)

	existingCart := repository.Cart{
		ID:        cartUUID,
		UserID:    userUUID,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(existingCart, nil)
	mockQuerier.On("ClearCartProductsByUserID", mock.Anything, userUUID).Return(nil)

	redisMock.MatchExpectationsInOrder(false)
	redisMock.ExpectDel(cartCachePrefix + uuidTest).SetVal(1)

	_, err := server.ClearCart(context.Background(), req)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}
