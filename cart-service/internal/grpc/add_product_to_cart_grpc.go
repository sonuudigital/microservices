package grpc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) AddProductToCart(ctx context.Context, req *cartv1.AddProductToCartRequest) (*cartv1.AddProductToCartResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	cart, wasRecreated, err := s.getOrCreateCartByUserID(ctx, userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get or create cart: %v", err)
	}

	if wasRecreated {
		return nil, status.Errorf(codes.Aborted, "cart has expired and was cleared, please try again")
	}

	productsMap, err := s.productFetcher.GetProductsByIDs(ctx, []string{req.ProductId})
	if err != nil {
		return nil, err
	}

	product, exists := productsMap[req.ProductId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "product not found: %s", req.ProductId)
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(req.ProductId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id format: %s", req.ProductId)
	}

	priceNumeric := pgtype.Numeric{}
	if err := priceNumeric.Scan(fmt.Sprintf("%.2f", product.Price)); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to scan price to numeric: %v", err)
	}

	params := repository.AddOrUpdateProductInCartParams{
		CartID:    cart.ID,
		ProductID: productUUID,
		Quantity:  req.Quantity,
		Price:     priceNumeric,
	}

	cartProduct, err := s.queries.AddOrUpdateProductInCart(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add or update product in cart: %v", err)
	}

	return &cartv1.AddProductToCartResponse{
		Id:        cartProduct.ID.String(),
		CartId:    cartProduct.CartID.String(),
		ProductId: cartProduct.ProductID.String(),
		Quantity:  cartProduct.Quantity,
		Price:     product.Price,
	}, nil
}
