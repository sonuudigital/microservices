package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/clients"
	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddProductToCartHandler(t *testing.T) {
	t.Run("Success new cart", testAddProductToCartSuccessNewCart)
	t.Run("Success existing cart", testAddProductToCartSuccessExistingCart)
	t.Run(invalidUserIDErrorTitleMsg, testAddProductToCartInvalidUserID)
	t.Run("Invalid request body", testAddProductToCartInvalidRequestBody)
	t.Run("Product not found", testAddProductToCartProductNotFound)
	t.Run("Product fetcher error", testAddProductToCartProductFetcherError)
	t.Run("DB error on add", testAddProductToCartDBErrorOnAdd)
}

func testAddProductToCartSuccessNewCart(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, cartUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = productUUID.Scan(productIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{}, pgx.ErrNoRows).Once()
	mockQuerier.On("CreateCart", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]clients.ProductByIDResponse{
		productIDTest: {ID: productIDTest, Name: testProductTitleMsg, Price: 99.99},
	}, nil).Once()

	priceNumeric := pgtype.Numeric{}
	_ = priceNumeric.Scan("99.99")
	params := repository.AddOrUpdateProductInCartParams{
		CartID:    cartUUID,
		ProductID: productUUID,
		Quantity:  1,
		Price:     priceNumeric,
	}
	expectedProduct := repository.CartsProduct{
		ID:        pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		CartID:    cartUUID,
		ProductID: productUUID,
		Quantity:  1,
		Price:     priceNumeric,
		AddedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	mockQuerier.On("AddOrUpdateProductInCart", mock.Anything, params).Return(expectedProduct, nil).Once()

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+productsPath, bytes.NewBuffer(body))
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp handlers.AddProductResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, productIDTest, resp.ProductID)
	assert.Equal(t, int32(1), resp.Quantity)

	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testAddProductToCartSuccessExistingCart(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, cartUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = productUUID.Scan(productIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]clients.ProductByIDResponse{
		productIDTest: {ID: productIDTest, Name: testProductTitleMsg, Price: 99.99},
	}, nil).Once()

	priceNumeric := pgtype.Numeric{}
	_ = priceNumeric.Scan("99.99")
	params := repository.AddOrUpdateProductInCartParams{
		CartID:    cartUUID,
		ProductID: productUUID,
		Quantity:  2,
		Price:     priceNumeric,
	}
	expectedProduct := repository.CartsProduct{
		ID:        pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		CartID:    cartUUID,
		ProductID: productUUID,
		Quantity:  2,
		Price:     priceNumeric,
		AddedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	mockQuerier.On("AddOrUpdateProductInCart", mock.Anything, params).Return(expectedProduct, nil).Once()

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 2}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+productsPath, bytes.NewBuffer(body))
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp handlers.AddProductResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, productIDTest, resp.ProductID)
	assert.Equal(t, int32(2), resp.Quantity)

	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testAddProductToCartInvalidUserID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+productsPath, bytes.NewBuffer(body))
	req.Header.Set(userIDHeader, invalidUUIDPathTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func testAddProductToCartInvalidRequestBody(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	req, _ := http.NewRequest("POST", cartsURL+productsPath, bytes.NewBufferString("{invalid-json}"))
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func testAddProductToCartProductNotFound(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, cartUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]clients.ProductByIDResponse{}, nil).Once()

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+productsPath, bytes.NewBuffer(body))
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testAddProductToCartProductFetcherError(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, cartUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(nil, errors.New(networkErrorMsg)).Once()

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+productsPath, bytes.NewBuffer(body))
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testAddProductToCartDBErrorOnAdd(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, cartUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = productUUID.Scan(productIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]clients.ProductByIDResponse{
		productIDTest: {ID: productIDTest, Name: testProductTitleMsg, Price: 99.99},
	}, nil).Once()

	priceNumeric := pgtype.Numeric{}
	_ = priceNumeric.Scan("99.99")
	params := repository.AddOrUpdateProductInCartParams{
		CartID:    cartUUID,
		ProductID: productUUID,
		Quantity:  1,
		Price:     priceNumeric,
	}
	mockQuerier.On("AddOrUpdateProductInCart", mock.Anything, params).Return(repository.CartsProduct{}, errors.New(dbErrorMsg)).Once()

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+productsPath, bytes.NewBuffer(body))
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}
