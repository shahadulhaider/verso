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
	"github.com/shahadulhaider/verso/services/verso-review-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-review-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-review-service/internal/service"
)

type testEnv struct {
	handler *handler.ReviewHandler
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
		`CREATE SCHEMA IF NOT EXISTS review`,
		`SET search_path TO review`,
		`CREATE TABLE IF NOT EXISTS review (
			id CHAR(26) PRIMARY KEY, user_id CHAR(26) NOT NULL, work_id CHAR(26) NOT NULL,
			edition_id CHAR(26) NULL,
			rating_overall NUMERIC(2,1) NOT NULL CHECK (rating_overall >= 0.5 AND rating_overall <= 5.0 AND rating_overall * 2 = FLOOR(rating_overall * 2)),
			rating_plot NUMERIC(2,1) NULL, rating_characters NUMERIC(2,1) NULL,
			rating_pacing NUMERIC(2,1) NULL, rating_prose NUMERIC(2,1) NULL,
			title VARCHAR(255) NULL, body TEXT NULL, contains_spoilers BOOLEAN NOT NULL DEFAULT FALSE,
			like_count INT NOT NULL DEFAULT 0, comment_count INT NOT NULL DEFAULT 0,
			helpful_count INT NOT NULL DEFAULT 0, is_featured BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ NULL, version INT NOT NULL DEFAULT 1)`,
		`CREATE UNIQUE INDEX uq_review_user_work ON review (user_id, work_id) WHERE deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS review_comment (
			id CHAR(26) PRIMARY KEY, review_id CHAR(26) NOT NULL REFERENCES review(id) ON DELETE CASCADE,
			user_id CHAR(26) NOT NULL, parent_comment_id CHAR(26) NULL,
			body TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), deleted_at TIMESTAMPTZ NULL)`,
		`CREATE TABLE IF NOT EXISTS review_vote (
			user_id CHAR(26) NOT NULL, review_id CHAR(26) NOT NULL REFERENCES review(id) ON DELETE CASCADE,
			vote_type VARCHAR(10) NOT NULL CHECK (vote_type IN ('like','helpful')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (user_id, review_id))`,
		`CREATE TABLE IF NOT EXISTS outbox_events (
			event_id CHAR(26) PRIMARY KEY, aggregate_type TEXT NOT NULL, aggregate_id TEXT NOT NULL,
			event_type TEXT NOT NULL, payload JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			delivered BOOLEAN NOT NULL DEFAULT FALSE)`,
	}
	for _, ddl := range migrations {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	pool.Close()
	pool, err = pgxpool.New(ctx, connStr+"&search_path=review")
	if err != nil {
		t.Fatalf("reconnect with search_path: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := repository.New(pool)
	svc := service.New(repo, log)
	cb := gobreaker.NewCircuitBreaker[any](gobreaker.Settings{Name: "test", Timeout: 30 * time.Second})
	h := handler.New(svc, repo, cb)

	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := versojwt.NewContext(r.Context(), &versojwt.Claims{UserID: testUserID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/v1/reviews", h.CreateReview)
	r.Get("/v1/reviews/{id}", h.GetReview)
	r.Patch("/v1/reviews/{id}", h.UpdateReview)
	r.Delete("/v1/reviews/{id}", h.DeleteReview)
	r.Post("/v1/reviews/{id}/votes", h.CastVote)
	r.Post("/v1/reviews/{id}/comments", h.AddComment)
	r.Get("/v1/works/{workId}/reviews", h.ListReviews)
	r.Get("/v1/works/{workId}/aggregate-rating", h.GetAggregateRating)

	return &testEnv{handler: h, router: r, pool: pool}
}

const testUserID = "01TESTUSER0000000000000001"

func TestHealthReturnsOK(t *testing.T) {
	env := setupTestEnv(t)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("health: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestReadyReturnsOK(t *testing.T) {
	env := setupTestEnv(t)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ready: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCreateReview(t *testing.T) {
	env := setupTestEnv(t)
	body, _ := json.Marshal(map[string]any{
		"workId": "01WORKID00000000000000001", "ratingOverall": 4.5,
		"title": "Great book", "body": "Loved every page of this masterpiece.",
		"ratingPlot": 4.0, "ratingProse": 5.0,
	})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews", bytes.NewReader(body)))

	if w.Code != http.StatusCreated {
		t.Fatalf("create review: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ratingOverall"].(float64) != 4.5 {
		t.Errorf("ratingOverall: got %v", resp["ratingOverall"])
	}
	if resp["title"] != "Great book" {
		t.Errorf("title: got %v", resp["title"])
	}
	if resp["version"].(float64) != 1 {
		t.Errorf("version: got %v", resp["version"])
	}

	var outboxType string
	env.pool.QueryRow(context.Background(),
		"SELECT event_type FROM outbox_events LIMIT 1").Scan(&outboxType)
	if outboxType != "verso.review.review-published.v1" {
		t.Errorf("outbox event: got %q", outboxType)
	}
}

func TestCreateReviewDuplicateRejected(t *testing.T) {
	env := setupTestEnv(t)
	body, _ := json.Marshal(map[string]any{
		"workId": "01WORKID00000000000000002", "ratingOverall": 3.0,
	})

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("first create: got %d", w.Code)
	}

	body, _ = json.Marshal(map[string]any{
		"workId": "01WORKID00000000000000002", "ratingOverall": 4.0,
	})
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews", bytes.NewReader(body)))
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestCreateReviewInvalidRating(t *testing.T) {
	env := setupTestEnv(t)
	body, _ := json.Marshal(map[string]any{
		"workId": "01WORKID00000000000000003", "ratingOverall": 3.3,
	})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid rating: expected 400, got %d", w.Code)
	}
}

func createTestReview(t *testing.T, env *testEnv, workID string, rating float64) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"workId": workID, "ratingOverall": rating,
	})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create test review: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["id"].(string)
}

func TestUpdateReviewAndRatingEvent(t *testing.T) {
	env := setupTestEnv(t)
	id := createTestReview(t, env, "01WORKID00000000000000004", 3.0)

	body, _ := json.Marshal(map[string]any{"ratingOverall": 4.5})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/v1/reviews/"+id, bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("update: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ratingOverall"].(float64) != 4.5 {
		t.Errorf("updated rating: got %v", resp["ratingOverall"])
	}
	if resp["version"].(float64) != 2 {
		t.Errorf("version: got %v", resp["version"])
	}

	var outboxType string
	env.pool.QueryRow(context.Background(),
		"SELECT event_type FROM outbox_events WHERE event_type = 'verso.review.rating-updated.v1' LIMIT 1").Scan(&outboxType)
	if outboxType != "verso.review.rating-updated.v1" {
		t.Errorf("rating-updated outbox: got %q", outboxType)
	}
}

func TestDeleteReview(t *testing.T) {
	env := setupTestEnv(t)
	id := createTestReview(t, env, "01WORKID00000000000000005", 2.5)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/v1/reviews/"+id, nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d", w.Code)
	}

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/reviews/"+id, nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("after delete: expected 404, got %d", w.Code)
	}
}

func TestVoteToggle(t *testing.T) {
	env := setupTestEnv(t)
	id := createTestReview(t, env, "01WORKID00000000000000006", 4.0)

	body, _ := json.Marshal(map[string]any{"voteType": "like"})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews/"+id+"/votes", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("vote: got %d, body: %s", w.Code, w.Body.String())
	}
	var voteResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &voteResp)
	if voteResp["action"] != "created" {
		t.Errorf("first vote action: got %v", voteResp["action"])
	}

	body, _ = json.Marshal(map[string]any{"voteType": "like"})
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews/"+id+"/votes", bytes.NewReader(body)))
	json.Unmarshal(w.Body.Bytes(), &voteResp)
	if voteResp["action"] != "removed" {
		t.Errorf("toggle vote action: got %v", voteResp["action"])
	}

	body, _ = json.Marshal(map[string]any{"voteType": "helpful"})
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews/"+id+"/votes", bytes.NewReader(body)))
	json.Unmarshal(w.Body.Bytes(), &voteResp)
	if voteResp["action"] != "created" {
		t.Errorf("helpful vote action: got %v", voteResp["action"])
	}

	body, _ = json.Marshal(map[string]any{"voteType": "like"})
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews/"+id+"/votes", bytes.NewReader(body)))
	json.Unmarshal(w.Body.Bytes(), &voteResp)
	if voteResp["action"] != "switched" {
		t.Errorf("switch vote action: got %v", voteResp["action"])
	}
}

func TestAddComment(t *testing.T) {
	env := setupTestEnv(t)
	id := createTestReview(t, env, "01WORKID00000000000000007", 3.5)

	body, _ := json.Marshal(map[string]any{"body": "I agree with this review"})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/reviews/"+id+"/comments", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("add comment: got %d, body: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/reviews/"+id, nil))
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	comments := resp["comments"].([]any)
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
}

func TestListReviewsAndAggregate(t *testing.T) {
	env := setupTestEnv(t)
	workID := "01WORKID00000000000000008"
	createTestReview(t, env, workID, 4.0)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"/reviews", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list reviews: got %d", w.Code)
	}
	var listResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &listResp)
	items := listResp["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 review, got %d", len(items))
	}

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"/aggregate-rating", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("aggregate: got %d", w.Code)
	}
	var aggResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &aggResp)
	if aggResp["averageRating"].(float64) != 4.0 {
		t.Errorf("averageRating: got %v", aggResp["averageRating"])
	}
	if aggResp["ratingsCount"].(float64) != 1 {
		t.Errorf("ratingsCount: got %v", aggResp["ratingsCount"])
	}
}

func TestReviewNotFound(t *testing.T) {
	env := setupTestEnv(t)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/reviews/01NONEXISTENT00000000000", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
