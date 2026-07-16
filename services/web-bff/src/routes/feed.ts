import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function feedRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.feedServiceUrl;

  app.get("/api/v1/feed/timeline", async (req, reply) => {
    const qs = req.url.split("?")[1];
    const target = qs
      ? `${base}/v1/feed/timeline?${qs}`
      : `${base}/v1/feed/timeline`;
    await proxyRequest(req, reply, { target });
  });
}
