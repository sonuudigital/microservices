package category_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateProductCategory(t *testing.T) {
	t.Run("Success", testCreateProductCategorySuccess)
	t.Run("Invalid Argument - Empty Name", testCreateProductCategoryInvalidArgument)
	t.Run("Not Found", testCreateProductCategoryNotFound)
	t.Run("DB Error", testCreateProductCategoryDBError)
	t.Run("Context Canceled", testCreateProductCategoryContextCanceled)
}

func testCreateProductCategorySuccess(t *testing.T) {
	req := &product_categoriesv1.CreateProductCategoryRequest{
		Name:        "Electronics",
		Description: "Phones and gadgets",
	}

	mockQuerier, redisMock, server := initializeMocksAndServer()

	var id pgtype.UUID
	_ = id.Scan("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22")
	now := time.Now()
	repoPC := repository.ProductCategory{
		ID:          id,
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: true},
		CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Valid: false},
	}

	mockQuerier.
		On("CreateProductCategory", mock.Anything, mock.MatchedBy(func(arg repository.CreateProductCategoryParams) bool {
			return arg.Name == req.Name && arg.Description.String == req.Description && arg.Description.Valid
		})).
		Return(repoPC, nil).
		Once()

	redisMock.ExpectDel("product_categories:all").SetVal(1)

	resp, err := server.CreateProductCategory(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, repoPC.Name, resp.Name)
	assert.Equal(t, repoPC.Description.String, resp.Description)
	assert.NotEmpty(t, resp.Id)
	mockQuerier.AssertExpectations(t)
}

func testCreateProductCategoryInvalidArgument(t *testing.T) {
	req := &product_categoriesv1.CreateProductCategoryRequest{
		Name:        "",
		Description: "desc",
	}

	mockQuerier, _, server := initializeMocksAndServer()

	mockQuerier.On("CreateProductCategory", mock.Anything, mock.Anything).Return(req, nil).Once()

	_, err := server.CreateProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	mockQuerier.AssertNotCalled(t, "CreateProductCategory", mock.Anything, mock.Anything)
}

func testCreateProductCategoryNotFound(t *testing.T) {
	req := &product_categoriesv1.CreateProductCategoryRequest{
		Name:        "Books",
		Description: "Fiction",
	}

	mockQuerier, _, server := initializeMocksAndServer()

	mockQuerier.
		On("CreateProductCategory", mock.Anything, mock.Anything).
		Return(repository.ProductCategory{}, pgx.ErrNoRows).
		Once()

	_, err := server.CreateProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testCreateProductCategoryDBError(t *testing.T) {
	req := &product_categoriesv1.CreateProductCategoryRequest{
		Name:        "Home",
		Description: "Kitchen",
	}

	mockQuerier, _, server := initializeMocksAndServer()

	mockQuerier.
		On("CreateProductCategory", mock.Anything, mock.Anything).
		Return(repository.ProductCategory{}, pgx.ErrTxClosed).
		Once()

	_, err := server.CreateProductCategory(context.Background(), req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	mockQuerier.AssertExpectations(t)
}

func testCreateProductCategoryContextCanceled(t *testing.T) {
	req := &product_categoriesv1.CreateProductCategoryRequest{
		Name:        "Garden",
		Description: "Plants",
	}

	mockQuerier, _, server := initializeMocksAndServer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := server.CreateProductCategory(ctx, req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
	mockQuerier.AssertNotCalled(t, "CreateProductCategory", mock.Anything, mock.Anything)
}
