package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sonuudigital/microservices/shared/logs"
)

type UserClient struct {
	httpClient *http.Client
	baseURL    string
	logger     logs.Logger
}

func NewUserClient(baseURL string, logger logs.Logger) *UserClient {
	return &UserClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
	}
}

func (c *UserClient) ValidateUserExists(ctx context.Context, userID string) (bool, error) {
	url := fmt.Sprintf("%s/api/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create request to user-service", "error", err, "url", url)
		return false, fmt.Errorf("could not create request to user-service: %w", err)
	}

	c.logger.Debug("sending request to user-service", "url", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to send request to user-service", "error", err, "url", url)
		return false, fmt.Errorf("request to user-service failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		c.logger.Debug("user validation successful", "user_id", userID, "status_code", http.StatusOK)
		return true, nil
	case http.StatusNotFound:
		c.logger.Warn("user not found in user-service", "user_id", userID, "status_code", http.StatusNotFound)
		return false, nil
	default:
		c.logger.Error("unexpected status code from user-service", "status_code", resp.StatusCode, "user_id", userID)
		return false, fmt.Errorf("unexpected status code from user-service: %d", resp.StatusCode)
	}
}
