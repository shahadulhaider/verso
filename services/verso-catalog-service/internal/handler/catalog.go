package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	versoerrors "github.com/shahadulhaider/verso/libs/go/errors"

	"github.com/shahadulhaider/verso/services/verso-catalog-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-catalog-service/internal/service"
)

type CatalogHandler struct {
	svc  *service.CatalogService
	repo *repository.Repo
}

func New(svc *service.CatalogService, repo *repository.Repo) *CatalogHandler {
	return &CatalogHandler{svc: svc, repo: repo}
}

type createWorkRequest struct {
	Title                   string  `json:"title"`
	Description             *string `json:"description,omitempty"`
	OriginalLanguage        *string `json:"originalLanguage,omitempty"`
	OriginalPublicationYear *int    `json:"originalPublicationYear,omitempty"`
}

type createEditionRequest struct {
	Title           *string `json:"title,omitempty"`
	Language        *string `json:"language,omitempty"`
	Publisher       *string `json:"publisher,omitempty"`
	PublicationDate *string `json:"publicationDate,omitempty"`
	PageCount       *int    `json:"pageCount,omitempty"`
}

type listResponse struct {
	Items      any    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type workWithEditions struct {
	repository.Work
	Editions []repository.Edition `json:"editions"`
}

type editionWithFormats struct {
	repository.Edition
	Formats []repository.Format `json:"formats"`
}

func (h *CatalogHandler) CreateWork(w http.ResponseWriter, r *http.Request) {
	var req createWorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	work, err := h.svc.CreateWork(r.Context(), service.CreateWorkRequest{
		Title:                   req.Title,
		Description:             req.Description,
		OriginalLanguage:        req.OriginalLanguage,
		OriginalPublicationYear: req.OriginalPublicationYear,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, work)
}

func (h *CatalogHandler) ListWorks(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	works, nextCursor, err := h.svc.ListWorks(r.Context(), cursor, limit)
	if err != nil {
		versoerrors.InternalError("failed to list works").WriteJSON(w)
		return
	}

	if works == nil {
		works = []repository.Work{}
	}

	writeJSON(w, http.StatusOK, listResponse{
		Items:      works,
		NextCursor: nextCursor,
	})
}

func (h *CatalogHandler) GetWork(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	work, editions, err := h.svc.GetWork(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if editions == nil {
		editions = []repository.Edition{}
	}

	writeJSON(w, http.StatusOK, workWithEditions{Work: *work, Editions: editions})
}

func (h *CatalogHandler) CreateEdition(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "id")

	var req createEditionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	var pubDate *time.Time
	if req.PublicationDate != nil {
		t, err := time.Parse("2006-01-02", *req.PublicationDate)
		if err != nil {
			versoerrors.BadRequest("publicationDate must be in YYYY-MM-DD format").WriteJSON(w)
			return
		}
		pubDate = &t
	}

	edition, err := h.svc.CreateEdition(r.Context(), workID, service.CreateEditionRequest{
		Title:           req.Title,
		Language:        req.Language,
		Publisher:       req.Publisher,
		PublicationDate: pubDate,
		PageCount:       req.PageCount,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, edition)
}

func (h *CatalogHandler) GetEdition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	edition, formats, err := h.svc.GetEdition(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if formats == nil {
		formats = []repository.Format{}
	}

	writeJSON(w, http.StatusOK, editionWithFormats{Edition: *edition, Formats: formats})
}

func (h *CatalogHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *CatalogHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Pool().Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *CatalogHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		versoerrors.NotFound("resource not found").WriteJSON(w)
	default:
		msg := err.Error()
		if msg == "title is required" {
			versoerrors.BadRequest(msg).WriteJSON(w)
			return
		}
		versoerrors.InternalError("an unexpected error occurred").WriteJSON(w)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
