import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function socialRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.socialServiceUrl;

  // ── Follow ───────────────────────────────────────────────────────────
  app.post("/api/v1/social/follow", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/social/follow`,
    });
  });

  app.delete<{ Params: { userId: string } }>(
    "/api/v1/social/follow/:userId",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/social/follow/${req.params.userId}`,
      });
    },
  );

  app.get<{ Params: { userId: string } }>(
    "/api/v1/social/followers/:userId",
    async (req, reply) => {
      const qs = req.url.split("?")[1];
      const target = qs
        ? `${base}/v1/social/followers/${req.params.userId}?${qs}`
        : `${base}/v1/social/followers/${req.params.userId}`;
      await proxyRequest(req, reply, { target });
    },
  );

  app.get<{ Params: { userId: string } }>(
    "/api/v1/social/following/:userId",
    async (req, reply) => {
      const qs = req.url.split("?")[1];
      const target = qs
        ? `${base}/v1/social/following/${req.params.userId}?${qs}`
        : `${base}/v1/social/following/${req.params.userId}`;
      await proxyRequest(req, reply, { target });
    },
  );

  app.get<{ Params: { userId: string } }>(
    "/api/v1/social/counts/:userId",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/social/counts/${req.params.userId}`,
      });
    },
  );

  // ── Block ────────────────────────────────────────────────────────────
  app.post("/api/v1/social/block", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/social/block`,
    });
  });

  app.delete<{ Params: { userId: string } }>(
    "/api/v1/social/block/:userId",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/social/block/${req.params.userId}`,
      });
    },
  );
}
