package grpc

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*emptypb.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var uid pgtype.UUID
	if err := uid.Scan(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id format: %s", req.Id)
	}

	_, err := s.queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}

	if err := s.queries.DeleteProduct(ctx, uid); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete product: %v", err)
	}

	cacheKey := productCachePrefix + req.Id
	ctx, cancel := context.WithTimeout(context.Background(), cacheContextTimeout)
	defer cancel()
	if err := s.redisClient.Del(ctx, cacheKey).Err(); err != nil {
		s.logger.Error("failed to delete product cache after deletion", "key", cacheKey, "error", err)
	}

	return &emptypb.Empty{}, nil
}
