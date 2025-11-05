package payment

import (
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/shared/logs"
)

type Server struct {
	paymentv1.UnimplementedPaymentServiceServer
	logger logs.Logger
}

func New(logger logs.Logger) *Server {
	return &Server{
		logger: logger,
	}
}
