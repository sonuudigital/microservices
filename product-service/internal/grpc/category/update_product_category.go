package category

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) UpdateProductCategory(ctx context.Context, req *product_categoriesv1.UpdateProductCategoryRequest) (*emptypb.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "category ID is required")
	}
	var categoryUUID pgtype.UUID
	if err := categoryUUID.Scan(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category ID: %v", err)
	}

	var description pgtype.Text
	if err := description.Scan(req.Description); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid description: %v", err)
	}

	arg := repository.UpdateProductCategoryParams{
		ID:          categoryUUID,
		Name:        req.Name,
		Description: description,
	}

	err := s.queries.UpdateProductCategory(ctx, arg)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			return nil, status.Errorf(codes.NotFound, "product category not found: %v", err)
		case pgx.ErrTooManyRows:
			return nil, status.Errorf(codes.Internal, "multiple product categories found with the same ID: %v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to update product category: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}
