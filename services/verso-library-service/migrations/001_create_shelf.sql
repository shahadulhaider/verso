-- +goose Up
CREATE TABLE IF NOT EXISTS shelf (
    id              CHAR(26)        PRIMARY KEY,
    user_id         CHAR(26)        NOT NULL,
    name            VARCHAR(100)    NOT NULL,
    slug            VARCHAR(100)    NOT NULL,
    shelf_type      VARCHAR(20)     NOT NULL
                                    CHECK (shelf_type IN ('want_to_read','reading','read','dnf','custom')),
    is_system       BOOLEAN         NOT NULL DEFAULT FALSE,
    is_private      BOOLEAN         NOT NULL DEFAULT FALSE,
    display_order   INT             NOT NULL DEFAULT 0,
    item_count      INT             NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_shelf_user_slug ON shelf (user_id, slug);
CREATE INDEX IF NOT EXISTS ix_shelf_user_type ON shelf (user_id, shelf_type);

-- +goose Down
DROP TABLE IF EXISTS shelf;
