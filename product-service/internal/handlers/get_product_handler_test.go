package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/handlers"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr := uuidTest
		var pgUUID pgtype.UUID
		err := pgUUID.Scan(uuidStr)
		assert.NoError(t, err)

		product := repository.Product{ID: pgUUID, Name: "Test Product"}
		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(product, nil).Once()

		req, err := http.NewRequest("GET", productsURL+"/"+uuidStr, nil)
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respProduct repository.Product
		err = json.NewDecoder(rr.Body).Decode(&respProduct)
		assert.NoError(t, err)
		assert.Equal(t, product.Name, respProduct.Name)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr := uuidTest
		var pgUUID pgtype.UUID
		err := pgUUID.Scan(uuidStr)
		assert.NoError(t, err)

		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(repository.Product{}, pgx.ErrNoRows).Once()

		req, err := http.NewRequest("GET", productsURL+"/"+uuidStr, nil)
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		req, err := http.NewRequest("GET", "/api/products/invalid-id", nil)
		assert.NoError(t, err)
		req.SetPathValue("id", "invalid-id")

		rr := httptest.NewRecorder()
		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
