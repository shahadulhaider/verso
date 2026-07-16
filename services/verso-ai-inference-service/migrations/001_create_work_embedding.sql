-- +goose Up
CREATE TABLE ai.work_embedding (
  id CHAR(26) PRIMARY KEY,
  work_id CHAR(26) NOT NULL UNIQUE,
  embedding vector(1024) NOT NULL,
  embedding_model VARCHAR(100) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_work_embedding_hnsw ON ai.work_embedding USING hnsw (embedding vector_cosine_ops);

CREATE TABLE ai.outbox_events (
  id CHAR(26) PRIMARY KEY,
  aggregate_type VARCHAR(100) NOT NULL,
  aggregate_id CHAR(26) NOT NULL,
  type VARCHAR(255) NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS ai.outbox_events;
DROP INDEX IF EXISTS ix_work_embedding_hnsw;
DROP TABLE IF EXISTS ai.work_embedding;
