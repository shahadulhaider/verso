// Package errors provides RFC 9457 Problem Details for HTTP APIs.
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ProblemDetail represents an RFC 9457 problem detail object.
type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// New creates a ProblemDetail with Type set to "about:blank".
func New(status int, title, detail string) *ProblemDetail {
	return &ProblemDetail{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

// Error implements the error interface.
func (p *ProblemDetail) Error() string {
	return fmt.Sprintf("%s: %s", p.Title, p.Detail)
}

// WriteJSON writes the problem detail as application/problem+json to the response writer.
func (p *ProblemDetail) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

// Factory functions for common HTTP error responses.

func NotFound(detail string) *ProblemDetail {
	return New(http.StatusNotFound, "Not Found", detail)
}

func BadRequest(detail string) *ProblemDetail {
	return New(http.StatusBadRequest, "Bad Request", detail)
}

func Unauthorized(detail string) *ProblemDetail {
	return New(http.StatusUnauthorized, "Unauthorized", detail)
}

func Forbidden(detail string) *ProblemDetail {
	return New(http.StatusForbidden, "Forbidden", detail)
}

func InternalError(detail string) *ProblemDetail {
	return New(http.StatusInternalServerError, "Internal Server Error", detail)
}

func Conflict(detail string) *ProblemDetail {
	return New(http.StatusConflict, "Conflict", detail)
}
