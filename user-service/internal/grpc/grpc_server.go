package grpc

import (
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	userv1.UnimplementedUserServiceServer
	queries repository.Querier
	logger  logs.Logger
}

func NewGRPCServer(queries repository.Querier, logger logs.Logger) *GRPCServer {
	return &GRPCServer{
		queries: queries,
		logger:  logger,
	}
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
