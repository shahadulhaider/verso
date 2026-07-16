-- +goose Up
CREATE TABLE IF NOT EXISTS genre (
    id               CHAR(26)       PRIMARY KEY,
    name             VARCHAR        NOT NULL,
    slug             VARCHAR        NOT NULL UNIQUE,
    parent_genre_id  CHAR(26)       REFERENCES genre(id),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS work_genre (
    work_id          CHAR(26)       NOT NULL REFERENCES work(id),
    genre_id         CHAR(26)       NOT NULL REFERENCES genre(id),
    relevance_score  NUMERIC(2,1)   DEFAULT 1.0,
    PRIMARY KEY (work_id, genre_id)
);

CREATE TABLE IF NOT EXISTS trope (
    id          CHAR(26)       PRIMARY KEY,
    name        VARCHAR        NOT NULL,
    slug        VARCHAR        NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS work_trope (
    work_id     CHAR(26)       NOT NULL REFERENCES work(id),
    trope_id    CHAR(26)       NOT NULL REFERENCES trope(id),
    vote_count  INT            NOT NULL DEFAULT 0,
    PRIMARY KEY (work_id, trope_id)
);

CREATE TABLE IF NOT EXISTS mood (
    id          CHAR(26)       PRIMARY KEY,
    name        VARCHAR        NOT NULL,
    slug        VARCHAR        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS work_mood (
    work_id     CHAR(26)       NOT NULL REFERENCES work(id),
    mood_id     CHAR(26)       NOT NULL REFERENCES mood(id),
    vote_count  INT            NOT NULL DEFAULT 0,
    PRIMARY KEY (work_id, mood_id)
);

CREATE TABLE IF NOT EXISTS content_warning (
    id          CHAR(26)       PRIMARY KEY,
    name        VARCHAR        NOT NULL,
    severity    VARCHAR(10)    NOT NULL CHECK (severity IN ('mild','moderate','severe')),
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS work_content_warning (
    work_id             CHAR(26)       NOT NULL REFERENCES work(id),
    content_warning_id  CHAR(26)       NOT NULL REFERENCES content_warning(id),
    vote_count          INT            NOT NULL DEFAULT 0,
    PRIMARY KEY (work_id, content_warning_id)
);

-- +goose Down
DROP TABLE IF EXISTS work_content_warning;
DROP TABLE IF EXISTS content_warning;
DROP TABLE IF EXISTS work_mood;
DROP TABLE IF EXISTS mood;
DROP TABLE IF EXISTS work_trope;
DROP TABLE IF EXISTS trope;
DROP TABLE IF EXISTS work_genre;
DROP TABLE IF EXISTS genre;
