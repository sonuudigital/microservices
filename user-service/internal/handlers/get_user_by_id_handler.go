package handlers

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/shared/web"
)

func (h *Handler) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("GetUserByIDHandler received a request")
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutMsg, web.ReqCancelledMsg)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid User ID", "missing user id")
		return
	}

	var uid pgtype.UUID
	err := uid.Scan(id)
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid User ID", "invalid user id")
		return
	}

	user, err := h.queries.GetUserByID(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, "User Not Found", "user not found")
			return
		}
		h.logger.Error("failed to get user by id", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "failed to get user")
		return
	}

	user.Password = ""
	web.RespondWithJSON(w, h.logger, http.StatusOK, user)
}
