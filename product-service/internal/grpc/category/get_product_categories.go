package category

import (
	"context"

	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) GetProductCategories(ctx context.Context, emptypb *emptypb.Empty) (*product_categoriesv1.GetProductCategoriesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	productCategories, err := s.queries.GetProductCategories(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get product categories: %v", err)
	}

	return &product_categoriesv1.GetProductCategoriesResponse{Categories: toGrpcProductCategories(productCategories)}, nil
}
