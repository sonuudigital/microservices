package handlers

import (
	"encoding/json"
	"net/http"

	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductCategoriesHandler struct {
	logger                  logs.Logger
	productCategoriesClient product_categoriesv1.ProductCategoriesServiceClient
}

type CreateProductCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewProductCategoriesHandler(logger logs.Logger, productCategoriesClient product_categoriesv1.ProductCategoriesServiceClient) *ProductCategoriesHandler {
	return &ProductCategoriesHandler{
		logger:                  logger,
		productCategoriesClient: productCategoriesClient,
	}
}

func (h *ProductCategoriesHandler) GetProductCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, w, r, h.logger) {
		return
	}

	resp, err := h.productCategoriesClient.GetProductCategories(ctx, &emptypb.Empty{})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			h.logger.Error("failed to parse gRPC error status", "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Internal Server Error", "an internal server error occurred")
			return
		}
		h.logger.Error("failed to get product categories via gRPC", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	if len(resp.Categories) == 0 {
		h.logger.Info("no product categories found")
		web.RespondWithJSON(w, h.logger, http.StatusOK, []product_categoriesv1.ProductCategory{})
	} else {
		web.RespondWithJSON(w, h.logger, http.StatusOK, resp.Categories)
	}
}

func (h *ProductCategoriesHandler) CreateProductCategoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, w, r, h.logger) {
		return
	}

	var req CreateProductCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request body", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	if req.Name == "" {
		h.logger.Warn("name is required")
	}

	grpcReq := &product_categoriesv1.CreateProductCategoryRequest{
		Name:        req.Name,
		Description: req.Description,
	}

	resp, err := h.productCategoriesClient.CreateProductCategory(ctx, grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			h.logger.Error("failed to parse gRPC error status", "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Internal Server Error", "an internal server error occurred")
			return
		}
		h.logger.Error("failed to create product category via gRPC", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, resp)
}
