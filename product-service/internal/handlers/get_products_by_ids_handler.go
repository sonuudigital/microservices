package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) GetProductsByIDsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		web.RespondWithJSON(w, h.logger, http.StatusOK, []repository.Product{})
		return
	}

	idStrings := strings.Split(idsParam, ",")
	pgUUIDs := make([]pgtype.UUID, len(idStrings))
	for i, idStr := range idStrings {
		var uid pgtype.UUID
		if err := uid.Scan(idStr); err != nil {
			web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDTitleMsg, fmt.Sprintf("invalid product id: %s", idStr))
			return
		}
		pgUUIDs[i] = uid
	}

	products, err := h.queries.GetProductsByIDs(ctx, pgUUIDs)
	if err != nil {
		h.logger.Error("failed to get products by ids", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to get products by ids.")
		return
	}

	if products == nil {
		products = []repository.Product{}
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, products)
}
