package product

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sonuudigital/microservices/shared/events"
	"github.com/sonuudigital/microservices/shared/web"
)

type Query map[string]any

type SearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source events.Product `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (h *ProductHandler) SearchProduct(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		web.RespondWithError(w, h.logger, r, http.StatusBadRequest, "Query parameter 'q' is required", "The 'q' parameter cannot be empty")
		return
	}

	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	from, _ := strconv.Atoi(r.URL.Query().Get("from"))
	if from < 0 {
		from = 0
	}

	searchQuery := Query{
		"from": from,
		"size": size,
		"query": Query{
			"multi_match": Query{
				"query":  query,
				"fields": []string{"name", "description"},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Failed to encode search query", err.Error())
		return
	}

	res, err := h.searcher.Search(r.Context(), h.index, &buf)
	if err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Failed to execute search query", err.Error())
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Opensearch returned an error", "Status code: "+strconv.Itoa(res.StatusCode))
		return
	}

	var searchRes SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchRes); err != nil {
		web.RespondWithError(w, h.logger, r, http.StatusInternalServerError, "Failed to decode search response", err.Error())
		return
	}

	if searchRes.Hits.Total.Value == 0 {
		web.RespondWithError(w, h.logger, r, http.StatusNotFound, "No products found", "No results match the search query")
		return
	}

	products := make([]events.Product, len(searchRes.Hits.Hits))
	for i, hit := range searchRes.Hits.Hits {
		products[i] = hit.Source
	}

	web.RespondWithJSON(w, h.logger, http.StatusOK, products)
}
