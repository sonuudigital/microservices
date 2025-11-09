package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/logs"
)

type RabbitMQ struct {
	logger     logs.Logger
	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewConnection(logger logs.Logger, url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	logger.Info("Connected to RabbitMQ", "url", url)
	return &RabbitMQ{
		logger:     logger,
		connection: conn,
		channel:    ch,
	}, nil
}

func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.connection != nil {
		r.connection.Close()
	}
	r.logger.Info("RabbitMQ connection closed")
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	publishing := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}

	err := r.channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		false,
		false,
		publishing,
	)
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}
	return nil
}

func (r *RabbitMQ) Subscribe(ctx context.Context, exchange, queueName, consumerTag string, handler func(d amqp.Delivery)) error {
	err := r.channel.ExchangeDeclare(
		exchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	q, err := r.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	err = r.channel.QueueBind(
		q.Name,
		"",
		exchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind a queue: %w", err)
	}

	msgs, err := r.channel.Consume(
		q.Name,
		consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	go func() {
		shouldReturn := r.consumeMessages(ctx, consumerTag, msgs, handler)
		if shouldReturn {
			return
		}
	}()

	r.logger.Info("Consumer subscribed", "consumerTag", consumerTag, "queue", q.Name)
	return nil
}

func (r *RabbitMQ) consumeMessages(ctx context.Context, consumerTag string, msgs <-chan amqp.Delivery, handler func(d amqp.Delivery)) bool {
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context cancelled, stopping consumer", "consumerTag", consumerTag)
			return true
		case d, ok := <-msgs:
			if !ok {
				r.logger.Info("Message channel closed, stopping consumer", "consumerTag", consumerTag)
				return true
			}
			go handler(d)
		}
	}
}
