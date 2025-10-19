package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

const (
	uuidTest = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	cartsURL = "/api/carts"
)

type MockQuerier struct {
	mock.Mock
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
	if c, ok := args.Get(0).(repository.Cart); ok {
		return c, args.Error(1)
	}
	return repository.Cart{}, args.Error(1)
}

type MockUserValidator struct {
	mock.Mock
}

func (m *MockUserValidator) ValidateUserExists(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func TestGetCartByUserIDHandler(t *testing.T) {
	t.Run("Success", testGetCartByUserIDSuccess)
	t.Run("Not Found", testGetCartByUserIDNotFound)
	t.Run("Invalid ID", testGetCartByUserIDInvalidID)
}

func testGetCartByUserIDSuccess(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	expectedCart := repository.Cart{ID: pgUUID, UserID: pgUUID}
	mockQuerier.On("GetCartByUserID", mock.Anything, pgUUID).Return(expectedCart, nil).Once()

	req, _ := http.NewRequest("GET", cartsURL+"/"+uuidTest, nil)
	req.SetPathValue("id", uuidTest)
	rr := httptest.NewRecorder()

	handler.GetCartByUserIDHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var respCart handlers.CartResponse
	err := json.NewDecoder(rr.Body).Decode(&respCart)
	assert.NoError(t, err)
	assert.Equal(t, uuidTest, respCart.UserID)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testGetCartByUserIDNotFound(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

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
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

	req, _ := http.NewRequest("GET", cartsURL+"/invalid-uuid", nil)
	req.SetPathValue("id", "invalid-uuid")
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
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("CreateCart", mock.Anything, pgUUID).Return(repository.Cart{ID: pgUUID, UserID: pgUUID}, nil).Once()

	cartReq := handlers.CartRequest{UserID: uuidTest}
	body, _ := json.Marshal(cartReq)
	req, _ := http.NewRequest("POST", cartsURL, bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.CreateCartHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var respCart handlers.CartResponse
	err := json.NewDecoder(rr.Body).Decode(&respCart)
	assert.NoError(t, err)
	assert.Equal(t, uuidTest, respCart.UserID)
	mockQuerier.AssertExpectations(t)
	mockUserValidator.AssertExpectations(t)
}

func testCreateCartUserDoesNotExist(t *testing.T) {
	mockQuerier := new(MockQuerier)
	mockUserValidator := new(MockUserValidator)
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

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
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(false, errors.New("network error")).Once()

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
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidTest)

	mockUserValidator.On("ValidateUserExists", mock.Anything, uuidTest).Return(true, nil).Once()
	mockQuerier.On("CreateCart", mock.Anything, pgUUID).Return(repository.Cart{}, errors.New("db error")).Once()

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
	logger := logs.NewSlogLogger()
	handler := handlers.NewHandler(mockQuerier, mockUserValidator, logger)

	req, _ := http.NewRequest("POST", cartsURL, bytes.NewBufferString("{invalid-json}"))
	rr := httptest.NewRecorder()

	handler.CreateCartHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
