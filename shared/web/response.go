package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sonuudigital/microservices/shared/logs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	contentType             = "Content-Type"
	failedToEncodeMsg       = "failed to encode response"
	failedToEncodeErrRspMsg = "failed to encode error response"

	httpStatusClientClosedRequest = 499
	ReqCancelledMsg               = "request cancelled"
)

type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

type GRPCProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Details  []any  `json:"details,omitempty"`
}

func RespondWithJSON(w http.ResponseWriter, logger logs.Logger, status int, payload any) {
	w.Header().Set(contentType, "application/json")
	w.WriteHeader(status)

	if payload != nil {
		var err error
		var marshalledPayload []byte

		if p, ok := payload.(proto.Message); ok {
			marshalledPayload, err = protojson.Marshal(p)
		} else {
			marshalledPayload, err = json.Marshal(payload)
		}

		if err != nil {
			if logger != nil {
				logger.Error(failedToEncodeMsg, "error", err)
			}
			http.Error(w, failedToEncodeMsg, http.StatusInternalServerError)
			return
		}

		_, err = w.Write(marshalledPayload)
		if err != nil {
			if logger != nil {
				logger.Error("failed to write response", "error", err)
			}
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

	w.Header().Set(contentType, "application/problem+json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(problem); err != nil {
		if logger != nil {
			logger.Error(failedToEncodeErrRspMsg, "error", err)
		}
		http.Error(w, failedToEncodeErrRspMsg, http.StatusInternalServerError)
	}
}

func RespondWithGRPCError(w http.ResponseWriter, r *http.Request, grpcStatus *status.Status, logger logs.Logger) {
	httpStatus := HTTPStatusFromGRPC(grpcStatus.Code())

	var problem = GRPCProblemDetail{}
	if httpStatus != http.StatusInternalServerError {
		problem = GRPCProblemDetail{
			Type:     getErrorDocumentationLink(httpStatus),
			Title:    grpcStatus.Message(),
			Status:   httpStatus,
			Detail:   grpcStatus.Message(),
			Instance: r.URL.Path,
			Details:  grpcStatus.Details(),
		}
	} else {
		problem = GRPCProblemDetail{
			Type:     getErrorDocumentationLink(httpStatus),
			Title:    "Internal Server Error",
			Status:   httpStatus,
			Detail:   "an internal server error occurred",
			Instance: r.URL.Path,
			Details:  grpcStatus.Details(),
		}
	}

	w.Header().Set(contentType, "application/problem+json")
	w.WriteHeader(httpStatus)

	if err := json.NewEncoder(w).Encode(problem); err != nil {
		if logger != nil {
			logger.Error(failedToEncodeErrRspMsg, "error", err)
		}
		http.Error(w, failedToEncodeErrRspMsg, http.StatusInternalServerError)
	}
}

func CheckContext(ctx context.Context, w http.ResponseWriter, r *http.Request, logger logs.Logger) bool {
	if ctx.Err() != nil {
		ctxErr := ctx.Err()
		switch ctxErr {
		case context.Canceled:
			logger.Warn("request canceled by the client", "error", ctxErr)
			RespondWithError(w, logger, r, httpStatusClientClosedRequest, "Request Canceled", "the request was canceled by the client")
		case context.DeadlineExceeded:
			logger.Warn("request deadline exceeded", "error", ctxErr)
			RespondWithError(w, logger, r, http.StatusGatewayTimeout, "Deadline Exceeded", "the request deadline was exceeded")
		default:
			logger.Error("context error", "error", ctxErr)
			RespondWithError(w, logger, r, http.StatusInternalServerError, "Internal Server Error", "an internal server error occurred")
		}
		return false
	}
	return true
}

func HTTPStatusFromGRPC(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return httpStatusClientClosedRequest
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
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
	case http.StatusServiceUnavailable:
		return "https://tools.ietf.org/html/rfc7231#section-6.6.4"
	case http.StatusRequestTimeout:
		return "https://tools.ietf.org/html/rfc7231#section-6.5.7"
	case http.StatusConflict:
		return "https://tools.ietf.org/html/rfc7231#section-6.5.8"
	case http.StatusTooManyRequests:
		return "https://tools.ietf.org/html/rfc6585#section-4"
	case http.StatusNotImplemented:
		return "https://tools.ietf.org/html/rfc7231#section-6.6.2"
	case http.StatusGatewayTimeout:
		return "https://tools.ietf.org/html/rfc7231#section-6.6.3"
	case http.StatusBadGateway:
		return "https://tools.ietf.org/html/rfc7231#section-6.6.3"
	case 499:
		return "https://httpstatuses.com/499"
	default:
		return "about:blank"
	}
}
