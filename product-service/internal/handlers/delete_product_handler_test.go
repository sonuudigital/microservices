package handlers_test

import (
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

func TestDeleteProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	uuidStr := uuidTest
	var pgUUID pgtype.UUID
	err := pgUUID.Scan(uuidStr)
	assert.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(repository.Product{}, nil).Once()
		mockQuerier.On("DeleteProduct", mock.Anything, pgUUID).Return(nil).Once()

		req, err := http.NewRequest("DELETE", "/api/products/"+uuidStr, nil)
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.DeleteProductHandler(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		mockQuerier.AssertExpectations(t)
	})
}
