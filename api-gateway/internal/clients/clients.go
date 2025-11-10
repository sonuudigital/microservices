package clients

import (
	"fmt"

	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClientError struct {
	ServiceName string
	Err         error
}

func (e *GRPCClientError) Error() string {
	return fmt.Sprintf("failed to connect to %s: %v", e.ServiceName, e.Err)
}

type ClientURL struct {
	UserServiceURL    string
	ProductServiceURL string
	CartServiceURL    string
	OrderServiceURL   string
}

type GRPCClient struct {
	userv1.UserServiceClient
	productv1.ProductServiceClient
	product_categoriesv1.ProductCategoriesServiceClient
	cartv1.CartServiceClient
	orderv1.OrderServiceClient
}

func NewGRPCClient(urls ClientURL) (*GRPCClient, error) {
	userServiceClient, err := grpc.NewClient(urls.UserServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, &GRPCClientError{ServiceName: "User Service", Err: err}
	}

	productServiceClient, err := grpc.NewClient(urls.ProductServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, &GRPCClientError{ServiceName: "Product Service", Err: err}
	}

	productCategoriesServiceClient, err := grpc.NewClient(urls.ProductServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, &GRPCClientError{ServiceName: "Product Categories Service", Err: err}
	}

	cartServiceClient, err := grpc.NewClient(urls.CartServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, &GRPCClientError{ServiceName: "Cart Service", Err: err}
	}

	orderServiceClient, err := grpc.NewClient(urls.OrderServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, &GRPCClientError{ServiceName: "Order Service", Err: err}
	}

	return &GRPCClient{
		UserServiceClient:              userv1.NewUserServiceClient(userServiceClient),
		ProductServiceClient:           productv1.NewProductServiceClient(productServiceClient),
		ProductCategoriesServiceClient: product_categoriesv1.NewProductCategoriesServiceClient(productCategoriesServiceClient),
		CartServiceClient:              cartv1.NewCartServiceClient(cartServiceClient),
		OrderServiceClient:             orderv1.NewOrderServiceClient(orderServiceClient),
	}, nil
}
