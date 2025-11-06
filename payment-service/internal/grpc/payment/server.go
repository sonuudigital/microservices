package payment

import (
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/payment-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
)

type Server struct {
	paymentv1.UnimplementedPaymentServiceServer
	logger  logs.Logger
	querier repository.Querier
}

func New(logger logs.Logger, querier repository.Querier) *Server {
	return &Server{
		logger:  logger,
		querier: querier,
	}
}
