import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function mediaRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.mediaServiceUrl;

  // Forward multipart upload as-is — proxy.ts forwards content-type + raw body
  app.post("/api/v1/media/upload", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/media/upload`,
    });
  });

  app.get<{ Params: { id: string } }>(
    "/api/v1/media/:id",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/media/${req.params.id}`,
      });
    },
  );
}
