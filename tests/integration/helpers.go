package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	ApiGatewayURLKey          = "API_GATEWAY_URL"
	ApiUsers                  = "%s/api/users"
	ApiUsersWithPath          = ApiUsers + "/%s"
	ApiProductsWithPath       = "%s/api/products/%s"
	ApiCarts                  = "%s/api/carts"
	ApiCartsProducts          = "%s/api/carts/products"
	ApiCartsProductsWithID    = "%s/api/carts/products/%s"
	ApiOrders                 = "%s/api/orders"
	ProductCategoriesEndpoint = "api/products/categories"
	ContentTypeJSON           = "application/json"
	ContentTypeHeader         = "Content-Type"
	BearerWithSpace           = "Bearer "
	CreateProductStepMsg      = "Create Product step must run first"
	SleepDuration             = 6 * time.Second
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
	StockQuantity int32   `json:"stockQuantity"`
}

type ProductCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdateCategoryRequest struct {
	ID string `json:"id"`
	ProductCategoryRequest
}

type ProductCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
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

type Order struct {
	ID          string  `json:"id"`
	UserID      string  `json:"userId"`
	TotalAmount float64 `json:"totalAmount"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"createdAt"`
}

func RegisterAndLogin(require *require.Assertions) (User, string) {
	apiGatewayURL := os.Getenv(ApiGatewayURLKey)

	email := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	password := "password123"

	createUserReqBody, err := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	require.NoError(err)

	registerURL := fmt.Sprintf(ApiUsers, apiGatewayURL)
	resp, err := http.Post(registerURL, ContentTypeJSON, bytes.NewBuffer(createUserReqBody))
	require.NoError(err)
	defer resp.Body.Close()
	require.Equal(http.StatusCreated, resp.StatusCode)

	loginReqBody, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(err)

	loginURL := fmt.Sprintf("%s/api/auth/login", apiGatewayURL)
	resp, err = http.Post(loginURL, ContentTypeJSON, bytes.NewBuffer(loginReqBody))
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

func CreateProduct(require *require.Assertions, apiGatewayURL, authToken, name string, price float64, stockQuantity int32, description ...string) Product {
	categoryReq := ProductCategoryRequest{
		Name:        fmt.Sprintf("Category for %s", name),
		Description: fmt.Sprintf("Auto-generated category for %s", name),
	}

	body, err := json.Marshal(categoryReq)
	require.NoError(err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", apiGatewayURL, ProductCategoriesEndpoint), bytes.NewBuffer(body))
	require.NoError(err)
	req.Header.Set("Authorization", BearerWithSpace+authToken)
	req.Header.Set(ContentTypeHeader, ContentTypeJSON)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(err)
	defer resp.Body.Close()

	require.Equal(http.StatusCreated, resp.StatusCode)

	var category ProductCategory
	err = json.NewDecoder(resp.Body).Decode(&category)
	require.NoError(err)

	desc := fmt.Sprintf("Description for %s", name)
	if len(description) > 0 {
		desc = description[0]
	}

	productReq := Product{
		CategoryID:    category.ID,
		Name:          name,
		Description:   desc,
		Price:         price,
		StockQuantity: stockQuantity,
	}

	body, err = json.Marshal(productReq)
	require.NoError(err)

	req, err = http.NewRequest("POST", fmt.Sprintf("%s/api/products", apiGatewayURL), bytes.NewBuffer(body))
	require.NoError(err)
	req.Header.Set("Authorization", BearerWithSpace+authToken)
	req.Header.Set(ContentTypeHeader, ContentTypeJSON)

	resp, err = client.Do(req)
	require.NoError(err)
	defer resp.Body.Close()

	require.Equal(http.StatusCreated, resp.StatusCode)

	var createdProduct Product
	err = json.NewDecoder(resp.Body).Decode(&createdProduct)
	require.NoError(err)
	require.NotEmpty(createdProduct.ID)

	return createdProduct
}
