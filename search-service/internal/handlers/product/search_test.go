package product_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/sonuudigital/microservices/search-service/internal/handlers/product"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSearchProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	const (
		searchPathWithQuery    = "/search/products?q=laptop"
		problemJSONContentType = "application/problem+json"
	)

	tests := []struct {
		name                string
		requestPath         string
		setupMock           func(*MockSearcher)
		expectedStatusCode  int
		expectedContentType string
	}{
		{
			name:        "Success",
			requestPath: "/search?q=laptop&size=2&from=0",
			setupMock: func(ms *MockSearcher) {
				searchRes := map[string]any{
					"hits": map[string]any{
						"total": map[string]int{"value": 2},
						"hits": []map[string]any{
							{"_source": map[string]any{"id": "p1", "name": "Laptop 1", "description": "Desc", "price": "10"}},
							{"_source": map[string]any{"id": "p2", "name": "Laptop 2", "description": "Desc", "price": "20"}},
						},
					},
				}
				bodyBytes, _ := json.Marshal(searchRes)
				resp := &opensearchapi.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(bodyBytes))}
				ms.On("Search", mock.Anything, "products", mock.Anything).Return(resp, nil).Once()
			},
			expectedStatusCode:  http.StatusOK,
			expectedContentType: "application/json",
		},
		{
			name:        "MissingQueryParam",
			requestPath: "/search",
			setupMock: func(ms *MockSearcher) {
				// Empty since the handler should return before calling Search
			},
			expectedStatusCode:  http.StatusBadRequest,
			expectedContentType: problemJSONContentType,
		},
		{
			name:        "SearcherError",
			requestPath: searchPathWithQuery,
			setupMock: func(ms *MockSearcher) {
				ms.On("Search", mock.Anything, "products", mock.Anything).Return(nil, assert.AnError).Once()
			},
			expectedStatusCode:  http.StatusInternalServerError,
			expectedContentType: problemJSONContentType,
		},
		{
			name:        "SearchResponseErrorStatus",
			requestPath: searchPathWithQuery,
			setupMock: func(ms *MockSearcher) {
				resp := &opensearchapi.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewReader([]byte("{}")))}
				ms.On("Search", mock.Anything, "products", mock.Anything).Return(resp, nil).Once()
			},
			expectedStatusCode:  http.StatusInternalServerError,
			expectedContentType: problemJSONContentType,
		},
		{
			name:        "DecodeError",
			requestPath: searchPathWithQuery,
			setupMock: func(ms *MockSearcher) {
				resp := &opensearchapi.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte("{bad}")))}
				ms.On("Search", mock.Anything, "products", mock.Anything).Return(resp, nil).Once()
			},
			expectedStatusCode:  http.StatusInternalServerError,
			expectedContentType: problemJSONContentType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := new(MockSearcher)
			h, err := product.NewProductHandler(logger, ms, "products")
			assert.NoError(t, err)

			tt.setupMock(ms)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			rr := httptest.NewRecorder()

			h.SearchProduct(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			assert.Contains(t, rr.Header().Get("Content-Type"), tt.expectedContentType)
			ms.AssertExpectations(t)
		})
	}
}
