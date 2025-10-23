package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClearCart(t *testing.T) {
	mockQuerier := new(MockQuerier)
	server := grpc_server.NewGRPCServer(mockQuerier, nil, nil)

	req := &cartv1.ClearCartRequest{UserId: uuidTest}

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)

	mockQuerier.On("ClearCartProductsByUserID", mock.Anything, userUUID).Return(nil)

	_, err := server.ClearCart(context.Background(), req)

	assert.NoError(t, err)
}
