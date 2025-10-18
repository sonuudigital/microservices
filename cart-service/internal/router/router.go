package router

import (
	"net/http"

	"github.com/sonuudigital/microservices/cart-service/internal/db"
	"github.com/sonuudigital/microservices/cart-service/internal/handlers"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
)

func ConfigRoutes(db db.DB, logger logs.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	registerCartRoutes(mux, db, logger)

	return mux
}

func registerCartRoutes(mux *http.ServeMux, db db.DB, logger logs.Logger) {
	queries := repository.New(db)
	h := handlers.NewHandler(queries, logger)

	mux.HandleFunc("GET /api/carts/{id}", h.GetCartByUserIDHandler)
	mux.HandleFunc("POST /api/carts", h.CreateCartHandler)
}
