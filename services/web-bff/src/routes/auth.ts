import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function authRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.identityServiceUrl;

  app.post("/api/v1/auth/register", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/auth/register`,
    });
  });

  app.post("/api/v1/auth/login", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/auth/login`,
    });
  });

  app.post("/api/v1/auth/refresh", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/auth/token/refresh`,
    });
  });
}
