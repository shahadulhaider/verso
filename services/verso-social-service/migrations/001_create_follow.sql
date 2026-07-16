-- +goose Up
CREATE TABLE IF NOT EXISTS follow (
    follower_id  CHAR(26)     NOT NULL,
    followed_id  CHAR(26)     NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followed_id)
);

CREATE INDEX IF NOT EXISTS ix_follow_followed ON follow (followed_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS follow;
