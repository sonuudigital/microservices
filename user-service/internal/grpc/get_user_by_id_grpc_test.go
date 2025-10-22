package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
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

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewGRPCServer(mockQuerier, nil)
		mockQuerier.On("GetUserByID", mock.Anything, pgUUID).
			Return(repository.User{
				ID:    pgUUID,
				Email: testEmail,
			}, nil).Once()

		res, err := server.GetUserByID(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, testUUID, res.Id)
		assert.Equal(t, testEmail, res.Email)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := grpc_server.NewGRPCServer(mockQuerier, nil)
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
		server := grpc_server.NewGRPCServer(mockQuerier, nil)
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
		server := grpc_server.NewGRPCServer(mockQuerier, nil)
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
