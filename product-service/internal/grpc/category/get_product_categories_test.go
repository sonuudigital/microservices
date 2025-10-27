package category_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sonuudigital/microservices/product-service/internal/grpc/category"
	product_service_mock "github.com/sonuudigital/microservices/product-service/internal/mock"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetProductCategories(t *testing.T) {
	t.Run("Success", testGetProductCategoriesSuccess)
	t.Run("DB Error", testGetProductCategoriesDBError)
	t.Run("Context Canceled", testGetProductCategoriesContextCanceled)
}

func testGetProductCategoriesSuccess(t *testing.T) {
	req := &emptypb.Empty{}
	mockQuerier := new(product_service_mock.MockQuerier)
	server := category.New(mockQuerier)

	mockQuerier.On("GetProductCategories", mock.Anything).Return([]repository.ProductCategory{}, nil).Once()

	resp, err := server.GetProductCategories(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockQuerier.AssertExpectations(t)
}

func testGetProductCategoriesDBError(t *testing.T) {
	req := &emptypb.Empty{}
	mockQuerier := new(product_service_mock.MockQuerier)
	server := category.New(mockQuerier)

	mockQuerier.On("GetProductCategories", mock.Anything).Return([]repository.ProductCategory{}, pgx.ErrTxClosed).Once()

	_, err := server.GetProductCategories(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testGetProductCategoriesContextCanceled(t *testing.T) {
	req := &emptypb.Empty{}
	mockQuerier := new(product_service_mock.MockQuerier)
	server := category.New(mockQuerier)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := server.GetProductCategories(ctx, req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
	mockQuerier.AssertNotCalled(t, "GetProductCategories", mock.Anything)
}
