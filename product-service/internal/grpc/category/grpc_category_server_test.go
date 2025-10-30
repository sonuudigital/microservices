package category_test

import (
	"github.com/go-redis/redismock/v9"
	"github.com/sonuudigital/microservices/product-service/internal/grpc/category"
	product_service_mock "github.com/sonuudigital/microservices/product-service/internal/mock"
	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	categoryID                   = "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22"
	malformedID                  = "invalid-uuid"
	categoryName                 = "Sample Category"
	categoryDescription          = "This is a sample category description"
	allProductCategoriesCacheKey = "product_categories:all"
)

func initializeMocksAndServer() (*product_service_mock.MockQuerier, redismock.ClientMock, *category.GRPCServer) {
	mockQuerier := new(product_service_mock.MockQuerier)
	redisClient, redisMock := redismock.NewClientMock()
	server := category.New(logs.NewSlogLogger(), mockQuerier, redisClient)
	return mockQuerier, redisMock, server
}
