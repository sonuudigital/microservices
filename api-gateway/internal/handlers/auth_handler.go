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
	Token string `json:"token"`
}

const (
	internalServerErrorMsg = "internal server error"
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		h.logger.Error("user service url not found in environment variables")
		http.Error(w, internalServerErrorMsg, http.StatusInternalServerError)
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		h.logger.Error("failed to marshal login request", "error", err)
		http.Error(w, internalServerErrorMsg, http.StatusInternalServerError)
		return
	}

	downstreamReq, err := http.NewRequest("POST", userServiceURL+"/api/auth/login", bytes.NewBuffer(payload))
	if err != nil {
		h.logger.Error("failed to create request to user-service", "error", err)
		http.Error(w, internalServerErrorMsg, http.StatusInternalServerError)
		return
	}
	downstreamReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(downstreamReq)
	if err != nil {
		h.logger.Error("failed to call user-service", "error", err)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.logger.Error("failed to read error response from user-service", "error", err)
			http.Error(w, internalServerErrorMsg, http.StatusInternalServerError)
			return
		}

		http.Error(w, string(body), resp.StatusCode)
		return
	}

	var user UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		h.logger.Error("failed to decode user response", "error", err)
		http.Error(w, "failed to decode user response", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtManager.GenerateToken(user.Email)
	if err != nil {
		h.logger.Error("failed to generate token", "error", err)
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, LoginResponse{Token: token})
}
