package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sony/gobreaker/v2"

	versojwt "github.com/shahadulhaider/verso/libs/go/jwt"
	versoerrors "github.com/shahadulhaider/verso/libs/go/errors"

	"github.com/shahadulhaider/verso/services/verso-library-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-library-service/internal/service"
)

type LibraryHandler struct {
	svc  *service.LibraryService
	repo *repository.Repo
	cb   *gobreaker.CircuitBreaker[any]
}

func New(svc *service.LibraryService, repo *repository.Repo, cb *gobreaker.CircuitBreaker[any]) *LibraryHandler {
	return &LibraryHandler{svc: svc, repo: repo, cb: cb}
}

func (h *LibraryHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *LibraryHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Pool().Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *LibraryHandler) ListShelves(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.ListShelves(r.Context(), claims.UserID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	shelves := result.([]*repository.Shelf)
	resp := make([]shelfResponse, len(shelves))
	for i, s := range shelves {
		resp[i] = toShelfResponse(s)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *LibraryHandler) CreateShelf(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	var req service.CreateShelfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.CreateShelf(r.Context(), claims.UserID, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	shelf := result.(*repository.Shelf)
	writeJSON(w, http.StatusCreated, toShelfResponse(shelf))
}

func (h *LibraryHandler) ListShelfItems(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	shelfID := chi.URLParam(r, "shelfId")
	cursor := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.ListShelfItems(r.Context(), claims.UserID, shelfID, cursor, limit)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	items := result.([]*repository.ShelfItem)
	resp := make([]shelfItemResponse, len(items))
	for i, item := range items {
		resp[i] = toShelfItemResponse(item)
	}

	out := map[string]any{"items": resp}
	if len(items) > 0 {
		out["nextCursor"] = items[len(items)-1].DateAdded.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *LibraryHandler) AddShelfItem(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	shelfID := chi.URLParam(r, "shelfId")
	var req service.AddShelfItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.AddItemToShelf(r.Context(), claims.UserID, shelfID, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	item := result.(*repository.ShelfItem)
	writeJSON(w, http.StatusCreated, toShelfItemResponse(item))
}

func (h *LibraryHandler) RemoveShelfItem(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	shelfID := chi.URLParam(r, "shelfId")
	itemID := chi.URLParam(r, "itemId")

	_, err := h.cb.Execute(func() (any, error) {
		return nil, h.svc.RemoveItemFromShelf(r.Context(), claims.UserID, shelfID, itemID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *LibraryHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		versoerrors.NotFound("resource not found").WriteJSON(w)
	case errors.Is(err, repository.ErrDuplicateItem):
		versoerrors.Conflict("item already on shelf").WriteJSON(w)
	case errors.Is(err, repository.ErrDuplicateShelf):
		versoerrors.Conflict("shelf with this slug already exists").WriteJSON(w)
	case errors.Is(err, gobreaker.ErrOpenState):
		versoerrors.New(http.StatusServiceUnavailable, "Service Unavailable", "circuit breaker open").WriteJSON(w)
	default:
		msg := err.Error()
		if msg == "name is required" || msg == "workId is required" ||
			msg == "formatId is required" || msg == "formatId and workId are required" ||
			msg == "progressAfter must be >= progressBefore" ||
			msg == "progress must be between 0 and 100" ||
			msg == "invalid status" || msg == "progressPercent must be between 0 and 100" {
			versoerrors.BadRequest(msg).WriteJSON(w)
			return
		}
		versoerrors.InternalError("an unexpected error occurred").WriteJSON(w)
	}
}

type shelfResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ShelfType    string `json:"shelfType"`
	IsSystem     bool   `json:"isSystem"`
	IsPrivate    bool   `json:"isPrivate"`
	DisplayOrder int    `json:"displayOrder"`
	ItemCount    int    `json:"itemCount"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

func toShelfResponse(s *repository.Shelf) shelfResponse {
	return shelfResponse{
		ID:           strings.TrimSpace(s.ID),
		Name:         s.Name,
		Slug:         s.Slug,
		ShelfType:    s.ShelfType,
		IsSystem:     s.IsSystem,
		IsPrivate:    s.IsPrivate,
		DisplayOrder: s.DisplayOrder,
		ItemCount:    s.ItemCount,
		CreatedAt:    s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

type shelfItemResponse struct {
	ID        string  `json:"id"`
	WorkID    string  `json:"workId"`
	EditionID *string `json:"editionId,omitempty"`
	DateAdded string  `json:"dateAdded"`
}

func toShelfItemResponse(i *repository.ShelfItem) shelfItemResponse {
	return shelfItemResponse{
		ID:        strings.TrimSpace(i.ID),
		WorkID:    strings.TrimSpace(i.WorkID),
		EditionID: trimOptional(i.EditionID),
		DateAdded: i.DateAdded.Format("2006-01-02T15:04:05Z"),
	}
}

func trimOptional(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
