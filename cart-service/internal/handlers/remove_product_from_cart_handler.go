package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) RemoveProductFromCartHandler(w http.ResponseWriter, r *http.Request) {
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

	productID := r.PathValue("productId")
	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		h.logger.Warn(invalidProductIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDErrorTitleMsg, invalidProductIDErrorMsg)
		return
	}

	product, err := h.productFetcher.GetProductsByIDs(ctx, []string{productID})
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Product Fetch Failed", "Could not retrieve product details")
		return
	}
	if _, exists := product[productID]; !exists {
		h.logger.Warn("product not found", "product_id", productID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, productNotFoundErrorTitleMsg, productNotFoundErrorMsg)
		return
	}

	err = h.queries.RemoveProductFromCart(ctx, repository.RemoveProductFromCartParams{
		UserID:    userUUID,
		ProductID: productUUID,
	})
	if err != nil {
		h.logger.Error("failed to remove product from cart", "error", err, "user_id", userID, "product_id", productID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to remove product from cart")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
