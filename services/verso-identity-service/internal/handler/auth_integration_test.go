package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/auth"
	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/service"
)

type testEnv struct {
	handler *handler.AuthHandler
	router  *chi.Mux
	pool    *pgxpool.Pool
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("verso_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	migrations := []string{
		`CREATE SCHEMA IF NOT EXISTS identity`,
		`SET search_path TO identity`,
		`CREATE TABLE IF NOT EXISTS account (
			id              CHAR(26)        PRIMARY KEY,
			email           VARCHAR(320)    UNIQUE NOT NULL,
			email_verified  BOOLEAN         NOT NULL DEFAULT FALSE,
			password_hash   VARCHAR(255)    NULL,
			status          VARCHAR(20)     NOT NULL DEFAULT 'active',
			roles           VARCHAR[]       NOT NULL DEFAULT '{}',
			display_name    VARCHAR(255)    NOT NULL DEFAULT '',
			mfa_enabled     BOOLEAN         NOT NULL DEFAULT FALSE,
			erased_at       TIMESTAMPTZ     NULL,
			created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS outbox_events (
			event_id        CHAR(26)        PRIMARY KEY,
			aggregate_type  TEXT            NOT NULL,
			aggregate_id    TEXT            NOT NULL,
			event_type      TEXT            NOT NULL,
			payload         JSONB           NOT NULL,
			created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
			delivered       BOOLEAN         NOT NULL DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id              CHAR(26)        PRIMARY KEY,
			account_id      CHAR(26)        NOT NULL REFERENCES account(id),
			token_hash      VARCHAR(64)     NOT NULL,
			expires_at      TIMESTAMPTZ     NOT NULL,
			created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
		)`,
	}
	for _, ddl := range migrations {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	// Set search_path for the pool
	pool.Close()
	pool, err = pgxpool.New(ctx, connStr+"&search_path=identity")
	if err != nil {
		t.Fatalf("reconnect with search_path: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	keyPath := filepath.Join(t.TempDir(), "test-jwt-key.pem")
	tokens, err := auth.NewTokenManager(keyPath, time.Hour)
	if err != nil {
		t.Fatalf("token manager: %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := repository.New(pool)
	svc := service.NewAuthService(repo, tokens, log)
	h := handler.NewAuthHandler(svc, tokens, repo)

	r := chi.NewRouter()
	r.Post("/v1/auth/register", h.Register)
	r.Post("/v1/auth/login", h.Login)
	r.Post("/v1/auth/token/refresh", h.Refresh)
	r.Get("/.well-known/jwks.json", h.JWKS)
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)

	return &testEnv{handler: h, router: r, pool: pool}
}

func postJSON(router *chi.Mux, path string, body interface{}) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func getJSON(router *chi.Mux, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRegisterCreatesAccountAndOutboxEvent(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	body := map[string]string{
		"email":       "alice@example.com",
		"password":    "strongpassword123",
		"displayName": "Alice",
	}
	w := postJSON(env.router, "/v1/auth/register", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID          string `json:"id"`
			Email       string `json:"email"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("accessToken should not be empty")
	}
	if resp.RefreshToken == "" {
		t.Error("refreshToken should not be empty")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("email: got %q", resp.User.Email)
	}
	if resp.User.DisplayName != "Alice" {
		t.Errorf("displayName: got %q", resp.User.DisplayName)
	}
	if len(resp.User.ID) != 26 {
		t.Errorf("id should be ULID (26 chars): got %q", resp.User.ID)
	}

	// Verify account exists in DB
	var accountID string
	err := env.pool.QueryRow(ctx, "SELECT id FROM account WHERE email = $1", "alice@example.com").Scan(&accountID)
	if err != nil {
		t.Fatalf("account not in DB: %v", err)
	}

	// Verify outbox event written
	var eventType string
	err = env.pool.QueryRow(ctx, "SELECT event_type FROM outbox_events WHERE aggregate_id = $1", accountID).Scan(&eventType)
	if err != nil {
		t.Fatalf("outbox event not in DB: %v", err)
	}
	if eventType != "verso.identity.user-registered.v1" {
		t.Errorf("event_type: got %q", eventType)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]string{
		"email":    "dup@example.com",
		"password": "strongpassword123",
	}
	w1 := postJSON(env.router, "/v1/auth/register", body)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first register: got %d", w1.Code)
	}

	w2 := postJSON(env.router, "/v1/auth/register", body)
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate register: got %d, want 409, body: %s", w2.Code, w2.Body.String())
	}
}

func TestLoginReturnsValidJWT(t *testing.T) {
	env := setupTestEnv(t)

	regBody := map[string]string{
		"email":    "login@example.com",
		"password": "strongpassword123",
	}
	postJSON(env.router, "/v1/auth/register", regBody)

	loginBody := map[string]string{
		"email":    "login@example.com",
		"password": "strongpassword123",
	}
	w := postJSON(env.router, "/v1/auth/login", loginBody)
	if w.Code != http.StatusOK {
		t.Fatalf("login: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AccessToken == "" {
		t.Error("accessToken should not be empty")
	}
	if resp.User.Email != "login@example.com" {
		t.Errorf("email: got %q", resp.User.Email)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	env := setupTestEnv(t)

	regBody := map[string]string{
		"email":    "wrong@example.com",
		"password": "strongpassword123",
	}
	postJSON(env.router, "/v1/auth/register", regBody)

	loginBody := map[string]string{
		"email":    "wrong@example.com",
		"password": "incorrectpassword",
	}
	w := postJSON(env.router, "/v1/auth/login", loginBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: got %d, want 401", w.Code)
	}
}

func TestRefreshToken(t *testing.T) {
	env := setupTestEnv(t)

	regBody := map[string]string{
		"email":    "refresh@example.com",
		"password": "strongpassword123",
	}
	w := postJSON(env.router, "/v1/auth/register", regBody)

	var regResp struct {
		RefreshToken string `json:"refreshToken"`
	}
	json.Unmarshal(w.Body.Bytes(), &regResp)

	refreshBody := map[string]string{
		"refreshToken": regResp.RefreshToken,
	}
	w2 := postJSON(env.router, "/v1/auth/token/refresh", refreshBody)
	if w2.Code != http.StatusOK {
		t.Fatalf("refresh: got %d, body: %s", w2.Code, w2.Body.String())
	}

	var refreshResp struct {
		AccessToken string `json:"accessToken"`
	}
	json.Unmarshal(w2.Body.Bytes(), &refreshResp)
	if refreshResp.AccessToken == "" {
		t.Error("new access token should not be empty")
	}
}

func TestHealthAndReady(t *testing.T) {
	env := setupTestEnv(t)

	w := getJSON(env.router, "/health")
	if w.Code != http.StatusOK {
		t.Fatalf("health: got %d", w.Code)
	}

	w = getJSON(env.router, "/ready")
	if w.Code != http.StatusOK {
		t.Fatalf("ready: got %d", w.Code)
	}
}

func TestJWKSEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	w := getJSON(env.router, "/.well-known/jwks.json")
	if w.Code != http.StatusOK {
		t.Fatalf("jwks: got %d", w.Code)
	}

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	json.Unmarshal(w.Body.Bytes(), &jwks)
	if len(jwks.Keys) == 0 {
		t.Fatal("JWKS should contain at least one key")
	}
}
