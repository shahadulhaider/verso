-- +goose Up
CREATE TABLE IF NOT EXISTS ai.llm_audit_log (
    id            CHAR(26) PRIMARY KEY,
    prompt_id     VARCHAR(100),
    model_id      VARCHAR(100) NOT NULL,
    provider      VARCHAR(50) NOT NULL,
    input_tokens  INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    latency_ms    INT NOT NULL DEFAULT 0,
    cache_hit     BOOLEAN NOT NULL DEFAULT FALSE,
    cost_usd      NUMERIC(10,6) DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS ai.llm_audit_log;
