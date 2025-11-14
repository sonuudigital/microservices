package worker

import (
	"context"
	"time"

	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
)

type OutboxEventRepository interface {
	GetUnpublishedOutboxEvents(ctx context.Context, limit int32) ([]events.OutboxEvent, error)
	UpdateOutboxEventStatus(ctx context.Context, eventID string) error
}

type Publisher interface {
	Publish(ctx context.Context, exchange string, body []byte) error
}

type OutboxEventMessageRelayer struct {
	logger       logs.Logger
	publisher    Publisher
	repo         OutboxEventRepository
	pollInterval time.Duration
	batchSize    int32
}

func NewOutboxEventMessageRelayer(
	logger logs.Logger,
	publisher Publisher,
	repo OutboxEventRepository,
	pollInterval time.Duration,
	batchSize int32,
) *OutboxEventMessageRelayer {
	return &OutboxEventMessageRelayer{
		logger:       logger,
		publisher:    publisher,
		repo:         repo,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}
}

func (oemr *OutboxEventMessageRelayer) Start(ctx context.Context) {
	oemr.logger.Info("starting outbox event message relayer worker")
	ticker := time.NewTicker(oemr.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := oemr.processEvents(ctx); err != nil {
				oemr.logger.Error("error processing outbox events", "error", err)
			}
		case <-ctx.Done():
			oemr.logger.Info("stopping outbox event message relayer worker")
			return
		}
	}
}

func (oemr *OutboxEventMessageRelayer) processEvents(ctx context.Context) error {
	events, err := oemr.repo.GetUnpublishedOutboxEvents(ctx, oemr.batchSize)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := oemr.publisher.Publish(ctx, event.EventName, event.Payload); err != nil {
			oemr.logger.Error("failed to publish outbox event", "eventID", event.ID, "error", err)
			continue
		}

		if err := oemr.repo.UpdateOutboxEventStatus(ctx, event.ID); err != nil {
			oemr.logger.Error("failed to update outbox event status", "eventID", event.ID, "error", err)
			continue
		}

		oemr.logger.Info("successfully relayed outbox event", "eventID", event.ID)
	}

	return nil
}
