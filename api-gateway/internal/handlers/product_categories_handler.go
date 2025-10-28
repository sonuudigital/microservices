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

const (
	failedGRPCParseErrorStatusMsg      = "failed to parse gRPC error status"
	StatusInternalServerErrorTitleMsg  = "Internal Server Error"
	StatusInternalServerErrorDetailMsg = "an internal server error occurred"
	validationErrorTitleMsg            = "Validation Error"
)

type ProductCategoriesHandler struct {
	logger                  logs.Logger
	productCategoriesClient product_categoriesv1.ProductCategoriesServiceClient
}

type CreateProductCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdateProductCategoryRequest struct {
	ID string `json:"id"`
	CreateProductCategoryRequest
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
			h.logger.Error(failedGRPCParseErrorStatusMsg, "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, StatusInternalServerErrorTitleMsg, StatusInternalServerErrorDetailMsg)
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
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, validationErrorTitleMsg, "name is required")
		return
	}

	grpcReq := &product_categoriesv1.CreateProductCategoryRequest{
		Name:        req.Name,
		Description: req.Description,
	}

	resp, err := h.productCategoriesClient.CreateProductCategory(ctx, grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			h.logger.Error(failedGRPCParseErrorStatusMsg, "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, StatusInternalServerErrorTitleMsg, StatusInternalServerErrorDetailMsg)
			return
		}
		h.logger.Error("failed to create product category via gRPC", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, resp)
}

func (h *ProductCategoriesHandler) UpdateProductCategoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, w, r, h.logger) {
		return
	}

	var req UpdateProductCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request body", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	if req.ID == "" || req.Name == "" {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, validationErrorTitleMsg, "Category ID and name is required")
		return
	}

	grpcReq := &product_categoriesv1.UpdateProductCategoryRequest{
		Id:          req.ID,
		Name:        req.Name,
		Description: req.Description,
	}

	_, err := h.productCategoriesClient.UpdateProductCategory(ctx, grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			h.logger.Error(failedGRPCParseErrorStatusMsg, "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, StatusInternalServerErrorTitleMsg, StatusInternalServerErrorDetailMsg)
			return
		}
		h.logger.Error("failed to update product category via gRPC", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusNoContent, nil)
}

func (h *ProductCategoriesHandler) DeleteProductCategoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, w, r, h.logger) {
		return
	}

	categoryID := r.PathValue("id")
	if categoryID == "" {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, validationErrorTitleMsg, "category ID is required")
		return
	}

	grpcReq := &product_categoriesv1.DeleteProductCategoryRequest{
		Id: categoryID,
	}

	_, err := h.productCategoriesClient.DeleteProductCategory(ctx, grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			h.logger.Error(failedGRPCParseErrorStatusMsg, "error", err)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, StatusInternalServerErrorTitleMsg, StatusInternalServerErrorDetailMsg)
			return
		}
		h.logger.Error("failed to delete product category via gRPC", "error", st.Message())
		web.RespondWithGRPCError(w, r, st, h.logger)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusNoContent, nil)
}
