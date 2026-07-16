import { describe, it, expect, vi, beforeAll, afterAll, beforeEach } from "vitest";
import Fastify from "fastify";
import multipart from "@fastify/multipart";
import { mediaRoutes } from "../src/routes/media.js";
import { healthRoutes } from "../src/routes/health.js";
import type { Config } from "../src/config.js";
import FormData from "form-data";

vi.mock("../src/lib/jwt.js", () => ({
  initJwks: vi.fn(),
  verifyToken: vi.fn().mockResolvedValue({ sub: "01HQ3K8B0FXJN4T5Y7WKZM9P2R" }),
}));

const FAKE_PRESIGNED_URL = "http://minio:9000/verso-media/cover/fakeid/test.png?signed=yes";

function createMockPool() {
  return {
    query: vi.fn().mockResolvedValue({ rows: [], rowCount: 0 }),
  };
}

function createMockUploadBreaker() {
  return { fire: vi.fn().mockResolvedValue(undefined) };
}

function createMockPresignBreaker() {
  return { fire: vi.fn().mockResolvedValue(FAKE_PRESIGNED_URL) };
}

const testConfig: Config = {
  port: 8005,
  databaseUrl: "postgres://test:test@localhost:5432/test?search_path=media",
  minio: {
    endpoint: "http://localhost:9000",
    accessKey: "minioadmin",
    secretKey: "minioadmin_dev",
    bucket: "verso-media",
    region: "us-east-1",
  },
  jwksUrl: "http://localhost:8001/.well-known/jwks.json",
  presignedUrlExpiry: 3600,
  maxFileSize: 10 * 1024 * 1024,
};

function buildApp(
  pool = createMockPool(),
  uploadBreaker = createMockUploadBreaker(),
  presignBreaker = createMockPresignBreaker(),
) {
  const app = Fastify({ logger: false });
  app.register(multipart, { limits: { fileSize: testConfig.maxFileSize } });
  app.register(healthRoutes, { pool: pool as any });
  app.register(mediaRoutes, {
    config: testConfig,
    pool: pool as any,
    uploadBreaker: uploadBreaker as any,
    presignBreaker: presignBreaker as any,
  });
  return { app, pool, uploadBreaker, presignBreaker };
}

describe("Health endpoints", () => {
  it("GET /health returns ok", async () => {
    const { app } = buildApp();
    const res = await app.inject({ method: "GET", url: "/health" });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({ status: "ok" });
    await app.close();
  });

  it("GET /ready returns ready when DB is available", async () => {
    const pool = createMockPool();
    pool.query.mockResolvedValueOnce({ rows: [{ "?column?": 1 }] });
    const { app } = buildApp(pool);
    const res = await app.inject({ method: "GET", url: "/ready" });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({ status: "ready" });
    await app.close();
  });

  it("GET /ready returns 503 when DB is down", async () => {
    const pool = createMockPool();
    pool.query.mockRejectedValueOnce(new Error("connection refused"));
    const { app } = buildApp(pool);
    const res = await app.inject({ method: "GET", url: "/ready" });
    expect(res.statusCode).toBe(503);
    await app.close();
  });
});

describe("POST /v1/media/upload", () => {
  it("rejects request without Authorization header", async () => {
    const { app } = buildApp();
    const form = new FormData();
    form.append("file", Buffer.from("fake-png"), {
      filename: "test.png",
      contentType: "image/png",
    });
    form.append("entityType", "cover");

    const res = await app.inject({
      method: "POST",
      url: "/v1/media/upload",
      payload: form.getBuffer(),
      headers: {
        ...form.getHeaders(),
      },
    });
    expect(res.statusCode).toBe(401);
    await app.close();
  });

  it("uploads a file and returns 201 with asset metadata", async () => {
    const { app, pool, uploadBreaker, presignBreaker } = buildApp();
    const fileContent = Buffer.from("fake-png-content");
    const form = new FormData();
    form.append("file", fileContent, {
      filename: "test.png",
      contentType: "image/png",
    });
    form.append("entityType", "cover");

    const res = await app.inject({
      method: "POST",
      url: "/v1/media/upload",
      payload: form.getBuffer(),
      headers: {
        ...form.getHeaders(),
        authorization: "Bearer valid-token",
      },
    });

    expect(res.statusCode).toBe(201);
    const body = res.json();
    expect(body.id).toHaveLength(26);
    expect(body.fileName).toBe("test.png");
    expect(body.mimeType).toBe("image/png");
    expect(body.fileSize).toBe(fileContent.length);
    expect(body.url).toBe(FAKE_PRESIGNED_URL);

    expect(uploadBreaker.fire).toHaveBeenCalledOnce();
    expect(presignBreaker.fire).toHaveBeenCalledOnce();

    const insertCall = pool.query.mock.calls[0];
    expect(insertCall[0]).toContain("INSERT INTO media.media_asset");
    expect(insertCall[1][0]).toHaveLength(26);
    expect(insertCall[1][2]).toBe("test.png");
    expect(insertCall[1][3]).toBe("image/png");

    await app.close();
  });

  it("rejects unsupported mime type", async () => {
    const { app } = buildApp();
    const form = new FormData();
    form.append("file", Buffer.from("not-an-image"), {
      filename: "test.txt",
      contentType: "text/plain",
    });
    form.append("entityType", "attachment");

    const res = await app.inject({
      method: "POST",
      url: "/v1/media/upload",
      payload: form.getBuffer(),
      headers: {
        ...form.getHeaders(),
        authorization: "Bearer valid-token",
      },
    });
    expect(res.statusCode).toBe(400);
    expect(res.json().message).toContain("Unsupported mime type");
    await app.close();
  });

  it("rejects missing entityType", async () => {
    const { app } = buildApp();
    const form = new FormData();
    form.append("file", Buffer.from("fake-png"), {
      filename: "test.png",
      contentType: "image/png",
    });

    const res = await app.inject({
      method: "POST",
      url: "/v1/media/upload",
      payload: form.getBuffer(),
      headers: {
        ...form.getHeaders(),
        authorization: "Bearer valid-token",
      },
    });
    expect(res.statusCode).toBe(400);
    expect(res.json().message).toContain("entityType");
    await app.close();
  });

  it("accepts upload with entityId", async () => {
    const { app } = buildApp();
    const form = new FormData();
    form.append("file", Buffer.from("fake-png"), {
      filename: "avatar.webp",
      contentType: "image/webp",
    });
    form.append("entityType", "avatar");
    form.append("entityId", "01HQ3K8B0FXJN4T5Y7WKZM9P2R");

    const res = await app.inject({
      method: "POST",
      url: "/v1/media/upload",
      payload: form.getBuffer(),
      headers: {
        ...form.getHeaders(),
        authorization: "Bearer valid-token",
      },
    });
    expect(res.statusCode).toBe(201);
    expect(res.json().fileName).toBe("avatar.webp");
    await app.close();
  });
});

describe("GET /v1/media/:id", () => {
  it("returns asset metadata with presigned URL", async () => {
    const pool = createMockPool();
    pool.query.mockResolvedValueOnce({
      rows: [
        {
          object_key: "cover/fakeid/test.png",
          bucket: "verso-media",
          file_name: "test.png",
          mime_type: "image/png",
          file_size: "12345",
        },
      ],
    });

    const { app } = buildApp(pool);
    const res = await app.inject({
      method: "GET",
      url: "/v1/media/01HQ3K8B0FXJN4T5Y7WKZM9P2R",
    });

    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.id).toBe("01HQ3K8B0FXJN4T5Y7WKZM9P2R");
    expect(body.fileName).toBe("test.png");
    expect(body.mimeType).toBe("image/png");
    expect(body.fileSize).toBe(12345);
    expect(body.url).toBe(FAKE_PRESIGNED_URL);
    await app.close();
  });

  it("returns 404 for non-existent asset", async () => {
    const { app } = buildApp();
    const res = await app.inject({
      method: "GET",
      url: "/v1/media/01HQ3K8B0FXJN4T5Y7WKZM9999",
    });
    expect(res.statusCode).toBe(404);
    await app.close();
  });
});
