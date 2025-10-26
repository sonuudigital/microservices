package main

import (
	"fmt"
	"net"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/grpc/category"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
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

	startGRPCServer(pgDb, logger)
}

func startGRPCServer(pgDb *pgxpool.Pool, logger logs.Logger) {
	grpcPort := os.Getenv("PRODUCT_SERVICE_GRPC_PORT")
	if grpcPort == "" {
		logger.Error("PRODUCT_SERVICE_GRPC_PORT is not set")
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		logger.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	queries := repository.New(pgDb)
	grpcServer := grpc.NewServer()

	categoryServer := category.New(queries)
	productServer := grpc_server.NewServer(queries)

	product_categoriesv1.RegisterProductCategoriesServiceServer(grpcServer, categoryServer)
	productv1.RegisterProductServiceServer(grpcServer, productServer)

	web.StartGRPCServerAndWaitForShutdown(grpcServer, lis, logger)
}
