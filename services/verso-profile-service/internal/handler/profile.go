package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sony/gobreaker/v2"

	versojwt "github.com/shahadulhaider/verso/libs/go/jwt"
	versoerrors "github.com/shahadulhaider/verso/libs/go/errors"

	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/service"
)

type ProfileHandler struct {
	svc  *service.ProfileService
	repo *repository.Repo
	cb   *gobreaker.CircuitBreaker[any]
}

func New(svc *service.ProfileService, repo *repository.Repo, cb *gobreaker.CircuitBreaker[any]) *ProfileHandler {
	return &ProfileHandler{svc: svc, repo: repo, cb: cb}
}

func (h *ProfileHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ProfileHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Pool().Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		versoerrors.BadRequest("userId is required").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.GetProfile(r.Context(), userID)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	profile := result.(*repository.Profile)
	writeJSON(w, http.StatusOK, toProfileResponse(profile))
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := versojwt.ClaimsFromContext(r.Context())
	if !ok {
		versoerrors.Unauthorized("missing claims").WriteJSON(w)
		return
	}

	var req service.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	result, err := h.cb.Execute(func() (any, error) {
		return h.svc.UpdateProfile(r.Context(), claims.UserID, req)
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	profile := result.(*repository.Profile)
	writeJSON(w, http.StatusOK, toProfileResponse(profile))
}

func (h *ProfileHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		versoerrors.NotFound("profile not found").WriteJSON(w)
	case errors.Is(err, repository.ErrDuplicateUsername):
		versoerrors.Conflict("username already taken").WriteJSON(w)
	case errors.Is(err, gobreaker.ErrOpenState):
		versoerrors.New(http.StatusServiceUnavailable, "Service Unavailable", "circuit breaker open").WriteJSON(w)
	default:
		msg := err.Error()
		if msg == "display_name cannot be empty" || msg == "bio exceeds 2000 characters" ||
			msg == "invalid privacy_level" || msg == "no fields to update" {
			versoerrors.BadRequest(msg).WriteJSON(w)
			return
		}
		versoerrors.InternalError("an unexpected error occurred").WriteJSON(w)
	}
}

type profileResponse struct {
	ID                string  `json:"id"`
	Username          string  `json:"username"`
	DisplayName       string  `json:"displayName"`
	Bio               *string `json:"bio,omitempty"`
	AvatarURL         *string `json:"avatarUrl,omitempty"`
	Location          *string `json:"location,omitempty"`
	WebsiteURL        *string `json:"websiteUrl,omitempty"`
	IsAuthor          bool    `json:"isAuthor"`
	IsPublisher       bool    `json:"isPublisher"`
	IsVerifiedCritic  bool    `json:"isVerifiedCritic"`
	PrivacyLevel      string  `json:"privacyLevel"`
	ReadingGoalAnnual *int    `json:"readingGoalAnnual,omitempty"`
	PreferredLanguage string  `json:"preferredLanguage"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

func toProfileResponse(p *repository.Profile) profileResponse {
	return profileResponse{
		ID:                strings.TrimSpace(p.ID),
		Username:          p.Username,
		DisplayName:       p.DisplayName,
		Bio:               p.Bio,
		AvatarURL:         p.AvatarURL,
		Location:          p.Location,
		WebsiteURL:        p.WebsiteURL,
		IsAuthor:          p.IsAuthor,
		IsPublisher:       p.IsPublisher,
		IsVerifiedCritic:  p.IsVerifiedCritic,
		PrivacyLevel:      p.PrivacyLevel,
		ReadingGoalAnnual: p.ReadingGoalAnnual,
		PreferredLanguage: p.PreferredLanguage,
		CreatedAt:         p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
