-- +goose Up
CREATE TABLE IF NOT EXISTS oauth_connection (
    id                  CHAR(26)        PRIMARY KEY,
    account_id          CHAR(26)        NOT NULL REFERENCES account(id),
    provider            VARCHAR(50)     NOT NULL,
    provider_user_id    VARCHAR(255)    NOT NULL,
    access_token_enc    TEXT            NULL,
    refresh_token_enc   TEXT            NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_oauth_provider_user
    ON oauth_connection (provider, provider_user_id);

-- +goose Down
DROP TABLE IF EXISTS oauth_connection;
