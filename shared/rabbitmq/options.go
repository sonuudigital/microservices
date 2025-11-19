package rabbitmq

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
)

type ExchangeType string

const (
	ExchangeFanout ExchangeType = "fanout"
	ExchangeTopic  ExchangeType = "topic"
	ExchangeDirect ExchangeType = "direct"
)

type PublishOptions struct {
	Exchange     string
	ExchangeType ExchangeType
	RoutingKey   string
	Body         []byte
}

type SubscribeOptions struct {
	Exchange     string
	ExchangeType ExchangeType
	QueueName    string
	ConsumerTag  string
	BindingKey   string
	Handler      func(ctx context.Context, d amqp091.Delivery)
}
