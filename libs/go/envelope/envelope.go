// Package envelope provides a canonical event envelope matching the proto event schema.
package envelope

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace"
)

// EventEnvelope carries metadata for every domain event published through the platform.
type EventEnvelope struct {
	EventID       string          `json:"eventId"`
	Type          string          `json:"type"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Producer      string          `json:"producer"`
	TraceID       string          `json:"traceId,omitempty"`
	PartitionKey  string          `json:"partitionKey"`
	Payload       json.RawMessage `json:"payload"`
	SchemaVersion int             `json:"schemaVersion"`
}

// New creates an EventEnvelope with a fresh ULID, current UTC time,
// and trace ID extracted from the OTel span context (if present).
func New(ctx context.Context, eventType, producer, partitionKey string, payload []byte) *EventEnvelope {
	env := &EventEnvelope{
		EventID:       ulid.MustNew(ulid.Now(), rand.Reader).String(),
		Type:          eventType,
		OccurredAt:    time.Now().UTC(),
		Producer:      producer,
		PartitionKey:  partitionKey,
		Payload:       payload,
		SchemaVersion: 1,
	}

	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		env.TraceID = sc.TraceID().String()
	}

	return env
}

// Marshal serializes the envelope to JSON.
func (e *EventEnvelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal deserializes JSON bytes into an EventEnvelope.
func Unmarshal(data []byte) (*EventEnvelope, error) {
	var env EventEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}
