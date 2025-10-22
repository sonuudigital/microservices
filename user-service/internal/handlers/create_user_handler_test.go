package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/user-service/internal/handlers"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateUserHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	userParams := repository.CreateUserParams{
		Username: "testuser",
		Email:    testEmail,
		Password: "password",
	}
	body, _ := json.Marshal(userParams)

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("CreateUser", mock.Anything, mock.AnythingOfType("repository.CreateUserParams")).Return(repository.User{}, nil)

		req, err := http.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.CreateUserHandler(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("DB Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		mockQuerier.On("CreateUser", mock.Anything, mock.AnythingOfType("repository.CreateUserParams")).Return(repository.User{}, errors.New("db error"))

		req, err := http.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.CreateUserHandler(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockQuerier.AssertExpectations(t)
	})
}
