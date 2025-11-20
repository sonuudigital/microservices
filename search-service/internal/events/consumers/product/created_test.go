package product

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	uint8ArrayType = "[]uint8"
)

type MockSubscriber struct {
	mock.Mock
}

func (m *MockSubscriber) Subscribe(ctx context.Context, opts rabbitmq.SubscribeOptions) error {
	args := m.Called(ctx, opts)
	return args.Error(0)
}

type MockIndexer struct {
	mock.Mock
}

func (m *MockIndexer) Index(ctx context.Context, indexName string, documentID string, body []byte) (*opensearchapi.Response, error) {
	args := m.Called(ctx, indexName, documentID, body)

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	if args.Get(0) != nil {
		behavior := args.String(0)
		switch behavior {
		case "error_response":
			return createMockResponse(true), nil
		case "success":
			return createMockResponse(false), nil
		}
	}

	return createMockResponse(false), nil
}

func createMockResponse(isError bool) *opensearchapi.Response {
	statusCode := 201
	if isError {
		statusCode = 400
	}

	return &opensearchapi.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("{}")),
	}
}

func TestNewProductCreatedEventConsumer(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockSubscriber := new(MockSubscriber)
	mockIndexer := new(MockIndexer)
	index := "products"

	consumer := NewProductCreatedEventConsumer(logger, mockSubscriber, mockIndexer, index)

	assert.NotNil(t, consumer)
	assert.Equal(t, logger, consumer.logger)
	assert.Equal(t, mockSubscriber, consumer.subscriber)
	assert.Equal(t, mockIndexer, consumer.indexser)
	assert.Equal(t, index, consumer.opensearchIndex)
}

func TestProductCreatedEventConsumerStart(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockSubscriber := new(MockSubscriber)
	mockIndexer := new(MockIndexer)
	index := "products"

	consumer := NewProductCreatedEventConsumer(logger, mockSubscriber, mockIndexer, index)

	expectedOpts := rabbitmq.SubscribeOptions{
		Exchange:     events.ProductExchangeName,
		ExchangeType: rabbitmq.ExchangeTopic,
		QueueName:    "search_product_events_queue",
		BindingKey:   events.ProductWaildCardRoutingKey,
		Handler:      consumer.handleProductCreatedEvent,
	}

	t.Run("Success", func(t *testing.T) {
		mockSubscriber.On("Subscribe", mock.Anything, mock.MatchedBy(func(opts rabbitmq.SubscribeOptions) bool {
			return opts.Exchange == expectedOpts.Exchange &&
				opts.ExchangeType == expectedOpts.ExchangeType &&
				opts.QueueName == expectedOpts.QueueName &&
				opts.BindingKey == expectedOpts.BindingKey &&
				opts.Handler != nil
		})).Return(nil).Once()

		err := consumer.Start(context.Background())

		assert.NoError(t, err)
		mockSubscriber.AssertExpectations(t)
	})

	t.Run("SubscribeError", func(t *testing.T) {
		expectedErr := errors.New("subscribe failed")
		mockSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(expectedErr).Once()

		err := consumer.Start(context.Background())

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockSubscriber.AssertExpectations(t)
	})
}

func TestProductCreatedEventConsumerHandleProductCreatedEvent(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockSubscriber := new(MockSubscriber)
	mockIndexer := new(MockIndexer)
	index := "products"

	consumer := NewProductCreatedEventConsumer(logger, mockSubscriber, mockIndexer, index)

	testProduct := events.Product{
		ID:          "product-123",
		CategoryID:  "category-456",
		Name:        "Test Product",
		Description: "A test product",
		Price:       "99.99",
	}

	t.Run("Success", func(t *testing.T) {
		productJSON := `{
			"id": "product-123",
			"categoryId": "category-456",
			"name": "Test Product",
			"description": "A test product",
			"price": "99.99",
			"stockQuantity": 10
		}`

		mockIndexer.On("Index", mock.Anything, index, testProduct.ID, mock.AnythingOfType(uint8ArrayType)).Return("success", nil).Once()

		delivery := amqp091.Delivery{Body: []byte(productJSON)}
		consumer.handleProductCreatedEvent(context.Background(), delivery)

		mockIndexer.AssertExpectations(t)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		invalidJSON := `{invalid json}`

		delivery := amqp091.Delivery{Body: []byte(invalidJSON)}
		consumer.handleProductCreatedEvent(context.Background(), delivery)

		mockIndexer.AssertNotCalled(t, "Index")
	})

	t.Run("IndexerError", func(t *testing.T) {
		productJSON := `{
			"id": "product-123",
			"categoryId": "category-456",
			"name": "Test Product",
			"description": "A test product",
			"price": "99.99",
			"stockQuantity": 10
		}`

		indexErr := errors.New("indexing failed")

		mockIndexer.On("Index", mock.Anything, index, testProduct.ID, mock.AnythingOfType(uint8ArrayType)).Return(nil, indexErr).Once()

		delivery := amqp091.Delivery{Body: []byte(productJSON)}
		consumer.handleProductCreatedEvent(context.Background(), delivery)

		mockIndexer.AssertExpectations(t)
	})

	t.Run("OpenSearchResponseError", func(t *testing.T) {
		productJSON := `{
			"id": "product-123",
			"categoryId": "category-456",
			"name": "Test Product",
			"description": "A test product",
			"price": "99.99",
			"stockQuantity": 10
		}`

		mockIndexer.On("Index", mock.Anything, index, testProduct.ID, mock.AnythingOfType(uint8ArrayType)).Return("error_response", nil).Once()

		delivery := amqp091.Delivery{Body: []byte(productJSON)}
		consumer.handleProductCreatedEvent(context.Background(), delivery)

		mockIndexer.AssertExpectations(t)
	})

	t.Run("MarshalErrorCannotOccur", func(t *testing.T) {
		productJSON := `{
			"id": "product-123",
			"categoryId": "category-456",
			"name": "Test Product",
			"description": "A test product",
			"price": "99.99",
			"stockQuantity": 10
		}`

		mockIndexer.On("Index", mock.Anything, index, "product-123", mock.AnythingOfType(uint8ArrayType)).Return("success", nil).Once()

		delivery := amqp091.Delivery{Body: []byte(productJSON)}
		consumer.handleProductCreatedEvent(context.Background(), delivery)

		mockIndexer.AssertExpectations(t)
	})
}
