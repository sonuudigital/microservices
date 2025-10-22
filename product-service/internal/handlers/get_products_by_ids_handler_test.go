package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/handlers"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetProductsByIDsHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr1 := uuidTest
		uuidStr2 := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
		var pgUUID1, pgUUID2 pgtype.UUID
		_ = pgUUID1.Scan(uuidStr1)
		_ = pgUUID2.Scan(uuidStr2)

		products := []repository.Product{
			{ID: pgUUID1, Name: "Product 1"},
			{ID: pgUUID2, Name: "Product 2"},
		}

		mockQuerier.On("GetProductsByIDs", mock.Anything, []pgtype.UUID{pgUUID1, pgUUID2}).Return(products, nil).Once()

		req, err := http.NewRequest("GET", apiProductsIdsPath+uuidStr1+","+uuidStr2, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.GetProductsByIDsHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respProducts []repository.Product
		err = json.NewDecoder(rr.Body).Decode(&respProducts)
		assert.NoError(t, err)
		assert.Equal(t, len(products), len(respProducts))
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Empty IDs", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		req, err := http.NewRequest("GET", apiProductsIdsPath, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.GetProductsByIDsHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respProducts []repository.Product
		err = json.NewDecoder(rr.Body).Decode(&respProducts)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(respProducts))
		mockQuerier.AssertNotCalled(t, "GetProductsByIDs")
	})

	t.Run("Invalid ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr1 := uuidTest
		req, err := http.NewRequest("GET", apiProductsIdsPath+uuidStr1+",invalid-uuid", nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.GetProductsByIDsHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockQuerier.AssertNotCalled(t, "GetProductsByIDs", mock.Anything, mock.Anything)
	})

	t.Run(dbErrorTitle, func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr1 := uuidTest
		var pgUUID1 pgtype.UUID
		_ = pgUUID1.Scan(uuidStr1)

		mockQuerier.On("GetProductsByIDs", mock.Anything, []pgtype.UUID{pgUUID1}).Return(nil, errors.New(dbErrorMsg)).Once()

		req, err := http.NewRequest("GET", apiProductsIdsPath+uuidStr1, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.GetProductsByIDsHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockQuerier.AssertExpectations(t)
	})
}
