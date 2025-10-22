package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		h.logger.Warn("failed to scan product id", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDTitleMsg, invalidProductIDBodyMsg)
		return
	}

	_, err := h.queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Debug(productNotFoundBodyMsg, "id", uid)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, productNotFoundTitleMsg, productNotFoundBodyMsg)
			return
		}
		h.logger.Error("failed to get product before delete", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to get product before delete.")
		return
	}

	err = h.queries.DeleteProduct(ctx, uid)
	if err != nil {
		h.logger.Error("failed to delete product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to delete product.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
