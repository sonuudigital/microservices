package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/user-service/internal/handlers"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthorizeUserHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		email := testEmail
		password := "password"
		hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
		assert.NoError(t, err)

		user := repository.User{Email: email, Password: hashedPassword}
		mockQuerier.On("GetUserByEmail", mock.Anything, email).Return(user, nil).Once()

		authReq := handlers.AuthRequest{Email: email, Password: password}
		body, err := json.Marshal(authReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.AuthorizeUserHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		email := testEmail
		password := "password"

		mockQuerier.On("GetUserByEmail", mock.Anything, email).Return(repository.User{}, pgx.ErrNoRows).Once()

		authReq := handlers.AuthRequest{Email: email, Password: password}
		body, err := json.Marshal(authReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.AuthorizeUserHandler(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Password mismatch", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		email := testEmail
		password := "password"
		wrongPassword := "wrongpassword"
		hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
		assert.NoError(t, err)

		user := repository.User{Email: email, Password: hashedPassword}
		mockQuerier.On("GetUserByEmail", mock.Anything, email).Return(user, nil).Once()

		authReq := handlers.AuthRequest{Email: email, Password: wrongPassword}
		body, err := json.Marshal(authReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(body))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.AuthorizeUserHandler(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		req, err := http.NewRequest("POST", authURL, bytes.NewBufferString("invalid json"))
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.AuthorizeUserHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
