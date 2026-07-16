-- +goose Up
CREATE TABLE IF NOT EXISTS ai.prompt_registry (
    id          CHAR(26) PRIMARY KEY,
    prompt_id   VARCHAR(100) NOT NULL,
    version     INT NOT NULL DEFAULT 1,
    model_tier  VARCHAR(50) NOT NULL DEFAULT 'default',
    feature     VARCHAR(100) NOT NULL,
    template    TEXT NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prompt_id, version)
);

-- +goose Down
DROP TABLE IF EXISTS ai.prompt_registry;
