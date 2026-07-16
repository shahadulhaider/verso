-- +goose Up
CREATE TABLE IF NOT EXISTS shelf_item (
    id              CHAR(26)        PRIMARY KEY,
    shelf_id        CHAR(26)        NOT NULL REFERENCES shelf(id) ON DELETE CASCADE,
    user_id         CHAR(26)        NOT NULL,
    work_id         CHAR(26)        NOT NULL,
    edition_id      CHAR(26)        NULL,
    date_added      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    date_started    DATE            NULL,
    date_finished   DATE            NULL,
    display_order   INT             NULL,
    notes           TEXT            NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_shelf_item_shelf_work ON shelf_item (shelf_id, work_id);
CREATE INDEX IF NOT EXISTS ix_shelf_item_user_work ON shelf_item (user_id, work_id);
CREATE INDEX IF NOT EXISTS ix_shelf_item_added ON shelf_item (shelf_id, date_added DESC);

-- +goose Down
DROP TABLE IF EXISTS shelf_item;
