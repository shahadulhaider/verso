import type { Pool, PoolClient } from "pg";

/** DDL for the outbox table — matches Go version exactly. */
export const CREATE_TABLE_SQL = `CREATE TABLE IF NOT EXISTS outbox_events (
    event_id       CHAR(26)     PRIMARY KEY,
    aggregate_type TEXT         NOT NULL,
    aggregate_id   TEXT         NOT NULL,
    event_type     TEXT         NOT NULL,
    payload        JSONB        NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    delivered      BOOLEAN      NOT NULL DEFAULT FALSE
)`;

export interface OutboxRow {
  event_id: string;
  aggregate_type: string;
  aggregate_id: string;
  event_type: string;
  payload: unknown;
  created_at: Date;
  delivered: boolean;
}

/**
 * Insert an outbox event within an existing transaction.
 * The caller must manage BEGIN/COMMIT around this call.
 *
 * @param client - A pg PoolClient (already in a transaction)
 * @param aggregateType - e.g. "book", "user"
 * @param aggregateId - e.g. the entity's UUID
 * @param envelope - The serialized EventEnvelope (as JSON-parseable object)
 */
export async function insertEvent(
  client: PoolClient,
  aggregateType: string,
  aggregateId: string,
  envelope: { eventId: string; type: string; [key: string]: unknown },
): Promise<void> {
  await client.query(
    `INSERT INTO outbox_events (event_id, aggregate_type, aggregate_id, event_type, payload)
     VALUES ($1, $2, $3, $4, $5)`,
    [envelope.eventId, aggregateType, aggregateId, envelope.type, JSON.stringify(envelope)],
  );
}

/**
 * Mark an outbox event as delivered.
 */
export async function markDelivered(pool: Pool, eventId: string): Promise<void> {
  await pool.query(
    `UPDATE outbox_events SET delivered = TRUE WHERE event_id = $1`,
    [eventId],
  );
}

/**
 * Fetch pending (undelivered) outbox events, ordered by creation time.
 */
export async function pendingEvents(pool: Pool, limit: number): Promise<OutboxRow[]> {
  const result = await pool.query<OutboxRow>(
    `SELECT event_id, aggregate_type, aggregate_id, event_type, payload, created_at, delivered
     FROM outbox_events
     WHERE delivered = FALSE
     ORDER BY created_at ASC
     LIMIT $1`,
    [limit],
  );
  return result.rows;
}
