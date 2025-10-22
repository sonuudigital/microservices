package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sonuudigital/microservices/product-service/internal/handlers"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListProductsHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		products := []repository.Product{{Name: "Product 1"}, {Name: "Product 2"}}
		mockQuerier.On("ListProductsPaginated", mock.Anything, mock.AnythingOfType("repository.ListProductsPaginatedParams")).Return(products, nil).Once()

		req, err := http.NewRequest("GET", productsURL, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ListProductsHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respProducts []repository.Product
		err = json.NewDecoder(rr.Body).Decode(&respProducts)
		assert.NoError(t, err)
		assert.Equal(t, len(products), len(respProducts))
		mockQuerier.AssertExpectations(t)
	})

	t.Run(dbErrorTitle, func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("ListProductsPaginated", mock.Anything, mock.AnythingOfType("repository.ListProductsPaginatedParams")).Return(nil, errors.New(dbErrorMsg)).Once()

		req, err := http.NewRequest("GET", productsURL, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ListProductsHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockQuerier.AssertExpectations(t)
	})
}
