package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/sonuudigital/microservices/shared/web"
)

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) AuthorizeUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutMsg, web.ReqCancelledMsg)
		return
	}

	var authReq AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
		h.logger.Error("failed to decode request body", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	authReq.Email = strings.TrimSpace(authReq.Email)
	authReq.Password = strings.TrimSpace(authReq.Password)

	user, err := h.queries.GetUserByEmail(ctx, authReq.Email)
	if err != nil {
		h.logger.Error("failed to get user by email", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Authorization Failed", "invalid email or password")
		return
	}

	match, err := argon2id.ComparePasswordAndHash(authReq.Password, user.Password)
	if err != nil || !match {
		h.logger.Warn("password mismatch", "error", err, "email", authReq.Email)
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Authorization Failed", "invalid email or password")
		return
	}

	user.Password = ""
	web.RespondWithJSON(w, h.logger, http.StatusOK, user)
}
