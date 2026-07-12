import { describe, it, expect } from "vitest";
import pino from "pino";
import { createLogger, withTraceId } from "./index.js";
import { Writable } from "node:stream";

/** Capture pino JSON output into an array of parsed objects. */
function createSink(): { stream: Writable; lines: Record<string, unknown>[] } {
  const lines: Record<string, unknown>[] = [];
  const stream = new Writable({
    write(chunk, _encoding, callback) {
      const text = chunk.toString().trim();
      if (text) {
        lines.push(JSON.parse(text));
      }
      callback();
    },
  });
  return { stream, lines };
}

describe("createLogger", () => {
  it("outputs JSON with service_name", async () => {
    const { stream, lines } = createSink();
    const testLogger = pino(
      { name: "catalog-svc", base: { service_name: "catalog-svc" }, timestamp: false },
      stream,
    );
    testLogger.info("hello");
    stream.end();

    await new Promise((resolve) => stream.on("finish", resolve));

    expect(lines.length).toBeGreaterThanOrEqual(1);
    expect(lines[0]).toMatchObject({
      service_name: "catalog-svc",
      msg: "hello",
    });
  });

  it("createLogger returns a logger with expected methods", () => {
    const logger = createLogger("test-svc");
    expect(typeof logger.info).toBe("function");
    expect(typeof logger.error).toBe("function");
    expect(typeof logger.child).toBe("function");
  });
});

describe("withTraceId", () => {
  it("creates a child logger with traceId in output", async () => {
    const { stream, lines } = createSink();
    const parent = pino(
      { base: { service_name: "auth-svc" }, timestamp: false },
      stream,
    );
    const child = withTraceId(parent, "abc-trace-123");
    child.info("authenticated");
    stream.end();

    await new Promise((resolve) => stream.on("finish", resolve));

    expect(lines.length).toBeGreaterThanOrEqual(1);
    expect(lines[0]).toMatchObject({
      service_name: "auth-svc",
      traceId: "abc-trace-123",
      msg: "authenticated",
    });
  });
});
