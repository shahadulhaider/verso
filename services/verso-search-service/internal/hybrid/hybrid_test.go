package hybrid

import (
	"testing"

	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
	"github.com/shahadulhaider/verso/services/verso-search-service/internal/semantic"
)

func TestNormalizeTextScores_SingleItem(t *testing.T) {
	hits := []opensearch.SearchHit{
		{WorkID: "w1", Score: 5.0},
	}
	norm := normalizeTextScores(hits)
	if norm["w1"] != 1.0 {
		t.Fatalf("single item should normalize to 1.0, got %f", norm["w1"])
	}
}

func TestNormalizeTextScores_MultipleItems(t *testing.T) {
	hits := []opensearch.SearchHit{
		{WorkID: "w1", Score: 10.0},
		{WorkID: "w2", Score: 5.0},
		{WorkID: "w3", Score: 0.0},
	}
	norm := normalizeTextScores(hits)

	if norm["w1"] != 1.0 {
		t.Fatalf("max score should normalize to 1.0, got %f", norm["w1"])
	}
	if norm["w3"] != 0.0 {
		t.Fatalf("min score should normalize to 0.0, got %f", norm["w3"])
	}
	if norm["w2"] != 0.5 {
		t.Fatalf("mid score should normalize to 0.5, got %f", norm["w2"])
	}
}

func TestNormalizeSemanticScores_Empty(t *testing.T) {
	norm := normalizeSemanticScores(nil)
	if len(norm) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(norm))
	}
}

func TestMerge_DeduplicatesAndWeights(t *testing.T) {
	textHits := []opensearch.SearchHit{
		{WorkID: "w1", Title: "Book A", Score: 10.0},
		{WorkID: "w2", Title: "Book B", Score: 5.0},
	}
	semHits := []semantic.SearchResult{
		{WorkID: "w1", Title: "Book A", Score: 0.9},
		{WorkID: "w3", Title: "Book C", Score: 0.8},
	}

	results := merge(textHits, semHits, 10)

	if len(results) != 3 {
		t.Fatalf("expected 3 merged results, got %d", len(results))
	}

	// w1 should appear once with combined score
	var w1Found bool
	for _, r := range results {
		if r.WorkID == "w1" {
			w1Found = true
			// text: 1.0 * 0.6 = 0.6, semantic: 1.0 * 0.4 = 0.4 → total = 1.0
			if r.Score < 0.99 {
				t.Fatalf("w1 combined score should be ~1.0, got %f", r.Score)
			}
		}
	}
	if !w1Found {
		t.Fatal("w1 should appear in merged results")
	}
}

func TestMerge_LimitResults(t *testing.T) {
	textHits := []opensearch.SearchHit{
		{WorkID: "w1", Score: 10.0},
		{WorkID: "w2", Score: 8.0},
		{WorkID: "w3", Score: 6.0},
	}
	semHits := []semantic.SearchResult{
		{WorkID: "w4", Score: 0.9},
		{WorkID: "w5", Score: 0.7},
	}

	results := merge(textHits, semHits, 3)
	if len(results) != 3 {
		t.Fatalf("expected 3 results (limit), got %d", len(results))
	}
}

func TestMerge_SortedByScore(t *testing.T) {
	textHits := []opensearch.SearchHit{
		{WorkID: "w1", Score: 2.0},
		{WorkID: "w2", Score: 10.0},
	}
	semHits := []semantic.SearchResult{}

	results := merge(textHits, semHits, 10)
	if len(results) < 2 {
		t.Fatal("expected at least 2 results")
	}
	if results[0].Score < results[1].Score {
		t.Fatalf("results should be sorted descending: %f < %f", results[0].Score, results[1].Score)
	}
}

func TestTextHitsToResults(t *testing.T) {
	hits := []opensearch.SearchHit{
		{WorkID: "w1", Title: "Test", Description: "Desc", Score: 1.5},
	}
	results := textHitsToResults(hits)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].WorkID != "w1" {
		t.Fatalf("expected workId=w1, got %s", results[0].WorkID)
	}
	if results[0].Title != "Test" {
		t.Fatalf("expected title=Test, got %s", results[0].Title)
	}
}
