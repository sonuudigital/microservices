package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/joho/godotenv"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/payment-service/internal/grpc/payment"
	"github.com/sonuudigital/microservices/shared/logs"
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
	startGRPCServer(logger)
}

func startGRPCServer(logger logs.Logger) {
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
	paymentGrpcServer := payment.New(logger)
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentGrpcServer)

	health.StartGRPCHealthCheckService(grpcServer, "payment-service", func(ctx context.Context) error {
		logger.Info("service is healthy and serving")
		return nil
	})

	web.StartGRPCServerAndWaitForShutdown(grpcServer, lis, logger)
}
