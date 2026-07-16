-- +goose Up
CREATE TABLE IF NOT EXISTS friendship (
    id            CHAR(26)      PRIMARY KEY,
    user_a_id     CHAR(26)      NOT NULL,
    user_b_id     CHAR(26)      NOT NULL,
    status        VARCHAR(20)   NOT NULL CHECK (status IN ('pending', 'accepted', 'declined')),
    initiated_by  CHAR(26)      NOT NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    accepted_at   TIMESTAMPTZ   NULL,
    CONSTRAINT uq_friendship_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX IF NOT EXISTS ix_friendship_user_a ON friendship (user_a_id, status);
CREATE INDEX IF NOT EXISTS ix_friendship_user_b ON friendship (user_b_id, status);

-- +goose Down
DROP TABLE IF EXISTS friendship;
