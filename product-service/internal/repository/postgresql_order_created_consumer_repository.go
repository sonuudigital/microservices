package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonuudigital/microservices/shared/events"
)

const (
	eventName = "order_created_exchange"
)

type PostgreSQLOrderCreatedConsumerRepository struct {
	*Queries
	db *pgxpool.Pool
}

func NewPostgreSQLOrderCreatedConsumerRepository(db *pgxpool.Pool) *PostgreSQLOrderCreatedConsumerRepository {
	return &PostgreSQLOrderCreatedConsumerRepository{
		db:      db,
		Queries: New(db),
	}
}

func (r *PostgreSQLOrderCreatedConsumerRepository) GetProcessedEventByAggregateIDAndEventName(ctx context.Context, arg GetProcessedEventByAggregateIDAndEventNameParams) (ProcessedEvent, error) {
	return r.Queries.GetProcessedEventByAggregateIDAndEventName(ctx, arg)
}

func (r *PostgreSQLOrderCreatedConsumerRepository) UpdateStockBatch(ctx context.Context, event *events.OrderCreatedEvent) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	q := r.WithTx(tx)

	encodedOrderItems, err := marshalOrderItems(event)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := q.UpdateStockBatch(ctx, encodedOrderItems)
	if err != nil {
		return 0, err
	}

	orderUUID, err := parseOrderID(event.OrderID)
	if err != nil {
		return 0, err
	}
	if err = q.CreateProcessedEvent(ctx, CreateProcessedEventParams{
		AggregateID: orderUUID,
		EventName:   eventName,
	}); err != nil {
		return 0, err
	}

	return rowsAffected, tx.Commit(ctx)
}

func marshalOrderItems(event *events.OrderCreatedEvent) ([]byte, error) {
	orderProductItems := make([]events.OrderItem, len(event.Products))
	for i, p := range event.Products {
		orderProductItems[i] = events.OrderItem{
			ProductID: p.ProductID,
			Quantity:  p.Quantity,
		}
	}
	return json.Marshal(orderProductItems)
}

func parseOrderID(orderID string) (pgtype.UUID, error) {
	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		return pgtype.UUID{}, err
	}
	return orderUUID, nil
}
