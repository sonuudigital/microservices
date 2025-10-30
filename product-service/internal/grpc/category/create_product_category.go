package category

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) CreateProductCategory(ctx context.Context, req *product_categoriesv1.CreateProductCategoryRequest) (*product_categoriesv1.ProductCategory, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "category name is required")
	}

	arg := repository.CreateProductCategoryParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	}

	newCategory, err := s.queries.CreateProductCategory(ctx, arg)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			return nil, status.Errorf(codes.NotFound, "related resource not found: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to create product category: %v", err)
		}
	}

	go s.deleteProductCategoriesCache()

	return toGrpcProductCategory(newCategory), nil
}
