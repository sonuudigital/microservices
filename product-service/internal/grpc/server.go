package grpc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	productv1.UnimplementedProductServiceServer
	queries repository.Querier
}

func NewServer(queries repository.Querier) *Server {
	return &Server{
		queries: queries,
	}
}

func (s *Server) GetProductsByIDs(ctx context.Context, req *productv1.GetProductsByIDsRequest) (*productv1.GetProductsByIDsResponse, error) {
	if len(req.Ids) == 0 {
		return &productv1.GetProductsByIDsResponse{Products: []*productv1.Product{}}, nil
	}

	pgUUIDs := make([]pgtype.UUID, len(req.Ids))
	for i, idStr := range req.Ids {
		var uid pgtype.UUID
		if err := uid.Scan(idStr); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid product id format: %s", idStr)
		}
		pgUUIDs[i] = uid
	}

	dbProducts, err := s.queries.GetProductsByIDs(ctx, pgUUIDs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get products: %v", err)
	}

	grpcProducts := make([]*productv1.Product, len(dbProducts))
	for i, p := range dbProducts {
		grpcProducts[i] = ToGRPCProduct(p)
	}

	return &productv1.GetProductsByIDsResponse{Products: grpcProducts}, nil
}

func ToGRPCProduct(p repository.Product) *productv1.Product {
	var price float64
	if p.Price.Valid {
		_ = p.Price.Scan(&price)
	}

	var updatedAt *timestamppb.Timestamp
	if p.UpdatedAt.Valid {
		updatedAt = timestamppb.New(p.UpdatedAt.Time)
	}

	return &productv1.Product{
		Id:            p.ID.String(),
		Name:          p.Name,
		Description:   p.Description.String,
		Code:          p.Code,
		Price:         price,
		StockQuantity: p.StockQuantity,
		CreatedAt:     timestamppb.New(p.CreatedAt.Time),
		UpdatedAt:     updatedAt,
	}
}
