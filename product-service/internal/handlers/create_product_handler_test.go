package handlers_test

import (
	"bytes"
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

func TestCreateProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	productRequest := handlers.ProductRequest{
		Name:          "Test Product",
		Description:   "Test Description",
		Price:         99.99,
		Code:          "TEST001",
		StockQuantity: 100,
	}
	body, _ := json.Marshal(productRequest)

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("CreateProduct", mock.Anything, mock.AnythingOfType("repository.CreateProductParams")).Return(repository.Product{}, nil).Once()

		req, err := http.NewRequest("POST", productsURL, bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.CreateProductHandler(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run(dbErrorTitle, func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("CreateProduct", mock.Anything, mock.AnythingOfType("repository.CreateProductParams")).Return(repository.Product{}, errors.New(dbErrorMsg)).Once()

		req, err := http.NewRequest("POST", productsURL, bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.CreateProductHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockQuerier.AssertExpectations(t)
	})
}
