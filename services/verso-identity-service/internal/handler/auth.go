// Package handler provides HTTP handlers for the identity service.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	versoerrors "github.com/shahadulhaider/verso/libs/go/errors"

	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/auth"
	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/service"
)

// AuthHandler holds HTTP handlers for authentication endpoints.
type AuthHandler struct {
	svc    *service.AuthService
	tokens *auth.TokenManager
	repo   *repository.Repo
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(svc *service.AuthService, tokens *auth.TokenManager, repo *repository.Repo) *AuthHandler {
	return &AuthHandler{svc: svc, tokens: tokens, repo: repo}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// Register handles POST /v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	resp, err := h.svc.Register(r.Context(), service.RegisterRequest{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login handles POST /v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	resp, err := h.svc.Login(r.Context(), service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Refresh handles POST /v1/auth/token/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		versoerrors.BadRequest("invalid request body").WriteJSON(w)
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// JWKS handles GET /.well-known/jwks.json.
func (h *AuthHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	h.tokens.WriteJWKS(w)
}

// Health handles GET /health.
func (h *AuthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready handles GET /ready.
func (h *AuthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Pool().Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *AuthHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrDuplicateEmail):
		versoerrors.Conflict("an account with this email already exists").WriteJSON(w)
	case errors.Is(err, repository.ErrNotFound):
		versoerrors.Unauthorized("invalid email or password").WriteJSON(w)
	default:
		msg := err.Error()
		if msg == "email and password required" || msg == "password must be at least 8 characters" || msg == "refresh token required" {
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
