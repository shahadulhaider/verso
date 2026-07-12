package envelope_test

import (
	"context"
	"testing"

	"github.com/shahadulhaider/verso/libs/go/envelope"
)

func TestNew_ULIDValid(t *testing.T) {
	env := envelope.New(context.Background(), "book.created.v1", "verso-catalog-service", "book-123", []byte(`{"title":"Test"}`))

	if len(env.EventID) != 26 {
		t.Fatalf("expected ULID length 26, got %d: %q", len(env.EventID), env.EventID)
	}
	if env.Type != "book.created.v1" {
		t.Fatalf("expected type book.created.v1, got %q", env.Type)
	}
	if env.Producer != "verso-catalog-service" {
		t.Fatalf("expected producer verso-catalog-service, got %q", env.Producer)
	}
	if env.PartitionKey != "book-123" {
		t.Fatalf("expected partition key book-123, got %q", env.PartitionKey)
	}
	if env.SchemaVersion != 1 {
		t.Fatalf("expected schema version 1, got %d", env.SchemaVersion)
	}
	if env.OccurredAt.IsZero() {
		t.Fatal("occurredAt should not be zero")
	}
	// No OTel span in context → traceId should be empty
	if env.TraceID != "" {
		t.Fatalf("expected empty traceId without span context, got %q", env.TraceID)
	}
}

func TestMarshalUnmarshal_Roundtrip(t *testing.T) {
	original := envelope.New(context.Background(), "user.registered.v1", "verso-identity-service", "user-456", []byte(`{"email":"a@b.com"}`))

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	restored, err := envelope.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.EventID != original.EventID {
		t.Errorf("eventId mismatch: got %q, want %q", restored.EventID, original.EventID)
	}
	if restored.Type != original.Type {
		t.Errorf("type mismatch: got %q, want %q", restored.Type, original.Type)
	}
	if restored.Producer != original.Producer {
		t.Errorf("producer mismatch: got %q, want %q", restored.Producer, original.Producer)
	}
	if restored.PartitionKey != original.PartitionKey {
		t.Errorf("partitionKey mismatch: got %q, want %q", restored.PartitionKey, original.PartitionKey)
	}
	if string(restored.Payload) != string(original.Payload) {
		t.Errorf("payload mismatch: got %q, want %q", restored.Payload, original.Payload)
	}
}

func TestNew_UniqueIDs(t *testing.T) {
	env1 := envelope.New(context.Background(), "test.event.v1", "svc", "key", nil)
	env2 := envelope.New(context.Background(), "test.event.v1", "svc", "key", nil)
	if env1.EventID == env2.EventID {
		t.Fatal("two envelopes should have unique ULIDs")
	}
}
