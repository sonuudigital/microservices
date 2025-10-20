package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	uuidTest            = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	cartUUIDTest        = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
	cartsURL            = "/api/carts"
	productIDTest       = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
	testProductTitleMsg = "Test Product"
	invalidUUIDPathTest = "invalid-uuid"
	networkErrorMsg     = "network error"
	dbErrorMsg          = "db error"
	productsPath        = "/products"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) AddOrUpdateProductInCart(ctx context.Context, arg repository.AddOrUpdateProductInCartParams) (repository.CartsProduct, error) {
	args := m.Called(ctx, arg)
	if c, ok := args.Get(0).(repository.CartsProduct); ok {
		return c, args.Error(1)
	}
	return repository.CartsProduct{}, args.Error(1)
}

func (m *MockQuerier) GetCartByUserID(ctx context.Context, userID pgtype.UUID) (repository.Cart, error) {
	args := m.Called(ctx, userID)
	if c, ok := args.Get(0).(repository.Cart); ok {
		return c, args.Error(1)
	}
	return repository.Cart{}, args.Error(1)
}

func (m *MockQuerier) CreateCart(ctx context.Context, userID pgtype.UUID) (repository.Cart, error) {
	args := m.Called(ctx, userID)

	if err := args.Error(1); err != nil {
		return repository.Cart{}, err
	}

	if c, ok := args.Get(0).(repository.Cart); ok {
		return c, args.Error(1)
	}

	return repository.Cart{}, args.Error(1)
}

func (m *MockQuerier) DeleteCartByUserID(ctx context.Context, userID pgtype.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockQuerier) GetCartProductsByCartID(ctx context.Context, cartID pgtype.UUID) ([]repository.GetCartProductsByCartIDRow, error) {
	args := m.Called(ctx, cartID)
	if c, ok := args.Get(0).([]repository.GetCartProductsByCartIDRow); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockUserValidator struct {
	mock.Mock
}

func (m *MockUserValidator) ValidateUserExists(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

type MockProductFetcher struct {
	mock.Mock
}

func (m *MockProductFetcher) GetProductsByIDs(ctx context.Context, ids []string) (map[string]handlers.ProductByIDResponse, error) {
	args := m.Called(ctx, ids)
	if c, ok := args.Get(0).(map[string]handlers.ProductByIDResponse); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func TestGetCartByUserIDHandler(t *testing.T) {
	t.Run("Success", testGetCartByUserIDSuccess)
	t.Run("Not Found", testGetCartByUserIDNotFound)
	t.Run("Invalid ID", testGetCartByUserIDInvalidID)
}

func testGetCartByUserIDSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var userUUID, cartUUID, product1UUID, product2UUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = product1UUID.Scan("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13")
	_ = product2UUID.Scan("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14")

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()

	expectedCart := repository.Cart{ID: cartUUID, UserID: userUUID}
	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(expectedCart, nil).Once()

	cartItems := []repository.GetCartProductsByCartIDRow{
		{ProductID: product1UUID, Quantity: 2},
		{ProductID: product2UUID, Quantity: 1},
	}
	mockQuerier.On("GetCartProductsByCartID", mock.Anything, cartUUID).Return(cartItems, nil).Once()

	productIDs := []string{product1UUID.String(), product2UUID.String()}
	fetchedProducts := map[string]handlers.ProductByIDResponse{
		product1UUID.String(): {ID: product1UUID.String(), Name: "Product 1", Price: 10.00},
		product2UUID.String(): {ID: product2UUID.String(), Name: "Product 2", Price: 5.50},
	}
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, productIDs).Return(fetchedProducts, nil).Once()

	req, _ := http.NewRequest("GET", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()
	handler.GetCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var respCart handlers.GetCartResponse
	err := json.NewDecoder(rr.Body).Decode(&respCart)
	assert.NoError(t, err)

	assert.Equal(t, cartUUID.String(), respCart.ID)
	assert.Equal(t, userUUID.String(), respCart.UserID)
	assert.Len(t, respCart.Products, 2)
	assert.InDelta(t, 25.50, respCart.TotalPrice, 0.001)

	responseProductsMap := make(map[string]handlers.CartProductResponse)
	for _, p := range respCart.Products {
		responseProductsMap[p.ProductID] = p
	}

	p1, ok := responseProductsMap[product1UUID.String()]
	assert.True(t, ok, "Product 1 not found in response")
	assert.Equal(t, "Product 1", p1.Name)
	assert.Equal(t, 2, p1.Quantity)
	assert.Equal(t, 10.00, p1.Price)

	p2, ok := responseProductsMap[product2UUID.String()]
	assert.True(t, ok, "Product 2 not found in response")
	assert.Equal(t, "Product 2", p2.Name)
	assert.Equal(t, 1, p2.Quantity)
	assert.Equal(t, 5.50, p2.Price)

	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testGetCartByUserIDNotFound(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("GetCartByUserID", mock.Anything, pgUUID).Return(repository.Cart{}, pgx.ErrNoRows).Once()

	req, _ := http.NewRequest("GET", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.GetCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testGetCartByUserIDInvalidID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	req, _ := http.NewRequest("GET", cartsURL+"/invalid-uuid", nil)
	req.SetPathValue("id", invalidUUIDPathTest)
	rr := httptest.NewRecorder()

	handler.GetCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateCartHandler(t *testing.T) {
	t.Run("Success", testCreateCartSuccess)
	t.Run("User Does Not Exist", testCreateCartUserDoesNotExist)
	t.Run("User Validation Fails", testCreateCartUserValidationFails)
	t.Run("DB Error on CreateCart", testCreateCartDBError)
	t.Run("Invalid Request Body", testCreateCartInvalidBody)
}

func testCreateCartSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("GetCartByUserID", mock.Anything, pgUUID).Return(repository.Cart{}, pgx.ErrNoRows).Once()
	mockQuerier.On("CreateCart", mock.Anything, pgUUID).Return(repository.Cart{ID: pgUUID, UserID: pgUUID}, nil).Once()

	cartReq := handlers.CartRequest{UserID: uuidTest}
	body, _ := json.Marshal(cartReq)
	req, _ := http.NewRequest("POST", cartsURL, bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.CreateCartHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var respCart handlers.CreateCartResponse
	err := json.NewDecoder(rr.Body).Decode(&respCart)
	assert.NoError(t, err)
	assert.Equal(t, uuidTest, respCart.UserID)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testCreateCartUserDoesNotExist(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(false, nil).Once()

	cartReq := handlers.CartRequest{UserID: uuidTest}
	body, _ := json.Marshal(cartReq)
	req, _ := http.NewRequest("POST", cartsURL, bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.CreateCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockQuerier.AssertNotCalled(t, "CreateCart", mock.Anything, mock.Anything)
	mockUserValidator.AssertExpectations(t)
}

func testCreateCartUserValidationFails(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(false, errors.New(networkErrorMsg)).Once()

	cartReq := handlers.CartRequest{UserID: uuidTest}
	body, _ := json.Marshal(cartReq)
	req, _ := http.NewRequest("POST", cartsURL, bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.CreateCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertNotCalled(t, "CreateCart", mock.Anything, mock.Anything)
	mockUserValidator.AssertExpectations(t)
}

func testCreateCartDBError(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("GetCartByUserID", mock.Anything, pgUUID).Return(repository.Cart{}, pgx.ErrNoRows).Once()
	mockQuerier.On("CreateCart", mock.Anything, pgUUID).Return(repository.Cart{}, errors.New(dbErrorMsg)).Once()

	cartReq := handlers.CartRequest{UserID: uuidTest}
	body, _ := json.Marshal(cartReq)
	req, _ := http.NewRequest("POST", cartsURL, bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.CreateCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testCreateCartInvalidBody(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	req, _ := http.NewRequest("POST", cartsURL, bytes.NewBufferString("{invalid-json}"))
	rr := httptest.NewRecorder()

	handler.CreateCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteCartByUserIDHandler(t *testing.T) {
	t.Run("Success", testDeleteCartByUserIDSuccess)
	t.Run("User Does Not Exist", testDeleteCartByUserIDUserDoesNotExist)
	t.Run("User Validation Fails", testDeleteCartByUserIDUserValidationFails)
	t.Run("Cart Not Found", testDeleteCartByUserIDCartNotFound)
	t.Run("DB Error on Delete", testDeleteCartByUserIDDBError)
	t.Run("Invalid User ID", testDeleteCartByUserIDInvalidID)
}

func testDeleteCartByUserIDSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("DeleteCartByUserID", mock.Anything, pgUUID).Return(nil).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testDeleteCartByUserIDUserDoesNotExist(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(false, nil).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockQuerier.AssertNotCalled(t, "DeleteCartByUserID", mock.Anything, mock.Anything)
	mockUserValidator.AssertExpectations(t)
}

func testDeleteCartByUserIDUserValidationFails(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(false, errors.New(networkErrorMsg)).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertNotCalled(t, "DeleteCartByUserID", mock.Anything, mock.Anything)
	mockUserValidator.AssertExpectations(t)
}

func testDeleteCartByUserIDCartNotFound(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("DeleteCartByUserID", mock.Anything, pgUUID).Return(pgx.ErrNoRows).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testDeleteCartByUserIDDBError(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("DeleteCartByUserID", mock.Anything, pgUUID).Return(errors.New(dbErrorMsg)).Once()

	req, _ := http.NewRequest("DELETE", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testDeleteCartByUserIDInvalidID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	req, _ := http.NewRequest("DELETE", cartsURL+"/invalid-uuid", nil)
	req.SetPathValue("id", invalidUUIDPathTest)
	rr := httptest.NewRecorder()

	handler.DeleteCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockQuerier.AssertNotCalled(t, "DeleteCartByUserID", mock.Anything, mock.Anything)
	mockUserValidator.AssertNotCalled(t, "ValidateUserExists", mock.Anything, mock.Anything)
}

func TestAddProductToCartHandler(t *testing.T) {
	t.Run("Success new cart", testAddProductToCartSuccessNewCart)
	t.Run("Success existing cart", testAddProductToCartSuccessExistingCart)
	t.Run("Invalid user ID", testAddProductToCartInvalidUserID)
	t.Run("Invalid request body", testAddProductToCartInvalidRequestBody)
	t.Run("Product not found", testAddProductToCartProductNotFound)
	t.Run("Product fetcher error", testAddProductToCartProductFetcherError)
	t.Run("DB error on add", testAddProductToCartDBErrorOnAdd)
}

func testAddProductToCartSuccessNewCart(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var userUUID, cartUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = productUUID.Scan(productIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{}, pgx.ErrNoRows).Once()
	mockQuerier.On("CreateCart", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]handlers.ProductByIDResponse{
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
	req, _ := http.NewRequest("POST", cartsURL+"/"+uuidTest+productsPath, bytes.NewBuffer(body))
	req.SetPathValue("id", uuidTest)
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
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var userUUID, cartUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = productUUID.Scan(productIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()

	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]handlers.ProductByIDResponse{
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
	req, _ := http.NewRequest("POST", cartsURL+"/"+uuidTest+productsPath, bytes.NewBuffer(body))
	req.SetPathValue("id", uuidTest)
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
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+"/invalid-uuid/products", bytes.NewBuffer(body))
	req.SetPathValue("id", invalidUUIDPathTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func testAddProductToCartInvalidRequestBody(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	req, _ := http.NewRequest("POST", cartsURL+"/"+uuidTest+productsPath, bytes.NewBufferString("{invalid-json}"))
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func testAddProductToCartProductNotFound(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var userUUID, cartUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]handlers.ProductByIDResponse{}, nil).Once()

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+"/"+uuidTest+productsPath, bytes.NewBuffer(body))
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testAddProductToCartProductFetcherError(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var userUUID, cartUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(nil, errors.New(networkErrorMsg)).Once()

	addProductReq := handlers.AddProductRequest{ProductID: productIDTest, Quantity: 1}
	body, _ := json.Marshal(addProductReq)
	req, _ := http.NewRequest("POST", cartsURL+"/"+uuidTest+productsPath, bytes.NewBuffer(body))
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}

func testAddProductToCartDBErrorOnAdd(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, mockProductFetcher, logger)

	var userUUID, cartUUID, productUUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = productUUID.Scan(productIDTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{ID: cartUUID, UserID: userUUID}, nil).Once()
	mockProductFetcher.On("GetProductsByIDs", mock.Anything, []string{productIDTest}).Return(map[string]handlers.ProductByIDResponse{
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
	req, _ := http.NewRequest("POST", cartsURL+"/"+uuidTest+productsPath, bytes.NewBuffer(body))
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.AddProductToCartHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockQuerier.AssertExpectations(t)
	mockProductFetcher.AssertExpectations(t)
}
