package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"product-service/internal/repository"
	"shared/logs"
	"shared/web"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	db     repository.DBTX
	logger logs.Logger
}

func NewHandler(db repository.DBTX, logger logs.Logger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

const (
	invalidIDMsg       = "invalid product id"
	productNotFoundMsg = "product not found"
)

type ProductRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	Code          string  `json:"code"`
	StockQuantity int32   `json:"stock_quantity"`
}

func (h *Handler) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		http.Error(w, web.ReqCancelledMsg, http.StatusRequestTimeout)
		return
	}

	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	params := repository.CreateProductParams{
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: true},
		Code:          req.Code,
		StockQuantity: req.StockQuantity,
	}
	if err := params.Price.Scan(fmt.Sprintf("%f", req.Price)); err != nil {
		http.Error(w, "invalid price", http.StatusBadRequest)
		return
	}

	queries := repository.New(h.db)
	product, err := queries.CreateProduct(ctx, params)
	if err != nil {
		h.logger.Error("failed to create product", "error", err)
		http.Error(w, "failed to create product", http.StatusInternalServerError)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, product)
}

func (h *Handler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		http.Error(w, web.ReqCancelledMsg, http.StatusRequestTimeout)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		http.Error(w, invalidIDMsg, http.StatusBadRequest)
		return
	}

	queries := repository.New(h.db)
	product, err := queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, productNotFoundMsg, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get product", "error", err)
		http.Error(w, "failed to get product", http.StatusInternalServerError)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, product)
}

func (h *Handler) ListProductsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		http.Error(w, web.ReqCancelledMsg, http.StatusRequestTimeout)
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

	queries := repository.New(h.db)
	products, err := queries.ListProductsPaginated(ctx, params)
	if err != nil {
		h.logger.Error("failed to list products", "error", err)
		http.Error(w, "failed to list products", http.StatusInternalServerError)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, products)
}

func (h *Handler) UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		http.Error(w, web.ReqCancelledMsg, http.StatusRequestTimeout)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		http.Error(w, invalidIDMsg, http.StatusBadRequest)
		return
	}

	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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
		http.Error(w, "invalid price", http.StatusBadRequest)
		return
	}

	queries := repository.New(h.db)
	product, err := queries.UpdateProduct(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, productNotFoundMsg, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to update product", "error", err)
		http.Error(w, "failed to update product", http.StatusInternalServerError)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, product)
}

func (h *Handler) DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		http.Error(w, web.ReqCancelledMsg, http.StatusRequestTimeout)
		return
	}

	id := r.PathValue("id")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		h.logger.Warn("failed to scan product id", "error", err)
		http.Error(w, invalidIDMsg, http.StatusBadRequest)
		return
	}

	queries := repository.New(h.db)

	_, err := queries.GetProduct(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Debug(productNotFoundMsg, "id", uid)
			http.Error(w, productNotFoundMsg, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get product before delete", "error", err)
		http.Error(w, "failed to get product before delete", http.StatusInternalServerError)
		return
	}

	err = queries.DeleteProduct(ctx, uid)
	if err != nil {
		h.logger.Error("failed to delete product", "error", err)
		http.Error(w, "failed to delete product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
