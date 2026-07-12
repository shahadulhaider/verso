package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("outbox_test"),
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

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}

	// Create outbox table
	if _, err := pool.Exec(ctx, outbox.CreateTableSQL); err != nil {
		t.Fatalf("create table: %v", err)
	}

	return pool, func() {
		pool.Close()
		_ = ctr.Terminate(ctx)
	}
}

func TestInsertAndPendingEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	env := envelope.New(ctx, "book.created.v1", "verso-catalog-service", "book-001", []byte(`{"title":"Go in Action"}`))

	// Insert within a transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := outbox.InsertEvent(ctx, tx, "Book", "book-001", env); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Retrieve pending
	events, err := outbox.PendingEvents(ctx, pool, 10)
	if err != nil {
		t.Fatalf("pending events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(events))
	}
	if events[0].EventID != env.EventID {
		t.Errorf("event ID mismatch: got %q, want %q", events[0].EventID, env.EventID)
	}
	if events[0].Type != "book.created.v1" {
		t.Errorf("event type mismatch: got %q", events[0].Type)
	}
}

func TestMarkDelivered(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	env := envelope.New(ctx, "user.registered.v1", "verso-identity-service", "user-001", []byte(`{}`))

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := outbox.InsertEvent(ctx, tx, "User", "user-001", env); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Mark as delivered
	if err := outbox.MarkDelivered(ctx, pool, env.EventID); err != nil {
		t.Fatalf("mark delivered: %v", err)
	}

	// Pending should be empty
	events, err := outbox.PendingEvents(ctx, pool, 10)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 pending after delivery, got %d", len(events))
	}
}
