package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	exchangeName string = "stock_update_failed_exchange"
	queueName    string = "order_stock_update_failed_queue"
	consumerName string = "order_stock_update_failed_consumer"

	orderStatusCancelled string = "CANCELLED"
)

type StockUpdateFailedConsumerRepository interface {
	GetOrderById(ctx context.Context, id pgtype.UUID) (repository.GetOrderByIdRow, error)
	GetOrderStatusByName(ctx context.Context, name string) (repository.GetOrderStatusByNameRow, error)
	UpdateOrderStatus(ctx context.Context, arg repository.UpdateOrderStatusParams) (repository.Order, error)
}

type MessageSubscriber interface {
	Subscribe(ctx context.Context, exchangeName, queueName, consumerName string, handler func(ctx context.Context, d amqp091.Delivery)) error
}

type StockUpdateFailedConsumer struct {
	logger                 logs.Logger
	repo                   StockUpdateFailedConsumerRepository
	subscriber             MessageSubscriber
	cancelledOrderStatusID pgtype.UUID
}

func NewStockUpdateFailedConsumer(logger logs.Logger, repo StockUpdateFailedConsumerRepository, subscriber MessageSubscriber) *StockUpdateFailedConsumer {
	return &StockUpdateFailedConsumer{
		logger:     logger,
		repo:       repo,
		subscriber: subscriber,
	}
}

func (sufc *StockUpdateFailedConsumer) initStatus(ctx context.Context) error {
	if sufc.cancelledOrderStatusID.Valid {
		return nil
	}
	status, err := sufc.repo.GetOrderStatusByName(ctx, orderStatusCancelled)
	if err != nil {
		return fmt.Errorf("could not fetch CANCELLED order status ID: %w", err)
	}
	sufc.cancelledOrderStatusID = status.ID
	sufc.logger.Debug("CANCELLED order status ID initialized", "id", sufc.cancelledOrderStatusID.String())
	return nil
}

func (sufc *StockUpdateFailedConsumer) Start(ctx context.Context) error {
	if err := sufc.initStatus(ctx); err != nil {
		return err
	}
	return sufc.subscriber.Subscribe(ctx, exchangeName, queueName, consumerName, sufc.handleStockUpdateFailedEvent)
}

func (sufc *StockUpdateFailedConsumer) handleStockUpdateFailedEvent(ctx context.Context, d amqp091.Delivery) {
	event, err := sufc.unmarshalEvent(d.Body)
	if err != nil {
		sufc.logger.Error("failed to unmarshal StockUpdateFailedEvent", "error", err)
		d.Nack(false, false)
		return
	}

	sufc.logger.Debug("received StockUpdateFailedEvent", "orderId", event.OrderID)

	orderUUID, err := parseOrderIDToUUID(event.OrderID)
	if err != nil {
		sufc.logger.Error("failed to parse order ID", "error", err, "orderId", event.OrderID)
		d.Nack(false, false)
		return
	}

	order, err := sufc.repo.GetOrderById(ctx, orderUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			sufc.logger.Error("order not found for cancellation", "orderId", event.OrderID)
			d.Ack(false)
			return
		}
		sufc.logger.Error("failed to get order for status check", "error", err, "orderId", event.OrderID)
		d.Nack(false, true)
		return
	}

	if order.Status == sufc.cancelledOrderStatusID {
		sufc.logger.Info("order is already cancelled, skipping", "orderId", event.OrderID)
		d.Ack(false)
		return
	}

	if err := sufc.cancelOrder(ctx, orderUUID); err != nil {
		sufc.logger.Error("failed to cancel order", "error", err, "orderId", event.OrderID)
		d.Nack(false, true)
	} else {
		sufc.logger.Info("order cancelled due to stock update failure", "orderId", event.OrderID)
		d.Ack(false)
	}
}

func (sufc *StockUpdateFailedConsumer) unmarshalEvent(body []byte) (*events.StockUpdateFailedEvent, error) {
	var event events.StockUpdateFailedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (sufc *StockUpdateFailedConsumer) cancelOrder(ctx context.Context, orderUUID pgtype.UUID) error {
	_, err := sufc.repo.UpdateOrderStatus(ctx, repository.UpdateOrderStatusParams{
		ID:     orderUUID,
		Status: sufc.cancelledOrderStatusID,
	})
	return err
}

func parseOrderIDToUUID(orderID string) (pgtype.UUID, error) {
	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid order ID format: %w", err)
	}
	return orderUUID, nil
}
