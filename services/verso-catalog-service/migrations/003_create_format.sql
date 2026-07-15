-- +goose Up
CREATE TABLE IF NOT EXISTS format (
    id               CHAR(26)       PRIMARY KEY,
    edition_id       CHAR(26)       NOT NULL REFERENCES edition(id),
    format_type      VARCHAR(20)    NOT NULL CHECK (format_type IN ('ebook', 'audiobook', 'print')),
    duration_seconds INT,
    file_size_bytes  BIGINT,
    drm_type         VARCHAR(20),
    file_format      VARCHAR(20),
    asset_url        VARCHAR(500),
    is_available     BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_format_edition_id ON format (edition_id);

-- +goose Down
DROP TABLE IF EXISTS format;
