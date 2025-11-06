package payment_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

const (
	repositoryCreatePaymentParamsType       = "repository.CreatePaymentParams"
	repositoryUpdatePaymentStatusParamsType = "repository.UpdatePaymentStatusParams"
)

func TestProcessPayment(t *testing.T) {
	var orderUUID pgtype.UUID
	_ = orderUUID.Scan(testOrderID)

	var userUUID pgtype.UUID
	_ = userUUID.Scan(testUserID)

	var paymentUUID pgtype.UUID
	_ = paymentUUID.Scan(testPaymentID)

	var statusUUID pgtype.UUID
	_ = statusUUID.Scan(testsPaymentStatusID)

	req := &paymentv1.ProcessPaymentRequest{
		OrderId: testOrderID,
		UserId:  testUserID,
		Amount:  99.99,
	}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		var amount pgtype.Numeric
		_ = amount.Scan("99.99")

		createdPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    amount,
			Status:    pgtype.UUID{},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		updatedPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    amount,
			Status:    statusUUID,
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("CreatePayment", mock.Anything, mock.AnythingOfType(repositoryCreatePaymentParamsType)).
			Return(createdPayment, nil).Once()
		mockQuerier.On("GetPaymentStatusByName", mock.Anything, "SUCCEEDED").
			Return(statusUUID, nil).Once()
		mockQuerier.On("UpdatePaymentStatus", mock.Anything, mock.AnythingOfType(repositoryUpdatePaymentStatusParamsType)).
			Return(updatedPayment, nil).Once()

		res, err := server.ProcessPayment(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, testPaymentID, res.Id)
		assert.Equal(t, testOrderID, res.OrderId)
		assert.Equal(t, testUserID, res.UserId)
		assert.Equal(t, 99.99, res.Amount)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid Order ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		invalidReq := &paymentv1.ProcessPaymentRequest{
			OrderId: "invalid-uuid",
			UserId:  testUserID,
			Amount:  99.99,
		}

		res, err := server.ProcessPayment(context.Background(), invalidReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "invalid order id format")
		mockQuerier.AssertNotCalled(t, "CreatePayment", mock.Anything, mock.Anything)
	})

	t.Run("Invalid User ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		invalidReq := &paymentv1.ProcessPaymentRequest{
			OrderId: testOrderID,
			UserId:  "invalid-uuid",
			Amount:  99.99,
		}

		res, err := server.ProcessPayment(context.Background(), invalidReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "invalid user id format")
		mockQuerier.AssertNotCalled(t, "CreatePayment", mock.Anything, mock.Anything)
	})

	t.Run("Invalid Amount", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		invalidReq := &paymentv1.ProcessPaymentRequest{
			OrderId: testOrderID,
			UserId:  testUserID,
			Amount:  -10.00,
		}

		res, err := server.ProcessPayment(context.Background(), invalidReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "amount must be positive")
		mockQuerier.AssertNotCalled(t, "CreatePayment", mock.Anything, mock.Anything)
	})

	t.Run("Create Payment Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		mockQuerier.On("CreatePayment", mock.Anything, mock.AnythingOfType(repositoryCreatePaymentParamsType)).
			Return(repository.Payment{}, errors.New("database error")).Once()

		res, err := server.ProcessPayment(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to create payment")
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Get Payment Status Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		var amount pgtype.Numeric
		_ = amount.Scan("99.99")

		createdPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    amount,
			Status:    pgtype.UUID{},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("CreatePayment", mock.Anything, mock.AnythingOfType(repositoryCreatePaymentParamsType)).
			Return(createdPayment, nil).Once()
		mockQuerier.On("GetPaymentStatusByName", mock.Anything, "SUCCEEDED").
			Return(pgtype.UUID{}, errors.New("status not found")).Once()

		res, err := server.ProcessPayment(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to get 'SUCCEEDED' status ID")
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Update Payment Status Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		var amount pgtype.Numeric
		_ = amount.Scan("99.99")

		createdPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    amount,
			Status:    pgtype.UUID{},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("CreatePayment", mock.Anything, mock.AnythingOfType(repositoryCreatePaymentParamsType)).
			Return(createdPayment, nil).Once()
		mockQuerier.On("GetPaymentStatusByName", mock.Anything, "SUCCEEDED").
			Return(statusUUID, nil).Once()
		mockQuerier.On("UpdatePaymentStatus", mock.Anything, mock.AnythingOfType(repositoryUpdatePaymentStatusParamsType)).
			Return(repository.Payment{}, errors.New("update failed")).Once()

		res, err := server.ProcessPayment(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to update payment status")
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := server.ProcessPayment(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "CreatePayment", mock.Anything, mock.Anything)
	})

	t.Run("Zero Amount", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		zeroReq := &paymentv1.ProcessPaymentRequest{
			OrderId: testOrderID,
			UserId:  testUserID,
			Amount:  0,
		}

		res, err := server.ProcessPayment(context.Background(), zeroReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "amount must be positive")
		mockQuerier.AssertNotCalled(t, "CreatePayment", mock.Anything, mock.Anything)
	})

	t.Run("Large Amount", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := payment.New(logs.NewSlogLogger(), mockQuerier)

		largeReq := &paymentv1.ProcessPaymentRequest{
			OrderId: testOrderID,
			UserId:  testUserID,
			Amount:  999999.99,
		}

		var amount pgtype.Numeric
		_ = amount.Scan("999999.99")

		createdPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    amount,
			Status:    pgtype.UUID{},
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		updatedPayment := repository.Payment{
			ID:        paymentUUID,
			OrderID:   orderUUID,
			UserID:    userUUID,
			Amount:    amount,
			Status:    statusUUID,
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("CreatePayment", mock.Anything, mock.AnythingOfType(repositoryCreatePaymentParamsType)).
			Return(createdPayment, nil).Once()
		mockQuerier.On("GetPaymentStatusByName", mock.Anything, "SUCCEEDED").
			Return(statusUUID, nil).Once()
		mockQuerier.On("UpdatePaymentStatus", mock.Anything, mock.AnythingOfType(repositoryUpdatePaymentStatusParamsType)).
			Return(updatedPayment, nil).Once()

		res, err := server.ProcessPayment(context.Background(), largeReq)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 999999.99, res.Amount)
		mockQuerier.AssertExpectations(t)
	})
}
