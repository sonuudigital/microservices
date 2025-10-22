package grpc

import (
	"context"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) AuthorizeUser(ctx context.Context, req *userv1.AuthorizeUserRequest) (*userv1.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)

	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	match, err := argon2id.ComparePasswordAndHash(password, user.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compare password: %v", err)
	}
	if !match {
		return nil, status.Errorf(codes.Unauthenticated, "invalid email or password")
	}

	return toGRPCUser(user), nil
}
