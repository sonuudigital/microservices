package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
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

	product, err := h.queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, productNotFoundTitleMsg, productNotFoundBodyMsg)
			return
		}
		h.logger.Error("failed to get product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to get product.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, product)
}
