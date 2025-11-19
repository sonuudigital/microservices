package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/product-service/internal/events/consumers"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/grpc/category"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	repo_postgres "github.com/sonuudigital/microservices/product-service/internal/repository/postgres"
	"github.com/sonuudigital/microservices/shared/events/worker"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
	"github.com/sonuudigital/microservices/shared/web"
	"github.com/sonuudigital/microservices/shared/web/health"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/joho/godotenv"
)

func main() {
	logger := logs.NewSlogLogger()
	err := godotenv.Load()
	if err == nil {
		logger.Info("loaded environment variables from .env file")
	} else {
		logger.Info("no .env file found, using environment variables")
	}

	pgDb, err := postgres.InitializePostgresDB()
	if err != nil {
		logger.Error("error connecting to database", "error", err)
		os.Exit(1)
	}
	logger.Info("database connected successfully")
	defer pgDb.Close()

	redisClient, err := initializeRedisClient()
	if err != nil {
		logger.Error("error connecting to redis", "error", err)
		os.Exit(1)
	}
	logger.Info("redis connected successfully")
	defer redisClient.Close()

	rabbitmq, err := initializeRabbitMQ(logger)
	if err != nil {
		logger.Error("failed to initialize RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbitmq.Close()

	initializeServicesAndWaitForShutdown(logger, rabbitmq, pgDb, redisClient)
}

func initializeServicesAndWaitForShutdown(logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, pgDb *pgxpool.Pool, redisClient *redis.Client) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return startRabbitMQConsumer(gCtx, logger, rabbitmq, redisClient, repository.NewPostgreSQLOrderCreatedConsumerRepository(pgDb))
	})

	g.Go(func() error {
		return startGRPCServer(gCtx, pgDb, redisClient, logger)
	})

	go startMessageRelayerWorker(gCtx, logger, rabbitmq, pgDb)

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}

	logger.Info("application shut down gracefully")
}

func startGRPCServer(ctx context.Context, pgDb *pgxpool.Pool, redisClient *redis.Client, logger logs.Logger) error {
	grpcPort := os.Getenv("PRODUCT_SERVICE_GRPC_PORT")
	if grpcPort == "" {
		return fmt.Errorf("PRODUCT_SERVICE_GRPC_PORT is not set")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		return fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	productRepo := repo_postgres.NewProductRepository(pgDb)
	grpcServer := grpc.NewServer()

	categoryServer := category.New(logger, productRepo, redisClient)
	productServer := grpc_server.NewServer(logger, productRepo, redisClient)

	product_categoriesv1.RegisterProductCategoriesServiceServer(grpcServer, categoryServer)
	productv1.RegisterProductServiceServer(grpcServer, productServer)

	health.StartGRPCHealthCheckService(grpcServer, "product-service", func(ctx context.Context) error {
		dbErr := pgDb.Ping(ctx)
		redisErr := redisClient.Ping(ctx).Err()

		if dbErr == nil && redisErr == nil {
			logger.Info("service is healthy and serving")
			return nil
		} else {
			errors := errors.Join(dbErr, redisErr)
			logger.Error("service is not healthy", "errors", errors)
			return errors
		}
	})

	return web.StartGRPCServerAndWaitForShutdown(ctx, grpcServer, lis, logger)
}

func startMessageRelayerWorker(ctx context.Context, logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, repo *pgxpool.Pool) {
	pollInterval, batchSize, err := getMessageRelayerConfigFromEnv()
	if err != nil {
		logger.Error("failed to get message relayer config from env", "error", err)
		os.Exit(1)
	}
	worker.NewOutboxEventMessageRelayer(
		logger,
		rabbitmq,
		repo_postgres.NewOutboxEventMessageRelayerRepository(repo),
		pollInterval,
		batchSize,
	).Start(ctx)
}

func startRabbitMQConsumer(ctx context.Context, logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, redisClient *redis.Client, repo consumers.OrderCreatedConsumerRepository) error {
	orderCreatedConsumer := consumers.NewOrderCreatedConsumer(logger, rabbitmq, redisClient, repo)
	logger.Info("starting OrderCreatedConsumer")

	if err := orderCreatedConsumer.Start(ctx); err != nil {
		return fmt.Errorf("OrderCreatedConsumer failed: %w", err)
	}

	logger.Info("OrderCreatedConsumer stopped gracefully")
	return nil
}

func initializeRedisClient() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is not set")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("could not ping redis: %w", err)
	}

	return client, nil
}

func initializeRabbitMQ(logger logs.Logger) (*rabbitmq.RabbitMQ, error) {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is not set")
	}

	rabbitmq, err := rabbitmq.NewConnection(logger, rabbitmqURL)
	if err != nil {
		return nil, err
	}

	return rabbitmq, nil
}

func getMessageRelayerConfigFromEnv() (time.Duration, int32, error) {
	pollIntervalStr := os.Getenv("MESSAGE_RELAYER_POLL_INTERVAL")
	if pollIntervalStr == "" {
		pollIntervalStr = "5s"
	}

	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid MESSAGE_RELAYER_POLL_INTERVAL: %w", err)
	}

	batchSizeStr := os.Getenv("MESSAGE_RELAYER_BATCH_SIZE")
	if batchSizeStr == "" {
		batchSizeStr = "25"
	}

	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid MESSAGE_RELAYER_BATCH_SIZE: %w", err)
	}

	return pollInterval, int32(batchSize), nil
}
