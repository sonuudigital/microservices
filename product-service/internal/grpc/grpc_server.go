package grpc

import (
	"strconv"

	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	productv1.UnimplementedProductServiceServer
	queries repository.Querier
}

func NewServer(queries repository.Querier) *GRPCServer {
	return &GRPCServer{
		queries: queries,
	}
}

func toGRPCProduct(p repository.Product) *productv1.Product {
	var price float64
	if p.Price.Valid {
		v, err := p.Price.Value()
		if err == nil {
			if s, ok := v.(string); ok {
				price, _ = strconv.ParseFloat(s, 64)
			}
		}
	}

	var updatedAt *timestamppb.Timestamp
	if p.UpdatedAt.Valid {
		updatedAt = timestamppb.New(p.UpdatedAt.Time)
	}

	return &productv1.Product{
		Id:            p.ID.String(),
		Name:          p.Name,
		Description:   p.Description.String,
		Price:         price,
		StockQuantity: p.StockQuantity,
		CreatedAt:     timestamppb.New(p.CreatedAt.Time),
		UpdatedAt:     updatedAt,
	}
}

func ToGRPCProducts(products []repository.Product) []*productv1.Product {
	grpcProducts := make([]*productv1.Product, len(products))
	for i, p := range products {
		grpcProducts[i] = toGRPCProduct(p)
	}
	return grpcProducts
}
