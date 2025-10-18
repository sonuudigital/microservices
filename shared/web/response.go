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

type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

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

func RespondWithError(w http.ResponseWriter, logger logs.Logger, r *http.Request, status int, title string, detail string) {
	problem := ProblemDetail{
		Type:     getErrorDocumentationLink(status),
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(problem); err != nil {
		if logger != nil {
			logger.Error("failed to encode error response", "error", err)
		}
		http.Error(w, "failed to encode error response", http.StatusInternalServerError)
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

func getErrorDocumentationLink(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "https://tools.ietf.org/html/rfc7231#section-6.5.1"
	case http.StatusUnauthorized:
		return "https://tools.ietf.org/html/rfc7235#section-3.1"
	case http.StatusForbidden:
		return "https://tools.ietf.org/html/rfc7231#section-6.5.3"
	case http.StatusNotFound:
		return "https://tools.ietf.org/html/rfc7231#section-6.5.4"
	case http.StatusInternalServerError:
		return "https://tools.ietf.org/html/rfc7231#section-6.6.1"
	default:
		return "about:blank"
	}
}
