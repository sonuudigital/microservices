package grpc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	redisCartPrefix     = "cart:"
	redisCacheTTL       = time.Minute * 15
	redisContextTimeout = time.Second * 3
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type ProductFetcher interface {
	GetProductsByIDs(ctx context.Context, ids []string) (map[string]Product, error)
}

type GRPCServer struct {
	cartv1.UnimplementedCartServiceServer
	queries        repository.Querier
	productFetcher ProductFetcher
	redisClient    *redis.Client
	logger         logs.Logger
}

func NewGRPCServer(queries repository.Querier, productFetcher ProductFetcher, redisClient *redis.Client, logger logs.Logger) *GRPCServer {
	return &GRPCServer{
		queries:        queries,
		productFetcher: productFetcher,
		redisClient:    redisClient,
		logger:         logger,
	}
}

func (s *GRPCServer) deleteCartCache(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), redisContextTimeout)
	defer cancel()

	cacheKey := redisCartPrefix + userID
	if err := s.redisClient.Del(ctx, cacheKey).Err(); err != nil {
		s.logger.Error("failed to delete cart cache", "userID", userID, "error", err)
		return fmt.Errorf("failed to delete cart cache for user %s: %w", userID, err)
	}

	return nil
}

func (s *GRPCServer) getOrCreateCartByUserID(ctx context.Context, userUUID pgtype.UUID) (repository.Cart, bool, error) {
	s.logger.Debug("attempting to get or create cart", "userId", userUUID.String())

	cart, err := s.queries.GetCartByUserID(ctx, userUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.logger.Debug("cart not found, creating new one", "userId", userUUID.String())
			newCart, _, createErr := s.createCart(ctx, userUUID)
			return newCart, true, createErr
		}
		s.logger.Error("failed to get cart by user id", "userId", userUUID.String(), "error", err)
		return repository.Cart{}, false, fmt.Errorf("failed to get cart by user id: %w", err)
	}

	s.logger.Debug("cart found", "cartId", cart.ID.String(), "userId", userUUID.String(), "createdAt", cart.CreatedAt.Time)

	expired, err := s.cartIsExpired(cart)
	if err != nil {
		s.logger.Error("failed to check if cart is expired", "cartId", cart.ID.String(), "error", err)
		return repository.Cart{}, false, fmt.Errorf("failed to check if cart is expired: %w", err)
	}

	if expired {
		s.logger.Warn("cart has expired, deleting and creating new one", "cartId", cart.ID.String(), "userId", userUUID.String())
		newCart, _, deleteErr := s.deleteExpiredCartAndCreateNewOne(ctx, userUUID)
		go s.deleteCartCache(userUUID.String())
		return newCart, true, deleteErr
	}

	return cart, false, nil
}

func (s *GRPCServer) createCart(ctx context.Context, userUUID pgtype.UUID) (repository.Cart, bool, error) {
	newCart, createErr := s.queries.CreateCart(ctx, userUUID)
	if createErr != nil {
		return repository.Cart{}, true, fmt.Errorf("failed to create a new cart: %w", createErr)
	}
	return newCart, true, nil
}

func (s *GRPCServer) deleteExpiredCartAndCreateNewOne(ctx context.Context, userUUID pgtype.UUID) (repository.Cart, bool, error) {
	if err := s.queries.DeleteCartByUserID(ctx, userUUID); err != nil {
		return repository.Cart{}, true, fmt.Errorf("failed to delete expired cart: %w", err)
	}

	return s.createCart(ctx, userUUID)
}

func (s *GRPCServer) cartIsExpired(cart repository.Cart) (bool, error) {
	cartTTLHours := os.Getenv("CART_TTL_HOURS")
	if cartTTLHours == "" {
		s.logger.Warn("CART_TTL_HOURS not set, defaulting to 168 hours")
		cartTTLHours = "168"
	}

	ttlHours, err := strconv.Atoi(cartTTLHours)
	if err != nil {
		return false, fmt.Errorf("invalid CART_TTL_HOURS value: %w", err)
	}

	if ttlHours == 0 {
		return false, nil
	}

	if !cart.CreatedAt.Valid {
		s.logger.Error("cart has invalid created_at timestamp", "cartId", cart.ID.String())
		return false, fmt.Errorf("cart has invalid created_at timestamp")
	}

	expirationTime := cart.CreatedAt.Time.Add(time.Duration(ttlHours) * time.Hour)
	isExpired := time.Now().After(expirationTime)

	s.logger.Debug(
		"checking cart expiration",
		"cartId", cart.ID.String(),
		"createdAt", cart.CreatedAt.Time,
		"expirationTime", expirationTime,
		"isExpired", isExpired,
		"ttlHours", ttlHours,
	)

	return isExpired, nil
}
