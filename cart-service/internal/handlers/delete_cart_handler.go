package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) DeleteCartHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := h.queries.DeleteCartByUserID(ctx, uid); err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Warn(cartNotFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, cartNotFoundErrorTitleMsg, cartNotFoundErrorMsg)
			return
		default:
			h.logger.Error("failed to delete cart by user id", "error", err, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to delete cart")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
