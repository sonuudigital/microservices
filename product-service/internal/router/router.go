package router

import (
	"context"
	"net/http"
	"product-service/internal/db"
	"product-service/internal/handlers"
	"shared/logs"
	"time"
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
	h := handlers.NewHandler(db, logger)

	mux.HandleFunc("POST /api/products", h.CreateProductHandler)
	mux.HandleFunc("GET /api/products/{id}", h.GetProductHandler)
	mux.HandleFunc("GET /api/products", h.ListProductsHandler)
	mux.HandleFunc("PUT /api/products/{id}", h.UpdateProductHandler)
	mux.HandleFunc("DELETE /api/products/{id}", h.DeleteProductHandler)
}
