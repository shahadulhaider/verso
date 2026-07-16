-- +goose Up
CREATE TABLE IF NOT EXISTS reading_progress (
    user_id             CHAR(26)        NOT NULL,
    work_id             CHAR(26)        NOT NULL,
    current_format_id   CHAR(26)        NOT NULL,
    progress_percent    NUMERIC(5,2)    NOT NULL DEFAULT 0,
    current_page        INT             NULL,
    status              VARCHAR(20)     NOT NULL DEFAULT 'not_started'
                                        CHECK (status IN ('not_started','reading','completed','dnf')),
    started_at          TIMESTAMPTZ     NULL,
    completed_at        TIMESTAMPTZ     NULL,
    read_count          INT             NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, work_id)
);

CREATE INDEX IF NOT EXISTS ix_progress_status ON reading_progress (user_id, status);

-- +goose Down
DROP TABLE IF EXISTS reading_progress;
