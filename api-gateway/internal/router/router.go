package router

import (
	"net/http"
	"os"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	productv1 "github.com/sonuudigital/microservices/gen/product/v1"
	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
)

type authMiddleware func(http.Handler) http.Handler

func New(authHandler *handlers.AuthHandler, jwtManager *auth.JWTManager, logger logs.Logger, userClient userv1.UserServiceClient, productClient productv1.ProductServiceClient) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("gateway is healthy"))
	})

	authMw := middlewares.AuthMiddleware(jwtManager, logger)

	userHandler := handlers.NewUserHandler(logger, userClient)

	err := configAuthAndUserRoutes(mux, authHandler, userHandler, authMw)
	if err != nil {
		return nil, err
	}

	productHandler := handlers.NewProductHandler(logger, productClient)
	err = configProductRoutes(mux, productHandler, authMw)
	if err != nil {
		return nil, err
	}

	err = configCartRoutes(mux, authMw, logger)
	if err != nil {
		return nil, err
	}

	return mux, nil
}

func configAuthAndUserRoutes(mux *http.ServeMux, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, authMiddleware authMiddleware) error {
	mux.Handle("GET /api/users/{id}", authMiddleware(http.HandlerFunc(userHandler.GetUserByIDHandler)))
	mux.HandleFunc("POST /api/users", userHandler.CreateUserHandler)
	mux.HandleFunc("POST /api/auth/login", authHandler.LoginHandler)

	return nil
}

func configProductRoutes(mux *http.ServeMux, productHandler *handlers.ProductHandler, authMiddleware authMiddleware) error {
	mux.HandleFunc("GET /api/products/{id}", productHandler.GetProductHandler)
	mux.HandleFunc("GET /api/products", productHandler.ListProductsHandler)
	mux.Handle("POST /api/products", authMiddleware(http.HandlerFunc(productHandler.CreateProductHandler)))
	mux.Handle("PUT /api/products/{id}", authMiddleware(http.HandlerFunc(productHandler.UpdateProductHandler)))
	mux.Handle("DELETE /api/products/{id}", authMiddleware(http.HandlerFunc(productHandler.DeleteProductHandler)))

	return nil
}

func configCartRoutes(mux *http.ServeMux, authMiddleware authMiddleware, logger logs.Logger) error {
	cartServiceURL := os.Getenv("CART_SERVICE_URL")
	cartProxy, err := handlers.NewProxyHandler(cartServiceURL, logger)
	if err != nil {
		logger.Error("failed to create cart service proxy", "error", err)
		return err
	}
	protectedCartProxy := authMiddleware(cartProxy)

	mux.Handle("GET /api/carts", protectedCartProxy)
	mux.Handle("DELETE /api/carts", protectedCartProxy)
	mux.Handle("POST /api/carts/products", protectedCartProxy)
	mux.Handle("DELETE /api/carts/products/{productId}", protectedCartProxy)
	mux.Handle("DELETE /api/carts/products", protectedCartProxy)

	return nil
}
