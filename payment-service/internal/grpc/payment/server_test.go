package payment_test

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/payment-service/internal/repository"
	"github.com/stretchr/testify/mock"
)

const (
	testPaymentID        = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	testOrderID          = "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
	testUserID           = "c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
	testsPaymentStatusID = "d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreatePayment(ctx context.Context, arg repository.CreatePaymentParams) (repository.Payment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(repository.Payment), args.Error(1)
}

func (m *MockQuerier) GetPaymentByID(ctx context.Context, id pgtype.UUID) (repository.Payment, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(repository.Payment), args.Error(1)
}

func (m *MockQuerier) GetPaymentStatusByName(ctx context.Context, name string) (pgtype.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(pgtype.UUID), args.Error(1)
}

func (m *MockQuerier) UpdatePaymentStatus(ctx context.Context, arg repository.UpdatePaymentStatusParams) (repository.Payment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(repository.Payment), args.Error(1)
}
