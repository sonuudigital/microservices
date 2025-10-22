package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/sonuudigital/microservices/user-service/internal/handlers"
	"github.com/sonuudigital/microservices/user-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
