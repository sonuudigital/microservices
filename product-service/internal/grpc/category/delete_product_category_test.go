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

func TestDeleteProductCategory(t *testing.T) {
	t.Run("Success", testDeleteProductCategorySuccess)
	t.Run("Invalid Argument - Empty ID", testDeleteProductCategoryInvalidArgumentEmptyID)
	t.Run("Invalid Argument - Malformed ID", testDeleteProductCategoryInvalidArgumentMalformedID)
	t.Run("Not Found", testDeleteProductCategoryNotFound)
	t.Run("Too Many Rows", testDeleteProductCategoryTooManyRows)
	t.Run("Internal Error", testDeleteProductCategoryInternalError)
	t.Run("Context Canceled", testDeleteProductCategoryContextCanceled)
}

func testDeleteProductCategorySuccess(t *testing.T) {
	req := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: categoryID,
	}
	mockQuerier, redisMock, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)

	mockQuerier.
		On("DeleteProductCategory", mock.Anything, id).
		Return(nil).
		Once()

	redisMock.ExpectDel("product_categories:all").SetVal(1)

	_, err := server.DeleteProductCategory(context.Background(), req)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func testDeleteProductCategoryInvalidArgumentEmptyID(t *testing.T) {
	req := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: "",
	}
	mockQuerier, _, server := initializeMocksAndServer()

	_, err := server.DeleteProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	mockQuerier.AssertNotCalled(t, "DeleteProductCategory", mock.Anything, mock.Anything)
}

func testDeleteProductCategoryInvalidArgumentMalformedID(t *testing.T) {
	req := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: malformedID,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	_, err := server.DeleteProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	mockQuerier.AssertNotCalled(t, "DeleteProductCategory", mock.Anything, mock.Anything)
}

func testDeleteProductCategoryNotFound(t *testing.T) {
	req := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: categoryID,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)

	mockQuerier.
		On("DeleteProductCategory", mock.Anything, id).
		Return(pgx.ErrNoRows).
		Once()

	_, err := server.DeleteProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testDeleteProductCategoryTooManyRows(t *testing.T) {
	req := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: categoryID,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)

	mockQuerier.
		On("DeleteProductCategory", mock.Anything, id).
		Return(pgx.ErrTooManyRows).
		Once()

	_, err := server.DeleteProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testDeleteProductCategoryInternalError(t *testing.T) {
	req := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: categoryID,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan(categoryID)

	mockQuerier.
		On("DeleteProductCategory", mock.Anything, id).
		Return(pgx.ErrTxClosed).
		Once()

	_, err := server.DeleteProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testDeleteProductCategoryContextCanceled(t *testing.T) {
	req := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: categoryID,
	}
	mockQuerier, _, server := initializeMocksAndServer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := server.DeleteProductCategory(ctx, req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
	mockQuerier.AssertNotCalled(t, "DeleteProductCategory", mock.Anything, mock.Anything)
}
