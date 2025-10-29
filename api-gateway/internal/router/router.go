package router

import (
	"net/http"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	product_categoriesv1 "github.com/sonuudigital/microservices/gen/product-categories/v1"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
)

type Router struct {
	Logger                  logs.Logger
	RateLimiter             *middlewares.RateLimiterMiddleware
	AuthHandler             *handlers.AuthHandler
	JwtManager              *auth.JWTManager
	UserClient              userv1.UserServiceClient
	ProductClient           productv1.ProductServiceClient
	ProductCategoriesClient product_categoriesv1.ProductCategoriesServiceClient
	CartClient              cartv1.CartServiceClient
}

type authMiddleware func(http.Handler) http.Handler

func New(r Router) (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("gateway is healthy"))
	})

	authMw := middlewares.AuthMiddleware(r.JwtManager, r.Logger)
	userHandler := handlers.NewUserHandler(r.Logger, r.UserClient)
	productHandler := handlers.NewProductHandler(r.Logger, r.ProductClient)
	productCategoriesHandler := handlers.NewProductCategoriesHandler(r.Logger, r.ProductCategoriesClient)
	cartHandler := handlers.NewCartHandler(r.Logger, r.CartClient)

	configAuthAndUserRoutes(mux, r.AuthHandler, userHandler, authMw)
	configProductRoutes(mux, productHandler, authMw)
	configProductCategoriesRoutes(mux, productCategoriesHandler, authMw)
	configCartRoutes(mux, cartHandler, authMw)

	var handler http.Handler = mux
	handler = r.RateLimiter.Middleware(handler)

	return handler, nil
}

func configAuthAndUserRoutes(mux *http.ServeMux, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, authMiddleware authMiddleware) {
	mux.Handle("GET /api/users/{id}", authMiddleware(http.HandlerFunc(userHandler.GetUserByIDHandler)))
	mux.HandleFunc("POST /api/users", userHandler.CreateUserHandler)
	mux.HandleFunc("POST /api/auth/login", authHandler.LoginHandler)
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
