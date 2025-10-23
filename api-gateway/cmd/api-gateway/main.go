package main

import (
	"os"
	"strconv"
	"time"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/api-gateway/internal/router"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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

	userServiceClient := initializeUserServiceClient(logger)
	productServiceClient := initializeProductServiceClient(logger)
	cartServiceClient := initializeCartServiceClient(logger)

	authHandler := handlers.NewAuthHandler(logger, jwtManager, userServiceClient)

	mux, err := router.New(authHandler, jwtManager, logger, userServiceClient, productServiceClient, cartServiceClient)
	if err != nil {
		logger.Error("failed to configure routes", "error", err)
		os.Exit(1)
	}
	logger.Info("routes configured successfully")

	srv, err := web.InitializeServer(os.Getenv("PORT"), mux, logger)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	logger.Info("server initialized successfully", "port", os.Getenv("PORT"))
	web.StartServerAndWaitForShutdown(srv, logger)
}

func initializeUserServiceClient(logger logs.Logger) userv1.UserServiceClient {
	userServiceURL := os.Getenv("USER_SERVICE_GRPC_URL")
	if userServiceURL == "" {
		logger.Error("USER_SERVICE_GRPC_URL not found in environment variables")
		os.Exit(1)
	}

	conn, err := grpc.NewClient(userServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to connect to user service", "error", err)
		os.Exit(1)
	}

	logger.Info("connected to user service", "address", userServiceURL)

	return userv1.NewUserServiceClient(conn)
}

func initializeProductServiceClient(logger logs.Logger) productv1.ProductServiceClient {
	productServiceURL := os.Getenv("PRODUCT_SERVICE_GRPC_URL")
	if productServiceURL == "" {
		logger.Error("PRODUCT_SERVICE_GRPC_URL not found in environment variables")
		os.Exit(1)
	}

	conn, err := grpc.NewClient(productServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to connect to product service", "error", err)
		os.Exit(1)
	}

	logger.Info("connected to product service", "address", productServiceURL)

	return productv1.NewProductServiceClient(conn)
}

func initializeCartServiceClient(logger logs.Logger) cartv1.CartServiceClient {
	cartServiceURL := os.Getenv("CART_SERVICE_GRPC_URL")
	if cartServiceURL == "" {
		logger.Error("CART_SERVICE_GRPC_URL not found in environment variables")
		os.Exit(1)
	}

	conn, err := grpc.NewClient(cartServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to connect to cart service", "error", err)
		os.Exit(1)
	}

	logger.Info("connected to cart service", "address", cartServiceURL)

	return cartv1.NewCartServiceClient(conn)
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
