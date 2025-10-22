package clients

import (
	"context"
	"fmt"

	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
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

func (c *ProductClient) GetProductsByIDs(ctx context.Context, ids []string) (map[string]handlers.ProductByIDResponse, error) {
	if len(ids) == 0 {
		return make(map[string]handlers.ProductByIDResponse), nil
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
				return nil, handlers.ErrInvalidProductID
			case codes.NotFound:
				return nil, handlers.ErrProductNotFound
			case codes.Unavailable:
				return nil, handlers.ErrProductServiceUnavailable
			default:
				return nil, handlers.ErrProductInternalError
			}
		}
		return nil, fmt.Errorf("request to product-service failed: %w", err)
	}

	productsMap := make(map[string]handlers.ProductByIDResponse, len(resp.Products))
	for _, p := range resp.Products {
		productsMap[p.Id] = handlers.ProductByIDResponse{
			ID:          p.Id,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
		}
	}

	return productsMap, nil
}
