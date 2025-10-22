package handlers

import (
	"encoding/json"
	"net/http"

	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/shared/web"
	"google.golang.org/grpc/status"
)

type UserHandler struct {
	logger     logs.Logger
	userClient userv1.UserServiceClient
}

func NewUserHandler(logger logs.Logger, userClient userv1.UserServiceClient) *UserHandler {
	return &UserHandler{
		logger:     logger,
		userClient: userClient,
	}
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Invalid Request Body", err.Error())
		return
	}

	grpcReq := &userv1.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	_, err := h.userClient.CreateUser(r.Context(), grpcReq)
	if err != nil {
		st, _ := status.FromError(err)
		h.logger.Error("failed to create user via grpc", "error", st.Message())
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Failed to create user", st.Message())
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusCreated, nil)
}

func (h *UserHandler) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	grpcReq := &userv1.GetUserByIDRequest{
		Id: id,
	}

	res, err := h.userClient.GetUserByID(r.Context(), grpcReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			web.RespondWithGRPCError(w, r, st, h.logger)
			return
		}
		h.logger.Error("failed to get user by id via grpc", "error", err)
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Failed to get user", err.Error())
		return
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, res)
}
