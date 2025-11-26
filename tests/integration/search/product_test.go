package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/sonuudigital/microservices/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxRetries    = 20
	retryInterval = 1 * time.Second
)

type SearchResultProduct struct {
	ID            string `json:"id"`
	CategoryID    string `json:"categoryId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         string `json:"price"`
	StockQuantity int32  `json:"stockQuantity"`
}

type testCase struct {
	name          string
	query         string
	expectedCount int
	expectedNames []string
	expectEmpty   bool
}

func TestProductSearch(t *testing.T) {
	if os.Getenv(integration.ApiGatewayURLKey) == "" {
		t.Skip("API_GATEWAY_URL not set, skipping integration tests")
	}

	req := require.New(t)
	apiGatewayURL := os.Getenv(integration.ApiGatewayURLKey)
	_, authToken := integration.RegisterAndLogin(req)

	timestamp := time.Now().UnixNano()
	productNames := generateProductNames(timestamp)
	createdProducts := seedProducts(req, apiGatewayURL, authToken, productNames)

	t.Logf("Waiting for products to be indexed in OpenSearch (timestamp: %d)...", timestamp)
	waitForIndexing(t, apiGatewayURL, fmt.Sprintf("%d", timestamp))

	tests := buildTestCases(productNames, timestamp)
	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executeSearchTest(t, client, apiGatewayURL, tt, createdProducts)
		})
	}
}

func TestProductDeletion(t *testing.T) {
	if os.Getenv(integration.ApiGatewayURLKey) == "" {
		t.Skip("API_GATEWAY_URL not set, skipping integration tests")
	}

	req := require.New(t)
	apiGatewayURL := os.Getenv(integration.ApiGatewayURLKey)
	_, authToken := integration.RegisterAndLogin(req)

	timestamp := time.Now().UnixNano()
	productName := fmt.Sprintf("ProductToDelete %d", timestamp)

	product := integration.CreateProduct(req, apiGatewayURL, authToken, productName, 50.00, 5)
	t.Logf("Created product to delete: %s (ID: %s)", productName, product.ID)

	waitForIndexing(t, apiGatewayURL, productName)
	deleteProduct(t, req, apiGatewayURL, authToken, product.ID)
	waitForDeletion(t, apiGatewayURL, productName, product.ID)
}

func generateProductNames(timestamp int64) map[string]string {
	return map[string]string{
		"iPhone":     fmt.Sprintf("SearchPhone iPhone 15 %d", timestamp),
		"Laptop":     fmt.Sprintf("SearchLaptop MacBook Pro %d", timestamp),
		"Headphones": fmt.Sprintf("SearchHeadphones Sony WH-1000XM5 %d", timestamp),
	}
}

func seedProducts(req *require.Assertions, apiGatewayURL, authToken string, names map[string]string) []integration.Product {
	productsToSeed := []struct {
		Name        string
		Description string
		Price       float64
	}{
		{
			Name:        names["iPhone"],
			Description: "Latest Apple smartphone with A17 Bionic chip",
			Price:       999.99,
		},
		{
			Name:        names["Laptop"],
			Description: "Powerful laptop for professionals with M3 chip",
			Price:       1999.99,
		},
		{
			Name:        names["Headphones"],
			Description: "Premium noise canceling headphones",
			Price:       349.99,
		},
	}

	createdProducts := make([]integration.Product, 0, len(productsToSeed))
	for _, p := range productsToSeed {
		prod := integration.CreateProduct(req, apiGatewayURL, authToken, p.Name, p.Price, 10, p.Description)
		createdProducts = append(createdProducts, prod)
	}
	return createdProducts
}

func buildTestCases(productNames map[string]string, timestamp int64) []testCase {
	return []testCase{
		{
			name:          "Search Exact Name",
			query:         productNames["iPhone"],
			expectedCount: 1,
			expectedNames: []string{productNames["iPhone"]},
		},
		{
			name:          "Search Partial Name",
			query:         fmt.Sprintf("MacBook Pro %d", timestamp),
			expectedCount: 1,
			expectedNames: []string{productNames["Laptop"]},
		},
		{
			name:          "Search by Description Keyword",
			query:         "noise canceling",
			expectedCount: 1,
			expectedNames: []string{productNames["Headphones"]},
		},
		{
			name:          "Search Multiple Matches (chip)",
			query:         "chip",
			expectedCount: 2,
			expectedNames: []string{productNames["iPhone"], productNames["Laptop"]},
		},
		{
			name:        "Search Non-Existent",
			query:       "Refrigerator",
			expectEmpty: true,
		},
	}
}

func executeSearchTest(t *testing.T, client *http.Client, apiGatewayURL string, tt testCase, createdProducts []integration.Product) {
	asrt := assert.New(t)
	req := require.New(t)

	statusCode, results := performSearch(req, client, apiGatewayURL, tt.query, tt.expectedCount)

	if tt.expectEmpty {
		asrt.Equal(http.StatusNotFound, statusCode)
		asrt.Empty(results)
	} else {
		asrt.Equal(http.StatusOK, statusCode)
		verifySearchResults(asrt, results, createdProducts, tt)
	}
}

func performSearch(req *require.Assertions, client *http.Client, apiGatewayURL, query string, expectedCount int) (int, []SearchResultProduct) {
	reqURL := buildSearchURL(apiGatewayURL, query, expectedCount)

	httpReq, err := http.NewRequest("GET", reqURL, nil)
	req.NoError(err)

	resp, err := client.Do(httpReq)
	req.NoError(err)
	defer resp.Body.Close()

	var results []SearchResultProduct
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&results)
		req.NoError(err)
	}

	return resp.StatusCode, results
}

func buildSearchURL(apiGatewayURL, query string, expectedCount int) string {
	baseURL := fmt.Sprintf("%s/api/search/products", apiGatewayURL)
	params := url.Values{}
	params.Add("q", query)
	if expectedCount > 10 {
		params.Add("size", fmt.Sprintf("%d", expectedCount+5))
	}
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func verifySearchResults(asrt *assert.Assertions, results []SearchResultProduct, createdProducts []integration.Product, tt testCase) {
	matched := filterMatchedProducts(results, createdProducts)
	asrt.GreaterOrEqual(len(matched), tt.expectedCount, "Expected at least %d matched seeded products", tt.expectedCount)

	if len(tt.expectedNames) > 0 {
		verifyExpectedNames(asrt, matched, tt.expectedNames)
	}
}

func filterMatchedProducts(results []SearchResultProduct, createdProducts []integration.Product) []SearchResultProduct {
	var matched []SearchResultProduct
	for _, res := range results {
		for _, seeded := range createdProducts {
			if res.ID == seeded.ID {
				matched = append(matched, res)
				break
			}
		}
	}
	return matched
}

func verifyExpectedNames(asrt *assert.Assertions, matched []SearchResultProduct, expectedNames []string) {
	foundNames := make(map[string]bool)
	for _, m := range matched {
		foundNames[m.Name] = true
	}
	for _, name := range expectedNames {
		asrt.True(foundNames[name], "Expected result to contain product with name: %s", name)
	}
}

func deleteProduct(t *testing.T, req *require.Assertions, apiGatewayURL, authToken, productID string) {
	deleteURL := fmt.Sprintf(integration.ApiProductsWithPath, apiGatewayURL, productID)
	httpReq, err := http.NewRequest("DELETE", deleteURL, nil)
	req.NoError(err)
	httpReq.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	req.NoError(err)
	defer resp.Body.Close()
	req.Equal(http.StatusNoContent, resp.StatusCode)
	t.Log("Product deleted via API")
}

func waitForIndexing(t *testing.T, apiGatewayURL, query string) {
	client := &http.Client{}
	reqURL := buildIndexingCheckURL(apiGatewayURL, query)

	t.Logf("Polling search endpoint: %s", reqURL)

	for i := range maxRetries {
		if checkIndexingComplete(t, client, reqURL, query, i) {
			return
		}
		time.Sleep(retryInterval)
	}

	t.Fatalf("Timed out waiting for query '%s' to return results after %d attempts", query, maxRetries)
}

func waitForDeletion(t *testing.T, apiGatewayURL, query, productID string) {
	client := &http.Client{}
	reqURL := buildIndexingCheckURL(apiGatewayURL, query)

	t.Logf("Polling search endpoint for deletion: %s", reqURL)

	for i := range maxRetries {
		if checkDeletionComplete(t, client, reqURL, query, productID, i) {
			return
		}
		time.Sleep(retryInterval)
	}

	t.Fatalf("Timed out waiting for product '%s' (ID: %s) to be removed from index after %d attempts", query, productID, maxRetries)
}

func buildIndexingCheckURL(apiGatewayURL, query string) string {
	baseURL := fmt.Sprintf("%s/api/search/products", apiGatewayURL)
	params := url.Values{}
	params.Add("q", query)
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func checkIndexingComplete(t *testing.T, client *http.Client, reqURL, query string, attempt int) bool {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Retry %d/%d: Request failed: %v", attempt+1, maxRetries, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Logf("Retry %d/%d: Status %d", attempt+1, maxRetries, resp.StatusCode)
		return false
	}

	var results []SearchResultProduct
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Logf("Retry %d/%d: Decode failed: %v", attempt+1, maxRetries, err)
		return false
	}

	if len(results) > 0 {
		t.Logf("Found %d results for query '%s'", len(results), query)
		return true
	}

	return false
}

func checkDeletionComplete(t *testing.T, client *http.Client, reqURL, query, productID string, attempt int) bool {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Retry %d/%d: Request failed: %v", attempt+1, maxRetries, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Logf("Product '%s' (ID: %s) successfully removed from index (Status 404)", query, productID)
		return true
	}

	if resp.StatusCode != http.StatusOK {
		t.Logf("Retry %d/%d: Status %d", attempt+1, maxRetries, resp.StatusCode)
		return false
	}

	var results []SearchResultProduct
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Logf("Retry %d/%d: Decode failed: %v", attempt+1, maxRetries, err)
		return false
	}

	found := false
	for _, p := range results {
		if p.ID == productID {
			found = true
			break
		}
	}

	if !found {
		t.Logf("Product '%s' (ID: %s) successfully removed from index", query, productID)
		return true
	}

	t.Logf("Retry %d/%d: Product still found in index", attempt+1, maxRetries)
	return false
}
