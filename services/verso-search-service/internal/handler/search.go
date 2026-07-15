package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
)

// SearchHandler serves the search API.
type SearchHandler struct {
	os *opensearch.Client
}

// New creates a new SearchHandler.
func New(osClient *opensearch.Client) *SearchHandler {
	return &SearchHandler{os: osClient}
}

type searchResponse struct {
	Results []opensearch.SearchHit `json:"results"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Search handles GET /v1/search?q={query}&type=work.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query().Get("q")
	typ := r.URL.Query().Get("type")

	if typ != "" && typ != "work" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "unsupported type: only 'work' is supported"})
		return
	}

	if q == "" {
		json.NewEncoder(w).Encode(searchResponse{Results: []opensearch.SearchHit{}})
		return
	}

	hits, err := h.os.Search(r.Context(), q)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "search failed"})
		return
	}

	if hits == nil {
		hits = []opensearch.SearchHit{}
	}
	json.NewEncoder(w).Encode(searchResponse{Results: hits})
}

// Health returns 200 OK if the service is alive.
func (h *SearchHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// Ready returns 200 if OpenSearch is reachable.
func (h *SearchHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := h.os.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(errorResponse{Error: "opensearch not reachable"})
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}
