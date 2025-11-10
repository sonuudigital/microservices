package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/payment-service/internal/grpc/payment"
	"github.com/sonuudigital/microservices/payment-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
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

	startGRPCServer(logger, pgDb)
}

func startGRPCServer(logger logs.Logger, pgDb *pgxpool.Pool) {
	grpcPort := os.Getenv("PAYMENT_SERVICE_GRPC_PORT")
	if grpcPort == "" {
		logger.Error("PAYMENT_SERVICE_GRPC_PORT is not set")
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		logger.Error("failed to listen on gRPC port", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	paymentGrpcServer := payment.New(logger, repository.New(pgDb))
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentGrpcServer)

	health.StartGRPCHealthCheckService(grpcServer, "payment-service", func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*3)
		defer cancel()

		if err := pgDb.Ping(ctx); err == nil {
			logger.Info("service is healthy and serving")
			return nil
		} else {
			logger.Error("service is not healthy", "error", err)
			return err
		}
	})

	web.StartGRPCServerAndWaitForShutdown(context.Background(), grpcServer, lis, logger)
}
