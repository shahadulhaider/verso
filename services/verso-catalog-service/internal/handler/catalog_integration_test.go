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
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/shahadulhaider/verso/services/verso-catalog-service/internal/handler"
	"github.com/shahadulhaider/verso/services/verso-catalog-service/internal/repository"
	"github.com/shahadulhaider/verso/services/verso-catalog-service/internal/service"
)

type testEnv struct {
	handler *handler.CatalogHandler
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
		`CREATE SCHEMA IF NOT EXISTS catalog`,
		`SET search_path TO catalog`,
		`CREATE TABLE IF NOT EXISTS work (
			id                         CHAR(26)       PRIMARY KEY,
			title                      VARCHAR(500)   NOT NULL,
			description                TEXT,
			original_language          VARCHAR(10),
			original_publication_year  INT,
			avg_rating                 NUMERIC(3,1)   NOT NULL DEFAULT 0,
			ratings_count              INT            NOT NULL DEFAULT 0,
			reviews_count              INT            NOT NULL DEFAULT 0,
			merged_into_work_id        CHAR(26)       NULL,
			created_at                 TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
			updated_at                 TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
			deleted_at                 TIMESTAMPTZ    NULL,
			version                    INT            NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS edition (
			id               CHAR(26)       PRIMARY KEY,
			work_id          CHAR(26)       NOT NULL REFERENCES work(id),
			title            VARCHAR(500),
			language         VARCHAR(10),
			publisher        VARCHAR(255),
			publication_date DATE,
			page_count       INT,
			word_count       INT,
			cover_image_url  VARCHAR(500),
			description      TEXT,
			created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
			updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
			deleted_at       TIMESTAMPTZ    NULL,
			version          INT            NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS format (
			id               CHAR(26)       PRIMARY KEY,
			edition_id       CHAR(26)       NOT NULL REFERENCES edition(id),
			format_type      VARCHAR(20)    NOT NULL CHECK (format_type IN ('ebook', 'audiobook', 'print')),
			duration_seconds INT,
			file_size_bytes  BIGINT,
			drm_type         VARCHAR(20),
			file_format      VARCHAR(20),
			asset_url        VARCHAR(500),
			is_available     BOOLEAN        NOT NULL DEFAULT TRUE,
			created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
			updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
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
	pool, err = pgxpool.New(ctx, connStr+"&search_path=catalog")
	if err != nil {
		t.Fatalf("reconnect with search_path: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := repository.New(pool)
	svc := service.New(repo, log)
	h := handler.New(svc, repo)

	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Post("/v1/works", h.CreateWork)
	r.Get("/v1/works", h.ListWorks)
	r.Get("/v1/works/{id}", h.GetWork)
	r.Post("/v1/works/{id}/editions", h.CreateEdition)
	r.Get("/v1/editions/{id}", h.GetEdition)

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

func TestCreateWorkAndOutbox(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	body := map[string]any{
		"title":                   "The Great Gatsby",
		"description":            "A novel by F. Scott Fitzgerald",
		"originalLanguage":       "en",
		"originalPublicationYear": 1925,
	}
	w := postJSON(env.router, "/v1/works", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp struct {
		ID                      string  `json:"id"`
		Title                   string  `json:"title"`
		Description             *string `json:"description"`
		OriginalLanguage        *string `json:"originalLanguage"`
		OriginalPublicationYear *int    `json:"originalPublicationYear"`
		Version                 int     `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.ID) != 26 {
		t.Errorf("id should be ULID (26 chars): got %q", resp.ID)
	}
	if resp.Title != "The Great Gatsby" {
		t.Errorf("title: got %q", resp.Title)
	}
	if resp.Version != 1 {
		t.Errorf("version: got %d", resp.Version)
	}

	var eventType string
	err := env.pool.QueryRow(ctx, "SELECT event_type FROM outbox_events WHERE aggregate_id = $1", resp.ID).Scan(&eventType)
	if err != nil {
		t.Fatalf("outbox event not in DB: %v", err)
	}
	if eventType != "verso.catalog.work-created.v1" {
		t.Errorf("event_type: got %q", eventType)
	}
}

func TestCreateWorkMissingTitle(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]any{"description": "no title"}
	w := postJSON(env.router, "/v1/works", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateEditionAndOutbox(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	workBody := map[string]any{"title": "Moby Dick"}
	wk := postJSON(env.router, "/v1/works", workBody)
	var workResp struct{ ID string `json:"id"` }
	json.Unmarshal(wk.Body.Bytes(), &workResp)

	edBody := map[string]any{
		"title":           "Moby Dick: Penguin Classics",
		"language":        "en",
		"publisher":       "Penguin",
		"publicationDate": "2003-05-01",
		"pageCount":       720,
	}
	w := postJSON(env.router, "/v1/works/"+workResp.ID+"/editions", edBody)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var edResp struct {
		ID     string `json:"id"`
		WorkID string `json:"workId"`
	}
	json.Unmarshal(w.Body.Bytes(), &edResp)
	if len(edResp.ID) != 26 {
		t.Errorf("edition id should be ULID: got %q", edResp.ID)
	}
	if edResp.WorkID != workResp.ID {
		t.Errorf("workId mismatch: got %q, want %q", edResp.WorkID, workResp.ID)
	}

	var eventType string
	err := env.pool.QueryRow(ctx, "SELECT event_type FROM outbox_events WHERE aggregate_id = $1", edResp.ID).Scan(&eventType)
	if err != nil {
		t.Fatalf("outbox event not in DB: %v", err)
	}
	if eventType != "verso.catalog.edition-published.v1" {
		t.Errorf("event_type: got %q", eventType)
	}
}

func TestCreateEditionWorkNotFound(t *testing.T) {
	env := setupTestEnv(t)

	edBody := map[string]any{"title": "Nonexistent"}
	w := postJSON(env.router, "/v1/works/01NONEXISTENT0000000000000/editions", edBody)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListWorksWithPagination(t *testing.T) {
	env := setupTestEnv(t)

	for i := 0; i < 5; i++ {
		body := map[string]any{"title": "Work " + string(rune('A'+i))}
		w := postJSON(env.router, "/v1/works", body)
		if w.Code != http.StatusCreated {
			t.Fatalf("create work %d: got %d", i, w.Code)
		}
	}

	w := getJSON(env.router, "/v1/works?limit=3")
	if w.Code != http.StatusOK {
		t.Fatalf("list: got %d", w.Code)
	}

	var resp struct {
		Items      []json.RawMessage `json:"items"`
		NextCursor string            `json:"nextCursor"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Items) != 3 {
		t.Errorf("items count: got %d, want 3", len(resp.Items))
	}
	if resp.NextCursor == "" {
		t.Error("nextCursor should be set for more pages")
	}

	w2 := getJSON(env.router, "/v1/works?limit=3&cursor="+resp.NextCursor)
	if w2.Code != http.StatusOK {
		t.Fatalf("list page 2: got %d", w2.Code)
	}
	var resp2 struct {
		Items      []json.RawMessage `json:"items"`
		NextCursor string            `json:"nextCursor"`
	}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if len(resp2.Items) != 2 {
		t.Errorf("page 2 items: got %d, want 2", len(resp2.Items))
	}
	if resp2.NextCursor != "" {
		t.Error("nextCursor should be empty on last page")
	}
}

func TestGetWorkIncludesEditions(t *testing.T) {
	env := setupTestEnv(t)

	workBody := map[string]any{"title": "War and Peace"}
	wk := postJSON(env.router, "/v1/works", workBody)
	var workResp struct{ ID string `json:"id"` }
	json.Unmarshal(wk.Body.Bytes(), &workResp)

	postJSON(env.router, "/v1/works/"+workResp.ID+"/editions", map[string]any{"title": "Edition A"})
	postJSON(env.router, "/v1/works/"+workResp.ID+"/editions", map[string]any{"title": "Edition B"})

	w := getJSON(env.router, "/v1/works/"+workResp.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("get work: got %d", w.Code)
	}

	var resp struct {
		ID       string            `json:"id"`
		Title    string            `json:"title"`
		Editions []json.RawMessage `json:"editions"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Title != "War and Peace" {
		t.Errorf("title: got %q", resp.Title)
	}
	if len(resp.Editions) != 2 {
		t.Errorf("editions count: got %d, want 2", len(resp.Editions))
	}
}

func TestGetEditionIncludesFormats(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	workBody := map[string]any{"title": "Test Work"}
	wk := postJSON(env.router, "/v1/works", workBody)
	var workResp struct{ ID string `json:"id"` }
	json.Unmarshal(wk.Body.Bytes(), &workResp)

	edBody := map[string]any{"title": "Test Edition"}
	ed := postJSON(env.router, "/v1/works/"+workResp.ID+"/editions", edBody)
	var edResp struct{ ID string `json:"id"` }
	json.Unmarshal(ed.Body.Bytes(), &edResp)

	_, err := env.pool.Exec(ctx,
		`INSERT INTO format (id, edition_id, format_type, is_available) VALUES ('01TESTFORMAT0000000000000', $1, 'ebook', true)`,
		edResp.ID)
	if err != nil {
		t.Fatalf("insert format: %v", err)
	}

	w := getJSON(env.router, "/v1/editions/"+edResp.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("get edition: got %d", w.Code)
	}

	var resp struct {
		ID      string            `json:"id"`
		Formats []json.RawMessage `json:"formats"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Formats) != 1 {
		t.Errorf("formats count: got %d, want 1", len(resp.Formats))
	}
}

func TestGetWorkNotFound(t *testing.T) {
	env := setupTestEnv(t)

	w := getJSON(env.router, "/v1/works/01NONEXISTENT0000000000000")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", w.Code)
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
