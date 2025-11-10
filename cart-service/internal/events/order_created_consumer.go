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
	redisCartPrefix     = "cart:"
	redisContextTimeout = time.Second * 3
)

type OrderCreatedConsumer struct {
	logger      logs.Logger
	rabbitmq    *rabbitmq.RabbitMQ
	querier     repository.Querier
	redisClient *redis.Client
}

func NewOrderCreatedConsumer(logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, querier repository.Querier, redisClient *redis.Client) *OrderCreatedConsumer {
	return &OrderCreatedConsumer{
		logger:      logger,
		rabbitmq:    rabbitmq,
		querier:     querier,
		redisClient: redisClient,
	}
}

func (occ *OrderCreatedConsumer) Start(ctx context.Context) error {
	return occ.rabbitmq.Subscribe(ctx, "order_created_exchange", "cart_queue", "cart_order_created_consumer", occ.handleOrderCreatedEvent)
}

func (occ *OrderCreatedConsumer) handleOrderCreatedEvent(ctx context.Context, d amqp091.Delivery) {
	var orderCreatedEvent events.OrderCreatedEvent
	if err := json.Unmarshal(d.Body, &orderCreatedEvent); err != nil {
		occ.logger.Error("failed to unmarshal OrderCreatedEvent", "error", err)
		d.Nack(false, false)
		return
	}

	occ.logger.Debug(
		"received OrderCreatedEvent",
		"orderId", orderCreatedEvent.OrderID,
		"userId", orderCreatedEvent.UserID,
		"products", orderCreatedEvent.Products,
	)

	var userUUID pgtype.UUID
	if err := userUUID.Scan(orderCreatedEvent.UserID); err != nil {
		occ.logger.Error("failed to scan user ID", "error", err)
		d.Nack(false, false)
		return
	}

	userCart, err := occ.querier.GetCartByUserID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			occ.logger.Info("no cart found for user, nothing to clear", "userId", orderCreatedEvent.UserID)
			d.Ack(false)
			return
		} else {
			occ.logger.Error("failed to get cart by user ID", "error", err)
			d.Nack(false, false)
			return
		}
	}

	err = occ.querier.DeleteCartByUserID(ctx, userUUID)
	if err != nil {
		occ.logger.Error("failed to delete cart after order creation", "error", err)
		d.Nack(false, true)
		return
	}

	go occ.deleteCartCache(orderCreatedEvent.UserID)

	occ.logger.Info("cleared cart after order creation", "userId", orderCreatedEvent.UserID, "cartId", userCart.ID.String())
	d.Ack(false)
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
