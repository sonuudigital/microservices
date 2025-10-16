package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sonuudigital/microservices/product-service/internal/handlers"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	uuidTest    = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	productsURL = "/api/products"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateProduct(ctx context.Context, arg repository.CreateProductParams) (repository.Product, error) {
	args := m.Called(ctx, arg)
	if p, ok := args.Get(0).(repository.Product); ok {
		return p, args.Error(1)
	}
	return repository.Product{}, args.Error(1)
}

func (m *MockQuerier) DeleteProduct(ctx context.Context, id pgtype.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) GetProduct(ctx context.Context, id pgtype.UUID) (repository.Product, error) {
	args := m.Called(ctx, id)
	if p, ok := args.Get(0).(repository.Product); ok {
		return p, args.Error(1)
	}
	return repository.Product{}, args.Error(1)
}

func (m *MockQuerier) ListProductsPaginated(ctx context.Context, arg repository.ListProductsPaginatedParams) ([]repository.Product, error) {
	args := m.Called(ctx, arg)
	if p, ok := args.Get(0).([]repository.Product); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuerier) UpdateProduct(ctx context.Context, arg repository.UpdateProductParams) (repository.Product, error) {
	args := m.Called(ctx, arg)
	if p, ok := args.Get(0).(repository.Product); ok {
		return p, args.Error(1)
	}
	return repository.Product{}, args.Error(1)
}

func TestCreateProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	productRequest := handlers.ProductRequest{
		Name:          "Test Product",
		Description:   "Test Description",
		Price:         99.99,
		Code:          "TEST001",
		StockQuantity: 100,
	}
	body, _ := json.Marshal(productRequest)

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("CreateProduct", mock.Anything, mock.AnythingOfType("repository.CreateProductParams")).Return(repository.Product{}, nil).Once()

		req, err := http.NewRequest("POST", productsURL, bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.CreateProductHandler(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("CreateProduct", mock.Anything, mock.AnythingOfType("repository.CreateProductParams")).Return(repository.Product{}, errors.New("db error")).Once()

		req, err := http.NewRequest("POST", productsURL, bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.CreateProductHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockQuerier.AssertExpectations(t)
	})
}

func TestGetProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr := uuidTest
		var pgUUID pgtype.UUID
		err := pgUUID.Scan(uuidStr)
		assert.NoError(t, err)

		product := repository.Product{ID: pgUUID, Name: "Test Product"}
		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(product, nil).Once()

		req, err := http.NewRequest("GET", productsURL+"/"+uuidStr, nil)
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respProduct repository.Product
		err = json.NewDecoder(rr.Body).Decode(&respProduct)
		assert.NoError(t, err)
		assert.Equal(t, product.Name, respProduct.Name)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr := uuidTest
		var pgUUID pgtype.UUID
		err := pgUUID.Scan(uuidStr)
		assert.NoError(t, err)

		mockQuerier.On("GetProduct", mock.Anything, pgUUID).Return(repository.Product{}, pgx.ErrNoRows).Once()

		req, err := http.NewRequest("GET", productsURL+"/"+uuidStr, nil)
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		req, err := http.NewRequest("GET", "/api/products/invalid-id", nil)
		assert.NoError(t, err)
		req.SetPathValue("id", "invalid-id")

		rr := httptest.NewRecorder()
		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestListProductsHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		products := []repository.Product{{Name: "Product 1"}, {Name: "Product 2"}}
		mockQuerier.On("ListProductsPaginated", mock.Anything, mock.AnythingOfType("repository.ListProductsPaginatedParams")).Return(products, nil).Once()

		req, err := http.NewRequest("GET", productsURL, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ListProductsHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respProducts []repository.Product
		err = json.NewDecoder(rr.Body).Decode(&respProducts)
		assert.NoError(t, err)
		assert.Equal(t, len(products), len(respProducts))
		mockQuerier.AssertExpectations(t)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("ListProductsPaginated", mock.Anything, mock.AnythingOfType("repository.ListProductsPaginatedParams")).Return(nil, errors.New("db error")).Once()

		req, err := http.NewRequest("GET", productsURL, nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ListProductsHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockQuerier.AssertExpectations(t)
	})
}

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
