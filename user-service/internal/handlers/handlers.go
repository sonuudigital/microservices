package handlers

import (
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
)

const (
	requestTimeoutMsg      = "Request Timeout"
	internalServerErrorMsg = "Internal Server Error"
)

type Handler struct {
	queries repository.Querier
	logger  logs.Logger
}

func NewHandler(queries repository.Querier, logger logs.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  logger,
	}
}
