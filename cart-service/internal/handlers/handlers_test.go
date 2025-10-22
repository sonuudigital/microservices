package handlers_test

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/clients"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/stretchr/testify/mock"
)

const (
	uuidTest                   = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	cartUUIDTest               = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
	cartsURL                   = "/api/carts"
	productIDTest              = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
	testProductTitleMsg        = "Test Product"
	invalidUUIDPathTest        = "invalid-uuid"
	networkErrorMsg            = "network error"
	dbErrorMsg                 = "db error"
	productsPath               = "/products"
	productsIDPath             = "/products/" + productIDTest
	invalidUserIDErrorTitleMsg = "Invalid user ID"
	userIDHeader               = "X-User-ID"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) AddOrUpdateProductInCart(ctx context.Context, arg repository.AddOrUpdateProductInCartParams) (repository.CartsProduct, error) {
	args := m.Called(ctx, arg)
	if c, ok := args.Get(0).(repository.CartsProduct); ok {
		return c, args.Error(1)
	}
	return repository.CartsProduct{}, args.Error(1)
}

func (m *MockQuerier) GetCartByUserID(ctx context.Context, userID pgtype.UUID) (repository.Cart, error) {
	args := m.Called(ctx, userID)
	if c, ok := args.Get(0).(repository.Cart); ok {
		return c, args.Error(1)
	}
	return repository.Cart{}, args.Error(1)
}

func (m *MockQuerier) CreateCart(ctx context.Context, userID pgtype.UUID) (repository.Cart, error) {
	args := m.Called(ctx, userID)

	if err := args.Error(1); err != nil {
		return repository.Cart{}, err
	}

	if c, ok := args.Get(0).(repository.Cart); ok {
		return c, args.Error(1)
	}

	return repository.Cart{}, args.Error(1)
}

func (m *MockQuerier) DeleteCartByUserID(ctx context.Context, userID pgtype.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockQuerier) GetCartProductsByCartID(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartProductsByCartIDRow, error) {
	args := m.Called(ctx, cartID)
	if c, ok := args.Get(0).([]repository.GetCartProductsByCartIDRow); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuerier) RemoveProductFromCart(ctx context.Context, arg repository.RemoveProductFromCartParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) ClearCartProductsByUserID(ctx context.Context, userID pgtype.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

type MockProductFetcher struct {
	mock.Mock
}

func (m *MockProductFetcher) GetProductsByIDs(ctx context.Context, ids []string) (map[string]clients.ProductByIDResponse, error) {
	args := m.Called(ctx, ids)
	if c, ok := args.Get(0).(map[string]clients.ProductByIDResponse); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}
