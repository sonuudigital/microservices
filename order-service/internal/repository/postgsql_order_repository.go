package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/shared/events"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	orderCreatedEventName = "order_created_exchange"
)

type PostgreSQLOrderRepository struct {
	*Queries
	db *pgxpool.Pool
}

func NewPostgreSQLOrderRepository(db *pgxpool.Pool) *PostgreSQLOrderRepository {
	return &PostgreSQLOrderRepository{
		db:      db,
		Queries: New(db),
	}
}

func (s *PostgreSQLOrderRepository) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := New(tx)
	err = fn(q)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *PostgreSQLOrderRepository) CreateOrder(ctx context.Context, userID string, totalAmount float64, products []*cartv1.CartProduct) (*orderv1.Order, error) {
	var createdOrder *orderv1.Order
	err := s.execTx(ctx, func(q *Queries) error {
		userUUID, pgTotalAmount, err := validateAndTransformCreateOrderParams(userID, totalAmount)
		if err != nil {
			return err
		}

		dbOrder, err := q.CreateOrder(ctx, CreateOrderParams{
			UserID:      userUUID,
			TotalAmount: pgTotalAmount,
		})
		if err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		encodedEvent, err := generateOrderCreatedEventPayload(dbOrder.ID.String(), dbOrder.UserID.String(), products)

		err = q.CreateOutboxEvent(ctx, CreateOutboxEventParams{
			AggregateID: dbOrder.ID,
			EventName:   orderCreatedEventName,
			Payload:     encodedEvent,
		})
		if err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}

		orderStatus, err := q.GetOrderStatusByName(ctx, "CREATED")
		if err != nil {
			return fmt.Errorf("failed to get CREATED status: %w", err)
		}

		grpcOrderResponse, err := mapRepositoryToGRPC(&dbOrder, orderStatus.Name)
		if err != nil {
			return fmt.Errorf("failed to map order to gRPC model: %w", err)
		}

		createdOrder = grpcOrderResponse

		return nil
	})
	if err != nil {
		return nil, err
	} else {
		return createdOrder, nil
	}
}

func validateAndTransformCreateOrderParams(userID string, totalAmount float64) (pgtype.UUID, pgtype.Numeric, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return pgtype.UUID{}, pgtype.Numeric{}, fmt.Errorf("invalid user ID: %w", err)
	}

	var pgTotalAmount pgtype.Numeric
	if err := pgTotalAmount.Scan(fmt.Sprintf("%.2f", totalAmount)); err != nil {
		return pgtype.UUID{}, pgtype.Numeric{}, fmt.Errorf("failed to parse total amount: %w", err)
	}

	return userUUID, pgTotalAmount, nil
}

func generateOrderCreatedEventPayload(orderID, userID string, products []*cartv1.CartProduct) ([]byte, error) {
	eventProducts := make([]events.OrderItem, len(products))
	for i, p := range products {
		eventProducts[i] = events.OrderItem{
			ProductID: p.ProductId,
			Quantity:  p.Quantity,
		}
	}

	orderCreatedEvent := events.OrderCreatedEvent{
		OrderID:  orderID,
		UserID:   userID,
		Products: eventProducts,
	}

	encodedEvent, err := json.Marshal(orderCreatedEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OrderCreatedEvent: %w", err)
	}

	return encodedEvent, nil
}

func mapRepositoryToGRPC(o *Order, statusName string) (*orderv1.Order, error) {
	totalAmount, err := o.TotalAmount.Float64Value()
	if err != nil {
		return nil, err
	}

	return &orderv1.Order{
		Id:          o.ID.String(),
		UserId:      o.UserID.String(),
		TotalAmount: totalAmount.Float64,
		Status:      statusName,
		CreatedAt:   timestamppb.New(o.CreatedAt.Time),
	}, nil
}
