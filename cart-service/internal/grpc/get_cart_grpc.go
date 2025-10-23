package grpc

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetCart(ctx context.Context, req *cartv1.GetCartRequest) (*cartv1.GetCartResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var uid pgtype.UUID
	if err := uid.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	cart, err := s.queries.GetCartByUserID(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "cart not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get cart: %v", err)
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

		grpcCartProducts = append(grpcCartProducts, &cartv1.CartProduct{
			ProductId:   product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
			Quantity:    cp.Quantity,
		})
		totalPrice += product.Price * float64(cp.Quantity)
	}

	return &cartv1.GetCartResponse{
		Id:         cart.ID.String(),
		UserId:     cart.UserID.String(),
		Products:   grpcCartProducts,
		TotalPrice: totalPrice,
	}, nil
}
