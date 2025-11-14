package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
)

type OutboxEventMessageRelayerRepository struct {
	*repository.Queries
}

func NewOutboxEventMessageRelayerRepository(db repository.DBTX) *OutboxEventMessageRelayerRepository {
	return &OutboxEventMessageRelayerRepository{
		Queries: repository.New(db),
	}
}

func (r *OutboxEventMessageRelayerRepository) GetUnpublishedOutboxEvents(ctx context.Context, limit int32) ([]events.OutboxEvent, error) {
	outboxEvents, err := r.Queries.GetUnpublishedOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}

	var result []events.OutboxEvent
	for _, oe := range outboxEvents {
		result = append(result, events.OutboxEvent{
			ID:          oe.ID.String(),
			AggregateID: oe.AggregateID.String(),
			EventName:   oe.EventName,
			Payload:     oe.Payload,
			Status:      oe.Status,
		})
	}

	return result, nil
}

func (r *OutboxEventMessageRelayerRepository) UpdateOutboxEventStatus(ctx context.Context, eventID string) error {
	eventUUID, err := parseIDStringToUUID(eventID)
	if err != nil {
		return err
	}
	return r.Queries.UpdateOutboxEventStatus(ctx, eventUUID)
}

func parseIDStringToUUID(id string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return pgtype.UUID{}, err
	} else {
		return uuid, nil
	}
}
