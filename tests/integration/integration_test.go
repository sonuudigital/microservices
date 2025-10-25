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
	apiGatewayURLKey       = "API_GATEWAY_URL"
	apiUsers               = "%s/api/users"
	apiUsersWithPath       = apiUsers + "/%s"
	apiProductsWithPath    = "%s/api/products/%s"
	apiCarts               = "%s/api/carts"
	apiCartsProducts       = "%s/api/carts/products"
	apiCartsProductsWithID = "%s/api/carts/products/%s"
	contentTypeJSON        = "application/json"
	contentTypeHeader      = "Content-Type"
	bearerWithSpace        = "Bearer "
	createProductStepMsg   = "Create Product step must run first"
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
	CategoryID    string  `json:"categoryId"`
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
			CategoryID:    "a1eebc99-9c0b-4ef8-bb6d-6bb9bd380b11",
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
		req.Header.Set(contentTypeHeader, contentTypeJSON)

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

	t.Run("Get Products By Category", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, createProductStepMsg)

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/products/category/%s", apiGatewayURL, createdProduct.CategoryID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var products []Product
		err = json.NewDecoder(resp.Body).Decode(&products)
		require.NoError(err)
		assert.GreaterOrEqual(len(products), 1)
	})

	t.Run("Update Product", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, createProductStepMsg)

		updateReq := Product{Name: "Laptop Office", Price: 1200.75, StockQuantity: 25}
		body, err := json.Marshal(updateReq)
		require.NoError(err)

		req, err := http.NewRequest("PUT", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, createdProduct.ID), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

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

type Cart struct {
	ID         string        `json:"id"`
	UserID     string        `json:"userId"`
	Products   []CartProduct `json:"products"`
	TotalPrice float64       `json:"totalPrice"`
}

type CartProduct struct {
	ProductID   string  `json:"productId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

type AddProductToCartRequest struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

type AddProductToCartResponse struct {
	ID        string  `json:"id"`
	CartID    string  `json:"cartId"`
	ProductID string  `json:"productId"`
	Quantity  int32   `json:"quantity"`
	Price     float64 `json:"price"`
	AddedAt   string  `json:"addedAt"`
}

func createProduct(require *require.Assertions, apiGatewayURL, authToken, name, code string, price float64, stockQuantity int32) Product {
	productReq := Product{
		Name:          name,
		Description:   fmt.Sprintf("Description for %s", name),
		Price:         price,
		Code:          code,
		StockQuantity: stockQuantity,
	}

	body, err := json.Marshal(productReq)
	require.NoError(err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/products", apiGatewayURL), bytes.NewBuffer(body))
	require.NoError(err)
	req.Header.Set("Authorization", bearerWithSpace+authToken)
	req.Header.Set(contentTypeHeader, contentTypeJSON)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(err)
	defer resp.Body.Close()

	require.Equal(http.StatusCreated, resp.StatusCode)

	var createdProduct Product
	err = json.NewDecoder(resp.Body).Decode(&createdProduct)
	require.NoError(err)
	require.NotEmpty(createdProduct.ID)

	return createdProduct
}

func TestCartOperations(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(apiGatewayURLKey)

	_, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)

	product1 := createProduct(require, apiGatewayURL, authToken, "Test Product 1", fmt.Sprintf("TP1-%d", time.Now().UnixNano()), 100.50, 20)
	product2 := createProduct(require, apiGatewayURL, authToken, "Test Product 2", fmt.Sprintf("TP2-%d", time.Now().UnixNano()), 250.75, 15)
	product3 := createProduct(require, apiGatewayURL, authToken, "Test Product 3", fmt.Sprintf("TP3-%d", time.Now().UnixNano()), 50.00, 30)

	t.Run("Get Empty Cart - Should Return 404", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Add First Product to Cart", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product1.ID,
			Quantity:  2,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var addProductResp AddProductToCartResponse
		err = json.NewDecoder(resp.Body).Decode(&addProductResp)
		require.NoError(err)
		assert.NotEmpty(addProductResp.ID)
		assert.NotEmpty(addProductResp.CartID)
		assert.Equal(product1.ID, addProductResp.ProductID)
		assert.Equal(int32(2), addProductResp.Quantity)
	})

	t.Run("View Cart Contents After Adding First Product", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)
		assert.NotEmpty(cart.ID)
		assert.NotEmpty(cart.UserID)
		assert.Len(cart.Products, 1)
		assert.Equal(product1.ID, cart.Products[0].ProductID)
		assert.Equal(2, cart.Products[0].Quantity)
		assert.Equal(product1.Price*2, cart.TotalPrice)
	})

	t.Run("Add Second Product to Cart", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product2.ID,
			Quantity:  1,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var addProductResp AddProductToCartResponse
		err = json.NewDecoder(resp.Body).Decode(&addProductResp)
		require.NoError(err)
		assert.Equal(product2.ID, addProductResp.ProductID)
		assert.Equal(int32(1), addProductResp.Quantity)
	})

	t.Run("Add Third Product to Cart", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product3.ID,
			Quantity:  3,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var addProductResp AddProductToCartResponse
		err = json.NewDecoder(resp.Body).Decode(&addProductResp)
		require.NoError(err)
		assert.Equal(product3.ID, addProductResp.ProductID)
		assert.Equal(int32(3), addProductResp.Quantity)
	})

	t.Run("View Cart with Multiple Products", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)
		assert.NotEmpty(cart.ID)
		assert.Len(cart.Products, 3)

		expectedTotal := (product1.Price * 2) + (product2.Price * 1) + (product3.Price * 3)
		assert.Equal(expectedTotal, cart.TotalPrice)
	})

	t.Run("Update Product Quantity in Cart", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product1.ID,
			Quantity:  5,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var addProductResp AddProductToCartResponse
		err = json.NewDecoder(resp.Body).Decode(&addProductResp)
		require.NoError(err)
		assert.Equal(product1.ID, addProductResp.ProductID)
		assert.Equal(int32(5), addProductResp.Quantity)
	})

	t.Run("Verify Updated Quantity in Cart", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)

		var foundProduct *CartProduct
		for _, p := range cart.Products {
			if p.ProductID == product1.ID {
				foundProduct = &p
				break
			}
		}

		require.NotNil(foundProduct, "Product 1 should be in the cart")
		assert.Equal(5, foundProduct.Quantity)

		expectedTotal := (product1.Price * 5) + (product2.Price * 1) + (product3.Price * 3)
		assert.Equal(expectedTotal, cart.TotalPrice)
	})

	t.Run("Remove One Product from Cart", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", fmt.Sprintf(apiCartsProductsWithID, apiGatewayURL, product2.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify Product Removed from Cart", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)
		assert.Len(cart.Products, 2, "Cart should have 2 products after removal")

		for _, p := range cart.Products {
			assert.NotEqual(product2.ID, p.ProductID, "Product 2 should not be in cart")
		}

		expectedTotal := (product1.Price * 5) + (product3.Price * 3)
		assert.Equal(expectedTotal, cart.TotalPrice)
	})

	t.Run("Clear All Products from Cart", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", fmt.Sprintf(apiCartsProducts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify Cart is Empty After Clear", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)
		assert.Len(cart.Products, 0, "Cart should be empty after clearing products")
		assert.Equal(0.0, cart.TotalPrice, "Total price should be 0 for empty cart")
	})

	t.Run("Add Product Again After Clear", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product1.ID,
			Quantity:  1,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var addProductResp AddProductToCartResponse
		err = json.NewDecoder(resp.Body).Decode(&addProductResp)
		require.NoError(err)
		assert.Equal(product1.ID, addProductResp.ProductID)
	})

	t.Run("Delete Entire Cart", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify Cart Does Not Exist After Deletion", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Try to Add Product with Invalid Product ID", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: "invalid-product-id",
			Quantity:  1,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Try to Remove Non-Existent Product from Cart", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product1.ID,
			Quantity:  1,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		resp.Body.Close()

		req, err = http.NewRequest("DELETE", fmt.Sprintf(apiCartsProductsWithID, apiGatewayURL, product2.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})
}
