// Package outbox provides transactional outbox helpers for reliable event publishing.
package outbox

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shahadulhaider/verso/libs/go/envelope"
)

// CreateTableSQL is the DDL for the outbox_events table.
const CreateTableSQL = `CREATE TABLE IF NOT EXISTS outbox_events (
	event_id       CHAR(26)     PRIMARY KEY,
	aggregate_type TEXT         NOT NULL,
	aggregate_id   TEXT         NOT NULL,
	event_type     TEXT         NOT NULL,
	payload        JSONB        NOT NULL,
	created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
	delivered      BOOLEAN      NOT NULL DEFAULT FALSE
)`

// InsertEvent writes an event envelope into the outbox within the given transaction.
func InsertEvent(ctx context.Context, tx pgx.Tx, aggregateType, aggregateID string, env *envelope.EventEnvelope) error {
	payload, err := env.Marshal()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox_events (event_id, aggregate_type, aggregate_id, event_type, payload)
		 VALUES ($1, $2, $3, $4, $5)`,
		env.EventID, aggregateType, aggregateID, env.Type, payload,
	)
	return err
}

// MarkDelivered flags an outbox event as delivered.
func MarkDelivered(ctx context.Context, pool *pgxpool.Pool, eventID string) error {
	_, err := pool.Exec(ctx,
		`UPDATE outbox_events SET delivered = TRUE WHERE event_id = $1`,
		eventID,
	)
	return err
}

// PendingEvents retrieves undelivered outbox events ordered by creation time.
func PendingEvents(ctx context.Context, pool *pgxpool.Pool, limit int) ([]*envelope.EventEnvelope, error) {
	rows, err := pool.Query(ctx,
		`SELECT payload FROM outbox_events WHERE delivered = FALSE ORDER BY created_at ASC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*envelope.EventEnvelope
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var env envelope.EventEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, err
		}
		events = append(events, &env)
	}
	return events, rows.Err()
}
