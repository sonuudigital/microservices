package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	productIdTest   = "product-123"
	apiProductsPath = "/api/products/"
)

type mockProductServiceClient struct {
	mock.Mock
}

func (m *mockProductServiceClient) GetProductsByIDs(ctx context.Context, in *productv1.GetProductsByIDsRequest, opts ...grpc.CallOption) (*productv1.GetProductsByIDsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*productv1.GetProductsByIDsResponse), args.Error(1)
}

func (m *mockProductServiceClient) CreateProduct(ctx context.Context, in *productv1.CreateProductRequest, opts ...grpc.CallOption) (*productv1.Product, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*productv1.Product), args.Error(1)
}

func (m *mockProductServiceClient) GetProduct(ctx context.Context, in *productv1.GetProductRequest, opts ...grpc.CallOption) (*productv1.Product, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*productv1.Product), args.Error(1)
}

func (m *mockProductServiceClient) ListProducts(ctx context.Context, in *productv1.ListProductsRequest, opts ...grpc.CallOption) (*productv1.ListProductsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*productv1.ListProductsResponse), args.Error(1)
}

func (m *mockProductServiceClient) UpdateProduct(ctx context.Context, in *productv1.UpdateProductRequest, opts ...grpc.CallOption) (*productv1.Product, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*productv1.Product), args.Error(1)
}

func (m *mockProductServiceClient) DeleteProduct(ctx context.Context, in *productv1.DeleteProductRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func TestGetProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockClient := new(mockProductServiceClient)
	handler := handlers.NewProductHandler(logger, mockClient)

	t.Run("Success", func(t *testing.T) {
		productID := productIdTest
		mockClient.On("GetProduct", mock.Anything, &productv1.GetProductRequest{Id: productID}).
			Return(&productv1.Product{Id: productID, Name: "Test Product"}, nil).Once()

		req, _ := http.NewRequest("GET", apiProductsPath+productID, nil)
		req.SetPathValue("id", productID)
		rr := httptest.NewRecorder()

		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockClient.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		productID := "product-404"
		mockClient.On("GetProduct", mock.Anything, &productv1.GetProductRequest{Id: productID}).
			Return(nil, status.Error(codes.NotFound, "not found")).Once()

		req, _ := http.NewRequest("GET", apiProductsPath+productID, nil)
		req.SetPathValue("id", productID)
		rr := httptest.NewRecorder()

		handler.GetProductHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockClient.AssertExpectations(t)
	})
}

func TestCreateProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockClient := new(mockProductServiceClient)
	handler := handlers.NewProductHandler(logger, mockClient)

	productReq := handlers.ProductRequest{
		Name:          "New Product",
		Description:   "New Description",
		Price:         10.0,
		Code:          "NEW001",
		StockQuantity: 10,
	}
	body, _ := json.Marshal(productReq)

	t.Run("Success", func(t *testing.T) {
		mockClient.On("CreateProduct", mock.Anything, mock.Anything).Return(&productv1.Product{}, nil).Once()

		req, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.CreateProductHandler(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockClient.AssertExpectations(t)
	})
}

func TestListProductsHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockClient := new(mockProductServiceClient)
	handler := handlers.NewProductHandler(logger, mockClient)

	t.Run("Success", func(t *testing.T) {
		mockClient.On("ListProducts", mock.Anything, &productv1.ListProductsRequest{Limit: 10, Offset: 0}).
			Return(&productv1.ListProductsResponse{Products: []*productv1.Product{}}, nil).Once()

		req, _ := http.NewRequest("GET", "/api/products", nil)
		rr := httptest.NewRecorder()

		handler.ListProductsHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockClient.AssertExpectations(t)
	})
}

func TestUpdateProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockClient := new(mockProductServiceClient)
	handler := handlers.NewProductHandler(logger, mockClient)
	productID := productIdTest

	productReq := handlers.ProductRequest{
		Name: "Updated Product",
	}
	body, _ := json.Marshal(productReq)

	t.Run("Success", func(t *testing.T) {
		mockClient.On("UpdateProduct", mock.Anything, mock.Anything).Return(&productv1.Product{}, nil).Once()

		req, _ := http.NewRequest("PUT", apiProductsPath+productID, bytes.NewBuffer(body))
		req.SetPathValue("id", productID)
		rr := httptest.NewRecorder()

		handler.UpdateProductHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockClient.AssertExpectations(t)
	})
}

func TestDeleteProductHandler(t *testing.T) {
	logger := logs.NewSlogLogger()
	mockClient := new(mockProductServiceClient)
	handler := handlers.NewProductHandler(logger, mockClient)
	productID := productIdTest

	t.Run("Success", func(t *testing.T) {
		mockClient.On("DeleteProduct", mock.Anything, &productv1.DeleteProductRequest{Id: productID}).
			Return(&emptypb.Empty{}, nil).Once()

		req, _ := http.NewRequest("DELETE", apiProductsPath+productID, nil)
		req.SetPathValue("id", productID)
		rr := httptest.NewRecorder()

		handler.DeleteProductHandler(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		mockClient.AssertExpectations(t)
	})
}
