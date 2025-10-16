package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sonuudigital/microservices/shared/logs"
)

const (
	ReqCancelledMsg = "request cancelled"
)

func RespondWithJSON(w http.ResponseWriter, logger logs.Logger, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			if logger != nil {
				logger.Error("failed to encode response", "error", err)
			}
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

func CheckContext(ctx context.Context, logger logs.Logger) bool {
	if ctx.Err() != nil {
		if logger != nil {
			logger.Error(ReqCancelledMsg, "error", ctx.Err())
		}
		return false
	}
	return true
}
