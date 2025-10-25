package grpc_test

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/mock"
)

const (
	uuidTest = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateProduct(ctx context.Context, arg repository.CreateProductParams) (repository.Product, error) {
	args := m.Called(ctx, arg)
	if p, ok := args.Get(0).(repository.Product); ok {
		return p, args.Error(1)
	}
	return repository.Product{}, args.Error(1)
}

func (m *MockQuerier) DeleteProduct(ctx context.Context, id pgtype.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) GetProduct(ctx context.Context, id pgtype.UUID) (repository.Product, error) {
	args := m.Called(ctx, id)
	if p, ok := args.Get(0).(repository.Product); ok {
		return p, args.Error(1)
	}
	return repository.Product{}, args.Error(1)
}

func (m *MockQuerier) ListProductsPaginated(ctx context.Context, arg repository.ListProductsPaginatedParams) ([]repository.Product, error) {
	args := m.Called(ctx, arg)
	if p, ok := args.Get(0).([]repository.Product); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuerier) UpdateProduct(ctx context.Context, arg repository.UpdateProductParams) (repository.Product, error) {
	args := m.Called(ctx, arg)
	if p, ok := args.Get(0).(repository.Product); ok {
		return p, args.Error(1)
	}
	return repository.Product{}, args.Error(1)
}

func (m *MockQuerier) GetProductsByIDs(ctx context.Context, productIds []pgtype.UUID) ([]repository.Product, error) {
	args := m.Called(ctx, productIds)
	if p, ok := args.Get(0).([]repository.Product); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuerier) GetProductsByCategoryID(ctx context.Context, categoryID pgtype.UUID) ([]repository.Product, error) {
	args := m.Called(ctx, categoryID)
	if p, ok := args.Get(0).([]repository.Product); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}
