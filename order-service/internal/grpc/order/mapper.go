package order

import (
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapRepositoryToGRPC(o *repository.Order, statusName string) (*orderv1.Order, error) {
	totalAmount, err := o.TotalAmount.Float64Value()
	if err != nil {
		return nil, err
	}

	return &orderv1.Order{
		Id:          o.ID.String(),
		UserId:      o.UserID.String(),
		TotalAmount: totalAmount.Float64,
		Status:      statusName,
		CreatedAt:   timestamppb.New(o.CreatedAt.Time),
	}, nil
}
