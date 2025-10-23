package main

import (
	"fmt"
	"net"
	"os"

	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/web"
	grpc_server "github.com/sonuudigital/microservices/user-service/internal/grpc"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
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

	grpcPort := os.Getenv("USER_SERVICE_GRPC_PORT")
	if grpcPort == "" {
		logger.Error("USER_SERVICE_GRPC_PORT is not set")
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		logger.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	queries := repository.New(pgDb)
	grpcServer := grpc.NewServer()
	userServer := grpc_server.NewGRPCServer(queries, logger)
	userv1.RegisterUserServiceServer(grpcServer, userServer)

	web.StartGRPCServerAndWaitForShutdown(grpcServer, lis, logger)
}
