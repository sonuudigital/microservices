package grpc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) ClearCart(ctx context.Context, req *cartv1.ClearCartRequest) (*emptypb.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	if err := s.queries.ClearCartProductsByUserID(ctx, userUUID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clear cart: %v", err)
	}

	return &emptypb.Empty{}, nil
}
