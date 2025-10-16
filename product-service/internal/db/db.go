package db

import (
	"context"
	"product-service/internal/repository"
)

type DB interface {
	repository.DBTX
	Ping(ctx context.Context) error
}
