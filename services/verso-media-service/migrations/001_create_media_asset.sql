-- +goose Up
CREATE TABLE IF NOT EXISTS media.media_asset (
    id              CHAR(26) PRIMARY KEY,
    uploader_id     CHAR(26) NOT NULL,
    file_name       VARCHAR(255) NOT NULL,
    mime_type       VARCHAR(100) NOT NULL,
    file_size       BIGINT NOT NULL,
    object_key      VARCHAR(512) NOT NULL,
    bucket          VARCHAR(100) NOT NULL DEFAULT 'verso-media',
    entity_type     VARCHAR(20) NOT NULL CHECK (entity_type IN ('cover','avatar','attachment')),
    entity_id       CHAR(26),
    upload_status   VARCHAR(20) NOT NULL DEFAULT 'completed' CHECK (upload_status IN ('pending','completed','failed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_media_entity ON media.media_asset(entity_type, entity_id);
CREATE INDEX ix_media_uploader ON media.media_asset(uploader_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS media.ix_media_uploader;
DROP INDEX IF EXISTS media.ix_media_entity;
DROP TABLE IF EXISTS media.media_asset;
