package product

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

type Subscriber interface {
	Subscribe(ctx context.Context, opts rabbitmq.SubscribeOptions) error
}

type DocumentStore interface {
	Index(ctx context.Context, indexName string, documentID string, body []byte) (*opensearchapi.Response, error)
	Delete(ctx context.Context, indexName string, documentID string) (*opensearchapi.Response, error)
}

type ProductEventsConsumer struct {
	logger          logs.Logger
	subscriber      Subscriber
	indexer         DocumentStore
	opensearchIndex string
}

func NewProductEventsConsumer(logger logs.Logger, subscriber Subscriber, indexer DocumentStore, index string) *ProductEventsConsumer {
	return &ProductEventsConsumer{
		logger:          logger,
		subscriber:      subscriber,
		indexer:         indexer,
		opensearchIndex: index,
	}
}

func (p *ProductEventsConsumer) Start(ctx context.Context) error {
	unixTime := time.Now().Unix()
	unixTimeStr := strconv.Itoa(int(unixTime))

	return p.subscriber.Subscribe(ctx, rabbitmq.SubscribeOptions{
		Exchange:     events.ProductExchangeName,
		ExchangeType: rabbitmq.ExchangeTopic,
		QueueName:    "search_product_events_queue",
		ConsumerTag:  "search_product_events_indexer_" + unixTimeStr,
		BindingKey:   events.ProductWaildCardRoutingKey,
		Handler:      p.handleProductCreatedEvent,
	})
}

func (p *ProductEventsConsumer) handleProductCreatedEvent(ctx context.Context, d amqp091.Delivery) {
	var productEvent events.Product
	if err := json.Unmarshal(d.Body, &productEvent); err != nil {
		p.logger.Error("failed to unmarshal product event", "error", err)
		d.Nack(false, false)
		return
	}

	p.logger.Info("product event received", "routingKey", d.RoutingKey, "productId", productEvent.ID)

	body, err := json.Marshal(productEvent)
	if err != nil {
		p.logger.Error("failed to marshal product event for opensearch", "error", err, "productId", productEvent.ID)
		d.Nack(false, true)
		return
	}

	if d.RoutingKey != events.ProductDeletedRoutingKey {
		p.indexProduct(ctx, productEvent, body, d)
	} else {
		p.deleteProduct(ctx, productEvent, d)
	}

}

func (p *ProductEventsConsumer) indexProduct(ctx context.Context, productEvent events.Product, body []byte, d amqp091.Delivery) {
	res, err := p.indexer.Index(
		ctx,
		p.opensearchIndex,
		productEvent.ID,
		body,
	)
	if err != nil {
		p.logger.Error("failed to index document in opensearch", "error", err, "productId", productEvent.ID)
		d.Nack(false, true)
		return
	}

	if res.IsError() {
		p.logger.Error("opensearch returned an error during indexing", "status", res.Status(), "productId", productEvent.ID)
		d.Nack(false, true)
		return
	}

	p.logger.Info("product indexed successfully", "index", p.opensearchIndex, "productId", productEvent.ID, "opensearchStatus", res.Status())
	d.Ack(false)
}

func (p *ProductEventsConsumer) deleteProduct(ctx context.Context, productEvent events.Product, d amqp091.Delivery) {
	res, err := p.indexer.Delete(
		ctx,
		p.opensearchIndex,
		productEvent.ID,
	)
	if err != nil {
		p.logger.Error("failed to delete document in opensearch", "error", err, "productId", productEvent.ID)
		d.Nack(false, true)
		return
	}

	if res.IsError() {
		p.logger.Error("opensearch returned an error during deletion", "status", res.Status(), "productId", productEvent.ID)
		d.Nack(false, true)
		return
	}

	p.logger.Info("product deleted successfully", "index", p.opensearchIndex, "productId", productEvent.ID, "opensearchStatus", res.Status())
	d.Ack(false)
}
