import "./lib/otel.js";
import Fastify from "fastify";
import { loadConfig } from "./config.js";
import { getPool, runMigrations } from "./lib/db.js";
import { getRedis } from "./lib/redis.js";
import { createKafka, createConsumer } from "./lib/kafka.js";
import { createSocialBreaker } from "./lib/social-client.js";
import { initJwks } from "./lib/jwt.js";
import { healthRoutes } from "./routes/health.js";
import { timelineRoutes } from "./routes/timeline.js";
import { buildHandlers, TOPICS } from "./consumers/event-handlers.js";
import type { FanOutDeps } from "./consumers/fan-out.js";

const config = loadConfig();

const app = Fastify({
  logger: {
    name: "verso-feed-service",
    timestamp: () => `,"time":"${new Date().toISOString()}"`,
  },
});

const pool = getPool(config.databaseUrl);
const redis = getRedis(config.redisUrl);
const kafka = createKafka(config.kafkaBrokers);
const socialBreaker = createSocialBreaker(config.socialServiceUrl);

initJwks(config.jwksUrl);

await runMigrations(pool);
app.log.info("Database migrations applied");

const fanOutDeps: FanOutDeps = {
  pool,
  redis,
  socialBreaker,
  cacheTtl: config.timelineCacheTtl,
  logger: app.log,
};

const handlers = buildHandlers(fanOutDeps);
const consumer = await createConsumer(kafka, [...TOPICS], handlers, app.log);
app.log.info({ topics: TOPICS }, "Kafka consumers started");

await app.register(healthRoutes, { pool, redis, kafka });
await app.register(timelineRoutes, { pool, redis });

const shutdown = async () => {
  app.log.info("Shutting down...");
  await consumer.disconnect();
  await redis.quit();
  await pool.end();
  await app.close();
};

process.on("SIGTERM", shutdown);
process.on("SIGINT", shutdown);

try {
  await app.listen({ port: config.port, host: "0.0.0.0" });
} catch (err) {
  app.log.error(err);
  process.exit(1);
}

export { app };
