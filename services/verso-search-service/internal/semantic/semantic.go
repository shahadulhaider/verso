// Package semantic provides vector similarity search via pgvector and
// query embedding generation via the LLM gateway.
package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	gobreaker "github.com/sony/gobreaker/v2"
)

// SearchResult represents a single vector similarity match.
type SearchResult struct {
	WorkID string  `json:"workId"`
	Title  string  `json:"title,omitempty"`
	Score  float64 `json:"score"`
}

// embedRequest is the JSON body for POST /v1/llm/embed.
type embedRequest struct {
	Text string `json:"text"`
}

// embedResponse is the JSON response from the LLM gateway embed endpoint.
type embedResponse struct {
	Embedding  []float32 `json:"embedding"`
	Dimensions int       `json:"dimensions"`
	ModelID    string    `json:"modelId"`
}

// Client provides semantic search operations.
type Client struct {
	pool       *pgxpool.Pool
	gatewayURL string
	httpClient *http.Client
	breaker    *gobreaker.CircuitBreaker[[]float32]
	log        *slog.Logger
}

// New creates a semantic search client backed by pgvector and the LLM gateway.
func New(pool *pgxpool.Pool, gatewayURL string, log *slog.Logger) *Client {
	cb := gobreaker.NewCircuitBreaker[[]float32](gobreaker.Settings{
		Name:        "llm-gateway-embed",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	return &Client{
		pool:       pool,
		gatewayURL: gatewayURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		breaker:    cb,
		log:        log,
	}
}

// IsAvailable reports whether the circuit breaker allows requests to the LLM gateway.
func (c *Client) IsAvailable() bool {
	return c.breaker.State() != gobreaker.StateOpen
}

// QueryEmbedding calls the LLM gateway to generate an embedding for the query text.
// The call is protected by a circuit breaker.
func (c *Client) QueryEmbedding(ctx context.Context, text string) ([]float32, error) {
	return c.breaker.Execute(func() ([]float32, error) {
		return c.callGateway(ctx, text)
	})
}

func (c *Client) callGateway(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	url := c.gatewayURL + "/v1/llm/embed"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm gateway request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm gateway: status=%d body=%s", resp.StatusCode, respBody)
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	return result.Embedding, nil
}

// FindSimilar runs a pgvector cosine nearest-neighbor search against
// ai.work_embedding using the provided embedding vector.
func (c *Client) FindSimilar(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// pgvector cosine distance: <=> (lower = more similar)
	// Score = 1 - distance
	rows, err := c.pool.Query(ctx, `
		SELECT we.work_id, 1 - (we.embedding <=> $1::vector) AS score
		FROM ai.work_embedding we
		ORDER BY we.embedding <=> $1::vector
		LIMIT $2
	`, pgvectorString(embedding), limit)
	if err != nil {
		return nil, fmt.Errorf("pgvector query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.WorkID, &r.Score); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []SearchResult{}
	}
	return results, nil
}

// pgvectorString formats a float32 slice as a pgvector literal string: "[0.1,0.2,...]".
func pgvectorString(v []float32) string {
	buf := bytes.NewBufferString("[")
	for i, f := range v {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(buf, "%g", f)
	}
	buf.WriteByte(']')
	return buf.String()
}
