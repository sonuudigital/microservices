package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/web"

	"github.com/jackc/pgx/v5/pgtype"
)

func (h *Handler) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	params := repository.CreateProductParams{
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		Code:          req.Code,
		StockQuantity: req.StockQuantity,
	}
	if err := params.Price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Price", err.Error())
		return
	}

	product, err := h.queries.CreateProduct(ctx, params)
	if err != nil {
		h.logger.Error("failed to create product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to create product.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, product)
}
