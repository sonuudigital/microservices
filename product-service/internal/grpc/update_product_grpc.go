package grpc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var uid pgtype.UUID
	if err := uid.Scan(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id format: %s", req.Id)
	}

	var categoryUUID pgtype.UUID
	if err := categoryUUID.Scan(req.CategoryId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category ID: %v", err)
	}

	params := repository.UpdateProductParams{
		ID:            uid,
		CategoryID:    categoryUUID,
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		StockQuantity: req.StockQuantity,
	}
	if err := params.Price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid price: %v", err)
	}

	product, err := s.queries.UpdateProduct(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update product: %v", err)
	}

	return toGRPCProduct(product), nil
}
