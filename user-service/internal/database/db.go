package database

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitializePostgresDB() (*pgxpool.Pool, error) {
	ctx, cancell := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancell()

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	if err = db.Ping(ctx); err != nil {
		return nil, err
	}

	if err = executeMigrations(); err != nil {
		return nil, err
	}

	return db, nil
}

func executeMigrations() error {
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "/migrations"
	}
	srcURL := (&url.URL{Scheme: "file", Path: dir}).String()

	m, err := migrate.New(srcURL, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
