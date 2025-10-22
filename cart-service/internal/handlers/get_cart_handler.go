package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) GetCartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	userID := r.Header.Get(userIdHeader)
	var uid pgtype.UUID
	if err := uid.Scan(userID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidUserIDErrorTitleMsg, invalidUserIDErrorMsg)
		return
	}

	cart, err := h.queries.GetCartByUserID(ctx, uid)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Error(cartNotFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, cartNotFoundErrorTitleMsg, cartNotFoundErrorMsg)
			return
		case pgx.ErrTooManyRows:
			h.logger.Error(multipleCartsFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, multipleCartsFoundErrorMsg)
			return
		default:
			h.logger.Error(failedGetCartErrorMsg, "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, failedGetCartErrorMsg)
			return
		}
	}

	cartProducts, err := h.queries.GetCartProductsByCartID(ctx, cart.ID)
	if err != nil {
		h.logger.Error("failed to get cart products by cart id", "error", err, "cart_id", cart.ID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to get cart products")
		return
	}

	productsID := make([]string, 0, len(cartProducts))
	for _, cp := range cartProducts {
		productsID = append(productsID, cp.ProductID.String())
	}

	productsMap, err := h.productFetcher.GetProductsByIDs(ctx, productsID)
	if err != nil {
		h.logger.Error("failed to fetch products by ids", "error", err, "product_ids", productsID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to fetch products")
		return
	}

	cartProductResponses := make([]CartProductResponse, 0, len(cartProducts))
	var totalPrice float64
	for _, cp := range cartProducts {
		productIDStr := cp.ProductID.String()
		product, exists := productsMap[productIDStr]
		if !exists {
			h.logger.Warn("product not found in fetched products", "product_id", productIDStr)
			continue
		}

		cartProductResponse := CartProductResponse{
			ProductID:   product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
			Quantity:    int(cp.Quantity),
		}
		cartProductResponses = append(cartProductResponses, cartProductResponse)
		totalPrice += product.Price * float64(cp.Quantity)
	}

	response := GetCartResponse{
		ID:         cart.ID.String(),
		UserID:     cart.UserID.String(),
		Products:   cartProductResponses,
		TotalPrice: totalPrice,
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, response)
}
