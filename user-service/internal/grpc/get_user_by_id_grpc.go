package grpc

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetUserByID(ctx context.Context, req *userv1.GetUserByIDRequest) (*userv1.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	var uid pgtype.UUID
	if err := uid.Scan(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id format: %s", req.Id)
	}

	user, err := s.queries.GetUserByID(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return toGRPCUser(user), nil
}
