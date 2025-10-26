package category

import (
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	product_categoriesv1.UnimplementedProductCategoriesServiceServer
	queries repository.Querier
}

func New(queries repository.Querier) *GRPCServer {
	return &GRPCServer{
		queries: queries,
	}
}

func toGrpcProductCategory(pc repository.ProductCategory) *product_categoriesv1.ProductCategory {
	var updatedAt *timestamppb.Timestamp
	if pc.UpdatedAt.Valid {
		updatedAt = timestamppb.New(pc.UpdatedAt.Time)
	}

	return &product_categoriesv1.ProductCategory{
		Id:          pc.ID.String(),
		Name:        pc.Name,
		Description: pc.Description.String,
		CreatedAt:   timestamppb.New(pc.CreatedAt.Time),
		UpdatedAt:   updatedAt,
	}
}

func toGrpcProductCategories(categories []repository.ProductCategory) []*product_categoriesv1.ProductCategory {
	grpcCategories := make([]*product_categoriesv1.ProductCategory, len(categories))
	for i, pc := range categories {
		grpcCategories[i] = toGrpcProductCategory(pc)
	}
	return grpcCategories
}
