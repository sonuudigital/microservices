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
	productCategoriesEndpoint = "api/products/categories"
)

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

func TestProductCategoriesCRUD(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")

	_, authToken := registerAndLogin(require)
	require.NotEmpty(authToken)

	var createdProductCategories ProductCategory

	t.Run("Create Product Category", func(t *testing.T) {
		productCategoryReq := ProductCategoryRequest{
			Name:        fmt.Sprintf("TEST CATEGORY%d", time.Now().Unix()),
			Description: fmt.Sprintf("TEST DESCRIPTION%d", time.Now().Unix()),
		}

		body, err := json.Marshal(productCategoryReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", apiGatewayURL, productCategoriesEndpoint), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Content-Type", "application/json")
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
		assert.NotEmpty(createdProductCategories.ID)
		assert.NotEmpty(createdProductCategories.CreatedAt)
	})

	t.Run("Get Product Categories", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", apiGatewayURL, productCategoriesEndpoint), nil)
		require.NoError(err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusOK, resp.StatusCode)

		var productCategories []ProductCategory
		err = json.NewDecoder(resp.Body).Decode(&productCategories)
		require.NoError(err)

		var found bool
		for _, category := range productCategories {
			if category.ID == createdProductCategories.ID {
				found = true
				assert.Equal(createdProductCategories.Name, category.Name)
				assert.Equal(createdProductCategories.Description, category.Description)
				break
			}
		}
		assert.True(found, "Created product category not found in the list")
	})

	t.Run("Update Product Category", func(t *testing.T) {
		updateNameCategory := UpdateCategoryRequest{
			ID: createdProductCategories.ID,
			ProductCategoryRequest: ProductCategoryRequest{
				Name:        "UPDATED CATEGORY NAME",
				Description: createdProductCategories.Description,
			},
		}

		body, err := json.Marshal(updateNameCategory)
		require.NoError(err)

		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/%s", apiGatewayURL, productCategoriesEndpoint), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
		assert.Empty(resp.ContentLength)
	})
}
