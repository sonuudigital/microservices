package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetCartHandler(t *testing.T) {
	t.Run("Success", testGetCartSuccess)
	t.Run("Not Found", testGetCartNotFound)
	t.Run("Invalid ID", testGetCartInvalidID)
}

func testGetCartSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var userUUID, cartUUID, product1UUID, product2UUID pgtype.UUID
	_ = userUUID.Scan(uuidTest)
	_ = cartUUID.Scan(cartUUIDTest)
	_ = product1UUID.Scan("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13")
	_ = product2UUID.Scan("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14")

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

	req, _ := http.NewRequest("GET", cartsURL, nil)
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()
	handler.GetCartHandler(rr, req)

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
	mockProductFetcher.AssertExpectations(t)
}

func testGetCartNotFound(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockQuerier.On("GetCartByUserID", mock.Anything, pgUUID).Return(repository.Cart{}, pgx.ErrNoRows).Once()

	req, _ := http.NewRequest("GET", cartsURL, nil)
	req.Header.Set(userIDHeader, uuidTest)
	rr := httptest.NewRecorder()

	handler.GetCartHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockQuerier.AssertExpectations(t)
}

func testGetCartInvalidID(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockProductFetcher := new(MockProductFetcher)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockProductFetcher, logger)

	req, _ := http.NewRequest("GET", cartsURL, nil)
	req.Header.Set(userIDHeader, invalidUUIDPathTest)
	rr := httptest.NewRecorder()

	handler.GetCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
