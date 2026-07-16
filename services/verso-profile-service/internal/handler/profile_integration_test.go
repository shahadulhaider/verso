package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sony/gobreaker/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	versojwt "github.com/shahadulhaider/verso/libs/go/jwt"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-profile-service/internal/service"
)

type testEnv struct {
	handler *handler.ProfileHandler
	router  *chi.Mux
	pool    *pgxpool.Pool
	svc     *service.ProfileService
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
		`CREATE SCHEMA IF NOT EXISTS profile`,
		`SET search_path TO profile`,
		`CREATE TABLE IF NOT EXISTS user_profile (
			id                  CHAR(26)        PRIMARY KEY,
			username            VARCHAR(30)     UNIQUE NOT NULL,
			display_name        VARCHAR(100)    NOT NULL,
			bio                 TEXT            NULL,
			avatar_url          VARCHAR(512)    NULL,
			location            VARCHAR(100)    NULL,
			website_url         VARCHAR(512)    NULL,
			is_author           BOOLEAN         NOT NULL DEFAULT FALSE,
			is_publisher        BOOLEAN         NOT NULL DEFAULT FALSE,
			is_verified_critic  BOOLEAN         NOT NULL DEFAULT FALSE,
			privacy_level       VARCHAR(20)     NOT NULL DEFAULT 'public'
										CHECK (privacy_level IN ('public', 'friends_only', 'private')),
			reading_goal_annual INT             NULL,
			preferred_language  VARCHAR(5)      NOT NULL DEFAULT 'en',
			created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
			updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
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
	}
	for _, ddl := range migrations {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	pool.Close()
	pool, err = pgxpool.New(ctx, connStr+"&search_path=profile")
	if err != nil {
		t.Fatalf("reconnect with search_path: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := repository.New(pool)
	svc := service.New(repo, log)

	cb := gobreaker.NewCircuitBreaker[any](gobreaker.Settings{
		Name:    "test",
		Timeout: 30 * time.Second,
	})

	h := handler.New(svc, repo, cb)

	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Get("/v1/profiles/{userId}", h.GetProfile)

	return &testEnv{handler: h, router: r, pool: pool, svc: svc}
}

func seedProfile(t *testing.T, env *testEnv, id, email, displayName string) {
	t.Helper()
	err := env.svc.CreateDefaultProfile(context.Background(), id, email, displayName)
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}
}

func authedRouter(h *handler.ProfileHandler, userID string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := versojwt.NewContext(r.Context(), &versojwt.Claims{UserID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Patch("/v1/profiles/me", h.UpdateProfile)
	return r
}

func TestHealthReturnsOK(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("health: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status: got %q", resp["status"])
	}
}

func TestReadyReturnsOK(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ready: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetProfileReturnsProfile(t *testing.T) {
	env := setupTestEnv(t)
	profileID := "01JXYZ1234567890ABCDEFGHIJ"
	seedProfile(t, env, profileID, "alice@example.com", "Alice")

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/"+profileID, nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get profile: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] != profileID {
		t.Errorf("id: got %v", resp["id"])
	}
	if resp["username"] != "alice" {
		t.Errorf("username: got %v", resp["username"])
	}
	if resp["displayName"] != "Alice" {
		t.Errorf("displayName: got %v", resp["displayName"])
	}
	if resp["privacyLevel"] != "public" {
		t.Errorf("privacyLevel: got %v", resp["privacyLevel"])
	}
	if resp["preferredLanguage"] != "en" {
		t.Errorf("preferredLanguage: got %v", resp["preferredLanguage"])
	}
}

func TestGetProfileNotFound(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/01JXYZ_NONEXISTENT_ID_ABCD", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("get missing: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateProfilePartialUpdate(t *testing.T) {
	env := setupTestEnv(t)
	profileID := "01JXYZ1234567890ABCDEFGHKL"
	seedProfile(t, env, profileID, "bob@example.com", "Bob")

	body := map[string]any{
		"displayName":  "Robert",
		"bio":          "I love reading",
		"privacyLevel": "friends_only",
	}
	data, _ := json.Marshal(body)

	r := authedRouter(env.handler, profileID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/v1/profiles/me", bytes.NewReader(data)))

	if w.Code != http.StatusOK {
		t.Fatalf("update profile: got %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["displayName"] != "Robert" {
		t.Errorf("displayName: got %v", resp["displayName"])
	}
	if resp["bio"] != "I love reading" {
		t.Errorf("bio: got %v", resp["bio"])
	}
	if resp["privacyLevel"] != "friends_only" {
		t.Errorf("privacyLevel: got %v", resp["privacyLevel"])
	}

	var eventType string
	err := env.pool.QueryRow(context.Background(),
		"SELECT event_type FROM outbox_events WHERE TRIM(aggregate_id) = $1", profileID,
	).Scan(&eventType)
	if err != nil {
		t.Fatalf("outbox event not in DB: %v", err)
	}
	if eventType != "verso.profile.profile-updated.v1" {
		t.Errorf("event_type: got %q", eventType)
	}
}

func TestUpdateProfileBadPrivacyLevel(t *testing.T) {
	env := setupTestEnv(t)
	profileID := "01JXYZ1234567890ABCDEFGHMN"
	seedProfile(t, env, profileID, "eve@example.com", "Eve")

	body := map[string]any{"privacyLevel": "invalid_level"}
	data, _ := json.Marshal(body)

	r := authedRouter(env.handler, profileID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/v1/profiles/me", bytes.NewReader(data)))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad privacy: got %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUpdateProfileBioTooLong(t *testing.T) {
	env := setupTestEnv(t)
	profileID := "01JXYZ1234567890ABCDEFGHOP"
	seedProfile(t, env, profileID, "long@example.com", "Long")

	longBio := make([]byte, 2001)
	for i := range longBio {
		longBio[i] = 'a'
	}
	body := map[string]any{"bio": string(longBio)}
	data, _ := json.Marshal(body)

	r := authedRouter(env.handler, profileID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/v1/profiles/me", bytes.NewReader(data)))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("bio too long: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateDefaultProfileDeduplication(t *testing.T) {
	env := setupTestEnv(t)
	profileID := "01JXYZ1234567890ABCDEFGHQR"

	err := env.svc.CreateDefaultProfile(context.Background(), profileID, "dedup@example.com", "Dedup")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	err = env.svc.CreateDefaultProfile(context.Background(), profileID, "dedup@example.com", "Dedup")
	if err != nil {
		t.Fatalf("duplicate create should not error: %v", err)
	}
}

func TestCreateDefaultProfileDeriveUsername(t *testing.T) {
	env := setupTestEnv(t)
	profileID := "01JXYZ1234567890ABCDEFGHST"

	err := env.svc.CreateDefaultProfile(context.Background(), profileID, "jane.doe@example.com", "Jane Doe")
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}

	profile, err := env.svc.GetProfile(context.Background(), profileID)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	if profile.Username != "jane.doe" {
		t.Errorf("username: got %q, want %q", profile.Username, "jane.doe")
	}
	if profile.DisplayName != "Jane Doe" {
		t.Errorf("displayName: got %q, want %q", profile.DisplayName, "Jane Doe")
	}
}
