package grpc

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	cacheKey := productCachePrefix + req.Id
	cachedData, err := s.redisClient.HGetAll(ctx, cacheKey).Result()
	if err == nil && len(cachedData) > 0 {
		product, err := mapToProduct(cachedData)
		if err == nil {
			return product, nil
		}
	}

	var uid pgtype.UUID
	if err := uid.Scan(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id format: %s", req.Id)
	}

	product, err := s.queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}

	grpcProduct := toGRPCProduct(product)

	go func() {
		if err := s.cacheProduct(req.Id, grpcProduct); err != nil {
			s.logger.Error("failed to cache product", "key", cacheKey, "error", err)
		}
	}()

	return grpcProduct, nil
}
