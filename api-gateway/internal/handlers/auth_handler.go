package handlers

import (
	"encoding/json"
	"net/http"

	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/auth"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	logger     logs.Logger
	jwtManager *auth.JWTManager
	userClient userv1.UserServiceClient
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

func NewAuthHandler(logger logs.Logger, jwtManager *auth.JWTManager, userClient userv1.UserServiceClient) *AuthHandler {
	return &AuthHandler{
		logger:     logger,
		jwtManager: jwtManager,
		userClient: userClient,
	}
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	grpcReq := &userv1.AuthorizeUserRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	res, err := h.userClient.AuthorizeUser(r.Context(), grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			web.RespondWithGRPCError(w, r, st, h.logger)
			return
		}
		h.logger.Error("failed to authorize user via grpc", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Failed to authorize user", err.Error())
		return
	}

	user := UserResponse{
		ID:       res.Id,
		Username: res.Username,
		Email:    res.Email,
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Email)
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
