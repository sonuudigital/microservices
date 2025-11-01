package grpc

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetCart(ctx context.Context, req *cartv1.GetCartRequest) (*cartv1.GetCartResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	cachedResp, err := s.checkCartCache(req.UserId)
	if err != nil {
		s.logger.Error("failed to check cart cache", "userID", req.UserId, "error", err)
	} else if cachedResp != nil {
		return cachedResp, nil
	}

	var uid pgtype.UUID
	if err := uid.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	cart, _, err := s.getOrCreateCartByUserID(ctx, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get or create cart: %v", err)
	}

	cartProducts, err := s.queries.GetCartProductsByCartID(ctx, cart.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get cart products: %v", err)
	}

	productIDs := make([]string, 0, len(cartProducts))
	for _, cp := range cartProducts {
		productIDs = append(productIDs, cp.ProductID.String())
	}

	productsMap, err := s.productFetcher.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch products: %v", err)
	}

	grpcCartProducts := make([]*cartv1.CartProduct, 0, len(cartProducts))
	var totalPrice float64
	for _, cp := range cartProducts {
		productIDStr := cp.ProductID.String()
		product, exists := productsMap[productIDStr]
		if !exists {
			continue
		}

		priceFloat, err := cp.Price.Float64Value()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert price to float64: %v", err)
		}

		grpcCartProducts = append(grpcCartProducts, &cartv1.CartProduct{
			ProductId:   product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       priceFloat.Float64,
			Quantity:    cp.Quantity,
		})
		totalPrice += priceFloat.Float64 * float64(cp.Quantity)
	}

	grpcGetCartResponse := &cartv1.GetCartResponse{
		Id:         cart.ID.String(),
		UserId:     cart.UserID.String(),
		Products:   grpcCartProducts,
		TotalPrice: totalPrice,
	}

	go s.cacheGetCartResponse(req.UserId, grpcGetCartResponse)

	return grpcGetCartResponse, nil
}

func (s *GRPCServer) checkCartCache(userID string) (*cartv1.GetCartResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), redisContextTimeout)
	defer cancel()

	cacheKey := redisCartPrefix + userID
	jsonResp, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err != nil && len(jsonResp) == 0 {
		if err == redis.Nil {
			return nil, nil
		} else {
			return nil, err
		}
	}

	var resp *cartv1.GetCartResponse
	if err := json.Unmarshal([]byte(jsonResp), &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *GRPCServer) cacheGetCartResponse(userID string, resp *cartv1.GetCartResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), redisContextTimeout)
	defer cancel()

	cacheKey := redisCartPrefix + userID
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("failed to marshal get cart response for caching", "error", err)
		return err
	}

	if err := s.redisClient.Set(ctx, cacheKey, data, redisCacheTTL).Err(); err != nil {
		s.logger.Error("failed to set get cart response in redis cache", "error", err)
		return err
	}

	return nil
}
