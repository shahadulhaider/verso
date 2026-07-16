-- +goose Up
CREATE TABLE IF NOT EXISTS review_vote (
    user_id     CHAR(26)        NOT NULL,
    review_id   CHAR(26)        NOT NULL REFERENCES review(id) ON DELETE CASCADE,
    vote_type   VARCHAR(10)     NOT NULL CHECK (vote_type IN ('like', 'helpful')),
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, review_id)
);

CREATE INDEX IF NOT EXISTS ix_vote_review ON review_vote (review_id);

-- +goose Down
DROP TABLE IF EXISTS review_vote;
