package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	versojwt "github.com/shahadulhaider/verso/libs/go/jwt"
	versoerrors "github.com/shahadulhaider/verso/libs/go/errors"

	"github.com/shahadulhaider/verso/services/verso-library-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-library-service/internal/service"
)

func (h *LibraryHandler) LogSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	var req service.LogSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.LogReadingSession(r.Context(), claims.UserID, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	session := result.(*repository.ReadingSession)
	writeJSON(w, http.StatusCreated, toSessionResponse(session))
}

func (h *LibraryHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	workID := chi.URLParam(r, "workId")
	if workID == "" {
		versoerrors.BadRequest("workId is required").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.GetReadingProgress(r.Context(), claims.UserID, workID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	progress := result.(*repository.ReadingProgress)
	writeJSON(w, http.StatusOK, toProgressResponse(progress))
}

func (h *LibraryHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	workID := chi.URLParam(r, "workId")
	if workID == "" {
		versoerrors.BadRequest("workId is required").WriteJSON(w)
		return
	}

	var req service.UpdateProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.UpdateReadingProgress(r.Context(), claims.UserID, workID, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	progress := result.(*repository.ReadingProgress)
	writeJSON(w, http.StatusOK, toProgressResponse(progress))
}

type sessionResponse struct {
	ID             string  `json:"id"`
	FormatID       string  `json:"formatId"`
	WorkID         string  `json:"workId"`
	StartedAt      string  `json:"startedAt"`
	ProgressBefore float64 `json:"progressBefore"`
	ProgressAfter  float64 `json:"progressAfter"`
	PagesRead      *int    `json:"pagesRead,omitempty"`
	DeviceType     *string `json:"deviceType,omitempty"`
	CreatedAt      string  `json:"createdAt"`
}

func toSessionResponse(s *repository.ReadingSession) sessionResponse {
	return sessionResponse{
		ID:             strings.TrimSpace(s.ID),
		FormatID:       strings.TrimSpace(s.FormatID),
		WorkID:         strings.TrimSpace(s.WorkID),
		StartedAt:      s.StartedAt.Format("2006-01-02T15:04:05Z"),
		ProgressBefore: s.ProgressBefore,
		ProgressAfter:  s.ProgressAfter,
		PagesRead:      s.PagesRead,
		DeviceType:     s.DeviceType,
		CreatedAt:      s.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

type progressResponse struct {
	WorkID          string   `json:"workId"`
	CurrentFormatID string   `json:"currentFormatId"`
	ProgressPercent float64  `json:"progressPercent"`
	CurrentPage     *int     `json:"currentPage,omitempty"`
	Status          string   `json:"status"`
	StartedAt       *string  `json:"startedAt,omitempty"`
	CompletedAt     *string  `json:"completedAt,omitempty"`
	ReadCount       int      `json:"readCount"`
	UpdatedAt       string   `json:"updatedAt"`
}

func toProgressResponse(p *repository.ReadingProgress) progressResponse {
	resp := progressResponse{
		WorkID:          strings.TrimSpace(p.WorkID),
		CurrentFormatID: strings.TrimSpace(p.CurrentFormatID),
		ProgressPercent: p.ProgressPercent,
		CurrentPage:     p.CurrentPage,
		Status:          p.Status,
		ReadCount:       p.ReadCount,
		UpdatedAt:       p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if p.StartedAt != nil {
		s := p.StartedAt.Format("2006-01-02T15:04:05Z")
		resp.StartedAt = &s
	}
	if p.CompletedAt != nil {
		s := p.CompletedAt.Format("2006-01-02T15:04:05Z")
		resp.CompletedAt = &s
	}
	return resp
}
