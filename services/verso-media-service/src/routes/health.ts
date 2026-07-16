import type { FastifyInstance } from "fastify";
import type pg from "pg";

interface HealthDeps {
  pool: pg.Pool;
}

export async function healthRoutes(
  app: FastifyInstance,
  { pool }: HealthDeps,
): Promise<void> {
  app.get("/health", async () => ({ status: "ok" }));

  app.get("/ready", async (_request, reply) => {
    try {
      await pool.query("SELECT 1");
      return { status: "ready" };
    } catch {
      return reply.code(503).send({ status: "not_ready" });
    }
  });
}
