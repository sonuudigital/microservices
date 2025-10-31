package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClearCart(t *testing.T) {
	t.Setenv("CART_TTL_HOURS", "24")

	mockQuerier := new(MockQuerier)
	server := grpc_server.NewGRPCServer(mockQuerier, nil, nil)

	req := &cartv1.ClearCartRequest{UserId: uuidTest}

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	var cartUUID pgtype.UUID
	_ = cartUUID.Scan("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")

	existingCart := repository.Cart{
		ID:        cartUUID,
		UserID:    userUUID,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(existingCart, nil)
	mockQuerier.On("ClearCartProductsByUserID", mock.Anything, userUUID).Return(nil)

	_, err := server.ClearCart(context.Background(), req)

	assert.NoError(t, err)
}
