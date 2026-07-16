package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shahadulhaider/verso/services/verso-search-service/internal/hybrid"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/semantic"
)

type SearchHandler struct {
	os       *opensearch.Client
	semantic *semantic.Client
	hybrid   *hybrid.Searcher
}

// New creates a SearchHandler. The semantic and hybrid parameters may be nil
// if semantic search is not configured.
func New(osClient *opensearch.Client, semClient *semantic.Client, hybridSearcher *hybrid.Searcher) *SearchHandler {
	return &SearchHandler{
		os:       osClient,
		semantic: semClient,
		hybrid:   hybridSearcher,
	}
}

type searchResponse struct {
	Results  []opensearch.SearchHit `json:"results"`
	Mode     string                 `json:"mode,omitempty"`
	Degraded bool                   `json:"degraded,omitempty"`
}

type semanticResponse struct {
	Results []semantic.SearchResult `json:"results"`
	Mode    string                  `json:"mode"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query().Get("q")
	typ := r.URL.Query().Get("type")
	mode := r.URL.Query().Get("mode")

	if typ != "" && typ != "work" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "unsupported type: only 'work' is supported"})
		return
	}

	if q == "" {
		json.NewEncoder(w).Encode(searchResponse{Results: []opensearch.SearchHit{}, Mode: modeOrDefault(mode)})
		return
	}

	if mode == "hybrid" && h.hybrid != nil {
		resp, err := h.hybrid.Search(r.Context(), q, 20)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "hybrid search failed"})
			return
		}
		json.NewEncoder(w).Encode(resp)
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
	json.NewEncoder(w).Encode(searchResponse{Results: hits, Mode: "text"})
}

// SemanticSearch handles GET /v1/search/semantic?q={query}.
func (h *SearchHandler) SemanticSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.semantic == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(errorResponse{Error: "semantic search not configured"})
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		json.NewEncoder(w).Encode(semanticResponse{Results: []semantic.SearchResult{}, Mode: "semantic"})
		return
	}

	if !h.semantic.IsAvailable() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(errorResponse{Error: "semantic search unavailable, use /v1/search for full-text"})
		return
	}

	embedding, err := h.semantic.QueryEmbedding(r.Context(), q)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(errorResponse{Error: "semantic search unavailable, use /v1/search for full-text"})
		return
	}

	results, err := h.semantic.FindSimilar(r.Context(), embedding, 15)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "semantic search failed"})
		return
	}

	json.NewEncoder(w).Encode(semanticResponse{Results: results, Mode: "semantic"})
}

func (h *SearchHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

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

func modeOrDefault(mode string) string {
	if mode == "" {
		return "text"
	}
	return mode
}
