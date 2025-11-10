package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/sonuudigital/microservices/cart-service/internal/clients"
	"github.com/sonuudigital/microservices/cart-service/internal/events"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
	"github.com/sonuudigital/microservices/shared/web"
	"github.com/sonuudigital/microservices/shared/web/health"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
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

	rabbitmq, err := initializeRabbitMQ(logger)
	if err != nil {
		logger.Error("failed to initialize RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbitmq.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return startRabbitMQConsumer(gCtx, logger, rabbitmq, repository.New(pgDb))
	})

	g.Go(func() error {
		return startGRPCServer(gCtx, pgDb, redisClient, logger)
	})

	if err := g.Wait(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}

	logger.Info("application shut down gracefully")
}

func initializeRedisClient() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is not set")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
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

func startGRPCServer(ctx context.Context, pgDb *pgxpool.Pool, redisClient *redis.Client, logger logs.Logger) error {
	productServiceGrpcURL := os.Getenv("PRODUCT_SERVICE_GRPC_URL")
	if productServiceGrpcURL == "" {
		return fmt.Errorf("PRODUCT_SERVICE_GRPC_URL is not set")
	}
	productClient, err := clients.NewProductClient(productServiceGrpcURL, logger)
	if err != nil {
		return fmt.Errorf("failed to create product client: %w", err)
	}
	logger.Info("product client created successfully", "url", productServiceGrpcURL)

	grpcPort := os.Getenv("CART_SERVICE_GRPC_PORT")
	if grpcPort == "" {
		return fmt.Errorf("CART_SERVICE_GRPC_PORT is not set")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		return fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	queries := repository.New(pgDb)
	grpcServer := grpc.NewServer()
	cartServer := grpc_server.NewGRPCServer(queries, productClient, redisClient, logger)
	cartv1.RegisterCartServiceServer(grpcServer, cartServer)

	health.StartGRPCHealthCheckService(grpcServer, "cart-service", func(ctx context.Context) error {
		dbErr := pgDb.Ping(ctx)
		redisErr := redisClient.Ping(ctx).Err()

		if dbErr == nil && redisErr == nil {
			return nil
		} else {
			return errors.Join(dbErr, redisErr)
		}
	})

	return web.StartGRPCServerAndWaitForShutdown(ctx, grpcServer, lis, logger)
}

func startRabbitMQConsumer(ctx context.Context, logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, querier repository.Querier) error {
	orderCreatedConsumer := events.NewOrderCreatedConsumer(logger, rabbitmq, querier)
	logger.Info("starting OrderCreatedConsumer")

	if err := orderCreatedConsumer.Start(ctx); err != nil {
		return fmt.Errorf("OrderCreatedConsumer failed: %w", err)
	}

	logger.Info("OrderCreatedConsumer stopped gracefully")
	return nil
}
