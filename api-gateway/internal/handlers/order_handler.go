package handlers

import (
	"net/http"

	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"google.golang.org/grpc/status"
)

type OrderHandler struct {
	logger      logs.Logger
	orderClient orderv1.OrderServiceClient
}

func NewOrderHandler(logger logs.Logger, orderClient orderv1.OrderServiceClient) *OrderHandler {
	return &OrderHandler{
		logger:      logger,
		orderClient: orderClient,
	}
}

func (h *OrderHandler) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middlewares.GetUserClaims(r)
	if !ok {
		web.RespondWithError(w, h.logger, r, http.StatusUnauthorized, "Unauthorized", userClaimsNotFoundErrMsg)
		return
	}

	order, err := h.orderClient.CreateOrder(r.Context(), &orderv1.CreateOrderRequest{
		UserId: claims.Subject,
	})
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to create order via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, order)
}
