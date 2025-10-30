package grpc

import (
	"strconv"
	"time"

	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func productToMap(p *productv1.Product) map[string]any {
	return map[string]any{
		"id":            p.Id,
		"categoryId":    p.CategoryId,
		"name":          p.Name,
		"description":   p.Description,
		"price":         p.Price,
		"stockQuantity": p.StockQuantity,
		"createdAt":     p.CreatedAt.AsTime().Unix(),
		"updatedAt":     p.UpdatedAt.AsTime().Unix(),
	}
}

func mapToProduct(data map[string]string) (*productv1.Product, error) {
	price, err := strconv.ParseFloat(data["price"], 64)
	if err != nil {
		return nil, err
	}

	stockQuantity, err := strconv.ParseInt(data["stockQuantity"], 10, 32)
	if err != nil {
		return nil, err
	}

	createdAt, err := strconv.ParseInt(data["createdAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	updatedAt, err := strconv.ParseInt(data["updatedAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	return &productv1.Product{
		Id:            data["id"],
		CategoryId:    data["categoryId"],
		Name:          data["name"],
		Description:   data["description"],
		Price:         price,
		StockQuantity: int32(stockQuantity),
		CreatedAt:     timestamppb.New(time.Unix(createdAt, 0)),
		UpdatedAt:     timestamppb.New(time.Unix(updatedAt, 0)),
	}, nil
}
