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

	var categoryUUID pgtype.UUID
	if err := categoryUUID.Scan(req.CategoryId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category ID: %v", err)
	}

	var price pgtype.Numeric
	if err := price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid price: %v", err)
	}

	params := repository.CreateProductParams{
		CategoryID:    categoryUUID,
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		Price:         price,
		StockQuantity: req.StockQuantity,
	}

	product, err := s.queries.CreateProduct(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	grpcProduct := toGRPCProduct(product)

	go func() {
		cacheKey := productCachePrefix + grpcProduct.Id
		if err := s.cacheProduct(grpcProduct.Id, grpcProduct); err != nil {
			s.logger.Error("failed to cache product", "key", cacheKey, "error", err)
		}
	}()

	return grpcProduct, nil
}
