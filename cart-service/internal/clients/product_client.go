package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sonuudigital/microservices/shared/logs"
)

type ProductClientError struct {
	StatusCode int
	Message    string
}

func (e *ProductClientError) Error() string {
	return e.Message
}

type ProductByIDResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

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

func (c *ProductClient) GetProductsByIDs(ctx context.Context, ids []string) (map[string]ProductByIDResponse, error) {
	if len(ids) == 0 {
		return make(map[string]ProductByIDResponse), nil
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
		c.logger.Warn("unexpected status code from product-service", "status_code", resp.StatusCode, "url", url)
		return nil, &ProductClientError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status code from product-service: %d", resp.StatusCode),
		}
	}

	var products []ProductByIDResponse
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		c.logger.Error("failed to decode product-service response", "error", err)
		return nil, fmt.Errorf("failed to decode product-service response: %w", err)
	}

	productsMap := make(map[string]ProductByIDResponse, len(products))
	for _, p := range products {
		productsMap[p.ID] = p
	}

	return productsMap, nil
}
