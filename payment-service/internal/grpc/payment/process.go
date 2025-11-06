package payment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/payment-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type processPaymentReqValidationResult struct {
	orderUUID pgtype.UUID
	userUUID  pgtype.UUID
	amount    pgtype.Numeric
}

func (s *Server) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.Payment, error) {
	s.logger.Debug(
		"ProccesPayment called",
		"orderId",
		req.OrderId,
		"userId",
		req.UserId,
		"amount",
		req.Amount,
	)
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	reqValidation, err := validateProcessPaymentRequest(req)
	if err != nil {
		return nil, err
	}

	repositoryPayment, err := s.processCreateAndUpdatePayment(ctx, reqValidation)
	if err != nil {
		return nil, err
	}

	gRPCPayment, err := toGRPCPayment(repositoryPayment)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert payment: %v", err)
	}

	return gRPCPayment, nil
}

func validateProcessPaymentRequest(req *paymentv1.ProcessPaymentRequest) (*processPaymentReqValidationResult, error) {
	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(req.OrderId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order id format: %s", req.OrderId)
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	var amount pgtype.Numeric
	if err := amount.Scan(fmt.Sprintf("%.2f", req.Amount)); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %f", req.Amount)
	}

	if req.Amount <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "amount must be positive: %f", req.Amount)
	}

	return &processPaymentReqValidationResult{
		orderUUID: orderUUID,
		userUUID:  userUUID,
		amount:    amount,
	}, nil
}

func (s *Server) processCreateAndUpdatePayment(ctx context.Context, reqValidation *processPaymentReqValidationResult) (*repository.Payment, error) {
	repoPayment, err := s.createDBPayment(ctx, reqValidation)
	if err != nil {
		return nil, err
	}

	// Simulate payment processing logic here (e.g., interacting with a payment gateway)

	updatedRepoPayment, err := s.updateDBPaymentStatus(ctx, repoPayment)
	if err != nil {
		return nil, err
	}

	return updatedRepoPayment, nil
}

func (s *Server) createDBPayment(ctx context.Context, reqValidation *processPaymentReqValidationResult) (*repository.Payment, error) {
	repositoryPayment, err := s.querier.CreatePayment(ctx, repository.CreatePaymentParams{
		OrderID: reqValidation.orderUUID,
		UserID:  reqValidation.userUUID,
		Amount:  reqValidation.amount,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create payment: %v", err)
	}
	return &repositoryPayment, nil
}

func (s *Server) updateDBPaymentStatus(ctx context.Context, repoPayment *repository.Payment) (*repository.Payment, error) {
	statusUUID, err := s.querier.GetPaymentStatusByName(ctx, "SUCCEEDED")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get 'SUCCEEDED' status ID: %v", err)
	}

	updatedRepoPayment, err := s.querier.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
		ID:     repoPayment.ID,
		Status: statusUUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update payment status: %v", err)
	}

	return &updatedRepoPayment, nil
}
