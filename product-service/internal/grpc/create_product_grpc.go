package grpc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	params := repository.CreateProductParams{
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		StockQuantity: req.StockQuantity,
	}
	if err := params.Price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid price: %v", err)
	}

	product, err := s.queries.CreateProduct(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	return toGRPCProduct(product), nil
}
