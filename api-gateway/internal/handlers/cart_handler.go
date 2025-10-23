package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"google.golang.org/grpc/status"
)

const (
	userClaimsNotFoundErrMsg = "user claims not found"
)

type CartHandler struct {
	logger     logs.Logger
	cartClient cartv1.CartServiceClient
}

func NewCartHandler(logger logs.Logger, cartClient cartv1.CartServiceClient) *CartHandler {
	return &CartHandler{
		logger:     logger,
		cartClient: cartClient,
	}
}

type AddProductToCartRequest struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

func (h *CartHandler) GetCartHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middlewares.GetUserClaims(r)
	if !ok {
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Unauthorized", userClaimsNotFoundErrMsg)
		return
	}

	grpcReq := &cartv1.GetCartRequest{
		UserId: claims.Subject,
	}

	res, err := h.cartClient.GetCart(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to get cart via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, res)
}

func (h *CartHandler) AddProductToCartHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middlewares.GetUserClaims(r)
	if !ok {
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Unauthorized", userClaimsNotFoundErrMsg)
		return
	}

	var req AddProductToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	grpcReq := &cartv1.AddProductToCartRequest{
		UserId:    claims.Subject,
		ProductId: req.ProductID,
		Quantity:  req.Quantity,
	}

	res, err := h.cartClient.AddProductToCart(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to add product to cart via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, res)
}

func (h *CartHandler) RemoveProductFromCartHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middlewares.GetUserClaims(r)
	if !ok {
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Unauthorized", userClaimsNotFoundErrMsg)
		return
	}

	productID := r.PathValue("productId")

	grpcReq := &cartv1.RemoveProductFromCartRequest{
		UserId:    claims.Subject,
		ProductId: productID,
	}

	_, err := h.cartClient.RemoveProductFromCart(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to remove product from cart via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) ClearCartHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middlewares.GetUserClaims(r)
	if !ok {
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Unauthorized", userClaimsNotFoundErrMsg)
		return
	}

	grpcReq := &cartv1.ClearCartRequest{
		UserId: claims.Subject,
	}

	_, err := h.cartClient.ClearCart(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to clear cart via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) DeleteCartHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middlewares.GetUserClaims(r)
	if !ok {
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Unauthorized", userClaimsNotFoundErrMsg)
		return
	}

	grpcReq := &cartv1.DeleteCartRequest{
		UserId: claims.Subject,
	}

	_, err := h.cartClient.DeleteCart(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to delete cart via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
