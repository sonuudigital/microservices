package router

import (
	"net/http"
	"os"

	"github.com/sonuudigital/microservices/api-gateway/internal/handlers"
	"github.com/sonuudigital/microservices/api-gateway/internal/middlewares"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
)

type authMiddleware func(http.Handler) http.Handler

func New(authHandler *handlers.AuthHandler, jwtManager *auth.JWTManager, logger logs.Logger) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("gateway is healthy"))
	})

	authMw := middlewares.AuthMiddleware(jwtManager, logger)

	err := configAuthAndUserRoutes(mux, authHandler, authMw, logger)
	if err != nil {
		return nil, err
	}

	err = configProductRoutes(mux, authMw, logger)
	if err != nil {
		return nil, err
	}

	return mux, nil
}

func configAuthAndUserRoutes(mux *http.ServeMux, authHandler *handlers.AuthHandler, authMiddleware authMiddleware, logger logs.Logger) error {
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	userProxy, err := handlers.NewProxyHandler(userServiceURL, logger)
	if err != nil {
		logger.Error("failed to create user service proxy", "error", err)
		return err
	}
	protectedUserProxy := authMiddleware(userProxy)

	mux.Handle("GET /api/users/{id}", protectedUserProxy)
	mux.Handle("POST /api/users", userProxy)
	mux.HandleFunc("POST /api/auth/login", authHandler.LoginHandler)

	return nil
}

func configProductRoutes(mux *http.ServeMux, authMiddleware authMiddleware, logger logs.Logger) error {
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	productProxy, err := handlers.NewProxyHandler(productServiceURL, logger)
	if err != nil {
		logger.Error("failed to create product service proxy", "error", err)
		return err
	}
	protectedProductProxy := authMiddleware(productProxy)

	mux.Handle("GET /api/products/{id}", productProxy)
	mux.Handle("GET /api/products", productProxy)
	mux.Handle("POST /api/products", protectedProductProxy)
	mux.Handle("PUT /api/products/{id}", protectedProductProxy)
	mux.Handle("DELETE /api/products/{id}", protectedProductProxy)

	return nil
}
