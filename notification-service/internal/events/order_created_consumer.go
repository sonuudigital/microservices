package events

import (
	"context"
	"encoding/json"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	exchangeName string = "order_created_exchange"
	queueName    string = "notification_queue"
	consumerName string = "notification_order_created_consumer"
)

type Sender interface {
	Send(data any) error
}

type MessageSubscriber interface {
	Subscribe(ctx context.Context, exchangeName, queueName, consumerName string, handler func(ctx context.Context, d amqp091.Delivery)) error
}

type OrderCreatedConsumer struct {
	logger     logs.Logger
	sender     Sender
	subscriber MessageSubscriber
}

func NewOrderCreatedConsumer(logger logs.Logger, sender Sender, subscriber MessageSubscriber) *OrderCreatedConsumer {
	return &OrderCreatedConsumer{
		logger:     logger,
		sender:     sender,
		subscriber: subscriber,
	}
}

func (occ *OrderCreatedConsumer) Start(ctx context.Context) error {
	return occ.subscriber.Subscribe(ctx, exchangeName, queueName, consumerName, occ.handleOrderCreatedEvent)
}

func (occ *OrderCreatedConsumer) handleOrderCreatedEvent(ctx context.Context, d amqp091.Delivery) {
	var orderCreatedEvent events.OrderCreatedEvent
	if err := json.Unmarshal(d.Body, &orderCreatedEvent); err != nil {
		occ.logger.Error("failed to unmarshal OrderCreatedEvent", "error", err)
		d.Nack(false, false)
		return
	}

	if err := occ.sender.Send(orderCreatedEvent); err != nil {
		occ.logger.Error("failed to send notification for OrderCreatedEvent", "error", err)
		d.Nack(false, true)
		return
	}

	occ.logger.Info(
		"successfully processed OrderCreatedEvent",
		"orderId", orderCreatedEvent.OrderID,
		"userId", orderCreatedEvent.UserID,
		"userEmail", orderCreatedEvent.UserEmail,
	)
	d.Ack(false)
}
