package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alexedwards/argon2id"
	"github.com/go-redis/redismock/v9"
	"github.com/jackc/pgx/v5"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	grpc_server "github.com/sonuudigital/microservices/user-service/internal/grpc"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthorizeUser(t *testing.T) {
	password := "password"
	hashedPassword, _ := argon2id.CreateHash(password, argon2id.DefaultParams)
	req := &userv1.AuthorizeUserRequest{
		Email:    testEmail,
		Password: password,
	}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())
		mockQuerier.On("GetUserByEmail", mock.Anything, testEmail).
			Return(repository.User{
				Email:    testEmail,
				Password: hashedPassword,
			}, nil).Once()

		res, err := server.AuthorizeUser(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, testEmail, res.Email)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())
		mockQuerier.On("GetUserByEmail", mock.Anything, testEmail).
			Return(repository.User{}, pgx.ErrNoRows).Once()

		res, err := server.AuthorizeUser(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Wrong Password", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())
		mockQuerier.On("GetUserByEmail", mock.Anything, testEmail).
			Return(repository.User{
				Email:    testEmail,
				Password: hashedPassword,
			}, nil).Once()

		wrongReq := &userv1.AuthorizeUserRequest{
			Email:    testEmail,
			Password: "wrong-password",
		}
		res, err := server.AuthorizeUser(context.Background(), wrongReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, _ := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())
		mockQuerier.On("GetUserByEmail", mock.Anything, testEmail).
			Return(repository.User{}, errors.New("db error")).Once()

		res, err := server.AuthorizeUser(context.Background(), req)

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

		res, err := server.AuthorizeUser(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "GetUserByEmail", mock.Anything, mock.Anything)
	})
}
