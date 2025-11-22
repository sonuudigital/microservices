package router

import (
	"errors"
	"net/http"

	"github.com/sonuudigital/microservices/shared/logs"
)

var (
	ErrLoggerIsNil         = errors.New("logger is nil")
	ErrProductHandlerIsNil = errors.New("product handler is nil")
)

type ProductHandler interface {
	SearchProduct(w http.ResponseWriter, r *http.Request)
}

type Router struct {
	logger         logs.Logger
	productHandler ProductHandler
	mux            *http.ServeMux
}

func New(logger logs.Logger, productHandler ProductHandler) (*Router, error) {
	if logger == nil {
		return nil, ErrLoggerIsNil
	}
	if productHandler == nil {
		return nil, ErrProductHandlerIsNil
	}

	r := &Router{
		logger:         logger,
		productHandler: productHandler,
		mux:            http.NewServeMux(),
	}
	r.setupRoutes()
	return r, nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) setupRoutes() {
	r.mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("search service is healthy"))
	})
	r.mux.HandleFunc("/api/products/search", r.productHandler.SearchProduct)
}
