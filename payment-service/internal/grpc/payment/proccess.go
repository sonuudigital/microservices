package payment

import (
	"context"

	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	//TODO: implement payment processing logic
	return nil, status.Errorf(codes.Unimplemented, "method ProcessPayment not implemented")
}
