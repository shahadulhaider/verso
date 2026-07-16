import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function profilesRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.profileServiceUrl;

  app.get<{ Params: { userId: string } }>(
    "/api/v1/profiles/:userId",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/profiles/${req.params.userId}`,
      });
    },
  );

  app.patch("/api/v1/profiles/me", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/profiles/me`,
    });
  });
}
