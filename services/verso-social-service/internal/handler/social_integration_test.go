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
	"github.com/shahadulhaider/verso/services/verso-social-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-social-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-social-service/internal/service"
)

type testEnv struct {
	handler *handler.SocialHandler
	pool    *pgxpool.Pool
	svc     *service.SocialService
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
		`CREATE SCHEMA IF NOT EXISTS social`,
		`SET search_path TO social`,
		`CREATE TABLE IF NOT EXISTS follow (
			follower_id  CHAR(26)     NOT NULL,
			followed_id  CHAR(26)     NOT NULL,
			created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			PRIMARY KEY (follower_id, followed_id)
		)`,
		`CREATE INDEX IF NOT EXISTS ix_follow_followed ON follow (followed_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS friendship (
			id            CHAR(26)      PRIMARY KEY,
			user_a_id     CHAR(26)      NOT NULL,
			user_b_id     CHAR(26)      NOT NULL,
			status        VARCHAR(20)   NOT NULL CHECK (status IN ('pending', 'accepted', 'declined')),
			initiated_by  CHAR(26)      NOT NULL,
			created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
			accepted_at   TIMESTAMPTZ   NULL,
			CONSTRAINT uq_friendship_pair UNIQUE (user_a_id, user_b_id)
		)`,
		`CREATE TABLE IF NOT EXISTS block (
			blocker_id  CHAR(26)     NOT NULL,
			blocked_id  CHAR(26)     NOT NULL,
			created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			PRIMARY KEY (blocker_id, blocked_id)
		)`,
		`CREATE INDEX IF NOT EXISTS ix_block_blocked ON block (blocked_id)`,
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
	pool, err = pgxpool.New(ctx, connStr+"&search_path=social")
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
	return &testEnv{handler: h, pool: pool, svc: svc}
}

func authedRouter(h *handler.SocialHandler, userID string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := versojwt.NewContext(r.Context(), &versojwt.Claims{UserID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/v1/social/follow", h.Follow)
	r.Delete("/v1/social/follow/{userId}", h.Unfollow)
	r.Post("/v1/social/block", h.Block)
	r.Delete("/v1/social/block/{userId}", h.Unblock)
	r.Get("/v1/social/followers/{userId}", h.Followers)
	r.Get("/v1/social/following/{userId}", h.Following)
	r.Get("/v1/social/counts/{userId}", h.Counts)
	return r
}

func publicRouter(h *handler.SocialHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Get("/v1/social/followers/{userId}", h.Followers)
	r.Get("/v1/social/following/{userId}", h.Following)
	r.Get("/v1/social/counts/{userId}", h.Counts)
	return r
}

func TestHealthReturnsOK(t *testing.T) {
	env := setupTestEnv(t)
	r := publicRouter(env.handler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

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
	r := publicRouter(env.handler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("ready: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestFollowAndUnfollow(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGHIJ"
	bob := "01JXYZ1234567890ABCDEFGHKL"

	r := authedRouter(env.handler, alice)

	body, _ := json.Marshal(map[string]string{"userId": bob})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/follow", bytes.NewReader(body)))

	if w.Code != http.StatusNoContent {
		t.Fatalf("follow: got %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Given: verify outbox event was written
	var eventType string
	err := env.pool.QueryRow(context.Background(),
		"SELECT event_type FROM outbox_events LIMIT 1",
	).Scan(&eventType)
	if err != nil {
		t.Fatalf("outbox event not found: %v", err)
	}
	if eventType != "verso.social.user-followed.v1" {
		t.Errorf("event_type: got %q", eventType)
	}

	// When: unfollow
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/v1/social/follow/"+bob, nil))

	// Then: 204 regardless
	if w.Code != http.StatusNoContent {
		t.Fatalf("unfollow: got %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestFollowSelfReturns400(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGHIJ"

	r := authedRouter(env.handler, alice)
	body, _ := json.Marshal(map[string]string{"userId": alice})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/follow", bytes.NewReader(body)))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("follow self: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFollowDuplicateReturns409(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGHMN"
	bob := "01JXYZ1234567890ABCDEFGHOP"

	r := authedRouter(env.handler, alice)
	body, _ := json.Marshal(map[string]string{"userId": bob})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/follow", bytes.NewReader(body)))
	if w.Code != http.StatusNoContent {
		t.Fatalf("first follow: got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/follow", bytes.NewReader(body)))
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate follow: got %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestUnfollowIdempotent(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGHQR"
	bob := "01JXYZ1234567890ABCDEFGHST"

	r := authedRouter(env.handler, alice)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/v1/social/follow/"+bob, nil))

	if w.Code != http.StatusNoContent {
		t.Fatalf("idempotent unfollow: got %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestBlockAutoUnfollowsBothDirections(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGUV0"
	bob := "01JXYZ1234567890ABCDEFGWX0"

	// Given: alice follows bob AND bob follows alice
	if err := env.svc.Follow(context.Background(), alice, bob); err != nil {
		t.Fatalf("alice follow bob: %v", err)
	}
	if err := env.svc.Follow(context.Background(), bob, alice); err != nil {
		t.Fatalf("bob follow alice: %v", err)
	}

	// When: alice blocks bob
	r := authedRouter(env.handler, alice)
	body, _ := json.Marshal(map[string]string{"userId": bob})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/block", bytes.NewReader(body)))

	if w.Code != http.StatusNoContent {
		t.Fatalf("block: got %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Then: both follow directions are deleted
	counts, err := env.svc.Counts(context.Background(), alice)
	if err != nil {
		t.Fatalf("counts alice: %v", err)
	}
	if counts.FollowersCount != 0 {
		t.Errorf("alice followers: got %d, want 0", counts.FollowersCount)
	}
	if counts.FollowingCount != 0 {
		t.Errorf("alice following: got %d, want 0", counts.FollowingCount)
	}

	counts, err = env.svc.Counts(context.Background(), bob)
	if err != nil {
		t.Fatalf("counts bob: %v", err)
	}
	if counts.FollowersCount != 0 {
		t.Errorf("bob followers: got %d, want 0", counts.FollowersCount)
	}
	if counts.FollowingCount != 0 {
		t.Errorf("bob following: got %d, want 0", counts.FollowingCount)
	}
}

func TestBlockedUserCannotFollow(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGYZ0"
	bob := "01JXYZ1234567890ABCDEFG120"

	// Given: alice blocks bob
	if err := env.svc.Block(context.Background(), alice, bob); err != nil {
		t.Fatalf("block: %v", err)
	}

	// When: bob tries to follow alice
	r := authedRouter(env.handler, bob)
	body, _ := json.Marshal(map[string]string{"userId": alice})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/follow", bytes.NewReader(body)))

	// Then: forbidden
	if w.Code != http.StatusForbidden {
		t.Fatalf("blocked follow: got %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestBlockSelfReturns400(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFG340"

	r := authedRouter(env.handler, alice)
	body, _ := json.Marshal(map[string]string{"userId": alice})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/block", bytes.NewReader(body)))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("block self: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUnblockIdempotent(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFG560"
	bob := "01JXYZ1234567890ABCDEFG780"

	r := authedRouter(env.handler, alice)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/v1/social/block/"+bob, nil))

	if w.Code != http.StatusNoContent {
		t.Fatalf("idempotent unblock: got %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestFollowersAndFollowingLists(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFG9A0"
	bob := "01JXYZ1234567890ABCDEFGBC0"
	carol := "01JXYZ1234567890ABCDEFGDE0"

	// Given: bob and carol follow alice
	if err := env.svc.Follow(context.Background(), bob, alice); err != nil {
		t.Fatalf("bob follow alice: %v", err)
	}
	if err := env.svc.Follow(context.Background(), carol, alice); err != nil {
		t.Fatalf("carol follow alice: %v", err)
	}

	r := publicRouter(env.handler)

	// When: get alice's followers
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/social/followers/"+alice, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("followers: got %d, want %d", w.Code, http.StatusOK)
	}

	var followersResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &followersResp)
	count := int(followersResp["count"].(float64))
	if count != 2 {
		t.Errorf("followers count: got %d, want 2", count)
	}
	userIDs := followersResp["userIds"].([]any)
	if len(userIDs) != 2 {
		t.Errorf("followers list len: got %d, want 2", len(userIDs))
	}

	// When: get bob's following list
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/social/following/"+bob, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("following: got %d, want %d", w.Code, http.StatusOK)
	}

	var followingResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &followingResp)
	fCount := int(followingResp["count"].(float64))
	if fCount != 1 {
		t.Errorf("following count: got %d, want 1", fCount)
	}
}

func TestCountsEndpoint(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGFG0"
	bob := "01JXYZ1234567890ABCDEFGHI0"

	if err := env.svc.Follow(context.Background(), alice, bob); err != nil {
		t.Fatalf("follow: %v", err)
	}

	r := publicRouter(env.handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/social/counts/"+alice, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("counts: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	followers := int(resp["followersCount"].(float64))
	following := int(resp["followingCount"].(float64))
	if followers != 0 {
		t.Errorf("alice followers: got %d, want 0", followers)
	}
	if following != 1 {
		t.Errorf("alice following: got %d, want 1", following)
	}
}

func TestFollowMissingUserIdReturns400(t *testing.T) {
	env := setupTestEnv(t)
	alice := "01JXYZ1234567890ABCDEFGJK0"

	r := authedRouter(env.handler, alice)
	body, _ := json.Marshal(map[string]string{"userId": ""})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/social/follow", bytes.NewReader(body)))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty userId: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}
