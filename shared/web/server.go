package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	serverReadHeaderTimeout time.Duration = 20 * time.Second
	serverWriteTimeout      time.Duration = 1 * time.Minute
	serverIdleTimeout       time.Duration = 3 * time.Minute
)

func InitializeServer(port string, handler http.Handler, logger logs.Logger) (*http.Server, error) {
	if port == "" {
		return nil, errors.New("port not found in environment variables")
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	return srv, nil
}
