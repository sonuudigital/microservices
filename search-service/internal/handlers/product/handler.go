package product

import (
	"context"
	"io"

	"errors"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/sonuudigital/microservices/shared/logs"
)

var (
	ErrNilLogger           = errors.New("logger is nil")
	ErrNilOpensearchClient = errors.New("opensearch client is nil")
)

type Searcher interface {
	Search(ctx context.Context, indexName string, body io.Reader) (*opensearchapi.Response, error)
}

type ProductHandler struct {
	logger   logs.Logger
	searcher Searcher
	index    string
}

func NewProductHandler(logger logs.Logger, searcher Searcher, index string) (*ProductHandler, error) {
	if logger == nil {
		return nil, ErrNilLogger
	}
	if searcher == nil {
		return nil, ErrNilOpensearchClient
	}

	return &ProductHandler{
		logger:   logger,
		searcher: searcher,
		index:    index,
	}, nil
}
