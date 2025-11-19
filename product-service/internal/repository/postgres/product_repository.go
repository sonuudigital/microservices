package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/events"
)

const (
	productCreatedEventName = "products_events#product.created"
)

type ProductRepository struct {
	*repository.Queries
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{
		db:      db,
		Queries: repository.New(db),
	}
}

func (r *ProductRepository) CreateProduct(ctx context.Context, arg repository.CreateProductParams) (repository.Product, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return repository.Product{}, err
	}
	defer tx.Rollback(ctx)

	q := r.WithTx(tx)

	product, err := q.CreateProduct(ctx, arg)
	if err != nil {
		return repository.Product{}, err
	}

	encodedProduct, err := r.marshalProductToProductEvent(product)
	if err != nil {
		return repository.Product{}, err
	}

	err = q.CreateOutboxEvent(ctx, repository.CreateOutboxEventParams{
		AggregateID: product.ID,
		EventName:   productCreatedEventName,
		Payload:     encodedProduct,
	})
	if err != nil {
		return repository.Product{}, err
	}

	return product, tx.Commit(ctx)
}

func (r *ProductRepository) marshalProductToProductEvent(p repository.Product) ([]byte, error) {
	priceJSON, err := p.Price.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal price: %w", err)
	}
	priceStr := string(priceJSON)
	if len(priceStr) >= 2 && priceStr[0] == '"' && priceStr[len(priceStr)-1] == '"' {
		priceStr = priceStr[1 : len(priceStr)-1]
	}

	event := events.Product{
		ID:            p.ID.String(),
		CategoryID:    p.CategoryID.String(),
		Name:          p.Name,
		Description:   p.Description.String,
		Price:         priceStr,
		StockQuantity: p.StockQuantity,
	}

	return json.Marshal(event)
}
