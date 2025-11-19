package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sonuudigital/microservices/shared/logs"
)

type Client struct {
	*connectionManager
}

func NewClient(logger logs.Logger, url string) (*Client, error) {
	manager, err := newConnectionManager(logger, url)
	if err != nil {
		return nil, err
	}
	return &Client{connectionManager: manager}, nil
}

func (c *Client) Publish(ctx context.Context, opts PublishOptions) error {
	return c.retryWithReconnect(ctx, "publish", func() error {
		if err := c.ensureExchange(opts.Exchange, opts.ExchangeType); err != nil {
			return err
		}
		return c.publishMessage(ctx, opts)
	})
}

func (c *Client) ensureExchange(name string, exchangeType ExchangeType) error {
	return c.channel.ExchangeDeclare(
		name,
		string(exchangeType),
		true,
		false,
		false,
		false,
		nil,
	)
}

func (c *Client) publishMessage(ctx context.Context, opts PublishOptions) error {
	publishing := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Body:         opts.Body,
		Timestamp:    time.Now(),
	}

	return c.channel.PublishWithContext(
		ctx,
		opts.Exchange,
		opts.RoutingKey,
		false,
		false,
		publishing,
	)
}

func (c *Client) Subscribe(ctx context.Context, opts SubscribeOptions) error {
	for {
		if err := c.setupSubscription(opts); err != nil {
			if !c.isConnectionError(err) {
				return fmt.Errorf("failed to setup subscription: %w", err)
			}
			c.logger.Warn("failed to setup subscription, attempting to reconnect...", "error", err)
			if reconnErr := c.reconnect(ctx); reconnErr != nil {
				return fmt.Errorf("failed to reconnect during subscription setup: %w", reconnErr)
			}
			c.logger.Info("reconnected, retrying subscription setup")
			continue
		}

		msgs, err := c.channel.Consume(
			opts.QueueName,
			opts.ConsumerTag,
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to start consuming: %w", err)
		}

		c.logger.Info("consumer subscribed", "consumerTag", opts.ConsumerTag, "queue", opts.QueueName)

		err = c.consumeMessages(ctx, opts.ConsumerTag, msgs, opts.Handler)

		if ctx.Err() != nil {
			c.logger.Info("context cancelled, stopping consumer", "consumerTag", opts.ConsumerTag)
			return ctx.Err()
		}

		c.logger.Warn("consumer connection lost, attempting to reconnect...", "consumerTag", opts.ConsumerTag, "error", err)

		if err := c.reconnect(ctx); err != nil {
			return fmt.Errorf(failedToReconnectMsg, err)
		}

		c.logger.Info("resubscribing consumer after reconnection", "consumerTag", opts.ConsumerTag)
	}
}

func (c *Client) setupSubscription(opts SubscribeOptions) error {
	if err := c.channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	dlxName := opts.Exchange + ".dlx"
	dlqName := opts.QueueName + ".dlq"

	if err := c.channel.ExchangeDeclare(dlxName, string(ExchangeTopic), true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLX %s: %w", dlxName, err)
	}

	if _, err := c.channel.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLQ %s: %w", dlqName, err)
	}

	if err := c.channel.QueueBind(dlqName, "#", dlxName, false, nil); err != nil {
		return fmt.Errorf("failed to bind DLQ to DLX: %w", err)
	}

	if err := c.ensureExchange(opts.Exchange, opts.ExchangeType); err != nil {
		return fmt.Errorf("failed to declare main exchange %s: %w", opts.Exchange, err)
	}

	args := amqp091.Table{"x-dead-letter-exchange": dlxName}
	if _, err := c.channel.QueueDeclare(opts.QueueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", opts.QueueName, err)
	}

	if err := c.channel.QueueBind(opts.QueueName, opts.BindingKey, opts.Exchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	return nil
}
