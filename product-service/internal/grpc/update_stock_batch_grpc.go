package grpc

import (
	"context"
	"encoding/json"

	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *GRPCServer) UpdateStockBatch(ctx context.Context, req *productv1.UpdateStockBatchRequest) (*emptypb.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	if len(req.GetUpdates()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no updates provided")
	}

	jsonReq, err := json.Marshal(req.Updates)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal update stock batch request: %v", err)
	}

	if _, err := s.queries.UpdateStockBatch(ctx, jsonReq); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update stock batch: %v", err)
	}

	go s.invaliteCacheForUpdatedStocks(req.Updates)

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) invaliteCacheForUpdatedStocks(products []*productv1.StockUpdate) {
	for _, product := range products {
		if err := s.deleteProductCache(product.Id); err != nil {
			s.logger.Error("failed to delete product cache for id %s: %v", product.Id, err)
		}
	}
}
