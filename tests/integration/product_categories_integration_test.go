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

func TestProductCategoriesCRUD(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	apiGatewayURL := os.Getenv(ApiGatewayURLKey)

	_, authToken := RegisterAndLogin(require)
	require.NotEmpty(authToken)

	httpClient := &http.Client{}
	var createdProductCategories ProductCategory

	t.Run("Create Product Category", func(t *testing.T) {
		productCategoryReq := ProductCategoryRequest{
			Name:        fmt.Sprintf("TEST CATEGORY%d", time.Now().Unix()),
			Description: fmt.Sprintf("TEST DESCRIPTION%d", time.Now().Unix()),
		}

		body, err := json.Marshal(productCategoryReq)
		require.NoError(err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", apiGatewayURL, ProductCategoriesEndpoint), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		resp, err := httpClient.Do(req)
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
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", apiGatewayURL, ProductCategoriesEndpoint), nil)
		require.NoError(err)

		resp, err := httpClient.Do(req)
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

		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/%s", apiGatewayURL, ProductCategoriesEndpoint), bytes.NewBuffer(body))
		require.NoError(err)
		req.Header.Set(ContentTypeHeader, ContentTypeJSON)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		resp, err := httpClient.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
		assert.Empty(resp.ContentLength)
	})

	t.Run("Delete Product Category", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s/%s", apiGatewayURL, ProductCategoriesEndpoint, createdProductCategories.ID), nil)
		require.NoError(err)
		req.Header.Set("Authorization", BearerWithSpace+authToken)

		resp, err := httpClient.Do(req)
		require.NoError(err)
		defer resp.Body.Close()

		assert.Equal(http.StatusNoContent, resp.StatusCode)
		assert.Empty(resp.ContentLength)
	})
}