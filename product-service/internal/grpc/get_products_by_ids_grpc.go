package grpc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) GetProductsByIDs(ctx context.Context, req *productv1.GetProductsByIDsRequest) (*productv1.GetProductsByIDsResponse, error) {
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

	return &productv1.GetProductsByIDsResponse{Products: ToGRPCProducts(dbProducts)}, nil
}
