package order

import (
	"context"

	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.Order, error) {
	s.logger.Debug("CreateOrder called", "userId", req.UserId)
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	cart, err := s.getCart(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	gRPCOrder, err := s.repository.CreateOrder(ctx, req.UserId, cart.TotalPrice, cart.Products)
	if err != nil {
		s.logger.Error(
			"failed to create order and outbox event",
			"error", err,
			"userId", req.UserId,
			"cartId", cart.Id,
		)
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	s.logger.Debug(
		"order created and outbox event recorded",
		"orderId", gRPCOrder.Id,
		"userId", gRPCOrder.UserId,
		"totalAmount", gRPCOrder.TotalAmount,
	)

	payment, err := s.processPayment(ctx, gRPCOrder.Id, req.UserId, cart.TotalPrice)
	if err != nil {
		s.logger.Error(
			"order created but payment processing failed",
			"error", err,
			"orderId", gRPCOrder.Id,
			"userId", req.UserId,
			"amount", cart.TotalPrice,
		)

		if err := s.repository.CancelOrder(ctx, gRPCOrder.Id); err != nil {
			s.logger.Error(
				"failed to cancel order after payment failure",
				"error", err,
				"orderId", gRPCOrder.Id,
			)
		} else {
			s.logger.Info(
				"order cancelled successfully after payment failure",
				"orderId", gRPCOrder.Id,
			)
		}

		return nil, err
	}

	s.logger.Info(
		"order created successfully with payment processed",
		"orderId", gRPCOrder.Id,
		"userId", gRPCOrder.UserId,
		"paymentId", payment.Id,
		"totalAmount", gRPCOrder.TotalAmount,
	)

	return gRPCOrder, nil
}

func (s *Server) getCart(ctx context.Context, userID string) (*cartv1.GetCartResponse, error) {
	s.logger.Debug("fetching cart for user", "userId", userID)

	cart, err := s.clients.CartServiceClient.GetCart(ctx, &cartv1.GetCartRequest{
		UserId: userID,
	})
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.NotFound:
			return nil, status.Errorf(codes.FailedPrecondition, "cannot create order: cart not found for user %s", userID)
		case codes.Unavailable, codes.DeadlineExceeded:
			return nil, status.Errorf(codes.Unavailable, "cart service temporarily unavailable: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to get cart: %v", err)
		}
	}

	if len(cart.Products) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot create order: cart is empty for user %s", userID)
	}

	if cart.TotalPrice <= 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot create order: invalid cart total price %.2f", cart.TotalPrice)
	}

	s.logger.Debug(
		"Fetched cart for user",
		"userId", userID,
		"cartId", cart.Id,
		"itemsCount", len(cart.Products),
		"totalPrice", cart.TotalPrice,
	)

	return cart, nil
}

func (s *Server) processPayment(ctx context.Context, orderID, userID string, amount float64) (*paymentv1.Payment, error) {
	payment, err := s.clients.PaymentServiceClient.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{
		OrderId: orderID,
		UserId:  userID,
		Amount:  amount,
	})
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.FailedPrecondition:
			return nil, status.Errorf(codes.FailedPrecondition, "payment failed for order %s: %v", orderID, err)
		case codes.Unavailable, codes.DeadlineExceeded:
			return nil, status.Errorf(codes.Unavailable, "payment service temporarily unavailable: %v", err)
		case codes.InvalidArgument:
			return nil, status.Errorf(codes.InvalidArgument, "invalid payment data for order %s: %v", orderID, err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
		}
	}

	s.logger.Debug(
		"Payment processed",
		"paymentId", payment.Id,
		"orderId", payment.OrderId,
		"status", payment.Status,
	)

	return payment, nil
}
