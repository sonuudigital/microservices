package main

import (
	"os"
	"strconv"
	"time"

	"github.com/sonuudigital/microservices/api-gateway/internal/clients"
	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	"github.com/sonuudigital/microservices/api-gateway/internal/router"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"golang.org/x/time/rate"

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

	logger.Info("starting api-gateway")

	jwtManager := initializeJWTManager(logger)
	rateLimiterMiddleware := initializeRateLimiterMiddleware(logger)

	if !verifyEnvironmentServiceURLs(logger) {
		os.Exit(1)
	}

	clients, err := clients.NewGRPCClient(clients.ClientURL{
		UserServiceURL:    os.Getenv("USER_SERVICE_GRPC_URL"),
		ProductServiceURL: os.Getenv("PRODUCT_SERVICE_GRPC_URL"),
		CartServiceURL:    os.Getenv("CART_SERVICE_GRPC_URL"),
	})
	if err != nil {
		logger.Error("failed to create gRPC clients", "error", err.Error())
		os.Exit(1)
	}

	handler, err := router.New(logger, jwtManager, rateLimiterMiddleware, clients)
	if err != nil {
		logger.Error("failed to configure routes", "error", err)
		os.Exit(1)
	}
	logger.Info("routes configured successfully")

	srv, err := web.InitializeServer(os.Getenv("PORT"), handler, logger)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	logger.Info("server initialized successfully", "port", os.Getenv("PORT"))
	web.StartServerAndWaitForShutdown(srv, logger)
}

func verifyEnvironmentServiceURLs(logger logs.Logger) bool {
	requiredEnvVars := []string{
		"USER_SERVICE_GRPC_URL",
		"PRODUCT_SERVICE_GRPC_URL",
		"CART_SERVICE_GRPC_URL",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			logger.Error(envVar + " not found in environment variables")
			return false
		}
	}

	return true
}

func initializeJWTManager(logger logs.Logger) *auth.JWTManager {
	jwtPrivateKeyPath := os.Getenv("JWT_PRIVATE_KEY_PATH")
	if jwtPrivateKeyPath == "" {
		logger.Error("jwt private key path not found in environment variables")
		os.Exit(1)
	}
	privateKey, err := os.ReadFile(jwtPrivateKeyPath)
	if err != nil {
		logger.Error("failed to read private key", "path", jwtPrivateKeyPath, "error", err)
		os.Exit(1)
	}

	jwtPublicKeyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
	if jwtPublicKeyPath == "" {
		logger.Error("jwt public key path not found in environment variables")
		os.Exit(1)
	}
	publicKey, err := os.ReadFile(jwtPublicKeyPath)
	if err != nil {
		logger.Error("failed to read public key", "path", jwtPublicKeyPath, "error", err)
		os.Exit(1)
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		logger.Error("jwt issuer not found in environment variables")
		os.Exit(1)
	}
	jwtAudience := os.Getenv("JWT_AUDIENCE")
	if jwtAudience == "" {
		logger.Error("jwt audience not found in environment variables")
		os.Exit(1)
	}
	jwtExpirationMinutes := os.Getenv("JWT_TTL_MINUTES")
	if jwtExpirationMinutes == "" {
		logger.Error("jwt expiration minutes not found in environment variables")
		os.Exit(1)
	}
	jwtExpirationMinutesInt, err := strconv.Atoi(jwtExpirationMinutes)
	if err != nil {
		logger.Error("invalid jwt expiration minutes", "error", err)
		os.Exit(1)
	}

	jwtManager, err := auth.NewJWTManager(
		privateKey,
		publicKey,
		jwtIssuer,
		jwtAudience,
		time.Duration(jwtExpirationMinutesInt)*time.Minute,
	)
	if err != nil {
		logger.Error("failed to create jwt manager", "error", err)
		os.Exit(1)
	}

	return jwtManager
}

func initializeRateLimiterMiddleware(logger logs.Logger) *middlewares.RateLimiterMiddleware {
	rateLimiterEnabled, err := strconv.ParseBool(os.Getenv("RATE_LIMITER_ENABLED"))
	if err != nil {
		logger.Info("rate limiter is disabled by default")
		rateLimiterEnabled = false
	}

	rps, err := strconv.ParseFloat(os.Getenv("RATE_LIMITER_RPS"), 64)
	if err != nil {
		logger.Info("rate limiter rps not found, using default 10")
		rps = 10
	}

	burst, err := strconv.Atoi(os.Getenv("RATE_LIMITER_BURST"))
	if err != nil {
		logger.Info("rate limiter burst not found, using default 20")
		burst = 20
	}

	logger.Info("rate limiter configured", "enabled", rateLimiterEnabled, "rps", rps, "burst", burst)

	return middlewares.NewRateLimiterMiddleware(logger, rate.Limit(rps), burst, rateLimiterEnabled)
}
