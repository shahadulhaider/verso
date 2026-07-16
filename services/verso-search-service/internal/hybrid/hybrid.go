// Package hybrid merges OpenSearch full-text results with pgvector semantic results.
package hybrid

import (
	"context"
	"sort"

	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/semantic"
)

const (
	textWeight     = 0.6
	semanticWeight = 0.4
)

// Result is a merged search result with a normalized combined score.
type Result struct {
	WorkID      string  `json:"workId"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Score       float64 `json:"score"`
}

// Response is the response body for hybrid search.
type Response struct {
	Results  []Result `json:"results"`
	Mode     string   `json:"mode"`
	Degraded bool     `json:"degraded,omitempty"`
}

// Searcher combines full-text and semantic search.
type Searcher struct {
	os       *opensearch.Client
	semantic *semantic.Client
}

// New creates a hybrid Searcher.
func New(osClient *opensearch.Client, semClient *semantic.Client) *Searcher {
	return &Searcher{os: osClient, semantic: semClient}
}

// Search runs both full-text and semantic search, normalizes scores, and merges.
// If semantic search is unavailable (circuit breaker open), returns text-only with degraded=true.
func (s *Searcher) Search(ctx context.Context, query string, limit int) (*Response, error) {
	if limit <= 0 {
		limit = 20
	}

	textHits, err := s.os.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	if !s.semantic.IsAvailable() {
		return &Response{
			Results:  textHitsToResults(textHits),
			Mode:     "hybrid",
			Degraded: true,
		}, nil
	}

	embedding, err := s.semantic.QueryEmbedding(ctx, query)
	if err != nil {
		return &Response{
			Results:  textHitsToResults(textHits),
			Mode:     "hybrid",
			Degraded: true,
		}, nil
	}

	semHits, err := s.semantic.FindSimilar(ctx, embedding, limit)
	if err != nil {
		return &Response{
			Results:  textHitsToResults(textHits),
			Mode:     "hybrid",
			Degraded: true,
		}, nil
	}

	merged := merge(textHits, semHits, limit)
	return &Response{
		Results: merged,
		Mode:    "hybrid",
	}, nil
}

func textHitsToResults(hits []opensearch.SearchHit) []Result {
	results := make([]Result, len(hits))
	for i, h := range hits {
		results[i] = Result{
			WorkID:      h.WorkID,
			Title:       h.Title,
			Description: h.Description,
			Score:       h.Score,
		}
	}
	return results
}

// merge normalizes both score sets to [0,1] via min-max, then combines
// with weighted average (0.6 text + 0.4 semantic). Deduplicates by workId.
func merge(textHits []opensearch.SearchHit, semHits []semantic.SearchResult, limit int) []Result {
	textNorm := normalizeTextScores(textHits)
	semNorm := normalizeSemanticScores(semHits)

	combined := make(map[string]*Result)

	for _, h := range textHits {
		r := &Result{
			WorkID:      h.WorkID,
			Title:       h.Title,
			Description: h.Description,
			Score:       textNorm[h.WorkID] * textWeight,
		}
		combined[h.WorkID] = r
	}

	for _, h := range semHits {
		if existing, ok := combined[h.WorkID]; ok {
			existing.Score += semNorm[h.WorkID] * semanticWeight
		} else {
			combined[h.WorkID] = &Result{
				WorkID: h.WorkID,
				Title:  h.Title,
				Score:  semNorm[h.WorkID] * semanticWeight,
			}
		}
	}

	results := make([]Result, 0, len(combined))
	for _, r := range combined {
		results = append(results, *r)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func normalizeTextScores(hits []opensearch.SearchHit) map[string]float64 {
	norm := make(map[string]float64, len(hits))
	if len(hits) == 0 {
		return norm
	}

	minScore, maxScore := hits[0].Score, hits[0].Score
	for _, h := range hits[1:] {
		if h.Score < minScore {
			minScore = h.Score
		}
		if h.Score > maxScore {
			maxScore = h.Score
		}
	}

	rng := maxScore - minScore
	for _, h := range hits {
		if rng == 0 {
			norm[h.WorkID] = 1.0
		} else {
			norm[h.WorkID] = (h.Score - minScore) / rng
		}
	}
	return norm
}

func normalizeSemanticScores(hits []semantic.SearchResult) map[string]float64 {
	norm := make(map[string]float64, len(hits))
	if len(hits) == 0 {
		return norm
	}

	minScore, maxScore := hits[0].Score, hits[0].Score
	for _, h := range hits[1:] {
		if h.Score < minScore {
			minScore = h.Score
		}
		if h.Score > maxScore {
			maxScore = h.Score
		}
	}

	rng := maxScore - minScore
	for _, h := range hits {
		if rng == 0 {
			norm[h.WorkID] = 1.0
		} else {
			norm[h.WorkID] = (h.Score - minScore) / rng
		}
	}
	return norm
}
