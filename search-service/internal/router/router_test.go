package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sonuudigital/microservices/search-service/internal/router"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProductHandler struct {
	mock.Mock
}

func (m *MockProductHandler) SearchProduct(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func TestNew(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockHandler := new(MockProductHandler)

	tests := []struct {
		name           string
		logger         logs.Logger
		productHandler router.ProductHandler
		expectedError  error
	}{
		{
			name:           "Success",
			logger:         logger,
			productHandler: mockHandler,
			expectedError:  nil,
		},
		{
			name:           "NilLogger",
			logger:         nil,
			productHandler: mockHandler,
			expectedError:  router.ErrLoggerIsNil,
		},
		{
			name:           "NilProductHandler",
			logger:         logger,
			productHandler: nil,
			expectedError:  router.ErrProductHandlerIsNil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := router.New(tt.logger, tt.productHandler)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, r)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, r)
			}
		})
	}
}

func TestRouterServeHTTPHealthzEndpoint(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockHandler := new(MockProductHandler)

	r, err := router.New(logger, mockHandler)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "search service is healthy", rr.Body.String())
}

func TestRouterServeHTTPSearchProductEndpoint(t *testing.T) {
	logger := logs.NewSlogLogger()

	tests := []struct {
		name               string
		requestPath        string
		requestMethod      string
		setupMock          func(*MockProductHandler)
		expectedStatusCode int
	}{
		{
			name:          "SearchProductCalled",
			requestPath:   "/api/products/search",
			requestMethod: http.MethodGet,
			setupMock: func(m *MockProductHandler) {
				m.On("SearchProduct", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					w := args.Get(0).(http.ResponseWriter)
					w.WriteHeader(http.StatusOK)
				}).Once()
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:          "SearchProductWithQueryParams",
			requestPath:   "/api/products/search?q=laptop",
			requestMethod: http.MethodGet,
			setupMock: func(m *MockProductHandler) {
				m.On("SearchProduct", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					w := args.Get(0).(http.ResponseWriter)
					r := args.Get(1).(*http.Request)
					assert.Equal(t, "laptop", r.URL.Query().Get("q"))
					w.WriteHeader(http.StatusOK)
				}).Once()
			},
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockProductHandler)
			tt.setupMock(mockHandler)

			r, err := router.New(logger, mockHandler)
			assert.NoError(t, err)

			req := httptest.NewRequest(tt.requestMethod, tt.requestPath, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			mockHandler.AssertExpectations(t)
		})
	}
}

func TestRouterServeHTTPNotFoundEndpoint(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockHandler := new(MockProductHandler)

	r, err := router.New(logger, mockHandler)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/notfound", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
