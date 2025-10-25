package router

import (
	"net/http"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
)

type authMiddleware func(http.Handler) http.Handler

func New(
	authHandler *handlers.AuthHandler,
	jwtManager *auth.JWTManager,
	logger logs.Logger,
	userClient userv1.UserServiceClient,
	productClient productv1.ProductServiceClient,
	cartClient cartv1.CartServiceClient,
) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("gateway is healthy"))
	})

	authMw := middlewares.AuthMiddleware(jwtManager, logger)
	userHandler := handlers.NewUserHandler(logger, userClient)
	productHandler := handlers.NewProductHandler(logger, productClient)
	cartHandler := handlers.NewCartHandler(logger, cartClient)

	configAuthAndUserRoutes(mux, authHandler, userHandler, authMw)
	configProductRoutes(mux, productHandler, authMw)
	configCartRoutes(mux, cartHandler, authMw)

	return mux, nil
}

func configAuthAndUserRoutes(mux *http.ServeMux, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, authMiddleware authMiddleware) {
	mux.Handle("GET /api/users/{id}", authMiddleware(http.HandlerFunc(userHandler.GetUserByIDHandler)))
	mux.HandleFunc("POST /api/users", userHandler.CreateUserHandler)
	mux.HandleFunc("POST /api/auth/login", authHandler.LoginHandler)
}

func configProductRoutes(mux *http.ServeMux, productHandler *handlers.ProductHandler, authMiddleware authMiddleware) {
	mux.HandleFunc("GET /api/products/{id}", productHandler.GetProductHandler)
	mux.HandleFunc("GET /api/products", productHandler.ListProductsHandler)
	mux.HandleFunc("GET /api/products/category/{categoryId}", productHandler.GetProductsByCategoryIDHandler)
	mux.Handle("POST /api/products", authMiddleware(http.HandlerFunc(productHandler.CreateProductHandler)))
	mux.Handle("PUT /api/products/{id}", authMiddleware(http.HandlerFunc(productHandler.UpdateProductHandler)))
	mux.Handle("DELETE /api/products/{id}", authMiddleware(http.HandlerFunc(productHandler.DeleteProductHandler)))
}

func configCartRoutes(mux *http.ServeMux, cartHandler *handlers.CartHandler, authMiddleware authMiddleware) {
	mux.Handle("GET /api/carts", authMiddleware(http.HandlerFunc(cartHandler.GetCartHandler)))
	mux.Handle("POST /api/carts/products", authMiddleware(http.HandlerFunc(cartHandler.AddProductToCartHandler)))
	mux.Handle("DELETE /api/carts/products/{productId}", authMiddleware(http.HandlerFunc(cartHandler.RemoveProductFromCartHandler)))
	mux.Handle("DELETE /api/carts/products", authMiddleware(http.HandlerFunc(cartHandler.ClearCartHandler)))
	mux.Handle("DELETE /api/carts", authMiddleware(http.HandlerFunc(cartHandler.DeleteCartHandler)))
}
