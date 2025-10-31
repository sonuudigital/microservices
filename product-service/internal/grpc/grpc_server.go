package grpc

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	productCachePrefix  = "product:"
	cacheExpirationTime = 10 * time.Minute
	cacheContextTimeout = 2 * time.Second
)

type GRPCServer struct {
	productv1.UnimplementedProductServiceServer
	logger      logs.Logger
	queries     repository.Querier
	redisClient *redis.Client
}

func NewServer(logger logs.Logger, queries repository.Querier, redisClient *redis.Client) *GRPCServer {
	return &GRPCServer{
		logger:      logger,
		queries:     queries,
		redisClient: redisClient,
	}
}

func (s *GRPCServer) cacheProduct(id string, product *productv1.Product) error {
	cacheKey := productCachePrefix + id
	ctx, cancel := context.WithTimeout(context.Background(), cacheContextTimeout)
	defer cancel()

	productMap := productToMap(product)
	pipe := s.redisClient.Pipeline()
	pipe.HSet(ctx, cacheKey, productMap)
	pipe.Expire(ctx, cacheKey, cacheExpirationTime)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *GRPCServer) deleteProductCache(id string) error {
	cacheKey := productCachePrefix + id
	ctx, cancel := context.WithTimeout(context.Background(), cacheContextTimeout)
	defer cancel()

	return s.redisClient.Del(ctx, cacheKey).Err()
}

func toGRPCProduct(p repository.Product) *productv1.Product {
	var price float64
	if p.Price.Valid {
		v, err := p.Price.Value()
		if err == nil {
			if s, ok := v.(string); ok {
				price, _ = strconv.ParseFloat(s, 64)
			}
		}
	}

	var updatedAt *timestamppb.Timestamp
	if p.UpdatedAt.Valid {
		updatedAt = timestamppb.New(p.UpdatedAt.Time)
	}

	return &productv1.Product{
		Id:            p.ID.String(),
		CategoryId:    p.CategoryID.String(),
		Name:          p.Name,
		Description:   p.Description.String,
		Price:         price,
		StockQuantity: p.StockQuantity,
		CreatedAt:     timestamppb.New(p.CreatedAt.Time),
		UpdatedAt:     updatedAt,
	}
}

func ToGRPCProducts(products []repository.Product) []*productv1.Product {
	grpcProducts := make([]*productv1.Product, len(products))
	for i, p := range products {
		grpcProducts[i] = toGRPCProduct(p)
	}
	return grpcProducts
}
