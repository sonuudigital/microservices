package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc_server "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/grpc/category"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
	"github.com/sonuudigital/microservices/shared/web"
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

	categoryServer := category.New(logger, queries, redisClient)
	productServer := grpc_server.NewServer(logger, queries, redisClient)

	product_categoriesv1.RegisterProductCategoriesServiceServer(grpcServer, categoryServer)
	productv1.RegisterProductServiceServer(grpcServer, productServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	go func() {
		healthServer.SetServingStatus("product-service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		dbErr := pgDb.Ping(ctx)
		redisErr := redisClient.Ping(ctx).Err()

		if dbErr == nil && redisErr == nil {
			logger.Info("service is healthy and serving")
			healthServer.SetServingStatus("product-service", grpc_health_v1.HealthCheckResponse_SERVING)
		} else {
			logger.Error("service is not healthy", "dbError", dbErr, "redisError", redisErr)
		}
	}()

	web.StartGRPCServerAndWaitForShutdown(grpcServer, lis, logger)
}
