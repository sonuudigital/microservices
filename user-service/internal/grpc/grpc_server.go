package grpc

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	redisUserKeyPrefix          = "user:"
	redisEmailToUserIDKeyPrefix = "user_email_to_id:"
	redisUserExpirationTime     = time.Hour * 24
	redisContextTimeout         = time.Second * 3
)

type GRPCServer struct {
	userv1.UnimplementedUserServiceServer
	queries     repository.Querier
	redisClient *redis.Client
	logger      logs.Logger
}

func NewGRPCServer(queries repository.Querier, redisClient *redis.Client, logger logs.Logger) *GRPCServer {
	return &GRPCServer{
		queries:     queries,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (s *GRPCServer) checkUserCache(ctx context.Context, userID string) (*CachedUser, error) {
	ctx, cancel := context.WithTimeout(ctx, redisContextTimeout)
	defer cancel()

	cacheKey := redisUserKeyPrefix + userID
	data, err := s.redisClient.HGetAll(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	user, err := mapToUser(data)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *GRPCServer) checkUserIDByEmailCache(ctx context.Context, email string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, redisContextTimeout)
	defer cancel()

	cacheKey := redisEmailToUserIDKeyPrefix + email
	return s.redisClient.Get(ctx, cacheKey).Result()
}

func (s *GRPCServer) cacheUser(user *userv1.User, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), redisContextTimeout)
	defer cancel()

	data := userToMap(user, hashedPassword)

	cacheKey := redisUserKeyPrefix + user.Id
	pipe := s.redisClient.Pipeline()
	pipe.HSet(ctx, cacheKey, data)
	pipe.Expire(ctx, cacheKey, redisUserExpirationTime)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *GRPCServer) cacheUserEmailToID(email, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), redisContextTimeout)
	defer cancel()

	cacheKey := redisEmailToUserIDKeyPrefix + email
	return s.redisClient.Set(ctx, cacheKey, userID, redisUserExpirationTime).Err()
}

func toGRPCUser(u repository.User) *userv1.User {
	var updatedAt *timestamppb.Timestamp
	if u.UpdatedAt.Valid {
		updatedAt = timestamppb.New(u.UpdatedAt.Time)
	}

	return &userv1.User{
		Id:        u.ID.String(),
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: timestamppb.New(u.CreatedAt.Time),
		UpdatedAt: updatedAt,
	}
}
