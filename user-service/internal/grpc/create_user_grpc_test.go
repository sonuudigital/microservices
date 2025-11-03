package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	grpc_server "github.com/sonuudigital/microservices/user-service/internal/grpc"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateUser(t *testing.T) {
	req := &userv1.CreateUserRequest{
		Username: "testuser",
		Email:    testEmail,
		Password: "password",
	}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())
		mockQuerier.On("CreateUser", mock.Anything, mock.AnythingOfType("repository.CreateUserParams")).
			Return(repository.User{
				Username: req.Username,
				Email:    req.Email,
			}, nil).Once()

		res, err := server.CreateUser(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.Username, res.Username)
		assert.Equal(t, req.Email, res.Email)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())
		mockQuerier.On("CreateUser", mock.Anything, mock.AnythingOfType("repository.CreateUserParams")).
			Return(repository.User{}, errors.New("db error")).Once()

		res, err := server.CreateUser(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := server.CreateUser(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
	})
}
