package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shahadulhaider/verso/services/verso-search-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
)

func TestSearch_EmptyQuery(t *testing.T) {
	h := handler.New(opensearch.New("http://localhost:9200", nil), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("expected empty results, got %d", len(resp.Results))
	}
}

func TestSearch_UnsupportedType(t *testing.T) {
	h := handler.New(opensearch.New("http://localhost:9200", nil), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=test&type=author", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSearch_TypeWorkAccepted(t *testing.T) {
	h := handler.New(opensearch.New("http://localhost:9200", nil), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=&type=work", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	h := handler.New(opensearch.New("http://localhost:9200", nil), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", resp["status"])
	}
}

func TestSemanticSearch_NotConfigured(t *testing.T) {
	h := handler.New(opensearch.New("http://localhost:9200", nil), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/semantic?q=cozy+mystery", nil)
	rec := httptest.NewRecorder()

	h.SemanticSearch(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var resp struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "semantic search not configured" {
		t.Fatalf("expected 'semantic search not configured', got %q", resp.Error)
	}
}

func TestSemanticSearch_EmptyQuery(t *testing.T) {
	h := handler.New(opensearch.New("http://localhost:9200", nil), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/semantic?q=", nil)
	rec := httptest.NewRecorder()

	h.SemanticSearch(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestSearch_ModeDefault(t *testing.T) {
	h := handler.New(opensearch.New("http://localhost:9200", nil), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	var resp struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Mode != "text" {
		t.Fatalf("expected mode=text, got %q", resp.Mode)
	}
}
