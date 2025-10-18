package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
)

const (
	contentType = "Content-Type"
)

type AuthHandler struct {
	logger     logs.Logger
	jwtManager *auth.JWTManager
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type LoginResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

const (
	internalServerErrorMsg = "Internal Server Error"
)

func NewAuthHandler(logger logs.Logger, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		logger:     logger,
		jwtManager: jwtManager,
	}
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		h.logger.Error("user service url not found in environment variables")
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "Service configuration error.")
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		h.logger.Error("failed to marshal login request", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "Could not process login request.")
		return
	}

	downstreamReq, err := http.NewRequest("POST", userServiceURL+"/api/auth/login", bytes.NewBuffer(payload))
	if err != nil {
		h.logger.Error("failed to create request to user-service", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "Could not create request to downstream service.")
		return
	}
	downstreamReq.Header.Set(contentType, "application/json")

	client := &http.Client{}
	resp, err := client.Do(downstreamReq)
	if err != nil {
		h.logger.Error("failed to call user-service", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusServiceUnavailable, "Service Unavailable", "The user service is currently unavailable.")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("failed to read response from user-service", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "Failed to read response from downstream service.")
		return
	}

	if resp.StatusCode != http.StatusOK {
		w.Header().Set(contentType, resp.Header.Get(contentType))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	var user UserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		h.logger.Error("failed to decode user response", "error", err, "body", string(body))
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "Failed to decode user response from downstream service.")
		return
	}

	token, err := h.jwtManager.GenerateToken(user.Email)
	if err != nil {
		h.logger.Error("failed to generate token", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, internalServerErrorMsg, "Failed to generate authentication token.")
		return
	}

	response := LoginResponse{
		User:  user,
		Token: token,
	}
	web.RespondWithJSON(w, h.logger, http.StatusOK, response)
}
