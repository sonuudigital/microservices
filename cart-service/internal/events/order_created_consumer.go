package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

const (
	exchangeName        = "order_created_exchange"
	queueName           = "cart_queue"
	consumerName        = "cart_order_created_consumer"
	redisCartPrefix     = "cart:"
	redisContextTimeout = time.Second * 3
)

type OrderCreatedConsumerRepository interface {
	GetProcessedEventByAggregateIDAndEventName(ctx context.Context, arg repository.GetProcessedEventByAggregateIDAndEventNameParams) (repository.ProcessedEvent, error)
	DeleteCartAndCreateProcessedEvent(ctx context.Context, event *events.OrderCreatedEvent, eventName string) error
}

type OrderCreatedConsumer struct {
	logger      logs.Logger
	rabbitmq    *rabbitmq.RabbitMQ
	repo        OrderCreatedConsumerRepository
	redisClient *redis.Client
}

func NewOrderCreatedConsumer(logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, repo OrderCreatedConsumerRepository, redisClient *redis.Client) *OrderCreatedConsumer {
	return &OrderCreatedConsumer{
		logger:      logger,
		rabbitmq:    rabbitmq,
		repo:        repo,
		redisClient: redisClient,
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
	)

	err = occ.repo.DeleteCartAndCreateProcessedEvent(ctx, orderCreatedEvent, exchangeName)
	if err != nil {
		occ.logger.Error("failed to delete cart and create processed event", "error", err, "orderId", orderCreatedEvent.OrderID)
		d.Nack(false, true)
		return
	}

	go occ.deleteCartCache(orderCreatedEvent.UserID)

	occ.logger.Info("cleared cart after order creation", "userId", orderCreatedEvent.UserID, "orderId", orderCreatedEvent.OrderID)
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
	var aggregateUUID pgtype.UUID
	if err := aggregateUUID.Scan(aggregateID); err != nil {
		return false, err
	}

	processedEvent, err := occ.repo.GetProcessedEventByAggregateIDAndEventName(ctx, repository.GetProcessedEventByAggregateIDAndEventNameParams{
		AggregateID: aggregateUUID,
		EventName:   eventName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return processedEvent.ID.Valid, nil
}

func (occ *OrderCreatedConsumer) deleteCartCache(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), redisContextTimeout)
	defer cancel()

	cacheKey := redisCartPrefix + userID
	if err := occ.redisClient.Del(ctx, cacheKey).Err(); err != nil {
		occ.logger.Error("failed to delete cart cache", "userID", userID, "error", err)
	} else {
		occ.logger.Debug("cart cache invalidated", "userID", userID)
	}
}
