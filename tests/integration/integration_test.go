package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	apiGatewayURLKey     = "API_GATEWAY_URL"
	apiUsers             = "%s/api/users"
	apiUsersWithPath     = apiUsers + "/%s"
	apiProductsWithPath  = "%s/api/products/%s"
	contentTypeJSON      = "application/json"
	bearerWithSpace      = "Bearer "
	createProductStepMsg = "Create Product step must run first"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type Product struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	Code          string  `json:"code"`
	StockQuantity int32   `json:"stock_quantity"`
}

func TestMain(m *testing.M) {
	if os.Getenv(apiGatewayURLKey) == "" {
		panic("API_GATEWAY_URL environment variable not set. Make sure docker-compose is running.")
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func registerAndLogin(require *require.Assertions) (User, string) {
	apiGatewayURL := os.Getenv(apiGatewayURLKey)

	email := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	password := "password123"

	createUserReqBody, err := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	require.NoError(err)

	registerURL := fmt.Sprintf(apiUsers, apiGatewayURL)
	resp, err := http.Post(registerURL, contentTypeJSON, bytes.NewBuffer(createUserReqBody))
	require.NoError(err)
	defer resp.Body.Close()
	require.Equal(http.StatusCreated, resp.StatusCode)

	loginReqBody, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(err)

	loginURL := fmt.Sprintf("%s/api/auth/login", apiGatewayURL)
	resp, err = http.Post(loginURL, contentTypeJSON, bytes.NewBuffer(loginReqBody))
	require.NoError(err)
	defer resp.Body.Close()
	require.Equal(http.StatusOK, resp.StatusCode)

	var loginResp LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(err)
	require.NotEmpty(loginResp.Token)
	require.NotEmpty(loginResp.User.ID)
	require.Equal(email, loginResp.User.Email)

	return loginResp.User, loginResp.Token
}

func TestUserRegistrationAndLogin(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	user, token := registerAndLogin(require)

	assert.NotEmpty(token)
	assert.NotEmpty(user.ID)
}

func TestAccessProtectedRoutes(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(apiGatewayURLKey)

	user, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)
	require.NotEmpty(user.ID)

	t.Run("Successful Access", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiUsersWithPath, apiGatewayURL, user.ID), nil)
		require.NoError(err)

		req.Header.Set("Authorization", bearerWithSpace+authToken)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var fetchedUser User
		err = json.NewDecoder(resp.Body).Decode(&fetchedUser)
		require.NoError(err)

		assert.Equal(user.ID, fetchedUser.ID)
		assert.Equal(user.Email, fetchedUser.Email)
	})

	t.Run("Access without Token", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiUsersWithPath, apiGatewayURL, user.ID), nil)
		require.NoError(err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access with Invalid Token", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiUsersWithPath, apiGatewayURL, user.ID), nil)
		require.NoError(err)

		req.Header.Set("Authorization", "Bearer invalidtoken")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestProductCRUD(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")

	_, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)

	var createdProduct Product

	t.Run("Create Product", func(t *testing.T) {
		productReq := Product{
			Name:          "Laptop Gamer",
			Description:   "The best laptop for gaming",
			Price:         2500.50,
			Code:          fmt.Sprintf("LP-%d", time.Now().UnixNano()),
			StockQuantity: 10,
		}

		body, err := json.Marshal(productReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/products", apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set("Content-Type", contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdProduct)
		require.NoError(err)
		assert.NotEmpty(createdProduct.ID)
		assert.Equal(productReq.Name, createdProduct.Name)
	})

	t.Run("Read Product", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, createProductStepMsg)

		req, err := http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, createdProduct.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var fetchedProduct Product
		err = json.NewDecoder(resp.Body).Decode(&fetchedProduct)
		require.NoError(err)
		assert.Equal(createdProduct.ID, fetchedProduct.ID)
	})

	t.Run("Update Product", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, createProductStepMsg)

		updateReq := Product{Name: "Laptop Office", Price: 1200.75, StockQuantity: 25}
		body, err := json.Marshal(updateReq)
		require.NoError(err)

		req, err := http.NewRequest("PUT", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, createdProduct.ID), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set("Content-Type", contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var updatedProduct Product
		err = json.NewDecoder(resp.Body).Decode(&updatedProduct)
		require.NoError(err)
		assert.Equal(updateReq.Name, updatedProduct.Name)
		assert.NotEqual(createdProduct.Name, updatedProduct.Name)
	})

	t.Run("Delete Product", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, createProductStepMsg)

		req, err := http.NewRequest("DELETE", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, createdProduct.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify Delete", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, createProductStepMsg)

		req, err := http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, createdProduct.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNotFound, resp.StatusCode)
	})
}
