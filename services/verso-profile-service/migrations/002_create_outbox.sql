-- +goose Up
CREATE TABLE IF NOT EXISTS outbox_events (
    event_id        CHAR(26)        PRIMARY KEY,
    aggregate_type  TEXT            NOT NULL,
    aggregate_id    TEXT            NOT NULL,
    event_type      TEXT            NOT NULL,
    payload         JSONB           NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    delivered       BOOLEAN         NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS ix_outbox_events_created_at ON outbox_events (created_at);

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
