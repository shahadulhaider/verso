-- Verso Platform — Postgres Init Extensions
-- Loaded via /docker-entrypoint-initdb.d/ on first container start

-- pgvector: required for semantic search (verso-search-service, verso-ai-inference-service)
CREATE EXTENSION IF NOT EXISTS vector;

-- uuid-ossp: utility for UUID generation helpers
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
