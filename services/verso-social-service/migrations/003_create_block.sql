-- +goose Up
CREATE TABLE IF NOT EXISTS block (
    blocker_id  CHAR(26)     NOT NULL,
    blocked_id  CHAR(26)     NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id)
);

CREATE INDEX IF NOT EXISTS ix_block_blocked ON block (blocked_id);

-- +goose Down
DROP TABLE IF EXISTS block;
