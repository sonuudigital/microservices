package category_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	mockOfTypeUpdateProductCategoryParams = "repository.UpdateProductCategoryParams"
)

func TestUpdateProductCategory(t *testing.T) {
	t.Run("Success", testUpdateProductCategorySuccess)
	t.Run("Invalid Argument - Empty ID", testUpdateProductCategoryInvalidArgument)
	t.Run("Not Found", testUpdateProductCategoryNotFound)
	t.Run("Internal Error", testUpdateProductCategoryInternalError)
	t.Run("Context Canceled", testUpdateProductCategoryContextCanceled)
}

func testUpdateProductCategorySuccess(t *testing.T) {
	req := &product_categoriesv1.UpdateProductCategoryRequest{
		Id:          categoryID,
		Name:        categoryName,
		Description: categoryDescription,
	}
	mockQuerier, redisMock, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)

	mockQuerier.
		On("UpdateProductCategory", mock.Anything, mock.AnythingOfType(mockOfTypeUpdateProductCategoryParams)).
		Return(nil).
		Once()

	redisMock.ExpectDel("product_categories:all").SetVal(1)

	_, err := server.UpdateProductCategory(context.Background(), req)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func testUpdateProductCategoryInvalidArgument(t *testing.T) {
	req := &product_categoriesv1.UpdateProductCategoryRequest{
		Id:          malformedID,
		Name:        categoryName,
		Description: categoryDescription,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	mockQuerier.
		On("UpdateProductCategory", mock.Anything, mock.Anything).
		Return(nil).
		Once()

	_, err := server.UpdateProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	mockQuerier.AssertNotCalled(t, "UpdateProductCategory", mock.Anything, mock.Anything)
}

func testUpdateProductCategoryNotFound(t *testing.T) {
	req := &product_categoriesv1.UpdateProductCategoryRequest{
		Id:          categoryID,
		Name:        categoryName,
		Description: categoryDescription,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)

	mockQuerier.
		On("UpdateProductCategory", mock.Anything, mock.AnythingOfType(mockOfTypeUpdateProductCategoryParams)).
		Return(pgx.ErrNoRows).
		Once()

	_, err := server.UpdateProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testUpdateProductCategoryInternalError(t *testing.T) {
	req := &product_categoriesv1.UpdateProductCategoryRequest{
		Id:          categoryID,
		Name:        categoryName,
		Description: categoryDescription,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)

	mockQuerier.
		On("UpdateProductCategory", mock.Anything, mock.AnythingOfType(mockOfTypeUpdateProductCategoryParams)).
		Return(pgx.ErrTooManyRows).
		Once()

	_, err := server.UpdateProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testUpdateProductCategoryContextCanceled(t *testing.T) {
	req := &product_categoriesv1.UpdateProductCategoryRequest{
		Id:          categoryID,
		Name:        categoryName,
		Description: categoryDescription,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := server.UpdateProductCategory(ctx, req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
	mockQuerier.AssertNotCalled(t, "UpdateProductCategory", mock.Anything, mock.Anything)
}
