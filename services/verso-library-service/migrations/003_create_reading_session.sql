-- +goose Up
CREATE TABLE IF NOT EXISTS reading_session (
    id                  CHAR(26)        PRIMARY KEY,
    user_id             CHAR(26)        NOT NULL,
    format_id           CHAR(26)        NOT NULL,
    work_id             CHAR(26)        NOT NULL,
    started_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    ended_at            TIMESTAMPTZ     NULL,
    duration_seconds    INT             NULL,
    progress_before     NUMERIC(5,2)    NOT NULL DEFAULT 0,
    progress_after      NUMERIC(5,2)    NOT NULL,
    pages_read          INT             NULL,
    device_type         VARCHAR(20)     NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_session_user_work ON reading_session (user_id, work_id, started_at DESC);
CREATE INDEX IF NOT EXISTS ix_session_user_recent ON reading_session (user_id, started_at DESC);
CREATE INDEX IF NOT EXISTS ix_session_format ON reading_session (format_id);

-- +goose Down
DROP TABLE IF EXISTS reading_session;
