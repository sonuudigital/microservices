package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	grpc_server "github.com/sonuudigital/microservices/user-service/internal/grpc"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetUserByID(t *testing.T) {
	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(testUUID)

	req := &userv1.GetUserByIDRequest{Id: testUUID}

	t.Run("Success - Cache Miss", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())

		redisMock.ExpectHGetAll(redisUserKeyPrefix + testUUID).SetVal(map[string]string{})

		mockQuerier.On("GetUserByID", mock.Anything, pgUUID).
			Return(repository.User{
				ID:    pgUUID,
				Email: testEmail,
			}, nil).Once()

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectHSet(redisUserKeyPrefix+testUUID, mock.Anything).SetVal(1)
		redisMock.ExpectExpire(redisUserKeyPrefix+testUUID, 24*time.Hour).SetVal(true)

		res, err := server.GetUserByID(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, testUUID, res.Id)
		assert.Equal(t, testEmail, res.Email)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Success - Cache Hit", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())

		cachedData := map[string]string{
			"id":             testUUID,
			"username":       "testuser",
			"email":          testEmail,
			"hashedPassword": "hashed_password",
			"createdAt":      "1698624000",
			"updatedAt":      "1698624000",
		}

		redisMock.ExpectHGetAll(redisUserKeyPrefix + testUUID).SetVal(cachedData)

		res, err := server.GetUserByID(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, testUUID, res.Id)
		assert.Equal(t, testEmail, res.Email)
		assert.Equal(t, "testuser", res.Username)
		mockQuerier.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.Anything)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())

		redisMock.ExpectHGetAll(redisUserKeyPrefix + testUUID).SetVal(map[string]string{})

		mockQuerier.On("GetUserByID", mock.Anything, pgUUID).
			Return(repository.User{}, pgx.ErrNoRows).Once()

		res, err := server.GetUserByID(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())

		redisMock.ExpectHGetAll(redisUserKeyPrefix + "invalid-uuid").SetVal(map[string]string{})

		invalidReq := &userv1.GetUserByIDRequest{Id: "invalid-uuid"}
		res, err := server.GetUserByID(context.Background(), invalidReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, redisClient, logs.NewSlogLogger())

		redisMock.MatchExpectationsInOrder(false)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := server.GetUserByID(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.Anything)
	})
}
