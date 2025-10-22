package handlers

import (
	"context"
	"errors"

	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	invalidUserIDErrorTitleMsg = "Invalid User ID"
	invalidUserIDErrorMsg      = "invalid user id"

	invalidProductIDErrorMsg      = "invalid product id"
	invalidProductIDErrorTitleMsg = "Invalid Product ID"

	productNotFoundErrorTitleMsg = "Product Not Found"
	productNotFoundErrorMsg      = "The specified product does not exist"

	cartNotFoundErrorTitleMsg  = "Cart Not Found"
	cartNotFoundErrorMsg       = "cart not found"
	multipleCartsFoundErrorMsg = "multiple carts found for user"
	failedGetCartErrorMsg      = "failed to get cart by user id"

	requestTimeoutTitleMsg      = "Request Timeout"
	internalServerErrorTitleMsg = "Internal Server Error"
	userIdHeader                = "X-User-ID"
)

var (
	ErrProductNotFound           = errors.New("product not found")
	ErrInvalidProductID          = errors.New(invalidProductIDErrorMsg)
	ErrProductServiceUnavailable = errors.New("product service unavailable")
	ErrProductInternalError      = errors.New("product service internal error")
)

type ProductByIDResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type ProductFetcher interface {
	GetProductsByIDs(ctx context.Context, ids []string) (map[string]ProductByIDResponse, error)
}

type Handler struct {
	queries        repository.Querier
	productFetcher ProductFetcher
	logger         logs.Logger
}

type CartProductResponse struct {
	ProductID   string  `json:"productId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

type GetCartResponse struct {
	ID         string                `json:"id"`
	UserID     string                `json:"userId"`
	Products   []CartProductResponse `json:"products"`
	TotalPrice float64               `json:"totalPrice"`
}

type AddProductRequest struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

type AddProductResponse struct {
	ID        string  `json:"id"`
	CartID    string  `json:"cartId"`
	ProductID string  `json:"productId"`
	Quantity  int32   `json:"quantity"`
	Price     float64 `json:"price"`
	AddedAt   string  `json:"addedAt"`
}

func newAddProductResponse(cp repository.CartsProduct) AddProductResponse {
	var price float64
	if cp.Price.Valid {
		err := cp.Price.Scan(&price)
		if err != nil {
			price = 0.0
		}
	}

	return AddProductResponse{
		ID:        cp.ID.String(),
		CartID:    cp.CartID.String(),
		ProductID: cp.ProductID.String(),
		Quantity:  cp.Quantity,
		Price:     price,
		AddedAt:   cp.AddedAt.Time.String(),
	}
}

func NewHandler(queries repository.Querier, productFetcher ProductFetcher, logger logs.Logger) *Handler {
	return &Handler{
		queries:        queries,
		productFetcher: productFetcher,
		logger:         logger,
	}
}
