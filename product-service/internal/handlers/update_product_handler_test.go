package handlers_test

import (
	"bytes"
	"encoding/json"
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

func TestUpdateProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	uuidStr := uuidTest
	var pgUUID pgtype.UUID
	err := pgUUID.Scan(uuidStr)
	assert.NoError(t, err)

	productRequest := handlers.ProductRequest{
		Name:          "Updated Product",
		Description:   "Updated Description",
		Price:         129.99,
		Code:          "UPDATED001",
		StockQuantity: 50,
	}
	body, _ := json.Marshal(productRequest)

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		updatedProduct := repository.Product{ID: pgUUID, Name: "Updated Product"}
		mockQuerier.On("UpdateProduct", mock.Anything, mock.AnythingOfType("repository.UpdateProductParams")).Return(updatedProduct, nil).Once()

		req, err := http.NewRequest("PUT", "/api/products/"+uuidStr, bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.UpdateProductHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respProduct repository.Product
		err = json.NewDecoder(rr.Body).Decode(&respProduct)
		assert.NoError(t, err)
		assert.Equal(t, updatedProduct.Name, respProduct.Name)
		mockQuerier.AssertExpectations(t)
	})
}
