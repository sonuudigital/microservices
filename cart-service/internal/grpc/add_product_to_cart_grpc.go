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

	s.logger.Debug("AddProductToCart called", "userId", req.UserId, "productId", req.ProductId, "quantity", req.Quantity)

	var userUUID pgtype.UUID
	if err := userUUID.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	cart, wasCreated, err := s.getOrCreateCartByUserID(ctx, userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get or create cart: %v", err)
	}

	if wasCreated {
		existingProducts, countErr := s.queries.GetCartProductsByCartID(ctx, cart.ID)
		if countErr != nil {
			s.logger.Error("failed to check cart products after recreation", "cartId", cart.ID.String(), "error", countErr)
			return nil, status.Errorf(codes.Internal, "failed to verify cart state: %v", countErr)
		}

		if len(existingProducts) > 0 {
			s.logger.Warn("cart was recreated and had products, aborting add product operation", "userId", req.UserId, "cartId", cart.ID.String(), "productsCount", len(existingProducts))
			return nil, status.Errorf(codes.Aborted, "cart has expired and was cleared, please try again")
		}

		s.logger.Debug("new cart created for user", "userId", req.UserId, "cartId", cart.ID.String())
	} else {
		s.logger.Debug("using existing cart", "userId", req.UserId, "cartId", cart.ID.String())
	}

	productsMap, err := s.productFetcher.GetProductsByIDs(ctx, []string{req.ProductId})
	if err != nil {
		s.logger.Error("failed to fetch product", "productId", req.ProductId, "error", err)
		return nil, err
	}

	product, exists := productsMap[req.ProductId]
	if !exists {
		s.logger.Warn("product not found", "productId", req.ProductId)
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
		s.logger.Error("failed to add or update product in cart", "cartId", cart.ID.String(), "productId", req.ProductId, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to add or update product in cart: %v", err)
	}

	s.logger.Debug("product added to cart successfully", "userId", req.UserId, "cartId", cart.ID.String(), "productId", req.ProductId, "quantity", cartProduct.Quantity)

	go s.deleteCartCache(req.UserId)

	return &cartv1.AddProductToCartResponse{
		Id:        cartProduct.ID.String(),
		CartId:    cartProduct.CartID.String(),
		ProductId: cartProduct.ProductID.String(),
		Quantity:  cartProduct.Quantity,
		Price:     product.Price,
	}, nil
}
