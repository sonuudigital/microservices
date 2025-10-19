package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
)

const (
	invalidUserIDErrorMsg = "invalid user id"

	cartNotFoundErrorMsg       = "cart not found"
	multipleCartsFoundErrorMsg = "multiple carts found for user"
	failedGetCartErrorMsg      = "failed to get cart by user id"

	requestTimeoutTitleMsg      = "Request Timeout"
	internalServerErrorTitleMsg = "Internal Server Error"
)

type UserValidator interface {
	ValidateUserExists(ctx context.Context, userID string) (bool, error)
}

type Handler struct {
	queries       repository.Querier
	userValidator UserValidator
	logger        logs.Logger
}

type CartRequest struct {
	UserID string `json:"userId"`
}

type CartResponse struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
}

func newCartResponse(cart repository.Cart) CartResponse {
	return CartResponse{
		ID:     cart.ID.String(),
		UserID: cart.UserID.String(),
	}
}

func NewHandler(queries repository.Querier, userValidator UserValidator, logger logs.Logger) *Handler {
	return &Handler{
		queries:       queries,
		userValidator: userValidator,
		logger:        logger,
	}
}

func (h *Handler) GetCartByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	userID := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(userID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid User ID", invalidUserIDErrorMsg)
		return
	}

	userExists, err := h.userValidator.ValidateUserExists(ctx, userID)
	if err != nil {
		h.logger.Error("error validating user existence", "error", err, "user_id", userID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Error validating user existence")
		return
	}
	if !userExists {
		h.logger.Warn("user does not exist", "user_id", userID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "User Does Not Exist", "The specified user does not exist")
		return
	}

	cart, err := h.queries.GetCartByUserID(ctx, uid)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Error(cartNotFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, "Cart Not Found", cartNotFoundErrorMsg)
			return
		case pgx.ErrTooManyRows:
			h.logger.Error(multipleCartsFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, multipleCartsFoundErrorMsg)
			return
		default:
			h.logger.Error(failedGetCartErrorMsg, "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, failedGetCartErrorMsg)
			return
		}
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, newCartResponse(cart))
}

func (h *Handler) CreateCartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	var req CartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	var userID pgtype.UUID
	if err := userID.Scan(req.UserID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "User ID malformed", invalidUserIDErrorMsg)
		return
	}

	userExists, err := h.userValidator.ValidateUserExists(ctx, req.UserID)
	if err != nil {
		h.logger.Error("error validating user existence", "error", err, "user_id", req.UserID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Error validating user existence")
		return
	}
	if !userExists {
		h.logger.Warn("user does not exist", "user_id", req.UserID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "User Does Not Exist", "The specified user does not exist")
		return
	}

	_, err = h.queries.GetCartByUserID(ctx, userID)
	if err == nil {
		h.logger.Warn("cart already exists for user", "user_id", req.UserID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Cart Already Exists", "A cart already exists for the specified user")
		return
	}
	if err != pgx.ErrNoRows {
		h.logger.Error("error checking existing cart for user", "error", err, "user_id", req.UserID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Error checking existing cart for user")
		return
	}

	cart, err := h.queries.CreateCart(ctx, userID)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Error(cartNotFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, "Cart Not Found", cartNotFoundErrorMsg)
			return
		case pgx.ErrTooManyRows:
			h.logger.Error(multipleCartsFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, multipleCartsFoundErrorMsg)
			return
		default:
			h.logger.Error(failedGetCartErrorMsg, "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, failedGetCartErrorMsg)
			return
		}
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, newCartResponse(cart))
}
