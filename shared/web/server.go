package web

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func StartServerAndWaitForShutdown(srv *http.Server, logger *logs.SlogLogger) {
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", "error", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	if err := srv.Shutdown(shCtx); err != nil {
		logger.Error("failed to shutdown server", "error", err)
	} else {
		logger.Info("shutdown complete")
	}
}
