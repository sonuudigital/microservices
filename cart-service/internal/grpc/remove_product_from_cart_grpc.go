package grpc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) RemoveProductFromCart(ctx context.Context, req *cartv1.RemoveProductFromCartRequest) (*emptypb.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(req.ProductId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id format: %s", req.ProductId)
	}

	params := repository.RemoveProductFromCartParams{
		UserID:    userUUID,
		ProductID: productUUID,
	}

	if err := s.queries.RemoveProductFromCart(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove product from cart: %v", err)
	}

	return &emptypb.Empty{}, nil
}
