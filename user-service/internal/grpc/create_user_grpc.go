package grpc

import (
	"context"
	"strings"

	"github.com/alexedwards/argon2id"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	hashedPassword, err := argon2id.CreateHash(strings.TrimSpace(req.Password), argon2id.DefaultParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	params := repository.CreateUserParams{
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		Password: hashedPassword,
	}

	user, err := s.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	grpcUser := toGRPCUser(user)
	go s.cacheUser(grpcUser, hashedPassword)
	return grpcUser, nil
}
