package payment

import (
	"fmt"

	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/payment-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toGRPCPayment(p *repository.Payment) (*paymentv1.Payment, error) {
	amountFloat, err := p.Amount.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount to float64: %w", err)
	}

	if !amountFloat.Valid {
		return nil, fmt.Errorf("amount is null")
	}

	return &paymentv1.Payment{
		Id:        p.ID.String(),
		OrderId:   p.OrderID.String(),
		UserId:    p.UserID.String(),
		Amount:    amountFloat.Float64,
		Status:    p.Status.String(),
		CreatedAt: timestamppb.New(p.CreatedAt.Time),
	}, nil
}
