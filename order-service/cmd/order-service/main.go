package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/order-service/internal/events/worker"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/clients"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/order"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
	"github.com/sonuudigital/microservices/shared/web"
	"github.com/sonuudigital/microservices/shared/web/health"
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

	grpcClients, err := initializegRPCClients(logger)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orderRepo := repository.NewPostgreSQLOrderRepository(pgDb)

	mrPollInterval, mrBatchSize, err := getMessageRelayerConfigFromEnv()
	if err != nil {
		logger.Error("failed to get message relayer config from env", "error", err)
		os.Exit(1)
	}
	mr := worker.New(logger, rabbitmq, orderRepo, mrPollInterval, mrBatchSize)
	go mr.Start(ctx)

	startGRPCServer(ctx, logger, pgDb, grpcClients, orderRepo)
}

func startGRPCServer(ctx context.Context, logger logs.Logger, pgDb *pgxpool.Pool, grpcClients *clients.Clients, orderRepo *repository.PostgreSQLOrderRepository) {
	gRPCPort := os.Getenv("ORDER_SERVICE_GRPC_PORT")
	if gRPCPort == "" {
		logger.Error("ORDER_SERVICE_GRPC_PORT is not set")
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", gRPCPort))
	if err != nil {
		logger.Error("failed to listen on gRPC port", "error", err)
		os.Exit(1)
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

	web.StartGRPCServerAndWaitForShutdown(ctx, grpcServer, lis, logger)
}

func initializegRPCClients(logger logs.Logger) (*clients.Clients, error) {
	cartServiceURL := os.Getenv("CART_SERVICE_GRPC_URL")
	if cartServiceURL == "" {
		logger.Error("CART_SERVICE_GRPC_URL is not set")
		os.Exit(1)
	}

	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_GRPC_URL")
	if paymentServiceURL == "" {
		logger.Error("PAYMENT_SERVICE_GRPC_URL is not set")
		os.Exit(1)
	}

	clientsURL := clients.NewClienstURL(cartServiceURL, paymentServiceURL)
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
