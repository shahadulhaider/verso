-- +goose Up
CREATE TABLE IF NOT EXISTS work_merge_history (
    id              CHAR(26)       PRIMARY KEY,
    source_work_id  CHAR(26)       NOT NULL,
    target_work_id  CHAR(26)       NOT NULL REFERENCES work(id),
    merged_by       CHAR(26)       NOT NULL,
    reason          TEXT,
    merged_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_merge_source ON work_merge_history (source_work_id);
CREATE INDEX IF NOT EXISTS ix_merge_target ON work_merge_history (target_work_id);

-- +goose Down
DROP TABLE IF EXISTS work_merge_history;
