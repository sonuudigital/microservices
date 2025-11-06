package payment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/payment-service/internal/grpc/payment"
	"github.com/sonuudigital/microservices/payment-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetPayment(t *testing.T) {
	var paymentUUID pgtype.UUID
	_ = paymentUUID.Scan(testPaymentID)

	var orderUUID pgtype.UUID
	_ = orderUUID.Scan(testOrderID)

	var userUUID pgtype.UUID
	_ = userUUID.Scan(testUserID)

	var statusUUID pgtype.UUID
	_ = statusUUID.Scan("d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14")

	req := &paymentv1.GetPaymentRequest{Id: testPaymentID}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		var amount pgtype.Numeric
		_ = amount.Scan("99.99")

		mockPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    amount,
			Status:    statusUUID,
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("GetPaymentByID", mock.Anything, paymentUUID).
			Return(mockPayment, nil).Once()

		res, err := server.GetPayment(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, testPaymentID, res.Id)
		assert.Equal(t, testOrderID, res.OrderId)
		assert.Equal(t, testUserID, res.UserId)
		assert.Equal(t, 99.99, res.Amount)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Payment Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		mockQuerier.On("GetPaymentByID", mock.Anything, paymentUUID).
			Return(repository.Payment{}, pgx.ErrNoRows).Once()

		res, err := server.GetPayment(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Contains(t, st.Message(), "not found")
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid Payment ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		invalidReq := &paymentv1.GetPaymentRequest{Id: "invalid-uuid"}

		res, err := server.GetPayment(context.Background(), invalidReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "invalid payment id format")
		mockQuerier.AssertNotCalled(t, "GetPaymentByID", mock.Anything, mock.Anything)
	})

	t.Run("Database Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		mockQuerier.On("GetPaymentByID", mock.Anything, paymentUUID).
			Return(repository.Payment{}, errors.New("database connection error")).Once()

		res, err := server.GetPayment(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to retrieve payment")
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := server.GetPayment(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "GetPaymentByID", mock.Anything, mock.Anything)
	})

	t.Run("Invalid Amount in Payment", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		mockPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    pgtype.Numeric{Valid: false},
			Status:    statusUUID,
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("GetPaymentByID", mock.Anything, paymentUUID).
			Return(mockPayment, nil).Once()

		res, err := server.GetPayment(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to convert payment")
		mockQuerier.AssertExpectations(t)
	})
}
