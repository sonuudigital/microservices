package category_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetProductCategories(t *testing.T) {
	t.Run("Success - Cache Miss", testGetProductCategoriesSuccess)
	t.Run("Success - Cache Hit", testGetProductCategoriesSuccessCacheHit)
	t.Run("DB Error", testGetProductCategoriesDBError)
	t.Run("Context Canceled", testGetProductCategoriesContextCanceled)
}

func testGetProductCategoriesSuccess(t *testing.T) {
	req := &emptypb.Empty{}
	mockQuerier, redisMock, server := initializeMocksAndServer()

	redisMock.ExpectGet(allProductCategoriesCacheKey).RedisNil()

	var id1, id2 pgtype.UUID
	_ = id1.Scan("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22")
	_ = id2.Scan("c2ffbc99-9c0b-4ef8-bb6d-6bb9bd380b33")
	now := time.Now()

	dbCategories := []repository.ProductCategory{
		{
			ID:          id1,
			Name:        "Electronics",
			Description: pgtype.Text{String: "Electronic devices", Valid: true},
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Valid: false},
		},
		{
			ID:          id2,
			Name:        "Books",
			Description: pgtype.Text{String: "All kinds of books", Valid: true},
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Valid: false},
		},
	}

	mockQuerier.On("GetProductCategories", mock.Anything).Return(dbCategories, nil).Once()

	redisMock.ExpectSet(allProductCategoriesCacheKey, mock.Anything, 1*time.Hour).SetVal("OK")

	resp, err := server.GetProductCategories(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Categories, 2)
	mockQuerier.AssertExpectations(t)
}

func testGetProductCategoriesSuccessCacheHit(t *testing.T) {
	req := &emptypb.Empty{}
	mockQuerier, redisMock, server := initializeMocksAndServer()

	cachedJSON := `[{"id":"b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22","name":"Electronics","description":"Electronic devices","createdAt":"2024-10-30T10:00:00Z","updatedAt":null}]`
	redisMock.ExpectGet(allProductCategoriesCacheKey).SetVal(cachedJSON)

	resp, err := server.GetProductCategories(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Categories, 1)
	assert.Equal(t, "Electronics", resp.Categories[0].Name)

	mockQuerier.AssertNotCalled(t, "GetProductCategories", mock.Anything)
}

func testGetProductCategoriesDBError(t *testing.T) {
	req := &emptypb.Empty{}
	mockQuerier, redisMock, server := initializeMocksAndServer()

	redisMock.ExpectGet(allProductCategoriesCacheKey).RedisNil()

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
	mockQuerier, _, server := initializeMocksAndServer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := server.GetProductCategories(ctx, req)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
	mockQuerier.AssertNotCalled(t, "GetProductCategories", mock.Anything)
}
