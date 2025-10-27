package category_test

import (
	"github.com/sonuudigital/microservices/product-service/internal/grpc/category"
	product_service_mock "github.com/sonuudigital/microservices/product-service/internal/mock"
)

const (
	categoryID          = "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22"
	malformedID         = "invalid-uuid"
	categoryName        = "Sample Category"
	categoryDescription = "This is a sample category description"
)

func initializeQuerierAndServer() (*product_service_mock.MockQuerier, *category.GRPCServer) {
	mockQuerier := new(product_service_mock.MockQuerier)
	server := category.New(mockQuerier)
	return mockQuerier, server
}
