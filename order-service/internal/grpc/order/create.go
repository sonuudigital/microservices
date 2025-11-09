package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
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

	DBOrder, err := s.createDBOrderByUserID(ctx, cart)
	if err != nil {
		return nil, err
	}

	gRPCOrder, err := mapRepositoryToGRPC(DBOrder)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to map order to gRPC: %v", err)
	}

	_, err = s.processPayment(ctx, gRPCOrder.Id, req.UserId, gRPCOrder.TotalAmount)
	if err != nil {
		cancelErr := s.cancelDBOrderByID(ctx, DBOrder.ID)
		if cancelErr != nil {
			return nil, errors.Join(err, cancelErr)
		}
		return nil, err
	}

	if err := s.publishOrderCreatedEvent(ctx, gRPCOrder.Id, gRPCOrder.UserId, cart.Products); err != nil {
		s.logger.Error(
			"CRITICAL: payment succeeded but failed to publish OrderCreatedEvent",
			"error", err,
			"orderId", gRPCOrder.Id,
			"userId", gRPCOrder.UserId,
		)

		return nil, status.Errorf(codes.Internal, "failed to publish order created event: Order ID %s", gRPCOrder.Id)
	}

	return gRPCOrder, nil
}

func (s *Server) getCart(ctx context.Context, userID string) (*cartv1.GetCartResponse, error) {
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

func (s *Server) createDBOrderByUserID(ctx context.Context, cart *cartv1.GetCartResponse) (*repository.Order, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(cart.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	var totalAmount pgtype.Numeric
	if err := totalAmount.Scan(fmt.Sprintf("%.2f", cart.TotalPrice)); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse total amount: %v", err)
	}

	order, err := s.querier.CreateOrder(ctx, repository.CreateOrderParams{
		UserID:      userUUID,
		TotalAmount: totalAmount,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	s.logger.Debug(
		"Order created",
		"orderId", order.ID.String(),
		"userId", order.UserID.String(),
		"status", order.Status.String(),
	)

	return &order, nil
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

func (s *Server) cancelDBOrderByID(ctx context.Context, orderUUID pgtype.UUID) error {
	canceledStatusUUID, err := s.querier.GetOrderStatusIDByName(ctx, "CANCELLED")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get CANCELLED status: %v", err)
	}

	_, err = s.querier.UpdateOrderStatus(ctx, repository.UpdateOrderStatusParams{
		ID:     orderUUID,
		Status: canceledStatusUUID,
	})

	if err != nil {
		return status.Errorf(codes.Internal, "failed to cancel order %s: %v", orderUUID.String(), err)
	}

	s.logger.Debug(
		"Order canceled",
		"orderId", orderUUID.String(),
	)

	return nil
}

func (s *Server) publishOrderCreatedEvent(ctx context.Context, orderID, userID string, products []*cartv1.CartProduct) error {
	eventProducts := make([]events.OrderItem, len(products))
	for i, p := range products {
		eventProducts[i] = events.OrderItem{
			ProductID: p.ProductId,
			Quantity:  p.Quantity,
		}
	}

	event := events.OrderCreatedEvent{
		OrderID:  orderID,
		UserID:   userID,
		Products: eventProducts,
	}

	encodedEvent, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if err := s.rabbitmq.Publish(ctx, "order_created_exchange", encodedEvent); err != nil {
		return err
	}

	s.logger.Debug(
		"OrderCreatedEvent published",
		"orderId", orderID,
		"userId", userID,
		"productsCount", len(products),
	)

	return nil
}
