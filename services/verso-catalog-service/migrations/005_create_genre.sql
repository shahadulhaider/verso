-- +goose Up
CREATE TABLE IF NOT EXISTS genre (
    id               CHAR(26)       PRIMARY KEY,
    name             VARCHAR(100)   NOT NULL,
    slug             VARCHAR(100)   NOT NULL UNIQUE,
    parent_genre_id  CHAR(26)       REFERENCES genre(id),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS work_genre (
    work_id          CHAR(26)       NOT NULL REFERENCES work(id),
    genre_id         CHAR(26)       NOT NULL REFERENCES genre(id),
    relevance_score  NUMERIC(3,2),
    PRIMARY KEY (work_id, genre_id)
);

-- +goose Down
DROP TABLE IF EXISTS work_genre;
DROP TABLE IF EXISTS genre;
