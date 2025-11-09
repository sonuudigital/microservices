package rabbitmq

import (
	"context"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/logs"
)

type RabbitMQ struct {
	logger     logs.Logger
	connection *amqp091.Connection
	channel    *amqp091.Channel
}

func NewConnection(logger logs.Logger, url string) (*RabbitMQ, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	logger.Info("connected to RabbitMQ")
	return &RabbitMQ{
		logger:     logger,
		connection: conn,
		channel:    ch,
	}, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange string, body []byte) error {
	publishing := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Body:         body,
	}

	err := r.channel.PublishWithContext(ctx,
		exchange,
		"",
		false,
		false,
		publishing,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) Subscribe(ctx context.Context, exchange, queueName, consumerTag string, handler func(d amqp091.Delivery)) error {
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
		return err
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
		return err
	}

	err = r.channel.QueueBind(
		q.Name,
		"",
		exchange,
		false,
		nil,
	)
	if err != nil {
		return err
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
		return err
	}

	r.logger.Info("Consumer subscribed", "consumerTag", consumerTag, "queue", q.Name)
	return r.consumeMessages(ctx, consumerTag, msgs, handler)
}

func (r *RabbitMQ) consumeMessages(ctx context.Context, consumerTag string, msgs <-chan amqp091.Delivery, handler func(d amqp091.Delivery)) error {
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Context cancelled, stopping consumer", "consumerTag", consumerTag)
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				r.logger.Error("Message channel closed, stopping consumer", "consumerTag", consumerTag)
				return fmt.Errorf("rabbitmq channel closed for consumer %s", consumerTag)
			}
			go handler(d)
		}
	}
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

func (r *RabbitMQ) Ping() error {
	if r.connection.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	} else {
		return nil
	}
}
