-- +goose Up
CREATE TABLE IF NOT EXISTS edition_identifier (
    id                CHAR(26)       PRIMARY KEY,
    edition_id        CHAR(26)       NOT NULL,
    identifier_type   VARCHAR(20)    NOT NULL CHECK (identifier_type IN ('isbn_13','isbn_10','asin','doi','open_library','wikidata')),
    identifier_value  VARCHAR(255)   NOT NULL,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (edition_id, identifier_type, identifier_value)
);

CREATE INDEX IF NOT EXISTS ix_edition_identifier_type_value ON edition_identifier (identifier_type, identifier_value);

-- +goose Down
DROP TABLE IF EXISTS edition_identifier;
