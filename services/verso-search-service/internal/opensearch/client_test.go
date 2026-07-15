package opensearch_test

import (
	"encoding/json"
	"testing"

	"github.com/shahadulhaider/verso/services/verso-search-service/internal/opensearch"
)

func TestWorkDocumentSerialization(t *testing.T) {
	doc := &opensearch.WorkDocument{
		WorkID:                  "01J5M3K7H8ABCDEF12345678",
		Title:                   "The Great Gatsby",
		Description:             "A novel by F. Scott Fitzgerald",
		Genres:                  []string{"fiction", "classic"},
		ContributorNames:        []string{"F. Scott Fitzgerald"},
		OriginalLanguage:        "en",
		OriginalPublicationYear: 1925,
		CreatedAt:               "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded opensearch.WorkDocument
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.WorkID != doc.WorkID {
		t.Errorf("work_id: got %q, want %q", decoded.WorkID, doc.WorkID)
	}
	if decoded.Title != doc.Title {
		t.Errorf("title: got %q, want %q", decoded.Title, doc.Title)
	}
	if decoded.OriginalPublicationYear != 1925 {
		t.Errorf("year: got %d, want 1925", decoded.OriginalPublicationYear)
	}
	if len(decoded.Genres) != 2 {
		t.Errorf("genres: got %d, want 2", len(decoded.Genres))
	}
}

func TestWorkDocumentOmitsEmpty(t *testing.T) {
	doc := &opensearch.WorkDocument{
		WorkID:    "01J5M3K7H8ABCDEF12345678",
		Title:     "Minimal Work",
		CreatedAt: "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if _, ok := raw["description"]; ok {
		t.Error("expected description to be omitted")
	}
	if _, ok := raw["genres"]; ok {
		t.Error("expected genres to be omitted")
	}
	if _, ok := raw["contributor_names"]; ok {
		t.Error("expected contributor_names to be omitted")
	}
}

func TestSearchHitSerialization(t *testing.T) {
	hit := opensearch.SearchHit{
		WorkID:      "01J5M3K7H8ABCDEF12345678",
		Title:       "Test Book",
		Description: "A test description",
		Score:       1.5,
	}

	data, err := json.Marshal(hit)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded opensearch.SearchHit
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.WorkID != hit.WorkID {
		t.Errorf("workId: got %q, want %q", decoded.WorkID, hit.WorkID)
	}
	if decoded.Score != 1.5 {
		t.Errorf("score: got %f, want 1.5", decoded.Score)
	}
}
