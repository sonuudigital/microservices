package main

import (
	"os"

	"github.com/sonuudigital/microservices/cart-service/internal/clients"
	"github.com/sonuudigital/microservices/cart-service/internal/router"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/web"

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

	mux := router.ConfigRoutes(pgDb, productClient, logger)

	port := os.Getenv("PORT")
	srv, err := web.InitializeServer(port, mux, logger)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	logger.Info("server initialized successfully", "port", port)
	web.StartServerAndWaitForShutdown(srv, logger)
}
