package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
)

const (
	invalidUserIDErrorTitleMsg         = "Invalid User ID"
	invalidUserIDErrorMsg              = "invalid user id"
	userValidationErrorTitleMsg        = "Error validating user existence"
	userValidationErrorMsg             = "error validating user existence"
	userDoesNotExistErrorTitleMsg      = "User Does Not Exist"
	userDoesNotExistErrorMsg           = "user does not exist"
	specifiedUserDoesNotExistsErrorMsg = "The specified user does not exist"

	invalidProductIDErrorMsg = "invalid product id"

	cartNotFoundErrorTitleMsg  = "Cart Not Found"
	cartNotFoundErrorMsg       = "cart not found"
	multipleCartsFoundErrorMsg = "multiple carts found for user"
	failedGetCartErrorMsg      = "failed to get cart by user id"

	requestTimeoutTitleMsg      = "Request Timeout"
	internalServerErrorTitleMsg = "Internal Server Error"
)

type UserValidator interface {
	ValidateUserExists(ctx context.Context, userID string) (bool, error)
}

type ProductFetcher interface {
	GetProductsByIDs(ctx context.Context, ids []string) (map[string]ProductByIDResponse, error)
}

type ProductByIDResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type Handler struct {
	queries        repository.Querier
	userValidator  UserValidator
	productFetcher ProductFetcher
	logger         logs.Logger
}

type CartProductResponse struct {
	ProductID   string  `json:"productId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

type GetCartResponse struct {
	ID         string                `json:"id"`
	UserID     string                `json:"userId"`
	Products   []CartProductResponse `json:"products"`
	TotalPrice float64               `json:"totalPrice"`
}

type AddProductRequest struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

type AddProductResponse struct {
	ID        string  `json:"id"`
	CartID    string  `json:"cartId"`
	ProductID string  `json:"productId"`
	Quantity  int32   `json:"quantity"`
	Price     float64 `json:"price"`
	AddedAt   string  `json:"addedAt"`
}

func newAddProductResponse(cp repository.CartsProduct) AddProductResponse {
	var price float64
	if cp.Price.Valid {
		err := cp.Price.Scan(&price)
		if err != nil {
			price = 0.0
		}
	}

	return AddProductResponse{
		ID:        cp.ID.String(),
		CartID:    cp.CartID.String(),
		ProductID: cp.ProductID.String(),
		Quantity:  cp.Quantity,
		Price:     price,
		AddedAt:   cp.AddedAt.Time.String(),
	}
}

func NewHandler(queries repository.Querier, userValidator UserValidator, productFetcher ProductFetcher, logger logs.Logger) *Handler {
	return &Handler{
		queries:        queries,
		userValidator:  userValidator,
		productFetcher: productFetcher,
		logger:         logger,
	}
}

func (h *Handler) GetCartByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	userID := r.PathValue("userId")
	var uid pgtype.UUID
	if err := uid.Scan(userID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidUserIDErrorTitleMsg, invalidUserIDErrorMsg)
		return
	}

	userExists, err := h.userValidator.ValidateUserExists(ctx, userID)
	if err != nil {
		h.logger.Error(userValidationErrorMsg, "error", err, "user_id", userID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, userValidationErrorTitleMsg)
		return
	}
	if !userExists {
		h.logger.Warn(userDoesNotExistErrorMsg, "user_id", userID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, userDoesNotExistErrorTitleMsg, specifiedUserDoesNotExistsErrorMsg)
		return
	}

	cart, err := h.queries.GetCartByUserID(ctx, uid)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Error(cartNotFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, cartNotFoundErrorTitleMsg, cartNotFoundErrorMsg)
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

	cartProducts, err := h.queries.GetCartProductsByCartID(ctx, cart.ID)
	if err != nil {
		h.logger.Error("failed to get cart products by cart id", "error", err, "cart_id", cart.ID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to get cart products")
		return
	}

	productsID := make([]string, 0, len(cartProducts))
	for _, cp := range cartProducts {
		productsID = append(productsID, cp.ProductID.String())
	}

	productsMap, err := h.productFetcher.GetProductsByIDs(ctx, productsID)
	if err != nil {
		h.logger.Error("failed to fetch products by ids", "error", err, "product_ids", productsID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to fetch products")
		return
	}

	cartProductResponses := make([]CartProductResponse, 0, len(cartProducts))
	var totalPrice float64
	for _, cp := range cartProducts {
		productIDStr := cp.ProductID.String()
		product, exists := productsMap[productIDStr]
		if !exists {
			h.logger.Warn("product not found in fetched products", "product_id", productIDStr)
			continue
		}

		cartProductResponse := CartProductResponse{
			ProductID:   product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
			Quantity:    int(cp.Quantity),
		}
		cartProductResponses = append(cartProductResponses, cartProductResponse)
		totalPrice += product.Price * float64(cp.Quantity)
	}

	response := GetCartResponse{
		ID:         cart.ID.String(),
		UserID:     cart.UserID.String(),
		Products:   cartProductResponses,
		TotalPrice: totalPrice,
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, response)
}

func (h *Handler) DeleteCartByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	userID := r.PathValue("userId")
	var uid pgtype.UUID
	if err := uid.Scan(userID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidUserIDErrorTitleMsg, invalidUserIDErrorMsg)
		return
	}

	userExists, err := h.userValidator.ValidateUserExists(ctx, userID)
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, userValidationErrorTitleMsg)
		return
	}
	if !userExists {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, userDoesNotExistErrorTitleMsg, specifiedUserDoesNotExistsErrorMsg)
		return
	}

	if err := h.queries.DeleteCartByUserID(ctx, uid); err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Warn(cartNotFoundErrorMsg, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusNotFound, cartNotFoundErrorTitleMsg, cartNotFoundErrorMsg)
			return
		default:
			h.logger.Error("failed to delete cart by user id", "error", err, "user_id", userID)
			web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to delete cart")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddProductToCartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	userID := r.PathValue("userId")
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidUserIDErrorTitleMsg, invalidUserIDErrorMsg)
		return
	}

	var req AddProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	cart, err := h.getOrCreateCartByUserID(ctx, userUUID)
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Cart Operation Failed", "Could not get or create a cart for the user")
		return
	}

	productsMap, err := h.productFetcher.GetProductsByIDs(ctx, []string{req.ProductID})
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Product Fetch Failed", "Could not retrieve product details")
		return
	}

	product, exists := productsMap[req.ProductID]
	if !exists {
		h.logger.Warn("product not found", "product_id", req.ProductID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Product Not Found", "The specified product does not exist")
		return
	}

	var productUUID pgtype.UUID
	if err := productUUID.Scan(req.ProductID); err != nil {
		h.logger.Warn(invalidProductIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Product ID", "The product ID format is invalid")
		return
	}

	priceNumeric := pgtype.Numeric{}
	priceStr := fmt.Sprintf("%.2f", product.Price)
	if err := priceNumeric.Scan(priceStr); err != nil {
		h.logger.Error("failed to scan price to numeric", "error", err, "price", product.Price)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to process product price")
		return
	}

	params := repository.AddOrUpdateProductInCartParams{
		CartID:    cart.ID,
		ProductID: productUUID,
		Quantity:  req.Quantity,
		Price:     priceNumeric,
	}

	cartProduct, err := h.queries.AddOrUpdateProductInCart(ctx, params)
	if err != nil {
		h.logger.Error("failed to add or update product in cart", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to update the cart")
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, newAddProductResponse(cartProduct))
}

func (h *Handler) getOrCreateCartByUserID(ctx context.Context, userUUID pgtype.UUID) (repository.Cart, error) {
	cart, err := h.queries.GetCartByUserID(ctx, userUUID)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			h.logger.Info("no cart found for user, creating a new one", "user_id", userUUID.String())
			newCart, createErr := h.queries.CreateCart(ctx, userUUID)
			if createErr != nil {
				h.logger.Error("failed to create a new cart", "error", createErr, "user_id", userUUID.String())
				return repository.Cart{}, createErr
			}
			return newCart, nil
		default:
			h.logger.Error(failedGetCartErrorMsg, "error", err, "user_id", userUUID.String())
			return repository.Cart{}, err
		}
	}
	return cart, nil
}

func (h *Handler) RemoveProductFromCartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !web.CheckContext(ctx, h.logger) {
		web.RespondWithError(w, h.logger, r, http.StatusRequestTimeout, requestTimeoutTitleMsg, web.ReqCancelledMsg)
		return
	}

	userID := r.PathValue("userId")
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		h.logger.Warn(invalidUserIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, invalidUserIDErrorTitleMsg, invalidUserIDErrorMsg)
		return
	}

	productID := r.PathValue("productId")
	var productUUID pgtype.UUID
	if err := productUUID.Scan(productID); err != nil {
		h.logger.Warn(invalidProductIDErrorMsg, "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Product ID", invalidProductIDErrorMsg)
		return
	}

	userExists, err := h.userValidator.ValidateUserExists(ctx, userID)
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, userValidationErrorTitleMsg)
		return
	}
	if !userExists {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, userDoesNotExistErrorTitleMsg, specifiedUserDoesNotExistsErrorMsg)
		return
	}

	product, err := h.productFetcher.GetProductsByIDs(ctx, []string{productID})
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Product Fetch Failed", "Could not retrieve product details")
		return
	}
	if _, exists := product[productID]; !exists {
		h.logger.Warn("product not found", "product_id", productID)
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Product Not Found", "The specified product does not exist")
		return
	}

	err = h.queries.RemoveProductFromCart(ctx, repository.RemoveProductFromCartParams{
		UserID:    userUUID,
		ProductID: productUUID,
	})
	if err != nil {
		h.logger.Error("failed to remove product from cart", "error", err, "user_id", userID, "product_id", productID)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorTitleMsg, "Failed to remove product from cart")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
