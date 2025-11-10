package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/logs"
)

type RabbitMQ struct {
	logger     logs.Logger
	url        string
	connection *amqp091.Connection
	channel    *amqp091.Channel
}

func NewConnection(logger logs.Logger, url string) (*RabbitMQ, error) {
	rabbitmq := &RabbitMQ{
		logger: logger,
		url:    url,
	}

	if err := rabbitmq.connect(); err != nil {
		return nil, err
	}

	return rabbitmq, nil
}

func (r *RabbitMQ) connect() error {
	conn, err := amqp091.Dial(r.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	r.connection = conn
	r.channel = ch
	r.logger.Info("connected to RabbitMQ")
	return nil
}

func (r *RabbitMQ) reconnect(ctx context.Context) error {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	maxAttempts := 10
	attempts := 0

	for {
		attempts++
		if attempts > maxAttempts {
			return fmt.Errorf("max reconnection attempts reached: %d", maxAttempts)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			r.logger.Info("attempting to reconnect to RabbitMQ", "attempt", attempts, "backoff", backoff)

			if err := r.connect(); err != nil {
				r.logger.Error("failed to reconnect", "error", err, "attempt", attempts, "nextRetry", backoff*2)

				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			r.logger.Info("successfully reconnected to RabbitMQ")
			return nil
		}
	}
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange string, body []byte) error {
	const maxRetries = 3
	const backoff = 100 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := r.ensureExchange(ctx, exchange, attempt, maxRetries, backoff); err != nil {
			return err
		}

		err := r.publishMessage(ctx, exchange, body, attempt, maxRetries, backoff)
		if err == nil {
			return nil
		}

		if !r.isConnectionError(err) || attempt == maxRetries {
			return err
		}
	}

	return fmt.Errorf("publish failed after %d retries", maxRetries)
}

func (r *RabbitMQ) ensureExchange(ctx context.Context, exchange string, attempt, maxRetries int, backoff time.Duration) error {
	err := r.channel.ExchangeDeclare(
		exchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)

	if err == nil {
		return nil
	}

	if attempt == maxRetries {
		return fmt.Errorf("failed to declare exchange after %d attempts: %w", maxRetries, err)
	}

	if r.isConnectionError(err) {
		return r.handleConnectionError(ctx, attempt, backoff, err)
	}

	return fmt.Errorf("failed to declare exchange: %w", err)
}

func (r *RabbitMQ) publishMessage(ctx context.Context, exchange string, body []byte, attempt, maxRetries int, backoff time.Duration) error {
	publishing := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Body:         body,
		Timestamp:    time.Now(),
	}

	err := r.channel.PublishWithContext(ctx, exchange, "", false, false, publishing)

	if err == nil {
		if attempt > 1 {
			r.logger.Info("publish succeeded after retry", "attempt", attempt, "exchange", exchange)
		}
		return nil
	}

	if attempt == maxRetries {
		return fmt.Errorf("failed to publish after %d attempts: %w", maxRetries, err)
	}

	if r.isConnectionError(err) {
		return r.handleConnectionError(ctx, attempt, backoff, err)
	}

	return fmt.Errorf("failed to publish: %w", err)
}

func (r *RabbitMQ) handleConnectionError(ctx context.Context, attempt int, backoff time.Duration, err error) error {
	const errReconnect = "failed to reconnect: %w"

	r.logger.Warn("publish: connection error, attempting reconnect", "attempt", attempt, "error", err)

	if reconnectErr := r.reconnect(ctx); reconnectErr != nil {
		return fmt.Errorf(errReconnect, reconnectErr)
	}

	time.Sleep(backoff * time.Duration(attempt))
	return nil
}

func (r *RabbitMQ) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if r.connection == nil || r.connection.IsClosed() {
		return true
	}

	if r.channel == nil {
		return true
	}

	errStr := err.Error()
	return errStr == "Exception (504) Reason: \"channel/connection is not open\"" ||
		errStr == "Exception (320) Reason: \"CONNECTION_FORCED\"" ||
		errStr == "write tcp: broken pipe" ||
		errStr == "EOF"
}

func (r *RabbitMQ) Subscribe(ctx context.Context, exchange, queueName, consumerTag string, handler func(ctx context.Context, d amqp091.Delivery)) error {
	for {
		err := r.channel.Qos(
			10,
			0,
			false,
		)
		if err != nil {
			return fmt.Errorf("failed to set QoS: %w", err)
		}

		if err := r.setupDeadLetterInfrastructure(exchange, queueName); err != nil {
			return err
		}

		if err := r.setupMainQueue(exchange, queueName); err != nil {
			return err
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

		r.logger.Warn("consumer connection lost, attempting to reconnect", "consumerTag", consumerTag, "error", err)

		if err := r.reconnect(ctx); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}

		r.logger.Info("resubscribing consumer after reconnection", "consumerTag", consumerTag)
	}
}

func (r *RabbitMQ) setupDeadLetterInfrastructure(exchange, queueName string) error {
	dlxName := exchange + ".dlx"
	dlqName := queueName + ".dlq"

	err := r.channel.ExchangeDeclare(
		dlxName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLX %s: %w", dlxName, err)
	}

	_, err = r.channel.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ %s: %w", dlqName, err)
	}

	err = r.channel.QueueBind(
		dlqName,
		"",
		dlxName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind DLQ to DLX: %w", err)
	}

	r.logger.Debug("dead Letter infrastructure configured",
		"dlx", dlxName,
		"dlq", dlqName,
	)

	return nil
}

func (r *RabbitMQ) setupMainQueue(exchange, queueName string) error {
	dlxName := exchange + ".dlx"

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
		return fmt.Errorf("failed to declare exchange %s: %w", exchange, err)
	}

	_, err = r.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp091.Table{

			"x-dead-letter-exchange": dlxName,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	err = r.channel.QueueBind(
		queueName,
		"",
		exchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	r.logger.Debug("main queue configured",
		"exchange", exchange,
		"queue", queueName,
		"dlx", dlxName,
	)

	return nil
}

func (r *RabbitMQ) consumeMessages(ctx context.Context, consumerTag string, msgs <-chan amqp091.Delivery, handler func(ctx context.Context, d amqp091.Delivery)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitmq channel closed for consumer %s", consumerTag)
			}
			go func(delivery amqp091.Delivery) {
				handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				handler(handlerCtx, delivery)
			}(d)
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
	r.logger.Info("rabbitmq connection closed")
}

func (r *RabbitMQ) Ping() error {
	if r.connection.IsClosed() {
		return fmt.Errorf("rabbitmq connection is closed")
	} else {
		return nil
	}
}
