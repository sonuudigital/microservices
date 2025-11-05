package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/web"
	"github.com/sonuudigital/microservices/shared/web/health"
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

	redisClient, err := initializeRedisClient()
	if err != nil {
		logger.Error("error connecting to redis", "error", err)
		os.Exit(1)
	}
	logger.Info("redis connected successfully")
	defer redisClient.Close()

	startGRPCServer(pgDb, redisClient, logger)
}

func initializeRedisClient() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is not set")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("could not ping redis: %w", err)
	}

	return client, nil
}

func startGRPCServer(pgDb *pgxpool.Pool, redisClient *redis.Client, logger logs.Logger) {
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
	userServer := grpc_server.NewGRPCServer(queries, redisClient, logger)
	userv1.RegisterUserServiceServer(grpcServer, userServer)

	health.StartGRPCHealthCheckService(grpcServer, "user-service", func(ctx context.Context) error {
		dbErr := pgDb.Ping(ctx)
		redisErr := redisClient.Ping(ctx).Err()

		if dbErr == nil && redisErr == nil {
			logger.Info("service is healthy and serving")
			return nil
		} else {
			errors := errors.Join(dbErr, redisErr)
			logger.Error("service is not healthy", "errors", errors)
			return errors
		}
	})

	web.StartGRPCServerAndWaitForShutdown(grpcServer, lis, logger)
}
