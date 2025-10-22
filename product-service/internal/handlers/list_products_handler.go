package handlers

import (
	"net/http"
	"strconv"

	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) ListProductsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	params := repository.ListProductsPaginatedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	products, err := h.queries.ListProductsPaginated(ctx, params)
	if err != nil {
		h.logger.Error("failed to list products", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to list products.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, products)
}
