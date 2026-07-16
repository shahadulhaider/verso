import pg from "pg";

let pool: pg.Pool | null = null;

export function getPool(databaseUrl: string): pg.Pool {
  if (!pool) {
    pool = new pg.Pool({ connectionString: databaseUrl, max: 10 });
  }
  return pool;
}

export async function runMigrations(pool: pg.Pool): Promise<void> {
  // 001: activity table
  await pool.query(`
    CREATE TABLE IF NOT EXISTS feed.activity (
      id              CHAR(26) PRIMARY KEY,
      actor_id        CHAR(26) NOT NULL,
      verb            VARCHAR(50) NOT NULL,
      object_type     VARCHAR(50) NOT NULL,
      object_id       CHAR(26) NOT NULL,
      metadata        JSONB DEFAULT '{}',
      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `);
  await pool.query(`
    CREATE INDEX IF NOT EXISTS ix_activity_actor
    ON feed.activity(actor_id, created_at DESC)
  `);
  await pool.query(`
    CREATE INDEX IF NOT EXISTS ix_activity_created
    ON feed.activity(created_at DESC)
  `);

  // 002: timeline_entry table
  await pool.query(`
    CREATE TABLE IF NOT EXISTS feed.timeline_entry (
      id              CHAR(26) PRIMARY KEY,
      user_id         CHAR(26) NOT NULL,
      activity_id     CHAR(26) NOT NULL REFERENCES feed.activity(id),
      score           NUMERIC(20,6) NOT NULL,
      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `);
  await pool.query(`
    CREATE INDEX IF NOT EXISTS ix_timeline_user_score
    ON feed.timeline_entry(user_id, score DESC)
  `);

  // 003: outbox_events table
  await pool.query(`
    CREATE TABLE IF NOT EXISTS feed.outbox_events (
      id              CHAR(26) PRIMARY KEY,
      aggregate_type  VARCHAR(100) NOT NULL,
      aggregate_id    CHAR(26) NOT NULL,
      type            VARCHAR(255) NOT NULL,
      payload         JSONB NOT NULL,
      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `);
}

export async function closePool(): Promise<void> {
  if (pool) {
    await pool.end();
    pool = null;
  }
}
