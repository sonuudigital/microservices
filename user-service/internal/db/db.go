package db

import (
	"context"
	"user-service/internal/repository"
)

type DB interface {
	repository.DBTX
	Ping(ctx context.Context) error
}
