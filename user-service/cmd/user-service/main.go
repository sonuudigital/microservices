package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/web"
	grpc_server "github.com/sonuudigital/microservices/user-service/internal/grpc"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

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

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	go func() {
		healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		if err := pgDb.Ping(context.Background()); err == nil {
			logger.Info("service is healthy and serving")
			healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_SERVING)
		} else {
			logger.Error("service is not healthy", "error", err)
		}
	}()

	web.StartGRPCServerAndWaitForShutdown(grpcServer, lis, logger)
}
