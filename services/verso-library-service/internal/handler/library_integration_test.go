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
	"github.com/shahadulhaider/verso/services/verso-library-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-library-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-library-service/internal/service"
)

type testEnv struct {
	handler *handler.LibraryHandler
	router  *chi.Mux
	pool    *pgxpool.Pool
	svc     *service.LibraryService
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
		`CREATE SCHEMA IF NOT EXISTS library`,
		`SET search_path TO library`,
		`CREATE TABLE IF NOT EXISTS shelf (
			id CHAR(26) PRIMARY KEY, user_id CHAR(26) NOT NULL, name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) NOT NULL, shelf_type VARCHAR(20) NOT NULL CHECK (shelf_type IN ('want_to_read','reading','read','dnf','custom')),
			is_system BOOLEAN NOT NULL DEFAULT FALSE, is_private BOOLEAN NOT NULL DEFAULT FALSE,
			display_order INT NOT NULL DEFAULT 0, item_count INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE UNIQUE INDEX uq_shelf_user_slug ON shelf (user_id, slug)`,
		`CREATE TABLE IF NOT EXISTS shelf_item (
			id CHAR(26) PRIMARY KEY, shelf_id CHAR(26) NOT NULL REFERENCES shelf(id) ON DELETE CASCADE,
			user_id CHAR(26) NOT NULL, work_id CHAR(26) NOT NULL, edition_id CHAR(26) NULL,
			date_added TIMESTAMPTZ NOT NULL DEFAULT NOW(), date_started DATE NULL, date_finished DATE NULL,
			display_order INT NULL, notes TEXT NULL)`,
		`CREATE UNIQUE INDEX uq_shelf_item_shelf_work ON shelf_item (shelf_id, work_id)`,
		`CREATE INDEX ix_shelf_item_user_work ON shelf_item (user_id, work_id)`,
		`CREATE TABLE IF NOT EXISTS reading_session (
			id CHAR(26) PRIMARY KEY, user_id CHAR(26) NOT NULL, format_id CHAR(26) NOT NULL,
			work_id CHAR(26) NOT NULL, started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), ended_at TIMESTAMPTZ NULL,
			duration_seconds INT NULL, progress_before NUMERIC(5,2) NOT NULL DEFAULT 0,
			progress_after NUMERIC(5,2) NOT NULL, pages_read INT NULL, device_type VARCHAR(20) NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS reading_progress (
			user_id CHAR(26) NOT NULL, work_id CHAR(26) NOT NULL, current_format_id CHAR(26) NOT NULL,
			progress_percent NUMERIC(5,2) NOT NULL DEFAULT 0, current_page INT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started','reading','completed','dnf')),
			started_at TIMESTAMPTZ NULL, completed_at TIMESTAMPTZ NULL, read_count INT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (user_id, work_id))`,
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
	pool, err = pgxpool.New(ctx, connStr+"&search_path=library")
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
			ctx := versojwt.NewContext(r.Context(), &versojwt.Claims{UserID: "01TESTUSER0000000000000001"})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/v1/library/shelves", h.ListShelves)
	r.Post("/v1/library/shelves", h.CreateShelf)
	r.Get("/v1/library/shelves/{shelfId}/items", h.ListShelfItems)
	r.Post("/v1/library/shelves/{shelfId}/items", h.AddShelfItem)
	r.Delete("/v1/library/shelves/{shelfId}/items/{itemId}", h.RemoveShelfItem)
	r.Post("/v1/library/sessions", h.LogSession)
	r.Get("/v1/library/progress/{workId}", h.GetProgress)
	r.Patch("/v1/library/progress/{workId}", h.UpdateProgress)

	return &testEnv{handler: h, router: r, pool: pool, svc: svc}
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

func TestCreateDefaultShelves(t *testing.T) {
	env := setupTestEnv(t)
	err := env.svc.CreateDefaultShelves(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("create default shelves: %v", err)
	}

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/library/shelves", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list shelves: got %d, body: %s", w.Code, w.Body.String())
	}

	var shelves []map[string]any
	json.Unmarshal(w.Body.Bytes(), &shelves)
	if len(shelves) != 4 {
		t.Fatalf("expected 4 system shelves, got %d", len(shelves))
	}

	expectedTypes := map[string]bool{"want_to_read": false, "reading": false, "read": false, "dnf": false}
	for _, s := range shelves {
		st := s["shelfType"].(string)
		if _, ok := expectedTypes[st]; !ok {
			t.Errorf("unexpected shelf type: %s", st)
		}
		expectedTypes[st] = true
		if s["isSystem"] != true {
			t.Errorf("shelf %s should be system", s["name"])
		}
	}
	for k, v := range expectedTypes {
		if !v {
			t.Errorf("missing shelf type: %s", k)
		}
	}
}

func TestCreateDefaultShelvesIdempotent(t *testing.T) {
	env := setupTestEnv(t)
	if err := env.svc.CreateDefaultShelves(context.Background(), testUserID); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := env.svc.CreateDefaultShelves(context.Background(), testUserID); err != nil {
		t.Fatalf("idempotent create should not error: %v", err)
	}
}

func TestCreateCustomShelf(t *testing.T) {
	env := setupTestEnv(t)
	body, _ := json.Marshal(map[string]any{"name": "Favorites", "isPrivate": true})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/library/shelves", bytes.NewReader(body)))

	if w.Code != http.StatusCreated {
		t.Fatalf("create shelf: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "Favorites" {
		t.Errorf("name: got %v", resp["name"])
	}
	if resp["slug"] != "favorites" {
		t.Errorf("slug: got %v", resp["slug"])
	}
	if resp["shelfType"] != "custom" {
		t.Errorf("shelfType: got %v", resp["shelfType"])
	}
	if resp["isPrivate"] != true {
		t.Errorf("isPrivate should be true")
	}
}

func TestAddAndRemoveShelfItem(t *testing.T) {
	env := setupTestEnv(t)
	env.svc.CreateDefaultShelves(context.Background(), testUserID)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/library/shelves", nil))
	var shelves []map[string]any
	json.Unmarshal(w.Body.Bytes(), &shelves)

	var wantToReadID string
	for _, s := range shelves {
		if s["shelfType"] == "want_to_read" {
			wantToReadID = s["id"].(string)
			break
		}
	}

	workID := "01WORKID00000000000000001"
	body, _ := json.Marshal(map[string]string{"workId": workID})
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/library/shelves/"+wantToReadID+"/items", bytes.NewReader(body)))

	if w.Code != http.StatusCreated {
		t.Fatalf("add item: got %d, body: %s", w.Code, w.Body.String())
	}

	var itemResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &itemResp)
	itemID := itemResp["id"].(string)

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/library/shelves/"+wantToReadID+"/items", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list items: got %d", w.Code)
	}
	var listResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &listResp)
	items := listResp["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	var outboxType string
	env.pool.QueryRow(context.Background(),
		"SELECT event_type FROM outbox_events LIMIT 1").Scan(&outboxType)
	if outboxType != "verso.library.shelf-item-added.v1" {
		t.Errorf("outbox event: got %q", outboxType)
	}

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/v1/library/shelves/"+wantToReadID+"/items/"+itemID, nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("remove item: got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestSystemShelfMoveLogic(t *testing.T) {
	env := setupTestEnv(t)
	env.svc.CreateDefaultShelves(context.Background(), testUserID)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/library/shelves", nil))
	var shelves []map[string]any
	json.Unmarshal(w.Body.Bytes(), &shelves)

	shelfIDs := map[string]string{}
	for _, s := range shelves {
		shelfIDs[s["shelfType"].(string)] = s["id"].(string)
	}

	workID := "01WORKID00000000000000002"
	body, _ := json.Marshal(map[string]string{"workId": workID})

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/library/shelves/"+shelfIDs["want_to_read"]+"/items", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("add to want_to_read: got %d", w.Code)
	}

	body, _ = json.Marshal(map[string]string{"workId": workID})
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/library/shelves/"+shelfIDs["reading"]+"/items", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("move to reading: got %d, body: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/library/shelves/"+shelfIDs["want_to_read"]+"/items", nil))
	var oldItems map[string]any
	json.Unmarshal(w.Body.Bytes(), &oldItems)
	if items := oldItems["items"].([]any); len(items) != 0 {
		t.Errorf("want_to_read should be empty after move, got %d items", len(items))
	}
}

func TestLogReadingSession(t *testing.T) {
	env := setupTestEnv(t)
	body, _ := json.Marshal(map[string]any{
		"formatId": "01FORMATID000000000000001", "workId": "01WORKID00000000000000003",
		"progressBefore": 10.0, "progressAfter": 25.5, "pagesRead": 30, "deviceType": "web",
	})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/library/sessions", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("log session: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["progressAfter"].(float64) != 25.5 {
		t.Errorf("progressAfter: got %v", resp["progressAfter"])
	}
}

func TestLogSessionInvalidProgress(t *testing.T) {
	env := setupTestEnv(t)
	body, _ := json.Marshal(map[string]any{
		"formatId": "01FORMATID000000000000001", "workId": "01WORKID00000000000000003",
		"progressBefore": 50.0, "progressAfter": 25.0,
	})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/library/sessions", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid progress, got %d", w.Code)
	}
}

func TestUpdateAndGetProgress(t *testing.T) {
	env := setupTestEnv(t)
	workID := "01WORKID00000000000000004"

	body, _ := json.Marshal(map[string]any{
		"formatId": "01FORMATID000000000000002", "progressPercent": 45.5,
		"currentPage": 120, "status": "reading",
	})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/v1/library/progress/"+workID, bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("update progress: got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["progressPercent"].(float64) != 45.5 {
		t.Errorf("progressPercent: got %v", resp["progressPercent"])
	}
	if resp["status"] != "reading" {
		t.Errorf("status: got %v", resp["status"])
	}
	if resp["startedAt"] == nil {
		t.Error("startedAt should be set when status=reading")
	}

	var outboxType string
	env.pool.QueryRow(context.Background(),
		"SELECT event_type FROM outbox_events WHERE event_type = 'verso.library.reading-progress-updated.v1' LIMIT 1").Scan(&outboxType)
	if outboxType != "verso.library.reading-progress-updated.v1" {
		t.Errorf("outbox event: got %q", outboxType)
	}

	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/library/progress/"+workID, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get progress: got %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["progressPercent"].(float64) != 45.5 {
		t.Errorf("get progressPercent: got %v", resp["progressPercent"])
	}
}

func TestProgressNotFound(t *testing.T) {
	env := setupTestEnv(t)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/library/progress/01NONEXISTENT00000000000", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
