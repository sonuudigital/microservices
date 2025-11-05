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
	"github.com/sonuudigital/microservices/cart-service/internal/clients"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/web"
	"github.com/sonuudigital/microservices/shared/web/health"
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

	startGRPCServer(pgDb, redisClient, logger)

}

func initializeRedisClient() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is not set")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}

func startGRPCServer(pgDb *pgxpool.Pool, redisClient *redis.Client, logger logs.Logger) {
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
	cartServer := grpc_server.NewGRPCServer(queries, productClient, redisClient, logger)
	cartv1.RegisterCartServiceServer(grpcServer, cartServer)

	health.StartGRPCHealthCheckService(grpcServer, "cart-service", func(ctx context.Context) error {
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
