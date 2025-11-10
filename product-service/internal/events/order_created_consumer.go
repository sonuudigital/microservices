package events

import (
	"context"
	"encoding/json"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
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
	return occ.rabbitmq.Subscribe(ctx, "order_created_exchange", "product_queue", "product_order_created_consumer", occ.handleOrderCreatedEvent)
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

	orderProductItems := make([]events.OrderItem, len(orderCreatedEvent.Products))
	for i, p := range orderCreatedEvent.Products {
		orderProductItems[i] = events.OrderItem{
			ProductID: p.ProductID,
			Quantity:  p.Quantity,
		}
	}

	encodedOrderProductItems, err := json.Marshal(orderProductItems)
	if err != nil {
		occ.logger.Error("failed to marshal order product items", "error", err)
		d.Nack(false, false)
		return
	}

	if err := occ.querier.UpdateStockBatch(ctx, encodedOrderProductItems); err != nil {
		occ.logger.Error("failed to update stock for products", "error", err)
		d.Nack(false, true)
		return
	}

	occ.logger.Info(
		"successfully updated stock for products in order",
		"orderId", orderCreatedEvent.OrderID,
		"userId", orderCreatedEvent.UserID,
		"products", orderCreatedEvent.Products,
	)

	d.Ack(false)
}
