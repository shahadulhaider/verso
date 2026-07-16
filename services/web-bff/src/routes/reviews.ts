import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function reviewsRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.reviewServiceUrl;

  // ── Reviews CRUD ─────────────────────────────────────────────────────
  app.post("/api/v1/reviews", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/reviews`,
    });
  });

  app.get<{ Params: { id: string } }>(
    "/api/v1/reviews/:id",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/reviews/${req.params.id}`,
      });
    },
  );

  app.patch<{ Params: { id: string } }>(
    "/api/v1/reviews/:id",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/reviews/${req.params.id}`,
      });
    },
  );

  app.delete<{ Params: { id: string } }>(
    "/api/v1/reviews/:id",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/reviews/${req.params.id}`,
      });
    },
  );

  // ── Review interactions ──────────────────────────────────────────────
  app.post<{ Params: { id: string } }>(
    "/api/v1/reviews/:id/votes",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/reviews/${req.params.id}/votes`,
      });
    },
  );

  app.post<{ Params: { id: string } }>(
    "/api/v1/reviews/:id/comments",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/reviews/${req.params.id}/comments`,
      });
    },
  );

  // ── Work-scoped reviews ──────────────────────────────────────────────
  app.get<{ Params: { workId: string } }>(
    "/api/v1/works/:workId/reviews",
    async (req, reply) => {
      const qs = req.url.split("?")[1];
      const target = qs
        ? `${base}/v1/works/${req.params.workId}/reviews?${qs}`
        : `${base}/v1/works/${req.params.workId}/reviews`;
      await proxyRequest(req, reply, { target });
    },
  );

  app.get<{ Params: { workId: string } }>(
    "/api/v1/works/:workId/aggregate-rating",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/works/${req.params.workId}/aggregate-rating`,
      });
    },
  );
}
