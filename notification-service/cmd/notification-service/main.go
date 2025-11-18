package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sonuudigital/microservices/notification-service/internal/email"
	"github.com/sonuudigital/microservices/notification-service/internal/events"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

func main() {
	logger := logs.NewSlogLogger()
	err := godotenv.Load()
	if err == nil {
		logger.Info("loaded environment variables from .env file")
	}

	smtpSender, err := initializeSMTPSender(logger)
	if err != nil {
		logger.Error("failed to initialize SMTP sender", "error", err)
		os.Exit(1)
	}

	rabbitmqConn, err := initializeRabbitMQ(logger)
	if err != nil {
		logger.Error("failed to initialize RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbitmqConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orderCreatedConsumer := events.NewOrderCreatedConsumer(logger, smtpSender, rabbitmqConn)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		logger.Info("shutdown signal received, cancelling context...")
		cancel()
	}()

	logger.Info("starting OrderCreatedConsumer...")
	if err := orderCreatedConsumer.Start(ctx); err != nil {
		logger.Error("OrderCreatedConsumer stopped", "error", err)
	}

	logger.Info("service shut down gracefully")
}

func initializeSMTPSender(logger logs.Logger) (*email.SMTPSender, error) {
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM_EMAIL")

	if host == "" || portStr == "" || from == "" {
		logger.Error(
			"SMTP configuration is not set properly",
			"host", host,
			"port", portStr,
			"username", username,
			"from", from,
		)
		return nil, fmt.Errorf("SMTP configuration is not set properly, check SMTP_HOST, SMTP_PORT, and SMTP_FROM_EMAIL")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	smtpSender := email.NewSMTPSender(host, port, username, password, from)
	return smtpSender, nil
}

func initializeRabbitMQ(logger logs.Logger) (*rabbitmq.RabbitMQ, error) {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is not set")
	}

	rabbitmqConn, err := rabbitmq.NewConnection(logger, rabbitmqURL)
	if err != nil {
		return nil, err
	}

	return rabbitmqConn, nil
}
