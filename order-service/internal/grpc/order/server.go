package order

import (
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/clients"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/rabbitmq"
)

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	logger   logs.Logger
	querier  repository.Querier
	clients  *clients.Clients
	rabbitmq *rabbitmq.RabbitMQ
}

func New(logger logs.Logger, querier repository.Querier, clients *clients.Clients, rabbitmq *rabbitmq.RabbitMQ) *Server {
	return &Server{
		logger:   logger,
		querier:  querier,
		clients:  clients,
		rabbitmq: rabbitmq,
	}
}
