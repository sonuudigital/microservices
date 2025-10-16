package server

import (
	"context"
	"net/http"
	"os"
	"shared/logs"
	"time"
	"user-service/internal/db"
	"user-service/internal/handlers"
)

type Server struct {
	*http.Server
}

const (
	serverReadHeaderTimeout time.Duration = 20 * time.Second
	serverWriteTimeout      time.Duration = 1 * time.Minute
	serverIdleTimeout       time.Duration = 3 * time.Minute
)

func newServer(srv *http.Server) *Server {
	return &Server{
		srv,
	}
}

func InitializeServer(db db.DB, logger logs.Logger) *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           configRoutes(db, logger),
		ReadHeaderTimeout: serverReadHeaderTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	return newServer(srv)
}

func configRoutes(db db.DB, logger logs.Logger) *http.ServeMux {
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

	registerUserRoutes(mux, db, logger)

	return mux
}

func registerUserRoutes(mux *http.ServeMux, db db.DB, logger logs.Logger) {
	handler := handlers.NewHandler(db, logger)

	mux.HandleFunc("POST /api/users", handler.CreateUserHandler)
	mux.HandleFunc("POST /api/auth/login", handler.AuthorizeUserHandler)
	mux.HandleFunc("GET /api/users/{id}", handler.GetUserByIDHandler)
}
