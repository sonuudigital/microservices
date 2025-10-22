package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/clients"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) AddProductToCartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	userID := r.Header.Get(userIdHeader)
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidUserIDErrorTitleMsg, invalidUserIDErrorMsg)
		return
	}

	var req AddProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	cart, err := h.getOrCreateCartByUserID(ctx, userUUID)
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Cart Operation Failed", "Could not get or create a cart for the user")
		return
	}

	productsMap, err := h.productFetcher.GetProductsByIDs(ctx, []string{req.ProductID})
	if err != nil {
		if pce, ok := err.(*clients.ProductClientError); ok {
			switch pce.StatusCode {
			case http.StatusBadRequest:
				h.logger.Warn("invalid product ID format", "product_id", req.ProductID, "error", err)
				web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDErrorTitleMsg, "The product ID format is invalid")
				return
			case http.StatusNotFound:
				h.logger.Warn("product not found in product service", "product_id", req.ProductID)
				web.RespondWithError(w, h.logger, r, http.StatusNotFound, productNotFoundErrorTitleMsg, productNotFoundErrorMsg)
				return
			}
		}
		h.logger.Error("failed to fetch product details", "error", err, "product_id", req.ProductID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Product Fetch Failed", "Could not retrieve product details")
		return
	}

	product, exists := productsMap[req.ProductID]
	if !exists {
		h.logger.Warn("product not found", "product_id", req.ProductID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, productNotFoundErrorTitleMsg, productNotFoundErrorMsg)
		return
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(req.ProductID); err != nil {
		h.logger.Warn(invalidProductIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDErrorTitleMsg, "The product ID format is invalid")
		return
	}

	priceNumeric := pgtype.Numeric{}
	priceStr := fmt.Sprintf("%.2f", product.Price)
	if err := priceNumeric.Scan(priceStr); err != nil {
		h.logger.Error("failed to scan price to numeric", "error", err, "price", product.Price)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to process product price")
		return
	}

	params := repository.AddOrUpdateProductInCartParams{
		CartID:    cart.ID,
		ProductID: productUUID,
		Quantity:  req.Quantity,
		Price:     priceNumeric,
	}

	cartProduct, err := h.queries.AddOrUpdateProductInCart(ctx, params)
	if err != nil {
		h.logger.Error("failed to add or update product in cart", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to update the cart")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, newAddProductResponse(cartProduct))
}

func (h *Handler) getOrCreateCartByUserID(ctx context.Context, userUUID pgtype.UUID) (repository.Cart, error) {
	cart, err := h.queries.GetCartByUserID(ctx, userUUID)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Info("no cart found for user, creating a new one", "user_id", userUUID.String())
			newCart, createErr := h.queries.CreateCart(ctx, userUUID)
			if createErr != nil {
				h.logger.Error("failed to create a new cart", "error", createErr, "user_id", userUUID.String())
				return repository.Cart{}, createErr
			}
			return newCart, nil
		default:
			h.logger.Error(failedGetCartErrorMsg, "error", err, "user_id", userUUID.String())
			return repository.Cart{}, err
		}
	}
	return cart, nil
}
