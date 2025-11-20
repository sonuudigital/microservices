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

type Indexser interface {
	Index(ctx context.Context, indexName string, documentID string, body []byte) (*opensearchapi.Response, error)
}

type ProductCreatedEventConsumer struct {
	logger          logs.Logger
	subscriber      Subscriber
	indexser        Indexser
	opensearchIndex string
}

func NewProductCreatedEventConsumer(logger logs.Logger, subscriber Subscriber, indexser Indexser, index string) *ProductCreatedEventConsumer {
	return &ProductCreatedEventConsumer{
		logger:          logger,
		subscriber:      subscriber,
		indexser:        indexser,
		opensearchIndex: index,
	}
}

func (p *ProductCreatedEventConsumer) Start(ctx context.Context) error {
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

func (p *ProductCreatedEventConsumer) handleProductCreatedEvent(ctx context.Context, d amqp091.Delivery) {
	var productEvent events.Product
	if err := json.Unmarshal(d.Body, &productEvent); err != nil {
		p.logger.Error("failed to unmarshal product event", "error", err)
		d.Nack(false, false)
		return
	}

	p.logger.Info("product event received", "productId", productEvent.ID)

	body, err := json.Marshal(productEvent)
	if err != nil {
		p.logger.Error("failed to marshal product event for opensearch", "error", err, "productId", productEvent.ID)
		d.Nack(false, true)
		return
	}

	res, err := p.indexser.Index(
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

	p.logger.Info("product indexed successfully", "productId", productEvent.ID, "opensearchStatus", res.Status())
	d.Ack(false)
}
