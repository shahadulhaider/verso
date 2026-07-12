package errors_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	verserr "github.com/shahadulhaider/verso/libs/go/errors"
)

func TestNew(t *testing.T) {
	p := verserr.New(404, "Not Found", "resource does not exist")
	if p.Type != "about:blank" {
		t.Errorf("expected type about:blank, got %q", p.Type)
	}
	if p.Status != 404 {
		t.Errorf("expected status 404, got %d", p.Status)
	}
}

func TestError(t *testing.T) {
	p := verserr.New(400, "Bad Request", "invalid input")
	want := "Bad Request: invalid input"
	if p.Error() != want {
		t.Errorf("Error() = %q, want %q", p.Error(), want)
	}
}

func TestWriteJSON(t *testing.T) {
	p := verserr.NotFound("book not found")

	rec := httptest.NewRecorder()
	p.WriteJSON(rec)

	// Check Content-Type
	ct := rec.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}

	// Check status code
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Check body
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["type"] != "about:blank" {
		t.Errorf("type = %v, want about:blank", body["type"])
	}
	if body["title"] != "Not Found" {
		t.Errorf("title = %v, want Not Found", body["title"])
	}
	if body["detail"] != "book not found" {
		t.Errorf("detail = %v, want book not found", body["detail"])
	}
	if int(body["status"].(float64)) != 404 {
		t.Errorf("status in body = %v, want 404", body["status"])
	}
}

func TestFactoryFunctions(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(string) *verserr.ProblemDetail
		status int
		title  string
	}{
		{"NotFound", verserr.NotFound, 404, "Not Found"},
		{"BadRequest", verserr.BadRequest, 400, "Bad Request"},
		{"Unauthorized", verserr.Unauthorized, 401, "Unauthorized"},
		{"Forbidden", verserr.Forbidden, 403, "Forbidden"},
		{"InternalError", verserr.InternalError, 500, "Internal Server Error"},
		{"Conflict", verserr.Conflict, 409, "Conflict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.fn("test detail")
			if p.Status != tt.status {
				t.Errorf("status = %d, want %d", p.Status, tt.status)
			}
			if p.Title != tt.title {
				t.Errorf("title = %q, want %q", p.Title, tt.title)
			}
		})
	}
}

// Verify ProblemDetail satisfies the error interface.
func TestErrorInterface(t *testing.T) {
	var err error = verserr.New(500, "Internal Server Error", "something broke")
	if err == nil {
		t.Fatal("should not be nil")
	}
}
