package order

import (
	"context"

	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/clients"
	"github.com/sonuudigital/microservices/shared/logs"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID, userEmail string, totalAmount float64, products []*cartv1.CartProduct) (*orderv1.Order, error)
	CancelOrder(ctx context.Context, orderID string) error
}

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	logger     logs.Logger
	repository OrderRepository
	clients    *clients.Clients
}

func New(logger logs.Logger, repository OrderRepository, clients *clients.Clients) *Server {
	return &Server{
		logger:     logger,
		repository: repository,
		clients:    clients,
	}
}
