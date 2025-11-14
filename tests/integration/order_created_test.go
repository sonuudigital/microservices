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
	apiOrders = "%s/api/orders"
)

type Order struct {
	ID          string  `json:"id"`
	UserID      string  `json:"userId"`
	TotalAmount float64 `json:"totalAmount"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"createdAt"`
}

func TestCreateOrderSuccess(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(apiGatewayURLKey)

	_, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)

	product1 := createProduct(require, apiGatewayURL, authToken, "Test Product Order 1", 100.50, 50)
	product2 := createProduct(require, apiGatewayURL, authToken, "Test Product Order 2", 250.75, 30)

	t.Run("Verify Products Exist", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product1.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode, "Product 1 should exist before adding to cart")

		var p1 Product
		err = json.NewDecoder(resp.Body).Decode(&p1)
		require.NoError(err)
		t.Logf("Product 1: ID=%s, Name=%s, Stock=%d", p1.ID, p1.Name, p1.StockQuantity)

		req, err = http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product2.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode, "Product 2 should exist before adding to cart")

		var p2 Product
		err = json.NewDecoder(resp.Body).Decode(&p2)
		require.NoError(err)
		t.Logf("Product 2: ID=%s, Name=%s, Stock=%d", p2.ID, p2.Name, p2.StockQuantity)
	})

	t.Run("Add Products to Cart", func(t *testing.T) {
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

		addProductReq = AddProductToCartRequest{
			ProductID: product2.ID,
			Quantity:  1,
		}

		body, err = json.Marshal(addProductReq)
		require.NoError(err)

		req, err = http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)
		req.Header.Set(contentTypeHeader, contentTypeJSON)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)
	})

	var createdOrder Order
	t.Run("Create Order", func(t *testing.T) {
		req, err := http.NewRequest("POST", fmt.Sprintf(apiOrders, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdOrder)
		require.NoError(err)
		assert.NotEmpty(createdOrder.ID)
		assert.Equal("CREATED", createdOrder.Status)

		expectedTotal := (product1.Price * 2) + (product2.Price * 1)
		assert.Equal(expectedTotal, createdOrder.TotalAmount)
	})

	t.Run("Verify Cart is Cleared", func(t *testing.T) {
		time.Sleep(3 * time.Second)

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
		assert.Empty(cart.Products)
		assert.Equal(0.0, cart.TotalPrice)
	})

	t.Run("Verify Stock is Updated", func(t *testing.T) {
		time.Sleep(2 * time.Second)

		req, err := http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product1.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var product Product
		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(err)
		assert.Equal(int32(48), product.StockQuantity)

		req, err = http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product2.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(err)
		assert.Equal(int32(29), product.StockQuantity)
	})
}

func TestCreateOrderWithMultipleProducts(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(apiGatewayURLKey)

	_, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)

	product1 := createProduct(require, apiGatewayURL, authToken, "Multi Product 1", 50.00, 100)
	product2 := createProduct(require, apiGatewayURL, authToken, "Multi Product 2", 75.50, 80)
	product3 := createProduct(require, apiGatewayURL, authToken, "Multi Product 3", 120.25, 60)
	product4 := createProduct(require, apiGatewayURL, authToken, "Multi Product 4", 200.00, 40)

	t.Run("Add Multiple Products to Cart", func(t *testing.T) {
		products := []struct {
			ID       string
			Quantity int32
		}{
			{product1.ID, 3},
			{product2.ID, 2},
			{product3.ID, 1},
			{product4.ID, 4},
		}

		client := &http.Client{}
		for _, p := range products {
			addProductReq := AddProductToCartRequest{
				ProductID: p.ID,
				Quantity:  p.Quantity,
			}

			body, err := json.Marshal(addProductReq)
			require.NoError(err)

			req, err := http.NewRequest("POST", fmt.Sprintf(apiCartsProducts, apiGatewayURL), bytes.NewBuffer(body))
			require.NoError(err)
			req.Header.Set("Authorization", bearerWithSpace+authToken)
			req.Header.Set(contentTypeHeader, contentTypeJSON)

			resp, err := client.Do(req)
			require.NoError(err)
			defer resp.Body.Close()

			assert.Equal(http.StatusOK, resp.StatusCode)
		}
	})

	var createdOrder Order
	t.Run("Create Order with Multiple Products", func(t *testing.T) {
		req, err := http.NewRequest("POST", fmt.Sprintf(apiOrders, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdOrder)
		require.NoError(err)
		assert.NotEmpty(createdOrder.ID)
		assert.Equal("CREATED", createdOrder.Status)

		expectedTotal := (product1.Price * 3) + (product2.Price * 2) + (product3.Price * 1) + (product4.Price * 4)
		assert.Equal(expectedTotal, createdOrder.TotalAmount)
	})

	t.Run("Verify All Products Stock Updated", func(t *testing.T) {
		time.Sleep(3 * time.Second)

		client := &http.Client{}

		req, err := http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product1.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		var product Product
		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(err)
		assert.Equal(int32(97), product.StockQuantity)

		req, err = http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product2.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(err)
		assert.Equal(int32(78), product.StockQuantity)

		req, err = http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product3.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(err)
		assert.Equal(int32(59), product.StockQuantity)

		req, err = http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product4.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(err)
		assert.Equal(int32(36), product.StockQuantity)
	})

	t.Run("Verify Cart Cleared After Order", func(t *testing.T) {
		time.Sleep(2 * time.Second)

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
		assert.Empty(cart.Products)
		assert.Equal(0.0, cart.TotalPrice)
	})
}

func TestCreateOrderWithSingleProduct(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(apiGatewayURLKey)

	_, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)

	product := createProduct(require, apiGatewayURL, authToken, "Single Product Order", 99.99, 25)

	t.Run("Add Single Product to Cart", func(t *testing.T) {
		addProductReq := AddProductToCartRequest{
			ProductID: product.ID,
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
	})

	var createdOrder Order
	t.Run("Create Order with Single Product", func(t *testing.T) {
		req, err := http.NewRequest("POST", fmt.Sprintf(apiOrders, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdOrder)
		require.NoError(err)
		assert.NotEmpty(createdOrder.ID)
		assert.Equal("CREATED", createdOrder.Status)
		assert.Equal(product.Price*5, createdOrder.TotalAmount)
	})

	t.Run("Verify Single Product Stock Updated", func(t *testing.T) {
		time.Sleep(5 * time.Second)

		req, err := http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var updatedProduct Product
		err = json.NewDecoder(resp.Body).Decode(&updatedProduct)
		require.NoError(err)
		assert.Equal(int32(20), updatedProduct.StockQuantity)
	})

	t.Run("Verify Cart Cleared", func(t *testing.T) {
		time.Sleep(1 * time.Second)

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
		assert.Empty(cart.Products)
		assert.Equal(0.0, cart.TotalPrice)
	})
}

func TestCreateOrderFailureDueToInsufficientStock(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(apiGatewayURLKey)

	_, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)

	initialStock := int32(1)
	product := createProduct(require, apiGatewayURL, authToken, "Low Stock Product", 10.00, initialStock)
	require.NotEmpty(product.ID)

	quantityToOrder := int32(2)
	addProductReq := AddProductToCartRequest{
		ProductID: product.ID,
		Quantity:  quantityToOrder,
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
	assert.Equal(http.StatusOK, resp.StatusCode, "Should be able to add to cart even if stock is insufficient at this stage")

	var createdOrder Order
	t.Run("Create Order with Insufficient Stock", func(t *testing.T) {
		req, err := http.NewRequest("POST", fmt.Sprintf(apiOrders, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err = client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusCreated, resp.StatusCode, "Order should be initially created")

		err = json.NewDecoder(resp.Body).Decode(&createdOrder)
		require.NoError(err)
		assert.NotEmpty(createdOrder.ID)
		assert.Equal("CREATED", createdOrder.Status, "Initial order status should be CREATED")
	})

	time.Sleep(5 * time.Second)

	// t.Run("Verify Order Status is CANCELLED", func(t *testing.T) {
	// 	req, err := http.NewRequest("GET", fmt.Sprintf(apiOrders, apiGatewayURL)+"/"+createdOrder.ID, nil)
	// 	require.NoError(err)
	// 	req.Header.Set("Authorization", bearerWithSpace+authToken)

	// 	resp, err := client.Do(req)
	// 	require.NoError(err)
	// 	defer resp.Body.Close()

	// 	assert.Equal(http.StatusOK, resp.StatusCode)

	// 	var fetchedOrder Order
	// 	err = json.NewDecoder(resp.Body).Decode(&fetchedOrder)
	// 	require.NoError(err)
	// 	assert.Equal("CANCELLED", fetchedOrder.Status, "Order status should be CANCELLED due to insufficient stock")
	// })

	t.Run("Verify Stock is NOT Updated", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiProductsWithPath, apiGatewayURL, product.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var fetchedProduct Product
		err = json.NewDecoder(resp.Body).Decode(&fetchedProduct)
		require.NoError(err)
		assert.Equal(initialStock, fetchedProduct.StockQuantity, "Product stock should remain unchanged")
	})

	t.Run("Verify Cart is Cleared", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf(apiCarts, apiGatewayURL), nil)
		require.NoError(err)
		req.Header.Set("Authorization", bearerWithSpace+authToken)

		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var cart Cart
		err = json.NewDecoder(resp.Body).Decode(&cart)
		require.NoError(err)
		assert.Empty(cart.Products, "Cart should be empty after order attempt")
		assert.Equal(0.0, cart.TotalPrice, "Total price should be 0 for empty cart")
	})
}
