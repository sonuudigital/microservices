package router

import (
	"net/http"

	"github.com/sonuudigital/microservices/api-gateway/internal/clients"
	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
)

type authMiddleware func(http.Handler) http.Handler

func New(logger logs.Logger, jwtManager *auth.JWTManager, rateLimiter *middlewares.RateLimiterMiddleware, clients *clients.GRPCClient, searchHandler http.Handler) (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("gateway is healthy"))
	})

	authMw := middlewares.AuthMiddleware(jwtManager, logger)
	authHandler := handlers.NewAuthHandler(logger, jwtManager, clients.UserServiceClient)
	userHandler := handlers.NewUserHandler(logger, clients.UserServiceClient)
	productHandler := handlers.NewProductHandler(logger, clients.ProductServiceClient)
	productCategoriesHandler := handlers.NewProductCategoriesHandler(logger, clients.ProductCategoriesServiceClient)
	cartHandler := handlers.NewCartHandler(logger, clients.CartServiceClient)
	orderHandler := handlers.NewOrderHandler(logger, clients.OrderServiceClient)

	configAuthAndUserRoutes(mux, authHandler, userHandler, authMw)
	configProductRoutes(mux, productHandler, authMw)
	configProductCategoriesRoutes(mux, productCategoriesHandler, authMw)
	configCartRoutes(mux, cartHandler, authMw)
	configOrderRoutes(mux, orderHandler, authMw)
	configSearchRoutes(mux, searchHandler)

	var handler http.Handler = mux
	handler = rateLimiter.Middleware(handler)

	return handler, nil
}

func configAuthAndUserRoutes(mux *http.ServeMux, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, authMiddleware authMiddleware) {
	mux.Handle("GET /api/users/{id}", authMiddleware(http.HandlerFunc(userHandler.GetUserByIDHandler)))
	mux.HandleFunc("POST /api/users", userHandler.CreateUserHandler)
	mux.HandleFunc("POST /api/auth/login", authHandler.LoginHandler)
	mux.HandleFunc("POST /api/auth/logout", authHandler.LogoutHandler)
}

func configProductRoutes(mux *http.ServeMux, productHandler *handlers.ProductHandler, authMiddleware authMiddleware) {
	mux.HandleFunc("GET /api/products/{id}", productHandler.GetProductHandler)
	mux.HandleFunc("GET /api/products", productHandler.ListProductsHandler)
	mux.HandleFunc("GET /api/products/categories/{categoryId}", productHandler.GetProductsByCategoryIDHandler)
	mux.Handle("POST /api/products", authMiddleware(http.HandlerFunc(productHandler.CreateProductHandler)))
	mux.Handle("PUT /api/products/{id}", authMiddleware(http.HandlerFunc(productHandler.UpdateProductHandler)))
	mux.Handle("DELETE /api/products/{id}", authMiddleware(http.HandlerFunc(productHandler.DeleteProductHandler)))
}

func configProductCategoriesRoutes(mux *http.ServeMux, productCategoriesHandler *handlers.ProductCategoriesHandler, authMiddleware authMiddleware) {
	mux.HandleFunc("GET /api/products/categories", productCategoriesHandler.GetProductCategoriesHandler)
	mux.Handle("POST /api/products/categories", authMiddleware(http.HandlerFunc(productCategoriesHandler.CreateProductCategoryHandler)))
	mux.Handle("PUT /api/products/categories", authMiddleware(http.HandlerFunc(productCategoriesHandler.UpdateProductCategoryHandler)))
	mux.Handle("DELETE /api/products/categories/{id}", authMiddleware(http.HandlerFunc(productCategoriesHandler.DeleteProductCategoryHandler)))
}

func configCartRoutes(mux *http.ServeMux, cartHandler *handlers.CartHandler, authMiddleware authMiddleware) {
	mux.Handle("GET /api/carts", authMiddleware(http.HandlerFunc(cartHandler.GetCartHandler)))
	mux.Handle("POST /api/carts/products", authMiddleware(http.HandlerFunc(cartHandler.AddProductToCartHandler)))
	mux.Handle("DELETE /api/carts/products/{productId}", authMiddleware(http.HandlerFunc(cartHandler.RemoveProductFromCartHandler)))
	mux.Handle("DELETE /api/carts/products", authMiddleware(http.HandlerFunc(cartHandler.ClearCartHandler)))
	mux.Handle("DELETE /api/carts", authMiddleware(http.HandlerFunc(cartHandler.DeleteCartHandler)))
}

func configOrderRoutes(mux *http.ServeMux, orderHandler *handlers.OrderHandler, authMiddleware authMiddleware) {
	mux.Handle("POST /api/orders", authMiddleware(http.HandlerFunc(orderHandler.CreateOrderHandler)))
}

func configSearchRoutes(mux *http.ServeMux, searchHandler http.Handler) {
	mux.Handle("GET /api/search/products", searchHandler)
}
