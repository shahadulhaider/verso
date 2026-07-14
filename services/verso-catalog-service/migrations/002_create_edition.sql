-- +goose Up
CREATE TABLE IF NOT EXISTS edition (
    id               CHAR(26)       PRIMARY KEY,
    work_id          CHAR(26)       NOT NULL REFERENCES work(id),
    title            VARCHAR(500),
    language         VARCHAR(10),
    publisher        VARCHAR(255),
    publication_date DATE,
    page_count       INT,
    word_count       INT,
    cover_image_url  VARCHAR(500),
    description      TEXT,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ    NULL,
    version          INT            NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS ix_edition_work_id ON edition (work_id);

-- +goose Down
DROP TABLE IF EXISTS edition;
