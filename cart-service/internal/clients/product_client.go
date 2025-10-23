package clients

import (
	"context"
	"fmt"

	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type ProductClient struct {
	client productv1.ProductServiceClient
	logger logs.Logger
}

func NewProductClient(grpcAddr string, logger logs.Logger) (*ProductClient, error) {
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &ProductClient{
		logger: logger,
		client: productv1.NewProductServiceClient(conn),
	}, nil
}

func (c *ProductClient) GetProductsByIDs(ctx context.Context, ids []string) (map[string]grpc_server.Product, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("request to product service canceled or timed out: %w", ctx.Err())
	}

	if len(ids) == 0 {
		return make(map[string]grpc_server.Product), nil
	}

	req := &productv1.GetProductsByIDsRequest{
		Ids: ids,
	}

	resp, err := c.client.GetProductsByIDs(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return nil, fmt.Errorf("invalid product ID")
			case codes.NotFound:
				return nil, fmt.Errorf("product not found")
			case codes.Unavailable:
				return nil, fmt.Errorf("product service unavailable")
			case codes.Canceled:
				return nil, fmt.Errorf("request to product service canceled")
			default:
				return nil, fmt.Errorf("product service internal error")
			}
		}
		return nil, fmt.Errorf("request to product-service failed: %w", err)
	}

	productsMap := make(map[string]grpc_server.Product, len(resp.Products))
	for _, p := range resp.Products {
		productsMap[p.Id] = grpc_server.Product{
			ID:          p.Id,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
		}
	}

	return productsMap, nil
}
