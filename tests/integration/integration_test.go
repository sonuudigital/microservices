package integration

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

func TestMain(m *testing.M) {
	if os.Getenv(ApiGatewayURLKey) == "" {
		panic("API_GATEWAY_URL environment variable not set. Make sure docker-compose is running.")
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestUserRegistrationAndLogin(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	user, token := RegisterAndLogin(require)

	assert.NotEmpty(token)
	assert.NotEmpty(user.ID)
}

func TestAccessProtectedRoutes(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(ApiGatewayURLKey)

	user, authToken := RegisterAndLogin(require)
	require.NotEmpty(authToken)
	require.NotEmpty(user.ID)

	t.Run("Successful Access", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiUsersWithPath, apiGatewayURL, user.ID), nil)
		require.NoError(err)

		req.Header.Set("Authorization", BearerWithSpace+authToken)
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
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiUsersWithPath, apiGatewayURL, user.ID), nil)
		require.NoError(err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access with Invalid Token", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiUsersWithPath, apiGatewayURL, user.ID), nil)
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
	apiGatewayURL := os.Getenv(ApiGatewayURLKey)

	_, authToken := RegisterAndLogin(require)
	require.NotEmpty(authToken)

	var createdProductCategories ProductCategory
	var createdProduct Product

	t.Run("Create Product Category", func(t *testing.T) {
		productCategoryReq := ProductCategoryRequest{
			Name:        fmt.Sprintf("TEST CATEGORY%d", time.Now().Unix()),
			Description: fmt.Sprintf("TEST DESCRIPTION%d", time.Now().Unix()),
		}

		body, err := json.Marshal(productCategoryReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", apiGatewayURL, ProductCategoriesEndpoint), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Content-Type", ContentTypeJSON)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdProductCategories)
		require.NoError(err)
		assert.Equal(productCategoryReq.Name, createdProductCategories.Name)
		assert.Equal(productCategoryReq.Description, createdProductCategories.Description)
		assert.NotEmpty(createdProductCategories.ID, "Product Category ID should not be empty")
		assert.NotEmpty(createdProductCategories.CreatedAt, "Product Category CreatedAt should not be empty")
		fmt.Println("Created Product Category ID:", createdProductCategories.ID)
	})

	t.Run("Create Product", func(t *testing.T) {
		require.NotEmpty(createdProductCategories.ID, "Product Category must be created first")
		productReq := Product{
			CategoryID:    createdProductCategories.ID,
			Name:          "Laptop Gamer",
			Description:   "The best laptop for gaming",
			Price:         2500.50,
			StockQuantity: 10,
		}

		body, err := json.Marshal(productReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/products", apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdProduct)
		fmt.Println("Created Product", createdProduct)
		require.NoError(err)
		assert.NotEmpty(createdProduct.ID)
		assert.Equal(productReq.Name, createdProduct.Name)
	})

	t.Run("Read Product", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, CreateProductStepMsg)

		req, err := http.NewRequest("GET", fmt.Sprintf(ApiProductsWithPath, apiGatewayURL, createdProduct.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

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
		require.NotEmpty(createdProduct.CategoryID, CreateProductStepMsg)

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/products/categories/%s", apiGatewayURL, createdProduct.CategoryID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

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
		require.NotEmpty(createdProduct.ID, CreateProductStepMsg)

		updateReq := Product{CategoryID: createdProduct.CategoryID, Name: "Laptop Office", Price: 1200.75, StockQuantity: 25}
		body, err := json.Marshal(updateReq)
		require.NoError(err)

		req, err := http.NewRequest("PUT", fmt.Sprintf(ApiProductsWithPath, apiGatewayURL, createdProduct.ID), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

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
		require.NotEmpty(createdProduct.ID, CreateProductStepMsg)

		req, err := http.NewRequest("DELETE", fmt.Sprintf(ApiProductsWithPath, apiGatewayURL, createdProduct.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify Delete", func(t *testing.T) {
		require.NotEmpty(createdProduct.ID, CreateProductStepMsg)

		req, err := http.NewRequest("GET", fmt.Sprintf(ApiProductsWithPath, apiGatewayURL, createdProduct.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNotFound, resp.StatusCode)
	})
}

func TestCartOperations(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(ApiGatewayURLKey)

	_, authToken := RegisterAndLogin(require)
	require.NotEmpty(authToken)

	product1 := CreateProduct(require, apiGatewayURL, authToken, "Test Product 1", 100.50, 20)
	product2 := CreateProduct(require, apiGatewayURL, authToken, "Test Product 2", 250.75, 15)
	product3 := CreateProduct(require, apiGatewayURL, authToken, "Test Product 3", 50.00, 30)

	t.Run("Get Empty Cart - Should Return Empty Cart with 200", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)
		assert.NotEmpty(cart.ID, "Cart ID should be created")
		assert.NotEmpty(cart.UserID, "User ID should be present")
		assert.Empty(cart.Products, "Products should be empty")
		assert.Equal(0.0, cart.TotalPrice, "Total price should be 0")
	})

	t.Run("Add First Product to Cart", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product1.ID,
			Quantity:  2,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

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
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

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

		req, err := http.NewRequest("POST", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

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

		req, err := http.NewRequest("POST", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

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
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

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

		req, err := http.NewRequest("POST", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

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
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

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
		req, err := http.NewRequest("DELETE", fmt.Sprintf(ApiCartsProductsWithID, apiGatewayURL, product2.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify Product Removed from Cart", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

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
		req, err := http.NewRequest("DELETE", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify Cart is Empty After Clear", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

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

		req, err := http.NewRequest("POST", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

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
		req, err := http.NewRequest("DELETE", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify New Empty Cart is Created After Deletion", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(ApiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)
		assert.NotEmpty(cart.ID, "A new cart should be created")
		assert.NotEmpty(cart.UserID, "User ID should be present")
		assert.Empty(cart.Products, "Products should be empty in new cart")
		assert.Equal(0.0, cart.TotalPrice, "Total price should be 0 in new cart")
	})

	t.Run("Try to Add Product with Invalid Product ID", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: "invalid-product-id",
			Quantity:  1,
		}

		body, err := json.Marshal(addProductReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

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

		req, err := http.NewRequest("POST", fmt.Sprintf(ApiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		resp.Body.Close()

		req, err = http.NewRequest("DELETE", fmt.Sprintf(ApiCartsProductsWithID, apiGatewayURL, product2.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
	})
}