package search_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers/search"
	"github.com/stretchr/testify/assert"
)

const (
	searchPath = "/api//search/products"
)

func TestNewSearchHandler(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8080")
	assert.NoError(t, err)

	tests := []struct {
		name          string
		targetURL     *url.URL
		expectedError error
	}{
		{
			name:          "Success",
			targetURL:     validURL,
			expectedError: nil,
		},
		{
			name:          "NilTargetURL",
			targetURL:     nil,
			expectedError: search.ErrTargetURLNil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := search.NewSearchHandler(tt.targetURL)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestSearchHandlerServeHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, searchPath, r.URL.Path)
		assert.Equal(t, "laptop", r.URL.Query().Get("q"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	assert.NoError(t, err)

	handler, err := search.NewSearchHandler(backendURL)
	assert.NoError(t, err)
	assert.NotNil(t, handler)

	req := httptest.NewRequest(http.MethodGet, searchPath+"?q=laptop", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, `{"results":[]}`, rr.Body.String())
}

func TestSearchHandlerProxyHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Host)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	assert.NoError(t, err)

	handler, err := search.NewSearchHandler(backendURL)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, searchPath, nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSearchHandlerDifferentMethods(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT",
			method:         http.MethodPut,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.method, r.Method)
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			backendURL, err := url.Parse(backend.URL)
			assert.NoError(t, err)

			handler, err := search.NewSearchHandler(backendURL)
			assert.NoError(t, err)

			req := httptest.NewRequest(tt.method, searchPath, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
