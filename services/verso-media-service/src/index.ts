import "./lib/otel.js";
import Fastify from "fastify";
import multipart from "@fastify/multipart";
import { loadConfig } from "./config.js";
import { getPool, runMigration } from "./lib/db.js";
import {
  createS3Client,
  createUploadBreaker,
  createPresignBreaker,
  ensureBucket,
} from "./lib/s3.js";
import { initJwks } from "./lib/jwt.js";
import { healthRoutes } from "./routes/health.js";
import { mediaRoutes } from "./routes/media.js";

const config = loadConfig();

const app = Fastify({
  logger: {
    name: "verso-media-service",
    timestamp: () => `,"time":"${new Date().toISOString()}"`,
  },
});

await app.register(multipart, {
  limits: { fileSize: config.maxFileSize },
});

const pool = getPool(config.databaseUrl);
const s3 = createS3Client(config);
const uploadBreaker = createUploadBreaker(s3);
const presignBreaker = createPresignBreaker(s3);

initJwks(config.jwksUrl);

await runMigration(pool);
app.log.info("Database migration applied");

await ensureBucket(s3, config.minio.bucket);
app.log.info({ bucket: config.minio.bucket }, "MinIO bucket ready");

await app.register(healthRoutes, { pool });
await app.register(mediaRoutes, {
  config,
  pool,
  uploadBreaker,
  presignBreaker,
});

try {
  await app.listen({ port: config.port, host: "0.0.0.0" });
} catch (err) {
  app.log.error(err);
  process.exit(1);
}

export { app };
