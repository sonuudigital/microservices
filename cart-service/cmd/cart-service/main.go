package main

import (
	"fmt"
	"net"
	"os"

	"github.com/sonuudigital/microservices/cart-service/internal/clients"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/web"
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

	productServiceGrpcURL := os.Getenv("PRODUCT_SERVICE_GRPC_URL")
	if productServiceGrpcURL == "" {
		logger.Error("PRODUCT_SERVICE_GRPC_URL is not set")
		os.Exit(1)
	}
	productClient, err := clients.NewProductClient(productServiceGrpcURL, logger)
	if err != nil {
		logger.Error("failed to create product client", "error", err)
		os.Exit(1)
	}
	logger.Info("product client created successfully", "url", productServiceGrpcURL)

	grpcPort := os.Getenv("CART_SERVICE_GRPC_PORT")
	if grpcPort == "" {
		logger.Error("CART_SERVICE_GRPC_PORT is not set")
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		logger.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	queries := repository.New(pgDb)
	grpcServer := grpc.NewServer()
	cartServer := grpc_server.NewGRPCServer(queries, productClient, logger)
	cartv1.RegisterCartServiceServer(grpcServer, cartServer)

	web.StartGRPCServerAndWaitForShutdown(grpcServer, lis, logger)
}
