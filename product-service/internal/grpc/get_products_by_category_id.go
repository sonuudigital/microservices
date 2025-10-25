package grpc

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetProductsByCategoryID(ctx context.Context, req *productv1.GetProductsByCategoryIDRequest) (*productv1.GetProductsByCategoryIDResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var uuid pgtype.UUID
	if err := uuid.Scan(req.CategoryId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category id format: %s", req.CategoryId)
	}

	products, err := s.queries.GetProductsByCategoryID(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "no products found for category id: %s", req.CategoryId)
		}
		return nil, status.Errorf(codes.Internal, "failed to get products by category id: %v", err)
	}

	return &productv1.GetProductsByCategoryIDResponse{Products: toGRPCProducts(products)}, nil
}
