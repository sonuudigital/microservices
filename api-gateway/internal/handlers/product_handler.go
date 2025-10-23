package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"google.golang.org/grpc/status"
)

type ProductHandler struct {
	logger        logs.Logger
	productClient productv1.ProductServiceClient
}

func NewProductHandler(logger logs.Logger, productClient productv1.ProductServiceClient) *ProductHandler {
	return &ProductHandler{
		logger:        logger,
		productClient: productClient,
	}
}

type ProductRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	Code          string  `json:"code"`
	StockQuantity int32   `json:"stockQuantity"`
}

func (h *ProductHandler) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	grpcReq := &productv1.CreateProductRequest{
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		Code:          req.Code,
		StockQuantity: req.StockQuantity,
	}

	res, err := h.productClient.CreateProduct(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to create product via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, res)
}

func (h *ProductHandler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	grpcReq := &productv1.GetProductRequest{
		Id: id,
	}

	res, err := h.productClient.GetProduct(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to get product via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, res)
}

func (h *ProductHandler) ListProductsHandler(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	grpcReq := &productv1.ListProductsRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	res, err := h.productClient.ListProducts(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to list products via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, res.Products)
}

func (h *ProductHandler) UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	grpcReq := &productv1.UpdateProductRequest{
		Id:            id,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		Code:          req.Code,
		StockQuantity: req.StockQuantity,
	}

	res, err := h.productClient.UpdateProduct(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to update product via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, res)
}

func (h *ProductHandler) DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	grpcReq := &productv1.DeleteProductRequest{
		Id: id,
	}

	_, err := h.productClient.DeleteProduct(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to delete product via grpc", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}