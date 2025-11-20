package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sonuudigital/microservices/search-service/internal/events/consumers/product"
	"github.com/sonuudigital/microservices/search-service/internal/opensearch"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

func main() {
	logger := logs.NewSlogLogger()
	err := godotenv.Load()
	if err == nil {
		logger.Info("loaded environment variables from .env file")
	} else {
		logger.Info("no .env file found, using environment variables")
	}

	rabbitmqClient, err := initializeRabbitMQClient(logger)
	if err != nil {
		logger.Error("failed to initialize RabbitMQ client", "error", err)
		os.Exit(1)
	}
	defer rabbitmqClient.Close()

	opensearchClient, err := initializeOpenSearchClient(logger)
	if err != nil {
		logger.Error("failed to initialize OpenSearch client", "error", err)
		os.Exit(1)
	}

	opensearchProductIndex := os.Getenv("OPENSEARCH_PRODUCT_INDEX")
	if opensearchProductIndex == "" {
		logger.Error("OPENSEARCH_PRODUCT_INDEX is not set")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go product.
		NewProductCreatedEventConsumer(logger, rabbitmqClient, opensearchClient, opensearchProductIndex).
		Start(ctx)

	<-ctx.Done()
	logger.Info("application shut down gracefully")
}

func initializeRabbitMQClient(logger logs.Logger) (*rabbitmq.Client, error) {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is not set")
	}

	client, err := rabbitmq.NewClient(logger, rabbitmqURL)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func initializeOpenSearchClient(logger logs.Logger) (*opensearch.Client, error) {
	opensearchAddresses := []string{os.Getenv("OPENSEARCH_ADDRESS")}
	opensearchUsername := os.Getenv("OPENSEARCH_USERNAME")
	opensearchPassword := os.Getenv("OPENSEARCH_PASSWORD")

	logger.Info(
		"connecting to OpenSearch",
		"addresses", opensearchAddresses,
		"username", opensearchUsername,
		"passwordSet", opensearchPassword != "",
	)

	client, err := opensearch.NewClient(opensearchAddresses, opensearchUsername, opensearchPassword)
	if err != nil {
		return nil, err
	}

	return client, nil
}
