package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteCartHandler(t *testing.T) {
	t.Run("Success", testDeleteCartSuccess)
	t.Run("Invalid User ID", testDeleteCartInvalidID)
}

func testDeleteCartSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockQuerier.On("DeleteCartByUserID", mock.Anything, pgUUID).Return(nil).Once()

	req, _ := http.NewRequest("DELETE", cartsURL, nil)
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartHandler(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockQuerier.AssertExpectations(t)
}

func testDeleteCartInvalidID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	req, _ := http.NewRequest("DELETE", cartsURL, nil)
	req.Header.Set(userIDHeader, invalidUUIDPathTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockQuerier.AssertNotCalled(t, "DeleteCartByUserID", mock.Anything, mock.Anything)
}
