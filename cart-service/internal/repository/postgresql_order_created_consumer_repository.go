package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonuudigital/microservices/shared/events"
)

type PostgreSQLOrderCreatedConsumerRepository struct {
	*Queries
	db *pgxpool.Pool
}

func NewPostgreSQLOrderCreatedConsumerRepository(db *pgxpool.Pool) *PostgreSQLOrderCreatedConsumerRepository {
	return &PostgreSQLOrderCreatedConsumerRepository{
		Queries: New(db),
		db:      db,
	}
}

func (r *PostgreSQLOrderCreatedConsumerRepository) DeleteCartAndCreateProcessedEvent(ctx context.Context, event *events.OrderCreatedEvent, eventName string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.Queries.WithTx(tx)

	var userUUID pgtype.UUID
	if err := userUUID.Scan(event.UserID); err != nil {
		return fmt.Errorf("failed to scan user ID: %w", err)
	}

	_ = qtx.DeleteCartByUserID(ctx, userUUID)

	var aggregateUUID pgtype.UUID
	if err := aggregateUUID.Scan(event.OrderID); err != nil {
		return fmt.Errorf("failed to scan order ID: %w", err)
	}

	err = qtx.CreateProcessedEvent(ctx, CreateProcessedEventParams{
		AggregateID: aggregateUUID,
		EventName:   eventName,
	})
	if err != nil {
		return fmt.Errorf("failed to create processed event: %w", err)
	}

	return tx.Commit(ctx)
}
