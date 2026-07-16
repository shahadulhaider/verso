-- +goose Up
CREATE TABLE IF NOT EXISTS series (
    id          CHAR(26)       PRIMARY KEY,
    name        VARCHAR        NOT NULL,
    slug        VARCHAR        NOT NULL UNIQUE,
    description TEXT,
    book_count  INT            NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS series_entry (
    work_id    CHAR(26)       NOT NULL REFERENCES work(id),
    series_id  CHAR(26)       NOT NULL REFERENCES series(id),
    position   NUMERIC(5,1)   NOT NULL,
    PRIMARY KEY (work_id, series_id)
);

CREATE INDEX IF NOT EXISTS ix_series_slug ON series (slug);
CREATE INDEX IF NOT EXISTS ix_series_entry_series ON series_entry (series_id, position);

-- +goose Down
DROP TABLE IF EXISTS series_entry;
DROP TABLE IF EXISTS series;
