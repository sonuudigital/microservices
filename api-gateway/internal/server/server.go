package server

import (
	"errors"
	"net/http"
	"shared/logs"
	"time"
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

	logger.Info("server initialized", "port", port)

	return srv, nil
}
