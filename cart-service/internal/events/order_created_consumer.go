package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

type OrderCreatedConsumer struct {
	logger   logs.Logger
	rabbitmq *rabbitmq.RabbitMQ
	querier  repository.Querier
}

func NewOrderCreatedConsumer(logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, querier repository.Querier) *OrderCreatedConsumer {
	return &OrderCreatedConsumer{
		logger:   logger,
		rabbitmq: rabbitmq,
		querier:  querier,
	}
}

func (occ *OrderCreatedConsumer) Start(ctx context.Context) error {
	return occ.rabbitmq.Subscribe(ctx, "order_created_exchange", "cart_queue", "cart_order_created_consumer", occ.handleOrderCreatedEvent)
}

func (occ *OrderCreatedConsumer) handleOrderCreatedEvent(d amqp091.Delivery) {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

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

	occ.logger.Info("cleared cart after order creation", "userId", orderCreatedEvent.UserID, "cartId", userCart.ID.String)
	d.Ack(false)
}
