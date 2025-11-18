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
	"github.com/joho/godotenv"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/order-service/internal/events/consumers"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/clients"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/order"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	postgres_repo "github.com/sonuudigital/microservices/order-service/internal/repository/postgres"
	"github.com/sonuudigital/microservices/shared/events/worker"
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

	grpcClients, err := initializegRPCClients()
	if err != nil {
		logger.Error("failed to initialize gRPC clients", "error", err)
		os.Exit(1)
	}

	rabbitmq, err := initializeRabbitMQ(logger)
	if err != nil {
		logger.Error("failed to initialize RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbitmq.Close()

	initializeServicesAndWaitForShutdown(logger, rabbitmq, pgDb, grpcClients)
}

func initializeServicesAndWaitForShutdown(logger logs.Logger, rabbitmq *rabbitmq.RabbitMQ, pgDb *pgxpool.Pool, grpcClients *clients.Clients) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(ctx)

	orderRepo := repository.NewPostgreSQLOrderRepository(pgDb)

	mrPollInterval, mrBatchSize, err := getMessageRelayerConfigFromEnv()
	if err != nil {
		logger.Error("failed to get message relayer config", "error", err)
		os.Exit(1)
	}
	go worker.NewOutboxEventMessageRelayer(
		logger,
		rabbitmq,
		postgres_repo.NewOutboxEventMessageRelayerRepository(pgDb),
		mrPollInterval,
		mrBatchSize,
	).Start(ctx)

	g.Go(func() error {
		stockUpdateFailedConsumer := consumers.NewStockUpdateFailedConsumer(logger, orderRepo, rabbitmq)
		logger.Info("starting StockUpdateFailedConsumer")

		if err := stockUpdateFailedConsumer.Start(gCtx); err != nil {
			return fmt.Errorf("StockUpdateFailedConsumer failed: %w", err)
		}

		logger.Info("StockUpdateFailedConsumer stopped gracefully")
		return nil
	})

	g.Go(func() error {
		return startGRPCServer(gCtx, logger, pgDb, grpcClients, orderRepo)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}

	logger.Info("application shut down gracefully")
}

func startGRPCServer(ctx context.Context, logger logs.Logger, pgDb *pgxpool.Pool, grpcClients *clients.Clients, orderRepo *repository.PostgreSQLOrderRepository) error {
	gRPCPort := os.Getenv("ORDER_SERVICE_GRPC_PORT")
	if gRPCPort == "" {
		return fmt.Errorf("ORDER_SERVICE_GRPC_PORT is not set")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", gRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	grpcServer := grpc.NewServer()
	orderGrpcServer := order.New(logger, orderRepo, grpcClients)
	orderv1.RegisterOrderServiceServer(grpcServer, orderGrpcServer)

	health.StartGRPCHealthCheckService(grpcServer, "order-service", func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*3)
		defer cancel()

		if err := pgDb.Ping(ctx); err != nil {
			logger.Info("service is not healthy", "error", err)
			return err
		} else {
			logger.Info("service is healthy and serving")
			return nil
		}
	})

	return web.StartGRPCServerAndWaitForShutdown(ctx, grpcServer, lis, logger)
}

func initializegRPCClients() (*clients.Clients, error) {
	cartServiceURL := os.Getenv("CART_SERVICE_GRPC_URL")
	if cartServiceURL == "" {
		return nil, fmt.Errorf("CART_SERVICE_GRPC_URL is not set")
	}

	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_GRPC_URL")
	if paymentServiceURL == "" {
		return nil, fmt.Errorf("PAYMENT_SERVICE_GRPC_URL is not set")
	}

	userServiceURL := os.Getenv("USER_SERVICE_GRPC_URL")
	if userServiceURL == "" {
		return nil, fmt.Errorf("USER_SERVICE_GRPC_URL is not set")
	}

	clientsURL := clients.NewClienstURL(cartServiceURL, paymentServiceURL, userServiceURL)
	grpcClients, err := clients.NewClients(clientsURL)
	if err != nil {
		return nil, err
	}

	return grpcClients, nil
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
