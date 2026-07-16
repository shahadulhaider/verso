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

	"github.com/shahadulhaider/verso/services/verso-social-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-social-service/internal/service"
)

type SocialHandler struct {
	svc  *service.SocialService
	repo *repository.Repo
	cb   *gobreaker.CircuitBreaker[any]
}

func New(svc *service.SocialService, repo *repository.Repo, cb *gobreaker.CircuitBreaker[any]) *SocialHandler {
	return &SocialHandler{svc: svc, repo: repo, cb: cb}
}

func (h *SocialHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SocialHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Pool().Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type followRequest struct {
	UserID string `json:"userId"`
}

func (h *SocialHandler) Follow(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	var req followRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	if strings.TrimSpace(req.UserID) == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	_, err := h.cb.Execute(func() (any, error) {
		return nil, h.svc.Follow(r.Context(), claims.UserID, req.UserID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	_, err := h.cb.Execute(func() (any, error) {
		return nil, h.svc.Unfollow(r.Context(), claims.UserID, userID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) Block(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	var req followRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	if strings.TrimSpace(req.UserID) == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	_, err := h.cb.Execute(func() (any, error) {
		return nil, h.svc.Block(r.Context(), claims.UserID, req.UserID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	_, err := h.cb.Execute(func() (any, error) {
		return nil, h.svc.Unblock(r.Context(), claims.UserID, userID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) Followers(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"))

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.Followers(r.Context(), userID, cursor, limit)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *SocialHandler) Following(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"))

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.Following(r.Context(), userID, cursor, limit)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *SocialHandler) Counts(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.Counts(r.Context(), userID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *SocialHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrSelfFollow):
		versoerrors.BadRequest("cannot follow yourself").WriteJSON(w)
	case errors.Is(err, repository.ErrSelfBlock):
		versoerrors.BadRequest("cannot block yourself").WriteJSON(w)
	case errors.Is(err, repository.ErrBlockedByTarget):
		versoerrors.Forbidden("blocked by target user").WriteJSON(w)
	case errors.Is(err, repository.ErrAlreadyFollowing):
		versoerrors.Conflict("already following this user").WriteJSON(w)
	case errors.Is(err, repository.ErrAlreadyBlocked):
		versoerrors.Conflict("already blocked this user").WriteJSON(w)
	case errors.Is(err, gobreaker.ErrOpenState):
		versoerrors.New(http.StatusServiceUnavailable, "Service Unavailable", "circuit breaker open").WriteJSON(w)
	default:
		versoerrors.InternalError("an unexpected error occurred").WriteJSON(w)
	}
}

func parseLimit(s string) int {
	if s == "" {
		return 20
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || n > 100 {
		return 20
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
