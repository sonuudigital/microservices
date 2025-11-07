package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/postgres"
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
}
