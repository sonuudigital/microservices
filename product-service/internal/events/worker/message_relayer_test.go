package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOutboxEventRepository struct {
	mock.Mock
}

func (m *MockOutboxEventRepository) GetUnpublishedOutboxEvents(ctx context.Context, limit int32) ([]repository.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]repository.OutboxEvent), args.Error(1)
}

func (m *MockOutboxEventRepository) UpdateOutboxEventStatus(ctx context.Context, eventID pgtype.UUID) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

type MockRabbitMQPublisher struct {
	mock.Mock
}

func (m *MockRabbitMQPublisher) Publish(ctx context.Context, exchange string, body []byte) error {
	args := m.Called(ctx, exchange, body)
	return args.Error(0)
}

func TestProcessEvents(t *testing.T) {
	eventID := pgtype.UUID{Bytes: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, Valid: true}
	testEvent := repository.OutboxEvent{
		ID:        eventID,
		EventName: "order_created_exchange",
		Payload:   []byte(`{"order_id":"test-order"}`),
	}

	t.Run("SuccessPath", func(t *testing.T) {

		mockRepo := new(MockOutboxEventRepository)
		mockPublisher := new(MockRabbitMQPublisher)
		relayer := New(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		events := []repository.OutboxEvent{testEvent}
		mockRepo.On("GetUnpublishedOutboxEvents", mock.Anything, int32(10)).Return(events, nil).Once()
		mockPublisher.On("Publish", mock.Anything, testEvent.EventName, testEvent.Payload).Return(nil).Once()
		mockRepo.On("UpdateOutboxEventStatus", mock.Anything, testEvent.ID).Return(nil).Once()

		err := relayer.processEvents(context.Background())

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("PublisherError", func(t *testing.T) {

		mockRepo := new(MockOutboxEventRepository)
		mockPublisher := new(MockRabbitMQPublisher)
		relayer := New(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		events := []repository.OutboxEvent{testEvent}
		publishErr := errors.New("rabbitmq is down")

		mockRepo.On("GetUnpublishedOutboxEvents", mock.Anything, int32(10)).Return(events, nil).Once()
		mockPublisher.On("Publish", mock.Anything, testEvent.EventName, testEvent.Payload).Return(publishErr).Once()

		err := relayer.processEvents(context.Background())

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "UpdateOutboxEventStatus", mock.Anything, mock.Anything)
	})

	t.Run("UpdateStatusError", func(t *testing.T) {

		mockRepo := new(MockOutboxEventRepository)
		mockPublisher := new(MockRabbitMQPublisher)
		relayer := New(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		events := []repository.OutboxEvent{testEvent}
		updateErr := errors.New("db connection lost")

		mockRepo.On("GetUnpublishedOutboxEvents", mock.Anything, int32(10)).Return(events, nil).Once()
		mockPublisher.On("Publish", mock.Anything, testEvent.EventName, testEvent.Payload).Return(nil).Once()
		mockRepo.On("UpdateOutboxEventStatus", mock.Anything, testEvent.ID).Return(updateErr).Once()

		err := relayer.processEvents(context.Background())

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("NoEvents", func(t *testing.T) {

		mockRepo := new(MockOutboxEventRepository)
		mockPublisher := new(MockRabbitMQPublisher)
		relayer := New(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		events := []repository.OutboxEvent{}
		mockRepo.On("GetUnpublishedOutboxEvents", mock.Anything, int32(10)).Return(events, nil).Once()

		err := relayer.processEvents(context.Background())

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
		mockRepo.AssertNotCalled(t, "UpdateOutboxEventStatus", mock.Anything, mock.Anything)
	})
}
