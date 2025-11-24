package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sonuudigital/microservices/search-service/internal/events/consumers/product"
	productHandler "github.com/sonuudigital/microservices/search-service/internal/handlers/product"
	"github.com/sonuudigital/microservices/search-service/internal/opensearch"
	"github.com/sonuudigital/microservices/search-service/internal/router"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
	"github.com/sonuudigital/microservices/shared/web"
	"golang.org/x/sync/errgroup"
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

	productHandler, err := productHandler.NewProductHandler(logger, opensearchClient, opensearchProductIndex)
	if err != nil {
		logger.Error("failed to create product handler", "error", err)
		os.Exit(1)
	}

	router, err := router.New(logger, productHandler)
	if err != nil {
		logger.Error("failed to create router", "error", err)
		os.Exit(1)
	}

	if err := startServices(logger, rabbitmqClient, opensearchClient, opensearchProductIndex, router); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}

	logger.Info("application shut down gracefully")
}

func startServices(logger *logs.SlogLogger, rabbitmqClient *rabbitmq.Client, opensearchClient *opensearch.Client, opensearchProductIndex string, router *router.Router) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return startProductEventConsumer(gCtx, logger, rabbitmqClient, opensearchClient, opensearchProductIndex)
	})

	g.Go(func() error {
		return startHTTPServer(gCtx, logger, router)
	})

	return g.Wait()
}

func startProductEventConsumer(ctx context.Context, logger logs.Logger, rabbitmqClient *rabbitmq.Client, opensearchClient *opensearch.Client, opensearchProductIndex string) error {
	return product.
		NewProductEventsConsumer(logger, rabbitmqClient, opensearchClient, opensearchProductIndex).
		Start(ctx)
}

func startHTTPServer(ctx context.Context, logger logs.Logger, handler http.Handler) error {
	srv, err := web.InitializeServer(os.Getenv("PORT"), handler, logger)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shCancel()
		if err := srv.Shutdown(shCtx); err != nil {
			logger.Error("failed to shutdown server", "error", err)
		} else {
			logger.Info("server shutdown complete")
		}
	}()

	logger.Info("starting HTTP server", "port", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server failed: %w", err)
	}
	return nil
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
