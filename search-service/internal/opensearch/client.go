package opensearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

type Client struct {
	*opensearch.Client
}

func NewClient(addresses []string, username, password string) (*Client, error) {
	if len(addresses) == 0 || username == "" || password == "" {
		return nil, fmt.Errorf("addresses, username, and password must be provided")
	}

	config := opensearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	client, err := opensearch.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create opensearch client: %w", err)
	}

	return &Client{Client: client}, nil
}

func (c *Client) Index(ctx context.Context, indexName string, documentID string, body []byte) (*opensearchapi.Response, error) {
	req := opensearchapi.IndexRequest{
		Index:      indexName,
		DocumentID: documentID,
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, c.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute index request: %w", err)
	}

	return res, nil
}

func (c *Client) Delete(ctx context.Context, indexName string, documentID string) (*opensearchapi.Response, error) {
	req := opensearchapi.DeleteRequest{
		Index:      indexName,
		DocumentID: documentID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, c.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute delete request: %w", err)
	}

	return res, nil
}
