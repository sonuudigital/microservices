package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/shared/logs"
)

type ProductClient struct {
	httpClient *http.Client
	baseURL    string
	logger     logs.Logger
}

func NewProductClient(baseURL string, logger logs.Logger) *ProductClient {
	return &ProductClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
	}
}

func (c *ProductClient) GetProductsByIDs(ctx context.Context, ids []string) (map[string]handlers.ProductByIDResponse, error) {
	if len(ids) == 0 {
		return make(map[string]handlers.ProductByIDResponse), nil
	}

	idsQueryParam := strings.Join(ids, ",")
	url := fmt.Sprintf("%s/api/products/ids?ids=%s", c.baseURL, idsQueryParam)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create request to product-service", "error", err, "url", url)
		return nil, fmt.Errorf("could not create request to product-service: %w", err)
	}

	c.logger.Debug("sending request to product-service", "url", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to send request to product-service", "error", err, "url", url)
		return nil, fmt.Errorf("request to product-service failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("unexpected status code from product-service", "status_code", resp.StatusCode, "url", url)
		return nil, fmt.Errorf("unexpected status code from product-service: %d", resp.StatusCode)
	}

	var products []handlers.ProductByIDResponse
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		c.logger.Error("failed to decode product-service response", "error", err)
		return nil, fmt.Errorf("failed to decode product-service response: %w", err)
	}

	productsMap := make(map[string]handlers.ProductByIDResponse, len(products))
	for _, p := range products {
		productsMap[p.ID] = p
	}

	return productsMap, nil
}
