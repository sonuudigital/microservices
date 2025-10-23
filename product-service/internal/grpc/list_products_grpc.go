package grpc

import (
	"context"

	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	params := repository.ListProductsPaginatedParams{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	products, err := s.queries.ListProductsPaginated(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list products: %v", err)
	}

	grpcProducts := make([]*productv1.Product, len(products))
	for i, p := range products {
		grpcProducts[i] = toGRPCProduct(p)
	}

	return &productv1.ListProductsResponse{Products: grpcProducts}, nil
}
