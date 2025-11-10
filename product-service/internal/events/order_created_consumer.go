package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

type OrderCreatedConsumer struct {
	logger      logs.Logger
	rabbitmq    *rabbitmq.RabbitMQ
	redisClient *redis.Client
	querier     repository.Querier
}

func NewOrderCreatedConsumer(logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, redisClient *redis.Client, querier repository.Querier) *OrderCreatedConsumer {
	return &OrderCreatedConsumer{
		logger:      logger,
		rabbitmq:    rabbitmq,
		redisClient: redisClient,
		querier:     querier,
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
		"productsCount", len(orderCreatedEvent.Products),
	)

	orderProductItems := make([]events.OrderItem, len(orderCreatedEvent.Products))
	productIDs := make([]string, len(orderCreatedEvent.Products))
	for i, p := range orderCreatedEvent.Products {
		orderProductItems[i] = events.OrderItem{
			ProductID: p.ProductID,
			Quantity:  p.Quantity,
		}
		productIDs[i] = p.ProductID
	}

	occ.logger.Debug("attempting to update stock for products", "products", orderProductItems, "productIDs", productIDs)

	encodedOrderProductItems, err := json.Marshal(orderProductItems)
	if err != nil {
		occ.logger.Error("failed to marshal order product items", "error", err)
		d.Nack(false, false)
		return
	}

	occ.logger.Debug("encoded order product items", "json", string(encodedOrderProductItems))

	rowsAffected, err := occ.querier.UpdateStockBatch(ctx, encodedOrderProductItems)
	if err != nil {
		occ.logger.Error("failed to update stock batch", "error", err, "json", string(encodedOrderProductItems))
		d.Nack(false, true)
		return
	}

	occ.logger.Debug("stock update completed", "rowsAffected", rowsAffected, "expectedRows", len(orderProductItems))

	if rowsAffected == 0 {
		occ.logger.Error(
			"no products were updated - products might not exist or have insufficient stock",
			"orderId", orderCreatedEvent.OrderID,
			"productIDs", productIDs,
			"products", orderProductItems,
		)
		d.Nack(false, false)
		return
	}

	expectedRows := int64(len(orderProductItems))
	if rowsAffected != expectedRows {
		occ.logger.Error(
			"stock update affected unexpected number of rows - some products might not exist or have insufficient stock",
			"expected", expectedRows,
			"actual", rowsAffected,
			"orderId", orderCreatedEvent.OrderID,
			"productIDs", productIDs,
			"products", orderProductItems,
			"json", string(encodedOrderProductItems),
		)
		//TODO: implement a compensation action to revert stock changes and cancel order
		d.Nack(false, false)
		return
	}

	occ.invalidateCacheForUpdatedProducts(ctx, orderProductItems)

	occ.logger.Info(
		"successfully updated stock for products in order",
		"orderId", orderCreatedEvent.OrderID,
		"productsCount", len(orderProductItems),
	)

	d.Ack(false)
}

func (occ *OrderCreatedConsumer) invalidateCacheForUpdatedProducts(ctx context.Context, products []events.OrderItem) {
	pipe := occ.redisClient.Pipeline()

	for _, product := range products {
		cacheKey := fmt.Sprintf("product:%s", product.ProductID)
		pipe.Del(ctx, cacheKey)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		occ.logger.Warn("failed to invalidate cache via pipeline", "error", err)
	} else {
		occ.logger.Debug("cache invalidated for products", "products", products)
	}
}
