package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
)

type OutboxEventRepository interface {
	GetUnpublishedOutboxEvents(ctx context.Context, limit int32) ([]repository.OutboxEvent, error)
	UpdateOutboxEventStatus(ctx context.Context, eventID pgtype.UUID) error
}

type RabbitMQPublisher interface {
	Publish(ctx context.Context, exchange string, body []byte) error
}

type MessageRelayer struct {
	logger       logs.Logger
	publisher    RabbitMQPublisher
	repo         OutboxEventRepository
	pollInterval time.Duration
	batchSize    int32
}

func New(
	logger logs.Logger,
	publisher RabbitMQPublisher,
	repo OutboxEventRepository,
	pollInterval time.Duration,
	batchSize int32,
) *MessageRelayer {
	return &MessageRelayer{
		logger:       logger,
		publisher:    publisher,
		repo:         repo,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}
}

func (mr *MessageRelayer) Start(ctx context.Context) {
	mr.logger.Info("starting message relayer worker")
	ticker := time.NewTicker(mr.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := mr.processEvents(ctx); err != nil {
				mr.logger.Error("error processing outbox events", "error", err)
			}
		case <-ctx.Done():
			mr.logger.Info("stopping message relayer worker")
			return
		}
	}
}

func (mr *MessageRelayer) processEvents(ctx context.Context) error {
	events, err := mr.repo.GetUnpublishedOutboxEvents(ctx, mr.batchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch unpublished outbox events: %w", err)
	}

	if len(events) > 0 {
		mr.logger.Debug("found events to publish", "count", len(events))
	}

	for _, event := range events {
		if err := mr.publisher.Publish(ctx, event.EventName, event.Payload); err != nil {
			mr.logger.Error("failed to publish event", "eventId", event.ID, "error", err)
			continue
		}

		mr.logger.Debug("successfully published event", "eventId", event.ID)
		if err := mr.repo.UpdateOutboxEventStatus(ctx, event.ID); err != nil {
			mr.logger.Error("CRITICAL: event published but failed to update status", "eventId", event.ID, "error", err)
			continue
		}

		mr.logger.Debug("updated outbox event status to PUBLISHED", "eventId", event.ID)
	}

	return nil
}
