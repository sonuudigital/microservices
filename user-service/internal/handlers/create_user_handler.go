package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/sonuudigital/microservices/shared/web"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
)

func (h *Handler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutMsg, web.ReqCancelledMsg)
		return
	}

	var userReq repository.CreateUserParams
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		h.logger.Error("failed to decode request body", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	hashedPassword, err := argon2id.CreateHash(strings.TrimSpace(userReq.Password), argon2id.DefaultParams)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "failed to hash password")
		return
	}

	userReq.Username = strings.TrimSpace(userReq.Username)
	userReq.Email = strings.TrimSpace(userReq.Email)
	userReq.Password = hashedPassword

	_, err = h.queries.CreateUser(ctx, userReq)
	if err != nil {
		h.logger.Error("failed to create user", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "failed to create user")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, nil)
}
