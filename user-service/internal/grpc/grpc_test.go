package grpc_test

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"github.com/stretchr/testify/mock"
)

const (
	testEmail = "test@example.com"
	testUUID  = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.User, error) {
	args := m.Called(ctx, arg)
	if u, ok := args.Get(0).(repository.User); ok {
		return u, args.Error(1)
	}
	return repository.User{}, args.Error(1)
}

func (m *MockQuerier) GetUserByEmail(ctx context.Context, email string) (repository.User, error) {
	args := m.Called(ctx, email)
	if u, ok := args.Get(0).(repository.User); ok {
		return u, args.Error(1)
	}
	return repository.User{}, args.Error(1)
}

func (m *MockQuerier) GetUserByID(ctx context.Context, id pgtype.UUID) (repository.User, error) {
	args := m.Called(ctx, id)
	if u, ok := args.Get(0).(repository.User); ok {
		return u, args.Error(1)
	}
	return repository.User{}, args.Error(1)
}
