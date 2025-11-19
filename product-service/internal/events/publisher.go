package events

import (
	"context"
	"strings"

	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

type DelegatingPublisher struct {
	fanoutPublisher *rabbitmq.RabbitMQ
	client          *rabbitmq.Client
}

func NewDelegatingPublisher(fanoutPublisher *rabbitmq.RabbitMQ, client *rabbitmq.Client) *DelegatingPublisher {
	return &DelegatingPublisher{
		fanoutPublisher: fanoutPublisher,
		client:          client,
	}
}

func (p *DelegatingPublisher) Publish(ctx context.Context, eventName string, body []byte) error {
	parts := strings.SplitN(eventName, ":", 2)

	if len(parts) == 2 {
		exchange := parts[0]
		routingKey := parts[1]

		opts := rabbitmq.PublishOptions{
			Exchange:     exchange,
			ExchangeType: rabbitmq.ExchangeTopic,
			RoutingKey:   routingKey,
			Body:         body,
		}
		return p.client.Publish(ctx, opts)
	}

	return p.fanoutPublisher.Publish(ctx, eventName, body)
}
