package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRemoveProductFromCartHandler(t *testing.T) {
	t.Run("Success", testRemoveProductFromCartSuccess)
	t.Run(invalidUserIDErrorTitleMsg, testRemoveProductFromCartInvalidUserID)
	t.Run("Invalid product ID", testRemoveProductFromCartInvalidProductID)
	t.Run("Product not found", testRemoveProductFromCartProductNotFound)
	t.Run("Product fetcher error", testRemoveProductFromCartProductFetcherError)
	t.Run("DB error", testRemoveProductFromCartDBError)
}

func testRemoveProductFromCartSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = productUUID.Scan(productIDTest)

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]handlers.ProductByIDResponse{
		productIDTest: {ID: productIDTest},
	}, nil).Once()
	mockQuerier.On("RemoveProductFromCart", mock.Anything, repository.RemoveProductFromCartParams{
		UserID:    userUUID,
		ProductID: productUUID,
	}).Return(nil).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+productsIDPath, nil)
	req.Header.Set(userIDHeader, uuidTest)
	req.SetPathValue("productId", productIDTest)
	rr := httptest.NewRecorder()

	handler.RemoveProductFromCartHandler(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testRemoveProductFromCartInvalidUserID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	req, _ := http.NewRequest("DELETE", cartsURL+"/products/"+productIDTest, nil)
	req.Header.Set(userIDHeader, invalidUUIDPathTest)
	req.SetPathValue("productId", productIDTest)
	rr := httptest.NewRecorder()

	handler.RemoveProductFromCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func testRemoveProductFromCartInvalidProductID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	req, _ := http.NewRequest("DELETE", cartsURL+"/products/invalid-uuid", nil)
	req.Header.Set(userIDHeader, uuidTest)
	req.SetPathValue("productId", invalidUUIDPathTest)
	rr := httptest.NewRecorder()

	handler.RemoveProductFromCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func testRemoveProductFromCartProductNotFound(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]handlers.ProductByIDResponse{}, nil).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+productsIDPath, nil)
	req.Header.Set(userIDHeader, uuidTest)
	req.SetPathValue("productId", productIDTest)
	rr := httptest.NewRecorder()

	handler.RemoveProductFromCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockProductFetcher.AssertExpectations(t)
	mockQuerier.AssertNotCalled(t, "RemoveProductFromCart", mock.Anything, mock.Anything)
}

func testRemoveProductFromCartProductFetcherError(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(nil, errors.New(networkErrorMsg)).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+productsIDPath, nil)
	req.Header.Set(userIDHeader, uuidTest)
	req.SetPathValue("productId", productIDTest)
	rr := httptest.NewRecorder()

	handler.RemoveProductFromCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockProductFetcher.AssertExpectations(t)
	mockQuerier.AssertNotCalled(t, "RemoveProductFromCart", mock.Anything, mock.Anything)
}

func testRemoveProductFromCartDBError(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = productUUID.Scan(productIDTest)

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]handlers.ProductByIDResponse{
		productIDTest: {ID: productIDTest},
	}, nil).Once()
	mockQuerier.On("RemoveProductFromCart", mock.Anything, repository.RemoveProductFromCartParams{
		UserID:    userUUID,
		ProductID: productUUID,
	}).Return(errors.New(dbErrorMsg)).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+productsIDPath, nil)
	req.Header.Set(userIDHeader, uuidTest)
	req.SetPathValue("productId", productIDTest)
	rr := httptest.NewRecorder()

	handler.RemoveProductFromCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}
