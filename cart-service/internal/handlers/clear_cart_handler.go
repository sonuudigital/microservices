package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) ClearCartHandler(w http.ResponseWriter, r *http.Request) {
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

	err := h.queries.ClearCartProductsByUserID(ctx, userUUID)
	if err != nil {
		h.logger.Error("failed to clear cart products by user id", "error", err, "user_id", userID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to clear cart products")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
