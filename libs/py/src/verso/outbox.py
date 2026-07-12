"""Transactional outbox pattern for reliable event delivery."""

from __future__ import annotations

from verso.envelope import EventEnvelope, marshal_envelope, unmarshal_envelope

CREATE_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS outbox_events (
    id              TEXT PRIMARY KEY,
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         BYTEA NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_outbox_pending
    ON outbox_events (created_at)
    WHERE delivered_at IS NULL;
"""


async def insert_event(
    conn,
    aggregate_type: str,
    aggregate_id: str,
    envelope: EventEnvelope,
) -> None:
    """Insert an event into the outbox within the current transaction."""
    await conn.execute(
        """
        INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload)
        VALUES ($1, $2, $3, $4, $5)
        """,
        envelope.event_id,
        aggregate_type,
        aggregate_id,
        envelope.type,
        marshal_envelope(envelope),
    )


async def mark_delivered(pool, event_id: str) -> None:
    """Mark an outbox event as delivered."""
    await pool.execute(
        """
        UPDATE outbox_events SET delivered_at = NOW() WHERE id = $1
        """,
        event_id,
    )


async def pending_events(pool, limit: int = 100) -> list[EventEnvelope]:
    """Fetch pending (undelivered) outbox events ordered by creation time."""
    rows = await pool.fetch(
        """
        SELECT payload FROM outbox_events
        WHERE delivered_at IS NULL
        ORDER BY created_at ASC
        LIMIT $1
        """,
        limit,
    )
    return [unmarshal_envelope(row["payload"]) for row in rows]
