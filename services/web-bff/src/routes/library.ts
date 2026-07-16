import type { FastifyInstance } from "fastify";
import { proxyRequest } from "../proxy.js";
import type { Config } from "../config.js";

export async function libraryRoutes(
  app: FastifyInstance,
  opts: { config: Config },
): Promise<void> {
  const base = opts.config.libraryServiceUrl;

  // ── Shelves ──────────────────────────────────────────────────────────
  app.get("/api/v1/library/shelves", async (req, reply) => {
    const qs = req.url.split("?")[1];
    const target = qs
      ? `${base}/v1/library/shelves?${qs}`
      : `${base}/v1/library/shelves`;
    await proxyRequest(req, reply, { target });
  });

  app.post("/api/v1/library/shelves", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/library/shelves`,
    });
  });

  app.get<{ Params: { shelfId: string } }>(
    "/api/v1/library/shelves/:shelfId/items",
    async (req, reply) => {
      const qs = req.url.split("?")[1];
      const target = qs
        ? `${base}/v1/library/shelves/${req.params.shelfId}/items?${qs}`
        : `${base}/v1/library/shelves/${req.params.shelfId}/items`;
      await proxyRequest(req, reply, { target });
    },
  );

  app.post<{ Params: { shelfId: string } }>(
    "/api/v1/library/shelves/:shelfId/items",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/library/shelves/${req.params.shelfId}/items`,
      });
    },
  );

  app.delete<{ Params: { shelfId: string; itemId: string } }>(
    "/api/v1/library/shelves/:shelfId/items/:itemId",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/library/shelves/${req.params.shelfId}/items/${req.params.itemId}`,
      });
    },
  );

  // ── Sessions ─────────────────────────────────────────────────────────
  app.post("/api/v1/library/sessions", async (req, reply) => {
    await proxyRequest(req, reply, {
      target: `${base}/v1/library/sessions`,
    });
  });

  // ── Progress ─────────────────────────────────────────────────────────
  app.get<{ Params: { workId: string } }>(
    "/api/v1/library/progress/:workId",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/library/progress/${req.params.workId}`,
      });
    },
  );

  app.patch<{ Params: { workId: string } }>(
    "/api/v1/library/progress/:workId",
    async (req, reply) => {
      await proxyRequest(req, reply, {
        target: `${base}/v1/library/progress/${req.params.workId}`,
      });
    },
  );
}
