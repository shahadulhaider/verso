// Package service provides business logic for the identity domain.
package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"

	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/auth"
	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/repository"
)

const (
	eventTypeUserRegistered = "verso.identity.user-registered.v1"
	producer                = "verso-identity-service"
	refreshTokenExpiry      = 30 * 24 * time.Hour // 30 days
)

// AuthService encapsulates identity business logic.
type AuthService struct {
	repo   *repository.Repo
	tokens *auth.TokenManager
	log    *slog.Logger
}

// NewAuthService creates an AuthService.
func NewAuthService(repo *repository.Repo, tokens *auth.TokenManager, log *slog.Logger) *AuthService {
	return &AuthService{repo: repo, tokens: tokens, log: log}
}

// RegisterRequest holds the data needed for user registration.
type RegisterRequest struct {
	Email       string
	Password    string
	DisplayName string
}

// AuthResponse is returned by Register and Login.
type AuthResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         UserResponse `json:"user"`
}

// UserResponse is the public user representation.
type UserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

// RefreshResponse is returned by RefreshToken.
type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
}

// Register creates a new account, publishes an outbox event, and returns tokens.
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password required")
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	userID := ulid.MustNew(ulid.Timestamp(now), rand.Reader).String()

	acct := &repository.Account{
		ID:            userID,
		Email:         email,
		EmailVerified: false,
		PasswordHash:  &hash,
		Status:        "active",
		Roles:         []string{"reader"},
		DisplayName:   req.DisplayName,
		MFAEnabled:    false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateAccount(ctx, tx, acct); err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, err
		}
		return nil, fmt.Errorf("create account: %w", err)
	}

	payload, _ := json.Marshal(map[string]string{
		"userId":       userID,
		"email":        email,
		"displayName":  req.DisplayName,
		"registeredAt": now.Format(time.RFC3339),
	})
	env := envelope.New(ctx, eventTypeUserRegistered, producer, userID, payload)
	if err := outbox.InsertEvent(ctx, tx, "account", userID, env); err != nil {
		return nil, fmt.Errorf("insert outbox event: %w", err)
	}

	refreshToken, rt, err := s.generateRefreshToken(userID, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.StoreRefreshToken(ctx, tx, rt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	accessToken, err := s.tokens.SignAccessToken(userID, email, acct.Roles)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	s.log.InfoContext(ctx, "user registered", slog.String("user_id", userID), slog.String("email", email))

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         UserResponse{ID: userID, Email: email, DisplayName: req.DisplayName},
	}, nil
}

// LoginRequest holds login credentials.
type LoginRequest struct {
	Email    string
	Password string
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password required")
	}

	acct, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get account: %w", err)
	}
	if acct.PasswordHash == nil {
		return nil, fmt.Errorf("account has no password (social-only)")
	}

	match, err := auth.VerifyPassword(req.Password, *acct.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !match {
		return nil, repository.ErrNotFound // deliberately vague for security
	}

	now := time.Now().UTC()
	refreshToken, rt, err := s.generateRefreshToken(acct.ID, now)
	if err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.StoreRefreshToken(ctx, tx, rt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	accessToken, err := s.tokens.SignAccessToken(acct.ID, acct.Email, acct.Roles)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	s.log.InfoContext(ctx, "user logged in", slog.String("user_id", acct.ID))

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         UserResponse{ID: acct.ID, Email: acct.Email, DisplayName: acct.DisplayName},
	}, nil
}

// Refresh validates a refresh token and returns a new access token.
func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*RefreshResponse, error) {
	if rawToken == "" {
		return nil, fmt.Errorf("refresh token required")
	}

	tokenHash := repository.HashRefreshToken(rawToken)
	rt, err := s.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	acct, err := s.repo.GetAccountByID(ctx, rt.AccountID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	accessToken, err := s.tokens.SignAccessToken(acct.ID, acct.Email, acct.Roles)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	return &RefreshResponse{AccessToken: accessToken}, nil
}

func (s *AuthService) generateRefreshToken(accountID string, now time.Time) (string, *repository.RefreshToken, error) {
	rawToken := ulid.MustNew(ulid.Timestamp(now), rand.Reader).String()
	tokenHash := repository.HashRefreshToken(rawToken)
	rtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader).String()

	rt := &repository.RefreshToken{
		ID:        rtID,
		AccountID: accountID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(refreshTokenExpiry),
		CreatedAt: now,
	}
	return rawToken, rt, nil
}
