package order

import (
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/clients"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
)

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	logger  logs.Logger
	querier repository.Querier
	clients *clients.Clients
}

func New(logger logs.Logger, querier repository.Querier, clients *clients.Clients) *Server {
	return &Server{
		logger:  logger,
		querier: querier,
		clients: clients,
	}
}
