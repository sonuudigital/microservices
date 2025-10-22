package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClearCartHandler(t *testing.T) {
	t.Run("Success", testClearCartSuccess)
	t.Run(invalidUserIDErrorTitleMsg, testClearCartInvalidID)
	t.Run("DB error", testClearCartDBError)
}

func testClearCartSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)

	mockQuerier.On("ClearCartProductsByUserID", mock.Anything, userUUID).Return(nil).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+productsPath, nil)
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.ClearCartHandler(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockQuerier.AssertExpectations(t)
}

func testClearCartInvalidID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	req, _ := http.NewRequest("DELETE", cartsURL+productsPath, nil)
	req.Header.Set(userIDHeader, "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ClearCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func testClearCartDBError(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)

	mockQuerier.On("ClearCartProductsByUserID", mock.Anything, userUUID).Return(errors.New(dbErrorMsg)).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+productsPath, nil)
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.ClearCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
}
