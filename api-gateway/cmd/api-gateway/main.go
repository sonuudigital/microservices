package main

import (
	"api-gateway/internal/handlers"
	"api-gateway/internal/router"
	"api-gateway/internal/server"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"shared/auth"
	"shared/logs"
	"strconv"
	"syscall"
	"time"

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

	authHandler := handlers.NewAuthHandler(logger, jwtManager)

	mux, err := router.New(authHandler, jwtManager, logger)
	if err != nil {
		logger.Error("failed to configure routes", "error", err)
		os.Exit(1)
	}

	srv, err := server.InitializeServer(os.Getenv("PORT"), mux, logger)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	startServerAndWaitForShutdown(srv, logger)
}

func startServerAndWaitForShutdown(srv *http.Server, logger *logs.SlogLogger) {
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", "error", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	if err := srv.Shutdown(shCtx); err != nil {
		logger.Error("failed to shutdown server", "error", err)
	} else {
		logger.Info("shutdown complete")
	}
}

func initializeJWTManager(logger logs.Logger) *auth.JWTManager {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Error("jwt secret not found in environment variables")
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

	return auth.NewJWTManager(
		jwtSecret,
		jwtIssuer,
		jwtAudience,
		time.Duration(jwtExpirationMinutesInt)*time.Minute,
	)
}
