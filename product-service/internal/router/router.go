package router

import (
	"context"
	"net/http"
	"time"

	"github.com/sonuudigital/microservices/product-service/internal/db"
	"github.com/sonuudigital/microservices/product-service/internal/handlers"
	"github.com/sonuudigital/microservices/product-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
)

func ConfigRoutes(db db.DB, logger logs.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, ccancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer ccancel()
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	registerProductRoutes(mux, db, logger)

	return mux
}

func registerProductRoutes(mux *http.ServeMux, db db.DB, logger logs.Logger) {
	queries := repository.New(db)
	h := handlers.NewHandler(queries, logger)

	mux.HandleFunc("POST /api/products", h.CreateProductHandler)
	mux.HandleFunc("GET /api/products/ids", h.GetProductsByIDsHandler)
	mux.HandleFunc("GET /api/products/{id}", h.GetProductHandler)
	mux.HandleFunc("GET /api/products", h.ListProductsHandler)
	mux.HandleFunc("PUT /api/products/{id}", h.UpdateProductHandler)
	mux.HandleFunc("DELETE /api/products/{id}", h.DeleteProductHandler)
}
