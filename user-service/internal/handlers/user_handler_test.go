package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"shared/logs"
	"testing"
	"user-service/internal/handlers"
	"user-service/internal/repository"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testEmail = "test@example.com"
	authURL   = "/api/auth/login"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.User, error) {
	args := m.Called(ctx, arg)
	if u, ok := args.Get(0).(repository.User); ok {
		return u, args.Error(1)
	}
	return repository.User{}, args.Error(1)
}

func (m *MockQuerier) GetUserByEmail(ctx context.Context, email string) (repository.User, error) {
	args := m.Called(ctx, email)
	if u, ok := args.Get(0).(repository.User); ok {
		return u, args.Error(1)
	}
	return repository.User{}, args.Error(1)
}

func (m *MockQuerier) GetUserByID(ctx context.Context, id pgtype.UUID) (repository.User, error) {
	args := m.Called(ctx, id)
	if u, ok := args.Get(0).(repository.User); ok {
		return u, args.Error(1)
	}
	return repository.User{}, args.Error(1)
}

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

func TestGetUserByIDHandler(t *testing.T) {
	logger := logs.NewSlogLogger()

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
		var pgUUID pgtype.UUID
		err := pgUUID.Scan(uuidStr)
		assert.NoError(t, err)

		user := repository.User{ID: pgUUID, Username: "test", Email: "test@test.com"}
		mockQuerier.On("GetUserByID", mock.Anything, pgUUID).Return(user, nil).Once()

		req, err := http.NewRequest("GET", "/api/users/"+uuidStr, nil)
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.GetUserByIDHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var respUser repository.User
		err = json.NewDecoder(rr.Body).Decode(&respUser)
		assert.NoError(t, err)
		assert.Equal(t, user.Username, respUser.Username)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		uuidStr := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
		var pgUUID pgtype.UUID
		err := pgUUID.Scan(uuidStr)
		assert.NoError(t, err)

		mockQuerier.On("GetUserByID", mock.Anything, pgUUID).Return(repository.User{}, pgx.ErrNoRows).Once()

		req, err := http.NewRequest("GET", "/api/users/"+uuidStr, nil)
		assert.NoError(t, err)
		req.SetPathValue("id", uuidStr)

		rr := httptest.NewRecorder()
		handler.GetUserByIDHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		handler := handlers.NewHandler(mockQuerier, logger)

		req, err := http.NewRequest("GET", "/api/users/invalid-id", nil)
		assert.NoError(t, err)
		req.SetPathValue("id", "invalid-id")

		rr := httptest.NewRecorder()
		handler.GetUserByIDHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

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
