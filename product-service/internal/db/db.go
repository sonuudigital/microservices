package db

import (
	"context"

	"github.com/sonuudigital/microservices/product-service/internal/repository"
)

type DB interface {
	repository.DBTX
	Ping(ctx context.Context) error
}
