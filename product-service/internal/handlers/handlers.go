package handlers

import (
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	invalidProductIDTitleMsg = "Invalid Product ID"
	invalidProductIDBodyMsg  = "invalid product id"

	productNotFoundTitleMsg = "Product Not Found"
	productNotFoundBodyMsg  = "product not found"

	requestTimeoutTitleMsg      = "Request Timeout"
	internalServerErrorTitleMsg = "Internal Server Error"
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

type ProductRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	Code          string  `json:"code"`
	StockQuantity int32   `json:"stockQuantity"`
}
