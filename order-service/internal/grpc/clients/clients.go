package clients

import (
	"fmt"

	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type clientsURL struct {
	cartServiceURL    string
	paymentServiceURL string
}

func NewClienstURL(cartURL, paymentURL string) clientsURL {
	return clientsURL{
		cartServiceURL:    cartURL,
		paymentServiceURL: paymentURL,
	}
}

type Clients struct {
	cartv1.CartServiceClient
	paymentv1.PaymentServiceClient
}

func NewClients(urls clientsURL) (*Clients, error) {
	cartServiceClient, err := grpc.NewClient(urls.cartServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		err := fmt.Errorf("failed to connect to cart gRPC client: %w", err)
		return nil, err
	}

	paymentServiceClient, err := grpc.NewClient(urls.paymentServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		err := fmt.Errorf("failed to connect to payment gRPC client: %w", err)
		return nil, err
	}

	return &Clients{
		CartServiceClient:    cartv1.NewCartServiceClient(cartServiceClient),
		PaymentServiceClient: paymentv1.NewPaymentServiceClient(paymentServiceClient),
	}, nil
}
