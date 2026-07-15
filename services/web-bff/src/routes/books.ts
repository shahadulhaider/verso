import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function booksRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.catalogServiceUrl;

  app.get("/api/v1/books", async (req, reply) => {
    const qs = req.url.split("?")[1];
    const target = qs ? `${base}/v1/works?${qs}` : `${base}/v1/works`;
    await proxyRequest(req, reply, { target });
  });

  app.get<{ Params: { id: string } }>(
    "/api/v1/books/:id",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/works/${req.params.id}`,
      });
    },
  );

  app.post("/api/v1/books", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/works`,
    });
  });
}
