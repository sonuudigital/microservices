package category

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	allProductCategoriesCacheKey = "product_categories:all"
	cacheExpirationTime          = 1 * time.Hour
	cacheContextTimeout          = 2 * time.Second
)

type GRPCServer struct {
	product_categoriesv1.UnimplementedProductCategoriesServiceServer
	logger      logs.Logger
	queries     repository.Querier
	redisClient *redis.Client
}

func New(logger logs.Logger, queries repository.Querier, redisClient *redis.Client) *GRPCServer {
	return &GRPCServer{
		logger:      logger,
		queries:     queries,
		redisClient: redisClient,
	}
}

func (s *GRPCServer) deleteProductCategoriesCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), cacheContextTimeout)
	defer cancel()

	return s.redisClient.Del(ctx, allProductCategoriesCacheKey).Err()
}

func toGrpcProductCategory(pc repository.ProductCategory) *product_categoriesv1.ProductCategory {
	var updatedAt *timestamppb.Timestamp
	if pc.UpdatedAt.Valid {
		updatedAt = timestamppb.New(pc.UpdatedAt.Time)
	}

	return &product_categoriesv1.ProductCategory{
		Id:          pc.ID.String(),
		Name:        pc.Name,
		Description: pc.Description.String,
		CreatedAt:   timestamppb.New(pc.CreatedAt.Time),
		UpdatedAt:   updatedAt,
	}
}

func toGrpcProductCategories(categories []repository.ProductCategory) []*product_categoriesv1.ProductCategory {
	if len(categories) == 0 {
		return []*product_categoriesv1.ProductCategory{}
	}
	grpcCategories := make([]*product_categoriesv1.ProductCategory, len(categories))
	for i, pc := range categories {
		grpcCategories[i] = toGrpcProductCategory(pc)
	}
	return grpcCategories
}
