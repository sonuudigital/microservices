package category

import (
	"context"
	"encoding/json"

	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) GetProductCategories(ctx context.Context, emptypb *emptypb.Empty) (*product_categoriesv1.GetProductCategoriesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	if cachedData, err := s.getProductCategoriesFromCache(ctx); err == nil {
		return &product_categoriesv1.GetProductCategoriesResponse{Categories: cachedData}, nil
	}

	bdProductCategories, err := s.getProductCategoriesFromDB(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get product categories: %v", err)
	}

	go s.cacheProductCategories(bdProductCategories)

	return &product_categoriesv1.GetProductCategoriesResponse{Categories: bdProductCategories}, nil
}

func (s *GRPCServer) getProductCategoriesFromCache(ctx context.Context) ([]*product_categoriesv1.ProductCategory, error) {
	cachedData, err := s.redisClient.Get(ctx, allProductCategoriesCacheKey).Result()
	if err != nil {
		s.logger.Error("failed to get product categories from cache", "error", err)
		return nil, err
	}

	productsCategoriesBd := make([]*product_categoriesv1.ProductCategory, 0)
	if err := json.Unmarshal([]byte(cachedData), &productsCategoriesBd); err != nil {
		s.logger.Error("failed to unmarshal cached product categories", "error", err)
		return nil, err
	}

	return productsCategoriesBd, nil
}

func (s *GRPCServer) getProductCategoriesFromDB(ctx context.Context) ([]*product_categoriesv1.ProductCategory, error) {
	bdProductCategories, err := s.queries.GetProductCategories(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get product categories: %v", err)
	}
	return toGrpcProductCategories(bdProductCategories), nil
}

func (s *GRPCServer) cacheProductCategories(categories []*product_categoriesv1.ProductCategory) {
	ctx, cancel := context.WithTimeout(context.Background(), cacheContextTimeout)
	defer cancel()

	data, err := json.Marshal(categories)
	if err != nil {
		s.logger.Error("failed to marshal product categories for caching", "error", err)
		return
	}

	s.redisClient.Set(ctx, allProductCategoriesCacheKey, data, cacheExpirationTime)
}
