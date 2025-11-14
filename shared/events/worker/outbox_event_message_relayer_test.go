package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOutboxEventRepository struct {
	mock.Mock
}

func (m *MockOutboxEventRepository) GetUnpublishedOutboxEvents(ctx context.Context, limit int32) ([]events.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]events.OutboxEvent), args.Error(1)
}

func (m *MockOutboxEventRepository) UpdateOutboxEventStatus(ctx context.Context, eventID string) error {
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
	testEvent := events.OutboxEvent{
		ID:        "test-event-id",
		EventName: "order_created_exchange",
		Payload:   []byte(`{"order_id":"test-order"}`),
	}

	t.Run("SuccessPath", func(t *testing.T) {

		mockRepo := new(MockOutboxEventRepository)
		mockPublisher := new(MockRabbitMQPublisher)
		relayer := NewOutboxEventMessageRelayer(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		events := []events.OutboxEvent{testEvent}
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
		relayer := NewOutboxEventMessageRelayer(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		events := []events.OutboxEvent{testEvent}
		mockRepo.On("GetUnpublishedOutboxEvents", mock.Anything, int32(10)).Return(events, nil).Once()
		mockPublisher.On("Publish", mock.Anything, testEvent.EventName, testEvent.Payload).Return(errors.New("publish error")).Once()

		err := relayer.processEvents(context.Background())

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("UpdateStatusError", func(t *testing.T) {
		mockRepo := new(MockOutboxEventRepository)
		mockPublisher := new(MockRabbitMQPublisher)
		relayer := NewOutboxEventMessageRelayer(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		events := []events.OutboxEvent{testEvent}
		mockRepo.On("GetUnpublishedOutboxEvents", mock.Anything, int32(10)).Return(events, nil).Once()
		mockPublisher.On("Publish", mock.Anything, testEvent.EventName, testEvent.Payload).Return(nil).Once()
		mockRepo.On("UpdateOutboxEventStatus", mock.Anything, testEvent.ID).Return(errors.New("update status error")).Once()

		err := relayer.processEvents(context.Background())

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("NoEvents", func(t *testing.T) {
		mockRepo := new(MockOutboxEventRepository)
		mockPublisher := new(MockRabbitMQPublisher)
		relayer := NewOutboxEventMessageRelayer(logs.NewSlogLogger(), mockPublisher, mockRepo, 0, 10)

		mockRepo.On("GetUnpublishedOutboxEvents", mock.Anything, int32(10)).Return([]events.OutboxEvent{}, nil).Once()

		err := relayer.processEvents(context.Background())

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})
}
