package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

const (
	exchangeName               string = "order_created_exchange"
	queueName                  string = "product_queue"
	consumerName               string = "product_order_created_consumer"
	stockUpdateFailedEventName string = "stock_update_failed_exchange"
)

type OrderCreatedConsumerRepository interface {
	GetProcessedEventByAggregateIDAndEventName(ctx context.Context, arg repository.GetProcessedEventByAggregateIDAndEventNameParams) (repository.ProcessedEvent, error)
	UpdateStockBatch(ctx context.Context, event *events.OrderCreatedEvent, createOutboxEventOnFailure bool, outboxEventName string, outboxEventPayload []byte) (int64, error)
}

type OrderCreatedConsumer struct {
	logger      logs.Logger
	rabbitmq    *rabbitmq.RabbitMQ
	redisClient *redis.Client
	repo        OrderCreatedConsumerRepository
}

func NewOrderCreatedConsumer(logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, redisClient *redis.Client, repo OrderCreatedConsumerRepository) *OrderCreatedConsumer {
	return &OrderCreatedConsumer{
		logger:      logger,
		rabbitmq:    rabbitmq,
		redisClient: redisClient,
		repo:        repo,
	}
}

func (occ *OrderCreatedConsumer) Start(ctx context.Context) error {
	return occ.rabbitmq.Subscribe(ctx, exchangeName, queueName, consumerName, occ.handleOrderCreatedEvent)
}

func (occ *OrderCreatedConsumer) handleOrderCreatedEvent(ctx context.Context, d amqp091.Delivery) {
	orderCreatedEvent, err := occ.unmarshalEvent(d.Body)
	if err != nil {
		occ.logger.Error("failed to unmarshal OrderCreatedEvent", "error", err)
		d.Nack(false, false)
		return
	}

	if occ.shouldSkipProcessing(ctx, orderCreatedEvent.OrderID, d) {
		return
	}

	occ.logger.Debug(
		"received OrderCreatedEvent",
		"orderId", orderCreatedEvent.OrderID,
		"userId", orderCreatedEvent.UserID,
		"productsCount", len(orderCreatedEvent.Products),
	)

	stockUpdateFailedEventData, err := json.Marshal(events.StockUpdateFailedEvent{
		OrderID:  orderCreatedEvent.OrderID,
		Products: orderCreatedEvent.Products,
	})
	if err != nil {
		occ.logger.Error("failed to marshal StockUpdateFailedEvent", "error", err, "orderId", orderCreatedEvent.OrderID)
		d.Nack(false, false)
		return
	}

	rowsAffected, err := occ.repo.UpdateStockBatch(ctx, orderCreatedEvent, true, stockUpdateFailedEventName, stockUpdateFailedEventData)
	if err != nil {
		occ.logger.Error("failed to update stock batch transactionally", "error", err, "orderId", orderCreatedEvent.OrderID)
		d.Nack(false, true)
		return
	}

	if !occ.validateStockUpdate(orderCreatedEvent, rowsAffected, d) {
		return
	}

	go occ.invalidateCacheForUpdatedProducts(orderCreatedEvent.Products)

	occ.logger.Info(
		"successfully updated stock for products in order",
		"orderId", orderCreatedEvent.OrderID,
		"productsCount", len(orderCreatedEvent.Products),
	)

	d.Ack(false)
}

func (occ *OrderCreatedConsumer) unmarshalEvent(body []byte) (*events.OrderCreatedEvent, error) {
	var orderCreatedEvent events.OrderCreatedEvent
	if err := json.Unmarshal(body, &orderCreatedEvent); err != nil {
		return nil, err
	}
	return &orderCreatedEvent, nil
}

func (occ *OrderCreatedConsumer) shouldSkipProcessing(ctx context.Context, orderID string, d amqp091.Delivery) bool {
	processed, err := occ.isEventProcessed(ctx, orderID, exchangeName)
	if err != nil {
		occ.logger.Error("failed to check if event is already processed", "error", err, "orderId", orderID)
		d.Nack(false, true)
		return true
	}
	if processed {
		occ.logger.Info("event already processed, acknowledging without reprocessing", "orderId", orderID)
		d.Ack(false)
		return true
	}
	return false
}

func (occ *OrderCreatedConsumer) isEventProcessed(ctx context.Context, aggregateID string, eventName string) (bool, error) {
	var agreggateUUID pgtype.UUID
	if err := agreggateUUID.Scan(aggregateID); err != nil {
		return false, err
	}

	processedEvent, err := occ.repo.GetProcessedEventByAggregateIDAndEventName(ctx, repository.GetProcessedEventByAggregateIDAndEventNameParams{
		AggregateID: agreggateUUID,
		EventName:   eventName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		} else {
			return false, err
		}
	}

	return processedEvent.ID.Valid, nil
}

func (occ *OrderCreatedConsumer) validateStockUpdate(event *events.OrderCreatedEvent, rowsAffected int64, d amqp091.Delivery) bool {
	expectedRows := int64(len(event.Products))
	if rowsAffected != expectedRows {
		occ.logger.Error(
			"stock update affected unexpected number of rows - some products might not exist or have insufficient stock",
			"expected", expectedRows,
			"actual", rowsAffected,
			"orderId", event.OrderID,
			"products", event.Products,
		)
		d.Nack(false, false)
		return false
	}

	return true
}

func (occ *OrderCreatedConsumer) invalidateCacheForUpdatedProducts(products []events.OrderItem) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pipe := occ.redisClient.Pipeline()

	productIDs := make([]string, len(products))
	for i, product := range products {
		productIDs[i] = product.ProductID
		cacheKey := fmt.Sprintf("product:%s", product.ProductID)
		pipe.Del(ctx, cacheKey)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		occ.logger.Warn("failed to invalidate cache for products", "error", err, "productIDs", productIDs)
	} else {
		occ.logger.Debug("cache invalidated for products", "productIDs", productIDs)
	}
}
