-- +goose Up
CREATE TABLE IF NOT EXISTS contributor (
    id                  CHAR(26)       PRIMARY KEY,
    name                VARCHAR(255)   NOT NULL,
    sort_name           VARCHAR(255),
    bio                 TEXT,
    photo_url           VARCHAR(500),
    claimed_by_user_id  CHAR(26),
    is_verified         BOOLEAN        NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ    NULL
);

CREATE TABLE IF NOT EXISTS credit (
    id               CHAR(26)       PRIMARY KEY,
    work_id          CHAR(26)       REFERENCES work(id),
    edition_id       CHAR(26)       REFERENCES edition(id),
    contributor_id   CHAR(26)       NOT NULL REFERENCES contributor(id),
    role             VARCHAR(50)    NOT NULL,
    display_order    INT            NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_credit_work_id ON credit (work_id);
CREATE INDEX IF NOT EXISTS ix_credit_edition_id ON credit (edition_id);
CREATE INDEX IF NOT EXISTS ix_credit_contributor_id ON credit (contributor_id);

-- +goose Down
DROP TABLE IF EXISTS credit;
DROP TABLE IF EXISTS contributor;
