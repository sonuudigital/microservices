package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
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

	startGRPCServer(logger, pgDb, grpcClients)
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

func startGRPCServer(logger logs.Logger, pgDb *pgxpool.Pool, grpcClients *clients.Clients) {
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
	orderGrpcServer := order.New(logger, repository.NewPostgreSQLOrderRepository(pgDb), grpcClients)
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

	web.StartGRPCServerAndWaitForShutdown(context.Background(), grpcServer, lis, logger)
}
