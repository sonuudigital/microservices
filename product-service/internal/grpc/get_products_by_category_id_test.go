package grpc_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	grpc "github.com/sonuudigital/microservices/product-service/internal/grpc"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetProductsByCategoryID(t *testing.T) {
	t.Run("Success", testSuccess)
	t.Run("Malformed UUID", testsMalformedUUID)
	t.Run("Not Found", testsNotFound)
	t.Run("DB Error", testDBError)
	t.Run("Context Canceled", testContextCanceled)
}

func testSuccess(t *testing.T) {
	pgUUID := scanAndGetCategoryUUID()
	req, mockQuerier, server := initializeProductService()
	mockQuerier.On("GetProductsByCategoryID", mock.Anything, pgUUID).Return([]repository.Product{}, nil).Once()

	resp, err := server.GetProductsByCategoryID(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Products, 0)
	mockQuerier.AssertExpectations(t)
}

func testsMalformedUUID(t *testing.T) {
	req := &productv1.GetProductsByCategoryIDRequest{CategoryId: uuidMalformed}
	mockQuerier := new(MockQuerier)
	server := grpc.NewServer(mockQuerier)

	_, err := server.GetProductsByCategoryID(context.Background(), req)

	assert.Error(t, err)
	mockQuerier.AssertNotCalled(t, "GetProductsByCategoryID", mock.Anything, mock.Anything)
}

func testsNotFound(t *testing.T) {
	pgUUID := scanAndGetCategoryUUID()
	req, mockQuerier, server := initializeProductService()
	mockQuerier.On("GetProductsByCategoryID", mock.Anything, pgUUID).Return([]repository.Product{}, pgx.ErrNoRows).Once()

	_, err := server.GetProductsByCategoryID(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testDBError(t *testing.T) {
	pgUUID := scanAndGetCategoryUUID()
	req, mockQuerier, server := initializeProductService()
	mockQuerier.On("GetProductsByCategoryID", mock.Anything, pgUUID).Return([]repository.Product{}, pgx.ErrTxClosed).Once()

	_, err := server.GetProductsByCategoryID(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testContextCanceled(t *testing.T) {
	scanAndGetCategoryUUID()
	req, mockQuerier, server := initializeProductService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := server.GetProductsByCategoryID(ctx, req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
	mockQuerier.AssertNotCalled(t, "GetProductsByCategoryID", mock.Anything, mock.Anything)
}

func initializeProductService() (*productv1.GetProductsByCategoryIDRequest, *MockQuerier, *grpc.GRPCServer) {
	req := &productv1.GetProductsByCategoryIDRequest{CategoryId: uuidCategoryTest}
	mockQuerier := new(MockQuerier)
	server := grpc.NewServer(mockQuerier)
	return req, mockQuerier, server
}
