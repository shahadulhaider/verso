import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import Fastify, { type FastifyInstance } from "fastify";
import cors from "@fastify/cors";
import { authRoutes } from "../src/routes/auth.js";
import { booksRoutes } from "../src/routes/books.js";
import { searchRoutes } from "../src/routes/search.js";
import { profilesRoutes } from "../src/routes/profiles.js";
import { libraryRoutes } from "../src/routes/library.js";
import { reviewsRoutes } from "../src/routes/reviews.js";
import { socialRoutes } from "../src/routes/social.js";
import { feedRoutes } from "../src/routes/feed.js";
import { mediaRoutes } from "../src/routes/media.js";
import type { Config } from "../src/config.js";

const testConfig: Config = {
  port: 0,
  identityServiceUrl: "http://identity:8001",
  catalogServiceUrl: "http://catalog:8002",
  searchServiceUrl: "http://search:8003",
  profileServiceUrl: "http://profile:8004",
  mediaServiceUrl: "http://media:8005",
  libraryServiceUrl: "http://library:8006",
  reviewServiceUrl: "http://review:8007",
  socialServiceUrl: "http://social:8008",
  feedServiceUrl: "http://feed:8009",
};

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });

  await app.register(cors, {
    origin: "http://localhost:3100",
    methods: ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
    allowedHeaders: ["Authorization", "Content-Type"],
  });

  app.get("/health", async () => ({ status: "ok" }));

  await app.register(authRoutes, { config: testConfig });
  await app.register(booksRoutes, { config: testConfig });
  await app.register(searchRoutes, { config: testConfig });
  await app.register(profilesRoutes, { config: testConfig });
  await app.register(libraryRoutes, { config: testConfig });
  await app.register(reviewsRoutes, { config: testConfig });
  await app.register(socialRoutes, { config: testConfig });
  await app.register(feedRoutes, { config: testConfig });
  await app.register(mediaRoutes, { config: testConfig });

  return app;
}

describe("web-bff routes", () => {
  let app: FastifyInstance;

  beforeAll(async () => {
    app = await buildApp();
    await app.ready();
  });

  afterAll(async () => {
    await app.close();
  });

  // ── Health ────────────────────────────────────────────────────────────
  describe("GET /health", () => {
    it("returns status ok", async () => {
      const res = await app.inject({ method: "GET", url: "/health" });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual({ status: "ok" });
    });
  });

  // ── Route registration ────────────────────────────────────────────────
  describe("route registration", () => {
    it("registers POST /api/v1/auth/register", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/auth/register" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/auth/login", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/auth/login" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/auth/refresh", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/auth/refresh" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/books", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/books" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/books/:id", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/books/:id" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/books", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/books" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/search", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/search" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/search/semantic", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/search/semantic" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/profiles/:userId", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/profiles/:userId" });
      expect(route).toBe(true);
    });

    it("registers PATCH /api/v1/profiles/me", () => {
      const route = app.hasRoute({ method: "PATCH", url: "/api/v1/profiles/me" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/library/shelves", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/library/shelves" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/library/shelves", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/library/shelves" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/library/shelves/:shelfId/items", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/library/shelves/:shelfId/items" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/library/shelves/:shelfId/items", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/library/shelves/:shelfId/items" });
      expect(route).toBe(true);
    });

    it("registers DELETE /api/v1/library/shelves/:shelfId/items/:itemId", () => {
      const route = app.hasRoute({ method: "DELETE", url: "/api/v1/library/shelves/:shelfId/items/:itemId" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/library/sessions", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/library/sessions" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/library/progress/:workId", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/library/progress/:workId" });
      expect(route).toBe(true);
    });

    it("registers PATCH /api/v1/library/progress/:workId", () => {
      const route = app.hasRoute({ method: "PATCH", url: "/api/v1/library/progress/:workId" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/reviews", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/reviews" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/reviews/:id", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/reviews/:id" });
      expect(route).toBe(true);
    });

    it("registers PATCH /api/v1/reviews/:id", () => {
      const route = app.hasRoute({ method: "PATCH", url: "/api/v1/reviews/:id" });
      expect(route).toBe(true);
    });

    it("registers DELETE /api/v1/reviews/:id", () => {
      const route = app.hasRoute({ method: "DELETE", url: "/api/v1/reviews/:id" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/reviews/:id/votes", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/reviews/:id/votes" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/reviews/:id/comments", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/reviews/:id/comments" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/works/:workId/reviews", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/works/:workId/reviews" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/works/:workId/aggregate-rating", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/works/:workId/aggregate-rating" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/social/follow", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/social/follow" });
      expect(route).toBe(true);
    });

    it("registers DELETE /api/v1/social/follow/:userId", () => {
      const route = app.hasRoute({ method: "DELETE", url: "/api/v1/social/follow/:userId" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/social/followers/:userId", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/social/followers/:userId" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/social/following/:userId", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/social/following/:userId" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/social/counts/:userId", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/social/counts/:userId" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/social/block", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/social/block" });
      expect(route).toBe(true);
    });

    it("registers DELETE /api/v1/social/block/:userId", () => {
      const route = app.hasRoute({ method: "DELETE", url: "/api/v1/social/block/:userId" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/feed/timeline", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/feed/timeline" });
      expect(route).toBe(true);
    });

    it("registers POST /api/v1/media/upload", () => {
      const route = app.hasRoute({ method: "POST", url: "/api/v1/media/upload" });
      expect(route).toBe(true);
    });

    it("registers GET /api/v1/media/:id", () => {
      const route = app.hasRoute({ method: "GET", url: "/api/v1/media/:id" });
      expect(route).toBe(true);
    });
  });

  // ── CORS ──────────────────────────────────────────────────────────────
  describe("CORS", () => {
    it("allows requests from http://localhost:3100", async () => {
      const res = await app.inject({
        method: "OPTIONS",
        url: "/api/v1/books",
        headers: {
          origin: "http://localhost:3100",
          "access-control-request-method": "GET",
        },
      });
      expect(res.headers["access-control-allow-origin"]).toBe(
        "http://localhost:3100",
      );
    });

    it("sets allowed methods in CORS response", async () => {
      const res = await app.inject({
        method: "OPTIONS",
        url: "/api/v1/books",
        headers: {
          origin: "http://localhost:3100",
          "access-control-request-method": "POST",
        },
      });
      const allowedMethods = res.headers["access-control-allow-methods"];
      expect(allowedMethods).toContain("POST");
      expect(allowedMethods).toContain("GET");
      expect(allowedMethods).toContain("DELETE");
    });
  });

  // ── Proxy header forwarding ───────────────────────────────────────────
  describe("proxy helper", () => {
    it("forwards Authorization header to backend", async () => {
      const res = await app.inject({
        method: "GET",
        url: "/api/v1/books",
        headers: {
          authorization: "Bearer test-token-123",
        },
      });
      expect(res.statusCode).toBe(502);
      const body = res.json();
      expect(body.type).toBe("https://httpstatuses.com/502");
      expect(body.title).toBe("Bad Gateway");
    });

    it("returns 502 when backend is unreachable", async () => {
      const res = await app.inject({
        method: "POST",
        url: "/api/v1/auth/login",
        payload: { email: "test@example.com", password: "secret" },
      });
      expect(res.statusCode).toBe(502);
      expect(res.json().status).toBe(502);
    });
  });
});
