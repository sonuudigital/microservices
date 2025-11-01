package grpc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) DeleteCart(ctx context.Context, req *cartv1.DeleteCartRequest) (*emptypb.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var uid pgtype.UUID
	if err := uid.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.UserId)
	}

	_, wasRecreated, err := s.getOrCreateCartByUserID(ctx, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate cart: %v", err)
	}
	if wasRecreated {
		return &emptypb.Empty{}, nil
	}

	if err := s.queries.DeleteCartByUserID(ctx, uid); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete cart: %v", err)
	}

	go s.deleteCartCache(req.UserId)

	return &emptypb.Empty{}, nil
}
