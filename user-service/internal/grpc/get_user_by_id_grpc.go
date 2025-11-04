package grpc

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetUserByID(ctx context.Context, req *userv1.GetUserByIDRequest) (*userv1.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	cachedUser, err := s.checkUserCache(ctx, req.Id)
	if err == nil {
		s.logger.Debug("user retrieved from cache", "userID", req.Id)
		return &userv1.User{
			Id:        cachedUser.Id,
			Username:  cachedUser.Username,
			Email:     cachedUser.Email,
			CreatedAt: cachedUser.CreatedAt,
			UpdatedAt: cachedUser.UpdatedAt,
		}, nil
	} else {
		if errors.Is(err, redis.Nil) {
			s.logger.Info("user not found in cache", "userID", req.Id)
		} else {
			s.logger.Error("failed to check user cache", "userID", req.Id, "error", err)
		}
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

	grpcUser := toGRPCUser(user)
	go s.cacheUser(grpcUser, user.Password)
	return grpcUser, nil
}
