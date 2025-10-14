package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"shared/logs"
	"strings"
	"user-service/internal/repository"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	db     repository.DBTX
	logger logs.Logger
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

const (
	reqCancelledMsg = "request cancelled"
)

func NewHandler(db repository.DBTX, logger logs.Logger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

func (h *Handler) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("GetUserByIDHandler received a request")
	ctx := r.Context()
	if !h.checkContext(ctx) {
		http.Error(w, reqCancelledMsg, http.StatusRequestTimeout)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	var uid pgtype.UUID
	err := uid.Scan(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	queries := repository.New(h.db)
	user, err := queries.GetUserByID(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get user by id", "error", err)
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	user.Password = ""
	h.respondWithJSON(w, http.StatusOK, user)
}

func (h *Handler) AuthorizeUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !h.checkContext(ctx) {
		http.Error(w, reqCancelledMsg, http.StatusRequestTimeout)
		return
	}

	var authReq AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
		h.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	authReq.Email = strings.TrimSpace(authReq.Email)
	authReq.Password = strings.TrimSpace(authReq.Password)

	queries := repository.New(h.db)
	user, err := queries.GetUserByEmail(ctx, authReq.Email)
	if err != nil {
		h.logger.Error("failed to get user by email", "error", err)
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	match, err := argon2id.ComparePasswordAndHash(authReq.Password, user.Password)
	if err != nil || !match {
		h.logger.Warn("password mismatch", "error", err, "email", authReq.Email)
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	user.Password = ""
	h.respondWithJSON(w, http.StatusOK, user)
}

func (h *Handler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !h.checkContext(ctx) {
		http.Error(w, reqCancelledMsg, http.StatusRequestTimeout)
		return
	}

	var userReq repository.CreateUserParams
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		h.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	hashedPassword, err := argon2id.CreateHash(strings.TrimSpace(userReq.Password), argon2id.DefaultParams)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	userReq.Username = strings.TrimSpace(userReq.Username)
	userReq.Email = strings.TrimSpace(userReq.Email)
	userReq.Password = hashedPassword

	queries := repository.New(h.db)
	_, err = queries.CreateUser(ctx, userReq)
	if err != nil {
		h.logger.Error("failed to create user", "error", err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, nil)
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

func (h *Handler) checkContext(ctx context.Context) bool {
	if ctx.Err() != nil {
		h.logger.Error(reqCancelledMsg, ctx.Err())
		return false
	}
	return true
}
