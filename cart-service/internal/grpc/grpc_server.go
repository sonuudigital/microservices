package grpc

import (
	"context"

	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/logs"
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type ProductFetcher interface {
	GetProductsByIDs(ctx context.Context, ids []string) (map[string]Product, error)
}

type GRPCServer struct {
	cartv1.UnimplementedCartServiceServer
	queries        repository.Querier
	productFetcher ProductFetcher
	logger         logs.Logger
}

func NewGRPCServer(queries repository.Querier, productFetcher ProductFetcher, logger logs.Logger) *GRPCServer {
	return &GRPCServer{
		queries:        queries,
		productFetcher: productFetcher,
		logger:         logger,
	}
}
