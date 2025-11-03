package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) AuthorizeUser(ctx context.Context, req *userv1.AuthorizeUserRequest) (*userv1.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)

	cachedUser, err := s.getUserFromCacheByEmail(ctx, email)
	if err == nil && cachedUser != nil {
		if err := s.verifyPassword(password, cachedUser.HashedPassword); err != nil {
			return nil, err
		}
		return cachedUser.User, nil
	}

	user, err := s.getUserFromDatabase(ctx, email)
	if err != nil {
		return nil, err
	}

	if err := s.verifyPassword(password, user.Password); err != nil {
		return nil, err
	}

	grpcUser := toGRPCUser(user)
	go s.cacheUserWithEmail(grpcUser, user.Password, email)

	return grpcUser, nil
}

func (s *GRPCServer) getUserFromCacheByEmail(ctx context.Context, email string) (*CachedUser, error) {
	userID, err := s.checkUserIDByEmailCache(ctx, email)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.logger.Info("email not found in secondary cache", "email", email)
			return nil, err
		}
		s.logger.Error("failed to check email to userID cache", "email", email, "error", err)
		return nil, err
	}

	cachedUser, err := s.checkUserCache(ctx, userID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.logger.Info("user not found in primary cache", "userID", userID)
			return nil, err
		}
		s.logger.Error("failed to check user cache", "userID", userID, "error", err)
		return nil, err
	}

	s.logger.Debug("user retrieved from cache", "email", email, "userID", userID)
	return cachedUser, nil
}

func (s *GRPCServer) getUserFromDatabase(ctx context.Context, email string) (repository.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return repository.User{}, status.Errorf(codes.Unauthenticated, "invalid email or password")
		}
		return repository.User{}, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	return user, nil
}

func (s *GRPCServer) verifyPassword(password, hashedPassword string) error {
	match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to compare password: %v", err)
	}
	if !match {
		return status.Errorf(codes.Unauthenticated, "invalid email or password")
	}
	return nil
}

func (s *GRPCServer) cacheUserWithEmail(user *userv1.User, hashedPassword, email string) {
	if err := s.cacheUser(user, hashedPassword); err != nil {
		s.logger.Error("failed to cache user", "userID", user.Id, "error", err)
	}

	if err := s.cacheUserEmailToID(email, user.Id); err != nil {
		s.logger.Error("failed to cache email to userID mapping", "email", email, "error", err)
	}
}
