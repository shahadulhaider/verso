// Package opensearch provides an HTTP client for OpenSearch operations.
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type WorkDocument struct {
	WorkID                  string   `json:"work_id"`
	Title                   string   `json:"title"`
	Description             string   `json:"description,omitempty"`
	Genres                  []string `json:"genres,omitempty"`
	ContributorNames        []string `json:"contributor_names,omitempty"`
	OriginalLanguage        string   `json:"original_language,omitempty"`
	OriginalPublicationYear int      `json:"original_publication_year,omitempty"`
	RatingsCount            int      `json:"ratings_count,omitempty"`
	AvgRating               float64  `json:"avg_rating,omitempty"`
	CreatedAt               string   `json:"created_at"`
}

// SearchHit represents a single search result.
type SearchHit struct {
	WorkID      string  `json:"workId"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Score       float64 `json:"score"`
}

// Client is an HTTP client for OpenSearch.
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

// New creates a new OpenSearch client.
func New(baseURL string, log *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: log,
	}
}

const worksIndexMapping = `{
	"mappings": {
		"properties": {
			"work_id":                   { "type": "keyword" },
			"title":                     { "type": "text" },
			"description":               { "type": "text" },
			"genres":                     { "type": "keyword" },
			"contributor_names":          { "type": "text" },
			"original_language":          { "type": "keyword" },
			"original_publication_year":  { "type": "integer" },
			"ratings_count":             { "type": "integer" },
			"avg_rating":                { "type": "float" },
			"created_at":                { "type": "date" }
		}
	}
}`

// EnsureIndex creates the works index if it does not exist.
func (c *Client) EnsureIndex(ctx context.Context) error {
	url := c.baseURL + "/works"

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("build HEAD request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("check index: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		c.log.Info("works index already exists")
		return nil
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBufferString(worksIndexMapping))
	if err != nil {
		return fmt.Errorf("build PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create index: status=%d body=%s", resp.StatusCode, body)
	}

	c.log.Info("works index created")
	return nil
}

// IndexDocument indexes a work document by its ID.
func (c *Client) IndexDocument(ctx context.Context, doc *WorkDocument) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/works/_doc/%s", c.baseURL, doc.WorkID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("index document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("index document: status=%d body=%s", resp.StatusCode, respBody)
	}

	return nil
}

// searchRequest is the OpenSearch query body for multi_match.
type searchRequest struct {
	Query searchQuery `json:"query"`
}

type searchQuery struct {
	MultiMatch *multiMatchQuery `json:"multi_match"`
}

type multiMatchQuery struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
}

// searchResponse represents the OpenSearch search response.
type searchResponse struct {
	Hits searchHits `json:"hits"`
}

type searchHits struct {
	Hits []searchHitWrapper `json:"hits"`
}

type searchHitWrapper struct {
	Source json.RawMessage `json:"_source"`
	Score  float64         `json:"_score"`
}

// Search queries the works index and returns matching hits.
func (c *Client) Search(ctx context.Context, query string) ([]SearchHit, error) {
	body, err := json.Marshal(searchRequest{
		Query: searchQuery{
			MultiMatch: &multiMatchQuery{
				Query:  query,
				Fields: []string{"title^3", "description", "contributor_names"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	url := c.baseURL + "/works/_search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search: status=%d body=%s", resp.StatusCode, respBody)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	hits := make([]SearchHit, 0, len(sr.Hits.Hits))
	for _, h := range sr.Hits.Hits {
		var doc WorkDocument
		if err := json.Unmarshal(h.Source, &doc); err != nil {
			c.log.Warn("skip malformed hit", slog.String("error", err.Error()))
			continue
		}
		hits = append(hits, SearchHit{
			WorkID:      doc.WorkID,
			Title:       doc.Title,
			Description: doc.Description,
			Score:       h.Score,
		})
	}

	return hits, nil
}

// UpdateWorkRating increments ratings_count and recalculates avg_rating
// using a painless script (running average).
func (c *Client) UpdateWorkRating(ctx context.Context, workID string, newRating float64) error {
	script := map[string]interface{}{
		"script": map[string]interface{}{
			"source": `
				ctx._source.ratings_count = (ctx._source.ratings_count ?: 0) + 1;
				int count = ctx._source.ratings_count;
				double oldAvg = (ctx._source.avg_rating ?: 0.0);
				ctx._source.avg_rating = oldAvg + (params.rating - oldAvg) / count;
			`,
			"lang": "painless",
			"params": map[string]interface{}{
				"rating": newRating,
			},
		},
	}

	body, err := json.Marshal(script)
	if err != nil {
		return fmt.Errorf("marshal update script: %w", err)
	}

	url := fmt.Sprintf("%s/works/_update/%s", c.baseURL, workID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update work rating: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update work rating: status=%d body=%s", resp.StatusCode, respBody)
	}

	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opensearch ping: status=%d", resp.StatusCode)
	}
	return nil
}
