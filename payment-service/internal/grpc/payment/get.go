package payment

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.Payment, error) {
	s.logger.Debug("GetPayment called", "paymentId", req.Id)
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var uuid pgtype.UUID
	if err := uuid.Scan(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment id format: %s", req.Id)
	}

	payment, err := s.querier.GetPaymentByID(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "payment with id %s not found", req.Id)
		} else {
			return nil, status.Errorf(codes.Internal, "failed to retrieve payment: %v", err)
		}
	}

	grpcPayment, err := mapRepositoryToGRPC(&payment)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert payment: %v", err)
	}

	return grpcPayment, nil
}
