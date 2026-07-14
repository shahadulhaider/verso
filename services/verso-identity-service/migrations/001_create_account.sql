-- +goose Up
CREATE TABLE IF NOT EXISTS account (
    id              CHAR(26)        PRIMARY KEY,
    email           VARCHAR(320)    UNIQUE NOT NULL,
    email_verified  BOOLEAN         NOT NULL DEFAULT FALSE,
    password_hash   VARCHAR(255)    NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active', 'suspended', 'erased')),
    roles           VARCHAR[]       NOT NULL DEFAULT '{}',
    display_name    VARCHAR(255)    NOT NULL DEFAULT '',
    mfa_enabled     BOOLEAN         NOT NULL DEFAULT FALSE,
    erased_at       TIMESTAMPTZ     NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ix_account_email ON account (email);
CREATE INDEX IF NOT EXISTS ix_account_status ON account (status);

-- +goose Down
DROP TABLE IF EXISTS account;
