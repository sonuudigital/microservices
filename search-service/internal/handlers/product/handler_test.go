package product_test

import (
	"context"
	"io"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/sonuudigital/microservices/search-service/internal/handlers/product"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSearcher struct {
	mock.Mock
}

func (m *MockSearcher) Search(ctx context.Context, indexName string, body io.Reader) (*opensearchapi.Response, error) {
	args := m.Called(ctx, indexName, body)
	resp, _ := args.Get(0).(*opensearchapi.Response)
	return resp, args.Error(1)
}

func TestNewProductHandlerSuccess(t *testing.T) {
	logger := logs.NewSlogLogger()
	ms := new(MockSearcher)
	h, err := product.NewProductHandler(logger, ms, "products")
	assert.NoError(t, err)
	assert.NotNil(t, h)
}

func TestNewProductHandlerNilLogger(t *testing.T) {
	ms := new(MockSearcher)
	h, err := product.NewProductHandler(nil, ms, "products")
	assert.Error(t, err)
	assert.Equal(t, product.ErrNilLogger, err)
	assert.Nil(t, h)
}

func TestNewProductHandlerNilSearcher(t *testing.T) {
	logger := logs.NewSlogLogger()
	h, err := product.NewProductHandler(logger, nil, "products")
	assert.Error(t, err)
	assert.Equal(t, product.ErrNilOpensearchClient, err)
	assert.Nil(t, h)
}
