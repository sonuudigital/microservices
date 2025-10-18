package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"github.com/sonuudigital/microservices/user-service/internal/repository"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	requestTimeoutMsg      = "Request Timeout"
	internalServerErrorMsg = "Internal Server Error"
)

type Handler struct {
	queries repository.Querier
	logger  logs.Logger
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewHandler(queries repository.Querier, logger logs.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  logger,
	}
}

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
