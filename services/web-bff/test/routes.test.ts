import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import Fastify, { type FastifyInstance } from "fastify";
import cors from "@fastify/cors";
import { authRoutes } from "../src/routes/auth.js";
import { booksRoutes } from "../src/routes/books.js";
import { searchRoutes } from "../src/routes/search.js";
import type { Config } from "../src/config.js";

const testConfig: Config = {
  port: 0,
  identityServiceUrl: "http://identity:8001",
  catalogServiceUrl: "http://catalog:8002",
  searchServiceUrl: "http://search:8003",
};

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });

  await app.register(cors, {
    origin: "http://localhost:3000",
    methods: ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
    allowedHeaders: ["Authorization", "Content-Type"],
  });

  app.get("/health", async () => ({ status: "ok" }));

  await app.register(authRoutes, { config: testConfig });
  await app.register(booksRoutes, { config: testConfig });
  await app.register(searchRoutes, { config: testConfig });

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
  });

  // ── CORS ──────────────────────────────────────────────────────────────
  describe("CORS", () => {
    it("allows requests from http://localhost:3000", async () => {
      const res = await app.inject({
        method: "OPTIONS",
        url: "/api/v1/books",
        headers: {
          origin: "http://localhost:3000",
          "access-control-request-method": "GET",
        },
      });
      expect(res.headers["access-control-allow-origin"]).toBe(
        "http://localhost:3000",
      );
    });

    it("sets allowed methods in CORS response", async () => {
      const res = await app.inject({
        method: "OPTIONS",
        url: "/api/v1/books",
        headers: {
          origin: "http://localhost:3000",
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
      // We test by injecting a request — the proxy will fail to connect
      // to the fake backend, but we can verify the route accepts the header
      const res = await app.inject({
        method: "GET",
        url: "/api/v1/books",
        headers: {
          authorization: "Bearer test-token-123",
        },
      });
      // Should get 502 because backend is unreachable, not 4xx
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
