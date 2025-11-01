package grpc_test

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/stretchr/testify/mock"
)

const (
	uuidTest        = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	productIDTest   = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
	cartUUIDTest    = "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
	cartCachePrefix = "cart:"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) AddOrUpdateProductInCart(ctx context.Context, arg repository.AddOrUpdateProductInCartParams) (repository.CartsProduct, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(repository.CartsProduct), args.Error(1)
}

func (m *MockQuerier) GetCartByUserID(ctx context.Context, userID pgtype.UUID) (repository.Cart, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(repository.Cart), args.Error(1)
}

func (m *MockQuerier) CreateCart(ctx context.Context, userID pgtype.UUID) (repository.Cart, error) {
	args := m.Called(ctx, userID)
	if args.Error(1) != nil {
		return repository.Cart{}, args.Error(1)
	}
	return args.Get(0).(repository.Cart), args.Error(1)
}

func (m *MockQuerier) DeleteCartByUserID(ctx context.Context, userID pgtype.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockQuerier) GetCartProductsByCartID(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartProductsByCartIDRow, error) {
	args := m.Called(ctx, cartID)
	return args.Get(0).([]repository.GetCartProductsByCartIDRow), args.Error(1)
}

func (m *MockQuerier) RemoveProductFromCart(ctx context.Context, arg repository.RemoveProductFromCartParams) error {
	return m.Called(ctx, arg).Error(0)
}

func (m *MockQuerier) ClearCartProductsByUserID(ctx context.Context, userID pgtype.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

type MockProductFetcher struct {
	mock.Mock
}

func (m *MockProductFetcher) GetProductsByIDs(ctx context.Context, ids []string) (map[string]grpc.Product, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(map[string]grpc.Product), args.Error(1)
}
