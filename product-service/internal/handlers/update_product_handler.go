package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDTitleMsg, invalidProductIDBodyMsg)
		return
	}

	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	params := repository.UpdateProductParams{
		ID:            uid,
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		Code:          req.Code,
		StockQuantity: req.StockQuantity,
	}
	if err := params.Price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Price", err.Error())
		return
	}

	product, err := h.queries.UpdateProduct(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, productNotFoundTitleMsg, productNotFoundBodyMsg)
			return
		}
		h.logger.Error("failed to update product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to update product.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, product)
}
