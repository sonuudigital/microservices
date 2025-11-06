package payment

import (
	"context"

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

	//TODO: implement payment retrieval logic
	return nil, status.Errorf(codes.Unimplemented, "method GetPayment not implemented")
}
