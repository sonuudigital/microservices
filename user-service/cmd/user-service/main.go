package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"shared/logs"
	"syscall"
	"time"
	"user-service/internal/database"
	"user-service/internal/server"

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

	pgDb, err := database.InitializePostgresDB()
	if err != nil {
		logger.Error("error connecting to database", "error", err)
		os.Exit(1)
	}

	logger.Info("database connected successfully")

	srv := server.InitializeServer(pgDb, logger)
	logger.Info("server initialized successfully")

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
	logger.Info("shutdown complete")
}
