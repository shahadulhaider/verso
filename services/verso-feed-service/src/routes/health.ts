import type { FastifyInstance } from "fastify";
import type pg from "pg";
import type { Redis } from "ioredis";
import type { Kafka } from "kafkajs";

interface HealthDeps {
  pool: pg.Pool;
  redis: Redis;
  kafka: Kafka;
}

export async function healthRoutes(
  app: FastifyInstance,
  { pool, redis, kafka }: HealthDeps,
): Promise<void> {
  app.get("/health", async () => ({ status: "ok" }));

  app.get("/ready", async (_request, reply) => {
    const checks: Record<string, string> = {};

    try {
      await pool.query("SELECT 1");
      checks.database = "ok";
    } catch {
      checks.database = "failed";
    }

    try {
      await redis.ping();
      checks.redis = "ok";
    } catch {
      checks.redis = "failed";
    }

    try {
      const admin = kafka.admin();
      await admin.connect();
      await admin.disconnect();
      checks.kafka = "ok";
    } catch {
      checks.kafka = "failed";
    }

    const allOk = Object.values(checks).every((v) => v === "ok");
    if (!allOk) {
      return reply.code(503).send({ status: "not_ready", checks });
    }
    return { status: "ready", checks };
  });
}
