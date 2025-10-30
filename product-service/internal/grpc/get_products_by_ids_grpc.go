package grpc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetProductsByIDs(ctx context.Context, req *productv1.GetProductsByIDsRequest) (*productv1.GetProductsByIDsResponse, error) {
	if len(req.Ids) == 0 {
		return &productv1.GetProductsByIDsResponse{Products: []*productv1.Product{}}, nil
	}

	products, missedIDs := s.getProductsFromCache(ctx, req.Ids)

	if len(missedIDs) > 0 {
		dbProducts, err := s.getProductsFromDB(ctx, missedIDs)
		if err != nil {
			return nil, err
		}

		products = append(products, dbProducts...)
		s.cacheProducts(dbProducts)
	}

	return &productv1.GetProductsByIDsResponse{Products: products}, nil
}

func (s *GRPCServer) getProductsFromCache(ctx context.Context, ids []string) ([]*productv1.Product, []string) {
	products := make([]*productv1.Product, 0, len(ids))
	missedIDs := make([]string, 0)

	pipe := s.redisClient.Pipeline()
	for _, id := range ids {
		pipe.HGetAll(ctx, productCachePrefix+id)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return products, ids
	}

	for i, cmd := range cmds {
		data, err := cmd.(*redis.MapStringStringCmd).Result()
		if err == nil && len(data) > 0 {
			product, err := mapToProduct(data)
			if err == nil {
				products = append(products, product)
			}
		} else {
			missedIDs = append(missedIDs, ids[i])
		}
	}

	return products, missedIDs
}

func (s *GRPCServer) getProductsFromDB(ctx context.Context, missedIDs []string) ([]*productv1.Product, error) {
	pgUUIDs, err := s.convertToUUIDs(missedIDs)
	if err != nil {
		return nil, err
	}

	dbProducts, err := s.queries.GetProductsByIDs(ctx, pgUUIDs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get products: %v", err)
	}

	products := make([]*productv1.Product, 0, len(dbProducts))
	for _, dbProduct := range dbProducts {
		products = append(products, toGRPCProduct(dbProduct))
	}

	return products, nil
}

func (s *GRPCServer) convertToUUIDs(ids []string) ([]pgtype.UUID, error) {
	pgUUIDs := make([]pgtype.UUID, len(ids))
	for i, idStr := range ids {
		var uid pgtype.UUID
		if err := uid.Scan(idStr); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid product id format: %s", idStr)
		}
		pgUUIDs[i] = uid
	}
	return pgUUIDs, nil
}

func (s *GRPCServer) cacheProducts(products []*productv1.Product) {
	for _, product := range products {
		go func(p *productv1.Product) {
			ctx, cancel := context.WithTimeout(context.Background(), cacheContextTimeout)
			defer cancel()

			cacheKey := productCachePrefix + p.Id
			productMap := productToMap(p)
			pipe := s.redisClient.Pipeline()
			pipe.HSet(ctx, cacheKey, productMap)
			pipe.Expire(ctx, cacheKey, cacheExpirationTime)
			pipe.Exec(ctx)
		}(product)
	}
}
