package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sonuudigital/microservices/product-service/internal/router"
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

	mux := router.ConfigRoutes(pgDb, logger)

	port := os.Getenv("PORT")
	srv, err := web.InitializeServer(port, mux, logger)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	logger.Info("server initialized successfully", "port", port)
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
