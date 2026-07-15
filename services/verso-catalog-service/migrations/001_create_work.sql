-- +goose Up
CREATE TABLE IF NOT EXISTS work (
    id                         CHAR(26)       PRIMARY KEY,
    title                      VARCHAR(500)   NOT NULL,
    description                TEXT,
    original_language          VARCHAR(10),
    original_publication_year  INT,
    avg_rating                 NUMERIC(3,1)   NOT NULL DEFAULT 0,
    ratings_count              INT            NOT NULL DEFAULT 0,
    reviews_count              INT            NOT NULL DEFAULT 0,
    merged_into_work_id        CHAR(26)       NULL,
    created_at                 TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at                 TIMESTAMPTZ    NULL,
    version                    INT            NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS ix_work_title ON work (title);
CREATE INDEX IF NOT EXISTS ix_work_created_at ON work (created_at);

-- +goose Down
DROP TABLE IF EXISTS work;
