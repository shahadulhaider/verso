-- Verso Platform — Per-service Schemas
-- Loaded via /docker-entrypoint-initdb.d/ on first container start
-- Schema-per-service isolation (doc 05 §3.1)

CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS catalog;
