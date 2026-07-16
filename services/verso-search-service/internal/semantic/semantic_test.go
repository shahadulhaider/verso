package semantic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryEmbedding_Success(t *testing.T) {
	mockGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/llm/embed" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Text == "" {
			t.Fatal("expected non-empty text")
		}

		resp := embedResponse{
			Embedding:  []float32{0.1, 0.2, 0.3},
			Dimensions: 3,
			ModelID:    "nomic-embed-text",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockGateway.Close()

	client := New(nil, mockGateway.URL, nil)

	embedding, err := client.QueryEmbedding(context.Background(), "cozy mystery")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embedding) != 3 {
		t.Fatalf("expected 3 dimensions, got %d", len(embedding))
	}
	if embedding[0] != 0.1 {
		t.Fatalf("expected first dim=0.1, got %f", embedding[0])
	}
}

func TestQueryEmbedding_GatewayError(t *testing.T) {
	mockGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"model unavailable"}`))
	}))
	defer mockGateway.Close()

	client := New(nil, mockGateway.URL, nil)

	_, err := client.QueryEmbedding(context.Background(), "test query")
	if err == nil {
		t.Fatal("expected error from gateway 500")
	}
}

func TestQueryEmbedding_CircuitBreaker(t *testing.T) {
	callCount := 0
	mockGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockGateway.Close()

	client := New(nil, mockGateway.URL, nil)

	for i := 0; i < 5; i++ {
		client.QueryEmbedding(context.Background(), "test")
	}

	if client.IsAvailable() {
		t.Fatal("circuit breaker should be open after 3+ consecutive failures")
	}

	if callCount > 4 {
		t.Fatalf("circuit breaker should have prevented some calls, but %d calls were made", callCount)
	}
}

func TestIsAvailable_InitiallyTrue(t *testing.T) {
	client := New(nil, "http://localhost:9999", nil)
	if !client.IsAvailable() {
		t.Fatal("circuit breaker should be closed initially")
	}
}

func TestPgvectorString(t *testing.T) {
	tests := []struct {
		input    []float32
		expected string
	}{
		{[]float32{}, "[]"},
		{[]float32{1.0}, "[1]"},
		{[]float32{0.1, 0.2, 0.3}, "[0.1,0.2,0.3]"},
		{[]float32{-1.5, 0, 2.5}, "[-1.5,0,2.5]"},
	}

	for _, tt := range tests {
		result := pgvectorString(tt.input)
		if result != tt.expected {
			t.Errorf("pgvectorString(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
