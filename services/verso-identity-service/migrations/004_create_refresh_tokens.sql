-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id              CHAR(26)        PRIMARY KEY,
    account_id      CHAR(26)        NOT NULL REFERENCES account(id),
    token_hash      VARCHAR(64)     NOT NULL,
    expires_at      TIMESTAMPTZ     NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_refresh_tokens_hash ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS ix_refresh_tokens_account ON refresh_tokens (account_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
