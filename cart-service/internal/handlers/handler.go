package handlers

import (
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

type Handler struct {
	queries repository.Querier
	logger  logs.Logger
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

func NewHandler(queries repository.Querier, logger logs.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  logger,
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
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid User ID", invalidUserIDErrorMsg)
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
