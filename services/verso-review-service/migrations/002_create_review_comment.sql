-- +goose Up
CREATE TABLE IF NOT EXISTS review_comment (
    id                  CHAR(26)        PRIMARY KEY,
    review_id           CHAR(26)        NOT NULL REFERENCES review(id) ON DELETE CASCADE,
    user_id             CHAR(26)        NOT NULL,
    parent_comment_id   CHAR(26)        NULL,
    body                TEXT            NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ     NULL
);

CREATE INDEX IF NOT EXISTS ix_comment_review ON review_comment (review_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS review_comment;
