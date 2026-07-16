import pg from "pg";

let pool: pg.Pool | null = null;

export function getPool(databaseUrl: string): pg.Pool {
  if (!pool) {
    pool = new pg.Pool({ connectionString: databaseUrl, max: 10 });
  }
  return pool;
}

export async function runMigration(pool: pg.Pool): Promise<void> {
  await pool.query(`
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
    )
  `);
  await pool.query(`
    CREATE INDEX IF NOT EXISTS ix_media_entity ON media.media_asset(entity_type, entity_id)
  `);
  await pool.query(`
    CREATE INDEX IF NOT EXISTS ix_media_uploader ON media.media_asset(uploader_id, created_at DESC)
  `);
}

export async function closePool(): Promise<void> {
  if (pool) {
    await pool.end();
    pool = null;
  }
}
