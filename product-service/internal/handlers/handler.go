package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	invalidProductIDTitleMsg = "Invalid Product ID"
	invalidProductIDBodyMsg  = "invalid product id"

	productNotFoundTitleMsg = "Product Not Found"
	productNotFoundBodyMsg  = "product not found"

	requestTimeoutTitleMsg      = "Request Timeout"
	internalServerErrorTitleMsg = "Internal Server Error"
)

type Handler struct {
	queries repository.Querier
	logger  logs.Logger
}

func NewHandler(queries repository.Querier, logger logs.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  logger,
	}
}

type ProductRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	Code          string  `json:"code"`
	StockQuantity int32   `json:"stockQuantity"`
}

func (h *Handler) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	params := repository.CreateProductParams{
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		Code:          req.Code,
		StockQuantity: req.StockQuantity,
	}
	if err := params.Price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Price", err.Error())
		return
	}

	product, err := h.queries.CreateProduct(ctx, params)
	if err != nil {
		h.logger.Error("failed to create product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to create product.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, product)
}

func (h *Handler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDTitleMsg, invalidProductIDBodyMsg)
		return
	}

	product, err := h.queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, productNotFoundTitleMsg, productNotFoundBodyMsg)
			return
		}
		h.logger.Error("failed to get product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to get product.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, product)
}

func (h *Handler) ListProductsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	params := repository.ListProductsPaginatedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	products, err := h.queries.ListProductsPaginated(ctx, params)
	if err != nil {
		h.logger.Error("failed to list products", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to list products.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, products)
}

func (h *Handler) UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDTitleMsg, invalidProductIDBodyMsg)
		return
	}

	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	params := repository.UpdateProductParams{
		ID:            uid,
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		Code:          req.Code,
		StockQuantity: req.StockQuantity,
	}
	if err := params.Price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Price", err.Error())
		return
	}

	product, err := h.queries.UpdateProduct(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, productNotFoundTitleMsg, productNotFoundBodyMsg)
			return
		}
		h.logger.Error("failed to update product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to update product.")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, product)
}

func (h *Handler) DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		h.logger.Warn("failed to scan product id", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidProductIDTitleMsg, invalidProductIDBodyMsg)
		return
	}

	_, err := h.queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Debug(productNotFoundBodyMsg, "id", uid)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, productNotFoundTitleMsg, productNotFoundBodyMsg)
			return
		}
		h.logger.Error("failed to get product before delete", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to get product before delete.")
		return
	}

	err = h.queries.DeleteProduct(ctx, uid)
	if err != nil {
		h.logger.Error("failed to delete product", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to delete product.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
