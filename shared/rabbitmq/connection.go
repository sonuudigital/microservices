package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	maxRetries           = 3
	backoff              = 100 * time.Millisecond
	failedToReconnectMsg = "failed to reconnect: %w"
)

type connectionManager struct {
	logger     logs.Logger
	url        string
	connection *amqp091.Connection
	channel    *amqp091.Channel
}

func newConnectionManager(logger logs.Logger, url string) (*connectionManager, error) {
	manager := &connectionManager{
		logger: logger,
		url:    url,
	}

	if err := manager.connect(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (cm *connectionManager) connect() error {
	conn, err := amqp091.Dial(cm.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	cm.connection = conn
	cm.channel = ch
	cm.logger.Info("connected to RabbitMQ")
	return nil
}

func (cm *connectionManager) reconnect(ctx context.Context) error {
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
			cm.logger.Info("attempting to reconnect to RabbitMQ", "attempt", attempts, "backoff", backoff)

			if err := cm.connect(); err != nil {
				cm.logger.Error("failed to reconnect", "error", err, "attempt", attempts, "nextRetry", backoff*2)

				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			cm.logger.Info("successfully reconnected to RabbitMQ")
			return nil
		}
	}
}

func (cm *connectionManager) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if cm.connection == nil || cm.connection.IsClosed() {
		return true
	}

	if cm.channel == nil {
		return true
	}

	errStr := err.Error()
	return errStr == "Exception (504) Reason: \"channel/connection is not open\"" ||
		errStr == "Exception (320) Reason: \"CONNECTION_FORCED\"" ||
		errStr == "write tcp: broken pipe" ||
		errStr == "EOF"
}

func (cm *connectionManager) Close() {
	if cm.channel != nil {
		cm.channel.Close()
	}
	if cm.connection != nil {
		cm.connection.Close()
	}
	cm.logger.Info("rabbitmq connection manager closed")
}

func (cm *connectionManager) Ping() error {
	if cm.connection.IsClosed() {
		return fmt.Errorf("rabbitmq connection is closed")
	}
	return nil
}

func (cm *connectionManager) shouldReturn(err error, attempt, maxRetries int) bool {
	return !cm.isConnectionError(err) || attempt == maxRetries
}

func (cm *connectionManager) tryReconnect(ctx context.Context) error {
	if err := cm.reconnect(ctx); err != nil {
		return fmt.Errorf(failedToReconnectMsg, err)
	}
	return nil
}

func (cm *connectionManager) retryWithReconnect(ctx context.Context, opName string, op func() error) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := op(); err != nil {
			if cm.shouldReturn(err, attempt, maxRetries) {
				return err
			}
			cm.logger.Warn(opName+": transient error, attempting reconnect", "attempt", attempt, "error", err)
			if err := cm.tryReconnect(ctx); err != nil {
				return err
			}
			time.Sleep(backoff * time.Duration(attempt))
			continue
		}
		return nil
	}
	return fmt.Errorf("%s failed after %d retries", opName, maxRetries)
}

func (cm *connectionManager) consumeMessages(ctx context.Context, consumerTag string, msgs <-chan amqp091.Delivery, handler func(ctx context.Context, d amqp091.Delivery)) error {
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
