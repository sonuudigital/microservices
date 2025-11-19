package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/logs"
)

type RabbitMQ struct {
	*connectionManager
}

func NewConnection(logger logs.Logger, url string) (*RabbitMQ, error) {
	manager, err := newConnectionManager(logger, url)
	if err != nil {
		return nil, err
	}
	return &RabbitMQ{connectionManager: manager}, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange string, body []byte) error {
	return r.retryWithReconnect(ctx, "publish", func() error {
		return r.attemptFanoutPublish(ctx, exchange, body)
	})
}

func (r *RabbitMQ) attemptFanoutPublish(ctx context.Context, exchange string, body []byte) error {
	if err := r.ensureFanoutExchange(exchange); err != nil {
		return err
	}
	return r.publishFanoutMessage(ctx, exchange, body)
}

func (r *RabbitMQ) ensureFanoutExchange(name string) error {
	return r.channel.ExchangeDeclare(
		name,
		string(ExchangeFanout),
		true,
		false,
		false,
		false,
		nil,
	)
}

func (r *RabbitMQ) publishFanoutMessage(ctx context.Context, exchange string, body []byte) error {
	publishing := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Body:         body,
		Timestamp:    time.Now(),
	}

	return r.channel.PublishWithContext(
		ctx,
		exchange,
		"",
		false,
		false,
		publishing,
	)
}

func (r *RabbitMQ) Subscribe(ctx context.Context, exchange, queueName, consumerTag string, handler func(ctx context.Context, d amqp091.Delivery)) error {
	for {
		if err := r.setupFanoutSubscription(exchange, queueName); err != nil {
			if !r.isConnectionError(err) {
				return fmt.Errorf("failed to setup fanout subscription: %w", err)
			}
			r.logger.Warn("failed to setup subscription, attempting to reconnect...", "error", err)
			if reconnErr := r.reconnect(ctx); reconnErr != nil {
				return fmt.Errorf("failed to reconnect during subscription setup: %w", reconnErr)
			}
			r.logger.Info("reconnected, retrying subscription setup")
			continue
		}

		msgs, err := r.channel.Consume(
			queueName,
			consumerTag,
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to start consuming: %w", err)
		}

		r.logger.Info("consumer subscribed", "consumerTag", consumerTag, "queue", queueName)

		err = r.consumeMessages(ctx, consumerTag, msgs, handler)

		if ctx.Err() != nil {
			r.logger.Info("context cancelled, stopping consumer", "consumerTag", consumerTag)
			return ctx.Err()
		}

		r.logger.Warn("consumer connection lost, attempting to reconnect...", "consumerTag", consumerTag, "error", err)

		if err := r.reconnect(ctx); err != nil {
			return fmt.Errorf(failedToReconnectMsg, err)
		}

		r.logger.Info("resubscribing consumer after reconnection", "consumerTag", consumerTag)
	}
}

func (r *RabbitMQ) setupFanoutSubscription(exchange, queueName string) error {
	if err := r.channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	dlxName := exchange + ".dlx"
	dlqName := queueName + ".dlq"

	if err := r.channel.ExchangeDeclare(dlxName, string(ExchangeFanout), true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLX %s: %w", dlxName, err)
	}

	if _, err := r.channel.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLQ %s: %w", dlqName, err)
	}

	if err := r.channel.QueueBind(dlqName, "", dlxName, false, nil); err != nil {
		return fmt.Errorf("failed to bind DLQ to DLX: %w", err)
	}

	if err := r.ensureFanoutExchange(exchange); err != nil {
		return fmt.Errorf("failed to declare main exchange %s: %w", exchange, err)
	}

	args := amqp091.Table{"x-dead-letter-exchange": dlxName}
	if _, err := r.channel.QueueDeclare(queueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	if err := r.channel.QueueBind(queueName, "", exchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	return nil
}
