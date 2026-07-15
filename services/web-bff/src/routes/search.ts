import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function searchRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.searchServiceUrl;

  app.get("/api/v1/search", async (req, reply) => {
    const qs = req.url.split("?")[1];
    const target = qs ? `${base}/v1/search?${qs}` : `${base}/v1/search`;
    await proxyRequest(req, reply, { target });
  });
}
